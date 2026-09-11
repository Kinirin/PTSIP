from __future__ import annotations

from dataclasses import FrozenInstanceError

import pytest

from ptsip.evidence.contract import (
    EvidenceEvaluationContext,
    NormalizedEvidenceSet,
    SnapshotBinding,
)
from ptsip.remediation.domain import RemediationContext, SemanticCandidate
from ptsip.remediation.solution.elimination import (
    EliminationContractError,
    EliminationReasonFamily,
    EliminationRecord,
    EliminationReference,
    EliminationReferenceKind,
    EliminationSolveBinding,
    reduce_survivors,
)
from ptsip.specification_binding import SPECIFICATION_037


def _context(*, tracked_content_fingerprint: str = "content") -> RemediationContext:
    evidence = NormalizedEvidenceSet(
        context=EvidenceEvaluationContext(
            evaluation_id="wu-02-s2-elimination-test",
            snapshot=SnapshotBinding(
                repository_root=".",
                revision=None,
                status_fingerprint="status",
                tracked_content_fingerprint=tracked_content_fingerprint,
            ),
        ),
        records=(),
        channels=(),
    )
    return RemediationContext(evidence=evidence, specification=SPECIFICATION_037)


def _candidate(candidate_id: str) -> SemanticCandidate:
    return SemanticCandidate(
        id=candidate_id,
        rule_id="PTSIP-PKG-001",
        remediation_family="PACKAGE_ISOLATION",
        target_state={"candidate": candidate_id},
    )


def _ref(kind: EliminationReferenceKind, identity: str) -> EliminationReference:
    return EliminationReference(kind=kind, identity=identity)


def test_reason_family_is_closed_to_approved_elimination_authority() -> None:
    assert [item.value for item in EliminationReasonFamily] == [
        "VIOLATES_NORMATIVE_CONSTRAINT",
        "CONFLICTS_WITH_EXPLICIT_CURRENT_AUTHORITY",
        "FAILS_REQUIRED_PRECONDITION",
        "BREAKS_REQUIRED_ARCHITECTURE_INVARIANT",
        "PROVABLY_SEMANTICALLY_EQUIVALENT_TO_RETAINED_CANDIDATE",
        "PROVABLY_DOMINATED_UNDER_MODELED_SEMANTIC_OBJECTIVE",
    ]

    binding = EliminationSolveBinding.from_context(_context())
    with pytest.raises(EliminationContractError) as exc_info:
        EliminationRecord(
            candidate_id="candidate:1",
            reason_family="LOW_CONFIDENCE",  # type: ignore[arg-type]
            solve_binding=binding,
            supporting_references=(
                _ref(EliminationReferenceKind.FACT, "fact:confidence"),
            ),
        )
    assert exc_info.value.code == "ELIMINATION_REASON_FAMILY_INVALID"


@pytest.mark.parametrize(
    ("reason", "references"),
    [
        (
            EliminationReasonFamily.VIOLATES_NORMATIVE_CONSTRAINT,
            (_ref(EliminationReferenceKind.FACT, "fact:1"),),
        ),
        (
            EliminationReasonFamily.FAILS_REQUIRED_PRECONDITION,
            (_ref(EliminationReferenceKind.FACT, "fact:1"),),
        ),
        (
            EliminationReasonFamily.BREAKS_REQUIRED_ARCHITECTURE_INVARIANT,
            (_ref(EliminationReferenceKind.FACT, "fact:1"),),
        ),
    ],
)
def test_each_elimination_reason_requires_machine_inspectable_basis_kind(
    reason: EliminationReasonFamily,
    references: tuple[EliminationReference, ...],
) -> None:
    with pytest.raises(EliminationContractError) as exc_info:
        EliminationRecord(
            candidate_id="candidate:1",
            reason_family=reason,
            solve_binding=EliminationSolveBinding.from_context(_context()),
            supporting_references=references,
        )
    assert exc_info.value.code == "ELIMINATION_REQUIRED_REFERENCE_KIND_MISSING"


