package machine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const cleanupWorkflow = "developer/automation/markdown_cleanup_workflow.yaml"
const cleanupRootPolicy = "MPD-GOV-0001"
const cleanupRootSection = "unit_mpd_0012_e0cfe7d01329"

func (r *Repository) CleanupWorkflow() (Object, error) {
	workflow, err := r.Read(cleanupWorkflow)
	if err != nil {
		return nil, err
	}
	if workflow["schema_version"] != "ptsip-markdown-cleanup-workflow/v1" || !equalStrings(Strings(workflow["stages"]), []string{"INSPECT", "PLAN", "VERIFY", "APPLY", "POST_VALIDATE"}) {
		return nil, Fail("CLEANUP_WORKFLOW_INVALID", "workflow version or stage order differs")
	}
	if workflow["policy_ref"] != cleanupRootPolicy+"#rules."+cleanupRootSection+".markdown_cleanup_preconditions" {
		return nil, Fail("CLEANUP_OWNER_MISMATCH", "workflow must reference the exact Root section")
	}
	if len(Strings(workflow["default_scopes"])) == 0 || len(List(workflow["targets"])) == 0 {
		return nil, Fail("CLEANUP_SCOPE_MISSING", "registered scopes and targets are required")
	}
	seen := map[string]bool{}
	for _, raw := range List(workflow["targets"]) {
		item := Map(raw)
		path := Text(item["path"])
		if !strings.HasSuffix(path, ".md") || seen[path] || item["disposition"] != "REMOVE_CANDIDATE" {
			return nil, Fail("CLEANUP_TARGET_INVALID", path)
		}
		seen[path] = true
		if !Has([]string{"NORMATIVE_RULE", "CODING_AGENT_BEHAVIOR", "OPERATION_PROCEDURE", "VALIDATION_OR_CONFORMANCE_SEMANTICS", "GOVERNANCE_OR_AUTHORITY_SEMANTICS", "PROFILE_TRANSITION_SEMANTICS", "NORMATIVE_TERMINOLOGY"}, Text(item["semantic_role"])) {
			return nil, Fail("CLEANUP_ROLE_UNREGISTERED", path)
		}
		if _, err := r.Path(path); err != nil {
			return nil, err
		}
	}
	decisions, err := r.RootSection(cleanupRootPolicy, cleanupRootSection)
	if err != nil {
		return nil, err
	}
	if Map(Map(decisions)["markdown_cleanup_preconditions"])["decision"] != "APPROVED" {
		return nil, Fail("CLEANUP_DECISION_NOT_APPROVED", "Root cleanup decision must be approved")
	}
	return workflow, nil
}
func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
func (r *Repository) RepositoryTextFiles() ([]string, error) {
	files, err := r.GitOutput("ls-files", "-co", "--exclude-standard", "-z")
	names := []string{}
	if err == nil {
		for _, path := range strings.Split(files, "\x00") {
			if path != "" {
				names = append(names, path)
			}
		}
	} else {
		skip := []string{".git", ".venv", "venv", "__pycache__", ".pytest_cache", ".mypy_cache", ".ruff_cache", "node_modules", "build", "dist"}
		if err := filepath.WalkDir(r.Root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if Has(skip, entry.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			relative, _ := filepath.Rel(r.Root, path)
			names = append(names, filepath.ToSlash(relative))
			return nil
		}); err != nil {
			return nil, err
		}
	}
	names = UniqueStrings(names)
	sort.Strings(names)
	return names, nil
}
func (r *Repository) MarkdownReferenceHits(target string, exclusions []string) ([]any, error) {
	files, err := r.RepositoryTextFiles()
	if err != nil {
		return nil, err
	}
	hits := []any{}
	for _, path := range files {
		if path == target || Has(exclusions, path) {
			continue
		}
		full, err := r.Path(path)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(full)
		if err != nil || info.IsDir() || info.Size() > 2_000_000 {
			continue
		}
		data, err := os.ReadFile(full)
		if err != nil || !utf8.Valid(data) {
			continue
		}
		kind := "ACTIVE_MACHINE_DEPENDENCY"
		if ext := strings.ToLower(filepath.Ext(path)); ext == ".md" || ext == ".rst" {
			kind = "HISTORICAL_OR_HUMAN_REFERENCE"
		}
		for line, text := range strings.Split(string(data), "\n") {
			if strings.Contains(text, target) || strings.Contains(text, strings.ReplaceAll(target, "/", "\\")) {
				hits = append(hits, Object{"source_path": path, "line": line + 1, "kind": kind})
			}
		}
	}
	return hits, nil
}
func (r *Repository) InspectMarkdown(scopes []string) (Object, error) {
	workflow, err := r.CleanupWorkflow()
	if err != nil {
		return nil, err
	}
	if len(scopes) == 0 {
		scopes = Strings(workflow["default_scopes"])
	}
	registered := map[string]Object{}
	for _, raw := range List(workflow["targets"]) {
		entry := Map(raw)
		registered[Text(entry["path"])] = entry
	}
	inventory := map[string]bool{}
	for _, scope := range scopes {
		root, err := r.Path(scope)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		}
		if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".md") {
				relative, _ := filepath.Rel(r.Root, path)
				inventory[filepath.ToSlash(relative)] = true
			}
			return nil
		}); err != nil {
			return nil, err
		}
		for path := range registered {
			if path == scope || strings.HasPrefix(path, strings.TrimRight(scope, "/")+"/") {
				inventory[path] = true
			}
		}
	}
	// Absent scopes still include their explicitly registered targets.
	for path := range registered {
		for _, scope := range scopes {
			if path == scope || strings.HasPrefix(path, strings.TrimRight(scope, "/")+"/") {
				inventory[path] = true
			}
		}
	}
	paths := []string{}
	for path := range inventory {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	items := []any{}
	counts := Object{}
	for _, path := range paths {
		entry := registered[path]
		hits, err := r.MarkdownReferenceHits(path, Strings(workflow["reference_scan_exclusions"]))
		if err != nil {
			return nil, err
		}
		active := false
		for _, raw := range hits {
			if Map(raw)["kind"] == "ACTIVE_MACHINE_DEPENDENCY" {
				active = true
			}
		}
		state := "UNRESOLVED"
		var digest any
		full, err := r.Path(path)
		if err != nil {
			return nil, err
		}
		data, readErr := os.ReadFile(full)
		if readErr != nil {
			if !os.IsNotExist(readErr) {
				return nil, readErr
			}
			if entry != nil {
				state = "ALREADY_ABSENT"
				if active {
					state = "BLOCKED_BY_ACTIVE_REFERENCE"
				}
			}
		} else {
			digest = SHA256(data)
			if entry != nil {
				state = "REMOVE_CANDIDATE"
				if active {
					state = "BLOCKED_BY_ACTIVE_REFERENCE"
				}
			}
		}
		count, _ := counts[state].(int)
		counts[state] = count + 1
		var role any
		if entry != nil {
			role = entry["semantic_role"]
		}
		items = append(items, Object{"path": path, "registered": entry != nil, "semantic_role": role, "state": state, "sha256": digest, "references": hits})
	}
	head, err := r.GitOutput("rev-parse", "HEAD")
	var repositoryHead any
	if err == nil {
		repositoryHead = head
	}
	return Object{"schema_version": "ptsip-markdown-cleanup-inspection/v1", "policy_ref": workflow["policy_ref"], "repository_head": repositoryHead, "scopes": scopes, "counts": counts, "items": items}, nil
}
func (r *Repository) MarkdownReadiness(inspection Object) (Object, error) {
	if inspection == nil {
		var err error
		inspection, err = r.InspectMarkdown(nil)
		if err != nil {
			return nil, err
		}
	}
	resolver, err := NewResolver(r)
	if err != nil {
		return nil, err
	}
	policy, err := resolver.Policy(cleanupRootPolicy)
	if err != nil {
		return nil, err
	}
	counts, structuralErr := ValidateAgentContractPlane(r)
	structural := "PASS"
	var errorDetail any
	if structuralErr != nil {
		structural = "FAIL"
		errorDetail = structuralErr.Error()
	}
	index, err := r.Read("src/agent_contracts/index.yaml")
	if err != nil {
		return nil, err
	}
	contract := Map(index["contract_set"])
	dependency := Map(index["dependency_policy"])
	sourceOK := contract["status"] == "CURRENT" && contract["authority_scope"] == "CODING_AGENT_BEHAVIOR" && dependency["markdown_normative_dependency"] == "FORBIDDEN"
	core, err := r.Read("src/agent_contracts/spec/core.yaml")
	if err != nil {
		return nil, err
	}
	failClosed := false
	for _, raw := range List(core["rules"]) {
		rule := Map(raw)
		if rule["subject"] == "unresolved_normative_input" && rule["predicate"] == "behavior" && rule["object"] == "FAIL_CLOSED" {
			failClosed = true
		}
	}
	vocabulary, err := r.Read("src/agent_contracts/vocabulary/outcomes.yaml")
	if err != nil {
		return nil, err
	}
	outcomes := []string{}
	for _, raw := range List(vocabulary["entries"]) {
		outcomes = append(outcomes, Text(Map(raw)["id"]))
	}
	present := []string{}
	missing := []string{}
	for _, required := range []string{"INCOMPLETE", "OWNER_DECISION_REQUIRED"} {
		if Has(outcomes, required) {
			present = append(present, required)
		} else {
			missing = append(missing, required)
		}
	}
	references := []any{}
	for _, raw := range List(inspection["items"]) {
		item := Map(raw)
		for _, rawRef := range List(item["references"]) {
			reference := Map(rawRef)
			if reference["kind"] == "ACTIVE_MACHINE_DEPENDENCY" {
				references = append(references, Object{"target": item["path"], "source_path": reference["source_path"], "line": reference["line"]})
			}
		}
	}
	checks := []any{Object{"id": "AGENT_CONTRACT_PLANE_STRUCTURAL_VALIDATION_PASS", "status": structural, "detail": Object{"counts": counts, "error": errorDetail}}, agentCheck("NO_ACTIVE_MACHINE_DEPENDENCY_ON_REMOVAL_TARGETS", len(references) == 0, Object{"active_references": references}), agentCheck("SRC_AGENT_CONTRACTS_CONFIRMED_AS_NORMATIVE_SOURCE", sourceOK, Object{"contract_set_status": contract["status"], "authority_scope": contract["authority_scope"], "markdown_normative_dependency": dependency["markdown_normative_dependency"]}), agentCheck("UNRESOLVED_SEMANTICS_CAN_BE_REPRESENTED_WITHOUT_LEGACY_FALLBACK", failClosed && len(missing) == 0, Object{"fail_closed_unresolved_normative_input": failClosed, "required_outcomes_present": present, "required_outcomes_missing": missing})}
	ready := "READY"
	for _, raw := range checks {
		if Map(raw)["status"] != "PASS" {
			ready = "BLOCKED"
		}
	}
	status := Map(policy["policy"])["status"]
	authorized := status == "ACTIVE"
	reason := cleanupRootPolicy + "_NOT_ACTIVE"
	if authorized {
		reason = cleanupRootPolicy + "_ACTIVE"
	}
	return Object{"schema_version": "ptsip-markdown-cleanup-readiness/v1", "policy_status": status, "cleanup_readiness": ready, "apply_authorized": authorized, "apply_authorization_reason": reason, "checks": checks}, nil
}
func (r *Repository) MarkdownPlan(scopes []string) (Object, error) {
	inspected, err := r.InspectMarkdown(scopes)
	if err != nil {
		return nil, err
	}
	ready, err := r.MarkdownReadiness(inspected)
	if err != nil {
		return nil, err
	}
	remove, blocked, unresolved := []string{}, []string{}, []string{}
	hashes := Object{}
	for _, raw := range List(inspected["items"]) {
		item := Map(raw)
		path := Text(item["path"])
		switch item["state"] {
		case "REMOVE_CANDIDATE":
			remove = append(remove, path)
		case "BLOCKED_BY_ACTIVE_REFERENCE":
			blocked = append(blocked, path)
		case "UNRESOLVED":
			unresolved = append(unresolved, path)
		}
		if item["sha256"] != nil {
			hashes[path] = item["sha256"]
		}
	}
	state := "BLOCKED"
	if len(blocked) == 0 && len(unresolved) == 0 && ready["cleanup_readiness"] == "READY" {
		state = "READY"
	}
	return Object{"schema_version": "ptsip-markdown-cleanup-plan/v1", "policy_ref": inspected["policy_ref"], "repository_head": inspected["repository_head"], "state": state, "apply_authorized": ready["apply_authorized"], "remove": remove, "blocked": blocked, "unresolved": unresolved, "target_hashes": hashes, "inspection": inspected, "readiness": ready}, nil
}
func (r *Repository) ApplyMarkdownPlan(plan Object) (Object, error) {
	if plan["schema_version"] != "ptsip-markdown-cleanup-plan/v1" || plan["state"] != "READY" || plan["apply_authorized"] != true {
		return nil, Fail("CLEANUP_PLAN_NOT_AUTHORIZED", "plan must be admitted and ready")
	}
	fresh, err := r.MarkdownReadiness(nil)
	if err != nil {
		return nil, err
	}
	if fresh["apply_authorized"] != true || fresh["cleanup_readiness"] != "READY" {
		return nil, Fail("CLEANUP_READINESS_CHANGED", "current Root readiness no longer permits apply")
	}
	head, err := r.GitOutput("rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	if plan["repository_head"] != head {
		return nil, Fail("CLEANUP_PLAN_STALE", "HEAD changed after planning")
	}
	for path, expected := range Map(plan["target_hashes"]) {
		data, err := r.ReadSource(path)
		if err != nil {
			return nil, err
		}
		if SHA256(data) != expected {
			return nil, Fail("CLEANUP_TARGET_CHANGED", path)
		}
	}
	workflow, err := r.CleanupWorkflow()
	if err != nil {
		return nil, err
	}
	targets := map[string]bool{}
	for _, raw := range List(workflow["targets"]) {
		targets[Text(Map(raw)["path"])] = true
	}
	backups := map[string][]byte{}
	for _, path := range Strings(plan["remove"]) {
		if !targets[path] {
			return nil, Fail("CLEANUP_TARGET_UNREGISTERED", path)
		}
		data, err := r.ReadSource(path)
		if err != nil {
			return nil, err
		}
		backups[path] = data
	}
	restore := func() {
		for path, data := range backups {
			_ = r.AtomicWrite(path, data, nil)
		}
	}
	for path := range backups {
		full, err := r.Path(path)
		if err != nil {
			restore()
			return nil, err
		}
		if err := os.Remove(full); err != nil {
			restore()
			return nil, err
		}
	}
	post, err := r.MarkdownReadiness(nil)
	if err != nil || post["cleanup_readiness"] != "READY" {
		restore()
		return nil, Fail("CLEANUP_POST_VALIDATION_FAILED", "original repository files restored")
	}
	return Object{"schema_version": "ptsip-markdown-cleanup-apply/v1", "status": "APPLIED", "removed": plan["remove"], "post_readiness": post}, nil
}
func init() {
	RegisterOperations("markdown-cleanup", func(r *Repository, command string, opts map[string]string, args []string) (any, error) {
		scopes := []string{}
		if opts["--scope"] != "" {
			scopes = append(scopes, opts["--scope"])
		}
		var result any
		var err error
		switch command {
		case "inspect":
			result, err = r.InspectMarkdown(scopes)
		case "verify":
			result, err = r.MarkdownReadiness(nil)
		case "plan":
			result, err = r.MarkdownPlan(scopes)
		case "simulate":
			var plan Object
			plan, err = r.MarkdownPlan(scopes)
			if err == nil {
				result = Object{"schema_version": "ptsip-markdown-cleanup-simulation/v1", "status": plan["state"], "mutation_performed": false, "plan": plan}
			}
		case "apply":
			var plan Object
			plan, err = r.Read(opts["--plan"])
			if err == nil {
				result, err = r.ApplyMarkdownPlan(plan)
			}
		default:
			return nil, fmt.Errorf("unregistered cleanup command")
		}
		if err != nil {
			return nil, err
		}
		if output := opts["--output"]; output != "" {
			if err := r.WriteJSON(output, result, nil); err != nil {
				return nil, err
			}
		}
		return result, nil
	})
}
