from __future__ import annotations

import argparse
import json
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any, Mapping, Sequence

import yaml

from developer.automation.policy_loader import load_yaml, repository_root
from developer.automation.policy_plan_binding.errors import PolicyPlanBindingError
from developer.automation.policy_plan_binding.ref_tracker import track_plan_ref
from developer.automation.policy_plan_binding.registry import load_registry
from developer.automation.policy_plan_binding.resolver import (
    binding_entries,
    resolve_bindings,
)


POLICY_INDEX_PATH = "developer/policy/index.yaml"


@dataclass(frozen=True)
class PolicyPlanConsistencyFailure:
    code: str
    message: str
    binding_id: str | None = None
    ref: str | None = None

    def to_payload(self) -> dict[str, Any]:
        return asdict(self)


@dataclass(frozen=True)
class PolicyPlanConsistencyReport:
    status: str
    binding_count: int
    checked_binding_count: int
    failures: tuple[PolicyPlanConsistencyFailure, ...]

    def to_payload(self) -> dict[str, Any]:
        return {
            "status": self.status,
            "binding_count": self.binding_count,
            "checked_binding_count": self.checked_binding_count,
            "failures": [failure.to_payload() for failure in self.failures],
        }


def _known_policy_ids(base: Path) -> set[str]:
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


