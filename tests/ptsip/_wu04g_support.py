from __future__ import annotations

import copy
import subprocess
from pathlib import Path

import yaml

from ptsip.constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION


PYTHON_PACKAGE_TEMPLATE_ID = "python-package-library"
PYTHON_PACKAGE_TEMPLATE_REVISION = "sha256:409acd1cd9907a60761a3cf26a051185d40b5e926e6952131b641b10bccc5c9b"


def git(repo: Path, *args: str, check: bool = True) -> subprocess.CompletedProcess[str]:
    """Run Git against one test-owned repository without retaining mutable global state."""
    return subprocess.run(
        ["git", "-C", str(repo), *args],
        check=check,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )


def init_git_repo(repo: Path, *, remote_url: str | None = None) -> Path:
    """Create a fresh Git fixture repository with deterministic test identity."""
    repo.mkdir(parents=True, exist_ok=False)
    git(repo, "init")
    git(repo, "config", "user.email", "ptsip-test@example.invalid")
    git(repo, "config", "user.name", "PTSIP Test")
    if remote_url is not None:
        git(repo, "remote", "add", "origin", remote_url)
    return repo


def commit_all(repo: Path, message: str = "fixture") -> str:
    """Commit the current fixture state and return the exact resulting HEAD SHA."""
    git(repo, "add", ".")
    git(repo, "commit", "-m", message)
    return git(repo, "rev-parse", "HEAD").stdout.strip()


def write_text(root: Path, relative_path: str, content: str) -> Path:
    """Write one UTF-8 fixture file below root, creating only its parent directories."""
    path = root / relative_path
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")
    return path


def clone_repo(source: Path, destination: Path, *, remote_url: str | None = None) -> Path:
    """Clone a test-owned fixture repository and optionally replace its origin URL."""
    subprocess.run(
        ["git", "clone", str(source), str(destination)],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    if remote_url is not None:
        git(destination, "remote", "set-url", "origin", remote_url)
    return destination


def canonical_v2_answer(
    *,
    classification: str = "DEVELOPMENT_TOOLING",
    purpose: str = "Repository-local generation tooling",
    shipped: bool = False,
    runtime_required: bool = False,
    executable: bool = True,
) -> dict[str, object]:
    """Return the exact canonical ptsip-clarification-answer/v2 decision shape."""
    return {
        "classification": classification,
        "purpose": purpose,
        "shipped": shipped,
        "runtime_required": runtime_required,
        "executable": executable,
    }


def legacy_v1_answer(
    *,
    classification: str = "DEVELOPMENT_TOOLING",
    purpose: str = "Repository-local generation tooling",
    shipped: bool = False,
    runtime_required: bool = False,
    lifecycle_owner: str = "DEVELOPMENT_TOOLING",
    executable: bool = True,
) -> dict[str, object]:
    """Return historical v1 input only for explicit compatibility tests."""
    return {
        "classification": classification,
        "purpose": purpose,
        "shipped": shipped,
        "runtime_required": runtime_required,
        "lifecycle_owner": lifecycle_owner,
        "executable": executable,
    }


def clarification_answer_text(
    decision: dict[str, object],
    *,
    format_name: str = "ptsip-clarification-answer/v2",
) -> str:
    return yaml.safe_dump(
        {"format": format_name, "decision": copy.deepcopy(decision)},
        sort_keys=False,
    )


def policy_payload() -> dict[str, object]:
    return {
        "product_to_nonproduct_runtime_dependency": "deny",
        "nonproduct_in_product_package": "deny",
        "independent_build_resolution": "required",
    }


def component_payload(
    component_id: str,
    include: list[str],
    *,
    classification: str = "DEVELOPMENT_TOOLING",
    purpose: str = "test_component",
    shipped: bool = False,
    runtime_required: bool = False,
    executable: bool = True,
    roles: list[str] | None = None,
    exclude: list[str] | None = None,
) -> dict[str, object]:
    result: dict[str, object] = {
        "id": component_id,
        "classification": classification,
        "include": list(include),
        "purpose": purpose,
        "shipped": shipped,
        "runtime_required": runtime_required,
        "executable": executable,
    }
    if roles:
        result["roles"] = list(roles)
    if exclude:
        result["exclude"] = list(exclude)
    return result


def associated_artifact_payload(
    artifact_id: str,
    anchor: str,
    include: list[str],
    *,
    purpose: str = "test_associated_artifact",
    exclude: list[str] | None = None,
) -> dict[str, object]:
    result: dict[str, object] = {
        "id": artifact_id,
        "anchor": anchor,
        "include": list(include),
        "purpose": purpose,
    }
    if exclude:
        result["exclude"] = list(exclude)
    return result


def _profile_base() -> dict[str, object]:
    return {
        "ptsip": {
            "version": SPEC_VERSION,
            "specification": {"source": SPEC_SOURCE, "revision": SPEC_REVISION},
        },
        "policies": policy_payload(),
    }


def explicit_profile_payload(
    components: list[dict[str, object]],
    *,
    associated_artifacts: list[dict[str, object]] | None = None,
    relationships: list[dict[str, object]] | None = None,
) -> dict[str, object]:
    payload = _profile_base()
    payload["responsibility_map"] = {"mode": "explicit"}
    payload["components"] = copy.deepcopy(components)
    if associated_artifacts:
        payload["associated_artifacts"] = copy.deepcopy(associated_artifacts)
    if relationships:
        payload["relationships"] = copy.deepcopy(relationships)
    return payload


def template_profile_payload(
    *,
    template_id: str = PYTHON_PACKAGE_TEMPLATE_ID,
    template_revision: str = PYTHON_PACKAGE_TEMPLATE_REVISION,
) -> dict[str, object]:
    payload = _profile_base()
    payload["responsibility_map"] = {
        "mode": "template",
        "template": {"id": template_id, "revision": template_revision},
    }
    return payload


def hybrid_profile_payload(
    overrides: dict[str, object],
    *,
    template_id: str = PYTHON_PACKAGE_TEMPLATE_ID,
    template_revision: str = PYTHON_PACKAGE_TEMPLATE_REVISION,
) -> dict[str, object]:
    payload = _profile_base()
    payload["responsibility_map"] = {
        "mode": "hybrid",
        "template": {"id": template_id, "revision": template_revision},
        "overrides": copy.deepcopy(overrides),
    }
    return payload


def write_profile(path: Path, payload: dict[str, object]) -> Path:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8")
    return path
