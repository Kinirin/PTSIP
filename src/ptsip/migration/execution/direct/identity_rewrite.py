from __future__ import annotations

import codecs
import copy
import hashlib
import os
import re
import tempfile
from pathlib import Path
from typing import Protocol

import yaml

from ptsip.repository.profile_path import profile_path_on_disk
from ptsip.repository.snapshot import capture_snapshot, compare_snapshots
from ptsip.validation.profile import validate_profile
from ptsip.migration.execution.direct.identity_plan import (
    IdentityRewriteAuthorization,
    IdentityRewriteError,
    IdentityRewritePlan,
    IdentityRewriteResult,
)

class AuthorityRevisionStore(Protocol):
    def ensure_head(self) -> str: ...

def _atomic_write(path: Path, raw: bytes) -> None:
    handle = tempfile.NamedTemporaryFile(
        mode="wb",
        dir=path.parent,
        prefix=f".{path.name}.ptsip-identity-",
        suffix=".tmp",
        delete=False,
    )
    temporary = Path(handle.name)
    try:
        with handle:
            handle.write(raw)
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(temporary, path)
    finally:
        if temporary.exists():
            temporary.unlink()

def _rewrite_version_text(
    text: str,
    *,
    source_version: str,
    target_version: str,
    specification_family: str,
) -> str:
    lines = text.splitlines(keepends=True)
    ptsip_index: int | None = None
    for index, line in enumerate(lines):
        body = line.rstrip("\r\n")
        if re.fullmatch(r"ptsip\s*:\s*(?:#.*)?", body):
            ptsip_index = index
            break
    if ptsip_index is None:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_SERIALIZATION_UNSUPPORTED",
            "Unable to locate the top-level ptsip mapping without reserializing the profile.",
        )

    block: list[tuple[int, int, str, str]] = []
    for index in range(ptsip_index + 1, len(lines)):
        line = lines[index]
        body = line.rstrip("\r\n")
        eol = line[len(body):]
        stripped = body.lstrip(" ")
        if not stripped or stripped.startswith("#"):
            continue
        indent = len(body) - len(stripped)
        if indent == 0:
            break
        block.append((index, indent, body, eol))

    if not block:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_SERIALIZATION_UNSUPPORTED",
            "The ptsip mapping has no serializable child fields.",
        )
    child_indent = min(item[1] for item in block)
    version_rows = [item for item in block if item[1] == child_indent and re.match(r"\s*version\s*:", item[2])]
    if len(version_rows) != 1:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_SERIALIZATION_UNSUPPORTED",
            "Expected exactly one direct ptsip.version scalar line.",
        )

    index, _indent, body, eol = version_rows[0]
    match = re.fullmatch(
        r"(?P<prefix>\s*version\s*:\s*)(?:(?P<dq>\"[^\"]*\")|(?P<sq>'[^']*')|(?P<plain>[^#\s]+))(?P<suffix>\s*(?:#.*)?)",
        body,
    )
    if match is None:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_SERIALIZATION_UNSUPPORTED",
            "ptsip.version must be a single-line scalar for in-place identity rewrite.",
        )
    if match.group("dq") is not None:
        current = match.group("dq")[1:-1]
        replacement = f'"{target_version}"'
    elif match.group("sq") is not None:
        current = match.group("sq")[1:-1]
        replacement = f"'{target_version}'"
    else:
        current = match.group("plain")
        replacement = target_version
    if current != source_version:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_SOURCE_MISMATCH",
            f"Serialized ptsip.version {current!r} does not match planned source {source_version!r}.",
        )

    lines[index] = f"{match.group('prefix')}{replacement}{match.group('suffix')}{eol}"

    specification_rows = [
        item
        for item in block
        if item[1] == child_indent
        and re.fullmatch(r"\s*specification\s*:\s*(?:#.*)?", item[2])
    ]
    if len(specification_rows) != 1:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_SERIALIZATION_UNSUPPORTED",
            "Expected exactly one direct ptsip.specification mapping line.",
        )
    specification_index, _indent, _body, specification_eol = specification_rows[0]
    specification_end = min(
        (
            item[0]
            for item in block
            if item[0] > specification_index and item[1] == child_indent
        ),
        default=len(lines),
    )
    nested_rows = [
        item
        for item in block
        if specification_index < item[0] < specification_end and item[1] > child_indent
    ]
    if not nested_rows:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_SERIALIZATION_UNSUPPORTED",
            "The ptsip.specification mapping has no serializable child fields.",
        )
    nested_indent = min(item[1] for item in nested_rows)
    family_rows = [
        item
        for item in nested_rows
        if item[1] == nested_indent and re.match(r"\s*family\s*:", item[2])
    ]
    if family_rows:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_SOURCE_MISMATCH",
            "Historical identity rewrite source must not already declare ptsip.specification.family.",
        )
    line_ending = specification_eol or next(
        (item[3] for item in nested_rows if item[3]),
        "\n",
    )
    lines.insert(
        specification_index + 1,
        f"{' ' * nested_indent}family: {specification_family}{line_ending}",
    )
    return "".join(lines)

