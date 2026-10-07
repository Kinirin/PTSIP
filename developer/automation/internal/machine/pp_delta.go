package machine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go.yaml.in/yaml/v3"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const PPCatalog = "profiles/index.yaml"
const PPRegistry = "registry/project-profile-contracts.yaml"
const PPEmbeddedRegistry = "src/ptsip/specdata/project-profile-contracts.yaml"

type PPSnapshot interface {
	Label() string
	ReadBytes(string) ([]byte, error)
	Resources() ([]string, error)
}
type PPGitSnapshot struct {
	Root, Revision   string
	Staged, Worktree bool
}

func (s PPGitSnapshot) Label() string {
	if s.Staged {
		return "STAGED_INDEX"
	}
	if s.Worktree {
		return "WORKTREE"
	}
	return s.Revision
}
func ppGit(root string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("GIT_OPERATION_FAILED: %s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return out, nil
}
func (s PPGitSnapshot) ReadBytes(path string) ([]byte, error) {
	if !ppSafeAsset(path) {
		return nil, fmt.Errorf("PP_ASSET_PATH_ESCAPE: %s", path)
	}
	if s.Worktree {
		raw, err := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(path)))
		if os.IsNotExist(err) {
			return nil, nil
		}
		return raw, err
	}
	ref := s.Revision + ":" + path
	if s.Staged {
		ref = ":" + path
	}
	raw, err := ppGit(s.Root, "show", ref)
	if err != nil {
		return nil, nil
	}
	return raw, nil
}
func ppSafeAsset(path string) bool {
	return path != "" && !strings.Contains(path, "\\") && !filepath.IsAbs(path) && !strings.HasPrefix(path, "/") && !strings.Contains(path, ":") && filepath.ToSlash(filepath.Clean(path)) == path && path != ".." && !strings.HasPrefix(path, "../")
}
func (s PPGitSnapshot) Resources() ([]string, error) {
	storage, err := ppSnapshotStorage(s)
	if err != nil {
		return nil, err
	}
	paths, err := s.files(storage.Root)
	if err != nil {
		return nil, err
	}
	values := map[string]bool{}
	for _, line := range paths {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, storage.Root+"/") {
			continue
		}
		rel := strings.TrimPrefix(line, storage.Root+"/")
		if !strings.Contains(rel, "/") && strings.HasSuffix(rel, ".ptsip.yaml") {
			values[rel] = true
		}
	}
	out := []string{}
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out, nil
}
func ppYAML(raw []byte, label string, required bool) (Object, error) {
	if raw == nil {
		if required {
			return nil, fmt.Errorf("REQUIRED_AUTHORITY_INPUT_MISSING: %s", label)
		}
		return nil, nil
	}
	var value any
	if err := yaml.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("AUTHORITY_INPUT_INVALID: %s: %w", label, err)
	}
	value, err := normalize(value)
	if err != nil {
		return nil, err
	}
	if Map(value) == nil {
		return nil, fmt.Errorf("AUTHORITY_INPUT_INVALID: %s must be mapping", label)
	}
	return Map(value), nil
}
func ppSemanticProfile(raw []byte) (string, string, error) {
	payload, err := ppYAML(raw, "profile", true)
	if err != nil {
		return "", "", err
	}
	ptsip := Map(payload["ptsip"])
	version := Text(ptsip["version"])
	delete(ptsip, "version")
	b, err := json.Marshal(payload)
	return string(b), version, err
}
func ppSemanticSchema(raw []byte) (string, error) {
	var payload Object
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", fmt.Errorf("CURRENT_PP_SCHEMA_INVALID: %w", err)
	}
	if payload == nil {
		return "", fmt.Errorf("CURRENT_PP_SCHEMA_INVALID")
	}
	delete(payload, "$id")
	delete(payload, "title")
	version := Map(Map(Map(Map(payload["properties"])["ptsip"])["properties"])["version"])
	delete(version, "const")
	delete(version, "description")
	out, err := json.Marshal(payload)
	return string(out), err
}

type PPAuthorityState struct {
	Label                           string
	CatalogPresent, RegistryPresent bool
	Current                         string
	Contracts                       map[string]string
	Entries                         map[string]Object
	Resources                       []string
	Raw                             map[string][]byte
	Semantic, Declared              map[string]string
	Schema                          []byte
	SchemaSemantic                  string
}

