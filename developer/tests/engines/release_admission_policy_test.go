package engines_test

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func TestWithdrawnCTRLReleaseAdmissionPolicyAbsent(t *testing.T) {
    root := testrepo.Root(t)
    for _, ref := range []string{
        "developer/policy/CTRL/release-admission.policy.json",
        "developer/policy/approvals/MPA-20261026-CTRL-RELEASE-ADMISSION-COMPLETION.yaml",
    } {
        if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(ref))); !os.IsNotExist(err) {
            t.Fatalf("withdrawn temporary CTRL publication authority remains: %s: %v", ref, err)
        }
    }
}
