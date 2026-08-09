from __future__ import annotations

import shutil
import sys
from pathlib import Path

from .storage.local_state import ptsip_home


def doctor(path: str | Path = ".") -> dict[str, object]:
    target = Path(path).expanduser().resolve()
    return {
        "python": sys.version.split()[0],
        "python_ok": sys.version_info >= (3, 11),
        "git": shutil.which("git"),
        "target": str(target),
        "target_exists": target.exists(),
        "ptsip_home": str(ptsip_home()),
    }
