package remediation

import (
	"cmp"
	"encoding/json"
	"slices"
)

type EliminationReasonFamily string

const (
	ViolatesNormativeConstraint               EliminationReasonFamily = "VIOLATES_NORMATIVE_CONSTRAINT"
	ConflictsWithExplicitCurrentAuthority     EliminationReasonFamily = "CONFLICTS_WITH_EXPLICIT_CURRENT_AUTHORITY"
	FailsRequiredPrecondition                 EliminationReasonFamily = "FAILS_REQUIRED_PRECONDITION"
	BreaksRequiredArchitectureInvariant       EliminationReasonFamily = "BREAKS_REQUIRED_ARCHITECTURE_INVARIANT"
	SemanticallyEquivalentToRetainedCandidate EliminationReasonFamily = "PROVABLY_SEMANTICALLY_EQUIVALENT_TO_RETAINED_CANDIDATE"
	DominatedUnderSemanticObjective           EliminationReasonFamily = "PROVABLY_DOMINATED_UNDER_MODELED_SEMANTIC_OBJECTIVE"
)

type EliminationReferenceKind string

const (
	RuleReference                  EliminationReferenceKind = "RULE"
	FactReference                  EliminationReferenceKind = "FACT"
	AuthorityReference             EliminationReferenceKind = "AUTHORITY"
	PreconditionReference          EliminationReferenceKind = "PRECONDITION"
	ArchitectureInvariantReference EliminationReferenceKind = "ARCHITECTURE_INVARIANT"
	CandidateReference             EliminationReferenceKind = "CANDIDATE"
	SemanticObjectiveReference     EliminationReferenceKind = "SEMANTIC_OBJECTIVE"
)

var requiredReferenceKinds = map[EliminationReasonFamily][]EliminationReferenceKind{
	ViolatesNormativeConstraint:               {RuleReference},
	ConflictsWithExplicitCurrentAuthority:     {AuthorityReference},
	FailsRequiredPrecondition:                 {PreconditionReference},
	BreaksRequiredArchitectureInvariant:       {ArchitectureInvariantReference},
	SemanticallyEquivalentToRetainedCandidate: {CandidateReference},
	DominatedUnderSemanticObjective:           {CandidateReference, SemanticObjectiveReference},
}

type EliminationReference struct {
	Kind     EliminationReferenceKind `json:"kind"`
	Identity string                   `json:"identity"`
}

// ConsumedAuthorityVerifier is supplied by the authority owner. It verifies
// that the exact provided references were consumed in the current solve.
type ConsumedAuthorityVerifier interface {
	VerifyConsumedAuthority(Context, []string) error
}

type EliminationSolveBinding struct {
	EvidenceIdentity            string               `json:"evidence_identity"`
	SpecificationIdentity       SpecificationBinding `json:"specification_identity"`
	ConsumedAuthorityIdentities []string             `json:"consumed_authority_identities"`
}

func NewEliminationSolveBinding(context Context, authorityIDs []string) (EliminationSolveBinding, error) {
	if err := context.Validate(); err != nil {
		return EliminationSolveBinding{}, err
	}
	return (EliminationSolveBinding{context.Evidence.DeterministicDigest, context.Specification, authorityIDs}).normalized()
}

func (binding EliminationSolveBinding) normalized() (EliminationSolveBinding, error) {
	if !sha256Digest.MatchString(binding.EvidenceIdentity) {
		return EliminationSolveBinding{}, contractError("ELIMINATION_EVIDENCE_IDENTITY_INVALID", "normalized evidence must have a lowercase SHA-256 digest")
	}
	if err := binding.SpecificationIdentity.Validate(); err != nil {
		return EliminationSolveBinding{}, err
	}
	if err := requireIDs("ELIMINATION", "consumed_authority_identities", binding.ConsumedAuthorityIdentities, false); err != nil {
		return EliminationSolveBinding{}, err
	}
	binding.ConsumedAuthorityIdentities = sortedIDs(binding.ConsumedAuthorityIdentities)
	return binding, nil
}

func (binding EliminationSolveBinding) ID() (string, error) {
	canonical, err := binding.normalized()
	if err != nil {
		return "", err
	}
	return stableID("solve-binding", canonical)
}

type EliminationRecord struct {
	CandidateID          string                  `json:"candidate_id"`
	ReasonFamily         EliminationReasonFamily `json:"reason_family"`
	SolveBinding         EliminationSolveBinding `json:"solve_binding"`
	SupportingReferences []EliminationReference  `json:"supporting_references"`
	RetainedCandidateID  *string                 `json:"retained_candidate_id"`
	Detail               *string                 `json:"detail"`
}

