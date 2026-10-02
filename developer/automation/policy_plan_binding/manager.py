from __future__ import annotations

import argparse
import json
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any, Mapping, Sequence

import yaml

from developer.automation.policy_loader import repository_root

from .errors import PolicyPlanBindingError
from .reconciler import validate_registry_integrity
from .registry import load_registry
from .resolver import binding_entries
from .store import replace_registry


@dataclass(frozen=True)
class PolicyPlanBindingMutation:
    status: str
    changed: bool
    binding: dict[str, Any]

    def to_payload(self) -> dict[str, Any]:
        return asdict(self)


def _next_binding_id(payload: Mapping[str, Any]) -> str:
    maximum = 0
    for binding in binding_entries(payload):
        value = binding.get("binding_id")
        if not isinstance(value, str) or not value.startswith("PPB-"):
            continue
        try:
            maximum = max(maximum, int(value[4:]))
        except ValueError:
            continue
    return f"PPB-{maximum + 1:04d}"


def _write(
    payload: Mapping[str, Any],
    *,
    expected_digest: str,
    root: str | Path | None,
) -> None:
    replace_registry(
        payload,
        validator=lambda value: validate_registry_integrity(value, root=root),
        expected_digest=expected_digest,
        root=root,
    )


def _plan_identity(
    plan_ref: str,
    *,
    root: str | Path | None,
) -> dict[str, str]:
    base = repository_root(root).resolve()
    candidate = (base / plan_ref).resolve()
    try:
        relative = candidate.relative_to(base)
    except ValueError as exc:
        raise PolicyPlanBindingError(
            "PLAN_REF_OUTSIDE_REPOSITORY",
            f"plan_ref escapes repository root: {plan_ref}",
        ) from exc

    relative_text = relative.as_posix()
    if not relative_text.startswith("developer/planning/"):
        raise PolicyPlanBindingError(
            "PLAN_REF_OUTSIDE_PLANNING_NAMESPACE",
            f"plan_ref must resolve under developer/planning/: {plan_ref}",
        )
    if not candidate.is_file():
        raise PolicyPlanBindingError(
            "PLAN_REF_NOT_FOUND",
            f"plan_ref does not exist: {plan_ref}",
        )

    payload = yaml.safe_load(candidate.read_text(encoding="utf-8"))
    if not isinstance(payload, Mapping):
        raise PolicyPlanBindingError(
            "PLAN_DOCUMENT_INVALID",
            f"plan document must be a mapping: {plan_ref}",
        )
    identity = payload.get("plan_identity")
    if not isinstance(identity, Mapping):
        raise PolicyPlanBindingError(
            "PLAN_IDENTITY_MISSING",
            f"plan document has no plan_identity mapping: {plan_ref}",
        )

    required = ("resolved_plan_id", "plan_file_id", "version", "revision")
    result: dict[str, str] = {}
    for field in required:
        value = identity.get(field)
        if not isinstance(value, str) or not value:
            raise PolicyPlanBindingError(
                "PLAN_IDENTITY_INCOMPLETE",
                f"plan_identity.{field} is required: {plan_ref}",
            )
        result[field] = value
    return result


def _require_plan_identity_match(
    *,
    plan_ref: str,
    resolved_plan_id: str,
    plan_file_id: str,
    root: str | Path | None,
    version: str | None = None,
    revision: str | None = None,
) -> dict[str, str]:
    identity = _plan_identity(plan_ref, root=root)

    if identity["resolved_plan_id"] != resolved_plan_id:
        raise PolicyPlanBindingError(
            "RESOLVED_PLAN_ID_MISMATCH",
            f"{plan_ref}: expected resolved_plan_id {resolved_plan_id}, "
            f"got {identity['resolved_plan_id']}",
        )
    if identity["plan_file_id"] != plan_file_id:
        raise PolicyPlanBindingError(
            "PLAN_FILE_ID_MISMATCH",
            f"{plan_ref}: expected plan_file_id {plan_file_id}, "
            f"got {identity['plan_file_id']}",
        )
    if version is not None and identity["version"] != version:
        raise PolicyPlanBindingError(
            "PLAN_VERSION_MISMATCH",
            f"{plan_ref}: expected version {version}, got {identity['version']}",
        )
    if revision is not None and identity["revision"] != revision:
        raise PolicyPlanBindingError(
            "PLAN_REVISION_MISMATCH",
            f"{plan_ref}: expected revision {revision}, got {identity['revision']}",
        )
    return identity


