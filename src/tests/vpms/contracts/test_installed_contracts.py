from __future__ import annotations

import json
from pathlib import Path
import subprocess
import sys
import zipfile
import yaml

import pytest

ROOT = Path(__file__).resolve().parents[4]


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


def test_actual_wheel_contains_exact_active_contracts_and_no_retired_selector(built_wheel):
    wheel, _ = built_wheel
    source = ROOT / "src/vpms/contracts"
    with zipfile.ZipFile(wheel) as archive:
        for path in source.rglob("*.json"):
            member = "vpms/contracts/" + path.relative_to(source).as_posix()
            assert archive.read(member) == path.read_bytes()
        index = yaml.safe_load((ROOT / "src/policy/index.yaml").read_text(encoding="utf-8"))
        for entry in index["policies"]:
            assert archive.read("ptsip/support/policy/" + entry["path"]) == (ROOT / "src/policy" / entry["path"]).read_bytes()
        assert archive.read("ptsip/support/registries/root-family-migration.json") == (ROOT / "src/policy/registries/root-family-migration.json").read_bytes()
        assert archive.read("ptsip/support/registries/root-family-projection.module.json") == (ROOT / "src/policy/registries/root-family-projection.module.json").read_bytes()
        assert archive.read("ptsip/support/schemas/root-family-projection-module.schema.json") == (ROOT / "src/policy/schemas/root-family-projection-module.schema.json").read_bytes()
        names = archive.namelist()
        assert "vpms/domain/selector.py" not in names
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
    assert contract["status"] == entry["status"] == "ACTIVE"
    assert contract["runtime_enabled"]
support = AuthorityCatalog(pathlib.Path(sys.argv[1]) / "not-a-repository")
support.validate_current_corpus()
_, route, contract = support.load_current_record("SFP-0023")
assert support.assets.source == "SHIPPED_PROJECTION"
assert route["status"] == contract["policy"]["status"] == "ACTIVE"
assert support.lifecycle_is_eligible(support.lifecycle_state(contract))
_, route, legacy = support.load_current_record("SFP-0006")
assert route["status"] == legacy["policy"]["status"] == "RETIRED"
assert not support.lifecycle_is_eligible(support.lifecycle_state(legacy))
assert all(not hasattr(vpms, name) for name in ("SelectionScope", "select_cases", "run_selected_cases"))
from vpms.domain.snapshot import load_registry_snapshot
from vpms.selection import resolve_selection
from vpms.execution.composition import run_cases
assert vpms.load_registry_snapshot is load_registry_snapshot
assert vpms.resolve_selection is resolve_selection
assert vpms.run_cases is run_cases
references = {"targets": ["t"], "formulas": ["f"], "variables": ["v"], "policies": ["p"], "runners": ["r"]}
raw = [{"id": "a", "purpose": "PRODUCT", "target": "t", "formula": "f", "variables": "v", "policy": "p", "runner": "r"}]
loaded = load_registry_snapshot(raw, references=references)
assert loaded.ok
selection = resolve_selection(loaded.snapshot, {"kind": "CASE_IDS", "case_ids": ["a"]})
assert selection.ok
calls = []
class Adapter:
    def execute(self, case):
        calls.append(case.id)
        return vpms.RunnerExecution(vpms.VerificationOutcome.PASS)
results = run_cases(loaded.snapshot, selection, executors={"r": Adapter()})
assert calls == ["a"] and results[0].outcome == vpms.VerificationOutcome.PASS
assert resolve_selection(loaded.snapshot, {"kind": "CASE_IDS", "case_ids": ["missing"]}).state == "REJECTED"
from ptsip.validation.handoff import load_validated_effective_map
assert callable(load_validated_effective_map)
from vpms.integration.ptsip_bridge import metadata_from_effective_map, PtsipMetadataError
import subprocess, yaml
from ptsip.profiles.metadata import current_project_profile_ptsip_metadata
from ptsip.validation.templates import template_catalog
consumer = pathlib.Path(sys.argv[1]).parent / "consumer"
consumer.mkdir()
for relative, content in {"src/package.py": "VALUE = 1\n", "tests/test_package.py": "def test_value(): pass\n"}.items():
    path = consumer / relative
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")
for args in (["init"], ["config", "user.email", "test@example.invalid"], ["config", "user.name", "Test"], ["add", "."], ["commit", "-m", "fixture"]):
    subprocess.run(["git", "-C", str(consumer), *args], check=True, capture_output=True)
template = template_catalog()[0]
payload = {"ptsip": current_project_profile_ptsip_metadata(),
    "responsibility_map": {"mode": "template", "template": {"id": template.id, "revision": template.revision}},
    "policies": {"product_to_nonproduct_runtime_dependency": "deny", "nonproduct_in_product_package": "deny", "independent_build_resolution": "required"}}
profile = consumer / "ptsip.yaml"
profile.write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8")
before = profile.read_bytes()
handoff = load_validated_effective_map(consumer)
projection = metadata_from_effective_map(handoff)
assert projection.get_target("package").classification == "PRODUCT"
assert profile.read_bytes() == before
try:
    metadata_from_effective_map({"validated": True, "components": []})
except PtsipMetadataError as error:
    assert "UNVALIDATED_EFFECTIVE_MAP" in str(error)
else:
    raise AssertionError("boolean assertion bypassed installed provider")
print(json.dumps({"installed_contracts": len(catalog["contracts"]), "support_status": "ACTIVE", "developer_dependency": False, "legacy_api_retired": True, "successor_apis_installed": True, "successor_runtime_executed": True}))
'''
    smoke = subprocess.run([
        sys.executable, "-I", "-c", code, str(target),
    ], cwd=temporary, capture_output=True, text=True)
    (temporary / "installed-smoke.log").write_text(smoke.stdout + smoke.stderr, encoding="utf-8")
    assert smoke.returncode == 0, smoke.stdout + smoke.stderr
    report = json.loads(smoke.stdout)
    assert report == {
        "installed_contracts": 3, "support_status": "ACTIVE",
        "developer_dependency": False, "legacy_api_retired": True,
        "successor_apis_installed": True, "successor_runtime_executed": True,
    }
    independent = subprocess.run([sys.executable, "-I", "-c", r'''
import importlib.abc, pathlib, sys
sys.path.insert(0, sys.argv[1])
class NoVPMS(importlib.abc.MetaPathFinder):
    def find_spec(self, fullname, path=None, target=None):
        if fullname == "vpms" or fullname.startswith("vpms."):
            raise AssertionError("PTSIP core imported optional VPMS")
sys.meta_path.insert(0, NoVPMS())
from ptsip.validation.profile import validate_profile
from ptsip.validation.handoff import load_validated_effective_map
from ptsip.governance import AuthorityCatalog
AuthorityCatalog(pathlib.Path(sys.argv[1]) / "not-a-repository").validate_current_corpus()
print("PTSIP_CORE_INDEPENDENT")
''', str(target)], cwd=temporary, capture_output=True, text=True)
    assert independent.returncode == 0, independent.stdout + independent.stderr
