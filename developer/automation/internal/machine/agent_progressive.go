package machine

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
)

const agentDirective = `PTSIP_AGENT_ENTRY version=1 index=".agent/index.yaml" integration="LOCAL_CLI_ONLY" mcp="ABSENT"`
const agentCompactState = "COMPACT_ENTRY_LEVEL_1"
const agentStageRef = "stages/level1.json"
const agentUnresolvedRef = "unresolved/level1.json"

func AgentBootstrapText() string { return "# AGENTS.md\n\n" + agentDirective + "\n" }

func AgentEntryDirective(mode string) (string, error) {
	if mode != "LOCAL_CLI_ONLY" && mode != "MCP_STDIO_READY" {
		return "", fmt.Errorf("unsupported integration mode: %s", mode)
	}
	mcp := "ABSENT"
	if mode == "MCP_STDIO_READY" {
		mcp = "READY"
	}
	return fmt.Sprintf(`PTSIP_AGENT_ENTRY version=1 index=".agent/index.yaml" integration="%s" mcp="%s"`, mode, mcp), nil
}
func AgentIntegrationContract() Object {
	return Object{"mode": "LOCAL_CLI_ONLY", "local_cli": Object{"state": "READY", "resolver_command": "go -C developer/automation run ./cmd/ptsip-dev agent-instruction-entry resolve --operation <READ|MODIFY|PLAN|VERIFY|RELEASE>"}, "mcp": Object{"state": "ABSENT", "optional": true, "transport": "STDIO", "implementation_state": "NOT_AVAILABLE", "user_approval_required_before_install": true, "automatic_install_without_user_approval": false, "offer_to_user_now": false, "intended_install_command": "go -C developer/automation run ./cmd/ptsip-dev agent-integration install-mcp --user-approved"}, "status_command": "go -C developer/automation run ./cmd/ptsip-dev agent-integration status"}
}
func AgentIntegrationStatus(r *Repository) (Object, error) {
	index, err := r.Read(".agent/index.yaml")
	if err != nil {
		return nil, err
	}
	integration := Map(index["integration"])
	mode := Text(integration["mode"])
	directive, err := AgentEntryDirective(mode)
	if err != nil {
		return nil, err
	}
	local := Map(integration["local_cli"])
	mcp := Map(integration["mcp"])
	if local == nil || mcp == nil {
		return nil, fmt.Errorf("agent integration contract invalid")
	}
	copyMCP := Object{}
	for k, v := range mcp {
		copyMCP[k] = v
	}
	copyMCP["offer_to_user_now"] = mode == "LOCAL_CLI_ONLY" && mcp["state"] == "ABSENT" && mcp["implementation_state"] == "READY"
	return Object{"schema_version": "ptsip-agent-integration-status/v1", "mode": mode, "entry_directive": directive, "local_cli": local, "mcp": copyMCP}, nil
}
func InstallAgentMCP(r *Repository, userApproved bool) (Object, error) {
	status, err := AgentIntegrationStatus(r)
	if err != nil {
		return nil, err
	}
	if Map(status["mcp"])["implementation_state"] != "READY" {
		return Object{"status": "NOT_AVAILABLE", "mode": "LOCAL_CLI_ONLY"}, Fail("NOT_AVAILABLE", "PTSIP MCP integration is not available yet")
	}
	if !userApproved {
		return nil, Fail("USER_APPROVAL_REQUIRED", "PTSIP MCP installation requires explicit user approval")
	}
	return nil, fmt.Errorf("MCP implementation is marked READY but no installer is bound")
}

var agentTrigger = agentRX(`^\s*(before|after|when|whenever|if|unless|while|during|for|on)\b`)
var agentModal = []struct {
	name string
	rx   *regexp.Regexp
}{
	{"PROHIBITION", agentRX(`\b(?:must\s+not|shall\s+not|do\s+not|don't|never|cannot|can't|may\s+not|forbidden|prohibited)\b`)},
	{"REQUIREMENT", agentRX(`\b(?:must|shall|required|always|ensure)\b`)},
	{"PREFERENCE", agentRX(`\b(?:should|prefer|avoid)\b`)},
	{"PERMISSION", agentRX(`\b(?:may|can|allowed)\b`)},
}
var agentMechanismAction = agentRX(`\b(resolve|read|load|scan|run|use|check|prepare|select|inspect|create|follow|record|evaluate|compare|apply|verify|build|publish|deploy|invoke|write|edit|generate|remove|update|add|install|format|lint|test|execute|commit|push|open|review|confirm|report|stop|keep|preserve|reject|infer|choose|enter|replace|derive|describe|claim|combine|discover|re-run|rerun)\b`)
var agentBacktick = regexp.MustCompile("`([^`]+)`")
var agentMechanismCommand = agentRX(`\b(?:python(?:\s+-m)?|pytest|git|pip|uv|npm|pnpm|yarn|cargo|mvn|gradle|make|cmake|ruff|mypy|pyright|twine)\b`)

