from pathlib import Path

import pytest

from ptsip.inspection.dependencies import scan_dependency_edges
from ptsip.dependency_analysis import analyze_dependencies
from test_dependency_reconciliation import _fixture, _git


def put(repo, path, text=""):
    destination = repo / path
    destination.parent.mkdir(parents=True, exist_ok=True)
    destination.write_text(text, encoding="utf-8")


def test_relative_imports_use_physical_tracked_context_and_preserve_multiple_targets(tmp_path):
    repo = tmp_path / "repo"
    _fixture(repo, "from . import left, right\nfrom .nested.target import Thing", "numpy==2")
    for path in ("product/__init__.py", "product/left.py", "product/right.py", "product/nested/target.py"):
        put(repo, path)
    _git(repo, "add", "product")
    scan = scan_dependency_edges(repo)
    targets = {edge.resolved_path for edge in scan.edges if edge.source == "product/app.py"}
    assert targets == {"product/left.py", "product/right.py", "product/nested/target.py"}
    report = analyze_dependencies(repo)
    assert report["summary"]["AUTO_RESOLVED"] == 3
    assert all(item["target_component"] == "product-app" for item in report["items"])


def test_namespace_source_layout_and_explicit_package_root_resolve(tmp_path):
    repo = tmp_path / "repo"
    _fixture(repo, "import app.feature\nimport plugin.helper", "numpy==2")
    put(repo, "src/app/feature.py")
    put(repo, "tools/component/lib/plugin/helper.py")
    put(repo, "tools/component/pyproject.toml", '[tool.setuptools.packages.find]\nwhere = ["lib"]\n')
    _git(repo, "add", "src", "tools")
    targets = {edge.resolved_path for edge in scan_dependency_edges(repo).edges}
    assert targets == {"src/app/feature.py", "tools/component/lib/plugin/helper.py"}
    # A resolvable path without component ownership is not AUTO_RESOLVED.
    assert analyze_dependencies(repo)["summary"]["AUTO_RESOLVED"] == 0


def test_ambiguous_local_target_is_never_external_even_when_declared(tmp_path):
    repo = tmp_path / "repo"
    _fixture(repo, "import shared_name", "shared_name==1")
    put(repo, "shared_name.py")
    put(repo, "src/shared_name.py")
    _git(repo, "add", "shared_name.py", "src")
    scan = scan_dependency_edges(repo)
    assert scan.edges[0].resolution.value == "UNRESOLVED"
    assert analyze_dependencies(repo)["summary"]["AUTO_RESOLVED"] == 0


def test_untracked_module_and_relative_escape_stay_unresolved(tmp_path):
    repo = tmp_path / "repo"
    _fixture(repo, "import untracked\nfrom ...outside import thing", "numpy==2")
    put(repo, "untracked.py")
    scan = scan_dependency_edges(repo)
    assert all(edge.resolved_path is None and edge.resolution.value == "UNRESOLVED" for edge in scan.edges)


def test_namespace_container_and_missing_local_child_are_not_externalized(tmp_path):
    repo = tmp_path / "repo"
    _fixture(repo, 'import shared_name\nfrom shared_name import child\n'
             'import shared_name.missing\n__import__("shared_name")', "shared_name==1")
    put(repo, "src/shared_name/child.py")
    put(repo, "requirements.txt", "shared_name==1\n")
    _git(repo, "add", "src", "requirements.txt")
    scan = scan_dependency_edges(repo)
    assert len(scan.edges) == 4
    assert all(edge.resolution.value == "UNRESOLVED" for edge in scan.edges)
    assert all("local target" in edge.note for edge in scan.edges)
    # Reconciliation must not use either root or cross-component declarations
    # to turn this incomplete local evidence into an external resolution.
    report = analyze_dependencies(repo)
    assert report["summary"]["AUTO_RESOLVED"] == 0
    assert report["dependency_reconciliation"]["summary"]["resolved_external_count"] == 0


@pytest.mark.parametrize("configuration", [
    'tool = "invalid"',
    '[tool]\nsetuptools = "invalid"',
    '[tool.setuptools]\npackage-dir = "invalid"',
    '[tool.setuptools.packages]\nfind = "invalid"',
    '[tool.setuptools.packages.find]\nwhere = "lib"',
    '[tool.setuptools.packages.find]\nwhere = 3',
])
def test_malformed_package_configuration_does_not_crash_or_invent_roots(tmp_path, configuration):
    repo = tmp_path / "repo"
    _fixture(repo, "import plugin.helper", "numpy==2")
    put(repo, "lib/plugin/helper.py")
    put(repo, "pyproject.toml", configuration)
    _git(repo, "add", "lib", "pyproject.toml")
    scan = scan_dependency_edges(repo)
    assert len(scan.edges) == 1
    assert scan.edges[0].resolution.value == "UNRESOLVED"
    assert scan.edges[0].resolved_path is None


def test_builtin_import_relative_or_opaque_level_is_not_absolute_authority(tmp_path):
    repo = tmp_path / "repo"
    _fixture(repo, '__import__("product.feature")\n'
             '__import__("product.feature", level=0)\n'
             '__import__("product.feature", level=1)\n'
             '__import__("product.feature", globals(), locals(), [], 1)\n'
             '__import__("product.feature", level=runtime_level)\n'
             '__import__("product.feature", *arguments)\n'
             '__import__("product.feature", **keywords)', "numpy==2")
    put(repo, "product/feature.py")
    _git(repo, "add", "product/feature.py")
    scan = scan_dependency_edges(repo)
    by_line = {edge.line: edge for edge in scan.edges}
    assert set(by_line) == set(range(1, 8))
    assert all(by_line[line].resolution.value == "RESOLVED" for line in (1, 2))
    assert all(by_line[line].resolution.value == "DYNAMIC" for line in range(3, 8))
    assert all(by_line[line].resolved_path is None for line in range(3, 8))
    report = analyze_dependencies(repo)
    assert report["summary"]["AUTO_RESOLVED"] == 2
    assert report["summary"]["REVIEW_REQUIRED"] == 5
