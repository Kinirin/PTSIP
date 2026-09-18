from pathlib import Path

import yaml

from developer.automation.agent_instruction_activation import MODE, activate, bootstrap_text, check_activation
from developer.automation.agent_instruction_entry_resolver import resolve_entry
from developer.automation.agent_instruction_materializer import materialize

ROOT = Path(__file__).resolve().parents[2]
POLICY = ROOT / "developer" / "policy" / "MPD-0010.yaml"


def _repo(tmp_path: Path) -> Path:
    policy = tmp_path / "developer" / "policy"
    policy.mkdir(parents=True)
    (policy / "MPD-0010.yaml").write_text(POLICY.read_text(encoding="utf-8"), encoding="utf-8")
    (tmp_path / "AGENTS.md").write_text(
        "# AGENTS\n\nBefore editing, read README.md.\n\nNever edit generated files directly.\n\n"
        "Architecture baseline: MVC plus EDA\n\nCanonical roles:\n\nALPHA BETA GAMMA\n",
        encoding="utf-8",
    )
    materialize(Path("AGENTS.md"), root=tmp_path)
    return tmp_path


def test_activation_promotes_registry_and_replaces_agents(tmp_path: Path) -> None:
    repo = _repo(tmp_path)
    original = (repo / "AGENTS.md").read_text(encoding="utf-8")
    result = activate(repo)
    assert result["managed"] > 0 and result["unresolved"] > 0
    assert (repo / "AGENTS.md").read_text(encoding="utf-8") == bootstrap_text()
    assert (repo / ".agent/provenance/AGENTS.pre-level1.md").read_text(encoding="utf-8") == original
    registry = yaml.safe_load((repo / ".agent/registry.yaml").read_text(encoding="utf-8"))
    assert registry["management_mode"] == MODE
    assert registry["authority_mode"] == "REGISTRY_ATOM_INSTRUCTION_TEXT"
    assert all("instruction_text" in atom for atom in registry["atoms"])
    assert any(atom["management_state"] == "UNRESOLVED" for atom in registry["atoms"])


def test_activation_check_and_entry_work(tmp_path: Path) -> None:
    repo = _repo(tmp_path)
    activate(repo)
    assert check_activation(repo) == ()
    result = resolve_entry(repo, operation="MODIFY")
    assert result["instruction_count"] > 0
    assert result["unresolved_count"] > 0
    assert result["level_2_used"] is False