def _rewrite_version_bytes(
    raw: bytes,
    *,
    source_version: str,
    target_version: str,
    specification_family: str,
) -> bytes:
    has_bom = raw.startswith(codecs.BOM_UTF8)
    try:
        text = raw.decode("utf-8-sig")
    except UnicodeError as exc:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_ENCODING_UNSUPPORTED",
            f"Project Profile must be UTF-8 for identity rewrite: {exc}",
        ) from exc
    rewritten = _rewrite_version_text(
        text,
        source_version=source_version,
        target_version=target_version,
        specification_family=specification_family,
    ).encode("utf-8")
    return codecs.BOM_UTF8 + rewritten if has_bom else rewritten

def _assert_identity_only_semantics(
    before_raw: bytes,
    after_raw: bytes,
    *,
    source_version: str,
    target_version: str,
    specification_family: str,
) -> None:
    try:
        before = yaml.safe_load(before_raw.decode("utf-8-sig"))
        after = yaml.safe_load(after_raw.decode("utf-8-sig"))
    except (UnicodeError, yaml.YAMLError) as exc:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_SEMANTIC_CHECK_FAILED",
            f"Unable to verify identity-only semantic preservation: {exc}",
        ) from exc
    if not isinstance(before, dict) or not isinstance(after, dict):
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_SEMANTIC_CHECK_FAILED",
            "Identity-only semantic verification requires mapping profiles.",
        )
    expected = copy.deepcopy(before)
    ptsip = expected.get("ptsip")
    if not isinstance(ptsip, dict) or ptsip.get("version") != source_version:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_SOURCE_MISMATCH",
            "Parsed source identity differs from the authorized identity rewrite plan.",
        )
    ptsip["version"] = target_version
    specification = ptsip.get("specification")
    if not isinstance(specification, dict) or "family" in specification:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_SOURCE_MISMATCH",
            "Historical identity rewrite source must have a mapping specification binding without family.",
        )
    specification["family"] = specification_family
    if after != expected:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_SEMANTIC_DRIFT",
            "Identity-only rewrite changed Project Profile semantics beyond explicit contract identity binding.",
        )

def execute_identity_rewrite(
    repository_root: str | Path,
    plan: IdentityRewritePlan,
    authorization: IdentityRewriteAuthorization,
    *,
    authority_store: AuthorityRevisionStore | None = None,
) -> IdentityRewriteResult:
    if authorization.plan_digest != plan.deterministic_digest:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_AUTHORIZATION_STALE",
            "Identity rewrite authorization belongs to a different plan.",
        )
    if authority_store is not None and authority_store.ensure_head() != authorization.authority_revision:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_AUTHORITY_STALE",
            "Owner authority revision changed after identity rewrite authorization.",
        )

    root = Path(repository_root).expanduser().resolve()
    comparison = compare_snapshots(plan.repository_snapshot, capture_snapshot(root))
    if comparison.reasons:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_STALE_REPOSITORY",
            "Repository changed after identity rewrite planning: " + "; ".join(comparison.reasons),
        )

    path = profile_path_on_disk(root, plan.source_path)
    try:
        before_raw = path.read_bytes()
    except OSError as exc:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_SOURCE_UNAVAILABLE",
            f"Unable to read authorized source profile: {exc}",
        ) from exc
    before_sha = hashlib.sha256(before_raw).hexdigest()
    if before_sha != plan.source_sha256:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_SOURCE_STALE",
            "Source profile content changed after identity rewrite planning.",
        )

    after_raw = _rewrite_version_bytes(
        before_raw,
        source_version=plan.source_declared_version,
        target_version=plan.target_contract,
        specification_family=plan.specification_family,
    )
    _assert_identity_only_semantics(
        before_raw,
        after_raw,
        source_version=plan.source_declared_version,
        target_version=plan.target_contract,
        specification_family=plan.specification_family,
    )

    _atomic_write(path, after_raw)
    validation = validate_profile(root, path)
    if not validation.valid:
        try:
            _atomic_write(path, before_raw)
        except OSError as exc:
            raise IdentityRewriteError(
                "PP_IDENTITY_REWRITE_ROLLBACK_FAILED",
                "Post-write validation failed and original profile restoration also failed: " + str(exc),
            ) from exc
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_POST_VALIDATION_FAILED",
            "Post-write Project Profile validation failed; original bytes were restored: "
            + "; ".join(validation.errors),
        )

    final_raw = path.read_bytes()
    if final_raw != after_raw:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_POST_WRITE_MISMATCH",
            "Profile bytes changed unexpectedly after identity rewrite validation.",
        )
    return IdentityRewriteResult(
        source_path=plan.source_path,
        before_sha256=before_sha,
        after_sha256=hashlib.sha256(final_raw).hexdigest(),
        source_declared_version=plan.source_declared_version,
        target_contract=plan.target_contract,
        specification_family=plan.specification_family,
        specification_revision=plan.specification_revision,
        validation_warnings=tuple(validation.warnings),
    )

