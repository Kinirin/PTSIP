from __future__ import annotations

import argparse
import json
from pathlib import Path, PurePosixPath, PureWindowsPath
from typing import Mapping

import yaml

from agent_contracts.resolver import resolve_operation
from agent_contracts.validator import validate_agent_contract_plane
from ptsip.local_profile_catalog import (
    LOCAL_PROFILE_CATALOG,
    LOCAL_PROFILE_ROOT,
    LocalProfileCatalogError,
    load_local_profile_selection,
)
from ptsip.repository.namespace import RepositoryNamespaceError, load_repository_index
from ptsip.validation.components import normalize_selector

from developer.automation.policy_loader import repository_root
from developer.automation.repository_state_resolver import (
    RepositoryStateResolutionError,
    resolve_state,
)


AGENT_INDEX = "src/agent_contracts/index.yaml"
EMBEDDED_AGENT_ROOT = "src/ptsip/agent_contracts"
MPD_0012 = "developer/policy/MPD-0012.yaml"
COVERAGE = "developer/planning/migrations/MPD-0012-agent-context-coverage.yaml"
STATE_INDEX = "developer/state/index.yaml"

OPERATIONS = (
    "PTSIP-OP-ADOPT-001",
    "PTSIP-OP-VALIDATE-001",
    "PTSIP-OP-CONFORM-001",
    "PTSIP-OP-RECONCILE-AUTHORITY-001",
    "PTSIP-OP-MIGRATE-PROFILE-001",
)

STATE_DOMAINS = (
    "developer_policy",
    "developer_planning",
    "project_profile",
    "policy_plan_binding",
    "agent_contract",
    "governance_source",
    "context_migration",
)

NORMATIVE_MARKDOWN_TARGETS = (
    "adoption/ADOPTION-GUIDE.md",
    "agents/AGENT-CONTRACT.md",
    "spec/PTSIP-CONFORMANCE.md",
    "spec/PTSIP-DRAFT-PROFILE-TRANSITION.md",
    "spec/PTSIP-GOVERNANCE.md",
    "spec/PTSIP-RESPONSIBILITY-MAP.md",
    "spec/PTSIP-SPEC.md",
    "spec/PTSIP-TERMINOLOGY.md",
)

# Retired operational roots remain migration evidence, never current profile selectors.
RETIRED_PROFILE_ROOTS = (
    "MEMORY.md",
    "STATUS.md",
    "ptsip.yaml",
    "spec",
    "adoption",
    "agents",
    "docs/planning",
)


class AgentContextMigrationError(RuntimeError):
    pass


def _yaml(base: Path, relative: str) -> dict[str, object]:
    from developer.automation.policy_loader import load_yaml
    try:
        return load_yaml(base / relative, root=base)
    except (OSError, ValueError) as exc:
        raise AgentContextMigrationError(f"invalid required file: {relative}: {exc}") from exc


def _indexed_rule_ids(base: Path) -> set[str]:
    index = _yaml(base, AGENT_INDEX)
    entries = index.get("specs")
    if not isinstance(entries, list):
        raise AgentContextMigrationError("Agent Contract index specs must be a list")
    result: set[str] = set()
    for entry in entries:
        if not isinstance(entry, Mapping):
            raise AgentContextMigrationError("Agent Contract spec entry is invalid")
        ref = entry.get("ref")
        if not isinstance(ref, str):
            raise AgentContextMigrationError("Agent Contract spec ref is invalid")
        payload = _yaml(base / "src/agent_contracts", ref)
        rules = payload.get("rules")
        if not isinstance(rules, list):
            raise AgentContextMigrationError(f"{ref}: rules must be a list")
        for rule in rules:
            if not isinstance(rule, Mapping) or not isinstance(rule.get("rule_id"), str):
                raise AgentContextMigrationError(f"{ref}: invalid rule")
            result.add(rule["rule_id"])
    return result


def _strings(value: object):
    if isinstance(value, str):
        yield value
    elif isinstance(value, Mapping):
        for item in value.values():
            yield from _strings(item)
    elif isinstance(value, list):
        for item in value:
            yield from _strings(item)


