from __future__ import annotations

import json
from pathlib import Path

from ptsip.cli import main
from ptsip.inspection.inventory import collect_inventory
from ptsip.pilot.runner import run_pilot
from ptsip.repository.discover import discover_repository
from ptsip.spec_identity import current_spec_identity
from ptsip.validation.profile import validate_profile


def test_spec_identity():
    spec = current_spec_identity()
    assert spec.version == "0.2.0-draft"
    assert spec.source == "https://github.com/kwaksinwoo01/ptsip-spec"
    assert spec.revision == "cb4164a803678a0364ce037af4addbad1d7ecc7d"


def test_inspection_is_read_only(tmp_path: Path):
    repo = tmp_path / "repo"
    repo.mkdir()
    (repo / "app.py").write_text("import json\n", encoding="utf-8")
    before = sorted(p.relative_to(repo).as_posix() for p in repo.rglob("*"))
    info = discover_repository(repo)
    inv = collect_inventory(info.root)
    after = sorted(p.relative_to(repo).as_posix() for p in repo.rglob("*"))
    assert inv.python_modules == 1
    assert before == after


def test_pilot_writes_external_state_only(tmp_path: Path, monkeypatch):
    repo = tmp_path / "repo"
    state = tmp_path / "state"
    repo.mkdir()
    (repo / "src").mkdir()
    (repo / "src" / "x.py").write_text("x = 1\n", encoding="utf-8")
    monkeypatch.setenv("PTSIP_HOME", str(state))
    before = sorted(p.relative_to(repo).as_posix() for p in repo.rglob("*"))
    result = run_pilot(repo)
    after = sorted(p.relative_to(repo).as_posix() for p in repo.rglob("*"))
    assert before == after
    assert result.report_path.is_file()
    assert state in result.report_path.parents
    assert result.report["consumer_repository_modified"] is False


def test_validate_profile(tmp_path: Path):
    repo = tmp_path / "repo"
    repo.mkdir()
    profile = repo / "ptsip.yaml"
    profile.write_text(
        """ptsip:\n  version: \"0.2.0-draft\"\n  specification:\n    source: \"https://github.com/kwaksinwoo01/ptsip-spec\"\nboundaries:\n  product:\n    roots: [\"src\"]\n  toolchain:\n    roots: [\"devtools\"]\npolicies:\n  product_to_toolchain_runtime_dependency: deny\n  toolchain_in_product_package: deny\n  independent_build_resolution: required\nexceptions: []\n""",
        encoding="utf-8",
    )
    result = validate_profile(repo)
    assert result.valid
    assert any("immutable revision" in warning for warning in result.warnings)


def test_cli_pilot_json(tmp_path: Path, monkeypatch, capsys):
    repo = tmp_path / "repo"
    repo.mkdir()
    monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))
    assert main(["pilot", str(repo), "--json"]) == 0
    payload = json.loads(capsys.readouterr().out)
    assert payload["consumer_repository_modified"] is False
