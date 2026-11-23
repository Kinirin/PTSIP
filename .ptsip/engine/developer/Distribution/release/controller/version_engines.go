package releasecontroller

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
	"github.com/pelletier/go-toml/v2"
	"go.yaml.in/yaml/v3"
)

const currentEngineIndex = "src/ptsip/engines/index.json"

type syncEngineEntry struct {
	Kind     string `json:"kind"`
	Engine   string `json:"engine_id"`
	Cap      string `json:"capability_id"`
	Version  string `json:"version"`
	Manifest string `json:"manifest_ref"`
	Contract string `json:"contract_ref"`
	Bindings string `json:"execution_binding_ref"`
	Cargo    string `json:"cargo_manifest_ref"`
}

func verifiedJSON(data []byte, target any) error {
	if _, err := jsondoc.Decode(data); err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func readProjection(root, relative string) ([]byte, os.FileMode, error) {
	if !filepath.IsLocal(filepath.FromSlash(relative)) || strings.Contains(relative, "\\") {
		return nil, 0, fmt.Errorf("unsafe projection path %q", relative)
	}
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, 0, err
	}
	p := filepath.Join(root, filepath.FromSlash(relative))
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		return nil, 0, fmt.Errorf("resolve %s: %w", relative, err)
	}
	if resolved != p {
		return nil, 0, fmt.Errorf("projection is an alias: %s", relative)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, 0, err
	}
	if !info.Mode().IsRegular() {
		return nil, 0, fmt.Errorf("projection must be a regular file: %s", relative)
	}
	data, err := os.ReadFile(resolved)
	return data, info.Mode().Perm(), err
}

func engineEntries(data []byte, source VersionSource) ([]json.RawMessage, []syncEngineEntry, error) {
	var index struct {
		Entries []json.RawMessage `json:"official_engine_entries"`
	}
	if err := verifiedJSON(data, &index); err != nil {
		return nil, nil, err
	}
	if len(index.Entries) != len(source.Engines) {
		return nil, nil, fmt.Errorf("Engine index and src/VERSION inventory differ")
	}
	seen := map[string]bool{}
	byID := map[string]bool{}
	entries := make([]syncEngineEntry, 0, len(index.Entries))
	for _, raw := range index.Entries {
		var entry syncEngineEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			return nil, nil, err
		}
		id := strings.ToLower(entry.Kind)
		base := "src/ptsip/engines/" + id + "/"
		if seen[id] || byID[entry.Engine] || byID[entry.Cap] || entry.Engine == "" || entry.Cap == "" ||
			entry.Engine == entry.Cap || source.Engines[id] == "" || entry.Version == "" ||
			entry.Manifest != base+"manifest.json" || entry.Contract != base+"contract.json" ||
			entry.Bindings != base+"execution-bindings.json" || entry.Cargo != base+"Cargo.toml" {
			return nil, nil, fmt.Errorf("invalid or ambiguous official Engine: %s", id)
		}
		seen[id], byID[entry.Engine], byID[entry.Cap] = true, true, true
		entries = append(entries, entry)
	}
	return index.Entries, entries, nil
}

func projectJSONField(data []byte, field, version string) ([]byte, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	if token, err := d.Token(); err != nil || token != json.Delim('{') {
		return nil, fmt.Errorf("expected object for %s", field)
	}
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return nil, err
		}
		var raw json.RawMessage
		if err := d.Decode(&raw); err != nil {
			return nil, err
		}
		if key != field {
			continue
		}
		var current string
		if err := json.Unmarshal(raw, &current); err != nil || current == "" {
			return nil, fmt.Errorf("expected version string for %s", field)
		}
		if current == version {
			return data, nil
		}
		end := int(d.InputOffset())
		replacement, _ := json.Marshal(version)
		result := append(bytes.Clone(data[:end-len(raw)]), replacement...)
		return append(result, data[end:]...), nil
	}
	return nil, fmt.Errorf("missing JSON field %s", field)
}

func projectIndexedEngines(data []byte, source VersionSource) ([]byte, error) {
	raw, entries, err := engineEntries(data, source)
	if err != nil {
		return nil, err
	}
	for i, entry := range entries {
		next, err := projectJSONField(raw[i], "version", source.Engines[strings.ToLower(entry.Kind)])
		if err != nil || bytes.Count(data, raw[i]) != 1 {
			return nil, fmt.Errorf("cannot project index entry %s", entry.Kind)
		}
		data = bytes.Replace(data, raw[i], next, 1)
	}
	return data, nil
}

type stageVersion func(string, func([]byte) ([]byte, error)) error

