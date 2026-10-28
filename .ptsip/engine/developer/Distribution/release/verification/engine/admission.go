package enginewheel

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"reflect"
)

const VerificationContract = "developer/policy/contracts/engine-verification.v1.json"

func policySection(root, id, section string) (map[string]any, error) {
	cmd := exec.Command("go", "-C", filepath.Join(root, "developer/automation"), "run", "-tags", "grammar_subset,grammar_subset_python", "./cmd/ptsip-dev", "policy-resolver", "get", id, "--section", section)
	data, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("canonical verification policy resolution failed: %w", err)
	}
	var output struct {
		PolicyID string         `json:"policy_id"`
		Record   map[string]any `json:"record"`
	}
	if err = json.Unmarshal(data, &output); err != nil {
		return nil, err
	}
	if output.PolicyID != id {
		return nil, fmt.Errorf("unexpected policy owner")
	}
	return output.Record, nil
}

func Admit(root string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	ctrl, err := policySection(root, "MPD-CTRL-0001", "release_execution_suspension")
	if err != nil {
		return err
	}
	scope, ok := ctrl["engine_verification_only"].(map[string]any)
	if !ok || ctrl["state"] != "BLOCKED" || ctrl["direct_release_commands"] != "BLOCKED" || ctrl["workflow_execution"] != "ENGINE_VERIFICATION_AND_ADMITTED_ENGINE_PUBLICATION" || scope["state"] != "ALLOWED" || scope["operation"] != "VERIFY" || scope["contract_ref"] != VerificationContract || scope["release_gate"] != "ENABLED" || scope["release_gate_scope"] != "SELECTED_ENGINE_VERIFICATION" || scope["tag_release_and_pypi_publication"] != "NOT_GRANTED_BY_VERIFICATION" || scope["engine_pass_is_release_authorization"] != false {
		return fmt.Errorf("engine verification exception not admitted")
	}
	assure, err := policySection(root, "MPD-ASSURE-0001", "engine_verification_only")
	if err != nil {
		return err
	}
	if assure["contract_ref"] != VerificationContract || assure["result_scope"] != "ENGINE_VERIFICATION_ONLY" || assure["engine_pass_is_release_gate_pass"] != true || assure["release_gate_scope"] != "SELECTED_ENGINE_VERIFICATION" || assure["full_release_authorization_from_engine_gate"] != false {
		return fmt.Errorf("verification owner contract mismatch")
	}
	var contract map[string]any
	if err = document(root, VerificationContract, &contract); err != nil {
		return err
	}
	expected := []any{}
	for _, kind := range kinds {
		expected = append(expected, kind)
	}
	if contract["schema_version"] != "ptsip-engine-verification/v1" || contract["operation"] != "VERIFY" || contract["release_gate"] != "ENABLED" || contract["release_gate_scope"] != "SELECTED_ENGINE_VERIFICATION" || contract["publication_authority"] != "NOT_GRANTED_BY_VERIFICATION" || contract["engine_pass_is_release_authorization"] != false || contract["legacy_verification_fallback"] != false || !reflect.DeepEqual(contract["engines"], expected) || !reflect.DeepEqual(contract["platforms"], []any{"windows-x64", "linux-x64", "macos-arm64"}) || !reflect.DeepEqual(contract["required_checks"], assure["required_checks"]) {
		return fmt.Errorf("unregistered or expanded engine verification contract")
	}
	if contract["approval_ref"] != scope["approval_ref"] || contract["approval_ref"] != assure["approval_ref"] || !reflect.DeepEqual(contract["gate_required_checks"], []any{"CANONICAL_ADMISSION", "ALL_SELECTED_PLATFORM_JOBS_SUCCESS", "ONE_UNEXPIRED_ARTIFACT_PER_SELECTED_ENGINE_PLATFORM", "EXACT_WORKFLOW_RUN_SOURCE_SHA"}) {
		return fmt.Errorf("gate approval or required evidence contract mismatch")
	}
	cache, ok := contract["rust_cache"].(map[string]any)
	policyCache, policyOK := assure["rust_build_cache"].(map[string]any)
	if !ok || !policyOK || cache["role"] != "NON_AUTHORITATIVE_BUILD_ACCELERATION" || cache["partition"] != policyCache["partition"] || cache["source_change_restore"] != policyCache["source_change_restore"] || cache["cached_result_is_verification_evidence"] != false || policyCache["cached_result_is_verification_evidence"] != false || cache["rerun_locked_test_build_and_artifact_verification"] != true || policyCache["rerun_locked_test_build_and_artifact_verification"] != "REQUIRED" {
		return fmt.Errorf("compilation cache must not replace verification")
	}
	return nil
}
