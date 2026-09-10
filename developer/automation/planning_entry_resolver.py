from __future__ import annotations

import argparse
import json
import subprocess
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any, Sequence

from developer.automation.policy_loader import load_yaml, repository_root


ROOT_INDEX = "docs/planning/index.yaml"


class PlanningEntryResolutionError(RuntimeError):
    """Raised when branch-aware planning entry resolution cannot be completed safely."""

    def __init__(self, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code


@dataclass(frozen=True)
class PlanningEntryResolution:
    status: str
    branch: str
    plan_version: str
    entry_document: str
    role: str
    work_unit: str | None = None

    def to_payload(self) -> dict[str, str | None]:
        return asdict(self)


def detect_current_branch(root: str | Path | None = None) -> str:
    base = repository_root(root)
    completed = subprocess.run(
        ["git", "-C", str(base), "branch", "--show-current"],
        check=False,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    if completed.returncode != 0:
        detail = completed.stderr.strip() or "git branch --show-current failed"
        raise PlanningEntryResolutionError("BRANCH_DETECTION_FAILED", detail)
    branch = completed.stdout.strip()
    if not branch:
        raise PlanningEntryResolutionError(
            "DETACHED_HEAD",
            "Current Git state has no branch name; planning entry resolution fails closed.",
        )
    return branch


def _entry_document_path(base: Path, entry_document: str) -> Path:
    candidate = (base / entry_document).resolve()
    try:
        candidate.relative_to(base.resolve())
    except ValueError as exc:
        raise PlanningEntryResolutionError(
            "ENTRY_DOCUMENT_OUTSIDE_REPOSITORY",
            f"Resolved planning entry escapes repository root: {entry_document}",
        ) from exc
    if not candidate.is_file():
        raise PlanningEntryResolutionError(
            "MISSING_ENTRY_DOCUMENT",
            f"Resolved planning entry does not exist: {entry_document}",
        )
    return candidate


def resolve_planning_entry(
    branch: str | None = None,
    *,
    root: str | Path | None = None,
) -> PlanningEntryResolution:
    base = repository_root(root)
    selected_branch = branch if branch is not None else detect_current_branch(base)
    if not selected_branch:
        raise PlanningEntryResolutionError(
            "EMPTY_BRANCH",
            "Planning entry resolution requires a non-empty exact branch name.",
        )

    root_index = load_yaml(ROOT_INDEX, root=base)
    matches: list[tuple[dict[str, Any], dict[str, Any]]] = []

    for plan in root_index.get("plans", []):
        if not isinstance(plan, dict):
            continue
        routing = plan.get("entry_routing")
        if not isinstance(routing, dict):
            continue
        for entry in routing.get("branch_entrypoints", []):
            if not isinstance(entry, dict):
                continue
            if entry.get("branch") == selected_branch:
                matches.append((plan, entry))

    if not matches:
        raise PlanningEntryResolutionError(
            "UNKNOWN_PLANNING_ENTRY",
            f"No exact planning entry is declared for branch {selected_branch!r}.",
        )
    if len(matches) != 1:
        raise PlanningEntryResolutionError(
            "AMBIGUOUS_PLANNING_ENTRY",
            f"Multiple planning entries are declared for branch {selected_branch!r}.",
        )

    plan, entry = matches[0]
    plan_version = plan.get("plan_version")
    entry_document = entry.get("entry_document")
    role = entry.get("role")
    work_unit = entry.get("work_unit")

    if not isinstance(plan_version, str):
        raise PlanningEntryResolutionError("INVALID_PLAN_VERSION", "Resolved plan_version is invalid.")
    if not isinstance(entry_document, str):
        raise PlanningEntryResolutionError(
            "INVALID_ENTRY_DOCUMENT",
            "Resolved entry_document is invalid.",
        )
    if not isinstance(role, str):
        raise PlanningEntryResolutionError("INVALID_ENTRY_ROLE", "Resolved entry role is invalid.")
    if work_unit is not None and not isinstance(work_unit, str):
        raise PlanningEntryResolutionError("INVALID_WORK_UNIT", "Resolved work_unit is invalid.")

    _entry_document_path(base, entry_document)

    return PlanningEntryResolution(
        status="RESOLVED",
        branch=selected_branch,
        plan_version=plan_version,
        entry_document=entry_document,
        role=role,
        work_unit=work_unit,
    )


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Resolve the exact planning entry document for the current Git branch."
    )
    parser.add_argument(
        "--branch",
        help="Exact branch name override. When omitted, use git branch --show-current.",
    )
    parser.add_argument(
        "--root",
        help="Repository root override for tests or tooling.",
    )
    parser.add_argument(
        "--path-only",
        action="store_true",
        help="Print only the resolved entry_document path.",
    )
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        resolution = resolve_planning_entry(args.branch, root=args.root)
    except PlanningEntryResolutionError as exc:
        print(
            json.dumps(
                {
                    "status": "UNRESOLVED",
                    "code": exc.code,
                    "message": str(exc),
                },
                sort_keys=True,
            ),
        )
        return 2

    if args.path_only:
        print(resolution.entry_document)
    else:
        print(json.dumps(resolution.to_payload(), sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
