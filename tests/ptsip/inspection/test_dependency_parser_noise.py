from __future__ import annotations

import json
import subprocess
from pathlib import Path

from ptsip.inspection.dependencies_030 import scan_dependency_edges


def _git(repo: Path, *args: str) -> str:
    return subprocess.run(
        ["git", "-C", str(repo), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    ).stdout.strip()


def _init(repo: Path) -> None:
    repo.mkdir(parents=True)
    _git(repo, "init")
    _git(repo, "config", "user.email", "ptsip-test@example.invalid")
    _git(repo, "config", "user.name", "PTSIP Test")


def _commit(repo: Path) -> str:
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")
    return _git(repo, "rev-parse", "HEAD")


def test_go_comment_import_is_ignored_but_real_import_remains(tmp_path: Path) -> None:
    repo = tmp_path / "go"
    _init(repo)
    (repo / "go.mod").write_text("module example.com/product\n", encoding="utf-8")
    (repo / "main.go").write_text(
        'package main\n/*\nimport "example.com/fake"\n*/\nimport "fmt"\n',
        encoding="utf-8",
    )
    _commit(repo)
    edges = [item for item in scan_dependency_edges(repo).edges if item.adapter == "go-source"]
    assert [item.target for item in edges] == ["fmt"]


def test_javascript_comments_and_strings_do_not_create_edges(tmp_path: Path) -> None:
    repo = tmp_path / "js"
    _init(repo)
    (repo / "package.json").write_text(
        json.dumps({"name": "fixture"}),
        encoding="utf-8",
    )
    (repo / "real.js").write_text("export const value = 1;\n", encoding="utf-8")
    (repo / "index.js").write_text(
        "// require('./comment')\n/* import('./block') */\nconst text = \"require('./string')\";\nconst real = require('./real');\n",
        encoding="utf-8",
    )
    _commit(repo)
    edges = [item for item in scan_dependency_edges(repo).edges if item.source == "index.js"]
    assert [item.target for item in edges] == ["./real"]


def test_csharp_comments_and_strings_do_not_create_using_edges(tmp_path: Path) -> None:
    repo = tmp_path / "cs"
    _init(repo)
    (repo / "App.csproj").write_text(
        '<Project Sdk="Microsoft.NET.Sdk" />',
        encoding="utf-8",
    )
    (repo / "Program.cs").write_text(
        '// using Fake.Line;\n/* using Fake.Block; */\nvar text = "using Fake.String;";\nusing System.Text;\n',
        encoding="utf-8",
    )
    _commit(repo)
    edges = [item for item in scan_dependency_edges(repo).edges if item.source == "Program.cs"]
    assert [item.target for item in edges] == ["System.Text"]
