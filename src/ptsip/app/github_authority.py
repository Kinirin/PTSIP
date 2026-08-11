from __future__ import annotations

import base64
import copy
import hashlib
import json
import re
import shutil
import subprocess
from datetime import datetime, timezone
from typing import Any, Protocol

from ..clarification.resolution import DecisionAnswer, validate_answer

AUTHORITY_BRANCH = "ptsip-policy"
AUTHORITY_FORMAT = "ptsip-github-authority/v1"


class AuthorityConflict(RuntimeError):
    """The GitHub authority ref changed after the caller read it."""


class CoordinationUnavailable(RuntimeError):
    """GitHub coordination cannot be reached or authenticated safely."""


class GitHubApiError(RuntimeError):
    def __init__(self, message: str, status: int | None = None):
        super().__init__(message)
        self.status = status


class AuthorityStore(Protocol):
    def ensure_head(self) -> str: ...

    def read_json(self, path: str) -> tuple[str, dict[str, object] | None]: ...

    def write_json(
        self,
        path: str,
        payload: dict[str, object],
        expected_head: str,
        message: str,
    ) -> str: ...


def _utc_now() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


def _answer_from_mapping(payload: dict[str, object]) -> DecisionAnswer:
    return DecisionAnswer(
        classification=str(payload["classification"]),
        purpose=str(payload["purpose"]),
        shipped=bool(payload["shipped"]),
        runtime_required=bool(payload["runtime_required"]),
        lifecycle_owner=str(payload["lifecycle_owner"]),
        executable=bool(payload["executable"]),
    )


def _normalize_selector(value: object) -> str:
    text = str(value).replace("\\", "/").strip()
    while text.startswith("./"):
        text = text[2:]
    return text.strip("/")


def _global_decision_id(repository: str, request: dict[str, object]) -> str:
    include = request.get("include")
    if not isinstance(include, list):
        include = list(include) if isinstance(include, tuple) else []
    selectors = sorted({normalized for item in include if (normalized := _normalize_selector(item))})
    if not selectors:
        raise ValueError("GitHub-coordinated decision request requires include selectors")
    digest = hashlib.sha256(
        (repository + "\0" + "\0".join(selectors)).encode("utf-8")
    ).hexdigest()[:20]
    return f"gdec-{digest}"


class GhApi:
    def __init__(self) -> None:
        self._ready = False

    @staticmethod
    def _run(args: list[str], input_text: str | None = None) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            ["gh", *args],
            input=input_text,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            check=False,
        )

    def ensure_ready(self) -> None:
        if self._ready:
            return
        if shutil.which("gh") is None:
            raise CoordinationUnavailable(
                "GitHub-coordinated PTSIP authority requires the GitHub CLI 'gh'. "
                "Install gh and authenticate it, or explicitly use --coordination local."
            )
        auth = self._run(["auth", "status", "--hostname", "github.com"])
        if auth.returncode != 0:
            detail = auth.stderr.strip() or auth.stdout.strip() or "authentication failed"
            raise CoordinationUnavailable(f"GitHub CLI authentication is not ready: {detail}")
        self._ready = True

    def request(
        self,
        method: str,
        path: str,
        payload: dict[str, object] | None = None,
    ) -> dict[str, object]:
        self.ensure_ready()
        args = ["api", path, "--method", method.upper()]
        input_text = None
        if payload is not None:
            args.extend(["--input", "-"])
            input_text = json.dumps(payload, ensure_ascii=False)
        completed = self._run(args, input_text)
        if completed.returncode != 0:
            detail = completed.stderr.strip() or completed.stdout.strip() or "GitHub API request failed"
            match = re.search(r"HTTP\s+(\d{3})", detail, flags=re.IGNORECASE)
            status = int(match.group(1)) if match else None
            raise GitHubApiError(detail, status)
        text = completed.stdout.strip()
        if not text:
            return {}
        try:
            parsed = json.loads(text)
        except json.JSONDecodeError as exc:
            raise CoordinationUnavailable(f"GitHub API returned invalid JSON for {path}: {exc}") from exc
        if not isinstance(parsed, dict):
            raise CoordinationUnavailable(f"GitHub API returned a non-object response for {path}")
        return parsed


