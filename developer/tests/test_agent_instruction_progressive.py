from __future__ import annotations

import json
from pathlib import Path

import yaml

from developer.automation.agent_instruction_entry_resolver import resolve_entry
from developer.automation.agent_instruction_progressive import check, migrate_level1


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
        "- Do not scan developer/policy/** merely to discover policy.\n\n"
        "Canonical roles:\n\n"
        "ALPHA BETA GAMMA\n",
        encoding="utf-8",
    )
    return tmp_path


def test_clean_repository_bootstraps_level1_without_legacy_agent_surface(
    tmp_path: Path,
) -> None:
    repo = _repo(tmp_path)
    original = (repo / "AGENTS.md").read_text(encoding="utf-8")
    assert not (repo / ".agent").exists()

    result = migrate_level1(repo)

    assert result["mode"] == "BOOTSTRAPPED_FROM_AGENTS"
    assert result["pass_count"] > 0
    assert result["unresolved_count"] > 0
    assert (repo / ".agent/index.yaml").is_file()
    assert (repo / ".agent/stages/level1.json").is_file()
    assert (repo / ".agent/unresolved/level1.json").is_file()
    assert not (repo / ".agent/registry.yaml").exists()
    assert not (repo / ".agent/provenance/AGENTS.pre-level1.md").exists()
    assert (repo / "AGENTS.md").read_text(encoding="utf-8") == original
    assert check(repo) == ()


def test_bootstrap_index_binds_source_without_making_it_runtime_registry(
    tmp_path: Path,
) -> None:
    repo = _repo(tmp_path)
    migrate_level1(repo)

    index = yaml.safe_load(
        (repo / ".agent/index.yaml").read_text(encoding="utf-8")
    )
    progressive = index["progressive_reasoning"]
    assert progressive["source_state"] == "SOURCE_PRESERVED"
    assert progressive["previous_level_rerun_forbidden"] is True
    assert progressive["unresolved_blocks_next_level_candidates"] is False
    assert index["source"]["path"] == "AGENTS.md"
    assert len(index["source"]["sha256"]) == 64

    stage = json.loads(
        (repo / ".agent/stages/level1.json").read_text(encoding="utf-8")
    )
    unresolved = json.loads(
        (repo / ".agent/unresolved/level1.json").read_text(encoding="utf-8")
    )
    assert stage["pass_count"] == len(stage["pass"])
    assert unresolved["count"] == len(unresolved["items"])


def test_entry_resolver_consumes_bootstrapped_progressive_stage(
    tmp_path: Path,
) -> None:
    repo = _repo(tmp_path)
    migrate_level1(repo)

    result = resolve_entry(repo, operation="MODIFY")

    assert result["stage"] == "PROGRESSIVE_LEVEL_1"
    assert result["previous_level_rerun"] is False
    assert result["instructions"]
    assert all(
        "machine" in item and "natural_residual" in item
        for item in result["instructions"]
    )
    assert result["unresolved"]
    assert all("natural_language" in item for item in result["unresolved"])


def test_source_change_marks_clean_bootstrap_stale(tmp_path: Path) -> None:
    repo = _repo(tmp_path)
    migrate_level1(repo)

    with (repo / "AGENTS.md").open("a", encoding="utf-8") as handle:
        handle.write("\nNew instruction.\n")

    assert "SOURCE_AGENTS_STALE" in check(repo)