func stageEngine(root string, stage stageVersion, entry syncEngineEntry, version string) error {
	id := strings.ToLower(entry.Kind)
	base := "src/ptsip/engines/" + id + "/"
	rawContract, _, err := readProjection(root, entry.Contract)
	if err != nil {
		return err
	}
	var contract struct {
		ID      string `json:"contract_id"`
		Engine  string `json:"engine_id"`
		Version string `json:"version"`
	}
	if err := verifiedJSON(rawContract, &contract); err != nil {
		return err
	}
	if contract.ID != entry.Cap || contract.Engine != entry.Engine || contract.Version != version {
		return fmt.Errorf("current contract does not confirm %s version %s and its owner; no version projection applied", id, version)
	}
	hash := sha256.Sum256(bytes.ReplaceAll(rawContract, []byte("\r\n"), []byte("\n")))
	if err := stage(entry.Manifest, func(data []byte) ([]byte, error) {
		var manifest struct {
			Engine string `json:"engine_id"`
			Origin string `json:"origin"`
			Cap    []struct {
				ID      string `json:"id"`
				Version string `json:"version"`
				Path    string `json:"path"`
				Hash    string `json:"sha256"`
			} `json:"capabilities"`
		}
		if err := verifiedJSON(data, &manifest); err != nil {
			return nil, err
		}
		if manifest.Engine != entry.Engine || manifest.Origin != "OFFICIAL" || len(manifest.Cap) != 1 ||
			manifest.Cap[0].ID != entry.Cap || manifest.Cap[0].Version != version ||
			manifest.Cap[0].Path != entry.Contract || manifest.Cap[0].Hash != fmt.Sprintf("%x", hash) {
			return nil, fmt.Errorf("official manifest does not pin the registered engine contract")
		}
		return projectJSONField(data, "implementation_version", version)
	}); err != nil {
		return err
	}
	if err := stage(entry.Bindings, func(data []byte) ([]byte, error) {
		var document struct {
			Bindings []json.RawMessage `json:"bindings"`
		}
		if err := verifiedJSON(data, &document); err != nil {
			return nil, err
		}
		if len(document.Bindings) != 1 {
			return nil, fmt.Errorf("ambiguous official execution binding")
		}
		raw := document.Bindings[0]
		var binding struct {
			Engine   string `json:"engine_id"`
			Cap      string `json:"capability_id"`
			Contract string `json:"contract_version"`
		}
		if err := json.Unmarshal(raw, &binding); err != nil {
			return nil, err
		}
		if binding.Engine != entry.Engine || binding.Cap != entry.Cap || binding.Contract != version || bytes.Count(data, raw) != 1 {
			return nil, fmt.Errorf("execution binding does not select the official engine contract")
		}
		updated, err := projectJSONField(raw, "engine_version", version)
		if err != nil {
			return nil, err
		}
		return bytes.Replace(data, raw, updated, 1), nil
	}); err != nil {
		return err
	}
	if err := stage(base+"version.yaml", func(data []byte) ([]byte, error) {
		var declared struct {
			Engine  string
			Version string
		}
		if err := yaml.Unmarshal(data, &declared); err != nil || declared.Engine != id || declared.Version == "" {
			return nil, fmt.Errorf("invalid engine metadata identity")
		}
		return projectSingleLine(data, "version", version)
	}); err != nil {
		return err
	}
	var cargoVersion string
	name := "ptsip-engine-" + id
	if err := stage(entry.Cargo, func(data []byte) ([]byte, error) {
		var crate struct {
			Package struct{ Name, Version string }
		}
		if err := toml.Unmarshal(data, &crate); err != nil || crate.Package.Name != name || crate.Package.Version == "" {
			return nil, fmt.Errorf("invalid engine Cargo package identity")
		}
		cargoVersion = crate.Package.Version
		return projectCargoVersion(data, "[package]", "", version)
	}); err != nil {
		return err
	}
	return stage(base+"Cargo.lock", func(data []byte) ([]byte, error) {
		var lock struct {
			Package []struct{ Name, Version, Source string }
		}
		if err := toml.Unmarshal(data, &lock); err != nil {
			return nil, err
		}
		matches := 0
		for _, crate := range lock.Package {
			if crate.Name == name {
				if crate.Source != "" || crate.Version != cargoVersion {
					return nil, fmt.Errorf("engine Cargo.lock disagrees with its current Cargo package")
				}
				matches++
			}
		}
		if matches != 1 {
			return nil, fmt.Errorf("engine package lock row is missing or ambiguous")
		}
		return projectCargoVersion(data, "[[package]]", name, version)
	})
}

func projectCargoVersion(data []byte, header, packageName, version string) ([]byte, error) {
	lines := strings.SplitAfter(string(data), "\n")
	start, end := -1, -1
	for i, line := range lines {
		if strings.TrimSpace(line) != header {
			continue
		}
		j := i + 1
		selected := packageName == ""
		for ; j < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[j]), "["); j++ {
			_, value, found, err := tomlProjectionLine(lines[j], "name", `""`)
			if selected || !found {
				continue
			}
			if err != nil {
				return nil, err
			}
			selected = value == fmt.Sprintf("%q", packageName)
		}
		if selected {
			if start != -1 {
				return nil, fmt.Errorf("ambiguous Cargo version target")
			}
			start, end = i+1, j
		}
	}
	if start == -1 {
		return nil, fmt.Errorf("missing Cargo version target")
	}
	matches := 0
	for i := start; i < end; i++ {
		updated, _, found, err := tomlProjectionLine(lines[i], "version", fmt.Sprintf("%q", version))
		if err != nil {
			return nil, err
		}
		if found {
			lines[i] = updated
			matches++
		}
	}
	if matches != 1 {
		return nil, fmt.Errorf("missing or ambiguous Cargo version declaration")
	}
	return []byte(strings.Join(lines, "")), nil
}
