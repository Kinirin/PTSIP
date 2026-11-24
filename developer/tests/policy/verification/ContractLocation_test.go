package schema_check_test

// Checks only this repository's owner-selected contract storage route.
// It does not define a universal PTSIP contract directory requirement.
import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/contract/entry"
	verification "github.com/Kinirin/PTSIP/.ptsip/engine/developer/policy/verification"
)

func checkCanonicalContractFile(root, name string) error {
	if !strings.HasPrefix(name, ".ptsip/contracts/") ||
		strings.Contains(name, "\\") ||
		strings.Contains(name, "..") ||
		filepath.ToSlash(filepath.Clean(name)) != name {
		return fmt.Errorf("unapproved or unsafe contract reference %q", name)
	}
	base, err := filepath.EvalSymlinks(filepath.Join(root, ".ptsip", "contracts"))
	if err != nil {
		return fmt.Errorf("canonical contract root: %w", err)
	}
	target, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	relative, err := filepath.Rel(base, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("contract reference escapes owner-selected root: %s", name)
	}
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s: not a regular contract file", name)
	}
	return nil
}

func TestCurrentDeveloperContractOwnerJSON(t *testing.T) {
	root := routingRepositoryRoot(t)
	paths, findings := schemaFiles(root)
	if len(findings) != 0 {
		t.Fatalf("current contract owner scan failed: %v", findings)
	}
	for _, path := range paths {
		var document map[string]any
		readContractJSON(t, path, &document)
		if document == nil {
			t.Fatalf("expected a contract object: %s", path)
		}
	}
}

func TestCanonicalDeveloperContractRoutes(t *testing.T) {
	root := routingRepositoryRoot(t)
	old := filepath.Join(root, "developer", "contracts")
	if _, err := os.Stat(old); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy contract root remains active or inaccessible: %v", err)
	}

	inputs, err := verification.ResolveRoutingInputs(root)
	if err != nil {
		t.Fatal(err)
	}
	vocabularyRef := routingString(routingMembers(inputs.Data["identity_sources"])["root_family"])
	vocabulary := readRoutingJSON(t, root, vocabularyRef)
	families, err := entry.IdentityValues(vocabulary, "root_family")
	if err != nil {
		t.Fatal(err)
	}
	if len(families) != 14 {
		t.Fatalf("expected 14 root-family vocabulary values, got %d", len(families))
	}
	seen := make(map[string]bool, len(families))
	for _, value := range families {
		family := routingString(value)
		if family == "" || seen[family] {
			t.Fatalf("missing or duplicate Family vocabulary value: %v", value)
		}
		seen[family] = true
		for _, route := range []struct {
			name, kind, version, output string
		}{
			{"developer/unified", "PTSIP_DEVELOPER_POLICY", entry.UnifiedPolicyVersion, "developer/policy"},
			{"support/unified", "PTSIP_SUPPORT_FEATURE", entry.UnifiedPolicyVersion, "src/policy"},
			{"developer/v2", "PTSIP_DEVELOPER_POLICY", "", "developer/policy"},
		} {
			t.Run(family+"/"+route.name, func(t *testing.T) {
				document := map[string]any{"Policy_Identity": map[string]any{"policy_kind": route.kind, "root_family": family}}
				if route.version != "" {
					document["schema_version"] = route.version
				}
				selection, err := entry.SelectPolicyDocument(root, document)
				if err != nil {
					t.Fatal(err)
				}
				if selection.OutputRoot != route.output || selection.RootFamily != family || selection.PolicyKind != route.kind ||
					selection.Validator != "policy-check" || selection.ExecutionAuthorized || selection.MutationAuthorized || selection.PolicyConformance != "NOT_ASSERTED" {
					t.Fatalf("unexpected policy selection: %+v", selection)
				}
				for field, ref := range selection.ContractRefs {
					if field != "family_fields" {
						if err := checkCanonicalContractFile(root, ref); err != nil {
							t.Fatalf("invalid selected resource %s: %v", field, err)
						}
					}
				}
				if route.version != "" {
					if selection.SchemaIDs["public"] != entry.UnifiedPolicySchemaID || selection.StructuralState != "CANDIDATE" {
						t.Fatalf("unified candidate route changed: %+v", selection)
					}
				} else if selection.ContractRefs["family_fields"] != selection.SchemaIDs["rules"]+"#/$defs/FamilyProfiles/"+family {
					t.Fatalf("v2 Family field profile was not resolved: %+v", selection)
				}
			})
		}
	}
}

func TestCanonicalContractReferenceRejectsUnsafeOrMissing(t *testing.T) {
	root := t.TempDir()
	for _, ref := range []string{
		"developer/contracts/identity/root_family.json",
		".ptsip/contracts/../outside.json",
		".ptsip/contracts/missing.json",
		"/absolute/contract.json",
	} {
		if checkCanonicalContractFile(root, ref) == nil {
			t.Fatalf("must reject missing or noncanonical reference: %s", ref)
		}
	}
}
func TestPolicyContractResponsibilityMigration(t *testing.T) {
	root := routingRepositoryRoot(t)
	required := []string{
		".ptsip/contracts/policy/boundary.json",
		".ptsip/contracts/policy/Policy_Identity/Public_Policy_Identity.json",
		".ptsip/contracts/policy/routing/policy_routes.json",
		".ptsip/contracts/policy/routing/Policy_Routing.schema.json",
		".ptsip/contracts/policy/Policy_lifecycle/status/Policy_status.json",
		".ptsip/contracts/policy/Policy_lifecycle/version/Policy_version.json",
	}
	for _, name := range required {
		if err := checkCanonicalContractFile(root, name); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	obsolete := []string{
		".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_DRAFT.json",
		".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_APPROVED.json",
		".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_RETIRED.json",
		".ptsip/contracts/policy/identity/policy_kind.json",
		".ptsip/contracts/policy/identity/root_family.json",
		".ptsip/contracts/policy/lifecycle/Policy_version/Policy_version.schema.json",
		".ptsip/contracts/policy/Plans",
		".ptsip/contracts/identity/policy_kind.json",
		".ptsip/contracts/identity/root_family.json",
		".ptsip/contracts/routing/policy_routes.json",
		".ptsip/contracts/schemas/Policy_Identity/Routing/Policy_Routing.schema.json",
		".ptsip/contracts/schemas/Policy_Identity/Policy_Identity_version.schema.json",
		".ptsip/contracts/lifecycle/Policy_status/Policy_status_APPROVED.json",
		".ptsip/contracts/lifecycle/Policy_version/Policy_version.schema.json",
		".ptsip/contracts/lifecycle/policy_status.json",
		".ptsip/contracts/Authority/Authority_Major_Category.json",
	}
	for _, name := range obsolete {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(name))); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("obsolete policy contract path remains or is inaccessible: %s: %v", name, err)
		}
	}
}

