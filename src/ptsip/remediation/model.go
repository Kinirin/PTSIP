// Package remediation models read-only semantic remediation and evidence-bound
// reconciliation. A validated model never grants execution authority.
package remediation

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/evidence"
)

type ContractError struct {
	Code   string
	Detail string
}

func (e *ContractError) Error() string { return e.Code + ": " + e.Detail }

func contractError(code, detail string) error { return &ContractError{code, detail} }

func validText(value string) bool { return value != "" && value == strings.TrimSpace(value) }

func requireText(scope string, fields ...string) error {
	for i := 0; i < len(fields); i += 2 {
		if !validText(fields[i+1]) {
			return contractError(scope+"_TEXT_INVALID", fields[i]+" must be non-empty canonical text")
		}
	}
	return nil
}

func optionalText(scope, field string, value *string) error {
	if value == nil {
		return nil
	}
	return requireText(scope, field, *value)
}

func requireIDs(scope, name string, ids []string, required bool) error {
	if required && len(ids) == 0 {
		return contractError(scope+"_IDS_EMPTY", name+" must contain at least one identifier")
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if err := requireText(scope, name, id); err != nil {
			return err
		}
		if seen[id] {
			return contractError(scope+"_IDS_DUPLICATE", name+" must contain unique identifiers")
		}
		seen[id] = true
	}
	return nil
}

func array[T any](value []T) []T { return append([]T{}, value...) }

func sortedIDs(ids []string) []string {
	out := array(ids)
	slices.Sort(out)
	return out
}

func canonicalValue(value json.RawMessage, invalidCode, encodingCode string) (json.RawMessage, error) {
	if len(value) == 0 || strings.TrimSpace(string(value)) == "null" {
		return nil, contractError(invalidCode, "the value must be explicit")
	}
	document, err := evidence.CanonicalJSON(value)
	if err != nil {
		return nil, contractError(encodingCode, err.Error())
	}
	return json.RawMessage(document), nil
}

func stableID(prefix string, payload any) (string, error) {
	digest, err := evidence.StableDigest(payload)
	if err != nil {
		return "", err
	}
	return prefix + ":" + digest[:24], nil
}

type ResolutionOutcome string

const (
	Deterministic        ResolutionOutcome = "DETERMINISTIC"
	OwnerIntentRequired  ResolutionOutcome = "OWNER_INTENT_REQUIRED"
	ExternalFactRequired ResolutionOutcome = "EXTERNAL_FACT_REQUIRED"
	Unsatisfiable        ResolutionOutcome = "UNSATISFIABLE"
	ToolCapabilityGap    ResolutionOutcome = "TOOL_CAPABILITY_GAP"
)

type MutationClass string

const (
	MechanicalReversible         MutationClass = "MECHANICAL_REVERSIBLE"
	StructuralSemanticPreserving MutationClass = "STRUCTURAL_SEMANTIC_PRESERVING"
	ArchitectureSemantic         MutationClass = "ARCHITECTURE_SEMANTIC"
	Destructive                  MutationClass = "DESTRUCTIVE"
)

type AuthorizationDecision string

const (
	Authorized                AuthorizationDecision = "AUTHORIZED"
	OwnerConfirmationRequired AuthorizationDecision = "OWNER_CONFIRMATION_REQUIRED"
	NotAuthorized             AuthorizationDecision = "NOT_AUTHORIZED"
)

type RemediationDisposition string

const (
	ExistingComponentCandidate  RemediationDisposition = "EXISTING_COMPONENT_CANDIDATE"
	AssociatedArtifactCandidate RemediationDisposition = "ASSOCIATED_ARTIFACT_CANDIDATE"
	RelocationCandidate         RemediationDisposition = "RELOCATION_CANDIDATE"
	RemovalCandidate            RemediationDisposition = "REMOVAL_CANDIDATE"
	NewComponentCandidate       RemediationDisposition = "NEW_COMPONENT_CANDIDATE"
	DecisionRequired            RemediationDisposition = "DECISION_REQUIRED"
)

type ProjectIntentSource string

const (
	ProjectProfile   ProjectIntentSource = "PROJECT_PROFILE"
	OwnerDecision    ProjectIntentSource = "OWNER_DECISION"
	GovernanceRecord ProjectIntentSource = "GOVERNANCE_RECORD"
)

