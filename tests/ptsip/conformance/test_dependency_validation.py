import yaml
import pytest

from ptsip.dependency_analysis import analyze_dependencies
from ptsip.dependency_validation import validation_plan
from test_dependency_reconciliation import _fixture, _git


def test_validation_plan_follows_declared_verifies_and_batches_tests(tmp_path):
    repo = tmp_path / "repo"
    _fixture(repo, "import numpy", "numpy==2")
    (repo / "tests").mkdir()
    (repo / "tests/test_app.py").write_text("import pytest\nimport product.app\n", encoding="utf-8")
    profile = yaml.safe_load((repo / "ptsip.yaml").read_text(encoding="utf-8"))
    profile["components"].append({"id": "product-checks", "classification": "PRODUCT", "roles": ["VERIFICATION"],
                                  "include": ["tests/**"], "purpose": "product_verification", "shipped": False,
                                  "runtime_required": False, "executable": True})
    profile["relationships"] = [{"id": "checks-app", "from": "product-checks", "to": "product-app", "type": "VERIFIES"}]
    (repo / "ptsip.yaml").write_text(yaml.safe_dump(profile), encoding="utf-8")
    _git(repo, "add", "tests", "ptsip.yaml")
    report = analyze_dependencies(repo)
    plan = validation_plan(repo, report, changed_paths=["product/app.py"])
    assert plan["status"] == "PROPOSED" and plan["executed"] is False
    assert plan["steps"][0]["command"] == ["python", "-m", "pytest", "tests/test_app.py", "-q"]
    assert plan["steps"][1]["command"] == ["python", "-m", "pytest", "tests", "-q", "--maxfail=10"]
    (repo / "ptsip.yaml").write_text("changed: true", encoding="utf-8")
    assert validation_plan(repo, report)["reason"] == "STALE_ANALYSIS"


def test_validation_plan_does_not_invent_verification_or_ownership(tmp_path):
    repo = tmp_path / "repo"
    _fixture(repo, "import numpy", "numpy==2")
    report = analyze_dependencies(repo)
    assert validation_plan(repo, report, changed_paths=["unknown.py"])["status"] == "BLOCKED"
    result = validation_plan(repo, report, changed_paths=["product/app.py"])
    assert result["status"] == "REVIEW_REQUIRED" and result["steps"] == []


@pytest.mark.parametrize("manifest", ['project = "invalid"', '[project]\ndependencies = ["pytest-unrelated"]'])
def test_validation_plan_requires_evidence_for_the_proposed_runner(tmp_path, manifest):
    repo = tmp_path / "repo"
    _fixture(repo, "import numpy", "numpy==2")
    (repo / "tests").mkdir()
    (repo / "tests/test_app.py").write_text("import unittest\n", encoding="utf-8")
    profile = yaml.safe_load((repo / "ptsip.yaml").read_text(encoding="utf-8"))
    profile["components"].append({"id": "product-checks", "classification": "PRODUCT", "roles": ["VERIFICATION"],
                                  "include": ["tests/**"], "purpose": "product_verification", "shipped": False,
                                  "runtime_required": False, "executable": True})
    (repo / "ptsip.yaml").write_text(yaml.safe_dump(profile), encoding="utf-8")
    (repo / "pyproject.toml").write_text(manifest, encoding="utf-8")
    _git(repo, "add", ".")
    report = analyze_dependencies(repo)
    plan = validation_plan(repo, report, changed_paths=["tests/test_app.py"])
    assert plan["status"] == "REVIEW_REQUIRED" and plan["steps"] == []
