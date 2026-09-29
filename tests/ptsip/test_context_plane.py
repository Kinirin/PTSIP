from __future__ import annotations

import json
from pathlib import Path

import pytest

from ptsip.context_plane import (
    JSON_PATH,
    MEMORY_PATH,
    SOURCE_PATH,
    ContextProjectionError,
    check_context,
    context_status,
    migrate_context,
    repair_context,
    write_context,
)


def _source() -> dict[str, object]:
    return {
        "format": "ptsip-context-source/v1",
        "projection_policy": {
            "authority": "CANONICAL_SEMANTIC_MODEL",
            "extension_policy": "EXTENSIBLE",
            "generated_only": True,
            "mandatory_formats": ["json", "jsonl", "schema-json"],
            "provider_scope": ["OPENAI", "ANTHROPIC", "GOOGLE", "XAI"],
            "semantic_equivalence": "REQUIRED",
        },
        "state": {
            "format": "ptsip-project-state/v1",
            "status": "CURRENT",
            "project_profile": {},
            "specification": {},
            "tool": {},
            "planning": {},
            "agent_contract": {},
            "verification": {},
            "context_plane": {},
            "legacy_markdown_state": {},
        },
        "memory": [],
    }


def _input(tmp_path: Path, name: str = "prepared-context.json") -> Path:
    path = tmp_path / name
    path.write_text(json.dumps(_source()), encoding="utf-8")
    return path


def test_legacy_memory_without_context_is_machine_authorized_for_migration(tmp_path: Path) -> None:
    (tmp_path / MEMORY_PATH).write_text("legacy context", encoding="utf-8")
    status = context_status(tmp_path)
    assert status["state"] == "LEGACY_MIGRATION_REQUIRED"
    assert status["agent_action"]["authorization"] == "PREAUTHORIZED"
    assert status["agent_action"]["user_decision_required"] is False


def test_migration_validates_context_before_deleting_memory(tmp_path: Path) -> None:
    (tmp_path / MEMORY_PATH).write_text("legacy context", encoding="utf-8")
    result = migrate_context(tmp_path, _input(tmp_path))
    assert result["status"] == "MIGRATED"
    assert (tmp_path / SOURCE_PATH).is_file()
    assert not (tmp_path / MEMORY_PATH).exists()
    check_context(tmp_path)


def test_failed_migration_preserves_memory(tmp_path: Path) -> None:
    memory = tmp_path / MEMORY_PATH
    memory.write_text("legacy context", encoding="utf-8")
    invalid = tmp_path / "invalid.json"
    invalid.write_text('{"format":"wrong"}', encoding="utf-8")
    with pytest.raises(ContextProjectionError):
        migrate_context(tmp_path, invalid)
    assert memory.is_file()


def test_projection_drift_is_preauthorized_and_repairable(tmp_path: Path) -> None:
    write_context(_input(tmp_path), tmp_path)
    (tmp_path / JSON_PATH).write_text("{}\n", encoding="utf-8")
    status = context_status(tmp_path)
    assert status["state"] == "PROJECTION_REPAIR_REQUIRED"
    assert status["agent_action"]["operation"] == "REGENERATE_PROJECTIONS"
    repair_context(tmp_path)
    check_context(tmp_path)


def test_active_context_can_retire_leftover_memory_without_semantic_rewrite(tmp_path: Path) -> None:
    write_context(_input(tmp_path), tmp_path)
    (tmp_path / MEMORY_PATH).write_text("already migrated", encoding="utf-8")
    assert context_status(tmp_path)["state"] == "ACTIVE_WITH_LEGACY_MEMORY"
    result = migrate_context(tmp_path)
    assert result["status"] == "LEGACY_MEMORY_RETIRED"
    assert not (tmp_path / MEMORY_PATH).exists()


def test_existing_source_replacement_requires_explicit_user_approval(tmp_path: Path) -> None:
    write_context(_input(tmp_path), tmp_path)
    replacement = _input(tmp_path, "replacement.json")
    with pytest.raises(ContextProjectionError, match="USER_DECISION_REQUIRED"):
        migrate_context(tmp_path, replacement)
