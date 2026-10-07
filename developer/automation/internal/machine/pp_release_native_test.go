package machine

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPPReleaseAssetIdentityUsesGitNormalizationAndRejectsChangedContent(t *testing.T) {
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		command := exec.Command("git", append([]string{"-C", root}, args...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	git("init", "-q")
	git("config", "core.autocrlf", "true")
	git("config", "user.name", "Fixture")
	git("config", "user.email", "fixture@example.invalid")
	path := filepath.Join(root, "asset.yaml")
	if err := os.WriteFile(path, []byte("version: pp.1.02\n"), 0644); err != nil {
		t.Fatal(err)
	}
	git("add", "asset.yaml")
	git("-c", "core.hooksPath=", "commit", "-qm", "fixture")
	repo := &Repository{Root: root}
	if err := os.WriteFile(path, []byte("version: pp.1.02\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if matches, err := repo.ppAssetMatchesCommit("asset.yaml", "HEAD"); err != nil || !matches {
		t.Fatalf("equivalent checkout bytes rejected: matches=%v, error=%v", matches, err)
	}
	if err := os.WriteFile(path, []byte("version: pp.1.03\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if matches, err := repo.ppAssetMatchesCommit("asset.yaml", "HEAD"); err != nil || matches {
		t.Fatalf("changed asset accepted: matches=%v, error=%v", matches, err)
	}
}
