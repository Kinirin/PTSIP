from __future__ import annotations

import argparse
import json
import subprocess
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Sequence

from developer.automation.planning_merge_reconciler import (
    PlanningStateReconciliationError,
    _find_plan_for_integration_branch,
    _leaf_entrypoint,
    detect_current_branch,
    reconcile_planning_state,
)
from developer.automation.policy_loader import load_yaml, repository_root


ROOT_INDEX = "docs/planning/index.yaml"


class PlanningLeafMergeError(RuntimeError):
    """Raised when an independent planning leaf cannot be merged safely."""

    def __init__(self, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code


@dataclass(frozen=True)
class PlanningLeafMergeResult:
    status: str
    source_branch: str
    source_ref: str
    integration_branch: str
    merged_work_unit: str
    merge_commit: str
    current_gate: str

    def to_payload(self) -> dict[str, str]:
        return asdict(self)


def _git(
    base: Path,
    *args: str,
    check: bool = False,
) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", "-C", str(base), *args],
        check=check,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )


def _require_clean_worktree(base: Path) -> None:
    completed = _git(base, "status", "--porcelain")
    if completed.returncode != 0:
        raise PlanningLeafMergeError(
            "WORKTREE_STATUS_FAILED",
            completed.stderr.strip() or "git status --porcelain failed",
        )
    if completed.stdout.strip():
        raise PlanningLeafMergeError(
            "DIRTY_INTEGRATION_WORKTREE",
            "Integration worktree must be clean before an automated leaf merge.",
        )


def _resolve_source_ref(base: Path, branch: str) -> str:
    candidates = (
        (f"refs/heads/{branch}", branch),
        (f"refs/remotes/origin/{branch}", f"origin/{branch}"),
    )
    for full_ref, merge_ref in candidates:
        completed = _git(base, "show-ref", "--verify", "--quiet", full_ref)
        if completed.returncode == 0:
            return merge_ref
    raise PlanningLeafMergeError(
        "LEAF_BRANCH_NOT_FOUND",
        f"Leaf branch {branch!r} is not available as a local or origin ref.",
    )


def _merge_base(base: Path, source_ref: str) -> str:
    completed = _git(base, "merge-base", "HEAD", source_ref)
    if completed.returncode != 0:
        raise PlanningLeafMergeError(
            "MERGE_BASE_FAILED",
            completed.stderr.strip() or f"Unable to resolve merge base for {source_ref!r}.",
        )
    merge_base = completed.stdout.strip()
    if not merge_base:
        raise PlanningLeafMergeError(
            "MERGE_BASE_FAILED",
            f"Unable to resolve merge base for {source_ref!r}.",
        )
    return merge_base


def _leaf_changed_paths(base: Path, merge_base: str, source_ref: str) -> tuple[str, ...]:
    completed = _git(
        base,
        "diff",
        "--name-only",
        f"{merge_base}..{source_ref}",
    )
    if completed.returncode != 0:
        raise PlanningLeafMergeError(
            "LEAF_DIFF_FAILED",
            completed.stderr.strip() or f"Unable to inspect changes on {source_ref!r}.",
        )
    return tuple(
        line.strip().replace("\\", "/")
        for line in completed.stdout.splitlines()
        if line.strip()
    )


def _forbidden_shared_paths(root_plan: dict[str, object]) -> set[str]:
    version_index = root_plan.get("path")
    if not isinstance(version_index, str):
        raise PlanningLeafMergeError(
            "VERSION_INDEX_PATH_INVALID",
            "Planning root entry has no valid version-index path.",
        )
    return {ROOT_INDEX, version_index}


def _assert_leaf_did_not_edit_shared_planning(
    base: Path,
    source_ref: str,
    root_plan: dict[str, object],
) -> None:
    merge_base = _merge_base(base, source_ref)
    changed = set(_leaf_changed_paths(base, merge_base, source_ref))
    forbidden = sorted(changed.intersection(_forbidden_shared_paths(root_plan)))
    if forbidden:
        raise PlanningLeafMergeError(
            "LEAF_SHARED_INDEX_MUTATION_FORBIDDEN",
            "Independent leaf branches must not modify parent-owned shared planning indexes: "
            + ", ".join(forbidden),
        )


def _abort_merge(base: Path) -> None:
    merge_head = _git(base, "rev-parse", "-q", "--verify", "MERGE_HEAD")
    if merge_head.returncode == 0:
        _git(base, "merge", "--abort")


