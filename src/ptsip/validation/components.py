from __future__ import annotations

import re
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Iterable, Protocol

from ..repository.snapshot import repository_files


COMPONENT_COVERED = "COMPONENT"
ASSOCIATED_ARTIFACT_COVERED = "ASSOCIATED_ARTIFACT"
UNCOVERED = "UNCOVERED"
AMBIGUOUS = "AMBIGUOUS"


class SelectorCandidateLike(Protocol):
    include: tuple[str, ...]


@dataclass(frozen=True)
class ComponentAssignment:
    path: str
    component_id: str
    selector: str

    def as_dict(self) -> dict[str, str]:
        return asdict(self)


@dataclass(frozen=True)
class ComponentConflict:
    path: str
    component_ids: tuple[str, ...]
    selectors: tuple[str, ...]

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


@dataclass(frozen=True)
class ComponentPartition:
    assignments: tuple[ComponentAssignment, ...]
    conflicts: tuple[ComponentConflict, ...]
    unmatched_selectors: tuple[str, ...]
    unassigned_files: tuple[str, ...]
    scan_errors: tuple[str, ...]

    def as_dict(self) -> dict[str, object]:
        return {
            "assignments": [item.as_dict() for item in self.assignments],
            "conflicts": [item.as_dict() for item in self.conflicts],
            "unmatched_selectors": list(self.unmatched_selectors),
            "unassigned_files": list(self.unassigned_files),
            "scan_errors": list(self.scan_errors),
            "assignment_count": len(self.assignments),
            "conflict_count": len(self.conflicts),
            "unassigned_count": len(self.unassigned_files),
        }


@dataclass(frozen=True)
class CandidateCoverage:
    status: str
    owner_ids: tuple[str, ...]
    owner_kinds: tuple[str, ...]
    selectors: tuple[str, ...]

    @property
    def covered(self) -> bool:
        return self.status in {COMPONENT_COVERED, ASSOCIATED_ARTIFACT_COVERED}

    def as_dict(self) -> dict[str, object]:
        return {
            "status": self.status,
            "owner_ids": list(self.owner_ids),
            "owner_kinds": list(self.owner_kinds),
            "selectors": list(self.selectors),
        }


def normalize_selector(pattern: str) -> str:
    text = pattern.replace("\\", "/").strip()
    while text.startswith("./"):
        text = text[2:]
    return text.strip("/")


def _has_glob(pattern: str) -> bool:
    return any(char in pattern for char in "*?[")


def _glob_regex(pattern: str) -> re.Pattern[str]:
    pattern = normalize_selector(pattern)
    out = ["^"]
    index = 0
    while index < len(pattern):
        char = pattern[index]
        if char == "*":
            if index + 1 < len(pattern) and pattern[index + 1] == "*":
                out.append(".*")
                index += 2
            else:
                out.append("[^/]*")
                index += 1
        elif char == "?":
            out.append("[^/]")
            index += 1
        elif char == "[":
            end = pattern.find("]", index + 1)
            if end == -1:
                out.append("\\[")
                index += 1
            else:
                out.append(pattern[index : end + 1])
                index = end + 1
        else:
            out.append(re.escape(char))
            index += 1
    out.append("$")
    return re.compile("".join(out))


def selector_matches_path(path: str, pattern: str) -> bool:
    normalized = normalize_selector(path)
    selector = normalize_selector(pattern)
    if not _has_glob(selector):
        return normalized == selector
    return bool(_glob_regex(selector).match(normalized))


def selector_specificity(pattern: str) -> tuple[int, int, int, int]:
    normalized = normalize_selector(pattern)
    exact = 1 if not _has_glob(normalized) else 0
    literal = len(re.sub(r"[*?\[\]]", "", normalized))
    depth = normalized.count("/")
    wildcards = normalized.count("*") + normalized.count("?") + normalized.count("[")
    return exact, literal, depth, -wildcards


def _recursive_root(pattern: str) -> str | None:
    normalized = normalize_selector(pattern)
    if normalized == "**":
        return ""
    if normalized.endswith("/**") and not _has_glob(normalized[:-3]):
        return normalized[:-3].rstrip("/")
    return None


def selector_covers_selector(declared: str, candidate: str) -> bool:
    """Return True only when the declared selector provably contains candidate scope.

    The check is deliberately conservative for general glob-to-glob containment.
    Equality and canonical recursive-root selectors are proven structurally; other
    glob shapes must match exactly rather than relying on heuristic expansion.
    """

    declared = normalize_selector(declared)
    candidate = normalize_selector(candidate)
    if declared == candidate:
        return True

    declared_root = _recursive_root(declared)
    if declared_root is None:
        return False
    if declared_root == "":
        return True

    candidate_root = _recursive_root(candidate)
    if candidate_root is not None:
        return candidate_root == declared_root or candidate_root.startswith(declared_root + "/")

    if _has_glob(candidate):
        return False
    return candidate == declared_root or candidate.startswith(declared_root + "/")


def selectors_overlap(left: str, right: str) -> bool:
    left = normalize_selector(left)
    right = normalize_selector(right)
    return (
        left == right
        or selector_covers_selector(left, right)
        or selector_covers_selector(right, left)
    )


def _declaration_candidate_score(
    declaration: dict[str, object],
    candidate_selectors: tuple[str, ...],
) -> tuple[tuple[int, int, int, int], tuple[str, ...]] | None:
    includes = tuple(str(item) for item in declaration.get("include", []))
    excludes = tuple(str(item) for item in declaration.get("exclude", []))
    if not includes or not candidate_selectors:
        return None

    best_for_candidate: list[tuple[tuple[int, int, int, int], str]] = []
    for candidate_selector in candidate_selectors:
        if any(selectors_overlap(exclude, candidate_selector) for exclude in excludes):
            return None
        covering = [
            selector
            for selector in includes
            if selector_covers_selector(selector, candidate_selector)
        ]
        if not covering:
            return None
        best = max(covering, key=selector_specificity)
        best_for_candidate.append((selector_specificity(best), best))

    weakest_score = min(item[0] for item in best_for_candidate)
    selectors = tuple(sorted({item[1] for item in best_for_candidate}))
    return weakest_score, selectors


