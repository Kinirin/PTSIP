from __future__ import annotations

import argparse
import copy
import json
import os
import re
import subprocess
import tempfile
from dataclasses import dataclass
from pathlib import Path
from typing import Mapping, Sequence

import yaml

from developer.automation.policy_loader import repository_root
from developer.automation.pp_transition_delta import (
    GitIndexSnapshot,
    GitSnapshot,
    PPTransitionDeltaError,
    SnapshotSource,
    compare_git_snapshots,
    load_authority_state,
)
from developer.automation.project_profile_registry import (
    EMBEDDED_PP_CONTRACT_REGISTRY,
    PP_CONTRACT_REGISTRY,
    PUBLIC_PROFILE_CATALOG,
    validate_project_profile_registry_plane,
)


class PPTransitionReconcileError(RuntimeError):
    def __init__(self, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code


@dataclass(frozen=True)
class TransitionPlan:
    status: str
    base_current: str
    target: str
    reasons: tuple[str, ...]
    outputs: Mapping[str, bytes]
    exact_stage_paths: tuple[str, ...]

    def as_dict(self) -> dict[str, object]:
        return {
            "status": self.status,
            "base_current": self.base_current,
            "target": self.target,
            "reasons": list(self.reasons),
            "outputs": list(self.exact_stage_paths),
        }


_VERSION_LINE = re.compile(
    r'^(?P<prefix>\s+version:\s*)(?P<value>"[^"]*"|\'[^\']*\'|[^#\s]+)(?P<suffix>\s*(?:#.*)?)$'
)


def _git(root: Path, *args: str) -> subprocess.CompletedProcess[bytes]:
    return subprocess.run(
        ["git", *args],
        cwd=root,
        check=False,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )


def _require_git(root: Path, *args: str) -> bytes:
    result = _git(root, *args)
    if result.returncode != 0:
        raise PPTransitionReconcileError(
            "GIT_OPERATION_FAILED",
            result.stderr.decode("utf-8", errors="replace").strip()
            or f"git {' '.join(args)} failed",
        )
    return result.stdout


def _yaml(raw: bytes | None, *, label: str) -> dict[str, object]:
    if raw is None:
        raise PPTransitionReconcileError(
            "REQUIRED_AUTHORITY_INPUT_MISSING",
            f"{label} is missing.",
        )
    try:
        payload = yaml.safe_load(raw.decode("utf-8"))
    except (UnicodeDecodeError, yaml.YAMLError) as exc:
        raise PPTransitionReconcileError(
            "AUTHORITY_INPUT_INVALID",
            f"{label} is not valid UTF-8 YAML: {exc}",
        ) from exc
    if not isinstance(payload, dict):
        raise PPTransitionReconcileError(
            "AUTHORITY_INPUT_INVALID",
            f"{label} must be a mapping.",
        )
    return payload


def _dump_yaml(payload: Mapping[str, object]) -> bytes:
    return yaml.safe_dump(
        dict(payload),
        sort_keys=False,
        allow_unicode=True,
        width=120,
    ).encode("utf-8")


def _rebind_profile(raw: bytes, *, source: str, target: str, label: str) -> bytes:
    payload = _yaml(raw, label=label)
    ptsip = payload.get("ptsip")
    if not isinstance(ptsip, Mapping):
        raise PPTransitionReconcileError(
            "PUBLIC_PROFILE_INVALID",
            f"{label}: ptsip mapping is missing.",
        )
    declared = ptsip.get("version")
    if declared not in {source, target}:
        raise PPTransitionReconcileError(
            "PUBLIC_PROFILE_VERSION_MISMATCH",
            f"{label}: expected {source!r} or {target!r}, got {declared!r}.",
        )
    if declared == target:
        return raw

    lines = raw.decode("utf-8").splitlines(keepends=True)
    in_ptsip = False
    replaced = False
    for index, line in enumerate(lines):
        stripped = line.rstrip("\r\n")
        if stripped == "ptsip:":
            in_ptsip = True
            continue
        if not in_ptsip:
            continue
        if stripped and not stripped.startswith((" ", "\t")):
            break
        match = _VERSION_LINE.fullmatch(stripped)
        if match is None:
            continue
        newline = line[len(stripped):]
        quote = '"' if match.group("value").startswith('"') else (
            "'" if match.group("value").startswith("'") else ""
        )
        rendered = f"{quote}{target}{quote}" if quote else target
        lines[index] = (
            match.group("prefix")
            + rendered
            + match.group("suffix")
            + newline
        )
        replaced = True
        break

    if not replaced:
        raise PPTransitionReconcileError(
            "PUBLIC_PROFILE_VERSION_LINE_UNRESOLVED",
            f"{label}: cannot locate ptsip.version for exact rebind.",
        )
    rebound = "".join(lines).encode("utf-8")
    rebound_payload = _yaml(rebound, label=label)
    rebound_ptsip = rebound_payload.get("ptsip")
    if not isinstance(rebound_ptsip, Mapping) or rebound_ptsip.get("version") != target:
        raise PPTransitionReconcileError(
            "PUBLIC_PROFILE_REBIND_FAILED",
            f"{label}: generated profile does not declare {target!r}.",
        )
    return rebound


def _schema_filename_token(version: str) -> str:
    if not version.startswith("pp."):
        raise PPTransitionReconcileError(
            "PP_IDENTITY_INVALID",
            f"Project Profile identity {version!r} is not canonical.",
        )
    return "pp-" + version[len("pp."):]


def _schema_path(version: str) -> str:
    return f"schemas/ptsip-profile-{_schema_filename_token(version)}.schema.json"


def _embedded_schema_path(version: str) -> str:
    return (
        "src/ptsip/specdata/ptsip-profile-"
        + _schema_filename_token(version)
        + ".schema.json"
    )


def _history_root(version: str) -> str:
    return f"profiles/history/{version}"


def _reidentity_schema(raw: bytes, *, target: str) -> bytes:
    try:
        payload = json.loads(raw.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise PPTransitionReconcileError(
            "CURRENT_PP_SCHEMA_INVALID",
            f"current PP schema is not valid UTF-8 JSON: {exc}",
        ) from exc
    if not isinstance(payload, dict):
        raise PPTransitionReconcileError(
            "CURRENT_PP_SCHEMA_INVALID",
            "current PP schema must be a JSON object.",
        )

    payload["$id"] = (
        "https://raw.githubusercontent.com/Kinirin/PTSIP/main/"
        + _schema_path(target)
    )
    payload["title"] = f"PTSIP Project Profile {target}"
    try:
        version_contract = payload["properties"]["ptsip"]["properties"]["version"]
    except (KeyError, TypeError) as exc:
        raise PPTransitionReconcileError(
            "CURRENT_PP_SCHEMA_INVALID",
            "current PP schema has no properties.ptsip.properties.version contract.",
        ) from exc
    if not isinstance(version_contract, dict):
        raise PPTransitionReconcileError(
            "CURRENT_PP_SCHEMA_INVALID",
            "current PP schema version contract must be a mapping.",
        )
    version_contract["const"] = target
    version_contract["description"] = "Canonical Project Profile contract identity."
    return (json.dumps(payload, indent=2, ensure_ascii=False) + "\n").encode("utf-8")


def _candidate_catalog(candidate: SnapshotSource) -> dict[str, object]:
    return _yaml(
        candidate.read_bytes(PUBLIC_PROFILE_CATALOG),
        label=f"{candidate.label}:{PUBLIC_PROFILE_CATALOG}",
    )


def _base_registry(base: SnapshotSource) -> dict[str, object]:
    return _yaml(
        base.read_bytes(PP_CONTRACT_REGISTRY),
        label=f"{base.label}:{PP_CONTRACT_REGISTRY}",
    )


def _registry_candidate_must_be_unmodified(
    base: SnapshotSource,
    candidate: SnapshotSource,
) -> None:
    left = base.read_bytes(PP_CONTRACT_REGISTRY)
    right = candidate.read_bytes(PP_CONTRACT_REGISTRY)
    if left != right:
        raise PPTransitionReconcileError(
            "MANUAL_PP_REGISTRY_MUTATION",
            "PP contract registry is automation-owned during an unreconciled T2 transition.",
        )


def _build_registry(
    base_registry: Mapping[str, object],
    *,
    source: str,
    target: str,
) -> bytes:
    registry = copy.deepcopy(dict(base_registry))
    contracts = registry.get("contracts")
    transitions = registry.get("transitions")
    if not isinstance(contracts, list) or not isinstance(transitions, list):
        raise PPTransitionReconcileError(
            "PP_CONTRACT_REGISTRY_INVALID",
            "PP contract registry contracts/transitions must be lists.",
        )

    source_contracts = [
        item
        for item in contracts
        if isinstance(item, dict) and item.get("version") == source
    ]
    if len(source_contracts) != 1:
        raise PPTransitionReconcileError(
            "PP_SOURCE_CONTRACT_UNRESOLVED",
            f"source contract {source!r} must resolve exactly once.",
        )
    source_contract = source_contracts[0]
    if any(isinstance(item, Mapping) and item.get("version") == target for item in contracts):
        raise PPTransitionReconcileError(
            "PP_TARGET_CONTRACT_ALREADY_EXISTS",
            f"target contract {target!r} already exists before reconciliation.",
        )

    source_contract["lifecycle"] = "SUPERSEDED"
    operations = source_contract.get("operations")
    if not isinstance(operations, list) or not operations:
        raise PPTransitionReconcileError(
            "PP_SOURCE_OPERATIONS_INVALID",
            f"source contract {source!r} has no operation set.",
        )

    contracts.append(
        {
            "version": target,
            "lifecycle": "CURRENT",
            "operations": list(operations),
            "schema": _schema_path(target),
            "baseline": _history_root(target),
        }
    )
    transitions.append(
        {
            "from": source,
            "to": target,
            "kind": "SEMANTIC_MIGRATION",
        }
    )
    registry["current"] = target
    return _dump_yaml(registry)


def _rebind_catalog(
    payload: Mapping[str, object],
    *,
    target: str,
) -> bytes:
    catalog = copy.deepcopy(dict(payload))
    profiles = catalog.get("profiles")
    if not isinstance(profiles, list) or not profiles:
        raise PPTransitionReconcileError(
            "PUBLIC_PROFILE_CATALOG_INVALID",
            "public profile catalog profiles must be a non-empty list.",
        )
    for item in profiles:
        if not isinstance(item, dict):
            raise PPTransitionReconcileError(
                "PUBLIC_PROFILE_CATALOG_INVALID",
                "public profile catalog entries must be mappings.",
            )
        item["contract"] = target
    return _dump_yaml(catalog)


def _verify_history_baseline(
    base: SnapshotSource,
    *,
    source: str,
) -> None:
    state = load_authority_state(base)
    prefix = _history_root(source)
    for resource in state.discovered_resources:
        historical = base.read_bytes(f"{prefix}/{resource}")
        current = base.read_bytes(f"profiles/{resource}")
        if historical is None or historical != current:
            raise PPTransitionReconcileError(
                "HISTORICAL_BASELINE_INVALID",
                f"{prefix}/{resource} must be an immutable byte-identical source baseline.",
            )


def _staged_history_paths(root: Path, *, base_revision: str) -> tuple[str, ...]:
    raw = _require_git(
        root,
        "diff",
        "--cached",
        "--name-only",
        base_revision,
        "--",
        "profiles/history",
    ).decode("utf-8", errors="replace")
    return tuple(
        sorted(
            line.strip().replace("\\", "/")
            for line in raw.splitlines()
            if line.strip()
        )
    )


def _reject_staged_history_mutation(root: Path, *, base_revision: str) -> None:
    paths = _staged_history_paths(root, base_revision=base_revision)
    if paths:
        raise PPTransitionReconcileError(
            "HISTORICAL_BASELINE_MUTATION",
            "historical Public Profile baselines are immutable: " + ", ".join(paths),
        )


def build_transition_plan(
    root: str | Path,
    *,
    base_revision: str = "HEAD",
) -> TransitionPlan:
    repo = repository_root(root)
    delta = compare_git_snapshots(
        repo,
        base_revision=base_revision,
        staged=True,
    )
    if not delta.valid:
        raise PPTransitionReconcileError(delta.classification, json.dumps(delta.as_dict(), sort_keys=True))
    if not delta.triggered:
        if delta.classification == "NO_T2_AUTHORITY_DELTA":
            current = delta.base_current or delta.candidate_current
            if current is None:
                raise PPTransitionReconcileError(
                    "PP_CURRENT_UNRESOLVED",
                    "current PP identity is unresolved.",
                )
            return TransitionPlan(
                status="NO_CHANGE",
                base_current=current,
                target=current,
                reasons=(),
                outputs={},
                exact_stage_paths=(),
            )
        raise PPTransitionReconcileError(
            delta.classification,
            "transition reconciler requires an actual T2 authority delta.",
        )
    if delta.base_current is None or delta.expected_next is None:
        raise PPTransitionReconcileError(
            "PP_TRANSITION_TARGET_UNRESOLVED",
            "T2 delta did not resolve source and adjacent-minor target identities.",
        )
    if delta.candidate_already_reconciled:
        return _verify_already_reconciled(
            repo,
            base_revision=base_revision,
            source=delta.base_current,
            target=delta.expected_next,
            reasons=delta.reasons,
        )

    _reject_staged_history_mutation(repo, base_revision=base_revision)

    base = GitSnapshot(repo, base_revision)
    candidate = GitIndexSnapshot(repo)
    _registry_candidate_must_be_unmodified(base, candidate)
    _verify_history_baseline(base, source=delta.base_current)

    catalog = _candidate_catalog(candidate)
    profiles = catalog.get("profiles")
    if not isinstance(profiles, list) or not profiles:
        raise PPTransitionReconcileError(
            "PUBLIC_PROFILE_CATALOG_INVALID",
            "candidate Public Profile catalog is empty.",
        )

    outputs: dict[str, bytes] = {}
    generated_profiles: dict[str, bytes] = {}
    for item in profiles:
        if not isinstance(item, Mapping):
            raise PPTransitionReconcileError(
                "PUBLIC_PROFILE_CATALOG_INVALID",
                "candidate Public Profile catalog entry is invalid.",
            )
        resource = item.get("resource")
        if not isinstance(resource, str):
            raise PPTransitionReconcileError(
                "PUBLIC_PROFILE_CATALOG_INVALID",
                "candidate Public Profile resource must be a string.",
            )
        raw = candidate.read_bytes(f"profiles/{resource}")
        if raw is None:
            raise PPTransitionReconcileError(
                "PUBLIC_PROFILE_RESOURCE_MISSING",
                f"candidate Public Profile {resource!r} is missing.",
            )
        rebound = _rebind_profile(
            raw,
            source=delta.base_current,
            target=delta.expected_next,
            label=f"profiles/{resource}",
        )
        generated_profiles[resource] = rebound
        outputs[f"profiles/{resource}"] = rebound
        outputs[f"{_history_root(delta.expected_next)}/{resource}"] = rebound

    base_state = load_authority_state(base)
    old_schema_path = base_state.contract_schemas.get(delta.base_current)
    if not isinstance(old_schema_path, str):
        raise PPTransitionReconcileError(
            "CURRENT_PP_SCHEMA_UNRESOLVED",
            f"source contract {delta.base_current!r} has no canonical schema.",
        )
    base_schema = base.read_bytes(old_schema_path)
    candidate_schema = candidate.read_bytes(old_schema_path)
    if base_schema is None or candidate_schema is None:
        raise PPTransitionReconcileError(
            "CURRENT_PP_SCHEMA_MISSING",
            f"source schema {old_schema_path!r} is missing.",
        )

    schema_source = (
        candidate_schema
        if "CURRENT_PP_CANONICAL_SCHEMA_CONTENT" in delta.reasons
        else base_schema
    )
    new_schema = _reidentity_schema(schema_source, target=delta.expected_next)
    outputs[_schema_path(delta.expected_next)] = new_schema
    outputs[_embedded_schema_path(delta.expected_next)] = new_schema

    if "CURRENT_PP_CANONICAL_SCHEMA_CONTENT" in delta.reasons:
        outputs[old_schema_path] = base_schema

    outputs[PUBLIC_PROFILE_CATALOG] = _rebind_catalog(
        catalog,
        target=delta.expected_next,
    )
    generated_registry = _build_registry(
        _base_registry(base),
        source=delta.base_current,
        target=delta.expected_next,
    )
    outputs[PP_CONTRACT_REGISTRY] = generated_registry
    outputs[EMBEDDED_PP_CONTRACT_REGISTRY] = generated_registry

    paths = tuple(sorted(outputs))
    return TransitionPlan(
        status="RECONCILE",
        base_current=delta.base_current,
        target=delta.expected_next,
        reasons=delta.reasons,
        outputs=outputs,
        exact_stage_paths=paths,
    )


def _verify_already_reconciled(
    root: Path,
    *,
    base_revision: str,
    source: str,
    target: str,
    reasons: tuple[str, ...],
) -> TransitionPlan:
    candidate = GitIndexSnapshot(root)
    base = GitSnapshot(root, base_revision)
    base_state = load_authority_state(base)
    source_prefix = _history_root(source)
    target_prefix = _history_root(target)
    staged_history = _staged_history_paths(root, base_revision=base_revision)
    for path in staged_history:
        if not path.startswith(target_prefix + "/"):
            raise PPTransitionReconcileError(
                "HISTORICAL_BASELINE_MUTATION",
                f"historical baseline mutation is forbidden: {path}",
            )
    for resource in base_state.discovered_resources:
        source_path = f"{source_prefix}/{resource}"
        if base.read_bytes(source_path) != candidate.read_bytes(source_path):
            raise PPTransitionReconcileError(
                "HISTORICAL_BASELINE_MUTATION",
                f"historical baseline mutation is forbidden: {source_path}",
            )

    registry = _yaml(
        candidate.read_bytes(PP_CONTRACT_REGISTRY),
        label=f"STAGED_INDEX:{PP_CONTRACT_REGISTRY}",
    )
    if registry.get("current") != target:
        raise PPTransitionReconcileError(
            "RECONCILED_CURRENT_MISMATCH",
            f"candidate registry current is not {target!r}.",
        )
    transitions = registry.get("transitions")
    if not isinstance(transitions, list) or not any(
        isinstance(item, Mapping)
        and item.get("from") == source
        and item.get("to") == target
        and item.get("kind") == "SEMANTIC_MIGRATION"
        for item in transitions
    ):
        raise PPTransitionReconcileError(
            "RECONCILED_TRANSITION_MISSING",
            f"candidate registry does not record {source!r} -> {target!r}.",
        )

    canonical_registry = candidate.read_bytes(PP_CONTRACT_REGISTRY)
    embedded_registry = candidate.read_bytes(EMBEDDED_PP_CONTRACT_REGISTRY)
    if canonical_registry is None or canonical_registry != embedded_registry:
        raise PPTransitionReconcileError(
            "RECONCILED_RUNTIME_REGISTRY_PROJECTION_MISMATCH",
            "reconciled embedded PP registry must be byte-identical to canonical registry.",
        )

    catalog = _candidate_catalog(candidate)
    profiles = catalog.get("profiles")
    if not isinstance(profiles, list) or not profiles:
        raise PPTransitionReconcileError(
            "PUBLIC_PROFILE_CATALOG_INVALID",
            "reconciled catalog is empty.",
        )
    for item in profiles:
        if not isinstance(item, Mapping) or item.get("contract") != target:
            raise PPTransitionReconcileError(
                "RECONCILED_CATALOG_MISMATCH",
                "every reconciled Public Profile catalog entry must bind the target PP.",
            )
        resource = item.get("resource")
        if not isinstance(resource, str):
            raise PPTransitionReconcileError(
                "PUBLIC_PROFILE_CATALOG_INVALID",
                "reconciled Public Profile resource must be a string.",
            )
        current = candidate.read_bytes(f"profiles/{resource}")
        baseline = candidate.read_bytes(f"{_history_root(target)}/{resource}")
        if current is None or baseline is None or current != baseline:
            raise PPTransitionReconcileError(
                "RECONCILED_BASELINE_MISMATCH",
                f"target baseline for {resource!r} is missing or not byte-identical.",
            )

    canonical = candidate.read_bytes(_schema_path(target))
    embedded = candidate.read_bytes(_embedded_schema_path(target))
    if canonical is None or canonical != embedded:
        raise PPTransitionReconcileError(
            "RECONCILED_SCHEMA_PROJECTION_MISMATCH",
            "target canonical and embedded PP schemas must be byte-identical.",
        )

    return TransitionPlan(
        status="ALREADY_RECONCILED",
        base_current=source,
        target=target,
        reasons=reasons,
        outputs={},
        exact_stage_paths=(),
    )


def _status_for_path(root: Path, path: str) -> str:
    raw = _require_git(
        root,
        "status",
        "--porcelain=v1",
        "--untracked-files=all",
        "--",
        path,
    ).decode("utf-8", errors="replace")
    line = raw.splitlines()[0] if raw.splitlines() else ""
    return line[:2]


def _ensure_no_unstaged_conflicts(root: Path, paths: Sequence[str]) -> None:
    for path in paths:
        status = _status_for_path(root, path)
        if status == "??" or (len(status) == 2 and status[1] != " "):
            raise PPTransitionReconcileError(
                "AUTOMATION_WRITE_CONFLICT",
                f"automation-owned output {path!r} has an unstaged or untracked user change.",
            )


def _atomic_write(path: Path, content: bytes) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    handle, temporary_name = tempfile.mkstemp(
        prefix=f".{path.name}.",
        suffix=".tmp",
        dir=str(path.parent),
    )
    temporary = Path(temporary_name)
    try:
        with os.fdopen(handle, "wb") as stream:
            stream.write(content)
        temporary.replace(path)
    finally:
        if temporary.exists():
            temporary.unlink()


def _restore_snapshot(
    root: Path,
    *,
    before: Mapping[str, bytes | None],
) -> None:
    for path, content in before.items():
        candidate = root / path
        if content is None:
            if candidate.exists():
                candidate.unlink()
            _git(root, "rm", "--cached", "--ignore-unmatch", "--", path)
        else:
            _atomic_write(candidate, content)
            _require_git(root, "add", "--", path)


def apply_transition_plan(
    root: str | Path,
    plan: TransitionPlan,
) -> TransitionPlan:
    repo = repository_root(root)
    if plan.status != "RECONCILE":
        return plan

    _ensure_no_unstaged_conflicts(repo, plan.exact_stage_paths)
    index = GitIndexSnapshot(repo)
    before = {path: index.read_bytes(path) for path in plan.exact_stage_paths}

    try:
        for path, content in plan.outputs.items():
            _atomic_write(repo / path, content)
        _require_git(repo, "add", "--", *plan.exact_stage_paths)

        post = compare_git_snapshots(
            repo,
            base_revision="HEAD",
            staged=True,
        )
        if not post.valid or not post.triggered or not post.candidate_already_reconciled:
            raise PPTransitionReconcileError(
                "POST_RECONCILIATION_DELTA_INVALID",
                json.dumps(post.as_dict(), sort_keys=True),
            )

        registry_failures = validate_project_profile_registry_plane(repo)
        if registry_failures:
            raise PPTransitionReconcileError(
                "POST_RECONCILIATION_REGISTRY_INVALID",
                "\n".join(registry_failures),
            )
    except Exception:
        _restore_snapshot(repo, before=before)
        raise

    return TransitionPlan(
        status="RECONCILED",
        base_current=plan.base_current,
        target=plan.target,
        reasons=plan.reasons,
        outputs={},
        exact_stage_paths=plan.exact_stage_paths,
    )


def reconcile_staged_transition(
    root: str | Path | None = None,
    *,
    apply: bool = False,
) -> TransitionPlan:
    repo = repository_root(root)
    plan = build_transition_plan(repo)
    return apply_transition_plan(repo, plan) if apply else plan


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Reconcile an MPD-0011 T2 staged delta into one adjacent-minor PP transition."
    )
    parser.add_argument("--repository", default=".")
    parser.add_argument("--apply", action="store_true")
    parser.add_argument("--json", action="store_true")
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        result = reconcile_staged_transition(
            args.repository,
            apply=args.apply,
        )
    except (PPTransitionReconcileError, PPTransitionDeltaError) as exc:
        payload = {
            "status": "UNRESOLVED",
            "code": getattr(exc, "code", "PP_TRANSITION_UNRESOLVED"),
            "message": str(exc),
        }
        print(json.dumps(payload, sort_keys=True))
        return 2

    payload = result.as_dict()
    print(json.dumps(payload, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
