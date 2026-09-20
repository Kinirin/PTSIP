from __future__ import annotations

import ast
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]


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
