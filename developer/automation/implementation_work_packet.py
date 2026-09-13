from __future__ import annotations

import argparse
import ast
import hashlib
import json
import re
import subprocess
import sys
from pathlib import Path
from typing import Mapping

import yaml
from jsonschema import Draft202012Validator

import developer.automation.policy_resolver as policy_resolver
from developer.automation.policy_loader import load_json, load_yaml, repository_root


REGISTRY_PATH = "developer/automation/implementation_workflows.yaml"
SCHEMA_PATH = "developer/automation/implementation_workflows.schema.json"


class WorkPacketError(RuntimeError):
    pass


def _mapping(value: object, label: str) -> Mapping[str, object]:
    if not isinstance(value, Mapping):
        raise WorkPacketError(f"{label} must be a mapping")
    return value


def _git(root: Path, *args: str) -> str:
    completed = subprocess.run(
        ["git", "-C", str(root), *args],
        check=False,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    if completed.returncode != 0:
        detail = completed.stderr.strip() or completed.stdout.strip()
        raise WorkPacketError(f"git {' '.join(args)} failed: {detail}")
    return completed.stdout.strip()


def _sha256(path: Path) -> str:
    digest = hashlib.sha256()
    digest.update(path.read_bytes())
    return digest.hexdigest()


def _selector_key(path: str, selector: Mapping[str, object]) -> str:
    return json.dumps(
        {"path": path, "selector": dict(selector)},
        sort_keys=True,
        separators=(",", ":"),
    )


def _registry(root: Path) -> dict[str, object]:
    payload = load_yaml(REGISTRY_PATH, root=root)
    schema = load_json(SCHEMA_PATH, root=root)
    Draft202012Validator.check_schema(schema)
    errors = sorted(
        Draft202012Validator(schema).iter_errors(payload),
        key=lambda error: tuple(str(part) for part in error.absolute_path),
    )
    if errors:
        rendered = "; ".join(
            f"{'.'.join(str(part) for part in error.absolute_path) or '<root>'}: {error.message}"
            for error in errors
        )
        raise WorkPacketError(f"implementation workflow registry is invalid: {rendered}")
    return payload


def _recipe(root: Path, *, branch: str, scope: str, operation: str) -> Mapping[str, object]:
    registry = _registry(root)
    tasks = registry.get("tasks")
    if not isinstance(tasks, list):
        raise WorkPacketError("implementation workflow registry has no tasks")
    matches = [
        _mapping(item, "workflow task")
        for item in tasks
        if isinstance(item, Mapping)
        and item.get("branch") == branch
        and item.get("scope") == scope
        and item.get("operation") == operation
    ]
    if len(matches) != 1:
        raise WorkPacketError(
            f"expected one exact implementation workflow for {branch}:{scope}:{operation}, found {len(matches)}"
        )
    return matches[0]


def _test_node_exists(root: Path, node_id: str) -> bool:
    parts = node_id.split("::")
    if len(parts) < 2:
        return False
    path = root / parts[0]
    if not path.is_file():
        return False
    try:
        tree = ast.parse(path.read_text(encoding="utf-8"), filename=parts[0])
    except SyntaxError:
        return False
    body: list[ast.stmt] = list(tree.body)
    for index, name in enumerate(parts[1:]):
        candidates = [
            item for item in body
            if isinstance(item, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef))
            and item.name == name
        ]
        if len(candidates) != 1:
            return False
        selected = candidates[0]
        if index < len(parts[1:]) - 1:
            if not isinstance(selected, ast.ClassDef):
                return False
            body = list(selected.body)
    return True


def _registered_test_mode(root: Path, component_ref: str) -> dict[str, object]:
    path = root / ".github" / "test_modes.yaml"
    if not path.is_file():
        return {"status": "REGISTRY_MISSING", "component_ref": component_ref}
    payload = yaml.safe_load(path.read_text(encoding="utf-8-sig"))
    modes = payload.get("modes") if isinstance(payload, dict) else None
    if not isinstance(modes, list):
        return {"status": "REGISTRY_INVALID", "component_ref": component_ref}
    for raw in modes:
        if isinstance(raw, dict) and raw.get("component_ref") == component_ref:
            return {"status": "REGISTERED", "component_ref": component_ref, "mode_id": raw.get("id")}
    return {"status": "NOT_REGISTERED", "component_ref": component_ref, "fallback": "DEVELOPER_LOCAL_TASK_LANE"}


