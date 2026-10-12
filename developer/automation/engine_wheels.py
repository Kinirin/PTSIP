"""Package one already-built, independently verified official Rust engine.

This is an artifact assembler, not a build or release authorization gate.
Never compiles engines, fetches dependencies, or grants runtime permission.
"""
from __future__ import annotations

import argparse
import base64
import csv
import hashlib
import io
import json
import re
import stat
import zipfile
from pathlib import Path

TARGETS = {
    "x86_64-pc-windows-msvc": ("win_amd64", ".exe", b"MZ"),
    "x86_64-unknown-linux-gnu": ("linux_x86_64", "", b"\x7fELF"),
    "aarch64-apple-darwin": ("macosx_11_0_arm64", "", None),
}
MAX_BINARY_SIZE = 268_435_456
SHA_RE = re.compile(r"[0-9a-f]{40}")


def _inside(root: Path, reference: str) -> Path:
    candidate = Path(reference)
    if candidate.is_absolute() or not candidate.parts or any(
        part in (".", "..") for part in candidate.parts
    ) or "\\" in reference:
        raise ValueError(f"untrusted repository reference: {reference}")
    target = root.joinpath(candidate)
    if any(p.is_symlink() for p in (target, *target.parents) if p != root):
        raise ValueError("symlinked package source is not admitted")
    resolved = target.resolve(strict=True)
    if not resolved.is_relative_to(root):
        raise ValueError("package source escaped repository root")
    return resolved


def _json(root: Path, reference: str) -> dict:
    path = _inside(root, reference)
    if path.stat().st_size > 2_097_152:
        raise ValueError("contract metadata too large")
    return json.loads(path.read_text(encoding="utf-8"))


