from __future__ import annotations

import os
import tempfile
from pathlib import Path
from typing import Any, Callable, Iterable, Mapping

import yaml

from .errors import PolicyPlanBindingError
from .registry import BindingRegistrySnapshot, load_registry, registry_path


RegistryValidator = Callable[[Mapping[str, Any]], Iterable[str]]


def replace_registry(
    payload: Mapping[str, Any],
    *,
    validator: RegistryValidator,
    expected_digest: str | None,
    root: str | Path | None = None,
) -> BindingRegistrySnapshot:
    """Atomically replace the binding registry after caller-supplied validation.

    The storage layer deliberately owns no Policy ↔ Planning semantics. A
    policy-approved validator must be supplied by the caller. The expected
    digest provides stale-state protection and prevents silent overwrites.
    """

    failures = tuple(str(item) for item in validator(payload))
    if failures:
        raise PolicyPlanBindingError(
            "BINDING_REGISTRY_VALIDATION_FAILED",
            "; ".join(failures),
        )

    current = load_registry(root, required=False)
    current_digest = None if current is None else current.digest
    if current_digest != expected_digest:
        raise PolicyPlanBindingError(
            "BINDING_REGISTRY_STALE",
            "binding registry changed since the caller's expected snapshot.",
        )

    path = registry_path(root)
    path.parent.mkdir(parents=True, exist_ok=True)

    rendered = yaml.safe_dump(
        dict(payload),
        sort_keys=False,
        allow_unicode=True,
    )

    fd, temporary_name = tempfile.mkstemp(
        prefix=".policy-plan-bindings.",
        suffix=".tmp",
        dir=path.parent,
        text=True,
    )
    temporary = Path(temporary_name)
    try:
        with os.fdopen(fd, "w", encoding="utf-8", newline="\n") as handle:
            handle.write(rendered)
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(temporary, path)
    finally:
        if temporary.exists():
            temporary.unlink()

    snapshot = load_registry(root, required=True)
    if snapshot is None:  # pragma: no cover - required=True is fail-closed.
        raise PolicyPlanBindingError(
            "BINDING_REGISTRY_WRITE_LOST",
            "binding registry disappeared after atomic replacement.",
        )
    return snapshot
