package releasetest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	release "github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/controller"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestPlatformRoutingSchemasMatchCurrentCatalog(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, ".ptsip/contracts/release/index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index struct {
		Dimension string           `json:"platform_dimension"`
		Platforms []map[string]any `json:"platforms"`
	}
	if err := json.Unmarshal(raw, &index); err != nil {
		t.Fatal(err)
	}
	if index.Dimension != "DISTRIBUTION_SERVICE" || len(index.Platforms) != 3 {
		t.Fatal("incomplete service platform catalog")
	}
	for _, record := range index.Platforms {
		name := record["platform"].(string)
		t.Run(name, func(t *testing.T) {
			schemaPath, ok := record["validation_ref"].(string)
			if !ok {
				t.Fatal("missing routing schema")
			}
			data, err := os.ReadFile(filepath.Join(root, schemaPath))
			if err != nil {
				t.Fatal(err)
			}
			var document map[string]any
			if err := json.Unmarshal(data, &document); err != nil {
				t.Fatal(err)
			}
			compiler := jsonschema.NewCompiler()
			id := document["$id"].(string)
			if err := compiler.AddResource(id, document); err != nil {
				t.Fatal(err)
			}
			schema, err := compiler.Compile(id)
			if err != nil {
				t.Fatal(err)
			}
			if err := schema.Validate(record); err != nil {
				t.Fatal(err)
			}
			forged := map[string]any{}
			for key, value := range record {
				forged[key] = value
			}
			forged["root"] = ".github/checks/release"
			if err := schema.Validate(forged); err == nil {
				t.Fatal("retired routing accepted")
			}
			delete(forged, "validation_ref")
			if err := schema.Validate(forged); err == nil {
				t.Fatal("missing schema binding accepted")
			}
		})
	}
}

func TestPlatformReleaseContractRetainsItsSchema(t *testing.T) {
	root := repoRoot(t)
	base := filepath.Join(root, ".ptsip/contracts/release/Platform/Common")
	raw, err := os.ReadFile(filepath.Join(base, "release-contract_validation.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	id := document["$id"].(string)
	if err := compiler.AddResource(id, document); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile(id)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(filepath.Join(base, "release-contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	var instance any
	if err := json.Unmarshal(raw, &instance); err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(instance); err != nil {
		t.Fatal(err)
	}
}

func TestProfileArchiveMigrationPreservesHistoricalData(t *testing.T) {
	const current = ".ptsip/contracts/release/Platform/Common/profile/project-profile-contracts.yaml"
	const historical = ".ptsip/contracts/release/profile/project-profile-contracts.yaml"
	for _, member := range []string{current, historical} {
		t.Run(member, func(t *testing.T) {
			files := map[string][]byte{"src/ptsip/profiles/index.yaml": []byte("catalog"), "src/ptsip/profiles/main.ptsip.yaml": []byte("profile"), member: []byte("current: pp.2.00\n")}
			resources, err := release.ProfileResources(bundleTestArchive(t, files))
			if err != nil {
				t.Fatal(err)
			}
			if string(resources["@project-profile-contracts.yaml"]) != "current: pp.2.00\n" {
				t.Fatal("immutable archive member changed")
			}
			files[current] = []byte("current")
			files[historical] = []byte("historical")
			if _, err := release.ProfileResources(bundleTestArchive(t, files)); err == nil {
				t.Fatal("duplicate current/historical registry accepted")
			}
		})
	}
}

func TestReleaseRootContainsOnlyPlatformEntry(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join(repoRoot(t), ".ptsip/contracts/release"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() != "index.json" && entry.Name() != "Platform" {
			t.Fatalf("unmigrated root contract: %s", entry.Name())
		}
	}
}
