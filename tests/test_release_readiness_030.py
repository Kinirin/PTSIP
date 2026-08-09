from __future__ import annotations

import tomllib
from pathlib import Path

from ptsip.constants import SPEC_REVISION, TOOL_VERSION


ROOT = Path(__file__).resolve().parents[1]


def test_tool_030_package_and_runtime_versions_match() -> None:
    payload = tomllib.loads((ROOT / "pyproject.toml").read_text(encoding="utf-8"))
    assert payload["project"]["version"] == "0.3.0"
    assert TOOL_VERSION == "0.3.0"
    assert SPEC_REVISION == "a877b2f66a7f94c1b844c979e1b08fb08a9a8e45"


def test_release_workflow_derives_tool_tag_from_package_version() -> None:
    workflow = (ROOT / ".github" / "workflows" / "tooling-release.yml").read_text(encoding="utf-8")
    assert "EXPECTED_TAG=\"tool-v${PACKAGE_VERSION}\"" in workflow
    assert "python -m build" in workflow
    assert "python -m twine check dist/*" in workflow
    assert "pypa/gh-action-pypi-publish@release/v1" in workflow


def test_routine_ci_keeps_conform_cli_smoke() -> None:
    workflow = (ROOT / ".github" / "workflows" / "tooling-test.yml").read_text(encoding="utf-8")
    assert 'python-version: "3.14"' in workflow
    assert "python -m pytest -q" in workflow
    assert "ptsip --version" in workflow
    assert "ptsip spec" in workflow
    assert "ptsip conform --help" in workflow


def test_release_package_contains_bound_machine_readable_contracts() -> None:
    specdata = ROOT / "src" / "ptsip" / "specdata"
    assert (specdata / "ptsip-profile.schema.json").is_file()
    assert (specdata / "ptsip-registry.yaml").is_file()
    assert (specdata / "ptsip-artifact-evidence.schema.json").is_file()
    assert (specdata / "ptsip-agent-classification.schema.json").is_file()
    assert (specdata / "ptsip-diagnostic.schema.json").is_file()


def test_documentation_exposes_conformance_without_claiming_publication() -> None:
    readme = (ROOT / "README.md").read_text(encoding="utf-8")
    status = (ROOT / "STATUS.md").read_text(encoding="utf-8")
    changelog = (ROOT / "TOOLING-CHANGELOG.md").read_text(encoding="utf-8")
    assert "ptsip conform ." in readme
    assert "--agent-decision" in readme
    assert "--external-evidence" in readme
    assert "not yet published/tagged" in status
    assert "Release candidate source, not yet published" in changelog
