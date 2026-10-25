package releasecontroller

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// The root project is the complete Package. A standalone Tool build is a checked
// temporary projection with independent metadata and no native dependencies.
func MaterializeToolProject(root string) error {
	c, err := ReadComposition(root)
	if err != nil {
		return err
	}
	version, err := ToolDistributionVersion(c.Tool)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	if err != nil {
		return err
	}
	var document map[string]any
	if err := toml.Unmarshal(b, &document); err != nil {
		return err
	}
	project, ok := document["project"].(map[string]any)
	if !ok {
		return fmt.Errorf("project metadata unavailable")
	}
	project["name"], project["version"] = "ptsip-tool", version
	// A Tool-only intermediate must not trigger the complete PTSIP native
	// source build backend: that backend belongs to the final host distribution.
	build, ok := document["build-system"].(map[string]any)
	if !ok {
		return fmt.Errorf("build system unavailable")
	}
	build["build-backend"] = "setuptools.build_meta"
	delete(build, "backend-path")
	deps := []string{}
	if values, ok := project["dependencies"].([]any); ok {
		for _, value := range values {
			dependency, ok := value.(string)
			if !ok {
				return fmt.Errorf("invalid dependency metadata")
			}
			if !strings.HasPrefix(dependency, "ptsip-engine-") {
				deps = append(deps, dependency)
			}
		}
	}
	project["dependencies"] = deps
	text, err := toml.Marshal(document)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "pyproject.toml"), text, 0644)
}
