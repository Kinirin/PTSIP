package machine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"
)

func agentHash(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
func agentText(r *Repository, ref string) (string, error) {
	file, err := r.Path(ref)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n"), nil
}
func agentJSON(value any) any {
	data, _ := json.Marshal(value)
	var decoded any
	_ = json.Unmarshal(data, &decoded)
	return decoded
}
func agentInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	case int64:
		return int(v)
	}
	return -1
}

func BuildAgentMaterialization(r *Repository, source, output string) (map[string]Object, error) {
	if source == "" {
		source = "AGENTS.md"
	}
	if output == "" {
		output = ".agent"
	}
	if err := ValidateAgentTaxonomy(r); err != nil {
		return nil, err
	}
	sourceRef, err := r.Scope(source)
	if err != nil {
		return nil, err
	}
	outputRef, err := r.Scope(output)
	if err != nil {
		return nil, err
	}
	text, err := agentText(r, source)
	if err != nil {
		return nil, err
	}
	atoms := ClassifyAgentMarkdown(text)
	lines := strings.SplitAfter(text, "\n")
	records := []any{}
	unresolvedIDs := []any{}
	for _, a := range atoms {
		end := a.LineEnd
		if end > len(lines) {
			end = len(lines)
		}
		start := a.LineStart - 1
		if start < 0 {
			start = 0
		}
		excerpt := strings.Join(lines[start:end], "")
		records = append(records, Object{"atom_id": a.AtomID, "kind": a.Kind, "normalized_text": a.Text, "source": Object{"path": sourceRef, "line_start": a.LineStart, "line_end": a.LineEnd, "heading_path": a.HeadingPath, "parent_atom_id": a.ParentAtomID, "raw_excerpt": excerpt, "raw_excerpt_sha256": agentHash(excerpt)}, "level_1": a.Level1, "unresolved": a.Unresolved})
		if a.Unresolved {
			unresolvedIDs = append(unresolvedIDs, a.AtomID)
		}
	}
	registry := Object{"schema_version": "ptsip-agent-instruction-level1-materialization/v1", "policy_ref": "MPD-INFO-0001#unit_mpd_0010_f1b93fa1851f", "stage": "LEVEL_1_ONLY", "source": Object{"path": sourceRef, "sha256": agentHash(text)}, "level_1_vocabulary": AgentLevel1, "level_2_materialized": false, "summary": agentAtomSummary(atoms), "atoms": records}
	refs := Object{}
	files := map[string]Object{path.Join(outputRef, "registry.yaml"): registry}
	for _, label := range AgentLevel1 {
		projection := path.Join("level1", strings.ToLower(label)+".yaml")
		refs[label] = projection
		ids := []any{}
		for _, a := range atoms {
			if agentHas(a.Level1, label) {
				ids = append(ids, a.AtomID)
			}
		}
		files[path.Join(outputRef, projection)] = Object{"schema_version": "ptsip-agent-instruction-level1-projection/v1", "stage": "LEVEL_1_ONLY", "level_1": label, "registry_ref": "../registry.yaml", "atom_ids": ids}
	}
	files[path.Join(outputRef, "unresolved.yaml")] = Object{"schema_version": "ptsip-agent-instruction-level1-projection/v1", "stage": "LEVEL_1_ONLY", "routing_state": "UNRESOLVED", "registry_ref": "registry.yaml", "atom_ids": unresolvedIDs}
	files[path.Join(outputRef, "index.yaml")] = Object{"schema_version": "ptsip-agent-instruction-level1-index/v1", "stage": "LEVEL_1_ONLY", "source_ref": sourceRef, "registry_ref": "registry.yaml", "level_1_projections": refs, "unresolved_ref": "unresolved.yaml", "level_2_materialized": false}
	return files, nil
}

