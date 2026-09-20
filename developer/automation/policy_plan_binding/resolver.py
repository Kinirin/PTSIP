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
    policy_ref: str | None
    plan_ref: str | None
    bindings: tuple[dict[str, str], ...]

    def to_payload(self) -> dict[str, Any]:
        payload = asdict(self)
        payload["bindings"] = list(self.bindings)
        return payload


def _bindings(payload: Mapping[str, Any]) -> tuple[dict[str, str], ...]:
    raw = payload.get("bindings")
    if not isinstance(raw, list):
        raise PolicyPlanBindingError(
            "BINDING_REGISTRY_INVALID_BINDINGS",
            "binding registry bindings must be a list.",
        )

    result: list[dict[str, str]] = []
    for item in raw:
        if not isinstance(item, Mapping):
            raise PolicyPlanBindingError(
                "BINDING_REGISTRY_INVALID_ENTRY",
                "binding registry entries must be mappings.",
            )
        policy_ref = item.get("policy_ref")
        plan_ref = item.get("plan_ref")
        if not isinstance(policy_ref, str) or not isinstance(plan_ref, str):
            raise PolicyPlanBindingError(
                "BINDING_REGISTRY_INVALID_ENTRY",
                "binding registry entries require string policy_ref and plan_ref.",
            )
        result.append({"policy_ref": policy_ref, "plan_ref": plan_ref})
    return tuple(result)


def resolve_bindings(
    *,
    policy_ref: str | None = None,
    plan_ref: str | None = None,
    root: str | Path | None = None,
) -> PolicyPlanBindingResolution:
    """Resolve Policy ↔ Planning relations by exact identity only."""

    if policy_ref is None and plan_ref is None:
        raise PolicyPlanBindingError(
            "BINDING_QUERY_EMPTY",
            "exact binding resolution requires policy_ref, plan_ref, or both.",
        )

    snapshot = load_registry(root, required=True, validate_schema=True)
    if snapshot is None:  # pragma: no cover - required=True fails closed.
        raise PolicyPlanBindingError(
            "BINDING_REGISTRY_MISSING",
            "binding registry is required for resolution.",
        )

    matches = tuple(
        binding
        for binding in _bindings(snapshot.payload)
        if (policy_ref is None or binding["policy_ref"] == policy_ref)
        and (plan_ref is None or binding["plan_ref"] == plan_ref)
    )

    return PolicyPlanBindingResolution(
        status="BOUND" if matches else "UNBOUND",
        policy_ref=policy_ref,
        plan_ref=plan_ref,
        bindings=matches,
    )


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Resolve exact Policy ↔ Planning bindings without semantic inference."
    )
    parser.add_argument("--policy", dest="policy_ref")
    parser.add_argument("--plan", dest="plan_ref")
    parser.add_argument("--root")
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        result = resolve_bindings(
            policy_ref=args.policy_ref,
            plan_ref=args.plan_ref,
            root=args.root,
        )
    except PolicyPlanBindingError as exc:
        print(
            json.dumps(
                {
                    "status": "UNRESOLVED",
                    "code": exc.code,
                    "message": str(exc),
                },
                sort_keys=True,
            )
        )
        return 2

    print(json.dumps(result.to_payload(), sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
