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
PROFILE_PATH = "ptsip.yaml"


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


def _dedupe(items: list[str]) -> list[str]:
    return list(dict.fromkeys(items))


def _merge_pytest_targets(root: Path, targets: list[str]) -> list[str]:
    merged: list[str] = []
    for target in targets:
        normalized = target.rstrip("/")
        already_covered = any(
            existing == normalized
            or ((root / existing).is_dir() and normalized.startswith(existing + "/"))
            for existing in merged
        )
        if already_covered:
            continue
        if (root / normalized).is_dir():
            merged = [
                existing
                for existing in merged
                if not existing.startswith(normalized + "/")
            ]
        merged.append(normalized)
    return merged


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
    return {
        "status": "NOT_REGISTERED",
        "component_ref": component_ref,
        "fallback": "CANONICAL_COMPONENT_INCLUDE_SELECTION",
    }


def _component_regression_targets(root: Path, component_ref: str, source: str = PROFILE_PATH) -> list[str]:
    payload = load_yaml(source, root=root)
    components = payload.get("components")
    if not isinstance(components, list):
        raise WorkPacketError(f"{source} has no components list")
    matches = [
        item for item in components
        if isinstance(item, Mapping) and item.get("id") == component_ref
    ]
    if len(matches) != 1:
        raise WorkPacketError(
            f"expected one verification component {component_ref!r} in {source}, found {len(matches)}"
        )
    component = _mapping(matches[0], f"component {component_ref}")
    roles = component.get("roles")
    if not isinstance(roles, list) or "VERIFICATION" not in roles:
        raise WorkPacketError(f"{component_ref!r} is not a VERIFICATION component")
    includes = component.get("include")
    if not isinstance(includes, list) or not includes:
        raise WorkPacketError(f"{component_ref!r} has no include selectors")

    targets: list[str] = []
    for raw in includes:
        if not isinstance(raw, str) or not raw.startswith("tests/"):
            continue
        pattern = raw.replace("\\", "/")
        if pattern.endswith("/**"):
            target = pattern[:-3].rstrip("/")
            if not (root / target).exists():
                raise WorkPacketError(f"core regression target does not exist: {target!r}")
            targets.append(target)
            continue
        if not any(token in pattern for token in ("*", "?", "[")):
            if not (root / pattern).exists():
                raise WorkPacketError(f"core regression target does not exist: {pattern!r}")
            targets.append(pattern)
            continue
        matches_for_pattern = sorted(
            candidate.relative_to(root).as_posix()
            for candidate in root.glob(pattern)
            if candidate.is_file()
        )
        if not matches_for_pattern:
            raise WorkPacketError(f"core regression selector matched nothing: {pattern!r}")
        targets.extend(matches_for_pattern)
    if not targets:
        raise WorkPacketError(f"{component_ref!r} produced no pytest targets")
    return _dedupe(targets)


def _selector_integrity(root: Path, edit_targets: object) -> list[dict[str, object]]:
    if not isinstance(edit_targets, list):
        return [{"reason": "EDIT_TARGETS_INVALID"}]
    violations: list[dict[str, object]] = []
    for raw in edit_targets:
        if not isinstance(raw, Mapping):
            violations.append({"reason": "EDIT_TARGET_INVALID"})
            continue
        path = raw.get("path")
        selector = raw.get("selector")
        if not isinstance(path, str) or not isinstance(selector, Mapping):
            violations.append({"reason": "EDIT_TARGET_INVALID", "target": dict(raw)})
            continue
        try:
            current = policy_resolver._validate_implementation_ref(
                root, {"path": path, "selector": dict(selector)}
            )
        except (OSError, ValueError, SyntaxError, policy_resolver.PolicyResolverError) as exc:
            violations.append({
                "path": path,
                "selector": dict(selector),
                "reason": "SELECTOR_NO_LONGER_RESOLVES",
                "detail": str(exc),
            })
            continue
        if not isinstance(current.get("resolved_location"), Mapping):
            violations.append({
                "path": path,
                "selector": dict(selector),
                "reason": "SELECTOR_LOCATION_MISSING",
            })
    return violations