func MaterializeAgent(r *Repository, source, output string) (Object, error) {
	files, err := BuildAgentMaterialization(r, source, output)
	if err != nil {
		return nil, err
	}
	paths := []string{}
	for ref := range files {
		paths = append(paths, ref)
	}
	sort.Strings(paths)
	for _, ref := range paths {
		if err := r.WriteYAML(ref, files[ref], nil); err != nil {
			return nil, err
		}
	}
	if output == "" {
		output = ".agent"
	}
	return Object{"state": "MATERIALIZED", "written": paths, "summary": files[path.Join(output, "registry.yaml")]["summary"], "level_2_materialized": false}, nil
}
func CheckAgentMaterialization(r *Repository, source, output string) (Object, error) {
	files, err := BuildAgentMaterialization(r, source, output)
	if err != nil {
		return nil, err
	}
	errors := []string{}
	for ref, expected := range files {
		actual, err := r.Read(ref)
		if err != nil {
			errors = append(errors, "MISSING:"+ref)
			continue
		}
		if !reflect.DeepEqual(agentJSON(actual), agentJSON(expected)) {
			errors = append(errors, "STALE:"+ref)
		}
	}
	sort.Strings(errors)
	state := "CURRENT"
	if len(errors) > 0 {
		state = "STALE"
	}
	if output == "" {
		output = ".agent"
	}
	return Object{"state": state, "errors": errors, "summary": files[path.Join(output, "registry.yaml")]["summary"], "level_2_materialized": false}, nil
}

func agentProjectionRef(r *Repository, base string, value any, label string) (string, error) {
	ref := Text(value)
	if ref == "" {
		return "", fmt.Errorf("%s must be a path", label)
	}
	return r.Scope(path.Join(base, ref))
}
func agentRoutes(r *Repository) (Object, []string, error) {
	value, err := agentSection(r, "MPD-CTRL-0001", "unit_mpd_0010_cdaf0c493ce7")
	if err != nil {
		return nil, nil, err
	}
	entry := Map(value)
	routes := Map(entry["default_routes"])
	if routes == nil {
		return nil, nil, fmt.Errorf("Level 1 entry routes missing")
	}
	for operation, raw := range routes {
		labels := List(raw)
		if len(labels) == 0 {
			return nil, nil, fmt.Errorf("invalid Level 1 route %s", operation)
		}
		for _, item := range labels {
			if !agentHas(AgentLevel1, Text(item)) {
				return nil, nil, fmt.Errorf("invalid route label %s", operation)
			}
		}
	}
	widen := []string{}
	raw, ok := entry["explicit_widen_allowed"].([]any)
	if !ok {
		return nil, nil, fmt.Errorf("explicit widen vocabulary is invalid")
	}
	for _, item := range raw {
		label := Text(item)
		if !agentHas(AgentLevel1, label) {
			return nil, nil, fmt.Errorf("invalid explicit widen %s", label)
		}
		widen = append(widen, label)
	}
	return routes, widen, nil
}

