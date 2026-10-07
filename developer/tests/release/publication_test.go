package release_test

import (
	"strings"
	"testing"
)

func TestPublicationArtifactVerificationRemainsBoundToExactProductArtifact(t *testing.T) {
	text := releaseWorkflowText(t)
	for _, required := range []string{
		"Verify publication Product Artifact evidence and exact snapshot binding",
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
		"actions/upload-artifact@v4",
		"actions/download-artifact@v4",
		"needs: build",
		"environment: pypi",
		"id-token: write",
		"pypa/gh-action-pypi-publish@release/v1",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("publication handoff missing %q", required)
		}
	}
}
