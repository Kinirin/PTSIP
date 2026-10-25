package releasetest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"
)

// Project Profile verification is release-local, source-only evidence.
// No developer/automation command, retired developer-policy resolver, or
// root-family-policy schema may be consulted by this verifier.
const (
	profileContractPath = "registry/project-profile-contracts.yaml"
	profileProjectionPath = "src/ptsip/specdata/project-profile-contracts.yaml"
	profileSourcePath = "src/ptsip/profiles"
)

var profileVersionPattern = regexp.MustCompile(`^pp\.[0-9]+\.[0-9]+$`)
var profileResourcePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*\.ptsip\.yaml$`)
var profileIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type profileContractRow struct {
	Version    string   `yaml:"version"`
	Lifecycle  string   `yaml:"lifecycle"`
	Operations []string `yaml:"operations"`
	Schema     *string  `yaml:"schema"`
	Baseline   string   `yaml:"baseline"`
}

type profileContractSource struct {
	Root                string `yaml:"root"`
	Catalog             string `yaml:"catalog"`
	HistoryRoot         string `yaml:"history_root"`
	HistoryDistribution string `yaml:"history_distribution"`
}

type profileContractRegistry struct {
	Schema      string                `yaml:"schema_version"`
	Authority   string                `yaml:"authority"`
	Current     string                `yaml:"current"`
	Source      profileContractSource `yaml:"profile_source"`
	Contracts   []profileContractRow  `yaml:"contracts"`
	Transitions []struct {
		From string `yaml:"from"`
		To   string `yaml:"to"`
		Kind string `yaml:"kind"`
	} `yaml:"transitions"`
}

type publicProfileEntry struct {
	ID                string `yaml:"id"`
	Resource          string `yaml:"resource"`
	Contract          string `yaml:"contract"`
	ResponsibilityMode string `yaml:"responsibility_mode"`
	Role              string `yaml:"profile_role"`
	Materialization   string `yaml:"materialization"`
}

type publicProfileCatalog struct {
	Schema    string               `yaml:"schema_version"`
	Authority string               `yaml:"authority"`
	Root      string               `yaml:"root"`
	Profiles  []publicProfileEntry `yaml:"profiles"`
}

type closedProfileSchemaLoader struct{}

func (closedProfileSchemaLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("PROFILE_EXTERNAL_SCHEMA_FORBIDDEN: %s", url)
}

func readProfileYAML(root, path string, out any) error {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	reader := yaml.NewDecoder(bytes.NewReader(data))
	reader.KnownFields(true)
	if err := reader.Decode(out); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func profileSchema(root, reference string) (*jsonschema.Schema, error) {
	if !regexp.MustCompile(`^schemas/ptsip-profile-pp-[0-9]+\.[0-9]+\.schema\.json$`).MatchString(reference) {
		return nil, fmt.Errorf("PROFILE_SCHEMA_UNREGISTERED: %s", reference)
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(reference)))
	if err != nil {
		return nil, err
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, err
	}
	id, ok := document["$id"].(string)
	if !ok || !strings.HasPrefix(id, "https://raw.githubusercontent.com/Kinirin/PTSIP/") {
		return nil, fmt.Errorf("PROFILE_SCHEMA_ID_INVALID: %s", reference)
	}
	compiler := jsonschema.NewCompiler()
	compiler.UseLoader(closedProfileSchemaLoader{})
	if err := compiler.AddResource(id, document); err != nil {
		return nil, err
	}
	return compiler.Compile(id)
}

func validateProfileContract(root string, registry profileContractRegistry) (map[string]profileContractRow, error) {
	if registry.Schema != "ptsip-project-profile-contract-registry/v1" ||
		registry.Authority != "PTSIP_PROJECT_PROFILE_CONTRACT_IDENTITY" ||
		!profileVersionPattern.MatchString(registry.Current) ||
		registry.Source != (profileContractSource{
			Root: profileSourcePath,
			Catalog: profileSourcePath + "/index.yaml",
			HistoryRoot: profileSourcePath + "/history",
			HistoryDistribution: "SOURCE_ONLY",
		}) || len(registry.Contracts) == 0 {
		return nil, fmt.Errorf("PROFILE_REGISTRY_CONTRACT_INVALID")
	}
	contracts := map[string]profileContractRow{}
	var currentCount int
	for _, row := range registry.Contracts {
		if !profileVersionPattern.MatchString(row.Version) || contracts[row.Version].Version != "" ||
			len(row.Operations) == 0 {
			return nil, fmt.Errorf("PROFILE_CONTRACT_IDENTITY_INVALID: %s", row.Version)
		}
		operations := map[string]bool{}
		for _, op := range row.Operations {
			if operations[op] || !map[string]bool{
				"IDENTIFY": true, "VALIDATE": true, "ANALYZE": true,
				"MIGRATE_SOURCE": true, "CREATE_TARGET": true,
			}[op] {
				return nil, fmt.Errorf("PROFILE_CONTRACT_OPERATIONS_INVALID: %s", row.Version)
			}
			operations[op] = true
		}
		switch row.Lifecycle {
		case "LEGACY_COMPATIBILITY_ONLY":
			if row.Version == registry.Current || row.Schema != nil ||
				len(row.Operations) != 1 || row.Operations[0] != "IDENTIFY" || row.Baseline != "" {
				return nil, fmt.Errorf("PROFILE_LEGACY_CONTRACT_INVALID: %s", row.Version)
			}
		case "CURRENT", "SUPERSEDED":
			if row.Schema == nil || *row.Schema !=
				"schemas/ptsip-profile-"+row.Version+".schema.json" ||
				row.Baseline != registry.Source.HistoryRoot+"/"+row.Version {
				return nil, fmt.Errorf("PROFILE_CONTRACT_BINDING_INVALID: %s", row.Version)
			}
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(row.Baseline))); err != nil {
				return nil, fmt.Errorf("PROFILE_BASELINE_MISSING: %s: %w", row.Version, err)
			}
			schema, err := profileSchema(root, *row.Schema)
			if err != nil || schema == nil {
				return nil, fmt.Errorf("PROFILE_SCHEMA_INVALID: %s: %v", row.Version, err)
			}
			if row.Lifecycle == "CURRENT" {
				currentCount++
				if row.Version != registry.Current {
					return nil, fmt.Errorf("PROFILE_CURRENT_CONTRACT_MISMATCH")
				}
			} else if row.Version == registry.Current {
				return nil, fmt.Errorf("PROFILE_CURRENT_LIFECYCLE_INVALID")
			}
		default:
			return nil, fmt.Errorf("PROFILE_LIFECYCLE_UNREGISTERED: %s", row.Lifecycle)
		}
		contracts[row.Version] = row
	}
	if currentCount != 1 {
		return nil, fmt.Errorf("PROFILE_CURRENT_CONTRACT_AMBIGUOUS")
	}
	transitions := map[string]bool{}
	for _, tr := range registry.Transitions {
		key := tr.From + "|" + tr.To
		if transitions[key] || tr.From == tr.To || contracts[tr.From].Version == "" ||
			contracts[tr.To].Version == "" ||
			(tr.Kind != "IDENTITY_ONLY" && tr.Kind != "SEMANTIC_MIGRATION") {
			return nil, fmt.Errorf("PROFILE_TRANSITION_INVALID: %s", key)
		}
		transitions[key] = true
	}
	return contracts, nil
}

// VerifyProjectProfiles establishes the current registry, catalog, schemas and
// immutable baseline before the source-bound Profile distribution is archived.
// It does not grant policy or publication authority.
func VerifyProjectProfiles(root, expectedVersion string) error {
	var registry, projection profileContractRegistry
	if err := readProfileYAML(root, profileContractPath, &registry); err != nil {
		return err
	}
	if err := readProfileYAML(root, profileProjectionPath, &projection); err != nil {
		return err
	}
	if !reflect.DeepEqual(registry, projection) {
		return fmt.Errorf("PROFILE_EMBEDDED_REGISTRY_PROJECTION_MISMATCH")
	}
	if expectedVersion == "" || registry.Current != expectedVersion {
		return fmt.Errorf("PROFILE_RELEASE_VERSION_MISMATCH: expected %s, registered %s", expectedVersion, registry.Current)
	}
	contracts, err := validateProfileContract(root, registry)
	if err != nil {
		return err
	}
	var catalog publicProfileCatalog
	if err := readProfileYAML(root, registry.Source.Catalog, &catalog); err != nil {
		return err
	}
	if catalog.Schema != "ptsip-public-profile-catalog/v1" ||
		catalog.Authority != "PTSIP_PUBLIC_PROFILE_CATALOG" ||
		catalog.Root != registry.Source.Root || len(catalog.Profiles) == 0 {
		return fmt.Errorf("PROFILE_PUBLIC_CATALOG_INVALID")
	}
	current := contracts[registry.Current]
	compiled, err := profileSchema(root, *current.Schema)
	if err != nil {
		return err
	}
	ids, resources := map[string]bool{}, map[string]bool{}
	baseline := filepath.Join(root, filepath.FromSlash(current.Baseline))
	for _, entry := range catalog.Profiles {
		if !profileIDPattern.MatchString(entry.ID) ||
			!profileResourcePattern.MatchString(entry.Resource) ||
			ids[entry.ID] || resources[entry.Resource] ||
			entry.Contract != registry.Current ||
			!map[string]bool{"explicit": true, "template": true, "hybrid": true}[entry.ResponsibilityMode] ||
			entry.Role != "DISTRIBUTED_EXAMPLE" ||
			entry.Materialization != "PROJECT_PATH_RESOLUTION_REQUIRED" {
			return fmt.Errorf("PROFILE_CATALOG_ENTRY_INVALID: %s", entry.Resource)
		}
		ids[entry.ID], resources[entry.Resource] = true, true
		source := filepath.Join(root, filepath.FromSlash(registry.Source.Root), entry.Resource)
		currentBytes, err := os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("PROFILE_SOURCE_MISSING: %s: %w", entry.Resource, err)
		}
		originalBytes, err := os.ReadFile(filepath.Join(baseline, entry.Resource))
		if err != nil || !bytes.Equal(originalBytes, currentBytes) {
			return fmt.Errorf("PROFILE_IMMUTABLE_BASELINE_MISMATCH: %s: %v", entry.Resource, err)
		}
		var profile map[string]any
		if err := yaml.Unmarshal(currentBytes, &profile); err != nil {
			return fmt.Errorf("PROFILE_SOURCE_YAML_INVALID: %s: %w", entry.Resource, err)
		}
		if err := compiled.Validate(profile); err != nil {
			return fmt.Errorf("PROFILE_SOURCE_SCHEMA_INVALID: %s: %w", entry.Resource, err)
		}
		ptsip, _ := profile["ptsip"].(map[string]any)
		responsibility, _ := profile["responsibility_map"].(map[string]any)
		if ptsip["version"] != registry.Current ||
			ptsip["profile_role"] != entry.Role ||
			responsibility["mode"] != entry.ResponsibilityMode {
			return fmt.Errorf("PROFILE_CATALOG_PROJECTION_INVALID: %s", entry.Resource)
		}
	}
	for _, directory := range []string{
		filepath.Join(root, filepath.FromSlash(registry.Source.Root)),
		baseline,
	} {
		names, err := filepath.Glob(filepath.Join(directory, "*.ptsip.yaml"))
		if err != nil {
			return err
		}
		discovered := make([]string, 0, len(names))
		for _, name := range names {
			discovered = append(discovered, filepath.Base(name))
		}
		sort.Strings(discovered)
		if len(discovered) != len(resources) {
			return fmt.Errorf("PROFILE_CATALOG_COVERAGE_MISMATCH: %s", directory)
		}
		for _, name := range discovered {
			if !resources[name] {
				return fmt.Errorf("PROFILE_CATALOG_UNDECLARED_RESOURCE: %s", name)
			}
		}
	}
	return nil
}
