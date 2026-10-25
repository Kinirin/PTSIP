"""Integrated PTSIP engine bundle diagnostics remain non-authoritative."""
from __future__ import annotations

import hashlib
import json
from pathlib import Path

import pytest

from ptsip import engine_installation

KINDS = ("CONTEXT", "STORAGE", "VALIDATION", "DISCOVERY", "PLANNING", "EXECUTION", "PROJECTION")


def registered():
    index = {"official_engine_entries": []}
    plan = {
        "schema_version": "ptsip-engine-installation-plan/v2",
        "selection": "REGISTERED_NATIVE_COMPONENTS_FOR_ONE_PTSIP_PLATFORM_WHEEL",
        "default_installation": "ONE_COMPLETE_PTSIP_PLATFORM_WHEEL_WITH_ALL_SEVEN_ENGINES",
        "distribution": {
            "model": "SINGLE_PLATFORM_SPECIFIC_PTSIP_WHEEL",
            "public_python_project": "PTSIP",
            "engine_pip_dependencies": "FORBIDDEN",
            "publication_authorized": False,
        },
        "default_available_engine_ids": [],
        "components": [],
    }
    bindings = {}
    for kind in KINDS:
        key = kind.lower()
        version = "0.2.0" if kind in ("CONTEXT", "PROJECTION") else "0.3.0"
        identity = "urn:uuid:" + key
        name = "ptsip-engine-" + key
        index["official_engine_entries"].append({
            "kind": kind, "engine_id": identity, "version": version
        })
        plan["default_available_engine_ids"].append(identity)
        plan["components"].append({
            "engine_id": identity, "version": version,
            "entrypoint": name,
            "distribution": "PTSIP_PLATFORM_WHEEL_NATIVE_COMPONENT",
            "installation": "INSTALLED_AS_PART_OF_PTSIP_PLATFORM_WHEEL",
            "build_targets": ["x86_64-unknown-linux-gnu"],
            "permission_from_availability": False,
            "installed_is_execution_authorization": False,
        })
        bindings[kind] = {"bindings": [{
            "engine_id": identity, "engine_version": version, "entrypoint": name
        }]}
    return index, plan, bindings


def bundle(index, scripts: Path):
    origins = {}
    for entry in index["official_engine_entries"]:
        key = entry["kind"].lower()
        name = "ptsip-engine-" + key
        binary = scripts / name
        binary.write_bytes(b"\x7fELF" + entry["kind"].encode())
        origins[key] = {
            "schema_version": "ptsip-independent-wheel-origin/v1",
            "engine_id": entry["engine_id"],
            "engine_version": entry["version"],
            "host_target": "x86_64-unknown-linux-gnu",
            "entrypoint": name,
            "source_sha": "a" * 40,
            "binary_sha256": hashlib.sha256(binary.read_bytes()).hexdigest(),
            "execution_authorized": False,
        }
    return {
        "schema_version": "ptsip-bundled-engine-origins/v1",
        "package_version": "0.3.8b6",
        "platform": "linux-x64",
        "execution_authorized": False,
        "engines": origins,
    }


def test_missing_bundle_fails_closed_without_separate_pip_fallback(tmp_path: Path, monkeypatch) -> None:
    index, plan, bindings = registered()
    seen = []

    def missing(name: str):
        seen.append(name)
        raise engine_installation.metadata.PackageNotFoundError(name)

    monkeypatch.setattr(engine_installation.metadata, "distribution", missing)
    result = engine_installation.inspect(
        index=index, plan=plan, bindings=bindings,
        scripts=tmp_path, host_target="x86_64-unknown-linux-gnu"
    )
    assert seen == ["PTSIP"]
    assert result["status"] == "INCOMPLETE"
    assert result["distribution"] == "SINGLE_PROJECT_PLATFORM_BUNDLES_ONLY"
    assert not result["all_installed"] and not result["execution_authorized"]
    assert len(result["entries"]) == 7
    assert {x["status"] for x in result["entries"]} == {"BUNDLE_NOT_INSTALLED"}


def test_installed_bundle_hashes_never_grant_release_attestation(tmp_path: Path, monkeypatch) -> None:
    index, plan, bindings = registered()
    record = bundle(index, tmp_path)

    class Dist:
        version = "0.3.8b6"

        def read_text(self, name):
            return json.dumps(record) if name == "ptsip-engines.json" else None

    monkeypatch.setattr(
        engine_installation.metadata, "distribution",
        lambda name: Dist() if name == "PTSIP" else None
    )
    result = engine_installation.inspect(
        index=index, plan=plan, bindings=bindings,
        scripts=tmp_path, host_target="x86_64-unknown-linux-gnu"
    )
    assert result["all_installed"]
    assert result["status"] == "AWAITING_OFFICIAL_RELEASE_ATTESTATION"
    assert result["release_attested"] is False
    assert result["execution_authorized"] is False
    assert {e["status"] for e in result["entries"]} == {"INSTALLED_UNATTESTED_RELEASE"}

    (tmp_path / "ptsip-engine-storage").write_text("tampered", encoding="utf-8")
    changed = engine_installation.inspect(
        index=index, plan=plan, bindings=bindings,
        scripts=tmp_path, host_target="x86_64-unknown-linux-gnu"
    )
    assert not changed["all_installed"]
    assert any(x["status"] == "ARTIFACT_IDENTITY_MISMATCH" for x in changed["entries"])


def test_legacy_independent_distribution_contract_is_rejected(tmp_path: Path) -> None:
    index, plan, bindings = registered()
    plan["components"][0]["distribution"] = "INDEPENDENT_VERSIONED_NATIVE_WHEEL"
    with pytest.raises(ValueError, match="outdated or incompatible"):
        engine_installation.inspect(
            index=index, plan=plan, bindings=bindings,
            scripts=tmp_path, host_target="x86_64-unknown-linux-gnu"
        )


def test_origin_unknown_platform_and_activation_must_fail_closed(tmp_path: Path, monkeypatch) -> None:
    index, plan, bindings = registered()
    record = bundle(index, tmp_path)

    class Dist:
        version = "0.3.8b6"

        def read_text(self, name):
            return json.dumps(record)

    monkeypatch.setattr(engine_installation.metadata, "distribution", lambda name: Dist())
    record["platform"] = "unknown"
    with pytest.raises(ValueError, match="incomplete or incompatible"):
        engine_installation.inspect(
            index=index, plan=plan, bindings=bindings,
            scripts=tmp_path, host_target="x86_64-unknown-linux-gnu"
        )
    record["platform"] = "linux-x64"
    record["engines"]["context"]["execution_authorized"] = True
    with pytest.raises(ValueError, match="bundled engine identity differs"):
        engine_installation.inspect(
            index=index, plan=plan, bindings=bindings,
            scripts=tmp_path, host_target="x86_64-unknown-linux-gnu"
        )
