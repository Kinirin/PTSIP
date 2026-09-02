from __future__ import annotations

from dataclasses import dataclass
from enum import StrEnum
from typing import Protocol, runtime_checkable

from ..domain import (
    DerivedFact,
    NormativeConstraint,
    PostconditionResult,
    RemediationContext,
    SemanticCandidate,
    SemanticRemediationPlan,
)
from ...specification_binding import SpecificationBinding


class OperationalRuleContractError(ValueError):
    """Stable fail-closed error for malformed OperationalRule contracts."""

    def __init__(self, code: str, message: str, value: object = None) -> None:
        super().__init__(message)
        self.code = code
        self.value = value


class RuleCapabilityState(StrEnum):
    SUPPORTED = "SUPPORTED"
    UNSUPPORTED = "UNSUPPORTED"


class RuleRequirementKind(StrEnum):
    PROJECT_INTENT = "PROJECT_INTENT"
    EXTERNAL_FACT = "EXTERNAL_FACT"


class RuleAutomationLevel(StrEnum):
    EVALUATION_ONLY = "EVALUATION_ONLY"
    SEMANTIC_REMEDIATION = "SEMANTIC_REMEDIATION"


class RuleEvaluationStatus(StrEnum):
    NOT_APPLICABLE = "NOT_APPLICABLE"
    CONFORMANT = "CONFORMANT"
    VIOLATION = "VIOLATION"
    UNRESOLVED = "UNRESOLVED"


def _require_text(name: str, value: object) -> str:
    if not isinstance(value, str) or not value or value != value.strip():
        raise OperationalRuleContractError(
            "OPERATIONAL_RULE_TEXT_INVALID",
            f"{name} must be a non-empty canonical string without surrounding whitespace.",
            value,
        )
    return value


def _require_unique_text_tuple(name: str, values: tuple[str, ...]) -> tuple[str, ...]:
    if not isinstance(values, tuple):
        raise OperationalRuleContractError(
            "OPERATIONAL_RULE_TUPLE_REQUIRED",
            f"{name} must be a tuple of canonical strings.",
            values,
        )
    normalized = tuple(_require_text(name, item) for item in values)
    if len(set(normalized)) != len(normalized):
        raise OperationalRuleContractError(
            "OPERATIONAL_RULE_DUPLICATE_VALUE",
            f"{name} must not contain duplicate values.",
            values,
        )
    return normalized


@dataclass(frozen=True)
class OperationalRuleCapabilityDeclaration:
    """Non-authoritative declaration of Tool support for one bound Specification rule."""

    supported_remediation_families: tuple[str, ...] = ()
    candidate_generation: RuleCapabilityState = RuleCapabilityState.UNSUPPORTED
    postcondition_verification: RuleCapabilityState = RuleCapabilityState.UNSUPPORTED
    possible_requirements: tuple[RuleRequirementKind, ...] = ()
    automation_level: RuleAutomationLevel = RuleAutomationLevel.EVALUATION_ONLY

    def __post_init__(self) -> None:
        families = _require_unique_text_tuple(
            "supported_remediation_families",
            self.supported_remediation_families,
        )
        if not isinstance(self.candidate_generation, RuleCapabilityState):
            raise OperationalRuleContractError(
                "OPERATIONAL_RULE_CANDIDATE_CAPABILITY_INVALID",
                "candidate_generation must be an explicit RuleCapabilityState.",
                self.candidate_generation,
            )
        if not isinstance(self.postcondition_verification, RuleCapabilityState):
            raise OperationalRuleContractError(
                "OPERATIONAL_RULE_VERIFICATION_CAPABILITY_INVALID",
                "postcondition_verification must be an explicit RuleCapabilityState.",
                self.postcondition_verification,
            )
        if not isinstance(self.possible_requirements, tuple) or any(
            not isinstance(item, RuleRequirementKind) for item in self.possible_requirements
        ):
            raise OperationalRuleContractError(
                "OPERATIONAL_RULE_REQUIREMENTS_INVALID",
                "possible_requirements must contain only explicit RuleRequirementKind values.",
                self.possible_requirements,
            )
        if len(set(self.possible_requirements)) != len(self.possible_requirements):
            raise OperationalRuleContractError(
                "OPERATIONAL_RULE_REQUIREMENTS_DUPLICATE",
                "possible_requirements must not contain duplicate values.",
                self.possible_requirements,
            )
        if not isinstance(self.automation_level, RuleAutomationLevel):
            raise OperationalRuleContractError(
                "OPERATIONAL_RULE_AUTOMATION_LEVEL_INVALID",
                "automation_level must be an explicit RuleAutomationLevel.",
                self.automation_level,
            )

        candidate_supported = self.candidate_generation is RuleCapabilityState.SUPPORTED
        if candidate_supported != bool(families):
            raise OperationalRuleContractError(
                "OPERATIONAL_RULE_REMEDIATION_CAPABILITY_INCONSISTENT",
                "Supported remediation families and candidate-generation capability must be declared together.",
                self,
            )
        if (
            self.automation_level is RuleAutomationLevel.SEMANTIC_REMEDIATION
            and not candidate_supported
        ):
            raise OperationalRuleContractError(
                "OPERATIONAL_RULE_AUTOMATION_CAPABILITY_INCONSISTENT",
                "SEMANTIC_REMEDIATION requires declared candidate-generation capability.",
                self,
            )

    def as_dict(self) -> dict[str, object]:
        return {
            "supported_remediation_families": list(self.supported_remediation_families),
            "candidate_generation": self.candidate_generation.value,
            "postcondition_verification": self.postcondition_verification.value,
            "possible_requirements": [item.value for item in self.possible_requirements],
            "automation_level": self.automation_level.value,
            "authoritative": False,
        }


