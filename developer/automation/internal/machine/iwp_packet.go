package machine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const iwpRegistryRef = "developer/automation/implementation_workflows.yaml"
const iwpSchemaRef = "developer/automation/implementation_workflows.schema.json"

func iwpObject(value any, label string) (Object, error) {
	object := Map(value)
	if object == nil {
		return nil, fmt.Errorf("%s must be a mapping", label)
	}
	return object, nil
}
func iwpStringList(value any, label string, nonempty bool) ([]string, error) {
	list, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be a list", label)
	}
	result := []string{}
	if nonempty && len(list) == 0 {
		return nil, fmt.Errorf("%s must be non-empty", label)
	}
	for _, raw := range list {
		text, ok := raw.(string)
		if !ok || (nonempty && text == "") {
			return nil, fmt.Errorf("%s must contain strings", label)
		}
		result = append(result, text)
	}
	return result, nil
}
func iwpSortedUnique(values []string) []string {
	result := UniqueStrings(values)
	sort.Strings(result)
	return result
}
func iwpGit(r *Repository, args ...string) (string, error) {
	command := exec.Command("git", append([]string{"-C", r.Root}, args...)...)
	data, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(data)), nil
}
func iwpFileHash(r *Repository, ref string) (string, error) {
	file, err := r.Path(ref)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	return SHA256(data), nil
}
func iwpSelectorKey(ref string, selector Object) string {
	data, _ := iwpCanonicalJSON(Object{"path": ref, "selector": selector})
	return string(data)
}
func iwpSelectorFingerprint(r *Repository, reference Object) (string, error) {
	resolved, err := ValidateImplementationRef(r, reference)
	if err != nil {
		return "", err
	}
	location := Map(resolved["resolved_location"])
	text, err := agentText(r, Text(reference["path"]))
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	start, end := agentInt(location["line_start"]), agentInt(location["line_end"])
	if start < 1 || end < start || end > len(lines) {
		return "", fmt.Errorf("selector has no exact line range")
	}
	return agentHash(strings.Join(lines[start-1:end], "\n") + "\n"), nil
}
func iwpRegistry(r *Repository) (Object, error) {
	registry, err := r.Read(iwpRegistryRef)
	if err != nil {
		return nil, err
	}
	if err := r.Validate(iwpSchemaRef, registry); err != nil {
		return nil, fmt.Errorf("implementation workflow registry is invalid: %w", err)
	}
	return registry, nil
}
func iwpRecipe(registry Object, branch, scope, operation string) (Object, error) {
	matches := []Object{}
	tasks, ok := registry["tasks"].([]any)
	if !ok {
		return nil, fmt.Errorf("implementation workflow registry has no tasks")
	}
	for _, raw := range tasks {
		item := Map(raw)
		if item["branch"] == branch && item["scope"] == scope && item["operation"] == operation {
			matches = append(matches, item)
		}
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("expected one exact implementation workflow for %s:%s:%s, found %d", branch, scope, operation, len(matches))
	}
	return matches[0], nil
}
func iwpPathExists(r *Repository, ref string) bool {
	file, err := r.Path(ref)
	if err != nil {
		return false
	}
	_, err = os.Stat(file)
	return err == nil
}
func iwpTestMode(r *Repository, component string) Object {
	payload, err := r.Read(".github/test_modes.yaml")
	if err != nil {
		return Object{"status": "REGISTRY_MISSING", "component_ref": component}
	}
	modes, ok := payload["modes"].([]any)
	if !ok {
		return Object{"status": "REGISTRY_INVALID", "component_ref": component}
	}
	for _, raw := range modes {
		mode := Map(raw)
		if mode["component_ref"] == component {
			return Object{"status": "REGISTERED", "component_ref": component, "mode_id": mode["id"]}
		}
	}
	return Object{"status": "NOT_REGISTERED", "component_ref": component, "fallback": "CANONICAL_COMPONENT_INCLUDE_SELECTION"}
}

