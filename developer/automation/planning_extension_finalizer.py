from __future__ import annotations

import re
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Mapping

from developer.automation.planning_merge_reconciler import (
    PlanningStateReconciliationError,
    reconcile_planning_state,
)
from developer.automation.policy_loader import load_yaml, repository_root


_COMPLETE = "COMPLETE"
_ACTIVE = "ACTIVE"
_AUTHORIZED = "AUTHORIZED"
_PLAN_EXTENSION_SCHEMA = "ptsip-plan-extension/v1"
_VALIDATION_PASS_VALUES = {"PASS", "AUTHORIZED", "COMPLETE"}


@dataclass(frozen=True)
class ExtensionFinalizationResult:
    extension_id: str | None
    eligible: bool
    finalized: bool
    current_gate_before: str | None = None
    current_gate_after: str | None = None
    failures: tuple[str, ...] = ()


def _status_records(payload: Mapping[str, object]) -> tuple[Mapping[str, object], ...]:
    records: list[Mapping[str, object]] = []

    migration_stages = payload.get("migration_stages")
    if isinstance(migration_stages, Mapping):
        records.extend(
            value
            for value in migration_stages.values()
            if isinstance(value, Mapping) and isinstance(value.get("status"), str)
        )

    for key, value in payload.items():
        if not isinstance(key, str) or not key.endswith("_execution_plan"):
            continue
        if not isinstance(value, Mapping):
            continue
        execution_order = value.get("execution_order")
        if not isinstance(execution_order, list):
            continue
        records.extend(
            item
            for item in execution_order
            if isinstance(item, Mapping) and isinstance(item.get("status"), str)
        )

    return tuple(records)


def _validation_is_complete(record: Mapping[str, object]) -> bool:
    validation = record.get("validation")
    if validation is None:
        return True
    if not isinstance(validation, Mapping):
        return False

    saw_machine_result = False
    for key in ("status", "result", "state"):
        value = validation.get(key)
        if value is None:
            continue
        saw_machine_result = True
        if not isinstance(value, str) or value not in _VALIDATION_PASS_VALUES:
            return False
    return saw_machine_result


def extension_is_machine_ready(payload: Mapping[str, object]) -> bool:
    """Return True only for a structurally complete Plan Extension.

    This is deliberately fail-closed. Extensions without explicit blocker state and
    machine-status-bearing completion records are not auto-closed.
    """

    if payload.get("schema_version") != _PLAN_EXTENSION_SCHEMA:
        return False

    extension = payload.get("extension")
    authorization = payload.get("implementation_authorization")
    blockers = payload.get("current_known_blockers")
    if not isinstance(extension, Mapping) or not isinstance(authorization, Mapping):
        return False
    lifecycle = extension.get("lifecycle")
    if not isinstance(lifecycle, Mapping):
        return False
    if lifecycle.get("status") not in {_ACTIVE, _COMPLETE}:
        return False
    if authorization.get("status") not in {_AUTHORIZED, _COMPLETE}:
        return False
    if not isinstance(blockers, list) or blockers:
        return False

    records = _status_records(payload)
    if not records:
        return False
    return all(
        record.get("status") == _COMPLETE and _validation_is_complete(record)
        for record in records
    )


def _top_level_section_span(text: str, section: str) -> tuple[int, int]:
    marker = re.compile(rf"(?m)^{re.escape(section)}:[ \t]*$")
    match = marker.search(text)
    if match is None:
        raise ValueError(f"top-level section not found: {section}")
    next_top = re.compile(r"(?m)^[A-Za-z0-9_][A-Za-z0-9_-]*:[ \t]*$")
    following = next_top.search(text, match.end())
    return match.start(), following.start() if following else len(text)


def _replace_extension_lifecycle(text: str, old: str, new: str) -> str:
    start, end = _top_level_section_span(text, "extension")
    block = text[start:end]
    pattern = re.compile(
        rf"(?m)^(  lifecycle:[ \t]*\n    status:) {re.escape(old)}[ \t]*$"
    )
    block, count = pattern.subn(rf"\1 {new}", block, count=1)
    if count != 1:
        if re.search(rf"(?m)^    status: {re.escape(new)}[ \t]*$", block):
            return text
        raise ValueError(f"extension lifecycle is not {old}")
    return text[:start] + block + text[end:]


def _replace_authorization_status(text: str, old: str, new: str) -> str:
    start, end = _top_level_section_span(text, "implementation_authorization")
    block = text[start:end]
    pattern = re.compile(rf"(?m)^(  status:) {re.escape(old)}[ \t]*$")
    block, count = pattern.subn(rf"\1 {new}", block, count=1)
    if count != 1:
        if re.search(rf"(?m)^  status: {re.escape(new)}[ \t]*$", block):
            return text
        raise ValueError(f"extension implementation authorization is not {old}")
    return text[:start] + block + text[end:]


