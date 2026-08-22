from __future__ import annotations

import hashlib
import json
import subprocess
from pathlib import Path

import pytest

import ptsip.conformance_engine as engine
from ptsip.build_resolution import evaluate_independent_build_resolution
from ptsip.conformance_engine import evaluate_conformance
from ptsip.constants import SPEC_REVISION
from ptsip.inspection.dependencies_030 import scan_dependency_edges
from ptsip.lifecycle_evidence import evaluate_lifecycle_evidence
from ptsip.repository.snapshot import capture_snapshot
from ptsip.validation.components import partition_components


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


def _components() -> list[dict[str, object]]:
    return [
        {
            "id": "product",
            "classification": "PRODUCT",
            "include": ["product/**"],
            "purpose": "product_runtime",
            "manifests": ["product/requirements.txt"],
            "release_owner": "product-release",
            "compatibility_owner": "product-compat",
        },
        {
            "id": "tools",
            "classification": "DEVELOPMENT_TOOLING",
            "include": ["tools/**"],
            "purpose": "development_tooling",
            "manifests": ["tools/requirements.txt"],
            "release_owner": "tool-release",
            "compatibility_owner": "tool-compat",
        },
    ]


def _profile(repo: Path, components: list[dict[str, object]]) -> None:
    lines = [
        "ptsip:",
        '  version: "0.3.6-draft"',
        "  specification:",
        '    source: "https://github.com/Kinirin/PTSIP"',
        f'    revision: "{SPEC_REVISION}"',
        "responsibility_map:",
        "  mode: explicit",
        "components:",
    ]
    for component in components:
        lines.extend(
            [
                f"  - id: {component['id']}",
                f"    classification: {component['classification']}",
                "    include:",
                *[f'      - "{item}"' for item in component["include"]],
                f"    purpose: {component['purpose']}",
                "    manifests:",
                *[f'      - "{item}"' for item in component["manifests"]],
                f"    release_owner: {component['release_owner']}",
                f"    compatibility_owner: {component['compatibility_owner']}",
            ]
        )
    lines.extend(
        [
            "policies:",
            "  product_to_nonproduct_runtime_dependency: deny",
            "  nonproduct_in_product_package: deny",
            "  independent_build_resolution: required",
        ]
    )
    (repo / "ptsip.yaml").write_text("\n".join(lines) + "\n", encoding="utf-8")


def _workflow(repo: Path, name: str, *, paths: str | None, release: bool = True) -> None:
    directory = repo / ".github" / "workflows"
    directory.mkdir(parents=True, exist_ok=True)
    scope = "" if paths is None else f"    paths:\n      - '{paths}'\n"
    action = "echo release" if release else "python -m pytest"
    job = "release" if release else "test"
    (directory / name).write_text(
        f"name: {name}\non:\n  push:\n{scope}jobs:\n  {job}:\n    runs-on: ubuntu-latest\n    steps:\n      - run: {action}\n",
        encoding="utf-8",
    )


def _artifact(path: Path, product_path: str = "product/app.py") -> None:
    path.write_text(
        json.dumps(
            {
                "format": "ptsip-artifact-evidence/v1",
                "artifact_id": "product-dist",
                "classification": "PRODUCT",
                "producer_component": "tools",
                "artifact_type": "fixture",
                "shipping_scope": "product-distribution",
                "contents": {"paths": [product_path], "components": ["product"], "complete": True},
                "derivation": [{"relation": "GENERATES", "source": "tools"}],
                "provenance": "OBSERVED",
                "evidence_ids": ["artifact:merge-gate:product-dist"],
            }
        ),
        encoding="utf-8",
    )


def _bind(path: Path, repo: Path, revision: str) -> Path:
    binding = Path(str(path) + ".binding.json")
    binding.write_text(
        json.dumps(
            {
                "format": "ptsip-artifact-evidence-binding/v1",
                "artifact_sha256": hashlib.sha256(path.read_bytes()).hexdigest(),
                "subject": {
                    "repository": str(repo.resolve()),
                    "revision": revision,
                    "tracked_content_sha256": capture_snapshot(repo).tracked_content_fingerprint,
                },
            }
        ),
        encoding="utf-8",
    )
    return binding


def _clean_repo(tmp_path: Path, product_source: str = "VALUE = 1\n") -> tuple[Path, Path, str]:
    repo = tmp_path / "repo"
    _init(repo)
    (repo / "product").mkdir()
    (repo / "tools").mkdir()
    (repo / "product" / "app.py").write_text(product_source, encoding="utf-8")
    (repo / "tools" / "check.py").write_text("VALUE = 2\n", encoding="utf-8")
    (repo / "product" / "requirements.txt").write_text("\n", encoding="utf-8")
    (repo / "tools" / "requirements.txt").write_text("\n", encoding="utf-8")
    _profile(repo, _components())
    _workflow(repo, "product-release.yml", paths="product/**")
    _workflow(repo, "tool-release.yml", paths="tools/**")
    artifact = tmp_path / "artifact.json"
    _artifact(artifact)
    revision = _commit(repo)
    _bind(artifact, repo, revision)
    return repo, artifact, revision