class GitHubAuthorityStore:
    """Git-ref-backed global decision authority with non-force CAS updates."""

    def __init__(
        self,
        repository: str,
        branch: str = AUTHORITY_BRANCH,
        api: GhApi | None = None,
    ) -> None:
        if not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", repository):
            raise ValueError("GitHub repository must use owner/repository form")
        if not re.fullmatch(r"[A-Za-z0-9._/-]+", branch):
            raise ValueError("Invalid GitHub authority branch")
        self.repository = repository
        self.branch = branch
        self.api = api or GhApi()

    @property
    def ref(self) -> str:
        return f"refs/heads/{self.branch}"

    def _ref_path(self) -> str:
        return f"repos/{self.repository}/git/ref/heads/{self.branch}"

    def _head_or_none(self) -> str | None:
        try:
            payload = self.api.request("GET", self._ref_path())
        except GitHubApiError as exc:
            if exc.status == 404:
                return None
            raise CoordinationUnavailable(f"Unable to read PTSIP GitHub authority ref: {exc}") from exc
        obj = payload.get("object")
        if not isinstance(obj, dict) or not obj.get("sha"):
            raise CoordinationUnavailable("GitHub authority ref returned no commit SHA")
        return str(obj["sha"])

    def _create_blob(self, content: str) -> str:
        payload = self.api.request(
            "POST",
            f"repos/{self.repository}/git/blobs",
            {"content": content, "encoding": "utf-8"},
        )
        sha = str(payload.get("sha", ""))
        if not sha:
            raise CoordinationUnavailable("GitHub blob creation returned no SHA")
        return sha

    def _create_tree(self, entries: list[dict[str, object]], base_tree: str | None = None) -> str:
        payload: dict[str, object] = {"tree": entries}
        if base_tree:
            payload["base_tree"] = base_tree
        response = self.api.request("POST", f"repos/{self.repository}/git/trees", payload)
        sha = str(response.get("sha", ""))
        if not sha:
            raise CoordinationUnavailable("GitHub tree creation returned no SHA")
        return sha

    def _create_commit(self, tree: str, message: str, parents: list[str]) -> str:
        response = self.api.request(
            "POST",
            f"repos/{self.repository}/git/commits",
            {"message": message, "tree": tree, "parents": parents},
        )
        sha = str(response.get("sha", ""))
        if not sha:
            raise CoordinationUnavailable("GitHub commit creation returned no SHA")
        return sha

    def _tree_for_commit(self, commit_sha: str) -> str:
        response = self.api.request("GET", f"repos/{self.repository}/git/commits/{commit_sha}")
        tree = response.get("tree")
        if not isinstance(tree, dict) or not tree.get("sha"):
            raise CoordinationUnavailable("GitHub authority commit returned no tree SHA")
        return str(tree["sha"])

    def _bootstrap(self) -> str:
        manifest = {
            "format": AUTHORITY_FORMAT,
            "repository": self.repository,
            "ref": self.ref,
        }
        blob = self._create_blob(json.dumps(manifest, indent=2, sort_keys=True) + "\n")
        tree = self._create_tree(
            [{"path": "authority.json", "mode": "100644", "type": "blob", "sha": blob}]
        )
        commit = self._create_commit(tree, "ptsip: initialize coordinated authority", [])
        try:
            self.api.request(
                "POST",
                f"repos/{self.repository}/git/refs",
                {"ref": self.ref, "sha": commit},
            )
            return commit
        except GitHubApiError as exc:
            if exc.status not in {409, 422}:
                raise CoordinationUnavailable(f"Unable to initialize PTSIP authority ref: {exc}") from exc
            head = self._head_or_none()
            if head:
                return head
            raise AuthorityConflict("PTSIP authority bootstrap raced but no winning ref is visible") from exc

    def ensure_head(self) -> str:
        head = self._head_or_none()
        return head if head is not None else self._bootstrap()

    def read_json(self, path: str) -> tuple[str, dict[str, object] | None]:
        head = self.ensure_head()
        tree_sha = self._tree_for_commit(head)
        tree = self.api.request(
            "GET",
            f"repos/{self.repository}/git/trees/{tree_sha}?recursive=1",
        )
        items = tree.get("tree")
        if not isinstance(items, list):
            raise CoordinationUnavailable("GitHub authority tree returned no entries")
        blob_sha = ""
        for item in items:
            if isinstance(item, dict) and item.get("path") == path and item.get("type") == "blob":
                blob_sha = str(item.get("sha", ""))
                break
        if not blob_sha:
            return head, None
        blob = self.api.request("GET", f"repos/{self.repository}/git/blobs/{blob_sha}")
        encoded = str(blob.get("content", "")).replace("\n", "")
        if not encoded:
            raise CoordinationUnavailable(f"GitHub authority blob {path} has no content")
        try:
            text = base64.b64decode(encoded).decode("utf-8")
            document = json.loads(text)
        except (ValueError, UnicodeError, json.JSONDecodeError) as exc:
            raise CoordinationUnavailable(f"GitHub authority document {path} is invalid: {exc}") from exc
        if not isinstance(document, dict):
            raise CoordinationUnavailable(f"GitHub authority document {path} is not an object")
        return head, document

    def write_json(
        self,
        path: str,
        payload: dict[str, object],
        expected_head: str,
        message: str,
    ) -> str:
        if not path or path.startswith("/") or ".." in path.split("/"):
            raise ValueError("Invalid authority document path")
        base_tree = self._tree_for_commit(expected_head)
        blob = self._create_blob(json.dumps(payload, indent=2, sort_keys=True, ensure_ascii=False) + "\n")
        tree = self._create_tree(
            [{"path": path, "mode": "100644", "type": "blob", "sha": blob}],
            base_tree=base_tree,
        )
        commit = self._create_commit(tree, message, [expected_head])
        try:
            self.api.request(
                "PATCH",
                f"repos/{self.repository}/git/refs/heads/{self.branch}",
                {"sha": commit, "force": False},
            )
        except GitHubApiError as exc:
            if exc.status in {409, 422}:
                raise AuthorityConflict(
                    f"PTSIP authority HEAD changed after {expected_head}; stale mutation rejected"
                ) from exc
            raise CoordinationUnavailable(f"Unable to update PTSIP authority ref: {exc}") from exc
        return commit