def test_solve_binding_reuses_evidence_and_specification_and_binds_consumed_authority() -> None:
    context = _context()
    binding = EliminationSolveBinding.from_context(
        context,
        consumed_authority_identities=("authority:2", "authority:1"),
    )

    assert binding.evidence_identity == context.evidence.deterministic_digest
    assert binding.specification_identity is SPECIFICATION_037
    assert binding.consumed_authority_identities == ("authority:1", "authority:2")

    equivalent = EliminationSolveBinding.from_context(
        context,
        consumed_authority_identities=("authority:1", "authority:2"),
    )
    assert binding.id == equivalent.id


def test_authority_conflict_must_reference_authority_consumed_by_current_solve() -> None:
    binding = EliminationSolveBinding.from_context(
        _context(),
        consumed_authority_identities=("authority:current",),
    )

    with pytest.raises(EliminationContractError) as exc_info:
        EliminationRecord(
            candidate_id="candidate:1",
            reason_family=EliminationReasonFamily.CONFLICTS_WITH_EXPLICIT_CURRENT_AUTHORITY,
            solve_binding=binding,
            supporting_references=(
                _ref(EliminationReferenceKind.AUTHORITY, "authority:stale"),
            ),
        )
    assert exc_info.value.code == "ELIMINATION_AUTHORITY_NOT_BOUND_TO_SOLVE"

    record = EliminationRecord(
        candidate_id="candidate:1",
        reason_family=EliminationReasonFamily.CONFLICTS_WITH_EXPLICIT_CURRENT_AUTHORITY,
        solve_binding=binding,
        supporting_references=(
            _ref(EliminationReferenceKind.AUTHORITY, "authority:current"),
        ),
    )
    assert record.as_dict()["project_authority"] is False


def test_record_identity_is_canonical_and_human_detail_does_not_rewrite_proof_identity() -> None:
    binding = EliminationSolveBinding.from_context(_context())
    refs = (
        _ref(EliminationReferenceKind.FACT, "fact:membership"),
        _ref(EliminationReferenceKind.RULE, "rule:PTSIP-PKG-001"),
    )
    first = EliminationRecord(
        candidate_id="candidate:tooling",
        reason_family=EliminationReasonFamily.VIOLATES_NORMATIVE_CONSTRAINT,
        solve_binding=binding,
        supporting_references=refs,
        detail="First explanation.",
    )
    second = EliminationRecord(
        candidate_id="candidate:tooling",
        reason_family=EliminationReasonFamily.VIOLATES_NORMATIVE_CONSTRAINT,
        solve_binding=binding,
        supporting_references=tuple(reversed(refs)),
        detail="Different prose, same machine proof.",
    )

    assert first.id == second.id
    assert first.as_dict()["supporting_references"] == [
        {"kind": "FACT", "identity": "fact:membership"},
        {"kind": "RULE", "identity": "rule:PTSIP-PKG-001"},
    ]
    with pytest.raises(FrozenInstanceError):
        first.candidate_id = "candidate:other"  # type: ignore[misc]


def test_valid_elimination_cannot_remain_in_survivor_set() -> None:
    binding = EliminationSolveBinding.from_context(_context())
    candidate_a = _candidate("candidate:a")
    candidate_b = _candidate("candidate:b")
    record = EliminationRecord(
        candidate_id=candidate_b.id,
        reason_family=EliminationReasonFamily.FAILS_REQUIRED_PRECONDITION,
        solve_binding=binding,
        supporting_references=(
            _ref(EliminationReferenceKind.PRECONDITION, "precondition:artifact-known"),
        ),
    )

    result = reduce_survivors(
        (candidate_b, candidate_a),
        (record,),
        solve_binding=binding,
    )

    assert result.survivor_candidate_ids == (candidate_a.id,)
    assert result.eliminated_candidate_ids == (candidate_b.id,)
    assert candidate_b.id not in result.survivor_candidate_ids


def test_downstream_reordering_cannot_reintroduce_eliminated_candidate() -> None:
    binding = EliminationSolveBinding.from_context(_context())
    eliminated = _candidate("candidate:eliminated")
    survivor = _candidate("candidate:survivor")
    record = EliminationRecord(
        candidate_id=eliminated.id,
        reason_family=EliminationReasonFamily.BREAKS_REQUIRED_ARCHITECTURE_INVARIANT,
        solve_binding=binding,
        supporting_references=(
            _ref(
                EliminationReferenceKind.ARCHITECTURE_INVARIANT,
                "invariant:required-boundary",
            ),
        ),
    )

    first = reduce_survivors(
        (eliminated, survivor),
        (record,),
        solve_binding=binding,
    )
    second = reduce_survivors(
        (survivor, eliminated),
        (record,),
        solve_binding=binding,
    )

    assert first.survivor_candidate_ids == second.survivor_candidate_ids == (
        survivor.id,
    )


