from __future__ import annotations

import argparse
import json
import os
import re
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass
from typing import Mapping, Sequence

from developer.automation.branch_creation_guard import (
    BranchCreationPolicyError,
    GITHUB_CREATE_BRANCH_API,
    validate_creation,
)


COMMAND_REGISTRY: Mapping[str, Mapping[str, object]] = {
    "commands": {
        "mutation": False,
        "purpose": "List the closed branch-control command vocabulary.",
    },
    "list": {
        "mutation": False,
        "purpose": "List remote GitHub branches.",
    },
    "inspect": {
        "mutation": False,
        "purpose": "Inspect one exact remote GitHub branch.",
    },
    "create": {
        "mutation": True,
        "purpose": "Create one exact user-approved development branch through the GitHub API.",
        "creation_mechanism": GITHUB_CREATE_BRANCH_API,
        "requires_exact_user_approved_name": True,
    },
    "recreate": {
        "mutation": True,
        "purpose": "Delete one fully-integrated development branch and recreate it from an exact base ref.",
        "creation_mechanism": GITHUB_CREATE_BRANCH_API,
        "requires_exact_user_approved_name": True,
        "requires_fully_contained_existing_branch": True,
    },
}
DEFAULT_API_ROOT = "https://api.github.com"
REPOSITORY_PATTERN = re.compile(r"^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$")


