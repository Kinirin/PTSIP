package branch

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBranchCLIContractAndClosedArguments(t *testing.T) {
	root := contractFixture(t)
	for _, args := range [][]string{{"branch-control", "commands", "--repository", root}, {"branch-guard", "validate", "--candidate", "codex/example", "--approved-name", "codex/example", "--authorization-source", "USER_EXPLICIT", "--request-kind", "DEVELOPMENT_VERSION_BRANCH", "--creation-mechanism", "GITHUB_CREATE_BRANCH_API", "--repository", root}} {
		var output bytes.Buffer
		if code := CLI(args, &output); code != 0 {
			t.Fatalf("CLI code=%d: %s", code, output.String())
		}
	}
	for _, args := range [][]string{{"branch-control", "commands", "--repository", root, "--invented", "yes"}, {"branch-control", "commands", "--repository", root, "--repository", root}, {"branch-control", "commands", "--repository"}} {
		if _, err := Execute(args); err == nil {
			t.Fatalf("invalid options accepted: %v", args)
		}
	}
	var output bytes.Buffer
	if code := CLI([]string{"branch-guard", "classify-existing", "--branch", "main", "--repository", root}, &output); code != 4 || !strings.Contains(output.String(), "UNVERIFIED") {
		t.Fatalf("classification claimed authority: %d %s", code, output.String())
	}
}
func TestStandaloneBranchCLIWorksWithoutLegacyDirectories(t *testing.T) {
	root := contractFixture(t)
	module := filepath.Join(moduleRoot(t), ".ptsip/engine/developer")
	binary := filepath.Join(t.TempDir(), "ptsip-branch")
	if os.PathSeparator == '\\' {
		binary += ".exe"
	}
	if raw, err := exec.Command("go", "-C", module, "build", "-o", binary, "./Distribution/branch/cmd/ptsip-branch").CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, raw)
	}
	raw, err := exec.Command(binary, "branch-control", "commands", "--repository", root).CombinedOutput()
	if err != nil {
		t.Fatalf("standalone: %v %s", err, raw)
	}
	var result Object
	if err = json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result["execution_authorized"] != false {
		t.Fatal("module load granted execution")
	}
}