func MechanizeAgentAtom(text string, labels []string) (Object, []any) {
	normalized := agentSpace(text)
	machine := Object{"level_1_labels": labels}
	residual := normalized
	lowered := strings.ToLower(normalized)
	if strings.Contains(lowered, "before broadly reading") && strings.Contains(lowered, "resolve") && strings.Contains(lowered, "policy") && strings.Contains(lowered, "context mechanically") {
		machine["operations"] = []string{"AUTOMATED_REASONING_DURING_REPOSITORY_DOCUMENT_RETRIEVAL"}
		return machine, []any{}
	}
	if trigger := agentTrigger.FindStringSubmatch(normalized); trigger != nil {
		machine["trigger"] = strings.ToUpper(trigger[1])
		loc := agentTrigger.FindStringIndex(residual)
		residual = strings.Trim(residual[loc[1]:], " ,:;-")
	}
	for _, modal := range agentModal {
		if modal.rx.MatchString(normalized) {
			machine["modality"] = modal.name
			residual = modal.rx.ReplaceAllString(residual, "")
			break
		}
	}
	refs := []string{}
	for _, match := range agentBacktick.FindAllStringSubmatch(normalized, -1) {
		if !agentHas(refs, match[1]) {
			refs = append(refs, match[1])
		}
	}
	if len(refs) > 0 {
		machine["references"] = refs
		for _, ref := range refs {
			residual = strings.ReplaceAll(residual, "`"+ref+"`", " ")
		}
	}
	ops := []string{}
	for _, match := range agentMechanismAction.FindAllStringSubmatch(normalized, -1) {
		op := strings.ToUpper(strings.ReplaceAll(match[1], "-", "_"))
		if op == "RERUN" {
			op = "RE_RUN"
		}
		if !agentHas(ops, op) {
			ops = append(ops, op)
		}
	}
	if len(ops) > 0 {
		machine["operations"] = ops
		residual = agentMechanismAction.ReplaceAllString(residual, "")
	}
	if agentMechanismCommand.MatchString(normalized) {
		machine["command_semantics"] = "COMMAND_OR_TOOL_INVOCATION"
	}
	residual = agentSpace(residual)
	residual = regexp.MustCompile(`\s+([,.;:])`).ReplaceAllString(residual, "$1")
	residual = strings.Trim(residual, " ,:;-.")
	items := []any{}
	if residual != "" {
		items = append(items, Object{"role": "UNMECHANIZED_ARGUMENT", "text": residual})
	}
	return machine, items
}