def build_packet(repository: str | Path, *, scope: str, operation: str) -> dict[str, object]:
    root = repository_root(repository)
    resolved = policy_resolver.resolve_policies(root, scope=scope, operation=operation)
    task_context = _mapping(resolved.get("task_context"), "task_context")
    branch_context = _mapping(task_context.get("branch_context"), "branch_context")
    branch = branch_context.get("actual")
    if not isinstance(branch, str) or not branch:
        raise WorkPacketError("task context has no actual branch")
    normalized_operation = operation.strip().upper()
    recipe = _recipe(root, branch=branch, scope=str(resolved["scope"]), operation=normalized_operation)

    refs = task_context.get("implementation_refs")
    if not isinstance(refs, list) or not refs:
        raise WorkPacketError("task context has no implementation refs")
    resolved_by_key: dict[str, dict[str, object]] = {}
    for raw in refs:
        item = dict(_mapping(raw, "implementation ref"))
        selector = _mapping(item.get("selector"), "implementation selector")
        key = _selector_key(str(item["path"]), selector)
        if key in resolved_by_key:
            raise WorkPacketError("Policy Resolver returned duplicate implementation refs")
        resolved_by_key[key] = item

    raw_edit_targets = recipe.get("edit_targets")
    if not isinstance(raw_edit_targets, list) or not raw_edit_targets:
        raise WorkPacketError("workflow has no edit targets")
    edit_targets: list[dict[str, object]] = []
    for raw in raw_edit_targets:
        target = _mapping(raw, "edit target")
        selector = _mapping(target.get("selector"), "edit target selector")
        key = _selector_key(str(target.get("path")), selector)
        matched = resolved_by_key.get(key)
        if matched is None:
            raise WorkPacketError("workflow edit target is not present in Policy Resolver implementation refs")
        edit_targets.append(matched)

    edit_keys = {
        _selector_key(str(item["path"]), _mapping(item["selector"], "selector"))
        for item in edit_targets
    }
    read_context = [item for key, item in resolved_by_key.items() if key not in edit_keys]

    verification = _mapping(recipe.get("verification"), "verification")
    baseline_nodes = verification.get("baseline_pytest_nodes")
    new_nodes = verification.get("required_new_pytest_nodes")
    regression_targets = verification.get("regression_pytest_targets")
    full_command = verification.get("full_command")
    sequences = (baseline_nodes, new_nodes, regression_targets, full_command)
    if not all(isinstance(value, list) and value for value in sequences):
        raise WorkPacketError("verification lists must be non-empty")
    if not all(isinstance(item, str) for seq in sequences for item in seq):
        raise WorkPacketError("verification entries must be strings")

    missing_baseline = [node for node in baseline_nodes if not _test_node_exists(root, node)]
    if missing_baseline:
        raise WorkPacketError("baseline verification nodes are missing: " + ", ".join(missing_baseline))
    missing_new = [node for node in new_nodes if not _test_node_exists(root, node)]
    for target in regression_targets:
        if not (root / target).exists():
            raise WorkPacketError(f"regression target does not exist: {target!r}")

    test_refs = task_context.get("test_refs")
    if not isinstance(test_refs, list) or not all(isinstance(item, str) for item in test_refs):
        raise WorkPacketError("task context test refs are invalid")

    policy_paths = [
        str(item["path"]) for item in resolved.get("policies", [])
        if isinstance(item, dict) and isinstance(item.get("path"), str)
    ]
    planning_entry = str(task_context["planning_entry"])
    normative_source = str(task_context["normative_rule_source"])
    context_files = sorted(set(
        [REGISTRY_PATH, SCHEMA_PATH, planning_entry, normative_source, *policy_paths]
        + [str(item["path"]) for item in read_context]
    ))
    tracked_files = sorted(set(
        context_files + [str(item["path"]) for item in edit_targets] + list(test_refs)
    ))
    file_hashes = {path: _sha256(root / path) for path in tracked_files if (root / path).is_file()}

    head = _git(root, "rev-parse", "HEAD")
    fingerprint_payload = {
        "head": head,
        "branch": branch,
        "scope": resolved["scope"],
        "operation": normalized_operation,
        "policy_context": resolved.get("policies"),
        "normative_rule_refs": task_context.get("normative_rule_refs"),
        "edit_targets": raw_edit_targets,
        "acceptance_vectors": recipe.get("acceptance_vectors"),
        "verification": verification,
        "file_hashes": file_hashes,
    }
    fingerprint = hashlib.sha256(
        json.dumps(fingerprint_payload, sort_keys=True, separators=(",", ":")).encode("utf-8")
    ).hexdigest()

    commands = {
        "baseline": ["python", "-m", "pytest", *baseline_nodes, "-vv"],
        "focused": ["python", "-m", "pytest", *baseline_nodes, *new_nodes, "-vv"],
        "regression": ["python", "-m", "pytest", *regression_targets, "-vv"],
        "full": list(full_command),
    }
    desired_component = recipe.get("desired_test_component")
    if not isinstance(desired_component, str) or not desired_component:
        raise WorkPacketError("desired_test_component must be non-empty")

    return {
        "schema_version": "ptsip-implementation-work-packet/v1",
        "projection_authority": False,
        "packet_id": "iwp-" + fingerprint[:16],
        "task": {
            "branch": branch,
            "head": head,
            "scope": resolved["scope"],
            "operation": normalized_operation,
            "planning_entry": planning_entry,
        },
        "policy_context": {
            "policies": resolved.get("policies"),
            "normative_rules": task_context.get("normative_rules"),
            "constraints": task_context.get("constraints"),
        },
        "read_context": read_context,
        "edit_targets": edit_targets,
        "acceptance_vectors": recipe.get("acceptance_vectors"),
        "verification": {
            "baseline_pytest_nodes": baseline_nodes,
            "required_new_pytest_nodes": new_nodes,
            "missing_required_new_tests": missing_new,
            "regression_pytest_targets": regression_targets,
            "commands": commands,
            "status": "REQUIRES_NEW_TESTS" if missing_new else "READY",
        },
        "edit_budget": {
            "allowed_code_paths": sorted({str(item["path"]) for item in edit_targets}),
            "allowed_test_paths": sorted(set(test_refs)),
            "unlisted_paths": "BLOCK",
        },
        "test_mode": _registered_test_mode(root, desired_component),
        "freshness": {
            "baseline_head": head,
            "context_files": context_files,
            "file_hashes": file_hashes,
            "context_fingerprint": fingerprint,
        },
    }


