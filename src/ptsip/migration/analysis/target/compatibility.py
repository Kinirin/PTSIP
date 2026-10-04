from __future__ import annotations

from dataclasses import dataclass
from enum import StrEnum

from ptsip.validation.components import selector_matches_path, selector_specificity
from ptsip.migration.analysis.findings.lifecycle import LifecycleFinding, LifecycleFindingKind
from ptsip.migration.analysis.source.projection import ExistingSourceElement, SourceProjectionKind
from ptsip.migration.analysis.target.semantics import TargetArchitectureState, TargetSemantics

class TargetCompatibility(StrEnum):
    NOT_EVALUATED = "NOT_EVALUATED"
    ALREADY_SATISFIED = "ALREADY_SATISFIED"
    COMPATIBLE_TARGET_STATE = "COMPATIBLE_TARGET_STATE"
    CONFLICTING_TARGET_STATE = "CONFLICTING_TARGET_STATE"
    TARGET_REVIEW_REQUIRED = "TARGET_REVIEW_REQUIRED"

@dataclass(frozen=True)
class _TargetOwner:
    kind: SourceProjectionKind
    id: str
    classification: str | None
    selector: str

def _target_match(path: str, target: TargetArchitectureState) -> tuple[_TargetOwner | None, bool]:
    component_matches: list[tuple[tuple[int, int, int, int], _TargetOwner]] = []
    for item in target.components:
        if any(selector_matches_path(path, selector) for selector in item.exclude):
            continue
        matching = [selector for selector in item.include if selector_matches_path(path, selector)]
        if matching:
            best = max(matching, key=selector_specificity)
            component_matches.append((selector_specificity(best), _TargetOwner(SourceProjectionKind.COMPONENT, item.id, item.classification, best)))
    artifact_matches: list[tuple[tuple[int, int, int, int], _TargetOwner]] = []
    for item in target.associated_artifacts:
        if any(selector_matches_path(path, selector) for selector in item.exclude):
            continue
        matching = [selector for selector in item.include if selector_matches_path(path, selector)]
        if matching:
            best = max(matching, key=selector_specificity)
            artifact_matches.append((selector_specificity(best), _TargetOwner(SourceProjectionKind.ASSOCIATED_ARTIFACT, item.id, None, best)))
    if component_matches and artifact_matches:
        return None, True
    selected = component_matches or artifact_matches
    if not selected:
        return None, False
    best_score = max(item[0] for item in selected)
    winners = [item[1] for item in selected if item[0] == best_score]
    if len({(item.kind.value, item.id) for item in winners}) != 1:
        return None, True
    return sorted(winners, key=lambda item: (item.kind.value, item.id))[0], False

def _target_compatibility(
    element: ExistingSourceElement,
    target: TargetArchitectureState | None,
    target_semantics: TargetSemantics,
) -> tuple[TargetCompatibility, LifecycleFinding | None]:
    source_classification = element.coverage.source_classification
    if target is None:
        if source_classification == "TOOLCHAIN":
            return TargetCompatibility.NOT_EVALUATED, LifecycleFinding(
                element.path,
                LifecycleFindingKind.HISTORICAL_TOOLCHAIN_AMBIGUITY,
                source_classification,
                None,
                "Historical TOOLCHAIN is source vocabulary and cannot select one target lifecycle without project-owned resolution.",
            )
        return TargetCompatibility.NOT_EVALUATED, None

    owner, ambiguous = _target_match(element.path, target)
    if ambiguous:
        return TargetCompatibility.TARGET_REVIEW_REQUIRED, LifecycleFinding(
            element.path,
            LifecycleFindingKind.TARGET_REVIEW_REQUIRED,
            source_classification,
            None,
            "Accepted target state has ambiguous coverage for this source obligation.",
        )
    if owner is None:
        return TargetCompatibility.TARGET_REVIEW_REQUIRED, LifecycleFinding(
            element.path,
            LifecycleFindingKind.TARGET_REVIEW_REQUIRED,
            source_classification,
            None,
            "No accepted target declaration provably covers this required source element.",
        )

    if element.coverage.kind == SourceProjectionKind.ASSOCIATED_ARTIFACT:
        if owner.kind != SourceProjectionKind.ASSOCIATED_ARTIFACT:
            return TargetCompatibility.TARGET_REVIEW_REQUIRED, LifecycleFinding(
                element.path,
                LifecycleFindingKind.TARGET_REVIEW_REQUIRED,
                None,
                owner.classification,
                "A source associated-artifact obligation is covered by a different target declaration kind.",
            )
        return (TargetCompatibility.ALREADY_SATISFIED if owner.id == element.coverage.declaration_id else TargetCompatibility.COMPATIBLE_TARGET_STATE), None

    if owner.kind != SourceProjectionKind.COMPONENT:
        return TargetCompatibility.TARGET_REVIEW_REQUIRED, LifecycleFinding(
            element.path,
            LifecycleFindingKind.TARGET_REVIEW_REQUIRED,
            source_classification,
            None,
            "A source component/boundary obligation is covered by a target associated-artifact scope.",
        )
    target_classification = owner.classification
    if source_classification == "TOOLCHAIN":
        return TargetCompatibility.TARGET_REVIEW_REQUIRED, LifecycleFinding(
            element.path,
            LifecycleFindingKind.HISTORICAL_TOOLCHAIN_AMBIGUITY,
            source_classification,
            target_classification,
            "Historical TOOLCHAIN can correspond to multiple target lifecycles; no automatic conversion is authoritative.",
        )
    if source_classification == target_classification and source_classification in target_semantics.classifications:
        return (
            TargetCompatibility.ALREADY_SATISFIED if owner.id == element.coverage.declaration_id else TargetCompatibility.COMPATIBLE_TARGET_STATE,
            LifecycleFinding(
                element.path,
                LifecycleFindingKind.EXACT_SEMANTIC_PRESERVATION,
                source_classification,
                target_classification,
                "Source lifecycle classification is preserved by accepted target coverage.",
            ),
        )
    if source_classification in target_semantics.classifications and target_classification in target_semantics.classifications:
        return TargetCompatibility.CONFLICTING_TARGET_STATE, LifecycleFinding(
            element.path,
            LifecycleFindingKind.POSSIBLE_LIFECYCLE_SEPARATION,
            source_classification,
            target_classification,
            "Source and accepted target lifecycle classifications differ; owner review is required before treating the obligation as resolved.",
        )
    return TargetCompatibility.TARGET_REVIEW_REQUIRED, LifecycleFinding(
        element.path,
        LifecycleFindingKind.TARGET_REVIEW_REQUIRED,
        source_classification,
        target_classification,
        "Lifecycle compatibility cannot be proven under the target Project Profile vocabulary.",
    )