func iwpComponentTargets(r *Repository, componentRef, source string) ([]string, error) {
	profile, err := r.Read(source)
	if err != nil {
		return nil, err
	}
	matches := []Object{}
	for _, raw := range List(profile["components"]) {
		component := Map(raw)
		if component["id"] == componentRef {
			matches = append(matches, component)
		}
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("expected one verification component %q in %s, found %d", componentRef, source, len(matches))
	}
	component := matches[0]
	if !Has(Strings(component["roles"]), "VERIFICATION") {
		return nil, fmt.Errorf("%q is not a VERIFICATION component", componentRef)
	}
	includes, err := iwpStringList(component["include"], "component.include", true)
	if err != nil {
		return nil, err
	}
	targets := []string{}
	for _, raw := range includes {
		pattern := strings.ReplaceAll(raw, "\\", "/")
		if !strings.HasPrefix(pattern, "src/tests/") {
			continue
		}
		if strings.HasSuffix(pattern, "/**") {
			target := strings.TrimSuffix(pattern, "/**")
			if !iwpPathExists(r, target) {
				return nil, fmt.Errorf("core regression target does not exist: %s", target)
			}
			targets = append(targets, target)
			continue
		}
		if !strings.ContainsAny(pattern, "*?[") {
			if !iwpPathExists(r, pattern) {
				return nil, fmt.Errorf("core regression target does not exist: %s", pattern)
			}
			targets = append(targets, pattern)
			continue
		}
		absolute, err := r.Path(pattern)
		if err != nil {
			return nil, err
		}
		found, err := filepath.Glob(absolute)
		if err != nil {
			return nil, err
		}
		if len(found) == 0 {
			return nil, fmt.Errorf("core regression selector matched nothing: %s", pattern)
		}
		for _, file := range found {
			info, err := os.Stat(file)
			if err != nil {
				return nil, err
			}
			if !info.IsDir() {
				ref, err := r.Scope(file)
				if err != nil {
					return nil, err
				}
				targets = append(targets, ref)
			}
		}
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("%s produced no pytest targets", componentRef)
	}
	return UniqueStrings(targets), nil
}
func iwpMergeTargets(r *Repository, targets []string) []string {
	merged := []string{}
	directory := func(ref string) bool {
		file, err := r.Path(ref)
		if err != nil {
			return false
		}
		info, err := os.Stat(file)
		return err == nil && info.IsDir()
	}
	for _, target := range targets {
		target = strings.TrimSuffix(target, "/")
		covered := false
		for _, existing := range merged {
			if existing == target || (directory(existing) && strings.HasPrefix(target, existing+"/")) {
				covered = true
				break
			}
		}
		if covered {
			continue
		}
		next := []string{}
		for _, existing := range merged {
			if !directory(target) || !strings.HasPrefix(existing, target+"/") {
				next = append(next, existing)
			}
		}
		merged = append(next, target)
	}
	return merged
}