def _changed_paths(root: Path) -> list[str]:
    tracked = _git(root, "diff", "--name-only", "HEAD").splitlines()
    untracked = _git(root, "ls-files", "--others", "--exclude-standard").splitlines()
    return sorted({item.replace("\\", "/") for item in tracked + untracked if item})


_HUNK = re.compile(r"^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@")


def _diff_hunks(root: Path, path: str) -> list[tuple[int, int]]:
    diff = _git(root, "diff", "--unified=0", "HEAD", "--", path)
    result: list[tuple[int, int]] = []
    for line in diff.splitlines():
        match = _HUNK.match(line)
        if match:
            result.append((int(match.group(1)), int(match.group(2) or "1")))
    return result


def _hunk_allowed(old_start: int, old_count: int, ranges: list[tuple[int, int]]) -> bool:
    if old_count == 0:
        return any(start - 1 <= old_start <= end for start, end in ranges)
    old_end = old_start + old_count - 1
    return any(start <= old_start and old_end <= end for start, end in ranges)


def check_packet(repository: str | Path, packet: Mapping[str, object]) -> dict[str, object]:
    root = repository_root(repository)
    task = _mapping(packet.get("task"), "packet.task")
    freshness = _mapping(packet.get("freshness"), "packet.freshness")
    budget = _mapping(packet.get("edit_budget"), "packet.edit_budget")

    actual_branch = _git(root, "branch", "--show-current")
    actual_head = _git(root, "rev-parse", "HEAD")
    expected_branch = str(task.get("branch", ""))
    expected_head = str(freshness.get("baseline_head", ""))

    changed = _changed_paths(root)
    code_paths = set(budget.get("allowed_code_paths", []))
    test_paths = set(budget.get("allowed_test_paths", []))
    allowed = code_paths | test_paths
    unexpected = [path for path in changed if path not in allowed]

    baseline_hashes = _mapping(freshness.get("file_hashes"), "file_hashes")
    context_files = freshness.get("context_files", [])
    context_changed: list[str] = []
    if isinstance(context_files, list):
        for path in context_files:
            if not isinstance(path, str):
                continue
            candidate = root / path
            baseline = baseline_hashes.get(path)
            if not candidate.is_file() or _sha256(candidate) != baseline:
                context_changed.append(path)

    ranges_by_path: dict[str, list[tuple[int, int]]] = {}
    edit_targets = packet.get("edit_targets", [])
    if isinstance(edit_targets, list):
        for raw in edit_targets:
            if not isinstance(raw, dict):
                continue
            path = raw.get("path")
            location = raw.get("resolved_location")
            if (
                isinstance(path, str) and isinstance(location, dict)
                and isinstance(location.get("line_start"), int)
                and isinstance(location.get("line_end"), int)
            ):
                ranges_by_path.setdefault(path, []).append(
                    (location["line_start"], location["line_end"])
                )

    code_scope_violations: list[dict[str, object]] = []
    for path in sorted(code_paths & set(changed)):
        ranges = ranges_by_path.get(path, [])
        for old_start, old_count in _diff_hunks(root, path):
            if not _hunk_allowed(old_start, old_count, ranges):
                code_scope_violations.append(
                    {"path": path, "old_start": old_start, "old_count": old_count}
                )

    verification = _mapping(packet.get("verification"), "packet.verification")
    required_nodes = verification.get("required_new_pytest_nodes", [])
    missing_required = (
        [node for node in required_nodes if isinstance(node, str) and not _test_node_exists(root, node)]
        if isinstance(required_nodes, list) else []
    )

    problems: list[str] = []
    if actual_branch != expected_branch:
        problems.append("BRANCH_CHANGED")
    if actual_head != expected_head:
        problems.append("HEAD_CHANGED")
    if context_changed:
        problems.append("CONTEXT_CHANGED")
    if unexpected:
        problems.append("UNEXPECTED_CHANGED_PATH")
    if code_scope_violations:
        problems.append("CODE_SCOPE_VIOLATION")

    return {
        "schema_version": "ptsip-implementation-work-check/v1",
        "status": "PASS" if not problems else "BLOCKED",
        "problems": problems,
        "branch": {"expected": expected_branch, "actual": actual_branch},
        "head": {"expected": expected_head, "actual": actual_head},
        "changed_paths": changed,
        "unexpected_changed_paths": unexpected,
        "context_changed": context_changed,
        "code_scope_violations": code_scope_violations,
        "missing_required_new_tests": missing_required,
        "locations_need_refresh": bool(code_paths & set(changed)),
    }


