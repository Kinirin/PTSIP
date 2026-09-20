from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Sequence

from developer.automation.policy_loader import repository_root
from developer.automation.pp_transition_delta import compare_git_snapshots
from developer.automation.pp_transition_reconciler import reconcile_staged_transition
from developer.automation.pp_remote_verify import verify_staged_parent_authority
from developer.automation.project_profile_registry import (
    validate_project_profile_registry_plane,
)


class PPPreCommitError(RuntimeError):
    pass


def verify_staged_pp_transition(root: str | Path | None = None) -> dict[str, object]:
    repo = repository_root(root)

    try:
        verify_staged_parent_authority(repo)
        reconciliation = reconcile_staged_transition(repo, apply=True)
    except Exception as exc:
        code = getattr(exc, "code", "PP_RECONCILIATION_FAILED")
        raise PPPreCommitError(f"{code}: {exc}") from exc

    registry_failures = validate_project_profile_registry_plane(repo)
    if registry_failures:
        raise PPPreCommitError(
            "PP_REGISTRY_VALIDATION_FAILED: " + " | ".join(registry_failures)
        )

    delta = compare_git_snapshots(repo, base_revision="HEAD", staged=True)
    if not delta.valid:
        raise PPPreCommitError(
            "PP_STAGED_DELTA_INVALID: " + json.dumps(delta.as_dict(), sort_keys=True)
        )
    if delta.triggered and not delta.candidate_already_reconciled:
        raise PPPreCommitError(
            "PP_STAGED_DELTA_NOT_RECONCILED: "
            + json.dumps(delta.as_dict(), sort_keys=True)
        )

    return {
        "status": "PASS",
        "reconciliation": reconciliation.status,
        "base_current": delta.base_current,
        "candidate_current": delta.candidate_current,
        "classification": delta.classification,
        "triggered": delta.triggered,
        "candidate_already_reconciled": delta.candidate_already_reconciled,
    }


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Auto-reconcile and verify MPD-0011 staged PP transitions."
    )
    parser.add_argument("--repository", default=".")
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        result = verify_staged_pp_transition(args.repository)
    except PPPreCommitError as exc:
        print(json.dumps({"status": "FAIL", "message": str(exc)}, sort_keys=True))
        return 2
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
