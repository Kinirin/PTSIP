from __future__ import annotations

import argparse
import json
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any, Mapping, Sequence

import yaml

from developer.automation.policy_loader import repository_root

from .errors import PolicyPlanBindingError
from .manager import move_plan_ref
from .resolver import resolve_bindings


PLANNING_ROOT = "developer/planning"


@dataclass(frozen=True)
class PlanRefTrackingResult:
    status: str
    binding_id: str
    policy_ref: str
    resolved_plan_id: str
    plan_file_id: str
    current_plan_ref: str
    discovered_plan_ref: str | None
    candidates: tuple[str, ...]
    changed: bool
    applied: bool

    def to_payload(self) -> dict[str, Any]:
        payload = asdict(self)
        payload["candidates"] = list(self.candidates)
        return payload


def _explicit_plan_identity(path: Path) -> dict[str, str] | None:
    try:
        payload = yaml.safe_load(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, yaml.YAMLError):
        return None
    if not isinstance(payload, Mapping):
        return None

    raw = payload.get("plan_identity")
    if not isinstance(raw, Mapping):
        return None

    resolved_plan_id = raw.get("resolved_plan_id")
    plan_file_id = raw.get("plan_file_id")
    if (
        not isinstance(resolved_plan_id, str)
        or not resolved_plan_id
        or not isinstance(plan_file_id, str)
        or not plan_file_id
    ):
        return None
    return {
        "resolved_plan_id": resolved_plan_id,
        "plan_file_id": plan_file_id,
    }


def _planning_documents(base: Path) -> tuple[Path, ...]:
    root = (base / PLANNING_ROOT).resolve()
    if not root.exists():
        return ()
    candidates = {
        *root.rglob("*.yaml"),
        *root.rglob("*.yml"),
    }
    return tuple(sorted((path for path in candidates if path.is_file()), key=str))


def _find_plan_file_candidates(
    *,
    plan_file_id: str,
    root: str | Path | None = None,
) -> tuple[tuple[str, dict[str, str]], ...]:
    base = repository_root(root).resolve()
    matches: list[tuple[str, dict[str, str]]] = []
    for path in _planning_documents(base):
        identity = _explicit_plan_identity(path)
        if identity is None or identity["plan_file_id"] != plan_file_id:
            continue
        relative = path.resolve().relative_to(base).as_posix()
        matches.append((relative, identity))
    return tuple(matches)


def track_plan_ref(
    binding_id: str,
    *,
    apply: bool = False,
    root: str | Path | None = None,
) -> PlanRefTrackingResult:
    """Locate an exact Plan file identity and optionally reconcile plan_ref."""

    resolution = resolve_bindings(binding_id=binding_id, root=root)
    if len(resolution.bindings) != 1:
        if not resolution.bindings:
            raise PolicyPlanBindingError(
                "BINDING_NOT_FOUND",
                f"no exact binding exists for {binding_id}.",
            )
        raise PolicyPlanBindingError(
            "BINDING_ID_AMBIGUOUS",
            f"multiple bindings exist for {binding_id}.",
        )

    binding = resolution.bindings[0]
    if binding.get("planning_state") != "CREATED":
        raise PolicyPlanBindingError(
            "PLAN_NOT_CREATED",
            f"{binding_id} has no created plan to track.",
        )

    required = ("policy_ref", "resolved_plan_id", "plan_file_id", "plan_ref")
    values: dict[str, str] = {}
    for field in required:
        value = binding.get(field)
        if not isinstance(value, str) or not value:
            raise PolicyPlanBindingError(
                "BINDING_IDENTITY_INCOMPLETE",
                f"{binding_id}: {field} is required for plan_ref tracking.",
            )
        values[field] = value

    matches = _find_plan_file_candidates(
        plan_file_id=values["plan_file_id"],
        root=root,
    )
    candidate_refs = tuple(path for path, _ in matches)

    if not matches:
        return PlanRefTrackingResult(
            status="UNRESOLVED",
            binding_id=binding_id,
            policy_ref=values["policy_ref"],
            resolved_plan_id=values["resolved_plan_id"],
            plan_file_id=values["plan_file_id"],
            current_plan_ref=values["plan_ref"],
            discovered_plan_ref=None,
            candidates=(),
            changed=False,
            applied=False,
        )

    if len(matches) > 1:
        raise PolicyPlanBindingError(
            "PLAN_FILE_ID_AMBIGUOUS",
            f"{values['plan_file_id']} resolves to multiple planning documents: "
            + ", ".join(candidate_refs),
        )

    discovered_ref, identity = matches[0]
    if identity["resolved_plan_id"] != values["resolved_plan_id"]:
        raise PolicyPlanBindingError(
            "PLAN_FILE_ID_CONFLICT",
            f"{values['plan_file_id']} resolves to {identity['resolved_plan_id']}, "
            f"expected {values['resolved_plan_id']}.",
        )

    if discovered_ref == values["plan_ref"]:
        return PlanRefTrackingResult(
            status="CURRENT",
            binding_id=binding_id,
            policy_ref=values["policy_ref"],
            resolved_plan_id=values["resolved_plan_id"],
            plan_file_id=values["plan_file_id"],
            current_plan_ref=values["plan_ref"],
            discovered_plan_ref=discovered_ref,
            candidates=candidate_refs,
            changed=False,
            applied=False,
        )

    if not apply:
        return PlanRefTrackingResult(
            status="RECONCILE_REQUIRED",
            binding_id=binding_id,
            policy_ref=values["policy_ref"],
            resolved_plan_id=values["resolved_plan_id"],
            plan_file_id=values["plan_file_id"],
            current_plan_ref=values["plan_ref"],
            discovered_plan_ref=discovered_ref,
            candidates=candidate_refs,
            changed=False,
            applied=False,
        )

    mutation = move_plan_ref(
        binding_id=binding_id,
        policy_ref=values["policy_ref"],
        resolved_plan_id=values["resolved_plan_id"],
        plan_file_id=values["plan_file_id"],
        from_plan_ref=values["plan_ref"],
        to_plan_ref=discovered_ref,
        root=root,
    )
    return PlanRefTrackingResult(
        status="RECONCILED",
        binding_id=binding_id,
        policy_ref=values["policy_ref"],
        resolved_plan_id=values["resolved_plan_id"],
        plan_file_id=values["plan_file_id"],
        current_plan_ref=values["plan_ref"],
        discovered_plan_ref=discovered_ref,
        candidates=candidate_refs,
        changed=mutation.changed,
        applied=mutation.changed,
    )


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Track exact Plan file identity and reconcile stale plan_ref values."
    )
    parser.add_argument("binding_id")
    parser.add_argument("--apply", action="store_true")
    parser.add_argument("--root")
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        result = track_plan_ref(
            args.binding_id,
            apply=args.apply,
            root=args.root,
        )
    except PolicyPlanBindingError as exc:
        print(json.dumps(
            {"status": "FAIL_CLOSED", "code": exc.code, "message": str(exc)},
            sort_keys=True,
        ))
        return 2

    print(json.dumps(result.to_payload(), sort_keys=True))
    return 0 if result.status != "UNRESOLVED" else 1


if __name__ == "__main__":
    raise SystemExit(main())
