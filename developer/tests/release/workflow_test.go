package release_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func releaseWorkflowText(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(
		filepath.Join(testrepo.Root(t), ".github", "workflows", "tooling-release.yml"),
	)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(raw), "\r\n", "\n")
}

func TestToolingReleaseIsSingleCanonicalReleaseWorkflow(t *testing.T) {
	root := testrepo.Root(t)
	legacy := filepath.Join(root, ".github", "workflows", "release.yml")
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("legacy release.yml must be retired: %v", err)
	}

	text := releaseWorkflowText(t)
	for _, required := range []string{
		"workflow_dispatch:",
		"release:\n    types: [published]",
		"prepare:",
		"targets:",
		"disabled_targets:",
		"fail-fast: false",
		"build:",
		"publish:",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("canonical tooling-release workflow missing %q", required)
		}
	}
}

func TestToolingReleaseDelegatesLifecycleSemanticsToNativeReleaseAutomation(t *testing.T) {
	text := releaseWorkflowText(t)
	for _, required := range []string{".github/checks/release/run_release.ps1", "-Action Build", "-Action Verify", "-Action Publish", "Action='Plan'"} {
		if !strings.Contains(text, required) {
			t.Fatalf("independent release dispatcher missing %s", required)
		}
	}
	source, err := os.ReadFile(filepath.Join(testrepo.Root(t), ".github", "checks", "release", "run_release.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(source), "./cmd/ptsip-release") {
		t.Fatal("dispatcher must call admitted native release semantics")
	}
	if strings.Contains(text, "python -m developer.automation") {
		t.Fatal("Python Developer automation was reintroduced")
	}
}

func TestPrepareAndPublishKeepIndependentExactSourceVerification(t *testing.T) {
	text := releaseWorkflowText(t)
	for _, required := range []string{"fetch-depth: 0", "pp release --sha HEAD", "github.event.release.tag_name || github.sha", "Upload verified component artifacts", "Publish this component after all its platforms verify"} {
		if !strings.Contains(text, required) {
			t.Fatalf("independent release verification missing %s", required)
		}
	}
	source, err := os.ReadFile(filepath.Join(testrepo.Root(t), ".github", "checks", "release", "publication.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"ReconfirmMain(repo, s.SourceSHA)", "VerifyReceipts(s, id, dir, true)"} {
		if !strings.Contains(string(source), required) {
			t.Fatalf("publication gate missing %s", required)
		}
	}
}
