package pp

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func ppRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "developer", "automation", "go.mod")); err != nil {
		t.Fatalf("repository root not resolved: %v", err)
	}
	return root
}

func ppGit(t *testing.T, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", ppRepositoryRoot(t)}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func ppBuildAutomation(t *testing.T) string {
	t.Helper()
	root := ppRepositoryRoot(t)
	name := filepath.Join(t.TempDir(), "ptsip-dev")
	if os.PathSeparator == '\\' {
		name += ".exe"
	}
	command := exec.Command("go", "-C", filepath.Join(root, "developer", "automation"), "build", "-o", name, "./cmd/ptsip-dev")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("build ptsip-dev: %v\n%s", err, output)
	}
	return name
}

func ppAutomation(t *testing.T, binary string, args ...string) (map[string]any, error, string) {
	t.Helper()
	full := append([]string{"--repository", ppRepositoryRoot(t)}, args...)
	command := exec.Command(binary, full...)
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, err, string(output)
	}
	var payload map[string]any
	if decodeErr := json.Unmarshal(output, &payload); decodeErr != nil {
		t.Fatalf("decode ptsip-dev output: %v\n%s", decodeErr, output)
	}
	return payload, nil, string(output)
}

func TestPPReleaseUsesExactHeadAndDoesNotMutateWorktree(t *testing.T) {
	binary := ppBuildAutomation(t)
	expected := ppGit(t, "rev-parse", "--verify", "HEAD^{commit}")
	before := ppGit(t, "status", "--porcelain=v1", "--untracked-files=all")

	result, err, output := ppAutomation(t, binary, "pp", "release", "--sha", "HEAD")
	if err != nil {
		t.Fatalf("pp release HEAD: %v\n%s", err, output)
	}
	if result["status"] != "PASS" {
		t.Fatalf("release status = %#v", result["status"])
	}
	if result["source_sha"] != expected {
		t.Fatalf("source_sha = %#v, want %s", result["source_sha"], expected)
	}
	if result["project_profile"] == "" || result["schema"] == "" {
		t.Fatalf("release result omitted Project Profile identity: %#v", result)
	}

	after := ppGit(t, "status", "--porcelain=v1", "--untracked-files=all")
	if after != before {
		t.Fatalf("verify-only release command mutated worktree\nbefore:\n%s\nafter:\n%s", before, after)
	}

	parent := ppGit(t, "rev-parse", "--verify", "HEAD^")
	if _, err, output := ppAutomation(t, binary, "pp", "release", "--sha", parent); err == nil || !strings.Contains(output, "RELEASE_EXACT_SHA_MISMATCH") {
		t.Fatalf("non-HEAD release SHA was not rejected: err=%v\n%s", err, output)
	}
}

func TestPPCommitAndRangeNormalizeExactCommit(t *testing.T) {
	binary := ppBuildAutomation(t)
	head := ppGit(t, "rev-parse", "--verify", "HEAD^{commit}")
	parent := ppGit(t, "rev-parse", "--verify", "HEAD^")

	commitResult, err, output := ppAutomation(t, binary, "pp", "commit", "--commit", "HEAD")
	if err != nil {
		t.Fatalf("pp commit HEAD: %v\n%s", err, output)
	}
	if commitResult["status"] != "PASS" || commitResult["verified_commit_count"] != float64(1) {
		t.Fatalf("unexpected commit verification result: %#v", commitResult)
	}
	commits, ok := commitResult["commits"].([]any)
	if !ok || len(commits) != 1 {
		t.Fatalf("commit result does not contain exactly one record: %#v", commitResult)
	}
	record, ok := commits[0].(map[string]any)
	if !ok {
		t.Fatalf("commit record is not an object: %#v", commits[0])
	}
	if record["commit"] != head || len(record["commit"].(string)) != 40 {
		t.Fatalf("symbolic HEAD was not normalized to exact SHA: %#v", record["commit"])
	}
	allowed := map[string]bool{
		"NO_T2_AUTHORITY_DELTA":       true,
		"T2_AUTHORITY_DELTA":          true,
		"MERGE_INHERITED_PP_AUTHORITY": true,
	}
	if !allowed[record["classification"].(string)] {
		t.Fatalf("unexpected PP classification: %#v", record["classification"])
	}

	rangeResult, err, output := ppAutomation(t, binary, "pp", "range", "--base", parent, "--head", head)
	if err != nil {
		t.Fatalf("pp range HEAD^..HEAD: %v\n%s", err, output)
	}
	if rangeResult["status"] != "PASS" || rangeResult["verified_commit_count"] != float64(1) {
		t.Fatalf("single-commit range did not verify exactly one commit: %#v", rangeResult)
	}
}