def verify_current_profile_selectors(root: str | Path | None = None) -> dict[str, object]:
    """Revalidate bounded current declarations even if retired paths reappear."""

    base = repository_root(root)
    profiles: list[str] = []
    violations: list[dict[str, str]] = []
    errors: list[str] = []

    try:
        profiles.append(str(resolve_state("project_profile", base)["ref"]))
        index = load_repository_index(base)
        if index is None:
            raise AgentContextMigrationError("missing required file: .ptsip/index.json")
        namespace = index["namespaces"]["profiles"]
        if namespace["status"] != "ACTIVE":
            raise AgentContextMigrationError(".ptsip/index.json: profiles namespace must be ACTIVE")
        catalog_path = base / ".ptsip" / namespace["index"]
        if not catalog_path.is_file():
            raise AgentContextMigrationError(
                f"missing required file: {LOCAL_PROFILE_CATALOG.as_posix()}"
            )
        if load_local_profile_selection(base) is None:
            raise AgentContextMigrationError("canonical local profile catalog did not resolve")
        # The selector loader validates every catalog row; enumerate that exact catalog,
        # including registered profiles other than its default selection.
        catalog = json.loads(catalog_path.read_text(encoding="utf-8-sig"))
        profiles.extend(
            (LOCAL_PROFILE_ROOT / row["resource"]).as_posix()
            for row in catalog["profiles"]
        )
    except (
        AgentContextMigrationError,
        LocalProfileCatalogError,
        RepositoryNamespaceError,
        RepositoryStateResolutionError,
        KeyError,
        OSError,
        TypeError,
        ValueError,
        yaml.YAMLError,
    ) as exc:
        errors.append(str(exc))

    for relative in dict.fromkeys(profiles):
        try:
            (base / relative).resolve().relative_to(base)
            profile = _yaml(base, relative)
        except (AgentContextMigrationError, OSError, ValueError, yaml.YAMLError) as exc:
            errors.append(f"{relative}: {exc}")
            continue
        for collection, fields in (
            ("components", ("include", "analysis_inputs")),
            ("associated_artifacts", ("include",)),
        ):
            items = profile.get(collection, [] if collection == "associated_artifacts" else None)
            if not isinstance(items, list) or (collection == "components" and not items):
                required = "non-empty " if collection == "components" else ""
                errors.append(f"{relative}: {collection} must be a {required}list")
                continue
            for position, item in enumerate(items):
                location = f"{collection}[{position}]"
                if (
                    not isinstance(item, Mapping)
                    or not isinstance(item.get("id"), str)
                    or not item["id"]
                ):
                    errors.append(f"{relative}: {location} must be a mapping with a non-empty id")
                    continue
                location = f"{collection}[{item['id']}]"
                for field in fields:
                    selectors = item.get(field, [] if field == "analysis_inputs" else None)
                    if (
                        not isinstance(selectors, list)
                        or (field == "include" and not selectors)
                        or not all(
                            isinstance(selector, str) and normalize_selector(selector)
                            for selector in selectors
                        )
                    ):
                        required = "non-empty " if field == "include" else ""
                        errors.append(
                            f"{relative}: {location}.{field} must be a "
                            f"{required}list of non-empty selectors"
                        )
                        continue
                    for selector in selectors:
                        normalized = normalize_selector(selector)
                        retired_root = next(
                            (
                                retired for retired in RETIRED_PROFILE_ROOTS
                                if normalized == retired or normalized.startswith(retired + "/")
                            ),
                            None,
                        )
                        if retired_root is not None:
                            violations.append({
                                "profile": relative,
                                "item": f"{location}.{field}",
                                "selector": selector,
                                "retired_root": retired_root,
                            })

    return {
        "id": "CURRENT_PROJECT_PROFILE_SELECTORS_REVALIDATED",
        "status": "PASS" if not errors and not violations else "FAIL",
        "detail": {
            "profiles": list(dict.fromkeys(profiles)),
            "retired_selectors": violations,
            "errors": errors,
        },
    }


