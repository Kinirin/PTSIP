from __future__ import annotations

import base64
import json
import os
import time
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass
from pathlib import Path
from typing import Any


@dataclass(frozen=True)
class GitHubIssue:
    number: int
    url: str


class GitHubAPIError(RuntimeError):
    pass


class GitHubAppClient:
    def __init__(self, app_id: str | None = None, private_key: str | None = None, api_url: str | None = None):
        self.app_id = app_id or os.environ.get("PTSIP_GITHUB_APP_ID")
        configured_key = private_key or os.environ.get("PTSIP_GITHUB_PRIVATE_KEY")
        key_path = os.environ.get("PTSIP_GITHUB_PRIVATE_KEY_PATH")
        if configured_key is None and key_path:
            configured_key = Path(key_path).expanduser().read_text(encoding="utf-8")
        self.private_key = configured_key
        self.api_url = (api_url or os.environ.get("PTSIP_GITHUB_API_URL") or "https://api.github.com").rstrip("/")
        self._tokens: dict[int, tuple[str, float]] = {}

    def _app_jwt(self) -> str:
        if not self.app_id or not self.private_key:
            raise GitHubAPIError("GitHub App credentials are not configured")
        try:
            import jwt
        except ImportError as exc:
            raise GitHubAPIError("PyJWT[crypto] is required for GitHub App authentication; install ptsip[github-app]") from exc
        now = int(time.time())
        token = jwt.encode(
            {"iat": now - 60, "exp": now + 9 * 60, "iss": self.app_id},
            self.private_key,
            algorithm="RS256",
        )
        return str(token)

    def _request(
        self,
        method: str,
        path: str,
        token: str,
        payload: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        data = json.dumps(payload).encode("utf-8") if payload is not None else None
        request = urllib.request.Request(
            self.api_url + path,
            data=data,
            method=method,
            headers={
                "Accept": "application/vnd.github+json",
                "Authorization": f"Bearer {token}",
                "X-GitHub-Api-Version": "2022-11-28",
                "Content-Type": "application/json",
                "User-Agent": "ptsip-github-app/0.3.1",
            },
        )
        try:
            with urllib.request.urlopen(request, timeout=30) as response:
                raw = response.read()
        except urllib.error.HTTPError as exc:
            detail = exc.read().decode("utf-8", errors="replace")
            raise GitHubAPIError(f"GitHub API {method} {path} failed: HTTP {exc.code}: {detail}") from exc
        if not raw:
            return {}
        parsed = json.loads(raw.decode("utf-8"))
        if not isinstance(parsed, dict):
            raise GitHubAPIError("GitHub API returned a non-object response")
        return parsed

    def repository_installation(self, repository: str) -> int:
        if repository.count("/") != 1:
            raise GitHubAPIError("repository must use owner/repository form")
        owner, repo = repository.split("/", 1)
        payload = self._request(
            "GET",
            f"/repos/{urllib.parse.quote(owner)}/{urllib.parse.quote(repo)}/installation",
            self._app_jwt(),
        )
        installation_id = payload.get("id")
        if not isinstance(installation_id, int):
            raise GitHubAPIError("GitHub did not return a repository installation id")
        return installation_id

    def installation_token(self, installation_id: int) -> str:
        cached = self._tokens.get(installation_id)
        now = time.time()
        if cached and cached[1] - 60 > now:
            return cached[0]
        payload = self._request(
            "POST",
            f"/app/installations/{installation_id}/access_tokens",
            self._app_jwt(),
            {},
        )
        token = str(payload.get("token", ""))
        if not token:
            raise GitHubAPIError("GitHub did not return an installation token")
        self._tokens[installation_id] = (token, now + 50 * 60)
        return token

    def create_issue(self, repository: str, installation_id: int, title: str, body: str) -> GitHubIssue:
        token = self.installation_token(installation_id)
        payload = self._request("POST", f"/repos/{repository}/issues", token, {"title": title, "body": body})
        return GitHubIssue(int(payload["number"]), str(payload["html_url"]))

    def update_issue_state(self, repository: str, installation_id: int, issue_number: int, state: str) -> None:
        token = self.installation_token(installation_id)
        self._request("PATCH", f"/repos/{repository}/issues/{issue_number}", token, {"state": state})

    def add_issue_comment(self, repository: str, installation_id: int, issue_number: int, body: str) -> None:
        token = self.installation_token(installation_id)
        self._request("POST", f"/repos/{repository}/issues/{issue_number}/comments", token, {"body": body})

    def permission(self, repository: str, installation_id: int, username: str) -> str:
        token = self.installation_token(installation_id)
        payload = self._request(
            "GET", f"/repos/{repository}/collaborators/{urllib.parse.quote(username)}/permission", token
        )
        return str(payload.get("permission", "none"))

    def branch_head(self, repository: str, installation_id: int, branch: str) -> str:
        token = self.installation_token(installation_id)
        encoded = urllib.parse.quote(branch, safe="")
        payload = self._request("GET", f"/repos/{repository}/git/ref/heads/{encoded}", token)
        obj = payload.get("object")
        if not isinstance(obj, dict) or not obj.get("sha"):
            raise GitHubAPIError("GitHub branch ref did not contain a SHA")
        return str(obj["sha"])

    def file_text(self, repository: str, installation_id: int, path: str, ref: str) -> str | None:
        token = self.installation_token(installation_id)
        encoded_path = "/".join(urllib.parse.quote(part) for part in path.split("/"))
        try:
            payload = self._request(
                "GET",
                f"/repos/{repository}/contents/{encoded_path}?ref={urllib.parse.quote(ref, safe='')}",
                token,
            )
        except GitHubAPIError as exc:
            if "HTTP 404" in str(exc):
                return None
            raise
        content = payload.get("content")
        if not isinstance(content, str):
            return None
        return base64.b64decode(content.encode("ascii")).decode("utf-8-sig")

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
        token = self.installation_token(installation_id)
        parent = self._request("GET", f"/repos/{repository}/git/commits/{parent_sha}", token)
        tree = parent.get("tree")
        if not isinstance(tree, dict) or not tree.get("sha"):
            raise GitHubAPIError("Parent commit did not contain a tree SHA")
        created_tree = self._request(
            "POST",
            f"/repos/{repository}/git/trees",
            token,
            {
                "base_tree": str(tree["sha"]),
                "tree": [{"path": path, "mode": "100644", "type": "blob", "content": content}],
            },
        )
        commit = self._request(
            "POST",
            f"/repos/{repository}/git/commits",
            token,
            {"message": message, "tree": str(created_tree["sha"]), "parents": [parent_sha]},
        )
        commit_sha = str(commit["sha"])
        encoded_branch = urllib.parse.quote(branch, safe="")
        self._request(
            "PATCH",
            f"/repos/{repository}/git/refs/heads/{encoded_branch}",
            token,
            {"sha": commit_sha, "force": False},
        )
        return commit_sha
