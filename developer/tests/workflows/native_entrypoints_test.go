package workflows_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func TestAgentCommandLifecycleWorksWithoutPythonAutomationOrExecutableSearchPath(t *testing.T) {
	binary := testrepo.BuildCLI(t)
	repo := testrepo.Open(t.TempDir())
	testrepo.CopyTree(t, repo, "developer/policy")
	if err := repo.AtomicWrite("pyproject.toml", []byte("[project]\nname='native-command-fixture'\n"), nil); err != nil {
		t.Fatal(err)
	}
	if err := repo.AtomicWrite("AGENTS.md", []byte("# AGENTS\n\nBefore editing, read README.md.\n\nALPHA BETA GAMMA\n"), nil); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"agent-instruction-classifier", "classify"},
		{"agent-instruction-materializer", "status"},
		{"agent-instruction-materializer", "materialize"},
		{"agent-instruction-materializer", "check"},
		{"agent-instruction-entry", "resolve", "--operation", "MODIFY"},
		{"agent-instruction-progressive", "bootstrap-level1"},
		{"agent-instruction-progressive", "check"},
		{"agent-instruction-activation", "check"},
		{"agent-integration", "status"},
	} {
		t.Run(strings.Join(args[:2], "/"), func(t *testing.T) {
			result, err, output := testrepo.CLI(t, binary, repo.Root, args...)
			if err != nil || len(result) == 0 {
				t.Fatalf("registered native command failed: %v\n%s", err, output)
			}
			if args[0] == "agent-integration" && result["mode"] != "LOCAL_CLI_ONLY" {
				t.Fatal("native CLI must remain available without MCP", result)
			}
		})
	}
	beforeStage := testrepo.ReadJSON(t, repo, ".agent/stages/level1.json")
	_, err, output := testrepo.CLI(t, binary, repo.Root, "agent-instruction-activation", "activate")
	if err == nil || !strings.Contains(output, "already active; use check instead") {
		t.Fatalf("repeated activation must reject the completed stage before reading archived atoms: %v\n%s", err, output)
	}
	afterStage := testrepo.ReadJSON(t, repo, ".agent/stages/level1.json")
	before, err := json.Marshal(beforeStage)
	if err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(afterStage)
	if err != nil || string(before) != string(after) {
		t.Fatal("rejected activation changed the admitted stage", err)
	}
	path, err := repo.Path("AGENTS.md")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(raw), "ALPHA BETA GAMMA") || !strings.Contains(string(raw), "PTSIP_AGENT_ENTRY") {
		t.Fatal("native command lifecycle lost unresolved text or compact entry", string(raw), err)
	}
}
