from __future__ import annotations

import json
import hashlib
import subprocess
from pathlib import Path

from ptsip.build_resolution import evaluate_independent_build_resolution
from ptsip.conformance_engine import evaluate_conformance
from ptsip.inspection.dependencies_030 import scan_dependency_edges
from ptsip.lifecycle_evidence import evaluate_lifecycle_evidence
from ptsip.validation.components import partition_components
from ptsip.repository.snapshot import capture_snapshot


SPEC_REVISION = "afba3531e23d96c21b7216e49614b839158ca7d5"


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


def _release_workflow(repo: Path, filename: str, path_scope: str | None) -> None:
    workflows = repo / ".github" / "workflows"
    workflows.mkdir(parents=True, exist_ok=True)
    if path_scope is None:
        on_block = "on:\n  push:\n    branches: [main]\n"
    else:
        on_block = f"on:\n  push:\n    paths:\n      - '{path_scope}'\n"
    (workflows / filename).write_text(
        f"name: {filename}\n{on_block}jobs:\n  release:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo release\n",
        encoding="utf-8",
    )


def _python_profile(repo: Path, *, lifecycle: bool = True) -> list[dict[str, object]]:
    (repo / "product").mkdir(exist_ok=True)
    (repo / "tools").mkdir(exist_ok=True)
    (repo / "product" / "app.py").write_text("VALUE = 1\n", encoding="utf-8")
    (repo / "tools" / "check.py").write_text("VALUE = 2\n", encoding="utf-8")
    (repo / "product" / "requirements.txt").write_text("\n", encoding="utf-8")
    (repo / "tools" / "requirements.txt").write_text("\n", encoding="utf-8")
    components: list[dict[str, object]] = [
        {
            "id": "product",
            "classification": "PRODUCT",
            "include": ["product/**"],
            "purpose": "product_runtime",
            "manifests": ["product/requirements.txt"],
        },
        {
            "id": "tools",
            "classification": "TOOLCHAIN",
            "include": ["tools/**"],
            "purpose": "development_tooling",
            "manifests": ["tools/requirements.txt"],
        },
    ]
    if lifecycle:
        for item in components:
            item["release_owner"] = f"{item['id']}-release"
            item["compatibility_owner"] = f"{item['id']}-compat"
    return components


def _write_profile(repo: Path, components: list[dict[str, object]]) -> None:
    lines = [
        "ptsip:",
        '  version: "0.3.4-draft"',
        "  specification:",
        '    source: "https://github.com/kwaksinwoo01/ptsip"',
        f'    revision: "{SPEC_REVISION}"',
        "components:",
    ]
    for component in components:
        lines.extend(
            [
                f"  - id: {component['id']}",
                f"    classification: {component['classification']}",
                "    include:",
                *[f'      - "{item}"' for item in component.get("include", [])],
                f"    purpose: {component['purpose']}",
            ]
        )
        manifests = component.get("manifests", [])
        if manifests:
            lines.append("    manifests:")
            lines.extend(f'      - "{item}"' for item in manifests)
        if component.get("release_owner"):
            lines.append(f"    release_owner: {component['release_owner']}")
        if component.get("compatibility_owner"):
            lines.append(f"    compatibility_owner: {component['compatibility_owner']}")
    lines.extend(
        [
            "policies:",
            "  product_to_toolchain_runtime_dependency: deny",
            "  toolchain_in_product_package: deny",
            "  independent_build_resolution: required",
        ]
    )
    (repo / "ptsip.yaml").write_text("\n".join(lines) + "\n", encoding="utf-8")


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
                "evidence_ids": ["artifact:fixture:product-dist"],
            }
        ),
        encoding="utf-8",
    )


def _bind_artifact(path: Path, repo: Path) -> None:
    revision = subprocess.run(
        ["git", "-C", str(repo), "rev-parse", "HEAD"],
        check=True,
        stdout=subprocess.PIPE,
        text=True,
    ).stdout.strip()
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


