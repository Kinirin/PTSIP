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
	source := filepath.Join(repoRoot(t), ".ptsip/contracts/release")
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(root, ".ptsip/contracts/release", relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, raw, 0644)
	})
	if err != nil {
		t.Fatal(err)
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
			file := filepath.Join(root, ".ptsip/contracts/release/Platform/Common/verification-boundary.json")
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
	for _, path := range []string{"index.json", "Platform/Common/verification-boundary.json", "Platform/Common/release_validation.json"} {
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
