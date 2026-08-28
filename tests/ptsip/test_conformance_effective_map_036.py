from __future__ import annotations

import copy
import json
import subprocess
from pathlib import Path

import yaml

from ptsip.conformance import evaluate_conformance
from ptsip.constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION
from ptsip.profile_identity import CURRENT_PROJECT_PROFILE_VERSION
from ptsip.validation.templates import template_catalog


POLICIES = {
    "product_to_nonproduct_runtime_dependency": "deny",
    "nonproduct_in_product_package": "deny",
    "independent_build_resolution": "required",
}


def _git(repo: Path, *args: str) -> str:
    return subprocess.run(
        ["git", "-C", str(repo), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    ).stdout.strip()


def _repo(tmp_path: Path, files: dict[str, str]) -> Path:
    repo = tmp_path / "repo"
    repo.mkdir(parents=True)
    _git(repo, "init")
    _git(repo, "config", "user.email", "ptsip-test@example.invalid")
    _git(repo, "config", "user.name", "PTSIP Test")
    for relative, content in files.items():
        path = repo / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")
    return repo


def _profile(mode: str, template_id: str | None = None, revision: str | None = None) -> dict[str, object]:
    responsibility_map: dict[str, object] = {"mode": mode}
    if template_id is not None and revision is not None:
        responsibility_map["template"] = {"id": template_id, "revision": revision}
    return {
        "ptsip": {
            "version": CURRENT_PROJECT_PROFILE_VERSION,
            "specification": {
                "family": SPEC_VERSION,
                "source": SPEC_SOURCE,
                "revision": SPEC_REVISION,
            },
        },
        "responsibility_map": responsibility_map,
        "policies": copy.deepcopy(POLICIES),
    }


def _write_profile(repo: Path, payload: dict[str, object], message: str) -> None:
    (repo / "ptsip.yaml").write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8")
    _git(repo, "add", "ptsip.yaml")
    _git(repo, "commit", "-m", message)


def _artifact(
    path: Path,
    *,
    producer: str,
    components: list[str],
    product_path: str,
) -> None:
    path.write_text(
        json.dumps(
            {
                "format": "ptsip-artifact-evidence/v1",
                "artifact_id": "product-dist",
                "classification": "PRODUCT",
                "producer_component": producer,
                "artifact_type": "fixture",
                "shipping_scope": "product-distribution",
                "contents": {
                    "paths": [product_path],
                    "components": components,
                    "complete": True,
                },
                "derivation": [{"relation": "GENERATES", "source": producer}],
                "provenance": "OBSERVED",
                "evidence_ids": ["artifact:wu04f:product-dist"],
            }
        ),
        encoding="utf-8",
    )


def _gap_ids(result) -> set[str]:
    coverage = result.report["coverage"]
    return {
        str(item["id"])
        for group in ("blocking_gaps", "non_blocking_gaps")
        for item in coverage[group]
    }


def test_template_profile_reaches_conformance_evaluators_without_source_mode_branch(tmp_path: Path) -> None:
    definition = next(item for item in template_catalog() if item.id == "mixed-product-development-delivery")
    repo = _repo(
        tmp_path,
        {
            "src/app.py": "VALUE = 1\n",
            "tests/test_app.py": "def test_app():\n    assert True\n",
            ".github/workflows/release.yml": "name: release\non: workflow_dispatch\njobs: {}\n",
        },
    )
    payload = _profile("template", definition.id, definition.revision)
    _write_profile(repo, payload, "template profile")
    artifact = tmp_path / "artifact.json"
    _artifact(artifact, producer="delivery", components=["product"], product_path="src/app.py")

    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])

    assert result.report["profile"]["valid"] is True
    assert result.report["profile"]["details"]["resolution"]["source_mode"] == "template"
    assert result.report["evaluators"]["declared_dependency_boundaries"]["status"] == "RAN"
    assert result.report["evaluators"]["product_artifact_boundary"]["status"] == "RAN"
    assert "ownership:materialization-required" not in _gap_ids(result)
    assert "profile:resolution-unavailable" not in _gap_ids(result)
    assert not any("unknown-producer" in item for item in _gap_ids(result))


