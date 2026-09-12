from __future__ import annotations

from dataclasses import dataclass
from enum import StrEnum
from typing import Mapping

from ..domain import ResolutionOutcome
from ..rules.contract import RuleRequirementKind
from ...evidence.contract import stable_digest
from .recovery import (
    CapabilityRecoveryAssessment,
    AuthorityRequirementView,
    authority_requirement_view,
)


class ResolutionContractError(ValueError):
    """Stable fail-closed error for S3 resolution-boundary contracts."""

    def __init__(self, code: str, message: str, value: object = None) -> None:
        super().__init__(message)
        self.code = code
        self.value = value


class ResolutionState(StrEnum):
    RECOVERY_REQUIRED = "RECOVERY_REQUIRED"
    TERMINAL = "TERMINAL"


class ResolutionResponsibility(StrEnum):
    PROJECT = "PROJECT"
    EXTERNAL_INPUT = "EXTERNAL_INPUT"
    PTSIP = "PTSIP"
    NONE = "NONE"


def _require_text(name: str, value: object) -> str:
    if not isinstance(value, str) or not value or value != value.strip():
        raise ResolutionContractError(
            "RESOLUTION_TEXT_INVALID",
            f"{name} must be a non-empty canonical string without surrounding whitespace.",
            value,
        )
    return value


def _require_ids(name: str, values: tuple[str, ...]) -> tuple[str, ...]:
    if not isinstance(values, tuple):
        raise ResolutionContractError(
            "RESOLUTION_IDS_TYPE",
            f"{name} must be a tuple of canonical identifiers.",
            values,
        )
    normalized = tuple(_require_text(name, item) for item in values)
    if len(set(normalized)) != len(normalized):
        raise ResolutionContractError(
            "RESOLUTION_IDS_DUPLICATE",
            f"{name} must not contain duplicate identifiers.",
            values,
        )
    return normalized


def _provenance_authority_requirements(
    *,
    surviving_candidate_ids: tuple[str, ...],
    surviving_candidate_provenance: Mapping[str, Mapping[str, object]] | None,
) -> tuple[AuthorityRequirementView, ...]:
    if surviving_candidate_provenance is None:
        if surviving_candidate_ids:
            raise ResolutionContractError(
                "RESOLUTION_CANDIDATE_PROVENANCE_REQUIRED",
                "Every current S2 survivor must carry its canonical CandidateProvenance into S3.",
                surviving_candidate_ids,
            )
        return ()
    if not isinstance(surviving_candidate_provenance, Mapping):
        raise ResolutionContractError(
            "RESOLUTION_CANDIDATE_PROVENANCE_INVALID",
            "surviving_candidate_provenance must be a mapping keyed by survivor identity.",
            surviving_candidate_provenance,
        )
    expected = set(surviving_candidate_ids)
    actual = set(surviving_candidate_provenance)
    if actual != expected:
        raise ResolutionContractError(
            "RESOLUTION_CANDIDATE_PROVENANCE_BINDING_MISMATCH",
            "Survivor provenance must be supplied for exactly the current S2 survivor set.",
            {
                "expected": sorted(expected),
                "actual": sorted(actual),
            },
        )

    by_id: dict[str, AuthorityRequirementView] = {}
    for candidate_id in surviving_candidate_ids:
        provenance = surviving_candidate_provenance[candidate_id]
        if not isinstance(provenance, Mapping):
            raise ResolutionContractError(
                "RESOLUTION_CANDIDATE_PROVENANCE_INVALID",
                "Each survivor provenance value must be a CandidateProvenance mapping.",
                candidate_id,
            )
        raw = provenance.get("authority_requirements", ())
        if isinstance(raw, list):
            raw = tuple(raw)
        if not isinstance(raw, tuple):
            raise ResolutionContractError(
                "RESOLUTION_AUTHORITY_REQUIREMENTS_INVALID",
                "CandidateProvenance.authority_requirements must be an ordered collection.",
                raw,
            )
        for item in raw:
            try:
                requirement = authority_requirement_view(item)
            except ValueError as exc:
                raise ResolutionContractError(
                    "RESOLUTION_AUTHORITY_REQUIREMENT_INVALID",
                    "Survivor provenance contains an invalid authority requirement.",
                    item,
                ) from exc
            previous = by_id.get(requirement.requirement_id)
            if previous is not None and (
                previous.subject_binding_digest != requirement.subject_binding_digest
                or previous.required_effects != requirement.required_effects
            ):
                raise ResolutionContractError(
                    "RESOLUTION_AUTHORITY_REQUIREMENT_ID_CONFLICT",
                    "One canonical requirement identity cannot carry conflicting semantics.",
                    requirement.requirement_id,
                )
            by_id[requirement.requirement_id] = requirement

    return tuple(by_id[key] for key in sorted(by_id))


