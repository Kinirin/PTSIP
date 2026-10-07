package planning

import (
	"fmt"
	"strings"
)

type PlanIdentityRepository interface {
	Read(string) (Object, error)
	Scope(string) (string, error)
}

// PlanIdentity resolves the identity owned by a concrete Planning document.
func PlanIdentity(repo PlanIdentityRepository, ref string) (Object, error) {
	scope, err := repo.Scope(ref)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(scope, "developer/planning/") {
		return nil, fmt.Errorf("PLAN_REF_OUTSIDE_PLANNING_NAMESPACE")
	}
	payload, err := repo.Read(scope)
	if err != nil {
		return nil, err
	}
	identity := Map(payload["plan_identity"])
	for _, key := range []string{"resolved_plan_id", "plan_file_id", "version", "revision"} {
		if Text(identity[key]) == "" {
			return nil, fmt.Errorf("PLAN_IDENTITY_INCOMPLETE: %s", key)
		}
	}
	return identity, nil
}
