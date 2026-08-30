from __future__ import annotations

import json
from pathlib import Path

import pytest
import yaml

from ptsip.app.github_client import GitHubAPIError
from ptsip.app.service import DecisionService
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
from ptsip.cli import main
from ptsip.inspection.components import ComponentCandidate
from ptsip.validation.components import AMBIGUOUS, resolve_candidate_coverage
from ptsip.validation.templates import materialize_profile
from _test_support import (
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


def _g4_repository(tmp_path: Path) -> Path:
    repo = init_git_repo(tmp_path / "repo")
    write_text(repo, "tools/generate.py", "print('generate')\n")
    (repo / "config").mkdir(parents=True, exist_ok=True)
    commit_all(repo)
    return repo


def _g4_resolve_args(repo: Path, decision_id: str, *extra: str) -> list[str]:
    return [
        "resolve",
        str(repo),
        "--decision",
        decision_id,
        "--classification",
        "DEVELOPMENT_TOOLING",
        "--purpose",
        "Repository-local generation tooling",
        "--shipped",
        "no",
        "--runtime-required",
        "no",
        "--executable",
        "yes",
        "--coordination",
        "local",
        "--json",
        *extra,
    ]


def _g4_gate_payload(profile_path: str, revision: str = "abc123") -> dict[str, object]:
    return {
        "id": "clr-g4",
        "repository": "example/product",
        "branch": "main",
        "subject_revision": revision,
        "profile_path": profile_path,
        "component_id": "tools",
        "request": {
            "component_id": "tools",
            "include": ["tools/**"],
            "anchors": ["top-level-tool-root"],
            "evidence_ids": ["root:tools"],
            "missing_fields": ["classification"],
            "reason_codes": ["MISSING_CLASSIFICATION"],
            "status": "INCOMPLETE",
        },
    }


class _G4GitHub:
    def __init__(
        self,
        *,
        branch_heads: list[str] | None = None,
        fail_commit: bool = False,
    ) -> None:
        self.branch_heads = list(branch_heads or ["abc123"])
        self.fail_commit = fail_commit
        self.read_paths: list[tuple[str, str]] = []
        self.write_paths: list[str] = []

    def file_text(self, repository: str, installation_id: int, path: str, ref: str) -> str | None:
        del repository, installation_id
        self.read_paths.append((path, ref))
        return None

    def branch_head(self, repository: str, installation_id: int, branch: str) -> str:
        del repository, installation_id, branch
        if len(self.branch_heads) > 1:
            return self.branch_heads.pop(0)
        return self.branch_heads[0]

    def commit_file_at_parent(
        self,
        repository: str,
        installation_id: int,
        branch: str,
        parent_sha: str,
        path: str,
        content: str,
        message: str,
    ) -> str:
        del repository, installation_id, branch, parent_sha, content, message
        self.write_paths.append(path)
        if self.fail_commit:
            raise GitHubAPIError("simulated branch-head race")
        return "def456"

    def add_issue_comment(self, *args: object, **kwargs: object) -> None:
        del args, kwargs

    def update_issue_state(self, *args: object, **kwargs: object) -> None:
        del args, kwargs


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
            include=("tools/**"],
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
    def test_non_root_profile_path_is_projection_target(self, tmp_path: Path) -> None:
        repo = _g4_repository(tmp_path)
        target = repo / "config" / "ptsip.yaml"

        prepared = prepare_local_profile(
            repo,
            "tools",
            ["tools/**"],
            _g3_answer(purpose="Repository-local generation tooling"),
            target,
        )

        assert prepared.path == target.resolve()
        assert prepared.expected_source is None
        assert not (repo / "ptsip.yaml").exists()

    def test_non_root_profile_path_survives_local_gate_and_reconciliation(
        self, tmp_path: Path, monkeypatch, capsys
    ) -> None:
        repo = _g4_repository(tmp_path)
        monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))

        assert main([
            "gate", str(repo), "--component", "tools", "--profile", "config/ptsip.yaml",
            "--coordination", "local", "--json",
        ]) == 7
        gated = json.loads(capsys.readouterr().out)
        decision = gated["decisions"][0]["decision"]
        assert decision["profile_path"] == "config/ptsip.yaml"

        assert main(_g4_resolve_args(repo, str(decision["id"]))) == 0
        resolved = json.loads(capsys.readouterr().out)
        assert resolved["selected_profile_path"] == "config/ptsip.yaml"
        assert (repo / "config" / "ptsip.yaml").is_file()
        assert not (repo / "ptsip.yaml").exists()

    def test_selected_profile_path_is_persisted_with_decision(self, tmp_path: Path) -> None:
        database = tmp_path / "decision.sqlite3"
        store = DecisionStore(database)

        record, _ = store.gate(_g4_gate_payload("./config\\ptsip.yaml"))
        reopened = DecisionStore(database).get(record.id)

        assert record.profile_path == "config/ptsip.yaml"
        assert reopened is not None
        assert reopened.profile_path == "config/ptsip.yaml"
        assert reopened.as_dict()["profile_path"] == "config/ptsip.yaml"

    def test_selected_profile_path_survives_retry_and_rebind(self, tmp_path: Path) -> None:
        store = DecisionStore(tmp_path / "decision.sqlite3")
        record, _ = store.gate(_g4_gate_payload("config/ptsip.yaml"))
        answer = canonical_v2_answer(purpose="Repository-local generation tooling")
        resolved, accepted = store.resolve(record.id, answer, "AGENT_CHAT", "tester")
        assert accepted is True
        store.mark_application(record.id, "STALE")

        rebound, _ = store.gate(_g4_gate_payload("config/ptsip.yaml", "new456"))

        assert rebound.id == resolved.id
        assert rebound.subject_revision == "new456"
        assert rebound.profile_path == "config/ptsip.yaml"
        assert rebound.answer == answer

    def test_remote_cas_reads_and_writes_exact_selected_profile_path(self, tmp_path: Path) -> None:
        store = DecisionStore(tmp_path / "decision.sqlite3")
        record, _ = store.gate(_g4_gate_payload("config/ptsip.yaml"))
        answer = _g3_answer(purpose="Repository-local generation tooling")
        record, accepted = store.resolve(record.id, answer.as_dict(), "AGENT_CHAT", "tester")
        assert accepted is True
        github = _G4GitHub(branch_heads=["abc123"])
        service = DecisionService(store, github)  # type: ignore[arg-type]

        projected = service._prepare_remote_projection(record, 99, answer)
        service._apply_remote(record, 99, projected)
        final = store.get(record.id)

        assert github.read_paths == [("config/ptsip.yaml", "abc123")]
        assert github.write_paths == ["config/ptsip.yaml"]
        assert final is not None
        assert final.application_status == "APPLIED"
        assert final.applied_revision == "def456"

    def test_changed_profile_path_does_not_apply_old_decision(
        self, tmp_path: Path, monkeypatch, capsys
    ) -> None:
        repo = _g4_repository(tmp_path)
        (repo / "other").mkdir()
        monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))

        assert main([
            "gate", str(repo), "--component", "tools", "--profile", "config/ptsip.yaml",
            "--coordination", "local", "--json",
        ]) == 7
        gated = json.loads(capsys.readouterr().out)
        decision_id = str(gated["decisions"][0]["decision"]["id"])

        assert main(_g4_resolve_args(repo, decision_id, "--profile", "other/ptsip.yaml")) == 8
        rejected = json.loads(capsys.readouterr().out)

        assert rejected["status"] == "PROFILE_PATH_MISMATCH"
        assert rejected["decision_profile_path"] == "config/ptsip.yaml"
        assert rejected["requested_profile_path"] == "other/ptsip.yaml"
        assert not (repo / "config" / "ptsip.yaml").exists()
        assert not (repo / "other" / "ptsip.yaml").exists()
        assert not (repo / "ptsip.yaml").exists()

    def test_stale_revision_does_not_apply_to_selected_profile(self, tmp_path: Path) -> None:
        store = DecisionStore(tmp_path / "decision.sqlite3")
        record, _ = store.gate(_g4_gate_payload("config/ptsip.yaml"))
        answer = _g3_answer(purpose="Repository-local generation tooling")
        record, accepted = store.resolve(record.id, answer.as_dict(), "AGENT_CHAT", "tester")
        assert accepted is True
        github = _G4GitHub(branch_heads=["new456"])
        service = DecisionService(store, github)  # type: ignore[arg-type]

        service._apply_remote(record, 99, "projected")
        final = store.get(record.id)

        assert github.write_paths == []
        assert final is not None
        assert final.application_status == "STALE"
        assert final.applied_revision is None
        assert final.profile_path == "config/ptsip.yaml"

    def test_branch_head_conflict_does_not_apply_selected_profile(self, tmp_path: Path) -> None:
        store = DecisionStore(tmp_path / "decision.sqlite3")
        record, _ = store.gate(_g4_gate_payload("config/ptsip.yaml"))
        answer = _g3_answer(purpose="Repository-local generation tooling")
        record, accepted = store.resolve(record.id, answer.as_dict(), "AGENT_CHAT", "tester")
        assert accepted is True
        github = _G4GitHub(branch_heads=["abc123", "new456"], fail_commit=True)
        service = DecisionService(store, github)  # type: ignore[arg-type]

        service._apply_remote(record, 99, "projected")
        final = store.get(record.id)

        assert github.write_paths == ["config/ptsip.yaml"]
        assert final is not None
        assert final.application_status == "STALE"
        assert final.applied_revision is None
        assert final.profile_path == "config/ptsip.yaml"


