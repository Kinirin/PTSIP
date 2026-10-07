package release_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/developer/tests/internal/testrepo"
	"go.yaml.in/yaml/v3"
)

func TestReleaseDistributionPublicProfileCatalogExactlyCoversAssets(t *testing.T) {
	root := testrepo.Root(t)
	raw, err := os.ReadFile(filepath.Join(root, "profiles", "index.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		Root     string `yaml:"root"`
		Profiles []struct {
			Resource string `yaml:"resource"`
		} `yaml:"profiles"`
	}
	if err := yaml.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	if catalog.Root != "profiles" || len(catalog.Profiles) == 0 {
		t.Fatalf("invalid public profile catalog: %#v", catalog)
	}

	registered := make([]string, 0, len(catalog.Profiles))
	for _, row := range catalog.Profiles {
		registered = append(registered, row.Resource)
	}
	sort.Strings(registered)

	matches, err := filepath.Glob(filepath.Join(root, "profiles", "*.ptsip.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	discovered := make([]string, 0, len(matches))
	for _, path := range matches {
		discovered = append(discovered, filepath.Base(path))
	}
	sort.Strings(discovered)

	if strings.Join(registered, "\\n") != strings.Join(discovered, "\\n") {
		t.Fatalf("public profile catalog mismatch\\nregistered=%v\\ndiscovered=%v", registered, discovered)
	}
}

func TestReleaseDistributionUsesCurrentProjectProfileSchemaAndManifestBindings(t *testing.T) {
	root := testrepo.Root(t)
	raw, err := os.ReadFile(filepath.Join(root, "registry", "project-profile-contracts.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var registry struct {
		Current   string `yaml:"current"`
		Contracts []struct {
			Version   string  `yaml:"version"`
			Lifecycle string  `yaml:"lifecycle"`
			Schema    *string `yaml:"schema"`
		} `yaml:"contracts"`
	}
	if err := yaml.Unmarshal(raw, &registry); err != nil {
		t.Fatal(err)
	}

	var schema string
	matches := 0
	for _, contract := range registry.Contracts {
		if contract.Version == registry.Current {
			matches++
			if contract.Lifecycle != "CURRENT" || contract.Schema == nil {
				t.Fatalf("current Project Profile contract is not distributable: %#v", contract)
			}
			schema = *contract.Schema
		}
	}
	if matches != 1 {
		t.Fatalf("current Project Profile contract resolves %d times", matches)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(schema))); err != nil {
		t.Fatalf("current Project Profile schema is missing: %s: %v", schema, err)
	}

	manifest, err := os.ReadFile(filepath.Join(root, "MANIFEST.in"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(manifest)
	for _, required := range []string{
		"recursive-include profiles *.ptsip.yaml",
		"include profiles/index.yaml",
		"include registry/project-profile-contracts.yaml",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("distribution manifest missing %q", required)
		}
	}
}
