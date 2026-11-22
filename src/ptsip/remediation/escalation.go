package remediation

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/Kinirin/PTSIP/src/ptsip/evidence"
)

type ResolutionInputKind string

const (
	ProjectOwnerIntent ResolutionInputKind = "PROJECT_OWNER_INTENT"
	ExternalFactInput  ResolutionInputKind = "EXTERNAL_FACT"
)

type InputMaterializationTarget string

const (
	ProjectAuthorityTarget InputMaterializationTarget = "PROJECT_AUTHORITY"
	EvidenceTarget         InputMaterializationTarget = "EVIDENCE"
)

type FreshSolveRestartPolicy string

const RestartRequired FreshSolveRestartPolicy = "REQUIRED"

type FreshSolveBinding struct {
	EvidenceDigest                      string               `json:"evidence_digest"`
	Specification                       SpecificationBinding `json:"specification"`
	RepositoryRevision                  *string              `json:"repository_revision"`
	RepositoryStatusFingerprint         string               `json:"repository_status_fingerprint"`
	RepositoryTrackedContentFingerprint string               `json:"repository_tracked_content_fingerprint"`
	AuthoritySnapshotID                 string               `json:"authority_snapshot_id"`
}

func NewFreshSolveBinding(context Context, authoritySnapshotID string) (FreshSolveBinding, error) {
	if err := context.Validate(); err != nil {
		return FreshSolveBinding{}, err
	}
	snapshot := context.Evidence.Context.Snapshot
	binding := FreshSolveBinding{
		EvidenceDigest: context.Evidence.DeterministicDigest, Specification: context.Specification,
		RepositoryRevision: snapshot.Revision, RepositoryStatusFingerprint: snapshot.StatusFingerprint,
		RepositoryTrackedContentFingerprint: snapshot.TrackedContentFingerprint, AuthoritySnapshotID: authoritySnapshotID,
	}
	if binding.RepositoryRevision != nil {
		revision := *binding.RepositoryRevision
		binding.RepositoryRevision = &revision
	}
	if err := binding.Validate(); err != nil {
		return FreshSolveBinding{}, err
	}
	return binding, nil
}

func (binding FreshSolveBinding) Validate() error {
	if err := binding.Specification.Validate(); err != nil {
		return err
	}
	if !sha256Digest.MatchString(binding.EvidenceDigest) {
		return contractError("ESCALATION_EVIDENCE_DIGEST_INVALID", "normalized evidence must have a lowercase SHA-256 digest")
	}
	if err := optionalText("ESCALATION", "repository_revision", binding.RepositoryRevision); err != nil {
		return err
	}
	return requireText("ESCALATION", "evidence_digest", binding.EvidenceDigest,
		"repository_status_fingerprint", binding.RepositoryStatusFingerprint,
		"repository_tracked_content_fingerprint", binding.RepositoryTrackedContentFingerprint,
		"authority_snapshot_id", binding.AuthoritySnapshotID)
}

func (binding FreshSolveBinding) IdentityDigest() (string, error) {
	if err := binding.Validate(); err != nil {
		return "", err
	}
	return evidence.StableDigest(binding)
}

type FreshSolveBindingComparison struct {
	PreviousDigest    string   `json:"previous_digest"`
	CurrentDigest     string   `json:"current_digest"`
	ChangedDimensions []string `json:"changed_dimensions"`
	Changed           bool     `json:"changed"`
}

