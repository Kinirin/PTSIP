package releasecontroller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/verification/boundary"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/versionsource"
	"github.com/pelletier/go-toml/v2"
)

const ProfileConfigPath = "src/ptsip/schemas/config.json"
const packageRegistryPath = ".ptsip/engine/developer/Distribution/release/state/package-registry.json"

type ProfileRegistration struct {
	ID        string `json:"$id"`
	Version   string `json:"version"`
	Path      string `json:"path"`
	Digest    string `json:"schema_sha256"`
	Lifecycle string `json:"registered_lifecycle"`
}

// ReadProfileRegistrations selects the current contract from the product registry.
func ReadProfileRegistrations(root string) (map[string]ProfileRegistration, string, error) {
	var config struct {
		Storage struct {
			Profile map[string]ProfileRegistration `json:"profile"`
		} `json:"storage"`
	}
	if err := ReadJSON(root, ProfileConfigPath, &config); err != nil {
		return nil, "", fmt.Errorf("read Profile registrations: %w", err)
	}
	seenVersions := map[string]bool{}
	seenIDs := map[string]bool{}
	seenPaths := map[string]bool{}
	active := ""
	digestPattern := regexp.MustCompile(`^[0-9a-f]{64}$`)
	for selector, row := range config.Storage.Profile {
		if !sourceProfileVersionPattern.MatchString(row.Version) || seenVersions[row.Version] ||
			row.ID == "" || seenIDs[row.ID] || seenPaths[row.Path] ||
			!strings.HasSuffix(row.Path, ".schema.json") ||
			!digestPattern.MatchString(row.Digest) {
			return nil, "", fmt.Errorf("invalid Profile registration: %s", selector)
		}
		if _, err := SafePath(filepath.Join(root, "src", "ptsip", "schemas"), row.Path); err != nil {
			return nil, "", fmt.Errorf("invalid Profile registration path: %w", err)
		}
		seenVersions[row.Version], seenIDs[row.ID], seenPaths[row.Path] = true, true, true
		switch row.Lifecycle {
		case "ACTIVE":
			if path.Dir(row.Path) != "storage/profile" {
				return nil, "", fmt.Errorf("active Profile registration is outside storage/profile: %s", selector)
			}
			if active != "" {
				return nil, "", fmt.Errorf("multiple active Profile registrations")
			}
			active = row.Version
		case "LEGACY", "DEPRECATED":
			if path.Dir(row.Path) != "history/profile" {
				return nil, "", fmt.Errorf("historical Profile registration is outside history/profile: %s", selector)
			}
		default:
			return nil, "", fmt.Errorf("invalid Profile registration lifecycle: %s", selector)
		}
	}
	if active == "" {
		return nil, "", fmt.Errorf("active Profile registration is missing")
	}
	return config.Storage.Profile, active, nil
}

func verifyProfileProjection(root, expected string) error {
	registrations, active, err := ReadProfileRegistrations(root)
	if err != nil {
		return err
	}
	if active != expected {
		return fmt.Errorf("Profile current projection differs: declared %s, registered %s", expected, active)
	}
	row := registrations[versionsource.ProfileKey]
	file, err := SafePath(filepath.Join(root, "src", "ptsip", "schemas"), row.Path)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	if Digest(bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))) != row.Digest {
		return fmt.Errorf("Profile current schema digest differs: %s", row.Path)
	}
	value, err := jsondoc.Decode(raw)
	if err != nil {
		return err
	}
	schema, _ := value.(map[string]any)
	if schema["$id"] != row.ID {
		return fmt.Errorf("Profile current schema identity differs: %s", row.Path)
	}
	return nil
}

type PublishedDeclaration struct {
	Identity string `toml:"identity"`
	Digest   string `toml:"composition_sha256"`
	Source   string `toml:"source_sha"`
	Tag      string `toml:"release_tag"`
	PyPI     string `toml:"pypi_version,omitempty"`
}

type Composition struct {
	Schema          string                `toml:"schema_version"`
	PackageBase     string                `toml:"package_base"`
	ToolRef         string                `toml:"tool_version_ref"`
	Profile         string                `toml:"profile_version"`
	SpecSource      string                `toml:"specification_source"`
	SpecRevision    string                `toml:"spec_revision"`
	Engines         map[string]string     `toml:"engines"`
	Published       *PublishedDeclaration `toml:"published"`
	Tool            string                `toml:"tool_version"`
	PackageRevision int                   `toml:"package_revision"`
	PackageState    string                `toml:"package_state"`
	PackagePyPI     string                `toml:"-"`
}

