package machine

import (
	"encoding/json"
	"os"
	"strings"
)

func (r *Repository) VerifyAPIImplementation(check bool) (Object, error) {
	activation, err := r.ActivationRecord()
	if err != nil {
		return nil, err
	}
	record, err := r.Read(implementationScopeRecord)
	if err != nil {
		return nil, err
	}
	if err := r.Validate("developer/policy/schemas/vpms-api-implementation.schema.json", record); err != nil {
		return nil, err
	}
	prior, err := r.ReadSource(Text(record["prior_materialization_ref"]))
	if err != nil {
		return nil, err
	}
	if LFDigest(prior) != record["prior_materialization_sha256"] {
		return nil, Fail("PRIOR_MATERIALIZATION_CHANGED", "prior materialization fingerprint differs")
	}
	if _, err := r.VerifyContractRegistration(false); err != nil {
		return nil, err
	}
	if err := r.VerifyPreservedFiles(List(record["preserved_files"]), activation); err != nil {
		return nil, err
	}
	retired := []string{}
	for _, raw := range List(record["subsequent_retirements"]) {
		retired = append(retired, Text(Map(raw)["path"]))
	}
	targets := Strings(record["mutation_targets"])
	for _, path := range retired {
		if !Has(targets, path) {
			return nil, Fail("RETIREMENT_OUTSIDE_ORIGINAL_SCOPE", path)
		}
		candidate, err := r.Path(path)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(candidate); !os.IsNotExist(err) {
			return nil, Fail("RETIRED_IMPLEMENTATION_TARGET_PRESENT", path)
		}
	}
	for _, path := range targets {
		if Has(retired, path) {
			continue
		}
		if err := r.VerifyAuditImplementationTarget(path); err != nil {
			return nil, Fail("MISSING_IMPLEMENTATION_TARGET", path)
		}
	}
	for _, path := range Strings(record["removed_uncommitted_paths"]) {
		candidate, err := r.Path(path)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(candidate); !os.IsNotExist(err) {
			return nil, Fail("RETIRED_WRITE_PATH_PRESENT", path)
		}
	}
	if record["support_policy_generation_root"] != "src/policy" {
		return nil, Fail("INVALID_SUPPORT_POLICY_WRITE_ROOT", "support generation root differs")
	}
	desired := "IMPLEMENTED_INACTIVE"
	if activation != nil {
		desired = "IMPLEMENTED_ACTIVE"
	}
	for _, raw := range List(record["bindings"]) {
		binding := Map(raw)
		file := map[string]string{"protocol": "protocol.json", "selection": "selection.json", "execution_composition": "execution-composition.json"}[Text(binding["contract"])]
		if file == "" {
			return nil, Fail("UNKNOWN_PRODUCT_CONTRACT_ROLE", Text(binding["contract"]))
		}
		payload, err := r.Read("src/vpms/contracts/" + file)
		if err != nil {
			return nil, err
		}
		matches := []Object{}
		for _, raw := range List(payload["planned_bindings"]) {
			item := Map(raw)
			if item["api"] == binding["api"] {
				matches = append(matches, item)
			}
		}
		if len(matches) != 1 || matches[0]["source"] != binding["source"] || matches[0]["availability"] != desired {
			return nil, Fail("IMPLEMENTATION_BINDING_MISMATCH", Text(binding["api"]))
		}
	}
	if check && (activation != nil || len(retired) > 0) {
		return nil, Fail("ORIGINAL_SCOPE_SUPERSEDED", "historical scope superseded by activation or retirement")
	}
	priorRecord := Object{}
	if err := json.Unmarshal(prior, &priorRecord); err != nil {
		return nil, err
	}
	expected := append(Strings(priorRecord["materialization_targets"]), targets...)
	for _, raw := range List(record["preserved_files"]) {
		path := Text(Map(raw)["path"])
		if strings.HasPrefix(path, "developer/") {
			expected = append(expected, path)
		}
	}
	if err := r.ExactAuditScope(record, expected, check); err != nil {
		return nil, err
	}
	return Object{"status": desired, "runtime_activation": activation != nil, "commit_push_authorized": true, "implementation_target_count": len(targets), "combined_changed_path_count": len(UniqueStrings(expected)), "preserved_file_count": len(List(record["preserved_files"])), "retired_target_count": len(retired)}, nil
}