func CompareFreshSolveBindings(previous, current FreshSolveBinding) (FreshSolveBindingComparison, error) {
	previousDigest, err := previous.IdentityDigest()
	if err != nil {
		return FreshSolveBindingComparison{}, err
	}
	currentDigest, err := current.IdentityDigest()
	if err != nil {
		return FreshSolveBindingComparison{}, err
	}
	changed := []string{}
	for _, item := range []struct {
		name string
		same bool
	}{
		{"evidence_digest", previous.EvidenceDigest == current.EvidenceDigest},
		{"specification", previous.Specification == current.Specification},
		{"repository_revision", sameRevision(previous.RepositoryRevision, current.RepositoryRevision)},
		{"repository_status_fingerprint", previous.RepositoryStatusFingerprint == current.RepositoryStatusFingerprint},
		{"repository_tracked_content_fingerprint", previous.RepositoryTrackedContentFingerprint == current.RepositoryTrackedContentFingerprint},
		{"authority_snapshot_id", previous.AuthoritySnapshotID == current.AuthoritySnapshotID},
	} {
		if !item.same {
			changed = append(changed, item.name)
		}
	}
	return FreshSolveBindingComparison{previousDigest, currentDigest, changed, len(changed) != 0}, nil
}

func sameRevision(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

type EscalationChoice struct {
	CandidateID       string          `json:"candidate_id"`
	RuleID            string          `json:"rule_id"`
	RemediationFamily string          `json:"remediation_family"`
	TargetState       json.RawMessage `json:"target_state"`
}

type EscalationProof struct {
	ID                     string   `json:"id"`
	UnresolvedDimension    string   `json:"unresolved_dimension"`
	SurvivingCandidateIDs  []string `json:"surviving_candidate_ids"`
	EliminatedCandidateIDs []string `json:"eliminated_candidate_ids"`
	FactIDs                []string `json:"fact_ids"`
	ConstraintRuleIDs      []string `json:"constraint_rule_ids"`
	AvailableAuthorityIDs  []string `json:"available_authority_ids"`
	RequiredInput          string   `json:"required_input"`
}

func (proof EscalationProof) Validate() error {
	if err := requireText("REMEDIATION", "id", proof.ID, "unresolved_dimension", proof.UnresolvedDimension, "required_input", proof.RequiredInput); err != nil {
		return err
	}
	for _, group := range []struct {
		name     string
		ids      []string
		required bool
	}{
		{"surviving_candidate_ids", proof.SurvivingCandidateIDs, true}, {"eliminated_candidate_ids", proof.EliminatedCandidateIDs, false},
		{"fact_ids", proof.FactIDs, false}, {"constraint_rule_ids", proof.ConstraintRuleIDs, false}, {"available_authority_ids", proof.AvailableAuthorityIDs, false},
	} {
		if err := requireIDs("REMEDIATION", group.name, group.ids, group.required); err != nil {
			return err
		}
	}
	return nil
}

type ExternalFactRequirement struct {
	ID                  string   `json:"id"`
	UnresolvedDimension string   `json:"unresolved_dimension"`
	RequiredFact        string   `json:"required_fact"`
	CandidateIDs        []string `json:"candidate_ids"`
	BasisFactIDs        []string `json:"basis_fact_ids"`
	ConstraintRuleIDs   []string `json:"constraint_rule_ids"`
}

func (r ExternalFactRequirement) Validate() error {
	if err := requireText("ESCALATION", "id", r.ID, "unresolved_dimension", r.UnresolvedDimension, "required_fact", r.RequiredFact); err != nil {
		return err
	}
	if err := requireIDs("ESCALATION", "candidate_ids", r.CandidateIDs, false); err != nil {
		return err
	}
	if err := requireIDs("ESCALATION", "basis_fact_ids", r.BasisFactIDs, false); err != nil {
		return err
	}
	return requireIDs("ESCALATION", "constraint_rule_ids", r.ConstraintRuleIDs, false)
}

func (r ExternalFactRequirement) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	r.CandidateIDs, r.BasisFactIDs, r.ConstraintRuleIDs = array(r.CandidateIDs), array(r.BasisFactIDs), array(r.ConstraintRuleIDs)
	type requirement ExternalFactRequirement
	return json.Marshal(struct {
		requirement
		Authoritative bool `json:"authoritative"`
	}{requirement(r), false})
}

