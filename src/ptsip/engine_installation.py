"""Read-only consumer diagnostics for the seven independently packaged engines.

Installation is delegated to the authorized package-manager transaction.
Presence or self-reported wheel provenance never grants runtime authority.
"""
from __future__ import annotations

import hashlib
import json
import os
import platform
import re
import sys
import sysconfig
from importlib import metadata, resources
from pathlib import Path
from typing import Any

HOST_TARGETS = {
    ("win32", "amd64"): "x86_64-pc-windows-msvc",
    ("win32", "x86_64"): "x86_64-pc-windows-msvc",
    ("linux", "x86_64"): "x86_64-unknown-linux-gnu",
    ("darwin", "arm64"): "aarch64-apple-darwin",
    ("darwin", "aarch64"): "aarch64-apple-darwin",
}


def _load(relative: str) -> dict[str, Any]:
    resource = resources.files("ptsip").joinpath("engines", *relative.split("/"))
    return json.loads(resource.read_text(encoding="utf-8"))


def _sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def _bundle(items: list[dict[str, Any]]) -> tuple[Any, dict[str, Any]] | None:
    """A broken installed bundle must not fall back to another distribution."""
    try:
        distribution = metadata.distribution("PTSIP")
    except metadata.PackageNotFoundError:
        return None
    text = distribution.read_text("ptsip-engines.json")
    if text is None:
        return None
    try:
        bundle = json.loads(text)
        origins = bundle["engines"]
        if (
            bundle["schema_version"] != "ptsip-bundled-engine-origins/v1"
            or bundle["package_version"] != distribution.version
            or bundle["execution_authorized"] is not False
            or set(origins) != {entry["kind"].lower() for entry in items}
            or len(origins) != len(items)
        ):
            raise ValueError("incomplete or incompatible PTSIP engine bundle")
        for entry in items:
            origin = origins[entry["kind"].lower()]
            if (
                origin["schema_version"] != "ptsip-independent-wheel-origin/v1"
                or origin["engine_id"] != entry["engine_id"]
                or origin["engine_version"] != entry["version"]
                or origin["execution_authorized"] is not False
                or re.fullmatch(r"[0-9a-f]{40}", origin["source_sha"]) is None
            ):
                raise ValueError("PTSIP bundled engine identity differs")
        return distribution, bundle
    except (KeyError, TypeError, json.JSONDecodeError) as exc:
        raise ValueError("PTSIP bundled engine metadata is invalid") from exc


def inspect(
    *,
    index: dict[str, Any] | None = None,
    plan: dict[str, Any] | None = None,
    bindings: dict[str, dict[str, Any]] | None = None,
    scripts: Path | None = None,
    host_target: str | None = None,
) -> dict[str, Any]:
    index = index if index is not None else _load("index.json")
    plan = plan if plan is not None else _load("contracts/installation-plan.json")
    items = index["official_engine_entries"]
    packages = {p["engine_id"]: p for p in plan["packages"]}
    if (
        len(items) != 7 or len(packages) != 7
        or len({i["engine_id"] for i in items}) != 7
        or set(packages) != {i["engine_id"] for i in items}
        or set(plan["default_available_engine_ids"]) != set(packages)
    ):
        raise ValueError("official engine index or installation registry incomplete/ambiguous")
    host_target = host_target or HOST_TARGETS.get(
        (sys.platform, platform.machine().lower())
    )
    scripts = scripts or Path(sysconfig.get_path("scripts"))
    bundled = _bundle(items)
    entries: list[dict[str, Any]] = []
    for entry in items:
        kind, engine_id, version = entry["kind"], entry["engine_id"], entry["version"]
        pkg = packages[engine_id]
        package_name = "ptsip-engine-" + kind.lower()
        if (
            pkg["distribution"] != "INDEPENDENT_VERSIONED_NATIVE_WHEEL"
            or pkg["package_distribution"] != package_name
            or pkg["installation"] != "AUTO_INSTALL_AS_PTSIP_PACKAGE_DEPENDENCY"
            or pkg["version"] != version
            or pkg["permission_from_availability"] is not False
        ):
            raise ValueError("outdated or incompatible official engine installation contract")
        b = bindings[kind] if bindings is not None else _load(
            f"{kind.lower()}/execution-bindings.json"
        )
        candidates = b["bindings"]
        if len(candidates) != 1:
            raise ValueError("ambiguous registered engine executable")
        binding = candidates[0]
        if binding["entrypoint"] != package_name or binding["engine_id"] != engine_id or binding["engine_version"] != version:
            raise ValueError("registered engine entrypoint mismatch")
        row: dict[str, Any] = {
            "engine": kind, "engine_id": engine_id, "expected_version": version,
            "package": package_name, "host_target": host_target,
            "status": "NOT_INSTALLED", "execution_authorized": False,
        }
        if host_target not in pkg["build_targets"]:
            row["status"] = "UNSUPPORTED_HOST"
            entries.append(row)
            continue
        if bundled is not None:
            distribution, bundle = bundled
            origin = bundle["engines"][kind.lower()]
            row["package"] = "PTSIP"
            row["package_version"] = distribution.version
            row["installed_version"] = origin["engine_version"]
        else:
            try:
                distribution = metadata.distribution(package_name)
            except metadata.PackageNotFoundError:
                entries.append(row)
                continue
            row["installed_version"] = distribution.version
        if row["installed_version"] != version:
            row["status"] = "VERSION_MISMATCH"
            entries.append(row)
            continue
        extension = ".exe" if host_target and "windows" in host_target else ""
        executable = scripts / (binding["entrypoint"] + extension)
        if executable.is_symlink() or not executable.is_file():
            row["status"] = "EXECUTABLE_MISSING_OR_UNTRUSTED_PATH"
            entries.append(row)
            continue
        try:
            if bundled is None:
                origin_text = distribution.read_text("ptsip-origin.json")
                origin = json.loads(origin_text or "")
            if (
                origin["engine_id"] != engine_id
                or origin["engine_version"] != version
                or origin["host_target"] != host_target
                or origin["entrypoint"] != executable.name
                or not isinstance(origin["source_sha"], str)
                or len(origin["source_sha"]) != 40
                or origin["binary_sha256"] != _sha256(executable)
            ):
                row["status"] = "ARTIFACT_IDENTITY_MISMATCH"
            else:
                row["status"] = "INSTALLED_UNATTESTED_RELEASE"
                row["binary_sha256"] = origin["binary_sha256"]
                row["source_sha"] = origin["source_sha"]
        except (OSError, json.JSONDecodeError, KeyError, TypeError, ValueError):
            row["status"] = "ORIGIN_PROOF_MISSING_OR_INVALID"
        entries.append(row)
    return {
        "schema_version": "ptsip-official-engine-consumer-status/v1",
        "host_target": host_target,
        "distribution": "SINGLE_PROJECT_PLATFORM_BUNDLES" if bundled else "AUTO_INSTALLED_INDEPENDENT_PACKAGES_ONLY",
        "entries": entries,
        "all_installed": all(e["status"] == "INSTALLED_UNATTESTED_RELEASE" for e in entries),
        "release_attested": False,
        "execution_authorized": False,
        "status": "AWAITING_OFFICIAL_RELEASE_ATTESTATION"
        if all(e["status"] == "INSTALLED_UNATTESTED_RELEASE" for e in entries)
        else "INCOMPLETE",
    }


def emit_status() -> tuple[dict[str, Any], int]:
    payload = inspect()
    return payload, 0 if payload["all_installed"] else 4