func TestLegacyPolicyAuthorityModelRemoved(t *testing.T) {
	root := routingRepositoryRoot(t)
	for _, path := range []string{
		".ptsip/contracts/policy/authority",
		".ptsip/contracts/schemas/Authority",
	} {
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(path))); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("retired policy authority directory must not exist: %s: %v", path, err)
		}
	}
	for _, policy := range []string{
		"developer/policy/OPS/MPD-OPS-0002.yaml",
		"developer/policy/INFO/MPD-INFO-0006.yaml",
		"developer/policy/ARCH/MPD-ARCH-0004.yaml",
		"developer/policy/REAL/MPD-REAL-0007.yaml",
		"developer/policy/CNTR/MPD-CNTR-0006.yaml",
		"developer/policy/CTRL/MPD-CTRL-0002.yaml",
		"developer/policy/RISK/MPD-RISK-0002.yaml",
		"developer/policy/NORM/MPD-NORM-0002.yaml",
		"developer/policy/SUPPLY/MPD-SUPPLY-0002.yaml",
		"developer/policy/ASSURE/MPD-ASSURE-0005.yaml",
		"developer/policy/INTENT/MPD-INTENT-0002.yaml",
		"developer/policy/CHANGE/MPD-CHANGE-0005.yaml",
		"developer/policy/RECORD/MPD-RECORD-0001.yaml",
		"developer/policy/RECORD/MPD-RECORD-0002.yaml",
		"developer/policy/RECORD/MPD-RECORD-0003.yaml",
		"developer/policy/RECORD/MPD-RECORD-0004.yaml",
	} {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(policy)))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(content), "\n  Authority_Category:") {
			t.Fatalf("retired Authority_Category remains in %s", policy)
		}
	}
}

// Identity version generation delegates to the pattern contract without adding
// identity instance fields or changing the lifecycle owner's version grammar.
func TestIdentityVersionPatternDelegatesToLifecycleOwner(t *testing.T) {
	root := routingRepositoryRoot(t)
	const target = ".ptsip/contracts/policy/Policy_lifecycle/version/Policy_version.json"
	raw, err := os.ReadFile(filepath.Join(root, ".ptsip/engine/developer/policy/index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index struct {
		Components []struct {
			ID      string `json:"id"`
			Version string `json:"version_contract"`
		} `json:"components"`
	}
	if err := json.Unmarshal(raw, &index); err != nil {
		t.Fatal(err)
	}
	var binding string
	for _, component := range index.Components {
		if component.ID == "identity" {
			binding = component.Version
		}
	}
	if binding != target {
		t.Fatalf("identity version does not delegate to lifecycle: %q", binding)
	}
	if err := checkCanonicalContractFile(root, binding); err != nil {
		t.Fatal(err)
	}
}

func TestCanonicalPolicySchemaGroupAndSeparation(t *testing.T) {
	root := routingRepositoryRoot(t)
	tests := []struct{ current, id string }{
		{".ptsip/contracts/policy/PTSIP_Public_Policy_Structure.schema.json", "urn:ptsip:candidate:contract:policy:public-structure:v2"},
		{".ptsip/contracts/policy/Policy_Identity/PTSIP_Policy_Identity.schema.json", "urn:ptsip:policy:identity:v2"},
		{".ptsip/contracts/policy/Policy_Rules/Public_Policy_Rules.schema.json", "urn:ptsip:candidate:contract:policy:rules:v2"},
		{".ptsip/contracts/policy/Policy_Rules/Contract_Types.schema.json", "urn:ptsip:candidate:contract:policy:types:v2"},
		{".ptsip/contracts/policy/Policy_Rules/rule_record.schema.json", "urn:ptsip:policy:rule-record:schema:v1"},
		{".ptsip/contracts/policy/Policy_lifecycle/Public_Policy_Status.schema.json", "urn:ptsip:policy:Status:v2"},
	}
	for _, tc := range tests {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(tc.current)))
		if err != nil {
			t.Fatalf("current contract %s: %v", tc.current, err)
		}
		var content struct {
			ID string `json:"$id"`
		}
		if err := json.Unmarshal(data, &content); err != nil || content.ID != tc.id {
			t.Fatalf("schema ID changed at %s: %s, %v", tc.current, content.ID, err)
		}
	}
	for _, directory := range []string{
		".ptsip/contracts/schemas/release",
		".ptsip/contracts/compile-sealed-rbac",
	} {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(directory)))
		if err != nil || !info.IsDir() {
			t.Fatalf("independent schema directory must remain: %s: %v", directory, err)
		}
	}
}
