from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

from ptsip.cli import _configure_console_encoding, main
from ptsip.inspection.dependencies import scan_dependency_edges
from ptsip.inspection.inventory import collect_inventory
from ptsip.pilot.runner import run_pilot
from ptsip.repository.discover import discover_repository
from ptsip.repository.snapshot import capture_snapshot, compare_snapshots
from ptsip.spec_identity import current_spec_identity
from ptsip.validation.profile import validate_profile


def _git(repo: Path, *args: str) -> None:
    subprocess.run(["git", "-C", str(repo), *args], check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)


def _init_git(repo: Path) -> None:
    repo.mkdir(parents=True, exist_ok=True)
    _git(repo, "init")
    _git(repo, "config", "user.email", "ptsip-test@example.invalid")
    _git(repo, "config", "user.name", "PTSIP Test")


def _commit_all(repo: Path) -> None:
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")


def test_spec_identity():
    spec = current_spec_identity()
    assert spec.tool_version == "0.3.4"
    assert spec.version == "0.2.0-draft"
    assert spec.source == "https://github.com/kwaksinwoo01/ptsip"
    assert spec.revision == "a877b2f66a7f94c1b844c979e1b08fb08a9a8e45"


def test_console_encoding_fallback_preserves_unencodable_text(monkeypatch) -> None:
    class Stream:
        def __init__(self) -> None:
            self.errors = "strict"

        def reconfigure(self, *, errors: str) -> None:
            self.errors = errors

    stream = Stream()
    error_stream = Stream()
    monkeypatch.setattr(sys, "stdout", stream)
    monkeypatch.setattr(sys, "stderr", error_stream)
    _configure_console_encoding()
    assert stream.errors == "backslashreplace"
    assert error_stream.errors == "backslashreplace"


def test_inventory_reports_parse_failures(tmp_path: Path):
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


def test_snapshot_detects_repository_change(tmp_path: Path):
    repo = tmp_path / "repo"
    _init_git(repo)
    target = repo / "app.py"
    target.write_text("x = 1\n", encoding="utf-8")
    _commit_all(repo)
    before = capture_snapshot(repo)
    target.write_text("x = 2\n", encoding="utf-8")
    after = capture_snapshot(repo)
    comparison = compare_snapshots(before, after)
    assert not comparison.stable
    assert comparison.status == "INVALIDATED"
    assert any("tracked file content" in reason or "working-tree" in reason for reason in comparison.reasons)


def test_pilot_v2_writes_external_state_only(tmp_path: Path, monkeypatch):
    repo = tmp_path / "repo"
    state = tmp_path / "state"
    _init_git(repo)
    (repo / "src").mkdir()
    (repo / "src" / "x.py").write_text("x = 1\n", encoding="utf-8")
    _commit_all(repo)
    monkeypatch.setenv("PTSIP_HOME", str(state))
    before = capture_snapshot(repo)
    result = run_pilot(repo)
    after = capture_snapshot(repo)
    assert result.report["format"] == "ptsip-pilot-report/v2"
    assert result.report_path.is_file()
    assert state in result.report_path.parents
    assert result.report["snapshot"]["comparison"]["stable"] is True
    assert result.report["non_intrusion"]["status"] == "VERIFIED_NO_OBSERVED_CHANGE"
    assert compare_snapshots(before, after).stable
    assert result.report["classification"]["allowed_classifications"] == [
        "PRODUCT",
        "TOOLCHAIN",
        "NEUTRAL_CONTRACT",
    ]
    assert "UNKNOWN" in result.report["classification"]["decision_statuses"]
    evaluator = result.report["evaluation"]["declared_dependency_boundaries"]
    assert evaluator["status"] == "BLOCKED"
    assert evaluator["reason"] == "NO_PROFILE"
    assert evaluator["finding_count"] is None


