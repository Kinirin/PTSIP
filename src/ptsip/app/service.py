from __future__ import annotations

from typing import Any

from ..clarification.resolution import dump_payload, load_profile_text, parse_answer, project_payload, validate_answer, validate_projected_payload
from ..clarification.resolution.model import DecisionAnswer
from .github_client import GitHubAppClient, GitHubAPIError
from .store import DecisionRecord, DecisionStore

WRITE_PERMISSIONS = {"admin", "maintain", "write"}


def _answer_from_mapping(payload: dict[str, object]) -> DecisionAnswer:
    return DecisionAnswer(
        classification=str(payload["classification"]),
        purpose=str(payload["purpose"]),
        shipped=bool(payload["shipped"]),
        runtime_required=bool(payload["runtime_required"]),
        lifecycle_owner=str(payload["lifecycle_owner"]),
        executable=bool(payload["executable"]),
    )


def _workflow_status(record: DecisionRecord) -> str:
    if record.status == "PENDING":
        return "DECISION_REQUIRED"
    if record.status == "RESOLVED" and record.application_status not in {"APPLIED", "LOCAL_APPLIED"}:
        return "RESOLVED_APPLICATION_REQUIRED"
    return record.status


class DecisionService:
    def __init__(self, store: DecisionStore, github: GitHubAppClient):
        self.store = store
        self.github = github

    def gate(self, payload: dict[str, Any]) -> dict[str, object]:
        for key in ("id", "repository", "branch", "subject_revision", "component_id", "request", "issue"):
            if key not in payload:
                raise ValueError(f"gate payload missing {key}")
        record, stale = self.store.gate(payload)
        installation = self.store.installation_for(record.repository)
        if installation is None:
            raise RuntimeError(f"PTSIP GitHub App installation is not registered for {record.repository}")
        for old in stale:
            if old.issue_number:
                try:
                    self.github.add_issue_comment(
                        old.repository,
                        installation,
                        old.issue_number,
                        "PTSIP marked this clarification as stale because a newer component decision request replaced it.",
                    )
                    self.github.update_issue_state(old.repository, installation, old.issue_number, "closed")
                except GitHubAPIError:
                    pass
        if record.status == "PENDING" and record.issue_number is None:
            issue = payload["issue"]
            if not isinstance(issue, dict):
                raise ValueError("issue must be an object")
            created = self.github.create_issue(
                record.repository,
                installation,
                str(issue.get("title", f"[PTSIP clarification] {record.component_id}")),
                str(issue.get("body", "")),
            )
            record = self.store.attach_issue(record.id, created.number, created.url)
        return {"status": _workflow_status(record), "decision": record.as_dict()}

    def decision(self, payload: dict[str, Any]) -> dict[str, object]:
        decision_id = str(payload.get("decision_id", ""))
        if not decision_id:
            raise ValueError("decision_id is required")
        record = self.store.get(decision_id)
        if record is None:
            raise KeyError(decision_id)
        return {"status": _workflow_status(record), "decision": record.as_dict()}

    def resolve_agent(self, payload: dict[str, Any]) -> dict[str, object]:
        decision_id = str(payload.get("decision_id", ""))
        raw_answer = payload.get("answer")
        if not decision_id or not isinstance(raw_answer, dict):
            raise ValueError("decision_id and answer are required")
        answer = _answer_from_mapping(raw_answer)
        validation = validate_answer(answer)
        if not validation.valid:
            return {"status": "CONFLICT", "validation": validation.as_dict()}
        actor = str(payload.get("actor") or "coding-agent-session")
        record, accepted = self.store.resolve(decision_id, answer.as_dict(), "AGENT_CHAT", actor)
        return {
            "status": "RESOLVED" if accepted else "ALREADY_RESOLVED",
            "accepted": accepted,
            "decision": record.as_dict(),
            "validation": validation.as_dict(),
        }

    def application(self, payload: dict[str, Any]) -> dict[str, object]:
        decision_id = str(payload.get("decision_id", ""))
        status = str(payload.get("status", ""))
        if status not in {"LOCAL_APPLIED", "FAILED", "STALE"}:
            raise ValueError("unsupported agent application status")
        existing = self.store.get(decision_id)
        if existing is None:
            raise KeyError(decision_id)
        if existing.status != "RESOLVED":
            raise ValueError("application state can be changed only for a resolved decision")
        record = self.store.mark_application(
            decision_id,
            status,
            str(payload.get("applied_revision") or "") or None,
        )
        if status == "LOCAL_APPLIED" and record.issue_number:
            installation = self.store.installation_for(record.repository)
            if installation is not None:
                try:
                    self.github.add_issue_comment(
                        record.repository,
                        installation,
                        record.issue_number,
                        f"PTSIP decision `{record.id}` was resolved via `{record.resolution_source}` and applied by the active coding-agent workflow. Late replies are ignored.",
                    )
                    self.github.update_issue_state(record.repository, installation, record.issue_number, "closed")
                except GitHubAPIError:
                    pass
        return {"status": status, "decision": record.as_dict()}

    def register_installation_event(self, payload: dict[str, Any]) -> None:
        installation = payload.get("installation")
        if not isinstance(installation, dict) or not isinstance(installation.get("id"), int):
            return
        installation_id = int(installation["id"])
        repository = payload.get("repository")
        if isinstance(repository, dict) and repository.get("full_name"):
            self.store.set_installation(str(repository["full_name"]), installation_id)
        for key in ("repositories", "repositories_added"):
            repos = payload.get(key)
            if isinstance(repos, list):
                for item in repos:
                    if isinstance(item, dict) and item.get("full_name"):
                        self.store.set_installation(str(item["full_name"]), installation_id)

    def issue_comment(self, payload: dict[str, Any]) -> dict[str, object]:
        if payload.get("action") != "created":
            return {"status": "IGNORED"}
        self.register_installation_event(payload)
        repository = payload.get("repository")
        issue = payload.get("issue")
        comment = payload.get("comment")
        sender = payload.get("sender")
        if not all(isinstance(item, dict) for item in (repository, issue, comment, sender)):
            return {"status": "IGNORED"}
        full_name = str(repository.get("full_name", ""))
        issue_number = issue.get("number")
        username = str(sender.get("login", ""))
        body = str(comment.get("body", ""))
        if not full_name or not isinstance(issue_number, int) or not username:
            return {"status": "IGNORED"}
        record = self.store.by_issue(full_name, issue_number)
        if record is None:
            return {"status": "IGNORED_NOT_PTSIP"}
        if record.status != "PENDING":
            return {"status": "IGNORED_TERMINAL_DECISION", "decision": record.as_dict()}
        installation = self.store.installation_for(full_name)
        if installation is None:
            return {"status": "IGNORED_NO_INSTALLATION"}
        try:
            permission = self.github.permission(full_name, installation, username)
        except GitHubAPIError:
            permission = "none"
        if permission not in WRITE_PERMISSIONS:
            return {"status": "IGNORED_UNAUTHORIZED"}
        try:
            answer = parse_answer(body)
        except ValueError as exc:
            return {"status": "IGNORED_UNSTRUCTURED", "error": str(exc)}
        validation = validate_answer(answer)
        if not validation.valid:
            self.github.add_issue_comment(
                full_name,
                installation,
                issue_number,
                "PTSIP could not accept this decision because it conflicts with the fixed resolution rules:\n\n- "
                + "\n- ".join(validation.errors),
            )
            return {"status": "CONFLICT", "validation": validation.as_dict()}

        # Validate the answer against the profile at the exact gate snapshot
        # before it is allowed to win the authoritative compare-and-set.
        try:
            projected_content = self._prepare_remote_projection(record, installation, answer)
        except ValueError as exc:
            self.github.add_issue_comment(
                full_name,
                installation,
                issue_number,
                "PTSIP could not accept this decision because it conflicts with the declared/projected profile:\n\n- "
                + str(exc),
            )
            return {"status": "CONFLICT", "error": str(exc)}

        resolved, accepted = self.store.resolve(record.id, answer.as_dict(), "GITHUB_ISSUE", username)
        if not accepted:
            return {"status": "IGNORED_TERMINAL_DECISION", "decision": resolved.as_dict()}
        try:
            self._apply_remote(resolved, installation, projected_content)
        except Exception as exc:
            self.store.mark_application(record.id, "FAILED")
            try:
                self.github.add_issue_comment(
                    record.repository,
                    installation,
                    issue_number,
                    "PTSIP accepted this decision but could not apply the profile automatically. "
                    f"A coding agent must reconcile the already resolved decision: `{exc}`",
                )
                self.github.update_issue_state(record.repository, installation, issue_number, "closed")
            except GitHubAPIError:
                pass
        final = self.store.get(record.id)
        assert final is not None
        return {"status": "RESOLVED", "decision": final.as_dict()}

    def _prepare_remote_projection(
        self,
        record: DecisionRecord,
        installation: int,
        answer: DecisionAnswer,
    ) -> str:
        existing_text = self.github.file_text(record.repository, installation, "ptsip.yaml", record.subject_revision)
        existing = load_profile_text(existing_text)
        include = record.request.get("include", [])
        if not isinstance(include, list):
            raise ValueError("decision request include must be a list")
        projected = project_payload(existing, record.component_id, [str(item) for item in include], answer)
        projected_errors = validate_projected_payload(projected)
        if projected_errors:
            raise ValueError("Projected PTSIP profile is invalid: " + "; ".join(projected_errors))
        return dump_payload(projected)

    def _apply_remote(self, record: DecisionRecord, installation: int, projected_content: str) -> None:
        current = self.github.branch_head(record.repository, installation, record.branch)
        if current != record.subject_revision:
            self.store.mark_application(record.id, "STALE")
            if record.issue_number:
                self.github.add_issue_comment(
                    record.repository,
                    installation,
                    record.issue_number,
                    "The decision was accepted, but the target branch changed after the last active decision gate. "
                    "PTSIP did not modify the profile. A coding agent must re-run the gate against the current revision and can reconcile the already authoritative answer without asking the user to decide again.",
                )
                self.github.update_issue_state(record.repository, installation, record.issue_number, "closed")
            return
        commit_sha = self.github.commit_file_at_parent(
            record.repository,
            installation,
            record.branch,
            record.subject_revision,
            "ptsip.yaml",
            projected_content,
            f"ptsip: apply decision {record.id}",
        )
        self.store.mark_application(record.id, "APPLIED", commit_sha)
        if record.issue_number:
            self.github.add_issue_comment(
                record.repository,
                installation,
                record.issue_number,
                f"PTSIP applied decision `{record.id}` in commit `{commit_sha}`. Late replies are ignored.",
            )
            self.github.update_issue_state(record.repository, installation, record.issue_number, "closed")
