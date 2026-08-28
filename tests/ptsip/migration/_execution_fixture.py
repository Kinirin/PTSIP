from __future__ import annotations

import hashlib
import subprocess
from pathlib import Path

import yaml

from ptsip.migration import (
    AsynchronousWorkTarget,
    EvidenceCorrelation,
    MigrationAnalysis,
    RemovalMigrationElement,
    RequiredWorkElement,
    SourceMigrationCompletion,
    TargetCompatibility,
)
from ptsip.repository.profile_transition import DraftVersion, ProfileGenerationIdentity
from ptsip.repository.snapshot import capture_snapshot
from ptsip.source_compat.model import SourceGenerationBinding


SPEC_SOURCE = "https://github.com/Kinirin/PTSIP"


def git(root: Path, *args: str) -> None:
    subprocess.run(
        ["git", "-C", str(root), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )


def init_git_repository(root: Path) -> Path:
    root.mkdir(parents=True, exist_ok=False)
    git(root, "init")
    git(root, "config", "user.name", "PTSIP Migration Fixture")
    git(root, "config", "user.email", "ptsip-migration-fixture@example.invalid")
    return root


def commit_all(root: Path, message: str = "fixture baseline") -> None:
    git(root, "add", ".")
    git(root, "commit", "-m", message)


def write_yaml(path: Path, payload: dict[str, object]) -> Path:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8", newline="\n")
    return path


def sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def profile_payload(version: str, revision: str) -> dict[str, object]:
    return {
        "ptsip": {
            "version": version,
            "specification": {
                "source": SPEC_SOURCE,
                "revision": revision,
            },
        },
        "responsibility_map": {"mode": "explicit"},
    }


def binding(
    path: str,
    version: str,
    revision: str,
    content_sha256: str,
    *,
    temporary: bool,
) -> SourceGenerationBinding:
    return SourceGenerationBinding(
        profile_path=path,
        declared_version=version,
        specification_revision=revision,
        specification_source=SPEC_SOURCE,
        content_sha256=content_sha256,
        temporary=temporary,
    )


def identity(source: SourceGenerationBinding) -> ProfileGenerationIdentity:
    parsed = DraftVersion.from_draft_label(source.declared_version)
    assert parsed is not None
    return ProfileGenerationIdentity(
        path=source.profile_path,
        version=parsed,
        declared_version=source.declared_version,
        specification_revision=source.specification_revision,
        specification_source=source.specification_source,
        content_sha256=source.content_sha256,
        temporary=source.temporary,
    )


def required(
    obligation_id: str = "required:src/a.py",
    *,
    path: str = "src/a.py",
    declaration_id: str = "core",
    resolved: bool = False,
    target_status: TargetCompatibility = TargetCompatibility.NOT_EVALUATED,
) -> RequiredWorkElement:
    return RequiredWorkElement(
        id=obligation_id,
        path=path,
        source_declaration_id=declaration_id,
        source_classification="PRODUCT",
        selector="src/**",
        evidence=EvidenceCorrelation((), (), ()),
        target_status=target_status,
        resolved=resolved,
    )


def removal(
    removal_id: str = "removal:retired.py",
    *,
    declaration_id: str = "retired",
) -> RemovalMigrationElement:
    return RemovalMigrationElement(
        id=removal_id,
        source_declaration_id=declaration_id,
        source_classification="PRODUCT",
        selector="retired/**",
        rationale="fixture source declaration is retired",
    )


def async_target(target_id: str = "async:docs/readme.md", *, path: str = "docs/readme.md") -> AsynchronousWorkTarget:
    return AsynchronousWorkTarget(target_id, path, EvidenceCorrelation((), (), ()))


def analysis(
    source: SourceGenerationBinding,
    root: Path,
    *,
    required_items: tuple[RequiredWorkElement, ...] = (),
    removal_items: tuple[RemovalMigrationElement, ...] = (),
    async_items: tuple[AsynchronousWorkTarget, ...] = (),
) -> MigrationAnalysis:
    snapshot = capture_snapshot(root)
    assert not snapshot.observation_errors
    resolved = sum(item.resolved for item in required_items)
    return MigrationAnalysis(
        source_generation=source,
        repository_head=snapshot.head,
        repository_status_fingerprint=snapshot.status_fingerprint,
        repository_content_fingerprint=snapshot.tracked_content_fingerprint,
        required=required_items,
        removals=removal_items,
        async_targets=async_items,
        ambiguous=(),
        lifecycle_findings=(),
        architecture_findings=(),
        issues=(),
        completion=SourceMigrationCompletion(
            required_total=len(required_items),
            required_resolved=resolved,
            required_unresolved=len(required_items) - resolved,
            removal_count=len(removal_items),
            async_count=len(async_items),
        ),
    )
