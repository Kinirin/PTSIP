package releasecontroller

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"go.yaml.in/yaml/v3"
)

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
				if document.Tool.PTSIP.Release.Schema == "ptsip-current-composition/v2" || document.Tool.PTSIP.Release.Schema == "ptsip-current-composition/v3" {
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
		if document.Project.Version != pypi {
			return c, fmt.Errorf("project.version must be the Package projection, not the Tool version")
		}
		c.PackagePyPI = pypi
	} else if c.Schema == "ptsip-current-composition/v3" {
		if c.ToolRef != "tool.ptsip.release.tool_version" || document.Project.Version != "0.3.8b2" || c.Tool != "T.0.3.8.b2" {
			return c, fmt.Errorf("explicit beta2 Package/Tool projection differs")
		}
		c.PackagePyPI = document.Project.Version
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
	b, err := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	if err != nil {
		return Composition{}, err
	}
	return ParseComposition(b)
}

func VerifyCompositionProjections(root string, c Composition, contract Contract) error {
	if c.PackagePyPI != "" {
		data, err := os.ReadFile(filepath.Join(root, "pyproject.toml"))
		if err != nil {
			return err
		}
		var project struct {
			Project struct {
				Dependencies []string `toml:"dependencies"`
			} `toml:"project"`
		}
		if err := toml.Unmarshal(data, &project); err != nil {
			return err
		}
		deps := map[string]bool{}
		for _, d := range project.Project.Dependencies {
			if strings.HasPrefix(d, "ptsip-engine-") {
				if deps[d] {
					return fmt.Errorf("duplicate native dependency")
				}
				deps[d] = true
			}
		}
		for id, v := range c.Engines {
			key := "ptsip-engine-" + id + "==" + v
			if !deps[key] {
				return fmt.Errorf("mandatory Package engine dependency missing or optional: %s", id)
			}
			delete(deps, key)
		}
		if len(deps) != 0 {
			return fmt.Errorf("unregistered or stale Package engine dependency")
		}
	}
	var index struct {
		Entries []struct{ Kind, Version string } `json:"official_engine_entries"`
	}
	if err := ReadJSON(root, contract.EngineRegistry, &index); err != nil {
		return err
	}
	if len(index.Entries) != len(c.Engines) {
		return fmt.Errorf("canonical engine inventory and index diverge")
	}
	seen := map[string]bool{}
	for _, e := range index.Entries {
		id := strings.ToLower(e.Kind)
		if seen[id] || c.Engines[id] == "" || c.Engines[id] != e.Version {
			return fmt.Errorf("engine index projection differs: %s", id)
		}
		seen[id] = true
	}
	var registry struct {
		Current string `yaml:"current"`
	}
	for _, ref := range []string{contract.ProfileRegistry, "src/ptsip/specdata/project-profile-contracts.yaml"} {
		p, err := SafePath(root, ref)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if yaml.Unmarshal(b, &registry) != nil || registry.Current != c.Profile {
			return fmt.Errorf("Profile current projection differs: %s", ref)
		}
	}
	constants, err := os.ReadFile(filepath.Join(root, "src/ptsip/constants.py"))
	if err != nil {
		return err
	}
	for name, expected := range map[string]string{"TOOL_VERSION": c.Tool, "SPEC_SOURCE": c.SpecSource, "SPEC_REVISION": c.SpecRevision} {
		match := regexp.MustCompile(`(?m)^` + name + `\s*=\s*"([^"]+)"`).FindSubmatch(constants)
		if len(match) != 2 || string(match[1]) != expected {
			return fmt.Errorf("runtime identity projection differs: %s", name)
		}
	}
	return nil
}

// Publication records describe the observed original source snapshot, not the
// source of the subsequent metadata commit. This avoids a self-referential hash.
func RecordPublishedDeclaration(root string, s Snapshot) error {
	p := filepath.Join(root, "pyproject.toml")
	b, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	c, err := ParseComposition(b)
	if err != nil {
		return err
	}
	if c.PackageBase != strings.TrimSuffix(s.Version, "."+s.Digest[:6]) {
		return fmt.Errorf("published Package numeric base differs from canonical declaration")
	}
	text := string(b)
	section := regexp.MustCompile(`(?ms)^\[tool\.ptsip\.release\.published\][^\[]*`)
	text = section.ReplaceAllString(text, "")
	block := fmt.Sprintf("\n[tool.ptsip.release.published]\nidentity = %q\ncomposition_sha256 = %q\nsource_sha = %q\nrelease_tag = %q\n", s.Version, s.Digest, s.SourceSHA, "ptsip-release-"+s.SourceSHA)
	if c.PackagePyPI != "" {
		block += fmt.Sprintf("pypi_version = %q\n", c.PackagePyPI)
	}
	text = strings.TrimRight(text, "\r\n") + "\n" + block
	if _, err := ParseComposition([]byte(text)); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(text), 0644)
}
