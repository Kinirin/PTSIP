from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path

from developer.automation.legacy_reference_scanner import removal_blockers
from developer.automation.policy_loader import load_yaml, repository_root


LEGACY_REFERENCE_INVENTORY = "developer/policy/legacy-reference-inventory.yaml"
SPLIT_TEXTUAL_REVIEW = "developer/policy/split-textual-reference-review.yaml"


@dataclass(frozen=True)
class TransitionDecision:
    state: str
    action: str | None
    blockers: tuple[str, ...]
    confirmation_required: bool


def evaluate_legacy_decisions_removal(root: str | Path | None = None) -> TransitionDecision:
    base = repository_root(root)
    inventory = load_yaml(LEGACY_REFERENCE_INVENTORY, root=base)
    removal_policy = inventory["removal_policy"]
    blockers = list(removal_blockers(base))

    if not removal_policy.get("preauthorized", False):
        blockers.append("LEGACY_DECISIONS_REMOVAL_NOT_PREAUTHORIZED")

    split_review = load_yaml(SPLIT_TEXTUAL_REVIEW, root=base)
    review_entries = split_review.get("entries", [])
    review_status = split_review.get("generation", {}).get("status")
    unresolved = [
        item
        for item in review_entries
        if isinstance(item, dict) and item.get("status") != "RESOLVED"
    ]
    if review_status != "COMPLETE" or unresolved:
        blockers.append("SPLIT_TEXTUAL_REFERENCE_REVIEW_INCOMPLETE")

    src = base / "src" / "ptsip"
    runtime_references = []
    for path in src.rglob("*.py"):
        text = path.read_text(encoding="utf-8")
        if "decisions/" in text and path.name != "github_authority.py":
            runtime_references.append(path.relative_to(base).as_posix())
    if runtime_references:
        blockers.append("PRODUCT_RUNTIME_DECISIONS_DEPENDENCY_NONZERO")

    if blockers:
        return TransitionDecision(
            state="HOLD_NOT_AUTHORIZED",
            action=None,
            blockers=tuple(sorted(set(blockers))),
            confirmation_required=False,
        )

    return TransitionDecision(
        state="AUTHORIZED",
        action=str(removal_policy["action_when_ready"]),
        blockers=(),
        confirmation_required=bool(removal_policy["confirmation_required_when_ready"]),
    )


if __name__ == "__main__":
    decision = evaluate_legacy_decisions_removal()
    print(decision)
