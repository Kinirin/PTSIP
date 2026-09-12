from __future__ import annotations

import pytest

from ptsip.remediation.domain import ResolutionOutcome
from ptsip.remediation.rules.contract import RuleRequirementKind
from ptsip.remediation.solution.recovery import (
    CapabilityRecoveryPath,
    RecoveredCapability,
    recover_on_next_path,
    reject_next_recovery_path,
    start_capability_recovery,
)
from ptsip.remediation.solution.resolution import (
    ResolutionContractError,
    ResolutionResponsibility,
    ResolutionState,
    build_resolution_record,
)


def _authority_requirement(
    requirement_id: str = "authority-requirement:classification",
) -> dict[str, object]:
    return {
        "requirement_id": requirement_id,
        "subject_binding": {
            "authority_domain": "PROJECT_GOVERNANCE",
            "repository_binding": {
                "scheme": "GITHUB_REPOSITORY_ID",
                "host": "github.com",
                "repository_id": "1327447827",
            },
            "subject_type": "GOVERNANCE_TOPIC",
            "subject_identity": {
                "scheme": "SUPPORT_POLICY_ID",
                "value": "SFP-0007",
            },
        },
        "required_effects": ["govern_classification_model"],
    }


def _provenance(
    *requirements: dict[str, object],
) -> dict[str, object]:
    return {
        "supporting_references": {
            "RULE": ["PTSIP-PKG-001"],
            "AUTHORITY": [],
        },
        "authority_requirements": list(requirements),
    }


def _survivor_provenance(
    candidate_ids: tuple[str, ...],
    *requirements: dict[str, object],
) -> dict[str, dict[str, object]]:
    return {
        candidate_id: _provenance(*requirements)
        for candidate_id in candidate_ids
    }


def _pending_recovery(
    requirement: dict[str, object] | None = None,
):
    return start_capability_recovery(
        required_capability="component.classification",
        subject="component:runtime-a",
        authority_requirement=requirement,
    )


def _exhausted_recovery(requirement: dict[str, object] | None = None):
    assessment = _pending_recovery(requirement)
    assessment = reject_next_recovery_path(
        assessment,
        reason_code="NORMALIZATION_NOT_LOSSLESS",
        reason="Normalization would require semantic interpretation.",
    )
    assessment = reject_next_recovery_path(
        assessment,
        reason_code="NO_EXPLICIT_ADAPTER",
        reason="No exact eligible Project Authority compatibility mapping exists.",
    )
    return reject_next_recovery_path(
        assessment,
        reason_code="NO_COMPATIBLE_MATERIALIZATION",
        reason="No existing PTSIP-compatible materialization exists.",
    )


def test_unsupported_semantics_enters_recovery_before_terminal_tool_gap() -> None:
    survivors = ("candidate:a", "candidate:b")
    record = build_resolution_record(
        surviving_candidate_ids=survivors,
        surviving_candidate_provenance=_survivor_provenance(survivors),
        recovery_assessment=_pending_recovery(),
    )

    assert record.state is ResolutionState.RECOVERY_REQUIRED
    assert record.outcome is None
    assert record.responsibility is ResolutionResponsibility.NONE


def test_authority_recovery_must_reference_candidate_provenance_requirement() -> None:
    requirement = _authority_requirement()
    assessment = _pending_recovery(requirement)

    with pytest.raises(ResolutionContractError) as exc_info:
        build_resolution_record(
            surviving_candidate_ids=("candidate:a", "candidate:b"),
            surviving_candidate_provenance={
                "candidate:a": _provenance(),
                "candidate:b": _provenance(),
            },
            recovery_assessment=assessment,
        )

    assert exc_info.value.code == "RESOLUTION_RECOVERY_REQUIREMENT_NOT_IN_PROVENANCE"


def test_authority_recovery_uses_requirement_reference_without_copying_requirement() -> None:
    requirement = _authority_requirement()
    survivors = ("candidate:a", "candidate:b")
    record = build_resolution_record(
        surviving_candidate_ids=survivors,
        surviving_candidate_provenance=_survivor_provenance(
            survivors,
            requirement,
        ),
        recovery_assessment=_pending_recovery(requirement),
    )

    assert record.state is ResolutionState.RECOVERY_REQUIRED
    assert record.authority_requirement_ids == (
        "authority-requirement:classification",
    )
    serialized = record.as_dict()
    assert "authority_requirements" not in serialized
    assert serialized["authority_requirement_ids"] == [
        "authority-requirement:classification"
    ]


