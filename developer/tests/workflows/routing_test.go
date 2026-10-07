package workflows_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
	"go.yaml.in/yaml/v3"
)

func workflow(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(testrepo.Root(t), ".github", "workflows", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestStaticAndReleaseVerificationAreNotPinnedToOneVersionBranch(t *testing.T) {
	text := workflow(t, "tooling-test.yml")
	for _, expected := range []string{"(github.event_name == 'push' && github.event.deleted == false) || (github.event_name == 'workflow_dispatch'", "startsWith(github.event.head_commit.message, 'release:')"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("verification route missing: %s", expected)
		}
	}
	if strings.Contains(text, "github.ref_name == 'dev/0.3.8'") {
		t.Fatal("verification pinned to one development branch")
	}
}

func TestWU02RoutingConsumesNativeControlContextAndExplicitExecutionBranch(t *testing.T) {
	text := workflow(t, "tooling-test.yml")
	for _, expected := range []string{"./cmd/ptsip-dev wu02 control-context", "$controlBranch = $controlContext.control_branch", "steps.wu02-route.outputs.required == 'true'", "./cmd/ptsip-dev wu02 validate --execution-branch \"$env:PTSIP_EXECUTION_BRANCH\""} {
		if !strings.Contains(text, expected) {
			t.Fatalf("native control route missing: %s", expected)
		}
	}
}

func TestGoToolchainPrecedesEveryDeveloperAutomationConsumer(t *testing.T) {
	for _, file := range []string{"tooling-test.yml", "tooling-release.yml"} {
		t.Run(file, func(t *testing.T) {
			text := workflow(t, file)
			if strings.Contains(text, "python -m developer.automation") || strings.Contains(text, "from developer.automation") {
				t.Fatal("workflow still invokes Python Developer automation")
			}
			var document map[string]any
			if err := yaml.Unmarshal([]byte(text), &document); err != nil {
				t.Fatal(err)
			}
			for name, raw := range document["jobs"].(map[string]any) {
				job := raw.(map[string]any)
				steps, ok := job["steps"].([]any)
				if !ok {
					continue
				}
				goReady := false
				for _, raw := range steps {
					step := raw.(map[string]any)
					uses, _ := step["uses"].(string)
					goReady = goReady || strings.HasPrefix(uses, "actions/setup-go@")
					run, _ := step["run"].(string)
					if strings.Contains(run, "go -C developer/") && !goReady {
						t.Fatalf("job %s calls Go before installing its toolchain: %v", name, step["name"])
					}
				}
			}
		})
	}
}

func TestRepositoryEntryPointsShareNativeInstallerAndDoNotStageUserFiles(t *testing.T) {
	root := testrepo.Root(t)
	for _, file := range []string{"setup_dev.bat", "bootstrap_repo.ps1"} {
		raw, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), "./cmd/ptsip-dev developer-setup install-hooks") || strings.Contains(string(raw), "python -m developer.automation.dev_setup") {
			t.Fatalf("%s does not use canonical Go installer", file)
		}
	}
	raw, err := os.ReadFile(filepath.Join(root, ".githooks", "pre-commit"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "./cmd/ptsip-dev pp pre-commit") || strings.Contains(string(raw), "git add -A") || strings.Contains(string(raw), "PYTHON=") {
		t.Fatal("pre-commit wrapper changed its native delegation or staging boundary")
	}
}

func TestContextMigrationAndDeveloperTestModeUseRetiredGoTestSurface(t *testing.T) {
	text := workflow(t, "tooling-test.yml")
	if strings.Contains(text, "developer/tests/test_agent_context_migration.py") || !strings.Contains(text, "-run '^TestAgent(Profile|OperationReferences|ContextNative)' ./internal/machine") || !strings.Contains(text, "src/tests/ptsip/test_modes/test_agent_context_routing.py") {
		t.Fatal("bounded context migration does not invoke the preserved native and routing regressions")
	}
	raw, err := os.ReadFile(filepath.Join(testrepo.Root(t), ".github/test_modes.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var registry map[string]any
	if err := yaml.Unmarshal(raw, &registry); err != nil {
		t.Fatal(err)
	}
	for _, raw := range registry["modes"].([]any) {
		mode := raw.(map[string]any)
		execution := mode["execution"].(map[string]any)
		targets, _ := execution["pytest"].([]any)
		for _, target := range targets {
			if strings.HasPrefix(target.(string), "developer/tests") {
				t.Fatal("a Go-only Developer test module was selected as pytest input", mode["id"], target)
			}
		}
	}
}