def test_changed_evidence_binding_invalidates_record_reuse_instead_of_rewriting_history() -> None:
    original = EliminationSolveBinding.from_context(
        _context(tracked_content_fingerprint="before")
    )
    record = EliminationRecord(
        candidate_id="candidate:a",
        reason_family=EliminationReasonFamily.FAILS_REQUIRED_PRECONDITION,
        solve_binding=original,
        supporting_references=(
            _ref(EliminationReferenceKind.PRECONDITION, "precondition:1"),
        ),
    )
    changed = EliminationSolveBinding.from_context(
        _context(tracked_content_fingerprint="after")
    )

    with pytest.raises(EliminationContractError) as exc_info:
        reduce_survivors(
            (_candidate("candidate:a"),),
            (record,),
            solve_binding=changed,
        )

    assert exc_info.value.code == "ELIMINATION_SOLVE_BINDING_MISMATCH"
    assert record.solve_binding is original


def test_changed_consumed_authority_identity_requires_fresh_solve() -> None:
    original = EliminationSolveBinding.from_context(
        _context(),
        consumed_authority_identities=("authority:rev-a",),
    )
    record = EliminationRecord(
        candidate_id="candidate:a",
        reason_family=EliminationReasonFamily.CONFLICTS_WITH_EXPLICIT_CURRENT_AUTHORITY,
        solve_binding=original,
        supporting_references=(
            _ref(EliminationReferenceKind.AUTHORITY, "authority:rev-a"),
        ),
    )
    changed = EliminationSolveBinding.from_context(
        _context(),
        consumed_authority_identities=("authority:rev-b",),
    )

    with pytest.raises(EliminationContractError) as exc_info:
        reduce_survivors(
            (_candidate("candidate:a"),),
            (record,),
            solve_binding=changed,
        )
    assert exc_info.value.code == "ELIMINATION_SOLVE_BINDING_MISMATCH"


def test_equivalence_or_dominance_must_keep_its_declared_retained_candidate_alive() -> None:
    binding = EliminationSolveBinding.from_context(_context())
    candidate_a = _candidate("candidate:a")
    candidate_b = _candidate("candidate:b")

    equivalent = EliminationRecord(
        candidate_id=candidate_b.id,
        reason_family=EliminationReasonFamily.PROVABLY_SEMANTICALLY_EQUIVALENT_TO_RETAINED_CANDIDATE,
        solve_binding=binding,
        supporting_references=(
            _ref(EliminationReferenceKind.CANDIDATE, candidate_a.id),
        ),
        retained_candidate_id=candidate_a.id,
    )
    eliminate_retained = EliminationRecord(
        candidate_id=candidate_a.id,
        reason_family=EliminationReasonFamily.FAILS_REQUIRED_PRECONDITION,
        solve_binding=binding,
        supporting_references=(
            _ref(EliminationReferenceKind.PRECONDITION, "precondition:missing"),
        ),
    )

    with pytest.raises(EliminationContractError) as exc_info:
        reduce_survivors(
            (candidate_a, candidate_b),
            (equivalent, eliminate_retained),
            solve_binding=binding,
        )
    assert exc_info.value.code == "ELIMINATION_RETAINED_CANDIDATE_NOT_SURVIVOR"


def test_record_cannot_eliminate_candidate_absent_from_bound_current_solve() -> None:
    binding = EliminationSolveBinding.from_context(_context())
    record = EliminationRecord(
        candidate_id="candidate:old",
        reason_family=EliminationReasonFamily.FAILS_REQUIRED_PRECONDITION,
        solve_binding=binding,
        supporting_references=(
            _ref(EliminationReferenceKind.PRECONDITION, "precondition:1"),
        ),
    )

    with pytest.raises(EliminationContractError) as exc_info:
        reduce_survivors(
            (_candidate("candidate:new"),),
            (record,),
            solve_binding=binding,
        )
    assert exc_info.value.code == "ELIMINATION_CANDIDATE_UNKNOWN"
