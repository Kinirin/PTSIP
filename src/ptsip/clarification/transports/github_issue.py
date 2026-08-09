from __future__ import annotations

import json
import re
import shutil
import subprocess
from dataclasses import asdict, dataclass
from pathlib import Path

from ...repository.remote import RepositoryRemote
from ...storage.local_state import clarification_directory
from ..model import ClarificationRequest
from ..render import render_issue


@dataclass(frozen=True)
class GitHubIssuePublication:
    clarification_id: str
    component_id: str
    repository: str
    status: str
    issue_number: int | None
    issue_url: str

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


def _valid_repo(value: str) -> bool:
    return bool(re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", value.strip()))


def resolve_repository(remote: RepositoryRemote | None, override: str | None) -> str:
    if override:
        target = override.strip()
        if not _valid_repo(target):
            raise ValueError("--repo must use GitHub owner/repository form")
        return target
    if remote and remote.provider == "github" and remote.repository:
        return remote.repository
    raise RuntimeError(
        "No GitHub origin was detected. Use --repo owner/repository to select the Consumer Repository explicitly."
    )


def _state_path(repository_root: str | Path) -> Path:
    return clarification_directory(repository_root) / "state.json"


def _load_state(repository_root: str | Path) -> dict[str, object]:
    path = _state_path(repository_root)
    if not path.exists():
        return {"format": "ptsip-clarification-state/v1", "issues": {}}
    try:
        payload = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise RuntimeError(f"Unable to read clarification state: {exc}") from exc
    if not isinstance(payload, dict) or not isinstance(payload.get("issues"), dict):
        raise RuntimeError("Clarification state is not a valid ptsip-clarification-state/v1 document.")
    return payload


def _save_state(repository_root: str | Path, state: dict[str, object]) -> None:
    path = _state_path(repository_root)
    path.parent.mkdir(parents=True, exist_ok=True)
    temp = path.with_suffix(".tmp")
    temp.write_text(json.dumps(state, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    temp.replace(path)


def _run_gh(args: list[str], input_text: str | None = None) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["gh", *args],
        input=input_text,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        check=False,
    )


def _issue_number(url: str) -> int | None:
    match = re.search(r"/issues/(\d+)(?:$|[/?#])", url)
    return int(match.group(1)) if match else None


def publish(
    repository_root: str | Path,
    remote: RepositoryRemote | None,
    repository_revision: str | None,
    requests: tuple[ClarificationRequest, ...],
    language: str,
    repo_override: str | None = None,
) -> tuple[GitHubIssuePublication, ...]:
    if not requests:
        return ()
    target = resolve_repository(remote, repo_override)
    state = _load_state(repository_root)
    issues = state["issues"]
    assert isinstance(issues, dict)
    needs_create = any(
        not (
            isinstance(issues.get(request.id), dict)
            and issues[request.id].get("repository") == target
            and issues[request.id].get("url")
        )
        for request in requests
    )
    if needs_create:
        if shutil.which("gh") is None:
            raise RuntimeError("GitHub CLI 'gh' is required for --publish github-issue.")
        auth = _run_gh(["auth", "status", "--hostname", "github.com"])
        if auth.returncode != 0:
            detail = auth.stderr.strip() or auth.stdout.strip() or "authentication failed"
            raise RuntimeError(f"GitHub CLI authentication is not ready: {detail}")

    results: list[GitHubIssuePublication] = []
    for request in requests:
        existing = issues.get(request.id)
        if isinstance(existing, dict) and existing.get("repository") == target and existing.get("url"):
            results.append(
                GitHubIssuePublication(
                    clarification_id=request.id,
                    component_id=request.component_id,
                    repository=target,
                    status="EXISTING",
                    issue_number=existing.get("number") if isinstance(existing.get("number"), int) else None,
                    issue_url=str(existing["url"]),
                )
            )
            continue

        title, body = render_issue(request, language, repository_revision)
        created = _run_gh(
            ["issue", "create", "--repo", target, "--title", title, "--body-file", "-"],
            input_text=body,
        )
        if created.returncode != 0:
            detail = created.stderr.strip() or created.stdout.strip() or "issue creation failed"
            raise RuntimeError(f"GitHub issue creation failed for {request.component_id}: {detail}")
        url = next((line.strip() for line in reversed(created.stdout.splitlines()) if line.strip()), "")
        if not url:
            raise RuntimeError(f"GitHub issue creation returned no issue URL for {request.component_id}.")
        number = _issue_number(url)
        issues[request.id] = {
            "repository": target,
            "number": number,
            "url": url,
            "component_id": request.component_id,
        }
        _save_state(repository_root, state)
        results.append(
            GitHubIssuePublication(
                clarification_id=request.id,
                component_id=request.component_id,
                repository=target,
                status="CREATED",
                issue_number=number,
                issue_url=url,
            )
        )
    return tuple(results)