class TestG5RecoveryAndIntegration:
    def test_invalid_profile_blocks_clarification_without_raw_fallback(self, tmp_path: Path) -> None:
        repo = _g4_repository(tmp_path)
        profile = repo / "ptsip.yaml"
        profile.write_text("ptsip: [invalid\n", encoding="utf-8")

        analysis = analyze_clarifications(repo, ["tools"])
        payload = analysis.as_dict("en")
        recovery = payload["profile"]["recovery"]

        assert analysis.status == "PROFILE_INVALID"
        assert analysis.requests == ()
        assert analysis.profile_parse_error is not None
        assert recovery["raw_profile_fallback"] is False
        assert recovery["reuse_partial_effective_state"] is False

    def test_invalid_profile_exposes_selected_path_and_recovery_information(self, tmp_path: Path) -> None:
        repo = _g4_repository(tmp_path)
        profile = repo / "config" / "ptsip.yaml"
        profile.write_text("ptsip: [invalid\n", encoding="utf-8")

        analysis = analyze_clarifications(repo, ["tools"], profile)
        payload = analysis.as_dict("en")
        recovery = payload["profile"]["recovery"]

        assert payload["profile"]["path"] == str(profile.resolve())
        assert payload["profile"]["failure_stage"] == "PARSE"
        assert recovery["status"] == "PROJECT_CORRECTION_REQUIRED"
        assert recovery["authoritative"] is False
        assert recovery["selected_profile_path"] == str(profile.resolve())
        assert recovery["retry_requires_fresh_repository_snapshot"] is True
        assert [item["id"] for item in recovery["actions"]] == [
            "correct_project_profile",
            "retry_clarification",
        ]

    def test_corrected_profile_retry_uses_fresh_resolved_profile(self, tmp_path: Path) -> None:
        repo = _g4_repository(tmp_path)
        profile = repo / "ptsip.yaml"
        profile.write_text("ptsip: [invalid\n", encoding="utf-8")

        first = analyze_clarifications(repo, ["tools"])
        assert first.status == "PROFILE_INVALID"

        write_profile(
            profile,
            explicit_profile_payload(
                [
                    component_payload(
                        "tools",
                        ["tools/**"],
                        purpose="Repository-local generation tooling",
                    )
                ]
            ),
        )
        second = analyze_clarifications(repo, ["tools"])

        assert second.status == "NO_CLARIFICATION_REQUIRED"
        assert second.requests == ()
        assert second.profile_parse_error is None
        assert second.profile_failure_stage is None
        assert second.profile_recovery is None

    def test_repeated_gate_does_not_reopen_effectively_declared_architecture(
        self, tmp_path: Path, monkeypatch, capsys
    ) -> None:
        repo = _g4_repository(tmp_path)
        monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))

        assert main([
            "gate", str(repo), "--component", "tools", "--coordination", "local", "--json",
        ]) == 7
        first = json.loads(capsys.readouterr().out)
        decision_id = str(first["decisions"][0]["decision"]["id"])

        assert main(_g4_resolve_args(repo, decision_id)) == 0
        capsys.readouterr()

        assert main([
            "gate", str(repo), "--component", "tools", "--coordination", "local", "--json",
        ]) == 0
        repeated = json.loads(capsys.readouterr().out)
        assert repeated["status"] == "NO_DECISION_REQUIRED"
        assert repeated["decisions"] == []

    def test_read_only_clarification_does_not_mutate_source_profile(self, tmp_path: Path) -> None:
        repo = _package_repo(tmp_path)
        profile = write_profile(repo / "ptsip.yaml", template_profile_payload())
        commit_all(repo)
        before = profile.read_text(encoding="utf-8")

        analysis = analyze_clarifications(repo, ["tests"])

        assert analysis.status == "NO_CLARIFICATION_REQUIRED"
        assert profile.read_text(encoding="utf-8") == before
        assert yaml.safe_load(before)["responsibility_map"]["mode"] == "template"

    def test_cross_track_template_decision_becomes_effective_and_is_not_reasked(
        self, tmp_path: Path, monkeypatch, capsys
    ) -> None:
        repo = init_git_repo(tmp_path / "repo")
        write_text(repo, "src/package.py", "VALUE = 1\n")
        write_text(repo, "tests/test_package.py", "def test_value(): assert True\n")
        write_text(repo, "tools/generate.py", "print('generate')\n")
        profile = write_profile(repo / "ptsip.yaml", template_profile_payload())
        commit_all(repo)
        monkeypatch.setenv("PTSIP_HOME", str(tmp_path / "state"))

        assert main([
            "gate", str(repo), "--component", "tools", "--coordination", "local", "--json",
        ]) == 7
        gated = json.loads(capsys.readouterr().out)
        decision_id = str(gated["decisions"][0]["decision"]["id"])

        assert main(_g4_resolve_args(repo, decision_id)) == 0
        capsys.readouterr()

        source = yaml.safe_load(profile.read_text(encoding="utf-8"))
        assert source["responsibility_map"]["mode"] == "hybrid"
        effective = materialize_profile(source).effective_payload
        assert "tools" in [item["id"] for item in effective["components"]]

        repeated = analyze_clarifications(repo, ["tools"])
        assert repeated.status == "NO_CLARIFICATION_REQUIRED"
        assert repeated.requests == ()
