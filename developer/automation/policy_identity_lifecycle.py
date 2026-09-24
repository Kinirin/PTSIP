from __future__ import annotations

import argparse
import json
import os
import re
import tempfile
from dataclasses import dataclass
from pathlib import Path
from typing import Mapping, Sequence

import yaml
from jsonschema import Draft202012Validator

from developer.automation.policy_loader import load_json, load_yaml, repository_root


INDEX = "developer/policy/index.yaml"
SUBJECT_REGISTRY = "developer/policy/registries/authority-subject-registry.yaml"
APPROVAL_SCHEMA = "developer/policy/schemas/policy-approval-provenance.schema.json"
MANAGEMENT_POLICY_SCHEMA = "developer/policy/schemas/management-policy.schema.json"
APPROVAL_ROOT = Path("developer/policy/approvals")
_POLICY_ID_RE = re.compile(r"^MPD-([0-9]{4})$")


class PolicyIdentityLifecycleError(RuntimeError):
    def __init__(self, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code


@dataclass(frozen=True)
class CorpusState:
    ids: tuple[str, ...]
    index: dict[str, object]
    subject_registry: dict[str, object]


def _mapping(value: object, *, label: str) -> Mapping[str, object]:
    if not isinstance(value, Mapping):
        raise PolicyIdentityLifecycleError("INVALID_CONTROL_PLANE", f"{label} must be a mapping")
    return value


def _approval_path(approval_ref: str | Path, *, base: Path) -> Path:
    path = Path(approval_ref)
    if not path.is_absolute():
        path = base / path
    try:
        relative = path.resolve().relative_to(base.resolve())
    except ValueError as exc:
        raise PolicyIdentityLifecycleError(
            "APPROVAL_PROVENANCE_OUTSIDE_REPOSITORY",
            f"approval provenance must be inside the repository: {path}",
        ) from exc
    if APPROVAL_ROOT not in (relative, *relative.parents):
        raise PolicyIdentityLifecycleError(
            "APPROVAL_PROVENANCE_OUTSIDE_CANONICAL_ROOT",
            f"approval provenance must be under {APPROVAL_ROOT.as_posix()}/",
        )
    return path


def _load_approval(approval_ref: str | Path, *, base: Path) -> dict[str, object]:
    path = _approval_path(approval_ref, base=base)
    if not path.is_file():
        raise PolicyIdentityLifecycleError(
            "APPROVAL_PROVENANCE_NOT_FOUND",
            f"approval provenance does not exist: {path.relative_to(base).as_posix()}",
        )
    payload = load_yaml(path, root=base)
    schema = load_json(APPROVAL_SCHEMA, root=base)
    errors = tuple(Draft202012Validator(schema).iter_errors(payload))
    if errors:
        detail = "; ".join(error.message for error in errors)
        raise PolicyIdentityLifecycleError("INVALID_APPROVAL_PROVENANCE", detail)
    approval = _mapping(payload.get("approval"), label="approval")
    if approval.get("decision") != "APPROVED":
        raise PolicyIdentityLifecycleError("APPROVAL_NOT_GRANTED", "approval.decision must be APPROVED")
    return dict(approval)


def _index_entries(index: Mapping[str, object]) -> list[Mapping[str, object]]:
    entries = index.get("policies")
    if not isinstance(entries, list):
        raise PolicyIdentityLifecycleError("INVALID_POLICY_INDEX", "developer policy index policies must be a list")
    result: list[Mapping[str, object]] = []
    for item in entries:
        result.append(_mapping(item, label="developer policy index entry"))
    return result


def _subject_values(registry: Mapping[str, object]) -> list[str]:
    schemes = _mapping(registry.get("subject_identity_schemes"), label="subject_identity_schemes")
    scheme = _mapping(schemes.get("MANAGEMENT_POLICY_ID"), label="MANAGEMENT_POLICY_ID")
    values = scheme.get("registered_values")
    if not isinstance(values, list) or not all(isinstance(item, str) for item in values):
        raise PolicyIdentityLifecycleError(
            "INVALID_SUBJECT_REGISTRY",
            "MANAGEMENT_POLICY_ID.registered_values must be a string list",
        )
    return list(values)


def _load_consistent_registered_corpus(base: Path) -> CorpusState:
    index = load_yaml(INDEX, root=base)
    registry = load_yaml(SUBJECT_REGISTRY, root=base)
    entries = _index_entries(index)
    ids = tuple(str(item.get("id")) for item in entries)
    paths = tuple(str(item.get("path")) for item in entries)

    if len(ids) != len(set(ids)):
        raise PolicyIdentityLifecycleError("DUPLICATE_POLICY_ID", "developer policy index contains duplicate IDs")
    if ids != tuple(sorted(ids)):
        raise PolicyIdentityLifecycleError("NONCANONICAL_POLICY_INDEX_ORDER", "developer policy index IDs are not ordered")

    expected_paths = tuple(f"developer/policy/{policy_id}.yaml" for policy_id in ids)
    if paths != expected_paths:
        raise PolicyIdentityLifecycleError(
            "POLICY_INDEX_PATH_MISMATCH",
            "developer policy index paths do not exactly match their policy IDs",
        )

    subject_ids = tuple(_subject_values(registry))
    if subject_ids != ids:
        raise PolicyIdentityLifecycleError(
            "SUBJECT_REGISTRY_MISMATCH",
            "authority subject registry IDs must exactly match the developer policy index",
        )

    for entry in entries:
        policy_id = str(entry["id"])
        payload = load_yaml(str(entry["path"]), root=base)
        policy = _mapping(payload.get("policy"), label=f"{policy_id}.policy")
        if policy.get("id") != policy_id:
            raise PolicyIdentityLifecycleError("POLICY_FILE_ID_MISMATCH", f"{policy_id}: policy.id mismatch")
        if policy.get("status") != entry.get("status"):
            raise PolicyIdentityLifecycleError("POLICY_STATUS_MISMATCH", f"{policy_id}: file/index status mismatch")

    return CorpusState(ids=ids, index=index, subject_registry=registry)


def _discover_policy_ids(base: Path) -> tuple[str, ...]:
    return tuple(path.stem for path in sorted((base / "developer/policy").glob("MPD-[0-9][0-9][0-9][0-9].yaml")))


def _assert_no_unregistered_policy_file(base: Path, state: CorpusState) -> None:
    discovered = _discover_policy_ids(base)
    if discovered != state.ids:
        raise PolicyIdentityLifecycleError(
            "POLICY_CORPUS_MISMATCH",
            "policy files, index, and subject identity registry must match exactly before allocation",
        )


def _next_policy_id(ids: Sequence[str]) -> str:
    numbers: list[int] = []
    for policy_id in ids:
        match = _POLICY_ID_RE.fullmatch(policy_id)
        if match is None:
            raise PolicyIdentityLifecycleError("INVALID_POLICY_ID", f"invalid policy ID: {policy_id}")
        numbers.append(int(match.group(1)))
    number = (max(numbers) + 1) if numbers else 1
    if number > 9999:
        raise PolicyIdentityLifecycleError("POLICY_ID_SPACE_EXHAUSTED", "MPD four-digit identity space exhausted")
    return f"MPD-{number:04d}"


def inspect_policy(policy_id: str, *, root: str | Path | None = None) -> dict[str, object]:
    base = repository_root(root)
    state = _load_consistent_registered_corpus(base)
    _assert_no_unregistered_policy_file(base, state)
    entries = {str(item["id"]): item for item in _index_entries(state.index)}
    entry = entries.get(policy_id)
    if entry is None:
        return {
            "status": "NOT_FOUND",
            "policy_id": policy_id,
            "next_available_id": _next_policy_id(state.ids),
        }

    payload = load_yaml(str(entry["path"]), root=base)
    policy = _mapping(payload.get("policy"), label=f"{policy_id}.policy")
    rules = _mapping(payload.get("rules"), label=f"{policy_id}.rules")
    authority = rules.get("authority_semantics")
    runtime_authority = None
    if isinstance(authority, Mapping):
        runtime_authority = authority.get("runtime_authority")
    return {
        "status": "FOUND",
        "policy_id": policy_id,
        "title": policy.get("title"),
        "policy_status": policy.get("status"),
        "index_status": entry.get("status"),
        "subject_identity_registered": policy_id in state.ids,
        "operationally_resolvable": policy.get("status") == "ACTIVE",
        "declared_runtime_authority": runtime_authority,
        "path": entry.get("path"),
    }


def preflight_new_policy(
    approval_ref: str | Path,
    *,
    root: str | Path | None = None,
) -> dict[str, object]:
    base = repository_root(root)
    state = _load_consistent_registered_corpus(base)
    _assert_no_unregistered_policy_file(base, state)
    approval = _load_approval(approval_ref, base=base)
    requested = approval.get("requested_policy_id")
    allocated = _next_policy_id(state.ids)

    if isinstance(requested, str):
        if requested in state.ids:
            existing = inspect_policy(requested, root=base)
            raise PolicyIdentityLifecycleError(
                "POLICY_ID_ALREADY_EXISTS",
                f"{requested} already exists with status {existing.get('policy_status')}",
            )
        if requested != allocated:
            raise PolicyIdentityLifecycleError(
                "REQUESTED_POLICY_ID_NOT_NEXT_AVAILABLE",
                f"requested {requested}, next available is {allocated}",
            )

    return {
        "status": "READY",
        "allocated_policy_id": allocated,
        "target_status": approval["target_status"],
        "approval_id": approval["approval_id"],
        "approval_scope": approval["approval_scope"],
        "implementation_authorized": approval["implementation_authorized"],
        "policy_content_review_scope": approval["policy_content_review_scope"],
        "registry_mutation_required": True,
    }


def status_preflight(
    policy_id: str,
    approval_ref: str | Path,
    *,
    root: str | Path | None = None,
) -> dict[str, object]:
    base = repository_root(root)
    state = _load_consistent_registered_corpus(base)
    _assert_no_unregistered_policy_file(base, state)
    approval = _load_approval(approval_ref, base=base)
    requested = approval.get("requested_policy_id")
    if requested != policy_id:
        raise PolicyIdentityLifecycleError(
            "APPROVAL_POLICY_ID_MISMATCH",
            "status change approval must explicitly bind requested_policy_id to the existing policy",
        )
    inspected = inspect_policy(policy_id, root=base)
    if inspected["status"] != "FOUND":
        raise PolicyIdentityLifecycleError("POLICY_NOT_FOUND", f"{policy_id} does not exist")
    return {
        "status": "READY",
        "policy_id": policy_id,
        "current_status": inspected["policy_status"],
        "target_status": approval["target_status"],
        "approval_id": approval["approval_id"],
        "implementation_authorized": approval["implementation_authorized"],
    }


def _atomic_write_yaml(path: Path, payload: Mapping[str, object]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    handle, temp_name = tempfile.mkstemp(prefix=f".{path.name}.", suffix=".tmp", dir=path.parent)
    try:
        with os.fdopen(handle, "w", encoding="utf-8", newline="\n") as stream:
            yaml.safe_dump(dict(payload), stream, sort_keys=False, allow_unicode=True)
        os.replace(temp_name, path)
    except Exception:
        try:
            os.unlink(temp_name)
        except OSError:
            pass
        raise


def register_policy(
    approval_ref: str | Path,
    policy_file: str | Path,
    *,
    root: str | Path | None = None,
) -> dict[str, object]:
    base = repository_root(root)
    state = _load_consistent_registered_corpus(base)
    approval = _load_approval(approval_ref, base=base)
    allocated = _next_policy_id(state.ids)
    requested = approval.get("requested_policy_id")
    if isinstance(requested, str) and requested != allocated:
        raise PolicyIdentityLifecycleError(
            "REQUESTED_POLICY_ID_NOT_NEXT_AVAILABLE",
            f"requested {requested}, next available is {allocated}",
        )

    expected_relative = Path(f"developer/policy/{allocated}.yaml")
    candidate = Path(policy_file)
    if candidate.is_absolute():
        try:
            relative = candidate.resolve().relative_to(base.resolve())
        except ValueError as exc:
            raise PolicyIdentityLifecycleError("POLICY_FILE_OUTSIDE_REPOSITORY", str(candidate)) from exc
    else:
        relative = candidate
        candidate = base / candidate
    if relative.as_posix() != expected_relative.as_posix():
        raise PolicyIdentityLifecycleError(
            "POLICY_FILE_PATH_MISMATCH",
            f"expected {expected_relative.as_posix()}, got {relative.as_posix()}",
        )
    if not candidate.is_file():
        raise PolicyIdentityLifecycleError("POLICY_FILE_NOT_FOUND", relative.as_posix())

    discovered = _discover_policy_ids(base)
    expected_discovered = tuple((*state.ids, allocated))
    if discovered != expected_discovered:
        raise PolicyIdentityLifecycleError(
            "POLICY_CORPUS_MISMATCH",
            "registration requires exactly one new next-ID policy file and no other unregistered MPD files",
        )

    payload = load_yaml(relative, root=base)
    schema = load_json(MANAGEMENT_POLICY_SCHEMA, root=base)
    errors = tuple(Draft202012Validator(schema).iter_errors(payload))
    if errors:
        raise PolicyIdentityLifecycleError(
            "INVALID_POLICY_FILE",
            "; ".join(error.message for error in errors),
        )
    policy = _mapping(payload.get("policy"), label=f"{allocated}.policy")
    if policy.get("id") != allocated:
        raise PolicyIdentityLifecycleError("POLICY_FILE_ID_MISMATCH", f"expected {allocated}")
    if policy.get("status") != approval["target_status"]:
        raise PolicyIdentityLifecycleError(
            "APPROVED_STATUS_MISMATCH",
            f"policy status {policy.get('status')!r} != approved target {approval['target_status']!r}",
        )

    original_index = (base / INDEX).read_text(encoding="utf-8")
    original_registry = (base / SUBJECT_REGISTRY).read_text(encoding="utf-8")
    index = dict(state.index)
    index_entries = list(index["policies"])
    index_entries.append({
        "id": allocated,
        "path": expected_relative.as_posix(),
        "status": approval["target_status"],
    })
    index["policies"] = index_entries

    registry = dict(state.subject_registry)
    schemes = dict(_mapping(registry["subject_identity_schemes"], label="subject_identity_schemes"))
    identity = dict(_mapping(schemes["MANAGEMENT_POLICY_ID"], label="MANAGEMENT_POLICY_ID"))
    identity["registered_values"] = [*state.ids, allocated]
    schemes["MANAGEMENT_POLICY_ID"] = identity
    registry["subject_identity_schemes"] = schemes

    try:
        _atomic_write_yaml(base / INDEX, index)
        _atomic_write_yaml(base / SUBJECT_REGISTRY, registry)
    except Exception:
        (base / INDEX).write_text(original_index, encoding="utf-8")
        (base / SUBJECT_REGISTRY).write_text(original_registry, encoding="utf-8")
        raise

    return {
        "status": "REGISTERED",
        "policy_id": allocated,
        "policy_status": approval["target_status"],
        "approval_id": approval["approval_id"],
        "index": INDEX,
        "subject_registry": SUBJECT_REGISTRY,
    }


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Deterministic MPD identity and lifecycle preflight.")
    parser.add_argument("--root")
    sub = parser.add_subparsers(dest="command", required=True)

    inspect_cmd = sub.add_parser("inspect")
    inspect_cmd.add_argument("policy_id")

    preflight_cmd = sub.add_parser("preflight")
    preflight_cmd.add_argument("--approval-ref", required=True)

    register_cmd = sub.add_parser("register")
    register_cmd.add_argument("--approval-ref", required=True)
    register_cmd.add_argument("--policy-file", required=True)

    status_cmd = sub.add_parser("status-preflight")
    status_cmd.add_argument("policy_id")
    status_cmd.add_argument("--approval-ref", required=True)
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        if args.command == "inspect":
            result = inspect_policy(args.policy_id, root=args.root)
        elif args.command == "preflight":
            result = preflight_new_policy(args.approval_ref, root=args.root)
        elif args.command == "register":
            result = register_policy(args.approval_ref, args.policy_file, root=args.root)
        else:
            result = status_preflight(args.policy_id, args.approval_ref, root=args.root)
    except PolicyIdentityLifecycleError as exc:
        print(json.dumps({"status": "BLOCKED", "code": exc.code, "message": str(exc)}, sort_keys=True))
        return 2
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
