from __future__ import annotations

from pathlib import Path
from types import SimpleNamespace

import yaml

from ptsip.inspection.candidate_evidence import (
    discover_candidate_evidence,
    validate_candidate_discovery_context,
)
from ptsip.model import DependencyPhase, EdgeType, EvidenceProvenance
from ptsip.validation.components import AMBIGUOUS, COMPONENT_COVERED, UNCOVERED


def _dependencies(*edges):
    return SimpleNamespace(edges=tuple(edges), issues=())


def _edge(
    *,
    evidence_id: str,
    source: str,
    resolved_path: str | None,
    adapter: str = "github-actions",
    phase: DependencyPhase = DependencyPhase.TEST,
    edge_type: EdgeType = EdgeType.INVOKES,
):
    return SimpleNamespace(
        evidence_id=evidence_id,
        source=source,
        resolved_path=resolved_path,
        adapter=adapter,
        phase=phase,
        edge_type=edge_type,
        provenance=EvidenceProvenance.OBSERVED,
    )


def _write_profile(
    root: Path,
    filename: str,
    version: str,
    *,
    revision: str,
    components: list[dict[str, object]] | None = None,
    associated_artifacts: list[dict[str, object]] | None = None,
) -> None:
    payload: dict[str, object] = {
        "ptsip": {
            "version": version,
            "specification": {
                "source": "https://github.com/Kinirin/PTSIP",
                "revision": revision,
            },
        },
        "components": components or [],
        "associated_artifacts": associated_artifacts or [],
    }
    (root / filename).write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8")


def _base_repository(root: Path, *, components: list[dict[str, object]] | None = None) -> None:
    (root / "src/pkg").mkdir(parents=True)
    (root / "src/pkg/__init__.py").write_text("", encoding="utf-8")
    (root / "tests").mkdir()
    (root / "tests/test_sample.py").write_text("def test_sample():\n    assert True\n", encoding="utf-8")
    (root / "schemas").mkdir()
    (root / "schemas/example.schema.json").write_text("{}\n", encoding="utf-8")
    (root / "pyproject.toml").write_text(
        "[project]\nname='fixture'\nversion='0.0.0'\n",
        encoding="utf-8",
    )
    _write_profile(
        root,
        "ptsip.yaml",
        "0.3.6-draft",
        revision="6" * 40,
        components=components
        or [
            {"id": "product", "include": ["src/pkg/**", "pyproject.toml"]},
            {"id": "verification", "include": ["tests/**"]},
            {"id": "contracts", "include": ["schemas/**"]},
        ],
    )


def _discover(root: Path, *, source_profile_path: str | None = None, edges=()):
    return discover_candidate_evidence(
        root,
        source_profile_path=source_profile_path,
        dependencies=_dependencies(*edges),
    )


def _candidate_by_include(result, selector: str):
    return next(item for item in result.candidates if item.include == (selector,))


def test_positive_discovery_uses_shared_coverage_and_stable_candidate_ids(tmp_path: Path) -> None:
    _base_repository(tmp_path)

    first = _discover(tmp_path)
    second = _discover(tmp_path)

    assert first.complete is True
    assert second.complete is True
    assert [item.id for item in first.candidates] == [item.id for item in second.candidates]
    assert first.context is not None and second.context is not None
    assert first.context.evaluation_id == second.context.evaluation_id

    source = _candidate_by_include(first, "src/pkg/**")
    tests = _candidate_by_include(first, "tests/**")
    contracts = _candidate_by_include(first, "schemas/**")
    assert source.coverage.status == COMPONENT_COVERED
    assert tests.coverage.status == COMPONENT_COVERED
    assert contracts.coverage.status == COMPONENT_COVERED


def test_duplicate_observations_converge_on_one_candidate_identity(tmp_path: Path) -> None:
    _base_repository(tmp_path)

    result = _discover(tmp_path)

    pyproject = _candidate_by_include(result, "pyproject.toml")
    kinds = {item.kind for item in pyproject.observations}
    assert {"MANIFEST", "PACKAGE_ASSEMBLY_INPUT"} <= kinds
    assert sum(item.include == ("pyproject.toml",) for item in result.candidates) == 1


def test_ambiguous_declared_coverage_stays_explicit_instead_of_guessing(tmp_path: Path) -> None:
    components = [
        {"id": "verification-a", "include": ["tests/**"]},
        {"id": "verification-b", "include": ["tests/**"]},
    ]
    _base_repository(tmp_path, components=components)

    result = _discover(tmp_path)

    candidate = _candidate_by_include(result, "tests/**")
    assert candidate.coverage.status == AMBIGUOUS
    assert candidate.ambiguous is True
    assert candidate.coverage.owner_ids == ("verification-a", "verification-b")


def test_uncovered_candidate_remains_evidence_without_automatic_classification(tmp_path: Path) -> None:
    _base_repository(tmp_path, components=[{"id": "product", "include": ["src/pkg/**"]}])

    result = _discover(tmp_path)

    candidate = _candidate_by_include(result, "tests/**")
    payload = candidate.as_dict()
    assert candidate.coverage.status == UNCOVERED
    assert payload["authority"] == "EVIDENCE_ONLY"
    assert "classification" not in payload
    assert "obligation" not in payload
    assert "required_work" not in payload


