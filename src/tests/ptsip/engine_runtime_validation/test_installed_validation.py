"""Installed PTSIP VALIDATION engine smoke verification (not policy conformance).

Only reads the installed PTSIP distribution and invokes its registered native
VALIDATION entrypoint. This test MUST fail, not skip, when the installed engine
is missing, incompatible, or unable to evaluate the two known schema cases.
"""
from __future__ import annotations

from importlib import metadata
import json
import os
from pathlib import Path
import subprocess
import sysconfig

import pytest


def _installed_validation() -> tuple[dict, dict, Path]:
    distribution = metadata.distribution("PTSIP")
    expected_package = os.environ.get("PTSIP_TEST_PACKAGE_VERSION")
    if expected_package:
        assert distribution.version == expected_package, (
            f"expected PTSIP {expected_package}, installed {distribution.version}"
        )

    package_root = Path(distribution.locate_file("ptsip"))
    assert package_root.is_dir(), "installed PTSIP package root is missing"

    def registered_resource(reference: str) -> dict:
        assert reference.startswith("src/ptsip/"), "unregistered resource root"
        relative = reference.removeprefix("src/ptsip/")
        parts = Path(relative).parts
        assert parts and ".." not in parts and not Path(relative).is_absolute()
        resource = package_root.joinpath(*parts)
        assert resource.is_file(), f"installed registered resource absent: {reference}"
        return json.loads(resource.read_text(encoding="utf-8"))

    index = json.loads((package_root / "engines" / "index.json").read_text(encoding="utf-8"))
    matches = [item for item in index["official_engine_entries"] if item["kind"] == "VALIDATION"]
    assert len(matches) == 1, "exactly one registered official VALIDATION engine required"
    engine = matches[0]
    assert "validation.schema" in engine["operations"], "schema capability is not registered"

    manifest = registered_resource(engine["manifest_ref"])
    contract = registered_resource(engine["contract_ref"])
    bindings = registered_resource(engine["execution_binding_ref"])["bindings"]
    assert len(bindings) == 1, "execution binding must be unambiguous"
    binding = bindings[0]

    assert manifest["engine_id"] == engine["engine_id"] == binding["engine_id"]
    assert manifest["implementation_version"] == engine["version"] == binding["engine_version"]
    assert contract["contract_id"] == engine["capability_id"] == binding["capability_id"]
    assert contract["version"] == binding["contract_version"]
    assert binding["permission_grant"] is False
    assert contract["execution_permission_from_declaration"] is False

    operation = [value for value in contract["operations"] if value["operation_id"] == "validation.schema"]
    assert len(operation) == 1, "schema operation contract missing or duplicated"

    entrypoint = binding["entrypoint"]
    assert entrypoint == "ptsip-engine-validation", "unexpected native binding"
    suffix = ".exe" if os.name == "nt" else ""
    executable = Path(sysconfig.get_path("scripts")) / (entrypoint + suffix)
    assert executable.is_file(), f"registered native engine is not installed: {executable}"
    return manifest, contract, executable


@pytest.fixture(scope="module")
def installed_validation():
    return _installed_validation()


def _invoke(installed_validation, schema: dict, instance: object) -> tuple[subprocess.CompletedProcess, dict]:
    manifest, contract, executable = installed_validation
    request = {
        "engine_id": manifest["engine_id"],
        "engine_version": manifest["implementation_version"],
        "contract_id": contract["contract_id"],
        "contract_version": contract["version"],
        "operation_id": "validation.schema",
        "input": {"schema": schema, "instance": instance},
    }
    result = subprocess.run(
        [str(executable)],
        input=json.dumps(request, separators=(",", ":")),
        capture_output=True,
        text=True,
        encoding="utf-8",
        timeout=20,
        check=False,
    )
    assert result.stdout.strip(), (
        f"native engine produced no JSON result (exit={result.returncode}): {result.stderr}"
    )
    return result, json.loads(result.stdout)


def test_native_manifest_matches_registered_identity(installed_validation):
    manifest, _, executable = installed_validation
    result = subprocess.run(
        [str(executable), "--manifest"],
        capture_output=True,
        text=True,
        encoding="utf-8",
        timeout=20,
        check=False,
    )
    assert result.returncode == 0, result.stderr
    assert json.loads(result.stdout) == manifest


@pytest.mark.parametrize(
    ("instance", "expected_valid"),
    [
        ({"status": "ok"}, True),
        ({"status": 123}, False),
    ],
    ids=["valid-instance", "invalid-instance"],
)
def test_native_schema_result(installed_validation, instance, expected_valid):
    schema = {
        "type": "object",
        "properties": {"status": {"type": "string"}},
        "required": ["status"],
        "additionalProperties": False,
    }
    result, response = _invoke(installed_validation, schema, instance)
    assert result.returncode == 0, result.stderr
    assert response["status"] == "PASS"
    assert response["error"] is None
    assert response["normative_authority"] is False
    assert response["execution_authorized"] is False
    assert response["output"]["ptsip_conformance_verdict"] == "NOT_ASSERTED"
    assert response["output"]["valid"] is expected_valid
    if not expected_valid:
        assert response["output"]["diagnostics"], "invalid input must produce diagnostics"


def test_native_schema_rejects_unresolved_reference(installed_validation):
    result, response = _invoke(
        installed_validation, {"$ref": "https://example.invalid/unknown"},
        {"status": "ok"},
    )
    assert result.returncode == 2
    assert response["status"] == "FAIL"
    assert response["output"] is None
    assert response["normative_authority"] is False
    assert response["execution_authorized"] is False
