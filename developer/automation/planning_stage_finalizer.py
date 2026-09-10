from __future__ import annotations

import argparse
import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Mapping

from developer.automation.current_dependency_gate import (
    validate_current_legacy_dependency_gate,
)
from developer.automation.planning_validator import validate_planning
from developer.automation.policy_loader import load_yaml, repository_root
from developer.automation.policy_validator import validate_developer_policy


_PENDING = "IMPLEMENTED_VALIDATION_PENDING"
_COMPLETE = "COMPLETE"


@dataclass(frozen=True)
class FinalizationResult:
    stage_id: str
    promoted: bool
    failures: tuple[str, ...]


def _find_stage(payload: object, stage_id: str) -> Mapping[str, object]:
    found: list[Mapping[str, object]] = []

    def visit(value: object) -> None:
        if isinstance(value, Mapping):
            if value.get("id") == stage_id and "status" in value:
                found.append(value)
            for child in value.values():
                visit(child)
        elif isinstance(value, list):
            for child in value:
                visit(child)

    visit(payload)
    if len(found) != 1:
        raise ValueError(f"stage {stage_id!r} must resolve exactly once; found {len(found)}")
    return found[0]


def _replace_stage_status(text: str, stage_id: str, old: str, new: str) -> str:
    marker = re.compile(rf"(?m)^(?P<indent>\s*)- id: {re.escape(stage_id)}\s*$")
    match = marker.search(text)
    if match is None:
        raise ValueError(f"stage marker not found: {stage_id}")
    indent = match.group("indent")
    next_marker = re.compile(rf"(?m)^{re.escape(indent)}- id: ")
    following = next_marker.search(text, match.end())
    end = following.start() if following else len(text)
    block = text[match.start():end]
    status = re.compile(
        rf"(?m)^{re.escape(indent)}  status: {re.escape(old)}\s*$"
    )
    block, count = status.subn(f"{indent}  status: {new}", block, count=1)
    if count != 1:
        raise ValueError(f"stage {stage_id!r} status is not {old}")
    return text[:match.start()] + block + text[end:]


def promote_stage_text(text: str, stage_id: str, automatic: Mapping[str, object]) -> str:
    result = _replace_stage_status(text, stage_id, _PENDING, _COMPLETE)

    marker = re.compile(rf"(?m)^(?P<indent>\s*)- id: {re.escape(stage_id)}\s*$")
    match = marker.search(result)
    if match is None:
        raise ValueError(f"stage marker not found after promotion: {stage_id}")
    indent = match.group("indent")
    next_marker = re.compile(rf"(?m)^{re.escape(indent)}- id: ")
    following = next_marker.search(result, match.end())
    end = following.start() if following else len(result)
    block = result[match.start():end]
    pending_validation = re.compile(
        rf"(?m)^({re.escape(indent)}  validation:\s*\n{re.escape(indent)}    status:) PENDING\s*$"
    )
    block = pending_validation.sub(r"\1 PASS", block, count=1)
    result = result[:match.start()] + block + result[end:]

    next_stage = automatic.get("next_stage")
    if isinstance(next_stage, Mapping):
        next_id = next_stage.get("id")
        old = next_stage.get("from_status")
        new = next_stage.get("to_status")
        if all(isinstance(value, str) and value for value in (next_id, old, new)):
            result = _replace_stage_status(result, next_id, old, new)
    return result


def _run_pytest(base: Path, targets: object) -> tuple[str, ...]:
    if not isinstance(targets, list) or not targets:
        return ()
    if not all(isinstance(item, str) and item for item in targets):
        return ("automatic completion pytest_targets must be non-empty strings",)
    completed = subprocess.run(
        [sys.executable, "-m", "pytest", *targets, "-q"],
        cwd=base,
        capture_output=True,
        text=True,
    )
    if completed.returncode == 0:
        return ()
    output = "\n".join(part for part in (completed.stdout, completed.stderr) if part).strip()
    return (f"pytest validation failed\n{output}",)


def _run_registered_check(base: Path, check: str) -> tuple[str, ...]:
    if check == "POLICY_VALIDATION":
        return tuple(f"policy validation: {item}" for item in validate_developer_policy(base))
    if check == "PLANNING_VALIDATION":
        return tuple(f"planning validation: {item}" for item in validate_planning(base))
    if check == "CURRENT_LEGACY_DEPENDENCY_ZERO":
        return validate_current_legacy_dependency_gate(base)
    return (f"unknown automatic completion check: {check}",)


def finalize_stage(
    relative_path: str,
    stage_id: str,
    *,
    root: str | Path | None = None,
) -> FinalizationResult:
    base = repository_root(root)
    path = (base / relative_path).resolve()
    if base not in path.parents or not path.is_file():
        return FinalizationResult(stage_id, False, (f"invalid planning path: {relative_path}",))

    payload = load_yaml(relative_path, root=base)
    try:
        stage = _find_stage(payload, stage_id)
    except ValueError as exc:
        return FinalizationResult(stage_id, False, (str(exc),))

    if stage.get("status") == _COMPLETE:
        return FinalizationResult(stage_id, False, ())
    if stage.get("status") != _PENDING:
        return FinalizationResult(
            stage_id,
            False,
            (f"stage status must be {_PENDING}; got {stage.get('status')!r}",),
        )

    automatic = stage.get("automatic_completion")
    if not isinstance(automatic, Mapping):
        return FinalizationResult(stage_id, False, ("automatic_completion contract is missing",))
    if automatic.get("from_status") != _PENDING or automatic.get("to_status") != _COMPLETE:
        return FinalizationResult(stage_id, False, ("automatic completion transition contract is invalid",))

    failures = list(_run_pytest(base, automatic.get("pytest_targets")))
    checks = automatic.get("required_checks", [])
    if not isinstance(checks, list) or not checks:
        failures.append("automatic completion required_checks must not be empty")
    else:
        for check in checks:
            if not isinstance(check, str):
                failures.append("automatic completion check ids must be strings")
                continue
            failures.extend(_run_registered_check(base, check))
    if failures:
        return FinalizationResult(stage_id, False, tuple(failures))

    original = path.read_text(encoding="utf-8")
    try:
        promoted = promote_stage_text(original, stage_id, automatic)
    except ValueError as exc:
        return FinalizationResult(stage_id, False, (str(exc),))
    path.write_text(promoted, encoding="utf-8")

    post_failures = validate_planning(base)
    if post_failures:
        path.write_text(original, encoding="utf-8")
        return FinalizationResult(
            stage_id,
            False,
            tuple(f"post-promotion planning validation: {item}" for item in post_failures),
        )
    return FinalizationResult(stage_id, True, ())


def _main() -> int:
    parser = argparse.ArgumentParser(
        description="Machine-validate a pending planning stage and promote it to COMPLETE."
    )
    parser.add_argument("planning_path")
    parser.add_argument("stage_id")
    args = parser.parse_args()
    result = finalize_stage(args.planning_path, args.stage_id)
    if result.failures:
        print("\n".join(result.failures), file=sys.stderr)
        return 1
    if result.promoted:
        print(f"Planning stage promoted to COMPLETE: {result.stage_id}")
    else:
        print(f"Planning stage already COMPLETE: {result.stage_id}")
    return 0


if __name__ == "__main__":
    raise SystemExit(_main())
