package lifecycle

const (
	PolicyIndex = policyIndex
	PolicySubjectRegistry = policySubjectRegistry
	PolicyApprovalSchema = policyApprovalSchema
	PolicyAnalysisRoot = policyAnalysisRoot
	PolicyAnalysisRecordRoot = policyAnalysisRecordRoot
	PolicyAnalysisRegistry = policyAnalysisRegistry
	PolicyAnalysisSchema = policyAnalysisSchema
	PolicyAnalysisRegistrySchema = policyAnalysisRegistrySchema
)

var (
	PolicyVersionPattern = policyVersionPattern
	PolicyReadableFamilyID = policyReadableFamilyID
	PolicyBoundaryID = policyBoundaryID
	AnalysisIDPattern = policyAnalysisID
	RootFamilies = policyRootFamilies
	LegacyFamilies = policyLegacyFamilies
	CollisionResolutions = policyCollisionResolutions
)

func Failure(code, detail string) error { return policyFailure(code, detail) }
func Contains(values []string, needle string) bool { return policyContains(values, needle) }
func Strings(value any) []string { return policyStrings(value) }
func Unique(values []string) bool { return policyUnique(values) }
func SetsEqual(a, b map[string]bool) bool { return policySetsEqual(a, b) }
func Clone[T any](value T) T { return policyClone(value) }
func ParseVersion(version string) (int, int, error) { return policyParseVersion(version) }
func CanonicalPath(id string) (string, error) { return policyCanonicalPath(id) }
func NextFamilyID(ids []string, family string) (string, error) { return policyNextFamilyID(ids, family) }

func DiscoveredIDs(r Repository) ([]string, error) { return policyDiscoveredIDs(r) }
func CheckDiscovery(r Repository, state *PolicyCorpus, extra string) error { return policyCheckDiscovery(r, state, extra) }
func Approval(r Repository, reference string) (Object, error) { return policyApproval(r, reference) }
func RequireFamilyClass(r Repository, class, family string) error { return policyRequireFamilyClass(r, class, family) }
func AnalysisGroup(r Repository, class, family, reference, groupID string) (Object, Object, error) {
	return policyAnalysisGroup(r, class, family, reference, groupID)
}
func WriteTransaction(r Repository, updates map[string]Object) error { return policyWriteTransaction(r, updates) }
func Classes(r Repository) ([]string, error) { return policyClasses(r) }
func AnalysisRecords(r Repository) ([]Object, error) { return analysisRecords(r) }
