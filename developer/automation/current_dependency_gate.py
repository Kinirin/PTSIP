from __future__ import annotations

import re
from dataclasses import dataclass
from pathlib import Path

from developer.automation.policy_loader import repository_root


_LOCAL_POLICY_TREE = "decisions" + "/"
_POLICY_ID_RE = re.compile(r"\bADR-" + r"[0-9]{4}\b")
_POLICY_PATH_RE = re.compile(
    re.escape(_LOCAL_POLICY_TREE)
    + r"[A-Za-z0-9_.*{}<>/\-]+(?:\.[A-Za-z0-9_.*{}<>/\-]+)?"
)

_CURRENT_EXACT_PATHS = (
    "AGENTS.md",
    "developer/profiles/ptsip-repository.yaml",
    "docs/planning/0.4.0/WU-02/WU-02.yaml",
    "docs/planning/0.4.0/WU-02/WU-02-P01.yaml",
    "releasenote/project-profile/pp.1.01.md",
    "developer/policy/index.yaml",
    "src/ptsip/specdata/support-policy-index.yaml",
    "schemas/ptsip-authorization-transition.schema.json",
)
_CURRENT_GLOBS = (
    "developer/automation/*.py",
    "developer/policy/MPD-*.yaml",
    "developer/policy/registries/*.yaml",
    "developer/policy/schemas/*.json",
    "src/ptsip/**/*.py",
    "src/ptsip/specdata/SFP-*.yaml",
    "src/ptsip/specdata/ptsip-support-*.yaml",
    "src/ptsip/specdata/ptsip-support-*.json",
    "schemas/ptsip-support-*.json",
)
_EXCLUDED_CURRENT_PATHS = {
    "src/ptsip/app/github_authority.py",
}


@dataclass(frozen=True)
class CurrentDependencyHit:
    path: str
    line: int
    reference: str


def _current_paths(base: Path) -> tuple[Path, ...]:
    paths: set[Path] = set()
    for relative in _CURRENT_EXACT_PATHS:
        path = base / relative
        if path.is_file():
            paths.add(path)
    for pattern in _CURRENT_GLOBS:
        for path in base.glob(pattern):
            if path.is_file():
                paths.add(path)
    return tuple(sorted(paths))


def scan_current_legacy_dependencies(
    root: str | Path | None = None,
) -> tuple[CurrentDependencyHit, ...]:
    """Scan current control surfaces only; historical/frozen records are out of scope."""

    base = repository_root(root)
    hits: list[CurrentDependencyHit] = []
    for path in _current_paths(base):
        relative = path.relative_to(base).as_posix()
        if relative in _EXCLUDED_CURRENT_PATHS:
            continue
        try:
            lines = path.read_text(encoding="utf-8", errors="strict").splitlines()
        except (UnicodeDecodeError, OSError):
            continue
        for line_no, line in enumerate(lines, start=1):
            matches = [*_POLICY_PATH_RE.finditer(line), *_POLICY_ID_RE.finditer(line)]
            for match in matches:
                hits.append(
                    CurrentDependencyHit(
                        path=relative,
                        line=line_no,
                        reference=match.group(0),
                    )
                )
    return tuple(hits)


def validate_current_legacy_dependency_gate(
    root: str | Path | None = None,
) -> tuple[str, ...]:
    return tuple(
        f"current legacy dependency: {hit.path}:{hit.line} {hit.reference}"
        for hit in scan_current_legacy_dependencies(root)
    )


if __name__ == "__main__":
    failures = validate_current_legacy_dependency_gate()
    if failures:
        raise SystemExit("\n".join(failures))
    print("Current legacy dependency gate: PASS")
