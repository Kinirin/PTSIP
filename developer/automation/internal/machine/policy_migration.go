package machine

import (
	policybinding "github.com/Kinirin/PTSIP/developer/automation/policy/binding"
	"reflect"
	"sort"
	"strings"
)

// LegacyAnalysisProjection is an audit comparison only. It does not resolve a
// policy or reconstruct authority from a former policy identity.
func LegacyAnalysisProjection(payload Object) (Object, error) {
	result := policyClone(payload)
	analysis := Map(result["analysis"])
	decision := Map(analysis["decision"])
	if analysis == nil || decision == nil {
		return nil, policyFailure("INVALID_RESPONSIBILITY_ANALYSIS", "analysis and decision are required")
	}
	result["schema_version"] = "ptsip-policy-responsibility-analysis/v1"
	for _, raw := range List(analysis["responsibilities"]) {
		unit := Map(raw)
		delete(unit, "policy_class")
		delete(unit, "referenced_policy_class")
		if lookup := Map(unit["existing_authority_lookup"]); lookup != nil {
			delete(lookup, "searched_policy_class")
		}
	}
	keys := List(decision["owned_authority_family_set"])
	delete(decision, "owned_authority_family_set")
	families := []any{}
	for _, raw := range keys {
		families = append(families, Map(raw)["family"])
	}
	decision["owned_family_set"] = families
	for _, raw := range List(decision["materialization_groups"]) {
		delete(Map(raw), "policy_class")
	}
	return result, nil
}
func PolicySemanticDigest(payload any) (string, error) {
	data, err := CanonicalJSON(payload)
	if err != nil {
		return "", err
	}
	return SHA256(data), nil
}
func BackfillLegacyAnalysis(original Object) (Object, error) {
	if original["schema_version"] != "ptsip-policy-responsibility-analysis/v1" {
		return nil, policyFailure("BACKFILL_REQUIRES_EXACT_LEGACY_INPUT", "unsupported analysis schema")
	}
	result := policyClone(original)
	analysis := Map(result["analysis"])
	decision := Map(analysis["decision"])
	if analysis == nil || decision == nil {
		return nil, policyFailure("INVALID_RESPONSIBILITY_ANALYSIS", "analysis and decision required")
	}
	result["schema_version"] = "developer-policy-responsibility-analysis/v2"
	for _, raw := range List(analysis["responsibilities"]) {
		unit := Map(raw)
		unit["policy_class"] = nil
		if unit["authority_relation"] == "OWN" {
			unit["policy_class"] = DeveloperClass
		}
		unit["referenced_policy_class"] = nil
		if unit["referenced_family"] != nil {
			unit["referenced_policy_class"] = DeveloperClass
		}
		if lookup := Map(unit["existing_authority_lookup"]); lookup != nil {
			lookup["searched_policy_class"] = DeveloperClass
		}
	}
	keys := []any{}
	for _, family := range List(decision["owned_family_set"]) {
		keys = append(keys, Object{"policy_class": DeveloperClass, "family": family})
	}
	delete(decision, "owned_family_set")
	decision["owned_authority_family_set"] = keys
	for _, raw := range List(decision["materialization_groups"]) {
		Map(raw)["policy_class"] = DeveloperClass
	}
	projected, err := LegacyAnalysisProjection(result)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(projected, original) {
		return nil, policyFailure("BACKFILL_CHANGED_PRIOR_MEANING", "identity backfill changed the prior semantic record")
	}
	return result, nil
}