type ResolutionInputRequest struct {
	ID                      string                   `json:"id"`
	Kind                    ResolutionInputKind      `json:"kind"`
	SolveBinding            FreshSolveBinding        `json:"solve_binding"`
	Question                string                   `json:"question"`
	Choices                 []EscalationChoice       `json:"choices"`
	OwnerEscalationProof    *EscalationProof         `json:"owner_escalation_proof"`
	ExternalFactRequirement *ExternalFactRequirement `json:"external_fact_requirement"`
}

func (r ResolutionInputRequest) Validate() error {
	if err := requireText("ESCALATION", "id", r.ID, "question", r.Question); err != nil {
		return err
	}
	if err := r.SolveBinding.Validate(); err != nil {
		return err
	}
	choiceIDs := make([]string, 0, len(r.Choices))
	for _, choice := range r.Choices {
		if err := requireText("ESCALATION", "candidate_id", choice.CandidateID, "rule_id", choice.RuleID, "remediation_family", choice.RemediationFamily); err != nil {
			return err
		}
		if _, err := canonicalValue(choice.TargetState, "ESCALATION_TARGET_STATE_REQUIRED", "ESCALATION_TARGET_STATE_REQUIRED"); err != nil {
			return err
		}
		choiceIDs = append(choiceIDs, choice.CandidateID)
	}
	if err := requireIDs("ESCALATION", "choice_candidate_ids", choiceIDs, false); err != nil {
		return err
	}
	switch r.Kind {
	case ProjectOwnerIntent:
		if r.OwnerEscalationProof == nil {
			return contractError("ESCALATION_OWNER_PROOF_REQUIRED", "owner request requires its escalation proof")
		}
		if r.ExternalFactRequirement != nil {
			return contractError("ESCALATION_INPUT_KIND_CONFLICT", "owner request cannot also ask for an external fact")
		}
		if len(choiceIDs) < 2 {
			return contractError("ESCALATION_OWNER_CARDINALITY_INVALID", "owner intent requires at least two legal semantic targets")
		}
		if err := r.OwnerEscalationProof.Validate(); err != nil {
			return err
		}
		if !slices.Equal(r.OwnerEscalationProof.SurvivingCandidateIDs, choiceIDs) {
			return contractError("ESCALATION_OWNER_PROOF_CHOICE_MISMATCH", "proof survivors must match the presented choices")
		}
	case ExternalFactInput:
		if r.ExternalFactRequirement == nil {
			return contractError("ESCALATION_EXTERNAL_FACT_REQUIREMENT_REQUIRED", "external fact request requires a requirement")
		}
		if r.OwnerEscalationProof != nil {
			return contractError("ESCALATION_INPUT_KIND_CONFLICT", "external fact request cannot also ask for owner intent")
		}
		if err := r.ExternalFactRequirement.Validate(); err != nil {
			return err
		}
		if !slices.Equal(r.ExternalFactRequirement.CandidateIDs, choiceIDs) {
			return contractError("ESCALATION_EXTERNAL_FACT_CHOICE_MISMATCH", "required fact candidates must match the presented choices")
		}
	default:
		return contractError("ESCALATION_INPUT_KIND_INVALID", "unknown resolution input kind")
	}
	return nil
}

func (r ResolutionInputRequest) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	r.Choices = array(r.Choices)
	type request ResolutionInputRequest
	return json.Marshal(struct {
		request
		Authoritative bool `json:"authoritative"`
	}{request(r), false})
}

func RequireCurrentResolutionInputRequest(request ResolutionInputRequest, current FreshSolveBinding) (FreshSolveBindingComparison, error) {
	if err := request.Validate(); err != nil {
		return FreshSolveBindingComparison{}, err
	}
	comparison, err := CompareFreshSolveBindings(request.SolveBinding, current)
	if err != nil {
		return FreshSolveBindingComparison{}, err
	}
	if comparison.Changed {
		return comparison, failedContract("ESCALATION_REQUEST_STALE", "fresh solve required after changes to %v", comparison.ChangedDimensions)
	}
	return comparison, nil
}

