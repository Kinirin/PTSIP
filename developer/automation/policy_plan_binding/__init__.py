"""Deterministic automation for Policy ↔ Planning binding maintenance.

The package owns mechanical registry access, exact resolution, integrity checks,
and canonical ordering. Policy meaning, approval semantics, lifecycle, and
transition rules remain outside this automation boundary until separately
approved.
"""

from .errors import PolicyPlanBindingError
from .registry import (
    BINDING_REGISTRY_PATH,
    BINDING_SCHEMA_PATH,
    BindingRegistrySnapshot,
    load_registry,
    validate_registry_schema,
)
from .reconciler import (
    PolicyPlanBindingReconciliation,
    reconcile_registry,
    validate_registry_integrity,
)
from .resolver import PolicyPlanBindingResolution, resolve_bindings
from .store import replace_registry

__all__ = [
    "BINDING_REGISTRY_PATH",
    "BINDING_SCHEMA_PATH",
    "BindingRegistrySnapshot",
    "PolicyPlanBindingError",
    "PolicyPlanBindingReconciliation",
    "PolicyPlanBindingResolution",
    "load_registry",
    "reconcile_registry",
    "replace_registry",
    "resolve_bindings",
    "validate_registry_integrity",
    "validate_registry_schema",
]