def _external(path: Path, repo: Path, revision: str, evidence: list[dict[str, object]], repository: str | None = None) -> None:
    path.write_text(
        json.dumps(
            {
                "format": "ptsip-external-evidence/v1",
                "producer": {"id": "fixture-validator", "version": "1"},
                "subject": {"repository": repository or str(repo.resolve()), "revision": revision},
                "evidence": evidence,
            }
        ),
        encoding="utf-8",
    )


def _gaps(result) -> list[dict[str, object]]:
    return result.report["coverage"]["blocking_gaps"]


def test_final_snapshot_covers_late_lifecycle_evaluation(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    repo, artifact, _revision = _clean_repo(tmp_path)
    original = engine.evaluate_lifecycle_evidence

    def mutate_during_lifecycle(*args, **kwargs):
        result = original(*args, **kwargs)
        (repo / "product" / "app.py").write_text("VALUE = 99\n", encoding="utf-8")
        return result

    monkeypatch.setattr(engine, "evaluate_lifecycle_evidence", mutate_during_lifecycle)
    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "INCOMPLETE"
    assert result.report["snapshot"]["comparison"]["stable"] is False
    assert any(item["id"] == "snapshot:invalidated" for item in _gaps(result))


@pytest.mark.parametrize("suffix", [".rs", ".java"])
def test_product_owned_unsupported_source_is_incomplete(tmp_path: Path, suffix: str) -> None:
    repo, artifact, _revision = _clean_repo(tmp_path)
    source = repo / "product" / f"main{suffix}"
    source.write_text("source\n", encoding="utf-8")
    revision = _commit(repo)
    _bind(artifact, repo, revision)
    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "INCOMPLETE"
    assert any(item["id"] == "language-coverage:unsupported-mandatory-source" for item in _gaps(result))


def test_unrelated_markdown_does_not_block_source_coverage(tmp_path: Path) -> None:
    repo, artifact, _revision = _clean_repo(tmp_path)
    (repo / "notes.md").write_text("documentation\n", encoding="utf-8")
    revision = _commit(repo)
    _bind(artifact, repo, revision)
    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "CONFORMANT", _gaps(result)


def test_scoped_release_workflows_are_sufficient(tmp_path: Path) -> None:
    repo, _artifact_path, _revision = _clean_repo(tmp_path)
    components = _components()
    result = evaluate_lifecycle_evidence(repo, components, partition_components(repo, components))
    assert result.status == "RAN"


def test_unscoped_release_workflow_is_observation_not_lifecycle_authority(tmp_path: Path) -> None:
    repo, _artifact_path, _revision = _clean_repo(tmp_path)
    _workflow(repo, "global-release.yml", paths=None)
    _commit(repo)
    components = _components()
    result = evaluate_lifecycle_evidence(repo, components, partition_components(repo, components))
    assert result.status == "RAN"
    assert not result.blocking_gaps
    workflow = next(item for item in result.workflows if item.path.endswith("global-release.yml"))
    assert workflow.trigger_paths == ()
    assert workflow.scope_complete is False
    assert workflow.reason is not None
    assert "not a classification failure" in workflow.reason


def test_global_non_release_ci_is_not_lifecycle_blocker(tmp_path: Path) -> None:
    repo, _artifact_path, _revision = _clean_repo(tmp_path)
    _workflow(repo, "ci.yml", paths=None, release=False)
    _commit(repo)
    components = _components()
    result = evaluate_lifecycle_evidence(repo, components, partition_components(repo, components))
    assert result.status == "RAN"


def _dotnet_repo(tmp_path: Path, *, project_reference: bool, package_reference: bool = False) -> Path:
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
    (repo / "product" / "Product.csproj").write_text(f'<Project Sdk="Microsoft.NET.Sdk">{item_group}</Project>', encoding="utf-8")
    (repo / "tools" / "Tools.csproj").write_text(
        '<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><RootNamespace>Tools.Utility</RootNamespace></PropertyGroup></Project>',
        encoding="utf-8",
    )
    (repo / "product" / "Program.cs").write_text("using Tools.Utility;\n", encoding="utf-8")
    (repo / "tools" / "Utility.cs").write_text("namespace Tools.Utility;\n", encoding="utf-8")
    _commit(repo)
    return repo


def test_dotnet_project_reference_corroborates_namespace_resolution(tmp_path: Path) -> None:
    scan = scan_dependency_edges(_dotnet_repo(tmp_path, project_reference=True))
    edge = next(item for item in scan.edges if item.evidence_id.startswith("dotnet-source:product/Program.cs"))
    assert edge.resolution.value == "RESOLVED"
    assert edge.resolved_path == "tools/Tools.csproj"


@pytest.mark.parametrize("package_reference", [False, True])
def test_dotnet_namespace_without_project_reference_is_not_local_identity(tmp_path: Path, package_reference: bool) -> None:
    scan = scan_dependency_edges(_dotnet_repo(tmp_path, project_reference=False, package_reference=package_reference))
    edge = next(item for item in scan.edges if item.evidence_id.startswith("dotnet-source:product/Program.cs"))
    assert edge.resolution.value == "UNRESOLVED"
    assert edge.resolved_path is None


@pytest.mark.parametrize("suffix", [".cs", ".mts", ".cts"])
def test_unassigned_supported_source_is_blocking(tmp_path: Path, suffix: str) -> None:
    repo, artifact, _revision = _clean_repo(tmp_path)
    (repo / f"unassigned{suffix}").write_text("// source\n", encoding="utf-8")
    revision = _commit(repo)
    _bind(artifact, repo, revision)
    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "INCOMPLETE"
    gap = next(item for item in _gaps(result) if item["id"] == "component-ownership:unassigned-relevant-files")
    assert f"unassigned:unassigned{suffix}" in gap["evidence_ids"]


@pytest.mark.parametrize("provenance", ["INFERRED", "DECLARED"])
def test_untrusted_external_evidence_cannot_clear_native_uncertainty(tmp_path: Path, provenance: str) -> None:
    repo, artifact, revision = _clean_repo(tmp_path, "import vendor_module\n")
    external = tmp_path / "external.json"
    _external(
        external,
        repo,
        revision,
        [{
            "kind": "dependency", "evidence_id": "vendor", "source": "product/app.py", "target": "vendor_module",
            "relationship_type": "IMPORTS", "phase": "RUNTIME", "resolution": "EXTERNAL",
            "target_scope": "EXTERNAL_DEPENDENCY", "provenance": provenance,
        }],
    )
    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact], external_evidence_paths=[external])
    assert result.outcome == "INCOMPLETE"
    assert any(item["id"].startswith("dependency-target:python:product/app.py") for item in _gaps(result))


