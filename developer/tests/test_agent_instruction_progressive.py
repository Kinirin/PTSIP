from __future__ import annotations

import hashlib
import json
from pathlib import Path

import yaml

from developer.automation.agent_instruction_entry_resolver import resolve_entry
from developer.automation.agent_instruction_progressive import (
    COMPACT_SOURCE_STATE,
    ENTRY_DIRECTIVE,
    check,
    migrate_level1,
)


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
        "These instructions apply to coding agents working anywhere in this repository.\n\n"
        "Before broadly reading repository policy or planning prose, "
        "resolve the developer-policy context mechanically:\n\n"
        "- Do not scan developer/policy/** merely to discover policy.\n\n"
        "Canonical roles:\n\n"
        "ALPHA BETA GAMMA\n",
        encoding="utf-8",
    )
    return tmp_path


def test_clean_repository_bootstrap_compacts_agents_to_one_machine_entry(
    tmp_path: Path,
) -> None:
    repo = _repo(tmp_path)

    result = migrate_level1(repo)

    assert result["mode"] == "COMPACTED_FROM_AGENTS"
    assert result["pass_count"] > 0
    assert result["unresolved_count"] > 0
    assert result["integration"] == "LOCAL_CLI_ONLY"

    agents = (repo / "AGENTS.md").read_text(encoding="utf-8")
    assert agents.count(ENTRY_DIRECTIVE) == 1
    assert "PTSIP_AGENT_ROUTE" not in agents
    assert (
        "These instructions apply to coding agents working anywhere in this repository."
        not in agents
    )
    assert "Canonical roles:" in agents

    assert (repo / ".agent/index.yaml").is_file()
    assert (repo / ".agent/stages/level1.json").is_file()
    assert (repo / ".agent/unresolved/level1.json").is_file()
    assert not (repo / ".agent/registry.yaml").exists()
    assert check(repo) == ()


def test_stage_keeps_exact_atom_lookup_outside_agents(tmp_path: Path) -> None:
    repo = _repo(tmp_path)
    migrate_level1(repo)

    stage = json.loads(
        (repo / ".agent/stages/level1.json").read_text(encoding="utf-8")
    )
    assert stage["pass_count"] == len(stage["pass_order"])
    assert stage["pass_count"] == len(stage["pass_by_atom"])
    for atom_id in stage["pass_order"]:
        assert stage["pass_by_atom"][atom_id]["atom_id"] == atom_id

    agents = (repo / "AGENTS.md").read_text(encoding="utf-8")
    assert not any(atom_id in agents for atom_id in stage["pass_order"])


def test_index_declares_local_cli_only_and_mcp_absent(tmp_path: Path) -> None:
    repo = _repo(tmp_path)
    migrate_level1(repo)

    index = yaml.safe_load(
        (repo / ".agent/index.yaml").read_text(encoding="utf-8")
    )
    progressive = index["progressive_reasoning"]
    assert progressive["source_state"] == COMPACT_SOURCE_STATE

    integration = index["integration"]
    assert integration["mode"] == "LOCAL_CLI_ONLY"
    assert integration["local_cli"]["state"] == "READY"
    assert integration["mcp"]["state"] == "ABSENT"
    assert integration["mcp"]["implementation_state"] == "NOT_AVAILABLE"
    assert integration["mcp"]["user_approval_required_before_install"] is True
    assert integration["mcp"]["offer_to_user_now"] is False

    agents = (repo / "AGENTS.md").read_text(encoding="utf-8")
    assert index["source"]["sha256"] == hashlib.sha256(
        agents.encode("utf-8")
    ).hexdigest()


def test_entry_resolver_consumes_stage_from_compact_agents(tmp_path: Path) -> None:
    repo = _repo(tmp_path)
    migrate_level1(repo)

    result = resolve_entry(repo, operation="MODIFY")

    assert result["stage"] == "PROGRESSIVE_LEVEL_1"
    assert result["previous_level_rerun"] is False
    assert result["instructions"]
    assert all(
        "machine" in item
        and "natural_residual" in item
        and item["stage_ref"].endswith(item["atom_id"])
        for item in result["instructions"]
    )
    assert result["unresolved"]
    assert all("natural_language" in item for item in result["unresolved"])


