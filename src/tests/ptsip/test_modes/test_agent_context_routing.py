"""Preserve test-mode routing coverage independently of Developer automation."""

from pathlib import Path
import runpy

import yaml


ROOT = Path(__file__).resolve().parents[4]


def test_operation_implementations_and_contract_edits_select_repository_validation():
    resolver = runpy.run_path(str(ROOT / ".github/scripts/resolve_test_modes.py"))
    profile = yaml.safe_load((ROOT / "developer/profiles/ptsip-repository.yaml").read_text(encoding="utf-8"))
    registry = yaml.safe_load((ROOT / ".github/test_modes.yaml").read_text(encoding="utf-8"))
    contract_root = ROOT / "src/ptsip/agent_contracts"
    index = yaml.safe_load((contract_root / "index.yaml").read_text(encoding="utf-8"))
    paths = {"src/ptsip/agent_contracts/operations/migrate-profile.yaml"}
    for entry in index["operations"]:
        operation = yaml.safe_load((contract_root / entry["ref"]).read_text(encoding="utf-8"))
        paths.update(operation["implementation_refs"])
    assert len(paths) > 1
    for path in sorted(paths):
        modes, _ = resolver["resolve_automatic_selection"](registry, profile, [path])
        assert "repository-architecture" in {mode["id"] for mode in modes}, path


def test_migration_sources_select_binding_and_repository_validation():
    resolver = runpy.run_path(str(ROOT / ".github/scripts/resolve_test_modes.py"))
    profile = yaml.safe_load((ROOT / "developer/profiles/ptsip-repository.yaml").read_text(encoding="utf-8"))
    registry = yaml.safe_load((ROOT / ".github/test_modes.yaml").read_text(encoding="utf-8"))
    sources = sorted((ROOT / "src/ptsip/migration").rglob("*.py"))
    assert sources
    for source in sources:
        path = source.relative_to(ROOT).as_posix()
        modes, _ = resolver["resolve_automatic_selection"](registry, profile, [path])
        assert {"agent-contract-plane", "repository-architecture"} <= {mode["id"] for mode in modes}, path
