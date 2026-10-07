from __future__ import annotations

from pathlib import Path
import tomllib

import vpms
from vpms.domain.model import VerificationCase, VerificationPurpose
from vpms.domain.registry import load_registry
from vpms.domain.snapshot import load_registry_snapshot
from vpms.selection import resolve_selection
from vpms.execution.composition import run_cases
from vpms.execution.adapters.command import CommandExecutor


_REPOSITORY_ROOT = Path(__file__).resolve().parents[4]
_EXPECTED_PUBLIC_SURFACE = (
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


def test_package_root_exposes_only_proven_vpms_contracts() -> None:
    assert vpms.__all__ == _EXPECTED_PUBLIC_SURFACE
    assert vpms.VerificationPurpose is VerificationPurpose
    assert vpms.VerificationCase is VerificationCase
    assert vpms.load_registry is load_registry
    assert vpms.load_registry_snapshot is load_registry_snapshot
    assert vpms.resolve_selection is resolve_selection
    assert vpms.run_cases is run_cases
    assert all(not hasattr(vpms, name) for name in ("SelectionScope", "select_cases", "run_selected_cases"))
    assert vpms.CommandExecutor is CommandExecutor


def test_public_root_does_not_flatten_ptsip_integration() -> None:
    assert "load_ptsip_metadata" not in vpms.__all__
    assert "resolve_target_metadata" not in vpms.__all__


def test_tool_035_intentionally_adds_no_vpms_console_script() -> None:
    with (_REPOSITORY_ROOT / "pyproject.toml").open("rb") as handle:
        config = tomllib.load(handle)

    scripts = config["project"]["scripts"]
    assert not any(
        name == "vpms" or name.startswith("vpms-")
        for name in scripts
    )
