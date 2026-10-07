package machine

import (
	"reflect"
	"strings"
	"testing"
)

type catalogSnapshot struct {
	catalog, subject, sourceIndex, sourceSubject Object
	records                                      map[string]Object
}

func nativeCatalogSnapshot(t *testing.T, r *Repository) *catalogSnapshot {
	t.Helper()
	index, err := r.Read(policyIndex)
	if err != nil {
		t.Fatal(err)
	}
	subject, err := r.Read(policySubjectRegistry)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := r.Read("developer/policy/registries/root-family-migration.json")
	if err != nil {
		t.Fatal(err)
	}
	historical := map[string]Object{}
	for _, raw := range List(graph["sources"]) {
		source := Map(raw)
		historical[Text(source["source_policy_id"])] = Map(source["header"])
	}
	records := map[string]Object{}
	for _, raw := range List(index["policies"]) {
		entry := Map(raw)
		id := Text(entry["id"])
		if entry["authority_role"] == "MIGRATION_SOURCE" {
			records[id] = policyClone(historical[id])
			if records[id] == nil {
				t.Fatal("historical header missing", id)
			}
		} else {
			record, err := r.Read(Text(entry["path"]))
			if err != nil {
				t.Fatal(err)
			}
			records[id] = record
		}
	}
	return &catalogSnapshot{catalog: policyClone(index), subject: policyClone(subject), records: records, sourceIndex: index, sourceSubject: subject}
}

func (s *catalogSnapshot) failures(r *Repository) []string {
	return r.ValidateNeutralCatalogSnapshot(s.catalog, s.subject, s.records, s.sourceIndex, s.sourceSubject)
}

func catalogEntry(t *testing.T, s *catalogSnapshot, id string) Object {
	t.Helper()
	for _, raw := range List(s.catalog["policies"]) {
		row := Map(raw)
		if row["id"] == id {
			return row
		}
	}
	t.Fatal("missing snapshot identity", id)
	return nil
}

func TestNeutralCatalogRegistrationAndExactContractResolution(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	if failures := r.ValidateNeutralCatalogContractRegistration(); len(failures) != 0 {
		t.Fatal(failures)
	}
	contract, err := r.Read(policyCatalogContracts)
	if err != nil {
		t.Fatal(err)
	}
	if contract["catalog_application_status"] != "APPLIED" || len(List(Map(contract["change_scope"])["materialization_targets"])) != 7 {
		t.Fatal(contract)
	}
	for _, raw := range List(Map(contract["change_scope"])["deferred_application_targets"]) {
		if Map(raw)["execution_authorized"] != false {
			t.Fatal("deferred execution inferred")
		}
	}
	resolved, err := r.ResolveNeutralCatalogContract("developer-policy-catalog/v1")
	if err != nil || resolved["canonical_path"] != policyIndex {
		t.Fatal(resolved, err)
	}
	if _, err := r.ResolveNeutralCatalogContract("developer-policy-catalog"); err == nil {
		t.Fatal("similar contract identity resolved")
	}
	for _, id := range Map(contract["entrypoints"]) {
		schema := Map(Map(contract["contracts"])[Text(id)])
		payload, err := r.Read(Text(schema["canonical_path"]))
		if err != nil {
			t.Fatal(err)
		}
		if payload["policy_class"] != nil || payload["schema_version"] != id || payload["artifact_class"] != schema["artifact_class"] {
			t.Fatal(payload)
		}
	}
}

func TestNeutralCatalogSnapshotPreservesExactRegisteredCorpus(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	snapshot := nativeCatalogSnapshot(t, r)
	if failures := snapshot.failures(r); len(failures) != 0 {
		t.Fatal(failures)
	}
}

func TestNeutralCatalogRejectsMissingUnknownAndDriftedMetadata(t *testing.T) {
	for _, test := range []struct {
		field  string
		value  any
		remove bool
	}{{"policy_class", nil, true}, {"policy_class", "UNKNOWN_POLICY_CLASS", false}, {"policy_class", "VPMS_DEVELOPER_POLICY", false}, {"status", "DRAFT", false}, {"path", "developer/policy/MPD-0010.yaml", false}, {"policy_class", "PTSIP_BOUND_POLICY", false}} {
		t.Run(test.field+"/"+Text(test.value), func(t *testing.T) {
			r, err := Open(".")
			if err != nil {
				t.Fatal(err)
			}
			snapshot := nativeCatalogSnapshot(t, r)
			row := catalogEntry(t, snapshot, "MPD-NORM-0001")
			if test.remove {
				delete(row, test.field)
			} else {
				row[test.field] = test.value
			}
			if failures := snapshot.failures(r); len(failures) == 0 {
				t.Fatal("invalid neutral catalog metadata admitted")
			}
		})
	}
}

