"""Automation boundary for Policy ↔ Planning binding maintenance.

This package intentionally does not define binding semantics. Canonical binding
fields, lifecycle, and transition rules remain policy decisions and must be
materialized separately before resolver/reconciler behavior is added.
"""

from .errors import PolicyPlanBindingError
from .registry import BINDING_REGISTRY_PATH, BindingRegistrySnapshot, load_registry
from .store import replace_registry

__all__ = [
    "BINDING_REGISTRY_PATH",
    "BindingRegistrySnapshot",
    "PolicyPlanBindingError",
    "load_registry",
    "replace_registry",
]