type ResolutionInput struct {
	Outcome                ResolutionOutcome
	SolveBinding           FreshSolveBinding
	UnresolvedDimension    string
	Candidates             []SemanticCandidate
	EliminatedCandidateIDs []string
	FactIDs                []string
	ConstraintRuleIDs      []string
	AvailableAuthorityIDs  []string
	RequiredInput          string
}

func BuildResolutionInputRequest(input ResolutionInput) (ResolutionInputRequest, error) {
	if input.Outcome != OwnerIntentRequired && input.Outcome != ExternalFactRequired {
		return ResolutionInputRequest{}, contractError("ESCALATION_OUTCOME_NOT_INPUT_ACTIONABLE", "only owner intent or missing external facts produce a resolution request")
	}
	bindingID, err := input.SolveBinding.IdentityDigest()
	if err != nil {
		return ResolutionInputRequest{}, err
	}
	if input.SolveBinding.RepositoryRevision != nil {
		revision := *input.SolveBinding.RepositoryRevision
		input.SolveBinding.RepositoryRevision = &revision
	}
	choices := make([]EscalationChoice, 0, len(input.Candidates))
	for _, candidate := range input.Candidates {
		candidate, err := normalizeCandidate(candidate, "CANDIDATE_ID_SCOPE_MISMATCH")
		if err != nil {
			return ResolutionInputRequest{}, err
		}
		choices = append(choices, EscalationChoice{candidate.ID, candidate.RuleID, candidate.RemediationFamily, candidate.TargetState})
	}
	slices.SortFunc(choices, func(a, b EscalationChoice) int { return cmp.Compare(a.CandidateID, b.CandidateID) })
	ids := make([]string, 0, len(choices))
	for _, choice := range choices {
		ids = append(ids, choice.CandidateID)
	}
	for index := 1; index < len(ids); index++ {
		if ids[index-1] == ids[index] {
			return ResolutionInputRequest{}, contractError("ESCALATION_CANDIDATE_IDS_DUPLICATE", "candidate IDs must be unique")
		}
	}
	request := ResolutionInputRequest{SolveBinding: input.SolveBinding, Choices: choices}
	if input.Outcome == OwnerIntentRequired {
		if len(ids) < 2 {
			return ResolutionInputRequest{}, contractError("ESCALATION_OWNER_CARDINALITY_INVALID", "owner intent requires at least two legal semantic targets")
		}
		proof := EscalationProof{
			UnresolvedDimension: input.UnresolvedDimension, SurvivingCandidateIDs: ids, EliminatedCandidateIDs: array(input.EliminatedCandidateIDs),
			FactIDs: array(input.FactIDs), ConstraintRuleIDs: array(input.ConstraintRuleIDs), AvailableAuthorityIDs: array(input.AvailableAuthorityIDs), RequiredInput: input.RequiredInput,
		}
		proof.ID, err = stableID("escalation-proof", map[string]any{
			"unresolved_dimension": proof.UnresolvedDimension, "surviving_candidate_ids": proof.SurvivingCandidateIDs,
			"eliminated_candidate_ids": proof.EliminatedCandidateIDs, "fact_ids": proof.FactIDs, "constraint_rule_ids": proof.ConstraintRuleIDs,
			"available_authority_ids": proof.AvailableAuthorityIDs, "required_input": proof.RequiredInput, "solve_binding": bindingID,
		})
		if err != nil {
			return ResolutionInputRequest{}, err
		}
		request.Kind = ProjectOwnerIntent
		request.Question = fmt.Sprintf("Select the project-authoritative target for %s.", input.UnresolvedDimension)
		request.OwnerEscalationProof = &proof
		request.ID, err = stableID("resolution-input", map[string]any{"kind": request.Kind, "solve_binding": bindingID, "proof_id": proof.ID})
	} else {
		fact := ExternalFactRequirement{UnresolvedDimension: input.UnresolvedDimension, RequiredFact: input.RequiredInput, CandidateIDs: ids,
			BasisFactIDs: array(input.FactIDs), ConstraintRuleIDs: array(input.ConstraintRuleIDs)}
		fact.ID, err = stableID("external-fact", map[string]any{
			"unresolved_dimension": fact.UnresolvedDimension, "required_fact": fact.RequiredFact, "candidate_ids": fact.CandidateIDs,
			"basis_fact_ids": fact.BasisFactIDs, "constraint_rule_ids": fact.ConstraintRuleIDs, "solve_binding": bindingID,
		})
		if err != nil {
			return ResolutionInputRequest{}, err
		}
		request.Kind = ExternalFactInput
		request.Question = fmt.Sprintf("Provide the external fact required for %s: %s.", input.UnresolvedDimension, input.RequiredInput)
		request.ExternalFactRequirement = &fact
		request.ID, err = stableID("resolution-input", map[string]any{"kind": request.Kind, "solve_binding": bindingID, "requirement_id": fact.ID})
	}
	if err != nil {
		return ResolutionInputRequest{}, err
	}
	if err := request.Validate(); err != nil {
		return ResolutionInputRequest{}, err
	}
	return request, nil
}