func TestNeutralCatalogRejectsDuplicateIdentitiesAcrossClassesAndNormativeFields(t *testing.T) {
	for _, mode := range []string{"duplicate", "cross_class_duplicate", "normative_rules"} {
		t.Run(mode, func(t *testing.T) {
			r, err := Open(".")
			if err != nil {
				t.Fatal(err)
			}
			snapshot := nativeCatalogSnapshot(t, r)
			if mode == "normative_rules" {
				catalogEntry(t, snapshot, "MPD-NORM-0001")["rules"] = Object{"unregistered_rule": true}
			} else {
				row := policyClone(catalogEntry(t, snapshot, "MPD-NORM-0001"))
				if mode == "cross_class_duplicate" {
					row["policy_class"] = "VPMS_DEVELOPER_POLICY"
				}
				snapshot.catalog["policies"] = append(List(snapshot.catalog["policies"]), row)
			}
			if failures := snapshot.failures(r); len(failures) == 0 {
				t.Fatal("duplicate identity or normative catalog field admitted")
			}
		})
	}
}

func TestNeutralSubjectRejectsMembershipAndSemanticProjectionDrift(t *testing.T) {
	for _, field := range []string{"membership", "repository_identity_schemes", "current_repository_bindings", "matching", "fuzzy_match", "ai_semantic_match", "index_policy_class", "subject_policy_class"} {
		t.Run(field, func(t *testing.T) {
			r, err := Open(".")
			if err != nil {
				t.Fatal(err)
			}
			snapshot := nativeCatalogSnapshot(t, r)
			switch field {
			case "membership":
				identity := Map(Map(snapshot.subject["subject_identity_schemes"])["MANAGEMENT_POLICY_ID"])
				values := List(identity["registered_values"])
				identity["registered_values"] = values[:len(values)-1]
			case "repository_identity_schemes":
				Map(Map(snapshot.subject[field])["GITHUB_REPOSITORY_ID"])["host_required"] = false
			case "current_repository_bindings":
				Map(List(snapshot.subject[field])[0])["repository_id"] = 1
			case "matching":
				values := List(Map(snapshot.subject[field])["order"])
				for i, j := 0, len(values)-1; i < j; i, j = i+1, j-1 {
					values[i], values[j] = values[j], values[i]
				}
			case "fuzzy_match", "ai_semantic_match":
				Map(snapshot.subject["matching"])[field] = true
			case "index_policy_class":
				snapshot.catalog["policy_class"] = DeveloperClass
			case "subject_policy_class":
				snapshot.subject["policy_class"] = DeveloperClass
			}
			if failures := snapshot.failures(r); len(failures) == 0 {
				t.Fatal("neutral subject projection drift admitted", field)
			}
		})
	}
}

func TestNeutralClassRecognitionDoesNotBypassMaterializationSchema(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := r.Read(policyCatalogContracts)
	if err != nil {
		t.Fatal(err)
	}
	if !Has(Strings(Map(Map(contract["$defs"])["developer_policy_class"])["enum"]), "VPMS_DEVELOPER_POLICY") || Map(contract["application_gate"])["vpms_policy_materialization_authorized"] != true || Map(contract["application_execution"])["m1_m7_verified"] != true || Map(contract["application_execution"])["vpms_class_materialization_enabled"] != true {
		t.Fatal(contract)
	}
	candidate := Object{"schema_version": "ptsip-developer-policy/v1", "policy_class": "VPMS_DEVELOPER_POLICY", "policy": Object{"id": "MPD-VERI-0001", "version": "0.0", "status": "DRAFT", "title": "fixture"}, "rules": Object{"fixture": Object{"enabled": true}}}
	schema := Text(Map(contract["application_gate"])["existing_policy_materialization_schema_ref"])
	if err := r.Validate(schema, candidate); err == nil {
		t.Fatal("v1 class opening bypassed materialization schema")
	}
	candidate["schema_version"] = "developer-policy/v2"
	if err := r.Validate(schema, candidate); err != nil {
		t.Fatal(err)
	}
}

func TestNeutralCatalogPreservedContractsStayIndependentFromGoSelectors(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := r.Read(policyCatalogContracts)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(Strings(Map(contract["change_scope"])["preserved_contracts"]), []string{"developer/policy/schemas/developer-policy-index.schema.json", "developer/policy/schemas/developer-governance-registry.schema.json", "developer/policy/schemas/management-policy.schema.json"}) {
		t.Fatal(contract)
	}
	for _, raw := range List(Map(contract["change_scope"])["materialization_targets"]) {
		row := Map(raw)
		if strings.HasSuffix(Text(row["path"]), ".go") && row["python_functions"] != nil {
			t.Fatal("Go selector mislabeled as Python", row)
		}
	}
}
