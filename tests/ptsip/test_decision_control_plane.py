from __future__ import annotations

from pathlib import Path

from ptsip.app.service import DecisionService
from ptsip.app.store import DecisionStore
from ptsip.clarification.resolution import DecisionAnswer, project_payload
from ptsip.constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION
from ptsip.profile_identity import CURRENT_PROJECT_PROFILE_VERSION
from _wu04g_support import canonical_v2_answer, clarification_answer_text


class FakeGitHub:
    def __init__(self):
        self.created: list[tuple[str, str]] = []
        self.closed: list[int] = []
        self.opened: list[int] = []
        self.comments: list[tuple[int, str]] = []
        self.file_content: str | None = None
        self.installation_lookups = 0

    def repository_installation(self, repository: str) -> int:
        self.installation_lookups += 1
        return 99

    def create_issue(self, repository: str, installation_id: int, title: str, body: str):
        from ptsip.app.github_client import GitHubIssue

        self.created.append((title, body))
        return GitHubIssue(11, f"https://github.com/{repository}/issues/11")

    def add_issue_comment(self, repository: str, installation_id: int, issue_number: int, body: str) -> None:
        self.comments.append((issue_number, body))

    def update_issue_state(self, repository: str, installation_id: int, issue_number: int, state: str) -> None:
        if state == "closed":
            self.closed.append(issue_number)
        elif state == "open":
            self.opened.append(issue_number)

    def permission(self, repository: str, installation_id: int, username: str) -> str:
        return "write"

    def branch_head(self, repository: str, installation_id: int, branch: str) -> str:
        return "abc123"

    def file_text(self, repository: str, installation_id: int, path: str, ref: str) -> str | None:
        return self.file_content

    def commit_file_at_parent(self, *args, **kwargs) -> str:
        return "def456"


def _gate_payload(revision: str = "abc123") -> dict[str, object]:
    return {
        "id": "clr-test",
        "repository": "example/product",
        "branch": "main",
        "subject_revision": revision,
        "component_id": "tools",
        "request": {
            "id": "clr-test",
            "component_id": "tools",
            "include": ["tools/**"],
            "anchors": ["tools/generate.py"],
            "evidence_ids": [],
            "missing_fields": ["classification", "purpose", "shipped", "runtime_required", "executable"],
            "reason_codes": [],
            "status": "INCOMPLETE",
        },
        "issue": {"title": "decision", "body": "body"},
    }


def _answer(classification: str = "DEVELOPMENT_TOOLING") -> DecisionAnswer:
    payload = canonical_v2_answer(
        classification=classification,
        purpose="Repository migration tooling" if classification != "PRODUCT" else "Product component",
        shipped=classification == "PRODUCT",
        runtime_required=classification == "PRODUCT",
        executable=True,
    )
    return DecisionAnswer(**payload)


def _issue_payload(classification: str = "PRODUCT") -> dict[str, object]:
    answer = canonical_v2_answer(
        classification=classification,
        purpose="Product component" if classification == "PRODUCT" else "Repository migration tooling",
        shipped=classification == "PRODUCT",
        runtime_required=classification == "PRODUCT",
        executable=True,
    )
    return {
        "action": "created",
        "installation": {"id": 99},
        "repository": {"full_name": "example/product"},
        "issue": {"number": 11},
        "comment": {"body": "```yaml\n" + clarification_answer_text(answer) + "```"},
        "sender": {"login": "owner"},
    }


def test_profile_projection_preserves_existing_boundary_and_structured_facts_and_rejects_conflict():
    existing = {
        "ptsip": {
            "version": CURRENT_PROJECT_PROFILE_VERSION,
            "specification": {
                "family": SPEC_VERSION,
                "source": SPEC_SOURCE,
                "revision": SPEC_REVISION,
            },
        },
        "responsibility_map": {"mode": "explicit"},
        "components": [
            {
                "id": "generator-sdk",
                "classification": "DEVELOPMENT_TOOLING",
                "include": ["tools/**", "scripts/generate.py"],
                "purpose": "Repository migration tooling",
                "shipped": False,
                "runtime_required": False,
                "executable": True,
            }
        ],
        "policies": {
            "product_to_nonproduct_runtime_dependency": "deny",
            "nonproduct_in_product_package": "deny",
            "independent_build_resolution": "required",
        },
    }
    answer = _answer()
    projected = project_payload(existing, "generator-sdk", ["tools/**"], answer)
    component = projected["components"][0]
    assert component["include"] == ["tools/**", "scripts/generate.py"]
    assert component["purpose"] == answer.purpose
    assert component["runtime_required"] is False
    assert component["executable"] is True
    assert "lifecycle_owner" not in component
    assert "release_owner" not in component
    assert "compatibility_owner" not in component

    conflicting = _answer("PRODUCT")
    try:
        project_payload(existing, "generator-sdk", ["tools/**"], conflicting)
    except ValueError as exc:
        assert "conflicts with the resolved decision" in str(exc)
    else:
        raise AssertionError("existing declaration conflict must be rejected")