def create_binding(
    policy_ref: str,
    *,
    root: str | Path | None = None,
) -> PolicyPlanBindingMutation:
    """Create one explicit NOT_CREATED Policy ↔ Planning relationship."""

    snapshot = load_registry(root, required=True, validate_schema=True)
    if snapshot is None:  # pragma: no cover
        raise PolicyPlanBindingError("BINDING_REGISTRY_MISSING", "binding registry is required.")

    payload = dict(snapshot.payload)
    bindings = [dict(item) for item in binding_entries(payload)]
    binding = {
        "binding_id": _next_binding_id(payload),
        "policy_ref": policy_ref,
        "planning_state": "NOT_CREATED",
    }
    bindings.append(binding)
    payload["bindings"] = bindings
    _write(payload, expected_digest=snapshot.digest, root=root)
    return PolicyPlanBindingMutation(status="CREATED", changed=True, binding=binding)


def _select_link_target(
    bindings: list[dict[str, Any]],
    *,
    policy_ref: str,
    binding_id: str | None,
) -> dict[str, Any]:
    if binding_id is not None:
        matches = [
            item for item in bindings
            if item.get("binding_id") == binding_id
            and item.get("policy_ref") == policy_ref
        ]
        if len(matches) != 1:
            raise PolicyPlanBindingError(
                "BINDING_NOT_FOUND",
                f"no exact binding {binding_id} exists for policy {policy_ref}.",
            )
        return matches[0]

    matches = [
        item for item in bindings
        if item.get("policy_ref") == policy_ref
        and item.get("planning_state") == "NOT_CREATED"
    ]
    if not matches:
        raise PolicyPlanBindingError(
            "UNCREATED_BINDING_NOT_FOUND",
            f"policy {policy_ref} has no NOT_CREATED binding to materialize.",
        )
    if len(matches) != 1:
        raise PolicyPlanBindingError(
            "AMBIGUOUS_UNCREATED_BINDING",
            f"policy {policy_ref} has {len(matches)} NOT_CREATED bindings; binding_id is required.",
        )
    return matches[0]


def link_plan(
    *,
    policy_ref: str,
    resolved_plan_id: str,
    plan_file_id: str,
    version: str,
    revision: str,
    plan_ref: str,
    binding_id: str | None = None,
    root: str | Path | None = None,
) -> PolicyPlanBindingMutation:
    """Materialize one binding using the approved v2 Plan identity contract."""

    _require_plan_identity_match(
        plan_ref=plan_ref,
        resolved_plan_id=resolved_plan_id,
        plan_file_id=plan_file_id,
        version=version,
        revision=revision,
        root=root,
    )

    snapshot = load_registry(root, required=True, validate_schema=True)
    if snapshot is None:  # pragma: no cover
        raise PolicyPlanBindingError("BINDING_REGISTRY_MISSING", "binding registry is required.")

    payload = dict(snapshot.payload)
    bindings = [dict(item) for item in binding_entries(payload)]
    target = _select_link_target(
        bindings,
        policy_ref=policy_ref,
        binding_id=binding_id,
    )

    expected = {
        "resolved_plan_id": resolved_plan_id,
        "plan_file_id": plan_file_id,
        "version": version,
        "revision": revision,
        "plan_ref": plan_ref,
    }
    if target.get("planning_state") == "CREATED":
        if all(target.get(field) == value for field, value in expected.items()):
            return PolicyPlanBindingMutation(
                status="CURRENT",
                changed=False,
                binding=dict(target),
            )
        raise PolicyPlanBindingError(
            "BINDING_ALREADY_MATERIALIZED",
            f"{target['binding_id']} is already linked to a created plan.",
        )

    duplicate = next(
        (
            item
            for item in bindings
            if item is not target
            and item.get("planning_state") == "CREATED"
            and item.get("policy_ref") == policy_ref
            and item.get("resolved_plan_id") == resolved_plan_id
        ),
        None,
    )
    if duplicate is not None:
        raise PolicyPlanBindingError(
            "DUPLICATE_CREATED_RELATION",
            f"policy {policy_ref} is already bound to {resolved_plan_id}.",
        )

    target["planning_state"] = "CREATED"
    target.update(expected)

    payload["bindings"] = bindings
    _write(payload, expected_digest=snapshot.digest, root=root)
    return PolicyPlanBindingMutation(status="LINKED", changed=True, binding=dict(target))


