package machine

import (
    "strings"
    "testing"
)

func TestCleanupIDValidationRejectsNonPolicyIdentity(t *testing.T) {
    for _, invalid := range []string{"../MPD-0010", "mpd-0010", "MPD-ABC/0001", "", "SFP-ABC.."} {
        if cleanupPolicyID.MatchString(invalid) {
            t.Fatalf("unsafe policy identity admitted: %q", invalid)
        }
    }
}

func TestCleanupApplyRequiresExpectedHeadBeforeGitAccess(t *testing.T) {
    r := policyTestRepo(t)
    report, err := r.CleanupApply("MPD-0010", "")
    if err != nil || report["status"] != "BLOCKED" ||
        !strings.Contains(report["blockers"].([]string)[0], "EXACT_EXPECTED_HEAD_REQUIRED") {
        t.Fatalf("missing expected head must fail closed: %#v %v", report, err)
    }
}

func TestCleanupMissingSubstatusFailsClosed(t *testing.T) {
    r := policyTestRepo(t)
    report, err := r.CleanupInspect("SFP-ARCH-0003")
    if err != nil || report["status"] != "BLOCKED" {
        t.Fatalf("missing Support Removable authorization admitted: %#v %v", report, err)
    }
}
