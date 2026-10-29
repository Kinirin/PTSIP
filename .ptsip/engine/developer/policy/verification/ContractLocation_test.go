package schema_check

// Checks only this repository's owner-selected contract storage route.
// It does not define a universal PTSIP contract directory requirement.
import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestCanonicalDeveloperContractRoutes(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(root, "developer", "contracts")
	if _, err := os.Stat(old); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy contract root remains active or inaccessible: %v", err)
	}

	canonical := filepath.Join(root, ".ptsip", "contracts")
	count := 0
	err = filepath.WalkDir(canonical, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			return nil
		}
		bytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !json.Valid(bytes) {
			return fmt.Errorf("invalid contract JSON: %s", path)
		}
		count++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count < 57 {
		t.Fatalf("contract migration coverage incomplete: %d JSON files (expected at least 55 migrated plus 2 release)", count)
	}

	routingPath := ".ptsip/contracts/policy/routing/policy_routes.json"
	if err := checkCanonicalContractFile(root, routingPath); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(routingPath)))
	if err != nil {
		t.Fatal(err)
	}
	var routing struct {
		IdentitySources map[string]string `json:"identity_sources"`
		SchemaRefs      map[string]string `json:"schema_refs"`
		Routes          map[string]struct {
			OutputRoot string `json:"output_root"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(raw, &routing); err != nil {
		t.Fatal(err)
	}
	if routing.Routes["PTSIP_DEVELOPER_POLICY"].OutputRoot != "developer/policy" {
		t.Fatal("developer output route missing")
	}

	for _, source := range []string{"policy_kind", "root_family"} {
		if err := checkCanonicalContractFile(root, routing.IdentitySources[source]); err != nil {
			t.Fatal(err)
		}
	}
	rootFamilyPath := routing.IdentitySources["root_family"]
	vocabBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rootFamilyPath)))
	if err != nil {
		t.Fatal(err)
	}
	var vocab struct {
		Enum []string `json:"enum"`
	}
	if err := json.Unmarshal(vocabBytes, &vocab); err != nil {
		t.Fatal(err)
	}
	if len(vocab.Enum) != 14 {
		t.Fatalf("expected 14 root-family vocabulary values, got %d", len(vocab.Enum))
	}
	for _, target := range routing.SchemaRefs {
		if err := checkCanonicalContractFile(root, target); err != nil {
			t.Fatal(err)
		}
	}
	for _, family := range vocab.Enum {
		target := ".ptsip/contracts/policy/Rules/" + family + "_rules.json"
		if err := checkCanonicalContractFile(root, target); err != nil {
			t.Fatal(err)
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
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	required := []string{
		".ptsip/contracts/policy/boundary.json",
		".ptsip/contracts/policy/identity/policy_kind.json",
		".ptsip/contracts/policy/identity/root_family.json",
		".ptsip/contracts/policy/routing/policy_routes.json",
		".ptsip/contracts/policy/routing/Policy_Routing.schema.json",
		".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_DRAFT.json",
		".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_APPROVED.json",
		".ptsip/contracts/policy/lifecycle/Policy_status/Policy_status_RETIRED.json",
		".ptsip/contracts/policy/lifecycle/Policy_version/Policy_version.schema.json",
		".ptsip/contracts/policy/lifecycle/Plan_status/Plan_status_ACTIVATE.json",
		".ptsip/contracts/policy/lifecycle/Plan_status/Plan_status_DRAFT.json",
		".ptsip/contracts/policy/lifecycle/Plan_status/Plan_status_INTERRUPTION.json",
	}
	for _, name := range required {
		if err := checkCanonicalContractFile(root, name); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	obsolete := []string{
		".ptsip/contracts/identity/policy_kind.json",
		".ptsip/contracts/identity/root_family.json",
		".ptsip/contracts/routing/policy_routes.json",
		".ptsip/contracts/schemas/Policy_Identity/Routing/Policy_Routing.schema.json",
		".ptsip/contracts/schemas/Policy_Identity/Policy_Identity_version.schema.json",
		".ptsip/contracts/lifecycle/Policy_status/Policy_status_APPROVED.json",
		".ptsip/contracts/lifecycle/Policy_version/Policy_version.schema.json",
		".ptsip/contracts/lifecycle/policy_status.json",
		".ptsip/contracts/lifecycle/Plan_status/Plan_status_ACTIVATE.json",
		".ptsip/contracts/lifecycle/Plan_status/Plan_status_DRAFT.json",
		".ptsip/contracts/lifecycle/Plan_status/Plan_status_INTERRUPTION.json",
		".ptsip/contracts/Authority/Authority_Major_Category.json",
	}
	for _, name := range obsolete {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(name))); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("obsolete policy contract path remains or is inaccessible: %s: %v", name, err)
		}
	}
}

func TestPlanStatusContractVocabulary(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		filename string
		status   string
		id       string
	}{
		{"Plan_status_ACTIVATE.json", "ACTIVATE", "urn:ptsip:contract:constant:plan_status_activate:v1"},
		{"Plan_status_DRAFT.json", "DRAFT", "urn:ptsip:contract:constant:plan_status_draft:v1"},
		{"Plan_status_INTERRUPTION.json", "INTERRUPTION", "urn:ptsip:contract:constant:plan_status_interruption:v1"},
	} {
		relative := ".ptsip/contracts/policy/lifecycle/Plan_status/" + tc.filename
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		var schema struct {
			ID         string   `json:"$id"`
			Required   []string `json:"required"`
			Properties struct {
				Status struct {
					Const string `json:"const"`
				} `json:"status"`
			} `json:"properties"`
			AdditionalProperties bool `json:"additionalProperties"`
		}
		if err := json.Unmarshal(raw, &schema); err != nil {
			t.Fatal(err)
		}
		if schema.ID != tc.id || len(schema.Required) != 1 || schema.Required[0] != "status" ||
			schema.Properties.Status.Const != tc.status || schema.AdditionalProperties {
			t.Fatalf("%s: Plan status semantics changed during relocation", relative)
		}
	}
}

func TestLegacyPolicyAuthorityModelRemoved(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
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
		"developer/policy/RECORD/MPD-RECORD-0002.yaml",
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

// Identity version storage delegates to the lifecycle contract without adding
// identity instance fields or changing the lifecycle owner's version grammar.
func TestIdentityVersionStorageDelegatesToLifecycleOwner(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	const target = ".ptsip/contracts/policy/lifecycle/Policy_version/Policy_version.schema.json"
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
 root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
 if err != nil { t.Fatal(err) }
 tests := []struct{ previous, current, id string }{
  {".ptsip/contracts/schemas/PTSIP_Public_Structure.schema.json", ".ptsip/contracts/schemas/Policy/PTSIP_Public_Structure.schema.json", "urn:ptsip:policy:public-structure:v7"},
  {".ptsip/contracts/schemas/Policy_Identity/PTSIP_Policy_Identity.schema.json", ".ptsip/contracts/schemas/Policy/Policy_Identity/PTSIP_Policy_Identity.schema.json", "urn:ptsip:policy:identity:v2"},
  {".ptsip/contracts/schemas/Policy_Rules/Family_Rules_Contract.schema.json", ".ptsip/contracts/schemas/Policy/Policy_Rules/Family_Rules_Contract.schema.json", "urn:ptsip:policy:family-rules-contract:v2"},
  {".ptsip/contracts/schemas/Policy_Rules/Public_Structure_Rules.schema.json", ".ptsip/contracts/schemas/Policy/Policy_Rules/Public_Structure_Rules.schema.json", "urn:ptsip:policy:Public_Structure_Rules:v3"},
  {".ptsip/contracts/schemas/Policy_Rules/rule_record.schema.json", ".ptsip/contracts/schemas/Policy/Policy_Rules/rule_record.schema.json", "urn:ptsip:policy:rule-record:schema:v1"},
  {".ptsip/contracts/schemas/Policy_Status/Public_Policy_Status.schema.json", ".ptsip/contracts/schemas/Policy/Policy_Status/Public_Policy_Status.schema.json", "urn:ptsip:policy:Status:v2"},
 }
 for _, tc := range tests {
  if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(tc.previous))); !errors.Is(err, os.ErrNotExist) {
   t.Fatalf("old schema path must not exist: %s: %v", tc.previous, err)
  }
  data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(tc.current)))
  if err != nil { t.Fatalf("relocated schema %s: %v", tc.current, err) }
  var content struct { ID string `json:"$id"` }
  if err := json.Unmarshal(data, &content); err != nil || content.ID != tc.id {
   t.Fatalf("schema ID changed at %s: %s, %v", tc.current, content.ID, err)
  }
 }
 for _, directory := range []string{
  ".ptsip/contracts/schemas/release",
  ".ptsip/contracts/schemas/compile-sealed-rbac",
 } {
  info, err := os.Stat(filepath.Join(root, filepath.FromSlash(directory)))
  if err != nil || !info.IsDir() {t.Fatalf("independent schema directory must remain: %s: %v", directory, err)}
 }
}
