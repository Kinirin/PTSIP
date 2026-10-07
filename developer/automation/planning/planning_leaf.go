package planning

import (
	"fmt"
	"strings"
)

func MergePlanningLeaf(r Repository, branch, message string) (Object, error) {
	activeRaw, err := ppGit(r.RootDir(), "branch", "--show-current")
	if err != nil {
		return nil, err
	}
	active := strings.TrimSpace(string(activeRaw))
	if active == "" {
		return nil, fmt.Errorf("DETACHED_HEAD")
	}
	index, err := r.Read(PlanningRootIndex)
	if err != nil {
		return nil, err
	}
	plan, err := planningFindPlan(index, active)
	if err != nil {
		return nil, err
	}
	contract := Map(Map(plan["entry_routing"])["merge_reconciliation"])
	if contract["target_branch"] != active && !(planningAlias(plan, active) && contract["target_branch"] == plan["integration_branch"]) {
		return nil, fmt.Errorf("WRONG_INTEGRATION_BRANCH")
	}
	if contract["leaf_shared_index_mutation"] != "FORBIDDEN" {
		return nil, fmt.Errorf("UNSAFE_SHARED_INDEX_POLICY")
	}
	entry, err := planningLeaf(plan, branch)
	if err != nil {
		return nil, err
	}
	if entry["state"] != "ACTIVE" {
		return nil, fmt.Errorf("LEAF_ENTRYPOINT_NOT_ACTIVE")
	}
	wu := Text(entry["work_unit"])
	if wu == "" {
		return nil, fmt.Errorf("LEAF_WORK_UNIT_MISSING")
	}
	status, err := ppGit(r.RootDir(), "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(status)) != "" {
		return nil, fmt.Errorf("DIRTY_INTEGRATION_WORKTREE")
	}
	source := ""
	if _, err = ppGit(r.RootDir(), "show-ref", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		source = branch
	} else if _, err = ppGit(r.RootDir(), "show-ref", "--verify", "--quiet", "refs/remotes/origin/"+branch); err == nil {
		source = "origin/" + branch
	}
	if source == "" {
		return nil, fmt.Errorf("LEAF_BRANCH_NOT_FOUND")
	}
	base, err := ppGit(r.RootDir(), "merge-base", "HEAD", source)
	if err != nil {
		return nil, err
	}
	diff, err := ppGit(r.RootDir(), "diff", "--name-only", strings.TrimSpace(string(base))+".."+source)
	if err != nil {
		return nil, err
	}
	for _, path := range strings.Split(strings.TrimSpace(string(diff)), "\n") {
		if path == PlanningRootIndex || path == plan["path"] {
			return nil, fmt.Errorf("LEAF_SHARED_INDEX_MUTATION_FORBIDDEN: %s", path)
		}
	}
	abort := func() {
		if _, err := ppGit(r.RootDir(), "rev-parse", "-q", "--verify", "MERGE_HEAD"); err == nil {
			ppGit(r.RootDir(), "merge", "--abort")
		}
	}
	if _, err = ppGit(r.RootDir(), "merge", "--no-ff", "--no-commit", source); err != nil {
		abort()
		return nil, err
	}
	if _, err = ppGit(r.RootDir(), "rev-parse", "-q", "--verify", "MERGE_HEAD"); err != nil {
		return nil, fmt.Errorf("LEAF_ALREADY_INTEGRATED")
	}
	reconciliation, err := ReconcilePlanning(r, active, branch, true)
	if err != nil {
		abort()
		return nil, err
	}
	if _, err = ppGit(r.RootDir(), "add", "--", PlanningRootIndex, Text(plan["path"])); err != nil {
		abort()
		return nil, err
	}
	if message == "" {
		message = "merge: integrate " + branch + " with planning reconciliation"
	}
	if _, err = ppGit(r.RootDir(), "commit", "-m", message); err != nil {
		abort()
		return nil, err
	}
	head, err := ppGit(r.RootDir(), "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	return Object{"status": "MERGED", "source_branch": branch, "source_ref": source, "integration_branch": plan["integration_branch"], "merged_work_unit": wu, "merge_commit": strings.TrimSpace(string(head)), "current_gate": reconciliation["current_gate_after"]}, nil
}