@dataclass(frozen=True)
class ResolutionRecord:
    """One machine-inspectable record for the S3 resolution boundary.

    authority_requirement_ids are references back to the upstream
    CandidateProvenance SSOT; this record does not copy or redefine them.
    """

    id: str
    state: ResolutionState
    surviving_candidate_ids: tuple[str, ...]
    elimination_record_ids: tuple[str, ...] = ()
    coverage_gap_ids: tuple[str, ...] = ()
    outcome: ResolutionOutcome | None = None
    responsibility: ResolutionResponsibility = ResolutionResponsibility.NONE
    unresolved_dimension: str | None = None
    required_input_kind: RuleRequirementKind | None = None
    authority_requirement_ids: tuple[str, ...] = ()
    recovery_assessment: CapabilityRecoveryAssessment | None = None

    def __post_init__(self) -> None:
        _require_text("id", self.id)
        if not isinstance(self.state, ResolutionState):
            raise ResolutionContractError(
                "RESOLUTION_STATE_INVALID",
                "state must be an explicit ResolutionState.",
                self.state,
            )
        _require_ids("surviving_candidate_ids", self.surviving_candidate_ids)
        _require_ids("elimination_record_ids", self.elimination_record_ids)
        _require_ids("coverage_gap_ids", self.coverage_gap_ids)
        authority_requirement_ids = _require_ids(
            "authority_requirement_ids",
            self.authority_requirement_ids,
        )
        if not isinstance(self.responsibility, ResolutionResponsibility):
            raise ResolutionContractError(
                "RESOLUTION_RESPONSIBILITY_INVALID",
                "responsibility must be an explicit ResolutionResponsibility.",
                self.responsibility,
            )
        if self.unresolved_dimension is not None:
            _require_text("unresolved_dimension", self.unresolved_dimension)
        if self.required_input_kind is not None and not isinstance(
            self.required_input_kind,
            RuleRequirementKind,
        ):
            raise ResolutionContractError(
                "RESOLUTION_REQUIREMENT_INVALID",
                "required_input_kind must be an explicit RuleRequirementKind.",
                self.required_input_kind,
            )
        if self.recovery_assessment is not None and not isinstance(
            self.recovery_assessment,
            CapabilityRecoveryAssessment,
        ):
            raise ResolutionContractError(
                "RESOLUTION_RECOVERY_ASSESSMENT_INVALID",
                "recovery_assessment must be CapabilityRecoveryAssessment when present.",
                self.recovery_assessment,
            )

        bound_requirement_id = (
            self.recovery_assessment.authority_requirement_id
            if self.recovery_assessment is not None
            else None
        )
        if bound_requirement_id is not None and bound_requirement_id not in set(
            authority_requirement_ids
        ):
            raise ResolutionContractError(
                "RESOLUTION_RECOVERY_REQUIREMENT_NOT_IN_PROVENANCE",
                "Authority recovery must reference a requirement present in CandidateProvenance.",
                bound_requirement_id,
            )

        if self.state is ResolutionState.RECOVERY_REQUIRED:
            if self.outcome is not None:
                raise ResolutionContractError(
                    "RESOLUTION_RECOVERY_OUTCOME_CONFLICT",
                    "RECOVERY_REQUIRED is a state and must not carry a terminal outcome.",
                    self.outcome,
                )
            if self.recovery_assessment is None or self.recovery_assessment.exhausted:
                raise ResolutionContractError(
                    "RESOLUTION_RECOVERY_ASSESSMENT_REQUIRED",
                    "RECOVERY_REQUIRED requires a non-exhausted recovery assessment.",
                    self.recovery_assessment,
                )
            if self.responsibility is not ResolutionResponsibility.NONE:
                raise ResolutionContractError(
                    "RESOLUTION_RECOVERY_RESPONSIBILITY_CONFLICT",
                    "Recovery is not yet a terminal responsibility assignment.",
                    self.responsibility,
                )
            if self.required_input_kind is not None:
                raise ResolutionContractError(
                    "RESOLUTION_RECOVERY_INPUT_CONFLICT",
                    "Safe recovery must be evaluated before requesting project or external input.",
                    self.required_input_kind,
                )
            return

        if not isinstance(self.outcome, ResolutionOutcome):
            raise ResolutionContractError(
                "RESOLUTION_TERMINAL_OUTCOME_REQUIRED",
                "TERMINAL state requires one ResolutionOutcome.",
                self.outcome,
            )

        if self.outcome is ResolutionOutcome.TOOL_CAPABILITY_GAP:
            if self.recovery_assessment is None or not self.recovery_assessment.exhausted:
                raise ResolutionContractError(
                    "RESOLUTION_TOOL_GAP_RECOVERY_NOT_EXHAUSTED",
                    "TOOL_CAPABILITY_GAP is terminal only after all recovery paths were rejected.",
                    self.recovery_assessment,
                )
            if self.responsibility is not ResolutionResponsibility.PTSIP:
                raise ResolutionContractError(
                    "RESOLUTION_TOOL_GAP_RESPONSIBILITY_INVALID",
                    "Terminal tool capability gaps are PTSIP responsibility.",
                    self.responsibility,
                )
            if self.required_input_kind is not None:
                raise ResolutionContractError(
                    "RESOLUTION_TOOL_GAP_INPUT_CONFLICT",
                    "TOOL_CAPABILITY_GAP must not be converted into a project or external input request.",
                    self.required_input_kind,
                )
            return

        if self.recovery_assessment is not None:
            raise ResolutionContractError(
                "RESOLUTION_TERMINAL_RECOVERY_CONFLICT",
                "Non-tool terminal outcomes must not carry a capability recovery assessment.",
                self.recovery_assessment,
            )

        if self.outcome is ResolutionOutcome.OWNER_INTENT_REQUIRED:
            if self.responsibility is not ResolutionResponsibility.PROJECT:
                raise ResolutionContractError(
                    "RESOLUTION_OWNER_RESPONSIBILITY_INVALID",
                    "Missing or ambiguous project intent is PROJECT responsibility.",
                    self.responsibility,
                )
            if self.required_input_kind is not RuleRequirementKind.PROJECT_INTENT:
                raise ResolutionContractError(
                    "RESOLUTION_OWNER_INPUT_INVALID",
                    "OWNER_INTENT_REQUIRED uses PROJECT_INTENT only as a routing classification.",
                    self.required_input_kind,
                )
            if not authority_requirement_ids:
                raise ResolutionContractError(
                    "RESOLUTION_OWNER_AUTHORITY_REQUIREMENT_REQUIRED",
                    "OWNER_INTENT_REQUIRED requires canonical unsatisfied authority requirement references.",
                    authority_requirement_ids,
                )
            if self.unresolved_dimension is None:
                raise ResolutionContractError(
                    "RESOLUTION_UNRESOLVED_DIMENSION_REQUIRED",
                    "OWNER_INTENT_REQUIRED must identify the unresolved semantic dimension.",
                    self.unresolved_dimension,
                )
            return

        if self.outcome is ResolutionOutcome.EXTERNAL_FACT_REQUIRED:
            if self.responsibility is not ResolutionResponsibility.EXTERNAL_INPUT:
                raise ResolutionContractError(
                    "RESOLUTION_EXTERNAL_RESPONSIBILITY_INVALID",
                    "Missing externally knowable fact is EXTERNAL_INPUT responsibility.",
                    self.responsibility,
                )
            if self.required_input_kind is not RuleRequirementKind.EXTERNAL_FACT:
                raise ResolutionContractError(
                    "RESOLUTION_EXTERNAL_INPUT_INVALID",
                    "EXTERNAL_FACT_REQUIRED must identify EXTERNAL_FACT as the unresolved input.",
                    self.required_input_kind,
                )
            if authority_requirement_ids:
                raise ResolutionContractError(
                    "RESOLUTION_EXTERNAL_AUTHORITY_CONFLICT",
                    "A resolution record cannot silently collapse simultaneous authority and external-input requirements.",
                    authority_requirement_ids,
                )
            if self.unresolved_dimension is None:
                raise ResolutionContractError(
                    "RESOLUTION_UNRESOLVED_DIMENSION_REQUIRED",
                    "EXTERNAL_FACT_REQUIRED must identify the unresolved semantic dimension.",
                    self.unresolved_dimension,
                )
            return

        if authority_requirement_ids:
            raise ResolutionContractError(
                "RESOLUTION_TERMINAL_AUTHORITY_REQUIREMENT_CONFLICT",
                "DETERMINISTIC or UNSATISFIABLE cannot retain unsatisfied canonical authority requirements.",
                authority_requirement_ids,
            )
        if self.responsibility is not ResolutionResponsibility.NONE:
            raise ResolutionContractError(
                "RESOLUTION_TERMINAL_RESPONSIBILITY_INVALID",
                "DETERMINISTIC and UNSATISFIABLE do not assign an action owner.",
                self.responsibility,
            )
        if self.required_input_kind is not None or self.unresolved_dimension is not None:
            raise ResolutionContractError(
                "RESOLUTION_TERMINAL_INPUT_CONFLICT",
                "DETERMINISTIC and UNSATISFIABLE must not request additional input.",
                (self.required_input_kind, self.unresolved_dimension),
            )

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "state": self.state.value,
            "surviving_candidate_ids": list(self.surviving_candidate_ids),
            "elimination_record_ids": list(self.elimination_record_ids),
            "coverage_gap_ids": list(self.coverage_gap_ids),
            "outcome": self.outcome.value if self.outcome else None,
            "responsibility": self.responsibility.value,
            "unresolved_dimension": self.unresolved_dimension,
            "required_input_kind": (
                self.required_input_kind.value if self.required_input_kind else None
            ),
            "authority_requirement_ids": list(self.authority_requirement_ids),
            "recovery_assessment": (
                self.recovery_assessment.as_dict()
                if self.recovery_assessment
                else None
            ),
        }


