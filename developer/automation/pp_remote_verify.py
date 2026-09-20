from __future__ import annotations

import argparse
import json
import subprocess
from dataclasses import dataclass
from pathlib import Path
from typing import Mapping, Sequence

import yaml

from developer.automation.policy_loader import repository_root
from developer.automation.pp_transition_delta import (
    AuthorityState,
    GitIndexSnapshot,
    GitSnapshot,
    PPTransitionDeltaError,
    SnapshotSource,
    T2DeltaResult,
    evaluate_t2_authority_delta,
    load_authority_state,
)


CANONICAL_REGISTRY = "registry/project-profile-contracts.yaml"
EMBEDDED_REGISTRY = "src/ptsip/specdata/project-profile-contracts.yaml"
PUBLIC_PROFILE_CATALOG = "profiles/index.yaml"


class PPRemoteVerifyError(RuntimeError):
    def __init__(self, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code


@dataclass(frozen=True)
class CommitVerification:
    commit: str
    parents: tuple[str, ...]
    classification: str
    triggered: bool
    current: str | None

    def as_dict(self) -> dict[str, object]:
        return {
            "commit": self.commit,
            "parents": list(self.parents),
            "classification": self.classification,
            "triggered": self.triggered,
            "current": self.current,
        }


def _git(root: Path, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", *args],
        cwd=root,
        check=False,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )


def _require_git(root: Path, *args: str) -> str:
    result = _git(root, *args)
    if result.returncode != 0:
        raise PPRemoteVerifyError(
            "GIT_OPERATION_FAILED",
            result.stderr.strip() or f"git {' '.join(args)} failed",
        )
    return result.stdout.strip()


def commit_parents(root: Path, commit: str) -> tuple[str, ...]:
    raw = _require_git(root, "show", "-s", "--format=%P", commit)
    return tuple(item for item in raw.split() if item)


def introduced_commits(root: Path, base: str, head: str) -> tuple[str, ...]:
    raw = _require_git(root, "rev-list", "--reverse", "--topo-order", f"{base}..{head}")
    return tuple(line.strip() for line in raw.splitlines() if line.strip())


def _yaml(raw: bytes | None, *, label: str) -> dict[str, object]:
    if raw is None:
        raise PPRemoteVerifyError("SNAPSHOT_ASSET_MISSING", f"{label} is missing.")
    try:
        value = yaml.safe_load(raw.decode("utf-8"))
    except Exception as exc:
        raise PPRemoteVerifyError(
            "SNAPSHOT_ASSET_INVALID",
            f"{label} is not valid UTF-8 YAML: {exc}",
        ) from exc
    if not isinstance(value, dict):
        raise PPRemoteVerifyError(
            "SNAPSHOT_ASSET_INVALID",
            f"{label} must be a mapping.",
        )
    return value


def authority_fingerprint(state: AuthorityState) -> str:
    entries = {
        key: {
            "resource": value.resource,
            "contract": value.contract,
            "semantic_profile": state.semantic_profiles.get(value.resource),
            "declared_version": state.declared_profile_versions.get(value.resource),
        }
        for key, value in sorted(state.entries.items())
    }
    payload = {
        "current": state.current,
        "entries": entries,
        "schema_semantic": state.current_schema_semantic,
    }
    return json.dumps(payload, sort_keys=True, separators=(",", ":"), ensure_ascii=False)


def require_comparable_parent_authority(states: Sequence[AuthorityState]) -> None:
    if len(states) < 2:
        return
    fingerprints = {authority_fingerprint(state) for state in states}
    if len(fingerprints) != 1:
        details = ", ".join(
            f"{state.label}:{state.current}" for state in states
        )
        raise PPRemoteVerifyError(
            "DIVERGENT_PARENT_PP_AUTHORITY",
            "merge parents expose divergent/incomparable PP authority state: " + details,
        )


def staged_parent_states(root: str | Path | None = None) -> tuple[AuthorityState, ...]:
    repo = repository_root(root)
    parents = ["HEAD"]
    merge_head = repo / ".git" / "MERGE_HEAD"
    if merge_head.is_file():
        parents.extend(
            line.strip()
            for line in merge_head.read_text(encoding="utf-8").splitlines()
            if line.strip()
        )
    return tuple(load_authority_state(GitSnapshot(repo, parent)) for parent in parents)


def verify_staged_parent_authority(root: str | Path | None = None) -> None:
    states = staged_parent_states(root)
    require_comparable_parent_authority(states)


def _evaluate(parent: SnapshotSource, candidate: SnapshotSource) -> T2DeltaResult:
    base = load_authority_state(parent)
    target = load_authority_state(candidate)

    old_current_schema: bytes | None = None
    if base.current is not None:
        old_path = base.contract_schemas.get(base.current)
        if isinstance(old_path, str):
            old_current_schema = candidate.read_bytes(old_path)

    return evaluate_t2_authority_delta(
        base,
        target,
        base_schema_bytes_for_candidate_current=old_current_schema,
    )


def _validate_snapshot(source: SnapshotSource) -> AuthorityState:
    state = load_authority_state(source)
    canonical = source.read_bytes(CANONICAL_REGISTRY)
    embedded = source.read_bytes(EMBEDDED_REGISTRY)
    if canonical is None or embedded is None or canonical != embedded:
        raise PPRemoteVerifyError(
            "PP_RUNTIME_REGISTRY_PROJECTION_MISMATCH",
            f"{source.label}: canonical and embedded PP registries differ.",
        )

    registry = _yaml(canonical, label=f"{source.label}:{CANONICAL_REGISTRY}")
    catalog = _yaml(
        source.read_bytes(PUBLIC_PROFILE_CATALOG),
        label=f"{source.label}:{PUBLIC_PROFILE_CATALOG}",
    )
    current = registry.get("current")
    contracts = registry.get("contracts")
    transitions = registry.get("transitions")
    if (
        not isinstance(current, str)
        or not isinstance(contracts, list)
        or not isinstance(transitions, list)
    ):
        raise PPRemoteVerifyError(
            "PP_REGISTRY_INVALID",
            f"{source.label}: current/contracts/transitions are invalid.",
        )

    current_rows = [
        row
        for row in contracts
        if isinstance(row, Mapping) and row.get("version") == current
    ]
    if len(current_rows) != 1 or current_rows[0].get("lifecycle") != "CURRENT":
        raise PPRemoteVerifyError(
            "PP_REGISTRY_CURRENT_INVALID",
            f"{source.label}: current PP contract must resolve exactly once as CURRENT.",
        )
    current_row = current_rows[0]
    schema_path = current_row.get("schema")
    baseline = current_row.get("baseline")
    if not isinstance(schema_path, str) or not isinstance(baseline, str):
        raise PPRemoteVerifyError(
            "PP_REGISTRY_CURRENT_INVALID",
            f"{source.label}: current PP contract requires schema and baseline.",
        )

    schema = source.read_bytes(schema_path)
    embedded_schema = source.read_bytes(
        "src/ptsip/specdata/" + Path(schema_path).name
    )
    if schema is None or embedded_schema is None or schema != embedded_schema:
        raise PPRemoteVerifyError(
            "PP_SCHEMA_PROJECTION_MISMATCH",
            f"{source.label}: current canonical and embedded PP schemas differ.",
        )

    rows = catalog.get("profiles")
    if not isinstance(rows, list) or not rows:
        raise PPRemoteVerifyError(
            "PUBLIC_PROFILE_CATALOG_INVALID",
            f"{source.label}: Public Profile catalog is empty.",
        )
    registered: list[str] = []
    for row in rows:
        if not isinstance(row, Mapping):
            raise PPRemoteVerifyError(
                "PUBLIC_PROFILE_CATALOG_INVALID",
                f"{source.label}: Public Profile catalog entry is invalid.",
            )
        resource = row.get("resource")
        contract = row.get("contract")
        mode = row.get("responsibility_mode")
        if not isinstance(resource, str) or contract != current:
            raise PPRemoteVerifyError(
                "PUBLIC_PROFILE_CATALOG_INVALID",
                f"{source.label}: Public Profile catalog contract binding is stale.",
            )
        registered.append(resource)
        raw = source.read_bytes(f"profiles/{resource}")
        historical = source.read_bytes(f"{baseline}/{resource}")
        if raw is None or historical is None or raw != historical:
            raise PPRemoteVerifyError(
                "CURRENT_PP_BASELINE_MISMATCH",
                f"{source.label}: current baseline for {resource!r} is missing or differs.",
            )
        profile = _yaml(raw, label=f"{source.label}:profiles/{resource}")
        ptsip = profile.get("ptsip")
        responsibility = profile.get("responsibility_map")
        if not isinstance(ptsip, Mapping) or ptsip.get("version") != current:
            raise PPRemoteVerifyError(
                "PUBLIC_PROFILE_VERSION_MISMATCH",
                f"{source.label}: {resource!r} does not declare current PP {current!r}.",
            )
        if not isinstance(responsibility, Mapping) or responsibility.get("mode") != mode:
            raise PPRemoteVerifyError(
                "PUBLIC_PROFILE_MODE_MISMATCH",
                f"{source.label}: {resource!r} responsibility mode differs from catalog.",
            )

    if tuple(sorted(registered)) != state.discovered_resources:
        raise PPRemoteVerifyError(
            "PUBLIC_PROFILE_CATALOG_COVERAGE_MISMATCH",
            f"{source.label}: catalog does not exactly cover current Public Profile resources.",
        )
    return state


def _changed_history_paths(root: Path, parent: str, commit: str) -> tuple[tuple[str, str], ...]:
    raw = _require_git(
        root,
        "diff",
        "--name-status",
        parent,
        commit,
        "--",
        "profiles/history",
    )
    rows: list[tuple[str, str]] = []
    for line in raw.splitlines():
        if not line.strip():
            continue
        parts = line.split("\t")
        status = parts[0]
        path = parts[-1].replace("\\", "/")
        rows.append((status, path))
    return tuple(rows)


def _verify_transition_shape(
    root: Path,
    *,
    parent: str,
    commit: str,
    delta: T2DeltaResult,
) -> None:
    if (
        not delta.triggered
        or not delta.valid
        or not delta.candidate_already_reconciled
        or delta.base_current is None
        or delta.expected_next is None
    ):
        raise PPRemoteVerifyError(
            "PP_TRANSITION_NOT_ATOMIC",
            json.dumps(delta.as_dict(), sort_keys=True),
        )

    parent_source = GitSnapshot(root, parent)
    candidate = GitSnapshot(root, commit)
    registry = _yaml(
        candidate.read_bytes(CANONICAL_REGISTRY),
        label=f"{commit}:{CANONICAL_REGISTRY}",
    )
    contracts = registry.get("contracts")
    transitions = registry.get("transitions")
    if not isinstance(contracts, list) or not isinstance(transitions, list):
        raise PPRemoteVerifyError(
            "PP_TRANSITION_REGISTRY_INVALID",
            f"{commit}: contract registry transition plane is invalid.",
        )

    source_rows = [
        row
        for row in contracts
        if isinstance(row, Mapping) and row.get("version") == delta.base_current
    ]
    target_rows = [
        row
        for row in contracts
        if isinstance(row, Mapping) and row.get("version") == delta.expected_next
    ]
    if (
        len(source_rows) != 1
        or source_rows[0].get("lifecycle") != "SUPERSEDED"
        or len(target_rows) != 1
        or target_rows[0].get("lifecycle") != "CURRENT"
    ):
        raise PPRemoteVerifyError(
            "PP_TRANSITION_REGISTRY_INVALID",
            f"{commit}: source/target lifecycle transition is incomplete.",
        )
    if not any(
        isinstance(row, Mapping)
        and row.get("from") == delta.base_current
        and row.get("to") == delta.expected_next
        and row.get("kind") == "SEMANTIC_MIGRATION"
        for row in transitions
    ):
        raise PPRemoteVerifyError(
            "PP_TRANSITION_RECORD_MISSING",
            f"{commit}: semantic migration record is missing.",
        )

    expected_history_prefix = f"profiles/history/{delta.expected_next}/"
    for status, path in _changed_history_paths(root, parent, commit):
        if not status.startswith("A") or not path.startswith(expected_history_prefix):
            raise PPRemoteVerifyError(
                "HISTORICAL_BASELINE_MUTATION",
                f"{commit}: forbidden historical baseline change {status} {path}.",
            )

    parent_state = load_authority_state(parent_source)
    old_schema = parent_state.contract_schemas.get(delta.base_current)
    if isinstance(old_schema, str):
        if parent_source.read_bytes(old_schema) != candidate.read_bytes(old_schema):
            raise PPRemoteVerifyError(
                "SUPERSEDED_PP_SCHEMA_MUTATION",
                f"{commit}: superseded schema {old_schema!r} was not preserved.",
            )


def verify_commit(root: str | Path, commit: str) -> CommitVerification:
    repo = repository_root(root)
    parents = commit_parents(repo, commit)
    candidate_source = GitSnapshot(repo, commit)
    candidate_state = _validate_snapshot(candidate_source)

    if not parents:
        return CommitVerification(
            commit=commit,
            parents=(),
            classification="ROOT_COMMIT",
            triggered=False,
            current=candidate_state.current,
        )

    parent_sources = [GitSnapshot(repo, parent) for parent in parents]
    parent_states = [load_authority_state(source) for source in parent_sources]
    require_comparable_parent_authority(parent_states)

    representative = parent_sources[0]
    delta = _evaluate(representative, candidate_source)
    if not delta.valid:
        raise PPRemoteVerifyError(
            delta.classification,
            f"{commit}: " + json.dumps(delta.as_dict(), sort_keys=True),
        )

    parent_fingerprint = authority_fingerprint(parent_states[0])
    candidate_fingerprint = authority_fingerprint(candidate_state)
    if len(parents) > 1 and candidate_fingerprint == parent_fingerprint:
        return CommitVerification(
            commit=commit,
            parents=parents,
            classification="MERGE_INHERITED_PP_AUTHORITY",
            triggered=False,
            current=candidate_state.current,
        )

    if delta.triggered:
        _verify_transition_shape(
            repo,
            parent=parents[0],
            commit=commit,
            delta=delta,
        )
    elif candidate_fingerprint != parent_fingerprint:
        raise PPRemoteVerifyError(
            "UNCLASSIFIED_PP_AUTHORITY_CHANGE",
            f"{commit}: PP authority changed without a classified T2 delta.",
        )

    return CommitVerification(
        commit=commit,
        parents=parents,
        classification=delta.classification,
        triggered=delta.triggered,
        current=candidate_state.current,
    )


def verify_range(
    root: str | Path,
    *,
    base: str,
    head: str,
) -> tuple[CommitVerification, ...]:
    repo = repository_root(root)
    commits = introduced_commits(repo, base, head)
    return tuple(verify_commit(repo, commit) for commit in commits)


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Verify MPD-0011 PP transition semantics without mutating repository state."
    )
    parser.add_argument("--repository", default=".")
    sub = parser.add_subparsers(dest="command", required=True)

    commit = sub.add_parser("commit")
    commit.add_argument("--commit", required=True)

    range_parser = sub.add_parser("range")
    range_parser.add_argument("--base", required=True)
    range_parser.add_argument("--head", required=True)
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        if args.command == "commit":
            result = [verify_commit(args.repository, args.commit)]
        else:
            result = list(
                verify_range(
                    args.repository,
                    base=args.base,
                    head=args.head,
                )
            )
    except (PPRemoteVerifyError, PPTransitionDeltaError) as exc:
        print(
            json.dumps(
                {
                    "status": "FAIL",
                    "code": getattr(exc, "code", "PP_REMOTE_VERIFY_FAILED"),
                    "message": str(exc),
                },
                sort_keys=True,
            )
        )
        return 2

    print(
        json.dumps(
            {
                "status": "PASS",
                "verified_commit_count": len(result),
                "commits": [item.as_dict() for item in result],
            },
            sort_keys=True,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
