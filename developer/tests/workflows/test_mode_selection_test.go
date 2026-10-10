package workflows_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

// These Go successors verify the existing compatibility resolver against the
// current registry's order. A fixture cannot create a second mode priority.
func TestRegisteredTestModeSelectionSuccessors(t *testing.T) {
	root := testrepo.Root(t)
	repo := testrepo.Open(root)
	registry, err := repo.Read(".github/test_modes.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var vectors struct {
		Cases []struct {
			ID      string   `json:"id"`
			Changed string   `json:"changed_file"`
			Members []string `json:"expected_members"`
		} `json:"cases"`
	}
	raw, err := os.ReadFile(filepath.Join(root, "developer/tests/workflows/test_mode_selection_vectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}
	python := os.Getenv("PTSIP_TEST_PYTHON")
	if python == "" {
		python = "python"
	}
	for _, vector := range vectors.Cases {
		t.Run(vector.ID, func(t *testing.T) {
			members := map[string]bool{}
			for _, id := range vector.Members {
				members[id] = true
			}
			want := []string{}
			for _, raw := range registry["modes"].([]any) {
				id := raw.(map[string]any)["id"].(string)
				if members[id] {
					want = append(want, id)
				}
			}
			if len(want) != len(members) {
				t.Fatal("fixture references an unregistered mode")
			}
			cmd := exec.Command(python, ".github/scripts/resolve_test_modes.py", "automatic", "--changed-file", vector.Changed)
			cmd.Dir = root
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("compatibility resolver: %v\n%s", err, output)
			}
			var result struct {
				Selected   []string `json:"selected_ids"`
				Resolution string   `json:"resolution"`
			}
			if err := json.Unmarshal(output, &result); err != nil {
				t.Fatal(err)
			}
			if result.Resolution != "SELECTED" || !reflect.DeepEqual(result.Selected, want) {
				t.Fatalf("registered selection: got %v want %v", result.Selected, want)
			}
		})
	}
}
