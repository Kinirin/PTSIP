from __future__ import annotations

import argparse
import json
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any, Mapping, Sequence

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
    plan_id: str,
    plan_ref: str,
    binding_id: str | None = None,
    root: str | Path | None = None,
) -> PolicyPlanBindingMutation:
    """Materialize a binding when the corresponding Plan is created."""

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

    if target.get("planning_state") == "CREATED":
        if target.get("plan_id") == plan_id and target.get("plan_ref") == plan_ref:
            return PolicyPlanBindingMutation(
                status="CURRENT",
                changed=False,
                binding=dict(target),
            )
        raise PolicyPlanBindingError(
            "BINDING_ALREADY_MATERIALIZED",
            f"{target['binding_id']} is already linked to a created plan.",
        )

    target["planning_state"] = "CREATED"
    target["plan_id"] = plan_id
    target["plan_ref"] = plan_ref

    payload["bindings"] = bindings
    _write(payload, expected_digest=snapshot.digest, root=root)
    return PolicyPlanBindingMutation(status="LINKED", changed=True, binding=dict(target))


def move_plan_ref(
    *,
    binding_id: str,
    plan_ref: str,
    root: str | Path | None = None,
) -> PolicyPlanBindingMutation:
    """Update only the physical path of an already-created Plan."""

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

    if target.get("plan_ref") == plan_ref:
        return PolicyPlanBindingMutation(
            status="CURRENT",
            changed=False,
            binding=dict(target),
        )

    target["plan_ref"] = plan_ref
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
    link.add_argument("--plan-id", required=True)
    link.add_argument("--plan-ref", required=True)

    move = subparsers.add_parser("move")
    move.add_argument("--binding-id", required=True)
    move.add_argument("--plan-ref", required=True)

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
                plan_id=args.plan_id,
                plan_ref=args.plan_ref,
                root=args.root,
            )
        else:
            result = move_plan_ref(
                binding_id=args.binding_id,
                plan_ref=args.plan_ref,
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
