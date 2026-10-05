from __future__ import annotations

import json
from pathlib import Path
import subprocess
import sys
import zipfile

import pytest

ROOT = Path(__file__).resolve().parents[3]


@pytest.fixture(scope="module")
def built_wheel(tmp_path_factory):
    temporary = tmp_path_factory.mktemp("vpms-contract-wheel")
    egg = temporary / "egg"
    egg.mkdir()
    result = subprocess.run([
        sys.executable, "setup.py", "egg_info", "--egg-base", str(egg),
        "build", "--build-base", str(temporary / "build"),
        "bdist_wheel", "--bdist-dir", str(temporary / "bdist"),
        "--dist-dir", str(temporary / "dist"),
    ], cwd=ROOT, capture_output=True, text=True)
    (temporary / "wheel-build.log").write_text(result.stdout + result.stderr, encoding="utf-8")
    assert result.returncode == 0, result.stdout + result.stderr
    wheels = list((temporary / "dist").glob("*.whl"))
    assert len(wheels) == 1
    return wheels[0], temporary


def test_actual_wheel_contains_exact_vpms_contracts_and_draft_support(built_wheel):
    wheel, _ = built_wheel
    source = ROOT / "src/vpms/contracts"
    with zipfile.ZipFile(wheel) as archive:
        for path in source.rglob("*.json"):
            member = "vpms/contracts/" + path.relative_to(source).as_posix()
            assert archive.read(member) == path.read_bytes()
        assert archive.read("ptsip/support/policy/SFP-0023.yaml") == (
            ROOT / "src/policy/SFP-0023.yaml"
        ).read_bytes()
        names = archive.namelist()
        assert not any(name.startswith(("developer/", "docs/Support_policy/automation/")) for name in names)
        assert not any("/MPD-" in name for name in names)


def test_installed_contracts_resolve_without_repository_or_developer_assets(built_wheel):
    wheel, temporary = built_wheel
    target = temporary / "installed"
    install = subprocess.run([
        sys.executable, "-m", "pip", "install", "--no-deps",
        "--no-compile", "--target", str(target), str(wheel),
    ], cwd=temporary, capture_output=True, text=True)
    assert install.returncode == 0, install.stdout + install.stderr
    # -I discards repository cwd/PYTHONPATH. Only the installed wheel target is added.
    code = r'''
import importlib.abc, importlib.resources, json, pathlib, sys
class NoDeveloperPolicy(importlib.abc.MetaPathFinder):
    def find_spec(self, fullname, path=None, target=None):
        if fullname == "developer" or fullname.startswith("developer."):
            raise AssertionError("installed product requested developer authority")
sys.meta_path.insert(0, NoDeveloperPolicy())
sys.path.insert(0, sys.argv[1])
import vpms
from jsonschema import Draft202012Validator
from referencing import Registry, Resource
from ptsip.governance import AuthorityCatalog
assert pathlib.Path(vpms.__file__).is_relative_to(pathlib.Path(sys.argv[1]))
base = importlib.resources.files("vpms").joinpath("contracts")
read = lambda p: json.loads(base.joinpath(p).read_text(encoding="utf-8"))
identity = read("identity-registry.json")
resources = Registry().with_resource(identity["$id"], Resource.from_contents(identity))
for path in identity["schema_resources"]:
    schema = read(path)
    Draft202012Validator.check_schema(schema)
    resources = resources.with_resource(schema["$id"], Resource.from_contents(schema))
catalog = read("index.json")
Draft202012Validator(read("schemas/catalog.schema.json"), registry=resources).validate(catalog)
for key, entry in catalog["contracts"].items():
    contract = read(entry["path"])
    Draft202012Validator(read("schemas/product-contract.schema.json"), registry=resources).validate(contract)
    assert contract["id"] == key
    assert contract["status"] == entry["status"] == "APPROVED"
    assert not contract["runtime_enabled"]
support = AuthorityCatalog(pathlib.Path(sys.argv[1]) / "not-a-repository")
support.validate_current_corpus()
_, route, contract = support.load_current_record("SFP-0023")
assert support.assets.source == "SHIPPED_PROJECTION"
assert route["status"] == contract["policy"]["status"] == "DRAFT"
assert not support.lifecycle_is_eligible(support.lifecycle_state(contract))
assert hasattr(vpms, "select_cases") and hasattr(vpms, "run_selected_cases")
assert not hasattr(vpms, "resolve_selection")
from vpms.contract_runtime import ContractUnavailable
from vpms.domain.snapshot import load_registry_snapshot
from vpms.selection import resolve_selection
from vpms.execution.composition import run_cases
for call in (
    lambda: load_registry_snapshot([], references={}),
    lambda: resolve_selection(True, {"kind": "CASE_IDS", "case_ids": ["a"]}),
    lambda: run_cases(True, True, executors={}),
):
    try:
        call()
    except ContractUnavailable as error:
        assert str(error) == "CONTRACT_NOT_ACTIVE"
    else:
        raise AssertionError("non-active product contracts granted execution")
print(json.dumps({"installed_contracts": len(catalog["contracts"]), "support_status": "DRAFT", "developer_dependency": False, "legacy_api_preserved": True, "successor_apis_installed": True, "successor_runtime_blocked": True}))
'''
    smoke = subprocess.run([
        sys.executable, "-I", "-c", code, str(target),
    ], cwd=temporary, capture_output=True, text=True)
    (temporary / "installed-smoke.log").write_text(smoke.stdout + smoke.stderr, encoding="utf-8")
    assert smoke.returncode == 0, smoke.stdout + smoke.stderr
    report = json.loads(smoke.stdout)
    assert report == {
        "installed_contracts": 3, "support_status": "DRAFT",
        "developer_dependency": False, "legacy_api_preserved": True,
        "successor_apis_installed": True, "successor_runtime_blocked": True,
    }
