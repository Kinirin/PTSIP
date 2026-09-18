from __future__ import annotations

from pathlib import Path

from developer.automation.agent_instruction_activation import (
    activate,
    check_activation,
)
from developer.automation.agent_instruction_progressive import ENTRY_DIRECTIVE


ROOT = Path(__file__).resolve().parents[2]
POLICY = ROOT / "developer" / "policy" / "MPD-0010.yaml"


def _repo(tmp_path: Path) -> Path:
    target = tmp_path / "developer" / "policy"
    target.mkdir(parents=True)
    (target / "MPD-0010.yaml").write_text(
        POLICY.read_text(encoding="utf-8"),
        encoding="utf-8",
    )
    (tmp_path / "AGENTS.md").write_text(
        "# AGENTS\n\n"
        "Before broadly reading repository policy or planning prose, "
        "resolve the developer-policy context mechanically:\n\n"
        "Canonical roles:\n\n"
        "ALPHA BETA GAMMA\n",
        encoding="utf-8",
    )
    return tmp_path


def test_activation_creates_compact_agents_surface(tmp_path: Path) -> None:
    repo = _repo(tmp_path)

    result = activate(repo)

    assert result["integration"] == "LOCAL_CLI_ONLY"
    agents = (repo / "AGENTS.md").read_text(encoding="utf-8")
    assert ENTRY_DIRECTIVE in agents
    assert "PTSIP_AGENT_ROUTE" not in agents
    assert "ALPHA BETA GAMMA" in agents
    assert check_activation(repo) == ()
