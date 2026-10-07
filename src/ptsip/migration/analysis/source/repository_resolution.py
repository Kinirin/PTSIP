from __future__ import annotations

from dataclasses import dataclass
from typing import Iterable

from ptsip.validation.components import normalize_selector, selector_matches_path, selector_specificity
from ptsip.migration.analysis.source.projection import (
    AmbiguousSourceElement,
    ExistingSourceElement,
    RemovedSourceElement,
    SourceCoverageProjection,
    SourceProjectionKind,
    UncoveredRepositoryElement,
    _ProjectedSourceArchitecture,
)

@dataclass(frozen=True)
class _RepositoryResolution:
    existing: tuple[ExistingSourceElement, ...]
    removed: tuple[RemovedSourceElement, ...]
    uncovered: tuple[UncoveredRepositoryElement, ...]
    ambiguous: tuple[AmbiguousSourceElement, ...]

def _boundary_matches(path: str, root: str) -> bool:
    path = normalize_selector(path)
    root = normalize_selector(root)
    return path == root or path.startswith(root + "/")

def _coverage_match(path: str, coverage: SourceCoverageProjection) -> tuple[tuple[int, int, int, int], str] | None:
    if coverage.kind == SourceProjectionKind.BOUNDARY:
        root = coverage.include[0] if coverage.include else ""
        if not root or not _boundary_matches(path, root):
            return None
        selector = normalize_selector(root)
        return selector_specificity(selector + "/**"), selector
    if any(selector_matches_path(path, selector) for selector in coverage.exclude):
        return None
    matching = [selector for selector in coverage.include if selector_matches_path(path, selector)]
    if not matching:
        return None
    best = max(matching, key=selector_specificity)
    return selector_specificity(best), normalize_selector(best)

def _resolve_repository(paths: Iterable[str], source: _ProjectedSourceArchitecture) -> _RepositoryResolution:
    existing: list[ExistingSourceElement] = []
    uncovered: list[UncoveredRepositoryElement] = []
    ambiguous: list[AmbiguousSourceElement] = []
    selector_hits: dict[tuple[str, str], int] = {}

    for path in sorted(set(paths)):
        matches: list[tuple[tuple[int, int, int, int], SourceCoverageProjection, str]] = []
        for coverage in source.coverages:
            matched = _coverage_match(path, coverage)
            if matched is None:
                continue
            score, selector = matched
            selector_hits[(coverage.declaration_id, selector)] = selector_hits.get((coverage.declaration_id, selector), 0) + 1
            matches.append((score, coverage, selector))
        if not matches:
            uncovered.append(UncoveredRepositoryElement(path))
            continue

        kinds = {item[1].kind for item in matches}
        if SourceProjectionKind.COMPONENT in kinds and SourceProjectionKind.ASSOCIATED_ARTIFACT in kinds:
            selected = matches
        else:
            best_score = max(item[0] for item in matches)
            selected = [item for item in matches if item[0] == best_score]

        if len({item[1].declaration_id for item in selected}) != 1:
            ambiguous.append(
                AmbiguousSourceElement(
                    path,
                    tuple(sorted({item[1].declaration_id: item[1] for item in selected}.values(), key=lambda coverage: (coverage.kind.value, coverage.declaration_id))),
                    tuple(sorted({item[2] for item in selected})),
                )
            )
            continue
        chosen = sorted(selected, key=lambda item: (item[1].kind.value, item[1].declaration_id, item[2]))[0]
        existing.append(ExistingSourceElement(path, chosen[1], chosen[2]))

    removed: list[RemovedSourceElement] = []
    for coverage in source.coverages:
        for selector in coverage.include:
            normalized = normalize_selector(selector)
            if selector_hits.get((coverage.declaration_id, normalized), 0):
                continue
            removed.append(RemovedSourceElement(f"{coverage.declaration_id}:{normalized}", coverage, normalized))

    return _RepositoryResolution(
        tuple(sorted(existing, key=lambda item: item.path)),
        tuple(sorted(removed, key=lambda item: item.element_id)),
        tuple(sorted(uncovered, key=lambda item: item.path)),
        tuple(sorted(ambiguous, key=lambda item: item.path)),
    )