def verify_operation_implementation_refs(root: str | Path | None = None) -> dict[str, object]:
    """Check source references in the repository without importing implementations."""

    base = repository_root(root)
    errors: list[str] = []
    checked: list[dict[str, str]] = []
    contract_root = (base / EMBEDDED_AGENT_ROOT).resolve()
    try:
        contract_root.relative_to(base)
        (contract_root / "index.yaml").resolve().relative_to(contract_root)
        (contract_root / "operations").resolve().relative_to(contract_root)
        index = _yaml(base, f"{EMBEDDED_AGENT_ROOT}/index.yaml")
        operations = index.get("operations")
        if not isinstance(operations, list) or not operations:
            raise AgentContextMigrationError("embedded index operations must be a non-empty list")
    except (AgentContextMigrationError, OSError, ValueError, yaml.YAMLError) as exc:
        return {
            "id": "OPERATION_IMPLEMENTATION_REFS_EXIST",
            "status": "FAIL",
            "detail": {"checked": [], "errors": [str(exc)]},
        }

    seen: set[str] = set()
    for entry in operations:
        ref = entry.get("ref") if isinstance(entry, Mapping) else None
        if not isinstance(ref, str):
            errors.append(f"embedded operation entry has invalid ref: {ref!r}")
            continue
        path = PurePosixPath(ref)
        if (
            len(path.parts) != 2
            or path.parts[0] != "operations"
            or path.suffix != ".yaml"
            or path.as_posix() != ref
            or "\\" in ref
        ):
            errors.append(f"unsafe embedded operation ref: {ref!r}")
            continue
        if ref in seen:
            errors.append(f"duplicate embedded operation ref: {ref}")
            continue
        seen.add(ref)
        operation_path = f"{EMBEDDED_AGENT_ROOT}/{ref}"
        try:
            (base / operation_path).resolve().relative_to(contract_root)
            operation = _yaml(base, operation_path)
        except (AgentContextMigrationError, OSError, ValueError, yaml.YAMLError) as exc:
            errors.append(f"{operation_path}: {exc}")
            continue
        operation_id = operation.get("operation_id")
        refs = operation.get("implementation_refs")
        if not isinstance(operation_id, str) or not operation_id:
            errors.append(f"{operation_path}: operation_id must be a non-empty string")
            continue
        if not isinstance(refs, list) or not refs:
            errors.append(f"{operation_path} ({operation_id}): implementation_refs must be non-empty")
            continue
        for implementation in refs:
            location = f"{operation_path} ({operation_id}) implementation_refs"
            if not isinstance(implementation, str) or not implementation:
                errors.append(f"{location}: invalid ref {implementation!r}")
                continue
            relative = PurePosixPath(implementation)
            if (
                relative.is_absolute()
                or PureWindowsPath(implementation).drive
                or ".." in relative.parts
                or "\\" in implementation
                or relative.as_posix() != implementation
                or any(token in implementation for token in ("*", "?", "[", ":"))
            ):
                errors.append(f"{location}: unsafe ref {implementation!r}")
                continue
            try:
                implementation_path = (base / implementation).resolve()
                implementation_path.relative_to(base)
                if not implementation_path.is_file():
                    raise AgentContextMigrationError("referenced implementation file does not exist")
            except (AgentContextMigrationError, OSError, ValueError) as exc:
                errors.append(f"{location}: {implementation}: {exc}")
                continue
            checked.append({"operation": operation_id, "ref": implementation})

    unindexed = sorted(
        f"operations/{path.name}"
        for path in (contract_root / "operations").glob("*.yaml")
        if f"operations/{path.name}" not in seen
    )
    if unindexed:
        errors.append(f"embedded operation resources missing from index: {unindexed}")

    return {
        "id": "OPERATION_IMPLEMENTATION_REFS_EXIST",
        "status": "FAIL" if errors else "PASS",
        "detail": {"checked": checked, "errors": errors},
    }


