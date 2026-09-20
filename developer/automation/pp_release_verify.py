from __future__ import annotations

import argparse
import json
import subprocess
import sys
from pathlib import Path
from typing import Mapping, Sequence

import yaml

from developer.automation.policy_loader import repository_root
from developer.automation.pp_remote_verify import verify_commit


class PPReleaseVerifyError(RuntimeError):
    def __init__(self, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code


def _git(root: Path, *args: str) -> str:
    result = subprocess.run(
        ["git", *args],
        cwd=root,
        check=False,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    if result.returncode != 0:
        raise PPReleaseVerifyError(
            "GIT_OPERATION_FAILED",
            result.stderr.strip() or f"git {' '.join(args)} failed",
        )
    return result.stdout.strip()


def _load_yaml(path: Path) -> dict[str, object]:
    try:
        value = yaml.safe_load(path.read_text(encoding="utf-8"))
    except Exception as exc:
        raise PPReleaseVerifyError(
            "PP_RELEASE_ASSET_INVALID",
            f"{path} is not valid UTF-8 YAML: {exc}",
        ) from exc
    if not isinstance(value, dict):
        raise PPReleaseVerifyError(
            "PP_RELEASE_ASSET_INVALID",
            f"{path} must be a YAML mapping.",
        )
    return value


def verify_exact_release_snapshot(
    root: str | Path | None = None,
    *,
    expected_sha: str,
) -> dict[str, object]:
    repo = repository_root(root)
    resolved = _git(repo, "rev-parse", "--verify", f"{expected_sha}^{{commit}}")
    head = _git(repo, "rev-parse", "--verify", "HEAD^{commit}")
    if head != resolved:
        raise PPReleaseVerifyError(
            "RELEASE_EXACT_SHA_MISMATCH",
            f"checked-out HEAD {head} does not match requested release SHA {resolved}.",
        )

    try:
        commit_result = verify_commit(repo, resolved)
    except Exception as exc:
        code = getattr(exc, "code", "PP_RELEASE_COMMIT_VERIFY_FAILED")
        raise PPReleaseVerifyError(
            code,
            f"exact release commit verification failed: {exc}",
        ) from exc

    canonical_registry = repo / "registry" / "project-profile-contracts.yaml"
    embedded_registry = (
        repo / "src" / "ptsip" / "specdata" / "project-profile-contracts.yaml"
    )
    if canonical_registry.read_bytes() != embedded_registry.read_bytes():
        raise PPReleaseVerifyError(
            "PP_RUNTIME_REGISTRY_PROJECTION_MISMATCH",
            "canonical and embedded Project Profile registries differ.",
        )

    registry = _load_yaml(canonical_registry)
    current = registry.get("current")
    contracts = registry.get("contracts")
    if not isinstance(current, str) or not isinstance(contracts, list):
        raise PPReleaseVerifyError(
            "PP_RELEASE_REGISTRY_INVALID",
            "Project Profile registry current/contracts are invalid.",
        )

    current_rows = [
        row
        for row in contracts
        if isinstance(row, Mapping) and row.get("version") == current
    ]
    if len(current_rows) != 1 or current_rows[0].get("lifecycle") != "CURRENT":
        raise PPReleaseVerifyError(
            "PP_RELEASE_REGISTRY_INVALID",
            "release PP current contract must resolve exactly once as CURRENT.",
        )
    schema_path = current_rows[0].get("schema")
    if not isinstance(schema_path, str):
        raise PPReleaseVerifyError(
            "PP_RELEASE_REGISTRY_INVALID",
            "release PP current contract has no canonical schema.",
        )

    canonical_schema = repo / schema_path
    embedded_schema = repo / "src" / "ptsip" / "specdata" / canonical_schema.name
    if (
        not canonical_schema.is_file()
        or not embedded_schema.is_file()
        or canonical_schema.read_bytes() != embedded_schema.read_bytes()
    ):
        raise PPReleaseVerifyError(
            "PP_RELEASE_SCHEMA_PROJECTION_MISMATCH",
            "current canonical and embedded Project Profile schemas differ.",
        )

    source_root = str(repo / "src")
    if source_root not in sys.path:
        sys.path.insert(0, source_root)

    from ptsip.profile_compatibility import current_project_profile_target
    from ptsip.profile_identity import CURRENT_PROJECT_PROFILE_VERSION
    from ptsip.project_profile_contracts import current_runtime_project_profile_contract

    runtime = current_runtime_project_profile_contract()
    target = current_project_profile_target()
    expected_schema_resource = canonical_schema.name

    if CURRENT_PROJECT_PROFILE_VERSION != current or runtime.version != current:
        raise PPReleaseVerifyError(
            "PP_RELEASE_RUNTIME_IDENTITY_MISMATCH",
            (
                "runtime current Project Profile identity does not match canonical "
                f"registry current {current!r}."
            ),
        )
    if runtime.schema_resource != expected_schema_resource:
        raise PPReleaseVerifyError(
            "PP_RELEASE_RUNTIME_SCHEMA_MISMATCH",
            "runtime current schema resource does not match canonical registry.",
        )
    if (
        target.contract.canonical != current
        or target.schema_resource != expected_schema_resource
    ):
        raise PPReleaseVerifyError(
            "PP_RELEASE_TARGET_IDENTITY_MISMATCH",
            "current Project Profile target does not match canonical registry.",
        )

    return {
        "status": "PASS",
        "source_sha": resolved,
        "project_profile": current,
        "schema": schema_path,
        "commit_classification": commit_result.classification,
        "transition_triggered": commit_result.triggered,
    }


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Verify the exact-SHA Project Profile release snapshot without mutation."
    )
    parser.add_argument("--repository", default=".")
    parser.add_argument("--sha", required=True)
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        result = verify_exact_release_snapshot(
            args.repository,
            expected_sha=args.sha,
        )
    except PPReleaseVerifyError as exc:
        print(
            json.dumps(
                {"status": "FAIL", "code": exc.code, "message": str(exc)},
                sort_keys=True,
            )
        )
        return 2

    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