func (r *Repository) MigrateAuthorityFamilyCatalog(apply bool) (Object, error) {
	contract, err := r.Read(policyCatalogContracts)
	if err != nil {
		return nil, err
	}
	gate := Map(contract["application_gate"])
	if gate["catalog_payload_migration_authorized"] != true || gate["m2_m8_implementation_authorized"] != true {
		return nil, policyFailure("MIGRATION_NOT_AUTHORIZED", "explicit catalog migration gate is closed")
	}
	index, err := r.Read(policyIndex)
	if err != nil {
		return nil, err
	}
	subject, err := r.Read(policySubjectRegistry)
	if err != nil {
		return nil, err
	}
	if index["schema_version"] != "ptsip-developer-policy-index/v1" {
		return nil, policyFailure("MIGRATION_REQUIRES_EXACT_LEGACY_CATALOG_INPUT", "current catalog already uses explicit class-aware identity")
	}
	records := map[string]Object{}
	for _, raw := range List(index["policies"]) {
		entry := Map(raw)
		id := Text(entry["id"])
		path := Text(entry["path"])
		// An obsolete source identity cannot be revived by this catalog migration.
		if !rootID.MatchString(id) || strings.Contains(path, "/legacy/") {
			return nil, policyFailure("LEGACY_POLICY_ID_RUNTIME_FORBIDDEN", id)
		}
		expected, err := policyCanonicalPath(id)
		if err != nil {
			return nil, err
		}
		if path != expected {
			return nil, policyFailure("POLICY_FILE_PATH_MISMATCH", path)
		}
		record, err := r.Read(path)
		if err != nil {
			return nil, err
		}
		if Map(record["policy"])["id"] != id || record["policy_class"] != DeveloperClass {
			return nil, policyFailure("AUTHORITY_METADATA_MISMATCH", id)
		}
		records[id] = record
	}
	catalog := policyClone(index)
	delete(catalog, "policy_class")
	catalogID := Text(Map(contract["entrypoints"])["index"])
	catalog["schema_version"] = catalogID
	catalog["artifact_class"] = Map(Map(contract["contracts"])[catalogID])["artifact_class"]
	for _, raw := range List(catalog["policies"]) {
		entry := Map(raw)
		entry["policy_class"] = records[Text(entry["id"])]["policy_class"]
	}
	migratedSubject := policyClone(subject)
	delete(migratedSubject, "policy_class")
	subjectID := Text(Map(contract["entrypoints"])["subject"])
	migratedSubject["schema_version"] = subjectID
	migratedSubject["artifact_class"] = Map(Map(contract["contracts"])[subjectID])["artifact_class"]
	if errors := r.ValidateNeutralCatalogSnapshot(catalog, migratedSubject, records, index, subject); len(errors) > 0 {
		return nil, policyFailure("INVALID_MIGRATION_SNAPSHOT", strings.Join(errors, "; "))
	}
	registry, err := r.Read(policyAnalysisRegistry)
	if err != nil {
		return nil, err
	}
	if registry["schema_version"] != "developer-policy-materialization-analysis-registry/v3" {
		return nil, policyFailure(
			"LEGACY_ANALYSIS_REGISTRY_REQUIRES_EXPLICIT_PRA_MIGRATION",
			"catalog migration must not infer PRA identity or path from a legacy analysis registry",
		)
	}
	if err := r.Validate(policyAnalysisRegistrySchema, registry); err != nil {
		return nil, err
	}
	updates := map[string]Object{policyIndex: catalog, policySubjectRegistry: migratedSubject}
	digests := Object{}

	allowed := policyStrings(Map(contract["application_execution"])["targets"])
	paths := []string{}
	for path := range updates {
		if !policyContains(allowed, path) {
			return nil, policyFailure("UNREGISTERED_MIGRATION_WRITE_TARGET", path)
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	if apply {
		if err := r.policyWriteTransaction(updates); err != nil {
			return nil, err
		}
	}
	status := "READY"
	if apply {
		status = "APPLIED"
	}
	return Object{"status": status, "paths": paths, "preserved_analysis_semantic_digests": digests}, nil
}

func (r *Repository) VerifyPolicyPlanConsistency() (Object, error) {
	return policybinding.NewStore(r).VerifyConsistency()
}
func policyErrorCode(err error, fallback string) string {
	if typed, ok := err.(*PolicyError); ok {
		return typed.Code
	}
	if typed, ok := err.(*OperationError); ok {
		return typed.Code
	}
	return fallback
}
