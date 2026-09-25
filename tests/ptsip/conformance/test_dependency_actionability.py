from __future__ import annotations

from pathlib import Path
import pytest

from ptsip.dependency_analysis import analyze_dependencies
from test_dependency_reconciliation import _fixture


def test_machine_buckets_preserve_declaration_provenance_and_bound_review(tmp_path: Path):
    repo = tmp_path / "repo"
    _fixture(repo, "import numpy\nfrom PIL import Image\nimport missing_package\n"
             "try:\n    import optional_package\nexcept ImportError:\n    optional_package = None\n"
             "__import__(module_name)\nfrom . import unknown", "numpy==2\nPillow==12")
    report = analyze_dependencies(repo)
    assert report["status"] == "RAN"
    assert report["authority"] == "ADVISORY_ONLY"
    items = {item["dependency"]["target"]: item for item in report["items"]}
    assert items["numpy"]["actionability"] == "AUTO_RESOLVED"
    assert items["PIL"]["declaration"]["reconciliation"]["declaration_paths"] == ["install/runtime-requirements.txt"]
    assert items["missing_package"]["actionability"] == "REPOSITORY_DEFECT"
    assert items["missing_package"]["remediation_candidate"]["automatic_apply"] is False
    assert items["optional_package"]["actionability"] == "REVIEW_REQUIRED"
    assert items["optional_package"]["usage"]["fallback"]["detected"] is True
    assert items["<dynamic-import>"]["actionability"] == "REVIEW_REQUIRED"
    assert report["summary"]["machine_classified"] == 6
    assert report["summary"]["ai_reviewed_items"] == 0
    assert sum(report["summary"][key] for key in (
        "AUTO_RESOLVED", "REPOSITORY_DEFECT", "REVIEW_REQUIRED", "RESOLVER_LIMITATION")) == 6


def test_invalid_profile_never_auto_resolves_or_proposes_mutation(tmp_path: Path):
    repo = tmp_path / "repo"
    _fixture(repo, "import numpy", "numpy==2")
    (repo / "ptsip.yaml").write_text("invalid: true\n", encoding="utf-8")
    report = analyze_dependencies(repo)
    assert report["status"] == "BLOCKED"
    assert report["summary"]["AUTO_RESOLVED"] == 0
    assert all(item["remediation_candidate"] is None for item in report["items"])


def test_local_name_ambiguity_is_resolver_queue_not_requirement_proposal(tmp_path: Path):
    repo = tmp_path / "repo"
    _fixture(repo, "import services", "numpy==2")
    (repo / "product" / "services.py").write_text("", encoding="utf-8")
    # Untracked files are deliberately not declaration evidence.
    from test_dependency_reconciliation import _git
    _git(repo, "add", "product/services.py")
    report = analyze_dependencies(repo)
    item = report["items"][0]
    assert item["actionability"] == "RESOLVER_LIMITATION"
    assert item["remediation_candidate"] is None


def test_dynamic_function_alias_context_and_platform_provenance(tmp_path: Path):
    repo = tmp_path / "repo"
    _fixture(repo, "from importlib import import_module as load\n"
             "def lookup(name):\n    try:\n        return load(name)\n"
             "    except ImportError:\n        return None\n", "numpy==2")
    report = analyze_dependencies(repo)
    platform, dynamic = report["items"][1], report["items"][0]
    assert platform["dependency"]["target"] == "importlib"
    assert platform["declaration"]["found"] is False
    assert dynamic["actionability"] == "REVIEW_REQUIRED"
    assert dynamic["usage"]["import_style"] == "lazy"
    assert dynamic["usage"]["fallback"]["detected"] is True


@pytest.mark.parametrize("import_name,distribution", [("yaml", "PyYAML"), ("PIL", "Pillow")])
def test_declared_distribution_with_unresolved_alias_is_resolver_queue(tmp_path, import_name, distribution):
    repo = tmp_path / "repo"
    _fixture(repo, f"import {import_name}", "numpy==2")
    (repo / "pyproject.toml").write_text(f'[project]\ndependencies = ["{distribution}"]\n', encoding="utf-8")
    from test_dependency_reconciliation import _git
    _git(repo, "add", "pyproject.toml")
    report = analyze_dependencies(repo)
    item = report["items"][0]
    assert item["actionability"] == "RESOLVER_LIMITATION"
    assert item["declaration"]["found"] is True
    assert item["remediation_candidate"] is None
    assert report["summary"]["blocking_after_reconciliation"] == 1


def test_owned_declaration_outside_reconciliation_scope_is_not_missing(tmp_path):
    repo = tmp_path / "repo"
    _fixture(repo, "from PIL import Image", "numpy==2")
    (repo / "install/requirements.txt").write_text("Pillow==12\n", encoding="utf-8")
    from test_dependency_reconciliation import _git
    _git(repo, "add", "install/requirements.txt")
    report = analyze_dependencies(repo)
    item = report["items"][0]
    assert item["actionability"] == "RESOLVER_LIMITATION"
    assert item["declaration"]["unresolved_candidate"]["owned_requirement_paths"] == ["install/requirements.txt"]
    assert item["remediation_candidate"] is None
    assert report["summary"]["blocking_after_reconciliation"] == 1


def test_manifest_read_issue_does_not_hide_dynamic_or_guarded_review(tmp_path):
    repo = tmp_path / "repo"
    _fixture(repo, 'import missing_package\n__import__(module_name)\n'
             'try:\n    import optional_package\nexcept ImportError:\n    pass', "numpy==2")
    (repo / "pyproject.toml").write_text("[invalid toml", encoding="utf-8")
    from test_dependency_reconciliation import _git
    _git(repo, "add", "pyproject.toml")
    report = analyze_dependencies(repo)
    items = {item["dependency"]["target"]: item for item in report["items"]}
    assert items["missing_package"]["actionability"] == "RESOLVER_LIMITATION"
    assert items["missing_package"]["remediation_candidate"] is None
    assert items["<dynamic-import>"]["actionability"] == "REVIEW_REQUIRED"
    assert items["optional_package"]["actionability"] == "REVIEW_REQUIRED"
