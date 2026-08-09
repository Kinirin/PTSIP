from __future__ import annotations

import re
from dataclasses import asdict, dataclass
from pathlib import Path

from ..repository.snapshot import repository_files

_GLOB_META = re.compile(r"[*?[]")


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


def _normalize(pattern: str) -> str:
    text = pattern.replace("\\", "/").strip()
    while text.startswith("./"):
        text = text[2:]
    return text.strip("/")


def _glob_regex(pattern: str) -> re.Pattern[str]:
    pattern = _normalize(pattern)
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


def _matches(path: str, pattern: str) -> bool:
    normalized = _normalize(path)
    selector = _normalize(pattern)
    if not _GLOB_META.search(selector):
        return normalized == selector
    return bool(_glob_regex(selector).match(normalized))


def _specificity(pattern: str) -> tuple[int, int, int, int]:
    normalized = _normalize(pattern)
    exact = 1 if not _GLOB_META.search(normalized) else 0
    literal = len(re.sub(r"[*?\[\]]", "", normalized))
    depth = normalized.count("/")
    wildcards = normalized.count("*") + normalized.count("?") + normalized.count("[")
    return exact, literal, depth, -wildcards


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
            if any(_matches(path, selector) for selector in excludes):
                continue
            matching = [selector for selector in includes if _matches(path, selector)]
            for selector in matching:
                selector_hits[(component_id, selector)] = selector_hits.get((component_id, selector), 0) + 1
            if not matching:
                continue
            best_selector = max(matching, key=_specificity)
            candidates.append((_specificity(best_selector), component_id, best_selector))
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
