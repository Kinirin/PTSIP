package releasetest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	release "github.com/Kinirin/PTSIP/.github/checks/release/controller"
)

func TestAutomaticVerificationRejectsDifferentSHAAndSelection(t *testing.T) {
	s := release.Selection{SourceSHA: strings.Repeat("a", 40), SelectionSHA: strings.Repeat("b", 64), Components: []release.Component{{ID: "context", Kind: "engine", Version: "0.2.0"}}, Platforms: []string{"windows-x64", "linux-x64"}}
	api := func(method, endpoint string, payload, out any) error {
		if method != "GET" || !strings.Contains(endpoint, "head_sha="+s.SourceSHA) {
			t.Fatal("verification lookup is not source bound")
		}
		runs := []release.WorkflowRun{{ID: 4, Head: strings.Repeat("c", 40), Path: ".github/workflows/tooling-test.yml", Status: "completed", Conclusion: "success"}, {ID: 3, Head: s.SourceSHA, Path: ".github/workflows/tooling-test.yml", Status: "completed", Conclusion: "success"}, {ID: 2, Head: s.SourceSHA, Path: ".github/workflows/tooling-test.yml", Status: "completed", Conclusion: "success"}}
		b, _ := json.Marshal(map[string]any{"workflow_runs": runs})
		return json.Unmarshal(b, out)
	}
	fetch := func(run release.WorkflowRun) (release.Selection, error) {
		if run.ID == 4 {
			t.Fatal("foreign source artifact consumed")
		}
		candidate := s
		if run.ID == 3 {
			candidate.SelectionSHA = "different-selection"
		}
		return candidate, nil
	}
	id, err := release.FindVerification(s, api, fetch)
	if err != nil || id != 2 {
		t.Fatalf("automatic run join failed: %d %v", id, err)
	}
	if _, err := release.FindVerification(s, api, func(release.WorkflowRun) (release.Selection, error) {
		return release.Selection{}, fmt.Errorf("expired artifact")
	}); err == nil {
		t.Fatal("expired or mismatched evidence accepted")
	}
}

func TestReleasePresentationUsesPackageTitleAndOrderedTypedSections(t *testing.T) {
	root := repoRoot(t)
	s, err := release.Select(root, "package", "")
	if err != nil {
		t.Fatal(err)
	}
	snapshot := release.Snapshot{Version: "0.3.8.42d9e6"}
	for _, c := range s.Components {
		snapshot.Components = append(snapshot.Components, release.PublishedComponent{ID: c.ID, Kind: c.Kind, Version: c.Version})
	}
	title, body, err := release.ReleasePresentation(root, s, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	composition, err := release.ReadComposition(root)
	if err != nil {
		t.Fatal(err)
	}
	if title != "PTSIP_Package 0.3.8.b3 (0.3.8.42d9e6)" || !strings.Contains(body, "PTSIP==0.3.8b3") || !strings.Contains(body, "PTSIP_Tool: "+composition.Tool) || strings.Contains(body, "textTool:") || strings.Contains(body, "SPEC_REVISION:") {
		t.Fatal("release identity projection differs from requested template")
	}
	previous := -1
	for _, heading := range []string{"# PTSIP Release\n", "# PTSIP Tool\n", "# Project Profile pp.1.02\n", "# PTSIP Engine\n"} {
		pos := strings.Index(body, heading)
		if pos <= previous {
			t.Fatalf("section order differs: %s", heading)
		}
		previous = pos
	}
	if strings.Count(body, "### Purpose\n") != 8 || strings.Count(body, "Engine ID:") != 7 || strings.Count(body, "Capability:") != 7 {
		t.Fatal("engine note contracts missing or profile headings not nested correctly")
	}
	if strings.Contains(body, "## context ") {
		t.Fatal("alphabetic component dump replaced the typed sections")
	}
}

func TestReissueBoundaryRejectsProducerAndVersionChanges(t *testing.T) {
	root := t.TempDir()
	write := func(ref, text string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(ref))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	contract, err := os.ReadFile(filepath.Join(repoRoot(t), release.EvidenceContractPath))
	if err != nil {
		t.Fatal(err)
	}
	write(release.EvidenceContractPath, string(contract))
	project := "[project]\nversion=\"0.3.8a6\"\n[tool.ptsip.release]\npackage_base=\"0.3.8\"\n"
	write("pyproject.toml", project)
	write("src/producer.go", "original")
	write("docs/releasenote/example.md", "old")
	run := func(args ...string) string {
		t.Helper()
		b, err := release.Run(root, args...)
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(b))
	}
	run("git", "init", "--quiet")
	run("git", "config", "user.name", "fixture")
	run("git", "config", "user.email", "fixture@example.invalid")
	commit := func() string {
		run("git", "add", ".")
		run("git", "commit", "--quiet", "-m", "fixture")
		return run("git", "rev-parse", "HEAD")
	}
	origin := commit()
	write("docs/releasenote/example.md", "new presentation")
	write("pyproject.toml", project+"[tool.ptsip.release.published]\nversion=\"0.3.8.42d9e6\"\n")
	commit()
	if delta, err := release.ReissueSourceChanges(root, origin, "HEAD"); err != nil || delta != "" {
		t.Fatalf("presentation/observed-record change affected original artifacts: %s %v", delta, err)
	}
	write("pyproject.toml", strings.Replace(project, "0.3.8a6", "0.3.8a7", 1))
	commit()
	if delta, err := release.ReissueSourceChanges(root, origin, "HEAD"); err != nil || delta != "pyproject.toml" {
		t.Fatal("new Tool version reused old admission")
	}
	write("pyproject.toml", project)
	write("src/producer.go", "changed")
	commit()
	if delta, err := release.ReissueSourceChanges(root, origin, "HEAD"); err != nil || !strings.Contains(delta, "src/producer.go") {
		t.Fatal("changed implementation reused original PASS")
	}
}
