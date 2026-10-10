package machine

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var agentRepresentativeOperations = []string{"PTSIP-OP-ADOPT-001", "PTSIP-OP-VALIDATE-001", "PTSIP-OP-CONFORM-001", "PTSIP-OP-RECONCILE-AUTHORITY-001", "PTSIP-OP-MIGRATE-PROFILE-001"}
var agentStateDomains = []string{"developer_policy", "developer_planning", "project_profile", "policy_plan_binding", "agent_contract", "governance_source", "context_migration"}
var agentRetiredProfileRoots = []string{"MEMORY.md", "STATUS.md", "ptsip.yaml", "spec", "adoption", "agents", "docs/planning"}
var agentNormativeMarkdownTargets = []string{"adoption/ADOPTION-GUIDE.md", "agents/AGENT-CONTRACT.md", "spec/PTSIP-CONFORMANCE.md", "spec/PTSIP-DRAFT-PROFILE-TRANSITION.md", "spec/PTSIP-GOVERNANCE.md", "spec/PTSIP-RESPONSIBILITY-MAP.md", "spec/PTSIP-SPEC.md", "spec/PTSIP-TERMINOLOGY.md"}

func agentCheck(id string, passed bool, detail any) Object {
	status := "FAIL"
	if passed {
		status = "PASS"
	}
	return Object{"id": id, "status": status, "detail": detail}
}
func normalizeAgentSelector(pattern string) string {
	text := strings.TrimSpace(NormalizeReference(pattern))
	for strings.HasPrefix(text, "./") {
		text = text[2:]
	}
	return text
}