func LoadPPAuthority(source PPSnapshot) (*PPAuthorityState, error) {
	registryRaw, err := source.ReadBytes(PPRegistry)
	if err != nil {
		return nil, err
	}
	registry, err := ppYAML(registryRaw, source.Label()+":"+PPRegistry, false)
	if err != nil {
		return nil, err
	}
	storage, err := ppStorageForRegistry(registry)
	if err != nil {
		return nil, err
	}
	catalogRaw, err := source.ReadBytes(storage.Catalog)
	if err != nil {
		return nil, err
	}
	catalog, err := ppYAML(catalogRaw, source.Label()+":"+storage.Catalog, false)
	if err != nil {
		return nil, err
	}
	if catalog != nil && Text(catalog["root"]) != storage.Root {
		return nil, fmt.Errorf("PP_CATALOG_STORAGE_MISMATCH")
	}
	state := &PPAuthorityState{Label: source.Label(), CatalogPresent: catalog != nil, RegistryPresent: registry != nil, Contracts: map[string]string{}, Entries: map[string]Object{}, Raw: map[string][]byte{}, Semantic: map[string]string{}, Declared: map[string]string{}}
	if catalog != nil {
		for _, raw := range List(catalog["profiles"]) {
			row := Map(raw)
			if row == nil {
				continue
			}
			id, res, contract := Text(row["id"]), Text(row["resource"]), Text(row["contract"])
			if id == "" || res == "" || contract == "" || state.Entries[id] != nil {
				return nil, fmt.Errorf("PUBLIC_PROFILE_CATALOG_INVALID")
			}
			if !ppSafeAsset(storage.Root+"/"+res) || strings.Contains(res, "/") {
				return nil, fmt.Errorf("PUBLIC_PROFILE_CATALOG_RESOURCE_ESCAPE")
			}
			state.Entries[id] = row
		}
	}
	if registry != nil {
		state.Current = Text(registry["current"])
		if state.Current == "" {
			return nil, fmt.Errorf("PP_CONTRACT_REGISTRY_INVALID")
		}
		for _, raw := range List(registry["contracts"]) {
			row := Map(raw)
			if version := Text(row["version"]); version != "" {
				state.Contracts[version] = Text(row["schema"])
			}
		}
	}
	state.Resources, err = source.Resources()
	if err != nil {
		return nil, err
	}
	for _, res := range state.Resources {
		raw, e := source.ReadBytes(storage.Root + "/" + res)
		if e != nil {
			return nil, e
		}
		if raw == nil {
			return nil, fmt.Errorf("PUBLIC_PROFILE_READ_FAILED")
		}
		state.Raw[res] = raw
		semantic, declared, e := ppSemanticProfile(raw)
		if e != nil {
			return nil, e
		}
		state.Semantic[res] = semantic
		state.Declared[res] = declared
	}
	if path := state.Contracts[state.Current]; path != "" {
		state.Schema, err = source.ReadBytes(path)
		if err != nil {
			return nil, err
		}
		if state.Schema != nil {
			state.SchemaSemantic, err = ppSemanticSchema(state.Schema)
			if err != nil {
				return nil, err
			}
		}
	}
	return state, nil
}

type PPDelta struct {
	Classification   string   `json:"classification"`
	Triggered        bool     `json:"triggered"`
	Valid            bool     `json:"valid"`
	Reasons          []string `json:"reasons"`
	BaseCurrent      any      `json:"base_current"`
	CandidateCurrent any      `json:"candidate_current"`
	ExpectedNext     any      `json:"expected_next"`
	Reconciled       bool     `json:"candidate_already_reconciled"`
}

var ppVersionPattern = regexp.MustCompile(`^pp\.([1-9][0-9]*|0)\.([0-9]{2,})$`)

