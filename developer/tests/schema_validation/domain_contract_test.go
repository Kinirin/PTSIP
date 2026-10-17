package schema_validation

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// Keep the new policy-domain tests within the registered developer-schema-contracts
// verification mode, without importing the retired developer/automation module.
func TestNewDeveloperPolicyDomainContract(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(root, ".ptsip", "engine", "developer")
	cmd := exec.Command("go", "-C", module, "test", "./policy", "-count=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("contract-derived domain package verification failed: %v\n%s", err, output)
	}
}
