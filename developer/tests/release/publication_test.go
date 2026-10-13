package release_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func TestPublicationArtifactVerificationRemainsBoundToExactProductArtifact(t *testing.T) {
	source, err := os.ReadFile(filepath.Join(testrepo.Root(t), ".github", "checks", "release", "Tool_artifact_verify.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, required := range []string{
		"Product Artifact boundary evaluator did not run",
		"ptsip-artifact-evidence/v1",
		"ptsip-artifact-evidence-binding/v1",
		"producer_component = 'repository-release-automation'",
		"artifact_snapshot_binding",
		"wheel-sha256:",
		"PTSIP-PKG-001",
		"--force-reinstall --no-deps",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("publication artifact verification missing %q", required)
		}
	}
}

func TestPublicationConsumesOnlyVerifiedBuildArtifactWithTrustedPublishing(t *testing.T) {
	text := releaseWorkflowText(t)
	for _, required := range []string{
		"actions/upload-artifact@v7",
		"actions/download-artifact@v8",
		"needs: [prepare, build]",
		"environment: pypi",
		"id-token: write",
		"packages-dir: pypi-dist/",
		"release-tool-any",
		"pypa/gh-action-pypi-publish@release/v1",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("publication handoff missing %q", required)
		}
	}
}
