from __future__ import annotations

import json
import hashlib
import subprocess
from pathlib import Path

from ptsip.conformance_engine import evaluate_conformance
from ptsip.inspection.dependencies_030 import scan_dependency_edges
from ptsip.repository.snapshot import capture_snapshot


SPEC_REVISION = "ccee8cd5e26e92d31a2b93a86157c03d9b796b2c"


def _git(repo: Path, *args: str) -> str:
    result = subprocess.run(
        ["git", "-C", str(repo), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    return result.stdout.strip()


def _init(repo: Path) -> None:
    repo.mkdir(parents=True, exist_ok=True)
    _git(repo, "init")
    _git(repo, "config", "user.email", "ptsip-test@example.invalid")
    _git(repo, "config", "user.name", "PTSIP Test")


def _commit(repo: Path) -> str:
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")
    return _git(repo, "rev-parse", "HEAD")


def _workflow(repo: Path, name: str, scope: str) -> None:
    path = repo / ".github" / "workflows"
    path.mkdir(parents=True, exist_ok=True)
    (path / name).write_text(
        f"name: {name}\non:\n  push:\n    paths:\n      - '{scope}'\njobs:\n  release:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo release\n",
        encoding="utf-8",
    )


def _profile(repo: Path, product_manifest: str, tool_manifest: str) -> None:
    (repo / "ptsip.yaml").write_text(
        f"""ptsip:
  version: "0.3.4-draft"
  specification:
    source: "https://github.com/kwaksinwoo01/ptsip"
    revision: "{SPEC_REVISION}"
components:
  - id: product
    classification: PRODUCT
    include: ["product/**"]
    purpose: product_runtime
    manifests: ["{product_manifest}"]
    release_owner: product-release
    compatibility_owner: product-compat
  - id: tools
    classification: TOOLCHAIN
    include: ["tools/**"]
    purpose: development_tooling
    manifests: ["{tool_manifest}"]
    release_owner: toolchain-release
    compatibility_owner: toolchain-compat
policies:
  product_to_toolchain_runtime_dependency: deny
  toolchain_in_product_package: deny
  independent_build_resolution: required
""",
        encoding="utf-8",
    )
    _workflow(repo, "product-release.yml", "product/**")
    _workflow(repo, "toolchain-release.yml", "tools/**")


def _artifact(path: Path, product_path: str) -> None:
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
                "evidence_ids": ["artifact:remaining:product-dist"],
            }
        ),
        encoding="utf-8",
    )


