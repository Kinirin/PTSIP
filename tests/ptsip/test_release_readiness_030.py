from __future__ import annotations

import runpy
import tomllib
from pathlib import Path

from ptsip.constants import SPEC_REVISION, SPEC_VERSION, TOOL_VERSION


ROOT = Path(__file__).resolve().parents[2]
EXPECTED_SPEC_REVISION = "b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e"


def test_tool_035_package_runtime_and_spec_binding_match() -> None:
    payload = tomllib.loads((ROOT / "pyproject.toml").read_text(encoding="utf-8"))
    assert payload["project"]["version"] == "0.3.5.post1"
    assert TOOL_VERSION == "0.3.5.post1"
    assert SPEC_VERSION == "0.3.4-draft"
    assert SPEC_REVISION == EXPECTED_SPEC_REVISION


def test_035_maintenance_keeps_034_spec_binding_but_036_requires_new_family() -> None:
    contract = runpy.run_path(str(ROOT / ".github" / "scripts" / "verify_release_contract.py"))
    expected = contract["_expected_spec_version_for_tool"]

    assert expected("0.3.5") == "0.3.4-draft"
    assert expected("0.3.5.post1") == "0.3.4-draft"
    assert expected("0.3.5.post2") == "0.3.4-draft"
    assert expected("0.3.6") == "0.3.6-draft"
    assert expected("0.4.0") == "0.4.0-draft"


def test_release_workflow_derives_tool_tag_from_package_version() -> None:
    workflow = (ROOT / ".github" / "workflows" / "tooling-release.yml").read_text(encoding="utf-8")
    assert 'runs-on: [self-hosted, Windows, X64]' in workflow
    assert 'DESKTOP-5HCCQIR' in workflow
    assert '$expectedTag = "tool-v$packageVersion"' in workflow
    assert "python -m build" in workflow
    assert "python -m twine check $distFiles" in workflow
    assert "pypa/gh-action-pypi-publish@release/v1" in workflow


def test_routine_ci_verifies_test_build_and_installed_wheel_boundary() -> None:
    workflow = (ROOT / ".github" / "workflows" / "tooling-test.yml").read_text(encoding="utf-8")
    assert 'runs-on: [self-hosted, Windows, X64]' in workflow
    assert 'DESKTOP-5HCCQIR' in workflow
    assert 'python-version: "3.14"' in workflow
    assert "python -m pytest -q" in workflow
    assert "python -m build" in workflow
    assert "python -m twine check $distFiles" in workflow
    assert 'python -m pip install --force-reinstall --no-deps "$($wheel.FullName)"' in workflow
    assert "ptsip --version" in workflow
    assert "ptsip spec" in workflow
    assert "ptsip conform --help" in workflow


def test_release_package_contains_bound_machine_readable_contracts() -> None:
    specdata = ROOT / "src" / "ptsip" / "specdata"
    for name in (
        "ptsip-profile.schema.json",
        "ptsip-registry.yaml",
        "ptsip-artifact-evidence.schema.json",
        "ptsip-agent-classification.schema.json",
        "ptsip-diagnostic.schema.json",
    ):
        assert (specdata / name).is_file()


def test_canonical_and_embedded_machine_readable_contracts_are_identical() -> None:
    pairs = (
        ("schemas/ptsip-profile.schema.json", "src/ptsip/specdata/ptsip-profile.schema.json"),
        ("registry/ptsip-registry.yaml", "src/ptsip/specdata/ptsip-registry.yaml"),
        ("schemas/ptsip-artifact-evidence.schema.json", "src/ptsip/specdata/ptsip-artifact-evidence.schema.json"),
        ("schemas/ptsip-agent-classification.schema.json", "src/ptsip/specdata/ptsip-agent-classification.schema.json"),
        ("schemas/ptsip-diagnostic.schema.json", "src/ptsip/specdata/ptsip-diagnostic.schema.json"),
    )
    for canonical, embedded in pairs:
        assert (ROOT / canonical).read_bytes() == (ROOT / embedded).read_bytes(), canonical


def test_documentation_records_034_authority_and_activation_contract() -> None:
    readme = (ROOT / "README.md").read_text(encoding="utf-8")
    spec = (ROOT / "spec" / "PTSIP-SPEC.md").read_text(encoding="utf-8")
    adr = (ROOT / "decisions" / "ADR-0005-activate-spec-0.3.4-draft.md").read_text(encoding="utf-8")
    release_note = (ROOT / "releasenote" / "spec-0.3.4-draft.md").read_text(encoding="utf-8")
    assert "0.3.4-draft" in readme
    assert "runtime_required" in readme
    assert "lifecycle_owner" in readme
    assert "AUTHORITY" in readme.upper()
    assert "PTSIP-ADP-001" in spec
    assert "PTSIP-AUT-001" in spec
    assert "PTSIP-AUT-007" in spec
    assert "first valid" in adr.lower()
    assert "Active draft family" in release_note