"""PEP 517 source builds of the complete, single-project PTSIP distribution.

Release host wheels use admitted native engines from the developer Go controller.
An sdist rebuild compiles the same independently versioned Rust crates locally.
Neither origin metadata nor installation grants runtime authority.
"""
from __future__ import annotations

import base64
import csv
import hashlib
import io
import json
import os
from pathlib import Path
import platform
import re
import subprocess
import sys
import sysconfig
import tempfile
import tomllib
import zipfile

from setuptools import build_meta as _setuptools

ROOT = Path(__file__).resolve().parents[5]


def _source() -> str:
    descriptor = ROOT / "ptsip-build-source.json"
    if descriptor.is_file():
        source = json.loads(descriptor.read_text())["source_sha"]
    else:
        if subprocess.check_output(["git", "status", "--porcelain", "--untracked-files=no"], cwd=ROOT).strip():
            raise RuntimeError("PTSIP source builds require an exact clean source or the sdist source descriptor")
        source = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
    if re.fullmatch(r"[0-9a-f]{40}", source) is None:
        raise RuntimeError("invalid PTSIP source revision")
    return source


def build_sdist(sdist_directory, config_settings=None):
    descriptor = ROOT / "ptsip-build-source.json"
    existed = descriptor.exists()
    source = _source()
    if not existed:
        descriptor.write_text(json.dumps({"schema_version": "ptsip-build-source/v1", "source_sha": source}) + "\n")
    try:
        return _setuptools.build_sdist(sdist_directory, config_settings)
    finally:
        if not existed:
            descriptor.unlink()


def build_wheel(wheel_directory, config_settings=None, metadata_directory=None):
    source = _source()
    project = tomllib.loads((ROOT / "pyproject.toml").read_text(encoding="utf-8"))
    version = project["project"]["version"]
    if project["project"]["name"] != "PTSIP":
        raise RuntimeError("the native backend builds PTSIP only")
    hosts = {
        ("win32", "amd64"): ("win_amd64", "windows-x64", "x86_64-pc-windows-msvc", ".exe"),
        ("win32", "x86_64"): ("win_amd64", "windows-x64", "x86_64-pc-windows-msvc", ".exe"),
        ("linux", "x86_64"): ("linux_x86_64", "linux-x64", "x86_64-unknown-linux-gnu", ""),
        ("darwin", "arm64"): (sysconfig.get_platform().replace("-", "_").replace(".", "_"), "macos-arm64", "aarch64-apple-darwin", ""),
    }
    try:
        tag, host, target, extension = hosts[(sys.platform, platform.machine().lower())]
    except KeyError as exc:
        raise RuntimeError("unsupported PTSIP native build host") from exc
    index = json.loads((ROOT / "src/ptsip/engines/index.json").read_text())
    entries = index["official_engine_entries"]
    versions = project["tool"]["ptsip"]["release"]["engines"]
    if len(entries) != len(versions) or {e["kind"].lower(): e["version"] for e in entries} != versions:
        raise RuntimeError("incomplete or stale PTSIP engine inventory")
    with tempfile.TemporaryDirectory(prefix="ptsip-source-wheel-") as temporary:
        stage = Path(temporary)
        pure = _setuptools.build_wheel(str(stage), config_settings, metadata_directory)
        with zipfile.ZipFile(stage / pure) as archive:
            files = {name: archive.read(name) for name in archive.namelist()}
        di = f"ptsip-{version}.dist-info/"
        files.pop(di + "RECORD", None)
        origins = {}
        executables = set()
        for entry in entries:
            kind = entry["kind"].lower()
            manifest = ROOT / f"src/ptsip/engines/{kind}/Cargo.toml"
            crate = tomllib.loads(manifest.read_text(encoding="utf-8"))
            if crate["package"]["version"] != entry["version"]:
                raise RuntimeError("Rust crate version differs from engine registry")
            build = stage / "cargo" / kind
            environment = dict(os.environ, CARGO_TARGET_DIR=str(build))
            subprocess.run(["cargo", "build", "--release", "--locked", "--manifest-path", str(manifest)], cwd=ROOT, env=environment, check=True)
            command = "ptsip-engine-" + kind + extension
            binary = (build / "release" / command).read_bytes()
            name = f"ptsip-{version}.data/scripts/{command}"
            files[name] = binary
            executables.add(name)
            origins[kind] = {
                "schema_version": "ptsip-independent-wheel-origin/v1",
                "source_sha": source,
                "engine_id": entry["engine_id"],
                "engine_version": entry["version"],
                "host_target": target,
                "entrypoint": command,
                "binary_sha256": hashlib.sha256(binary).hexdigest(),
                "execution_authorized": False,
            }
        bundle = {"schema_version": "ptsip-bundled-engine-origins/v1", "package_version": version, "platform": host, "engines": origins, "execution_authorized": False}
        files[di + "ptsip-engines.json"] = (json.dumps(bundle, sort_keys=True, indent=2) + "\n").encode()
        files[di + "WHEEL"] = f"Wheel-Version: 1.0\nGenerator: ptsip-source-build/1\nRoot-Is-Purelib: false\nTag: py3-none-{tag}\n".encode()
        record = io.StringIO(newline="")
        writer = csv.writer(record)
        for name, data in sorted(files.items()):
            digest = base64.urlsafe_b64encode(hashlib.sha256(data).digest()).decode().rstrip("=")
            writer.writerow((name, "sha256=" + digest, str(len(data))))
        writer.writerow((di + "RECORD", "", ""))
        files[di + "RECORD"] = record.getvalue().encode()
        output = Path(wheel_directory) / f"ptsip-{version}-py3-none-{tag}.whl"
        with zipfile.ZipFile(output, "w", compression=zipfile.ZIP_DEFLATED) as archive:
            for name, data in sorted(files.items()):
                info = zipfile.ZipInfo(name, (2020, 1, 1, 0, 0, 0))
                info.create_system = 3
                info.external_attr = (0o100755 if name in executables else 0o100644) << 16
                info.compress_type = zipfile.ZIP_DEFLATED
                archive.writestr(info, data)
        return output.name


get_requires_for_build_wheel = _setuptools.get_requires_for_build_wheel
get_requires_for_build_sdist = _setuptools.get_requires_for_build_sdist
prepare_metadata_for_build_wheel = _setuptools.prepare_metadata_for_build_wheel
