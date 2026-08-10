from __future__ import annotations

import json
import subprocess
from pathlib import Path

from ptsip.clarification.generator import analyze_clarifications
from ptsip.clarification.transports import github_issue
from ptsip.cli import main
from ptsip.repository.discover import discover_repository
from ptsip.repository.remote import parse_remote
from ptsip.repository.snapshot import capture_snapshot, compare_snapshots


SPEC_REVISION = "a877b2f66a7f94c1b844c979e1b08fb08a9a8e45"


def _git(repo: Path, *args: str) -> None:
    subprocess.run(
        ["git", "-C", str(repo), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )


def _init_git(repo: Path) -> None:
    repo.mkdir(parents=True, exist_ok=True)
    _git(repo, "init")
    _git(repo, "config", "user.email", "ptsip-test@example.invalid")
    _git(repo, "config", "user.name", "PTSIP Test")


def _commit_all(repo: Path) -> None:
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")


def _tool_repo(tmp_path: Path) -> Path:
    repo = tmp_path / "repo"
    _init_git(repo)
    (repo / "tools").mkdir()
    (repo / "tools" / "generate.py").write_text("print('generate')\n", encoding="utf-8")
    _commit_all(repo)
    return repo


def test_github_remote_parser_supports_https_and_ssh():
    for url in (
        "https://github.com/example/product.git",
        "git@github.com:example/product.git",
        "ssh://git@github.com/example/product.git",
    ):
        remote = parse_remote("origin", url)
        assert remote.provider == "github"
        assert remote.repository == "example/product"


def test_repository_discovery_includes_origin(tmp_path: Path):
    repo = _tool_repo(tmp_path)
    _git(repo, "remote", "add", "origin", "git@github.com:example/product.git")
    info = discover_repository(repo)
    assert info.remote is not None
    assert info.remote.provider == "github"
    assert info.remote.repository == "example/product"


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
        "lifecycle_owner",
        "executable",
    )
    payload = analysis.as_dict("en")
    assert payload["inference"]["mode"] == "DETERMINISTIC_RULES_ONLY"
    assert payload["inference"]["llm_calls"] == 0
    assert payload["inference"]["speculative_classification"] is False
    assert compare_snapshots(before, after).stable


def test_declared_component_purpose_suppresses_question(tmp_path: Path):
    repo = _tool_repo(tmp_path)
    (repo / "ptsip.yaml").write_text(
        f"""ptsip:\n  version: \"0.2.0-draft\"\n  specification:\n    source: \"https://github.com/kwaksinwoo01/ptsip\"\n    revision: \"{SPEC_REVISION}\"\ncomponents:\n  - id: generator\n    classification: TOOLCHAIN\n    include: [\"tools/**\"]\n    purpose: build_generation\npolicies:\n  product_to_toolchain_runtime_dependency: deny\n  toolchain_in_product_package: deny\n  independent_build_resolution: required\n""",
        encoding="utf-8",
    )
    _commit_all(repo)
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
    assert questions["classification"] == "이 컴포넌트의 PTSIP 아키텍처 분류는 무엇입니까?"
    assert questions["purpose"] == "이 컴포넌트를 만든 주된 목적은 무엇입니까?"


def test_github_publish_uses_origin_and_deduplicates_outside_repo(tmp_path: Path, monkeypatch):
    repo = _tool_repo(tmp_path)
    state = tmp_path / "state"
    monkeypatch.setenv("PTSIP_HOME", str(state))
    _git(repo, "remote", "add", "origin", "https://github.com/example/product.git")
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
        assert "ptsip-clarification-answer/v1" in input_text
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