class BranchControlError(RuntimeError):
    def __init__(self, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code


@dataclass(frozen=True)
class GitHubClient:
    repository: str
    token: str | None = None
    api_root: str = DEFAULT_API_ROOT

    def __post_init__(self) -> None:
        if REPOSITORY_PATTERN.fullmatch(self.repository) is None:
            raise BranchControlError("INVALID_REPOSITORY", "repository must be in owner/name form")

    def request(
        self,
        method: str,
        path: str,
        *,
        payload: Mapping[str, object] | None = None,
    ) -> object:
        url = f"{self.api_root.rstrip('/')}{path}"
        data = None
        headers = {
            "Accept": "application/vnd.github+json",
            "X-GitHub-Api-Version": "2022-11-28",
            "User-Agent": "PTSIP-branch-control",
        }
        if self.token:
            headers["Authorization"] = f"Bearer {self.token}"
        if payload is not None:
            data = json.dumps(dict(payload)).encode("utf-8")
            headers["Content-Type"] = "application/json"
        request = urllib.request.Request(url, data=data, headers=headers, method=method)
        try:
            with urllib.request.urlopen(request) as response:
                raw = response.read()
        except urllib.error.HTTPError as exc:
            body = exc.read().decode("utf-8", errors="replace")
            raise BranchControlError(
                "GITHUB_API_ERROR",
                f"GitHub API returned HTTP {exc.code}: {body}",
            ) from exc
        except urllib.error.URLError as exc:
            raise BranchControlError("GITHUB_API_UNREACHABLE", str(exc.reason)) from exc
        if not raw:
            return {}
        return json.loads(raw.decode("utf-8"))

    def _encoded_ref(self, ref: str) -> str:
        return urllib.parse.quote(ref.removeprefix("refs/heads/"), safe="")

    def list_branches(self) -> list[dict[str, object]]:
        result: list[dict[str, object]] = []
        page = 1
        while True:
            payload = self.request(
                "GET",
                f"/repos/{self.repository}/branches?per_page=100&page={page}",
            )
            if not isinstance(payload, list):
                raise BranchControlError("INVALID_GITHUB_RESPONSE", "branch list response is not a list")
            result.extend(item for item in payload if isinstance(item, dict))
            if len(payload) < 100:
                return result
            page += 1

    def inspect_branch(self, branch: str) -> dict[str, object]:
        payload = self.request(
            "GET",
            f"/repos/{self.repository}/branches/{self._encoded_ref(branch)}",
        )
        if not isinstance(payload, dict):
            raise BranchControlError("INVALID_GITHUB_RESPONSE", "branch response is not an object")
        return payload

    def resolve_ref_sha(self, ref: str) -> str:
        payload = self.request(
            "GET",
            f"/repos/{self.repository}/git/ref/heads/{self._encoded_ref(ref)}",
        )
        if not isinstance(payload, dict):
            raise BranchControlError("INVALID_GITHUB_RESPONSE", "ref response is not an object")
        obj = payload.get("object")
        if not isinstance(obj, dict) or not isinstance(obj.get("sha"), str):
            raise BranchControlError("INVALID_GITHUB_RESPONSE", "ref response has no object.sha")
        return str(obj["sha"])

    def compare(self, base: str, head: str) -> dict[str, object]:
        encoded_base = urllib.parse.quote(base, safe="")
        encoded_head = urllib.parse.quote(head, safe="")
        payload = self.request(
            "GET",
            f"/repos/{self.repository}/compare/{encoded_base}...{encoded_head}",
        )
        if not isinstance(payload, dict):
            raise BranchControlError("INVALID_GITHUB_RESPONSE", "compare response is not an object")
        return payload

    def create_branch_at_sha(self, branch: str, sha: str) -> dict[str, object]:
        if not self.token:
            raise BranchControlError("GITHUB_TOKEN_REQUIRED", "branch creation requires GITHUB_TOKEN or GH_TOKEN")
        payload = self.request(
            "POST",
            f"/repos/{self.repository}/git/refs",
            payload={"ref": f"refs/heads/{branch}", "sha": sha},
        )
        if not isinstance(payload, dict):
            raise BranchControlError("INVALID_GITHUB_RESPONSE", "create-ref response is not an object")
        return payload

    def delete_branch_ref(self, branch: str) -> None:
        if not self.token:
            raise BranchControlError("GITHUB_TOKEN_REQUIRED", "branch deletion requires GITHUB_TOKEN or GH_TOKEN")
        self.request(
            "DELETE",
            f"/repos/{self.repository}/git/refs/heads/{self._encoded_ref(branch)}",
        )

    def create_branch(self, branch: str, base_ref: str) -> dict[str, object]:
        base_sha = self.resolve_ref_sha(base_ref)
        payload = self.create_branch_at_sha(branch, base_sha)
        return {
            "status": "CREATED",
            "repository": self.repository,
            "branch": branch,
            "base_ref": base_ref,
            "base_sha": base_sha,
            "creation_mechanism": GITHUB_CREATE_BRANCH_API,
            "ref": payload.get("ref"),
        }


def _token() -> str | None:
    return os.environ.get("GITHUB_TOKEN") or os.environ.get("GH_TOKEN")


def _client(repository: str) -> GitHubClient:
    return GitHubClient(repository=repository, token=_token())


def _validate_client(repository: str, client: GitHubClient | None) -> GitHubClient:
    selected = client if client is not None else _client(repository)
    if selected.repository != repository:
        raise BranchControlError(
            "REPOSITORY_BINDING_MISMATCH",
            "client repository does not match requested repository",
        )
    return selected


def registered_commands() -> dict[str, object]:
    return {
        "status": "REGISTERED_COMMANDS",
        "commands": {name: dict(spec) for name, spec in COMMAND_REGISTRY.items()},
        "unregistered_operation": "FAIL_CLOSED",
        "canonical_entrypoint": "developer/automation/branch_control.py",
    }


def create_authorized_branch(
    *,
    repository: str,
    branch: str,
    approved_name: str,
    base_ref: str,
    client: GitHubClient | None = None,
) -> dict[str, object]:
    validate_creation(
        branch,
        approved_name,
        authorization_source="USER_EXPLICIT",
        request_kind="DEVELOPMENT_VERSION_BRANCH",
        creation_mechanism=GITHUB_CREATE_BRANCH_API,
    )
    return _validate_client(repository, client).create_branch(branch, base_ref)


def recreate_authorized_branch(
    *,
    repository: str,
    branch: str,
    approved_name: str,
    base_ref: str,
    client: GitHubClient | None = None,
) -> dict[str, object]:
    validate_creation(
        branch,
        approved_name,
        authorization_source="USER_EXPLICIT",
        request_kind="DEVELOPMENT_VERSION_BRANCH",
        creation_mechanism=GITHUB_CREATE_BRANCH_API,
    )
    if branch == base_ref:
        raise BranchControlError("RECREATE_BASE_EQUALS_BRANCH", "recreate base ref must differ from the branch")
    selected = _validate_client(repository, client)
    old_sha = selected.resolve_ref_sha(branch)
    comparison = selected.compare(branch, base_ref)
    if comparison.get("behind_by") != 0:
        raise BranchControlError(
            "BRANCH_HAS_UNMERGED_COMMITS",
            f"{branch} contains commits that are not contained in {base_ref}",
        )

    selected.delete_branch_ref(branch)
    try:
        base_sha = selected.resolve_ref_sha(base_ref)
        selected.create_branch_at_sha(branch, base_sha)
        recreated_sha = selected.resolve_ref_sha(branch)
        final_base_sha = selected.resolve_ref_sha(base_ref)
        if recreated_sha != base_sha or final_base_sha != base_sha:
            raise BranchControlError(
                "RECREATE_BASE_MOVED_OR_REF_MISMATCH",
                "base ref moved during recreation or recreated ref does not match the selected base SHA",
            )
    except Exception as exc:
        try:
            selected.create_branch_at_sha(branch, old_sha)
        except Exception as rollback_exc:
            raise BranchControlError(
                "RECREATE_FAILED_ROLLBACK_FAILED",
                f"recreate failed ({exc}); rollback to {old_sha} also failed ({rollback_exc})",
            ) from rollback_exc
        if isinstance(exc, BranchControlError):
            raise BranchControlError(
                "RECREATE_FAILED_ROLLED_BACK",
                f"{exc}; original ref restored to {old_sha}",
            ) from exc
        raise BranchControlError(
            "RECREATE_FAILED_ROLLED_BACK",
            f"recreate failed; original ref restored to {old_sha}",
        ) from exc

    return {
        "status": "RECREATED",
        "repository": repository,
        "branch": branch,
        "base_ref": base_ref,
        "old_sha": old_sha,
        "new_sha": recreated_sha,
        "creation_mechanism": GITHUB_CREATE_BRANCH_API,
        "safety": "OLD_BRANCH_FULLY_CONTAINED_IN_BASE",
    }


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Canonical closed-vocabulary branch control plane.")
    sub = parser.add_subparsers(dest="command", required=True)

    sub.add_parser("commands")

    list_cmd = sub.add_parser("list")
    list_cmd.add_argument("--repository", required=True)

    inspect_cmd = sub.add_parser("inspect")
    inspect_cmd.add_argument("--repository", required=True)
    inspect_cmd.add_argument("--branch", required=True)

    create_cmd = sub.add_parser("create")
    create_cmd.add_argument("--repository", required=True)
    create_cmd.add_argument("--branch", required=True)
    create_cmd.add_argument("--approved-name", required=True)
    create_cmd.add_argument("--base-ref", required=True)

    recreate_cmd = sub.add_parser("recreate")
    recreate_cmd.add_argument("--repository", required=True)
    recreate_cmd.add_argument("--branch", required=True)
    recreate_cmd.add_argument("--approved-name", required=True)
    recreate_cmd.add_argument("--base-ref", required=True)
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    arguments = list(argv) if argv is not None else None
    if arguments and arguments[0] not in COMMAND_REGISTRY:
        print(json.dumps({
            "status": "BLOCKED",
            "code": "UNREGISTERED_BRANCH_COMMAND",
            "command": arguments[0],
            "registered_commands": sorted(COMMAND_REGISTRY),
        }, sort_keys=True))
        return 2

    args = _parser().parse_args(arguments)
    try:
        if args.command == "commands":
            result = registered_commands()
        elif args.command == "list":
            branches = _client(args.repository).list_branches()
            result = {
                "status": "OK",
                "repository": args.repository,
                "branches": [
                    {"name": item.get("name"), "sha": (item.get("commit") or {}).get("sha")}
                    for item in branches
                ],
            }
        elif args.command == "inspect":
            item = _client(args.repository).inspect_branch(args.branch)
            result = {
                "status": "OK",
                "repository": args.repository,
                "branch": item.get("name"),
                "sha": (item.get("commit") or {}).get("sha"),
                "protected": item.get("protected"),
            }
        elif args.command == "create":
            result = create_authorized_branch(
                repository=args.repository,
                branch=args.branch,
                approved_name=args.approved_name,
                base_ref=args.base_ref,
            )
        else:
            result = recreate_authorized_branch(
                repository=args.repository,
                branch=args.branch,
                approved_name=args.approved_name,
                base_ref=args.base_ref,
            )
    except (BranchControlError, BranchCreationPolicyError) as exc:
        code = getattr(exc, "code", "BRANCH_CONTROL_ERROR")
        print(json.dumps({"status": "BLOCKED", "code": code, "message": str(exc)}, sort_keys=True))
        return 2

    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
