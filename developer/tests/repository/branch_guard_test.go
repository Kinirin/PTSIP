package repository

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func branchRepositoryRoot(t *testing.T) string {
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

func branchBuildAutomation(t *testing.T) string {
	t.Helper()
	root := branchRepositoryRoot(t)
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

func branchAutomation(t *testing.T, binary string, args ...string) (map[string]any, error, string) {
	t.Helper()
	full := append([]string{"--repository", branchRepositoryRoot(t)}, args...)
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

func TestBranchGuardAdmitsOnlyExactApprovedDevelopmentIdentity(t *testing.T) {
	binary := branchBuildAutomation(t)
	for _, branch := range []string{"dev/0.4.0", "dev/0.10.0", "dev/123.456.789"} {
		result, err, output := branchAutomation(
			t,
			binary,
			"branch-guard", "validate",
			"--candidate", branch,
			"--approved-name", branch,
			"--authorization-source", "USER_EXPLICIT",
			"--request-kind", "DEVELOPMENT_VERSION_BRANCH",
			"--creation-mechanism", "GITHUB_CREATE_BRANCH_API",
		)
		if err != nil {
			t.Fatalf("%s rejected: %v\n%s", branch, err, output)
		}
		if result["status"] != "AUTHORIZED" || result["branch_class"] != "DEVELOPMENT_VERSION" {
			t.Fatalf("unexpected branch authorization: %#v", result)
		}
	}

	_, err, output := branchAutomation(
		t,
		binary,
		"branch-guard", "validate",
		"--candidate", "verify/context-plane-20260925",
		"--approved-name", "dev/0.4.0",
		"--authorization-source", "USER_EXPLICIT",
		"--request-kind", "DEVELOPMENT_VERSION_BRANCH",
		"--creation-mechanism", "GITHUB_CREATE_BRANCH_API",
	)
	if err == nil || !strings.Contains(output, "BRANCH_NAME_NOT_EXACTLY_APPROVED") {
		t.Fatalf("invented branch identity was not rejected: err=%v\n%s", err, output)
	}
}

func TestBranchGuardFailsClosedOnAuthorityAndMechanism(t *testing.T) {
	binary := branchBuildAutomation(t)
	cases := []struct {
		name      string
		authority string
		mechanism string
		code      string
	}{
		{"non-user authority", "AGENT_INFERRED", "GITHUB_CREATE_BRANCH_API", "BRANCH_CREATION_REQUIRES_USER_EXPLICIT"},
		{"git push creation", "USER_EXPLICIT", "GIT_PUSH_BRANCH_CREATION", "UNAUTHORIZED_BRANCH_CREATION_MECHANISM"},
		{"git switch creation", "USER_EXPLICIT", "GIT_SWITCH_CREATE", "UNAUTHORIZED_BRANCH_CREATION_MECHANISM"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err, output := branchAutomation(
				t,
				binary,
				"branch-guard", "validate",
				"--candidate", "dev/0.4.1",
				"--approved-name", "dev/0.4.1",
				"--authorization-source", test.authority,
				"--request-kind", "DEVELOPMENT_VERSION_BRANCH",
				"--creation-mechanism", test.mechanism,
			)
			if err == nil || !strings.Contains(output, test.code) {
				t.Fatalf("fail-closed code %s missing: err=%v\n%s", test.code, err, output)
			}
		})
	}
}

func TestBranchGuardKeepsProjectProfileAndLegacyRetentionSeparate(t *testing.T) {
	binary := branchBuildAutomation(t)
	transition, err, output := branchAutomation(
		t,
		binary,
		"branch-guard", "profile-transition",
		"--branch", "dev/0.4.0",
		"--from-profile", "pp.1.01",
		"--to-profile", "pp.1.02",
	)
	if err != nil {
		t.Fatalf("profile transition: %v\n%s", err, output)
	}
	if transition["branch_change_required"] != false || transition["branch_creation_authorized"] != false || transition["branch_name"] != "dev/0.4.0" {
		t.Fatalf("Project Profile transition changed branch identity: %#v", transition)
	}

	classified, err, output := branchAutomation(
		t,
		binary,
		"branch-guard", "classify-existing",
		"--branch", "tool-0.3.4-authority-consistency",
	)
	if err != nil {
		t.Fatalf("classify legacy branch: %v\n%s", err, output)
	}
	if classified["classification"] != "GRANDFATHERED_RETENTION" {
		t.Fatalf("legacy branch classification = %#v", classified["classification"])
	}
}
