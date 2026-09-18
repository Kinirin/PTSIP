from __future__ import annotations

import hashlib
import json
from pathlib import Path

import yaml

from developer.automation.agent_instruction_entry_resolver import resolve_entry
from developer.automation.agent_instruction_progressive import (
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


def test_clean_repository_bootstrap_replaces_pass_prose_with_direct_routes(
    tmp_path: Path,
) -> None:
    repo = _repo(tmp_path)
    original = (repo / "AGENTS.md").read_text(encoding="utf-8")

    result = migrate_level1(repo)

    assert result["mode"] == "ROUTED_FROM_AGENTS"
    assert result["pass_count"] > 0
    assert result["unresolved_count"] > 0
    assert (repo / ".agent/index.yaml").is_file()
    assert (repo / ".agent/stages/level1.json").is_file()
    assert (repo / ".agent/unresolved/level1.json").is_file()
    assert not (repo / ".agent/registry.yaml").exists()
    assert not (repo / ".agent/provenance/AGENTS.pre-level1.md").exists()

    routed = (repo / "AGENTS.md").read_text(encoding="utf-8")
    assert routed != original
    assert ENTRY_DIRECTIVE in routed
    assert "These instructions apply to coding agents working anywhere in this repository." not in routed
    assert "PTSIP_AGENT_ROUTE level=1 atom_id=A0001" in routed
    assert "ALPHA BETA GAMMA" in routed
    assert check(repo) == ()


def test_stage_supports_exact_atom_pointer_without_semantic_matching(
    tmp_path: Path,
) -> None:
    repo = _repo(tmp_path)
    migrate_level1(repo)

    stage = json.loads(
        (repo / ".agent/stages/level1.json").read_text(encoding="utf-8")
    )
    assert stage["pass_count"] == len(stage["pass_order"])
    assert stage["pass_count"] == len(stage["pass_by_atom"])

    for atom_id in stage["pass_order"]:
        item = stage["pass_by_atom"][atom_id]
        assert item["atom_id"] == atom_id
        assert item["pass_header"]["status"] == "PASS"

    routed = (repo / "AGENTS.md").read_text(encoding="utf-8")
    first = stage["pass_order"][0]
    assert (
        f'ref=".agent/stages/level1.json#/pass_by_atom/{first}"'
        in routed
    )


def test_bootstrap_index_binds_routed_agents_and_exact_route_contract(
    tmp_path: Path,
) -> None:
    repo = _repo(tmp_path)
    migrate_level1(repo)

    index = yaml.safe_load(
        (repo / ".agent/index.yaml").read_text(encoding="utf-8")
    )
    progressive = index["progressive_reasoning"]
    assert progressive["source_state"] == "ROUTED_LEVEL_1"
    assert progressive["previous_level_rerun_forbidden"] is True
    assert progressive["unresolved_blocks_next_level_candidates"] is False
    assert index["source"]["path"] == "AGENTS.md"
    routed = (repo / "AGENTS.md").read_text(encoding="utf-8")
    assert index["source"]["sha256"] == hashlib.sha256(
        routed.encode("utf-8")
    ).hexdigest()
    contract = index["route_contract"]
    assert contract["syntax"] == "PTSIP_AGENT_ROUTE_V1"
    assert contract["exact_atom_pointer_required"] is True
    assert contract["pointer_template"] == "#/pass_by_atom/{atom_id}"


def test_entry_resolver_consumes_direct_atom_stage(
    tmp_path: Path,
) -> None:
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


def test_source_change_marks_routed_bootstrap_stale(tmp_path: Path) -> None:
    repo = _repo(tmp_path)
    migrate_level1(repo)

    with (repo / "AGENTS.md").open("a", encoding="utf-8") as handle:
        handle.write("\nNew instruction.\n")

    assert "ROUTED_AGENTS_STALE" in check(repo)


def test_source_preserved_bootstrap_can_upgrade_to_direct_routes(
    tmp_path: Path,
) -> None:
    repo = _repo(tmp_path)
    source = (repo / "AGENTS.md").read_text(encoding="utf-8")
    agent = repo / ".agent"
    agent.mkdir()
    (agent / "index.yaml").write_text(
        yaml.safe_dump(
            {
                "schema_version": "ptsip-agent-progressive-index/v1",
                "management_mode": "PROGRESSIVE_LEVEL_1",
                "progressive_reasoning": {
                    "highest_materialized_level": 1,
                    "per_atom_advancement": True,
                    "source_state": "SOURCE_PRESERVED",
                },
                "source": {
                    "path": "AGENTS.md",
                    "sha256": hashlib.sha256(
                        source.encode("utf-8")
                    ).hexdigest(),
                },
            },
            sort_keys=False,
        ),
        encoding="utf-8",
    )

    result = migrate_level1(repo)

    assert result["mode"] == "ROUTED_FROM_AGENTS"
    routed = (repo / "AGENTS.md").read_text(encoding="utf-8")
    assert ENTRY_DIRECTIVE in routed
    assert "PTSIP_AGENT_ROUTE level=1 atom_id=A0001" in routed
    assert check(repo) == ()