def test_external_repository_mismatch_is_blocking(tmp_path: Path) -> None:
    repo, artifact, revision = _clean_repo(tmp_path)
    external = tmp_path / "external.json"
    _external(external, repo, revision, [], repository="other/repository")
    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact], external_evidence_paths=[external])
    assert result.outcome == "INCOMPLETE"
    assert any(item["id"].startswith("external-evidence:invalid:") for item in _gaps(result))


def test_external_resolution_scope_pair_is_semantically_validated(tmp_path: Path) -> None:
    repo, artifact, revision = _clean_repo(tmp_path)
    external = tmp_path / "external.json"
    _external(
        external,
        repo,
        revision,
        [{
            "kind": "dependency", "evidence_id": "bad", "source": "product/app.py", "target": "vendor",
            "relationship_type": "IMPORTS", "phase": "RUNTIME", "resolution": "RESOLVED",
            "target_scope": "EXTERNAL_DEPENDENCY", "provenance": "OBSERVED",
        }],
    )
    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact], external_evidence_paths=[external])
    assert result.outcome == "INCOMPLETE"
    assert any("RESOLVED evidence requires PROJECT_COMPONENT" in item["message"] for item in _gaps(result))


def test_external_conflict_with_native_resolution_is_blocking(tmp_path: Path) -> None:
    repo, artifact, revision = _clean_repo(tmp_path, "import tools.check\n")
    external = tmp_path / "external.json"
    _external(
        external,
        repo,
        revision,
        [{
            "kind": "dependency", "evidence_id": "conflict", "source": "product/app.py", "target": "tools.check",
            "relationship_type": "IMPORTS", "phase": "UNKNOWN", "resolution": "RESOLVED",
            "target_scope": "PROJECT_COMPONENT", "resolved_path": "product/app.py", "provenance": "OBSERVED",
        }],
    )
    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact], external_evidence_paths=[external])
    assert any(item["id"].startswith("external-evidence:conflict:") for item in _gaps(result))


def test_exact_artifact_revision_binding_is_usable(tmp_path: Path) -> None:
    repo, artifact, _revision = _clean_repo(tmp_path)
    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "CONFORMANT"
    assert result.report["evaluators"]["artifact_snapshot_binding"]["status"] == "RAN"


