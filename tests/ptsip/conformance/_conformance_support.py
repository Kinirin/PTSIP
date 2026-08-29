from __future__ import annotations

import hashlib
import json
import subprocess
from pathlib import Path

from ptsip.constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION
from ptsip.profile_identity import CURRENT_PROJECT_PROFILE_VERSION
from ptsip.repository.snapshot import capture_snapshot


def init_repo(repo: Path) -> None:
    repo.mkdir(parents=True, exist_ok=True)
    git(repo, "init")
    git(repo, "config", "user.email", "ptsip-test@example.invalid")
    git(repo, "config", "user.name", "PTSIP Test")


def git(repo: Path, *args: str) -> str:
    result = subprocess.run(
        ["git", "-C", str(repo), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    return result.stdout.strip()


def commit(repo: Path) -> str:
    git(repo, "add", ".")
    git(repo, "commit", "-m", "fixture")
    return git(repo, "rev-parse", "HEAD")


def components() -> list[dict[str, object]]:
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


def write_workflow(
    repo: Path,
    name: str,
    scope: str | None,
    *,
    release: bool = True,
) -> None:
    directory = repo / ".github" / "workflows"
    directory.mkdir(parents=True, exist_ok=True)
    scope_text = "" if scope is None else f"    paths:\n      - '{scope}'\n"
    action = "echo release" if release else "python -m pytest"
    job = "release" if release else "test"
    (directory / name).write_text(
        f"name: {name}\non:\n  push:\n{scope_text}jobs:\n  {job}:\n    runs-on: ubuntu-latest\n    steps:\n      - run: {action}\n",
        encoding="utf-8",
    )


def write_component_profile(repo: Path, declared_components: list[dict[str, object]]) -> None:
    lines = [
        "ptsip:",
        f'  version: "{CURRENT_PROJECT_PROFILE_VERSION}"',
        "  specification:",
        f'    family: "{SPEC_VERSION}"',
        f'    source: "{SPEC_SOURCE}"',
        f'    revision: "{SPEC_REVISION}"',
        "responsibility_map:",
        "  mode: explicit",
        "components:",
    ]
    for component in declared_components:
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


def write_profile(repo: Path, product_manifest: str, tool_manifest: str) -> None:
    declared = components()
    declared[0]["manifests"] = [product_manifest]
    declared[1]["manifests"] = [tool_manifest]
    write_component_profile(repo, declared)
    write_workflow(repo, "product-release.yml", "product/**")
    write_workflow(repo, "development-tooling-release.yml", "tools/**")


def write_artifact(path: Path, product_path: str = "product/app.py") -> None:
    path.write_text(
        json.dumps(
            {
                "format": "ptsip-artifact-evidence/v1",
                "artifact_id": "product-dist",
                "classification": "PRODUCT",
                "producer_component": "tools",
                "artifact_type": "fixture",
                "shipping_scope": "product-distribution",
                "contents": {
                    "paths": [product_path],
                    "components": ["product"],
                    "complete": True,
                },
                "derivation": [{"relation": "GENERATES", "source": "tools"}],
                "provenance": "OBSERVED",
                "evidence_ids": ["artifact:fixture:product-dist"],
            }
        ),
        encoding="utf-8",
    )


def bind_artifact(path: Path, repo: Path, revision: str | None = None) -> None:
    revision = revision or git(repo, "rev-parse", "HEAD")
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


def python_clean(repo: Path) -> Path:
    (repo / "product").mkdir(exist_ok=True)
    (repo / "tools").mkdir(exist_ok=True)
    (repo / "product" / "app.py").write_text("VALUE = 1\n", encoding="utf-8")
    (repo / "tools" / "check.py").write_text("VALUE = 2\n", encoding="utf-8")
    (repo / "product" / "requirements.txt").write_text("\n", encoding="utf-8")
    (repo / "tools" / "requirements.txt").write_text("\n", encoding="utf-8")
    write_profile(repo, "product/requirements.txt", "tools/requirements.txt")
    artifact = repo.parent / "artifact.json"
    write_artifact(artifact)
    return artifact


def clean_repo(
    tmp_path: Path,
    product_source: str = "VALUE = 1\n",
) -> tuple[Path, Path, str]:
    repo = tmp_path / "repo"
    init_repo(repo)
    (repo / "product").mkdir()
    (repo / "tools").mkdir()
    (repo / "product" / "app.py").write_text(product_source, encoding="utf-8")
    (repo / "tools" / "check.py").write_text("VALUE = 2\n", encoding="utf-8")
    (repo / "product" / "requirements.txt").write_text("\n", encoding="utf-8")
    (repo / "tools" / "requirements.txt").write_text("\n", encoding="utf-8")
    write_component_profile(repo, components())
    write_workflow(repo, "product-release.yml", "product/**")
    write_workflow(repo, "tool-release.yml", "tools/**")
    artifact = tmp_path / "artifact.json"
    write_artifact(artifact)
    revision = commit(repo)
    bind_artifact(artifact, repo, revision)
    return repo, artifact, revision


def write_external_evidence(
    path: Path,
    repo: Path,
    revision: str,
    evidence: list[dict[str, object]],
    repository: str | None = None,
) -> None:
    path.write_text(
        json.dumps(
            {
                "format": "ptsip-external-evidence/v1",
                "producer": {"id": "fixture-validator", "version": "1"},
                "subject": {
                    "repository": repository or str(repo.resolve()),
                    "revision": revision,
                },
                "evidence": evidence,
            }
        ),
        encoding="utf-8",
    )


def blocking_gaps(result) -> list[dict[str, object]]:
    return result.report["coverage"]["blocking_gaps"]


def build_repo(tmp_path: Path) -> tuple[Path, list[dict[str, object]]]:
    repo = tmp_path / "build"
    init_repo(repo)
    (repo / "product").mkdir()
    (repo / "tools").mkdir()
    (repo / "product" / "app.py").write_text("VALUE = 1\n", encoding="utf-8")
    (repo / "tools" / "check.py").write_text("VALUE = 2\n", encoding="utf-8")
    (repo / "product" / "requirements.txt").write_text("\n", encoding="utf-8")
    (repo / "tools" / "requirements.txt").write_text("\n", encoding="utf-8")
    declared = components()
    commit(repo)
    return repo, declared
