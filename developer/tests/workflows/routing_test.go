package workflows_test

import (
	"encoding/json"
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

// readLegacyCheck follows the executable registry binding rather than requiring
// per-Check commands to be hardcoded into the generic workflow.
func readLegacyCheck(t *testing.T, id string) (string, []string) {
	t.Helper()
	var registry struct {
		LegacyChecks []struct {
			ID       string   `json:"id"`
			Mode     string   `json:"mode"`
			Script   string   `json:"script"`
			Triggers []string `json:"triggers"`
		} `json:"legacy_checks"`
	}
	raw, err := os.ReadFile(filepath.Join(testrepo.Root(t), ".github", "checks.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &registry); err != nil {
		t.Fatal(err)
	}
	for _, check := range registry.LegacyChecks {
		if check.ID != id {
			continue
		}
		if check.Mode == "" || !strings.HasPrefix(check.Script, ".github/checks/") || len(check.Triggers) == 0 {
			t.Fatal("Check has no registered owner or executable applicability", id)
		}
		code, err := os.ReadFile(filepath.Join(testrepo.Root(t), filepath.FromSlash(check.Script)))
		if err != nil {
			t.Fatal("registered Check script cannot be resolved", id, err)
		}
		return string(code), check.Triggers
	}
	t.Fatal("unregistered required Check", id)
	return "", nil
}

func hasTrigger(triggers []string, want string) bool {
	for _, trigger := range triggers {
		if trigger == want {
			return true
		}
	}
	return false
}

func TestStaticAndReleaseVerificationAreNotPinnedToOneVersionBranch(t *testing.T) {
	text := workflow(t, "tooling-test.yml")
	var document map[string]any
	if err := yaml.Unmarshal([]byte(text), &document); err != nil {
		t.Fatal(err)
	}
	jobs := document["jobs"].(map[string]any)
	for _, job := range []string{"resolve", "check", "coverage"} {
		if jobs[job] == nil {
			t.Fatal("generic Check orchestration stage absent", job)
		}
	}
	for _, marker := range []string{"github.event.deleted == false", "check_execution.py plan", "fromJSON(needs.resolve.outputs.checks)", "check_execution.py coverage"} {
		if !strings.Contains(text, marker) {
			t.Fatal("generic verification route missing", marker)
		}
	}
	if strings.Contains(text, "github.ref_name == 'dev/0.3.8'") {
		t.Fatal("verification pinned to one development branch")
	}
	runner, err := os.ReadFile(filepath.Join(testrepo.Root(t), ".github/scripts/check_execution.py"))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{`if event == "push":`, `startswith("release:")`, `preset = "full"`} {
		if !strings.Contains(string(runner), marker) {
			t.Fatal("push/release routing was not preserved in Check selector", marker)
		}
	}
	for _, id := range []string{"policy-lifecycle", "release-contract"} {
		_, triggers := readLegacyCheck(t, id)
		if id == "policy-lifecycle" && !hasTrigger(triggers, "push") {
			t.Fatal("push policy validation obligation omitted")
		}
		if id == "release-contract" && !hasTrigger(triggers, "full") {
			t.Fatal("release verification obligation omitted")
		}
	}
}

func TestWU02RoutingConsumesNativeControlContextAndExplicitExecutionBranch(t *testing.T) {
	script, triggers := readLegacyCheck(t, "wu02-control")
	if !hasTrigger(triggers, "push") {
		t.Fatal("WU-02 control verification not selected on push")
	}
	for _, expected := range []string{
		"./cmd/ptsip-dev wu02 control-context",
		"$controlBranch = $controlContext.control_branch",
		"$env:PTSIP_EXECUTION_BRANCH -eq $controlBranch",
		"./cmd/ptsip-dev wu02 validate --execution-branch \"$env:PTSIP_EXECUTION_BRANCH\"",
	} {
		if !strings.Contains(script, expected) {
			t.Fatalf("native WU-02 Check contract missing: %s", expected)
		}
	}
	code, err := os.ReadFile(filepath.Join(testrepo.Root(t), ".github/scripts/check_execution.py"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(code), "PTSIP_EXECUTION_BRANCH") {
		t.Fatal("generic Check runner does not supply the execution branch to WU-02")
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
	script, triggers := readLegacyCheck(t, "agent-context-migration")
	if !hasTrigger(triggers, "push") || !hasTrigger(triggers, "agent-context-migration") {
		t.Fatal("agent context verification dispatch applicability was lost")
	}
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
