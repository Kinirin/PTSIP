from __future__ import annotations

import ast
from pathlib import Path

import pytest

from developer.automation import seed_pp_102_transition as seed_module


ROOT = Path(__file__).resolve().parents[2]


def test_completed_pp_102_seed_rejects_replay_before_reading_retired_root(monkeypatch) -> None:
    original = seed_module._load_yaml
    reads = []

    def load_current_registry(path):
        assert path == ROOT / "registry" / "project-profile-contracts.yaml"
        reads.append(path)
        return original(path)

    monkeypatch.setattr(seed_module, "_require_clean", lambda _root: None)
    monkeypatch.setattr(seed_module, "_load_yaml", load_current_registry)
    assert not (ROOT / "ptsip.yaml").exists()
    with pytest.raises(seed_module.SeedError, match="Seed requires current pp.1.01.*refusing replay"):
        seed_module.seed(ROOT)
    assert reads == [ROOT / "registry" / "project-profile-contracts.yaml"]


def test_pp_102_seed_does_not_own_transition_outputs() -> None:
    path = ROOT / "developer" / "automation" / "seed_pp_102_transition.py"
    source = path.read_text(encoding="utf-8")
    ast.parse(source)

    assert 'SOURCE_PP = "pp.1.01"' in source
    assert '"pp.1.02"' in source
    assert 'USER_BASELINE_REVISION = "Rev.0001"' in source
    assert 'DISTRIBUTED_EXAMPLE = "DISTRIBUTED_EXAMPLE"' in source
    assert 'MATERIALIZATION_REQUIRED = "PROJECT_PATH_RESOLUTION_REQUIRED"' in source

    assert "profiles/history/" in source
    assert "schemas/ptsip-profile-pp-1.02.schema.json" in source
    assert "src/ptsip/specdata/ptsip-profile-pp-1.02.schema.json" in source
    assert "AUTOMATION_OWNED_FORBIDDEN" in source


def test_pp_102_seed_declares_placeholderized_example_paths() -> None:
    source = (
        ROOT / "developer" / "automation" / "seed_pp_102_transition.py"
    ).read_text(encoding="utf-8")

    for placeholder in (
        "$" + "{PRODUCT_RUNTIME_ROOT}/**",
        "$" + "{PRODUCT_SDK_ROOT}/**",
        "$" + "{PRODUCT_TEST_ROOT}/**",
        "$" + "{DEVELOPMENT_TOOLING_ROOT}/**",
        "$" + "{RELEASE_AUTOMATION_FILE}",
        "$" + "{OPERATIONS_ROOT}/**",
        "$" + "{CONTRACT_ROOT}/**",
    ):
        assert placeholder in source


def test_pp_102_seed_updates_pp_surfaces_without_rewriting_tool_history() -> None:
    source = (
        ROOT / "developer" / "automation" / "seed_pp_102_transition.py"
    ).read_text(encoding="utf-8")

    assert "releasenote/project-profile" in source
    assert "_project_profile_note(target, specification_revision)" in source
    assert 'repo / "README.md"' in source
    assert 'repo / "STATUS.md"' in source
    assert "releasenote/tool/0.3.8a1.md" not in source


def test_pp_102_seed_writes_transition_acceptance_test() -> None:
    source = (
        ROOT / "developer" / "automation" / "seed_pp_102_transition.py"
    ).read_text(encoding="utf-8")

    assert "_write_pp_102_acceptance_test(repo, changed)" in source
    assert "test_pp_102_current_contract.py" in source
    assert 'registry["current"] == CURRENT_PROJECT_PROFILE_VERSION == "pp.1.02"' in source
    assert 'payload["ptsip"]["revision"] == "Rev.0001"' in source
