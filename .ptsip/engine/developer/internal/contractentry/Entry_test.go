package contractentry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	put := func(p string, data any) {
		t.Helper()
		file := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
			t.Fatal(err)
		}
		var raw []byte
		if s, ok := data.(string); ok {
			raw = []byte(s)
		} else {
			var err error
			raw, err = json.Marshal(data)
			if err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(file, raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
	put(".ptsip/index.json", map[string]any{"namespaces": map[string]any{"contracts": map[string]any{"status": "ACTIVE", "index": "contracts/index.json"}}})
	put(".ptsip/contracts/index.json", map[string]any{"policy": "policy/boundary.json", "release_documentation": "release/Platform/Common/releasenote.json", "authoring": map[string]any{"entrypoint": ".ptsip/engine/developer", "validators": []string{"policy-rules", "responsibility-authority"}, "execution_authorized": false}})
	put(".ptsip/engine/developer/index.json", map[string]any{"policy_authority": "NONE", "execution_authorized": false, "validators": map[string]any{"contract-entry": "policy/index.json", "policy-check": "policy/index.json"}})
	put(".ptsip/contracts/policy/boundary.json", map[string]any{"domain": "policy", "routing_binding": map[string]any{"data": ".ptsip/contracts/policy/routing/policy_routes.json"}, "rules_binding": map[string]any{"family_source": ".ptsip/contracts/policy/Policy_Identity/Public_Policy_Identity.json", "directory": ".ptsip/contracts/policy/Policy_Rules", "selection_schema": ".ptsip/contracts/policy/Policy_Rules/Public_Policy_Rules.schema.json", "responsibility_profile": ".ptsip/profiles/Responsibility-Authority-list.yaml"}})
	put(".ptsip/contracts/policy/routing/policy_routes.json", map[string]any{"identity_sources": map[string]any{"policy_kind": ".ptsip/contracts/policy/Policy_Identity/Public_Policy_Identity.json", "root_family": ".ptsip/contracts/policy/Policy_Identity/Public_Policy_Identity.json"}, "routes": map[string]any{"PTSIP_DEVELOPER_POLICY": map[string]any{"output_root": "developer/policy"}}, "schema_refs": map[string]any{"public": ".ptsip/contracts/policy/PTSIP_Public_Policy_Structure.schema.json", "rules": ".ptsip/contracts/policy/Policy_Rules/Public_Policy_Rules.schema.json"}})
	put(".ptsip/contracts/policy/Policy_Identity/Public_Policy_Identity.json", map[string]any{"$id": "urn:ptsip:identity:policy:v1", "$defs": map[string]any{"policy_kind": map[string]any{"type": "string", "enum": []string{"PTSIP_DEVELOPER_POLICY"}}, "root_family": map[string]any{"type": "string", "enum": []string{"GOV"}}}})
	put(".ptsip/contracts/policy/candidate.index.json", map[string]any{"active_registration": true, "active_policy_cutover": "STRUCTURE_V2_ACTIVE", "schema_refs": map[string]any{"public": "urn:ptsip:candidate:contract:policy:public-structure:v2", "rules": "urn:ptsip:candidate:contract:policy:rules:v2"}, "family_selection_ref": ".ptsip/contracts/policy/Policy_Rules/Family_Selection.json"})
	put(".ptsip/contracts/policy/Policy_Rules/Family_Selection.json", map[string]any{"active": true, "selection_basis": "DECLARED_POLICY_IDENTITY_ROOT_FAMILY", "family_routes": map[string]any{"GOV": map[string]any{"rules_schema_ref": "urn:ptsip:candidate:contract:policy:rules:v2", "document_fields_ref": "urn:ptsip:candidate:contract:policy:rules:v2#/$defs/FamilyProfiles/GOV"}}})
	put(".ptsip/contracts/policy/PTSIP_Public_Policy_Structure.schema.json", map[string]any{"$id": "urn:ptsip:candidate:contract:policy:public-structure:v2"})
	put(".ptsip/contracts/policy/Policy_Rules/Public_Policy_Rules.schema.json", map[string]any{"$id": "urn:ptsip:candidate:contract:policy:rules:v2"})
	put(".ptsip/profiles/Responsibility-Authority-list.yaml", "responsibility: []\n")
	put("developer/policy/GOV/MPD-GOV-0006.yaml", "Policy_Identity:\n  policy_kind: PTSIP_DEVELOPER_POLICY\n  root_family: GOV\nPolicy_Lifecycle:\n  status: APPROVED.Applying\n")
	return root
}

func TestPolicySelectionIgnoresUnrelatedMissingReleaseContract(t *testing.T) {
	root := fixture(t)
	got, err := Resolve(root, "developer/policy/GOV/MPD-GOV-0006.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if got.Validator != "policy-check" || got.RootFamily != "GOV" || got.ExecutionAuthorized || got.MutationAuthorized {
		t.Fatalf("invalid selection: %+v", got)
	}
	if got.ContractRefs["family"] != ".ptsip/contracts/policy/Policy_Rules/Family_Selection.json" {
		t.Fatalf("wrong family ref: %+v", got)
	}
}

func TestMissingSelectedContractFailsClosed(t *testing.T) {
	root := fixture(t)
	if err := os.Remove(filepath.Join(root, ".ptsip/contracts/policy/Policy_Rules/Family_Selection.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(root, "developer/policy/GOV/MPD-GOV-0006.yaml"); err == nil || !strings.Contains(err.Error(), "MISSING_REGISTERED_FILE") {
		t.Fatalf("missing selected contract accepted: %v", err)
	}
}

func TestUnknownAndMismatchedIdentityFailClosed(t *testing.T) {
	for _, example := range []string{
		"Policy_Identity:\n  policy_kind: PTSIP_DEVELOPER_POLICY\n  root_family: UNKNOWN\n",
		"Policy_Identity:\n  policy_kind: PTSIP_SUPPORT_FEATURE\n  root_family: GOV\n",
		"Policy_Identity:\n  policy_kind: PTSIP_DEVELOPER_POLICY\n  root_family: GOV\n  root_family: GOV\n",
		"Policy_Identity:\n  root_family: GOV\n",
		"Policy_Identity: {policy_kind: PTSIP_DEVELOPER_POLICY, root_family: GOV}\n",
	} {
		root := fixture(t)
		if err := os.WriteFile(filepath.Join(root, "developer/policy/GOV/MPD-GOV-0006.yaml"), []byte(example), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := Resolve(root, "developer/policy/GOV/MPD-GOV-0006.yaml"); err == nil {
			t.Errorf("accepted invalid selector: %q", example)
		}
	}
}

func TestWrongDomainAndTraversalFailClosed(t *testing.T) {
	root := fixture(t)
	for _, target := range []string{"../outside.yaml", "developer/policy/GOV/../../GOV/MPD-GOV-0006.yaml", "src/policy/ARCH/SFP-ARCH-0005.yaml"} {
		if _, err := Resolve(root, target); err == nil {
			t.Fatalf("accepted unregistered target: %s", target)
		}
	}
}

func TestAuthorityEscalationFailsClosed(t *testing.T) {
	root := fixture(t)
	filename := filepath.Join(root, filepath.FromSlash(".ptsip/engine/developer/index.json"))
	if err := os.WriteFile(filename, []byte(`{"policy_authority":"NONE","execution_authorized":true,"validators":{"contract-entry":"policy/index.json","policy-check":"policy/index.json"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(root, "developer/policy/GOV/MPD-GOV-0006.yaml"); err == nil || !strings.Contains(err.Error(), "AUTHORITY_BOUNDARY_CHANGED") {
		t.Fatalf("accepted authority escalation: %v", err)
	}
}

func TestInactiveV2RegistryFailsClosed(t *testing.T) {
	root := fixture(t)
	p := filepath.Join(root, filepath.FromSlash(".ptsip/contracts/policy/candidate.index.json"))
	if err := os.WriteFile(p, []byte(`{"active_registration":false,"active_policy_cutover":"DEFERRED"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(root, "developer/policy/GOV/MPD-GOV-0006.yaml"); err == nil || !strings.Contains(err.Error(), "NOT_ACTIVE") {
		t.Fatalf("inactive registry selected: %v", err)
	}
}

func TestUnifiedIdentitySourceFailsClosed(t *testing.T) {
	const relative = ".ptsip/contracts/policy/Policy_Identity/Public_Policy_Identity.json"
	for _, raw := range []string{
		`{"$id":"urn:ptsip:identity:policy:v1","$defs":{"policy_kind":{"type":"string","enum":["PTSIP_DEVELOPER_POLICY"]}}}`,
		`{"$id":"urn:ptsip:identity:policy:v1","$defs":{"policy_kind":{"type":"string","enum":["PTSIP_DEVELOPER_POLICY","PTSIP_DEVELOPER_POLICY"]},"root_family":{"type":"string","enum":["GOV"]}}}`,
		`{"$id":"urn:ptsip:unknown","$defs":{"policy_kind":{"type":"string","enum":["PTSIP_DEVELOPER_POLICY"]},"root_family":{"type":"string","enum":["GOV"]}}}`,
	} {
		root := fixture(t)
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(relative)), []byte(raw), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := Resolve(root, "developer/policy/GOV/MPD-GOV-0006.yaml"); err == nil {
			t.Fatal("invalid unified vocabulary accepted")
		}
	}
	root := fixture(t)
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(relative))); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(root, "developer/policy/GOV/MPD-GOV-0006.yaml"); err == nil {
		t.Fatal("missing unified vocabulary fell back to another file")
	}
}
