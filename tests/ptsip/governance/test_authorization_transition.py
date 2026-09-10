from __future__ import annotations

from pathlib import Path

from ptsip.governance import AuthorizationState, AuthorizationTransitionEvaluator


ROOT = Path(__file__).resolve().parents[3]


def test_product_runtime_does_not_ship_project_owner_authorization_grants() -> None:
    evaluator = AuthorizationTransitionEvaluator(ROOT)
    assert evaluator.derive_project_authority_runtime_readiness() == {
        "OWNER_AUTHORIZATION_RULE_SHIPPED": False,
        "OWNER_GRANT_DATA_SHIPPED": False,
    }
    results = evaluator.evaluate_current_project_authority_runtime()
    assert {item.state for item in results} == {AuthorizationState.HOLD_NOT_AUTHORIZED}
    assert {item.blockers for item in results} == {("NO_SHIPPED_OWNER_AUTHORIZATION_RULE",)}


def test_unknown_scope_is_fail_closed_without_developer_policy() -> None:
    evaluator = AuthorizationTransitionEvaluator(ROOT)
    result = evaluator.evaluate("UNKNOWN_FUTURE_SCOPE", {})
    assert result.state is AuthorizationState.HOLD_NOT_AUTHORIZED
    assert result.blockers == ("NO_SHIPPED_OWNER_AUTHORIZATION_RULE",)
