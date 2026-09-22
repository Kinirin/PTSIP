from __future__ import annotations

import argparse
import hashlib
import json
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Iterable, Mapping, Sequence

import yaml

from developer.automation.policy_loader import repository_root


POLICY_PATH = "developer/policy/MPD-0012.yaml"
WORKFLOW_PATH = "developer/automation/markdown_cleanup_workflow.yaml"
AGENT_CONTRACT_INDEX = "src/agent_contracts/index.yaml"
AGENT_CORE_SPEC = "src/agent_contracts/spec/core.yaml"
AGENT_OUTCOME_VOCAB = "src/agent_contracts/vocabulary/outcomes.yaml"

_SKIP_DIR_NAMES = {
    ".git",
    ".venv",
    "venv",
    "__pycache__",
    ".pytest_cache",
    ".mypy_cache",
    ".ruff_cache",
    "node_modules",
    "build",
    "dist",
}


class MarkdownCleanupError(RuntimeError):
    pass


@dataclass(frozen=True)
class ReferenceHit:
    source_path: str
    line: int
    kind: str

    def as_dict(self) -> dict[str, object]:
        return {
            "source_path": self.source_path,
            "line": self.line,
            "kind": self.kind,
        }


def _mapping(value: object, *, label: str) -> Mapping[str, object]:
    if not isinstance(value, Mapping):
        raise MarkdownCleanupError(f"{label} must be a mapping")
    return value


def _load_yaml(path: Path, *, label: str) -> dict[str, object]:
    try:
        payload = yaml.safe_load(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeDecodeError, yaml.YAMLError) as exc:
        raise MarkdownCleanupError(f"{label} could not be loaded: {exc}") from exc
    if not isinstance(payload, dict):
        raise MarkdownCleanupError(f"{label} must contain a YAML mapping")
    return payload


def load_cleanup_workflow(root: str | Path | None = None) -> dict[str, object]:
    base = repository_root(root)
    payload = _load_yaml(base / WORKFLOW_PATH, label="Markdown cleanup workflow")
    if payload.get("schema_version") != "ptsip-markdown-cleanup-workflow/v1":
        raise MarkdownCleanupError("unsupported Markdown cleanup workflow schema")
    if payload.get("policy_ref") != "MPD-0012#rules.approved_decisions.markdown_cleanup_preconditions":
        raise MarkdownCleanupError("cleanup workflow policy_ref must target MPD-0012 cleanup policy")
    stages = payload.get("stages")
    if stages != ["INSPECT", "PLAN", "VERIFY", "APPLY", "POST_VALIDATE"]:
        raise MarkdownCleanupError("cleanup workflow stages are not the approved deterministic order")
    scopes = payload.get("default_scopes")
    if not isinstance(scopes, list) or not scopes or not all(isinstance(item, str) and item for item in scopes):
        raise MarkdownCleanupError("cleanup workflow default_scopes must be non-empty strings")
    targets = payload.get("targets")
    if not isinstance(targets, list) or not targets:
        raise MarkdownCleanupError("cleanup workflow must register at least one target")
    seen: set[str] = set()
    for raw in targets:
        item = _mapping(raw, label="cleanup target")
        path = item.get("path")
        role = item.get("semantic_role")
        disposition = item.get("disposition")
        if not isinstance(path, str) or not path.endswith(".md"):
            raise MarkdownCleanupError("cleanup target path must be a Markdown path")
        if path in seen:
            raise MarkdownCleanupError(f"duplicate cleanup target: {path}")
        seen.add(path)
        if role not in {
            "NORMATIVE_RULE",
            "CODING_AGENT_BEHAVIOR",
            "OPERATION_PROCEDURE",
            "VALIDATION_OR_CONFORMANCE_SEMANTICS",
            "GOVERNANCE_OR_AUTHORITY_SEMANTICS",
            "PROFILE_TRANSITION_SEMANTICS",
            "NORMATIVE_TERMINOLOGY",
        }:
            raise MarkdownCleanupError(f"{path}: unsupported semantic_role {role!r}")
        if disposition != "REMOVE_CANDIDATE":
            raise MarkdownCleanupError(f"{path}: disposition must be REMOVE_CANDIDATE")
    return payload


