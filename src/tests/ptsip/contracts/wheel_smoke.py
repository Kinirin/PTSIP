"""Install a local PTSIP wheel and run its common contracts outside the checkout."""

import argparse
import hashlib
from importlib import resources
import json
import os
from pathlib import Path, PurePosixPath
import subprocess
import sys
import tempfile
import venv


FORMAT = "ptsip-common-contract/v1"
POLICY = "urn:ptsip:contract:public-structure-policy:v4"
LIFECYCLE = "urn:ptsip:contract:policy:lifecycle:v4"
IDENTITY = "urn:ptsip:contract:policy:identity:v4"
URNS = {
    POLICY, IDENTITY, LIFECYCLE, "urn:ptsip:contract:policy:rules:v4", "urn:ptsip:contract:policy:content:v4",
}
RETIRED_URNS = {
    "urn:ptsip:common_policy:identity:v1", "urn:ptsip:common_policy:lifecycle:v1",
    "urn:ptsip:common_policy:rules:v1", "urn:ptsip:common:plan:identity:v1",
    "urn:ptsip:common:plan:lifecycle:v1", "urn:ptsip:common:plan:unit:v1",
}
PLAN_FIELDS = {"Plan_Document", "Plan_Identity", "Plan_Lifecycle", "Plan_Unit"}
STATUSES = [
    "DRAFT.Default", "DRAFT.Reviewed", "DRAFT.Denied", "DRAFT.Pass", "APPROVED.Default",
    "APPROVED.Applying", "APPROVED.Application_Problems", "APPROVED.Unresolved", "APPROVED.Activate",
    "APPROVED.provisional", "RETIRED.Default", "RETIRED.Removable", "RETIRED.Waiting",
]


def verify_registered_rbac_structures(package):
    root = package.joinpath("schemas")
    config = json.loads(root.joinpath("config.json").read_bytes())
    registrations = config["storage"]["Responsibility-Authority"]
    assert set(registrations) == {"RA_list", "RAC_list", "AEC_list"}
    cases = json.loads(Path(__file__).with_name("rbac-list-conformance.json").read_bytes())
    validation = package.joinpath("engines/validation")
    manifest = json.loads(validation.joinpath("manifest.json").read_bytes())
    contract = json.loads(validation.joinpath("contract.json").read_bytes())
    binary = Path(sys.executable).with_name("ptsip-engine-validation" + (".exe" if os.name == "nt" else ""))
    schemas = {}
    for key, entry in registrations.items():
        path = PurePosixPath(entry["path"])
        assert path.parent == PurePosixPath("storage/rbac") and path.as_posix() == entry["path"]
        schema = json.loads(root.joinpath(*path.parts).read_bytes())
        assert schema["$id"] == entry["$id"]
        assert schema["$schema"] == "https://json-schema.org/draft/2020-12/schema"
        schemas[key] = schema
    assert schemas.keys() == {case["registration"] for case in cases}
    for case in cases:
        request = {
            "engine_id": manifest["engine_id"],
            "engine_version": manifest["implementation_version"],
            "contract_id": contract["contract_id"],
            "contract_version": contract["version"],
            "operation_id": "validation.schema",
            "input": {"schema": schemas[case["registration"]], "instance": case["instance"]},
        }
        result = subprocess.run([str(binary)], input=json.dumps(request).encode("utf-8"),
                                capture_output=True, check=True, timeout=20)
        report = json.loads(result.stdout)
        assert report["output"]["valid"] is case["valid"], (case["name"], report)
        assert report["output"]["ptsip_conformance_verdict"] == "NOT_ASSERTED"
        assert report["execution_authorized"] is False
    return len(cases)


