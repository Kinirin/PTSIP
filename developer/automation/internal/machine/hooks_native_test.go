package machine

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestH3NativeInstallerSetsOnlyRepositoryLocalCanonicalHooksPath(t *testing.T) {
	root := t.TempDir()
	command := exec.Command("git", "-C", root, "init", "-q")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("fixture init: %v\n%s", err, output)
	}
	if err := os.MkdirAll(filepath.Join(root, ".githooks"), 0755); err != nil {
		t.Fatal(err)
	}
	hook := filepath.Join(root, ".githooks", "pre-commit")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	repo := &Repository{Root: root}
	result, err := repo.InstallHooks()
	if err != nil || result["status"] != "PASS" || result["pre_commit"] != hook {
		t.Fatalf("hook activation: %#v, error=%v", result, err)
	}
	actual, err := repo.GitOutput("config", "--local", "--get", "core.hooksPath")
	if err != nil || actual != ".githooks" {
		t.Fatalf("local hooksPath=%s error=%v", actual, err)
	}
}
