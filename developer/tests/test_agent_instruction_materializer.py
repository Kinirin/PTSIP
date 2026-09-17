from __future__ import annotations

from pathlib import Path

import yaml

from developer.automation.agent_instruction_materializer import (
    DEFAULT_OUTPUT_ROOT,
    build_materialization,
    check_materialization,
    materialize,
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
    return tmp_path


def test_level1_materialization_stores_each_atom_once_and_uses_id_projections(
    tmp_path: Path,
) -> None:
    repo = _repo(tmp_path)
    source = repo / "AGENTS.md"
    source.write_text(
        "# AGENTS\n\n"
        "For SDK work:\n\n"
        "- Read `docs/sdk.md`.\n"
        "- Read source files; do not edit generated files.\n\n"
        "Architecture baseline: MVC plus EDA\n",
        encoding="utf-8",
    )

    registry, index, projections, unresolved = build_materialization(
        Path("AGENTS.md"),
        root=repo,
    )

    atoms = registry["atoms"]
    assert isinstance(atoms, list)
    ids = [item["atom_id"] for item in atoms]
    assert len(ids) == len(set(ids))
    assert index["level_2_materialized"] is False
    assert registry["level_2_materialized"] is False

    rule_ids = projections["RULE"]["atom_ids"]
    action_ids = projections["ACTION"]["atom_ids"]
    assert set(rule_ids) & set(action_ids)
    assert all("normalized_text" not in item for item in projections.values())
    assert unresolved["routing_state"] == "UNRESOLVED"


def test_materialization_preserves_raw_source_excerpt_and_digest(
    tmp_path: Path,
) -> None:
    repo = _repo(tmp_path)
    source = repo / "AGENTS.md"
    source.write_text(
        "# AGENTS\n\n- Never edit generated files directly.\n",
        encoding="utf-8",
    )

    registry, _, _, _ = build_materialization(Path("AGENTS.md"), root=repo)
    item = next(
        item
        for item in registry["atoms"]
        if item["normalized_text"] == "Never edit generated files directly."
    )
    assert item["source"]["raw_excerpt"] == "- Never edit generated files directly.\n"
    assert len(item["source"]["raw_excerpt_sha256"]) == 64


def test_level2_is_not_materialized_even_when_classifier_can_suggest_it(
    tmp_path: Path,
) -> None:
    repo = _repo(tmp_path)
    (repo / "AGENTS.md").write_text(
        "# AGENTS\n\nBefore publishing, run pytest.\n",
        encoding="utf-8",
    )

    registry, _, projections, _ = build_materialization(Path("AGENTS.md"), root=repo)
    rendered = yaml.safe_dump(
        {"registry": registry, "projections": projections},
        sort_keys=False,
    )
    assert "level_2:" not in rendered
    assert "level_2_materialized: false" in rendered
    assert "PUBLISH:" not in rendered
    assert "TEST:" not in rendered


def test_unresolved_remains_separate_and_is_not_coerced_to_other(
    tmp_path: Path,
) -> None:
    repo = _repo(tmp_path)
    (repo / "AGENTS.md").write_text(
        "# AGENTS\n\nCanonical roles:\n\nALPHA BETA GAMMA\n",
        encoding="utf-8",
    )

    registry, _, projections, unresolved = build_materialization(
        Path("AGENTS.md"),
        root=repo,
    )
    unresolved_ids = set(unresolved["atom_ids"])
    assert unresolved_ids
    assert unresolved_ids.isdisjoint(set(projections["OTHER"]["atom_ids"]))
    assert all(
        item["level_1"] != ["OTHER"]
        for item in registry["atoms"]
        if item["atom_id"] in unresolved_ids
    )


def test_materialize_then_check_is_reproducible_and_source_change_goes_stale(
    tmp_path: Path,
) -> None:
    repo = _repo(tmp_path)
    source = repo / "AGENTS.md"
    source.write_text(
        "# AGENTS\n\nRead `README.md` before editing.\n",
        encoding="utf-8",
    )

    written = materialize(Path("AGENTS.md"), root=repo)
    assert written
    assert (repo / DEFAULT_OUTPUT_ROOT / "registry.yaml").is_file()
    assert check_materialization(Path("AGENTS.md"), root=repo) == ()

    source.write_text(
        "# AGENTS\n\nRead `README.md` before editing.\n\nNever rewrite history.\n",
        encoding="utf-8",
    )
    errors = check_materialization(Path("AGENTS.md"), root=repo)
    assert errors
    assert any(error.startswith("STALE:") for error in errors)
