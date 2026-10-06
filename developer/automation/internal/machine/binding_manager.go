package machine

import policybinding "github.com/Kinirin/PTSIP/developer/automation/policy/binding"

func (r *Repository) PlanIdentity(ref string) (Object, error) {
	return policybinding.NewStore(r).PlanIdentity(ref)
}
func (r *Repository) CreateBinding(policy string) (Object, error) {
	return policybinding.NewStore(r).CreateBinding(policy)
}
func (r *Repository) LinkPlan(values Object) (Object, error) {
	return policybinding.NewStore(r).LinkPlan(values)
}
func (r *Repository) MovePlanRef(values Object) (Object, error) {
	return policybinding.NewStore(r).MovePlanRef(values)
}
func (r *Repository) TrackPlanRef(id string, apply bool) (Object, error) {
	return policybinding.NewStore(r).TrackPlanRef(id, apply)
}