var (
	specificationFamily = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[A-Za-z0-9][A-Za-z0-9.-]*)?$`)
	immutableRevision   = regexp.MustCompile(`^[0-9a-f]{40}$`)
	sha256Digest        = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// SpecificationBinding identifies the explicit specification read by a solve.
// It deliberately provides no default catalog or assertion of authority.
type SpecificationBinding struct {
	Family   string `json:"family"`
	Source   string `json:"source"`
	Revision string `json:"revision"`
}

func (b SpecificationBinding) Validate() error {
	if !specificationFamily.MatchString(b.Family) {
		return contractError("SPEC_BINDING_FAMILY_MALFORMED", "family must be a canonical version-family identity")
	}
	parsed, err := url.Parse(b.Source)
	if !validText(b.Source) || err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return contractError("SPEC_BINDING_SOURCE_MALFORMED", "source must be an absolute URI")
	}
	if !immutableRevision.MatchString(b.Revision) {
		return contractError("SPEC_BINDING_REVISION_MALFORMED", "revision must be an immutable lowercase Git SHA")
	}
	return nil
}

type Context struct {
	Evidence      evidence.NormalizedSet
	Specification SpecificationBinding
}

func (c Context) Validate() error {
	if err := c.Specification.Validate(); err != nil {
		return err
	}
	if c.Evidence.Format != "ptsip-normalized-evidence/v1" || c.Evidence.RecordCount != len(c.Evidence.Records) {
		return contractError("REMEDIATION_EVIDENCE_TYPE", "context requires normalized evidence v1")
	}
	conflicts := 0
	for _, record := range c.Evidence.Records {
		if record.Status == "CONFLICT" {
			conflicts++
		}
	}
	if c.Evidence.ConflictCount != conflicts {
		return contractError("REMEDIATION_EVIDENCE_TYPE", "evidence conflict count must match its records")
	}
	digest, err := evidence.StableDigest(c.Evidence.Content)
	if err != nil || digest != c.Evidence.DeterministicDigest {
		return contractError("REMEDIATION_EVIDENCE_DIGEST_INVALID", "evidence content must match its deterministic digest")
	}
	return nil
}

func (c Context) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{"evidence_digest": c.Evidence.DeterministicDigest, "specification": c.Specification})
}

type CoverageGap struct {
	ID          string   `json:"id"`
	Code        string   `json:"code"`
	Subject     string   `json:"subject"`
	EvidenceIDs []string `json:"evidence_ids"`
	Detail      *string  `json:"detail"`
}

func (v CoverageGap) Validate() error {
	if err := requireText("REMEDIATION", "id", v.ID, "code", v.Code, "subject", v.Subject); err != nil {
		return err
	}
	if err := optionalText("REMEDIATION", "detail", v.Detail); err != nil {
		return err
	}
	return requireIDs("REMEDIATION", "evidence_ids", v.EvidenceIDs, false)
}

func (v CoverageGap) MarshalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	type gap CoverageGap
	v.EvidenceIDs = array(v.EvidenceIDs)
	return json.Marshal(gap(v))
}

type DerivedFact struct {
	ID          string          `json:"id"`
	Subject     string          `json:"subject"`
	Predicate   string          `json:"predicate"`
	Value       json.RawMessage `json:"value"`
	EvidenceIDs []string        `json:"evidence_ids"`
}

func (v DerivedFact) Validate() error {
	if err := requireText("REMEDIATION", "id", v.ID, "subject", v.Subject, "predicate", v.Predicate); err != nil {
		return err
	}
	if _, err := canonicalValue(v.Value, "REMEDIATION_FACT_VALUE_INVALID", "REMEDIATION_FACT_VALUE_INVALID"); err != nil {
		return err
	}
	return requireIDs("REMEDIATION", "evidence_ids", v.EvidenceIDs, true)
}

type NormativeConstraint struct {
	RuleID        string               `json:"rule_id"`
	Specification SpecificationBinding `json:"specification"`
	Subject       string               `json:"subject"`
	Requirement   string               `json:"requirement"`
}

func (v NormativeConstraint) Validate() error {
	if err := requireText("REMEDIATION", "rule_id", v.RuleID, "subject", v.Subject, "requirement", v.Requirement); err != nil {
		return err
	}
	return v.Specification.Validate()
}

type ProjectIntentAuthority struct {
	ID        string              `json:"id"`
	Subject   string              `json:"subject"`
	Intent    string              `json:"intent"`
	Source    ProjectIntentSource `json:"source"`
	SourceRef string              `json:"source_ref"`
}

func (v ProjectIntentAuthority) Validate() error {
	switch v.Source {
	case ProjectProfile, OwnerDecision, GovernanceRecord:
	default:
		return contractError("REMEDIATION_INTENT_SOURCE_INVALID", "project intent requires an explicit project-owned source")
	}
	return requireText("REMEDIATION", "id", v.ID, "subject", v.Subject, "intent", v.Intent, "source_ref", v.SourceRef)
}

type DispositionCandidate struct {
	ID          string                 `json:"id"`
	GapID       string                 `json:"gap_id"`
	Disposition RemediationDisposition `json:"disposition"`
	Rationale   string                 `json:"rationale"`
	EvidenceIDs []string               `json:"evidence_ids"`
}

func (v DispositionCandidate) Validate() error {
	if err := requireText("REMEDIATION", "id", v.ID, "gap_id", v.GapID, "rationale", v.Rationale); err != nil {
		return err
	}
	switch v.Disposition {
	case ExistingComponentCandidate, AssociatedArtifactCandidate, RelocationCandidate, RemovalCandidate, NewComponentCandidate, DecisionRequired:
	default:
		return contractError("REMEDIATION_DISPOSITION_INVALID", "unknown disposition")
	}
	return requireIDs("REMEDIATION", "evidence_ids", v.EvidenceIDs, false)
}

func (v DispositionCandidate) MarshalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	type disposition DispositionCandidate
	v.EvidenceIDs = array(v.EvidenceIDs)
	return json.Marshal(disposition(v))
}

type SemanticRemediationPlan struct {
	ID                string   `json:"id"`
	TargetCandidateID string   `json:"target_candidate_id"`
	RuleIDs           []string `json:"rule_ids"`
	FactIDs           []string `json:"fact_ids"`
	AuthorityIDs      []string `json:"authority_ids"`
	Rationale         *string  `json:"rationale"`
}

func (v SemanticRemediationPlan) Validate() error {
	if err := requireText("REMEDIATION", "id", v.ID, "target_candidate_id", v.TargetCandidateID); err != nil {
		return err
	}
	if err := optionalText("REMEDIATION", "rationale", v.Rationale); err != nil {
		return err
	}
	if err := requireIDs("REMEDIATION", "rule_ids", v.RuleIDs, true); err != nil {
		return err
	}
	if err := requireIDs("REMEDIATION", "fact_ids", v.FactIDs, false); err != nil {
		return err
	}
	return requireIDs("REMEDIATION", "authority_ids", v.AuthorityIDs, false)
}

func (v SemanticRemediationPlan) MarshalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	type plan SemanticRemediationPlan
	v.FactIDs, v.AuthorityIDs = array(v.FactIDs), array(v.AuthorityIDs)
	return json.Marshal(plan(v))
}

// RepositoryChangePlan is a description for an execution owner. This package
// cannot authorize, apply, or execute its operation references.
type RepositoryChangePlan struct {
	ID                  string        `json:"id"`
	SemanticPlanID      string        `json:"semantic_plan_id"`
	MutationClass       MutationClass `json:"mutation_class"`
	PrestateFingerprint string        `json:"prestate_fingerprint"`
	OperationIDs        []string      `json:"operation_ids"`
}

func (v RepositoryChangePlan) Validate() error {
	if err := requireText("REMEDIATION", "id", v.ID, "semantic_plan_id", v.SemanticPlanID, "prestate_fingerprint", v.PrestateFingerprint); err != nil {
		return err
	}
	switch v.MutationClass {
	case MechanicalReversible, StructuralSemanticPreserving, ArchitectureSemantic, Destructive:
	default:
		return contractError("REMEDIATION_MUTATION_CLASS_INVALID", "unknown mutation class")
	}
	return requireIDs("REMEDIATION", "operation_ids", v.OperationIDs, true)
}

type PostconditionResult struct {
	SemanticPlanID  string   `json:"semantic_plan_id"`
	Satisfied       bool     `json:"satisfied"`
	VerifiedRuleIDs []string `json:"verified_rule_ids"`
	FailedCheckIDs  []string `json:"failed_check_ids"`
}

func (v PostconditionResult) Validate() error {
	if err := requireText("REMEDIATION", "semantic_plan_id", v.SemanticPlanID); err != nil {
		return err
	}
	if v.Satisfied && len(v.FailedCheckIDs) != 0 {
		return contractError("REMEDIATION_POSTCONDITION_CONFLICT", "a satisfied postcondition cannot have failed checks")
	}
	if err := requireIDs("REMEDIATION", "verified_rule_ids", v.VerifiedRuleIDs, false); err != nil {
		return err
	}
	return requireIDs("REMEDIATION", "failed_check_ids", v.FailedCheckIDs, false)
}

func (v PostconditionResult) MarshalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	type postcondition PostconditionResult
	v.VerifiedRuleIDs, v.FailedCheckIDs = array(v.VerifiedRuleIDs), array(v.FailedCheckIDs)
	return json.Marshal(postcondition(v))
}

func failedContract(code, format string, args ...any) error {
	return contractError(code, fmt.Sprintf(format, args...))
}
