package machine

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

func (r *Repository) ScopeRecord() (Object, error) {
	record, err := r.Read(supportScopeRecord)
	if err != nil {
		return nil, err
	}
	if err := r.Validate("developer/policy/schemas/vpms-contract-materialization.schema.json", record); err != nil {
		return nil, err
	}
	return record, nil
}

func (r *Repository) VerifyContractRegistration(check bool) (Object, error) {
	activation, err := r.ActivationRecord()
	if err != nil {
		return nil, err
	}
	record, err := r.ScopeRecord()
	if err != nil {
		return nil, err
	}
	allocation := Map(record["allocation"])
	if allocation == nil {
		return nil, Fail("PREFLIGHT_ALLOCATION_REQUIRED", "allocation must already exist")
	}
	identity, err := r.Read("src/vpms/contracts/identity-registry.json")
	if err != nil {
		return nil, err
	}
	catalog, err := r.Read("src/vpms/contracts/index.json")
	if err != nil {
		return nil, err
	}
	expected := Map(allocation["product_contract_ids"])
	if !reflect.DeepEqual(expected, Map(catalog["entrypoints"])) || len(Map(catalog["contracts"])) != 3 {
		return nil, Fail("ALLOCATION_CATALOG_MISMATCH", "product entrypoints differ from the allocated identities")
	}
	seen := map[string]bool{}
	active := activation != nil
	desired := "APPROVED"
	if active {
		desired = "ACTIVE"
	}
	design := Map(record["approved_design"])
	if identity["contract_class"] != design["contract_class"] {
		return nil, Fail("UNAPPROVED_CONTRACT_CLASS", "contract class differs from approval")
	}
	for role, value := range expected {
		id := Text(value)
		if seen[id] {
			return nil, Fail("CONTRACT_ID_COLLISION", id)
		}
		seen[id] = true
		payload, err := r.ProductContract(id, active)
		if err != nil {
			return nil, err
		}
		if payload["status"] != desired || payload["runtime_enabled"] != active {
			return nil, Fail("UNAUTHORIZED_RUNTIME_ACTIVATION", id)
		}
		if payload["owner_approval"] != "USER_EXPLICIT" {
			return nil, Fail("OWNER_APPROVAL_REQUIRED", id)
		}
		for _, owned := range Strings(payload["owns"]) {
			if Has(Strings(payload["excludes"]), owned) {
				return nil, Fail("OWN_EXCLUDE_OVERLAP", owned)
			}
		}
		semantics := Map(payload["semantics"])
		if role == "protocol" {
			for _, field := range []string{"expanded_identity", "governing_question"} {
				if Map(semantics["identity"])[field] != design[field] {
					return nil, Fail("UNAPPROVED_PROTOCOL_IDENTITY", field)
				}
			}
		}
		if role == "selection" {
			for field, approved := range map[string]string{"request_kinds": "selection_request_kinds", "rule_kind": "selection_rule_kind", "ordering": "ordering", "invalid_request": "duplicate_unknown_empty", "failure_atomicity": "failure_atomicity", "execution_side_effects": "selection_execution_side_effects"} {
				if !reflect.DeepEqual(semantics[field], design[approved]) {
					return nil, Fail("UNAPPROVED_SELECTION_SEMANTICS", field)
				}
			}
		}
		if role == "execution_composition" && payload["responsibility"] != "VPMS_EXECUTION_COMPOSITION" {
			return nil, Fail("EXECUTION_COMPOSITION_OWNER_MISMATCH", id)
		}
	}
	invariants, err := r.Read("src/vpms/contracts/invariants.json")
	if err != nil {
		return nil, err
	}
	if invariants["activation_requires_explicit_owner_approval"] != true {
		return nil, Fail("ACTIVATION_GATE_MISSING", "explicit owner activation invariant is missing")
	}
	root, _ := r.Path("src/vpms/contracts")
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(data, []byte("MPD-")) || bytes.Contains(data, []byte("developer/")) {
			return Fail("PRODUCT_DEVELOPER_POLICY_DEPENDENCY", path)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	runtimeSurface, err := r.SupportField("integration_runtime_surface")
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(Strings(runtimeSurface), []string{"src/vpms/integration/ptsip_bridge.py"}) {
		return nil, Fail("INTEGRATION_SCOPE_EXPANDED", "support runtime surface differs from approved boundary")
	}
	if err := r.VerifyPreservedFiles(List(record["preserved_files"]), activation); err != nil {
		return nil, err
	}
	expectedPaths := Strings(record["materialization_targets"])
	for _, relative := range expectedPaths {
		if err := r.VerifyAuditImplementationTarget(relative); err != nil {
			return nil, Fail("MISSING_SCOPE_TARGET", relative)
		}
	}
	if check && active {
		return nil, Fail("ORIGINAL_SCOPE_SUPERSEDED_BY_ACTIVATION", "original worktree scope is no longer applicable")
	}
	for _, raw := range List(record["preserved_files"]) {
		path := Text(Map(raw)["path"])
		if strings.HasPrefix(path, "developer/") {
			expectedPaths = append(expectedPaths, path)
		}
	}
	if err := r.ExactAuditScope(record, expectedPaths, check); err != nil {
		return nil, err
	}
	supportStatus := "DRAFT"
	status := "REGISTERED_NON_ACTIVE"
	if active {
		supportStatus = "ACTIVE"
		status = "REGISTERED_ACTIVE"
	}
	return Object{"status": status, "support_policy_id": allocation["support_policy_id"], "support_status": supportStatus, "product_contract_count": 3, "product_status": desired, "runtime_enabled": active, "scope_target_count": len(List(record["materialization_targets"])), "preserved_file_count": len(List(record["preserved_files"]))}, nil
}

func (r *Repository) SupportRegistrationPreflight() (Object, error) {
	record, err := r.ScopeRecord()
	if err != nil {
		return nil, err
	}
	if allocation := Map(record["allocation"]); allocation != nil {
		result := Object{"status": "ALLOCATED"}
		for key, value := range allocation {
			result[key] = value
		}
		return result, nil
	}
	return nil, Fail("ROOT_FAMILY_REGISTRATION_REQUIRED", "generic SFP numeric allocation has retired; use explicit class + Root Family entry and lifecycle admission")
}