def _read_plan_identity(
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

    try:
        payload = yaml.safe_load(candidate.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, yaml.YAMLError) as exc:
        raise PolicyPlanBindingError(
            "PLAN_DOCUMENT_INVALID",
            f"unable to read Plan document: {plan_ref}",
        ) from exc

    if not isinstance(payload, Mapping):
        raise PolicyPlanBindingError(
            "PLAN_DOCUMENT_INVALID",
            f"plan document must be a mapping: {plan_ref}",
        )

    raw = payload.get("plan_identity")
    if not isinstance(raw, Mapping):
        raise PolicyPlanBindingError(
            "PLAN_IDENTITY_MISSING",
            f"plan document has no plan_identity mapping: {plan_ref}",
        )

    identity: dict[str, str] = {}
    for field in ("resolved_plan_id", "plan_file_id", "version", "revision"):
        value = raw.get(field)
        if not isinstance(value, str) or not value:
            raise PolicyPlanBindingError(
                "PLAN_IDENTITY_INCOMPLETE",
                f"plan_identity.{field} is required: {plan_ref}",
            )
        identity[field] = value
    return identity


def _failure(
    failures: list[PolicyPlanConsistencyFailure],
    code: str,
    message: str,
    *,
    binding_id: str | None = None,
    ref: str | None = None,
) -> None:
    failures.append(
        PolicyPlanConsistencyFailure(
            code=code,
            message=message,
            binding_id=binding_id,
            ref=ref,
        )
    )


def verify_policy_plan_consistency(
    *,
    root: str | Path | None = None,
) -> PolicyPlanConsistencyReport:
    """Verify Policy ↔ Plan consistency without mutating repository state."""

    base = repository_root(root)
    failures: list[PolicyPlanConsistencyFailure] = []

    try:
        snapshot = load_registry(base, required=True, validate_schema=True)
    except PolicyPlanBindingError as exc:
        _failure(failures, exc.code, str(exc))
        return PolicyPlanConsistencyReport(
            status="FAIL",
            binding_count=0,
            checked_binding_count=0,
            failures=tuple(failures),
        )

    if snapshot is None:  # pragma: no cover
        _failure(failures, "BINDING_REGISTRY_MISSING", "binding registry is required.")
        return PolicyPlanConsistencyReport(
            status="FAIL",
            binding_count=0,
            checked_binding_count=0,
            failures=tuple(failures),
        )

    try:
        bindings = binding_entries(snapshot.payload)
        known_policies = _known_policy_ids(base)
    except PolicyPlanBindingError as exc:
        _failure(failures, exc.code, str(exc))
        return PolicyPlanConsistencyReport(
            status="FAIL",
            binding_count=len(snapshot.payload.get("bindings", [])),
            checked_binding_count=0,
            failures=tuple(failures),
        )

    seen_binding_ids: set[str] = set()
    seen_relations: set[tuple[str, str]] = set()
    plan_file_owners: dict[str, str] = {}
    checked = 0

    for binding in bindings:
        checked += 1
        binding_id = binding.get("binding_id")
        policy_ref = binding.get("policy_ref")
        planning_state = binding.get("planning_state")
        binding_id_text = binding_id if isinstance(binding_id, str) else None

        if not isinstance(binding_id, str):
            _failure(
                failures,
                "BINDING_ID_INVALID",
                "binding_id must be a string.",
            )
            continue
        if binding_id in seen_binding_ids:
            _failure(
                failures,
                "DUPLICATE_BINDING_ID",
                f"duplicate binding_id {binding_id}",
                binding_id=binding_id,
            )
        seen_binding_ids.add(binding_id)

        if not isinstance(policy_ref, str) or policy_ref not in known_policies:
            _failure(
                failures,
                "UNKNOWN_POLICY_REF",
                f"{binding_id}: policy_ref does not resolve through developer policy index.",
                binding_id=binding_id,
            )
            continue

        try:
            exact_binding = resolve_bindings(
                binding_id=binding_id,
                policy_ref=policy_ref,
                root=base,
            )
        except PolicyPlanBindingError as exc:
            _failure(
                failures,
                exc.code,
                str(exc),
                binding_id=binding_id,
            )
            continue

        if len(exact_binding.bindings) != 1:
            _failure(
                failures,
                "EXACT_BINDING_RESOLUTION_FAILED",
                f"{binding_id}: exact binding_id + policy_ref did not resolve exactly once.",
                binding_id=binding_id,
            )
            continue

        if planning_state == "NOT_CREATED":
            continue
        if planning_state != "CREATED":
            _failure(
                failures,
                "PLANNING_STATE_INVALID",
                f"{binding_id}: unsupported planning_state {planning_state!r}.",
                binding_id=binding_id,
            )
            continue

        resolved_plan_id = binding.get("resolved_plan_id")
        plan_file_id = binding.get("plan_file_id")
        version = binding.get("version")
        revision = binding.get("revision")
        plan_ref = binding.get("plan_ref")
        if not all(
            isinstance(value, str) and value
            for value in (
                resolved_plan_id,
                plan_file_id,
                version,
                revision,
                plan_ref,
            )
        ):
            _failure(
                failures,
                "CREATED_BINDING_IDENTITY_INCOMPLETE",
                f"{binding_id}: CREATED binding identity is incomplete.",
                binding_id=binding_id,
            )
            continue

        assert isinstance(resolved_plan_id, str)
        assert isinstance(plan_file_id, str)
        assert isinstance(version, str)
        assert isinstance(revision, str)
        assert isinstance(plan_ref, str)

        relation = (policy_ref, resolved_plan_id)
        if relation in seen_relations:
            _failure(
                failures,
                "DUPLICATE_CREATED_RELATION",
                f"{binding_id}: duplicate relation {policy_ref} -> {resolved_plan_id}.",
                binding_id=binding_id,
            )
        seen_relations.add(relation)

        prior_owner = plan_file_owners.get(plan_file_id)
        if prior_owner is not None and prior_owner != resolved_plan_id:
            _failure(
                failures,
                "PLAN_FILE_ID_CONFLICT",
                f"{binding_id}: plan_file_id {plan_file_id} is associated with "
                f"both {prior_owner} and {resolved_plan_id}.",
                binding_id=binding_id,
            )
        else:
            plan_file_owners[plan_file_id] = resolved_plan_id

        try:
            exact_plan_binding = resolve_bindings(
                binding_id=binding_id,
                policy_ref=policy_ref,
                resolved_plan_id=resolved_plan_id,
                plan_file_id=plan_file_id,
                plan_ref=plan_ref,
                root=base,
            )
        except PolicyPlanBindingError as exc:
            _failure(
                failures,
                exc.code,
                str(exc),
                binding_id=binding_id,
                ref=plan_ref,
            )
            continue

        if len(exact_plan_binding.bindings) != 1:
            _failure(
                failures,
                "EXACT_PLAN_BINDING_RESOLUTION_FAILED",
                f"{binding_id}: full v2 identity did not resolve exactly once.",
                binding_id=binding_id,
                ref=plan_ref,
            )
            continue

        try:
            tracking = track_plan_ref(binding_id, apply=False, root=base)
        except PolicyPlanBindingError as exc:
            _failure(
                failures,
                exc.code,
                str(exc),
                binding_id=binding_id,
                ref=plan_ref,
            )
            continue

        target_ref: str | None = None
        if tracking.status == "CURRENT":
            target_ref = tracking.discovered_plan_ref
        elif tracking.status == "RECONCILE_REQUIRED":
            target_ref = tracking.discovered_plan_ref
            _failure(
                failures,
                "PLAN_REF_RECONCILE_REQUIRED",
                f"{binding_id}: registered plan_ref {tracking.current_plan_ref} "
                f"must be reconciled to {tracking.discovered_plan_ref}.",
                binding_id=binding_id,
                ref=tracking.discovered_plan_ref,
            )
        elif tracking.status == "UNRESOLVED":
            _failure(
                failures,
                "PLAN_REF_UNRESOLVED",
                f"{binding_id}: no planning document resolves plan_file_id {plan_file_id}.",
                binding_id=binding_id,
                ref=plan_ref,
            )
        else:
            _failure(
                failures,
                "PLAN_REF_TRACKING_STATE_INVALID",
                f"{binding_id}: unexpected read-only tracking status {tracking.status}.",
                binding_id=binding_id,
                ref=plan_ref,
            )

        if target_ref is None:
            continue

        try:
            identity = _read_plan_identity(target_ref, root=base)
        except PolicyPlanBindingError as exc:
            _failure(
                failures,
                exc.code,
                str(exc),
                binding_id=binding_id,
                ref=target_ref,
            )
            continue

        expected = {
            "resolved_plan_id": resolved_plan_id,
            "plan_file_id": plan_file_id,
            "version": version,
            "revision": revision,
        }
        for field, expected_value in expected.items():
            actual_value = identity[field]
            if actual_value != expected_value:
                _failure(
                    failures,
                    {
                        "resolved_plan_id": "RESOLVED_PLAN_ID_MISMATCH",
                        "plan_file_id": "PLAN_FILE_ID_MISMATCH",
                        "version": "PLAN_VERSION_MISMATCH",
                        "revision": "PLAN_REVISION_MISMATCH",
                    }[field],
                    f"{binding_id}: {field} expected {expected_value}, got {actual_value}.",
                    binding_id=binding_id,
                    ref=target_ref,
                )

    return PolicyPlanConsistencyReport(
        status="PASS" if not failures else "FAIL",
        binding_count=len(bindings),
        checked_binding_count=checked,
        failures=tuple(failures),
    )


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Read-only Policy ↔ Plan consistency verification."
    )
    parser.add_argument("--root")
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    report = verify_policy_plan_consistency(root=args.root)
    print(json.dumps(report.to_payload(), sort_keys=True))
    return 0 if report.status == "PASS" else 1


if __name__ == "__main__":
    raise SystemExit(main())
