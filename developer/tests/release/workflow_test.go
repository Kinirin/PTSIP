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
		"if: ${{ github.event_name == 'workflow_dispatch' }}",
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
	for _, command := range []string{
		"release prepare",
		"release gate",
		"release reconfirm",
		"release draft",
		"release tag-verify",
		"release build",
		"release metadata-check",
	} {
		if !strings.Contains(text, command) {
			t.Fatalf("native release command missing from tooling-release.yml: %s", command)
		}
	}
	if strings.Contains(text, "python -m developer.automation") ||
		strings.Contains(text, "from developer.automation") {
		t.Fatal("canonical release workflow reintroduced Python Developer automation")
	}
}

func TestPrepareAndPublishKeepIndependentExactSourceVerification(t *testing.T) {
	text := releaseWorkflowText(t)
	for _, required := range []string{
		"release gate --source-sha",
		"pp release --sha $env:SOURCE_SHA",
		"Reconfirm candidate remains current main",
		"release tag-verify --release-tag",
		"pp release --sha HEAD",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("independent release verification missing %q", required)
		}
	}
}