def _workflow_status(record: dict[str, object]) -> str:
    status = str(record.get("status", ""))
    if status == "PENDING":
        return "DECISION_REQUIRED"
    if status == "RESOLVED":
        return "RESOLVED_APPLICATION_REQUIRED"
    return status


class GithubControlPlaneClient:
    """PTSIP decision client coordinated by a GitHub ref instead of shared SQLite."""

    def __init__(
        self,
        repository: str,
        *,
        store: AuthorityStore | None = None,
        branch: str = AUTHORITY_BRANCH,
    ) -> None:
        self.repository = repository
        self.store: AuthorityStore = store or GitHubAuthorityStore(repository, branch)

    @staticmethod
    def _decision_path(decision_id: str) -> str:
        if not re.fullmatch(r"[A-Za-z0-9._-]+", decision_id):
            raise ValueError("Invalid decision ID for GitHub authority")
        return f"decisions/{decision_id}.json"

    def gate(self, payload: dict[str, Any]) -> dict[str, object]:
        for key in ("id", "repository", "branch", "subject_revision", "component_id", "request"):
            if key not in payload:
                raise ValueError(f"gate payload missing {key}")
        if str(payload["repository"]) != self.repository:
            raise ValueError("gate repository does not match GitHub authority repository")
        raw_request = payload["request"]
        if not isinstance(raw_request, dict):
            raise ValueError("gate request must be an object")
        decision_id = _global_decision_id(self.repository, raw_request)
        path = self._decision_path(decision_id)

        for _attempt in range(5):
            head, existing = self.store.read_json(path)
            if existing is not None:
                if str(existing.get("repository", "")) != self.repository:
                    return {
                        "backend": "GITHUB",
                        "status": "CONFLICT",
                        "authority_revision": head,
                        "decision": existing,
                    }
                return {
                    "backend": "GITHUB",
                    "status": _workflow_status(existing),
                    "authority_revision": head,
                    "decision": existing,
                }

            now = _utc_now()
            record: dict[str, object] = {
                "id": decision_id,
                "clarification_id": str(payload["id"]),
                "repository": self.repository,
                "branch": str(payload["branch"]),
                "subject_revision": str(payload["subject_revision"]),
                "component_id": str(payload["component_id"]),
                "request": copy.deepcopy(raw_request),
                "status": "PENDING",
                "answer": None,
                "resolution_source": None,
                "resolved_by": None,
                "application_status": "NOT_APPLIED",
                "applied_revision": None,
                "issue_number": None,
                "issue_url": None,
                "created_at": now,
                "updated_at": now,
            }
            try:
                new_head = self.store.write_json(
                    path,
                    record,
                    head,
                    f"ptsip: register decision {decision_id}",
                )
            except AuthorityConflict:
                continue
            return {
                "backend": "GITHUB",
                "status": "DECISION_REQUIRED",
                "authority_revision": new_head,
                "decision": record,
            }
        raise CoordinationUnavailable("Unable to register PTSIP decision after repeated authority races")

    def decision(self, payload: dict[str, Any]) -> dict[str, object]:
        decision_id = str(payload.get("decision_id", ""))
        if not decision_id:
            raise ValueError("decision_id is required")
        head, record = self.store.read_json(self._decision_path(decision_id))
        if record is None:
            raise RuntimeError(f"GitHub-coordinated decision does not exist: {decision_id}")
        return {
            "backend": "GITHUB",
            "status": _workflow_status(record),
            "authority_revision": head,
            "decision": record,
        }

    def resolve(self, payload: dict[str, Any]) -> dict[str, object]:
        decision_id = str(payload.get("decision_id", ""))
        raw_answer = payload.get("answer")
        if not decision_id or not isinstance(raw_answer, dict):
            raise ValueError("decision_id and answer are required")
        answer = _answer_from_mapping(raw_answer)
        validation = validate_answer(answer)
        if not validation.valid:
            return {
                "backend": "GITHUB",
                "status": "CONFLICT",
                "validation": validation.as_dict(),
            }
        source = str(payload.get("source") or "AGENT_CHAT")
        if source not in {"AGENT_CHAT", "PROJECT_ADOPTION"}:
            raise ValueError("unsupported GitHub authority resolution source")
        actor = str(payload.get("actor") or "coding-agent-session")
        path = self._decision_path(decision_id)

        for _attempt in range(5):
            head, record = self.store.read_json(path)
            if record is None:
                raise RuntimeError(f"GitHub-coordinated decision does not exist: {decision_id}")
            status = str(record.get("status", ""))
            if status == "RESOLVED":
                return {
                    "backend": "GITHUB",
                    "status": "ALREADY_RESOLVED",
                    "accepted": False,
                    "authority_revision": head,
                    "decision": record,
                    "validation": validation.as_dict(),
                }
            if status != "PENDING":
                return {
                    "backend": "GITHUB",
                    "status": status or "DECISION_ERROR",
                    "accepted": False,
                    "authority_revision": head,
                    "decision": record,
                    "validation": validation.as_dict(),
                }

            resolved = copy.deepcopy(record)
            resolved["status"] = "RESOLVED"
            resolved["answer"] = answer.as_dict()
            resolved["resolution_source"] = source
            resolved["resolved_by"] = actor
            resolved["updated_at"] = _utc_now()
            try:
                new_head = self.store.write_json(
                    path,
                    resolved,
                    head,
                    f"ptsip: resolve decision {decision_id}",
                )
            except AuthorityConflict:
                continue
            return {
                "backend": "GITHUB",
                "status": "RESOLVED",
                "accepted": True,
                "authority_revision": new_head,
                "decision": resolved,
                "validation": validation.as_dict(),
            }
        raise CoordinationUnavailable("Unable to resolve PTSIP decision after repeated authority races")

    def application(self, payload: dict[str, Any]) -> dict[str, object]:
        decision_id = str(payload.get("decision_id", ""))
        status = str(payload.get("status", ""))
        if not decision_id:
            raise ValueError("decision_id is required")
        if status not in {"LOCAL_APPLIED", "FAILED", "STALE"}:
            raise ValueError("unsupported agent application status")
        return {
            "backend": "GITHUB",
            "status": status,
            "decision_id": decision_id,
            "applied_revision": str(payload.get("applied_revision") or "") or None,
        }
