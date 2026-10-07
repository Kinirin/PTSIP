"""Public Python surface for VPMS — Verification Protocol Management System.

The active product protocol exposes factory-validated Registry snapshots,
explicit selection and separately owned execution composition. PTSIP-specific
integration remains explicit under ``vpms.integration`` and is intentionally
not imported here, preserving the sibling subsystem dependency boundary.
"""

from __future__ import annotations

from .domain.model import (
    FormulaRef,
    PolicyRef,
    RunnerRef,
    TargetRef,
    VariablesRef,
    VerificationCase,
    VerificationOutcome,
    VerificationPurpose,
    VerificationResult,
)
from .domain.registry import (
    FormulaRegistry,
    RegisteredFormula,
    Registry,
    RegistryDiagnostic,
    RegistryDiagnosticCode,
    RegistryLoadResult,
    RegistryReferenceIndex,
    load_registry,
    register_formulas,
)
from .domain.snapshot import SnapshotLoadResult, ValidatedRegistrySnapshot, load_registry_snapshot
from .selection import SelectionDiagnostic, SelectionResult, resolve_selection
from .execution.adapters.command import CommandExecutor
from .execution.runner import CaseExecutor, RunnerExecution, run_case
from .execution.composition import run_cases


__all__ = (
    "CaseExecutor",
    "CommandExecutor",
    "FormulaRef",
    "FormulaRegistry",
    "PolicyRef",
    "RegisteredFormula",
    "Registry",
    "RegistryDiagnostic",
    "RegistryDiagnosticCode",
    "RegistryLoadResult",
    "RegistryReferenceIndex",
    "RunnerExecution",
    "RunnerRef",
    "SelectionDiagnostic",
    "SelectionResult",
    "SnapshotLoadResult",
    "TargetRef",
    "VariablesRef",
    "ValidatedRegistrySnapshot",
    "VerificationCase",
    "VerificationOutcome",
    "VerificationPurpose",
    "VerificationResult",
    "load_registry",
    "load_registry_snapshot",
    "register_formulas",
    "run_case",
    "resolve_selection",
    "run_cases",
)
