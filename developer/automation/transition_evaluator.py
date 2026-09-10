from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Mapping

from developer.automation.current_dependency_gate import (
    validate_current_legacy_dependency_gate,
)
from developer.automation.policy_loader import load_yaml, repository_root


P01_PLAN = "docs/planning/0.4.0/WU-02/WU-02-P01.yaml"
E4_STAGE = "P01_E4_MIGRATION_ONLY_RETIREMENT_AND_GATE_SIMPLIFICATION"


@dataclass(frozen=True)
class TransitionDecision:
    state: str
    action: str | None
    blockers: tuple[str, ...]
    confirmation_required: bool


def _execution_stage(plan: Mapping[str, object], stage_id: str) -> Mapping[str, object] | None:
    execution = plan.get("p01_e_execution_plan")
    if not isinstance(execution, Mapping):
        return None
    items = execution.get("execution_order")
    if not isinstance(items, list):
        return None
    found = [item for item in items if isinstance(item, Mapping) and item.get("id") == stage_id]
    return found[0] if len(found) == 1 else None


def evaluate_legacy_decisions_removal(root: str | Path | None = None) -> TransitionDecision:
    """Evaluate removal readiness from current control-plane state only."""

    base = repository_root(root)
    plan = load_yaml(P01_PLAN, root=base)
    migration = plan.get("migration_stages", {}).get("P01_E_LEGACY_REMOVAL", {})
    if not isinstance(migration, Mapping):
        migration = {}

    blockers: list[str] = []
    if validate_current_legacy_dependency_gate(base):
        blockers.append("CURRENT_LEGACY_DEPENDENCY_NONZERO")

    e4 = _execution_stage(plan, E4_STAGE)
    if e4 is None or e4.get("status") != "COMPLETE":
        blockers.append("P01_E4_VALIDATION_NOT_COMPLETE")

    action = migration.get("preauthorized_action")
    confirmation = migration.get("confirmation_required")
    if not isinstance(action, str) or not action:
        blockers.append("LEGACY_DECISIONS_REMOVAL_NOT_PREAUTHORIZED")
    if confirmation is not False:
        blockers.append("LEGACY_DECISIONS_REMOVAL_CONFIRMATION_POLICY_INVALID")

    if blockers:
        return TransitionDecision(
            state="HOLD_NOT_AUTHORIZED",
            action=None,
            blockers=tuple(sorted(set(blockers))),
            confirmation_required=False,
        )

    return TransitionDecision(
        state="AUTHORIZED",
        action=action,
        blockers=(),
        confirmation_required=False,
    )


if __name__ == "__main__":
    print(evaluate_legacy_decisions_removal())