def _selector_fingerprint(root: Path, item: Mapping[str, object]) -> str:
    path = item.get("path")
    selector = item.get("selector")
    if not isinstance(path, str) or not isinstance(selector, Mapping):
        raise WorkPacketError("read-context selector fingerprint input is invalid")
    resolved = policy_resolver._validate_implementation_ref(
        root,
        {"path": path, "selector": dict(selector)},
    )
    location = _mapping(
        resolved.get("resolved_location"),
        "read-context resolved_location",
    )
    line_start = location.get("line_start")
    line_end = location.get("line_end")
    if not isinstance(line_start, int) or not isinstance(line_end, int):
        raise WorkPacketError("read-context selector has no exact line range")
    lines = (root / path).read_text(encoding="utf-8").splitlines()
    segment = "\n".join(lines[line_start - 1:line_end]) + "\n"
    return hashlib.sha256(segment.encode("utf-8")).hexdigest()


def _required_new_tests(verification: Mapping[str, object]) -> list[dict[str, object]]:
    raw = verification.get("required_new_tests")
    if not isinstance(raw, list) or not raw:
        raise WorkPacketError("verification.required_new_tests must be non-empty")
    result: list[dict[str, object]] = []
    for item in raw:
        entry = dict(_mapping(item, "required new test"))
        node = entry.get("node")
        acceptance_ids = entry.get("acceptance_ids")
        if not isinstance(node, str) or not node:
            raise WorkPacketError("required new test node must be non-empty")
        if not isinstance(acceptance_ids, list) or not acceptance_ids or not all(
            isinstance(value, str) and value for value in acceptance_ids
        ):
            raise WorkPacketError(f"required new test {node!r} has invalid acceptance_ids")
        result.append(entry)
    return result


