package machine

import (
	"fmt"
	planning "github.com/Kinirin/PTSIP/developer/automation/planning"
	"os"
)

func (r *Repository) State(domain string) (Object, error) {
	index, err := r.Read("developer/state/index.yaml")
	if err != nil {
		return nil, err
	}
	if err := r.Validate("developer/state/repository-state-index.schema.json", index); err != nil {
		return nil, err
	}
	owner := Map(Map(index["domains"])[domain])
	if owner == nil {
		return nil, fmt.Errorf("unknown state domain: %s", domain)
	}
	reference := Text(owner["ref"])
	candidate, err := r.Path(reference)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(candidate)
	if err != nil || info.IsDir() {
		return nil, fmt.Errorf("unresolved state owner: %s", reference)
	}
	return Object{"schema_version": "ptsip-repository-state-resolution/v1", "domain": domain, "ref": reference, "role": owner["role"], "authority": false}, nil
}

func (r *Repository) PlanningEntry(branch string) (Object, error) {
	return planning.ResolvePlanningEntry(r, branch)
}