def test_hybrid_conformance_uses_overridden_and_removed_effective_components(tmp_path: Path) -> None:
    definition = next(item for item in template_catalog() if item.id == "mixed-product-development-delivery")
    repo = _repo(
        tmp_path,
        {
            "src/app.py": "VALUE = 1\n",
            "verification/test_app.py": "def test_app():\n    assert True\n",
        },
    )
    payload = _profile("hybrid", definition.id, definition.revision)
    payload["responsibility_map"]["overrides"] = {
        "components": [
            {
                "id": "development-verification",
                "classification": "DEVELOPMENT_TOOLING",
                "roles": ["VERIFICATION", "AUTOMATION"],
                "include": ["verification/**"],
                "purpose": "custom_reusable_verification",
                "shipped": False,
                "runtime_required": False,
                "executable": True,
            }
        ],
        "remove_component_ids": ["delivery"],
        "remove_relationship_ids": ["delivery-builds-product", "delivery-publishes-product"],
    }
    _write_profile(repo, payload, "hybrid profile")
    artifact = tmp_path / "artifact.json"
    _artifact(
        artifact,
        producer="development-verification",
        components=["product"],
        product_path="src/app.py",
    )

    result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])

    assert result.report["profile"]["valid"] is True
    details = result.report["profile"]["details"]
    assert details["resolution"]["source_mode"] == "hybrid"
    assert details["resolution_provenance"]["components"]["development-verification"] == "PROJECT_OVERRIDE"
    assert details["resolution_provenance"]["removals"]["components"] == ["delivery"]
    assert result.report["evaluators"]["declared_dependency_boundaries"]["status"] == "RAN"
    assert result.report["evaluators"]["product_artifact_boundary"]["status"] == "RAN"
    assert not any("unknown-producer" in item for item in _gap_ids(result))
    assert "ownership:materialization-required" not in _gap_ids(result)


def test_equivalent_explicit_and_template_profiles_share_architecture_evaluation(tmp_path: Path) -> None:
    definition = next(item for item in template_catalog() if item.id == "python-package-library")
    repo = _repo(
        tmp_path,
        {
            "src/package.py": "VALUE = 1\n",
            "tests/test_package.py": "def test_value():\n    assert True\n",
        },
    )
    artifact = tmp_path / "artifact.json"
    _artifact(
        artifact,
        producer="package-tests",
        components=["package"],
        product_path="src/package.py",
    )

    template_payload = _profile("template", definition.id, definition.revision)
    _write_profile(repo, template_payload, "template profile")
    template_result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])

    explicit_payload = _profile("explicit")
    explicit_payload.update(copy.deepcopy(definition.map_payload))
    _write_profile(repo, explicit_payload, "explicit profile")
    explicit_result = evaluate_conformance(repo, artifact_evidence_paths=[artifact])

    assert template_result.report["profile"]["valid"] is True
    assert explicit_result.report["profile"]["valid"] is True
    assert (
        template_result.report["profile"]["details"]["resolution"]["effective_map_digest"]
        == explicit_result.report["profile"]["details"]["resolution"]["effective_map_digest"]
    )
    for evaluator in ("declared_dependency_boundaries", "component_dependency_policy", "product_artifact_boundary"):
        assert template_result.report["evaluators"][evaluator] == explicit_result.report["evaluators"][evaluator]
    assert _gap_ids(template_result) == _gap_ids(explicit_result)
    assert template_result.report["diagnostics"] == explicit_result.report["diagnostics"]


def test_invalid_template_binding_stays_fail_closed_before_architecture_evaluation(tmp_path: Path) -> None:
    definition = next(item for item in template_catalog() if item.id == "python-package-library")
    repo = _repo(tmp_path, {"src/package.py": "VALUE = 1\n"})
    payload = _profile("template", definition.id, "sha256:" + "0" * 64)
    _write_profile(repo, payload, "invalid template profile")

    result = evaluate_conformance(repo)

    assert result.report["profile"]["valid"] is False
    assert result.report["evaluators"]["declared_dependency_boundaries"] == {
        "status": "BLOCKED",
        "reason": "INVALID_PROFILE",
    }
    assert "profile:invalid" in _gap_ids(result)
