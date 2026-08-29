from __future__ import annotations

import json
import subprocess
from pathlib import Path

from ptsip.clarification.generator import analyze_clarifications
from ptsip.clarification.transports import github_issue
from ptsip.cli import main
from ptsip.repository.snapshot import capture_snapshot, compare_snapshots
from _test_support import (
    commit_all,
    component_payload,
    explicit_profile_payload,
    git,
    init_git_repo,
    write_profile,
    write_text,
)


def _tool_repo(tmp_path: Path) -> Path:
    repo = init_git_repo(tmp_path / "repo")
    write_text(repo, "tools/generate.py", "print('generate')\n")
    commit_all(repo)
    return repo


def test_no_profile_requests_fixed_facts_without_llm(tmp_path: Path):
    repo = _tool_repo(tmp_path)
    before = capture_snapshot(repo)
    analysis = analyze_clarifications(repo)
    after = capture_snapshot(repo)
    assert analysis.status == "CLARIFICATION_REQUIRED"
    request = next(item for item in analysis.requests if item.component_id == "tools")
    assert request.status.value == "INCOMPLETE"
    assert request.missing_fields == (
        "classification",
        "purpose",
        "shipped",
        "runtime_required",
        "executable",
    )
    payload = analysis.as_dict("en")
    assert payload["inference"]["mode"] == "DETERMINISTIC_RULES_ONLY"
    assert payload["inference"]["llm_calls"] == 0
    assert payload["inference"]["speculative_classification"] is False
    assert compare_snapshots(before, after).stable


def test_declared_component_purpose_suppresses_question(tmp_path: Path):
    repo = _tool_repo(tmp_path)
    write_profile(
        repo / "ptsip.yaml",
        explicit_profile_payload(
            [
                component_payload(
                    "generator",
                    ["tools/**"],
                    purpose="build_generation",
                )
            ]
        ),
    )
    commit_all(repo)
    analysis = analyze_clarifications(repo, ["tools"])
    assert analysis.requests == ()
    assert analysis.status == "NO_CLARIFICATION_REQUIRED"


def test_partial_declared_scope_does_not_claim_broader_candidate(tmp_path: Path):
    repo = _tool_repo(tmp_path)
    write_text(repo, "tools/generated/config.py", "VALUE = 1\n")
    write_profile(
        repo / "ptsip.yaml",
        explicit_profile_payload(
            [
                component_payload(
                    "generator-sdk",
                    ["tools/generated/**"],
                    purpose="generated_tool_support",
                )
            ]
        ),
    )
    commit_all(repo)

    analysis = analyze_clarifications(repo, ["tools"])
    assert analysis.status == "CLARIFICATION_REQUIRED"
    assert len(analysis.requests) == 1
    request = analysis.requests[0]
    assert request.component_id == "tools"
    assert request.include == ("tools/**",)


def test_associated_artifact_scope_does_not_reopen_component_clarification(tmp_path: Path):
    repo = _tool_repo(tmp_path)
    write_text(repo, "sdk/core.py", "VALUE = 1\n")
    write_profile(
        repo / "ptsip.yaml",
        explicit_profile_payload(
            [component_payload("sdk", ["sdk/**"], purpose="reusable_sdk")],
            associated_artifacts=[
                {
                    "id": "sdk-support",
                    "anchor": "sdk",
                    "include": ["tools/**"],
                    "purpose": "sdk_owned_support_surface",
                }
            ],
            relationships=[
                {
                    "id": "support-documents-sdk",
                    "from": "sdk-support",
                    "to": "sdk",
                    "type": "DOCUMENTS",
                }
            ],
        ),
    )
    commit_all(repo)

    analysis = analyze_clarifications(repo, ["tools"])
    assert analysis.requests == ()
    assert analysis.status == "NO_CLARIFICATION_REQUIRED"


def test_cli_uses_korean_fixed_template(tmp_path: Path, monkeypatch, capsys):
    repo = _tool_repo(tmp_path)
    monkeypatch.setenv("PTSIP_LANG", "ko")
    assert main(["clarify", str(repo), "--component", "tools", "--json"]) == 0
    payload = json.loads(capsys.readouterr().out)
    assert payload["language"] == "ko"
    assert payload["inference"]["llm_calls"] == 0
    questions = {item["field"]: item["prompt"] for item in payload["requests"][0]["questions"]}
    assert questions["classification"] == "이 PTSIP 컴포넌트의 primary lifecycle ownership은 무엇입니까?"
    assert questions["purpose"] == "이 컴포넌트를 만든 주된 목적은 무엇입니까?"


def test_github_publish_uses_origin_and_deduplicates_outside_repo(tmp_path: Path, monkeypatch):
    repo = _tool_repo(tmp_path)
    state = tmp_path / "state"
    monkeypatch.setenv("PTSIP_HOME", str(state))
    git(repo, "remote", "add", "origin", "https://github.com/example/product.git")
    analysis = analyze_clarifications(repo, ["tools"])
    before = capture_snapshot(repo)
    calls: list[tuple[list[str], str | None]] = []

    monkeypatch.setattr(github_issue.shutil, "which", lambda name: "/usr/bin/gh" if name == "gh" else None)

    def fake_run(args: list[str], input_text: str | None = None) -> subprocess.CompletedProcess[str]:
        calls.append((args, input_text))
        if args[:2] == ["auth", "status"]:
            return subprocess.CompletedProcess(args, 0, stdout="", stderr="")
        assert args[:2] == ["issue", "create"]
        assert "--body-file" in args
        assert args[args.index("--body-file") + 1] == "-"
        assert input_text is not None
        assert "PTSIP does not call an LLM" in input_text
        assert "ptsip-clarification-answer/v2" in input_text
        structured = input_text.split("```yaml", 1)[1].split("```", 1)[0]
        assert "lifecycle_owner" not in structured
        assert "DEVELOPMENT_TOOLING" in input_text
        assert "DELIVERY" in input_text
        assert "OPERATIONS" in input_text
        return subprocess.CompletedProcess(args, 0, stdout="https://github.com/example/product/issues/123\n", stderr="")

    monkeypatch.setattr(github_issue, "_run_gh", fake_run)
    first = github_issue.publish(
        repository_root=repo,
        remote=analysis.repository.remote,
        repository_revision=analysis.repository.commit,
        requests=analysis.requests,
        language="en",
    )
    assert first[0].status == "CREATED"
    assert first[0].repository == "example/product"
    assert first[0].issue_number == 123
    assert len(calls) == 2
    assert list((state / "clarifications").rglob("state.json"))
    assert compare_snapshots(before, capture_snapshot(repo)).stable

    second = github_issue.publish(
        repository_root=repo,
        remote=analysis.repository.remote,
        repository_revision=analysis.repository.commit,
        requests=analysis.requests,
        language="en",
    )
    assert second[0].status == "EXISTING"
    assert second[0].issue_url == first[0].issue_url
    assert len(calls) == 2
