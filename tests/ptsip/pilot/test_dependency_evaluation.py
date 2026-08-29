from __future__ import annotations

import subprocess
from pathlib import Path

from ptsip.constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION
from ptsip.pilot.runner import run_pilot
from ptsip.profile_identity import CURRENT_PROJECT_PROFILE_VERSION


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


def test_dependency_evaluator_reports_ran_instead_of_inferring_from_empty_findings(
    tmp_path: Path,
    monkeypatch,
) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    (repo / "product").mkdir()
    (repo / "tools").mkdir()
    (repo / "product" / "app.py").write_text("VALUE = 1\n", encoding="utf-8")
    (repo / "tools" / "check.py").write_text("VALUE = 2\n", encoding="utf-8")
    (repo / "ptsip.yaml").write_text(
        f"""ptsip:\n  version: \"{CURRENT_PROJECT_PROFILE_VERSION}\"\n  specification:\n    family: \"{SPEC_VERSION}\"\n    source: \"{SPEC_SOURCE}\"\n    revision: \"{SPEC_REVISION}\"\nresponsibility_map:\n  mode: explicit\ncomponents:\n  - id: product\n    classification: PRODUCT\n    include: [\"product/**\"]\n    purpose: product_runtime\n  - id: tools\n    classification: DEVELOPMENT_TOOLING\n    include: [\"tools/**\"]\n    purpose: development_tooling\npolicies:\n  product_to_nonproduct_runtime_dependency: deny\n  nonproduct_in_product_package: deny\n  independent_build_resolution: required\n""",
        encoding="utf-8",
    )
    _commit_all(repo)
    monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))

    result = run_pilot(repo)
    evaluator = result.report["evaluation"]["declared_dependency_boundaries"]
    assert evaluator["status"] == "RAN"
    assert evaluator["reason"] is None
    assert evaluator["finding_count"] == 0
    assert result.report["conformance"]["status"] == "NOT_EVALUATED"
