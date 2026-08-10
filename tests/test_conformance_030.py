from __future__ import annotations

import json
import subprocess
from pathlib import Path

from jsonschema import Draft202012Validator

from ptsip.cli import main
from ptsip.conformance import evaluate_conformance
from ptsip.validation.profile import validate_profile


SPEC_REVISION = "a877b2f66a7f94c1b844c979e1b08fb08a9a8e45"


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


def _write_profile(
    repo: Path,
    product_import: str | None = None,
    tool_import: str | None = None,
    component_policy: str = "",
) -> None:
    (repo / "product").mkdir(exist_ok=True)
    (repo / "tools").mkdir(exist_ok=True)
    (repo / "product" / "app.py").write_text((product_import or "") + "VALUE = 1\n", encoding="utf-8")
    (repo / "tools" / "check.py").write_text((tool_import or "") + "VALUE = 2\n", encoding="utf-8")
    (repo / "ptsip.yaml").write_text(
        f"""ptsip:\n  version: \"0.2.0-draft\"\n  specification:\n    source: \"https://github.com/kwaksinwoo01/ptsip\"\n    revision: \"{SPEC_REVISION}\"\ncomponents:\n  - id: product\n    classification: PRODUCT\n    include: [\"product/**\"]\n    purpose: product_runtime\n  - id: tools\n    classification: TOOLCHAIN\n    include: [\"tools/**\"]\n    purpose: development_tooling\n{component_policy}policies:\n  product_to_toolchain_runtime_dependency: deny\n  toolchain_in_product_package: deny\n  independent_build_resolution: required\n""",
        encoding="utf-8",
    )


def _write_artifact(
    path: Path,
    *,
    producer: str | None = "tools",
    components: list[str] | None = None,
    complete: bool = True,
) -> None:
    path.write_text(
        json.dumps(
            {
                "format": "ptsip-artifact-evidence/v1",
                "artifact_id": "product-dist",
                "classification": "PRODUCT",
                "producer_component": producer,
                "artifact_type": "test-bundle",
                "shipping_scope": "product-distribution",
                "contents": {
                    "paths": ["product/app.py"],
                    "components": components if components is not None else ["product"],
                    "complete": complete,
                },
                "derivation": [{"relation": "GENERATES", "source": "tools"}],
                "provenance": "OBSERVED",
                "evidence_ids": ["artifact:test:product-dist"],
            }
        ),
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
    assert result.report["tool"]["version"] == "0.3.1"
    assert result.report["evaluators"]["declared_dependency_boundaries"]["status"] == "BLOCKED"
    gap_ids = {item["id"] for item in result.report["coverage"]["blocking_gaps"]}
    assert "profile:missing" in gap_ids
    assert "artifact-evidence:product-missing" in gap_ids
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


def test_toolchain_producer_does_not_make_product_artifact_nonconformant(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    _write_profile(repo)
    artifact = tmp_path / "artifact.json"
    _write_artifact(artifact, producer="tools", components=["product"])
    _commit_all(repo)

    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert not any(item["rule_id"] == "PTSIP-PKG-001" for item in result.report["diagnostics"])
    assert result.report["evaluators"]["product_artifact_boundary"]["status"] == "RAN"
    assert result.outcome == "INCOMPLETE"


def test_product_artifact_with_toolchain_content_is_nonconformant(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    _write_profile(repo)
    artifact = tmp_path / "artifact.json"
    _write_artifact(artifact, producer="tools", components=["product", "tools"])
    _commit_all(repo)

    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    assert result.outcome == "NON_CONFORMANT"
    diagnostic = next(item for item in result.report["diagnostics"] if item["rule_id"] == "PTSIP-PKG-001")
    assert diagnostic["outcome_effect"] == "NON_CONFORMANT"
    assert diagnostic["evidence_ids"] == ["artifact:test:product-dist"]


def test_incomplete_product_artifact_blocks_conformance(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    _write_profile(repo)
    artifact = tmp_path / "artifact.json"
    _write_artifact(artifact, complete=False)
    _commit_all(repo)

    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])
    gap_ids = {item["id"] for item in result.report["coverage"]["blocking_gaps"]}
    assert "artifact-evidence:product-dist:contents-incomplete" in gap_ids


def test_component_dependency_policy_violation_is_reported_separately(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    _write_profile(
        repo,
        tool_import="import product.app\n",
        component_policy=(
            "component_dependency_policy:\n"
            "  default: allow\n"
            "  deny:\n"
            "    - from: tools\n"
            "      to: product\n"
        ),
    )
    _commit_all(repo)

    result = evaluate_conformance(repo)
    assert result.report["project_policy"]["finding_count"] == 1
    finding = result.report["project_policy"]["findings"][0]
    assert finding["source_component"] == "tools"
    assert finding["target_component"] == "product"
    assert result.report["project_policy"]["affects_ptsip_outcome"] is False
    assert not any(item["rule_id"] == "PTSIP-POL-001" for item in result.report["diagnostics"])
    assert result.outcome == "INCOMPLETE"


def test_component_dependency_policy_allow_deny_conflict_is_invalid_profile(tmp_path: Path) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    _write_profile(
        repo,
        component_policy=(
            "component_dependency_policy:\n"
            "  default: deny\n"
            "  allow:\n"
            "    - from: tools\n"
            "      to: product\n"
            "  deny:\n"
            "    - from: tools\n"
            "      to: product\n"
        ),
    )
    _commit_all(repo)

    result = validate_profile(repo)
    assert not result.valid
    assert any("appears in both allow and deny" in error for error in result.errors)


def test_cli_conform_accepts_repeatable_artifact_evidence(tmp_path: Path, capsys) -> None:
    repo = tmp_path / "repo"
    _init_git(repo)
    _write_profile(repo)
    artifact = tmp_path / "artifact.json"
    _write_artifact(artifact)
    _commit_all(repo)

    assert main(["conform", str(repo), "--artifact-evidence", str(artifact), "--json"]) == 6
    payload = json.loads(capsys.readouterr().out)
    assert payload["outcome"] == "INCOMPLETE"
    assert payload["artifacts"]["document_count"] == 1
    assert payload["evaluators"]["product_artifact_boundary"]["status"] == "RAN"


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
