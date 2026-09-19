from __future__ import annotations

import argparse
import copy
import json
import subprocess
from dataclasses import dataclass
from pathlib import Path
from typing import Mapping, Protocol, Sequence

import yaml

from developer.automation.policy_loader import repository_root
from developer.automation.project_profile_registry import (
    PP_CONTRACT_REGISTRY,
    PUBLIC_PROFILE_CATALOG,
)
from ptsip.profile_identity import ProjectProfileVersion


class PPTransitionDeltaError(RuntimeError):
    def __init__(self, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code


class SnapshotSource(Protocol):
    label: str

    def read_bytes(self, path: str) -> bytes | None: ...

    def list_public_profile_resources(self) -> tuple[str, ...]: ...


@dataclass(frozen=True)
class PublicProfileEntry:
    profile_id: str
    resource: str
    contract: str


@dataclass(frozen=True)
class AuthorityState:
    label: str
    catalog_present: bool
    registry_present: bool
    current: str | None
    contract_schemas: Mapping[str, str | None]
    entries: Mapping[str, PublicProfileEntry]
    discovered_resources: tuple[str, ...]
    raw_profiles: Mapping[str, bytes]
    semantic_profiles: Mapping[str, str]
    declared_profile_versions: Mapping[str, str | None]
    current_schema_bytes: bytes | None


@dataclass(frozen=True)
class T2DeltaResult:
    classification: str
    triggered: bool
    valid: bool
    reasons: tuple[str, ...]
    base_current: str | None
    candidate_current: str | None
    expected_next: str | None
    candidate_already_reconciled: bool

    def as_dict(self) -> dict[str, object]:
        return {
            "classification": self.classification,
            "triggered": self.triggered,
            "valid": self.valid,
            "reasons": list(self.reasons),
            "base_current": self.base_current,
            "candidate_current": self.candidate_current,
            "expected_next": self.expected_next,
            "candidate_already_reconciled": self.candidate_already_reconciled,
        }


class GitSnapshot:
    def __init__(self, root: Path, revision: str) -> None:
        self.root = root
        self.revision = revision
        self.label = revision

    def _run(self, *args: str) -> subprocess.CompletedProcess[bytes]:
        return subprocess.run(
            ["git", *args],
            cwd=self.root,
            check=False,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )

    def read_bytes(self, path: str) -> bytes | None:
        result = self._run("show", f"{self.revision}:{path}")
        return result.stdout if result.returncode == 0 else None

    def list_public_profile_resources(self) -> tuple[str, ...]:
        result = self._run("ls-tree", "-r", "--name-only", self.revision, "--", "profiles")
        if result.returncode != 0:
            raise PPTransitionDeltaError(
                "GIT_TREE_READ_FAILED",
                result.stderr.decode("utf-8", errors="replace").strip(),
            )
        return _profile_resources(result.stdout.decode("utf-8").splitlines())


class GitIndexSnapshot:
    label = "STAGED_INDEX"

    def __init__(self, root: Path) -> None:
        self.root = root

    def _run(self, *args: str) -> subprocess.CompletedProcess[bytes]:
        return subprocess.run(
            ["git", *args],
            cwd=self.root,
            check=False,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )

    def read_bytes(self, path: str) -> bytes | None:
        result = self._run("show", f":{path}")
        return result.stdout if result.returncode == 0 else None

    def list_public_profile_resources(self) -> tuple[str, ...]:
        result = self._run("ls-files", "--cached", "--", "profiles")
        if result.returncode != 0:
            raise PPTransitionDeltaError(
                "GIT_INDEX_READ_FAILED",
                result.stderr.decode("utf-8", errors="replace").strip(),
            )
        return _profile_resources(result.stdout.decode("utf-8").splitlines())


def _profile_resources(paths: Sequence[str]) -> tuple[str, ...]:
    prefix = "profiles/"
    values = []
    for value in paths:
        normalized = value.strip().replace("\\", "/")
        if not normalized.startswith(prefix):
            continue
        relative = normalized[len(prefix) :]
        if "/" not in relative and relative.endswith(".ptsip.yaml"):
            values.append(relative)
    return tuple(sorted(set(values)))


def _yaml_mapping(raw: bytes | None, *, label: str) -> dict[str, object] | None:
    if raw is None:
        return None
    try:
        value = yaml.safe_load(raw.decode("utf-8"))
    except (UnicodeDecodeError, yaml.YAMLError) as exc:
        raise PPTransitionDeltaError(
            "AUTHORITY_INPUT_INVALID",
            f"{label} is not valid UTF-8 YAML: {exc}",
        ) from exc
    if not isinstance(value, dict):
        raise PPTransitionDeltaError(
            "AUTHORITY_INPUT_INVALID",
            f"{label} must be a mapping.",
        )
    return value


def _semantic_profile(raw: bytes, *, label: str) -> tuple[str, str | None]:
    payload = _yaml_mapping(raw, label=label)
    assert payload is not None
    ptsip = payload.get("ptsip")
    declared_version = ptsip.get("version") if isinstance(ptsip, Mapping) else None

    semantic = copy.deepcopy(payload)
    semantic_ptsip = semantic.get("ptsip")
    if isinstance(semantic_ptsip, dict):
        semantic_ptsip.pop("version", None)
    return (
        json.dumps(semantic, sort_keys=True, separators=(",", ":"), ensure_ascii=False),
        declared_version if isinstance(declared_version, str) else None,
    )


def load_authority_state(source: SnapshotSource) -> AuthorityState:
    catalog_payload = _yaml_mapping(
        source.read_bytes(PUBLIC_PROFILE_CATALOG),
        label=f"{source.label}:{PUBLIC_PROFILE_CATALOG}",
    )
    registry_payload = _yaml_mapping(
        source.read_bytes(PP_CONTRACT_REGISTRY),
        label=f"{source.label}:{PP_CONTRACT_REGISTRY}",
    )

    entries: dict[str, PublicProfileEntry] = {}
    if catalog_payload is not None:
        for item in catalog_payload.get("profiles", []):
            if not isinstance(item, Mapping):
                continue
            profile_id = item.get("id")
            resource = item.get("resource")
            contract = item.get("contract")
            if not all(isinstance(value, str) for value in (profile_id, resource, contract)):
                raise PPTransitionDeltaError(
                    "PUBLIC_PROFILE_CATALOG_INVALID",
                    "public profile catalog entries require string id/resource/contract.",
                )
            if profile_id in entries:
                raise PPTransitionDeltaError(
                    "PUBLIC_PROFILE_CATALOG_INVALID",
                    f"duplicate public profile id {profile_id!r}.",
                )
            entries[profile_id] = PublicProfileEntry(profile_id, resource, contract)

    current: str | None = None
    contract_schemas: dict[str, str | None] = {}
    if registry_payload is not None:
        current_value = registry_payload.get("current")
        if not isinstance(current_value, str):
            raise PPTransitionDeltaError(
                "PP_CONTRACT_REGISTRY_INVALID",
                "project-profile contract registry current must be a string.",
            )
        current = current_value
        for item in registry_payload.get("contracts", []):
            if not isinstance(item, Mapping):
                continue
            version = item.get("version")
            schema = item.get("schema")
            if isinstance(version, str):
                contract_schemas[version] = schema if isinstance(schema, str) else None

    discovered = source.list_public_profile_resources()
    raw_profiles: dict[str, bytes] = {}
    semantic_profiles: dict[str, str] = {}
    declared_versions: dict[str, str | None] = {}
    for resource in discovered:
        path = f"profiles/{resource}"
        raw = source.read_bytes(path)
        if raw is None:
            raise PPTransitionDeltaError(
                "PUBLIC_PROFILE_READ_FAILED",
                f"{source.label}:{path} is listed but cannot be read.",
            )
        raw_profiles[resource] = raw
        semantic, declared = _semantic_profile(raw, label=f"{source.label}:{path}")
        semantic_profiles[resource] = semantic
        declared_versions[resource] = declared

    current_schema_bytes: bytes | None = None
    if current is not None:
        schema_path = contract_schemas.get(current)
        if isinstance(schema_path, str):
            current_schema_bytes = source.read_bytes(schema_path)

    return AuthorityState(
        label=source.label,
        catalog_present=catalog_payload is not None,
        registry_present=registry_payload is not None,
        current=current,
        contract_schemas=contract_schemas,
        entries=entries,
        discovered_resources=discovered,
        raw_profiles=raw_profiles,
        semantic_profiles=semantic_profiles,
        declared_profile_versions=declared_versions,
        current_schema_bytes=current_schema_bytes,
    )


def _next_minor(value: str) -> str:
    version = ProjectProfileVersion.parse(value, require_canonical=True)
    return ProjectProfileVersion(version.major, version.minor + 1).canonical


def _is_baseline_materialization(base: AuthorityState, candidate: AuthorityState) -> bool:
    if base.catalog_present or base.registry_present:
        return False
    if not candidate.catalog_present or not candidate.registry_present or candidate.current is None:
        return False

    candidate_resources = tuple(sorted(entry.resource for entry in candidate.entries.values()))
    if candidate_resources != base.discovered_resources:
        return False
    if candidate.discovered_resources != base.discovered_resources:
        return False

    for resource in base.discovered_resources:
        if base.raw_profiles.get(resource) != candidate.raw_profiles.get(resource):
            return False
        if base.declared_profile_versions.get(resource) != candidate.current:
            return False

    if any(entry.contract != candidate.current for entry in candidate.entries.values()):
        return False

    schema_path = candidate.contract_schemas.get(candidate.current)
    if not isinstance(schema_path, str):
        return False
    candidate_schema = candidate.current_schema_bytes
    if candidate_schema is None:
        return False

    # The baseline registry may be introduced only when it points at the exact
    # pre-existing canonical schema bytes for the already-distributed identity.
    # The conventional path is deterministic from the candidate identity.
    return True


def evaluate_t2_authority_delta(
    base: AuthorityState,
    candidate: AuthorityState,
    *,
    base_schema_bytes_for_candidate_current: bytes | None = None,
) -> T2DeltaResult:
    if _is_baseline_materialization(base, candidate):
        if (
            base_schema_bytes_for_candidate_current is not None
            and base_schema_bytes_for_candidate_current != candidate.current_schema_bytes
        ):
            return T2DeltaResult(
                classification="INVALID_BASELINE_MATERIALIZATION",
                triggered=False,
                valid=False,
                reasons=("CURRENT_PP_CANONICAL_SCHEMA_CONTENT",),
                base_current=None,
                candidate_current=candidate.current,
                expected_next=None,
                candidate_already_reconciled=False,
            )
        return T2DeltaResult(
            classification="BASELINE_MATERIALIZATION_EXISTING_DISTRIBUTION",
            triggered=False,
            valid=True,
            reasons=(),
            base_current=None,
            candidate_current=candidate.current,
            expected_next=None,
            candidate_already_reconciled=False,
        )

    if not base.catalog_present or not base.registry_present:
        raise PPTransitionDeltaError(
            "T2_BASELINE_UNRESOLVED",
            "base snapshot does not contain the canonical Public Profile catalog and PP registry.",
        )
    if not candidate.catalog_present or not candidate.registry_present:
        return T2DeltaResult(
            classification="AUTHORITY_PLANE_REMOVED",
            triggered=True,
            valid=False,
            reasons=("PUBLIC_PROFILE_CATALOG_MEMBERSHIP",),
            base_current=base.current,
            candidate_current=candidate.current,
            expected_next=_next_minor(base.current) if base.current else None,
            candidate_already_reconciled=False,
        )
    if base.current is None or candidate.current is None:
        raise PPTransitionDeltaError(
            "PP_CURRENT_UNRESOLVED",
            "base and candidate snapshots must resolve current PP identity.",
        )

    reasons: list[str] = []
    base_ids = set(base.entries)
    candidate_ids = set(candidate.entries)
    if base_ids != candidate_ids:
        reasons.append("PUBLIC_PROFILE_CATALOG_MEMBERSHIP")

    for profile_id in sorted(base_ids & candidate_ids):
        left = base.entries[profile_id]
        right = candidate.entries[profile_id]
        if left.resource != right.resource:
            reasons.append("PUBLIC_PROFILE_CATALOG_RESOURCE_IDENTITY")
            continue
        if base.semantic_profiles.get(left.resource) != candidate.semantic_profiles.get(right.resource):
            reasons.append("REGISTERED_CANONICAL_PUBLIC_PROFILE_CONTENT")

    base_schema_path = base.contract_schemas.get(base.current)
    if not isinstance(base_schema_path, str):
        raise PPTransitionDeltaError(
            "CURRENT_PP_SCHEMA_UNRESOLVED",
            f"base current contract {base.current!r} has no canonical schema path.",
        )

    # current_schema_bytes on candidate follows candidate.current, so callers comparing
    # an already-reconciled candidate must supply the old-current path separately.
    candidate_old_current_schema = (
        candidate.current_schema_bytes
        if candidate.current == base.current
        else base_schema_bytes_for_candidate_current
    )
    if base.current_schema_bytes != candidate_old_current_schema:
        reasons.append("CURRENT_PP_CANONICAL_SCHEMA_CONTENT")

    unique_reasons = tuple(dict.fromkeys(reasons))
    if not unique_reasons:
        if candidate.current != base.current:
            return T2DeltaResult(
                classification="MANUAL_PP_TRANSITION_WITHOUT_AUTHORITY",
                triggered=False,
                valid=False,
                reasons=(),
                base_current=base.current,
                candidate_current=candidate.current,
                expected_next=None,
                candidate_already_reconciled=False,
            )
        return T2DeltaResult(
            classification="NO_T2_AUTHORITY_DELTA",
            triggered=False,
            valid=True,
            reasons=(),
            base_current=base.current,
            candidate_current=candidate.current,
            expected_next=None,
            candidate_already_reconciled=False,
        )

    expected_next = _next_minor(base.current)
    valid_identity = candidate.current in {base.current, expected_next}
    return T2DeltaResult(
        classification=(
            "T2_AUTHORITY_DELTA"
            if valid_identity
            else "PP_TRANSITION_IDENTITY_MISMATCH"
        ),
        triggered=True,
        valid=valid_identity,
        reasons=unique_reasons,
        base_current=base.current,
        candidate_current=candidate.current,
        expected_next=expected_next,
        candidate_already_reconciled=candidate.current == expected_next,
    )


def compare_git_snapshots(
    root: str | Path,
    *,
    base_revision: str,
    candidate_revision: str | None = None,
    staged: bool = False,
) -> T2DeltaResult:
    repo = repository_root(root)
    base_source = GitSnapshot(repo, base_revision)
    candidate_source: SnapshotSource = (
        GitIndexSnapshot(repo) if staged else GitSnapshot(repo, str(candidate_revision))
    )
    base = load_authority_state(base_source)
    candidate = load_authority_state(candidate_source)

    old_current_schema_bytes: bytes | None = None
    candidate_current = candidate.current
    if base.current is None and candidate_current is not None:
        path = candidate.contract_schemas.get(candidate_current)
        if isinstance(path, str):
            old_current_schema_bytes = base_source.read_bytes(path)
    elif base.current is not None:
        path = base.contract_schemas.get(base.current)
        if isinstance(path, str):
            old_current_schema_bytes = candidate_source.read_bytes(path)

    return evaluate_t2_authority_delta(
        base,
        candidate,
        base_schema_bytes_for_candidate_current=old_current_schema_bytes,
    )


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Resolve MPD-0011 T2 authority delta.")
    parser.add_argument("--repository", default=".")
    sub = parser.add_subparsers(dest="command", required=True)

    staged = sub.add_parser("staged")
    staged.add_argument("--base", default="HEAD")

    revisions = sub.add_parser("revisions")
    revisions.add_argument("--base", required=True)
    revisions.add_argument("--candidate", required=True)

    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        if args.command == "staged":
            result = compare_git_snapshots(
                args.repository,
                base_revision=args.base,
                staged=True,
            )
        else:
            result = compare_git_snapshots(
                args.repository,
                base_revision=args.base,
                candidate_revision=args.candidate,
            )
    except PPTransitionDeltaError as exc:
        print(json.dumps({"status": "UNRESOLVED", "code": exc.code, "message": str(exc)}, sort_keys=True))
        return 2

    print(json.dumps(result.as_dict(), sort_keys=True))
    return 0 if result.valid else 2


if __name__ == "__main__":
    raise SystemExit(main())
