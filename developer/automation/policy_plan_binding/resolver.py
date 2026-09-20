from __future__ import annotations

import argparse
import json
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any, Mapping, Sequence

from .errors import PolicyPlanBindingError
from .registry import load_registry


@dataclass(frozen=True)
class PolicyPlanBindingResolution:
    status: str
    binding_id: str | None
    policy_ref: str | None
    plan_id: str | None
    plan_ref: str | None
    bindings: tuple[dict[str, Any], ...]

    def to_payload(self) -> dict[str, Any]:
        payload = asdict(self)
        payload["bindings"] = list(self.bindings)
        return payload


def binding_entries(payload: Mapping[str, Any]) -> tuple[dict[str, Any], ...]:
    raw = payload.get("bindings")
    if not isinstance(raw, list):
        raise PolicyPlanBindingError(
            "BINDING_REGISTRY_INVALID_BINDINGS",
            "binding registry bindings must be a list.",
        )

    result: list[dict[str, Any]] = []
    for item in raw:
        if not isinstance(item, Mapping):
            raise PolicyPlanBindingError(
                "BINDING_REGISTRY_INVALID_ENTRY",
                "binding registry entries must be mappings.",
            )
        result.append(dict(item))
    return tuple(result)


def resolve_bindings(
    *,
    binding_id: str | None = None,
    policy_ref: str | None = None,
    plan_id: str | None = None,
    plan_ref: str | None = None,
    root: str | Path | None = None,
) -> PolicyPlanBindingResolution:
    """Resolve Policy ↔ Planning relations by exact identity only."""

    if all(value is None for value in (binding_id, policy_ref, plan_id, plan_ref)):
        raise PolicyPlanBindingError(
            "BINDING_QUERY_EMPTY",
            "exact binding resolution requires binding_id, policy_ref, plan_id, or plan_ref.",
        )

    snapshot = load_registry(root, required=True, validate_schema=True)
    if snapshot is None:  # pragma: no cover
        raise PolicyPlanBindingError(
            "BINDING_REGISTRY_MISSING",
            "binding registry is required for resolution.",
        )

    matches = tuple(
        binding
        for binding in binding_entries(snapshot.payload)
        if (binding_id is None or binding.get("binding_id") == binding_id)
        and (policy_ref is None or binding.get("policy_ref") == policy_ref)
        and (plan_id is None or binding.get("plan_id") == plan_id)
        and (plan_ref is None or binding.get("plan_ref") == plan_ref)
    )

    return PolicyPlanBindingResolution(
        status="BOUND" if matches else "UNBOUND",
        binding_id=binding_id,
        policy_ref=policy_ref,
        plan_id=plan_id,
        plan_ref=plan_ref,
        bindings=matches,
    )


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Resolve exact Policy ↔ Planning bindings without semantic inference."
    )
    parser.add_argument("--binding-id")
    parser.add_argument("--policy", dest="policy_ref")
    parser.add_argument("--plan-id")
    parser.add_argument("--plan", dest="plan_ref")
    parser.add_argument("--root")
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        result = resolve_bindings(
            binding_id=args.binding_id,
            policy_ref=args.policy_ref,
            plan_id=args.plan_id,
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
