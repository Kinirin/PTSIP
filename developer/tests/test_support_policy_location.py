from __future__ import annotations

import json
from pathlib import Path

import pytest
import yaml

from developer.automation.vpms_api_implementation import RECORD, verify

ROOT = Path(__file__).resolve().parents[2]


def test_support_policy_namespace_has_no_retired_documentation_dependency():
    policy = yaml.safe_load((ROOT / "developer/policy/SPEC/MPD-SPEC-0001.yaml").read_text(encoding="utf-8"))
    namespace = policy["rules"]["namespace"]["support_feature"]
    assert namespace["machine_policy_path"] == "src/policy/"
    assert "human_documentation_path" not in namespace
    assert "docs/Support_policy" not in (ROOT / "AGENTS.md").read_text(encoding="utf-8")


@pytest.mark.parametrize("relative", ["developer/profiles/ptsip-repository.yaml", ".ptsip/profiles/main.ptsip.yaml"])
def test_current_profile_has_no_retired_support_policy_selectors(relative):
    profile = yaml.safe_load((ROOT / relative).read_text(encoding="utf-8"))
    for record in profile["components"] + profile["associated_artifacts"]:
        for field in ("include", "analysis_inputs"):
            assert all(not path.startswith("docs/Support_policy/") for path in record.get(field, []))


def test_unapproved_missing_implementation_target_still_fails(monkeypatch):
    record = json.loads((ROOT / RECORD).read_text(encoding="utf-8"))
    record.pop("subsequent_retirements")
    original = Path.read_text

    def read_text(path, *args, **kwargs):
        if path == ROOT / RECORD:
            return json.dumps(record)
        return original(path, *args, **kwargs)

    monkeypatch.setattr(Path, "read_text", read_text)
    with pytest.raises(ValueError, match="MISSING_IMPLEMENTATION_TARGET"):
        verify(ROOT)


def test_approved_retirement_rejects_a_reintroduced_file(monkeypatch):
    original = Path.exists
    retired = ROOT / "docs/Support_policy/automation/README.md"
    monkeypatch.setattr(Path, "exists", lambda path: True if path == retired else original(path))
    with pytest.raises(ValueError, match="RETIRED_IMPLEMENTATION_TARGET_PRESENT"):
        verify(ROOT)


def test_retirement_does_not_waive_a_missing_live_implementation(monkeypatch):
    original = Path.is_file
    live = ROOT / "src/vpms/domain/snapshot.py"
    monkeypatch.setattr(Path, "is_file", lambda path: False if path == live else original(path))
    with pytest.raises(ValueError, match="MISSING_IMPLEMENTATION_TARGET: src/vpms/domain/snapshot.py"):
        verify(ROOT)