type FreshSolveHandoff struct {
	RequestID             string                     `json:"request_id"`
	InputKind             ResolutionInputKind        `json:"input_kind"`
	MaterializationTarget InputMaterializationTarget `json:"materialization_target"`
	PriorSolveBinding     FreshSolveBinding          `json:"prior_solve_binding"`
	SuppliedInputRef      string                     `json:"supplied_input_ref"`
	RestartPolicy         FreshSolveRestartPolicy    `json:"restart_policy"`
}

func (handoff FreshSolveHandoff) Validate() error {
	if err := requireText("ESCALATION", "request_id", handoff.RequestID, "supplied_input_ref", handoff.SuppliedInputRef); err != nil {
		return err
	}
	if err := handoff.PriorSolveBinding.Validate(); err != nil {
		return err
	}
	if handoff.RestartPolicy != RestartRequired {
		return contractError("ESCALATION_RESTART_POLICY_INVALID", "every resolution input requires a fresh solve")
	}
	want, err := targetFor(handoff.InputKind)
	if err != nil {
		return err
	}
	if handoff.MaterializationTarget != want {
		return contractError("ESCALATION_MATERIALIZATION_TARGET_CONFLICT", "the request kind requires its declared materialization boundary")
	}
	return nil
}

func targetFor(kind ResolutionInputKind) (InputMaterializationTarget, error) {
	switch kind {
	case ProjectOwnerIntent:
		return ProjectAuthorityTarget, nil
	case ExternalFactInput:
		return EvidenceTarget, nil
	default:
		return "", contractError("ESCALATION_INPUT_KIND_INVALID", "unknown resolution input kind")
	}
}

// PrepareFreshSolveHandoff routes the reference only. It cannot materialize
// owner intent, admit evidence, continue a prior solve, or execute a plan.
func PrepareFreshSolveHandoff(request ResolutionInputRequest, current FreshSolveBinding, suppliedRef string) (FreshSolveHandoff, error) {
	if _, err := RequireCurrentResolutionInputRequest(request, current); err != nil {
		return FreshSolveHandoff{}, err
	}
	target, err := targetFor(request.Kind)
	if err != nil {
		return FreshSolveHandoff{}, err
	}
	handoff := FreshSolveHandoff{request.ID, request.Kind, target, request.SolveBinding, suppliedRef, RestartRequired}
	if handoff.PriorSolveBinding.RepositoryRevision != nil {
		revision := *handoff.PriorSolveBinding.RepositoryRevision
		handoff.PriorSolveBinding.RepositoryRevision = &revision
	}
	if err := handoff.Validate(); err != nil {
		return FreshSolveHandoff{}, err
	}
	return handoff, nil
}
