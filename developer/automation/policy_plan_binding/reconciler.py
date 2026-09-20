from __future__ import annotations

import argparse
import json
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any, Mapping, Sequence

from developer.automation.policy_loader import load_yaml, repository_root

from .errors import PolicyPlanBindingError
from .registry import load_registry, validate_registry_schema
from .store import replace_registry


POLICY_INDEX_PATH = "developer/policy/index.yaml"


@dataclass(frozen=True)
class PolicyPlanBindingReconciliation:
    status: str
    changed: bool
    applied: bool
    binding_count: int
    failures: tuple[str, ...]

    def to_payload(self) -> dict[str, Any]:
        payload = asdict(self)
        payload["failures"] = list(self.failures)
        return payload


def _policy_ids(base: Path) -> set[str]:
    payload = load_yaml(POLICY_INDEX_PATH, root=base)
    entries = payload.get("policies")
    if not isinstance(entries, list):
        raise PolicyPlanBindingError(
            "POLICY_INDEX_INVALID",
            "developer policy index must contain a policies list.",
        )
    return {
        str(entry["id"])
        for entry in entries
        if isinstance(entry, Mapping) and isinstance(entry.get("id"), str)
    }


def validate_registry_integrity(
    payload: Mapping[str, Any],
    *,
    root: str | Path | None = None,
) -> tuple[str, ...]:
    """Validate approved binding identity, state, exact refs, and M:N integrity."""

    base = repository_root(root)
    failures = list(validate_registry_schema(payload, root=base))
    if failures:
        return tuple(failures)

    known_policies = _policy_ids(base)
    seen_binding_ids: set[str] = set()
    seen_created_relations: set[tuple[str, str]] = set()

    bindings = payload.get("bindings", [])
    for index, raw in enumerate(bindings):
        if not isinstance(raw, Mapping):
            failures.append(f"bindings[{index}]: entry must be a mapping")
            continue

        binding_id = raw.get("binding_id")
        policy_ref = raw.get("policy_ref")
        planning_state = raw.get("planning_state")

        if isinstance(binding_id, str):
            if binding_id in seen_binding_ids:
                failures.append(
                    f"bindings[{index}]: duplicate binding_id {binding_id}"
                )
            seen_binding_ids.add(binding_id)

        if isinstance(policy_ref, str) and policy_ref not in known_policies:
            failures.append(
                f"bindings[{index}]: unknown developer policy {policy_ref}"
            )

        if planning_state != "CREATED":
            continue

        plan_id = raw.get("plan_id")
        plan_ref = raw.get("plan_ref")
        if not isinstance(policy_ref, str) or not isinstance(plan_id, str):
            continue
        if not isinstance(plan_ref, str):
            continue

        relation = (policy_ref, plan_id)
        if relation in seen_created_relations:
            failures.append(
                f"bindings[{index}]: duplicate created relation {policy_ref} -> {plan_id}"
            )
        seen_created_relations.add(relation)

        candidate = (base / plan_ref).resolve()
        try:
            candidate.relative_to(base.resolve())
        except ValueError:
            failures.append(
                f"bindings[{index}]: plan_ref escapes repository root: {plan_ref}"
            )
            continue
        if not candidate.is_file():
            failures.append(
                f"bindings[{index}]: plan_ref does not exist: {plan_ref}"
            )

    return tuple(failures)


def _binding_sort_key(item: Mapping[str, Any]) -> tuple[int, str]:
    binding_id = str(item.get("binding_id", ""))
    try:
        return int(binding_id.split("-", 1)[1]), binding_id
    except (IndexError, ValueError):
        return 2**31 - 1, binding_id


def _canonical_payload(payload: Mapping[str, Any]) -> dict[str, Any]:
    bindings = payload.get("bindings", [])
    canonical_bindings = sorted(
        (dict(item) for item in bindings if isinstance(item, Mapping)),
        key=_binding_sort_key,
    )
    return {
        "schema_version": payload["schema_version"],
        "registry_role": payload["registry_role"],
        "schema_ref": payload["schema_ref"],
        "bindings": canonical_bindings,
    }


def reconcile_registry(
    *,
    apply: bool = False,
    root: str | Path | None = None,
) -> PolicyPlanBindingReconciliation:
    snapshot = load_registry(root, required=True, validate_schema=True)
    if snapshot is None:  # pragma: no cover
        raise PolicyPlanBindingError(
            "BINDING_REGISTRY_MISSING",
            "binding registry is required for reconciliation.",
        )

    failures = validate_registry_integrity(snapshot.payload, root=root)
    if failures:
        return PolicyPlanBindingReconciliation(
            status="INVALID",
            changed=False,
            applied=False,
            binding_count=len(snapshot.payload.get("bindings", [])),
            failures=failures,
        )

    canonical = _canonical_payload(snapshot.payload)
    changed = canonical != snapshot.payload
    if changed and apply:
        replace_registry(
            canonical,
            validator=lambda value: validate_registry_integrity(value, root=root),
            expected_digest=snapshot.digest,
            root=root,
        )

    return PolicyPlanBindingReconciliation(
        status="RECONCILED" if changed and apply else ("WOULD_RECONCILE" if changed else "CURRENT"),
        changed=changed,
        applied=bool(changed and apply),
        binding_count=len(canonical["bindings"]),
        failures=(),
    )


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Validate and deterministically reconcile Policy ↔ Planning bindings."
    )
    parser.add_argument("--apply", action="store_true")
    parser.add_argument("--root")
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        result = reconcile_registry(apply=args.apply, root=args.root)
    except PolicyPlanBindingError as exc:
        print(json.dumps(
            {"status": "UNRESOLVED", "code": exc.code, "message": str(exc)},
            sort_keys=True,
        ))
        return 2

    print(json.dumps(result.to_payload(), sort_keys=True))
    return 1 if result.failures else 0


if __name__ == "__main__":
    raise SystemExit(main())