def test_ci_invoked_script_and_maintenance_signal_merge_without_ownership_guess(tmp_path: Path) -> None:
    _base_repository(tmp_path)
    (tmp_path / ".github/scripts").mkdir(parents=True)
    (tmp_path / ".github/scripts/check.py").write_text("print('ok')\n", encoding="utf-8")
    edge = _edge(
        evidence_id="github-actions:.github/workflows/test.yml:check",
        source=".github/workflows/test.yml",
        resolved_path=".github/scripts/check.py",
    )

    result = _discover(tmp_path, edges=(edge,))

    candidate = _candidate_by_include(result, ".github/scripts/check.py")
    kinds = {item.kind for item in candidate.observations}
    assert {"CI_INVOKED_SCRIPT", "MAINTENANCE_SCRIPT", "DEPENDENCY_BOUNDARY"} <= kinds
    assert candidate.coverage.status == UNCOVERED


def test_release_phase_and_cross_root_dependency_are_descriptive_only(tmp_path: Path) -> None:
    _base_repository(tmp_path)
    (tmp_path / ".github/workflows").mkdir(parents=True)
    (tmp_path / ".github/workflows/release.yml").write_text("name: release\n", encoding="utf-8")
    edge = _edge(
        evidence_id="release-edge",
        source=".github/workflows/release.yml",
        resolved_path="src/pkg/__init__.py",
        phase=DependencyPhase.RELEASE,
    )

    result = _discover(tmp_path, edges=(edge,))

    source = _candidate_by_include(result, ".github/workflows/release.yml")
    target = _candidate_by_include(result, "src/pkg/__init__.py")
    assert "RELEASE_ACTIVITY_SOURCE" in {item.kind for item in source.observations}
    assert {"RELEASE_ACTIVITY_TARGET", "DEPENDENCY_BOUNDARY"} <= {item.kind for item in target.observations}
    assert source.as_dict()["authority"] == "EVIDENCE_ONLY"
    assert target.as_dict()["authority"] == "EVIDENCE_ONLY"


def test_embedded_contract_copy_is_observed_as_its_own_scope(tmp_path: Path) -> None:
    _base_repository(tmp_path)
    (tmp_path / "src/pkg/specdata").mkdir()
    (tmp_path / "src/pkg/specdata/profile.schema.json").write_text("{}\n", encoding="utf-8")

    result = _discover(tmp_path)

    candidate = _candidate_by_include(result, "src/pkg/specdata/**")
    assert "EMBEDDED_CONTRACT_COPY" in {item.kind for item in candidate.observations}


def test_multi_generation_evaluation_reuses_candidate_identity_but_not_source_context(tmp_path: Path) -> None:
    _base_repository(tmp_path)
    canonical_payload = yaml.safe_load((tmp_path / "ptsip.yaml").read_text(encoding="utf-8"))
    _write_profile(
        tmp_path,
        "ptsip_0.3.7.yaml",
        "0.3.7-draft",
        revision="7" * 40,
        components=canonical_payload["components"],
    )
    _write_profile(
        tmp_path,
        "ptsip_0.4.0.yaml",
        "0.4.0-draft",
        revision="4" * 40,
        components=canonical_payload["components"],
    )

    newer_source = _discover(tmp_path, source_profile_path="ptsip_0.3.7.yaml")
    canonical_source = _discover(tmp_path, source_profile_path="ptsip.yaml")

    assert newer_source.complete is True
    assert canonical_source.complete is True
    assert [item.id for item in newer_source.candidates] == [item.id for item in canonical_source.candidates]
    assert newer_source.context is not None and canonical_source.context is not None
    assert newer_source.context.source_generation.profile_path == "ptsip_0.3.7.yaml"
    assert canonical_source.context.source_generation.profile_path == "ptsip.yaml"
    assert newer_source.context.evaluation_id != canonical_source.context.evaluation_id


def test_final_point_cannot_be_selected_as_source_generation(tmp_path: Path) -> None:
    _base_repository(tmp_path)
    canonical_payload = yaml.safe_load((tmp_path / "ptsip.yaml").read_text(encoding="utf-8"))
    _write_profile(
        tmp_path,
        "ptsip_0.3.7.yaml",
        "0.3.7-draft",
        revision="7" * 40,
        components=canonical_payload["components"],
    )

    result = _discover(tmp_path, source_profile_path="ptsip_0.3.7.yaml")

    assert result.complete is False
    assert result.candidates == ()
    assert [item.code for item in result.issues] == ["FINAL_POINT_IS_NOT_SOURCE"]


def test_discovery_context_detects_repository_change_after_collection(tmp_path: Path) -> None:
    _base_repository(tmp_path)
    result = _discover(tmp_path)
    assert result.complete is True
    assert validate_candidate_discovery_context(tmp_path, result) == ()

    (tmp_path / "src/pkg/__init__.py").write_text("VALUE = 1\n", encoding="utf-8")

    issues = validate_candidate_discovery_context(tmp_path, result)
    assert [item.code for item in issues] == ["STALE_TRANSITION_SNAPSHOT"]
