from __future__ import annotations

import hashlib
import re
from pathlib import Path


DEFAULT_PROFILE_PATH = "ptsip.yaml"


def normalize_profile_path(value: object | None) -> str:
    """Return a canonical repository-relative POSIX profile path.

    Control-plane records must never carry absolute paths or parent traversal.
    ``None`` preserves the historical root-profile default.
    """

    if value is None:
        return DEFAULT_PROFILE_PATH
    text = str(value).replace("\\", "/").strip()
    while text.startswith("./"):
        text = text[2:]
    if not text:
        raise ValueError("profile_path must not be empty")
    if text.startswith("/") or re.match(r"^[A-Za-z]:/", text):
        raise ValueError("profile_path must be repository-relative")

    parts: list[str] = []
    for part in text.split("/"):
        if part in {"", "."}:
            continue
        if part == "..":
            raise ValueError("profile_path must not escape the repository")
        parts.append(part)
    if not parts:
        raise ValueError("profile_path must name a file inside the repository")
    return "/".join(parts)


def selected_profile_path(
    repository_root: str | Path,
    explicit: str | Path | None = None,
) -> str:
    """Resolve a CLI/local profile selection to repository-relative identity.

    Explicit paths are preserved. For implicit selection, an existing canonical
    local-profile catalog wins, an existing historical root profile remains
    compatible, and a repository with no declaration targets the canonical
    .ptsip profile location.
    """

    root = Path(repository_root).expanduser().resolve()
    if explicit is None:
        # Import lazily so the generic path-normalization helpers remain usable
        # without creating a module initialization cycle.
        from ..local_profile_catalog import (
            canonical_new_profile_path,
            load_local_profile_selection,
        )

        local = load_local_profile_selection(root)
        if local is not None:
            return normalize_profile_path(local.path.relative_to(root).as_posix())
        legacy = root / DEFAULT_PROFILE_PATH
        if legacy.is_file():
            return DEFAULT_PROFILE_PATH
        return normalize_profile_path(
            canonical_new_profile_path(root).relative_to(root).as_posix()
        )
    raw = Path(explicit).expanduser()
    candidate = raw.resolve() if raw.is_absolute() else (root / raw).resolve()
    try:
        relative = candidate.relative_to(root)
    except ValueError as exc:
        raise ValueError("Selected PTSIP profile must be inside the repository") from exc
    return normalize_profile_path(relative.as_posix())


def profile_path_on_disk(repository_root: str | Path, profile_path: object | None) -> Path:
    root = Path(repository_root).expanduser().resolve()
    relative = normalize_profile_path(profile_path)
    candidate = (root / Path(relative)).resolve()
    try:
        candidate.relative_to(root)
    except ValueError as exc:  # defensive after lexical normalization
        raise ValueError("Selected PTSIP profile must be inside the repository") from exc
    return candidate


def bind_decision_id(clarification_id: str, profile_path: object | None) -> str:
    """Bind non-root profile identity without changing historical root IDs."""

    normalized = normalize_profile_path(profile_path)
    if normalized == DEFAULT_PROFILE_PATH:
        return clarification_id
    digest = hashlib.sha256(normalized.encode("utf-8")).hexdigest()[:12]
    return f"{clarification_id}-p{digest}"
