from __future__ import annotations

from pathlib import Path
from typing import Mapping

import yaml

from .model import AuthorizationState, AuthorizationTransitionResult, GovernanceAuthorityError


class AuthorizationTransitionEvaluator:
    """Product-side authorization contract.

    Shipped Support Feature data contains authorization state semantics but no
    PTSIP Project Owner grant. Therefore this evaluator never manufactures an
    implementation authorization from developer policy.
    """

    REGISTRY = "src/ptsip/specdata/ptsip-support-authorization-registry.yaml"

    def __init__(self, repository_root: str | Path) -> None:
        self.root = Path(repository_root).resolve()
        path = (self.root / self.REGISTRY).resolve()
        if self.root not in path.parents or not path.is_file():
            raise GovernanceAuthorityError("AUTHORIZATION_TRANSITION_REGISTRY_MISSING", "missing shipped support authorization registry.", self.REGISTRY)
        value = yaml.safe_load(path.read_text(encoding="utf-8"))
        if not isinstance(value, dict):
            raise GovernanceAuthorityError("AUTHORIZATION_TRANSITION_REGISTRY_INVALID", "support authorization registry must be a mapping.", value)
        self.registry = value

    def derive_project_authority_runtime_readiness(self) -> dict[str, bool]:
        return {
            "OWNER_AUTHORIZATION_RULE_SHIPPED": False,
            "OWNER_GRANT_DATA_SHIPPED": False,
        }

    def evaluate(self, scope: str, readiness: Mapping[str, bool]) -> AuthorizationTransitionResult:
        return AuthorizationTransitionResult(
            scope=scope,
            state=AuthorizationState.HOLD_NOT_AUTHORIZED,
            rule_id=None,
            blockers=("NO_SHIPPED_OWNER_AUTHORIZATION_RULE",),
        )

    def evaluate_current_project_authority_runtime(self) -> tuple[AuthorizationTransitionResult, ...]:
        readiness=self.derive_project_authority_runtime_readiness()
        scopes=(
            "PROJECT_AUTHORITY_ELIGIBILITY_RUNTIME",
            "PROJECT_AUTHORITY_PROJECTION_RUNTIME",
            "PROJECT_AUTHORITY_RECORD_MATERIALIZATION",
            "AUTHORIZATION_READINESS_TRANSITION_ENGINE",
        )
        return tuple(self.evaluate(scope,readiness) for scope in scopes)
