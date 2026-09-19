from __future__ import annotations

from pathlib import Path
from shutil import copy2, copytree, rmtree

from setuptools import setup
from setuptools.command.build_py import build_py as _build_py


ROOT = Path(__file__).resolve().parent
CANONICAL_SUPPORT_POLICY = ROOT / "docs" / "Support_policy" / "policy"
CANONICAL_PUBLIC_PROFILES = ROOT / "profiles"


def _project_support_policy(build_lib: Path) -> None:
    source = CANONICAL_SUPPORT_POLICY
    required = (source / "index.yaml", source / "schemas", source / "registries")
    if not all(path.exists() for path in required):
        missing = [str(path) for path in required if not path.exists()]
        raise RuntimeError(f"canonical Support Policy source is incomplete: {missing}")

    target = build_lib / "ptsip" / "support"
    if target.exists():
        rmtree(target)

    policy_target = target / "policy"
    policy_target.mkdir(parents=True, exist_ok=True)
    copy2(source / "index.yaml", policy_target / "index.yaml")

    sfp_files = sorted(source.glob("SFP-*.yaml"))
    if not sfp_files:
        raise RuntimeError("canonical Support Policy source contains no SFP records")
    for path in sfp_files:
        copy2(path, policy_target / path.name)

    copytree(source / "schemas", target / "schemas")
    copytree(source / "registries", target / "registries")


def _project_public_profiles(build_lib: Path) -> None:
    source = CANONICAL_PUBLIC_PROFILES
    public_profiles = sorted(source.glob("*.ptsip.yaml"))
    if not public_profiles:
        raise RuntimeError("canonical public Profile source contains no *.ptsip.yaml files")

    target = build_lib / "ptsip" / "profiles"
    if target.exists():
        rmtree(target)
    target.mkdir(parents=True, exist_ok=True)

    for path in public_profiles:
        copy2(path, target / path.name)


class build_py(_build_py):
    """Project canonical repository assets into the built ptsip package."""

    def run(self) -> None:
        super().run()
        build_lib = Path(self.build_lib)
        _project_support_policy(build_lib)
        _project_public_profiles(build_lib)


setup(cmdclass={"build_py": build_py})
