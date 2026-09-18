from pathlib import Path

import pytest

from developer.automation.agent_instruction_entry_resolver import AgentInstructionEntryResolverError, resolve_entry
from developer.automation.agent_instruction_materializer import materialize

ROOT = Path(__file__).resolve().parents[2]
POLICY = ROOT / "developer" / "policy" / "MPD-0010.yaml"


def _repo(tmp_path: Path) -> Path:
    policy = tmp_path / "developer" / "policy"
    policy.mkdir(parents=True)
    (policy / "MPD-0010.yaml").write_text(POLICY.read_text(encoding="utf-8"), encoding="utf-8")
    (tmp_path / "AGENTS.md").write_text(
        "# AGENTS\n\nBefore editing, read README.md.\n\nNever edit generated files directly.\n\n"
        "Validation evidence must include pytest status.\n\nArchitecture baseline: MVC plus EDA\n\n"
        "Canonical roles:\n\nALPHA BETA GAMMA\n",
        encoding="utf-8",
    )
    materialize(Path("AGENTS.md"), root=tmp_path)
    return tmp_path


def test_modify_is_level1_only_and_keeps_unresolved(tmp_path: Path) -> None:
    result = resolve_entry(_repo(tmp_path), operation="MODIFY")
    assert result["selected_level_1"] == ["APPLICABILITY", "RULE", "ACTION"]
    assert result["level_2_used"] is False
    assert result["unresolved_count"] >= 1


def test_other_is_explicit_widen_only(tmp_path: Path) -> None:
    repo = _repo(tmp_path)
    base = resolve_entry(repo, operation="MODIFY")
    widened = resolve_entry(repo, operation="MODIFY", include=["OTHER"])
    assert "OTHER" not in base["selected_level_1"]
    assert "OTHER" in widened["selected_level_1"]
    assert widened["instruction_count"] >= base["instruction_count"]


def test_verify_adds_evidence_without_level2(tmp_path: Path) -> None:
    result = resolve_entry(_repo(tmp_path), operation="VERIFY")
    assert result["selected_level_1"] == ["APPLICABILITY", "RULE", "ACTION", "EVIDENCE"]
    assert result["level_2_used"] is False


def test_unkown_operation_fails_closed(tmp_path: Path) -> None:
    with pytest.raises(AgentInstructionEntryResolverError):
        resolve_entry(_repo(tmp_path), operation="DEPLOY")