def test_supported_clean_fixture_can_reach_conformant(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    components = _python_profile(repo)
    _write_profile(repo, components)
    _release_workflow(repo, "product-release.yml", "product/**")
    _release_workflow(repo, "toolchain-release.yml", "tools/**")
    artifact = tmp_path / "artifact.json"
    _artifact(artifact)
    _commit_all(repo)
    _bind_artifact(artifact, repo)

    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "CONFORMANT", result.report["coverage"]
    assert result.report["evaluators"]["independent_build_resolution"]["status"] == "RAN"
    assert result.report["evaluators"]["lifecycle_independence"]["status"] == "RAN"
    assert result.report["coverage"]["blocking_gaps"] == []


def test_cross_plane_shared_manifest_blocks_build_resolution(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    components = _python_profile(repo)
    (repo / "requirements.txt").write_text("requests\n", encoding="utf-8")
    for component in components:
        component["manifests"] = ["requirements.txt"]
    _commit_all(repo)

    result = evaluate_independent_build_resolution(repo, components)
    assert result.status == "BLOCKED"
    assert any("shared-cross-plane-manifest" in item["id"] for item in result.blocking_gaps)


def test_global_release_trigger_is_not_treated_as_lifecycle_proof_or_violation(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    components = _python_profile(repo)
    _write_profile(repo, components)
    _release_workflow(repo, "release.yml", None)
    _commit_all(repo)

    partition = partition_components(repo, components)
    lifecycle = evaluate_lifecycle_evidence(repo, components, partition)
    assert lifecycle.status == "BLOCKED"
    assert any(item["id"] == "lifecycle:product-release-evidence" for item in lifecycle.blocking_gaps)
    assert not any("violation" in item.lower() for item in lifecycle.observations)


def _npm_fixture(repo: Path, product_source: str) -> tuple[list[dict[str, object]], Path]:
    (repo / "product").mkdir(exist_ok=True)
    (repo / "tools").mkdir(exist_ok=True)
    (repo / "product" / "index.ts").write_text(product_source, encoding="utf-8")
    (repo / "tools" / "index.ts").write_text("export const tool = 1;\n", encoding="utf-8")
    (repo / "product" / "package.json").write_text(
        json.dumps({"name": "@fixture/product", "dependencies": {"@fixture/tools": "workspace:*"}}),
        encoding="utf-8",
    )
    (repo / "tools" / "package.json").write_text(
        json.dumps({"name": "@fixture/tools"}),
        encoding="utf-8",
    )
    components: list[dict[str, object]] = [
        {
            "id": "product",
            "classification": "PRODUCT",
            "include": ["product/**"],
            "purpose": "product_runtime",
            "manifests": ["product/package.json"],
            "release_owner": "product-release",
            "compatibility_owner": "product-compat",
        },
        {
            "id": "tools",
            "classification": "TOOLCHAIN",
            "include": ["tools/**"],
            "purpose": "development_tooling",
            "manifests": ["tools/package.json"],
            "release_owner": "tools-release",
            "compatibility_owner": "tools-compat",
        },
    ]
    _write_profile(repo, components)
    _release_workflow(repo, "product-release.yml", "product/**")
    _release_workflow(repo, "toolchain-release.yml", "tools/**")
    artifact = repo.parent / "npm-artifact.json"
    _artifact(artifact, "product/index.ts")
    return components, artifact


def test_npm_local_product_to_toolchain_dependency_is_nonconformant(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    _components, artifact = _npm_fixture(repo, 'import { tool } from "@fixture/tools";\nconsole.log(tool);\n')
    _commit_all(repo)

    scan = scan_dependency_edges(repo)
    assert "javascript-typescript" in scan.adapters
    assert "npm-manifest" in scan.adapters
    edge = next(item for item in scan.edges if item.evidence_id == "npm-manifest:product/package.json:dependencies:@fixture/tools")
    assert edge.resolved_path == "tools/package.json"
    assert edge.phase.value == "RUNTIME"

    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "NON_CONFORMANT"
    assert any(item["rule_id"] == "PTSIP-DEP-001" and item["outcome_effect"] == "NON_CONFORMANT" for item in result.report["diagnostics"])


def test_dynamic_javascript_import_blocks_product_dependency_conclusion(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    components, artifact = _npm_fixture(repo, "const target = process.env.TARGET;\nimport(target);\n")
    # Remove the declared local dependency so only the dynamic source edge remains.
    (repo / "product" / "package.json").write_text(json.dumps({"name": "@fixture/product"}), encoding="utf-8")
    _write_profile(repo, components)
    _commit_all(repo)

    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "INCOMPLETE"
    gap_ids = {item["id"] for item in result.report["coverage"]["blocking_gaps"]}
    assert any(item.startswith("dependency-target:javascript-dynamic:product/index.ts") for item in gap_ids)
