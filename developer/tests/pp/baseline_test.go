package pp

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCurrentProjectProfileContractHasImmutableBaseline(t *testing.T) {
	binary := ppBuildAutomation(t)

	current, err, output := ppAutomation(t, binary, "profile-registry", "current")
	if err != nil {
		t.Fatalf("profile-registry current: %v\n%s", err, output)
	}
	baseline, ok := current["baseline"].(string)
	if !ok || baseline != "profiles/history/pp.1.02" {
		t.Fatalf("current baseline = %#v", current["baseline"])
	}

	catalog, err, output := ppAutomation(t, binary, "profile-registry", "catalog")
	if err != nil {
		t.Fatalf("profile-registry catalog: %v\n%s", err, output)
	}

	root := ppRepositoryRoot(t)
	for _, raw := range catalog["profiles"].([]any) {
		resource := raw.(map[string]any)["resource"].(string)
		historical, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(baseline), resource))
		if err != nil {
			t.Fatalf("read baseline %s: %v", resource, err)
		}
		currentBytes, err := os.ReadFile(filepath.Join(root, "profiles", resource))
		if err != nil {
			t.Fatalf("read current profile %s: %v", resource, err)
		}
		if !bytes.Equal(historical, currentBytes) {
			t.Fatalf("current profile differs from immutable baseline: %s", resource)
		}
	}
}