def test_compact_agents_change_marks_stage_stale(tmp_path: Path) -> None:
    repo = _repo(tmp_path)
    migrate_level1(repo)

    with (repo / "AGENTS.md").open("a", encoding="utf-8") as handle:
        handle.write("\nNew instruction.\n")

    assert "COMPACT_AGENTS_STALE" in check(repo)


def test_existing_routed_level1_compacts_without_reclassifying_stage(
    tmp_path: Path,
) -> None:
    repo = _repo(tmp_path)
    agent = repo / ".agent"
    (agent / "stages").mkdir(parents=True)
    (agent / "unresolved").mkdir(parents=True)

    stage = {
        "schema_version": "ptsip-agent-progressive-reasoning/v1",
        "level": 1,
        "pass_count": 1,
        "unresolved_count": 1,
        "pass_order": ["A0001"],
        "pass_by_atom": {
            "A0001": {
                "atom_id": "A0001",
                "pass_header": {
                    "level": 1,
                    "status": "PASS",
                    "labels": ["RULE"],
                },
                "machine": {"level_1_labels": ["RULE"]},
                "natural_residual": [],
            }
        },
    }
    unresolved = {
        "schema_version": "ptsip-agent-progressive-reasoning/v1",
        "level": 1,
        "routing_state": "UNRESOLVED",
        "count": 1,
        "items": [
            {
                "atom_id": "A0002",
                "status": "UNRESOLVED",
                "heading_path": ["AGENTS.md", "Context"],
                "natural_language": "Needs natural reasoning.",
            }
        ],
    }
    stage_text = json.dumps(stage, indent=2) + "\n"
    unresolved_text = json.dumps(unresolved, indent=2) + "\n"
    (agent / "stages/level1.json").write_text(stage_text, encoding="utf-8")
    (agent / "unresolved/level1.json").write_text(
        unresolved_text,
        encoding="utf-8",
    )
    (agent / "index.yaml").write_text(
        yaml.safe_dump(
            {
                "schema_version": "ptsip-agent-progressive-index/v1",
                "management_mode": "PROGRESSIVE_LEVEL_1",
                "progressive_reasoning": {
                    "highest_materialized_level": 1,
                    "per_atom_advancement": True,
                    "level_1_ref": "stages/level1.json",
                    "level_1_unresolved_ref": "unresolved/level1.json",
                    "current_next_level_candidate_count": 1,
                    "next_level_candidate_set_is_dynamic": True,
                    "unresolved_reassessment_source": "unresolved/level1.json",
                    "unresolved_blocks_next_level_candidates": False,
                    "source_state": "ROUTED_LEVEL_1",
                },
            },
            sort_keys=False,
        ),
        encoding="utf-8",
    )
    (repo / "AGENTS.md").write_text(
        "# AGENTS\n\n"
        'PTSIP_AGENT_ROUTE level=1 atom_id=A0001 '
        'ref=".agent/stages/level1.json#/pass_by_atom/A0001"\n\n'
        "Needs natural reasoning.\n",
        encoding="utf-8",
    )

    result = migrate_level1(repo)

    assert result["mode"] == "COMPACTED_EXISTING_LEVEL_1"
    assert (agent / "stages/level1.json").read_text(encoding="utf-8") == stage_text
    assert (
        (agent / "unresolved/level1.json").read_text(encoding="utf-8")
        == unresolved_text
    )
    agents = (repo / "AGENTS.md").read_text(encoding="utf-8")
    assert ENTRY_DIRECTIVE in agents
    assert "PTSIP_AGENT_ROUTE" not in agents
    assert "Needs natural reasoning." in agents
    assert check(repo) == ()
