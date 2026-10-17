package releasecontroller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

func ComponentSourceChanges(root, kind, origin, target string) (string, error) {
	var contract struct {
		Reuse map[string]json.RawMessage `json:"component_reuse"`
	}
	if err := ReadJSON(root, ContractPath, &contract); err != nil {
		return "", err
	}
	raw, ok := contract.Reuse[kind]
	var rule struct {
		Boundaries []string `json:"source_boundaries"`
	}
	if !ok || json.Unmarshal(raw, &rule) != nil || len(rule.Boundaries) == 0 {
		return "", fmt.Errorf("component producer boundary missing: %s", kind)
	}
	args := []string{"git", "diff", "--name-only", origin, target, "--"}
	for _, ref := range rule.Boundaries {
		if _, err := SafePath(root, ref); err != nil {
			return "", err
		}
		args = append(args, ref)
	}
	delta, err := Run(root, args...)
	if err != nil || len(bytes.TrimSpace(delta)) != 0 {
		return strings.TrimSpace(string(delta)), err
	}
	if kind == "tool" {
		declaration := func(source string) ([]byte, error) {
			b, err := Run(root, "git", "show", source+":pyproject.toml")
			if err != nil {
				return nil, err
			}
			var p map[string]any
			if err := toml.Unmarshal(b, &p); err != nil {
				return nil, err
			}
			if t, ok := p["tool"].(map[string]any); ok {
				if ptsip, ok := t["ptsip"].(map[string]any); ok {
					if release, ok := ptsip["release"].(map[string]any); ok {
						delete(release, "published")
					}
				}
			}
			return json.Marshal(p)
		}
		a, err := declaration(origin)
		if err != nil {
			return "", err
		}
		b, err := declaration(target)
		if err != nil {
			return "", err
		}
		if !bytes.Equal(a, b) {
			return "pyproject.toml", nil
		}
	}
	return "", nil
}

// Complete Package releases retain unchanged Tool/Profile files as well as
// unchanged engines. A new archive timestamp is not a new component version.
func ReusePublishedComponents(root string, s Selection, registry Registry, components []PublishedComponent, locations map[string]string) ([]PublishedComponent, error) {
	if len(registry.Snapshots) == 0 {
		return components, nil
	}
	prior := registry.Snapshots[len(registry.Snapshots)-1]
componentsLoop:
	for i, c := range components {
		if c.Kind != "tool" && c.Kind != "profiles" {
			continue
		}
		for _, old := range prior.Components {
			if old.ID != c.ID || old.Kind != c.Kind || old.Version != c.Version {
				continue
			}
			delta, err := ComponentSourceChanges(root, c.Kind, old.SourceSHA, s.SourceSHA)
			if err != nil {
				return nil, err
			}
			if delta != "" {
				if authorizedSingleProjectToolRebuild(root, s, c, old, delta) {
					continue componentsLoop
				}
				return nil, fmt.Errorf("published %s %s producer changed (%s); explicit new component version required", c.ID, c.Version, delta)
			}
			for _, file := range old.Files {
				b, err := DownloadArtifact(file.URL, file.SHA256)
				if err != nil {
					return nil, err
				}
				ref := ".artifacts/release/reused/" + c.ID + "/" + file.Name
				p, err := SafePath(root, ref)
				if err != nil {
					return nil, err
				}
				if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
					return nil, err
				}
				if err := os.WriteFile(p, b, 0644); err != nil {
					return nil, err
				}
				platform := file.Platform
				if platform == "any" {
					platform = "linux-x64"
				}
				locations[c.ID+"/"+platform+"/"+file.Name] = p
			}
			components[i] = old
			break
		}
	}
	return components, nil
}

// The owner retained b3 for the first PTSIP-only PyPI publication. The previous
// GitHub carrier is preserved. This exception admits only packaging inputs from
// that exact prior Tool source; it never changes or rebuilds an engine artifact.
func authorizedSingleProjectToolRebuild(root string, s Selection, current, old PublishedComponent, delta string) bool {
	if s.Type != "PTSIP_PACKAGE" || current.Kind != "tool" || current.Version != "T.0.3.8.b3" || current.SourceSHA != s.SourceSHA || old.SourceSHA != "916333f9aa0f9589272911818d8352fb0cae09e1" {
		return false
	}
	var auth struct {
		Migration struct {
			Source, Branch string
			Version        string `json:"package_version"`
			Prior          string `json:"prior_tool_source"`
			Rebuild        string `json:"tool_packaging_rebuild"`
		} `json:"single_project_migration"`
	}
	if ReadJSON(root, Beta2AuthorizationPath, &auth) != nil || auth.Migration.Source != "PROJECT_OWNER_DIRECT_INSTRUCTION_2026_10_17" || auth.Migration.Branch != "dev/0.3.8b2" || auth.Migration.Version != "0.3.8b3" || auth.Migration.Prior != old.SourceSHA || auth.Migration.Rebuild != "ALLOW_FOR_FIRST_SINGLE_PROJECT_B3_PUBLICATION_ONLY" {
		return false
	}
	allowed := map[string]bool{"pyproject.toml": true, "MANIFEST.in": true, "ptsip_build_backend.py": true, "src/ptsip/engine_installation.py": true, "src/ptsip/package-distribution.json": true, "src/policy/CNTR/SFP-CNTR-0005.yaml": true, "src/tests/ptsip/cli/test_engine_installation.py": true}
	for _, ref := range strings.Fields(delta) {
		if !allowed[ref] {
			return false
		}
	}
	return true
}