def test_store_first_valid_resolution_wins(tmp_path: Path):
    store = DecisionStore(tmp_path / "state.sqlite3")
    record, _ = store.gate(_gate_payload())
    assert record.status == "PENDING"
    first, accepted = store.resolve(record.id, _answer().as_dict(), "AGENT_CHAT", "agent")
    assert accepted
    assert first.answer is not None
    assert "lifecycle_owner" not in first.answer
    second_answer = _answer("PRODUCT").as_dict()
    second, accepted_second = store.resolve(record.id, second_answer, "GITHUB_ISSUE", "owner")
    assert not accepted_second
    assert second.answer == first.answer
    assert second.resolution_source == "AGENT_CHAT"


def test_active_gate_rebinds_retryable_resolution_without_changing_winner(tmp_path: Path):
    store = DecisionStore(tmp_path / "state.sqlite3")
    record, _ = store.gate(_gate_payload())
    answer = _answer().as_dict()
    resolved, accepted = store.resolve(record.id, answer, "GITHUB_ISSUE", "owner")
    assert accepted
    assert resolved.resolution_source == "GITHUB_ISSUE"
    store.mark_application(record.id, "STALE")

    rebound, _ = store.gate(_gate_payload("new456"))
    assert rebound.status == "RESOLVED"
    assert rebound.subject_revision == "new456"
    assert rebound.answer == answer
    assert rebound.resolution_source == "GITHUB_ISSUE"

    retried, retry_accepted = store.resolve(record.id, answer, "AGENT_CHAT", "agent")
    assert retry_accepted
    assert retried.answer == answer
    assert retried.resolution_source == "GITHUB_ISSUE"

    contradictory, contradictory_accepted = store.resolve(
        record.id, _answer("PRODUCT").as_dict(), "AGENT_CHAT", "agent"
    )
    assert not contradictory_accepted
    assert contradictory.answer == answer


def test_gate_recovers_installation_and_reopens_existing_pending_issue(tmp_path: Path):
    store = DecisionStore(tmp_path / "state.sqlite3")
    github = FakeGitHub()
    service = DecisionService(store, github)  # type: ignore[arg-type]
    first = service.gate(_gate_payload())
    second = service.gate(_gate_payload())
    assert first["status"] == "DECISION_REQUIRED"
    assert second["status"] == "DECISION_REQUIRED"
    assert github.installation_lookups == 1
    assert store.installation_for("example/product") == 99
    assert len(github.created) == 1
    assert github.opened == [11]


def test_issue_profile_conflict_does_not_win_authoritative_cas(tmp_path: Path):
    store = DecisionStore(tmp_path / "state.sqlite3")
    store.set_installation("example/product", 99)
    github = FakeGitHub()
    github.file_content = f"""ptsip:
  version: {CURRENT_PROJECT_PROFILE_VERSION}
  specification:
    family: {SPEC_VERSION}
    source: {SPEC_SOURCE}
    revision: {SPEC_REVISION}
responsibility_map:
  mode: explicit
components:
  - id: tools
    classification: PRODUCT
    include: [\"tools/**\"]
    purpose: Existing product component
    shipped: true
    runtime_required: true
    executable: true
policies:
  product_to_nonproduct_runtime_dependency: deny
  nonproduct_in_product_package: deny
  independent_build_resolution: required
"""
    service = DecisionService(store, github)  # type: ignore[arg-type]
    service.gate(_gate_payload())
    result = service.issue_comment(_issue_payload("DEVELOPMENT_TOOLING"))
    assert result["status"] == "CONFLICT"
    record = store.get("clr-test")
    assert record is not None
    assert record.status == "PENDING"
    assert record.answer is None


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
    result = service.issue_comment(_issue_payload("PRODUCT"))
    assert result["status"] == "IGNORED_TERMINAL_DECISION"
    final = store.get(str(decision["id"]))
    assert final is not None
    assert final.answer is not None
    assert final.answer["classification"] == "DEVELOPMENT_TOOLING"
    assert "lifecycle_owner" not in final.answer
