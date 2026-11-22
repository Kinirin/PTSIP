"""Translate PEP 517 hooks to setuptools metadata and the Go product builder."""

import json
from pathlib import Path
import subprocess
import sys
import sysconfig
from tempfile import TemporaryDirectory

from setuptools import build_meta as setuptools


ROOT = Path(__file__).resolve().parents[2]


def _builder(operation, *args):
    command = ["go", "-C", str(ROOT / "src/build"), "run", "-mod=readonly", ".",
               operation, "-root", str(ROOT), *args]
    return subprocess.check_output(command, cwd=ROOT, text=True, stderr=sys.stderr).strip()


def build_wheel(wheel_directory, config_settings=None, metadata_directory=None):
    _builder("revision")
    with TemporaryDirectory(prefix="ptsip-wheel-base-") as base:
        filename = setuptools.build_wheel(base, config_settings, metadata_directory)
        return _builder("wheel", "-base", str(Path(base) / filename),
                        "-output", str(Path(wheel_directory).resolve()),
                        "-platform-tag", sysconfig.get_platform().replace("-", "_").replace(".", "_"))


def build_sdist(sdist_directory, config_settings=None):
    descriptor = ROOT / "ptsip-build-source.json"
    revision = _builder("revision")
    if descriptor.exists():
        data = json.loads(descriptor.read_bytes())
        if data != {"schema_version": "ptsip-build-source/v1", "source_sha": revision}:
            raise ValueError("source descriptor differs from the selected Git revision")
        return setuptools.build_sdist(sdist_directory, config_settings)

    with descriptor.open("x", encoding="ascii") as file:
        json.dump({"schema_version": "ptsip-build-source/v1", "source_sha": revision}, file)
        file.write("\n")
    try:
        return setuptools.build_sdist(sdist_directory, config_settings)
    finally:
        descriptor.unlink()


get_requires_for_build_wheel = setuptools.get_requires_for_build_wheel
get_requires_for_build_sdist = setuptools.get_requires_for_build_sdist
prepare_metadata_for_build_wheel = setuptools.prepare_metadata_for_build_wheel
