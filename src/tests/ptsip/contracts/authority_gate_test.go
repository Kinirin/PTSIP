package contracts_test

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Verify the live machine boundary directly. Archive existence and historical
// semantic bodies must not supply the expected value or a lookup fallback.
func TestLiveReferenceGateRejectsHistoricalPolicyIdentity(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the registered contract verification source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", "..", ".."))
	binary := filepath.Join(t.TempDir(), "ptsip-dev")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "-C", "developer/automation", "build", "-tags", "grammar_subset,grammar_subset_python", "-o", binary, "./cmd/ptsip-dev")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("native verification dependency unavailable: %v\n%s", err, output)
	}
	for _, id := range []string{"MPD-0012", "MPD-SPEC-0001"} {
		t.Run(id, func(t *testing.T) {
			inspect := exec.Command(binary, "policy-lifecycle", "inspect", id)
			inspect.Dir = root
			output, err := inspect.CombinedOutput()
			if err != nil {
				t.Fatalf("inspect infrastructure error: %v\n%s", err, output)
			}
			var result map[string]any
			if err := json.Unmarshal(output, &result); err != nil {
				t.Fatal(err)
			}
			if result["status"] != "NOT_FOUND" || result["operationally_resolvable"] == true {
				t.Fatal("historical or audit-only identity became a live owner", result)
			}
			lookup := exec.Command(binary, "policy-resolver", "get", id)
			lookup.Dir = root
			output, err = lookup.CombinedOutput()
			if err == nil || !strings.Contains(string(output), "unknown or audit-only (not executable)") {
				t.Fatalf("live resolver must deny history and audit-only IDs: %v\n%s", err, output)
			}
		})
	}
	current := exec.Command(binary, "root-family-entry", "inspect", "SFP-GOV-0001")
	current.Dir = root
	output, err := current.CombinedOutput()
	if err != nil {
		t.Fatalf("independent current Support route failed: %v\n%s", err, output)
	}
	var result map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatal(err)
	}
	if result["policy_id"] != "SFP-GOV-0001" || result["policy_class"] != "PTSIP_SUPPORT_FEATURE" || result["semantic_inheritance"] != "FORBIDDEN" {
		t.Fatal("current Support owner route or class isolation changed", result)
	}
}