func VerifyCurrentAgentProfileSelectors(r *Repository) Object {
	profiles := []string{}
	violations := []any{}
	errors := []string{}
	state, err := r.State("project_profile")
	if err != nil {
		errors = append(errors, err.Error())
	} else {
		profiles = append(profiles, Text(state["ref"]))
	}
	index, err := r.Read(".ptsip/index.json")
	if err != nil {
		errors = append(errors, err.Error())
	} else {
		if err := r.Validate("src/ptsip/repository/schemas/repository-index.schema.json", index); err != nil {
			errors = append(errors, err.Error())
		} else {
			namespace := Map(Map(index["namespaces"])["profiles"])
			if namespace["status"] != "ACTIVE" {
				errors = append(errors, ".ptsip/index.json: profiles namespace must be ACTIVE")
			} else {
				ref := Text(namespace["index"])
				if !agentSafeRef(ref) {
					errors = append(errors, "unsafe local profile index reference")
				} else {
					catalogRef := path.Join(".ptsip", ref)
					catalog, err := r.Read(catalogRef)
					if err != nil {
						errors = append(errors, err.Error())
					} else {
						if iwpPathExists(r, ".ptsip/profiles/index.yaml") {
							errors = append(errors, "Canonical and legacy local profile indexes both exist")
						}
						if catalog["schema_version"] != "ptsip-local-profile-catalog/v1" {
							errors = append(errors, "Local profile catalog schema_version unsupported")
						}
						rows, ok := catalog["profiles"].([]any)
						if !ok || len(rows) == 0 {
							errors = append(errors, "Local profile catalog profiles must be non-empty")
						}
						ids, resources := map[string]bool{}, map[string]bool{}
						defaultFound := false
						for _, raw := range rows {
							row := Map(raw)
							id, resource := Text(row["id"]), Text(row["resource"])
							if id == "" || ids[id] || resources[resource] || !agentSafeRef(resource) || path.Base(resource) != resource || !strings.HasSuffix(resource, ".ptsip.yaml") {
								errors = append(errors, "invalid or duplicate local profile catalog row")
								continue
							}
							ids[id] = true
							resources[resource] = true
							profile := path.Join(".ptsip/profiles", resource)
							profiles = append(profiles, profile)
							if catalog["default_profile"] == id {
								defaultFound = true
								if !iwpPathExists(r, profile) {
									errors = append(errors, "Default local profile resource missing: "+profile)
								}
							}
						}
						if !defaultFound {
							errors = append(errors, "default_profile does not resolve to a catalog entry")
						}
					}
				}
			}
		}
	}
	profiles = UniqueStrings(profiles)
	for _, ref := range profiles {
		profile, err := r.Read(ref)
		if err != nil {
			errors = append(errors, ref+": "+err.Error())
			continue
		}
		for _, collection := range []string{"components", "associated_artifacts"} {
			items, ok := profile[collection].([]any)
			if !ok && collection == "associated_artifacts" && profile[collection] == nil {
				items = []any{}
				ok = true
			}
			if !ok || collection == "components" && len(items) == 0 {
				errors = append(errors, ref+": "+collection+" must be a list")
				continue
			}
			for position, raw := range items {
				item := Map(raw)
				id := Text(item["id"])
				if id == "" {
					errors = append(errors, fmt.Sprintf("%s: %s[%d] must have non-empty id", ref, collection, position))
					continue
				}
				fields := []string{"include"}
				if collection == "components" {
					fields = append(fields, "analysis_inputs")
				}
				for _, field := range fields {
					selectors, ok := item[field].([]any)
					if !ok && field == "analysis_inputs" && item[field] == nil {
						selectors = []any{}
						ok = true
					}
					if !ok || field == "include" && len(selectors) == 0 {
						errors = append(errors, ref+": "+collection+"["+id+"]."+field+" must be a list")
						continue
					}
					for _, rawSelector := range selectors {
						selector, ok := rawSelector.(string)
						normalized := normalizeAgentSelector(selector)
						if !ok || normalized == "" {
							errors = append(errors, ref+": invalid selector")
							continue
						}
						for _, retired := range agentRetiredProfileRoots {
							if normalized == retired || strings.HasPrefix(normalized, retired+"/") {
								violations = append(violations, Object{"profile": ref, "item": collection + "[" + id + "]." + field, "selector": selector, "retired_root": retired})
								break
							}
						}
					}
				}
			}
		}
	}
	return agentCheck("CURRENT_PROJECT_PROFILE_SELECTORS_REVALIDATED", len(errors) == 0 && len(violations) == 0, Object{"profiles": profiles, "retired_selectors": violations, "errors": errors})
}

func VerifyAgentOperationImplementationRefs(r *Repository) Object {
	const root = "src/ptsip/agent_contracts"
	errors := []string{}
	checked := []any{}
	index, err := r.Read(path.Join(root, "index.yaml"))
	if err != nil {
		return agentCheck("OPERATION_IMPLEMENTATION_REFS_EXIST", false, Object{"checked": checked, "errors": []string{err.Error()}})
	}
	entries, ok := index["operations"].([]any)
	if !ok || len(entries) == 0 {
		return agentCheck("OPERATION_IMPLEMENTATION_REFS_EXIST", false, Object{"checked": checked, "errors": []string{"embedded index operations must be a non-empty list"}})
	}
	seen := map[string]bool{}
	for _, raw := range entries {
		entry := Map(raw)
		ref := Text(entry["ref"])
		if !agentSafeRef(ref) || path.Dir(ref) != "operations" || !strings.HasSuffix(ref, ".yaml") || seen[ref] {
			errors = append(errors, "unsafe or duplicate embedded operation ref: "+ref)
			continue
		}
		seen[ref] = true
		operationRef := path.Join(root, ref)
		operation, err := r.Read(operationRef)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		id := Text(operation["operation_id"])
		refs, ok := operation["implementation_refs"].([]any)
		if id == "" || !ok || len(refs) == 0 {
			errors = append(errors, operationRef+": operation identity and implementation refs required")
			continue
		}
		for _, raw := range refs {
			implementation, ok := raw.(string)
			if !ok || !agentSafeRef(implementation) || strings.ContainsAny(implementation, "*?[:") {
				errors = append(errors, operationRef+": unsafe implementation ref "+implementation)
				continue
			}
			if !iwpPathExists(r, implementation) {
				errors = append(errors, operationRef+": referenced implementation does not exist: "+implementation)
				continue
			}
			file, err := r.Path(implementation)
			if err != nil {
				errors = append(errors, err.Error())
				continue
			}
			info, err := os.Stat(file)
			if err != nil || info.IsDir() {
				errors = append(errors, operationRef+": implementation must be a file: "+implementation)
				continue
			}
			checked = append(checked, Object{"operation": id, "ref": implementation})
		}
	}
	absolute, err := r.Path(path.Join(root, "operations"))
	if err != nil {
		errors = append(errors, err.Error())
	} else {
		files, err := filepath.Glob(filepath.Join(absolute, "*.yaml"))
		if err != nil {
			errors = append(errors, err.Error())
		}
		for _, file := range files {
			ref := "operations/" + filepath.Base(file)
			if !seen[ref] {
				errors = append(errors, "embedded operation resource missing from index: "+ref)
			}
		}
	}
	return agentCheck("OPERATION_IMPLEMENTATION_REFS_EXIST", len(errors) == 0, Object{"checked": checked, "errors": errors})
}

