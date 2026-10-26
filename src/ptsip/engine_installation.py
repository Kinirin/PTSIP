"""Read-only diagnostics for seven native engines in one PTSIP host wheel.

An installed bundle is not a release admission, PTSIP conformance verdict,
activation grant or execution authorization. Independent ptsip-engine-* pip
distributions are retired from the official PTSIP consumer install route.
"""
from __future__ import annotations

import hashlib
import json
import platform
import re
import sys
import sysconfig
import subprocess
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


def _bundle(items: list[dict[str, Any]], host_target: str | None) -> tuple[Any, dict[str, Any]] | None:
    try:
        distribution = metadata.distribution("PTSIP")
    except metadata.PackageNotFoundError:
        return None
    content = distribution.read_text("ptsip-engines.json")
    if content is None:
        return None
    try:
        bundle = json.loads(content)
        engines = bundle["engines"]
        expected = {entry["kind"].lower() for entry in items}
        platform_id = bundle["platform"]
        target_for_platform = {
            "windows-x64": "x86_64-pc-windows-msvc",
            "linux-x64": "x86_64-unknown-linux-gnu",
            "macos-arm64": "aarch64-apple-darwin",
        }
        if (
            bundle["schema_version"] != "ptsip-bundled-engine-origins/v1"
            or bundle["package_version"] != distribution.version
            or bundle["execution_authorized"] is not False
            or len(items) != 7 or len(expected) != 7
            or not isinstance(engines, dict) or set(engines) != expected
            or target_for_platform.get(platform_id) != host_target
        ):
            raise ValueError("incomplete or incompatible PTSIP engine bundle")
        for entry in items:
            origin = engines[entry["kind"].lower()]
            name = "ptsip-engine-" + entry["kind"].lower()
            expected_script = name + (".exe" if platform_id == "windows-x64" else "")
            if (
                origin["schema_version"] != "ptsip-independent-wheel-origin/v1"
                or origin["engine_id"] != entry["engine_id"]
                or origin["engine_version"] != entry["version"]
                or origin["host_target"] != host_target
                or origin["entrypoint"] != expected_script
                or origin["execution_authorized"] is not False
                or not isinstance(origin["binary_sha256"], str)
                or re.fullmatch(r"[0-9a-f]{64}", origin["binary_sha256"]) is None
                or re.fullmatch(r"[0-9a-f]{40}", origin["source_sha"]) is None
            ):
                raise ValueError("PTSIP bundled engine identity differs")
        return distribution, bundle
    except (KeyError, TypeError, json.JSONDecodeError) as exc:
        raise ValueError("PTSIP bundled engine metadata invalid") from exc


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
    components = {p["engine_id"]: p for p in plan["components"]}
    if (
        plan["schema_version"] != "ptsip-engine-installation-plan/v2"
        or plan["selection"] != "REGISTERED_NATIVE_COMPONENTS_FOR_ONE_PTSIP_PLATFORM_WHEEL"
        or plan["default_installation"] != "ONE_COMPLETE_PTSIP_PLATFORM_WHEEL_WITH_ALL_SEVEN_ENGINES"
        or plan["distribution"]["model"] != "SINGLE_PLATFORM_SPECIFIC_PTSIP_WHEEL"
        or plan["distribution"]["public_python_project"] != "PTSIP"
        or plan["distribution"]["engine_pip_dependencies"] != "FORBIDDEN"
        or plan["distribution"]["publication_authorized"] is not False
        or len(items) != 7 or len(components) != 7
        or len({i["engine_id"] for i in items}) != 7
        or set(components) != {i["engine_id"] for i in items}
        or set(plan["default_available_engine_ids"]) != set(components)
    ):
        raise ValueError("official integrated engine installation plan incomplete/ambiguous")
    host_target = host_target or HOST_TARGETS.get((sys.platform, platform.machine().lower()))
    scripts = scripts or Path(sysconfig.get_path("scripts"))
    bundle = _bundle(items, host_target)
    entries: list[dict[str, Any]] = []
    for entry in items:
        kind, engine_id, version = entry["kind"], entry["engine_id"], entry["version"]
        component = components[engine_id]
        name = "ptsip-engine-" + kind.lower()
        if (
            component["version"] != version
            or component["entrypoint"] != name
            or component["distribution"] != "PTSIP_PLATFORM_WHEEL_NATIVE_COMPONENT"
            or component["installation"] != "INSTALLED_AS_PART_OF_PTSIP_PLATFORM_WHEEL"
            or component["permission_from_availability"] is not False
            or component["installed_is_execution_authorization"] is not False
            or "package_distribution" in component
        ):
            raise ValueError("outdated or incompatible native engine bundle component")
        b = bindings[kind] if bindings is not None else _load(
            f"{kind.lower()}/execution-bindings.json"
        )
        candidates = b["bindings"]
        if len(candidates) != 1:
            raise ValueError("ambiguous registered engine executable")
        binding = candidates[0]
        if (
            binding["entrypoint"] != name
            or binding["engine_id"] != engine_id
            or binding["engine_version"] != version
        ):
            raise ValueError("registered engine entrypoint mismatch")
        row: dict[str, Any] = {
            "engine": kind, "engine_id": engine_id, "expected_version": version,
            "package": "PTSIP", "host_target": host_target,
            "status": "BUNDLE_NOT_INSTALLED", "execution_authorized": False,
        }
        if host_target not in component["build_targets"]:
            row["status"] = "UNSUPPORTED_HOST"
        elif bundle is not None:
            distribution, record = bundle
            origin = record["engines"][kind.lower()]
            row["package_version"] = distribution.version
            row["installed_version"] = origin["engine_version"]
            if row["installed_version"] != version:
                row["status"] = "VERSION_MISMATCH"
            else:
                suffix = ".exe" if host_target and "windows" in host_target else ""
                executable = scripts / (name + suffix)
                if executable.is_symlink() or not executable.is_file():
                    row["status"] = "EXECUTABLE_MISSING_OR_UNTRUSTED_PATH"
                else:
                    try:
                        digest = _sha256(executable)
                        if origin["binary_sha256"] != digest:
                            row["status"] = "ARTIFACT_IDENTITY_MISMATCH"
                        else:
                            row["status"] = "INSTALLED_UNATTESTED_RELEASE"
                            row["binary_sha256"] = digest
                            row["source_sha"] = origin["source_sha"]
                    except OSError:
                        row["status"] = "ORIGIN_PROOF_MISSING_OR_INVALID"
        entries.append(row)
    complete = all(row["status"] == "INSTALLED_UNATTESTED_RELEASE" for row in entries)
    return {
        "schema_version": "ptsip-official-engine-consumer-status/v1",
        "host_target": host_target,
        "distribution": "SINGLE_PROJECT_PLATFORM_BUNDLES_ONLY",
        "entries": entries,
        "all_installed": complete,
        "release_attested": False,
        "execution_authorized": False,
        "status": "AWAITING_OFFICIAL_RELEASE_ATTESTATION" if complete else "INCOMPLETE",
    }


