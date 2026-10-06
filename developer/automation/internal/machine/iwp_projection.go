package machine

import (
	"fmt"
	"strings"
	"unicode/utf16"
)

// The packet protocol uses Python's historical ASCII JSON wire digest; this encoder
// preserves it independently of the language that executes the implementation.
func iwpCanonicalJSON(value any) ([]byte, error) {
	raw, err := CanonicalJSON(value)
	if err != nil {
		return nil, err
	}
	out := strings.Builder{}
	for _, r := range string(raw) {
		if r < 128 {
			out.WriteRune(r)
		} else if r <= 0xffff {
			fmt.Fprintf(&out, "\\u%04x", r)
		} else {
			first, second := utf16.EncodeRune(r)
			fmt.Fprintf(&out, "\\u%04x\\u%04x", first, second)
		}
	}
	return []byte(out.String()), nil
}
func iwpSourceRefs(raw any, role string) ([]any, error) {
	refs, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s source-context references must be a list", role)
	}
	result := []any{}
	seen := map[string]bool{}
	for _, raw := range refs {
		item, err := iwpObject(raw, role+" source-context ref")
		if err != nil {
			return nil, err
		}
		ref := Text(item["path"])
		selector := Map(item["selector"])
		if ref == "" || selector == nil {
			return nil, fmt.Errorf("source-context ref has invalid path or selector")
		}
		bytes, err := iwpCanonicalJSON(Object{"path": ref, "selector": selector})
		if err != nil {
			return nil, err
		}
		id := role + "-" + SHA256(bytes)[:12]
		if seen[id] {
			return nil, fmt.Errorf("duplicate source-context id: %s", id)
		}
		seen[id] = true
		copy := Object{}
		for k, v := range item {
			copy[k] = v
		}
		copy["context_id"] = id
		result = append(result, copy)
	}
	return result, nil
}
func iwpRuleHeadlines(packet Object) ([]any, error) {
	policyContext, err := iwpObject(packet["policy_context"], "packet.policy_context")
	if err != nil {
		return nil, err
	}
	rules, ok := policyContext["normative_rules"].([]any)
	if !ok {
		return nil, fmt.Errorf("packet policy_context normative_rules must be a list")
	}
	result := []any{}
	for _, raw := range rules {
		rule, err := iwpObject(raw, "packet normative rule")
		if err != nil {
			return nil, err
		}
		registry, err := iwpObject(rule["registry_record"], "normative rule registry_record")
		if err != nil {
			return nil, err
		}
		result = append(result, Object{"rule_id": rule["rule_id"], "title": registry["title"], "severity": registry["severity"], "applies_to": registry["applies_to"], "canonical_source": rule["canonical_source"], "line_start": rule["line_start"], "line_end": rule["line_end"]})
	}
	return result, nil
}
func BuildWorkPacketBrief(packet Object) (Object, error) {
	context, err := iwpObject(packet["policy_context"], "packet.policy_context")
	if err != nil {
		return nil, err
	}
	verification, err := iwpObject(packet["verification"], "packet.verification")
	if err != nil {
		return nil, err
	}
	core, err := iwpObject(verification["core_regression"], "verification.core_regression")
	if err != nil {
		return nil, err
	}
	freshness, err := iwpObject(packet["freshness"], "packet.freshness")
	if err != nil {
		return nil, err
	}
	budget, err := iwpObject(packet["edit_budget"], "packet.edit_budget")
	if err != nil {
		return nil, err
	}
	headlines, err := iwpRuleHeadlines(packet)
	if err != nil {
		return nil, err
	}
	read, err := iwpSourceRefs(packet["read_context"], "read")
	if err != nil {
		return nil, err
	}
	cli := func(args ...string) []string {
		return append([]string{"go", "-C", "developer/automation", "run", "./cmd/ptsip-dev"}, args...)
	}
	return Object{"schema_version": "ptsip-agent-implementation-brief/v1", "projection_authority": false, "packet_id": packet["packet_id"], "task": packet["task"], "policy_refs": context["policies"], "normative_rules": headlines, "constraints": context["constraints"], "read_context": read, "mutation_plan": packet["mutation_plan"], "acceptance_vectors": packet["acceptance_vectors"], "guard_contract": Object{"unlisted_paths": budget["unlisted_paths"], "selector_removal_or_rename": budget["selector_removal_or_rename"]}, "verification": Object{"status": verification["status"], "baseline_pytest_nodes": verification["baseline_pytest_nodes"], "required_new_tests": verification["required_new_tests"], "missing_required_new_tests": verification["missing_required_new_tests"], "task_regression_pytest_targets": verification["task_regression_pytest_targets"], "core_regression": Object{"component_ref": core["component_ref"], "source": core["source"], "selection": core["selection"]}, "recommended_order": []string{"baseline", "focused", "task-regression", "core", "regression"}}, "freshness": Object{"baseline_head": freshness["baseline_head"], "strategy": freshness["strategy"]}, "on_demand": Object{"normative_rule": cli("policy-resolver", "rule", "<RULE_ID>"), "mutation_context": cli("implementation-work-packet", "context", "--packet", "<PACKET>", "--role", "mutation"), "read_context": cli("implementation-work-packet", "context", "--packet", "<PACKET>", "--role", "read", "--context-id", "<CONTEXT_ID>"), "read_context_all": cli("implementation-work-packet", "context", "--packet", "<PACKET>", "--role", "read")}}, nil
}
func BuildWorkPacketSourceContext(r *Repository, packet Object, role, contextID string) (Object, error) {
	if role != "mutation" && role != "read" && role != "all" {
		return nil, fmt.Errorf("unsupported source-context role: %s", role)
	}
	mutation, err := iwpObject(packet["mutation_plan"], "packet.mutation_plan")
	if err != nil {
		return nil, err
	}
	selected := []any{}
	if role == "mutation" || role == "all" {
		refs, err := iwpSourceRefs(mutation["targets"], "mutation")
		if err != nil {
			return nil, err
		}
		selected = append(selected, refs...)
	}
	if role == "read" || role == "all" {
		refs, err := iwpSourceRefs(packet["read_context"], "read")
		if err != nil {
			return nil, err
		}
		selected = append(selected, refs...)
	}
	var requested any
	if contextID != "" {
		requested = contextID
		matches := []any{}
		for _, raw := range selected {
			if Map(raw)["context_id"] == contextID {
				matches = append(matches, raw)
			}
		}
		if len(matches) != 1 {
			return nil, fmt.Errorf("source-context id must resolve exactly once for role %s: %s", role, contextID)
		}
		selected = matches
	}
	items := []any{}
	seen := map[string]bool{}
	for _, raw := range selected {
		ref := Map(raw)
		key := iwpSelectorKey(Text(ref["path"]), Map(ref["selector"]))
		if seen[key] {
			continue
		}
		seen[key] = true
		live, err := ValidateImplementationRef(r, ref)
		if err != nil {
			return nil, err
		}
		location := Map(live["resolved_location"])
		start, end := agentInt(location["line_start"]), agentInt(location["line_end"])
		text, err := agentText(r, Text(ref["path"]))
		if err != nil {
			return nil, err
		}
		lines := strings.SplitAfter(text, "\n")
		if start < 1 || end < start || end > len(lines) {
			return nil, fmt.Errorf("source-context selector has no exact line range")
		}
		source := strings.Join(lines[start-1:end], "")
		item := Object{"context_id": ref["context_id"], "path": ref["path"], "selector": ref["selector"], "resolved_location": location, "source_sha256": agentHash(source), "source": source}
		if rationale := Text(ref["rationale"]); rationale != "" {
			item["rationale"] = rationale
		}
		items = append(items, item)
	}
	return Object{"schema_version": "ptsip-agent-source-context/v1", "projection_authority": false, "packet_id": packet["packet_id"], "role": role, "requested_context_id": requested, "items": items}, nil
}