func VerifyAgentContextMigration(r *Repository, stage string) (Object, error) {
	stage = strings.ToUpper(stage)
	if stage == "CONTEXT_EVIDENCE_CANDIDATE" {
		return VerifyContextEvidenceCandidate(r)
	}
	if stage == "" {
		stage = "AUTO"
	}
	if stage != "M5" && stage != "M8" && stage != "AUTO" {
		return nil, fmt.Errorf("unsupported verification stage: %s", stage)
	}
	index, err := r.Read("src/agent_contracts/index.yaml")
	if err != nil {
		return nil, err
	}
	contractSet := Map(index["contract_set"])
	if contractSet == nil {
		return nil, fmt.Errorf("Agent Contract contract_set missing")
	}
	resolver, err := NewResolver(r)
	if err != nil {
		return nil, err
	}
	policy, err := resolver.Policy("MPD-GOV-0001")
	if err != nil {
		return nil, err
	}
	if Map(policy["rules"])["unit_mpd_0012_e0cfe7d01329"] == nil {
		return nil, fmt.Errorf("exact machine migration authority section missing")
	}
	active := Map(policy["policy"])["status"] == "ACTIVE"
	if stage == "AUTO" {
		stage = "M5"
		if active && contractSet["status"] == "CURRENT" {
			stage = "M8"
		}
	}
	checks := []any{}
	graph, graphErr := loadAgentContractGraph(r)
	if graphErr == nil {
		graphErr = graph.validate()
	}
	if graphErr != nil {
		checks = append(checks, agentCheck("AGENT_CONTRACT_PLANE_STRUCTURAL_VALIDATION_PASS", false, graphErr.Error()))
	} else {
		checks = append(checks, agentCheck("AGENT_CONTRACT_PLANE_STRUCTURAL_VALIDATION_PASS", true, graph.counts))
	}
	checks = append(checks, VerifyAgentOperationImplementationRefs(r))
	coverage, err := r.Read("developer/planning/migrations/MPD-0012-agent-context-coverage.yaml")
	if err != nil {
		return nil, err
	}
	expected := Strings(Map(coverage["m1"])["stable_rule_gap_before_m2"])
	missing := []string{}
	for _, id := range expected {
		if graph == nil || graph.rules[id] == nil {
			missing = append(missing, id)
		}
	}
	checks = append(checks, agentCheck("M1_STABLE_RULE_GAP_MATERIALIZED", len(expected) > 0 && len(missing) == 0, Object{"expected": len(expected), "missing": missing}))
	operationFailures, sizes := Object{}, Object{}
	for _, id := range agentRepresentativeOperations {
		if graph == nil {
			operationFailures[id] = "Agent Contract graph unavailable"
			continue
		}
		resolved, err := graph.resolveOperation(id)
		if err != nil {
			operationFailures[id] = err.Error()
			continue
		}
		markdown := false
		for _, text := range agentAllStrings(resolved) {
			lower := strings.ToLower(text)
			markdown = markdown || strings.HasSuffix(lower, ".md") || strings.Contains(lower, ".md#")
		}
		if markdown {
			operationFailures[id] = "resolved operation contains Markdown dependency"
			continue
		}
		size := Object{"rules": len(List(resolved["rules"]))}
		for _, field := range []string{"actions", "conditions", "gates", "io_schemas", "vocabularies"} {
			size[field] = len(Map(resolved[field]))
		}
		sizes[id] = size
	}
	checks = append(checks, agentCheck("REPRESENTATIVE_OPERATIONS_RESOLVE_EXACTLY", len(operationFailures) == 0, Object{"failures": operationFailures, "sizes": sizes}))
	stateFailures, stateRefs := Object{}, Object{}
	for _, domain := range agentStateDomains {
		state, err := r.State(domain)
		if err != nil {
			stateFailures[domain] = err.Error()
		} else {
			stateRefs[domain] = state["ref"]
		}
	}
	stateIndex, err := r.Read("developer/state/index.yaml")
	if err != nil {
		return nil, err
	}
	defaults := Map(stateIndex["default_agent_context"])
	expectedDefaults := Object{"prose_history_required": false, "status_markdown_required": false, "memory_markdown_required": false, "reference_markdown_required": false}
	stateOK := len(stateFailures) == 0 && len(defaults) == len(expectedDefaults)
	for k, v := range expectedDefaults {
		stateOK = stateOK && defaults[k] == v
	}
	checks = append(checks, agentCheck("BOUNDED_REPOSITORY_STATE_ROUTING", stateOK, Object{"failures": stateFailures, "refs": stateRefs, "default_agent_context": defaults}))
	dependency := Map(index["dependency_policy"])
	forbidden := dependency["markdown_normative_dependency"] == "FORBIDDEN"
	checks = append(checks, agentCheck("NORMATIVE_MARKDOWN_DEPENDENCY_FORBIDDEN", forbidden, Object{"dependency_policy": dependency}))
	if stage == "M8" {
		remaining := []string{}
		for _, ref := range agentNormativeMarkdownTargets {
			if iwpPathExists(r, ref) {
				remaining = append(remaining, ref)
			}
		}
		checks = append(checks, agentCheck("ROOT_AGENT_MIGRATION_AUTHORITY_ACTIVE", active, Object{"policy_ref": "MPD-GOV-0001#unit_mpd_0012_e0cfe7d01329", "status": Map(policy["policy"])["status"]}), agentCheck("AGENT_CONTRACT_SET_CURRENT", contractSet["status"] == "CURRENT", Object{"status": contractSet["status"]}), agentCheck("NORMATIVE_MARKDOWN_TARGETS_REMOVED", len(remaining) == 0, Object{"remaining": remaining}))
		var count any
		if stateOK && forbidden {
			count = 0
		}
		checks = append(checks, agentCheck("DEFAULT_AGENT_NORMATIVE_MARKDOWN_READ_COUNT_ZERO", stateOK && forbidden, Object{"count": count}), VerifyCurrentAgentProfileSelectors(r))
	}
	passed := true
	for _, raw := range checks {
		passed = passed && Map(raw)["status"] == "PASS"
	}
	status := "FAIL"
	if passed {
		status = "PASS"
	}
	return Object{"schema_version": "ptsip-agent-context-migration-verification/v1", "stage": stage, "status": status, "binding_verification": "STATIC_EXACT_SOURCE_DECLARATION", "checks": checks}, nil
}
