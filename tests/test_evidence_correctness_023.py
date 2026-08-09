from __future__ import annotations

import json
import subprocess
from importlib.resources import files
from pathlib import Path

import yaml

from ptsip.inspection.dependencies import scan_dependency_edges
from ptsip.inspection.inventory import collect_inventory
from ptsip.pilot.runner import run_pilot


SPEC_REVISION = "14a0c2f54bb486de6a109979224f998b04fd04a3"


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


def test_python_bom_is_decoded_for_inventory_and_dependencies(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    (repo / "app.py").write_bytes(b"\xef\xbb\xbfimport json\n")
    _commit_all(repo)

    inventory = collect_inventory(repo)
    assert inventory.python_imports == 1
    assert not inventory.scan_issues

    scan = scan_dependency_edges(repo)
    edge = next(item for item in scan.edges if item.source == "app.py" and item.target == "json")
    assert edge.target_scope.value == "PLATFORM"
    assert edge.resolution.value == "EXTERNAL"


def test_relative_python_import_resolves_from_package_evidence(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    package = repo / "pkg"
    package.mkdir()
    (package / "__init__.py").write_text("", encoding="utf-8")
    (package / "a.py").write_text("from . import b\n", encoding="utf-8")
    (package / "b.py").write_text("VALUE = 1\n", encoding="utf-8")
    _commit_all(repo)

    scan = scan_dependency_edges(repo)
    edge = next(item for item in scan.edges if item.source == "pkg/a.py")
    assert edge.target == "pkg.b"
    assert edge.resolved_path == "pkg/b.py"
    assert edge.target_scope.value == "PROJECT_COMPONENT"
    assert edge.resolution.value == "RESOLVED"


def test_python_target_scope_distinguishes_external_and_unresolved(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    (repo / "pyproject.toml").write_text(
        """[project]\nname = \"fixture\"\nversion = \"0.0.0\"\ndependencies = [\"requests>=2\"]\n""",
        encoding="utf-8",
    )
    (repo / "app.py").write_text("import requests\nimport mystery_package\n", encoding="utf-8")
    _commit_all(repo)

    scan = scan_dependency_edges(repo)
    by_target = {edge.target: edge for edge in scan.edges if edge.source == "app.py"}
    assert by_target["requests"].target_scope.value == "EXTERNAL_DEPENDENCY"
    assert by_target["requests"].resolution.value == "EXTERNAL"
    assert by_target["mystery_package"].target_scope.value == "UNRESOLVED_TARGET"
    assert by_target["mystery_package"].resolution.value == "UNRESOLVED"


def test_dynamic_python_import_uses_loads_and_unresolved_scope(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    (repo / "app.py").write_text("import importlib\nname = 'x'\nimportlib.import_module(name)\n", encoding="utf-8")
    _commit_all(repo)

    scan = scan_dependency_edges(repo)
    edge = next(item for item in scan.edges if item.target == "<dynamic-import>")
    assert edge.edge_type.value == "LOADS"
    assert edge.resolution.value == "DYNAMIC"
    assert edge.target_scope.value == "UNRESOLVED_TARGET"


def test_github_actions_resolves_scripts_from_effective_working_directory(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    (repo / ".github" / "workflows").mkdir(parents=True)
    (repo / "tools").mkdir()
    (repo / "tools" / "check.py").write_text("print('ok')\n", encoding="utf-8")
    (repo / ".github" / "workflows" / "test.yml").write_text(
        """name: test\non: [push]\ndefaults:\n  run:\n    working-directory: tools\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: python check.py\n      - run: echo not-a-local-file.py\n      - run: python ./missing.py\n""",
        encoding="utf-8",
    )
    _commit_all(repo)

    scan = scan_dependency_edges(repo)
    workflow_edges = [edge for edge in scan.edges if edge.adapter == "github-actions"]
    assert any(edge.target == "check.py" and edge.resolved_path == "tools/check.py" for edge in workflow_edges)
    resolved = next(edge for edge in workflow_edges if edge.target == "check.py")
    assert resolved.working_directory == "tools"
    assert resolved.provenance.value == "DECLARED"
    assert not any(edge.target == "not-a-local-file.py" for edge in workflow_edges)
    missing = next(edge for edge in workflow_edges if edge.target == "./missing.py")
    assert missing.resolution.value == "UNRESOLVED"
    assert missing.target_scope.value == "UNRESOLVED_TARGET"


def test_dependency_evaluator_reports_ran_instead_of_inferring_from_empty_findings(tmp_path: Path, monkeypatch) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    (repo / "product").mkdir()
    (repo / "tools").mkdir()
    (repo / "product" / "app.py").write_text("VALUE = 1\n", encoding="utf-8")
    (repo / "tools" / "check.py").write_text("VALUE = 2\n", encoding="utf-8")
    (repo / "ptsip.yaml").write_text(
        f"""ptsip:\n  version: \"0.2.0-draft\"\n  specification:\n    source: \"https://github.com/kwaksinwoo01/ptsip\"\n    revision: \"{SPEC_REVISION}\"\ncomponents:\n  - id: product\n    classification: PRODUCT\n    include: [\"product/**\"]\n    purpose: product_runtime\n  - id: tools\n    classification: TOOLCHAIN\n    include: [\"tools/**\"]\n    purpose: development_tooling\npolicies:\n  product_to_toolchain_runtime_dependency: deny\n  toolchain_in_product_package: deny\n  independent_build_resolution: required\n""",
        encoding="utf-8",
    )
    _commit_all(repo)
    monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))

    result = run_pilot(repo)
    evaluator = result.report["evaluation"]["declared_dependency_boundaries"]
    assert evaluator["status"] == "RAN"
    assert evaluator["reason"] is None
    assert evaluator["finding_count"] == 0
    assert result.report["conformance"]["status"] == "NOT_EVALUATED"


def test_embedded_profile_schema_and_registry_match_bound_snapshot_assets() -> None:
    root = Path(__file__).resolve().parents[1]
    canonical_schema = json.loads((root / "schemas/ptsip-profile.schema.json").read_text(encoding="utf-8"))
    embedded_schema = json.loads(files("ptsip").joinpath("specdata/ptsip-profile.schema.json").read_text(encoding="utf-8"))
    assert embedded_schema == canonical_schema

    canonical_registry = yaml.safe_load((root / "registry/ptsip-registry.yaml").read_text(encoding="utf-8"))
    embedded_registry = yaml.safe_load(files("ptsip").joinpath("specdata/ptsip-registry.yaml").read_text(encoding="utf-8"))
    assert embedded_registry == canonical_registry
