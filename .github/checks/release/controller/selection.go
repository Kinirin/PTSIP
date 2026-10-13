package releasecontroller

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

const ContractPath = ".github/checks/release/composition-contract.json"

type Component struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Version string `json:"version"`
	Note    string `json:"note"`
}
type Selection struct {
	Schema       string      `json:"schema_version"`
	SourceSHA    string      `json:"source_sha"`
	Type         string      `json:"release_type"`
	Tag          string      `json:"tag"`
	Components   []Component `json:"components"`
	Platforms    []string    `json:"platforms"`
	SelectionSHA string      `json:"selection_sha256"`
	PackageBase  string      `json:"package_base"`
}
type MatrixEntry struct {
	Component string `json:"component"`
	Kind      string `json:"kind"`
	Version   string `json:"version"`
	Platform  string `json:"platform"`
	Runner    string `json:"runner"`
}
type Contract struct {
	Schema          string         `json:"schema_version"`
	PackageBase     string         `json:"package_base"`
	EngineRegistry  string         `json:"engine_registry"`
	ProfileRegistry string         `json:"profile_registry"`
	Required        []string       `json:"required_platforms"`
	Optional        []string       `json:"optional_platforms"`
	Notes           map[string]any `json:"notes"`
}

func ReadJSON(root, ref string, out any) error {
	p, err := SafePath(root, ref)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}
func SafePath(root, ref string) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil || ref == "" || filepath.IsAbs(ref) || strings.ContainsAny(ref, "\\:") || filepath.ToSlash(filepath.Clean(ref)) != ref || strings.HasPrefix(ref, "../") || ref == ".." {
		return "", fmt.Errorf("invalid repository reference: %s", ref)
	}
	p := filepath.Join(root, filepath.FromSlash(ref))
	if !strings.HasPrefix(p, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("reference escapes repository")
	}
	return p, nil
}
func Digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func Run(root string, args ...string) ([]byte, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = root
	var diagnostic bytes.Buffer
	cmd.Stderr = &diagnostic
	data, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s failed (%v): %s%s", args[0], err, data, diagnostic.Bytes())
	}
	return data, nil
}
func WriteJSON(path string, data any) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0644)
}

func Classify(tool, profiles bool, engines int, full bool) (string, error) {
	if full && tool && profiles && engines > 0 {
		return "PTSIP_PACKAGE", nil
	}
	switch {
	case tool && !profiles && engines == 0:
		return "TOOL_ONLY", nil
	case !tool && !profiles && engines == 1:
		return "ENGINE_ONLY", nil
	case !tool && profiles && engines == 0:
		return "PROFILES_ONLY", nil
	case tool && !profiles && engines > 0:
		return "TOOL_ENGINES", nil
	case tool && profiles && engines == 0:
		return "TOOL_PROFILES", nil
	case !tool && profiles && engines > 0:
		return "ENGINES_PROFILES", nil
	case !tool && !profiles && engines > 1:
		return "MULTIPLE_ENGINES", nil
	default:
		return "", fmt.Errorf("empty or unsupported release selection")
	}
}

