from __future__ import annotations

from pathlib import Path
import tomllib

import vpms
from vpms.domain.model import VerificationCase, VerificationPurpose
from vpms.domain.registry import load_registry
from vpms.domain.selector import SelectionScope, select_cases
from vpms.execution.adapters.command import CommandExecutor
from vpms.execution.runner import run_selected_cases


_REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
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


def test_package_root_exposes_only_proven_vpms_contracts() -> None:
    assert vpms.__all__ == _EXPECTED_PUBLIC_SURFACE
    assert vpms.VerificationPurpose is VerificationPurpose
    assert vpms.VerificationCase is VerificationCase
    assert vpms.load_registry is load_registry
    assert vpms.SelectionScope is SelectionScope
    assert vpms.select_cases is select_cases
    assert vpms.run_selected_cases is run_selected_cases
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
