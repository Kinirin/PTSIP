from __future__ import annotations

import tomllib
from pathlib import Path

import ptsip
from ptsip.constants import SPEC_REVISION, SPEC_VERSION, TOOL_VERSION
from ptsip.spec_identity import current_spec_identity


_ROOT = Path(__file__).resolve().parents[2]
_EXPECTED_SPEC_REVISION = "b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e"


def test_tool_035_distribution_and_runtime_identity_match() -> None:
    with (_ROOT / "pyproject.toml").open("rb") as handle:
        project = tomllib.load(handle)["project"]

    assert project["version"] == "0.3.5.post1"
    assert TOOL_VERSION == "0.3.5.post1"
    assert ptsip.__version__ == TOOL_VERSION


def test_tool_035_preserves_specification_binding() -> None:
    identity = current_spec_identity()

    assert identity.tool_version == "0.3.5.post1"
    assert SPEC_VERSION == "0.3.4-draft"
    assert SPEC_REVISION == _EXPECTED_SPEC_REVISION
    assert identity.version == SPEC_VERSION
    assert identity.revision == SPEC_REVISION