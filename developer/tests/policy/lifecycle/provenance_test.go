package lifecycle_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func TestCurrentPolicyCorpusHasNoLegacyADRSourceProvenance(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	seen := map[string]bool{}
	for _, plane := range []string{"src/policy", "developer/policy"} {
		index, err := repo.Read(plane + "/index.yaml")
		if err != nil {
			t.Fatal(err)
		}
		for _, raw := range index["policies"].([]any) {
			entry := raw.(object)
			if entry["authority_role"] == "MIGRATION_SOURCE" {
				continue
			}
			ref := entry["path"].(string)
			if plane == "src/policy" {
				ref = plane + "/" + ref
			}
			if seen[ref] {
				t.Fatal("duplicate policy path", ref)
			}
			seen[ref] = true
			payload, err := repo.Read(ref)
			if err != nil {
				t.Fatal(err)
			}
			if _, exists := payload["source_provenance"]; exists {
				t.Fatal("legacy provenance", ref)
			}
			path, err := repo.Path(ref)
			if err != nil {
				t.Fatal(err)
			}
			text, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(text), "source_type: LEGACY_ADR") || strings.Contains(string(text), "source_path: decisions/ADR-") {
				t.Fatal("legacy ADR source in current policy", ref)
			}
		}
	}
	if len(seen) == 0 {
		t.Fatal("empty canonical policy inventory")
	}
}

func TestCurrentPolicySchemasExcludeLegacySourceProvenance(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	for _, ref := range []string{"src/policy/schemas/ptsip-support-root-family-policy.schema.json", "developer/policy/schemas/root-family-policy.schema.json", "developer/policy/schemas/management-policy.schema.json"} {
		schema, err := repo.Read(ref)
		if err != nil {
			t.Fatal(err)
		}
		if _, exists := schema["properties"].(object)["source_provenance"]; exists {
			t.Fatal("legacy provenance schema field", ref)
		}
		if ref == "developer/policy/schemas/management-policy.schema.json" {
			if schema["additionalProperties"] != false {
				t.Fatal(schema)
			}
		} else if schema["additionalProperties"] != false && !reflect.DeepEqual(schema["not"], object{"required": []any{"source_provenance"}}) {
			t.Fatal("legacy provenance is not excluded by the current schema", ref)
		}
	}
}

func TestRetiredLegacyPolicyMaterializerIsNotUsedByCurrentValidator(t *testing.T) {
	repo := testrepo.Open(testrepo.Root(t))
	path, err := repo.Path("developer/automation/policy_materializer.py")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("legacy materializer still exists", err)
	}
	for _, ref := range []string{"developer/automation/internal/machine/policy_validator.go", "developer/automation/policy/lifecycle/transition.go"} {
		path, err := repo.Path(ref)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"policy_materializer", "legacy-decisions-inventory.yaml", "policy-relation-migration.yaml", "decision_reference_migrator"} {
			if strings.Contains(string(raw), forbidden) {
				t.Fatal("retired validator dependency", ref, forbidden)
			}
		}
	}
}
