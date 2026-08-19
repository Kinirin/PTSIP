from __future__ import annotations

from pathlib import Path

import pytest

from ptsip.clarification.generator import analyze_clarifications
from ptsip.clarification.generator_core import build_requests
from ptsip.inspection.components import ComponentCandidate
from ptsip.validation.components import AMBIGUOUS, resolve_candidate_coverage
from _wu04g_support import (
    associated_artifact_payload,
    commit_all,
    component_payload,
    explicit_profile_payload,
    hybrid_profile_payload,
    init_git_repo,
    template_profile_payload,
    write_profile,
    write_text,
)


def _template_equivalent_components() -> list[dict[str, object]]:
    return [
        component_payload(
            "package",
            ["src/**"],
            classification="PRODUCT",
            purpose="python_package_library_implementation",
            shipped=True,
            runtime_required=True,
            executable=True,
            roles=["IMPLEMENTATION"],
        ),
        component_payload(
            "package-tests",
            ["tests/**"],
            classification="PRODUCT",
            purpose="product_owned_package_verification",
            shipped=False,
            runtime_required=False,
            executable=True,
            roles=["VERIFICATION"],
        ),
    ]


def _template_relationship() -> dict[str, object]:
    return {
        "id": "package-tests-verify-package",
        "from": "package-tests",
        "to": "package",
        "type": "VERIFIES",
    }


def _package_repo(tmp_path: Path, *, test_root: str = "tests") -> Path:
    repo = init_git_repo(tmp_path / "repo")
    write_text(repo, "src/package.py", "VALUE = 1\n")
    write_text(repo, f"{test_root}/test_package.py", "def test_value(): assert True\n")
    return repo


def _write_mode_profile(repo: Path, mode: str) -> None:
    if mode == "explicit":
        payload = explicit_profile_payload(
            _template_equivalent_components(),
            relationships=[_template_relationship()],
        )
    elif mode == "template":
        payload = template_profile_payload()
    elif mode == "hybrid":
        payload = hybrid_profile_payload(
            {"components": [_template_equivalent_components()[1]]}
        )
    else:  # pragma: no cover - test helper invariant
        raise AssertionError(mode)
    write_profile(repo / "ptsip.yaml", payload)


class TestG1EffectiveReadCoverage:
    @pytest.mark.parametrize("mode", ["explicit", "template", "hybrid"])
    def test_equivalent_component_coverage_suppresses_clarification(
        self, tmp_path: Path, mode: str
    ) -> None:
        repo = _package_repo(tmp_path)
        _write_mode_profile(repo, mode)
        commit_all(repo)

        analysis = analyze_clarifications(repo, ["tests"])

        assert analysis.profile_parse_error is None
        assert analysis.status == "NO_CLARIFICATION_REQUIRED"
        assert analysis.requests == ()

    def test_template_component_is_not_reasked(self, tmp_path: Path) -> None:
        repo = _package_repo(tmp_path)
        write_profile(repo / "ptsip.yaml", template_profile_payload())
        commit_all(repo)

        analysis = analyze_clarifications(repo, ["tests"])

        assert analysis.status == "NO_CLARIFICATION_REQUIRED"
        assert analysis.requests == ()

    def test_hybrid_override_selector_is_effective_selector(self, tmp_path: Path) -> None:
        repo = _package_repo(tmp_path, test_root="testing")
        overridden_tests = component_payload(
            "package-tests",
            ["testing/**"],
            classification="PRODUCT",
            purpose="product_owned_package_verification",
            shipped=False,
            runtime_required=False,
            executable=True,
            roles=["VERIFICATION"],
        )
        write_profile(
            repo / "ptsip.yaml",
            hybrid_profile_payload({"components": [overridden_tests]}),
        )
        commit_all(repo)

        analysis = analyze_clarifications(repo, ["testing"])

        assert analysis.profile_parse_error is None
        assert analysis.status == "NO_CLARIFICATION_REQUIRED"
        assert analysis.requests == ()

    def test_hybrid_removal_exposes_uncovered_candidate(self, tmp_path: Path) -> None:
        repo = _package_repo(tmp_path)
        write_profile(
            repo / "ptsip.yaml",
            hybrid_profile_payload(
                {
                    "remove_component_ids": ["package-tests"],
                    "remove_relationship_ids": ["package-tests-verify-package"],
                }
            ),
        )
        commit_all(repo)

        analysis = analyze_clarifications(repo, ["tests"])

        assert analysis.profile_parse_error is None
        assert analysis.status == "CLARIFICATION_REQUIRED"
        assert len(analysis.requests) == 1
        assert analysis.requests[0].component_id == "tests"

    def test_effective_associated_artifact_suppresses_component_question(
        self, tmp_path: Path
    ) -> None:
        repo = init_git_repo(tmp_path / "repo")
        write_text(repo, "src/sdk.py", "VALUE = 1\n")
        write_text(repo, "tools/generate.py", "print('generate')\n")
        sdk = component_payload(
            "sdk",
            ["src/**"],
            classification="PRODUCT",
            purpose="sdk_runtime",
            shipped=True,
            runtime_required=True,
            executable=True,
        )
        artifact = associated_artifact_payload(
            "sdk-support", "sdk", ["tools/**"], purpose="sdk_support_surface"
        )
        write_profile(
            repo / "ptsip.yaml",
            explicit_profile_payload(
                [sdk],
                associated_artifacts=[artifact],
                relationships=[
                    {
                        "id": "sdk-support-documents-sdk",
                        "from": "sdk-support",
                        "to": "sdk",
                        "type": "DOCUMENTS",
                    }
                ],
            ),
        )
        commit_all(repo)

        analysis = analyze_clarifications(repo, ["tools"])

        assert analysis.profile_parse_error is None
        assert analysis.status == "NO_CLARIFICATION_REQUIRED"
        assert analysis.requests == ()

    def test_selector_ambiguity_fails_review_instead_of_guessing(self) -> None:
        candidate = ComponentCandidate(
            id="tools",
            include=("tools/**",),
            anchors=("top-level-tool-root",),
            evidence_ids=("root:tools",),
        )
        components = [
            component_payload("alpha", ["tools/**"], purpose="alpha_tools"),
            component_payload("beta", ["tools/**"], purpose="beta_tools"),
        ]

        coverage = resolve_candidate_coverage(candidate, components)
        requests = build_requests("example/repo", [candidate], components)

        assert coverage.status == AMBIGUOUS
        assert coverage.owner_ids == ("alpha", "beta")
        assert len(requests) == 1
        assert requests[0].component_id == "tools"


class TestG2DecisionProtocolV2:
    """Reserved namespace; G2 has not been entered."""


class TestG3HybridSafeApply:
    """Reserved namespace; G3 has not been entered."""


class TestG4ProfilePathControlPlane:
    """Reserved namespace; G4 has not been entered."""


class TestG5RecoveryAndIntegration:
    """Reserved namespace; G5 has not been entered."""
