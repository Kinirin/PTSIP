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
}

type Composition struct {
	Schema       string                `toml:"schema_version"`
	PackageBase  string                `toml:"package_base"`
	ToolRef      string                `toml:"tool_version_ref"`
	Profile      string                `toml:"profile_version"`
	SpecSource   string                `toml:"specification_source"`
	SpecRevision string                `toml:"spec_revision"`
	Engines      map[string]string     `toml:"engines"`
	Published    *PublishedDeclaration `toml:"published"`
	Tool         string                `toml:"-"`
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
				for key := range section {
					if !allowed[key] {
						return Composition{}, fmt.Errorf("unregistered canonical composition field: %s", key)
					}
				}
			}
		}
	}
	c := document.Tool.PTSIP.Release
	c.Tool = document.Project.Version
	if document.Project.Name != "PTSIP" || c.Schema != "ptsip-current-composition/v1" || c.ToolRef != "project.version" {
		return c, fmt.Errorf("canonical Package composition declaration missing or unsupported")
	}
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(c.PackageBase) || !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:(a|b|rc)[1-9][0-9]*)?$`).MatchString(c.Tool) || !regexp.MustCompile(`^pp\.[0-9]+\.[0-9]+$`).MatchString(c.Profile) || len(c.Engines) == 0 {
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
	text = strings.TrimRight(text, "\r\n") + "\n" + block
	if _, err := ParseComposition([]byte(text)); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(text), 0644)
}
