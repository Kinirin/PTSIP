package responsibilityauthority_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	verification "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Responsibility-Authority/verification"
	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
)

func TestResponsibilityAuthoringNeedsNoPolicyOrLegacyTree(t *testing.T) {
	const configPath = "src/ptsip/schemas/config.json"
	const schemaPath = "src/ptsip/schemas/storage/rbac/ra-list.schema.json"
	const vectorsPath = "src/tests/ptsip/rbac-list-conformance.json"
	repo := testrepo.Open(t.TempDir())
	testrepo.CopyFiles(t, repo, configPath, schemaPath, vectorsPath)
	var catalog struct {
		Storage map[string]map[string]struct {
			Path string `json:"path"`
			ID   string `json:"$id"`
		} `json:"storage"`
	}
	readFixture(t, repo, configPath, &catalog)
	binding := catalog.Storage["Responsibility-Authority"]["RA_list"]
	schema := testrepo.ReadJSON(t, repo, schemaPath)
	if binding.Path != "storage/rbac/ra-list.schema.json" || binding.ID != schema["$id"] {
		t.Fatalf("canonical RA registration does not identify its schema: %+v", binding)
	}
	var vectors []struct {
		Name         string `json:"name"`
		Registration string `json:"registration"`
		Valid        bool   `json:"valid"`
		Instance     any    `json:"instance"`
	}
	readFixture(t, repo, vectorsPath, &vectors)
	var accepts, rejects bool
	for _, vector := range vectors {
		if vector.Registration != "RA_list" {
			continue
		}
		accepts = accepts || vector.Valid
		rejects = rejects || !vector.Valid
		t.Run(vector.Name, func(t *testing.T) {
			err := repo.Validate(schemaPath, vector.Instance)
			if (err == nil) != vector.Valid {
				t.Fatalf("wanted valid=%t, got %v", vector.Valid, err)
			}
		})
	}
	if !accepts || !rejects {
		t.Fatal("standalone structural validation needs both valid and invalid RA vectors")
	}

	// The retired developer definition may not be recreated from the product list schema.
	result, code := verification.Run(testrepo.Root(t), "check", "", "", func(any, any, []any) (bool, error) {
		t.Fatal("developer verification reached validation without its definition contract")
		return false, nil
	})
	report, ok := result.(map[string]any)
	problem, _ := report["problem"].(string)
	if !ok || code != 2 || report["status"] != "FAIL" || report["execution_authorized"] != false ||
		!strings.Contains(strings.ReplaceAll(problem, `\`, "/"), ".ptsip/contracts/Responsibility-Authority/Responsibility-Authority.json") {
		t.Fatalf("developer RA verification did not fail closed: code=%d result=%v", code, result)
	}
}

func readFixture(t *testing.T, repo *testrepo.Repository, ref string, target any) {
	t.Helper()
	file, err := repo.Path(ref)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatalf("decode %s: %v", ref, err)
	}
}
