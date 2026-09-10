from __future__ import annotations

import argparse
import copy
import json
import os
import subprocess
import tempfile
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any, Sequence

import yaml

from developer.automation.policy_loader import load_yaml, repository_root


ROOT_INDEX = "docs/planning/index.yaml"
TERMINAL_WU_STATUSES = {"COMPLETE", "SUPERSEDED", "CANCELLED"}
NONTERMINAL_WU_STATUSES = {"DRAFT", "ACTIVE", "BLOCKED"}
DEPENDENCY_SATISFIED_STATUSES = {"COMPLETE"}


class PlanningStateReconciliationError(RuntimeError):
    """Raised when planning state cannot be reconciled deterministically."""

    def __init__(self, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code


@dataclass(frozen=True)
class PlanningStateReconciliation:
    status: str
    integration_branch: str
    merged_branch: str | None
    merged_work_unit: str | None
    current_gate_before: str
    current_gate_after: str
    current_gate_document: str
    version_index_revision_before: str
    version_index_revision_after: str
    changed: bool

    def to_payload(self) -> dict[str, str | bool | None]:
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


def detect_current_branch(root: str | Path | None = None) -> str:
    base = repository_root(root)
    completed = _git(base, "branch", "--show-current")
    if completed.returncode != 0:
        detail = completed.stderr.strip() or "git branch --show-current failed"
        raise PlanningStateReconciliationError("BRANCH_DETECTION_FAILED", detail)
    branch = completed.stdout.strip()
    if not branch:
        raise PlanningStateReconciliationError(
            "DETACHED_HEAD",
            "Current Git state has no branch name; planning state reconciliation fails closed.",
        )
    return branch


def _find_plan_for_integration_branch(
    root_index: dict[str, Any],
    integration_branch: str,
) -> dict[str, Any]:
    matches = [
        plan
        for plan in root_index.get("plans", [])
        if isinstance(plan, dict) and plan.get("integration_branch") == integration_branch
    ]
    if not matches:
        raise PlanningStateReconciliationError(
            "UNKNOWN_INTEGRATION_BRANCH",
            f"No planning root entry is declared for integration branch {integration_branch!r}.",
        )
    if len(matches) != 1:
        raise PlanningStateReconciliationError(
            "AMBIGUOUS_INTEGRATION_BRANCH",
            f"Multiple planning root entries are declared for integration branch {integration_branch!r}.",
        )
    return matches[0]


def _wu_by_id(version_index: dict[str, Any]) -> dict[str, dict[str, Any]]:
    return {
        str(entry["id"]): entry
        for entry in version_index.get("work_units", [])
        if isinstance(entry, dict) and isinstance(entry.get("id"), str)
    }


def _load_work_unit_documents(
    version_index: dict[str, Any],
    *,
    base: Path,
) -> dict[str, dict[str, Any]]:
    result: dict[str, dict[str, Any]] = {}
    for entry in version_index.get("work_units", []):
        if not isinstance(entry, dict):
            continue
        work_unit_id = entry.get("id")
        path = entry.get("path")
        if not isinstance(work_unit_id, str) or not isinstance(path, str):
            continue
        payload = load_yaml(path, root=base)
        declared_id = payload.get("work_unit", {}).get("id")
        if declared_id != work_unit_id:
            raise PlanningStateReconciliationError(
                "WORK_UNIT_ID_MISMATCH",
                f"{path}: work_unit.id {declared_id!r} does not match index id {work_unit_id!r}.",
            )
        result[work_unit_id] = payload
    return result


def _copy_work_unit_state_into_version_index(
    version_index: dict[str, Any],
    work_unit_documents: dict[str, dict[str, Any]],
) -> None:
    for index_entry in version_index.get("work_units", []):
        if not isinstance(index_entry, dict):
            continue
        work_unit_id = index_entry.get("id")
        if not isinstance(work_unit_id, str):
            continue
        payload = work_unit_documents.get(work_unit_id)
        if payload is None:
            continue
        work_unit = payload.get("work_unit", {})
        lifecycle = work_unit.get("lifecycle", {})
        approval = work_unit.get("approval", {})
        implementation_authorization = work_unit.get("implementation_authorization")
        depends_on = work_unit.get("depends_on", [])

        lifecycle_status = lifecycle.get("status")
        approval_status = approval.get("status")
        inherited_from = approval.get("inherited_from")
        if not isinstance(lifecycle_status, str):
            raise PlanningStateReconciliationError(
                "INVALID_WORK_UNIT_LIFECYCLE",
                f"{work_unit_id}: lifecycle.status is missing or invalid.",
            )
        if not isinstance(approval_status, str):
            raise PlanningStateReconciliationError(
                "INVALID_WORK_UNIT_APPROVAL",
                f"{work_unit_id}: approval.status is missing or invalid.",
            )
        if not isinstance(inherited_from, list):
            raise PlanningStateReconciliationError(
                "INVALID_WORK_UNIT_APPROVAL_PROVENANCE",
                f"{work_unit_id}: approval.inherited_from is missing or invalid.",
            )
        if not isinstance(implementation_authorization, str):
            raise PlanningStateReconciliationError(
                "INVALID_WORK_UNIT_AUTHORIZATION",
                f"{work_unit_id}: implementation_authorization is missing or invalid.",
            )
        if not isinstance(depends_on, list):
            raise PlanningStateReconciliationError(
                "INVALID_WORK_UNIT_DEPENDENCIES",
                f"{work_unit_id}: depends_on is missing or invalid.",
            )

        index_entry["lifecycle"] = {"status": lifecycle_status}
        index_entry["approval"] = {
            "status": approval_status,
            "inherited_from": list(inherited_from),
        }
        index_entry["implementation_authorization"] = {
            "status": implementation_authorization
        }
        index_entry["depends_on"] = list(depends_on)

        if lifecycle_status == "COMPLETE":
            completion_evidence = payload.get("completion_evidence")
            if not isinstance(completion_evidence, list) or not completion_evidence:
                raise PlanningStateReconciliationError(
                    "MISSING_COMPLETION_EVIDENCE",
                    f"{work_unit_id}: COMPLETE work unit requires completion_evidence before parent reconciliation.",
                )
            index_entry["completion_evidence"] = copy.deepcopy(completion_evidence)
        else:
            index_entry.pop("completion_evidence", None)


def _gate_parent_id(gate: str) -> str:
    return gate.split("-P", 1)[0] if "-P" in gate else gate


def resolve_gate_document(
    gate: str,
    version_index: dict[str, Any],
    *,
    base: Path,
    work_unit_documents: dict[str, dict[str, Any]] | None = None,
) -> tuple[str, str]:
    indexed = _wu_by_id(version_index)
    if "-P" not in gate:
        work_unit = indexed.get(gate)
        if work_unit is None:
            raise PlanningStateReconciliationError(
                "UNKNOWN_CURRENT_GATE",
                f"Current gate {gate!r} is not indexed.",
            )
        path = work_unit.get("path")
        if not isinstance(path, str):
            raise PlanningStateReconciliationError(
                "CURRENT_GATE_HAS_NO_DOCUMENT",
                f"Current gate {gate!r} has no planning document.",
            )
        return str(work_unit.get("lifecycle", {}).get("status")), path

    parent_id = _gate_parent_id(gate)
    documents = work_unit_documents or _load_work_unit_documents(version_index, base=base)
    parent = documents.get(parent_id)
    if parent is None:
        raise PlanningStateReconciliationError(
            "CURRENT_GATE_PARENT_MISSING",
            f"Current gate {gate!r} parent {parent_id!r} has no work-unit document.",
        )
    matches = [
        extension
        for extension in parent.get("extensions", [])
        if isinstance(extension, dict) and extension.get("id") == gate
    ]
    if len(matches) != 1:
        raise PlanningStateReconciliationError(
            "CURRENT_GATE_EXTENSION_UNRESOLVED",
            f"Current gate {gate!r} must resolve to exactly one Plan Extension.",
        )
    extension_path = matches[0].get("path")
    if not isinstance(extension_path, str):
        raise PlanningStateReconciliationError(
            "CURRENT_GATE_EXTENSION_PATH_INVALID",
            f"Current gate {gate!r} has no valid extension path.",
        )
    extension_payload = load_yaml(extension_path, root=base)
    status = extension_payload.get("extension", {}).get("lifecycle", {}).get("status")
    if not isinstance(status, str):
        raise PlanningStateReconciliationError(
            "CURRENT_GATE_EXTENSION_STATUS_INVALID",
            f"Current gate {gate!r} extension lifecycle status is invalid.",
        )
    return status, extension_path


def _dependencies_satisfied(
    work_unit: dict[str, Any],
    indexed: dict[str, dict[str, Any]],
) -> bool:
    dependencies = work_unit.get("depends_on", [])
    if not isinstance(dependencies, list):
        return False
    for dependency in dependencies:
        dependency_entry = indexed.get(dependency)
        if dependency_entry is None:
            return False
        status = dependency_entry.get("lifecycle", {}).get("status")
        if status not in DEPENDENCY_SATISFIED_STATUSES:
            return False
    return True


def select_current_gate(
    version_index: dict[str, Any],
    root_plan: dict[str, Any],
    *,
    base: Path,
    work_unit_documents: dict[str, dict[str, Any]],
    merged_work_unit: str | None = None,
) -> tuple[str, str]:
    current_gate = version_index.get("plan", {}).get("current_gate")
    if not isinstance(current_gate, str):
        raise PlanningStateReconciliationError(
            "INVALID_CURRENT_GATE",
            "Version planning index has no valid current_gate.",
        )

    current_status, current_document = resolve_gate_document(
        current_gate,
        version_index,
        base=base,
        work_unit_documents=work_unit_documents,
    )
    if current_status in NONTERMINAL_WU_STATUSES:
        return current_gate, current_document

    indexed = _wu_by_id(version_index)
    parent_id = _gate_parent_id(current_gate)
    if "-P" in current_gate:
        parent = indexed.get(parent_id)
        if parent is not None:
            parent_status = parent.get("lifecycle", {}).get("status")
            parent_path = parent.get("path")
            if parent_status in NONTERMINAL_WU_STATUSES and isinstance(parent_path, str):
                return parent_id, parent_path

    if merged_work_unit is not None:
        merged = indexed.get(merged_work_unit)
        if merged is None:
            raise PlanningStateReconciliationError(
                "MERGED_WORK_UNIT_NOT_INDEXED",
                f"Merged work unit {merged_work_unit!r} is not indexed.",
            )
        merged_status = merged.get("lifecycle", {}).get("status")
        merged_path = merged.get("path")
        if merged_status in NONTERMINAL_WU_STATUSES and isinstance(merged_path, str):
            return merged_work_unit, merged_path

    routing = root_plan.get("entry_routing", {})
    entrypoints = {
        entry.get("branch"): entry
        for entry in routing.get("branch_entrypoints", [])
        if isinstance(entry, dict) and isinstance(entry.get("branch"), str)
    }
    leaf_routes = [
        route
        for route in routing.get("independent_leaf_work_units", [])
        if isinstance(route, dict)
    ]

    for expected_state in ("MERGED", "ACTIVE"):
        for route in leaf_routes:
            work_unit_id = route.get("id")
            branch = route.get("branch")
            if not isinstance(work_unit_id, str) or not isinstance(branch, str):
                continue
            entrypoint = entrypoints.get(branch)
            work_unit = indexed.get(work_unit_id)
            if entrypoint is None or work_unit is None:
                continue
            if entrypoint.get("state") != expected_state:
                continue
            lifecycle_status = work_unit.get("lifecycle", {}).get("status")
            path = work_unit.get("path")
            if lifecycle_status in NONTERMINAL_WU_STATUSES and isinstance(path, str):
                return work_unit_id, path

    convergence_id = routing.get("dependency_bearing_convergence", {}).get("id")
    ordered_ids: list[str] = []
    if isinstance(convergence_id, str):
        ordered_ids.append(convergence_id)
    ordered_ids.extend(
        work_unit_id
        for work_unit_id in indexed
        if work_unit_id not in ordered_ids
    )

    for work_unit_id in ordered_ids:
        work_unit = indexed[work_unit_id]
        lifecycle_status = work_unit.get("lifecycle", {}).get("status")
        path = work_unit.get("path")
        if lifecycle_status not in NONTERMINAL_WU_STATUSES or not isinstance(path, str):
            continue
        if _dependencies_satisfied(work_unit, indexed):
            return work_unit_id, path

    raise PlanningStateReconciliationError(
        "NO_CURRENT_GATE_CANDIDATE",
        "No deterministic nonterminal planning gate is available after reconciliation.",
    )


def _leaf_entrypoint(
    root_plan: dict[str, Any],
    branch: str,
) -> dict[str, Any]:
    routing = root_plan.get("entry_routing", {})
    matches = [
        entry
        for entry in routing.get("branch_entrypoints", [])
        if isinstance(entry, dict)
        and entry.get("branch") == branch
        and entry.get("role") == "INDEPENDENT_LEAF"
    ]
    if not matches:
        raise PlanningStateReconciliationError(
            "UNKNOWN_MERGED_LEAF_BRANCH",
            f"No independent leaf planning entrypoint is declared for branch {branch!r}.",
        )
    if len(matches) != 1:
        raise PlanningStateReconciliationError(
            "AMBIGUOUS_MERGED_LEAF_BRANCH",
            f"Multiple independent leaf planning entrypoints are declared for branch {branch!r}.",
        )
    return matches[0]


def mark_leaf_merged(
    root_plan: dict[str, Any],
    *,
    branch: str,
) -> str:
    entrypoint = _leaf_entrypoint(root_plan, branch)
    if entrypoint.get("state") not in {"ACTIVE", "MERGED"}:
        raise PlanningStateReconciliationError(
            "INVALID_LEAF_ENTRYPOINT_STATE",
            f"Leaf branch {branch!r} has invalid entrypoint state {entrypoint.get('state')!r}.",
        )
    work_unit_id = entrypoint.get("work_unit")
    if not isinstance(work_unit_id, str):
        raise PlanningStateReconciliationError(
            "LEAF_ENTRYPOINT_WORK_UNIT_MISSING",
            f"Leaf branch {branch!r} has no work_unit binding.",
        )
    entrypoint["state"] = "MERGED"
    entrypoint["merged_into"] = str(root_plan["integration_branch"])
    return work_unit_id


def build_materialized_state(
    root_plan: dict[str, Any],
    version_index: dict[str, Any],
    *,
    base: Path,
    work_unit_documents: dict[str, dict[str, Any]] | None = None,
) -> dict[str, Any]:
    documents = work_unit_documents or _load_work_unit_documents(version_index, base=base)
    gate = version_index.get("plan", {}).get("current_gate")
    if not isinstance(gate, str):
        raise PlanningStateReconciliationError(
            "INVALID_CURRENT_GATE",
            "Version planning index has no valid current_gate.",
        )
    _, gate_document = resolve_gate_document(
        gate,
        version_index,
        base=base,
        work_unit_documents=documents,
    )

    routing = root_plan.get("entry_routing", {})
    integration_branch = root_plan.get("integration_branch")
    if not isinstance(integration_branch, str):
        raise PlanningStateReconciliationError(
            "INVALID_INTEGRATION_BRANCH",
            "Root planning entry has no valid integration_branch.",
        )
    leaf_route_by_id = {
        route.get("id"): route
        for route in routing.get("independent_leaf_work_units", [])
        if isinstance(route, dict) and isinstance(route.get("id"), str)
    }
    entrypoint_by_branch = {
        entry.get("branch"): entry
        for entry in routing.get("branch_entrypoints", [])
        if isinstance(entry, dict) and isinstance(entry.get("branch"), str)
    }

    work_units: list[dict[str, Any]] = []
    for indexed in version_index.get("work_units", []):
        if not isinstance(indexed, dict):
            continue
        work_unit_id = indexed.get("id")
        if not isinstance(work_unit_id, str):
            continue
        lifecycle_status = indexed.get("lifecycle", {}).get("status")
        authorization_status = indexed.get("implementation_authorization", {}).get("status")
        if not isinstance(lifecycle_status, str) or not isinstance(authorization_status, str):
            raise PlanningStateReconciliationError(
                "INVALID_INDEXED_WORK_UNIT_STATE",
                f"{work_unit_id}: version-index lifecycle or authorization state is invalid.",
            )

        record: dict[str, Any] = {
            "id": work_unit_id,
            "lifecycle_status": lifecycle_status,
            "implementation_authorization_status": authorization_status,
        }
        path = indexed.get("path")
        leaf_route = leaf_route_by_id.get(work_unit_id)
        if lifecycle_status == "NOT_CREATED":
            record["execution_location"] = "NOT_CREATED"
        elif isinstance(leaf_route, dict):
            leaf_branch = leaf_route.get("branch")
            entrypoint = entrypoint_by_branch.get(leaf_branch)
            if not isinstance(leaf_branch, str) or not isinstance(entrypoint, dict):
                raise PlanningStateReconciliationError(
                    "LEAF_ROUTING_INCOMPLETE",
                    f"{work_unit_id}: leaf routing is incomplete.",
                )
            if entrypoint.get("state") == "MERGED":
                record["execution_location"] = "INTEGRATION_BRANCH"
                record["branch"] = integration_branch
            else:
                record["execution_location"] = "LEAF_BRANCH"
                record["branch"] = leaf_branch
        else:
            record["execution_location"] = "INTEGRATION_BRANCH"
            record["branch"] = integration_branch

        if isinstance(path, str):
            record["entry_document"] = path
        work_units.append(record)

    return {
        "mode": "DERIVED_CACHE",
        "source": str(root_plan["path"]),
        "generated_by": "developer.automation.planning_merge_reconciler",
        "current_gate": gate,
        "current_gate_document": gate_document,
        "work_units": work_units,
    }


def _next_revision(revision: str) -> str:
    if len(revision) != 2 or not revision.isdigit():
        raise PlanningStateReconciliationError(
            "INVALID_PLAN_REVISION",
            f"Plan revision {revision!r} is not a two-digit revision.",
        )
    value = int(revision)
    if value >= 99:
        raise PlanningStateReconciliationError(
            "PLAN_REVISION_EXHAUSTED",
            "Two-digit planning revision space is exhausted.",
        )
    return f"{value + 1:02d}"


def _dump_yaml(payload: dict[str, Any]) -> str:
    return yaml.safe_dump(
        payload,
        sort_keys=False,
        allow_unicode=True,
        width=120,
    )


def _atomic_write(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    handle, temporary_name = tempfile.mkstemp(
        prefix=f".{path.name}.",
        suffix=".tmp",
        dir=str(path.parent),
        text=True,
    )
    temporary = Path(temporary_name)
    try:
        with os.fdopen(handle, "w", encoding="utf-8", newline="\n") as stream:
            stream.write(content)
        temporary.replace(path)
    finally:
        if temporary.exists():
            temporary.unlink()


def reconcile_planning_state(
    *,
    root: str | Path | None = None,
    merged_branch: str | None = None,
    apply: bool = False,
    current_branch: str | None = None,
) -> PlanningStateReconciliation:
    base = repository_root(root)
    root_path = base / ROOT_INDEX
    root_text_before = root_path.read_text(encoding="utf-8")
    root_index = load_yaml(ROOT_INDEX, root=base)

    active_branch = current_branch or detect_current_branch(base)
    root_plan = _find_plan_for_integration_branch(root_index, active_branch)
    routing = root_plan.get("entry_routing", {})
    reconciliation = routing.get("merge_reconciliation", {})
    target_branch = reconciliation.get("target_branch")
    if target_branch != active_branch:
        raise PlanningStateReconciliationError(
            "RECONCILIATION_WRONG_BRANCH",
            f"Planning reconciliation may run only on target integration branch {target_branch!r}; current branch is {active_branch!r}.",
        )

    version_path_value = root_plan.get("path")
    if not isinstance(version_path_value, str):
        raise PlanningStateReconciliationError(
            "VERSION_INDEX_PATH_INVALID",
            "Root planning entry has no valid version index path.",
        )
    version_path = base / version_path_value
    version_text_before = version_path.read_text(encoding="utf-8")
    version_index = load_yaml(version_path_value, root=base)
    version_before = copy.deepcopy(version_index)
    root_before = copy.deepcopy(root_index)

    work_unit_documents = _load_work_unit_documents(version_index, base=base)
    _copy_work_unit_state_into_version_index(version_index, work_unit_documents)

    merged_work_unit: str | None = None
    if merged_branch is not None:
        merged_work_unit = mark_leaf_merged(root_plan, branch=merged_branch)

    current_gate_before = str(version_before.get("plan", {}).get("current_gate"))
    current_gate_after, current_gate_document = select_current_gate(
        version_index,
        root_plan,
        base=base,
        work_unit_documents=work_unit_documents,
        merged_work_unit=merged_work_unit,
    )
    version_index["plan"]["current_gate"] = current_gate_after

    root_plan["materialized_state"] = build_materialized_state(
        root_plan,
        version_index,
        base=base,
        work_unit_documents=work_unit_documents,
    )

    changed_without_revision = root_index != root_before or version_index != version_before
    revision_before = str(version_before.get("plan", {}).get("revision"))
    revision_after = revision_before
    if changed_without_revision:
        revision_after = _next_revision(revision_before)
        version_index["plan"]["revision"] = revision_after

    root_text_after = _dump_yaml(root_index)
    version_text_after = _dump_yaml(version_index)
    changed = (
        root_text_after != root_text_before
        or version_text_after != version_text_before
    )

    if apply and changed:
        _atomic_write(root_path, root_text_after)
        _atomic_write(version_path, version_text_after)
        try:
            from developer.automation.planning_validator import validate_planning

            failures = validate_planning(base)
            if failures:
                raise PlanningStateReconciliationError(
                    "POST_RECONCILIATION_VALIDATION_FAILED",
                    "\n".join(failures),
                )
        except Exception:
            _atomic_write(root_path, root_text_before)
            _atomic_write(version_path, version_text_before)
            raise

    return PlanningStateReconciliation(
        status="RECONCILED" if apply else "PREVIEW",
        integration_branch=active_branch,
        merged_branch=merged_branch,
        merged_work_unit=merged_work_unit,
        current_gate_before=current_gate_before,
        current_gate_after=current_gate_after,
        current_gate_document=current_gate_document,
        version_index_revision_before=revision_before,
        version_index_revision_after=revision_after,
        changed=changed,
    )


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description=(
            "Reconcile parent planning indexes from canonical work-unit documents "
            "after a leaf merge or parent work-unit state change."
        )
    )
    parser.add_argument(
        "--merged-branch",
        help="Exact independent leaf branch that has just been merged into the integration branch.",
    )
    parser.add_argument(
        "--apply",
        action="store_true",
        help="Write the reconciled planning indexes. Without this flag, print a preview only.",
    )
    parser.add_argument(
        "--root",
        help="Repository root override for tests or tooling.",
    )
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        result = reconcile_planning_state(
            root=args.root,
            merged_branch=args.merged_branch,
            apply=args.apply,
        )
    except PlanningStateReconciliationError as exc:
        print(
            json.dumps(
                {
                    "status": "UNRESOLVED",
                    "code": exc.code,
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