def test_recovery_success_still_requires_fresh_reduction_before_terminal_classification() -> None:
    requirement = _authority_requirement()
    assessment = recover_on_next_path(
        _pending_recovery(requirement),
        recovered_capability=RecoveredCapability(
            required_capability="component.classification",
            subject="component:runtime-a",
            canonical_key="component.classification",
            canonical_value="DEVELOPMENT_TOOLING",
            closed_vocabulary=("DEVELOPMENT_TOOLING",),
            source_path=CapabilityRecoveryPath.LOSSLESS_CANONICAL_NORMALIZATION,
            authority_requirement_id=requirement["requirement_id"],
        ),
        reason_code="ALREADY_CANONICAL",
        reason="Existing machine-readable semantics are losslessly canonical.",
    )
    survivors = ("candidate:a", "candidate:b")

    record = build_resolution_record(
        surviving_candidate_ids=survivors,
        surviving_candidate_provenance=_survivor_provenance(
            survivors,
            requirement,
        ),
        recovery_assessment=assessment,
    )

    assert record.state is ResolutionState.RECOVERY_REQUIRED
    assert record.outcome is None
    assert record.recovery_assessment is not None
    assert record.recovery_assessment.recovered is True


def test_tool_capability_gap_is_terminal_only_after_recovery_exhaustion() -> None:
    requirement = _authority_requirement()
    survivors = ("candidate:a", "candidate:b")
    record = build_resolution_record(
        surviving_candidate_ids=survivors,
        elimination_record_ids=("elimination:1",),
        surviving_candidate_provenance=_survivor_provenance(
            survivors,
            requirement,
        ),
        recovery_assessment=_exhausted_recovery(requirement),
    )

    assert record.state is ResolutionState.TERMINAL
    assert record.outcome is ResolutionOutcome.TOOL_CAPABILITY_GAP
    assert record.responsibility is ResolutionResponsibility.PTSIP


def test_project_intent_enum_is_only_routing_and_requires_canonical_authority_requirement() -> None:
    with pytest.raises(ResolutionContractError) as exc_info:
        build_resolution_record(
            surviving_candidate_ids=("candidate:a", "candidate:b"),
            surviving_candidate_provenance={
                "candidate:a": _provenance(),
                "candidate:b": _provenance(),
            },
            unresolved_requirement=RuleRequirementKind.PROJECT_INTENT,
            unresolved_dimension="component.classification",
        )

    assert exc_info.value.code == "RESOLUTION_PROJECT_INTENT_WITHOUT_AUTHORITY_REQUIREMENT"


def test_owner_intent_required_is_grounded_in_candidate_provenance_authority_requirement() -> None:
    requirement = _authority_requirement()
    survivors = ("candidate:a", "candidate:b")
    record = build_resolution_record(
        surviving_candidate_ids=survivors,
        surviving_candidate_provenance=_survivor_provenance(
            survivors,
            requirement,
        ),
        unresolved_requirement=RuleRequirementKind.PROJECT_INTENT,
        unresolved_dimension="component.classification",
    )

    assert record.outcome is ResolutionOutcome.OWNER_INTENT_REQUIRED
    assert record.responsibility is ResolutionResponsibility.PROJECT
    assert record.required_input_kind is RuleRequirementKind.PROJECT_INTENT
    assert record.authority_requirement_ids == (
        "authority-requirement:classification",
    )


def test_unsatisfied_authority_requirement_cannot_silently_become_deterministic() -> None:
    requirement = _authority_requirement()

    with pytest.raises(ResolutionContractError) as exc_info:
        build_resolution_record(
            surviving_candidate_ids=("candidate:only",),
            surviving_candidate_provenance={
                "candidate:only": _provenance(requirement),
            },
        )

    assert exc_info.value.code == "RESOLUTION_UNSATISFIED_AUTHORITY_REQUIREMENT_UNROUTED"


def test_external_fact_route_rejects_simultaneous_authority_requirement() -> None:
    requirement = _authority_requirement()

    with pytest.raises(ResolutionContractError) as exc_info:
        build_resolution_record(
            surviving_candidate_ids=("candidate:only",),
            surviving_candidate_provenance={
                "candidate:only": _provenance(requirement),
            },
            unresolved_requirement=RuleRequirementKind.EXTERNAL_FACT,
            unresolved_dimension="artifact.membership",
        )

    assert exc_info.value.code == "RESOLUTION_MULTIPLE_UNRESOLVED_CHANNELS"


