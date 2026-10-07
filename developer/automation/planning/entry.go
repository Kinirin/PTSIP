package planning

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func ResolvePlanningEntry(r Repository, branch string) (Object, error) {
	if branch == "" {
		command := exec.Command("git", "-C", r.RootDir(), "branch", "--show-current")
		output, err := command.Output()
		if err != nil {
			return nil, fmt.Errorf("branch detection failed: %w", err)
		}
		branch = strings.TrimSpace(string(output))
		if branch == "" {
			return nil, fmt.Errorf("detached HEAD: planning resolution fails closed")
		}
	}
	overlayPath := "developer/planning/0.3.8a1/emergency-implementation-overlay.yaml"
	if candidate, err := r.Path(overlayPath); err == nil {
		if _, err := os.Stat(candidate); err == nil {
			overlay, err := r.Read(overlayPath)
			if err != nil {
				return nil, err
			}
			if Map(overlay["branch"])["name"] == branch {
				if Text(overlay["plan_version"]) == "" {
					return nil, fmt.Errorf("invalid emergency plan version")
				}
				return Object{"status": "RESOLVED", "branch": branch, "plan_version": overlay["plan_version"], "entry_document": overlayPath, "role": "EMERGENCY_RELEASE_OVERLAY", "work_unit": nil}, nil
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	index, err := r.Read("developer/planning/index.yaml")
	if err != nil {
		return nil, err
	}
	matches := []Object{}
	for _, rawPlan := range List(index["plans"]) {
		plan := Map(rawPlan)
		for _, rawEntry := range List(Map(plan["entry_routing"])["branch_entrypoints"]) {
			entry := Map(rawEntry)
			state, exists := entry["state"]
			if entry["branch"] != branch || (exists && state != "ACTIVE") {
				continue
			}
			if _, ok := plan["plan_version"].(string); !ok {
				return nil, fmt.Errorf("invalid planning version")
			}
			reference := Text(entry["entry_document"])
			if reference == "" || Text(entry["role"]) == "" {
				return nil, fmt.Errorf("invalid planning entry reference or role")
			}
			if unit := entry["work_unit"]; unit != nil {
				if _, ok := unit.(string); !ok {
					return nil, fmt.Errorf("invalid work unit")
				}
			}
			candidate, err := r.Path(reference)
			if err != nil {
				return nil, err
			}
			info, err := os.Stat(candidate)
			if err != nil || info.IsDir() {
				return nil, fmt.Errorf("unresolved planning entry: %s", reference)
			}
			matches = append(matches, Object{"status": "RESOLVED", "branch": branch, "plan_version": plan["plan_version"], "entry_document": reference, "role": entry["role"], "work_unit": entry["work_unit"]})
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no active exact planning entry for branch %q", branch)
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("ambiguous planning entry for branch %q", branch)
	}
	return matches[0], nil
}
