from __future__ import annotations

import subprocess
from pathlib import Path

from ptsip.validation.profile import validate_profile


SPEC_REVISION = "afba3531e23d96c21b7216e49614b839158ca7d5"


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
    return f"""ptsip:\n  version: \"0.3.4-draft\"\n  specification:\n    source: \"https://github.com/kwaksinwoo01/ptsip\"\n    revision: \"{revision}\"\ncomponents:\n  - id: product\n    classification: PRODUCT\n    include: [\"product/**\"]\n    purpose: runtime\n  - id: tools\n    classification: TOOLCHAIN\n    include: [\"tools/**\"]\n    purpose: tooling\n{policy}policies:\n  product_to_toolchain_runtime_dependency: deny\n  toolchain_in_product_package: deny\n  independent_build_resolution: required\n"""


def test_explicit_profile_revision_must_match_tool_binding(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_repo(repo)
    (repo / "ptsip.yaml").write_text(_profile("895e12d27230af2bb99ad17a96e8df8ef41bc3e0"), encoding="utf-8")
    result = validate_profile(repo)
    assert not result.valid
    assert any("not supported by tooling snapshot" in item for item in result.errors)


def test_component_dependency_policy_references_declared_components(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_repo(repo)
    policy = """component_dependency_policy:\n  default: deny\n  allow:\n    - from: tools\n      to: missing-component\n"""
    (repo / "ptsip.yaml").write_text(_profile(SPEC_REVISION, policy), encoding="utf-8")
    result = validate_profile(repo)
    assert not result.valid
    assert any("missing-component" in item and "not declared" in item for item in result.errors)