def _hash(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def _info(name: str, mode: int = 0o644) -> zipfile.ZipInfo:
    zi = zipfile.ZipInfo(name, (2020, 1, 1, 0, 0, 0))
    zi.create_system = 3
    zi.external_attr = ((stat.S_IFREG | mode) << 16)
    zi.compress_type = zipfile.ZIP_DEFLATED
    return zi


def build(
    root: Path,
    *,
    kind: str,
    target: str,
    binary: Path,
    output: Path,
    source_sha: str,
) -> tuple[Path, Path]:
    root = root.resolve(strict=True)
    if not SHA_RE.fullmatch(source_sha):
        raise ValueError("exact 40-character source SHA required")
    if target not in TARGETS:
        raise ValueError("unsupported official engine build target")
    platform, extension, signature = TARGETS[target]
    index = _json(root, "src/ptsip/engines/index.json")
    plan = _json(root, "src/ptsip/engines/contracts/installation-plan.json")
    entries = index["official_engine_entries"]
    packages = plan["packages"]
    if len(entries) != 7 or len(packages) != 7:
        raise ValueError("exact seven independent official engines required")
    if len({e["engine_id"] for e in entries}) != 7 or len(
        {p["engine_id"] for p in packages}
    ) != 7:
        raise ValueError("duplicate official engine identity")
    if {e["engine_id"] for e in entries} != {p["engine_id"] for p in packages}:
        raise ValueError("official registry and installation contract diverged")
    matches = [e for e in entries if e["kind"].lower() == kind.lower()]
    if len(matches) != 1:
        raise ValueError("engine kind not registered exactly once")
    entry = matches[0]
    package = next(p for p in packages if p["engine_id"] == entry["engine_id"])
    version = entry["version"]
    distribution = f"ptsip-engine-{kind.lower()}"
    if (
        package["version"] != version
        or package["package_distribution"] != distribution
        or package["distribution"] != "INDEPENDENT_VERSIONED_NATIVE_WHEEL"
        or package["installation"] != "AUTO_INSTALL_AS_PTSIP_PACKAGE_DEPENDENCY"
        or target not in package["build_targets"]
        or package["permission_from_availability"] is not False
        or package["installed_is_execution_authorization"] is not False
    ):
        raise ValueError("independent auto-installed package contract mismatch")
    manifest = _json(root, entry["manifest_ref"])
    if manifest["engine_id"] != entry["engine_id"] or manifest["implementation_version"] != version:
        raise ValueError("engine manifest identity/version mismatch")
    bindings = _json(root, entry["execution_binding_ref"])["bindings"]
    if len(bindings) != 1:
        raise ValueError("ambiguous binary entrypoint")
    binding = bindings[0]
    command = distribution
    if (
        binding["engine_id"] != entry["engine_id"]
        or binding["engine_version"] != version
        or binding["entrypoint"] != command
    ):
        raise ValueError("registered execution binding mismatch")
    binary = binary.resolve(strict=True)
    if not binary.is_file() or binary.name != command + extension:
        raise ValueError("wrong independent executable name")
    if binary.stat().st_size < 64 or binary.stat().st_size > MAX_BINARY_SIZE:
        raise ValueError("invalid independent engine executable size")
    contents = binary.read_bytes()
    if signature is not None and not contents.startswith(signature):
        raise ValueError("engine artifact does not match declared platform")
    if signature is None and not (
        contents.startswith(bytes.fromhex("cffaedfe"))
        or contents.startswith(bytes.fromhex("feedfacf"))
    ):
        raise ValueError("engine artifact is not Mach-O")
    filename = f"{distribution.replace('-', '_')}-{version}-py3-none-{platform}.whl"
    wheel_dist = distribution.replace("-", "_")
    dist_info = f"{wheel_dist}-{version}.dist-info"
    script = f"{wheel_dist}-{version}.data/scripts/{command}{extension}"
    metadata = (
        "Metadata-Version: 2.3\n"
        f"Name: {distribution}\n"
        f"Version: {version}\n"
        "Summary: Independently versioned official PTSIP native engine\n"
        "Requires-Python: >=3.11\n"
        "License-Expression: LicenseRef-PTSIP-Community-Reciprocity-1.0\n"
        "License-File: LICENSE.md\n"
        f"X-PTSIP-Engine-Id: {entry['engine_id']}\n"
        f"X-PTSIP-Source-Sha: {source_sha}\n"
    ).encode()
    wheel_meta = (
        "Wheel-Version: 1.0\n"
        "Generator: ptsip-independent-engine-wheel/1\n"
        "Root-Is-Purelib: false\n"
        f"Tag: py3-none-{platform}\n"
    ).encode()
    license_bytes = _inside(root, "License-Authority/LICENSE.md").read_bytes()
    origin = json.dumps(
        {
            "schema_version": "ptsip-independent-wheel-origin/v1",
            "source_sha": source_sha,
            "engine_id": entry["engine_id"],
            "engine_version": version,
            "host_target": target,
            "entrypoint": command + extension,
            "binary_sha256": _hash(contents),
            "execution_authorized": False,
        },
        sort_keys=True,
        separators=(",", ":"),
    ).encode() + b"\n"
    members = {
        script: (contents, 0o755),
        f"{dist_info}/METADATA": (metadata, 0o644),
        f"{dist_info}/WHEEL": (wheel_meta, 0o644),
        f"{dist_info}/licenses/LICENSE.md": (license_bytes, 0o644),
        f"{dist_info}/ptsip-origin.json": (origin, 0o644),
    }
    record_name = f"{dist_info}/RECORD"
    rows = []
    for name, (data, _) in sorted(members.items()):
        digest = base64.urlsafe_b64encode(hashlib.sha256(data).digest()).rstrip(b"=").decode()
        rows.append((name, f"sha256={digest}", str(len(data))))
    rows.append((record_name, "", ""))
    buffer = io.StringIO(newline="")
    writer = csv.writer(buffer, lineterminator="\n")
    writer.writerows(rows)
    output.mkdir(parents=True, exist_ok=True)
    destination = output / filename
    with zipfile.ZipFile(destination, "w", strict_timestamps=True) as wheel:
        for name, (data, mode) in sorted(members.items()):
            wheel.writestr(_info(name, mode), data)
        wheel.writestr(_info(record_name), buffer.getvalue().encode())
    receipt = {
        "schema_version": "ptsip-independent-wheel-receipt/v1",
        "source_sha": source_sha,
        "engine_id": entry["engine_id"],
        "version": version,
        "target": target,
        "wheel": filename,
        "wheel_sha256": _hash(destination.read_bytes()),
        "binary_sha256": _hash(contents),
        "install_authorized": False,
        "runtime_authorized": False,
        "release_verification": "NOT_PERFORMED_BY_ASSEMBLER",
    }
    receipt_path = output / f"{filename}.receipt.json"
    receipt_path.write_text(json.dumps(receipt, indent=2) + "\n", encoding="utf-8")
    return destination, receipt_path


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Build one independent official engine wheel")
    parser.add_argument("--root", type=Path, default=Path("."))
    parser.add_argument("--kind", required=True, choices=[*map(str.lower, ("CONTEXT", "STORAGE", "VALIDATION", "DISCOVERY", "PLANNING", "EXECUTION", "PROJECTION"))])
    parser.add_argument("--target", choices=tuple(TARGETS), required=True)
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--source-sha", required=True)
    args = parser.parse_args(argv)
    try:
        wheel, receipt = build(args.root, kind=args.kind, target=args.target, binary=args.binary, output=args.output, source_sha=args.source_sha)
    except (OSError, KeyError, ValueError, TypeError) as exc:
        parser.exit(2, f"ENGINE_WHEEL_BUILD_BLOCKED: {exc}\n")
    print(json.dumps({"status": "PACKAGED_UNRELEASED", "wheel": str(wheel), "receipt": str(receipt)}))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
