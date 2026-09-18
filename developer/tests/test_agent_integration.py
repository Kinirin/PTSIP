from __future__ import annotations

from pathlib import Path

from developer.automation.agent_integration import install_mcp, status
from developer.automation.agent_instruction_progressive import migrate_level1


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
        "Before editing, read README.md.\n\n"
        "Canonical roles:\n\n"
        "ALPHA BETA GAMMA\n",
        encoding="utf-8",
    )
    migrate_level1(tmp_path)
    return tmp_path


def test_status_reports_local_cli_only_when_mcp_is_absent(
    tmp_path: Path,
) -> None:
    result = status(_repo(tmp_path))

    assert result["mode"] == "LOCAL_CLI_ONLY"
    assert result["local_cli"]["state"] == "READY"
    assert result["mcp"]["state"] == "ABSENT"
    assert result["mcp"]["implementation_state"] == "NOT_AVAILABLE"
    assert result["mcp"]["user_approval_required_before_install"] is True
    assert result["mcp"]["automatic_install_without_user_approval"] is False
    assert result["mcp"]["offer_to_user_now"] is False
    assert 'mcp="ABSENT"' in result["entry_directive"]
    assert result["mcp"]["intended_install_command"].endswith(
        "install-mcp --user-approved"
    )


def test_unavailable_mcp_install_preserves_local_cli_mode(
    tmp_path: Path,
) -> None:
    repo = _repo(tmp_path)
    before = (repo / ".agent/index.yaml").read_text(encoding="utf-8")

    assert install_mcp(repo) == 3
    assert (repo / ".agent/index.yaml").read_text(encoding="utf-8") == before
    assert status(repo)["mode"] == "LOCAL_CLI_ONLY"