func BuildAgentStage(items []any) (Object, Object, error) {
	order := []any{}
	passed := Object{}
	unresolved := []any{}
	seen := map[string]bool{}
	for _, raw := range items {
		atom := Map(raw)
		id := Text(atom["atom_id"])
		text, ok := atom["text"].(string)
		labels, labelsOK := atom["level_1"].([]any)
		headings, headingsOK := atom["heading_path"].([]any)
		if !headingsOK && atom["heading_path"] == nil {
			headings = []any{}
			headingsOK = true
		}
		if id == "" || seen[id] || !ok || !labelsOK || !headingsOK {
			return nil, nil, fmt.Errorf("invalid Level 1 source atom %s", id)
		}
		seen[id] = true
		if atom["unresolved"] == true || len(labels) == 0 {
			unresolved = append(unresolved, Object{"atom_id": id, "status": "UNRESOLVED", "heading_path": headings, "natural_language": text})
			continue
		}
		names := []string{}
		for _, rawLabel := range labels {
			label := Text(rawLabel)
			if !agentHas(AgentLevel1, label) {
				return nil, nil, fmt.Errorf("invalid Level 1 label on %s", id)
			}
			names = append(names, label)
		}
		machine, residual := MechanizeAgentAtom(text, names)
		order = append(order, id)
		passed[id] = Object{"atom_id": id, "pass_header": Object{"level": 1, "status": "PASS", "labels": labels}, "machine": machine, "natural_residual": residual}
	}
	status := "PASS"
	if len(unresolved) > 0 {
		status = "PASS_WITH_UNRESOLVED"
	}
	stage := Object{"schema_version": "ptsip-agent-progressive-reasoning/v1", "level": 1, "status": status, "input_contract": "SOURCE_NATURAL_LANGUAGE", "next_level_input_contract": "CURRENT_LEVEL_PASS_SUBSET_ONLY", "next_level_payload": "PASS_HEADER_PLUS_MACHINE_FIELDS_PLUS_NATURAL_RESIDUAL", "rerun_previous_level_for_next_level": false, "unresolved_is_direct_next_level_candidate": false, "unresolved_may_become_candidate_after_same_level_pass": true, "next_level_candidate_set_is_dynamic": true, "unresolved_blocks_passed_atoms": false, "pass_count": len(order), "unresolved_count": len(unresolved), "pass_order": order, "pass_by_atom": passed}
	unresolvedDoc := Object{"schema_version": "ptsip-agent-progressive-reasoning/v1", "level": 1, "routing_state": "UNRESOLVED", "count": len(unresolved), "items": unresolved}
	return stage, unresolvedDoc, nil
}
func agentSourceItems(r *Repository) ([]any, string, error) {
	text, err := agentText(r, "AGENTS.md")
	if err != nil {
		return nil, "", err
	}
	items := []any{}
	for _, atom := range ClassifyAgentMarkdown(text) {
		items = append(items, Object{"atom_id": atom.AtomID, "kind": atom.Kind, "line_start": atom.LineStart, "line_end": atom.LineEnd, "level_1": agentJSON(atom.Level1), "unresolved": atom.Unresolved, "text": atom.Text, "heading_path": agentJSON(atom.HeadingPath)})
	}
	return items, text, nil
}
func agentLegacyItems(registry Object) ([]any, error) {
	rawAtoms, ok := registry["atoms"].([]any)
	if !ok {
		return nil, fmt.Errorf("registry.atoms must be a list")
	}
	items := []any{}
	for _, raw := range rawAtoms {
		atom := Map(raw)
		text := Text(atom["instruction_text"])
		if text == "" {
			text = Text(atom["normalized_text"])
		}
		if text == "" {
			text = Text(Map(atom["source"])["raw_excerpt"])
		}
		if strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("atom %s has no natural-language source", Text(atom["atom_id"]))
		}
		headings := List(Map(atom["source"])["heading_path"])
		if headings == nil {
			headings = []any{}
		}
		labels := List(atom["level_1"])
		if labels == nil {
			labels = []any{}
		}
		items = append(items, Object{"atom_id": atom["atom_id"], "level_1": labels, "unresolved": atom["unresolved"], "text": agentSpace(text), "heading_path": headings})
	}
	return items, nil
}
func RenderCompactAgent(unresolved Object) (string, error) {
	items, ok := unresolved["items"].([]any)
	if !ok {
		return "", fmt.Errorf("unresolved.items must be a list")
	}
	lines := []string{"# AGENTS.md", "", agentDirective}
	last := ""
	if len(items) > 0 {
		lines = append(lines, "", "## Level 1 unresolved")
	}
	for _, raw := range items {
		item := Map(raw)
		natural, ok := item["natural_language"].(string)
		if !ok {
			return "", fmt.Errorf("invalid unresolved natural language")
		}
		context := []string{}
		for _, v := range List(item["heading_path"]) {
			text := Text(v)
			if strings.TrimSpace(text) != "" && strings.TrimSpace(text) != "AGENTS.md" {
				context = append(context, text)
			}
		}
		heading := strings.Join(context, " > ")
		if heading != "" && heading != last {
			lines = append(lines, "", "### "+heading)
			last = heading
		}
		lines = append(lines, "", natural)
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n \t") + "\n", nil
}
func agentProgressiveIndex(stage Object, text string) Object {
	return Object{"schema_version": "ptsip-agent-progressive-index/v1", "management_mode": "PROGRESSIVE_LEVEL_1", "progressive_reasoning": Object{"highest_materialized_level": 1, "per_atom_advancement": true, "level_1_ref": agentStageRef, "level_1_unresolved_ref": agentUnresolvedRef, "previous_level_rerun_forbidden": true, "provenance_is_reasoning_input": false, "current_next_level_candidate_count": agentInt(stage["pass_count"]), "next_level_candidate_set_is_dynamic": true, "unresolved_reassessment_source": agentUnresolvedRef, "unresolved_blocks_next_level_candidates": false, "source_state": agentCompactState}, "authority_refs": Object{"level_1_pass": agentStageRef, "level_1_unresolved": agentUnresolvedRef}, "entry_contract": Object{"syntax": "PTSIP_AGENT_ENTRY_V1", "entry_directive": agentDirective, "index_ref": ".agent/index.yaml", "pass_atoms_in_agents": false, "unresolved_natural_language_in_agents": true}, "integration": AgentIntegrationContract(), "source": Object{"path": "AGENTS.md", "sha256": agentHash(text)}}
}

func MigrateAgentLevel1(r *Repository) (Object, error) {
	if err := ValidateAgentTaxonomy(r); err != nil {
		return nil, err
	}
	index, indexErr := r.Read(".agent/index.yaml")
	registry, registryErr := r.Read(".agent/registry.yaml")
	if indexErr != nil && iwpPathExists(r, ".agent/index.yaml") {
		return nil, indexErr
	}
	if registryErr != nil && iwpPathExists(r, ".agent/registry.yaml") {
		return nil, registryErr
	}
	if indexErr == nil && registryErr != nil && Map(index["progressive_reasoning"]) == nil {
		return nil, fmt.Errorf("partial .agent surface exists and cannot be routed safely")
	}
	var stage, unresolved Object
	mode := "COMPACTED_FROM_AGENTS"
	if indexErr == nil && registryErr != nil && Map(index["progressive_reasoning"]) != nil {
		state := Text(Map(index["progressive_reasoning"])["source_state"])
		if state == agentCompactState {
			return nil, fmt.Errorf("Level 1 compact entry is already active; use check instead")
		}
		if state == "ROUTED_LEVEL_1" {
			var err error
			stage, unresolved, err = loadAgentStages(r, ".agent", index)
			if err != nil {
				return nil, err
			}
			mode = "COMPACTED_EXISTING_LEVEL_1"
		} else if state == "SOURCE_PRESERVED" {
			source := Map(index["source"])
			text, err := agentText(r, Text(source["path"]))
			if err != nil || agentHash(text) != source["sha256"] {
				return nil, fmt.Errorf("source-preserved AGENTS.md changed before route cutover")
			}
		} else {
			return nil, fmt.Errorf("partial .agent surface exists and cannot be routed safely")
		}
	}
	if stage == nil {
		var items []any
		var err error
		if indexErr == nil && registryErr == nil {
			items, err = agentLegacyItems(registry)
			mode = "MIGRATED_LEGACY_AGENT_SURFACE"
		} else if indexErr != nil && registryErr == nil {
			return nil, fmt.Errorf("partial .agent surface exists and cannot be routed safely")
		} else {
			items, _, err = agentSourceItems(r)
		}
		if err != nil {
			return nil, err
		}
		stage, unresolved, err = BuildAgentStage(items)
		if err != nil {
			return nil, err
		}
	}
	compact, err := RenderCompactAgent(unresolved)
	if err != nil {
		return nil, err
	}
	newIndex := agentProgressiveIndex(stage, compact)
	if mode == "MIGRATED_LEGACY_AGENT_SURFACE" {
		for _, raw := range List(registry["atoms"]) {
			atom := Map(raw)
			delete(atom, "instruction_text")
			delete(atom, "normalized_text")
			delete(Map(atom["source"]), "raw_excerpt")
		}
		delete(registry, "provenance_ref")
		registry["management_mode"] = "PROGRESSIVE_LEVEL_1"
		registry["reasoning_payload"] = "STAGED_ONLY"
		for key, value := range newIndex {
			index[key] = value
		}
		delete(index, "provenance_ref")
		delete(index, "authority_ref")
		delete(index, "route_contract")
		newIndex = index
		if err := r.WriteYAML(".agent/registry.yaml", registry, nil); err != nil {
			return nil, err
		}
	}
	if err := r.WriteJSON(path.Join(".agent", agentStageRef), stage, nil); err != nil {
		return nil, err
	}
	if err := r.WriteJSON(path.Join(".agent", agentUnresolvedRef), unresolved, nil); err != nil {
		return nil, err
	}
	if err := r.WriteYAML(".agent/index.yaml", newIndex, nil); err != nil {
		return nil, err
	}
	if err := r.AtomicWrite("AGENTS.md", []byte(compact), nil); err != nil {
		return nil, err
	}
	provenance, err := r.Path(".agent/provenance/AGENTS.pre-level1.md")
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(provenance); err == nil {
		if err := os.Remove(provenance); err != nil {
			return nil, err
		}
	}
	return Object{"mode": mode, "level": 1, "pass_count": stage["pass_count"], "unresolved_count": unresolved["count"], "current_next_level_candidate_count": agentInt(stage["pass_count"]), "next_level_candidate_set_is_dynamic": true, "unresolved_reassessment_source": agentUnresolvedRef, "unresolved_blocks_next_level_candidates": false, "integration": "LOCAL_CLI_ONLY"}, nil
}

func loadAgentStages(r *Repository, agentRoot string, index Object) (Object, Object, error) {
	progressive := Map(index["progressive_reasoning"])
	stageRef, err := agentProjectionRef(r, agentRoot, progressive["level_1_ref"], "level_1_ref")
	if err != nil {
		return nil, nil, err
	}
	unresolvedRef, err := agentProjectionRef(r, agentRoot, progressive["level_1_unresolved_ref"], "level_1_unresolved_ref")
	if err != nil {
		return nil, nil, err
	}
	stage, err := r.Read(stageRef)
	if err != nil {
		return nil, nil, err
	}
	unresolved, err := r.Read(unresolvedRef)
	return stage, unresolved, err
}
func resolveAgentProgressive(r *Repository, agentRoot string, index Object, operation string, labels []string) (Object, error) {
	progressive := Map(index["progressive_reasoning"])
	if agentInt(progressive["highest_materialized_level"]) != 1 || progressive["per_atom_advancement"] != true {
		return nil, fmt.Errorf("progressive Level 1 state is invalid")
	}
	stage, unresolved, err := loadAgentStages(r, agentRoot, index)
	if err != nil {
		return nil, err
	}
	if agentInt(stage["level"]) != 1 || agentInt(unresolved["level"]) != 1 {
		return nil, fmt.Errorf("Level 1 staged artifact mismatch")
	}
	order, ok := stage["pass_order"].([]any)
	passed := Map(stage["pass_by_atom"])
	rawUnresolved, unresolvedOK := unresolved["items"].([]any)
	if !ok || passed == nil || !unresolvedOK {
		return nil, fmt.Errorf("Level 1 staged collections invalid")
	}
	instructions := []any{}
	seen := map[string]bool{}
	for _, value := range order {
		id := Text(value)
		atom := Map(passed[id])
		header := Map(atom["pass_header"])
		itemLabels, ok := header["labels"].([]any)
		if id == "" || atom["atom_id"] != id || !ok || Map(atom["machine"]) == nil || List(atom["natural_residual"]) == nil {
			return nil, fmt.Errorf("invalid direct Level 1 atom %s", id)
		}
		selected := false
		for _, value := range itemLabels {
			label := Text(value)
			if !agentHas(AgentLevel1, label) {
				return nil, fmt.Errorf("invalid pass label %s", id)
			}
			selected = selected || agentHas(labels, label)
		}
		if !selected || seen[id] {
			continue
		}
		instructions = append(instructions, Object{"atom_id": id, "stage_ref": path.Join(agentRoot, agentStageRef) + "#/pass_by_atom/" + id, "pass_header": header, "machine": atom["machine"], "natural_residual": atom["natural_residual"]})
		seen[id] = true
	}
	unresolvedItems := []any{}
	for _, raw := range rawUnresolved {
		item := Map(raw)
		id := Text(item["atom_id"])
		natural, ok := item["natural_language"].(string)
		if id == "" || !ok {
			return nil, fmt.Errorf("invalid Level 1 unresolved item")
		}
		headings := List(item["heading_path"])
		if headings == nil {
			headings = []any{}
		}
		unresolvedItems = append(unresolvedItems, Object{"atom_id": id, "status": "UNRESOLVED", "heading_path": headings, "natural_language": natural})
	}
	return Object{"schema_version": "ptsip-agent-instruction-entry-resolution/v2", "stage": "PROGRESSIVE_LEVEL_1", "operation": operation, "selected_level_1": labels, "scope_filtering": "NOT_AVAILABLE_AT_LEVEL_1", "level_2_used": false, "previous_level_rerun": false, "unresolved_policy": "ALWAYS_INCLUDE_UNTIL_CLASSIFIED", "instruction_count": len(instructions), "unresolved_count": len(unresolvedItems), "instructions": instructions, "unresolved": unresolvedItems}, nil
}

func CheckAgentProgressive(r *Repository) ([]string, error) {
	index, err := r.Read(".agent/index.yaml")
	if err != nil {
		return nil, err
	}
	progressive := Map(index["progressive_reasoning"])
	if agentInt(progressive["highest_materialized_level"]) != 1 || progressive["per_atom_advancement"] != true {
		return []string{"LEVEL_1_PROGRESSIVE_MODE_NOT_ACTIVE"}, nil
	}
	stage, unresolved, err := loadAgentStages(r, ".agent", index)
	if err != nil {
		return []string{"LEVEL_1_STAGE_INVALID"}, nil
	}
	errors := []string{}
	add := func(condition bool, label string) {
		if condition {
			errors = append(errors, label)
		}
	}
	add(agentInt(stage["level"]) != 1 || agentInt(unresolved["level"]) != 1, "LEVEL_1_STAGE_MISMATCH")
	order, ok := stage["pass_order"].([]any)
	passed := Map(stage["pass_by_atom"])
	count := agentInt(stage["pass_count"])
	ids := map[string]bool{}
	for _, raw := range order {
		ids[Text(raw)] = true
	}
	mismatch := !ok || passed == nil || count != len(order) || count != len(passed) || len(ids) != len(passed)
	for id := range passed {
		mismatch = mismatch || !ids[id]
	}
	add(mismatch, "LEVEL_1_PASS_COUNT_MISMATCH")
	add(agentInt(unresolved["count"]) != len(List(unresolved["items"])), "LEVEL_1_UNRESOLVED_COUNT_MISMATCH")
	add(agentInt(progressive["current_next_level_candidate_count"]) != count, "NEXT_LEVEL_CANDIDATE_COUNT_MISMATCH")
	add(progressive["next_level_candidate_set_is_dynamic"] != true, "NEXT_LEVEL_CANDIDATE_SET_MUST_BE_DYNAMIC")
	add(progressive["unresolved_blocks_next_level_candidates"] != false, "UNRESOLVED_MUST_NOT_BLOCK_PASSED_ATOMS")
	state := Text(progressive["source_state"])
	if state == agentCompactState {
		source := Map(index["source"])
		if source == nil {
			errors = append(errors, "SOURCE_BINDING_MISSING")
		} else {
			ref := Text(source["path"])
			digest := Text(source["sha256"])
			if ref == "" || digest == "" {
				errors = append(errors, "SOURCE_BINDING_INVALID")
			} else {
				text, err := agentText(r, ref)
				if err != nil {
					errors = append(errors, "COMPACT_AGENTS_MISSING")
				} else {
					add(agentHash(text) != digest, "COMPACT_AGENTS_STALE")
					add(strings.Count(text, agentDirective) != 1, "AGENT_ENTRY_DIRECTIVE_INVALID")
					add(strings.Contains(text, "PTSIP_AGENT_ROUTE"), "PER_ATOM_ROUTE_MUST_NOT_BE_IN_AGENTS")
					for _, raw := range List(unresolved["items"]) {
						item := Map(raw)
						natural, ok := item["natural_language"].(string)
						if ok && !strings.Contains(agentSpace(text), agentSpace(natural)) {
							errors = append(errors, "UNRESOLVED_NATURAL_LANGUAGE_MISSING:"+Text(item["atom_id"]))
						}
					}
				}
			}
		}
		entry := Map(index["entry_contract"])
		if entry == nil {
			errors = append(errors, "ENTRY_CONTRACT_MISSING")
		} else {
			add(entry["pass_atoms_in_agents"] != false, "PASS_ATOMS_MUST_NOT_BE_IN_AGENTS")
			add(entry["unresolved_natural_language_in_agents"] != true, "UNRESOLVED_AGENTS_CONTRACT_INVALID")
		}
		integration := Map(index["integration"])
		if integration == nil {
			errors = append(errors, "INTEGRATION_CONTRACT_MISSING")
		} else {
			add(integration["mode"] != "LOCAL_CLI_ONLY", "INTEGRATION_MODE_MISMATCH")
			add(Map(integration["mcp"])["state"] != "ABSENT", "MCP_ABSENT_STATE_REQUIRED")
		}
	} else if state == "ROUTED_LEVEL_1" || state == "MANAGED_BOOTSTRAP" {
		errors = append(errors, "ENTRY_COMPACTION_REQUIRED")
	} else {
		errors = append(errors, "UNKNOWN_PROGRESSIVE_SOURCE_STATE")
	}
	if ref, err := r.Path(".agent/provenance/AGENTS.pre-level1.md"); err == nil {
		if _, err := os.Stat(ref); err == nil {
			errors = append(errors, "PROVENANCE_MD_MUST_NOT_BE_REASONING_SURFACE")
		}
	}
	return errors, nil
}
