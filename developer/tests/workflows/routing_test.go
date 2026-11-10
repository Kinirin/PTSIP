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

// The former shared executor is explicitly retired. Its absence is not PASS.
func TestRetiredCommonVerificationIsExplicitlyFailClosed(t *testing.T) {
    text := workflow(t, "tooling-test.yml")
    if strings.Contains(text, "check_execution.py") || strings.Contains(text, "  push:") {
        t.Fatal("retired workflow still invokes common legacy verification")
    }
    if !strings.Contains(text, "workflow_dispatch:") || !strings.Contains(text, "exit 1") {
        t.Fatal("manual execution must fail closed")
    }
    root := testrepo.Root(t)
    for _, file := range []string{
        ".github/checks.yaml", ".github/checks-legacy-inventory.json",
        ".github/scripts/check_execution.py", ".github/checks/readme-translation.ps1",
    } {
        if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(file))); !os.IsNotExist(err) {
            t.Fatalf("retired artifact remains: %s (%v)", file, err)
        }
    }
    raw, err := os.ReadFile(filepath.Join(root, ".github/verification-retirement.json"))
    if err != nil { t.Fatal(err) }
    for _, s := range []string{"\"state\": \"OPEN\"", "\"successor_common_runner\": \"NOT_IMPLEMENTED\"", "\"release_and_merge_readiness\": \"NOT_ESTABLISHED\""} {
        if !strings.Contains(string(raw), s) { t.Fatal("gap record incomplete", s) }
    }
}

func TestRetiredWU02LegacyScriptIsHistoricalOnly(t *testing.T) {
    raw, err := os.ReadFile(filepath.Join(testrepo.Root(t), ".github/checks/wu02-control.ps1"))
    if err != nil { t.Fatal(err) }
    for _, s := range []string{
        "./cmd/ptsip-dev wu02 control-context",
        "$controlBranch = $controlContext.control_branch",
        "$env:PTSIP_EXECUTION_BRANCH -eq $controlBranch",
        "./cmd/ptsip-dev wu02 validate --execution-branch \"$env:PTSIP_EXECUTION_BRANCH\"",
    } {
        if !strings.Contains(string(raw), s) { t.Fatal("historical WU02 script changed", s) }
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

func TestRepositoryEntryPointsDoNotInvokeRetiredAutomation(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"setup_dev.bat", "bootstrap_repo.ps1", ".github/workflows/branch-verification.yml"} {
		raw, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "developer/automation") || strings.Contains(string(raw), "developer.automation") {
			t.Fatalf("%s still depends on retired automation", file)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".githooks", "pre-commit")); !os.IsNotExist(err) {
		t.Fatalf("retired automation hook remains (%v)", err)
	}
}

func TestContextMigrationAndDeveloperTestModeUseRetiredGoTestSurface(t *testing.T) {
	rawScript, err := os.ReadFile(filepath.Join(testrepo.Root(t), ".github/checks/agent-context-migration.ps1"))
	if err != nil { t.Fatal(err) }
	script := string(rawScript)
	if strings.Contains(script, "developer/tests/test_agent_context_migration.py") ||
		!strings.Contains(script, "-run '^TestAgent(Profile|OperationReferences|ContextNative)' ./internal/machine") ||
		!strings.Contains(script, "src/tests/ptsip/test_modes/test_agent_context_routing.py") {
		t.Fatal("registered context migration Check lost native or routing regression")
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
