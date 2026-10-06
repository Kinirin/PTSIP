package machine

import (
	"fmt"
	domainpolicy "github.com/Kinirin/PTSIP/developer/automation/policy"
	policybinding "github.com/Kinirin/PTSIP/developer/automation/policy/binding"
	policylifecycle "github.com/Kinirin/PTSIP/developer/automation/policy/lifecycle"
	"strings"
)

const DeveloperClass = domainpolicy.DeveloperClass
const ResolverContract = policybinding.ResolverContract

var (
	rootID                     = domainpolicy.RootID
	policyVersionPattern       = policylifecycle.PolicyVersionPattern
	policyReadableFamilyID     = policylifecycle.PolicyReadableFamilyID
	policyBoundaryID           = policylifecycle.PolicyBoundaryID
	policyAnalysisID           = policylifecycle.AnalysisIDPattern
	policyRootFamilies         = policylifecycle.RootFamilies
	policyLegacyFamilies       = policylifecycle.LegacyFamilies
	policyCollisionResolutions = policylifecycle.CollisionResolutions
)

const (
	policyIndex                  = policylifecycle.PolicyIndex
	policySubjectRegistry        = policylifecycle.PolicySubjectRegistry
	policyApprovalSchema         = policylifecycle.PolicyApprovalSchema
	policyAnalysisRoot           = policylifecycle.PolicyAnalysisRoot
	policyAnalysisRecordRoot     = policylifecycle.PolicyAnalysisRecordRoot
	policyAnalysisRegistry       = policylifecycle.PolicyAnalysisRegistry
	policyAnalysisSchema         = policylifecycle.PolicyAnalysisSchema
	policyAnalysisRegistrySchema = policylifecycle.PolicyAnalysisRegistrySchema
)

type Resolver = policybinding.Resolver
type PolicyError = policylifecycle.PolicyError
type PolicyCorpus = policylifecycle.PolicyCorpus

func NewResolver(repo *Repository) (*Resolver, error) { return policybinding.NewResolver(repo) }

func (r *Repository) RootSection(id, section string) (any, error) {
	resolver, err := NewResolver(r)
	if err != nil {
		return nil, err
	}
	result, err := resolver.Get(id, section)
	if err != nil {
		return nil, err
	}
	return result["record"], nil
}

func (r *Repository) CurrentBranch() (string, error) {
	output, err := ppGit(r.Root, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "", fmt.Errorf("detached HEAD: task context branch resolution fails closed")
	}
	return branch, nil
}

func (r *Repository) ValidateImplementationReference(reference Object) (Object, error) {
	return ValidateImplementationRef(r, reference)
}

func policyFailure(code, detail string) error { return policylifecycle.Failure(code, detail) }
func policyContains(values []string, needle string) bool {
	return policylifecycle.Contains(values, needle)
}
func policyStrings(value any) []string          { return policylifecycle.Strings(value) }
func policyUnique(values []string) bool         { return policylifecycle.Unique(values) }
func policySetsEqual(a, b map[string]bool) bool { return policylifecycle.SetsEqual(a, b) }
func policyClone[T any](value T) T              { return policylifecycle.Clone(value) }
func policyParseVersion(version string) (int, int, error) {
	return policylifecycle.ParseVersion(version)
}
func policyCanonicalPath(id string) (string, error) { return policylifecycle.CanonicalPath(id) }
func policyNextFamilyID(ids []string, family string) (string, error) {
	return policylifecycle.NextFamilyID(ids, family)
}

func InitialPolicyVersion() string { return policylifecycle.InitialPolicyVersion() }
func ResolvePolicyVersionTransition(version, status, changeClass, target string) (Object, error) {
	return policylifecycle.ResolvePolicyVersionTransition(version, status, changeClass, target)
}
func (r *Repository) LoadConsistentPolicyCorpus() (*PolicyCorpus, error) {
	return policylifecycle.LoadConsistentPolicyCorpus(r)
}
func (r *Repository) policyDiscoveredIDs() ([]string, error) { return policylifecycle.DiscoveredIDs(r) }
func (r *Repository) policyCheckDiscovery(state *PolicyCorpus, extra string) error {
	return policylifecycle.CheckDiscovery(r, state, extra)
}
func (r *Repository) policyApproval(reference string) (Object, error) {
	return policylifecycle.Approval(r, reference)
}
func (r *Repository) policyRequireFamilyClass(class, family string) error {
	return policylifecycle.RequireFamilyClass(r, class, family)
}
func (r *Repository) policyAnalysisGroup(class, family, reference, groupID string) (Object, Object, error) {
	return policylifecycle.AnalysisGroup(r, class, family, reference, groupID)
}
func (r *Repository) PreflightFamilyPolicy(class, family, approvalRef, analysisID, groupID string) (Object, error) {
	return policylifecycle.PreflightFamilyPolicy(r, class, family, approvalRef, analysisID, groupID)
}
func (r *Repository) InspectPolicy(id string) (Object, error) {
	return policylifecycle.InspectPolicy(r, id)
}
func (r *Repository) StatusPreflight(id, approvalRef string) (Object, error) {
	return policylifecycle.StatusPreflight(r, id, approvalRef)
}
func (r *Repository) PreflightNewPolicy(approvalRef, analysisID, groupID string) (Object, error) {
	return policylifecycle.PreflightNewPolicy(r, approvalRef, analysisID, groupID)
}
func (r *Repository) RegisterPolicy(approvalRef, analysisID, groupID, policyFile string) (Object, error) {
	return policylifecycle.RegisterPolicy(r, approvalRef, analysisID, groupID, policyFile)
}
func (r *Repository) policyWriteTransaction(updates map[string]Object) error {
	return policylifecycle.WriteTransaction(r, updates)
}
func (r *Repository) RegisterFamilyPolicy(class, family, approvalRef, analysisID, groupID, policyFile string) (Object, error) {
	return policylifecycle.RegisterFamilyPolicy(r, class, family, approvalRef, analysisID, groupID, policyFile)
}
func (r *Repository) policyClasses() ([]string, error) { return policylifecycle.Classes(r) }
func (r *Repository) ActiveFamilyIDs(class, family string) ([]string, error) {
	return policylifecycle.ActiveFamilyIDs(r, class, family)
}
func (r *Repository) ValidateAnalysisSemantics(payload Object) []string {
	return policylifecycle.ValidateAnalysisSemantics(r, payload)
}
func (r *Repository) analysisRecords() ([]Object, error) { return policylifecycle.AnalysisRecords(r) }
func (r *Repository) ResolveAnalysisRecord(analysisID, subjectType, subjectID, analysisKind string) (Object, error) {
	return policylifecycle.ResolveAnalysisRecord(r, analysisID, subjectType, subjectID, analysisKind)
}
func (r *Repository) ValidateResponsibilityAnalysis(analysisID string, currentLookup bool) (Object, error) {
	return policylifecycle.ValidateResponsibilityAnalysis(r, analysisID, currentLookup)
}
func (r *Repository) FamilyEntry(policyClass, family string) (Object, error) {
	return policylifecycle.FamilyEntry(r, policyClass, family)
}
func (r *Repository) InspectFamilyID(id string) (Object, error) {
	return policylifecycle.InspectFamilyID(r, id)
}
