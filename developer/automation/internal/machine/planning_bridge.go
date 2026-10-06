package machine

import planning "github.com/Kinirin/PTSIP/developer/automation/planning"

// RootDir is the mechanical repository-root adapter consumed by domain packages.
func (r *Repository) RootDir() string { return r.Root }

// Planning Automation domain API.
const PlanningRootIndex = planning.PlanningRootIndex

func ExtensionMachineReady(payload Object) bool { return planning.ExtensionMachineReady(payload) }
func ExtensionParentConsistency(parent, extension Object, id, ref string) []string {
	return planning.ExtensionParentConsistency(parent, extension, id, ref)
}
func PlanningPromoteStageText(text, id string, automatic Object) (string, error) {
	return planning.PlanningPromoteStageText(text, id, automatic)
}
func (r *Repository) FinalizeExtension(ref string) (Object, error) {
	return planning.FinalizeExtension(r, ref)
}
func (r *Repository) RunPlanningRegression(targets []any, goTargets []any) []string {
	return planning.RunPlanningRegression(r, targets, goTargets)
}
func (r *Repository) FinalizePlanningStage(ref, id string) (Object, error) {
	return planning.FinalizePlanningStage(r, ref, id)
}
func (r *Repository) MergePlanningLeaf(branch, message string) (Object, error) {
	return planning.MergePlanningLeaf(r, branch, message)
}
func (r *Repository) ResolvePlanningGate(gate string, index Object, docs map[string]Object) (string, string, error) {
	return planning.ResolvePlanningGate(r, gate, index, docs)
}
func (r *Repository) SelectPlanningGate(index, rootPlan Object, docs map[string]Object, mergedWU string) (string, string, error) {
	return planning.SelectPlanningGate(r, index, rootPlan, docs, mergedWU)
}
func (r *Repository) BuildPlanningMaterializedState(rootPlan, index Object, docs map[string]Object) (Object, error) {
	return planning.BuildPlanningMaterializedState(r, rootPlan, index, docs)
}
func (r *Repository) ReconcilePlanning(currentBranch, mergedBranch string, apply bool) (Object, error) {
	return planning.ReconcilePlanning(r, currentBranch, mergedBranch, apply)
}
func (r *Repository) ValidatePlanning() []string { return planning.ValidatePlanning(r) }

// Temporary symbol bridges keep the mixed responsibility regression file
// compiling without moving that mixed file into the Planning domain.
func planningClone(value Object) Object { return planning.Clone(value) }
func planningStatus(value Object) string { return planning.Status(value) }
func planningNonterminal(status string) bool { return planning.Nonterminal(status) }
func planningAlias(plan Object, branch string) bool { return planning.Alias(plan, branch) }
func planningFindPlan(index Object, branch string) (Object, error) { return planning.FindPlan(index, branch) }
func planningIndexed(index Object) (map[string]Object, error) { return planning.Indexed(index) }
func (r *Repository) planningDocuments(index Object) (map[string]Object, error) { return planning.Documents(r, index) }
func planningCopyState(index Object, docs map[string]Object) error { return planning.CopyState(index, docs) }
func planningLeaf(rootPlan Object, branch string) (Object, error) { return planning.Leaf(rootPlan, branch) }
func (r *Repository) planningWriteTransaction(documents map[string]Object, validate func() []string, expectedSnapshots ...map[string]string) error {
	return planning.WriteTransaction(r, documents, validate, expectedSnapshots...)
}
func planningEqual(a, b any) bool { return planning.Equal(a, b) }
func planningFindStage(payload any, id string) (Object, error) { return planning.FindStage(payload, id) }
func planningPromotePayload(payload Object, id string, automatic Object) error {
	return planning.PromotePayload(payload, id, automatic)
}
func planningSectionSpan(text, section string) (int, int, error) { return planning.SectionSpan(text, section) }
func planningStageSpan(text, id string) (int, int, string, error) { return planning.StageSpan(text, id) }
func planningReplaceStageStatus(text, id, old, next string) (string, error) {
	return planning.ReplaceStageStatus(text, id, old, next)
}
func planningReplaceExtensionStatuses(text string, payload Object) (string, error) {
	return planning.ReplaceExtensionStatuses(text, payload)
}
func planningGovernanceErrors(payload any, registry Object, label string) []string {
	return planning.GovernanceErrors(payload, registry, label)
}
func planningLeafTuples(routing Object) []string { return planning.LeafTuples(routing) }
func (r *Repository) validatePlanningFormalIdentity(identity Object) error {
	return planning.ValidateFormalIdentity(r, identity)
}
func planningRoutingErrors(entry, plan Object, indexed map[string]Object) []string {
	return planning.RoutingErrors(entry, plan, indexed)
}
func planningPrereleaseDependencies(plan Object, docs map[string]Object) []string {
	return planning.PrereleaseDependencies(plan, docs)
}
func extensionStatusRecords(payload Object) []Object { return planning.ExtensionStatusRecords(payload) }
func extensionValidationComplete(record Object) bool { return planning.ExtensionValidationComplete(record) }
func (r *Repository) extensionContext(payload Object, ref string) (string, string, string, error) {
	return planning.ExtensionContext(r, payload, ref)
}
func (r *Repository) planningRegisteredCheck(check string) []string {
	return planning.RegisteredCheck(r, check)
}