def verify(stage: str = "M5", root: str | Path | None = None) -> dict[str, object]:
    base = repository_root(root)
    stage = stage.upper()
    if stage not in {"M5", "M8", "AUTO"}:
        raise AgentContextMigrationError(f"unsupported verification stage: {stage}")

    agent_index = _yaml(base, AGENT_INDEX)
    contract_set = agent_index.get("contract_set")
    if not isinstance(contract_set, Mapping):
        raise AgentContextMigrationError("Agent Contract contract_set is missing")
    mpd = _yaml(base, MPD_0012)
    policy = mpd.get("policy")
    if not isinstance(policy, Mapping):
        raise AgentContextMigrationError("MPD-0012 policy identity is missing")

    if stage == "AUTO":
        stage = (
            "M8"
            if contract_set.get("status") == "CURRENT" and policy.get("status") == "ACTIVE"
            else "M5"
        )

    checks: list[dict[str, object]] = []

    try:
        counts = validate_agent_contract_plane()
        checks.append({"id": "AGENT_CONTRACT_PLANE_STRUCTURAL_VALIDATION_PASS", "status": "PASS", "detail": counts})
    except Exception as exc:
        checks.append({"id": "AGENT_CONTRACT_PLANE_STRUCTURAL_VALIDATION_PASS", "status": "FAIL", "detail": str(exc)})

    checks.append(verify_operation_implementation_refs(base))

    coverage = _yaml(base, COVERAGE)
    m1 = coverage.get("m1")
    expected = (
        set(m1.get("stable_rule_gap_before_m2", []))
        if isinstance(m1, Mapping)
        else set()
    )
    actual = _indexed_rule_ids(base)
    missing = sorted(expected - actual)
    checks.append({
        "id": "M1_STABLE_RULE_GAP_MATERIALIZED",
        "status": "PASS" if expected and not missing else "FAIL",
        "detail": {"expected": len(expected), "missing": missing},
    })

    operation_failures: dict[str, str] = {}
    operation_sizes: dict[str, object] = {}
    for operation_id in OPERATIONS:
        try:
            result = resolve_operation(operation_id)
            markdown_refs = [
                value for value in _strings(result)
                if value.casefold().endswith(".md") or ".md#" in value.casefold()
            ]
            if markdown_refs:
                raise AgentContextMigrationError(
                    f"resolved operation contains Markdown dependency: {markdown_refs[:3]}"
                )
            operation_sizes[operation_id] = {
                "rules": len(result["rules"]),
                "actions": len(result["actions"]),
                "conditions": len(result["conditions"]),
                "gates": len(result["gates"]),
                "io_schemas": len(result["io_schemas"]),
                "vocabularies": len(result["vocabularies"]),
            }
        except Exception as exc:
            operation_failures[operation_id] = str(exc)
    checks.append({
        "id": "REPRESENTATIVE_OPERATIONS_RESOLVE_EXACTLY",
        "status": "PASS" if not operation_failures else "FAIL",
        "detail": {"failures": operation_failures, "sizes": operation_sizes},
    })

    state_failures: dict[str, str] = {}
    state_refs: dict[str, str] = {}
    for domain in STATE_DOMAINS:
        try:
            result = resolve_state(domain, base)
            state_refs[domain] = str(result["ref"])
        except Exception as exc:
            state_failures[domain] = str(exc)
    state_index = _yaml(base, STATE_INDEX)
    defaults = state_index.get("default_agent_context")
    expected_defaults = {
        "prose_history_required": False,
        "status_markdown_required": False,
        "memory_markdown_required": False,
        "reference_markdown_required": False,
    }
    state_ok = not state_failures and defaults == expected_defaults
    checks.append({
        "id": "BOUNDED_REPOSITORY_STATE_ROUTING",
        "status": "PASS" if state_ok else "FAIL",
        "detail": {"failures": state_failures, "refs": state_refs, "default_agent_context": defaults},
    })

    dependency = agent_index.get("dependency_policy")
    markdown_forbidden = (
        isinstance(dependency, Mapping)
        and dependency.get("markdown_normative_dependency") == "FORBIDDEN"
    )
    checks.append({
        "id": "NORMATIVE_MARKDOWN_DEPENDENCY_FORBIDDEN",
        "status": "PASS" if markdown_forbidden else "FAIL",
        "detail": {"dependency_policy": dependency},
    })

    if stage == "M8":
        active = policy.get("status") == "ACTIVE"
        current = contract_set.get("status") == "CURRENT"
        remaining = [path for path in NORMATIVE_MARKDOWN_TARGETS if (base / path).exists()]
        checks.extend([
            {
                "id": "MPD_0012_ACTIVE",
                "status": "PASS" if active else "FAIL",
                "detail": {"status": policy.get("status")},
            },
            {
                "id": "AGENT_CONTRACT_SET_CURRENT",
                "status": "PASS" if current else "FAIL",
                "detail": {"status": contract_set.get("status")},
            },
            {
                "id": "NORMATIVE_MARKDOWN_TARGETS_REMOVED",
                "status": "PASS" if not remaining else "FAIL",
                "detail": {"remaining": remaining},
            },
            {
                "id": "DEFAULT_AGENT_NORMATIVE_MARKDOWN_READ_COUNT_ZERO",
                "status": "PASS" if state_ok and markdown_forbidden else "FAIL",
                "detail": {"count": 0 if state_ok and markdown_forbidden else None},
            },
        ])
        checks.append(verify_current_profile_selectors(base))

    passed = all(item["status"] == "PASS" for item in checks)
    return {
        "schema_version": "ptsip-agent-context-migration-verification/v1",
        "stage": stage,
        "status": "PASS" if passed else "FAIL",
        "checks": checks,
    }


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Verify the MPD-0012 agent-context machine migration.")
    sub = parser.add_subparsers(dest="command", required=True)
    verify_parser = sub.add_parser("verify")
    verify_parser.add_argument("--stage", default="AUTO")
    verify_parser.add_argument("--repository", default=".")
    args = parser.parse_args(argv)
    try:
        result = verify(args.stage, args.repository)
    except (AgentContextMigrationError, OSError, ValueError, yaml.YAMLError) as exc:
        print(json.dumps({"status": "FAIL", "error": str(exc)}, sort_keys=True))
        return 2
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0 if result["status"] == "PASS" else 2


if __name__ == "__main__":
    raise SystemExit(main())
