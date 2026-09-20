from __future__ import annotations

from dataclasses import dataclass
from hashlib import sha256
from pathlib import Path
from typing import Any, Mapping

import yaml
from jsonschema import Draft202012Validator

from developer.automation.policy_loader import load_json, repository_root

from .errors import PolicyPlanBindingError


BINDING_REGISTRY_PATH = "developer/bindings/policy-plan-bindings.yaml"
BINDING_SCHEMA_PATH = "developer/bindings/schemas/policy-plan-bindings.schema.json"


@dataclass(frozen=True)
class BindingRegistrySnapshot:
    """Exact repository snapshot of the Policy ↔ Planning binding registry."""

    path: str
    payload: dict[str, Any]
    digest: str


def registry_path(root: str | Path | None = None) -> Path:
    base = repository_root(root).resolve()
    candidate = (base / BINDING_REGISTRY_PATH).resolve()
    try:
        candidate.relative_to(base)
    except ValueError as exc:
        raise PolicyPlanBindingError(
            "BINDING_REGISTRY_PATH_ESCAPE",
            f"binding registry path escapes repository root: {BINDING_REGISTRY_PATH}",
        ) from exc
    return candidate


def validate_registry_schema(
    payload: Mapping[str, Any],
    *,
    root: str | Path | None = None,
) -> tuple[str, ...]:
    schema = load_json(BINDING_SCHEMA_PATH, root=root)
    Draft202012Validator.check_schema(schema)
    validator = Draft202012Validator(schema)
    errors = sorted(
        validator.iter_errors(dict(payload)),
        key=lambda error: tuple(str(part) for part in error.absolute_path),
    )
    return tuple(
        f"{'.'.join(str(part) for part in error.absolute_path) or '<root>'}: {error.message}"
        for error in errors
    )


def load_registry(
    root: str | Path | None = None,
    *,
    required: bool = True,
    validate_schema: bool = True,
) -> BindingRegistrySnapshot | None:
    """Load the canonical registry without inventing binding semantics."""

    path = registry_path(root)
    if not path.is_file():
        if required:
            raise PolicyPlanBindingError(
                "BINDING_REGISTRY_MISSING",
                f"binding registry does not exist: {BINDING_REGISTRY_PATH}",
            )
        return None

    text = path.read_text(encoding="utf-8")
    payload = yaml.safe_load(text)
    if not isinstance(payload, dict):
        raise PolicyPlanBindingError(
            "BINDING_REGISTRY_INVALID_ROOT",
            "binding registry root must be a YAML mapping.",
        )

    if validate_schema:
        failures = validate_registry_schema(payload, root=root)
        if failures:
            raise PolicyPlanBindingError(
                "BINDING_REGISTRY_SCHEMA_INVALID",
                "; ".join(failures),
            )

    return BindingRegistrySnapshot(
        path=BINDING_REGISTRY_PATH,
        payload=payload,
        digest=sha256(text.encode("utf-8")).hexdigest(),
    )
