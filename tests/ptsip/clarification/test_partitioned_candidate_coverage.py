from __future__ import annotations

from pathlib import Path

from ptsip.clarification.generator import _candidate_scope_is_fully_partitioned
from ptsip.inspection.components import ComponentCandidate


def _candidate() -> ComponentCandidate:
    return ComponentCandidate(
        id="tests",
        include=("tests/**",),
        anchors=("top-level-test-root",),
        evidence_ids=("root:tests",),
    )


def _components() -> list[dict[str, object]]:
    return [
        {
            "id": "verification-a",
            "include": ["tests/a/**"],
        },
        {
            "id": "verification-b",
            "include": ["tests/b/**"],
        },
    ]


def test_coarse_candidate_is_satisfied_by_complete_declared_partition(tmp_path: Path) -> None:
    (tmp_path / "tests" / "a").mkdir(parents=True)
    (tmp_path / "tests" / "b").mkdir(parents=True)
    (tmp_path / "tests" / "a" / "test_a.py").write_text("pass\n", encoding="utf-8")
    (tmp_path / "tests" / "b" / "test_b.py").write_text("pass\n", encoding="utf-8")

    assert _candidate_scope_is_fully_partitioned(
        tmp_path,
        _candidate(),
        _components(),
        [],
    )


def test_coarse_candidate_reopens_when_any_observed_path_is_unowned(tmp_path: Path) -> None:
    (tmp_path / "tests" / "a").mkdir(parents=True)
    (tmp_path / "tests" / "b").mkdir(parents=True)
    (tmp_path / "tests" / "unowned").mkdir(parents=True)
    (tmp_path / "tests" / "a" / "test_a.py").write_text("pass\n", encoding="utf-8")
    (tmp_path / "tests" / "b" / "test_b.py").write_text("pass\n", encoding="utf-8")
    (tmp_path / "tests" / "unowned" / "test_c.py").write_text("pass\n", encoding="utf-8")

    assert not _candidate_scope_is_fully_partitioned(
        tmp_path,
        _candidate(),
        _components(),
        [],
    )