def _coverage_candidates(
    declarations: Iterable[dict[str, object]],
    *,
    kind: str,
    candidate_selectors: tuple[str, ...],
) -> list[tuple[tuple[int, int, int, int], str, str, tuple[str, ...]]]:
    matches: list[tuple[tuple[int, int, int, int], str, str, tuple[str, ...]]] = []
    for declaration in declarations:
        owner_id = str(declaration.get("id", "")).strip()
        if not owner_id:
            continue
        scored = _declaration_candidate_score(declaration, candidate_selectors)
        if scored is None:
            continue
        score, selectors = scored
        matches.append((score, kind, owner_id, selectors))
    return matches


def resolve_candidate_coverage(
    candidate: SelectorCandidateLike | Iterable[str],
    components: list[dict[str, object]],
    associated_artifacts: list[dict[str, object]] | None = None,
) -> CandidateCoverage:
    """Resolve candidate selector ownership using canonical selector semantics.

    Component and associated-artifact scopes are evaluated separately. Within a
    declaration kind, canonical specificity chooses one owner unless the best
    score ties. Any simultaneous component/artifact coverage is ambiguous because
    those scopes are architecturally distinct rather than competing aliases.
    """

    raw_selectors = candidate.include if hasattr(candidate, "include") else candidate
    candidate_selectors = tuple(normalize_selector(str(item)) for item in raw_selectors)
    component_matches = _coverage_candidates(
        components,
        kind=COMPONENT_COVERED,
        candidate_selectors=candidate_selectors,
    )
    artifact_matches = _coverage_candidates(
        associated_artifacts or [],
        kind=ASSOCIATED_ARTIFACT_COVERED,
        candidate_selectors=candidate_selectors,
    )

    def winners(
        matches: list[tuple[tuple[int, int, int, int], str, str, tuple[str, ...]]]
    ) -> list[tuple[tuple[int, int, int, int], str, str, tuple[str, ...]]]:
        if not matches:
            return []
        best_score = max(item[0] for item in matches)
        return [item for item in matches if item[0] == best_score]

    component_winners = winners(component_matches)
    artifact_winners = winners(artifact_matches)

    if component_winners and artifact_winners:
        combined = component_winners + artifact_winners
        return CandidateCoverage(
            status=AMBIGUOUS,
            owner_ids=tuple(sorted({item[2] for item in combined})),
            owner_kinds=tuple(sorted({item[1] for item in combined})),
            selectors=tuple(sorted({selector for item in combined for selector in item[3]})),
        )

    selected = component_winners or artifact_winners
    if not selected:
        return CandidateCoverage(UNCOVERED, (), (), ())

    owner_ids = tuple(sorted({item[2] for item in selected}))
    kind = selected[0][1]
    selectors = tuple(sorted({selector for item in selected for selector in item[3]}))
    if len(owner_ids) > 1:
        return CandidateCoverage(
            status=AMBIGUOUS,
            owner_ids=owner_ids,
            owner_kinds=(kind,),
            selectors=selectors,
        )
    return CandidateCoverage(
        status=kind,
        owner_ids=owner_ids,
        owner_kinds=(kind,),
        selectors=selectors,
    )


def partition_components(repository_root: str | Path, components: list[dict[str, object]]) -> ComponentPartition:
    root = Path(repository_root).resolve()
    _mode, paths, scan_errors = repository_files(root)
    selector_hits: dict[tuple[str, str], int] = {}
    assignments: list[ComponentAssignment] = []
    conflicts: list[ComponentConflict] = []
    unassigned: list[str] = []

    for path in paths:
        candidates: list[tuple[tuple[int, int, int, int], str, str]] = []
        for component in components:
            component_id = str(component.get("id", ""))
            includes = [str(item) for item in component.get("include", [])]
            excludes = [str(item) for item in component.get("exclude", [])]
            if any(selector_matches_path(path, selector) for selector in excludes):
                continue
            matching = [selector for selector in includes if selector_matches_path(path, selector)]
            for selector in matching:
                selector_hits[(component_id, selector)] = selector_hits.get((component_id, selector), 0) + 1
            if not matching:
                continue
            best_selector = max(matching, key=selector_specificity)
            candidates.append((selector_specificity(best_selector), component_id, best_selector))
        if not candidates:
            unassigned.append(path)
            continue
        best_score = max(item[0] for item in candidates)
        winners = [item for item in candidates if item[0] == best_score]
        winner_components = sorted({item[1] for item in winners})
        if len(winner_components) > 1:
            conflicts.append(
                ComponentConflict(
                    path=path,
                    component_ids=tuple(winner_components),
                    selectors=tuple(sorted({item[2] for item in winners})),
                )
            )
            continue
        winner = winners[0]
        assignments.append(ComponentAssignment(path=path, component_id=winner[1], selector=winner[2]))

    unmatched: list[str] = []
    for component in components:
        component_id = str(component.get("id", ""))
        for selector in [str(item) for item in component.get("include", [])]:
            if selector_hits.get((component_id, selector), 0) == 0:
                unmatched.append(f"{component_id}:{selector}")

    return ComponentPartition(
        assignments=tuple(assignments),
        conflicts=tuple(conflicts),
        unmatched_selectors=tuple(sorted(unmatched)),
        unassigned_files=tuple(unassigned),
        scan_errors=tuple(scan_errors),
    )
