from __future__ import annotations

from ptsip.migration.analysis.findings.architecture import ArchitectureFinding, ArchitectureFindingKind
from ptsip.migration.analysis.target.semantics import TargetArchitectureState

def _relationship_findings(
    source_relationships: tuple[tuple[str, str, str], ...], target: TargetArchitectureState | None,
) -> tuple[ArchitectureFinding, ...]:
    if target is None or not source_relationships:
        return ()
    target_semantics = {(item.source, item.target, item.relationship_type) for item in target.relationships}
    return tuple(
        ArchitectureFinding(
            f"relationship:{relation[0]}:{relation[1]}:{relation[2]}",
            ArchitectureFindingKind.MISSING_RELATIONSHIP,
            "A source typed relationship has no exact semantic counterpart in the accepted target state; this is reviewable analysis, not an automatic target delta.",
        )
        for relation in source_relationships
        if relation not in target_semantics
    )

