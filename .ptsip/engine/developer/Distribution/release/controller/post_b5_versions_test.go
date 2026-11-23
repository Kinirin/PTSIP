package releasecontroller

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCurrentEngineVersionsAndProjectionsWithoutApprovalRegistry(t *testing.T) {
	root, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	composition, err := ReadComposition(root)
	if err != nil {
		t.Fatal(err)
	}
	var contract Contract
	if err := ReadJSON(root, ContractPath, &contract); err != nil {
		t.Fatal(err)
	}
	if err := VerifyCompositionProjections(root, composition, contract); err != nil {
		t.Fatal(err)
	}
	for id, version := range composition.Engines {
		if err := ValidateEngineVersion(root, id, version); err != nil {
			t.Fatal(err)
		}
		lock, err := os.ReadFile(filepath.Join(root, "src/ptsip/engines", id, "Cargo.lock"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(strings.ReplaceAll(string(lock), "\r\n", "\n"), fmt.Sprintf("name = %q\nversion = %q", "ptsip-engine-"+id, version)) {
			t.Fatalf("lockfile differs: %s", id)
		}
	}
}
