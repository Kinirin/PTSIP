from __future__ import annotations

import json
from pathlib import Path

import yaml


REPO_ROOT = Path(__file__).resolve().parents[4]
MODE_ID = "policy-responsibility-analysis"
COMPONENT_ID = "policy-responsibility-analysis-verification"
TEST_TARGET = "src/tests/ptsip/policy_responsibility_analysis"


def _yaml(relative: str) -> dict:
    payload = yaml.safe_load((REPO_ROOT / relative).read_text(encoding="utf-8-sig"))
    assert isinstance(payload, dict)
    return payload


def test_pra_test_mode_has_exact_verification_owner() -> None:
    registry = _yaml(".github/test_modes.yaml")
    profile = _yaml(".ptsip/profiles/main.ptsip.yaml")

    modes = [item for item in registry["modes"] if item.get("id") == MODE_ID]
    assert len(modes) == 1
    mode = modes[0]
    assert mode["component_ref"] == COMPONENT_ID
    assert mode["execution"] == {"pytest": [TEST_TARGET]}

    components = [item for item in profile["components"] if item.get("id") == COMPONENT_ID]
    assert len(components) == 1
    component = components[0]
    assert "VERIFICATION" in component["roles"]
    assert "src/tests/ptsip/policy_responsibility_analysis/**" in component["include"]


def test_pra_verification_component_watches_canonical_sources() -> None:
    profile = _yaml(".ptsip/profiles/main.ptsip.yaml")
    component = next(item for item in profile["components"] if item.get("id") == COMPONENT_ID)
    analysis_inputs = set(component["analysis_inputs"])

    assert "developer/policy/analysis/**" in analysis_inputs
    assert "developer/policy/analysis/schemas/policy-responsibility-analysis*.json" in analysis_inputs
    assert "developer/policy/analysis/schemas/policy-materialization-analysis-registry*.json" in analysis_inputs
    assert "developer/policy/contracts/policy-responsibility-analysis-generation.v1.yaml" in analysis_inputs
    assert "developer/automation/internal/machine/policy_responsibility*.go" in analysis_inputs

    schema = json.loads(
        (REPO_ROOT / "developer/policy/analysis/schemas/policy-responsibility-analysis.schema.json").read_text(
            encoding="utf-8"
        )
    )
    assert schema["$id"] == "urn:developer-policy:policy-responsibility-analysis:v3"
