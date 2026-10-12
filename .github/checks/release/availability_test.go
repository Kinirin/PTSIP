package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseCommandsFailBeforeArtifactsOrPublicationWhenGapIsOpen(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte("[project]\nname='release-gap-test'\n"), 0644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, ".github", "checks", "release")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	state := `{"state":"BLOCKED","verification_contract":null,"legacy_fallback":false,"reason":"user-approved release gap"}`
	if err := os.WriteFile(filepath.Join(dir, "release-availability.json"), []byte(state), 0644); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"plan", "selection", "contracts", "build", "verify", "publish", "package", "sync-versions"} {
		if _, err := CLI([]string{command, "--root", root}); err == nil || !strings.Contains(err.Error(), "RELEASE_EXECUTION_BLOCKED") {
			t.Fatalf("%s did not reject release execution: %v", command, err)
		}
	}
	for _, path := range []string{".artifacts", "dist", "src"} {
		if _, err := os.Stat(filepath.Join(root, path)); !os.IsNotExist(err) {
			t.Fatalf("blocked execution created output %s: %v", path, err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "release-availability.json"), []byte(`{"state":"ENABLED"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := CLI([]string{"publish", "--root", root}); err == nil || !strings.Contains(err.Error(), "RELEASE_EXECUTION_BLOCKED") {
		t.Fatalf("editing a state projection reenabled publication: %v", err)
	}
}

func TestMissingReleaseStateCannotFallBackToLegacyVerification(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte("[project]\nname='release-gap-test'\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := CLI([]string{"publish", "--root", root}); err == nil || !strings.Contains(err.Error(), "RELEASE_EXECUTION_BLOCKED") {
		t.Fatalf("missing state admitted release execution: %v", err)
	}
}