def verify(expected_sha):
    package = resources.files("ptsip")
    root = package.joinpath("common_contract")
    assert Path(str(root)).is_relative_to(Path(sys.prefix))
    assert not package.joinpath("policy_contracts").is_dir()
    assert not root.joinpath("go.mod").is_file()
    assert not root.joinpath("source/verification.go").is_file()
    assert not root.joinpath("schemas").is_dir()
    origin = json.loads(root.joinpath("origin.json").read_bytes())
    assert origin["format"] == "ptsip-common-contract-wheel-origin/v1"
    assert origin["source_sha"] == expected_sha
    binary = root.joinpath(origin["binary"])
    assert hashlib.sha256(binary.read_bytes()).hexdigest() == origin["binary_sha256"]
    manifest_path = root.joinpath("registry/manifest.json")
    manifest = manifest_path.read_bytes()
    assert hashlib.sha256(manifest).hexdigest() == origin["registry_sha256"]
    registry = json.loads(manifest)
    assert registry["revision"] == "v2"
    entries = registry["contracts"]
    assert len(entries) == len(URNS) and {entry["urn"] for entry in entries} == URNS
    diagnostics = registry["diagnostics"]
    diagnostic_fields = PLAN_FIELDS | {entry["field"] for entry in entries}
    assert len(diagnostics) == len(diagnostic_fields) and {entry["field"] for entry in diagnostics} == diagnostic_fields

    def request(operation, **values):
        payload = json.dumps({"format": FORMAT, "operation": operation, **values}).encode("ascii")
        result = subprocess.run([str(binary)], input=payload, capture_output=True, check=True, timeout=20)
        return json.loads(result.stdout)

    for entry in entries:
        assert entry["version"] == "v4" and entry["resource"].startswith("schemas/policy/")
        data = package.joinpath(entry["resource"]).read_bytes()
        assert hashlib.sha256(data).hexdigest() == entry["sha256"]
        lookup = request("lookup", urn=entry["urn"])
        assert lookup["status"] == "RESOLVED", lookup
        assert lookup["schema"]["$id"] == entry["urn"]
        assert lookup["contract"] == entry
        assert lookup["provenance"]["registry_revision"] == registry["revision"]
        assert lookup["provenance"]["source_revision"] == expected_sha
        assert lookup["provenance"]["registry_sha256"] == origin["registry_sha256"]
    for urn in RETIRED_URNS:
        lookup = request("lookup", urn=urn)
        assert lookup["status"] == "UNRESOLVED" and "UNREGISTERED_URN" in lookup["reference_resolution"]["problem"], lookup

    for entry in diagnostics:
        path = entry["resource"]
        if entry["field"] in PLAN_FIELDS:
            assert path.startswith("schemas/plan/")
        data = package.joinpath(path).read_bytes()
        assert hashlib.sha256(data).hexdigest() == entry["sha256"]
        diagnostic = request("inspect-resource", field=entry["field"])
        assert diagnostic["status"] == "RESOLVED" and diagnostic["binding"] == "DIAGNOSTIC_ONLY", diagnostic
        assert diagnostic["resource"] == path and diagnostic["schema"] == json.loads(data)
    plan_status = request("inspect-resource", field="Plan_Lifecycle", instance={"status": "Halt.Paused"})
    assert plan_status["status"] == "VALID" and plan_status["binding"] == "DIAGNOSTIC_ONLY", plan_status

    for status in STATUSES:
        report = request("validate", urn=LIFECYCLE, instance={"status": status})
        assert report["status"] == "VALID", report
    policy = {
        "Policy_Identity": {"id": "example"},
        "Policy_Lifecycle": {"status": "APPROVED.Default"},
        "Policy_Rules": {"rules": [{"id": "example"}]},
        "Policy_Content": {"items": [{"id": "example", "value": True}]},
    }
    report = request("validate", urn=POLICY, instance=policy)
    assert report["status"] == "VALID", report
    policy["Policy_Lifecycle"]["status"] = "APPROVED"
    report = request("validate", urn=POLICY, instance=policy)
    assert report["status"] == "INVALID", report

    lifecycle_entry = next(entry for entry in entries if entry["urn"] == LIFECYCLE)
    lifecycle = package.joinpath(lifecycle_entry["resource"])
    lifecycle_schema = lifecycle.read_bytes()
    try:
        lifecycle.write_bytes(b"{}")
        for urn in (LIFECYCLE, POLICY):
            stale = request("lookup", urn=urn)
            assert stale["status"] == "UNRESOLVED" and "STALE_SCHEMA" in stale["reference_resolution"]["problem"], stale
        lifecycle.unlink()
        missing = request("lookup", urn=POLICY)
        assert missing["status"] == "UNRESOLVED" and "RESOURCE_UNAVAILABLE" in missing["reference_resolution"]["problem"], missing
        assert request("lookup", urn=IDENTITY)["status"] == "RESOLVED"
    finally:
        lifecycle.write_bytes(lifecycle_schema)
    try:
        manifest_path.write_bytes(manifest + b"\n")
        stale = request("lookup", urn=IDENTITY)
        assert stale["status"] == "UNRESOLVED" and "STALE_REGISTRY" in stale["reference_resolution"]["problem"], stale
    finally:
        manifest_path.write_bytes(manifest)
    rbac_cases = verify_registered_rbac_structures(package)
    print(json.dumps({"status": "PASS", "source_sha": expected_sha, "platform": origin["goos"],
                      "architecture": origin["goarch"], "schemas": len(entries), "statuses": len(STATUSES),
                      "plan_diagnostic_schemas": len(PLAN_FIELDS), "rbac_structural_cases": rbac_cases,
                      "registry_sha256": origin["registry_sha256"], "binary_sha256": origin["binary_sha256"]}))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("artifact", type=Path)
    parser.add_argument("source_sha")
    parser.add_argument("--verify", action="store_true", help=argparse.SUPPRESS)
    args = parser.parse_args()
    if args.verify:
        verify(args.source_sha)
        return
    with tempfile.TemporaryDirectory(prefix="ptsip-contract-wheel-") as directory:
        temp = Path(directory)
        environment = temp / ".venv"
        venv.EnvBuilder(with_pip=True).create(environment)
        python = environment / ("Scripts/python.exe" if os.name == "nt" else "bin/python")
        env = {key: value for key, value in os.environ.items() if key not in {"PYTHONPATH", "PYTHONHOME"}}
        wheel = args.artifact.resolve()
        subprocess.run([str(python), "-m", "pip", "--disable-pip-version-check", "install", "--no-deps", str(wheel)],
                       cwd=temp, env=env, check=True, stdout=sys.stderr)
        smoke = temp / "wheel_smoke.py"
        smoke.write_bytes(Path(__file__).read_bytes())
        fixtures = Path(__file__).resolve().parents[1] / "rbac-list-conformance.json"
        (temp / fixtures.name).write_bytes(fixtures.read_bytes())
        command = [str(python), "-I", str(smoke), str(wheel), args.source_sha, "--verify"]
        subprocess.run(command, cwd=temp, env=env, check=True)


if __name__ == "__main__":
    main()
