package machine

import (
	"encoding/json"
	"fmt"
	"go.yaml.in/yaml/v3"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

const PlanningRootIndex = "developer/planning/index.yaml"

func planningClone(value Object) Object {
	raw, _ := json.Marshal(value)
	var out Object
	json.Unmarshal(raw, &out)
	return out
}
func planningStatus(value Object) string { return Text(Map(value["lifecycle"])["status"]) }
func planningNonterminal(status string) bool {
	return status == "DRAFT" || status == "ACTIVE" || status == "BLOCKED"
}
func planningAlias(plan Object, branch string) bool {
	migration := Map(plan["branch_identity_migration"])
	return migration["status"] == "RENAME_PENDING" && migration["legacy_branch"] == branch && migration["canonical_branch"] == plan["integration_branch"]
}
func planningFindPlan(index Object, branch string) (Object, error) {
	matches := []Object{}
	for _, raw := range List(index["plans"]) {
		plan := Map(raw)
		if plan["integration_branch"] == branch || planningAlias(plan, branch) {
			matches = append(matches, plan)
		}
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("INTEGRATION_BRANCH_UNREGISTERED_OR_AMBIGUOUS: %s", branch)
	}
	return matches[0], nil
}
func planningIndexed(index Object) (map[string]Object, error) {
	out := map[string]Object{}
	for _, raw := range List(index["work_units"]) {
		row := Map(raw)
		id := Text(row["id"])
		if id == "" || out[id] != nil {
			return nil, fmt.Errorf("WORK_UNIT_INDEX_INVALID_OR_DUPLICATE")
		}
		out[id] = row
	}
	return out, nil
}
func (r *Repository) planningDocuments(index Object) (map[string]Object, error) {
	rows, err := planningIndexed(index)
	if err != nil {
		return nil, err
	}
	out := map[string]Object{}
	for id, row := range rows {
		ref := Text(row["path"])
		if ref == "" {
			continue
		}
		doc, err := r.Read(ref)
		if err != nil {
			return nil, err
		}
		if Map(doc["work_unit"])["id"] != id {
			return nil, fmt.Errorf("WORK_UNIT_ID_MISMATCH: %s", ref)
		}
		out[id] = doc
	}
	return out, nil
}
func planningCopyState(index Object, docs map[string]Object) error {
	for _, raw := range List(index["work_units"]) {
		row := Map(raw)
		doc := docs[Text(row["id"])]
		if doc == nil {
			continue
		}
		wu := Map(doc["work_unit"])
		lifecycle, approval, authorization := Map(wu["lifecycle"]), Map(wu["approval"]), Map(wu["implementation_authorization"])
		if Text(lifecycle["status"]) == "" || Text(approval["status"]) == "" || List(approval["inherited_from"]) == nil || Text(authorization["status"]) == "" || List(wu["depends_on"]) == nil {
			return fmt.Errorf("INVALID_WORK_UNIT_STATE: %s", Text(row["id"]))
		}
		if (approval["status"] == "APPROVED" || approval["status"] == "REJECTED") && approval["approval_source"] != "USER_EXPLICIT" {
			return fmt.Errorf("INVALID_WORK_UNIT_APPROVAL_SOURCE")
		}
		if authorization["status"] == "AUTHORIZED" && authorization["authorization_source"] != "USER_EXPLICIT" {
			return fmt.Errorf("INVALID_WORK_UNIT_AUTHORIZATION_SOURCE")
		}
		row["lifecycle"] = Object{"status": lifecycle["status"]}
		copiedApproval := Object{"status": approval["status"], "inherited_from": approval["inherited_from"]}
		if approval["approval_source"] != nil {
			copiedApproval["approval_source"] = approval["approval_source"]
		}
		row["approval"] = copiedApproval
		copiedAuthorization := Object{"status": authorization["status"]}
		if authorization["authorization_source"] != nil {
			copiedAuthorization["authorization_source"] = authorization["authorization_source"]
		}
		row["implementation_authorization"] = copiedAuthorization
		row["depends_on"] = wu["depends_on"]
		if lifecycle["status"] == "COMPLETE" {
			if len(List(doc["completion_evidence"])) == 0 {
				return fmt.Errorf("MISSING_COMPLETION_EVIDENCE: %s", Text(row["id"]))
			}
			row["completion_evidence"] = doc["completion_evidence"]
		} else {
			delete(row, "completion_evidence")
		}
	}
	return nil
}
func (r *Repository) ResolvePlanningGate(gate string, index Object, docs map[string]Object) (string, string, error) {
	rows, err := planningIndexed(index)
	if err != nil {
		return "", "", err
	}
	if !strings.Contains(gate, "-P") {
		row := rows[gate]
		if row == nil || Text(row["path"]) == "" {
			return "", "", fmt.Errorf("UNKNOWN_CURRENT_GATE: %s", gate)
		}
		return planningStatus(row), Text(row["path"]), nil
	}
	parent := strings.SplitN(gate, "-P", 2)[0]
	doc := docs[parent]
	if doc == nil {
		return "", "", fmt.Errorf("CURRENT_GATE_PARENT_MISSING")
	}
	matches := []Object{}
	for _, raw := range List(doc["extensions"]) {
		ext := Map(raw)
		if ext["id"] == gate {
			matches = append(matches, ext)
		}
	}
	if len(matches) != 1 {
		return "", "", fmt.Errorf("CURRENT_GATE_EXTENSION_UNRESOLVED")
	}
	ref := Text(matches[0]["path"])
	payload, err := r.Read(ref)
	if err != nil {
		return "", "", err
	}
	status := planningStatus(Map(payload["extension"]))
	if status == "" {
		return "", "", fmt.Errorf("CURRENT_GATE_EXTENSION_STATUS_INVALID")
	}
	return status, ref, nil
}
func (r *Repository) SelectPlanningGate(index, rootPlan Object, docs map[string]Object, mergedWU string) (string, string, error) {
	gate := Text(Map(index["plan"])["current_gate"])
	if gate == "" {
		return "", "", fmt.Errorf("INVALID_CURRENT_GATE")
	}
	status, path, err := r.ResolvePlanningGate(gate, index, docs)
	if err != nil {
		return "", "", err
	}
	if planningNonterminal(status) {
		return gate, path, nil
	}
	rows, _ := planningIndexed(index)
	candidate := func(id string) (string, string, bool) {
		row := rows[id]
		return id, Text(row["path"]), row != nil && planningNonterminal(planningStatus(row)) && Text(row["path"]) != ""
	}
	if strings.Contains(gate, "-P") {
		if id, path, ok := candidate(strings.SplitN(gate, "-P", 2)[0]); ok {
			return id, path, nil
		}
	}
	if mergedWU != "" {
		if rows[mergedWU] == nil {
			return "", "", fmt.Errorf("MERGED_WORK_UNIT_NOT_INDEXED")
		}
		if id, path, ok := candidate(mergedWU); ok {
			return id, path, nil
		}
	}
	routing := Map(rootPlan["entry_routing"])
	entrypoints := map[string]Object{}
	for _, raw := range List(routing["branch_entrypoints"]) {
		row := Map(raw)
		entrypoints[Text(row["branch"])] = row
	}
	for _, state := range []string{"MERGED", "ACTIVE"} {
		for _, raw := range List(routing["independent_leaf_work_units"]) {
			route := Map(raw)
			entry := entrypoints[Text(route["branch"])]
			if entry["state"] == state {
				if id, path, ok := candidate(Text(route["id"])); ok {
					return id, path, nil
				}
			}
		}
	}
	ordered := []string{}
	if convergence := Text(Map(routing["dependency_bearing_convergence"])["id"]); convergence != "" {
		ordered = append(ordered, convergence)
	}
	for _, raw := range List(index["work_units"]) {
		id := Text(Map(raw)["id"])
		if len(ordered) == 0 || ordered[0] != id {
			ordered = append(ordered, id)
		}
	}
	for _, id := range ordered {
		if _, path, ok := candidate(id); ok {
			satisfied := true
			for _, dep := range List(rows[id]["depends_on"]) {
				satisfied = satisfied && planningStatus(rows[Text(dep)]) == "COMPLETE"
			}
			if satisfied {
				return id, path, nil
			}
		}
	}
	return "", "", fmt.Errorf("NO_CURRENT_GATE_CANDIDATE")
}
func planningLeaf(rootPlan Object, branch string) (Object, error) {
	matches := []Object{}
	for _, raw := range List(Map(rootPlan["entry_routing"])["branch_entrypoints"]) {
		row := Map(raw)
		if row["branch"] == branch && row["role"] == "INDEPENDENT_LEAF" {
			matches = append(matches, row)
		}
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("MERGED_LEAF_BRANCH_UNRESOLVED")
	}
	return matches[0], nil
}
func (r *Repository) BuildPlanningMaterializedState(rootPlan, index Object, docs map[string]Object) (Object, error) {
	gate := Text(Map(index["plan"])["current_gate"])
	_, document, err := r.ResolvePlanningGate(gate, index, docs)
	if err != nil {
		return nil, err
	}
	routing := Map(rootPlan["entry_routing"])
	leaves := map[string]Object{}
	entrypoints := map[string]Object{}
	for _, raw := range List(routing["independent_leaf_work_units"]) {
		row := Map(raw)
		leaves[Text(row["id"])] = row
	}
	for _, raw := range List(routing["branch_entrypoints"]) {
		row := Map(raw)
		entrypoints[Text(row["branch"])] = row
	}
	records := []any{}
	for _, raw := range List(index["work_units"]) {
		row := Map(raw)
		id := Text(row["id"])
		lifecycle := planningStatus(row)
		auth := Text(Map(row["implementation_authorization"])["status"])
		if lifecycle == "" || auth == "" {
			return nil, fmt.Errorf("INVALID_INDEXED_WORK_UNIT_STATE")
		}
		record := Object{"id": id, "lifecycle_status": lifecycle, "implementation_authorization_status": auth}
		if lifecycle == "NOT_CREATED" {
			record["execution_location"] = "NOT_CREATED"
		} else if leaf := leaves[id]; leaf != nil {
			branch := Text(leaf["branch"])
			entry := entrypoints[branch]
			if branch == "" || entry == nil {
				return nil, fmt.Errorf("LEAF_ROUTING_INCOMPLETE")
			}
			if entry["state"] == "MERGED" {
				record["execution_location"] = "INTEGRATION_BRANCH"
				record["branch"] = rootPlan["integration_branch"]
			} else {
				record["execution_location"] = "LEAF_BRANCH"
				record["branch"] = branch
			}
		} else {
			record["execution_location"] = "INTEGRATION_BRANCH"
			record["branch"] = rootPlan["integration_branch"]
		}
		if row["path"] != nil {
			record["entry_document"] = row["path"]
		}
		records = append(records, record)
	}
	return Object{"mode": "DERIVED_CACHE", "source": rootPlan["path"], "generated_by": "developer.automation.planning.planning_merge_reconciler", "current_gate": gate, "current_gate_document": document, "work_units": records}, nil
}
func (r *Repository) ReconcilePlanning(currentBranch, mergedBranch string, apply bool) (Object, error) {
	if currentBranch == "" {
		raw, err := ppGit(r.Root, "branch", "--show-current")
		if err != nil {
			return nil, err
		}
		currentBranch = strings.TrimSpace(string(raw))
		if currentBranch == "" {
			return nil, fmt.Errorf("DETACHED_HEAD")
		}
	}
	rootSnapshotPath, err := r.Path(PlanningRootIndex)
	if err != nil {
		return nil, err
	}
	rootSnapshot, err := os.ReadFile(rootSnapshotPath)
	if err != nil {
		return nil, err
	}
	rootIndex, err := r.Read(PlanningRootIndex)
	if err != nil {
		return nil, err
	}
	rootBefore := planningClone(rootIndex)
	rootPlan, err := planningFindPlan(rootIndex, currentBranch)
	if err != nil {
		return nil, err
	}
	contract := Map(Map(rootPlan["entry_routing"])["merge_reconciliation"])
	if contract["target_branch"] != currentBranch && !(planningAlias(rootPlan, currentBranch) && contract["target_branch"] == rootPlan["integration_branch"]) {
		return nil, fmt.Errorf("RECONCILIATION_WRONG_BRANCH")
	}
	ref := Text(rootPlan["path"])
	versionSnapshotPath, err := r.Path(ref)
	if err != nil {
		return nil, err
	}
	versionSnapshot, err := os.ReadFile(versionSnapshotPath)
	if err != nil {
		return nil, err
	}
	index, err := r.Read(ref)
	if err != nil {
		return nil, err
	}
	before := planningClone(index)
	docs, err := r.planningDocuments(index)
	if err != nil {
		return nil, err
	}
	if err = planningCopyState(index, docs); err != nil {
		return nil, err
	}
	mergedWU := ""
	if mergedBranch != "" {
		entry, err := planningLeaf(rootPlan, mergedBranch)
		if err != nil {
			return nil, err
		}
		if entry["state"] != "ACTIVE" && entry["state"] != "MERGED" {
			return nil, fmt.Errorf("INVALID_LEAF_ENTRYPOINT_STATE")
		}
		mergedWU = Text(entry["work_unit"])
		if mergedWU == "" {
			return nil, fmt.Errorf("LEAF_ENTRYPOINT_WORK_UNIT_MISSING")
		}
		entry["state"] = "MERGED"
		entry["merged_into"] = rootPlan["integration_branch"]
	}
	gate, path, err := r.SelectPlanningGate(index, rootPlan, docs, mergedWU)
	if err != nil {
		return nil, err
	}
	Map(index["plan"])["current_gate"] = gate
	materialized, err := r.BuildPlanningMaterializedState(rootPlan, index, docs)
	if err != nil {
		return nil, err
	}
	rootPlan["materialized_state"] = materialized
	changed := !reflect.DeepEqual(rootBefore, rootIndex) || !reflect.DeepEqual(before, index)
	revision := Text(Map(before["plan"])["revision"])
	next := revision
	if changed {
		n, err := strconv.Atoi(revision)
		if err != nil || len(revision) != 2 || n < 0 || n >= 99 {
			return nil, fmt.Errorf("INVALID_OR_EXHAUSTED_PLAN_REVISION")
		}
		next = fmt.Sprintf("%02d", n+1)
		Map(index["plan"])["revision"] = next
	}
	if changed && apply {
		if err = r.planningWriteTransaction(map[string]Object{PlanningRootIndex: rootIndex, ref: index}, func() []string { return r.ValidatePlanning() }, map[string]string{PlanningRootIndex: SHA256(rootSnapshot), ref: SHA256(versionSnapshot)}); err != nil {
			return nil, err
		}
	}
	status := "PREVIEW"
	if apply {
		status = "RECONCILED"
	}
	return Object{"status": status, "integration_branch": rootPlan["integration_branch"], "merged_branch": ppOptional(mergedBranch), "merged_work_unit": ppOptional(mergedWU), "current_gate_before": Map(before["plan"])["current_gate"], "current_gate_after": gate, "current_gate_document": path, "version_index_revision_before": revision, "version_index_revision_after": next, "changed": changed}, nil
}
func (r *Repository) planningWriteTransaction(documents map[string]Object, validate func() []string, expectedSnapshots ...map[string]string) error {
	lockPath, err := r.Path("developer/planning/.mutation.lock")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		return err
	}
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("PLANNING_MUTATION_LOCKED: %w", err)
	}
	lock.Close()
	defer os.Remove(lockPath)
	before := map[string][]byte{}
	refs := []string{}
	for ref := range documents {
		refs = append(refs, ref)
		path, err := r.Path(ref)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		before[ref] = raw
		if len(expectedSnapshots) > 0 && expectedSnapshots[0][ref] != "" && SHA256(raw) != expectedSnapshots[0][ref] {
			return fmt.Errorf("PLANNING_SNAPSHOT_STALE: %s", ref)
		}
	}
	sort.Strings(refs)
	written := map[string]string{}
	restore := func() {
		for ref, digest := range written {
			r.AtomicWrite(ref, before[ref], &digest)
		}
	}
	for _, ref := range refs {
		raw, err := yaml.Marshal(documents[ref])
		if err != nil {
			restore()
			return err
		}
		digest := SHA256(before[ref])
		if err = r.AtomicWrite(ref, raw, &digest); err != nil {
			restore()
			return err
		}
		written[ref] = SHA256(raw)
	}
	if validate != nil {
		if failures := validate(); len(failures) > 0 {
			restore()
			return fmt.Errorf("POST_PLANNING_MUTATION_VALIDATION_FAILED: %s", strings.Join(failures, "; "))
		}
	}
	return nil
}
