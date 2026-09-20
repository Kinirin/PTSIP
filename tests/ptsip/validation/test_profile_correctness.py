from __future__ import annotations

import subprocess
from pathlib import Path

import yaml

from ptsip.constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION
from ptsip.profile_identity import CURRENT_PROJECT_PROFILE_VERSION
from ptsip.profile_metadata import current_project_profile_ptsip_metadata
from ptsip.validation.profile import validate_profile


def _git(repo: Path, *args: str) -> None:
    subprocess.run(["git", "-C", str(repo), *args], check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)


def _init_repo(repo: Path) -> None:
    repo.mkdir(parents=True)
    _git(repo, "init")
    _git(repo, "config", "user.email", "ptsip-test@example.invalid")
    _git(repo, "config", "user.name", "PTSIP Test")
    (repo / "product").mkdir()
    (repo / "tools").mkdir()
    (repo / "product" / "app.py").write_text("VALUE = 1\n", encoding="utf-8")
    (repo / "tools" / "check.py").write_text("VALUE = 2\n", encoding="utf-8")
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")


def _profile(revision: str, policy: str = "") -> str:
    ptsip = current_project_profile_ptsip_metadata()
    specification = ptsip["specification"]
    assert isinstance(specification, dict)
    specification["revision"] = revision
    header = yaml.safe_dump({"ptsip": ptsip}, sort_keys=False, allow_unicode=True)
    return header + f"""responsibility_map:
  mode: explicit
components:
  - id: product
    classification: PRODUCT
    include: ["product/**"]
    purpose: runtime
  - id: tools
    classification: DEVELOPMENT_TOOLING
    include: ["tools/**"]
    purpose: tooling
{policy}policies:
  product_to_nonproduct_runtime_dependency: deny
  nonproduct_in_product_package: deny
  independent_build_resolution: required
"""


def test_explicit_profile_revision_must_match_tool_binding(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_repo(repo)
    (repo / "ptsip.yaml").write_text(_profile("895e12d27230af2bb99ad17a96e8df8ef41bc3e0"), encoding="utf-8")
    result = validate_profile(repo)
    assert not result.valid
    assert any("[SPEC_BINDING_UNSUPPORTED]" in item for item in result.errors)


def test_component_dependency_policy_references_declared_components(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_repo(repo)
    policy = """component_dependency_policy:\n  default: deny\n  allow:\n    - from: tools\n      to: missing-component\n"""
    (repo / "ptsip.yaml").write_text(_profile(SPEC_REVISION, policy), encoding="utf-8")
    result = validate_profile(repo)
    assert not result.valid
    assert any("missing-component" in item and "not declared" in item for item in result.errors)
