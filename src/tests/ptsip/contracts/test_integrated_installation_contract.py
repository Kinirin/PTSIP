"""Contract guards for the approved integrated native PTSIP distribution route.

These tests prove source declarations and fail-closed boundaries, not native
wheel composition, installation execution, release admission or publication.
"""
from __future__ import annotations

import json
import tomllib
from pathlib import Path

_ROOT = Path(__file__).resolve().parents[4]


def _json(path: str) -> dict:
    return json.loads((_ROOT / path).read_text(encoding="utf-8"))


def test_integrated_delivery_preserves_seven_independent_engine_version_identities() -> None:
    plan = _json("src/ptsip/engines/contracts/installation-plan.json")
    registry = _json("src/ptsip/engines/index.json")
    components = plan["components"]
    entries = registry["official_engine_entries"]
    assert plan["schema_version"] == "ptsip-engine-installation-plan/v2"
    assert plan["selection"] == "REGISTERED_NATIVE_COMPONENTS_FOR_ONE_PTSIP_PLATFORM_WHEEL"
    assert plan["default_installation"] == (
        "ONE_COMPLETE_PTSIP_PLATFORM_WHEEL_WITH_ALL_SEVEN_ENGINES"
    )
    assert len(components) == len(entries) == 7
    assert set(plan["default_available_engine_ids"]) == {
        item["engine_id"] for item in entries
    }
    by_id = {row["engine_id"]: row for row in components}
    assert len(by_id) == 7
    for entry in entries:
        item = by_id[entry["engine_id"]]
        assert item["version"] == entry["version"]
        assert item["distribution"] == "PTSIP_PLATFORM_WHEEL_NATIVE_COMPONENT"
        assert item["installation"] == "INSTALLED_AS_PART_OF_PTSIP_PLATFORM_WHEEL"
        assert item["permission_from_availability"] is False
        assert item["installed_is_execution_authorization"] is False
        assert "package_distribution" not in item
        assert item["entrypoint"] == "ptsip-engine-" + entry["kind"].lower()


def test_current_pyproject_no_longer_installs_engine_pip_distributions() -> None:
    with (_ROOT / "pyproject.toml").open("rb") as stream:
        cfg = tomllib.load(stream)
    dependencies = cfg["project"]["dependencies"]
    assert not any(name.lower().startswith("ptsip-engine-") for name in dependencies)
    assert cfg["project"]["version"] == "0.3.8.1b1"
    assert cfg["tool"]["ptsip"]["release"]["profile_version"] == "pp.1.02"
    declared = cfg["tool"]["ptsip"]["release"]["engines"]
    assert len(declared) == 7
    assert set(declared) == {
        "context", "storage", "validation", "discovery",
        "planning", "execution", "projection"
    }


def test_consumer_delivery_must_not_reactivate_retired_controller_or_release() -> None:
    plan = _json("src/ptsip/engines/contracts/installation-plan.json")
    distribution = plan["distribution"]
    assert distribution["public_python_project"] == "PTSIP"
    assert distribution["model"] == "SINGLE_PLATFORM_SPECIFIC_PTSIP_WHEEL"
    assert distribution["required_verified_engines"] == 7
    assert distribution["engine_pip_dependencies"] == "FORBIDDEN"
    assert distribution["verification_owner"] == ".ptsip/engine/developer/Distribution/release/"
    assert distribution["legacy_controller_execution"] == "FORBIDDEN"
    assert distribution["publication_authorized"] is False

    policy = _json("src/policy/CNTR/SFP-CNTR-0005.yaml")
    approved = policy["authority_semantics"]["unit_ptsip_engine_draft"]["approved_decisions"]
    assert approved["official_distribution"] == (
        "INTEGRATED_USER_INSTALL_PACKAGE_WITH_INDEPENDENT_COMPONENTS"
    )
    assert approved["official_consumer_package_delivery"]["public_distribution"] == "PTSIP"
    assert approved["official_consumer_package_delivery"]["public_release_execution"] == (
        "UNCHANGED_EXISTING_ADMISSION_GATE"
    )
    assert policy["authority_semantics"]["unit_ptsip_engine_draft"]["invariants"][
        "official_engine_distribution_forbids_bundled_package_delivery"
    ] is False
    admission = _json(".ptsip/contracts/release/admission.json")
    assert admission["publication_authorized"] is False


def test_new_installation_candidate_not_misrepresented_as_registered_operation() -> None:
    rust = (_ROOT / "src/ptsip/engines/planning/src/installation.rs").read_text(encoding="utf-8")
    assert "pub fn select_integrated_candidate(input: &Value)" in rust
    assert "pub fn select(_input: &Value)" in rust
    assert "integrated planning operation not yet version-registered" in rust
    assert "wheel_composition_verified" in rust
    assert '"ready":false' in rust
    assert '"publication_authorized":false' in rust


def test_native_intermediate_builder_is_not_a_consumer_distribution_resolver() -> None:
    producer = (_ROOT / (
        ".ptsip/engine/developer/Distribution/release/verification/engine/wheel.go"
    )).read_text(encoding="utf-8")
    native_test = (_ROOT / (
        "developer/tests/engines/independent_engine_wheel_test.go"
    )).read_text(encoding="utf-8")
    for retired_assumption in (
        'json:"package_distribution"',
        'json:"packages"',
        'AUTO_INSTALL_AS_PTSIP_PACKAGE_DEPENDENCY',
        'INDEPENDENT_VERSIONED_NATIVE_WHEEL',
        'plan.Distribution',
        'plan.DefaultInstallation',
    ):
        assert retired_assumption not in producer

    assert 'plan.Components' in producer
    assert 'component.Entrypoint != entrypoint' in producer
    assert 'component.Version != selected.Version' in producer
    assert 'component.Authorized == nil || *component.Authorized' in producer
    assert 'metadata := fmt.Sprintf' in producer

    assert '"pip", "install"' not in native_test
    assert 'probeNativeComponentBinary(t, wheel, expected)' in native_test
    assert 'exec.Command(binary, "--manifest")' in native_test

    # Integration still owns the full consumer installation and hashes.
    consumer_join = (_ROOT / (
        ".ptsip/engine/developer/Distribution/release/test/package_install.go"
    )).read_text(encoding="utf-8")
    assert '"pip", "install"' in consumer_join
    assert '"SINGLE_PROJECT_PLATFORM_BUNDLES_ONLY"' in (
        (_ROOT / "src/ptsip/engine_installation.py").read_text(encoding="utf-8")
    )