def _replace_parent_extension_status(
    text: str,
    *,
    extension_id: str,
    old: str,
    new: str,
) -> str:
    start, end = _top_level_section_span(text, "extensions")
    section = text[start:end]
    marker = re.compile(
        rf"(?m)^(?P<indent>[ \t]*)- id: {re.escape(extension_id)}[ \t]*$"
    )
    match = marker.search(section)
    if match is None:
        raise ValueError(f"parent extension entry not found: {extension_id}")
    indent = match.group("indent")
    next_marker = re.compile(rf"(?m)^{re.escape(indent)}- id: ")
    following = next_marker.search(section, match.end())
    block_end = following.start() if following else len(section)
    block = section[match.start():block_end]
    status = re.compile(
        rf"(?m)^{re.escape(indent)}  status: {re.escape(old)}[ \t]*$"
    )
    block, count = status.subn(f"{indent}  status: {new}", block, count=1)
    if count != 1:
        if re.search(
            rf"(?m)^{re.escape(indent)}  status: {re.escape(new)}[ \t]*$",
            block,
        ):
            return text
        raise ValueError(f"parent extension status is not {old}: {extension_id}")
    section = section[:match.start()] + block + section[block_end:]
    return text[:start] + section + text[end:]


def extension_parent_consistency_errors(
    parent_payload: Mapping[str, object],
    extension_payload: Mapping[str, object],
    *,
    extension_id: str,
    extension_path: str,
) -> tuple[str, ...]:
    errors: list[str] = []
    extension = extension_payload.get("extension")
    if not isinstance(extension, Mapping):
        return (f"{extension_path}: extension metadata is missing",)

    if extension.get("id") != extension_id:
        errors.append(
            f"{extension_path}: extension.id does not match parent entry {extension_id!r}"
        )
    parent_id = parent_payload.get("work_unit", {}).get("id") if isinstance(parent_payload.get("work_unit"), Mapping) else None
    if extension.get("parent") != parent_id:
        errors.append(
            f"{extension_path}: extension.parent does not match parent work unit {parent_id!r}"
        )

    entries = [
        item
        for item in parent_payload.get("extensions", [])
        if isinstance(item, Mapping) and item.get("id") == extension_id
    ]
    if len(entries) != 1:
        errors.append(
            f"{extension_path}: parent must declare extension {extension_id!r} exactly once"
        )
        return tuple(errors)
    parent_entry = entries[0]
    if parent_entry.get("path") != extension_path:
        errors.append(
            f"{extension_path}: parent extension path does not match canonical path"
        )
    lifecycle = extension.get("lifecycle")
    lifecycle_status = lifecycle.get("status") if isinstance(lifecycle, Mapping) else None
    if parent_entry.get("status") != lifecycle_status:
        errors.append(
            f"{extension_path}: parent extension status {parent_entry.get('status')!r} "
            f"does not match extension lifecycle {lifecycle_status!r}"
        )
    return tuple(errors)


def _resolve_parent_context(
    payload: Mapping[str, object],
    *,
    base: Path,
    relative_path: str,
) -> tuple[str, str, str]:
    extension = payload.get("extension")
    plan_version = payload.get("plan_version")
    if not isinstance(extension, Mapping):
        raise ValueError("extension metadata is missing")
    extension_id = extension.get("id")
    parent_id = extension.get("parent")
    if not all(isinstance(value, str) and value for value in (extension_id, parent_id, plan_version)):
        raise ValueError("extension id, parent, and plan_version must be canonical strings")

    root_index = load_yaml("docs/planning/index.yaml", root=base)
    plan_matches = [
        item
        for item in root_index.get("plans", [])
        if isinstance(item, Mapping) and item.get("plan_version") == plan_version
    ]
    if len(plan_matches) != 1:
        raise ValueError(f"plan version {plan_version!r} must resolve exactly once")
    root_plan = plan_matches[0]
    version_path = root_plan.get("path")
    integration_branch = root_plan.get("integration_branch")
    if not isinstance(version_path, str) or not isinstance(integration_branch, str):
        raise ValueError("plan path or integration branch is invalid")

    version_index = load_yaml(version_path, root=base)
    parent_matches = [
        item
        for item in version_index.get("work_units", [])
        if isinstance(item, Mapping) and item.get("id") == parent_id
    ]
    if len(parent_matches) != 1:
        raise ValueError(f"parent work unit {parent_id!r} must resolve exactly once")
    parent_path = parent_matches[0].get("path")
    if not isinstance(parent_path, str):
        raise ValueError(f"parent work unit {parent_id!r} has no planning path")

    parent_payload = load_yaml(parent_path, root=base)
    errors = extension_parent_consistency_errors(
        parent_payload,
        payload,
        extension_id=extension_id,
        extension_path=relative_path,
    )
    # ACTIVE/ACTIVE is the expected pre-closure state. Other inconsistencies fail closed.
    if errors:
        lifecycle = extension.get("lifecycle")
        lifecycle_status = lifecycle.get("status") if isinstance(lifecycle, Mapping) else None
        parent_entries = [
            item
            for item in parent_payload.get("extensions", [])
            if isinstance(item, Mapping) and item.get("id") == extension_id
        ]
        recoverable = (
            len(parent_entries) == 1
            and parent_entries[0].get("path") == relative_path
            and parent_entries[0].get("status") == _ACTIVE
            and lifecycle_status == _COMPLETE
        )
        if not recoverable:
            raise ValueError("\n".join(errors))
    return parent_path, version_path, integration_branch