// ParseComposition decodes original pyproject composition data for historical
// source inspection.
func ParseComposition(data []byte) (Composition, error) {
	var document struct {
		Project struct {
			Name    string `toml:"name"`
			Version string `toml:"version"`
		} `toml:"project"`
		Tool struct {
			PTSIP struct {
				Release Composition `toml:"release"`
			} `toml:"ptsip"`
		} `toml:"tool"`
	}
	if err := toml.Unmarshal(data, &document); err != nil {
		return Composition{}, fmt.Errorf("invalid canonical pyproject TOML: %w", err)
	}
	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return Composition{}, err
	}
	if tool, ok := raw["tool"].(map[string]any); ok {
		if ptsip, ok := tool["ptsip"].(map[string]any); ok {
			if section, ok := ptsip["release"].(map[string]any); ok {
				allowed := map[string]bool{"schema_version": true, "package_base": true, "tool_version_ref": true, "profile_version": true, "specification_source": true, "spec_revision": true, "engines": true, "published": true}
				if document.Tool.PTSIP.Release.Schema == "ptsip-current-composition/v2" {
					allowed["tool_version"] = true
					allowed["package_revision"] = true
					allowed["package_state"] = true
				}
				for key := range section {
					if !allowed[key] {
						return Composition{}, fmt.Errorf("unregistered canonical composition field: %s", key)
					}
				}
			}
		}
	}
	c := document.Tool.PTSIP.Release
	if c.Schema == "ptsip-current-composition/v1" {
		c.Tool = document.Project.Version
		if c.ToolRef != "project.version" {
			return c, fmt.Errorf("legacy Tool version binding invalid")
		}
	} else if c.Schema == "ptsip-current-composition/v2" {
		if c.ToolRef != "tool.ptsip.release.tool_version" {
			return c, fmt.Errorf("Tool canonical version binding invalid")
		}
		pypi, err := PackageDistributionVersion(c.PackageBase, c.PackageRevision, c.PackageState)
		if err != nil {
			return c, err
		}
		// PEP 440 accepts both 0.3.8-b7 and the normalized 0.3.8b7.
		// src/VERSION owns the exact canonical spelling in repository builds.
		projectVersion := pypi
		if c.PackageRevision == 0 && c.PackageState != "" {
			projectVersion = c.PackageBase + "-" + c.PackageState
		}
		if document.Project.Version != pypi && document.Project.Version != projectVersion {
			return c, fmt.Errorf("project.version must be the Package projection, not the Tool version")
		}
		c.PackagePyPI = pypi
	} else {
		return c, fmt.Errorf("unregistered current composition schema")
	}
	if document.Project.Name != "PTSIP" {
		return c, fmt.Errorf("canonical Package composition declaration missing or unsupported")
	}
	_, toolErr := ToolDistributionVersion(c.Tool)
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(c.PackageBase) || toolErr != nil || !regexp.MustCompile(`^pp\.[0-9]+\.[0-9]+$`).MatchString(c.Profile) || len(c.Engines) == 0 {
		return c, fmt.Errorf("canonical component version declaration invalid")
	}
	if !strings.HasPrefix(c.SpecSource, "https://") || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(c.SpecRevision) {
		return c, fmt.Errorf("canonical Specification binding invalid")
	}
	for id, version := range c.Engines {
		if !regexp.MustCompile(`^[a-z][a-z0-9-]*$`).MatchString(id) || !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(version) {
			return c, fmt.Errorf("canonical engine identity/version invalid")
		}
	}
	if c.Published != nil {
		p := c.Published
		if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(p.Digest) || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(p.Source) || !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+\.[0-9a-f]{6}$`).MatchString(p.Identity) || !strings.HasSuffix(p.Identity, "."+p.Digest[:6]) || p.Tag != "ptsip-release-"+p.Source {
			return c, fmt.Errorf("published snapshot declaration invalid")
		}
	}
	return c, nil
}

func ReadComposition(root string) (Composition, error) {
	contract, err := ReadContract(root)
	if err != nil {
		return Composition{}, err
	}
	return readComposition(root, contract)
}

func readComposition(root string, contract Contract) (Composition, error) {
	versions, err := ReadSourceVersions(root)
	if err != nil {
		return Composition{}, err
	}
	base, state, ok := versionsource.PackageParts(versions.PackageVersion)
	if !ok {
		return Composition{}, fmt.Errorf("invalid Package source version")
	}
	if contract.Publication.Registry != packageRegistryPath {
		return Composition{}, fmt.Errorf("unregistered Package state registry")
	}
	registry, err := readPackageState(root, contract.Publication.Registry)
	if err != nil {
		return Composition{}, err
	}
	composition := Composition{
		Schema: contract.Schema, PackageBase: base, PackageState: state,
		ToolRef: versionsource.Path + "#tool.version", Tool: versions.ToolVersion,
		Profile: versions.ProfileVersion, Engines: versions.Engines,
		SpecSource: contract.Specification.Source, SpecRevision: contract.Specification.Revision,
		PackagePyPI: versions.DistributionVersion(),
	}
	if registry.Latest != nil {
		snapshot, err := ResolvePackage(registry, *registry.Latest)
		if err != nil {
			return Composition{}, err
		}
		composition.Published = &PublishedDeclaration{
			Identity: snapshot.Version, Digest: snapshot.Digest, Source: snapshot.SourceSHA,
			Tag: "ptsip-release-" + snapshot.SourceSHA,
		}
	}
	return composition, nil
}

func readPackageState(root, ref string) (Registry, error) {
	file, err := SafePath(root, ref)
	if err != nil {
		return Registry{}, err
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return Registry{}, err
	}
	if _, err := jsondoc.Decode(raw); err != nil {
		return Registry{}, err
	}
	var registry Registry
	if err := json.Unmarshal(raw, &registry); err != nil {
		return Registry{}, err
	}
	return registry, ValidateRegistry(registry)
}

func VerifyCompositionProjections(root string, c Composition, contract Contract) error {
	if c.Schema != boundary.CompositionVersion || contract.Schema != boundary.CompositionVersion ||
		contract.Sources.EngineRegistry != currentEngineIndex || contract.Sources.SchemaConfig != ProfileConfigPath {
		return fmt.Errorf("unregistered release composition or source inventory")
	}
	versions, err := ReadSourceVersions(root)
	if err != nil {
		return err
	}
	base, state, _ := versionsource.PackageParts(versions.PackageVersion)
	if c.PackageBase != base || c.PackageState != state || c.PackageRevision != 0 || c.PackagePyPI != versions.DistributionVersion() {
		return fmt.Errorf("Package projection differs from src/VERSION")
	}
	if c.Tool != versions.ToolVersion {
		return fmt.Errorf("Tool projection differs from src/VERSION")
	}
	if c.Profile != versions.ProfileVersion {
		return fmt.Errorf("Profile projection differs from src/VERSION")
	}
	if err := verifyProfileProjection(root, c.Profile); err != nil {
		return err
	}
	if _, err := SyncPackageVersion(root, true); err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	if err != nil {
		return err
	}
	var project struct {
		Project struct {
			Name         string   `toml:"name"`
			Dependencies []string `toml:"dependencies"`
		} `toml:"project"`
	}
	if err := toml.Unmarshal(data, &project); err != nil {
		return err
	}
	if project.Project.Name != "PTSIP" {
		return fmt.Errorf("integrated Package project name differs from source composition")
	}
	for _, dep := range project.Project.Dependencies {
		normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(dep, "_", "-"), ".", "-"))
		if strings.HasPrefix(normalized, "ptsip-engine-") {
			return fmt.Errorf("integrated PTSIP Wheel must not require separate engine pip packages")
		}
	}
	var index struct {
		Entries []struct{ Kind, Version string } `json:"official_engine_entries"`
	}
	if err := ReadJSON(root, contract.Sources.EngineRegistry, &index); err != nil {
		return err
	}
	if len(index.Entries) != len(c.Engines) {
		return fmt.Errorf("canonical engine inventory and index diverge")
	}
	seen := map[string]bool{}
	for _, e := range index.Entries {
		id := strings.ToLower(e.Kind)
		if seen[id] || c.Engines[id] == "" || c.Engines[id] != e.Version || c.Engines[id] != versions.Engines[id] {
			return fmt.Errorf("engine index projection differs: %s", id)
		}
		seen[id] = true
	}
	return nil
}