def build_packet(repository: str | Path, *, scope: str, operation: str) -> dict[str, object]:
    root = repository_root(repository)
    registry = _registry(root)
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

    mutation = _mapping(recipe.get("mutation"), "mutation")
    raw_targets = mutation.get("targets")
    if not isinstance(raw_targets, list) or not raw_targets:
        raise WorkPacketError("workflow mutation.targets must be non-empty")
    edit_targets: list[dict[str, object]] = []
    for raw in raw_targets:
        target = _mapping(raw, "mutation target")
        selector = _mapping(target.get("selector"), "mutation target selector")
        key = _selector_key(str(target.get("path")), selector)
        matched = resolved_by_key.get(key)
        if matched is None:
            raise WorkPacketError("workflow mutation target is not present in Policy Resolver implementation refs")
        enriched = dict(matched)
        enriched["rationale"] = str(target.get("rationale"))
        edit_targets.append(enriched)

    edit_keys = {
        _selector_key(str(item["path"]), _mapping(item["selector"], "selector"))
        for item in edit_targets
    }
    read_context = [item for key, item in resolved_by_key.items() if key not in edit_keys]

    test_refs = task_context.get("test_refs")
    if not isinstance(test_refs, list) or not all(isinstance(item, str) for item in test_refs):
        raise WorkPacketError("task context test refs are invalid")
    allowed_test_paths = mutation.get("allowed_test_paths")
    if not isinstance(allowed_test_paths, list) or not allowed_test_paths or not all(
        isinstance(item, str) and item for item in allowed_test_paths
    ):
        raise WorkPacketError("mutation.allowed_test_paths must be non-empty strings")
    unbound_test_paths = sorted(set(allowed_test_paths) - set(test_refs))
    if unbound_test_paths:
        raise WorkPacketError(
            "mutation test paths are not Policy Resolver test refs: " + ", ".join(unbound_test_paths)
        )

    acceptance = recipe.get("acceptance_vectors")
    if not isinstance(acceptance, list) or not acceptance:
        raise WorkPacketError("workflow acceptance_vectors must be non-empty")
    acceptance_ids: set[str] = set()
    for raw in acceptance:
        item = _mapping(raw, "acceptance vector")
        vector_id = item.get("id")
        test_nodes = item.get("test_nodes")
        invariants = item.get("invariants")
        if not isinstance(vector_id, str) or not vector_id:
            raise WorkPacketError("acceptance vector id must be non-empty")
        if vector_id in acceptance_ids:
            raise WorkPacketError(f"duplicate acceptance vector id: {vector_id}")
        acceptance_ids.add(vector_id)
        if not isinstance(test_nodes, list) or not test_nodes or not all(
            isinstance(node, str) and node for node in test_nodes
        ):
            raise WorkPacketError(f"acceptance vector {vector_id!r} must declare test_nodes")
        if not isinstance(invariants, list) or not invariants or not all(
            isinstance(value, str) and value for value in invariants
        ):
            raise WorkPacketError(f"acceptance vector {vector_id!r} must declare invariants")

    verification = _mapping(recipe.get("verification"), "verification")
    baseline_nodes = verification.get("baseline_pytest_nodes")
    task_regression = verification.get("task_regression_pytest_targets")
    full_command = verification.get("full_command")
    if not isinstance(baseline_nodes, list) or not baseline_nodes or not all(
        isinstance(item, str) for item in baseline_nodes
    ):
        raise WorkPacketError("verification.baseline_pytest_nodes must be non-empty strings")
    if not isinstance(task_regression, list) or not task_regression or not all(
        isinstance(item, str) for item in task_regression
    ):
        raise WorkPacketError("verification.task_regression_pytest_targets must be non-empty strings")
    if not isinstance(full_command, list) or not full_command or not all(
        isinstance(item, str) for item in full_command
    ):
        raise WorkPacketError("verification.full_command must be non-empty strings")

    required_tests = _required_new_tests(verification)
    required_nodes = [str(item["node"]) for item in required_tests]
    for item in required_tests:
        unknown = sorted(set(item["acceptance_ids"]) - acceptance_ids)
        if unknown:
            raise WorkPacketError(
                f"required test {item['node']!r} references unknown acceptance ids: " + ", ".join(unknown)
            )
        path = str(item["node"]).split("::", 1)[0]
        if path not in allowed_test_paths:
            raise WorkPacketError(f"required test {item['node']!r} is outside mutation.allowed_test_paths")

    declared_acceptance_tests = {
        node
        for raw in acceptance
        for node in _mapping(raw, "acceptance vector").get("test_nodes", [])
        if isinstance(node, str)
    }
    missing_declared_mapping = sorted(set(required_nodes) - declared_acceptance_tests)
    if missing_declared_mapping:
        raise WorkPacketError(
            "required tests are not mapped by acceptance vectors: " + ", ".join(missing_declared_mapping)
        )

    missing_baseline = [node for node in baseline_nodes if not _test_node_exists(root, node)]
    if missing_baseline:
        raise WorkPacketError("baseline verification nodes are missing: " + ", ".join(missing_baseline))
    missing_new = [node for node in required_nodes if not _test_node_exists(root, node)]
    for target in task_regression:
        if not (root / target).exists():
            raise WorkPacketError(f"task regression target does not exist: {target!r}")

    core_policy = _mapping(verification.get("core_regression"), "verification.core_regression")
    component_ref = core_policy.get("component_ref")
    component_source = core_policy.get("source")
    if not isinstance(component_ref, str) or not component_ref:
        raise WorkPacketError("core regression component_ref must be non-empty")
    if not isinstance(component_source, str) or not component_source:
        raise WorkPacketError("core regression source must be non-empty")
    core_targets = _component_regression_targets(root, component_ref, component_source)
    combined_regression = _merge_pytest_targets(
        root,
        [*task_regression, *core_targets],
    )

    acceptance_coverage: list[dict[str, object]] = []
    for raw in acceptance:
        item = _mapping(raw, "acceptance vector")
        nodes = [str(node) for node in item["test_nodes"]]
        missing_nodes = [node for node in nodes if not _test_node_exists(root, node)]
        acceptance_coverage.append({
            "id": item["id"],
            "test_nodes": nodes,
            "missing_test_nodes": missing_nodes,
            "covered": not missing_nodes,
        })

    policy_paths = [
        str(item["path"]) for item in resolved.get("policies", [])
        if isinstance(item, dict) and isinstance(item.get("path"), str)
    ]
    planning_entry = str(task_context["planning_entry"])
    normative_source = str(task_context["normative_rule_source"])
    context_files = sorted(set(
        [REGISTRY_PATH, SCHEMA_PATH, component_source, planning_entry, normative_source, *policy_paths]
    ))
    read_context_fingerprints = [
        {
            "path": str(item["path"]),
            "selector": dict(_mapping(item["selector"], "read-context selector")),
            "fingerprint": _selector_fingerprint(root, item),
        }
        for item in read_context
    ]
    tracked_files = sorted(set(
        context_files
        + [str(item["path"]) for item in read_context]
        + [str(item["path"]) for item in edit_targets]
        + list(test_refs)
        + list(allowed_test_paths)
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
        "mutation": mutation,
        "acceptance_vectors": acceptance,
        "verification": verification,
        "core_regression_targets": core_targets,
        "file_hashes": file_hashes,
    }
    fingerprint = hashlib.sha256(
        json.dumps(fingerprint_payload, sort_keys=True, separators=(",", ":")).encode("utf-8")
    ).hexdigest()

    commands = {
        "baseline": ["python", "-m", "pytest", *baseline_nodes, "-vv"],
        "focused": ["python", "-m", "pytest", *baseline_nodes, *required_nodes, "-vv"],
        "task-regression": ["python", "-m", "pytest", *task_regression, "-vv"],
        "core": ["python", "-m", "pytest", *core_targets, "-vv"],
        "regression": ["python", "-m", "pytest", *combined_regression, "-vv"],
        "full": list(full_command),
    }

    failure_routing = _mapping(registry.get("failure_routing"), "failure_routing")
    scope_expansion = mutation.get("scope_expansion")
    if scope_expansion != "RE_RESOLVE_REQUIRED":
        raise WorkPacketError("mutation.scope_expansion must be RE_RESOLVE_REQUIRED")

    return {
        "schema_version": "ptsip-implementation-work-packet/v2",
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
        "mutation_plan": {
            "targets": edit_targets,
            "allowed_test_paths": sorted(set(allowed_test_paths)),
            "scope_expansion": scope_expansion,
        },
        "acceptance_vectors": acceptance,
        "acceptance_coverage": acceptance_coverage,
        "verification": {
            "baseline_pytest_nodes": baseline_nodes,
            "required_new_tests": required_tests,
            "required_new_pytest_nodes": required_nodes,
            "missing_required_new_tests": missing_new,
            "task_regression_pytest_targets": task_regression,
            "core_regression": {
                "component_ref": component_ref,
                "source": component_source,
                "selection": core_policy.get("selection"),
                "pytest_targets": core_targets,
            },
            "combined_regression_pytest_targets": combined_regression,
            "commands": commands,
            "status": "REQUIRES_NEW_TESTS" if missing_new else "READY",
        },
        "edit_budget": {
            "allowed_code_paths": sorted({str(item["path"]) for item in edit_targets}),
            "allowed_test_paths": sorted(set(allowed_test_paths)),
            "unlisted_paths": "BLOCK",
            "selector_removal_or_rename": "BLOCK",
        },
        "test_mode": _registered_test_mode(root, component_ref),
        "failure_routing": dict(failure_routing),
        "freshness": {
            "baseline_head": head,
            "context_files": context_files,
            "read_context_fingerprints": read_context_fingerprints,
            "file_hashes": file_hashes,
            "context_fingerprint": fingerprint,
            "strategy": "FILE_AND_SELECTOR_RECHECK_BEFORE_EVERY_VERIFICATION",
        },
    }


def _changed_paths(def _changed_paths(root: Path) -> list[str]:
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


def _iteration_fingerprint(root: Path, changed: list[str], allowed: set[str]) -> str:
    payload: dict[str, object] = {
        "head": _git(root, "rev-parse", "HEAD"),
        "changed": changed,
        "files": {},
    }
    files = payload["files"]
    assert isinstance(files, dict)
    for path in sorted(set(changed) & allowed):
        candidate = root / path
        files[path] = _sha256(candidate) if candidate.is_file() else "MISSING"
    return hashlib.sha256(
        json.dumps(payload, sort_keys=True, separators=(",", ":")).encode("utf-8")
    ).hexdigest()


def check_packet(repository: str | Path, packet: Mapping[str, object]) -> dict[str, object]:
    root = repository_root(repository)
    task = _mapping(packet.get("task"), "packet.task")
    freshness = _mapping(packet.get("freshness"), "packet.freshness")
    budget = _mapping(packet.get("edit_budget"), "packet.edit_budget")
    mutation_plan = _mapping(packet.get("mutation_plan"), "packet.mutation_plan")

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

    read_context_changed: list[dict[str, object]] = []
    raw_read_fingerprints = freshness.get("read_context_fingerprints", [])
    if isinstance(raw_read_fingerprints, list):
        for raw in raw_read_fingerprints:
            if not isinstance(raw, Mapping):
                read_context_changed.append({"reason": "READ_CONTEXT_FINGERPRINT_INVALID"})
                continue
            baseline = raw.get("fingerprint")
            try:
                current = _selector_fingerprint(root, raw)
            except (OSError, ValueError, policy_resolver.PolicyResolverError, WorkPacketError) as exc:
                read_context_changed.append({
                    "path": raw.get("path"),
                    "selector": raw.get("selector"),
                    "reason": "READ_CONTEXT_SELECTOR_NO_LONGER_RESOLVES",
                    "detail": str(exc),
                })
                continue
            if not isinstance(baseline, str) or current != baseline:
                read_context_changed.append({
                    "path": raw.get("path"),
                    "selector": raw.get("selector"),
                    "reason": "READ_CONTEXT_SELECTOR_CHANGED",
                })

    ranges_by_path: dict[str, list[tuple[int, int]]] = {}
    edit_targets = mutation_plan.get("targets", [])
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

    selector_violations = _selector_integrity(root, edit_targets)
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
    if read_context_changed:
        problems.append("READ_CONTEXT_CHANGED")
    if unexpected:
        problems.append("UNEXPECTED_CHANGED_PATH")
    if code_scope_violations:
        problems.append("CODE_SCOPE_VIOLATION")
    if selector_violations:
        problems.append("MUTATION_SELECTOR_VIOLATION")

    reprepare_required = any(
        problem in {"BRANCH_CHANGED", "HEAD_CHANGED", "CONTEXT_CHANGED", "READ_CONTEXT_CHANGED"}
        for problem in problems
    )
    return {
        "schema_version": "ptsip-implementation-work-check/v2",
        "status": "PASS" if not problems else "BLOCKED",
        "problems": problems,
        "branch": {"expected": expected_branch, "actual": actual_branch},
        "head": {"expected": expected_head, "actual": actual_head},
        "changed_paths": changed,
        "unexpected_changed_paths": unexpected,
        "context_changed": context_changed,
        "read_context_changed": read_context_changed,
        "code_scope_violations": code_scope_violations,
        "selector_violations": selector_violations,
        "missing_required_new_tests": missing_required,
        "iteration_fingerprint": _iteration_fingerprint(root, changed, allowed),
        "reprepare_required": reprepare_required,
        "safe_to_continue_iteration": (
            not reprepare_required and not unexpected
            and not code_scope_violations and not selector_violations
        ),
    }


def _load_packet(def _failure_signature(stage: str, returncode: int, output: str) -> str:
    lines = [line.rstrip() for line in output.splitlines() if line.strip()]
    tail = "\n".join(lines[-80:])
    normalized = re.sub(r"\b\d+(?:\.\d+)?s\b", "<duration>", tail)
    normalized = re.sub(r"0x[0-9a-fA-F]+", "0x<addr>", normalized)
    raw = f"{stage}\n{returncode}\n{normalized}"
    return hashlib.sha256(raw.encode("utf-8", errors="replace")).hexdigest()[:24]


def _load_failure_state(path: Path) -> dict[str, object]:
    if not path.exists():
        return {"schema_version": "ptsip-implementation-failure-state/v1", "entries": {}}
    payload = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(payload, dict):
        raise WorkPacketError("failure state must contain a JSON object")
    entries = payload.get("entries")
    if not isinstance(entries, dict):
        raise WorkPacketError("failure state entries must be a mapping")
    return payload


def route_failure(
    *,
    packet_id: str,
    stage: str,
    returncode: int,
    output: str,
    state_path: Path,
    policy: Mapping[str, object],
) -> dict[str, object]:
    signature = _failure_signature(stage, returncode, output)
    state = _load_failure_state(state_path)
    entries = state["entries"]
    assert isinstance(entries, dict)
    key = f"{packet_id}:{stage}:{signature}"
    previous = entries.get(key)
    count = int(previous.get("count", 0)) + 1 if isinstance(previous, dict) else 1
    entries[key] = {"count": count, "returncode": returncode}
    state_path.parent.mkdir(parents=True, exist_ok=True)
    state_path.write_text(json.dumps(state, indent=2) + "\n", encoding="utf-8")

    collection_failure = any(
        marker in output
        for marker in ("ERROR collecting", "not found:", "found no collectors", "no tests ran")
    )
    recheck_at = int(policy.get("recheck_context_at", 2))
    reresolve_at = int(policy.get("reresolve_scope_at", 3))
    if collection_failure:
        classification = "TEST_CONTRACT_FAILURE"
        action = "REPAIR_TEST_SELECTION_OR_REQUIRED_TEST"
    elif count >= reresolve_at:
        classification = "REPEATED_IMPLEMENTATION_FAILURE"
        action = "RE_RESOLVE_MUTATION_SCOPE"
    elif count >= recheck_at:
        classification = "REPEATED_IMPLEMENTATION_FAILURE"
        action = "RECHECK_PACKET_ACCEPTANCE_AND_CONTEXT"
    else:
        classification = "IMPLEMENTATION_FAILURE"
        action = "FIX_WITHIN_CURRENT_MUTATION_PLAN"
    return {
        "schema_version": "ptsip-implementation-failure-route/v1",
        "packet_id": packet_id,
        "stage": stage,
        "signature": signature,
        "repeat_count": count,
        "classification": classification,
        "next_action": action,
        "scope_expansion_allowed": False,
    }


def _clear_failure_state(state_path: Path, packet_id: str, stage: str) -> None:
    if not state_path.exists():
        return
    state = _load_failure_state(state_path)
    entries = state["entries"]
    assert isinstance(entries, dict)
    prefix = f"{packet_id}:{stage}:"
    retained = {key: value for key, value in entries.items() if not key.startswith(prefix)}
    if retained == entries:
        return
    state["entries"] = retained
    state_path.write_text(json.dumps(state, indent=2) + "\n", encoding="utf-8")


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
    verify.add_argument(
        "--stage",
        required=True,
        choices=("baseline", "focused", "task-regression", "core", "regression", "full"),
    )
    verify.add_argument("--packet", required=True)
    verify.add_argument("--log")
    verify.add_argument("--failure-state")
    verify.add_argument("--json-routing", action="store_true")
    return parser


def main(argv: list[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        root = repository_root(args.repository)
        if args.command == "prepare":
            payload = build_packet(root, scope=args.scope, operation=args.operation)
            if args.output:
                Path(args.output).write_text(
                    json.dumps(payload, indent=2, ensure_ascii=False) + "\n",
                    encoding="utf-8",
                )
            _emit(payload, bool(args.json))
            return 0

        packet = _load_packet(Path(args.packet))
        checked = check_packet(root, packet)
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

        packet_id = str(packet.get("packet_id", "unknown-packet"))
        log_path = Path(args.log) if args.log else root / ".git" / f"ptsip-iwp-{args.stage}.log"
        state_path = (
            Path(args.failure_state)
            if args.failure_state
            else root / ".git" / "ptsip-iwp-failure-state.json"
        )
        completed = subprocess.run(
            run_command,
            cwd=root,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
        )
        output = completed.stdout or ""
        log_path.parent.mkdir(parents=True, exist_ok=True)
        log_path.write_text(output, encoding="utf-8")
        if output:
            print(output, end="" if output.endswith("\n") else "\n")
        if completed.returncode == 0:
            _clear_failure_state(state_path, packet_id, args.stage)
            return 0

        routing = route_failure(
            packet_id=packet_id,
            stage=args.stage,
            returncode=int(completed.returncode),
            output=output,
            state_path=state_path,
            policy=_mapping(packet.get("failure_routing"), "packet.failure_routing"),
        )
        _emit(routing, bool(args.json_routing))
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