def test_component_profile_allows_specific_nested_override(tmp_path: Path):
    repo = tmp_path / "repo"
    _init_git(repo)
    (repo / "src" / "install").mkdir(parents=True)
    (repo / "src" / "app.py").write_text("x = 1\n", encoding="utf-8")
    (repo / "src" / "install" / "plugin_build.py").write_text("x = 1\n", encoding="utf-8")
    _commit_all(repo)
    profile = repo / "ptsip.yaml"
    profile.write_text(
        """ptsip:\n  version: \"0.2.0-draft\"\n  specification:\n    source: \"https://github.com/kwaksinwoo01/ptsip\"\n    revision: \"a877b2f66a7f94c1b844c979e1b08fb08a9a8e45\"\ncomponents:\n  - id: product-runtime\n    classification: PRODUCT\n    include: [\"src/**\"]\n    purpose: product_runtime\n  - id: plugin-builder\n    classification: TOOLCHAIN\n    include: [\"src/install/plugin_build.py\"]\n    purpose: build_and_release\npolicies:\n  product_to_toolchain_runtime_dependency: deny\n  toolchain_in_product_package: deny\n  independent_build_resolution: required\n""",
        encoding="utf-8",
    )
    result = validate_profile(repo)
    assert result.valid, result.errors
    partition = result.details["component_partition"]
    owners = {item["path"]: item["component_id"] for item in partition["assignments"]}
    assert owners["src/app.py"] == "product-runtime"
    assert owners["src/install/plugin_build.py"] == "plugin-builder"


def test_legacy_exception_waiver_is_rejected(tmp_path: Path):
    repo = tmp_path / "repo"
    _init_git(repo)
    (repo / "src").mkdir()
    (repo / "src" / "app.py").write_text("x = 1\n", encoding="utf-8")
    _commit_all(repo)
    profile = repo / "ptsip.yaml"
    profile.write_text(
        """ptsip:\n  version: \"0.2.0-draft\"\n  specification:\n    source: \"https://github.com/kwaksinwoo01/ptsip\"\nboundaries:\n  product:\n    roots: [\"src\"]\n  toolchain:\n    roots: [\"tools\"]\npolicies:\n  product_to_toolchain_runtime_dependency: deny\n  toolchain_in_product_package: deny\n  independent_build_resolution: required\nexceptions: []\n""",
        encoding="utf-8",
    )
    result = validate_profile(repo)
    assert not result.valid
    assert any("exceptions" in error and "unexpected" in error.lower() for error in result.errors)


def test_python_dependency_edge_is_preserved(tmp_path: Path):
    repo = tmp_path / "repo"
    _init_git(repo)
    (repo / "src").mkdir()
    (repo / "tools").mkdir()
    (repo / "src" / "app.py").write_text("VALUE = 1\n", encoding="utf-8")
    (repo / "tools" / "check.py").write_text("import app\n", encoding="utf-8")
    _commit_all(repo)
    scan = scan_dependency_edges(repo)
    edge = next(item for item in scan.edges if item.source == "tools/check.py" and item.target == "app")
    assert edge.resolved_path == "src/app.py"
    assert edge.edge_type.value == "IMPORTS"
    assert edge.phase.value == "UNKNOWN"
    assert edge.target_scope.value == "PROJECT_COMPONENT"
    assert edge.provenance.value == "OBSERVED"


def test_cli_pilot_json(tmp_path: Path, monkeypatch, capsys):
    repo = tmp_path / "repo"
    _init_git(repo)
    (repo / "app.py").write_text("x = 1\n", encoding="utf-8")
    _commit_all(repo)
    monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))
    assert main(["pilot", str(repo), "--json"]) == 0
    payload = json.loads(capsys.readouterr().out)
    assert payload["format"] == "ptsip-pilot-report/v2"
    assert payload["tool"]["version"] == "0.3.4"
    assert payload["non_intrusion"]["status"] == "VERIFIED_NO_OBSERVED_CHANGE"
