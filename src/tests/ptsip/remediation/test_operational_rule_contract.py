from __future__ import annotations

import pytest

from ptsip.remediation import (
    OperationalRule,
    OperationalRuleBinding,
    OperationalRuleCapabilityDeclaration,
    OperationalRuleContractError,
    OperationalRuleEvaluation,
    RuleAutomationLevel,
    RuleCapabilityState,
    RuleEvaluationStatus,
    RuleRequirementKind,
)
from ptsip.remediation.domain import NormativeConstraint
from ptsip.specification_binding import current_target_specification_binding


def _capability() -> OperationalRuleCapabilityDeclaration:
    return OperationalRuleCapabilityDeclaration(
        supported_remediation_families=("PACKAGE_ISOLATION",),
        candidate_generation=RuleCapabilityState.SUPPORTED,
        postcondition_verification=RuleCapabilityState.SUPPORTED,
        possible_requirements=(RuleRequirementKind.PROJECT_INTENT, RuleRequirementKind.EXTERNAL_FACT),
        automation_level=RuleAutomationLevel.SEMANTIC_REMEDIATION,
    )


def test_operational_rule_binding_requires_explicit_specification_binding() -> None:
    with pytest.raises(OperationalRuleContractError) as exc_info:
        OperationalRuleBinding(  # type: ignore[arg-type]
            rule_id="PTSIP-PKG-001",
            specification="0.3.7-draft",
            capability=_capability(),
        )

    assert exc_info.value.code == "OPERATIONAL_RULE_SPECIFICATION_BINDING_REQUIRED"


def test_capability_declaration_is_non_authoritative_and_explicit() -> None:
    capability = _capability()

    assert capability.as_dict() == {
        "supported_remediation_families": ["PACKAGE_ISOLATION"],
        "candidate_generation": "SUPPORTED",
        "postcondition_verification": "SUPPORTED",
        "possible_requirements": ["PROJECT_INTENT", "EXTERNAL_FACT"],
        "automation_level": "SEMANTIC_REMEDIATION",
        "authoritative": False,
    }


def test_candidate_generation_and_supported_families_must_be_declared_together() -> None:
    with pytest.raises(OperationalRuleContractError) as exc_info:
        OperationalRuleCapabilityDeclaration(
            candidate_generation=RuleCapabilityState.SUPPORTED,
        )

    assert exc_info.value.code == "OPERATIONAL_RULE_REMEDIATION_CAPABILITY_INCONSISTENT"


def test_semantic_remediation_level_requires_candidate_generation_capability() -> None:
    with pytest.raises(OperationalRuleContractError) as exc_info:
        OperationalRuleCapabilityDeclaration(
            automation_level=RuleAutomationLevel.SEMANTIC_REMEDIATION,
        )

    assert exc_info.value.code == "OPERATIONAL_RULE_AUTOMATION_CAPABILITY_INCONSISTENT"


def test_rule_evaluation_is_bound_to_same_normative_rule_identity() -> None:
    specification = current_target_specification_binding()
    constraint = NormativeConstraint(
        rule_id="PTSIP-PKG-001",
        specification=specification,
        subject="artifact:wheel",
        requirement="exclude explicitly non-product implementation",
    )

    result = OperationalRuleEvaluation(
        rule_id="PTSIP-PKG-001",
        status=RuleEvaluationStatus.VIOLATION,
        fact_ids=("fact:package-leak",),
        constraint=constraint,
    )

    assert result.rule_id == constraint.rule_id
    assert result.as_dict()["status"] == "VIOLATION"


def test_rule_evaluation_rejects_constraint_from_another_rule() -> None:
    constraint = NormativeConstraint(
        rule_id="PTSIP-DEP-001",
        specification=current_target_specification_binding(),
        subject="component:product",
        requirement="deny runtime dependency on tooling",
    )

    with pytest.raises(OperationalRuleContractError) as exc_info:
        OperationalRuleEvaluation(
            rule_id="PTSIP-PKG-001",
            status=RuleEvaluationStatus.VIOLATION,
            constraint=constraint,
        )

    assert exc_info.value.code == "OPERATIONAL_RULE_CONSTRAINT_RULE_ID_MISMATCH"


def test_not_applicable_evaluation_cannot_claim_applicable_constraint() -> None:
    constraint = NormativeConstraint(
        rule_id="PTSIP-PKG-001",
        specification=current_target_specification_binding(),
        subject="artifact:wheel",
        requirement="exclude explicitly non-product implementation",
    )

    with pytest.raises(OperationalRuleContractError) as exc_info:
        OperationalRuleEvaluation(
            rule_id="PTSIP-PKG-001",
            status=RuleEvaluationStatus.NOT_APPLICABLE,
            constraint=constraint,
        )

    assert exc_info.value.code == "OPERATIONAL_RULE_NOT_APPLICABLE_CONSTRAINT_CONFLICT"


def test_operational_rule_protocol_exposes_no_mutation_or_authority_creation_surface() -> None:
    assert hasattr(OperationalRule, "applies")
    assert hasattr(OperationalRule, "evaluate")
    assert hasattr(OperationalRule, "propose_solutions")
    assert hasattr(OperationalRule, "verify")
    assert not hasattr(OperationalRule, "apply")
    assert not hasattr(OperationalRule, "authorize")
    assert not hasattr(OperationalRule, "create_authority")
    assert "tool_version" not in OperationalRuleBinding.__dataclass_fields__
