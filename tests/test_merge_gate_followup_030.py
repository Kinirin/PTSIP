from __future__ import annotations

import subprocess
from pathlib import Path

from ptsip.conformance_engine import _externally_supplemented_native_ids
from ptsip.inspection.dependencies import DependencyScan
from ptsip.inspection.dependencies_030 import scan_dependency_edges
from ptsip.model import (
    DependencyEdge,
    DependencyPhase,
    EdgeType,
    EvidenceNodeScope,
    EvidenceProvenance,
    ResolutionStatus,
)


def _git(repo: Path, *args: str) -> None:
    subprocess.run(
        ["git", "-C", str(repo), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )


def _init(repo: Path) -> None:
    repo.mkdir(parents=True)
    _git(repo, "init")
    _git(repo, "config", "user.email", "ptsip-test@example.invalid")
    _git(repo, "config", "user.name", "PTSIP Test")


def _commit(repo: Path) -> None:
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")


def test_external_sentinel_cannot_clear_native_dynamic_uncertainty() -> None:
    native_edge = DependencyEdge(
        evidence_id="javascript-dynamic:product/app.js:1:import",
        source="product/app.js",
        target="<dynamic-import>",
        edge_type=EdgeType.LOADS,
        phase=DependencyPhase.UNKNOWN,
        resolution=ResolutionStatus.DYNAMIC,
        target_scope=EvidenceNodeScope.UNRESOLVED_TARGET,
        provenance=EvidenceProvenance.OBSERVED,
        adapter="javascript-typescript",
    )
    external_edge = DependencyEdge(
        evidence_id="external:fixture:dynamic-sentinel",
        source="product/app.js",
        target="<dynamic-import>",
        edge_type=EdgeType.LOADS,
        phase=DependencyPhase.UNKNOWN,
        resolution=ResolutionStatus.EXTERNAL,
        target_scope=EvidenceNodeScope.EXTERNAL_DEPENDENCY,
        provenance=EvidenceProvenance.OBSERVED,
        adapter="external:fixture",
    )
    native = DependencyScan((native_edge,), (), ("javascript-typescript",))

    supplemented = _externally_supplemented_native_ids(native, (external_edge,))

    assert native_edge.evidence_id not in supplemented


def test_template_interpolation_literal_import_is_observed(tmp_path: Path) -> None:
    repo = tmp_path / "js-template"
    _init(repo)
    (repo / "real.js").write_text("export const value = 1;\n", encoding="utf-8")
    (repo / "index.js").write_text(
        "const value = `${import('./real.js')}`;\n",
        encoding="utf-8",
    )
    _commit(repo)

    edges = [item for item in scan_dependency_edges(repo).edges if item.source == "index.js"]

    edge = next(item for item in edges if item.target == "./real.js")
    assert edge.edge_type == EdgeType.LOADS
    assert edge.resolution == ResolutionStatus.RESOLVED
    assert edge.resolved_path == "real.js"


def test_template_interpolation_nonliteral_import_remains_dynamic(tmp_path: Path) -> None:
    repo = tmp_path / "js-template-dynamic"
    _init(repo)
    (repo / "index.js").write_text(
        "const target = './real.js';\nconst value = `${import(target)}`;\n",
        encoding="utf-8",
    )
    (repo / "real.js").write_text("export const value = 1;\n", encoding="utf-8")
    _commit(repo)

    edges = [item for item in scan_dependency_edges(repo).edges if item.source == "index.js"]

    edge = next(item for item in edges if item.evidence_id.startswith("javascript-dynamic:index.js"))
    assert edge.edge_type == EdgeType.LOADS
    assert edge.resolution == ResolutionStatus.DYNAMIC
    assert edge.target_scope == EvidenceNodeScope.UNRESOLVED_TARGET
