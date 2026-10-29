package boundary

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func rewritePlatformIndex(t *testing.T, root string, edit func(map[string]any)) {
	t.Helper()
	file := filepath.Join(root, ".ptsip/contracts/release/index.json")
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	edit(data)
	raw, err = json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, raw, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestPlatformCatalogRoutesOnlyCurrentDistributionServices(t *testing.T) {
	root := fixture(t)
	if err := checkPlatforms(root); err != nil {
		t.Fatal(err)
	}
	for _, route := range expectedPlatforms() {
		if _, err := os.Stat(filepath.Join(root, route.Validation)); err != nil {
			t.Fatal(err)
		}
	}
	if err := RequirePublication(root); err == nil {
		t.Fatal("platform routing admitted publication")
	}
}

func TestPlatformCatalogRejectsMissingUnknownAndReroutedRecords(t *testing.T) {
	edits := map[string]func(map[string]any){
		"dimension":          func(data map[string]any) { data["platform_dimension"] = "RUNTIME_OS" },
		"missing_platform":   func(data map[string]any) { data["platforms"] = data["platforms"].([]any)[:2] },
		"unknown_platform":   func(data map[string]any) { data["platforms"].([]any)[0].(map[string]any)["platform"] = "Other" },
		"duplicate_platform": func(data map[string]any) { routes := data["platforms"].([]any); routes[1] = routes[0] },
		"unknown_field": func(data map[string]any) {
			data["platforms"].([]any)[0].(map[string]any)["publication_authorized"] = true
		},
		"wrong_root": func(data map[string]any) {
			data["platforms"].([]any)[0].(map[string]any)["root"] = ".ptsip/contracts/release/"
		},
		"wrong_schema": func(data map[string]any) {
			data["platforms"].([]any)[0].(map[string]any)["validation_ref"] = platformRoot + "GitHub/GitHub_validation.json"
		},
		"unknown_contract": func(data map[string]any) {
			data["platforms"].([]any)[0].(map[string]any)["contracts"].(map[string]any)["unregistered"] = platformRoot + "Common/new.json"
		},
		"rerouted_contract": func(data map[string]any) {
			data["platforms"].([]any)[0].(map[string]any)["contracts"].(map[string]any)["admission"] = ".ptsip/contracts/release/admission.json"
		},
	}
	for name, edit := range edits {
		t.Run(name, func(t *testing.T) {
			root := fixture(t)
			rewritePlatformIndex(t, root, edit)
			if err := checkPlatforms(root); err == nil {
				t.Fatal("unregistered platform routing accepted")
			}
		})
	}
}

func TestPlatformResourcesCannotBeMissingOrReplacedByOldRootFiles(t *testing.T) {
	for _, relative := range []string{
		platformRoot + "GitHub/publication-evidence.json",
		platformRoot + "PyPI/PyPI_validation.json",
		platformRoot + "Common/profile/project-profile-contracts.yaml",
	} {
		t.Run(relative, func(t *testing.T) {
			root := fixture(t)
			file := filepath.Join(root, relative)
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(file); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, ".ptsip/contracts/release", filepath.Base(file)), raw, 0644); err != nil {
				t.Fatal(err)
			}
			if err := checkPlatforms(root); err == nil {
				t.Fatal("old root resource substituted for the current platform route")
			}
		})
	}
}

func TestPlatformCatalogRejectsUnregisteredFilesAndDirectories(t *testing.T) {
	for _, relative := range []string{platformRoot + "Unknown", platformRoot + "Common/new.json"} {
		t.Run(relative, func(t *testing.T) {
			root := fixture(t)
			file := filepath.Join(root, relative)
			if filepath.Ext(file) == ".json" {
				if err := os.WriteFile(file, []byte("{}"), 0644); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Mkdir(file, 0755); err != nil {
				t.Fatal(err)
			}
			if err := checkPlatforms(root); err == nil {
				t.Fatal("unregistered platform resource accepted")
			}
		})
	}
}

func TestPlatformRoutingSchemaMustPreserveItsDeclaredBinding(t *testing.T) {
	root := fixture(t)
	file := filepath.Join(root, platformRoot+"GitHub/GitHub_validation.json")
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	schema["properties"].(map[string]any)["root"].(map[string]any)["const"] = platformRoot + "Common/"
	raw, err = json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, raw, 0644); err != nil {
		t.Fatal(err)
	}
	if err := checkPlatforms(root); err == nil {
		t.Fatal("mismatched routing validation schema accepted")
	}
}
