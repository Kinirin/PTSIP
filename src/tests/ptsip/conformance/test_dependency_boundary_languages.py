from __future__ import annotations

from pathlib import Path

from ptsip.conformance_engine import evaluate_conformance
from ptsip.inspection.dependencies_030 import scan_dependency_edges

from _conformance_support import commit, init_repo, write_artifact, write_profile


def test_go_source_product_to_development_tooling_is_nonconformant(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    init_repo(repo)
    (repo / "product").mkdir()
    (repo / "tools" / "check").mkdir(parents=True)
    (repo / "product" / "go.mod").write_text("module example.com/product\n\ngo 1.23\n", encoding="utf-8")
    (repo / "tools" / "go.mod").write_text("module example.com/tools\n\ngo 1.23\n", encoding="utf-8")
    (repo / "product" / "main.go").write_text(
        'package main\nimport "example.com/tools/check"\nfunc main() { check.Run() }\n',
        encoding="utf-8",
    )
    (repo / "tools" / "check" / "check.go").write_text(
        "package check\nfunc Run() {}\n",
        encoding="utf-8",
    )
    write_profile(repo, "product/go.mod", "tools/go.mod")
    artifact = tmp_path / "go-artifact.json"
    write_artifact(artifact, "product/main.go")
    commit(repo)

    scan = scan_dependency_edges(repo)
    edge = next(item for item in scan.edges if item.evidence_id.startswith("go:product/main.go"))
    assert edge.resolved_path == "tools/check/check.go"
    assert edge.phase.value == "RUNTIME"

    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "NON_CONFORMANT"
    assert any(item["rule_id"] == "PTSIP-DEP-001" for item in result.report["diagnostics"])


def test_dotnet_source_namespace_resolves_cross_project_dependency(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    init_repo(repo)
    (repo / "product").mkdir()
    (repo / "tools").mkdir()
    (repo / "product" / "Product.csproj").write_text(
        '<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><RootNamespace>Product.App</RootNamespace></PropertyGroup><ItemGroup><ProjectReference Include="../tools/Tools.csproj" /></ItemGroup></Project>',
        encoding="utf-8",
    )
    (repo / "tools" / "Tools.csproj").write_text(
        '<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><RootNamespace>Tools.Utility</RootNamespace></PropertyGroup></Project>',
        encoding="utf-8",
    )
    (repo / "product" / "Program.cs").write_text(
        "using Tools.Utility;\nnamespace Product.App;\n",
        encoding="utf-8",
    )
    (repo / "tools" / "Utility.cs").write_text("namespace Tools.Utility;\n", encoding="utf-8")
    write_profile(repo, "product/Product.csproj", "tools/Tools.csproj")
    artifact = tmp_path / "dotnet-artifact.json"
    write_artifact(artifact, "product/Program.cs")
    commit(repo)

    scan = scan_dependency_edges(repo)
    edge = next(item for item in scan.edges if item.evidence_id.startswith("dotnet-source:product/Program.cs"))
    assert edge.resolved_path == "tools/Tools.csproj"
    assert edge.phase.value == "RUNTIME"

    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "NON_CONFORMANT"
    assert any(
        item["rule_id"] == "PTSIP-DEP-001" and item["outcome_effect"] == "NON_CONFORMANT"
        for item in result.report["diagnostics"]
    )
