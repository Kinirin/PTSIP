"""Public Python surface for VPMS — Verification Purpose Management System.

Tool 0.3.5 exposes the proven VPMS model, Registry, selection, runner, and
generic command-adapter contracts from this package root. PTSIP-specific
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
from .domain.selector import SelectionScope, select_cases
from .execution.adapters.command import CommandExecutor
from .execution.runner import CaseExecutor, RunnerExecution, run_case, run_selected_cases


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
    "SelectionScope",
    "TargetRef",
    "VariablesRef",
    "VerificationCase",
    "VerificationOutcome",
    "VerificationPurpose",
    "VerificationResult",
    "load_registry",
    "register_formulas",
    "run_case",
    "run_selected_cases",
    "select_cases",
)
