from __future__ import annotations

import copy

import pytest

from ptsip.validation.templates import (
    TemplateMaterializationError,
    calculate_template_revision,
    materialize_profile,
    resolve_template,
    template_catalog,
)


POLICIES = {
    "product_to_nonproduct_runtime_dependency": "deny",
    "nonproduct_in_product_package": "deny",
    "independent_build_resolution": "required",
}


def _profile(mode: str, template_id: str, revision: str, overrides: dict[str, object] | None = None) -> dict[str, object]:
    responsibility_map: dict[str, object] = {
        "mode": mode,
        "template": {"id": template_id, "revision": revision},
    }
    if overrides is not None:
        responsibility_map["overrides"] = overrides
    return {
        "ptsip": {
            "version": "0.3.6-draft",
            "specification": {"source": "https://github.com/Kinirin/PTSIP"},
        },
        "responsibility_map": responsibility_map,
        "policies": copy.deepcopy(POLICIES),
    }


def test_wu04_catalog_has_three_stable_initial_templates() -> None:
    catalog = template_catalog()
    assert [item.id for item in catalog] == [
        "python-package-library",
        "python-cli-application",
        "mixed-product-development-delivery",
    ]
    assert all(item.revision.startswith("sha256:") for item in catalog)
    assert all(item.revision == calculate_template_revision(item.map_payload) for item in catalog)


def test_template_revision_changes_when_semantics_change() -> None:
    definition = resolve_template(
        {
            "id": "python-package-library",
            "revision": "sha256:409acd1cd9907a60761a3cf26a051185d40b5e926e6952131b641b10bccc5c9b",
        }
    )
    changed = copy.deepcopy(definition.map_payload)
    changed["components"][0]["purpose"] = "different-purpose"
    assert calculate_template_revision(changed) != definition.revision


def test_template_materialization_is_deterministic_and_read_only() -> None:
    definition = template_catalog()[0]
    profile = _profile("template", definition.id, definition.revision)
    original = copy.deepcopy(profile)

    first = materialize_profile(profile)
    second = materialize_profile(profile)

    assert profile == original
    assert first.payload == second.payload
    assert first.source_mode == "template"
    assert first.template_id == definition.id
    assert first.template_revision == definition.revision
    assert first.payload["responsibility_map"] == {"mode": "explicit"}
    assert [item["id"] for item in first.payload["components"]] == ["package", "package-tests"]
    assert first.payload["relationships"][0]["type"] == "VERIFIES"


def test_hybrid_override_replaces_stable_ids_in_place_and_appends_extensions() -> None:
    definition = template_catalog()[0]
    profile = _profile(
        "hybrid",
        definition.id,
        definition.revision,
        {
            "components": [
                {
                    "id": "package",
                    "classification": "PRODUCT",
                    "roles": ["IMPLEMENTATION"],
                    "include": ["lib/**"],
                    "purpose": "custom_package_layout",
                    "shipped": True,
                    "runtime_required": True,
                    "executable": True,
                },
                {
                    "id": "repository-tools",
                    "classification": "DEVELOPMENT_TOOLING",
                    "roles": ["AUTOMATION"],
                    "include": ["tools/**"],
                    "purpose": "repository_automation",
                    "shipped": False,
                    "runtime_required": False,
                    "executable": True,
                },
            ],
            "relationships": [
                {
                    "id": "tools-verify-package",
                    "from": "repository-tools",
                    "to": "package",
                    "type": "VERIFIES",
                }
            ],
        },
    )

    materialized = materialize_profile(profile)
    components = materialized.payload["components"]
    assert [item["id"] for item in components] == ["package", "package-tests", "repository-tools"]
    assert components[0]["include"] == ["lib/**"]
    assert [item["id"] for item in materialized.payload["relationships"]] == [
        "package-tests-verify-package",
        "tools-verify-package",
    ]


def test_hybrid_removal_requires_known_template_id() -> None:
    definition = template_catalog()[0]
    profile = _profile(
        "hybrid",
        definition.id,
        definition.revision,
        {"remove_component_ids": ["missing-component"]},
    )
    with pytest.raises(TemplateMaterializationError, match="unknown component"):
        materialize_profile(profile)


def test_hybrid_cannot_replace_and_remove_same_id() -> None:
    definition = template_catalog()[0]
    profile = _profile(
        "hybrid",
        definition.id,
        definition.revision,
        {
            "components": [
                {
                    "id": "package-tests",
                    "classification": "DEVELOPMENT_TOOLING",
                    "roles": ["VERIFICATION"],
                    "include": ["qa/**"],
                    "purpose": "shared_verification",
                    "shipped": False,
                    "runtime_required": False,
                    "executable": True,
                }
            ],
            "remove_component_ids": ["package-tests"],
        },
    )
    with pytest.raises(TemplateMaterializationError, match="replace and remove"):
        materialize_profile(profile)


def test_unknown_template_revision_fails_closed() -> None:
    definition = template_catalog()[0]
    profile = _profile("template", definition.id, "sha256:" + "0" * 64)
    with pytest.raises(TemplateMaterializationError, match="Unknown revision"):
        materialize_profile(profile)


def test_unknown_template_id_fails_closed() -> None:
    profile = _profile("template", "not-a-template", "sha256:" + "0" * 64)
    with pytest.raises(TemplateMaterializationError, match="Unknown PTSIP template id"):
        materialize_profile(profile)


def test_explicit_mode_is_pass_through_copy() -> None:
    profile = {
        "ptsip": {
            "version": "0.3.6-draft",
            "specification": {"source": "https://github.com/Kinirin/PTSIP"},
        },
        "responsibility_map": {"mode": "explicit"},
        "components": [
            {
                "id": "product",
                "classification": "PRODUCT",
                "include": ["src/**"],
                "purpose": "product",
            }
        ],
        "policies": copy.deepcopy(POLICIES),
    }
    materialized = materialize_profile(profile)
    assert materialized.source_mode == "explicit"
    assert materialized.template_id is None
    assert materialized.payload == profile
    assert materialized.payload is not profile