func ppNext(version string) (string, error) {
	match := ppVersionPattern.FindStringSubmatch(version)
	if match == nil {
		return "", fmt.Errorf("PP_IDENTITY_INVALID: %s", version)
	}
	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])
	if fmt.Sprintf("pp.%d.%02d", major, minor) != version {
		return "", fmt.Errorf("PP_IDENTITY_NONCANONICAL")
	}
	return fmt.Sprintf("pp.%d.%02d", major, minor+1), nil
}
func ppOptional(text string) any {
	if text == "" {
		return nil
	}
	return text
}
func EvaluatePPDelta(base, candidate *PPAuthorityState, oldCandidateSchema []byte) (PPDelta, error) {
	delta := PPDelta{Reasons: []string{}, BaseCurrent: ppOptional(base.Current), CandidateCurrent: ppOptional(candidate.Current)}
	baseline := !base.CatalogPresent && !base.RegistryPresent && candidate.CatalogPresent && candidate.RegistryPresent && candidate.Current != "" && reflect.DeepEqual(base.Resources, candidate.Resources)
	resources := []string{}
	for _, row := range candidate.Entries {
		resources = append(resources, Text(row["resource"]))
		baseline = baseline && row["contract"] == candidate.Current
	}
	sort.Strings(resources)
	baseline = baseline && reflect.DeepEqual(resources, base.Resources) && candidate.Schema != nil
	for _, res := range base.Resources {
		baseline = baseline && bytes.Equal(base.Raw[res], candidate.Raw[res]) && base.Declared[res] == candidate.Current
	}
	if baseline {
		delta.Classification = "BASELINE_MATERIALIZATION_EXISTING_DISTRIBUTION"
		delta.Valid = oldCandidateSchema != nil && bytes.Equal(oldCandidateSchema, candidate.Schema)
		if !delta.Valid {
			delta.Classification = "INVALID_BASELINE_MATERIALIZATION"
			delta.Reasons = []string{"CURRENT_PP_CANONICAL_SCHEMA_CONTENT"}
		}
		return delta, nil
	}
	if !base.CatalogPresent || !base.RegistryPresent {
		return delta, fmt.Errorf("T2_BASELINE_UNRESOLVED")
	}
	next, err := ppNext(base.Current)
	if err != nil {
		return delta, err
	}
	if !candidate.CatalogPresent || !candidate.RegistryPresent {
		delta.Classification = "AUTHORITY_PLANE_REMOVED"
		delta.Triggered = true
		delta.ExpectedNext = next
		delta.Reasons = []string{"PUBLIC_PROFILE_CATALOG_MEMBERSHIP"}
		return delta, nil
	}
	if candidate.Current == "" {
		return delta, fmt.Errorf("PP_CURRENT_UNRESOLVED")
	}
	if len(base.Entries) != len(candidate.Entries) {
		delta.Reasons = append(delta.Reasons, "PUBLIC_PROFILE_CATALOG_MEMBERSHIP")
	} else {
		for id := range base.Entries {
			if candidate.Entries[id] == nil {
				delta.Reasons = append(delta.Reasons, "PUBLIC_PROFILE_CATALOG_MEMBERSHIP")
				break
			}
		}
	}
	ids := []string{}
	for id := range base.Entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		left, right := base.Entries[id], candidate.Entries[id]
		if right == nil {
			continue
		}
		res := Text(left["resource"])
		if left["resource"] != right["resource"] {
			delta.Reasons = append(delta.Reasons, "PUBLIC_PROFILE_CATALOG_RESOURCE_IDENTITY")
			continue
		}
		if base.Semantic[res] != candidate.Semantic[res] {
			delta.Reasons = append(delta.Reasons, "REGISTERED_CANONICAL_PUBLIC_PROFILE_CONTENT")
		}
	}
	if base.Contracts[base.Current] == "" {
		return delta, fmt.Errorf("CURRENT_PP_SCHEMA_UNRESOLVED")
	}
	if candidate.Current == base.Current {
		if base.SchemaSemantic != candidate.SchemaSemantic {
			delta.Reasons = append(delta.Reasons, "CURRENT_PP_CANONICAL_SCHEMA_CONTENT")
		}
	} else if candidate.SchemaSemantic != "" && base.SchemaSemantic != candidate.SchemaSemantic {
		delta.Reasons = append(delta.Reasons, "CURRENT_PP_CANONICAL_SCHEMA_CONTENT")
	} else if oldCandidateSchema != nil && !bytes.Equal(base.Schema, oldCandidateSchema) {
		delta.Reasons = append(delta.Reasons, "CURRENT_PP_CANONICAL_SCHEMA_CONTENT")
	}
	seen := map[string]bool{}
	unique := []string{}
	for _, reason := range delta.Reasons {
		if !seen[reason] {
			unique = append(unique, reason)
			seen[reason] = true
		}
	}
	delta.Reasons = unique
	if len(unique) == 0 {
		delta.Valid = candidate.Current == base.Current
		delta.Classification = "NO_T2_AUTHORITY_DELTA"
		if !delta.Valid {
			delta.Classification = "MANUAL_PP_TRANSITION_WITHOUT_AUTHORITY"
		}
		return delta, nil
	}
	delta.ExpectedNext = next
	delta.Triggered = true
	delta.Valid = candidate.Current == base.Current || candidate.Current == next
	delta.Reconciled = candidate.Current == next
	delta.Classification = "T2_AUTHORITY_DELTA"
	if !delta.Valid {
		delta.Classification = "PP_TRANSITION_IDENTITY_MISMATCH"
	}
	return delta, nil
}
func ComparePPSnapshots(base, candidate PPSnapshot) (PPDelta, error) {
	left, err := LoadPPAuthority(base)
	if err != nil {
		return PPDelta{}, err
	}
	right, err := LoadPPAuthority(candidate)
	if err != nil {
		return PPDelta{}, err
	}
	var old []byte
	if !left.RegistryPresent && right.Current != "" {
		old, err = base.ReadBytes(right.Contracts[right.Current])
	} else if left.Current != "" {
		old, err = candidate.ReadBytes(left.Contracts[left.Current])
	}
	if err != nil {
		return PPDelta{}, err
	}
	return EvaluatePPDelta(left, right, old)
}
func (r *Repository) ComparePP(base, candidate string, staged bool) (PPDelta, error) {
	return ComparePPSnapshots(PPGitSnapshot{Root: r.Root, Revision: base}, PPGitSnapshot{Root: r.Root, Revision: candidate, Staged: staged})
}
