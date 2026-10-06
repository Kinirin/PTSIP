package machine

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var iwpHunk = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

func iwpChangedPaths(r *Repository) ([]string, error) {
	tracked, err := iwpGit(r, "diff", "--name-only", "HEAD")
	if err != nil {
		return nil, err
	}
	untracked, err := iwpGit(r, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	all := []string{}
	for _, line := range strings.Split(tracked+"\n"+untracked, "\n") {
		if line != "" {
			all = append(all, NormalizeReference(line))
		}
	}
	return iwpSortedUnique(all), nil
}
func iwpHunkAllowed(start, count int, ranges [][2]int) bool {
	for _, span := range ranges {
		if count == 0 && span[0]-1 <= start && start <= span[1] {
			return true
		}
		if count > 0 && span[0] <= start && start+count-1 <= span[1] {
			return true
		}
	}
	return false
}
func iwpSelectorIntegrity(r *Repository, targets any) []any {
	list, ok := targets.([]any)
	if !ok {
		return []any{Object{"reason": "EDIT_TARGETS_INVALID"}}
	}
	violations := []any{}
	for _, raw := range list {
		item := Map(raw)
		if item == nil {
			violations = append(violations, Object{"reason": "EDIT_TARGET_INVALID"})
			continue
		}
		if Text(item["path"]) == "" || Map(item["selector"]) == nil {
			violations = append(violations, Object{"reason": "EDIT_TARGET_INVALID", "target": item})
			continue
		}
		current, err := ValidateImplementationRef(r, item)
		if err != nil {
			violations = append(violations, Object{"path": item["path"], "selector": item["selector"], "reason": "SELECTOR_NO_LONGER_RESOLVES", "detail": err.Error()})
			continue
		}
		if Map(current["resolved_location"]) == nil {
			violations = append(violations, Object{"path": item["path"], "selector": item["selector"], "reason": "SELECTOR_LOCATION_MISSING"})
		}
	}
	return violations
}

func CheckWorkPacket(r *Repository, packet Object) (Object, error) {
	task, err := iwpObject(packet["task"], "packet.task")
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
	mutation, err := iwpObject(packet["mutation_plan"], "packet.mutation_plan")
	if err != nil {
		return nil, err
	}
	branch, err := iwpGit(r, "branch", "--show-current")
	if err != nil {
		return nil, err
	}
	head, err := iwpGit(r, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	changed, err := iwpChangedPaths(r)
	if err != nil {
		return nil, err
	}
	codePaths, testPaths := Strings(budget["allowed_code_paths"]), Strings(budget["allowed_test_paths"])
	allowed := append(append([]string{}, codePaths...), testPaths...)
	unexpected := []string{}
	for _, ref := range changed {
		if !Has(allowed, ref) {
			unexpected = append(unexpected, ref)
		}
	}
	baseline, err := iwpObject(freshness["file_hashes"], "file_hashes")
	if err != nil {
		return nil, err
	}
	contextChanged := []string{}
	for _, ref := range Strings(freshness["context_files"]) {
		hash, err := iwpFileHash(r, ref)
		if err != nil || baseline[ref] != hash {
			contextChanged = append(contextChanged, ref)
		}
	}
	readChanged := []any{}
	for _, raw := range List(freshness["read_context_fingerprints"]) {
		ref := Map(raw)
		if ref == nil {
			readChanged = append(readChanged, Object{"reason": "READ_CONTEXT_FINGERPRINT_INVALID"})
			continue
		}
		hash, err := iwpSelectorFingerprint(r, ref)
		if err != nil {
			readChanged = append(readChanged, Object{"path": ref["path"], "selector": ref["selector"], "reason": "READ_CONTEXT_SELECTOR_NO_LONGER_RESOLVES", "detail": err.Error()})
		} else if Text(ref["fingerprint"]) == "" || hash != ref["fingerprint"] {
			readChanged = append(readChanged, Object{"path": ref["path"], "selector": ref["selector"], "reason": "READ_CONTEXT_SELECTOR_CHANGED"})
		}
	}
	ranges := map[string][][2]int{}
	for _, raw := range List(mutation["targets"]) {
		item := Map(raw)
		ref := Text(item["path"])
		location := Map(item["resolved_location"])
		start, end := agentInt(location["line_start"]), agentInt(location["line_end"])
		if ref != "" && start >= 1 && end >= start {
			ranges[ref] = append(ranges[ref], [2]int{start, end})
		}
	}
	codeViolations := []any{}
	for _, ref := range iwpSortedUnique(codePaths) {
		if !Has(changed, ref) {
			continue
		}
		diff, err := iwpGit(r, "diff", "--unified=0", "HEAD", "--", ref)
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(diff, "\n") {
			match := iwpHunk.FindStringSubmatch(line)
			if match == nil {
				continue
			}
			start, _ := strconv.Atoi(match[1])
			count := 1
			if match[2] != "" {
				count, _ = strconv.Atoi(match[2])
			}
			if !iwpHunkAllowed(start, count, ranges[ref]) {
				codeViolations = append(codeViolations, Object{"path": ref, "old_start": start, "old_count": count})
			}
		}
	}
	selectorViolations := iwpSelectorIntegrity(r, mutation["targets"])
	verification, err := iwpObject(packet["verification"], "packet.verification")
	if err != nil {
		return nil, err
	}
	missing := []string{}
	for _, node := range Strings(verification["required_new_pytest_nodes"]) {
		if !PythonTestNodeExists(r, node) {
			missing = append(missing, node)
		}
	}
	problems := []string{}
	add := func(condition bool, name string) {
		if condition {
			problems = append(problems, name)
		}
	}
	add(branch != task["branch"], "BRANCH_CHANGED")
	add(head != freshness["baseline_head"], "HEAD_CHANGED")
	add(len(contextChanged) > 0, "CONTEXT_CHANGED")
	add(len(readChanged) > 0, "READ_CONTEXT_CHANGED")
	add(len(unexpected) > 0, "UNEXPECTED_CHANGED_PATH")
	add(len(codeViolations) > 0, "CODE_SCOPE_VIOLATION")
	add(len(selectorViolations) > 0, "MUTATION_SELECTOR_VIOLATION")
	reprepare := Has(problems, "BRANCH_CHANGED") || Has(problems, "HEAD_CHANGED") || Has(problems, "CONTEXT_CHANGED") || Has(problems, "READ_CONTEXT_CHANGED")
	hashes := Object{}
	for _, ref := range changed {
		if Has(allowed, ref) {
			hash, err := iwpFileHash(r, ref)
			if err != nil {
				hash = "MISSING"
			}
			hashes[ref] = hash
		}
	}
	bytes, err := iwpCanonicalJSON(Object{"head": head, "changed": changed, "files": hashes})
	if err != nil {
		return nil, err
	}
	status := "PASS"
	if len(problems) > 0 {
		status = "BLOCKED"
	}
	return Object{"schema_version": "ptsip-implementation-work-check/v2", "status": status, "problems": problems, "branch": Object{"expected": task["branch"], "actual": branch}, "head": Object{"expected": freshness["baseline_head"], "actual": head}, "changed_paths": changed, "unexpected_changed_paths": unexpected, "context_changed": contextChanged, "read_context_changed": readChanged, "code_scope_violations": codeViolations, "selector_violations": selectorViolations, "missing_required_new_tests": missing, "iteration_fingerprint": SHA256(bytes), "reprepare_required": reprepare, "safe_to_continue_iteration": !reprepare && len(unexpected) == 0 && len(codeViolations) == 0 && len(selectorViolations) == 0}, nil
}

var iwpDuration = regexp.MustCompile(`\b\d+(?:\.\d+)?s\b`)
var iwpAddress = regexp.MustCompile(`0x[0-9a-fA-F]+`)

func WorkPacketFailureSignature(stage string, returncode int, output string) string {
	lines := []string{}
	for _, line := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, strings.TrimRight(line, " \t\r"))
		}
	}
	if len(lines) > 80 {
		lines = lines[len(lines)-80:]
	}
	tail := strings.Join(lines, "\n")
	tail = iwpDuration.ReplaceAllString(tail, "<duration>")
	tail = iwpAddress.ReplaceAllString(tail, "0x<addr>")
	return agentHash(fmt.Sprintf("%s\n%d\n%s", stage, returncode, tail))[:24]
}
func iwpFailureState(r *Repository, ref string) (Object, error) {
	state, err := r.Read(ref)
	if err != nil {
		file, pathErr := r.Path(ref)
		if pathErr != nil {
			return nil, pathErr
		}
		if _, statErr := os.Stat(file); os.IsNotExist(statErr) {
			return Object{"schema_version": "ptsip-implementation-failure-state/v1", "entries": Object{}}, nil
		}
		return nil, err
	}
	if Map(state["entries"]) == nil {
		return nil, fmt.Errorf("failure state entries must be a mapping")
	}
	return state, nil
}
func RouteWorkPacketFailure(r *Repository, packetID, stage string, returncode int, output, stateRef string, policy Object) (Object, error) {
	signature := WorkPacketFailureSignature(stage, returncode, output)
	state, err := iwpFailureState(r, stateRef)
	if err != nil {
		return nil, err
	}
	entries := Map(state["entries"])
	key := packetID + ":" + stage + ":" + signature
	count := 1
	if previous := Map(entries[key]); previous != nil {
		count = agentInt(previous["count"]) + 1
	}
	entries[key] = Object{"count": count, "returncode": returncode}
	if err := r.WriteJSON(stateRef, state, nil); err != nil {
		return nil, err
	}
	collection := false
	for _, marker := range []string{"ERROR collecting", "not found:", "found no collectors", "no tests ran"} {
		collection = collection || strings.Contains(output, marker)
	}
	recheck, reresolve := agentInt(policy["recheck_context_at"]), agentInt(policy["reresolve_scope_at"])
	if recheck < 0 {
		recheck = 2
	}
	if reresolve < 0 {
		reresolve = 3
	}
	classification, action := "IMPLEMENTATION_FAILURE", "FIX_WITHIN_CURRENT_MUTATION_PLAN"
	if collection {
		classification = "TEST_CONTRACT_FAILURE"
		action = "REPAIR_TEST_SELECTION_OR_REQUIRED_TEST"
	} else if count >= reresolve {
		classification = "REPEATED_IMPLEMENTATION_FAILURE"
		action = "RE_RESOLVE_MUTATION_SCOPE"
	} else if count >= recheck {
		classification = "REPEATED_IMPLEMENTATION_FAILURE"
		action = "RECHECK_PACKET_ACCEPTANCE_AND_CONTEXT"
	}
	return Object{"schema_version": "ptsip-implementation-failure-route/v1", "packet_id": packetID, "stage": stage, "signature": signature, "repeat_count": count, "classification": classification, "next_action": action, "scope_expansion_allowed": false}, nil
}
func ClearWorkPacketFailure(r *Repository, stateRef, packetID, stage string) error {
	file, err := r.Path(stateRef)
	if err != nil {
		return err
	}
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil
	}
	state, err := iwpFailureState(r, stateRef)
	if err != nil {
		return err
	}
	entries := Map(state["entries"])
	prefix := packetID + ":" + stage + ":"
	for key := range entries {
		if strings.HasPrefix(key, prefix) {
			delete(entries, key)
		}
	}
	return r.WriteJSON(stateRef, state, nil)
}