def finalize_extension_if_ready(
    relative_path: str,
    *,
    root: str | Path | None = None,
) -> ExtensionFinalizationResult:
    base = repository_root(root)
    path = (base / relative_path).resolve()
    if base not in path.parents or not path.is_file():
        return ExtensionFinalizationResult(
            None, False, False, failures=(f"invalid extension path: {relative_path}",)
        )

    payload = load_yaml(relative_path, root=base)
    if payload.get("schema_version") != _PLAN_EXTENSION_SCHEMA:
        return ExtensionFinalizationResult(None, False, False)
    extension = payload.get("extension")
    extension_id = extension.get("id") if isinstance(extension, Mapping) else None
    if not isinstance(extension_id, str):
        return ExtensionFinalizationResult(
            None, False, False, failures=("extension.id is missing",)
        )
    if not extension_is_machine_ready(payload):
        return ExtensionFinalizationResult(extension_id, False, False)

    try:
        parent_path, version_path, integration_branch = _resolve_parent_context(
            payload,
            base=base,
            relative_path=relative_path,
        )
    except ValueError as exc:
        return ExtensionFinalizationResult(
            extension_id, True, False, failures=(str(exc),)
        )

    parent = base / parent_path
    root_index = base / "docs/planning/index.yaml"
    version_index = base / version_path
    originals = {
        path: path.read_text(encoding="utf-8"),
        parent: parent.read_text(encoding="utf-8"),
        root_index: root_index.read_text(encoding="utf-8"),
        version_index: version_index.read_text(encoding="utf-8"),
    }

    lifecycle = extension.get("lifecycle")
    lifecycle_status = lifecycle.get("status") if isinstance(lifecycle, Mapping) else None
    authorization = payload.get("implementation_authorization")
    authorization_status = authorization.get("status") if isinstance(authorization, Mapping) else None

    try:
        extension_text = originals[path]
        if lifecycle_status == _ACTIVE:
            extension_text = _replace_extension_lifecycle(
                extension_text, _ACTIVE, _COMPLETE
            )
        if authorization_status == _AUTHORIZED:
            extension_text = _replace_authorization_status(
                extension_text, _AUTHORIZED, _COMPLETE
            )
        parent_text = _replace_parent_extension_status(
            originals[parent],
            extension_id=extension_id,
            old=_ACTIVE,
            new=_COMPLETE,
        )
        path.write_text(extension_text, encoding="utf-8")
        parent.write_text(parent_text, encoding="utf-8")

        reconciliation = reconcile_planning_state(
            root=base,
            apply=True,
            current_branch=integration_branch,
        )
        if (
            reconciliation.current_gate_before == extension_id
            and reconciliation.current_gate_after != extension.get("parent")
        ):
            raise PlanningStateReconciliationError(
                "EXTENSION_GATE_NOT_RELEASED",
                f"terminal extension {extension_id!r} did not return current gate to its parent",
            )

        from developer.automation.planning_validator import validate_planning

        failures = validate_planning(base)
        if failures:
            raise PlanningStateReconciliationError(
                "POST_EXTENSION_FINALIZATION_VALIDATION_FAILED",
                "\n".join(failures),
            )
    except Exception as exc:
        for target, content in originals.items():
            target.write_text(content, encoding="utf-8")
        return ExtensionFinalizationResult(
            extension_id,
            True,
            False,
            failures=(f"extension finalization failed: {exc}",),
        )

    finalized = any(
        target.read_text(encoding="utf-8") != content
        for target, content in originals.items()
    )
    return ExtensionFinalizationResult(
        extension_id,
        True,
        finalized,
        current_gate_before=reconciliation.current_gate_before,
        current_gate_after=reconciliation.current_gate_after,
    )
