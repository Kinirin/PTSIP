from __future__ import annotations

import tomllib
from pathlib import Path

from ptsip.constants import SPEC_REVISION, SPEC_VERSION, TOOL_VERSION


ROOT = Path(__file__).resolve().parents[2]
EXPECTED_SPEC_REVISION = "12e2ccd15634ecb3d0a4195b0f61ac3f620e7540"


def test_tool_036_package_runtime_and_spec_binding_match() -> None:
    payload = tomllib.loads((ROOT / "pyproject.toml").read_text(encoding="utf-8"))
    assert payload["project"]["version"] == "0.3.6"
    assert TOOL_VERSION == "0.3.6"
    assert SPEC_VERSION == "0.3.6-draft"
    assert SPEC_REVISION == EXPECTED_SPEC_REVISION


def test_release_workflow_derives_tool_tag_from_package_version() -> None:
    workflow = (ROOT / ".github" / "workflows" / "tooling-release.yml").read_text(encoding="utf-8")
    assert "EXPECTED_TAG=\"tool-v${PACKAGE_VERSION}\"" in workflow
    assert "python -m build" in workflow
    assert "python -m twine check dist/*" in workflow
    assert "pypa/gh-action-pypi-publish@release/v1" in workflow


def test_routine_ci_verifies_test_build_and_installed_wheel_boundary() -> None:
    workflow = (ROOT / ".github" / "workflows" / "tooling-test.yml").read_text(encoding="utf-8")
    assert 'python-version: "3.14"' in workflow
    assert "python -m pytest -q" in workflow
    assert "python -m build" in workflow
    assert "python -m twine check dist/*" in workflow
    assert "--force-reinstall --no-deps dist/*.whl" in workflow
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


def test_release_contract_requires_full_036_normative_family() -> None:
    release_contract = (ROOT / ".github" / "scripts" / "verify_release_contract.py").read_text(encoding="utf-8")
    for path in (
        "spec/PTSIP-SPEC.md",
        "spec/PTSIP-CONFORMANCE.md",
        "spec/PTSIP-TERMINOLOGY.md",
        "spec/PTSIP-GOVERNANCE.md",
        "spec/PTSIP-RESPONSIBILITY-MAP.md",
    ):
        assert path in release_contract

    spec = (ROOT / "spec" / "PTSIP-SPEC.md").read_text(encoding="utf-8")
    map_spec = (ROOT / "spec" / "PTSIP-RESPONSIBILITY-MAP.md").read_text(encoding="utf-8")
    spec_note = (ROOT / "releasenote" / "spec-0.3.6-draft.md").read_text(encoding="utf-8")
    assert "0.3.6-draft" in spec
    assert "DEVELOPMENT_TOOLING" in spec
    assert "DELIVERY" in spec
    assert "OPERATIONS" in spec
    assert "PTSIP-RMAP-012" in map_spec
    assert EXPECTED_SPEC_REVISION in spec_note
