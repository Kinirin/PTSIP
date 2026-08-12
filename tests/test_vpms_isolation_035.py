from __future__ import annotations

import ast
import os
import subprocess
import sys
from pathlib import Path

import vpms


_REPO_ROOT = Path(__file__).resolve().parents[1]
_SRC_ROOT = _REPO_ROOT / "src"
_PTSIP_ROOT = _SRC_ROOT / "ptsip"


def _imports_vpms(path: Path) -> bool:
    tree = ast.parse(path.read_text(encoding="utf-8"), filename=str(path))
    for node in ast.walk(tree):
        if isinstance(node, ast.Import):
            if any(alias.name == "vpms" or alias.name.startswith("vpms.") for alias in node.names):
                return True
        elif isinstance(node, ast.ImportFrom):
            module = node.module or ""
            if module == "vpms" or module.startswith("vpms."):
                return True
    return False


def test_vpms_package_has_independent_import_root() -> None:
    assert vpms.__name__ == "vpms"
    assert vpms.__all__ == ()


def test_ptsip_source_has_no_dependency_on_vpms() -> None:
    offenders = [
        path.relative_to(_REPO_ROOT).as_posix()
        for path in sorted(_PTSIP_ROOT.rglob("*.py"))
        if _imports_vpms(path)
    ]
    assert offenders == []


def test_ptsip_import_succeeds_when_vpms_is_unavailable() -> None:
    script = r'''
import importlib.abc
import sys

class BlockVpms(importlib.abc.MetaPathFinder):
    def find_spec(self, fullname, path=None, target=None):
        if fullname == "vpms" or fullname.startswith("vpms."):
            raise ImportError("VPMS intentionally unavailable for isolation verification")
        return None

sys.meta_path.insert(0, BlockVpms())
import ptsip
assert ptsip.__name__ == "ptsip"
'''
    env = os.environ.copy()
    existing = env.get("PYTHONPATH")
    env["PYTHONPATH"] = os.pathsep.join(
        [str(_SRC_ROOT), *([existing] if existing else [])]
    )
    completed = subprocess.run(
        [sys.executable, "-c", script],
        cwd=_REPO_ROOT,
        env=env,
        text=True,
        capture_output=True,
        check=False,
    )
    assert completed.returncode == 0, completed.stderr


def test_importing_vpms_does_not_implicitly_import_ptsip() -> None:
    script = r'''
import sys
import vpms
assert "ptsip" not in sys.modules
'''
    env = os.environ.copy()
    existing = env.get("PYTHONPATH")
    env["PYTHONPATH"] = os.pathsep.join(
        [str(_SRC_ROOT), *([existing] if existing else [])]
    )
    completed = subprocess.run(
        [sys.executable, "-c", script],
        cwd=_REPO_ROOT,
        env=env,
        text=True,
        capture_output=True,
        check=False,
    )
    assert completed.returncode == 0, completed.stderr
