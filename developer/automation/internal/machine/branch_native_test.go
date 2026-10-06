package machine

import (
	"errors"
	"testing"
)

func branchGuardTestRepo(t *testing.T) *Repository {
	t.Helper()
	repo, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

func branchGuardErrorCode(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		t.Fatal("expected branch guard error")
	}
	var operation *OperationError
	if !errors.As(err, &operation) {
		t.Fatalf("expected OperationError, got %T: %v", err, err)
	}
	return operation.Code
}

func TestBranchGuardExactApprovedDevelopmentBranch(t *testing.T) {
	repo := branchGuardTestRepo(t)
	for _, branch := range []string{"dev/0.4.0", "dev/0.10.0", "dev/12.3.45", "dev/123.456.789"} {
		result, err := repo.ValidateBranchCreation(
			branch,
			branch,
			"USER_EXPLICIT",
			"DEVELOPMENT_VERSION_BRANCH",
			"GITHUB_CREATE_BRANCH_API",
		)
		if err != nil {
			t.Fatalf("%s rejected: %v", branch, err)
		}
		if result["status"] != "AUTHORIZED" ||
			result["branch_class"] != "DEVELOPMENT_VERSION" ||
			result["creation_mechanism"] != "GITHUB_CREATE_BRANCH_API" {
			t.Fatalf("%s returned unexpected decision: %#v", branch, result)
		}
	}
}

func TestBranchGuardFailsClosedForUnauthorizedInputs(t *testing.T) {
	repo := branchGuardTestRepo(t)
	cases := []struct {
		name          string
		candidate     string
		approved      string
		authorization string
		request       string
		mechanism     string
		code          string
	}{
		{"substituted-name", "verify/context-plane-20260925", "dev/0.4.0", "USER_EXPLICIT", "DEVELOPMENT_VERSION_BRANCH", "GITHUB_CREATE_BRANCH_API", "BRANCH_NAME_NOT_EXACTLY_APPROVED"},
		{"inferred-authorization", "dev/0.4.0", "dev/0.4.0", "AGENT_INFERRED", "DEVELOPMENT_VERSION_BRANCH", "GITHUB_CREATE_BRANCH_API", "BRANCH_CREATION_REQUIRES_USER_EXPLICIT"},
		{"unregistered-shape", "fix/js-ts-dependency-resolution", "fix/js-ts-dependency-resolution", "USER_EXPLICIT", "DEVELOPMENT_VERSION_BRANCH", "GITHUB_CREATE_BRANCH_API", "UNAUTHORIZED_BRANCH_NAME"},
		{"unregistered-request", "dev/0.4.1", "dev/0.4.1", "USER_EXPLICIT", "UNREGISTERED", "GITHUB_CREATE_BRANCH_API", "UNREGISTERED_BRANCH_REQUEST_KIND"},
		{"missing-approved-name", "dev/0.4.1", "", "USER_EXPLICIT", "DEVELOPMENT_VERSION_BRANCH", "GITHUB_CREATE_BRANCH_API", "APPROVED_BRANCH_NAME_REQUIRED"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := repo.ValidateBranchCreation(
				test.candidate,
				test.approved,
				test.authorization,
				test.request,
				test.mechanism,
			)
			if code := branchGuardErrorCode(t, err); code != test.code {
				t.Fatalf("code=%s want=%s err=%v", code, test.code, err)
			}
		})
	}

	for _, mechanism := range []string{
		"GIT_PUSH_BRANCH_CREATION",
		"GIT_SWITCH_CREATE",
		"GIT_CHECKOUT_CREATE",
		"GIT_UPDATE_REF",
		"GITHUB_UPDATE_REF_API",
		"WORKFLOW_REF_CREATION",
	} {
		t.Run("mechanism/"+mechanism, func(t *testing.T) {
			_, err := repo.ValidateBranchCreation(
				"dev/0.4.1",
				"dev/0.4.1",
				"USER_EXPLICIT",
				"DEVELOPMENT_VERSION_BRANCH",
				mechanism,
			)
			if code := branchGuardErrorCode(t, err); code != "UNAUTHORIZED_BRANCH_CREATION_MECHANISM" {
				t.Fatalf("code=%s err=%v", code, err)
			}
		})
	}
}

func TestBranchGuardProjectProfileTransitionKeepsBranchIdentity(t *testing.T) {
	repo := branchGuardTestRepo(t)
	result, err := repo.BranchProfileTransition("dev/0.4.0", "pp.1.01", "pp.1.02")
	if err != nil {
		t.Fatal(err)
	}
	if result["status"] != "NO_BRANCH_IDENTITY_CHANGE" ||
		result["branch_name"] != "dev/0.4.0" ||
		result["branch_change_required"] != false ||
		result["branch_creation_authorized"] != false {
		t.Fatalf("unexpected transition result: %#v", result)
	}

	if _, err := repo.BranchProfileTransition("feature/example", "pp.1.01", "pp.1.02"); branchGuardErrorCode(t, err) != "INVALID_DEVELOPMENT_BRANCH" {
		t.Fatalf("unexpected invalid branch error: %v", err)
	}
	if _, err := repo.BranchProfileTransition("dev/0.4.0", "invalid", "pp.1.02"); branchGuardErrorCode(t, err) != "INVALID_PROJECT_PROFILE_IDENTITY" {
		t.Fatalf("unexpected invalid profile error: %v", err)
	}
}

func TestBranchGuardClassifiesExistingBranches(t *testing.T) {
	repo := branchGuardTestRepo(t)
	cases := map[string]string{
		"dev/12.3.45":                       "AUTHORIZED_DEVELOPMENT_VERSION",
		"tool-0.3.4-authority-consistency":  "GRANDFATHERED_RETENTION",
		"feature/unregistered-branch-shape": "UNREGISTERED_SHAPE",
	}
	for branch, expected := range cases {
		result, err := repo.ClassifyBranch(branch)
		if err != nil {
			t.Fatalf("%s: %v", branch, err)
		}
		if result["classification"] != expected {
			t.Fatalf("%s classification=%v want=%s", branch, result["classification"], expected)
		}
	}

	if _, err := repo.ValidateBranchCreation(
		"tool-0.3.4-authority-consistency",
		"tool-0.3.4-authority-consistency",
		"USER_EXPLICIT",
		"DEVELOPMENT_VERSION_BRANCH",
		"GITHUB_CREATE_BRANCH_API",
	); branchGuardErrorCode(t, err) != "UNAUTHORIZED_BRANCH_NAME" {
		t.Fatalf("legacy branch gained creation authority: %v", err)
	}
}