def test_external_fact_without_authority_requirement_remains_external_input() -> None:
    record = build_resolution_record(
        surviving_candidate_ids=("candidate:only",),
        surviving_candidate_provenance={
            "candidate:only": _provenance(),
        },
        unresolved_requirement=RuleRequirementKind.EXTERNAL_FACT,
        unresolved_dimension="artifact.membership",
    )

    assert record.outcome is ResolutionOutcome.EXTERNAL_FACT_REQUIRED
    assert record.responsibility is ResolutionResponsibility.EXTERNAL_INPUT


def test_two_or_more_survivors_do_not_automatically_mean_owner_intent_required() -> None:
    survivors = ("candidate:a", "candidate:b")
    with pytest.raises(ResolutionContractError) as exc_info:
        build_resolution_record(
            surviving_candidate_ids=survivors,
            surviving_candidate_provenance=_survivor_provenance(survivors),
        )

    assert exc_info.value.code == "RESOLUTION_STALL_REASON_REQUIRED"


def test_nonempty_survivor_set_requires_candidate_provenance() -> None:
    with pytest.raises(ResolutionContractError) as exc_info:
        build_resolution_record(
            surviving_candidate_ids=("candidate:only",),
        )

    assert exc_info.value.code == "RESOLUTION_CANDIDATE_PROVENANCE_REQUIRED"


def test_zero_and_one_survivor_terminal_cardinality_are_preserved_without_unsatisfied_requirements() -> None:
    zero = build_resolution_record(surviving_candidate_ids=())
    one = build_resolution_record(
        surviving_candidate_ids=("candidate:only",),
        surviving_candidate_provenance={
            "candidate:only": _provenance(),
        },
    )

    assert zero.outcome is ResolutionOutcome.UNSATISFIABLE
    assert one.outcome is ResolutionOutcome.DETERMINISTIC
    assert zero.responsibility is ResolutionResponsibility.NONE
    assert one.responsibility is ResolutionResponsibility.NONE


def test_coverage_gap_is_observation_not_automatic_tool_gap() -> None:
    record = build_resolution_record(
        surviving_candidate_ids=("candidate:only",),
        surviving_candidate_provenance={
            "candidate:only": _provenance(),
        },
        coverage_gap_ids=("gap:coverage:1",),
    )

    assert record.coverage_gap_ids == ("gap:coverage:1",)
    assert record.outcome is ResolutionOutcome.DETERMINISTIC


def test_resolution_record_references_s2_elimination_records_without_recreating_them() -> None:
    record = build_resolution_record(
        surviving_candidate_ids=("candidate:only",),
        surviving_candidate_provenance={
            "candidate:only": _provenance(),
        },
        elimination_record_ids=("elimination:a", "elimination:b"),
    )

    assert record.elimination_record_ids == ("elimination:a", "elimination:b")
    assert "eliminated_candidate_ids" not in record.as_dict()


def test_survivor_provenance_must_match_exact_s2_survivor_set() -> None:
    with pytest.raises(ResolutionContractError) as exc_info:
        build_resolution_record(
            surviving_candidate_ids=("candidate:a",),
            surviving_candidate_provenance={
                "candidate:a": _provenance(),
                "candidate:stale": _provenance(),
            },
        )

    assert exc_info.value.code == "RESOLUTION_CANDIDATE_PROVENANCE_BINDING_MISMATCH"


def test_same_requirement_id_with_conflicting_semantics_fails_closed() -> None:
    first = _authority_requirement()
    second = {
        **_authority_requirement(),
        "required_effects": ["require_project_owner_decision"],
    }

    with pytest.raises(ResolutionContractError) as exc_info:
        build_resolution_record(
            surviving_candidate_ids=("candidate:a", "candidate:b"),
            surviving_candidate_provenance={
                "candidate:a": _provenance(first),
                "candidate:b": _provenance(second),
            },
            unresolved_requirement=RuleRequirementKind.PROJECT_INTENT,
            unresolved_dimension="component.classification",
        )

    assert exc_info.value.code == "RESOLUTION_AUTHORITY_REQUIREMENT_ID_CONFLICT"


def test_resolution_record_id_is_stable_for_same_canonical_references() -> None:
    kwargs = dict(
        surviving_candidate_ids=("candidate:only",),
        surviving_candidate_provenance={
            "candidate:only": _provenance(),
        },
        elimination_record_ids=("elimination:a",),
        coverage_gap_ids=("gap:coverage:1",),
    )
    first = build_resolution_record(**kwargs)
    second = build_resolution_record(**kwargs)

    assert first.id == second.id
    assert first.as_dict() == second.as_dict()