def test_artifact_binding_rejects_dirty_tracked_content_drift(tmp_path: Path) -> None:
    repo, artifact, _revision = _clean_repo(tmp_path)
    (repo / "product" / "app.py").write_text("VALUE = 42\n", encoding="utf-8")
    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "INCOMPLETE"
    assert any("tracked-content fingerprint" in item["message"] for item in _gaps(result))


@pytest.mark.parametrize("binding_state", ["stale", "missing"])
def test_stale_or_missing_artifact_binding_is_incomplete(tmp_path: Path, binding_state: str) -> None:
    repo, artifact, _revision = _clean_repo(tmp_path)
    binding = Path(str(artifact) + ".binding.json")
    if binding_state == "stale":
        _bind(artifact, repo, "deadbeef")
    else:
        binding.unlink()
    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "INCOMPLETE"
    assert result.report["evaluators"]["artifact_snapshot_binding"]["status"] == "BLOCKED"


def _build_repo(tmp_path: Path) -> tuple[Path, list[dict[str, object]]]:
    repo = tmp_path / "build"
    _init(repo)
    (repo / "product").mkdir()
    (repo / "tools").mkdir()
    (repo / "product" / "app.py").write_text("VALUE = 1\n", encoding="utf-8")
    (repo / "tools" / "check.py").write_text("VALUE = 2\n", encoding="utf-8")
    (repo / "product" / "requirements.txt").write_text("\n", encoding="utf-8")
    (repo / "tools" / "requirements.txt").write_text("\n", encoding="utf-8")
    components = _components()
    _commit(repo)
    return repo, components


def test_owned_build_manifests_are_usable(tmp_path: Path) -> None:
    repo, components = _build_repo(tmp_path)
    result = evaluate_independent_build_resolution(repo, components, partition_components(repo, components))
    assert result.status == "RAN"


def test_cross_plane_owned_manifest_is_not_usable_for_declaring_component(tmp_path: Path) -> None:
    repo, components = _build_repo(tmp_path)
    components[0]["manifests"] = ["tools/requirements.txt"]
    result = evaluate_independent_build_resolution(repo, components, partition_components(repo, components))
    assert result.status == "BLOCKED"
    assert any("owned by component 'tools'" in item["message"] for item in result.blocking_gaps)


def test_unassigned_manifest_is_not_build_isolation_evidence(tmp_path: Path) -> None:
    repo, components = _build_repo(tmp_path)
    (repo / "requirements.txt").write_text("\n", encoding="utf-8")
    _commit(repo)
    components[0]["manifests"] = ["requirements.txt"]
    result = evaluate_independent_build_resolution(repo, components, partition_components(repo, components))
    assert result.status == "BLOCKED"
    assert any("outside declared component ownership" in item["message"] for item in result.blocking_gaps)


def test_go_comment_import_is_ignored_but_real_import_remains(tmp_path: Path) -> None:
    repo = tmp_path / "go"
    _init(repo)
    (repo / "go.mod").write_text("module example.com/product\n", encoding="utf-8")
    (repo / "main.go").write_text('package main\n/*\nimport "example.com/fake"\n*/\nimport "fmt"\n', encoding="utf-8")
    _commit(repo)
    edges = [item for item in scan_dependency_edges(repo).edges if item.adapter == "go-source"]
    assert [item.target for item in edges] == ["fmt"]


def test_javascript_comments_and_strings_do_not_create_edges(tmp_path: Path) -> None:
    repo = tmp_path / "js"
    _init(repo)
    (repo / "package.json").write_text(json.dumps({"name": "fixture"}), encoding="utf-8")
    (repo / "real.js").write_text("export const value = 1;\n", encoding="utf-8")
    (repo / "index.js").write_text(
        "// require('./comment')\n/* import('./block') */\nconst text = \"require('./string')\";\nconst real = require('./real');\n",
        encoding="utf-8",
    )
    _commit(repo)
    edges = [item for item in scan_dependency_edges(repo).edges if item.source == "index.js"]
    assert [item.target for item in edges] == ["./real"]


def test_csharp_comments_and_strings_do_not_create_using_edges(tmp_path: Path) -> None:
    repo = tmp_path / "cs"
    _init(repo)
    (repo / "App.csproj").write_text('<Project Sdk="Microsoft.NET.Sdk" />', encoding="utf-8")
    (repo / "Program.cs").write_text(
        '// using Fake.Line;\n/* using Fake.Block; */\nvar text = "using Fake.String;";\nusing System.Text;\n',
        encoding="utf-8",
    )
    _commit(repo)
    edges = [item for item in scan_dependency_edges(repo).edges if item.source == "Program.cs"]
    assert [item.target for item in edges] == ["System.Text"]
