from __future__ import annotations

import subprocess
from pathlib import Path

import pytest

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


def _dotnet_repo(
    tmp_path: Path,
    *,
    project_reference: bool,
    package_reference: bool = False,
) -> Path:
    repo = tmp_path / "dotnet"
    _init(repo)
    (repo / "product").mkdir()
    (repo / "tools").mkdir()
    refs = []
    if project_reference:
        refs.append('<ProjectReference Include="../tools/Tools.csproj" />')
    if package_reference:
        refs.append('<PackageReference Include="Tools.Utility" Version="1.0.0" />')
    item_group = f"<ItemGroup>{''.join(refs)}</ItemGroup>" if refs else ""
    (repo / "product" / "Product.csproj").write_text(
        f'<Project Sdk="Microsoft.NET.Sdk">{item_group}</Project>',
        encoding="utf-8",
    )
    (repo / "tools" / "Tools.csproj").write_text(
        '<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><RootNamespace>Tools.Utility</RootNamespace></PropertyGroup></Project>',
        encoding="utf-8",
    )
    (repo / "product" / "Program.cs").write_text("using Tools.Utility;\n", encoding="utf-8")
    (repo / "tools" / "Utility.cs").write_text(
        "namespace Tools.Utility;\n",
        encoding="utf-8",
    )
    _commit(repo)
    return repo


def test_dotnet_project_reference_corroborates_namespace_resolution(tmp_path: Path) -> None:
    scan = scan_dependency_edges(_dotnet_repo(tmp_path, project_reference=True))
    edge = next(
        item
        for item in scan.edges
        if item.evidence_id.startswith("dotnet-source:product/Program.cs")
    )
    assert edge.resolution.value == "RESOLVED"
    assert edge.resolved_path == "tools/Tools.csproj"


@pytest.mark.parametrize("package_reference", [False, True])
def test_dotnet_namespace_without_project_reference_is_not_local_identity(
    tmp_path: Path,
    package_reference: bool,
) -> None:
    scan = scan_dependency_edges(
        _dotnet_repo(
            tmp_path,
            project_reference=False,
            package_reference=package_reference,
        )
    )
    edge = next(
        item
        for item in scan.edges
        if item.evidence_id.startswith("dotnet-source:product/Program.cs")
    )
    assert edge.resolution.value == "UNRESOLVED"
    assert edge.resolved_path is None
