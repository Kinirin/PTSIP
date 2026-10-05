from __future__ import annotations

import subprocess
from pathlib import Path

from ptsip.inspection.dependencies import scan_dependency_edges
from ptsip.inspection.inventory import collect_inventory


def _git(repo: Path, *args: str) -> None:
    subprocess.run(
        ["git", "-C", str(repo), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )


def _init_git(repo: Path) -> None:
    repo.mkdir(parents=True, exist_ok=True)
    _git(repo, "init")
    _git(repo, "config", "user.email", "ptsip-test@example.invalid")
    _git(repo, "config", "user.name", "PTSIP Test")


def _commit_all(repo: Path) -> None:
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")


def test_inventory_reports_parse_failures(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    (repo / "good.py").write_text("import json\n", encoding="utf-8")
    (repo / "bad.py").write_text("def broken(:\n", encoding="utf-8")
    _commit_all(repo)
    inv = collect_inventory(repo)
    assert inv.scan_mode == "git-tracked"
    assert inv.python_modules == 2
    assert inv.python_imports == 1
    assert any(issue.category == "PYTHON_PARSE_ERROR" for issue in inv.scan_issues)
    assert not inv.coverage_complete


def test_python_dependency_edge_is_preserved(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    (repo / "src").mkdir()
    (repo / "tools").mkdir()
    (repo / "src" / "app.py").write_text("VALUE = 1\n", encoding="utf-8")
    (repo / "tools" / "check.py").write_text("import app\n", encoding="utf-8")
    _commit_all(repo)
    scan = scan_dependency_edges(repo)
    edge = next(
        item
        for item in scan.edges
        if item.source == "tools/check.py" and item.target == "app"
    )
    assert edge.resolved_path == "src/app.py"
    assert edge.edge_type.value == "IMPORTS"
    assert edge.phase.value == "UNKNOWN"
    assert edge.target_scope.value == "PROJECT_COMPONENT"
    assert edge.provenance.value == "OBSERVED"
