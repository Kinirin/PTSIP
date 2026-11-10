"""Install a local PTSIP wheel and run its common contracts outside the checkout."""

import argparse
import hashlib
from importlib import resources
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import venv


FORMAT = "ptsip-common-contract/v1"
LIFECYCLE = "urn:ptsip:common_policy:lifecycle:v1"
IDENTITY = "urn:ptsip:common_policy:identity:v1"
URNS = {
    IDENTITY, LIFECYCLE, "urn:ptsip:common_policy:rules:v1", "urn:ptsip:common:plan:unit:v1",
    "urn:ptsip:common:plan:lifecycle:v1", "urn:ptsip:common:plan:identity:v1",
}
STATUSES = [
    "DRAFT.Default", "DRAFT.Reviewed", "DRAFT.Denied", "DRAFT.Pass", "APPROVED.Default",
    "APPROVED.Applying", "APPROVED.Application_Problems", "APPROVED.Unresolved", "APPROVED.Activate",
    "APPROVED.provisional", "RETIRED.Default", "RETIRED.Removable", "RETIRED.Waiting",
]


def verify_rbac_profile_structures(package):
    root = package.joinpath("compile_sealed_rbac")
    profile_contract = json.loads(root.joinpath("pp.3.00.contract.json").read_bytes())
    assert profile_contract["profile_version"] == "pp.3.00"
    assert profile_contract["scope"] == "STRUCTURE_ONLY"
    assert profile_contract["activation_state"] == "UNREGISTERED"
    assert profile_contract["authorization_permitted"] is False
    cases = json.loads(Path(__file__).with_name("rbac-pp300-conformance.json").read_bytes())
    validation = package.joinpath("engines/validation")
    manifest = json.loads(validation.joinpath("manifest.json").read_bytes())
    contract = json.loads(validation.joinpath("contract.json").read_bytes())
    binary = Path(sys.executable).with_name("ptsip-engine-validation" + (".exe" if os.name == "nt" else ""))
    schemas = {}
    for entry in profile_contract["schemas"].values():
        schema = json.loads(root.joinpath(entry["resource"]).read_bytes())
        assert schema["$id"] == entry["id"]
        schemas[entry["resource"]] = schema
    assert schemas.keys() == {case["schema"] for case in cases}
    for case in cases:
        request = {
            "engine_id": manifest["engine_id"],
            "engine_version": manifest["implementation_version"],
            "contract_id": contract["contract_id"],
            "contract_version": contract["version"],
            "operation_id": "validation.schema",
            "input": {"schema": schemas[case["schema"]], "instance": case["instance"]},
        }
        result = subprocess.run([str(binary)], input=json.dumps(request).encode("utf-8"),
                                capture_output=True, check=True, timeout=20)
        report = json.loads(result.stdout)
        assert report["output"]["valid"] is case["valid"], (case["name"], report)
        assert report["execution_authorized"] is False
    return len(cases)


def verify(expected_sha):
    package = resources.files("ptsip")
    root = package.joinpath("common_contract")
    assert Path(str(root)).is_relative_to(Path(sys.prefix))
    assert not package.joinpath("policy_contracts").is_dir()
    assert not root.joinpath("go.mod").is_file()
    assert not root.joinpath("source/verification.go").is_file()
    origin = json.loads(root.joinpath("origin.json").read_bytes())
    assert origin["format"] == "ptsip-common-contract-wheel-origin/v1"
    assert origin["source_sha"] == expected_sha
    binary = root.joinpath(origin["binary"])
    assert hashlib.sha256(binary.read_bytes()).hexdigest() == origin["binary_sha256"]
    manifest_path = root.joinpath("registry/manifest.json")
    manifest = manifest_path.read_bytes()
    assert hashlib.sha256(manifest).hexdigest() == origin["registry_sha256"]
    entries = json.loads(manifest)["contracts"]
    assert len(entries) == len(URNS) and {entry["urn"] for entry in entries} == URNS
    assert len([p for p in root.joinpath("schemas").iterdir() if p.is_file()]) == len(URNS)

    def request(operation, urn, **values):
        payload = json.dumps({"format": FORMAT, "operation": operation, "urn": urn, **values}).encode("ascii")
        result = subprocess.run([str(binary)], input=payload, capture_output=True, check=True, timeout=20)
        return json.loads(result.stdout)

    for entry in entries:
        data = package.joinpath(entry["resource"]).read_bytes()
        assert hashlib.sha256(data).hexdigest() == entry["sha256"]
        lookup = request("lookup", entry["urn"])
        assert lookup["status"] == "RESOLVED", lookup
        assert lookup["schema"]["$id"] == entry["urn"]
        assert lookup["provenance"]["source_revision"] == expected_sha
        assert lookup["provenance"]["registry_sha256"] == origin["registry_sha256"]
    for status in STATUSES:
        report = request("validate", LIFECYCLE, instance={"status": status, "extra": {}})
        assert report["status"] == "VALID", report
    assert request("validate", LIFECYCLE, instance={"status": "APPROVED"})["status"] == "INVALID"
    missing = root.joinpath("schemas/policy_lifecycle.schema.json")
    missing.write_bytes(b"{}")
    stale = request("lookup", LIFECYCLE)
    assert stale["status"] == "UNRESOLVED" and "STALE_SCHEMA" in stale["reference_resolution"]["problem"]
    missing.unlink()
    assert request("lookup", LIFECYCLE)["status"] == "UNRESOLVED"
    assert request("lookup", IDENTITY)["status"] == "RESOLVED"
    manifest_path.write_bytes(manifest + b"\n")
    stale = request("lookup", IDENTITY)
    assert stale["status"] == "UNRESOLVED" and "STALE_REGISTRY" in stale["reference_resolution"]["problem"]
    rbac_cases = verify_rbac_profile_structures(package)
    print(json.dumps({"status": "PASS", "source_sha": expected_sha, "platform": origin["goos"],
                      "architecture": origin["goarch"], "schemas": len(entries), "statuses": len(STATUSES),
                      "rbac_structural_cases": rbac_cases,
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
        fixtures = Path(__file__).resolve().parents[3] / "ptsip/compile_sealed_rbac/testdata/conformance.json"
        (temp / "rbac-pp300-conformance.json").write_bytes(fixtures.read_bytes())
        command = [str(python), "-I", str(smoke), str(wheel), args.source_sha, "--verify"]
        subprocess.run(command, cwd=temp, env=env, check=True)


if __name__ == "__main__":
    main()
