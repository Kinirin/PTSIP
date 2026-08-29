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


def write_workflow(repo: Path, name: str, scope: str) -> None:
    path = repo / ".github" / "workflows"
    path.mkdir(parents=True, exist_ok=True)
    (path / name).write_text(
        f"name: {name}\non:\n  push:\n    paths:\n      - '{scope}'\njobs:\n  release:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo release\n",
        encoding="utf-8",
    )


def write_profile(repo: Path, product_manifest: str, tool_manifest: str) -> None:
    (repo / "ptsip.yaml").write_text(
        f"""ptsip:
  version: "{CURRENT_PROJECT_PROFILE_VERSION}"
  specification:
    family: "{SPEC_VERSION}"
    source: "{SPEC_SOURCE}"
    revision: "{SPEC_REVISION}"
responsibility_map:
  mode: explicit
components:
  - id: product
    classification: PRODUCT
    include: ["product/**"]
    purpose: product_runtime
    manifests: ["{product_manifest}"]
    release_owner: product-release
    compatibility_owner: product-compat
  - id: tools
    classification: DEVELOPMENT_TOOLING
    include: ["tools/**"]
    purpose: development_tooling
    manifests: ["{tool_manifest}"]
    release_owner: development-tooling-release
    compatibility_owner: development-tooling-compat
policies:
  product_to_nonproduct_runtime_dependency: deny
  nonproduct_in_product_package: deny
  independent_build_resolution: required
""",
        encoding="utf-8",
    )
    write_workflow(repo, "product-release.yml", "product/**")
    write_workflow(repo, "development-tooling-release.yml", "tools/**")


def write_artifact(path: Path, product_path: str) -> None:
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
    write_artifact(artifact, "product/app.py")
    return artifact
