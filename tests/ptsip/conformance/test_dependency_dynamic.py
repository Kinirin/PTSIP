from ptsip.inspection.dependencies import scan_dependency_edges
from test_dependency_reconciliation import _fixture, _git


def test_literal_and_bounded_dynamic_targets_resolve_without_executing_consumer(tmp_path):
    repo = tmp_path / "repo"
    _fixture(repo, 'import importlib as loader\nfrom importlib import import_module as load\n'
             'loader.import_module("product.feature")\nload("product.feature")\n'
             'for name in ("feature", "missing"):\n    loader.import_module("product." + name)\n'
             'loader.import_module(runtime_name)\n', "numpy==2")
    (repo / "product/feature.py").write_text('raise RuntimeError("MUST NOT EXECUTE")', encoding="utf-8")
    _git(repo, "add", "product/feature.py")
    edges = [edge for edge in scan_dependency_edges(repo).edges if edge.edge_type.value == "LOADS"]
    assert len(edges) == 5
    assert sum(edge.resolution.value == "RESOLVED" for edge in edges) == 3
    assert sum(edge.resolution.value == "DYNAMIC" for edge in edges) == 1
    assert sum(edge.note.startswith("BOUNDED_DYNAMIC_IMPORT") for edge in edges) == 2
    assert any(edge.target == "product.missing" and edge.resolution.value == "UNRESOLVED" for edge in edges)


def test_reassignment_and_deferred_closure_do_not_become_bounded_authority(tmp_path):
    repo = tmp_path / "repo"
    _fixture(repo, 'import importlib\nfor name in ("product.a", "product.b"):\n'
             '    name = user_input\n    importlib.import_module(name)\n'
             'for name in ("product.a",):\n    def later():\n        importlib.import_module(name)\n', "numpy==2")
    edges = [edge for edge in scan_dependency_edges(repo).edges if edge.edge_type.value == "LOADS"]
    assert len(edges) == 2 and all(edge.resolution.value == "DYNAMIC" for edge in edges)
