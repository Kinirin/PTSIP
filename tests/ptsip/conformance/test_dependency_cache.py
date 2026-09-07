import json

from ptsip.inspection.dependencies import scan_dependency_edges
from test_dependency_reconciliation import _fixture, _git


def test_source_cache_reuses_unaffected_evidence_and_invalidates_semantic_inputs(tmp_path, monkeypatch):
    repo = tmp_path / "repo"
    monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))
    _fixture(repo, "import numpy", "numpy==2")
    (repo / "product/other.py").write_text("import os\n", encoding="utf-8")
    _git(repo, "add", "product/other.py")
    cold = scan_dependency_edges(repo)
    assert cold.cache["recomputed"] == 2 and cold.cache["hits"] == 0
    warm = scan_dependency_edges(repo)
    assert warm.edges == cold.edges and warm.cache["hits"] == 2
    (repo / "product/app.py").write_text("import numpy\nimport sys\n", encoding="utf-8")
    changed = scan_dependency_edges(repo)
    assert changed.cache["recomputed"] == 1 and changed.cache["hits"] == 1
    (repo / "install/runtime-requirements.txt").write_text("Pillow==12\n", encoding="utf-8")
    assert scan_dependency_edges(repo).cache["recomputed"] == 2
    (repo / "ptsip.yaml").write_text("invalid: true\n", encoding="utf-8")
    assert scan_dependency_edges(repo).cache["recomputed"] == 2


def test_corrupt_or_incomplete_cache_recomputes_and_in_repo_state_is_disabled(tmp_path, monkeypatch):
    repo = tmp_path / "repo"
    state = tmp_path / "state"
    monkeypatch.setenv("PTSIP_HOME", str(state))
    _fixture(repo, "__import__(unknown)", "numpy==2")
    baseline = scan_dependency_edges(repo)
    path = next(state.rglob("*.json"))
    path.write_text(json.dumps({"key": "bad", "value": {"edges": []}}), encoding="utf-8")
    after = scan_dependency_edges(repo)
    assert after.edges == baseline.edges and after.cache["invalid"] == 1
    monkeypatch.setenv("PTSIP_HOME", str(repo / ".ptsip"))
    disabled = scan_dependency_edges(repo)
    assert disabled.cache["enabled"] is False
    assert disabled.edges == baseline.edges
    assert not (repo / ".ptsip").exists()
