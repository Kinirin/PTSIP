from __future__ import annotations

from pathlib import Path

from ptsip.app.service import DecisionService
from ptsip.app.store import DecisionStore
from ptsip.clarification.resolution import DecisionAnswer, parse_answer, validate_answer


class FakeGitHub:
    def __init__(self):
        self.created: list[tuple[str, str]] = []
        self.closed: list[int] = []
        self.comments: list[tuple[int, str]] = []

    def create_issue(self, repository: str, installation_id: int, title: str, body: str):
        from ptsip.app.github_client import GitHubIssue

        self.created.append((title, body))
        return GitHubIssue(11, f"https://github.com/{repository}/issues/11")

    def add_issue_comment(self, repository: str, installation_id: int, issue_number: int, body: str) -> None:
        self.comments.append((issue_number, body))

    def update_issue_state(self, repository: str, installation_id: int, issue_number: int, state: str) -> None:
        if state == "closed":
            self.closed.append(issue_number)

    def permission(self, repository: str, installation_id: int, username: str) -> str:
        return "write"

    def branch_head(self, repository: str, installation_id: int, branch: str) -> str:
        return "abc123"

    def file_text(self, repository: str, installation_id: int, path: str, ref: str) -> str | None:
        return None

    def commit_file_at_parent(self, *args, **kwargs) -> str:
        return "def456"


def _gate_payload() -> dict[str, object]:
    return {
        "id": "clr-test",
        "repository": "example/product",
        "branch": "main",
        "subject_revision": "abc123",
        "component_id": "tools",
        "request": {
            "id": "clr-test",
            "component_id": "tools",
            "include": ["tools/**"],
            "anchors": ["tools/generate.py"],
            "evidence_ids": [],
            "missing_fields": ["classification", "purpose", "shipped", "runtime_required", "lifecycle_owner", "executable"],
            "reason_codes": [],
            "status": "INCOMPLETE",
        },
        "issue": {"title": "decision", "body": "body"},
    }


def _answer(classification: str = "TOOLCHAIN") -> DecisionAnswer:
    return DecisionAnswer(
        classification=classification,
        purpose="Repository migration tooling",
        shipped=False,
        runtime_required=False,
        lifecycle_owner="DEVELOPMENT_TOOLING" if classification == "TOOLCHAIN" else "PRODUCT",
        executable=True,
    )


def test_structured_answer_parser_and_validation():
    answer = parse_answer(
        """```yaml
format: ptsip-clarification-answer/v1
decision:
  classification: TOOLCHAIN
  purpose: Repository migration tooling
  shipped: NO
  runtime_required: NO
  lifecycle_owner: DEVELOPMENT_TOOLING
  executable: YES
```"""
    )
    assert answer.classification == "TOOLCHAIN"
    assert validate_answer(answer).valid


def test_toolchain_runtime_answer_is_conflict():
    answer = DecisionAnswer(
        classification="TOOLCHAIN",
        purpose="Migration tooling",
        shipped=False,
        runtime_required=True,
        lifecycle_owner="DEVELOPMENT_TOOLING",
        executable=True,
    )
    result = validate_answer(answer)
    assert not result.valid
    assert result.status == "CONFLICT"


def test_store_first_valid_resolution_wins(tmp_path: Path):
    store = DecisionStore(tmp_path / "state.sqlite3")
    record, _ = store.gate(_gate_payload())
    assert record.status == "PENDING"
    first, accepted = store.resolve(record.id, _answer().as_dict(), "AGENT_CHAT", "agent")
    assert accepted
    assert first.answer is not None
    second_answer = _answer("PRODUCT").as_dict()
    second, accepted_second = store.resolve(record.id, second_answer, "GITHUB_ISSUE", "owner")
    assert not accepted_second
    assert second.answer == first.answer
    assert second.resolution_source == "AGENT_CHAT"


def test_gate_creates_one_issue_and_reuses_it(tmp_path: Path):
    store = DecisionStore(tmp_path / "state.sqlite3")
    store.set_installation("example/product", 99)
    github = FakeGitHub()
    service = DecisionService(store, github)  # type: ignore[arg-type]
    first = service.gate(_gate_payload())
    second = service.gate(_gate_payload())
    assert first["status"] == "DECISION_REQUIRED"
    assert second["status"] == "DECISION_REQUIRED"
    assert len(github.created) == 1


def test_late_issue_reply_is_ignored_after_agent_resolution(tmp_path: Path):
    store = DecisionStore(tmp_path / "state.sqlite3")
    store.set_installation("example/product", 99)
    github = FakeGitHub()
    service = DecisionService(store, github)  # type: ignore[arg-type]
    gated = service.gate(_gate_payload())
    decision = gated["decision"]
    assert isinstance(decision, dict)
    resolved = service.resolve_agent(
        {"decision_id": decision["id"], "answer": _answer().as_dict(), "actor": "agent"}
    )
    assert resolved["status"] == "RESOLVED"
    result = service.issue_comment(
        {
            "action": "created",
            "installation": {"id": 99},
            "repository": {"full_name": "example/product"},
            "issue": {"number": 11},
            "comment": {
                "body": """```yaml
format: ptsip-clarification-answer/v1
decision:
  classification: PRODUCT
  purpose: Product component
  shipped: YES
  runtime_required: YES
  lifecycle_owner: PRODUCT
  executable: YES
```"""
            },
            "sender": {"login": "owner"},
        }
    )
    assert result["status"] == "IGNORED_TERMINAL_DECISION"
    final = store.get(str(decision["id"]))
    assert final is not None
    assert final.answer is not None
    assert final.answer["classification"] == "TOOLCHAIN"