def _load_packet(path: Path) -> dict[str, object]:
    payload = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(payload, dict):
        raise WorkPacketError("packet file must contain a JSON object")
    return payload


def _emit(payload: object, as_json: bool) -> None:
    if as_json:
        print(json.dumps(payload, indent=2, ensure_ascii=False))
    else:
        print(yaml.safe_dump(payload, sort_keys=False, allow_unicode=True).rstrip())


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="python -m developer.automation.implementation_work_packet")
    parser.add_argument("--repository", default=".")
    sub = parser.add_subparsers(dest="command", required=True)

    prepare = sub.add_parser("prepare")
    prepare.add_argument("--scope", required=True)
    prepare.add_argument("--operation", required=True)
    prepare.add_argument("--output")
    prepare.add_argument("--json", action="store_true")

    check = sub.add_parser("check")
    check.add_argument("--packet", required=True)
    check.add_argument("--json", action="store_true")

    verify = sub.add_parser("verify")
    verify.add_argument("--packet", required=True)
    verify.add_argument("--stage", required=True, choices=("baseline", "focused", "regression", "full"))
    return parser


def main(argv: list[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        if args.command == "prepare":
            payload = build_packet(args.repository, scope=args.scope, operation=args.operation)
            if args.output:
                Path(args.output).write_text(
                    json.dumps(payload, indent=2, ensure_ascii=False) + "\n",
                    encoding="utf-8",
                )
            _emit(payload, bool(args.json))
            return 0

        packet = _load_packet(Path(args.packet))
        checked = check_packet(args.repository, packet)
        if args.command == "check":
            _emit(checked, bool(args.json))
            return 0 if checked["status"] == "PASS" else 2

        if checked["status"] != "PASS":
            _emit(checked, True)
            return 2
        verification = _mapping(packet.get("verification"), "packet.verification")
        if args.stage == "focused" and checked["missing_required_new_tests"]:
            print(
                "Implementation Work Packet error: focused verification requires new tests: "
                + ", ".join(checked["missing_required_new_tests"])
            )
            return 3
        commands = _mapping(verification.get("commands"), "verification.commands")
        command = commands.get(args.stage)
        if not isinstance(command, list) or not all(isinstance(item, str) for item in command):
            raise WorkPacketError(f"verification command is invalid for stage {args.stage}")
        run_command = list(command)
        if run_command and run_command[0] == "python":
            run_command[0] = sys.executable
        completed = subprocess.run(run_command, cwd=repository_root(args.repository))
        return int(completed.returncode)
    except (
        OSError,
        ValueError,
        json.JSONDecodeError,
        policy_resolver.PolicyResolverError,
        WorkPacketError,
    ) as exc:
        print(f"Implementation Work Packet error: {exc}")
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
