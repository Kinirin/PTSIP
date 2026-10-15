package releasecontroller

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StageEngineUploads joins only pending, admitted native wheels into one upload
// directory. Twine uploads these files sequentially in one publisher invocation.
// Already public files were removed from each component stage by PyPIStage.
func StageEngineUploads(root string, m ShippingManifest, output string) (int, error) {
	target, err := SafePath(root, output+"/pypi/engines")
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(target, 0755); err != nil {
		return 0, err
	}
	previous, err := os.ReadDir(target)
	if err != nil {
		return 0, err
	}
	for _, entry := range previous {
		if !entry.Type().IsRegular() {
			return 0, fmt.Errorf("unexpected aggregate upload entry: %s", entry.Name())
		}
		if err := os.Remove(filepath.Join(target, entry.Name())); err != nil {
			return 0, err
		}
	}
	count := 0
	seen := map[string]bool{}
	for _, c := range m.Snapshot.Components {
		if c.Kind != "engine" {
			continue
		}
		selected := false
		for _, request := range m.Selection.Components {
			if request.ID == c.ID && request.Kind == "engine" && request.Version == c.Version {
				selected = true
			}
		}
		if !selected {
			continue
		}
		if _, err := PublisherEnvironment(root, Component{ID: c.ID, Kind: c.Kind, Version: c.Version}); err != nil {
			return 0, err
		}
		source, err := SafePath(root, output+"/pypi/"+c.ID)
		if err != nil {
			return 0, err
		}
		entries, err := os.ReadDir(source)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return 0, err
		}
		allowed := map[string]string{}
		for _, f := range c.Files {
			if strings.HasSuffix(f.Name, ".whl") {
				allowed[f.Name] = f.SHA256
			}
		}
		for _, entry := range entries {
			name := entry.Name()
			if !entry.Type().IsRegular() || allowed[name] == "" || seen[name] {
				return 0, fmt.Errorf("unadmitted or duplicate engine upload: %s", name)
			}
			b, err := os.ReadFile(filepath.Join(source, name))
			if err != nil || Digest(b) != allowed[name] {
				return 0, fmt.Errorf("pending engine upload digest differs: %s", name)
			}
			if err := os.WriteFile(filepath.Join(target, name), b, 0644); err != nil {
				return 0, err
			}
			seen[name] = true
			count++
		}
	}
	return count, nil
}
