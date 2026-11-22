package contracts_test

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/src/ptsip/evidence"
	"github.com/Kinirin/PTSIP/src/ptsip/remediation"
)

func remediationContext(t *testing.T, status string) remediation.Context {
	t.Helper()
	revision := strings.Repeat("1", 40)
	observation, err := evidence.BuildEvaluationContext(evidence.SnapshotBinding{
		RepositoryRoot: "repo", Revision: &revision, StatusFingerprint: status, TrackedContentFingerprint: "content",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	set, err := evidence.Normalize(nil, observation, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return remediation.Context{Evidence: set, Specification: remediation.SpecificationBinding{
		Family: "0.3.7-draft", Source: "https://github.com/Kinirin/PTSIP", Revision: strings.Repeat("a", 40),
	}}
}

func remediationProposal(choice string) remediation.SemanticCandidateProposal {
	return remediation.SemanticCandidateProposal{
		InterpretationID: "interpretation:rule:" + choice, RuleID: "rule", RemediationFamily: "component",
		TargetState:          json.RawMessage(`{"z":"한글","a":1.0}`),
		SupportingReferences: remediation.CandidateSupportingReferences{Rules: []string{"rule", "another-rule"}, Facts: []string{"z-fact", "a-fact"}},
	}
}

func remediationError(t *testing.T, err error, want string) {
	t.Helper()
	var failure *remediation.ContractError
	if !errors.As(err, &failure) || failure.Code != want {
		t.Fatalf("got %v, wanted contract error %s", err, want)
	}
}

func TestRemediationCandidatesPreserveCanonicalProvenance(t *testing.T) {
	first := remediationProposal("choice-b")
	first.AuthorityRequirements = []remediation.AuthorityRequirement{{
		RequirementID: "owner-request", RequiredEffects: []string{"select", "bind"},
		SubjectBinding: remediation.SolveSubject{
			AuthorityDomain: "project", SubjectType: "component",
			RepositoryBinding: remediation.RepositoryBinding{Scheme: "github", Host: "github.com", RepositoryID: "Kinirin/PTSIP"},
			SubjectIdentity:   remediation.SubjectIdentity{Scheme: "component-id", Value: "core"},
		},
	}}
	second := remediationProposal("choice-a")
	candidates, err := remediation.MaterializeCandidates([]remediation.SemanticCandidateProposal{first, second})
	if err != nil {
		t.Fatal(err)
	}
	want := candidates[0]
	if len(candidates) != 2 || want.ID != first.InterpretationID || string(want.TargetState) != `{"a":1.0,"z":"한글"}` {
		t.Fatalf("candidate identity, source order or canonical target changed: %#v", candidates)
	}
	first.SupportingReferences.Rules[0] = "changed"
	first.AuthorityRequirements[0].RequiredEffects[0] = "changed"
	if !slices.Equal(want.Provenance.SupportingReferences.Rules, []string{"another-rule", "rule"}) ||
		!slices.Equal(want.Provenance.AuthorityRequirements[0].RequiredEffects, []string{"bind", "select"}) {
		t.Fatalf("candidate did not preserve isolated canonical references: %#v", want.Provenance)
	}
	_, err = remediation.MaterializeCandidates([]remediation.SemanticCandidateProposal{second, second})
	remediationError(t, err, "CANDIDATE_ID_DUPLICATE")
	first.InterpretationID = "interpretation:another-rule:choice-b"
	_, err = remediation.MaterializeCandidate(first)
	remediationError(t, err, "CANDIDATE_INTERPRETATION_ID_SCOPE_MISMATCH")
	second.TargetState = json.RawMessage(`{"duplicate":1,"duplicate":2}`)
	_, err = remediation.MaterializeCandidate(second)
	remediationError(t, err, "CANDIDATE_TARGET_STATE_UNSERIALIZABLE")
}

type remediationAuthorityVerifier func(remediation.Context, []string) error

func (f remediationAuthorityVerifier) VerifyConsumedAuthority(context remediation.Context, ids []string) error {
	return f(context, ids)
}

func TestRemediationReductionRequiresExactCurrentProof(t *testing.T) {
	context := remediationContext(t, "status")
	candidates, err := remediation.MaterializeCandidates([]remediation.SemanticCandidateProposal{remediationProposal("choice-b"), remediationProposal("choice-a")})
	if err != nil {
		t.Fatal(err)
	}
	binding, err := remediation.NewEliminationSolveBinding(context, nil)
	if err != nil {
		t.Fatal(err)
	}
	retained := candidates[1].ID
	record := remediation.EliminationRecord{CandidateID: candidates[0].ID, ReasonFamily: remediation.SemanticallyEquivalentToRetainedCandidate,
		SolveBinding: binding, RetainedCandidateID: &retained,
		SupportingReferences: []remediation.EliminationReference{{Kind: remediation.CandidateReference, Identity: retained}},
	}
	result, err := remediation.ReduceSurvivors(context, candidates, []remediation.EliminationRecord{record}, nil, nil)
	if err != nil || len(result.Survivors) != 1 || result.Survivors[0].ID != retained {
		t.Fatalf("unexpected reduction %#v, %v", result, err)
	}
	encoded, err := json.Marshal(result)
	if err != nil || !strings.Contains(string(encoded), `"project_authority":false`) || !strings.Contains(string(encoded), `"solve_binding_id":"solve-binding:`) {
		t.Fatalf("reduction lost proof metadata %s, %v", encoded, err)
	}
	_, err = remediation.ReduceSurvivors(remediationContext(t, "changed"), candidates, []remediation.EliminationRecord{record}, nil, nil)
	remediationError(t, err, "ELIMINATION_SOLVE_BINDING_MISMATCH")
	_, err = remediation.ReduceSurvivors(context, candidates, []remediation.EliminationRecord{record, record}, nil, nil)
	remediationError(t, err, "ELIMINATION_RECORD_DUPLICATE")
	other := remediation.EliminationRecord{CandidateID: retained, ReasonFamily: remediation.ViolatesNormativeConstraint, SolveBinding: binding,
		SupportingReferences: []remediation.EliminationReference{{Kind: remediation.RuleReference, Identity: "rule"}},
	}
	_, err = remediation.ReduceSurvivors(context, candidates, []remediation.EliminationRecord{record, other}, nil, nil)
	remediationError(t, err, "ELIMINATION_RETAINED_CANDIDATE_NOT_SURVIVOR")
	ids := []string{"consumed-owner-record"}
	authorityBinding, err := remediation.NewEliminationSolveBinding(context, ids)
	if err != nil {
		t.Fatal(err)
	}
	authorityRecord := remediation.EliminationRecord{CandidateID: candidates[0].ID, ReasonFamily: remediation.ConflictsWithExplicitCurrentAuthority,
		SolveBinding: authorityBinding, SupportingReferences: []remediation.EliminationReference{{Kind: remediation.AuthorityReference, Identity: ids[0]}},
	}
	_, err = remediation.ReduceSurvivors(context, candidates, []remediation.EliminationRecord{authorityRecord}, ids, nil)
	remediationError(t, err, "ELIMINATION_AUTHORITY_NOT_BOUND_TO_SOLVE")
	verifier := remediationAuthorityVerifier(func(ctx remediation.Context, got []string) error {
		if ctx.Evidence.DeterministicDigest != context.Evidence.DeterministicDigest || !slices.Equal(got, ids) {
			return errors.New("authority was not consumed by this solve")
		}
		return nil
	})
	result, err = remediation.ReduceSurvivors(context, candidates, []remediation.EliminationRecord{authorityRecord}, ids, verifier)
	if err != nil || len(result.Survivors) != 1 {
		t.Fatalf("supplied current authority verification not consumed: %#v, %v", result, err)
	}
	authorityRecord.SupportingReferences[0].Identity = "unconsumed"
	_, err = remediation.ReduceSurvivors(context, candidates, []remediation.EliminationRecord{authorityRecord}, ids, verifier)
	remediationError(t, err, "ELIMINATION_AUTHORITY_NOT_BOUND_TO_SOLVE")
}

func TestRemediationEscalationUsesStableIdentityAndRequiresFreshSolve(t *testing.T) {
	context := remediationContext(t, "status")
	if _, err := remediation.NewFreshSolveBinding(context, "authority-snapshot:alpha"); err != nil {
		t.Fatal(err)
	}
	// Match the existing canonical JSON digest algorithm across Unicode, nil
	// repository revision, differing candidate order and empty proof lists.
	binding := remediation.FreshSolveBinding{EvidenceDigest: strings.Repeat("e", 64), Specification: context.Specification,
		RepositoryStatusFingerprint: "status", RepositoryTrackedContentFingerprint: "content", AuthoritySnapshotID: "authority-snapshot:alpha"}
	candidates, err := remediation.MaterializeCandidates([]remediation.SemanticCandidateProposal{remediationProposal("choice-b"), remediationProposal("choice-a")})
	if err != nil {
		t.Fatal(err)
	}
	input := remediation.ResolutionInput{Outcome: remediation.OwnerIntentRequired, SolveBinding: binding, Candidates: candidates, UnresolvedDimension: "선택", RequiredInput: "결정"}
	request, err := remediation.BuildResolutionInputRequest(input)
	if err != nil || request.ID != "resolution-input:93aecb717c859c7e3221b3b5" || request.OwnerEscalationProof.ID != "escalation-proof:0dd5a284e75f4f651f469c03" {
		t.Fatalf("unexpected canonical request %#v, %v", request, err)
	}
	handoff, err := remediation.PrepareFreshSolveHandoff(request, binding, "owner-intent-ref")
	if err != nil || handoff.MaterializationTarget != remediation.ProjectAuthorityTarget || handoff.RestartPolicy != remediation.RestartRequired {
		t.Fatalf("unexpected owner handoff %#v, %v", handoff, err)
	}
	changed := binding
	changed.AuthoritySnapshotID = "authority-snapshot:beta"
	comparison, err := remediation.RequireCurrentResolutionInputRequest(request, changed)
	remediationError(t, err, "ESCALATION_REQUEST_STALE")
	if !slices.Equal(comparison.ChangedDimensions, []string{"authority_snapshot_id"}) {
		t.Fatalf("unexpected stale dimensions %#v", comparison)
	}
	input.Candidates = candidates[:1]
	_, err = remediation.BuildResolutionInputRequest(input)
	remediationError(t, err, "ESCALATION_OWNER_CARDINALITY_INVALID")
	input.Outcome, input.Candidates = remediation.ExternalFactRequired, nil
	request, err = remediation.BuildResolutionInputRequest(input)
	if err != nil || request.ExternalFactRequirement == nil || request.OwnerEscalationProof != nil {
		t.Fatalf("unexpected external fact request %#v, %v", request, err)
	}
	handoff, err = remediation.PrepareFreshSolveHandoff(request, binding, "evidence-ref")
	if err != nil || handoff.MaterializationTarget != remediation.EvidenceTarget || handoff.RestartPolicy != remediation.RestartRequired {
		t.Fatalf("unexpected external fact handoff %#v, %v", handoff, err)
	}
	context.Evidence.RecordCount++
	_, err = remediation.NewFreshSolveBinding(context, "authority-snapshot:alpha")
	remediationError(t, err, "REMEDIATION_EVIDENCE_TYPE")
}

func TestRemediationRuleContractsRetainNonAuthority(t *testing.T) {
	capability := remediation.OperationalRuleCapabilityDeclaration{CandidateGeneration: remediation.Unsupported,
		PostconditionVerification: remediation.Unsupported, AutomationLevel: remediation.EvaluationOnly}
	got, err := json.Marshal(capability)
	if err != nil || !strings.Contains(string(got), `"authoritative":false`) || !strings.Contains(string(got), `"possible_requirements":[]`) {
		t.Fatalf("invalid declaration %s, %v", got, err)
	}
	capability.AutomationLevel = remediation.SemanticRemediation
	remediationError(t, capability.Validate(), "OPERATIONAL_RULE_AUTOMATION_CAPABILITY_INCONSISTENT")
	context := remediationContext(t, "status")
	evaluation := remediation.OperationalRuleEvaluation{RuleID: "rule", Status: remediation.Conformant}
	remediationError(t, evaluation.Validate(), "OPERATIONAL_RULE_CONSTRAINT_REQUIRED")
	evaluation.Constraint = &remediation.NormativeConstraint{RuleID: "other", Specification: context.Specification, Subject: "component", Requirement: "rule requirement"}
	remediationError(t, evaluation.Validate(), "OPERATIONAL_RULE_CONSTRAINT_RULE_ID_MISMATCH")
	evaluation.Constraint.RuleID = evaluation.RuleID
	if err := evaluation.Validate(); err != nil {
		t.Fatal(err)
	}
	result := remediation.PostconditionResult{SemanticPlanID: "plan", Satisfied: true, FailedCheckIDs: []string{"failed"}}
	remediationError(t, result.Validate(), "REMEDIATION_POSTCONDITION_CONFLICT")
}