def move_plan_ref(
    *,
    binding_id: str,
    policy_ref: str,
    resolved_plan_id: str,
    plan_file_id: str,
    from_plan_ref: str,
    to_plan_ref: str,
    root: str | Path | None = None,
) -> PolicyPlanBindingMutation:
    """Move only plan_ref after exact registry and destination identity checks."""

    snapshot = load_registry(root, required=True, validate_schema=True)
    if snapshot is None:  # pragma: no cover
        raise PolicyPlanBindingError("BINDING_REGISTRY_MISSING", "binding registry is required.")

    payload = dict(snapshot.payload)
    bindings = [dict(item) for item in binding_entries(payload)]
    matches = [item for item in bindings if item.get("binding_id") == binding_id]
    if len(matches) != 1:
        raise PolicyPlanBindingError(
            "BINDING_NOT_FOUND",
            f"no exact binding exists for {binding_id}.",
        )

    target = matches[0]
    if target.get("planning_state") != "CREATED":
        raise PolicyPlanBindingError(
            "PLAN_NOT_CREATED",
            f"{binding_id} has no created plan whose path can be moved.",
        )
    if target.get("policy_ref") != policy_ref:
        raise PolicyPlanBindingError(
            "BINDING_POLICY_REF_MISMATCH",
            f"{binding_id}: policy_ref does not match.",
        )
    if target.get("resolved_plan_id") != resolved_plan_id:
        raise PolicyPlanBindingError(
            "BINDING_RESOLVED_PLAN_ID_MISMATCH",
            f"{binding_id}: resolved_plan_id does not match.",
        )
    if target.get("plan_file_id") != plan_file_id:
        raise PolicyPlanBindingError(
            "BINDING_PLAN_FILE_ID_MISMATCH",
            f"{binding_id}: plan_file_id does not match.",
        )
    if target.get("plan_ref") != from_plan_ref:
        raise PolicyPlanBindingError(
            "STALE_PLAN_REF",
            f"{binding_id}: current plan_ref does not match from_plan_ref.",
        )

    _require_plan_identity_match(
        plan_ref=to_plan_ref,
        resolved_plan_id=resolved_plan_id,
        plan_file_id=plan_file_id,
        root=root,
    )

    if from_plan_ref == to_plan_ref:
        return PolicyPlanBindingMutation(
            status="CURRENT",
            changed=False,
            binding=dict(target),
        )

    target["plan_ref"] = to_plan_ref
    payload["bindings"] = bindings
    _write(payload, expected_digest=snapshot.digest, root=root)
    return PolicyPlanBindingMutation(status="MOVED", changed=True, binding=dict(target))


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Manage deterministic Policy ↔ Planning binding identity and linkage."
    )
    parser.add_argument("--root")
    subparsers = parser.add_subparsers(dest="command", required=True)

    create = subparsers.add_parser("create")
    create.add_argument("--policy", required=True)

    link = subparsers.add_parser("link")
    link.add_argument("--policy", required=True)
    link.add_argument("--binding-id")
    link.add_argument("--resolved-plan-id", required=True)
    link.add_argument("--plan-file-id", required=True)
    link.add_argument("--version", required=True)
    link.add_argument("--revision", required=True)
    link.add_argument("--plan-ref", required=True)

    move = subparsers.add_parser("move")
    move.add_argument("--binding-id", required=True)
    move.add_argument("--policy", required=True)
    move.add_argument("--resolved-plan-id", required=True)
    move.add_argument("--plan-file-id", required=True)
    move.add_argument("--from-plan-ref", required=True)
    move.add_argument("--to-plan-ref", required=True)

    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        if args.command == "create":
            result = create_binding(args.policy, root=args.root)
        elif args.command == "link":
            result = link_plan(
                policy_ref=args.policy,
                binding_id=args.binding_id,
                resolved_plan_id=args.resolved_plan_id,
                plan_file_id=args.plan_file_id,
                version=args.version,
                revision=args.revision,
                plan_ref=args.plan_ref,
                root=args.root,
            )
        else:
            result = move_plan_ref(
                binding_id=args.binding_id,
                policy_ref=args.policy,
                resolved_plan_id=args.resolved_plan_id,
                plan_file_id=args.plan_file_id,
                from_plan_ref=args.from_plan_ref,
                to_plan_ref=args.to_plan_ref,
                root=args.root,
            )
    except PolicyPlanBindingError as exc:
        print(json.dumps(
            {"status": "UNRESOLVED", "code": exc.code, "message": str(exc)},
            sort_keys=True,
        ))
        return 2

    print(json.dumps(result.to_payload(), sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
