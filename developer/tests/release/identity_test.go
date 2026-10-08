package release_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func runReleaseCommand(t *testing.T, binary, root string, env map[string]string, args ...string) (testrepo.Object, error, string) {
	t.Helper()
	command := exec.Command(binary, append([]string{"--repository", root}, args...)...)
	command.Env = os.Environ()
	for key, value := range env {
		prefix := key + "="
		filtered := command.Env[:0]
		for _, entry := range command.Env {
			if !strings.HasPrefix(entry, prefix) {
				filtered = append(filtered, entry)
			}
		}
		command.Env = append(filtered, prefix+value)
	}
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, err, string(output)
	}
	var payload testrepo.Object
	if err := json.Unmarshal(output, &payload); err != nil {
		t.Fatalf("decode release command output: %v\n%s", err, output)
	}
	return payload, nil, string(output)
}

func releaseGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func writeReleaseFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func newReleaseFixture(t *testing.T, version string) (string, string, string) {
	t.Helper()
	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	root := filepath.Join(base, "work")

	if output, err := exec.Command("git", "init", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatalf("init bare remote: %v\n%s", err, output)
	}
	if output, err := exec.Command("git", "init", "-b", "main", root).CombinedOutput(); err != nil {
		t.Fatalf("init release fixture: %v\n%s", err, output)
	}
	releaseGit(t, root, "config", "user.email", "release-test@example.invalid")
	releaseGit(t, root, "config", "user.name", "PTSIP Release Test")

	writeReleaseFile(t, root, "pyproject.toml", "[project]\nname = \"fixture\"\nversion = \""+version+"\"\n")
	writeReleaseFile(t, root, "docs/releasenote/tool/"+version+".md", "# Fixture\n\n## Changes\n\n- release fixture\n")
	testrepo.CopyTree(t, testrepo.Open(root), "developer/policy")
	releaseGit(t, root, "add", ".")
	releaseGit(t, root, "commit", "-m", "fixture")
	releaseGit(t, root, "remote", "add", "origin", remote)
	releaseGit(t, root, "push", "-u", "origin", "main")
	return root, remote, releaseGit(t, root, "rev-parse", "HEAD")
}

func TestReleaseIdentityDerivesVersionTagNoteAndExactSource(t *testing.T) {
	binary := testrepo.BuildCLI(t)
	root, _, head := newReleaseFixture(t, "1.2.3a1")

	result, err, output := runReleaseCommand(
		t,
		binary,
		root,
		nil,
		"release", "prepare",
		"--dispatched-sha", head,
		"--dispatched-ref", "refs/heads/main",
	)
	if err != nil {
		t.Fatalf("release prepare: %v\n%s", err, output)
	}
	if result["status"] != "PASS" ||
		result["version"] != "1.2.3a1" ||
		result["tag"] != "tool-v1.2.3a1" ||
		result["note"] != "docs/releasenote/tool/1.2.3a1.md" ||
		result["source_sha"] != head {
		t.Fatalf("unexpected release identity: %#v", result)
	}
}

func TestReleaseIdentityRejectsNonMainDispatchAndUncategorizedNote(t *testing.T) {
	binary := testrepo.BuildCLI(t)
	root, _, head := newReleaseFixture(t, "1.2.3a1")

	_, err, output := runReleaseCommand(
		t,
		binary,
		root,
		nil,
		"release", "prepare",
		"--dispatched-sha", head,
		"--dispatched-ref", "refs/heads/dev/fixture",
	)
	if err == nil || !strings.Contains(output, "RELEASE_NOT_MAIN") {
		t.Fatalf("non-main dispatch was not rejected: %v\n%s", err, output)
	}

	writeReleaseFile(t, root, "docs/releasenote/tool/1.2.3a1.md", "# Fixture\n\nuncategorized\n")
	_, err, output = runReleaseCommand(
		t,
		binary,
		root,
		nil,
		"release", "prepare",
		"--dispatched-sha", head,
		"--dispatched-ref", "refs/heads/main",
	)
	if err == nil || !strings.Contains(output, "RELEASE_NOTE_UNCATEGORIZED") {
		t.Fatalf("uncategorized release note was not rejected: %v\n%s", err, output)
	}
}

func TestPublishedTagMustMatchPackageVersionAndHead(t *testing.T) {
	binary := testrepo.BuildCLI(t)
	root, _, _ := newReleaseFixture(t, "1.2.3a1")
	releaseGit(t, root, "tag", "tool-v1.2.3a1")

	result, err, output := runReleaseCommand(
		t,
		binary,
		root,
		nil,
		"release", "tag-verify",
		"--release-tag", "tool-v1.2.3a1",
	)
	if err != nil || result["status"] != "PASS" {
		t.Fatalf("matching release tag rejected: %v\n%s\n%#v", err, output, result)
	}

	_, err, output = runReleaseCommand(
		t,
		binary,
		root,
		nil,
		"release", "tag-verify",
		"--release-tag", "tool-v9.9.9",
	)
	if err == nil || !strings.Contains(output, "RELEASE_TAG_VERSION_MISMATCH") {
		t.Fatalf("mismatched release tag was not rejected: %v\n%s", err, output)
	}
}
