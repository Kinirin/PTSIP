package branch

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func moduleRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}
func contractFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	raw, err := os.ReadFile(filepath.Join(moduleRoot(t), IndexPath))
	if err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(root, IndexPath)
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, raw, 0644); err != nil {
		t.Fatal(err)
	}
	return root
}
func errorCode(t *testing.T, err error) string {
	t.Helper()
	var operation *OperationError
	if !errors.As(err, &operation) {
		t.Fatalf("expected operation error, got %v", err)
	}
	return operation.Code
}
func TestBranchGuardWithoutLegacyPolicyTrees(t *testing.T) {
	root := contractFixture(t)
	for _, name := range []string{"dev/0.4.0", "codex/migrate-branch", "feature/example", "tool-retained"} {
		result, err := ValidateCreation(root, name, name, "USER_EXPLICIT", "DEVELOPMENT_VERSION_BRANCH", "GITHUB_CREATE_BRANCH_API")
		if err != nil {
			t.Fatal(err)
		}
		if result["status"] != "STRUCTURE_VALID" || result["authorization_status"] != "UNVERIFIED" || result["branch_creation_authorized"] != false {
			t.Fatalf("request became authority: %v", result)
		}
	}
	for _, name := range []string{"-invalid", "a..b", "ref.lock", "name@{old}", "name\nother"} {
		if err := CheckName(root, name); err == nil {
			t.Fatalf("invalid Git branch accepted: %q", name)
		}
	}
	cases := []struct{ candidate, approved, source, request, mechanism, code string }{
		{"dev/one", "dev/two", "USER_EXPLICIT", "DEVELOPMENT_VERSION_BRANCH", "GITHUB_CREATE_BRANCH_API", "BRANCH_NAME_NOT_EXACTLY_APPROVED"},
		{"dev/one", "dev/one", "AGENT_INFERRED", "DEVELOPMENT_VERSION_BRANCH", "GITHUB_CREATE_BRANCH_API", "BRANCH_CREATION_REQUIRES_USER_EXPLICIT"},
		{"dev/one", "dev/one", "USER_EXPLICIT", "DEVELOPMENT_VERSION_BRANCH", "GIT_SWITCH_CREATE", "UNAUTHORIZED_BRANCH_CREATION_MECHANISM"},
	}
	for _, item := range cases {
		_, err := ValidateCreation(root, item.candidate, item.approved, item.source, item.request, item.mechanism)
		if errorCode(t, err) != item.code {
			t.Fatal(err)
		}
	}
}
func TestBranchClassificationAndTransitionCannotRestoreRetiredAuthority(t *testing.T) {
	root := contractFixture(t)
	result, err := ClassifyExisting(root, "tool-0.3.4-authority-consistency")
	if err != nil || result["classification"] != "UNVERIFIED" {
		t.Fatalf("retired retention classification restored: %v %v", result, err)
	}
	result, err = ProfileTransition(root, "dev/0.4.0", "pp.1.01", "pp.1.02")
	if err != nil || result["status"] != "UNVERIFIED" || result["branch_creation_authorized"] != false {
		t.Fatalf("profile became branch authority: %v %v", result, err)
	}
}
func TestBranchContractCannotSelfAuthorizeMutation(t *testing.T) {
	root := contractFixture(t)
	filename := filepath.Join(root, IndexPath)
	raw, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	var data Object
	if err = json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	data["execution_authorized"] = true
	raw, _ = json.Marshal(data)
	if err = os.WriteFile(filename, raw, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err = Load(root); err == nil {
		t.Fatal("editable completion flag admitted branch mutation")
	}
	if _, err = Load(t.TempDir()); err == nil {
		t.Fatal("missing successor index fell back to retired policy")
	}
}
