"""Independent package availability never constitutes release/activation proof."""
from __future__ import annotations

import hashlib
import json
from pathlib import Path

from ptsip import engine_installation


KINDS = ("CONTEXT", "STORAGE", "VALIDATION", "DISCOVERY", "PLANNING", "EXECUTION", "PROJECTION")


def registered():
    index = {"official_engine_entries": []}
    plan = {"default_available_engine_ids": [], "packages": []}
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
        plan["packages"].append({
            "engine_id": identity, "version": version,
            "package_distribution": name,
            "distribution": "INDEPENDENT_VERSIONED_NATIVE_WHEEL",
            "installation": "AUTO_INSTALL_AS_PTSIP_PACKAGE_DEPENDENCY",
            "build_targets": ["x86_64-unknown-linux-gnu"],
            "permission_from_availability": False,
        })
        bindings[kind] = {"bindings": [{
            "engine_id": identity, "engine_version": version, "entrypoint": name
        }]}
    return index, plan, bindings


def test_all_seven_missing_is_a_failure_not_silent_fallback(tmp_path: Path, monkeypatch) -> None:
    index, plan, bindings = registered()

    def missing(name: str):
        raise engine_installation.metadata.PackageNotFoundError(name)

    monkeypatch.setattr(engine_installation.metadata, "distribution", missing)
    result = engine_installation.inspect(
        index=index, plan=plan, bindings=bindings,
        scripts=tmp_path, host_target="x86_64-unknown-linux-gnu",
    )
    assert result["status"] == "INCOMPLETE"
    assert not result["all_installed"] and not result["execution_authorized"]
    assert len(result["entries"]) == 7
    assert {x["status"] for x in result["entries"]} == {"NOT_INSTALLED"}


def test_seven_installed_never_claim_official_release_attestation(tmp_path: Path, monkeypatch) -> None:
    index, plan, bindings = registered()
    distributions = {}
    for kind in KINDS:
        key = kind.lower()
        entry = next(x for x in index["official_engine_entries"] if x["kind"] == kind)
        name = "ptsip-engine-" + key
        binary = tmp_path / name
        binary.write_bytes(b"\x7fELF" + kind.encode())
        origin = {
            "engine_id": entry["engine_id"],
            "engine_version": entry["version"],
            "host_target": "x86_64-unknown-linux-gnu",
            "entrypoint": name,
            "source_sha": "a" * 40,
            "binary_sha256": hashlib.sha256(binary.read_bytes()).hexdigest(),
        }

        class Dist:
            pass

        dist = Dist()
        dist.version = entry["version"]
        dist.read_text = lambda path, value=json.dumps(origin): value
        distributions[name] = dist

    monkeypatch.setattr(
        engine_installation.metadata, "distribution",
        lambda name: distributions[name]
    )
    result = engine_installation.inspect(
        index=index, plan=plan, bindings=bindings,
        scripts=tmp_path, host_target="x86_64-unknown-linux-gnu",
    )
    assert result["all_installed"]
    assert result["status"] == "AWAITING_OFFICIAL_RELEASE_ATTESTATION"
    assert result["release_attested"] is False
    assert result["execution_authorized"] is False
    assert {e["status"] for e in result["entries"]} == {"INSTALLED_UNATTESTED_RELEASE"}
    (tmp_path / "ptsip-engine-storage").write_text("tampered", encoding="utf-8")
    mismatch = engine_installation.inspect(
        index=index, plan=plan, bindings=bindings,
        scripts=tmp_path, host_target="x86_64-unknown-linux-gnu",
    )
    assert not mismatch["all_installed"]
    assert any(x["status"] == "ARTIFACT_IDENTITY_MISMATCH" for x in mismatch["entries"])


def test_old_manual_hybrid_binding_is_rejected(tmp_path: Path, monkeypatch) -> None:
    index, plan, bindings = registered()
    plan["packages"][0]["distribution"] = "INDEPENDENT_CARGO_PACKAGE"
    try:
        engine_installation.inspect(
            index=index, plan=plan, bindings=bindings,
            scripts=tmp_path, host_target="x86_64-unknown-linux-gnu",
        )
    except ValueError as exc:
        assert "outdated or incompatible" in str(exc)
    else:
        raise AssertionError("legacy manual/hybrid contract incorrectly admitted")
