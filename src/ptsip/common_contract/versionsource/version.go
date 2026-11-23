// Package versionsource reads the current, owner-bound src/VERSION layout.
package versionsource

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
	"github.com/pelletier/go-toml/v2"
)

const Path = "src/VERSION"
const ProfileKey = "ptsip.project-profile"
const PolicyKey = "ptsip.policy.v4"

var engineIDs = []string{"context", "storage", "validation", "discovery", "execution", "projection"}
var rbacKeys = []string{"RA_list", "RAC_list", "AEC_list"}
var schemaKeys = []string{PolicyKey, ProfileKey}

const semver = `(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)`
const pre = `(?:a|b|rc)[1-9][0-9]*`

var packagePattern = regexp.MustCompile(`^(` + semver + `)(?:-(` + pre + `))?$`)
var toolPattern = regexp.MustCompile(`^T\.` + semver + `(?:\.` + pre + `)?$`)
var enginePattern = regexp.MustCompile(`^` + semver + `$`)
var rbacPattern = regexp.MustCompile(`^v\.[1-9][0-9]*$`)
var revisionPattern = regexp.MustCompile(`^v[1-9][0-9]*$`)
var profilePattern = regexp.MustCompile(`^pp\.[0-9]+\.[0-9]+$`)

type Versions struct {
	PackageVersion string            `json:"package_version"`
	ToolVersion    string            `json:"tool_version"`
	ProfileVersion string            `json:"profile_version"`
	RBAC           map[string]string `json:"RBAC"`
	CommonContract string            `json:"common_contract_revision"`
	Engines        map[string]string `json:"engines"`
	Schemas        map[string]string `json:"schemas"`
}

func EngineIDs() []string { return slices.Clone(engineIDs) }

func PackageParts(value string) (base, state string, ok bool) {
	match := packagePattern.FindStringSubmatch(value)
	if len(match) != 3 {
		return "", "", false
	}
	return match[1], match[2], true
}

func (v Versions) DistributionVersion() string {
	return strings.Replace(v.PackageVersion, "-", "", 1)
}

func Read(root string) (Versions, error) {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(Path)))
	if err != nil {
		return Versions{}, fmt.Errorf("read %s: %w", Path, err)
	}
	return Parse(raw)
}

func Parse(raw []byte) (Versions, error) {
	var doc map[string]map[string]string
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return Versions{}, fmt.Errorf("invalid %s TOML: %w", Path, err)
	}
	tables := []struct {
		name    string
		keys    []string
		pattern *regexp.Regexp
	}{
		{"package", []string{"version"}, packagePattern},
		{"tool", []string{"version"}, toolPattern},
		{"RBAC", rbacKeys, rbacPattern},
		{"common_contract", []string{"registry_revision"}, revisionPattern},
		{"engines", engineIDs, enginePattern},
		{"schemas", schemaKeys, nil},
	}
	if len(doc) != len(tables) {
		return Versions{}, fmt.Errorf("%s requires exactly six registered tables", Path)
	}
	for _, table := range tables {
		values := doc[table.name]
		if len(values) != len(table.keys) {
			return Versions{}, fmt.Errorf("invalid or missing %s table in %s", table.name, Path)
		}
		for _, key := range table.keys {
			pattern := table.pattern
			if table.name == "schemas" {
				pattern = revisionPattern
				if key == ProfileKey {
					pattern = profilePattern
				}
			}
			if !pattern.MatchString(values[key]) {
				return Versions{}, fmt.Errorf("invalid or missing %s.%s in %s", table.name, key, Path)
			}
		}
	}
	return Versions{
		PackageVersion: doc["package"]["version"], ToolVersion: doc["tool"]["version"],
		ProfileVersion: doc["schemas"][ProfileKey], CommonContract: doc["common_contract"]["registry_revision"],
		RBAC: doc["RBAC"], Engines: doc["engines"], Schemas: doc["schemas"],
	}, nil
}

// VerifyOwners checks the initial version projections without modifying current
// contracts. A version lacking an aligned registered resource is unavailable.
func (v Versions) VerifyOwners(config, registry []byte) error {
	catalog, err := jsondoc.Decode(config)
	if err != nil {
		return fmt.Errorf("schema config: %w", err)
	}
	object := func(value any, keys ...string) any {
		for _, key := range keys {
			doc, _ := value.(map[string]any)
			value = doc[key]
		}
		return value
	}
	for _, key := range rbacKeys {
		if expected := v.RBAC[key]; expected == "" || object(catalog, "storage", "Responsibility-Authority", key, "version") != expected {
			return fmt.Errorf("RBAC version differs from registered %s", key)
		}
	}
	if object(catalog, "Policy", PolicyKey, "version") != v.Schemas[PolicyKey] {
		return fmt.Errorf("Policy version differs from registered %s", PolicyKey)
	}
	if object(catalog, "storage", "profile", ProfileKey, "version") != v.ProfileVersion ||
		object(catalog, "storage", "profile", ProfileKey, "registered_lifecycle") != "ACTIVE" {
		return fmt.Errorf("Profile current projection differs from %s", ProfileKey)
	}
	common, err := jsondoc.Decode(registry)
	if err != nil {
		return fmt.Errorf("common contract manifest: %w", err)
	}
	if object(common, "revision") != v.CommonContract {
		return fmt.Errorf("Common Contract revision differs from registered manifest")
	}
	return nil
}
