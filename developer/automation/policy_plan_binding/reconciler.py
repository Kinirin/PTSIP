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
    """Validate only approved structural facts: schema, exact refs, and duplicates."""

    base = repository_root(root)
    failures = list(validate_registry_schema(payload, root=base))
    if failures:
        return tuple(failures)

    known_policies = _policy_ids(base)
    seen: set[tuple[str, str]] = set()

    bindings = payload.get("bindings", [])
    for index, raw in enumerate(bindings):
        if not isinstance(raw, Mapping):
            failures.append(f"bindings[{index}]: entry must be a mapping")
            continue

        policy_ref = raw.get("policy_ref")
        plan_ref = raw.get("plan_ref")
        if not isinstance(policy_ref, str) or not isinstance(plan_ref, str):
            failures.append(
                f"bindings[{index}]: policy_ref and plan_ref must be strings"
            )
            continue

        pair = (policy_ref, plan_ref)
        if pair in seen:
            failures.append(
                f"bindings[{index}]: duplicate exact binding {policy_ref} -> {plan_ref}"
            )
        seen.add(pair)

        if policy_ref not in known_policies:
            failures.append(
                f"bindings[{index}]: unknown developer policy {policy_ref}"
            )

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


def _canonical_payload(payload: Mapping[str, Any]) -> dict[str, Any]:
    bindings = payload.get("bindings", [])
    canonical_bindings = sorted(
        (dict(item) for item in bindings if isinstance(item, Mapping)),
        key=lambda item: (str(item.get("policy_ref")), str(item.get("plan_ref"))),
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
    """Deterministically validate and canonicalize the binding registry.

    No relationship is created, removed, or semantically reclassified here.
    Reconciliation is limited to fail-closed integrity checks and stable ordering.
    """

    snapshot = load_registry(root, required=True, validate_schema=True)
    if snapshot is None:  # pragma: no cover - required=True fails closed.
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
    parser.add_argument(
        "--apply",
        action="store_true",
        help="Apply canonical ordering after all fail-closed integrity checks pass.",
    )
    parser.add_argument("--root")
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        result = reconcile_registry(apply=args.apply, root=args.root)
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
    return 1 if result.failures else 0


if __name__ == "__main__":
    raise SystemExit(main())
