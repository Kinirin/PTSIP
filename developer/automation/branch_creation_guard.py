from __future__ import annotations

import argparse
import json
import re
from dataclasses import dataclass
from typing import Sequence


DEVELOPMENT_VERSION_PATTERN = re.compile(r"^dev/[0-9]\.[0-9]\.[0-9]$")
PROJECT_PROFILE_PATTERN = re.compile(r"^pp\.[0-9]\.[0-9]{2}$")
LEGACY_TOOL_PATTERN = re.compile(r"^tool-0\.3\.[0-9]-.*$")
USER_EXPLICIT = "USER_EXPLICIT"
DEVELOPMENT_VERSION_BRANCH = "DEVELOPMENT_VERSION_BRANCH"


class BranchCreationPolicyError(RuntimeError):
    def __init__(self, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code


@dataclass(frozen=True)
class BranchCreationDecision:
    status: str
    branch_name: str
    branch_class: str
    authorization_source: str
    request_kind: str

    def as_dict(self) -> dict[str, str]:
        return {
            "status": self.status,
            "branch_name": self.branch_name,
            "branch_class": self.branch_class,
            "authorization_source": self.authorization_source,
            "request_kind": self.request_kind,
        }


def validate_creation(
    candidate: str,
    approved_name: str,
    *,
    authorization_source: str,
    request_kind: str,
) -> BranchCreationDecision:
    if authorization_source != USER_EXPLICIT:
        raise BranchCreationPolicyError(
            "BRANCH_CREATION_REQUIRES_USER_EXPLICIT",
            "branch creation requires USER_EXPLICIT authorization",
        )
    if request_kind != DEVELOPMENT_VERSION_BRANCH:
        raise BranchCreationPolicyError(
            "UNREGISTERED_BRANCH_REQUEST_KIND",
            f"unsupported branch request kind: {request_kind}",
        )
    if not approved_name:
        raise BranchCreationPolicyError(
            "APPROVED_BRANCH_NAME_REQUIRED",
            "the exact user-approved branch name is required",
        )
    if candidate != approved_name:
        raise BranchCreationPolicyError(
            "BRANCH_NAME_NOT_EXACTLY_APPROVED",
            f"candidate {candidate!r} does not equal approved name {approved_name!r}",
        )
    if DEVELOPMENT_VERSION_PATTERN.fullmatch(candidate) is None:
        raise BranchCreationPolicyError(
            "UNAUTHORIZED_BRANCH_NAME",
            "current creation authority only permits dev/[0-9].[0-9].[0-9]",
        )
    return BranchCreationDecision(
        status="AUTHORIZED",
        branch_name=candidate,
        branch_class="DEVELOPMENT_VERSION",
        authorization_source=authorization_source,
        request_kind=request_kind,
    )


def project_profile_transition(
    branch_name: str,
    previous_profile: str,
    next_profile: str,
) -> dict[str, object]:
    if DEVELOPMENT_VERSION_PATTERN.fullmatch(branch_name) is None:
        raise BranchCreationPolicyError(
            "INVALID_DEVELOPMENT_BRANCH",
            f"branch is not an authorized development-version shape: {branch_name}",
        )
    for value in (previous_profile, next_profile):
        if PROJECT_PROFILE_PATTERN.fullmatch(value) is None:
            raise BranchCreationPolicyError(
                "INVALID_PROJECT_PROFILE_IDENTITY",
                f"invalid Project Profile identity: {value}",
            )
    return {
        "status": "NO_BRANCH_IDENTITY_CHANGE",
        "branch_name": branch_name,
        "previous_profile": previous_profile,
        "next_profile": next_profile,
        "branch_change_required": False,
        "branch_creation_authorized": False,
    }


def classify_existing(branch_name: str) -> dict[str, str]:
    if DEVELOPMENT_VERSION_PATTERN.fullmatch(branch_name):
        state = "AUTHORIZED_DEVELOPMENT_VERSION"
    elif LEGACY_TOOL_PATTERN.fullmatch(branch_name):
        state = "GRANDFATHERED_RETENTION"
    else:
        state = "UNREGISTERED_SHAPE"
    return {"branch_name": branch_name, "classification": state}


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Deterministic developer branch creation guard.")
    sub = parser.add_subparsers(dest="command", required=True)

    validate = sub.add_parser("validate")
    validate.add_argument("--candidate", required=True)
    validate.add_argument("--approved-name", required=True)
    validate.add_argument("--authorization-source", required=True)
    validate.add_argument("--request-kind", required=True)

    profile = sub.add_parser("profile-transition")
    profile.add_argument("--branch", required=True)
    profile.add_argument("--from-profile", required=True)
    profile.add_argument("--to-profile", required=True)

    existing = sub.add_parser("classify-existing")
    existing.add_argument("--branch", required=True)
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        if args.command == "validate":
            result = validate_creation(
                args.candidate,
                args.approved_name,
                authorization_source=args.authorization_source,
                request_kind=args.request_kind,
            ).as_dict()
        elif args.command == "profile-transition":
            result = project_profile_transition(
                args.branch,
                args.from_profile,
                args.to_profile,
            )
        else:
            result = classify_existing(args.branch)
    except BranchCreationPolicyError as exc:
        print(json.dumps({"status": "BLOCKED", "code": exc.code, "message": str(exc)}, sort_keys=True))
        return 2
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