func (record EliminationRecord) normalized() (EliminationRecord, error) {
	if err := requireText("ELIMINATION", "candidate_id", record.CandidateID); err != nil {
		return EliminationRecord{}, err
	}
	if err := optionalText("ELIMINATION", "detail", record.Detail); err != nil {
		return EliminationRecord{}, err
	}
	needed, valid := requiredReferenceKinds[record.ReasonFamily]
	if !valid {
		return EliminationRecord{}, contractError("ELIMINATION_REASON_FAMILY_INVALID", "unknown elimination reason")
	}
	binding, err := record.SolveBinding.normalized()
	if err != nil {
		return EliminationRecord{}, err
	}
	record.SolveBinding = binding
	if len(record.SupportingReferences) == 0 {
		return EliminationRecord{}, contractError("ELIMINATION_REFERENCES_REQUIRED", "elimination requires supporting references")
	}
	refs := array(record.SupportingReferences)
	slices.SortFunc(refs, func(a, b EliminationReference) int {
		if order := cmp.Compare(a.Kind, b.Kind); order != 0 {
			return order
		}
		return cmp.Compare(a.Identity, b.Identity)
	})
	for i, ref := range refs {
		switch ref.Kind {
		case RuleReference, FactReference, AuthorityReference, PreconditionReference, ArchitectureInvariantReference, CandidateReference, SemanticObjectiveReference:
		default:
			return EliminationRecord{}, contractError("ELIMINATION_REFERENCE_KIND_INVALID", "unknown proof reference kind")
		}
		if err := requireText("ELIMINATION", "reference identity", ref.Identity); err != nil {
			return EliminationRecord{}, err
		}
		if i > 0 && refs[i-1] == ref {
			return EliminationRecord{}, contractError("ELIMINATION_REFERENCE_DUPLICATE", "elimination references must be unique")
		}
		if ref.Kind == AuthorityReference && !slices.Contains(binding.ConsumedAuthorityIdentities, ref.Identity) {
			return EliminationRecord{}, contractError("ELIMINATION_AUTHORITY_NOT_BOUND_TO_SOLVE", "authority reference was not consumed in this solve")
		}
	}
	for _, kind := range needed {
		if !slices.ContainsFunc(refs, func(ref EliminationReference) bool { return ref.Kind == kind }) {
			return EliminationRecord{}, failedContract("ELIMINATION_REQUIRED_REFERENCE_KIND_MISSING", "elimination requires %s references", kind)
		}
	}
	comparative := record.ReasonFamily == SemanticallyEquivalentToRetainedCandidate || record.ReasonFamily == DominatedUnderSemanticObjective
	if comparative {
		if record.RetainedCandidateID == nil {
			return EliminationRecord{}, contractError("ELIMINATION_RETAINED_CANDIDATE_REQUIRED", "comparative elimination requires a retained candidate")
		}
		retained := *record.RetainedCandidateID
		if err := requireText("ELIMINATION", "retained_candidate_id", retained); err != nil {
			return EliminationRecord{}, err
		}
		if retained == record.CandidateID {
			return EliminationRecord{}, contractError("ELIMINATION_SELF_COMPARISON_INVALID", "candidate cannot retain itself")
		}
		if !slices.Contains(refs, EliminationReference{CandidateReference, retained}) {
			return EliminationRecord{}, contractError("ELIMINATION_RETAINED_CANDIDATE_REFERENCE_REQUIRED", "the retained candidate must appear in the proof references")
		}
		record.RetainedCandidateID = &retained
	} else if record.RetainedCandidateID != nil {
		return EliminationRecord{}, contractError("ELIMINATION_RETAINED_CANDIDATE_UNEXPECTED", "only comparative elimination may retain a candidate")
	}
	if record.Detail != nil {
		value := *record.Detail
		record.Detail = &value
	}
	record.SupportingReferences = refs
	return record, nil
}

func (record EliminationRecord) ID() (string, error) {
	canonical, err := record.normalized()
	if err != nil {
		return "", err
	}
	bindingID, err := canonical.SolveBinding.ID()
	if err != nil {
		return "", err
	}
	return stableID("elimination", map[string]any{
		"candidate_id": canonical.CandidateID, "reason_family": canonical.ReasonFamily, "solve_binding_id": bindingID,
		"supporting_references": canonical.SupportingReferences, "retained_candidate_id": canonical.RetainedCandidateID,
	})
}

func (record EliminationRecord) MarshalJSON() ([]byte, error) {
	canonical, err := record.normalized()
	if err != nil {
		return nil, err
	}
	id, err := canonical.ID()
	if err != nil {
		return nil, err
	}
	bindingID, err := canonical.SolveBinding.ID()
	if err != nil {
		return nil, err
	}
	type entry EliminationRecord
	return json.Marshal(struct {
		entry
		ID               string `json:"id"`
		SolveBindingID   string `json:"solve_binding_id"`
		ProjectAuthority bool   `json:"project_authority"`
	}{entry(canonical), id, bindingID, false})
}

type SurvivorReductionResult struct {
	SolveBinding       EliminationSolveBinding
	Survivors          []SemanticCandidate
	EliminationRecords []EliminationRecord
}

