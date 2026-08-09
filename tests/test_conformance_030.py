from __future__ import annotations

import json
import subprocess
from pathlib import Path

from jsonschema import Draft202012Validator

from ptsip.cli import main
from ptsip.conformance import evaluate_conformance


SPEC_REVISION = "14a0c2f54bb486de6a109979224f998b04fd04a3"


def _git(repo: Path, *args: str) -> None:
    subprocess.run(["git", "-C", str(repo), *args], check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)


def _init_git(repo: Path) -> None:
    repo.mkdir(parents=True, exist_ok=True)
    _git(repo, "init")
    _git(repo, "config", "user.email", "ptsip-test@example.invalid")
    _git(repo, "config", "user.name", "PTSIP Test")


def _commit_all(repo: Path) -> None:
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")


def _write_profile(repo: Path, product_import: str | None = None, tool_import: str | None = None) -> None:
    (repo / "product").mkdir(exist_ok=True)
    (repo / "tools").mkdir(exist_ok=True)
    (repo / "product" / "app.py").write_text((product_import or "") + "VALUE = 1\n", encoding="utf-8")
    (repo / "tools" / "check.py").write_text((tool_import or "") + "VALUE = 2\n", encoding="utf-8")
    (repo / "ptsip.yaml").write_text(
        f"""ptsip:\n  version: \"0.2.0-draft\"\n  specification:\n    source: \"https://github.com/kwaksinwoo01/ptsip\"\n    revision: \"{SPEC_REVISION}\"\ncomponents:\n  - id: product\n    classification: PRODUCT\n    include: [\"product/**\"]\n    purpose: product_runtime\n  - id: tools\n    classification: TOOLCHAIN\n    include: [\"tools/**\"]\n    purpose: development_tooling\npolicies:\n  product_to_toolchain_runtime_dependency: deny\n  toolchain_in_product_package: deny\n  independent_build_resolution: required\n""",
        encoding="utf-8",
    )


def test_conform_without_profile_is_incomplete(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    (repo / "app.py").write_text("VALUE = 1\n", encoding="utf-8")
    _commit_all(repo)

    result = evaluate_conformance(repo)
    assert result.outcome == "INCOMPLETE"
    assert result.report["format"] == "ptsip-conformance-report/v1"
    assert result.report["tool"]["version"] == "0.3.0"
    assert result.report["evaluators"]["declared_dependency_boundaries"]["status"] == "BLOCKED"
    gap_ids = {item["id"] for item in result.report["coverage"]["blocking_gaps"]}
    assert "profile:missing" in gap_ids
    assert "artifact-evidence:not-inspected" in gap_ids
    assert "build-resolution:coverage" in gap_ids


def test_product_to_toolchain_unknown_phase_is_blocking_diagnostic(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    _write_profile(repo, product_import="import tools.check\n")
    _commit_all(repo)

    result = evaluate_conformance(repo)
    assert result.outcome == "INCOMPLETE"
    diagnostic = next(item for item in result.report["diagnostics"] if item["rule_id"] == "PTSIP-DEP-001")
    assert diagnostic["outcome_effect"] == "INCOMPLETE"
    assert diagnostic["source_component"] == "product"
    assert diagnostic["target_component"] == "tools"

    schema = json.loads(
        (Path(__file__).resolve().parents[1] / "schemas" / "ptsip-diagnostic.schema.json").read_text(encoding="utf-8")
    )
    Draft202012Validator(schema).validate(diagnostic)


def test_toolchain_to_product_review_is_nonblocking_diagnostic(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    _write_profile(repo, tool_import="import product.app\n")
    _commit_all(repo)

    result = evaluate_conformance(repo)
    diagnostic = next(item for item in result.report["diagnostics"] if item["rule_id"] == "PTSIP-DEP-002")
    assert diagnostic["outcome_effect"] == "NONE"
    assert diagnostic["severity"] == "INFO"


def test_cli_conform_uses_distinct_incomplete_exit_code(tmp_path: Path, capsys) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    _write_profile(repo)
    _commit_all(repo)

    assert main(["conform", str(repo), "--json"]) == 6
    payload = json.loads(capsys.readouterr().out)
    assert payload["outcome"] == "INCOMPLETE"
    assert payload["level"] == "ENFORCED"
    assert payload["snapshot"]["comparison"]["stable"] is True