def _record_id(payload: dict[str, object]) -> str:
    return f"resolution:{stable_digest(payload)[:24]}"


def build_resolution_record(
    *,
    surviving_candidate_ids: tuple[str, ...],
    elimination_record_ids: tuple[str, ...] = (),
    coverage_gap_ids: tuple[str, ...] = (),
    surviving_candidate_provenance: (
        Mapping[str, Mapping[str, object]] | None
    ) = None,
    unresolved_requirement: RuleRequirementKind | None = None,
    unresolved_dimension: str | None = None,
    recovery_assessment: CapabilityRecoveryAssessment | None = None,
) -> ResolutionRecord:
    """Classify why S2 can no longer reduce without creating a second S1 SSOT."""

    survivors = _require_ids("surviving_candidate_ids", surviving_candidate_ids)
    eliminations = _require_ids("elimination_record_ids", elimination_record_ids)
    gaps = _require_ids("coverage_gap_ids", coverage_gap_ids)
    authority_requirements = _provenance_authority_requirements(
        surviving_candidate_ids=survivors,
        surviving_candidate_provenance=surviving_candidate_provenance,
    )
    authority_requirement_ids = tuple(
        item.requirement_id for item in authority_requirements
    )

    if unresolved_requirement is not None and not isinstance(
        unresolved_requirement,
        RuleRequirementKind,
    ):
        raise ResolutionContractError(
            "RESOLUTION_REQUIREMENT_INVALID",
            "unresolved_requirement must be RuleRequirementKind when present.",
            unresolved_requirement,
        )
    if unresolved_requirement is not None and recovery_assessment is not None:
        raise ResolutionContractError(
            "RESOLUTION_RECOVERY_INPUT_AMBIGUOUS",
            "Capability recovery must be completed before assigning PROJECT or EXTERNAL_INPUT responsibility.",
            (unresolved_requirement, recovery_assessment),
        )

    if recovery_assessment is not None:
        if not isinstance(recovery_assessment, CapabilityRecoveryAssessment):
            raise ResolutionContractError(
                "RESOLUTION_RECOVERY_ASSESSMENT_INVALID",
                "recovery_assessment must be CapabilityRecoveryAssessment.",
                recovery_assessment,
            )
        if (
            recovery_assessment.authority_requirement_id is not None
            and recovery_assessment.authority_requirement_id
            not in set(authority_requirement_ids)
        ):
            raise ResolutionContractError(
                "RESOLUTION_RECOVERY_REQUIREMENT_NOT_IN_PROVENANCE",
                "Authority recovery must remain bound to CandidateProvenance.authority_requirements.",
                recovery_assessment.authority_requirement_id,
            )

        if recovery_assessment.exhausted:
            state = ResolutionState.TERMINAL
            outcome = ResolutionOutcome.TOOL_CAPABILITY_GAP
            responsibility = ResolutionResponsibility.PTSIP
        else:
            state = ResolutionState.RECOVERY_REQUIRED
            outcome = None
            responsibility = ResolutionResponsibility.NONE
        required_input_kind = None
        unresolved_dimension = None
    elif unresolved_requirement is RuleRequirementKind.PROJECT_INTENT:
        if unresolved_dimension is None:
            raise ResolutionContractError(
                "RESOLUTION_UNRESOLVED_DIMENSION_REQUIRED",
                "Project-intent routing requires an explicit unresolved semantic dimension.",
                unresolved_dimension,
            )
        if not authority_requirement_ids:
            raise ResolutionContractError(
                "RESOLUTION_PROJECT_INTENT_WITHOUT_AUTHORITY_REQUIREMENT",
                "PROJECT_INTENT is only a routing classification and cannot replace CandidateProvenance.authority_requirements.",
                survivors,
            )
        state = ResolutionState.TERMINAL
        outcome = ResolutionOutcome.OWNER_INTENT_REQUIRED
        responsibility = ResolutionResponsibility.PROJECT
        required_input_kind = RuleRequirementKind.PROJECT_INTENT
    elif unresolved_requirement is RuleRequirementKind.EXTERNAL_FACT:
        if unresolved_dimension is None:
            raise ResolutionContractError(
                "RESOLUTION_UNRESOLVED_DIMENSION_REQUIRED",
                "External-input responsibility requires an explicit unresolved semantic dimension.",
                unresolved_dimension,
            )
        if authority_requirement_ids:
            raise ResolutionContractError(
                "RESOLUTION_MULTIPLE_UNRESOLVED_CHANNELS",
                "External fact routing cannot silently consume unresolved authority requirements.",
                authority_requirement_ids,
            )
        state = ResolutionState.TERMINAL
        outcome = ResolutionOutcome.EXTERNAL_FACT_REQUIRED
        responsibility = ResolutionResponsibility.EXTERNAL_INPUT
        required_input_kind = RuleRequirementKind.EXTERNAL_FACT
    elif authority_requirement_ids:
        raise ResolutionContractError(
            "RESOLUTION_UNSATISFIED_AUTHORITY_REQUIREMENT_UNROUTED",
            "Unsatisfied canonical authority requirements require explicit recovery or PROJECT_INTENT routing.",
            authority_requirement_ids,
        )
    elif len(survivors) == 0:
        state = ResolutionState.TERMINAL
        outcome = ResolutionOutcome.UNSATISFIABLE
        responsibility = ResolutionResponsibility.NONE
        required_input_kind = None
        unresolved_dimension = None
    elif len(survivors) == 1:
        state = ResolutionState.TERMINAL
        outcome = ResolutionOutcome.DETERMINISTIC
        responsibility = ResolutionResponsibility.NONE
        required_input_kind = None
        unresolved_dimension = None
    else:
        raise ResolutionContractError(
            "RESOLUTION_STALL_REASON_REQUIRED",
            "Two or more survivors do not by themselves prove missing project intent; the unresolved boundary must be explicit.",
            survivors,
        )

    payload: dict[str, object] = {
        "state": state.value,
        "surviving_candidate_ids": list(survivors),
        "elimination_record_ids": list(eliminations),
        "coverage_gap_ids": list(gaps),
        "outcome": outcome.value if outcome else None,
        "responsibility": responsibility.value,
        "unresolved_dimension": unresolved_dimension,
        "required_input_kind": (
            required_input_kind.value if required_input_kind else None
        ),
        "authority_requirement_ids": list(authority_requirement_ids),
        "recovery_assessment": (
            recovery_assessment.as_dict() if recovery_assessment else None
        ),
    }
    return ResolutionRecord(
        id=_record_id(payload),
        state=state,
        surviving_candidate_ids=survivors,
        elimination_record_ids=eliminations,
        coverage_gap_ids=gaps,
        outcome=outcome,
        responsibility=responsibility,
        unresolved_dimension=unresolved_dimension,
        required_input_kind=required_input_kind,
        authority_requirement_ids=authority_requirement_ids,
        recovery_assessment=recovery_assessment,
    )
