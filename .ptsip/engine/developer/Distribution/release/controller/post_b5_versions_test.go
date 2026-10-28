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
	var plan struct {
		Components []struct {
			Version  string `json:"version"`
			Manifest string `json:"cargo_manifest_ref"`
		} `json:"components"`
	}
	if err := ReadJSON(root, "src/ptsip/engines/contracts/installation-plan.json", &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Components) != len(composition.Engines) {
		t.Fatal("engine inventory differs from declared source")
	}
	seen := map[string]bool{}
	for _, item := range plan.Components {
		id := strings.TrimSuffix(strings.TrimPrefix(item.Manifest, "src/ptsip/engines/"), "/Cargo.toml")
		version, found := composition.Engines[id]
		if !found || seen[id] || item.Version != version {
			t.Fatalf("installation plan differs: %s", id)
		}
		seen[id] = true
		if err := ValidateEngineVersion(root, id, version); err != nil {
			t.Fatal(err)
		}
		lock, err := os.ReadFile(filepath.Join(root, "src/ptsip/engines", id, "Cargo.lock"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(lock), fmt.Sprintf("name = %q\nversion = %q", "ptsip-engine-"+id, version)) {
			t.Fatalf("lockfile differs: %s", id)
		}
	}
}