def _load_policy(root: Path) -> dict[str, object]:
    payload = _load_yaml(root / POLICY_PATH, label="MPD-0012")
    identity = _mapping(payload.get("policy"), label="MPD-0012.policy")
    if identity.get("id") != "MPD-0012":
        raise MarkdownCleanupError("cleanup policy identity must be MPD-0012")
    rules = _mapping(payload.get("rules"), label="MPD-0012.rules")
    approved = _mapping(rules.get("approved_decisions"), label="MPD-0012 approved_decisions")
    cleanup = _mapping(
        approved.get("markdown_cleanup_preconditions"),
        label="MPD-0012 markdown_cleanup_preconditions",
    )
    if cleanup.get("decision") != "APPROVED":
        raise MarkdownCleanupError("MPD-0012 Markdown cleanup decision is not APPROVED")
    return payload


def _git_output(root: Path, *args: str) -> str | None:
    result = subprocess.run(
        ["git", *args],
        cwd=root,
        check=False,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    if result.returncode != 0:
        return None
    value = result.stdout.strip()
    return value or None


def _git_head(root: Path) -> str | None:
    return _git_output(root, "rev-parse", "--verify", "HEAD^{commit}")


def _repository_files(root: Path) -> tuple[Path, ...]:
    raw = subprocess.run(
        ["git", "ls-files", "-co", "--exclude-standard", "-z"],
        cwd=root,
        check=False,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    if raw.returncode == 0:
        result: list[Path] = []
        for item in raw.stdout.decode("utf-8", errors="strict").split("\0"):
            if not item:
                continue
            candidate = root / item
            if candidate.is_file():
                result.append(candidate)
        return tuple(sorted(set(result)))

    result = []
    for candidate in root.rglob("*"):
        if not candidate.is_file():
            continue
        relative = candidate.relative_to(root)
        if any(part in _SKIP_DIR_NAMES for part in relative.parts):
            continue
        result.append(candidate)
    return tuple(sorted(result))


def _read_text_lines(path: Path) -> list[str] | None:
    try:
        if path.stat().st_size > 2_000_000:
            return None
        return path.read_text(encoding="utf-8", errors="strict").splitlines()
    except (OSError, UnicodeDecodeError):
        return None


def _is_human_reference_source(relative: str) -> bool:
    return Path(relative).suffix.lower() in {".md", ".rst"}


def scan_reference_hits(
    root: str | Path,
    target_path: str,
    *,
    exclusions: Iterable[str] = (),
) -> tuple[ReferenceHit, ...]:
    base = repository_root(root)
    excluded = {item.replace("\\", "/") for item in exclusions}
    target = target_path.replace("\\", "/")
    forms = {target, target.replace("/", "\\")}
    hits: list[ReferenceHit] = []
    for source in _repository_files(base):
        relative = source.relative_to(base).as_posix()
        if relative == target or relative in excluded:
            continue
        lines = _read_text_lines(source)
        if lines is None:
            continue
        for line_no, line in enumerate(lines, start=1):
            if not any(form in line for form in forms):
                continue
            hits.append(
                ReferenceHit(
                    source_path=relative,
                    line=line_no,
                    kind=(
                        "HISTORICAL_OR_HUMAN_REFERENCE"
                        if _is_human_reference_source(relative)
                        else "ACTIVE_MACHINE_DEPENDENCY"
                    ),
                )
            )
    return tuple(hits)


def _target_index(workflow: Mapping[str, object]) -> dict[str, Mapping[str, object]]:
    targets = workflow.get("targets")
    if not isinstance(targets, list):
        raise MarkdownCleanupError("cleanup workflow targets must be a list")
    return {
        str(_mapping(item, label="cleanup target")["path"]): _mapping(item, label="cleanup target")
        for item in targets
    }


def _scoped_markdown_paths(root: Path, scopes: Sequence[str]) -> tuple[str, ...]:
    result: set[str] = set()
    for raw_scope in scopes:
        scope = (root / raw_scope).resolve()
        try:
            scope.relative_to(root)
        except ValueError as exc:
            raise MarkdownCleanupError(f"cleanup scope escapes repository: {raw_scope}") from exc
        if not scope.exists():
            continue
        if scope.is_file():
            candidates = (scope,)
        else:
            candidates = tuple(scope.rglob("*.md"))
        for candidate in candidates:
            if candidate.is_file() and candidate.suffix.lower() == ".md":
                result.add(candidate.relative_to(root).as_posix())
    return tuple(sorted(result))


def _sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def inspect_repository(
    root: str | Path | None = None,
    *,
    scopes: Sequence[str] | None = None,
) -> dict[str, object]:
    base = repository_root(root)
    workflow = load_cleanup_workflow(base)
    _load_policy(base)
    selected_scopes = tuple(scopes or workflow["default_scopes"])
    target_index = _target_index(workflow)
    exclusions_raw = workflow.get("reference_scan_exclusions", [])
    if not isinstance(exclusions_raw, list) or not all(isinstance(item, str) for item in exclusions_raw):
        raise MarkdownCleanupError("reference_scan_exclusions must be a list of strings")
    exclusions = tuple(exclusions_raw)

    inventory = set(_scoped_markdown_paths(base, selected_scopes))
    inventory.update(
        path
        for path in target_index
        if any(path == scope or path.startswith(scope.rstrip("/") + "/") for scope in selected_scopes)
    )

    items: list[dict[str, object]] = []
    for relative in sorted(inventory):
        candidate = base / relative
        registered = target_index.get(relative)
        hits = scan_reference_hits(base, relative, exclusions=exclusions)
        active = [hit for hit in hits if hit.kind == "ACTIVE_MACHINE_DEPENDENCY"]
        if not candidate.is_file():
            if registered is not None and active:
                state = "BLOCKED_BY_ACTIVE_REFERENCE"
            else:
                state = "ALREADY_ABSENT" if registered is not None else "UNRESOLVED"
            digest = None
        else:
            if registered is None:
                state = "UNRESOLVED"
            elif active:
                state = "BLOCKED_BY_ACTIVE_REFERENCE"
            else:
                state = "REMOVE_CANDIDATE"
            digest = _sha256(candidate)

        items.append(
            {
                "path": relative,
                "registered": registered is not None,
                "semantic_role": None if registered is None else registered.get("semantic_role"),
                "state": state,
                "sha256": digest,
                "references": [hit.as_dict() for hit in hits],
            }
        )

    counts: dict[str, int] = {}
    for item in items:
        state = str(item["state"])
        counts[state] = counts.get(state, 0) + 1

    return {
        "schema_version": "ptsip-markdown-cleanup-inspection/v1",
        "policy_ref": workflow["policy_ref"],
        "repository_head": _git_head(base),
        "scopes": list(selected_scopes),
        "counts": counts,
        "items": items,
    }


def _agent_contract_structural_check() -> tuple[str, dict[str, int] | None, str | None]:
    try:
        from agent_contracts.validator import validate_agent_contract_plane

        counts = validate_agent_contract_plane()
        return "PASS", counts, None
    except Exception as exc:  # fail closed; validator owns detailed semantics
        return "FAIL", None, f"{type(exc).__name__}: {exc}"


def _normative_source_check(root: Path) -> tuple[str, dict[str, object]]:
    index = _load_yaml(root / AGENT_CONTRACT_INDEX, label="Agent Contract index")
    contract_set = _mapping(index.get("contract_set"), label="agent contract_set")
    dependency = _mapping(index.get("dependency_policy"), label="agent dependency_policy")
    status = contract_set.get("status")
    authority_scope = contract_set.get("authority_scope")
    markdown_dependency = dependency.get("markdown_normative_dependency")
    passed = (
        status == "CURRENT"
        and authority_scope == "CODING_AGENT_BEHAVIOR"
        and markdown_dependency == "FORBIDDEN"
    )
    return (
        "PASS" if passed else "FAIL",
        {
            "contract_set_status": status,
            "authority_scope": authority_scope,
            "markdown_normative_dependency": markdown_dependency,
        },
    )


def _unresolved_semantics_check(root: Path) -> tuple[str, dict[str, object]]:
    core = _load_yaml(root / AGENT_CORE_SPEC, label="Agent core spec")
    rules = core.get("rules")
    fail_closed = False
    if isinstance(rules, list):
        for raw in rules:
            if not isinstance(raw, Mapping):
                continue
            if (
                raw.get("subject") == "unresolved_normative_input"
                and raw.get("predicate") == "behavior"
                and raw.get("object") == "FAIL_CLOSED"
            ):
                fail_closed = True
                break

    outcomes = _load_yaml(root / AGENT_OUTCOME_VOCAB, label="Agent outcome vocabulary")
    entries = outcomes.get("entries")
    ids = {
        str(item.get("id"))
        for item in entries
        if isinstance(entries, list) and isinstance(item, Mapping)
    } if isinstance(entries, list) else set()
    required = {"OWNER_DECISION_REQUIRED", "INCOMPLETE"}
    passed = fail_closed and required.issubset(ids)
    return (
        "PASS" if passed else "FAIL",
        {
            "fail_closed_unresolved_normative_input": fail_closed,
            "required_outcomes_present": sorted(required.intersection(ids)),
            "required_outcomes_missing": sorted(required - ids),
        },
    )


def verify_cleanup_readiness(
    root: str | Path | None = None,
    *,
    inspection: Mapping[str, object] | None = None,
) -> dict[str, object]:
    base = repository_root(root)
    policy = _load_policy(base)
    inspected = dict(inspection or inspect_repository(base))

    structural_status, structural_counts, structural_error = _agent_contract_structural_check()
    source_status, source_detail = _normative_source_check(base)
    unresolved_status, unresolved_detail = _unresolved_semantics_check(base)

    items = inspected.get("items")
    if not isinstance(items, list):
        raise MarkdownCleanupError("inspection items must be a list")
    active_refs = [
        {
            "target": item.get("path"),
            "source_path": ref.get("source_path"),
            "line": ref.get("line"),
        }
        for item in items
        if isinstance(item, Mapping)
        for ref in item.get("references", [])
        if isinstance(ref, Mapping) and ref.get("kind") == "ACTIVE_MACHINE_DEPENDENCY"
    ]
    no_active_status = "PASS" if not active_refs else "FAIL"

    identity = _mapping(policy.get("policy"), label="MPD-0012.policy")
    apply_authorized = identity.get("status") == "ACTIVE"

    checks = [
        {
            "id": "AGENT_CONTRACT_PLANE_STRUCTURAL_VALIDATION_PASS",
            "status": structural_status,
            "detail": {
                "counts": structural_counts,
                "error": structural_error,
            },
        },
        {
            "id": "NO_ACTIVE_MACHINE_DEPENDENCY_ON_REMOVAL_TARGETS",
            "status": no_active_status,
            "detail": {"active_references": active_refs},
        },
        {
            "id": "SRC_AGENT_CONTRACTS_CONFIRMED_AS_NORMATIVE_SOURCE",
            "status": source_status,
            "detail": source_detail,
        },
        {
            "id": "UNRESOLVED_SEMANTICS_CAN_BE_REPRESENTED_WITHOUT_LEGACY_FALLBACK",
            "status": unresolved_status,
            "detail": unresolved_detail,
        },
    ]
    ready = all(item["status"] == "PASS" for item in checks)
    return {
        "schema_version": "ptsip-markdown-cleanup-readiness/v1",
        "policy_status": identity.get("status"),
        "cleanup_readiness": "READY" if ready else "BLOCKED",
        "apply_authorized": bool(apply_authorized),
        "apply_authorization_reason": (
            "MPD-0012_ACTIVE"
            if apply_authorized
            else "MPD-0012_NOT_ACTIVE"
        ),
        "checks": checks,
    }


def build_cleanup_plan(
    root: str | Path | None = None,
    *,
    scopes: Sequence[str] | None = None,
) -> dict[str, object]:
    base = repository_root(root)
    inspection = inspect_repository(base, scopes=scopes)
    readiness = verify_cleanup_readiness(base, inspection=inspection)
    items = inspection["items"]
    remove = [item["path"] for item in items if item["state"] == "REMOVE_CANDIDATE"]
    blocked = [item["path"] for item in items if item["state"] == "BLOCKED_BY_ACTIVE_REFERENCE"]
    unresolved = [item["path"] for item in items if item["state"] == "UNRESOLVED"]

    ready = (
        not blocked
        and not unresolved
        and readiness["cleanup_readiness"] == "READY"
    )
    target_hashes = {
        item["path"]: item["sha256"]
        for item in items
        if item.get("sha256") is not None
    }
    return {
        "schema_version": "ptsip-markdown-cleanup-plan/v1",
        "policy_ref": inspection["policy_ref"],
        "repository_head": inspection["repository_head"],
        "state": "READY" if ready else "BLOCKED",
        "apply_authorized": readiness["apply_authorized"],
        "remove": remove,
        "blocked": blocked,
        "unresolved": unresolved,
        "target_hashes": target_hashes,
        "inspection": inspection,
        "readiness": readiness,
    }


def _load_plan(path: Path) -> dict[str, object]:
    try:
        payload = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise MarkdownCleanupError(f"cleanup plan could not be loaded: {exc}") from exc
    if not isinstance(payload, dict) or payload.get("schema_version") != "ptsip-markdown-cleanup-plan/v1":
        raise MarkdownCleanupError("cleanup plan schema is invalid")
    return payload


def _validate_plan_freshness(root: Path, plan: Mapping[str, object]) -> None:
    expected_head = plan.get("repository_head")
    current_head = _git_head(root)
    if expected_head != current_head:
        raise MarkdownCleanupError(
            f"cleanup plan is stale: expected HEAD {expected_head!r}, current {current_head!r}"
        )
    hashes = plan.get("target_hashes")
    if not isinstance(hashes, Mapping):
        raise MarkdownCleanupError("cleanup plan target_hashes must be a mapping")
    for raw_path, raw_hash in hashes.items():
        path = str(raw_path)
        expected = str(raw_hash)
        candidate = root / path
        if not candidate.is_file():
            raise MarkdownCleanupError(f"cleanup target disappeared after planning: {path}")
        if _sha256(candidate) != expected:
            raise MarkdownCleanupError(f"cleanup target changed after planning: {path}")


def apply_cleanup_plan(
    root: str | Path | None,
    plan: Mapping[str, object],
) -> dict[str, object]:
    base = repository_root(root)
    policy = _load_policy(base)
    identity = _mapping(policy.get("policy"), label="MPD-0012.policy")
    if identity.get("status") != "ACTIVE":
        raise MarkdownCleanupError("physical cleanup is forbidden while MPD-0012 is not ACTIVE")
    if plan.get("state") != "READY" or plan.get("apply_authorized") is not True:
        raise MarkdownCleanupError("cleanup plan is not authorized for apply")

    _validate_plan_freshness(base, plan)
    remove = plan.get("remove")
    if not isinstance(remove, list) or not all(isinstance(item, str) for item in remove):
        raise MarkdownCleanupError("cleanup plan remove list is invalid")

    backups: dict[Path, bytes] = {}
    try:
        for relative in remove:
            candidate = (base / relative).resolve()
            try:
                candidate.relative_to(base)
            except ValueError as exc:
                raise MarkdownCleanupError(f"cleanup target escapes repository: {relative}") from exc
            if not candidate.is_file():
                raise MarkdownCleanupError(f"cleanup target is missing: {relative}")
            backups[candidate] = candidate.read_bytes()

        for candidate in backups:
            candidate.unlink()

        post_inspection = inspect_repository(base)
        post_readiness = verify_cleanup_readiness(base, inspection=post_inspection)
        dangling = [
            item["path"]
            for item in post_inspection["items"]
            if item["state"] == "BLOCKED_BY_ACTIVE_REFERENCE"
        ]
        if dangling or post_readiness["cleanup_readiness"] != "READY":
            raise MarkdownCleanupError(
                "post-cleanup validation failed; repository was restored"
            )
    except Exception:
        for candidate, content in backups.items():
            candidate.parent.mkdir(parents=True, exist_ok=True)
            candidate.write_bytes(content)
        raise

    return {
        "schema_version": "ptsip-markdown-cleanup-apply/v1",
        "status": "APPLIED",
        "removed": list(remove),
        "post_readiness": post_readiness,
    }


def simulate_cleanup(
    root: str | Path | None = None,
    *,
    scopes: Sequence[str] | None = None,
) -> dict[str, object]:
    plan = build_cleanup_plan(root, scopes=scopes)
    return {
        "schema_version": "ptsip-markdown-cleanup-simulation/v1",
        "status": "READY" if plan["state"] == "READY" else "BLOCKED",
        "mutation_performed": False,
        "plan": plan,
    }


def _emit(payload: object, *, output: str | None = None) -> None:
    text = json.dumps(payload, indent=2, ensure_ascii=False, sort_keys=True)
    if output:
        Path(output).write_text(text + "\n", encoding="utf-8")
    print(text)


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="python -m developer.automation.markdown_cleanup",
        description="Deterministic MPD-0012 Markdown cleanup inspection, planning, verification, and guarded apply.",
    )
    parser.add_argument("--repository", default=".")
    sub = parser.add_subparsers(dest="command", required=True)

    for name in ("inspect", "plan", "simulate"):
        command = sub.add_parser(name)
        command.add_argument("--scope", action="append", default=[])
        command.add_argument("--output")

    verify = sub.add_parser("verify")
    verify.add_argument("--scope", action="append", default=[])
    verify.add_argument("--output")

    apply_parser = sub.add_parser("apply")
    apply_parser.add_argument("--plan", required=True)
    apply_parser.add_argument("--output")
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    scopes = tuple(getattr(args, "scope", []) or ()) or None
    try:
        if args.command == "inspect":
            payload = inspect_repository(args.repository, scopes=scopes)
        elif args.command == "plan":
            payload = build_cleanup_plan(args.repository, scopes=scopes)
        elif args.command == "verify":
            inspection = inspect_repository(args.repository, scopes=scopes)
            payload = verify_cleanup_readiness(args.repository, inspection=inspection)
        elif args.command == "simulate":
            payload = simulate_cleanup(args.repository, scopes=scopes)
        else:
            payload = apply_cleanup_plan(
                args.repository,
                _load_plan(Path(args.plan)),
            )
        _emit(payload, output=getattr(args, "output", None))
    except (MarkdownCleanupError, OSError, ValueError, yaml.YAMLError) as exc:
        print(
            json.dumps(
                {
                    "status": "FAIL",
                    "error": str(exc),
                },
                ensure_ascii=False,
                sort_keys=True,
            ),
            file=sys.stderr,
        )
        return 2

    if args.command == "verify" and payload["cleanup_readiness"] != "READY":
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
