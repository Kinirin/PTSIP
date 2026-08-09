from __future__ import annotations

import hashlib
import os
import sys
from pathlib import Path


def ptsip_home() -> Path:
    override = os.environ.get("PTSIP_HOME")
    if override:
        return Path(override).expanduser().resolve()
    if sys.platform == "win32":
        base = Path(os.environ.get("LOCALAPPDATA", Path.home() / "AppData" / "Local"))
        return base / "PTSIP"
    if sys.platform == "darwin":
        return Path.home() / "Library" / "Application Support" / "PTSIP"
    base = Path(os.environ.get("XDG_STATE_HOME", Path.home() / ".local" / "state"))
    return base / "ptsip"


def repository_fingerprint(repository_root: str | Path) -> str:
    normalized = str(Path(repository_root).resolve()).casefold().encode("utf-8")
    return hashlib.sha256(normalized).hexdigest()[:20]


def pilot_directory(repository_root: str | Path) -> Path:
    return ptsip_home() / "pilots" / repository_fingerprint(repository_root)