def emit_status() -> tuple[dict[str, Any], int]:
    payload = inspect()
    return payload, 0 if payload["all_installed"] else 4


def emit_compatibility() -> tuple[dict[str, Any], int]:
    """Locate the selected installation and delegate diagnostics to native Discovery.

    This adapter makes no compatibility judgment. Only its own launcher artifact
    is checked here before executing the bundled, versioned native operation.
    """
    try:
        distribution = metadata.distribution("PTSIP")
        metadata_files = [
            f for f in distribution.files or ()
            if str(f).replace("\\", "/").endswith(".dist-info/METADATA")
        ]
        if len(metadata_files) != 1:
            raise ValueError("A complete installed PTSIP wheel is required")
        distribution_root = Path(distribution.locate_file(metadata_files[0])).parent
        bundle = json.loads((distribution_root / "ptsip-engines.json").read_text(encoding="utf-8"))
        package_root = Path(str(resources.files("ptsip")))
        scripts_root = Path(sysconfig.get_path("scripts"))
        suffix = ".exe" if sys.platform == "win32" else ""
        executable = scripts_root / ("ptsip-engine-discovery" + suffix)
        origin = bundle["engines"]["discovery"]
        if executable.is_symlink() or not executable.is_file() or _sha256(executable) != origin["binary_sha256"]:
            raise ValueError("Bundled Discovery executable is missing or its hash differs")
        manifest = _load("discovery/manifest.json")
        contract = _load("discovery/contract.json")
        request = {
            "engine_id": manifest["engine_id"],
            "engine_version": manifest["implementation_version"],
            "contract_id": contract["contract_id"], "contract_version": contract["version"],
            "operation_id": "discovery.check-environment",
            "input": {"package_root": str(package_root), "scripts_root": str(scripts_root),
                      "distribution_root": str(distribution_root)},
        }
        completed = subprocess.run(
            [str(executable)], input=json.dumps(request), capture_output=True,
            encoding="utf-8", timeout=30, check=False,
        )
        result = json.loads(completed.stdout)
        if not isinstance(result, dict):
            raise ValueError("Native Discovery returned an invalid response envelope")
        if completed.returncode != 0 or result.get("status") != "PASS":
            raise ValueError(str(result.get("error", "Native Discovery diagnosis failed")))
        if result.get("normative_authority") is not False or result.get("execution_authorized") is not False:
            raise ValueError("Native Discovery response expanded diagnostic authority")
        for field in ("engine_id", "engine_version", "contract_id", "contract_version"):
            if result.get(field) != request[field]:
                raise ValueError("Native Discovery response identity differs")
        output = result["output"]
        if not isinstance(output, dict) or output.get("schema_version") != "ptsip-official-engine-compatibility/v1":
            raise ValueError("Native Discovery returned an unsupported diagnostic contract")
        from .constants import TOOL_VERSION
        output["runtime"] = {"python_executable": sys.executable, "tool_version": TOOL_VERSION}
        return output, 0 if output["status"] == "VERIFIED" else 4
    except (OSError, ValueError, KeyError, TypeError, metadata.PackageNotFoundError, subprocess.TimeoutExpired) as exc:
        return {
            "schema_version": "ptsip-official-engine-compatibility-error/v1",
            "status": "UNVERIFIED", "reason": str(exc),
            "required_action": "Use a complete PTSIP installation containing the current native Discovery diagnostic contract",
            "execution_authorized": False, "pairwise_compatibility_guaranteed": False,
        }, 4