func BuildWorkPacket(r *Repository, scope, operation string) (Object, error) {
	registry, err := iwpRegistry(r)
	if err != nil {
		return nil, err
	}
	resolver, err := NewResolver(r)
	if err != nil {
		return nil, err
	}
	resolved, err := resolver.Resolve(scope, operation)
	if err != nil {
		return nil, err
	}
	return BuildWorkPacketFromResolution(r, registry, resolved, operation)
}
func BuildWorkPacketFromResolution(r *Repository, registry, resolved Object, operation string) (Object, error) {
	taskContext, err := iwpObject(resolved["task_context"], "task_context")
	if err != nil {
		return nil, err
	}
	branchContext, err := iwpObject(taskContext["branch_context"], "branch_context")
	if err != nil {
		return nil, err
	}
	branch := Text(branchContext["actual"])
	if branch == "" {
		return nil, fmt.Errorf("task context has no actual branch")
	}
	operation = strings.ToUpper(strings.TrimSpace(operation))
	recipe, err := iwpRecipe(registry, branch, Text(resolved["scope"]), operation)
	if err != nil {
		return nil, err
	}
	refs, ok := taskContext["implementation_refs"].([]any)
	if !ok || len(refs) == 0 {
		return nil, fmt.Errorf("task context has no implementation refs")
	}
	resolvedByKey := map[string]Object{}
	keys := []string{}
	for _, raw := range refs {
		item, err := iwpObject(raw, "implementation ref")
		if err != nil {
			return nil, err
		}
		selector, err := iwpObject(item["selector"], "implementation selector")
		if err != nil {
			return nil, err
		}
		key := iwpSelectorKey(Text(item["path"]), selector)
		if resolvedByKey[key] != nil {
			return nil, fmt.Errorf("Policy Resolver returned duplicate implementation refs")
		}
		resolvedByKey[key] = item
		keys = append(keys, key)
	}
	mutation, err := iwpObject(recipe["mutation"], "mutation")
	if err != nil {
		return nil, err
	}
	rawTargets, ok := mutation["targets"].([]any)
	if !ok || len(rawTargets) == 0 {
		return nil, fmt.Errorf("workflow mutation.targets must be non-empty")
	}
	editTargets := []any{}
	editKeys := map[string]bool{}
	codePaths := []string{}
	for _, raw := range rawTargets {
		target, err := iwpObject(raw, "mutation target")
		if err != nil {
			return nil, err
		}
		selector, err := iwpObject(target["selector"], "mutation target selector")
		if err != nil {
			return nil, err
		}
		key := iwpSelectorKey(Text(target["path"]), selector)
		matched := resolvedByKey[key]
		if matched == nil {
			return nil, fmt.Errorf("workflow mutation target is not present in Policy Resolver implementation refs")
		}
		copy := Object{}
		for k, v := range matched {
			copy[k] = v
		}
		copy["rationale"] = target["rationale"]
		editTargets = append(editTargets, copy)
		editKeys[key] = true
		codePaths = append(codePaths, Text(copy["path"]))
	}
	readContext := []any{}
	readPaths := []string{}
	for _, key := range keys {
		if !editKeys[key] {
			readContext = append(readContext, resolvedByKey[key])
			readPaths = append(readPaths, Text(resolvedByKey[key]["path"]))
		}
	}
	testRefs, err := iwpStringList(taskContext["test_refs"], "task context test refs", false)
	if err != nil {
		return nil, err
	}
	allowedTests, err := iwpStringList(mutation["allowed_test_paths"], "mutation.allowed_test_paths", true)
	if err != nil {
		return nil, err
	}
	for _, ref := range allowedTests {
		if !Has(testRefs, ref) {
			return nil, fmt.Errorf("mutation test path is not Policy Resolver test ref: %s", ref)
		}
	}
	acceptance, ok := recipe["acceptance_vectors"].([]any)
	if !ok || len(acceptance) == 0 {
		return nil, fmt.Errorf("workflow acceptance_vectors must be non-empty")
	}
	acceptanceIDs := map[string]bool{}
	declaredNodes := map[string]bool{}
	acceptanceCoverage := []any{}
	for _, raw := range acceptance {
		item, err := iwpObject(raw, "acceptance vector")
		if err != nil {
			return nil, err
		}
		id := Text(item["id"])
		if id == "" || acceptanceIDs[id] {
			return nil, fmt.Errorf("empty or duplicate acceptance vector id: %s", id)
		}
		acceptanceIDs[id] = true
		nodes, err := iwpStringList(item["test_nodes"], "acceptance.test_nodes", true)
		if err != nil {
			return nil, err
		}
		if _, err := iwpStringList(item["invariants"], "acceptance.invariants", true); err != nil {
			return nil, err
		}
		missing := []string{}
		for _, node := range nodes {
			declaredNodes[node] = true
			if !PythonTestNodeExists(r, node) {
				missing = append(missing, node)
			}
		}
		acceptanceCoverage = append(acceptanceCoverage, Object{"id": id, "test_nodes": nodes, "missing_test_nodes": missing, "covered": len(missing) == 0})
	}
	verification, err := iwpObject(recipe["verification"], "verification")
	if err != nil {
		return nil, err
	}
	baseline, err := iwpStringList(verification["baseline_pytest_nodes"], "verification.baseline_pytest_nodes", true)
	if err != nil {
		return nil, err
	}
	taskRegression, err := iwpStringList(verification["task_regression_pytest_targets"], "verification.task_regression_pytest_targets", true)
	if err != nil {
		return nil, err
	}
	fullCommand, err := iwpStringList(verification["full_command"], "verification.full_command", true)
	if err != nil {
		return nil, err
	}
	requiredTests, ok := verification["required_new_tests"].([]any)
	if !ok || len(requiredTests) == 0 {
		return nil, fmt.Errorf("verification.required_new_tests must be non-empty")
	}
	requiredNodes := []string{}
	missingNew := []string{}
	for _, raw := range requiredTests {
		item, err := iwpObject(raw, "required new test")
		if err != nil {
			return nil, err
		}
		node := Text(item["node"])
		if node == "" {
			return nil, fmt.Errorf("required new test node must be non-empty")
		}
		ids, err := iwpStringList(item["acceptance_ids"], "required test acceptance_ids", true)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			if !acceptanceIDs[id] {
				return nil, fmt.Errorf("required test references unknown acceptance id: %s", id)
			}
		}
		if !Has(allowedTests, strings.SplitN(node, "::", 2)[0]) {
			return nil, fmt.Errorf("required test is outside mutation.allowed_test_paths: %s", node)
		}
		if !declaredNodes[node] {
			return nil, fmt.Errorf("required test is not mapped by acceptance vectors: %s", node)
		}
		requiredNodes = append(requiredNodes, node)
		if !PythonTestNodeExists(r, node) {
			missingNew = append(missingNew, node)
		}
	}
	for _, node := range baseline {
		if !PythonTestNodeExists(r, node) {
			return nil, fmt.Errorf("baseline verification node is missing: %s", node)
		}
	}
	for _, target := range taskRegression {
		if !iwpPathExists(r, target) {
			return nil, fmt.Errorf("task regression target does not exist: %s", target)
		}
	}
	corePolicy, err := iwpObject(verification["core_regression"], "verification.core_regression")
	if err != nil {
		return nil, err
	}
	component, componentSource := Text(corePolicy["component_ref"]), Text(corePolicy["source"])
	if component == "" || componentSource == "" {
		return nil, fmt.Errorf("core regression must name component and source")
	}
	coreTargets, err := iwpComponentTargets(r, component, componentSource)
	if err != nil {
		return nil, err
	}
	combined := iwpMergeTargets(r, append(append([]string{}, taskRegression...), coreTargets...))
	policyPaths := []string{}
	for _, raw := range List(resolved["policies"]) {
		ref := Text(Map(raw)["path"])
		if ref != "" {
			policyPaths = append(policyPaths, ref)
		}
	}
	planningEntry, normativeSource := Text(taskContext["planning_entry"]), Text(taskContext["normative_rule_source"])
	if planningEntry == "" || normativeSource == "" {
		return nil, fmt.Errorf("task context exact planning and normative sources are required")
	}
	contextFiles := iwpSortedUnique(append([]string{iwpRegistryRef, iwpSchemaRef, componentSource, planningEntry, normativeSource}, policyPaths...))
	tracked := append(append(append(append(append([]string{}, contextFiles...), readPaths...), codePaths...), testRefs...), allowedTests...)
	fileHashes := Object{}
	for _, ref := range iwpSortedUnique(tracked) {
		file, err := r.Path(ref)
		if err != nil {
			return nil, err
		}
		if info, err := os.Stat(file); err == nil && !info.IsDir() {
			hash, err := iwpFileHash(r, ref)
			if err != nil {
				return nil, err
			}
			fileHashes[ref] = hash
		}
	}
	readFingerprints := []any{}
	for _, raw := range readContext {
		item := Map(raw)
		hash, err := iwpSelectorFingerprint(r, item)
		if err != nil {
			return nil, err
		}
		readFingerprints = append(readFingerprints, Object{"path": item["path"], "selector": item["selector"], "fingerprint": hash})
	}
	head, err := iwpGit(r, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	fingerprintPayload := Object{"head": head, "branch": branch, "scope": resolved["scope"], "operation": operation, "policy_context": resolved["policies"], "normative_rule_refs": taskContext["normative_rule_refs"], "mutation": mutation, "acceptance_vectors": acceptance, "verification": verification, "core_regression_targets": coreTargets, "file_hashes": fileHashes}
	bytes, err := iwpCanonicalJSON(fingerprintPayload)
	if err != nil {
		return nil, err
	}
	fingerprint := SHA256(bytes)
	pytest := func(targets []string) []string {
		return append(append([]string{"python", "-m", "pytest"}, targets...), "-vv")
	}
	commands := Object{"baseline": pytest(baseline), "focused": pytest(append(append([]string{}, baseline...), requiredNodes...)), "task-regression": pytest(taskRegression), "core": pytest(coreTargets), "regression": pytest(combined), "full": fullCommand}
	failure, err := iwpObject(registry["failure_routing"], "failure_routing")
	if err != nil {
		return nil, err
	}
	if mutation["scope_expansion"] != "RE_RESOLVE_REQUIRED" {
		return nil, fmt.Errorf("mutation.scope_expansion must be RE_RESOLVE_REQUIRED")
	}
	status := "READY"
	if len(missingNew) > 0 {
		status = "REQUIRES_NEW_TESTS"
	}
	return Object{"schema_version": "ptsip-implementation-work-packet/v2", "projection_authority": false, "packet_id": "iwp-" + fingerprint[:16], "task": Object{"branch": branch, "head": head, "scope": resolved["scope"], "operation": operation, "planning_entry": planningEntry}, "policy_context": Object{"policies": resolved["policies"], "normative_rules": taskContext["normative_rules"], "constraints": taskContext["constraints"]}, "read_context": readContext, "mutation_plan": Object{"targets": editTargets, "allowed_test_paths": iwpSortedUnique(allowedTests), "scope_expansion": mutation["scope_expansion"]}, "acceptance_vectors": acceptance, "acceptance_coverage": acceptanceCoverage, "verification": Object{"baseline_pytest_nodes": baseline, "required_new_tests": requiredTests, "required_new_pytest_nodes": requiredNodes, "missing_required_new_tests": missingNew, "task_regression_pytest_targets": taskRegression, "core_regression": Object{"component_ref": component, "source": componentSource, "selection": corePolicy["selection"], "pytest_targets": coreTargets}, "combined_regression_pytest_targets": combined, "commands": commands, "status": status}, "edit_budget": Object{"allowed_code_paths": iwpSortedUnique(codePaths), "allowed_test_paths": iwpSortedUnique(allowedTests), "unlisted_paths": "BLOCK", "selector_removal_or_rename": "BLOCK"}, "test_mode": iwpTestMode(r, component), "failure_routing": failure, "freshness": Object{"baseline_head": head, "context_files": contextFiles, "read_context_fingerprints": readFingerprints, "file_hashes": fileHashes, "context_fingerprint": fingerprint, "strategy": "FILE_AND_SELECTOR_RECHECK_BEFORE_EVERY_VERIFICATION"}}, nil
}
