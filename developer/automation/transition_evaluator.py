from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path

from developer.automation.policy_loader import load_yaml, repository_root


@dataclass(frozen=True)
class TransitionDecision:
    state: str
    action: str | None
    blockers: tuple[str, ...]
    confirmation_required: bool


def evaluate_legacy_decisions_removal(root: str | Path | None = None) -> TransitionDecision:
    base = repository_root(root)
    index = load_yaml("developer/policy/index.yaml", root=base)
    migration = index["legacy_decisions_migration"]
    automatic = migration["automatic_removal"]
    blockers = list(migration.get("current_blockers", []))
    split_review = load_yaml("developer/policy/split-textual-reference-review.yaml", root=base)
    review_entries = split_review.get("entries", [])
    review_status = split_review.get("generation", {}).get("status")
    unresolved = [
        item for item in review_entries
        if isinstance(item, dict) and item.get("status") != "RESOLVED"
    ]
    if review_status != "COMPLETE" or unresolved:
        blockers.append("SPLIT_TEXTUAL_REFERENCE_REVIEW_INCOMPLETE")

    src = base / "src" / "ptsip"
    runtime_references = []
    for path in src.rglob("*.py"):
        text = path.read_text(encoding="utf-8")
        if "decisions/" in text or '"decisions"' in text or "'decisions'" in text:
            runtime_references.append(path.relative_to(base).as_posix())
    if runtime_references and "SRC_PTSIP_GOVERNANCE_READS_DECISIONS" not in blockers:
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
        action=str(automatic["action_when_ready"]),
        blockers=(),
        confirmation_required=bool(automatic["confirmation_required_when_ready"]),
    )


if __name__ == "__main__":
    decision = evaluate_legacy_decisions_removal()
    print(decision)
