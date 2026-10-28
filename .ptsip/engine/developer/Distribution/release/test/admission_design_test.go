package releasetest

import (
	"encoding/json"
	release "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release"
	"os"
	"path/filepath"
	"testing"
)

func releaseBoundaryFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, path := range []string{".ptsip/contracts/release/index.json", ".ptsip/contracts/release/verification-boundary.json", ".ptsip/contracts/release/validation-rules.json"} {
		raw, err := os.ReadFile(filepath.Join(repoRoot(t), path))
		if err != nil {
			t.Fatal(err)
		}
		file := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
func TestReleaseStructureCheckHasNoLegacyApprovalDependency(t *testing.T) {
	root := releaseBoundaryFixture(t)
	report, code := release.RunAdmission(root)
	if code != 0 || report.(map[string]any)["status"] != "CONTRACT_STRUCTURE_VALID" || report.(map[string]any)["publication_authorized"] != false {
		t.Fatalf("structural validation failed: %v", report)
	}
}
func TestReleaseStructureRejectsMissingAndForgedContracts(t *testing.T) {
	for _, field := range []string{"publication_authorized", "runtime_activation_authorized", "legacy_fallback"} {
		t.Run(field, func(t *testing.T) {
			root := releaseBoundaryFixture(t)
			file := filepath.Join(root, ".ptsip/contracts/release/verification-boundary.json")
			raw, _ := os.ReadFile(file)
			var data map[string]any
			if err := json.Unmarshal(raw, &data); err != nil {
				t.Fatal(err)
			}
			data[field] = true
			raw, _ = json.Marshal(data)
			if err := os.WriteFile(file, raw, 0644); err != nil {
				t.Fatal(err)
			}
			if _, code := release.RunAdmission(root); code == 0 {
				t.Fatalf("forged %s accepted", field)
			}
		})
	}
	for _, path := range []string{"index.json", "verification-boundary.json", "validation-rules.json"} {
		t.Run(path, func(t *testing.T) {
			root := releaseBoundaryFixture(t)
			if err := os.Remove(filepath.Join(root, ".ptsip/contracts/release", path)); err != nil {
				t.Fatal(err)
			}
			if _, code := release.RunAdmission(root); code == 0 {
				t.Fatalf("missing %s accepted", path)
			}
		})
	}
}