@dataclass(frozen=True)
class OperationalRuleBinding:
    """Exact Specification rule identity plus non-authoritative Tool capability."""

    rule_id: str
    specification: SpecificationBinding
    capability: OperationalRuleCapabilityDeclaration

    def __post_init__(self) -> None:
        _require_text("rule_id", self.rule_id)
        if not isinstance(self.specification, SpecificationBinding):
            raise OperationalRuleContractError(
                "OPERATIONAL_RULE_SPECIFICATION_BINDING_REQUIRED",
                "OperationalRuleBinding requires an explicit SpecificationBinding.",
                self.specification,
            )
        if not isinstance(self.capability, OperationalRuleCapabilityDeclaration):
            raise OperationalRuleContractError(
                "OPERATIONAL_RULE_CAPABILITY_DECLARATION_REQUIRED",
                "OperationalRuleBinding requires an explicit capability declaration.",
                self.capability,
            )

    def as_dict(self) -> dict[str, object]:
        return {
            "rule_id": self.rule_id,
            "specification": self.specification.as_dict(),
            "capability": self.capability.as_dict(),
        }


@dataclass(frozen=True)
class OperationalRuleEvaluation:
    """Rule-local evaluation result; it is not a remediation or authorization decision."""

    rule_id: str
    status: RuleEvaluationStatus
    fact_ids: tuple[str, ...] = ()
    constraint: NormativeConstraint | None = None
    detail: str | None = None

    def __post_init__(self) -> None:
        _require_text("rule_id", self.rule_id)
        if not isinstance(self.status, RuleEvaluationStatus):
            raise OperationalRuleContractError(
                "OPERATIONAL_RULE_EVALUATION_STATUS_INVALID",
                "status must be an explicit RuleEvaluationStatus.",
                self.status,
            )
        _require_unique_text_tuple("fact_ids", self.fact_ids)
        if self.constraint is not None:
            if not isinstance(self.constraint, NormativeConstraint):
                raise OperationalRuleContractError(
                    "OPERATIONAL_RULE_CONSTRAINT_INVALID",
                    "constraint must be a NormativeConstraint when present.",
                    self.constraint,
                )
            if self.constraint.rule_id != self.rule_id:
                raise OperationalRuleContractError(
                    "OPERATIONAL_RULE_CONSTRAINT_RULE_ID_MISMATCH",
                    "Evaluation rule_id must match its NormativeConstraint rule_id.",
                    self.constraint.rule_id,
                )
        if self.status in (RuleEvaluationStatus.CONFORMANT, RuleEvaluationStatus.VIOLATION):
            if self.constraint is None:
                raise OperationalRuleContractError(
                    "OPERATIONAL_RULE_CONSTRAINT_REQUIRED",
                    "A conformant or violating evaluation must identify its bound NormativeConstraint.",
                    self,
                )
        if self.status is RuleEvaluationStatus.NOT_APPLICABLE and self.constraint is not None:
            raise OperationalRuleContractError(
                "OPERATIONAL_RULE_NOT_APPLICABLE_CONSTRAINT_CONFLICT",
                "A NOT_APPLICABLE evaluation must not claim an applicable NormativeConstraint.",
                self.constraint,
            )
        if self.detail is not None:
            _require_text("detail", self.detail)

    def as_dict(self) -> dict[str, object]:
        return {
            "rule_id": self.rule_id,
            "status": self.status.value,
            "fact_ids": list(self.fact_ids),
            "constraint": self.constraint.as_dict() if self.constraint else None,
            "detail": self.detail,
        }


@runtime_checkable
class OperationalRule(Protocol):
    """Executable bridge for a single explicitly bound Specification rule."""

    @property
    def binding(self) -> OperationalRuleBinding:
        ...

    def applies(
        self,
        context: RemediationContext,
        facts: tuple[DerivedFact, ...],
    ) -> bool:
        ...

    def evaluate(
        self,
        context: RemediationContext,
        facts: tuple[DerivedFact, ...],
    ) -> OperationalRuleEvaluation:
        ...

    def propose_solutions(
        self,
        context: RemediationContext,
        evaluation: OperationalRuleEvaluation,
        facts: tuple[DerivedFact, ...],
    ) -> tuple[SemanticCandidate, ...]:
        ...

    def verify(
        self,
        context: RemediationContext,
        semantic_plan: SemanticRemediationPlan,
    ) -> PostconditionResult:
        ...
