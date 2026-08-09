from __future__ import annotations

import json
import subprocess
from pathlib import Path

from ptsip.cli import main


def _git(repo: Path, *args: str) -> None:
    subprocess.run(["git", "-C", str(repo), *args], check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)


def test_inspect_uses_030_composite_dependency_scanner(tmp_path: Path, capsys) -> None:
    repo = tmp_path / "repo"
    repo.mkdir()
    _git(repo, "init")
    _git(repo, "config", "user.email", "ptsip-test@example.invalid")
    _git(repo, "config", "user.name", "PTSIP Test")
    (repo / "package.json").write_text(json.dumps({"name": "fixture", "dependencies": {"left-pad": "1.3.0"}}), encoding="utf-8")
    (repo / "index.ts").write_text('import leftPad from "left-pad";\nconsole.log(leftPad);\n', encoding="utf-8")
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")

    assert main(["inspect", str(repo), "--json"]) == 0
    payload = json.loads(capsys.readouterr().out)
    assert "javascript-typescript" in payload["dependencies"]["adapters"]
    assert "npm-manifest" in payload["dependencies"]["adapters"]
    assert any(item["target"] == "left-pad" and item["target_scope"] == "EXTERNAL_DEPENDENCY" for item in payload["dependencies"]["edges"])