def merge_leaf_branch(
    branch: str,
    *,
    root: str | Path | None = None,
    message: str | None = None,
) -> PlanningLeafMergeResult:
    base = repository_root(root)
    integration_branch = detect_current_branch(base)
    root_index = load_yaml(ROOT_INDEX, root=base)
    root_plan = _find_plan_for_integration_branch(root_index, integration_branch)
    routing = root_plan.get("entry_routing", {})
    if not isinstance(routing, dict):
        raise PlanningLeafMergeError(
            "ENTRY_ROUTING_MISSING",
            f"No entry routing is declared for integration branch {integration_branch!r}.",
        )
    merge_contract = routing.get("merge_reconciliation", {})
    if not isinstance(merge_contract, dict):
        raise PlanningLeafMergeError(
            "MERGE_RECONCILIATION_MISSING",
            "Planning merge reconciliation contract is not declared.",
        )
    if merge_contract.get("target_branch") != integration_branch:
        raise PlanningLeafMergeError(
            "WRONG_INTEGRATION_BRANCH",
            f"Automated leaf merge must run on {merge_contract.get('target_branch')!r}, "
            f"not {integration_branch!r}.",
        )
    if merge_contract.get("leaf_shared_index_mutation") != "FORBIDDEN":
        raise PlanningLeafMergeError(
            "UNSAFE_SHARED_INDEX_POLICY",
            "Automated leaf merge requires leaf_shared_index_mutation: FORBIDDEN.",
        )

    try:
        entrypoint = _leaf_entrypoint(root_plan, branch)
    except PlanningStateReconciliationError as exc:
        raise PlanningLeafMergeError(exc.code, str(exc)) from exc

    if entrypoint.get("state") != "ACTIVE":
        raise PlanningLeafMergeError(
            "LEAF_ENTRYPOINT_NOT_ACTIVE",
            f"Leaf branch {branch!r} is already merged or is not an active leaf entrypoint.",
        )
    work_unit = entrypoint.get("work_unit")
    if not isinstance(work_unit, str):
        raise PlanningLeafMergeError(
            "LEAF_WORK_UNIT_MISSING",
            f"Leaf branch {branch!r} has no work-unit binding.",
        )

    _require_clean_worktree(base)
    source_ref = _resolve_source_ref(base, branch)
    _assert_leaf_did_not_edit_shared_planning(base, source_ref, root_plan)

    merge = _git(base, "merge", "--no-ff", "--no-commit", source_ref)
    if merge.returncode != 0:
        _abort_merge(base)
        raise PlanningLeafMergeError(
            "LEAF_MERGE_FAILED",
            merge.stderr.strip() or merge.stdout.strip() or f"git merge {source_ref!r} failed",
        )

    merge_head = _git(base, "rev-parse", "-q", "--verify", "MERGE_HEAD")
    if merge_head.returncode != 0:
        raise PlanningLeafMergeError(
            "LEAF_ALREADY_INTEGRATED",
            f"Leaf branch {branch!r} produced no merge state; it may already be integrated.",
        )

    try:
        reconciliation = reconcile_planning_state(
            root=base,
            merged_branch=branch,
            apply=True,
            current_branch=integration_branch,
        )

        version_index = root_plan.get("path")
        if not isinstance(version_index, str):
            raise PlanningLeafMergeError(
                "VERSION_INDEX_PATH_INVALID",
                "Planning root entry has no valid version-index path.",
            )
        stage = _git(base, "add", "--", ROOT_INDEX, version_index)
        if stage.returncode != 0:
            raise PlanningLeafMergeError(
                "PLANNING_STAGE_FAILED",
                stage.stderr.strip() or "Unable to stage reconciled planning indexes.",
            )

        commit_message = message or f"merge: integrate {branch} with planning reconciliation"
        commit = _git(base, "commit", "-m", commit_message)
        if commit.returncode != 0:
            raise PlanningLeafMergeError(
                "MERGE_COMMIT_FAILED",
                commit.stderr.strip() or commit.stdout.strip() or "Unable to create merge commit.",
            )
    except Exception:
        _abort_merge(base)
        raise

    head = _git(base, "rev-parse", "HEAD")
    if head.returncode != 0:
        raise PlanningLeafMergeError(
            "MERGE_COMMIT_RESOLUTION_FAILED",
            head.stderr.strip() or "Unable to resolve merge commit SHA.",
        )

    return PlanningLeafMergeResult(
        status="MERGED",
        source_branch=branch,
        source_ref=source_ref,
        integration_branch=integration_branch,
        merged_work_unit=work_unit,
        merge_commit=head.stdout.strip(),
        current_gate=reconciliation.current_gate_after,
    )


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description=(
            "Merge an independent WU leaf into its integration branch while "
            "reconciling parent-owned planning indexes in the same merge commit."
        )
    )
    parser.add_argument("branch", help="Exact independent leaf branch to merge.")
    parser.add_argument("--root", help="Repository root override for tests or tooling.")
    parser.add_argument("--message", help="Optional merge commit message.")
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        result = merge_leaf_branch(
            args.branch,
            root=args.root,
            message=args.message,
        )
    except (PlanningLeafMergeError, PlanningStateReconciliationError) as exc:
        code = getattr(exc, "code", "PLANNING_LEAF_MERGE_FAILED")
        print(
            json.dumps(
                {
                    "status": "UNMERGED",
                    "code": code,
                    "message": str(exc),
                },
                sort_keys=True,
            )
        )
        return 2

    print(json.dumps(result.to_payload(), sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
