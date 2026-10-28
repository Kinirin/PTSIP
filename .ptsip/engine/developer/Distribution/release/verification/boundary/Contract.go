// Package boundary owns the temporary developer-only release verification
// boundary. Structure checks never admit publication or runtime activation.
package boundary

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"

	authoring "github.com/Kinirin/PTSIP/.ptsip/engine/developer/internal/contract_authoring"
)

const Path = ".ptsip/contracts/release/verification-boundary.json"

type Contract struct {
	Schema      string   `json:"schema_version"`
	Lifecycle   string   `json:"lifecycle"`
	Replacement bool     `json:"replacement_required"`
	Authority   string   `json:"policy_authority"`
	Admission   string   `json:"publication_admission"`
	Publication *bool    `json:"publication_authorized"`
	Activation  *bool    `json:"runtime_activation_authorized"`
	Legacy      *bool    `json:"legacy_fallback"`
	Waivers     []string `json:"verification_waivers"`
	Operations  []string `json:"operations"`
	Source      string   `json:"source_sha"`
	Digest      string   `json:"artifact_digest"`
	Engines     []string `json:"engine_inventory"`
	Scope       string   `json:"scope"`
}

func Load(root string) (Contract, error) {
	var c Contract
	file, err := authoring.LocalFile(root, Path)
	if err != nil {
		return c, err
	}
	handle, err := os.Open(file)
	if err != nil {
		return c, err
	}
	defer handle.Close()
	decoder := json.NewDecoder(handle)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&c); err != nil {
		return c, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return c, fmt.Errorf("RELEASE_VERIFICATION_BOUNDARY_TRAILING_JSON")
	}
	if c.Schema != "ptsip-release-verification-boundary/v1" || c.Lifecycle != "TEMPORARY_REPLACEABLE" || !c.Replacement || c.Authority != "NONE" || c.Admission != "UNRESOLVED" ||
		c.Publication == nil || *c.Publication || c.Activation == nil || *c.Activation || c.Legacy == nil || *c.Legacy || c.Waivers == nil || len(c.Waivers) != 0 ||
		c.Source != "EXACT_CHECKED_OUT_HEAD" || c.Digest != "REQUIRED_SHA256" || c.Scope != "LOCAL_STRUCTURE_AND_VERIFICATION_ARTIFACTS_ONLY" ||
		!slices.Equal(c.Operations, []string{"RELEASE_STRUCTURE_CHECK", "ENGINE_LOCAL_VERIFICATION", "SOURCE_BOUND_TEST_EVIDENCE"}) ||
		!slices.Equal(c.Engines, []string{"context", "discovery", "execution", "planning", "projection", "storage", "validation"}) {
		return c, fmt.Errorf("RELEASE_VERIFICATION_BOUNDARY_INVALID: temporary verification cannot grant publication, activation or waivers")
	}
	return c, nil
}

func CheckOperation(root, operation string) error {
	c, err := Load(root)
	if err != nil {
		return err
	}
	for _, registered := range c.Operations {
		if operation == registered {
			return nil
		}
	}
	return fmt.Errorf("UNREGISTERED_RELEASE_VERIFICATION_OPERATION: %s", operation)
}

func RequirePublication(root string) error {
	if _, err := Load(root); err != nil {
		return fmt.Errorf("RELEASE_PUBLICATION_BLOCKED: %w", err)
	}
	return fmt.Errorf("RELEASE_PUBLICATION_BLOCKED: SUCCESSOR_AUTHORITY_UNRESOLVED; verification results do not grant publication")
}

func Run(root string) (any, int) {
	report := map[string]any{"status": "FAIL", "contract_ref": Path, "scope": "RELEASE_VERIFICATION_CONTRACT_STRUCTURE", "publication_authorized": false, "release_verification": "NOT_ASSERTED", "publication_admission": "UNVERIFIED"}
	if err := CheckOperation(root, "RELEASE_STRUCTURE_CHECK"); err != nil {
		report["problem"] = err.Error()
		return report, 2
	}
	if err := checkRouting(root); err != nil {
		report["problem"] = err.Error()
		return report, 2
	}
	report["status"] = "CONTRACT_STRUCTURE_VALID"
	return report, 0
}

func checkRouting(root string) error {
	read := func(relative string, target any) error {
		file, err := authoring.LocalFile(root, relative)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		return json.Unmarshal(raw, target)
	}
	var index struct {
		Schema      string `json:"schema_version"`
		Contract    string `json:"contract_ref"`
		Validator   string `json:"validator_name"`
		Entry       string `json:"engine_entry"`
		Root        string `json:"verifier_root"`
		Authority   string `json:"contract_authority"`
		Activation  string `json:"normative_policy_activation"`
		Publication *bool  `json:"publication_authorized"`
		Legacy      *bool  `json:"legacy_release_verifier_fallback"`
		Rules       string `json:"validation_rules_ref"`
	}
	if err := read(".ptsip/contracts/release/index.json", &index); err != nil {
		return err
	}
	if index.Schema != "ptsip-release-contract-index/v2" || index.Contract != Path || index.Authority != "NONE" || index.Activation != "UNRESOLVED" ||
		index.Validator != "release-admission-check" || index.Entry != ".ptsip/engine/developer/main.go" || index.Root != ".ptsip/engine/developer/Distribution/release/" ||
		index.Publication == nil || *index.Publication || index.Legacy == nil || *index.Legacy || index.Rules != ".ptsip/contracts/release/validation-rules.json" {
		return fmt.Errorf("RELEASE_INDEX_ROUTING_INVALID")
	}
	var rules struct {
		Schema      string `json:"schema_version"`
		Boundary    string `json:"boundary_ref"`
		Scope       string `json:"scope"`
		Publication *bool  `json:"publication_authorized"`
		Retired     *bool  `json:"retired_approval_lookup"`
	}
	if err := read(index.Rules, &rules); err != nil {
		return err
	}
	if rules.Schema != "ptsip-release-structural-rules/v2" || rules.Boundary != Path || rules.Scope != "STRUCTURE_AND_VERIFICATION_ONLY" || rules.Publication == nil || *rules.Publication || rules.Retired == nil || *rules.Retired {
		return fmt.Errorf("RELEASE_STRUCTURAL_RULES_INVALID")
	}
	return nil
}