func VerifyWorkPacket(r *Repository, packet Object, stage, logRef, stateRef string) (Object, error) {
	checked, err := CheckWorkPacket(r, packet)
	if err != nil {
		return nil, err
	}
	if checked["status"] != "PASS" {
		return checked, Fail("PACKET_BLOCKED", "work packet must be current before verification")
	}
	if !Has([]string{"baseline", "focused", "task-regression", "core", "regression", "full"}, stage) {
		return nil, fmt.Errorf("unsupported verification stage %s", stage)
	}
	if stage == "focused" && len(Strings(checked["missing_required_new_tests"])) > 0 {
		return nil, Fail("REQUIRED_TEST_MISSING", "focused verification requires new tests")
	}
	verification := Map(packet["verification"])
	command := Strings(Map(verification["commands"])[stage])
	if len(command) == 0 {
		return nil, fmt.Errorf("verification command is invalid")
	}
	for i, item := range command {
		if strings.Contains(item, "developer.automation") || strings.Contains(item, "developer/automation/") && strings.HasSuffix(item, ".py") {
			return nil, fmt.Errorf("Python automation delegation is forbidden")
		}
		if i == 0 && item == "python" {
			candidate, err := r.Path(".venv/Scripts/python.exe")
			if err == nil {
				if _, err := os.Stat(candidate); err == nil {
					command[0] = candidate
				}
			}
		}
	}
	execute := exec.Command(command[0], command[1:]...)
	execute.Dir = r.Root
	output, runErr := execute.CombinedOutput()
	returncode := 0
	if runErr != nil {
		if exit, ok := runErr.(*exec.ExitError); ok {
			returncode = exit.ExitCode()
		} else {
			return nil, runErr
		}
	}
	if logRef == "" {
		logRef = ".git/ptsip-iwp-" + stage + ".log"
	}
	if stateRef == "" {
		stateRef = ".git/ptsip-iwp-failure-state.json"
	}
	if err := r.AtomicWrite(logRef, output, nil); err != nil {
		return nil, err
	}
	packetID := Text(packet["packet_id"])
	if returncode == 0 {
		if err := ClearWorkPacketFailure(r, stateRef, packetID, stage); err != nil {
			return nil, err
		}
		return Object{"status": "PASS", "stage": stage, "returncode": 0, "log": logRef, "output": string(output)}, nil
	}
	routing, err := RouteWorkPacketFailure(r, packetID, stage, returncode, string(output), stateRef, Map(packet["failure_routing"]))
	if err != nil {
		return nil, err
	}
	routing["log"] = logRef
	routing["output"] = string(output)
	return routing, Fail("VERIFICATION_FAILED", fmt.Sprintf("verification returned %d", returncode))
}
