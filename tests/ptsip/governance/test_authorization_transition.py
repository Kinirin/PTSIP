from __future__ import annotations

from pathlib import Path

from ptsip.governance import AuthorizationState, AuthorizationTransitionEvaluator


ROOT = Path(__file__).resolve().parents[3]


def test_current_project_authority_runtime_scopes_are_deterministically_authorized() -> None:
    evaluator = AuthorizationTransitionEvaluator(ROOT)
    readiness = evaluator.derive_project_authority_runtime_readiness()
    assert readiness == {
        "AUTHORITY_SCHEMA_REGISTRY_VALID": True,
        "AUTHORITY_ROLE_REGISTRY_VALID": True,
        "AUTHORITY_SUBJECT_REGISTRY_VALID": True,
        "CURRENT_ADR_CORPUS_VALID": True,
        "ROLE_EFFECT_VOCABULARY_FROZEN": True,
        "SUBJECT_BINDING_CONTRACT_FROZEN": True,
        "PROJECT_AUTHORITY_RUNTIME_OWNER_PREAUTHORIZED": True,
    }
    results = evaluator.evaluate_current_project_authority_runtime()
    assert {item.scope for item in results} == {
        "PROJECT_AUTHORITY_ELIGIBILITY_RUNTIME",
        "PROJECT_AUTHORITY_PROJECTION_RUNTIME",
        "PROJECT_AUTHORITY_RECORD_MATERIALIZATION",
        "AUTHORIZATION_READINESS_TRANSITION_ENGINE",
    }
    assert {item.state for item in results} == {AuthorizationState.AUTHORIZED}


def test_failed_readiness_predicate_holds_preapproved_scope() -> None:
    evaluator = AuthorizationTransitionEvaluator(ROOT)
    readiness = evaluator.derive_project_authority_runtime_readiness()
    readiness["CURRENT_ADR_CORPUS_VALID"] = False
    result = evaluator.evaluate("PROJECT_AUTHORITY_ELIGIBILITY_RUNTIME", readiness)
    assert result.state is AuthorizationState.HOLD_NOT_AUTHORIZED
    assert result.failed_predicates == ("CURRENT_ADR_CORPUS_VALID",)
    assert result.blockers == ("PREDICATE_NOT_SATISFIED:CURRENT_ADR_CORPUS_VALID",)


def test_non_preauthorized_downstream_scopes_remain_held() -> None:
    evaluator = AuthorizationTransitionEvaluator(ROOT)
    readiness = evaluator.derive_project_authority_runtime_readiness()
    solution = evaluator.evaluate("SOLUTION_SPACE_AUTHORITY_CONSUMPTION_RUNTIME", readiness)
    remediation = evaluator.evaluate("REMEDIATION_RUNTIME_IMPLEMENTATION", readiness)
    assert solution.state is AuthorizationState.HOLD_NOT_AUTHORIZED
    assert "NO_OWNER_PREAUTHORIZED_TRANSITION_RULE_FOR_THIS_SCOPE" in solution.blockers
    assert remediation.state is AuthorizationState.HOLD_NOT_AUTHORIZED
    assert "OUTSIDE_CURRENT_OWNER_APPROVAL_SCOPE" in remediation.blockers


def test_unknown_scope_cannot_be_auto_authorized() -> None:
    evaluator = AuthorizationTransitionEvaluator(ROOT)
    result = evaluator.evaluate("UNKNOWN_FUTURE_SCOPE", {})
    assert result.state is AuthorizationState.HOLD_NOT_AUTHORIZED
    assert result.blockers == ("NO_OWNER_PREAUTHORIZED_TRANSITION_RULE",)
