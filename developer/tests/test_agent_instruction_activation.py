from __future__ import annotations

import json
from pathlib import Path

import yaml

from developer.automation.agent_instruction_activation import (
    activate,
    bootstrap_text,
    check_activation,
)
from developer.automation.agent_instruction_entry_resolver import resolve_entry
from developer.automation.agent_instruction_materializer import materialize
from developer.automation.agent_instruction_progressive import check, migrate_level1

ROOT = Path(__file__).resolve().parents[2]
POLICY = ROOT / "developer" / "policy" / "MPD-0010.yaml"


def _repo(tmp_path: Path) -> Path:
    policy = tmp_path / "developer" / "policy"
    policy.mkdir(parents=True)
    (policy / "MPD-0010.yaml").write_text(
        POLICY.read_text(encoding="utf-8"),
        encoding="utf-8",
    )
    (tmp_path / "AGENTS.md").write_text(
        "# AGENTS\n\n"
        "Before broadly reading repository policy or planning prose, resolve the developer-policy context mechanically:\n\n"
        "- Do not scan developer/policy/** merely to discover policy.\n\n"
        "Canonical roles:\n\nALPHA BETA GAMMA\n",
        encoding="utf-8",
    )
    materialize(Path("AGENTS.md"), root=tmp_path)
    return tmp_path


def test_progressive_level1_replaces_pass_prose_with_machine_plus_residual(
    tmp_path: Path,
) -> None:
    repo = _repo(tmp_path)
    result = migrate_level1(repo)
    assert result["pass_count"] > 0
    assert result["unresolved_count"] > 0

    stage = json.loads(
        (repo / ".agent/stages/level1.json").read_text(encoding="utf-8")
    )
    special = next(
        item
        for item in stage["pass"]
        if "AUTOMATED_REASONING_DURING_REPOSITORY_DOCUMENT_RETRIEVAL"
        in item["machine"].get("operations", [])
    )
    assert special["pass_header"]["status"] == "PASS"
    assert special["natural_residual"] == []

    registry = yaml.safe_load(
        (repo / ".agent/registry.yaml").read_text(encoding="utf-8")
    )
    assert all(
        "normalized_text" not in item and "instruction_text" not in item
        for item in registry["atoms"]
    )
    assert all(
        "raw_excerpt" not in item["source"]
        for item in registry["atoms"]
    )


def test_level1_unresolved_stays_natural_language_without_blocking_passed_atoms(
    tmp_path: Path,
) -> None:
    repo = _repo(tmp_path)
    migrate_level1(repo)

    unresolved = json.loads(
        (repo / ".agent/unresolved/level1.json").read_text(encoding="utf-8")
    )
    assert unresolved["count"] > 0
    assert all(
        item["status"] == "UNRESOLVED" and item["natural_language"]
        for item in unresolved["items"]
    )

    index = yaml.safe_load(
        (repo / ".agent/index.yaml").read_text(encoding="utf-8")
    )
    progressive = index["progressive_reasoning"]
    assert progressive["highest_materialized_level"] == 1
    assert progressive["per_atom_advancement"] is True
    assert progressive["current_next_level_candidate_count"] > 0
    assert progressive["next_level_candidate_set_is_dynamic"] is True
    assert progressive["unresolved_reassessment_source"] == "unresolved/level1.json"
    assert progressive["unresolved_blocks_next_level_candidates"] is False
    assert progressive["previous_level_rerun_forbidden"] is True


def test_entry_resolver_reads_stage_not_original_pass_sentences(
    tmp_path: Path,
) -> None:
    repo = _repo(tmp_path)
    migrate_level1(repo)

    result = resolve_entry(repo, operation="MODIFY")
    assert result["stage"] == "PROGRESSIVE_LEVEL_1"
    assert result["previous_level_rerun"] is False
    assert result["instructions"]
    assert "text" not in result["instructions"][0]
    assert all(
        "machine" in item and "natural_residual" in item
        for item in result["instructions"]
    )
    assert result["unresolved"]
    assert all(
        "natural_language" in item
        for item in result["unresolved"]
    )


def test_activation_creates_progressive_surface_without_provenance_md(
    tmp_path: Path,
) -> None:
    repo = _repo(tmp_path)
    result = activate(repo)

    assert result["pass_count"] > 0
    assert (repo / "AGENTS.md").read_text(encoding="utf-8") == bootstrap_text()
    assert not (repo / ".agent/provenance/AGENTS.pre-level1.md").exists()
    assert check(repo) == ()
    assert check_activation(repo) == ()
