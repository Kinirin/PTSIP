from __future__ import annotations

import subprocess
from pathlib import Path

from ptsip.lifecycle_evidence import evaluate_lifecycle_evidence
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


def _commit(repo: Path) -> None:
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")


def _components(*, tool_shipped: bool | None = False) -> list[dict[str, object]]:
    tool: dict[str, object] = {
        "id": "tool",
        "classification": "TOOLCHAIN",
        "include": ["tools/**", ".github/workflows/**"],
        "purpose": "development_tooling",
        "release_owner": "tool-release",
        "compatibility_owner": "tool-compat",
    }
    if tool_shipped is not None:
        tool["shipped"] = tool_shipped
    return [
        {
            "id": "product",
            "classification": "PRODUCT",
            "include": ["product/**"],
            "purpose": "product_runtime",
            "shipped": True,
            "release_owner": "product-release",
            "compatibility_owner": "product-compat",
        },
        tool,
    ]


def _repo(tmp_path: Path, workflow_run: str, *, cache_path: str | None = None) -> Path:
    repo = tmp_path / "repo"
    _init(repo)
    (repo / "product").mkdir()
    (repo / "tools").mkdir()
    (repo / ".github" / "workflows").mkdir(parents=True)
    (repo / "product" / "package.json").write_text('{"name":"fixture-product"}\n', encoding="utf-8")
    (repo / "tools" / "check.py").write_text("VALUE = 1\n", encoding="utf-8")
    cache = "" if cache_path is None else f"          cache-dependency-path: {cache_path}\n"
    (repo / ".github" / "workflows" / "release.yml").write_text(
        "name: Manual Release\n"
        "on:\n"
        "  workflow_dispatch:\n"
        "jobs:\n"
        "  release:\n"
        "    runs-on: ubuntu-latest\n"
        "    steps:\n"
        "      - uses: actions/setup-node@v6\n"
        "        with:\n"
        "          node-version: '22'\n"
        + cache
        + "      - name: Release\n"
        + f"        run: {workflow_run}\n",
        encoding="utf-8",
    )
    _commit(repo)
    return repo


def _gap_ids(result) -> set[str]:
    return {str(item["id"]) for item in result.blocking_gaps}


def test_manual_npm_prefix_release_is_scoped_to_product_manifest(tmp_path: Path) -> None:
    repo = _repo(
        tmp_path,
        "npm --prefix product run dist",
        cache_path="product/package.json",
    )
    components = _components(tool_shipped=False)

    result = evaluate_lifecycle_evidence(repo, components, partition_components(repo, components))

    assert result.status == "RAN"
    assert result.blocking_gaps == ()
    assert len(result.workflows) == 1
    workflow = result.workflows[0]
    assert workflow.manual_only is True
    assert workflow.scope_complete is True
    assert workflow.scope_source == "manual-subject-paths"
    assert workflow.scoped_classifications == ("PRODUCT",)


def test_manual_release_without_subject_or_artifact_remains_blocked(tmp_path: Path) -> None:
    repo = _repo(tmp_path, "echo release")
    components = _components(tool_shipped=False)

    result = evaluate_lifecycle_evidence(repo, components, partition_components(repo, components))

    assert result.status == "BLOCKED"
    ids = _gap_ids(result)
    assert "lifecycle:product-release-evidence" in ids
    assert "lifecycle:ambiguous-release-workflow:.github/workflows/release.yml" in ids


def test_explicit_nonshipped_toolchain_does_not_require_release_workflow(tmp_path: Path) -> None:
    repo = _repo(tmp_path, "npm --prefix product run dist")
    components = _components(tool_shipped=False)

    result = evaluate_lifecycle_evidence(repo, components, partition_components(repo, components))

    assert result.status == "RAN"
    assert "lifecycle:toolchain-release-evidence" not in _gap_ids(result)
    assert any("shipped=false" in item for item in result.observations)


def test_toolchain_without_explicit_shipped_false_remains_fail_closed(tmp_path: Path) -> None:
    repo = _repo(tmp_path, "npm --prefix product run dist")
    components = _components(tool_shipped=None)

    result = evaluate_lifecycle_evidence(repo, components, partition_components(repo, components))

    assert result.status == "BLOCKED"
    assert "lifecycle:toolchain-release-evidence" in _gap_ids(result)


def test_bound_observed_artifact_can_scope_manual_release_as_fallback(tmp_path: Path) -> None:
    repo = _repo(tmp_path, "echo release")
    components = _components(tool_shipped=False)
    artifacts = [
        {
            "binding_valid": True,
            "payload": {
                "format": "ptsip-artifact-evidence/v1",
                "artifact_id": "product-dist",
                "classification": "PRODUCT",
                "producer_component": "tool",
                "artifact_type": "fixture",
                "shipping_scope": "product-distribution",
                "contents": {
                    "paths": ["product/package.json"],
                    "components": ["product"],
                    "complete": True,
                },
                "provenance": "OBSERVED",
            },
        }
    ]

    result = evaluate_lifecycle_evidence(
        repo,
        components,
        partition_components(repo, components),
        artifact_documents=artifacts,
    )

    assert result.status == "RAN"
    workflow = result.workflows[0]
    assert workflow.scope_complete is True
    assert workflow.scope_source == "artifact-evidence"
    assert workflow.scoped_classifications == ("PRODUCT",)
