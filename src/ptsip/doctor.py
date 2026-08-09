from __future__ import annotations

import shutil
import sys
from pathlib import Path

from .storage.local_state import ptsip_home


def doctor(path: str | Path = ".") -> dict[str, object]:
    target = Path(path).expanduser().resolve()
    current_minor = (sys.version_info.major, sys.version_info.minor)
    metadata_supported = current_minor >= (3, 11)
    ci_verified = (3, 11) <= current_minor <= (3, 14)
    github_cli = shutil.which("gh")
    return {
        "python": sys.version.split()[0],
        "python_ok": metadata_supported,
        "python_supported_by_metadata": metadata_supported,
        "python_ci_verified": ci_verified,
        "python_ci_verified_range": "3.11-3.14",
        "python_support_note": (
            "This interpreter is within the current CI-verified range."
            if ci_verified
            else "Package metadata allows this interpreter, but this minor version is outside the current CI matrix."
        ),
        "git": shutil.which("git"),
        "github_cli": github_cli,
        "github_issue_publish_available": github_cli is not None,
        "target": str(target),
        "target_exists": target.exists(),
        "ptsip_home": str(ptsip_home()),
    }