def _bind_artifact(path: Path, repo: Path, revision: str | None = None) -> None:
    revision = revision or _git(repo, "rev-parse", "HEAD")
    Path(str(path) + ".binding.json").write_text(
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


def _python_clean(repo: Path) -> Path:
    (repo / "product").mkdir(exist_ok=True)
    (repo / "tools").mkdir(exist_ok=True)
    (repo / "product" / "app.py").write_text("VALUE = 1\n", encoding="utf-8")
    (repo / "tools" / "check.py").write_text("VALUE = 2\n", encoding="utf-8")
    (repo / "product" / "requirements.txt").write_text("\n", encoding="utf-8")
    (repo / "tools" / "requirements.txt").write_text("\n", encoding="utf-8")
    _profile(repo, "product/requirements.txt", "tools/requirements.txt")
    artifact = repo.parent / "artifact.json"
    _artifact(artifact, "product/app.py")
    return artifact


def test_go_source_product_to_toolchain_is_nonconformant(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init(repo)
    (repo / "product").mkdir()
    (repo / "tools" / "check").mkdir(parents=True)
    (repo / "product" / "go.mod").write_text("module example.com/product\n\ngo 1.23\n", encoding="utf-8")
    (repo / "tools" / "go.mod").write_text("module example.com/tools\n\ngo 1.23\n", encoding="utf-8")
    (repo / "product" / "main.go").write_text('package main\nimport "example.com/tools/check"\nfunc main() { check.Run() }\n', encoding="utf-8")
    (repo / "tools" / "check" / "check.go").write_text("package check\nfunc Run() {}\n", encoding="utf-8")
    _profile(repo, "product/go.mod", "tools/go.mod")
    artifact = tmp_path / "go-artifact.json"
    _artifact(artifact, "product/main.go")
    _commit(repo)

    scan = scan_dependency_edges(repo)
    edge = next(item for item in scan.edges if item.evidence_id.startswith("go:product/main.go"))
    assert edge.resolved_path == "tools/check/check.go"
    assert edge.phase.value == "RUNTIME"

    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "NON_CONFORMANT"
    assert any(item["rule_id"] == "PTSIP-DEP-001" for item in result.report["diagnostics"])


def test_dotnet_source_namespace_resolves_cross_project_dependency(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init(repo)
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
    (repo / "product" / "Program.cs").write_text("using Tools.Utility;\nnamespace Product.App;\n", encoding="utf-8")
    (repo / "tools" / "Utility.cs").write_text("namespace Tools.Utility;\n", encoding="utf-8")
    _profile(repo, "product/Product.csproj", "tools/Tools.csproj")
    artifact = tmp_path / "dotnet-artifact.json"
    _artifact(artifact, "product/Program.cs")
    _commit(repo)

    scan = scan_dependency_edges(repo)
    edge = next(item for item in scan.edges if item.evidence_id.startswith("dotnet-source:product/Program.cs"))
    assert edge.resolved_path == "tools/Tools.csproj"
    assert edge.phase.value == "RUNTIME"

    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "NON_CONFORMANT"
    assert any(item["rule_id"] == "PTSIP-DEP-001" and item["outcome_effect"] == "NON_CONFORMANT" for item in result.report["diagnostics"])


def test_agent_decision_conflict_never_overrides_profile(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init(repo)
    artifact = _python_clean(repo)
    _commit(repo)
    decision = tmp_path / "decision.json"
    decision.write_text(
        json.dumps(
            {
                "component_id": "product",
                "status": "RESOLVED",
                "classification": "TOOLCHAIN",
                "origin": "AGENT",
                "confidence": 0.9,
                "evidence_ids": ["agent:evidence:1"],
                "rationale": "Review fixture",
                "counter_evidence": [],
            }
        ),
        encoding="utf-8",
    )

    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact], agent_decision_paths=[decision])
    assert result.outcome == "INCOMPLETE"
    assert result.report["agent_decisions"]["affects_declared_classification"] is False
    assert any(item["id"].startswith("agent-decision:declaration-conflict:product") for item in result.report["coverage"]["blocking_gaps"])


def test_stale_external_evidence_is_blocking(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init(repo)
    artifact = _python_clean(repo)
    _commit(repo)
    external = tmp_path / "external.json"
    external.write_text(
        json.dumps(
            {
                "format": "ptsip-external-evidence/v1",
                "producer": {"id": "fixture-validator", "version": "1"},
                "subject": {"repository": str(repo.resolve()), "revision": "deadbeef"},
                "evidence": [],
            }
        ),
        encoding="utf-8",
    )

    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact], external_evidence_paths=[external])
    assert result.outcome == "INCOMPLETE"
    assert result.report["evaluators"]["external_evidence_import"]["status"] == "BLOCKED"
    assert any(item["id"].startswith("external-evidence:invalid:") for item in result.report["coverage"]["blocking_gaps"])


def test_trusted_external_evidence_can_resolve_native_target_uncertainty(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init(repo)
    artifact = _python_clean(repo)
    (repo / "product" / "app.py").write_text("import vendor_module\nVALUE = 1\n", encoding="utf-8")
    revision = _commit(repo)
    _bind_artifact(artifact, repo, revision)
    external = tmp_path / "external.json"
    external.write_text(
        json.dumps(
            {
                "format": "ptsip-external-evidence/v1",
                "producer": {"id": "fixture-validator", "version": "1"},
                "subject": {"repository": str(repo.resolve()), "revision": revision},
                "evidence": [
                    {
                        "kind": "dependency",
                        "evidence_id": "vendor-module-resolution",
                        "source": "product/app.py",
                        "target": "vendor_module",
                        "relationship_type": "IMPORTS",
                        "phase": "RUNTIME",
                        "resolution": "EXTERNAL",
                        "target_scope": "EXTERNAL_DEPENDENCY",
                        "provenance": "OBSERVED"
                    }
                ],
            }
        ),
        encoding="utf-8",
    )

    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact], external_evidence_paths=[external])
    assert result.outcome == "CONFORMANT", result.report["coverage"]
    assert result.report["external_evidence"]["edge_count"] == 1
    assert result.report["audit"]["status"] == "PASS"
    assert not any(item["id"].startswith("dependency-target:python:product/app.py") for item in result.report["coverage"]["blocking_gaps"])


def test_clean_supported_report_passes_contract_audit(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init(repo)
    artifact = _python_clean(repo)
    revision = _commit(repo)
    _bind_artifact(artifact, repo, revision)

    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "CONFORMANT"
    assert result.report["audit"]["status"] == "PASS"
    assert result.report["audit"]["problem_count"] == 0
