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
    """Resolve a CLI/local profile selection to repository-relative identity."""

    root = Path(repository_root).expanduser().resolve()
    if explicit is None:
        return DEFAULT_PROFILE_PATH
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
