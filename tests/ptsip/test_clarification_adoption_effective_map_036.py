from __future__ import annotations

from pathlib import Path

import pytest
import yaml

from ptsip.app.store import DecisionStore
from ptsip.clarification.generator import analyze_clarifications
from ptsip.clarification.generator_core import build_requests
from ptsip.clarification.model import ClarificationRequest
from ptsip.clarification.render import render_issue
from ptsip.clarification.resolution import (
    DecisionAnswer,
    canonicalize_legacy_answer,
    parse_answer,
    parse_legacy_answer,
    prepare_local_profile,
    project_payload,
    validate_answer,
)
from ptsip.inspection.components import ComponentCandidate
from ptsip.validation.components import AMBIGUOUS, resolve_candidate_coverage
from ptsip.validation.templates import materialize_profile
from _wu04g_support import (
    associated_artifact_payload,
    canonical_v2_answer,
    clarification_answer_text,
    commit_all,
    component_payload,
    explicit_profile_payload,
    hybrid_profile_payload,
    init_git_repo,
    legacy_v1_answer,
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


def _g3_answer(**overrides: object) -> DecisionAnswer:
    payload = canonical_v2_answer(**overrides)
    return DecisionAnswer(**payload)


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
    def test_v2_parser_accepts_canonical_answer_without_lifecycle_owner(self) -> None:
        payload = canonical_v2_answer()
        answer = parse_answer(clarification_answer_text(payload))

        assert answer.as_dict() == payload
        assert "lifecycle_owner" not in answer.as_dict()
        assert validate_answer(answer).valid

    def test_v2_rejects_lifecycle_owner_field_when_contract_requires_exact_v2_shape(self) -> None:
        payload = legacy_v1_answer()
        text = clarification_answer_text(payload, format_name="ptsip-clarification-answer/v2")

        with pytest.raises(ValueError, match="unexpected: lifecycle_owner"):
            parse_answer(text)

    def test_v2_rejects_toolchain_classification(self) -> None:
        payload = canonical_v2_answer(classification="TOOLCHAIN")
        answer = parse_answer(clarification_answer_text(payload))

        validation = validate_answer(answer)
        assert validation.valid is False
        assert validation.status == "CONFLICT"
        assert any("classification must be" in error for error in validation.errors)

    def test_v1_reader_is_explicit_compatibility_only(self) -> None:
        text = clarification_answer_text(
            legacy_v1_answer(),
            format_name="ptsip-clarification-answer/v1",
        )

        with pytest.raises(ValueError):
            parse_answer(text)
        legacy = parse_legacy_answer(text)
        canonical = canonicalize_legacy_answer(legacy)
        assert canonical.as_dict() == canonical_v2_answer()
        assert "lifecycle_owner" not in canonical.as_dict()

    def test_v1_toolchain_is_not_auto_translated(self) -> None:
        text = clarification_answer_text(
            legacy_v1_answer(
                classification="TOOLCHAIN",
                lifecycle_owner="DEVELOPMENT_TOOLING",
            ),
            format_name="ptsip-clarification-answer/v1",
        )
        legacy = parse_legacy_answer(text)

        assert legacy.to_canonical().classification == "TOOLCHAIN"
        with pytest.raises(ValueError, match="classification must be"):
            canonicalize_legacy_answer(legacy)

    def test_new_rendered_clarification_uses_v2(self) -> None:
        request = ClarificationRequest(
            id="clr-test",
            component_id="tools",
            include=("tools/**",),
            anchors=("top-level-tool-root",),
            evidence_ids=("root:tools",),
            missing_fields=("classification", "purpose", "shipped", "runtime_required", "executable"),
            reason_codes=(
                "MISSING_CLASSIFICATION",
                "MISSING_PURPOSE",
                "MISSING_PACKAGING_RESPONSIBILITY",
                "MISSING_RUNTIME_ROLE",
                "MISSING_EXECUTABLE_ROLE",
            ),
        )

        _title, body = render_issue(request, "en", "abc123")

        assert "ptsip-clarification-answer/v2" in body
        structured = body.split("```yaml", 1)[1].split("```", 1)[0]
        assert "lifecycle_owner" not in structured
        assert "TOOLCHAIN" not in structured

    def test_new_stored_decision_uses_v2_semantics(self, tmp_path: Path) -> None:
        store = DecisionStore(tmp_path / "decision.sqlite3")
        request = {
            "component_id": "tools",
            "include": ["tools/**"],
            "anchors": ["top-level-tool-root"],
            "evidence_ids": ["root:tools"],
            "missing_fields": ["classification"],
            "reason_codes": ["MISSING_CLASSIFICATION"],
            "status": "INCOMPLETE",
        }
        record, _stale = store.gate(
            {
                "id": "clr-test",
                "repository": "local:test",
                "branch": "main",
                "subject_revision": "abc123",
                "component_id": "tools",
                "request": request,
            }
        )
        payload = canonical_v2_answer()

        resolved, accepted = store.resolve(record.id, payload, "AGENT_CHAT", "tester")

        assert accepted is True
        assert resolved.answer == payload
        assert set(resolved.answer or {}) == set(payload)
        assert "lifecycle_owner" not in (resolved.answer or {})


class TestG3HybridSafeApply:
    def test_explicit_apply_remains_canonical_baseline(self) -> None:
        source = explicit_profile_payload([])
        projected = project_payload(source, "tools", ["tools/**"], _g3_answer())

        assert source["components"] == []
        assert projected["responsibility_map"] == {"mode": "explicit"}
        assert projected["components"] == [component_payload(
            "tools",
            ["tools/**"],
            purpose="Repository-local generation tooling",
        )]

    def test_template_decision_converts_to_hybrid_preserving_template_identity(self) -> None:
        source = template_profile_payload()
        template_identity = dict(source["responsibility_map"]["template"])

        projected = project_payload(source, "tools", ["tools/**"], _g3_answer())
        map_meta = projected["responsibility_map"]

        assert map_meta["mode"] == "hybrid"
        assert map_meta["template"] == template_identity
        assert [item["id"] for item in map_meta["overrides"]["components"]] == ["tools"]
        assert "components" not in projected

    def test_template_to_hybrid_writes_only_accepted_project_delta(self) -> None:
        source = template_profile_payload()

        projected = project_payload(source, "tools", ["tools/**"], _g3_answer())
        overrides = projected["responsibility_map"]["overrides"]

        assert overrides == {
            "components": [
                component_payload(
                    "tools",
                    ["tools/**"],
                    purpose="Repository-local generation tooling",
                )
            ]
        }
        assert set(projected) == {"ptsip", "responsibility_map", "policies"}
        resolved = materialize_profile(projected)
        assert [item["id"] for item in resolved.effective_payload["components"]] == [
            "package",
            "package-tests",
            "tools",
        ]

    def test_existing_hybrid_decision_preserves_unrelated_overrides_and_removals(self) -> None:
        existing_override = component_payload(
            "package-tests",
            ["testing/**"],
            classification="PRODUCT",
            purpose="product_owned_package_verification",
            shipped=False,
            runtime_required=False,
            executable=True,
            roles=["VERIFICATION"],
        )
        source = hybrid_profile_payload(
            {
                "components": [existing_override],
                "remove_relationship_ids": ["package-tests-verify-package"],
            }
        )

        projected = project_payload(source, "tools", ["tools/**"], _g3_answer())
        overrides = projected["responsibility_map"]["overrides"]

        assert overrides["remove_relationship_ids"] == ["package-tests-verify-package"]
        assert overrides["components"][0] == existing_override
        assert overrides["components"][1]["id"] == "tools"
        assert source["responsibility_map"]["overrides"]["components"] == [existing_override]

    def test_existing_project_declaration_conflict_fails_without_partial_write(self) -> None:
        conflicting = component_payload(
            "tools",
            ["tools/**"],
            classification="PRODUCT",
            purpose="existing_product_component",
            shipped=True,
            runtime_required=True,
            executable=True,
        )
        source = hybrid_profile_payload({"components": [conflicting]})
        before = yaml.safe_dump(source, sort_keys=True)

        with pytest.raises(ValueError, match="conflicts with the resolved decision"):
            project_payload(source, "tools", ["tools/**"], _g3_answer())

        assert yaml.safe_dump(source, sort_keys=True) == before

    def test_invalid_decision_never_mutates_profile(self, tmp_path: Path) -> None:
        repo = _package_repo(tmp_path)
        profile = write_profile(repo / "ptsip.yaml", template_profile_payload())
        before = profile.read_text(encoding="utf-8")
        invalid = _g3_answer(classification="TOOLCHAIN")

        with pytest.raises(ValueError, match="Projected PTSIP profile is invalid"):
            prepare_local_profile(repo, "tools", ["tools/**"], invalid, profile)

        assert profile.read_text(encoding="utf-8") == before
        assert yaml.safe_load(before)["responsibility_map"]["mode"] == "template"


class TestG4ProfilePathControlPlane:
    """Reserved namespace; G4 has not been entered."""


class TestG5RecoveryAndIntegration:
    """Reserved namespace; G5 has not been entered."""
