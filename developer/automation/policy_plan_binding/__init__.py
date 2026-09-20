"""Deterministic automation for Policy ↔ Planning binding maintenance."""

from .errors import PolicyPlanBindingError
from .manager import (
    PolicyPlanBindingMutation,
    create_binding,
    link_plan,
    move_plan_ref,
)
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
from .resolver import (
    PolicyPlanBindingResolution,
    binding_entries,
    resolve_bindings,
)
from .store import replace_registry

__all__ = [
    "BINDING_REGISTRY_PATH",
    "BINDING_SCHEMA_PATH",
    "BindingRegistrySnapshot",
    "PolicyPlanBindingError",
    "PolicyPlanBindingMutation",
    "PolicyPlanBindingReconciliation",
    "PolicyPlanBindingResolution",
    "binding_entries",
    "create_binding",
    "link_plan",
    "load_registry",
    "move_plan_ref",
    "reconcile_registry",
    "replace_registry",
    "resolve_bindings",
    "validate_registry_integrity",
    "validate_registry_schema",
]
