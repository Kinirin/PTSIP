package planning

// Compatibility exports expose the already-owned Planning semantics to the
// command/control plane while their implementation remains in this domain.

func Clone(value Object) Object { return planningClone(value) }
func Status(value Object) string { return planningStatus(value) }
func Nonterminal(status string) bool { return planningNonterminal(status) }
func Alias(plan Object, branch string) bool { return planningAlias(plan, branch) }
func FindPlan(index Object, branch string) (Object, error) { return planningFindPlan(index, branch) }
func Indexed(index Object) (map[string]Object, error) { return planningIndexed(index) }
func Documents(r Repository, index Object) (map[string]Object, error) { return planningDocuments(r, index) }
func CopyState(index Object, docs map[string]Object) error { return planningCopyState(index, docs) }
func Leaf(rootPlan Object, branch string) (Object, error) { return planningLeaf(rootPlan, branch) }
func WriteTransaction(r Repository, documents map[string]Object, validate func() []string, expectedSnapshots ...map[string]string) error {
	return planningWriteTransaction(r, documents, validate, expectedSnapshots...)
}
func Equal(a, b any) bool { return planningEqual(a, b) }
func FindStage(payload any, id string) (Object, error) { return planningFindStage(payload, id) }
func PromotePayload(payload Object, id string, automatic Object) error { return planningPromotePayload(payload, id, automatic) }
func SectionSpan(text, section string) (int, int, error) { return planningSectionSpan(text, section) }
func StageSpan(text, id string) (int, int, string, error) { return planningStageSpan(text, id) }
func ReplaceStageStatus(text, id, old, next string) (string, error) { return planningReplaceStageStatus(text, id, old, next) }
func ReplaceExtensionStatuses(text string, payload Object) (string, error) { return planningReplaceExtensionStatuses(text, payload) }
func GovernanceErrors(payload any, registry Object, label string) []string { return planningGovernanceErrors(payload, registry, label) }
func LeafTuples(routing Object) []string { return planningLeafTuples(routing) }
func RoutingErrors(entry, plan Object, indexed map[string]Object) []string { return planningRoutingErrors(entry, plan, indexed) }
func PrereleaseDependencies(plan Object, docs map[string]Object) []string { return planningPrereleaseDependencies(plan, docs) }
func ValidateFormalIdentity(r Repository, identity Object) error { return validatePlanningFormalIdentity(r, identity) }
func ExtensionStatusRecords(payload Object) []Object { return extensionStatusRecords(payload) }
func ExtensionValidationComplete(record Object) bool { return extensionValidationComplete(record) }
func ExtensionContext(r Repository, payload Object, ref string) (string, string, string, error) { return extensionContext(r, payload, ref) }
func RegisteredCheck(r Repository, check string) []string { return planningRegisteredCheck(r, check) }
