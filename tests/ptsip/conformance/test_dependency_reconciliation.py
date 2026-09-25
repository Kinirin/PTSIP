from __future__ import annotations

import subprocess
from pathlib import Path

from ptsip.conformance import evaluate_conformance
from ptsip.conformance_engine import evaluate_conformance as evaluate_engine_conformance
from ptsip.constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION
from ptsip.profile_identity import CURRENT_PROJECT_PROFILE_VERSION


def _git(repo: Path, *args: str) -> None:
    subprocess.run(
        ["git", "-C", str(repo), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )


def _fixture(repo: Path, import_line: str, requirement: str) -> None:
    (repo / "product").mkdir(parents=True)
    (repo / "install").mkdir(parents=True)
    (repo / "product" / "app.py").write_text(import_line + "\n", encoding="utf-8")
    (repo / "install" / "runtime-requirements.txt").write_text(
        requirement + "\n",
        encoding="utf-8",
    )
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
  - id: product-app
    classification: PRODUCT
    roles: [IMPLEMENTATION]
    include: ["product/**"]
    purpose: product_runtime
    shipped: true
    runtime_required: true
    executable: true
  - id: product-install
    classification: PRODUCT
    roles: [CONFIGURATION]
    include: ["install/**"]
    purpose: product_runtime_dependencies
    shipped: true
    runtime_required: true
    executable: false
policies:
  product_to_nonproduct_runtime_dependency: deny
  nonproduct_in_product_package: deny
  independent_build_resolution: required
""",
        encoding="utf-8",
    )
    _git(repo, "init")
    _git(repo, "config", "user.email", "ptsip-test@example.invalid")
    _git(repo, "config", "user.name", "PTSIP Test")
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")


def _dependency_gap_ids(result) -> set[str]:
    return {
        str(item["id"])
        for item in result.report["coverage"]["blocking_gaps"]
        if str(item["id"]).startswith("dependency-target:")
    }


def test_product_runtime_manifest_reconciles_cross_component_python_dependency(
    tmp_path: Path,
) -> None:
    repo = tmp_path / "repo"
    _fixture(repo, "import numpy", "numpy==2.5.2")

    result = evaluate_conformance(repo)

    assert not any("numpy" in item for item in _dependency_gap_ids(result))
    reconciliation = result.report["dependency_reconciliation"]
    assert reconciliation["summary"]["resolved_external_count"] == 1
    resolved = reconciliation["resolved_external"][0]
    assert resolved["distribution"] == "numpy"
    assert resolved["basis"] == "PRODUCT_RUNTIME_MANIFEST"
    assert resolved["declaration_paths"] == ["install/runtime-requirements.txt"]


def test_import_distribution_alias_is_reconciled_explicitly(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _fixture(repo, "from PIL import Image", "Pillow==12.2.0")

    result = evaluate_conformance(repo)

    assert not any("PIL" in item for item in _dependency_gap_ids(result))
    reconciliation = result.report["dependency_reconciliation"]
    resolved = reconciliation["resolved_external"][0]
    assert resolved["distribution"] == "pillow"
    assert resolved["alias_applied"] is True


def test_unlisted_product_dependency_remains_blocking(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _fixture(repo, "import undeclared_runtime_package", "numpy==2.5.2")

    result = evaluate_conformance(repo)

    assert any(
        "undeclared_runtime_package" in item
        for item in _dependency_gap_ids(result)
    )
    reconciliation = result.report["dependency_reconciliation"]
    assert reconciliation["summary"]["resolved_external_count"] == 0
    assert reconciliation["summary"]["unresolved_candidate_count"] == 1


def test_engine_preserves_runtime_reconciliation_and_unresolved_blockers(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _fixture(
        repo,
        "import numpy\nfrom PIL import Image\nimport undeclared_runtime_package\n__import__(module_name)",
        "numpy==2.5.2\nPillow==12.2.0",
    )

    result = evaluate_engine_conformance(repo)

    gap_ids = _dependency_gap_ids(result)
    assert not any("numpy" in item or "PIL" in item for item in gap_ids)
    assert any("undeclared_runtime_package" in item for item in gap_ids)
    assert "dependency-target:python-dynamic:product/app.py:4" in gap_ids
    reconciliation = result.report["dependency_reconciliation"]
    assert reconciliation["summary"]["resolved_external_count"] == 2
    assert reconciliation["summary"]["unresolved_candidate_count"] == 1
    assert reconciliation["summary"]["alias_resolved_count"] == 1
    assert {item["distribution"] for item in reconciliation["resolved_external"]} == {"numpy", "pillow"}
    for item in reconciliation["resolved_external"]:
        assert item["basis"] == "PRODUCT_RUNTIME_MANIFEST"
        assert item["declaration_paths"] == ["install/runtime-requirements.txt"]
        assert f"dependency-target:{item['evidence_id']}" not in gap_ids
    assert result.report["evaluators"]["dependency_reconciliation"] == {
        "status": "RAN",
        "resolved_external_count": 2,
        "candidate_count": 3,
    }
    assert result.report["outcome"] == "INCOMPLETE"
