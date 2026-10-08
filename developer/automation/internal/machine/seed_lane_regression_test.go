package machine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPP102CompletedSeedRefusesReplayBeforeReadingRetiredRoot(t *testing.T) {
	r := policySeedFixture(t)
	registry, err := r.Read("registry/project-profile-contracts.yaml")
	if err != nil {
		t.Fatal(err)
	}
	registry["current"] = "pp.1.02"
	policyTestWrite(t, r, "registry/project-profile-contracts.yaml", registry)
	root, err := r.Path("ptsip.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(root); err != nil {
		t.Fatal(err)
	}
	policySeedCommit(t, r)
	if _, err := r.SeedPP102Transition(true); err == nil || !strings.Contains(err.Error(), "SEED_REPLAY_FORBIDDEN") {
		t.Fatal(err)
	}
}

func TestPP102SeedImplementationPreservesTransitionAndToolHistoryBoundaries(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	source, err := agentText(r, "developer/automation/internal/machine/policy_seed.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"pp.1.01", "pp.1.02", "Rev.0001", "DISTRIBUTED_EXAMPLE", "PROJECT_PATH_RESOLUTION_REQUIRED", "profiles/history/", "schemas/ptsip-profile-pp-1.02.schema.json", "src/ptsip/specdata/ptsip-profile-pp-1.02.schema.json", "docs/releasenote/project-profile", "README.md", "STATUS.md", policySeedAcceptancePath} {
		if !strings.Contains(source, required) {
			t.Fatal("native seed source contract missing", required)
		}
	}
	for _, placeholder := range []string{"PRODUCT_RUNTIME_ROOT", "PRODUCT_SDK_ROOT", "PRODUCT_TEST_ROOT", "DEVELOPMENT_TOOLING_ROOT", "RELEASE_AUTOMATION_FILE", "OPERATIONS_ROOT", "CONTRACT_ROOT"} {
		if !strings.Contains(source, "${"+placeholder+"}") {
			t.Fatal("distributed placeholder missing", placeholder)
		}
	}
	if strings.Contains(source, "docs/releasenote/tool/0.3.8a1.md") {
		t.Fatal("seed rewrites Tool history")
	}
}

func TestWU02ExplicitBranchWinsAndDetachedHEADDoesNotInferGithubRef(t *testing.T) {
	r := &Repository{Root: t.TempDir()}
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.name", "Go Branch Fixture"}, {"config", "user.email", "fixture@example.invalid"}} {
		if _, err := r.GitOutput(args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.AtomicWrite("fixture.txt", []byte("fixture\n"), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GitOutput("add", "fixture.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GitOutput("-c", "core.hooksPath=NUL", "commit", "-qm", "fixture"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GitOutput("checkout", "--detach", "HEAD"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PTSIP_EXECUTION_BRANCH", "dev/0.3.8")
	actual, err := r.CurrentBranch()
	if err != nil || actual != "dev/0.3.8" {
		t.Fatal(actual, err)
	}
	t.Setenv("PTSIP_EXECUTION_BRANCH", "")
	t.Setenv("GITHUB_REF_NAME", "dev/0.3.8")
	if _, err := r.CurrentBranch(); err == nil {
		t.Fatal("detached branch inferred from Github ref")
	}
}

func TestWU02LanePathsAndBranchesRemainUniqueAndCentrallyOwned(t *testing.T) {
	r, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	parent := "developer/planning/0.4.0/WU-02/WU-02-P01.yaml"
	if _, err := os.Stat(filepath.Join(r.Root, filepath.FromSlash(parent))); os.IsNotExist(err) {
		index, err := r.Read("developer/planning/index.yaml")
		if err != nil {
			t.Fatal(err)
		}
		for _, raw := range List(index["plans"]) {
			if Map(raw)["plan_version"] == "0.4.0" {
				t.Fatal("retired lane plan still registered")
			}
		}
		return
	}
	result, err := r.DispatchOperation("wu02", "control-context", map[string]string{}, nil)
	if err != nil || Map(result)["status"] != "PASS" {
		t.Fatal(result, err)
	}
}