func (result SurvivorReductionResult) MarshalJSON() ([]byte, error) {
	bindingID, err := result.SolveBinding.ID()
	if err != nil {
		return nil, err
	}
	var survivors, eliminated = []string{}, []string{}
	for _, candidate := range result.Survivors {
		survivors = append(survivors, candidate.ID)
	}
	for _, record := range result.EliminationRecords {
		eliminated = append(eliminated, record.CandidateID)
	}
	slices.Sort(eliminated)
	return json.Marshal(map[string]any{
		"solve_binding": result.SolveBinding, "solve_binding_id": bindingID, "survivor_candidate_ids": survivors,
		"eliminated_candidate_ids": slices.Compact(eliminated), "elimination_records": array(result.EliminationRecords),
	})
}

// ReduceSurvivors applies already modeled proofs to the exact current solve.
// Referenced authority must be verified by the external owner before use.
func ReduceSurvivors(context Context, candidates []SemanticCandidate, records []EliminationRecord, authorityIDs []string, verifier ConsumedAuthorityVerifier) (SurvivorReductionResult, error) {
	current, err := NewEliminationSolveBinding(context, authorityIDs)
	if err != nil {
		return SurvivorReductionResult{}, err
	}
	if len(current.ConsumedAuthorityIdentities) != 0 {
		if verifier == nil {
			return SurvivorReductionResult{}, contractError("ELIMINATION_AUTHORITY_NOT_BOUND_TO_SOLVE", "authority verification is unavailable")
		}
		if err := verifier.VerifyConsumedAuthority(context, array(current.ConsumedAuthorityIdentities)); err != nil {
			return SurvivorReductionResult{}, failedContract("ELIMINATION_AUTHORITY_NOT_BOUND_TO_SOLVE", "authority verification failed: %v", err)
		}
	}
	currentID, err := current.ID()
	if err != nil {
		return SurvivorReductionResult{}, err
	}
	byID := make(map[string]SemanticCandidate, len(candidates))
	for _, candidate := range candidates {
		candidate, err := normalizeCandidate(candidate, "CANDIDATE_ID_SCOPE_MISMATCH")
		if err != nil {
			return SurvivorReductionResult{}, err
		}
		if _, exists := byID[candidate.ID]; exists {
			return SurvivorReductionResult{}, contractError("ELIMINATION_CANDIDATE_ID_DUPLICATE", "candidate IDs must be unique")
		}
		byID[candidate.ID] = candidate
	}
	recordByID := make(map[string]EliminationRecord, len(records))
	eliminated := make(map[string]bool, len(records))
	for _, record := range records {
		record, err := record.normalized()
		if err != nil {
			return SurvivorReductionResult{}, err
		}
		bindingID, err := record.SolveBinding.ID()
		if err != nil {
			return SurvivorReductionResult{}, err
		}
		if bindingID != currentID {
			return SurvivorReductionResult{}, contractError("ELIMINATION_SOLVE_BINDING_MISMATCH", "changed evidence, specification or consumed authority requires a fresh solve")
		}
		if _, exists := byID[record.CandidateID]; !exists {
			return SurvivorReductionResult{}, contractError("ELIMINATION_CANDIDATE_UNKNOWN", "proof did not identify a current candidate")
		}
		if record.RetainedCandidateID != nil {
			if _, exists := byID[*record.RetainedCandidateID]; !exists {
				return SurvivorReductionResult{}, contractError("ELIMINATION_RETAINED_CANDIDATE_UNKNOWN", "retained candidate does not belong to the current solve")
			}
		}
		id, err := record.ID()
		if err != nil {
			return SurvivorReductionResult{}, err
		}
		if _, exists := recordByID[id]; exists {
			return SurvivorReductionResult{}, contractError("ELIMINATION_RECORD_DUPLICATE", "proof was supplied more than once")
		}
		recordByID[id] = record
		eliminated[record.CandidateID] = true
	}
	result := SurvivorReductionResult{SolveBinding: current, Survivors: []SemanticCandidate{}, EliminationRecords: []EliminationRecord{}}
	var recordIDs []string
	for id := range recordByID {
		recordIDs = append(recordIDs, id)
	}
	slices.Sort(recordIDs)
	for _, id := range recordIDs {
		record := recordByID[id]
		if record.RetainedCandidateID != nil && eliminated[*record.RetainedCandidateID] {
			return SurvivorReductionResult{}, contractError("ELIMINATION_RETAINED_CANDIDATE_NOT_SURVIVOR", "comparative proof retains an eliminated candidate")
		}
		result.EliminationRecords = append(result.EliminationRecords, record)
	}
	for id, candidate := range byID {
		if !eliminated[id] {
			result.Survivors = append(result.Survivors, candidate)
		}
	}
	slices.SortFunc(result.Survivors, func(a, b SemanticCandidate) int { return cmp.Compare(a.ID, b.ID) })
	return result, nil
}
