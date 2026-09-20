from __future__ import annotations

from dataclasses import dataclass
from hashlib import sha256
from pathlib import Path
from typing import Any

import yaml

from developer.automation.policy_loader import repository_root

from .errors import PolicyPlanBindingError


BINDING_REGISTRY_PATH = "developer/bindings/policy-plan-bindings.yaml"


@dataclass(frozen=True)
class BindingRegistrySnapshot:
    """Exact repository snapshot of the binding registry without semantic interpretation."""

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


def load_registry(
    root: str | Path | None = None,
    *,
    required: bool = True,
) -> BindingRegistrySnapshot | None:
    """Load the canonical registry as an opaque mapping.

    Field-level binding semantics are intentionally not interpreted here. They
    remain undefined until the governing policy and schema are approved.
    """

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

    return BindingRegistrySnapshot(
        path=BINDING_REGISTRY_PATH,
        payload=payload,
        digest=sha256(text.encode("utf-8")).hexdigest(),
    )