func Select(root, targets, optional string) (Selection, error) {
	var result Selection
	var contract Contract
	if err := ReadJSON(root, ContractPath, &contract); err != nil {
		return result, err
	}
	if contract.Schema != "ptsip-release-composition/v1" || len(contract.Required) != 2 {
		return result, fmt.Errorf("unadmitted composition contract")
	}
	head, err := Run(root, "git", "rev-parse", "HEAD")
	if err != nil {
		return result, err
	}
	result = Selection{Schema: "ptsip-release-selection/v2", SourceSHA: strings.TrimSpace(string(head)), Platforms: append([]string{}, contract.Required...), PackageBase: contract.PackageBase, Components: []Component{}}
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(result.SourceSHA) {
		return result, fmt.Errorf("exact source SHA required")
	}
	var index struct {
		Entries []struct{ Kind, Version string } `json:"official_engine_entries"`
	}
	if err := ReadJSON(root, contract.EngineRegistry, &index); err != nil {
		return result, err
	}
	engines := map[string]string{}
	for _, e := range index.Entries {
		id := strings.ToLower(e.Kind)
		if id == "" || engines[id] != "" {
			return result, fmt.Errorf("duplicate engine registration")
		}
		engines[id] = e.Version
	}
	var profile struct {
		Current string `yaml:"current"`
	}
	profilePath, err := SafePath(root, contract.ProfileRegistry)
	if err != nil {
		return result, err
	}
	profileData, err := os.ReadFile(profilePath)
	if err != nil || yaml.Unmarshal(profileData, &profile) != nil || profile.Current == "" {
		return result, fmt.Errorf("current Profile identity unavailable")
	}
	project, err := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	if err != nil {
		return result, err
	}
	version := regexp.MustCompile(`(?m)^version\s*=\s*"([^"]+)"`).FindSubmatch(project)
	if len(version) != 2 {
		return result, fmt.Errorf("Tool version unavailable")
	}
	full := targets == "package" || targets == "all" || targets == ""
	requested := strings.Split(targets, ",")
	if full {
		requested = []string{"tool", "profiles"}
		for id := range engines {
			requested = append(requested, "engine:"+id)
		}
	}
	seen := map[string]bool{}
	tool, profiles, count := false, false, 0
	for _, raw := range requested {
		id := strings.TrimSpace(raw)
		c := Component{ID: id}
		switch id {
		case "tool":
			c.Kind, c.Version, tool = "tool", string(version[1]), true
		case "profiles", "profile":
			c.ID, c.Kind, c.Version, profiles = "profiles", "profiles", profile.Current, true
		default:
			name := strings.TrimPrefix(id, "engine:")
			v, ok := engines[name]
			if !ok {
				return result, fmt.Errorf("unregistered component: %s", id)
			}
			c.ID, c.Kind, c.Version = name, "engine", v
			if err := ValidateEngineVersion(root, name, v); err != nil {
				return result, err
			}
			count++
		}
		if seen[c.ID] {
			return result, fmt.Errorf("duplicate release selection: %s", c.ID)
		}
		seen[c.ID] = true
		note, ok := contract.Notes[c.Kind].(string)
		if !ok {
			return result, fmt.Errorf("component release-note route missing")
		}
		note = strings.ReplaceAll(note, "{engine}", c.ID)
		c.Note = strings.ReplaceAll(note, "{version}", c.Version)
		result.Components = append(result.Components, c)
	}
	result.Type, err = Classify(tool, profiles, count, full || (tool && profiles && count == len(engines)))
	if err != nil {
		return result, err
	}
	if optional != "" {
		for _, p := range strings.Split(optional, ",") {
			p = strings.TrimSpace(p)
			valid := false
			for _, allowed := range contract.Optional {
				valid = valid || p == allowed
			}
			if !valid {
				return result, fmt.Errorf("unregistered optional platform: %s", p)
			}
			for _, prior := range result.Platforms {
				if prior == p {
					return result, fmt.Errorf("duplicate platform: %s", p)
				}
			}
			result.Platforms = append(result.Platforms, p)
		}
	}
	sort.Slice(result.Components, func(i, j int) bool { return result.Components[i].ID < result.Components[j].ID })
	result.Tag = "ptsip-release-" + result.SourceSHA
	canonical, _ := json.Marshal(result)
	result.SelectionSHA = Digest(canonical)
	return result, nil
}

func ValidateEngineVersion(root, id, projected string) error {
	if !regexp.MustCompile(`^[a-z][a-z0-9-]*$`).MatchString(id) {
		return fmt.Errorf("invalid registered engine kind")
	}
	var contract struct {
		Schema          string `yaml:"schema_version"`
		Engine, Version string
	}
	p, err := SafePath(root, "src/ptsip/engines/"+id+"/version.yaml")
	if err != nil {
		return err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	if yaml.Unmarshal(b, &contract) != nil || contract.Schema != "ptsip-engine-version/v1" || contract.Engine != id || contract.Version != projected {
		return fmt.Errorf("engine canonical version and index diverge: %s", id)
	}
	manifest, err := os.ReadFile(filepath.Join(root, "src/ptsip/engines", id, "Cargo.toml"))
	if err != nil {
		return err
	}
	v := regexp.MustCompile(`(?m)^version\s*=\s*"([^"]+)"`).FindSubmatch(manifest)
	if len(v) != 2 || string(v[1]) != projected {
		return fmt.Errorf("engine Cargo version projection differs: %s", id)
	}
	var compiled struct {
		Version string `json:"implementation_version"`
	}
	if err := ReadJSON(root, "src/ptsip/engines/"+id+"/manifest.json", &compiled); err != nil {
		return err
	}
	if compiled.Version != projected {
		return fmt.Errorf("engine manifest version projection differs: %s", id)
	}
	return nil
}

// ValidateSelection rejects edits to a derived execution selection, including its
// digest and canonical registered component/host set, before consuming evidence.
func ValidateSelection(root string, s Selection) error {
	targets := []string{}
	for _, c := range s.Components {
		if c.Kind == "engine" {
			targets = append(targets, "engine:"+c.ID)
		} else {
			targets = append(targets, c.ID)
		}
	}
	optional := []string{}
	var contract Contract
	if err := ReadJSON(root, ContractPath, &contract); err != nil {
		return err
	}
	for _, p := range s.Platforms {
		for _, allowed := range contract.Optional {
			if p == allowed {
				optional = append(optional, p)
			}
		}
	}
	canonical, err := Select(root, strings.Join(targets, ","), strings.Join(optional, ","))
	if err != nil {
		return err
	}
	a, _ := json.Marshal(canonical)
	b, _ := json.Marshal(s)
	if string(a) != string(b) {
		return fmt.Errorf("selection differs from exact-source canonical registration")
	}
	return nil
}

func Matrix(selection Selection, runners map[string]string) ([]MatrixEntry, error) {
	items := []MatrixEntry{}
	for _, c := range selection.Components {
		platforms := selection.Platforms
		for _, p := range platforms {
			if runners[p] == "" {
				return nil, fmt.Errorf("platform runner missing: %s", p)
			}
			items = append(items, MatrixEntry{c.ID, c.Kind, c.Version, p, runners[p]})
		}
	}
	return items, nil
}
