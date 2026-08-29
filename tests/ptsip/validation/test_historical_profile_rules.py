from __future__ import annotations

import subprocess
from pathlib import Path

from ptsip.validation.profile import validate_profile


HISTORICAL_SPEC_REVISION = "d6995ed232e845b88d8235b851e80ab54b7804ea"


def _git(repo: Path, *args: str) -> None:
    subprocess.run(
        ["git", "-C", str(repo), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )


def _init_git(repo: Path) -> None:
    repo.mkdir(parents=True, exist_ok=True)
    _git(repo, "init")
    _git(repo, "config", "user.email", "ptsip-test@example.invalid")
    _git(repo, "config", "user.name", "PTSIP Test")


def _commit_all(repo: Path) -> None:
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")


def test_component_profile_allows_specific_nested_override(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    (repo / "src" / "install").mkdir(parents=True)
    (repo / "src" / "app.py").write_text("x = 1\n", encoding="utf-8")
    (repo / "src" / "install" / "plugin_build.py").write_text(
        "x = 1\n",
        encoding="utf-8",
    )
    _commit_all(repo)
    profile = repo / "ptsip.yaml"
    profile.write_text(
        f"""ptsip:\n  version: \"0.3.6-draft\"\n  specification:\n    source: \"https://github.com/Kinirin/PTSIP\"\n    revision: \"{HISTORICAL_SPEC_REVISION}\"\nresponsibility_map:\n  mode: explicit\ncomponents:\n  - id: product-runtime\n    classification: PRODUCT\n    include: [\"src/**\"]\n    purpose: product_runtime\n  - id: plugin-builder\n    classification: DELIVERY\n    include: [\"src/install/plugin_build.py\"]\n    purpose: authoritative_release_build\n    shipped: false\n    runtime_required: false\npolicies:\n  product_to_nonproduct_runtime_dependency: deny\n  nonproduct_in_product_package: deny\n  independent_build_resolution: required\n""",
        encoding="utf-8",
    )
    result = validate_profile(repo)
    assert result.valid, result.errors
    partition = result.details["component_partition"]
    owners = {item["path"]: item["component_id"] for item in partition["assignments"]}
    assert owners["src/app.py"] == "product-runtime"
    assert owners["src/install/plugin_build.py"] == "plugin-builder"


def test_legacy_exception_waiver_is_rejected(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    (repo / "src").mkdir()
    (repo / "src" / "app.py").write_text("x = 1\n", encoding="utf-8")
    _commit_all(repo)
    profile = repo / "ptsip.yaml"
    profile.write_text(
        f"""ptsip:\n  version: \"0.3.6-draft\"\n  specification:\n    source: \"https://github.com/Kinirin/PTSIP\"\n    revision: \"{HISTORICAL_SPEC_REVISION}\"\nresponsibility_map:\n  mode: explicit\ncomponents:\n  - id: product-runtime\n    classification: PRODUCT\n    include: [\"src/**\"]\n    purpose: product_runtime\npolicies:\n  product_to_nonproduct_runtime_dependency: deny\n  nonproduct_in_product_package: deny\n  independent_build_resolution: required\nexceptions: []\n""",
        encoding="utf-8",
    )
    result = validate_profile(repo)
    assert not result.valid
    assert any(
        "exceptions" in error and "unexpected" in error.lower()
        for error in result.errors
    )
