package remediation

import (
	"cmp"
	"encoding/json"
	"slices"
	"strings"
)

type CandidateSupportingReferences struct {
	Rules       []string `json:"RULE"`
	Facts       []string `json:"FACT"`
	Authorities []string `json:"AUTHORITY"`
}

// SolveSubject identifies the subject of an unmet authority requirement. It
// preserves the existing governance coordinate without resolving any authority.
type SolveSubject struct {
	AuthorityDomain   string            `json:"authority_domain"`
	RepositoryBinding RepositoryBinding `json:"repository_binding"`
	SubjectType       string            `json:"subject_type"`
	SubjectIdentity   SubjectIdentity   `json:"subject_identity"`
}

type RepositoryBinding struct {
	Scheme       string `json:"scheme"`
	Host         string `json:"host"`
	RepositoryID string `json:"repository_id"`
}

type SubjectIdentity struct {
	Scheme string `json:"scheme"`
	Value  string `json:"value"`
}

type AuthorityRequirement struct {
	RequirementID   string       `json:"requirement_id"`
	SubjectBinding  SolveSubject `json:"subject_binding"`
	RequiredEffects []string     `json:"required_effects"`
}

type CandidateProvenance struct {
	SupportingReferences  CandidateSupportingReferences `json:"supporting_references"`
	AuthorityRequirements []AuthorityRequirement        `json:"authority_requirements"`
}

type SemanticCandidateProposal struct {
	InterpretationID      string                        `json:"interpretation_id"`
	RuleID                string                        `json:"rule_id"`
	RemediationFamily     string                        `json:"remediation_family"`
	TargetState           json.RawMessage               `json:"target_state"`
	SupportingReferences  CandidateSupportingReferences `json:"supporting_references"`
	AuthorityRequirements []AuthorityRequirement        `json:"authority_requirements"`
}

type SemanticCandidate struct {
	ID                string              `json:"id"`
	RuleID            string              `json:"rule_id"`
	RemediationFamily string              `json:"remediation_family"`
	TargetState       json.RawMessage     `json:"target_state"`
	Provenance        CandidateProvenance `json:"provenance"`
}

func candidateIDs(name string, ids []string, required bool) ([]string, error) {
	if err := requireIDs("CANDIDATE", name, ids, required); err != nil {
		if e := err.(*ContractError); e.Code == "CANDIDATE_IDS_DUPLICATE" {
			return nil, contractError("CANDIDATE_REFERENCE_DUPLICATE", e.Detail)
		} else if e.Code == "CANDIDATE_IDS_EMPTY" {
			return nil, contractError("CANDIDATE_REFERENCE_COLLECTION_EMPTY", e.Detail)
		}
		return nil, err
	}
	return sortedIDs(ids), nil
}

func normalizeProvenance(provenance CandidateProvenance, rule string) (CandidateProvenance, error) {
	refs := &provenance.SupportingReferences
	var err error
	refs.Rules, err = candidateIDs("rules", refs.Rules, true)
	if err != nil {
		return CandidateProvenance{}, err
	}
	if !slices.Contains(refs.Rules, rule) {
		return CandidateProvenance{}, contractError("CANDIDATE_PRODUCER_RULE_NOT_CONSUMED", "supporting rules must include the producing rule")
	}
	refs.Facts, err = candidateIDs("facts", refs.Facts, false)
	if err != nil {
		return CandidateProvenance{}, err
	}
	refs.Authorities, err = candidateIDs("authorities", refs.Authorities, false)
	if err != nil {
		return CandidateProvenance{}, err
	}
	provenance.AuthorityRequirements = array(provenance.AuthorityRequirements)
	slices.SortFunc(provenance.AuthorityRequirements, func(a, b AuthorityRequirement) int {
		return cmp.Compare(a.RequirementID, b.RequirementID)
	})
	for index, requirement := range provenance.AuthorityRequirements {
		if err := requireText("CANDIDATE", "requirement_id", requirement.RequirementID); err != nil {
			return CandidateProvenance{}, err
		}
		if index > 0 && provenance.AuthorityRequirements[index-1].RequirementID == requirement.RequirementID {
			return CandidateProvenance{}, contractError("AUTHORITY_REQUIREMENT_DUPLICATE", "authority requirement identities must be unique")
		}
		subject := requirement.SubjectBinding
		for _, field := range []string{subject.AuthorityDomain, subject.RepositoryBinding.Scheme, subject.RepositoryBinding.Host,
			subject.RepositoryBinding.RepositoryID, subject.SubjectType, subject.SubjectIdentity.Scheme, subject.SubjectIdentity.Value} {
			if field == "" {
				return CandidateProvenance{}, contractError("AUTHORITY_REQUIREMENT_SUBJECT_BINDING_INVALID", "requirement must identify a structured governance subject")
			}
		}
		requirement.RequiredEffects, err = candidateIDs("required_effects", requirement.RequiredEffects, true)
		if err != nil {
			return CandidateProvenance{}, err
		}
		provenance.AuthorityRequirements[index] = requirement
	}
	return provenance, nil
}

func normalizeCandidate(candidate SemanticCandidate, scopeCode string) (SemanticCandidate, error) {
	if err := requireText("CANDIDATE", "id", candidate.ID, "rule_id", candidate.RuleID, "remediation_family", candidate.RemediationFamily); err != nil {
		return SemanticCandidate{}, err
	}
	prefix := "interpretation:" + candidate.RuleID + ":"
	if !strings.HasPrefix(candidate.ID, prefix) || len(candidate.ID) == len(prefix) {
		return SemanticCandidate{}, contractError(scopeCode, "candidate must preserve its rule-scoped interpretation identity")
	}
	target, err := canonicalValue(candidate.TargetState, "CANDIDATE_TARGET_STATE_INVALID", "CANDIDATE_TARGET_STATE_UNSERIALIZABLE")
	if err != nil {
		return SemanticCandidate{}, err
	}
	provenance, err := normalizeProvenance(candidate.Provenance, candidate.RuleID)
	if err != nil {
		return SemanticCandidate{}, err
	}
	candidate.TargetState, candidate.Provenance = target, provenance
	return candidate, nil
}

func (candidate SemanticCandidate) Validate() error {
	_, err := normalizeCandidate(candidate, "CANDIDATE_ID_SCOPE_MISMATCH")
	return err
}

// MaterializeCandidate copies a declared target and its provenance, without
// ranking the target or treating references as authority to make a change.
func MaterializeCandidate(proposal SemanticCandidateProposal) (SemanticCandidate, error) {
	return normalizeCandidate(SemanticCandidate{
		ID: proposal.InterpretationID, RuleID: proposal.RuleID, RemediationFamily: proposal.RemediationFamily,
		TargetState: proposal.TargetState,
		Provenance:  CandidateProvenance{proposal.SupportingReferences, proposal.AuthorityRequirements},
	}, "CANDIDATE_INTERPRETATION_ID_SCOPE_MISMATCH")
}

func MaterializeCandidates(proposals []SemanticCandidateProposal) ([]SemanticCandidate, error) {
	result := make([]SemanticCandidate, 0, len(proposals))
	seen := make(map[string]bool, len(proposals))
	for _, proposal := range proposals {
		candidate, err := MaterializeCandidate(proposal)
		if err != nil {
			return nil, err
		}
		if seen[candidate.ID] {
			return nil, contractError("CANDIDATE_ID_DUPLICATE", "candidate IDs must be unique within a solve")
		}
		seen[candidate.ID] = true
		result = append(result, candidate)
	}
	return result, nil
}