func ResolveAgentEntry(r *Repository, operation string, include []string, agentRoot string) (Object, error) {
	routes, widen, err := agentRoutes(r)
	if err != nil {
		return nil, err
	}
	operation = strings.ToUpper(operation)
	raw, ok := routes[operation]
	if !ok {
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
	labels := []string{}
	for _, label := range List(raw) {
		labels = append(labels, Text(label))
	}
	for _, value := range include {
		label := strings.ToUpper(value)
		if !agentHas(widen, label) {
			return nil, fmt.Errorf("unsupported Level 1 widen: %s", label)
		}
		if !agentHas(labels, label) {
			labels = append(labels, label)
		}
	}
	if agentRoot == "" {
		agentRoot = ".agent"
	}
	agentRoot, err = r.Scope(agentRoot)
	if err != nil {
		return nil, err
	}
	index, err := r.Read(path.Join(agentRoot, "index.yaml"))
	if err != nil {
		return nil, err
	}
	if Map(index["progressive_reasoning"]) != nil {
		return resolveAgentProgressive(r, agentRoot, index, operation, labels)
	}
	registryRef, err := agentProjectionRef(r, agentRoot, index["registry_ref"], "registry_ref")
	if err != nil {
		return nil, err
	}
	registry, err := r.Read(registryRef)
	if err != nil {
		return nil, err
	}
	if index["level_2_materialized"] != false || registry["level_2_materialized"] != false {
		return nil, fmt.Errorf("Level 2 routing is not active")
	}
	ordered := []Object{}
	byID := map[string]Object{}
	rawAtoms, ok := registry["atoms"].([]any)
	if !ok {
		return nil, fmt.Errorf("registry.atoms must be a list")
	}
	for _, raw := range rawAtoms {
		atom := Map(raw)
		id := Text(atom["atom_id"])
		if id == "" || byID[id] != nil {
			return nil, fmt.Errorf("invalid or duplicate atom id: %s", id)
		}
		byID[id] = atom
		ordered = append(ordered, atom)
	}
	selected := map[string]bool{}
	refs := Map(index["level_1_projections"])
	if refs == nil {
		return nil, fmt.Errorf("level_1_projections missing")
	}
	for _, label := range labels {
		ref, err := agentProjectionRef(r, agentRoot, refs[label], label)
		if err != nil {
			return nil, err
		}
		projection, err := r.Read(ref)
		if err != nil {
			return nil, err
		}
		if projection["level_1"] != label {
			return nil, fmt.Errorf("projection mismatch: %s", label)
		}
		ids, ok := projection["atom_ids"].([]any)
		if !ok {
			return nil, fmt.Errorf("invalid atom ids: %s", label)
		}
		for _, value := range ids {
			id := Text(value)
			if id == "" || byID[id] == nil {
				return nil, fmt.Errorf("unknown atom id: %s", id)
			}
			selected[id] = true
		}
	}
	unresolvedRef, err := agentProjectionRef(r, agentRoot, index["unresolved_ref"], "unresolved_ref")
	if err != nil {
		return nil, err
	}
	projection, err := r.Read(unresolvedRef)
	if err != nil {
		return nil, err
	}
	if projection["routing_state"] != "UNRESOLVED" {
		return nil, fmt.Errorf("unresolved routing state mismatch")
	}
	ids, ok := projection["atom_ids"].([]any)
	if !ok {
		return nil, fmt.Errorf("invalid unresolved atom ids")
	}
	instructions := []any{}
	unresolved := []any{}
	compact := func(atom Object) (Object, error) {
		id := Text(atom["atom_id"])
		labels, ok := atom["level_1"].([]any)
		text := Text(atom["instruction_text"])
		if text == "" {
			text = Text(atom["normalized_text"])
		}
		if !ok || text == "" {
			return nil, fmt.Errorf("invalid registry atom %s", id)
		}
		return Object{"atom_id": id, "level_1": labels, "text": strings.TrimRight(text, "\r\n")}, nil
	}
	for _, atom := range ordered {
		if selected[Text(atom["atom_id"])] {
			item, err := compact(atom)
			if err != nil {
				return nil, err
			}
			instructions = append(instructions, item)
		}
	}
	for _, value := range ids {
		id := Text(value)
		if id == "" || byID[id] == nil {
			return nil, fmt.Errorf("unknown unresolved atom id: %s", id)
		}
		item, err := compact(byID[id])
		if err != nil {
			return nil, err
		}
		headings := List(Map(byID[id]["source"])["heading_path"])
		if headings == nil {
			headings = []any{}
		}
		item["heading_path"] = headings
		unresolved = append(unresolved, item)
	}
	return Object{"schema_version": "ptsip-agent-instruction-entry-resolution/v1", "stage": "LEGACY_LEVEL_1", "operation": operation, "selected_level_1": labels, "scope_filtering": "NOT_AVAILABLE_AT_LEVEL_1", "level_2_used": false, "unresolved_policy": "ALWAYS_INCLUDE_UNTIL_CLASSIFIED", "instruction_count": len(instructions), "unresolved_count": len(unresolved), "instructions": instructions, "unresolved": unresolved}, nil
}
