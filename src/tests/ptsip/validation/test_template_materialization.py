from __future__ import annotations

import copy

import pytest

from ptsip.validation.templates import (
    PROJECT_EXPLICIT,
    PROJECT_EXTENSION,
    PROJECT_OVERRIDE,
    TEMPLATE,
    TemplateMaterializationError,
    calculate_effective_map_digest,
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


def _profile(
    mode: str,
    template_id: str,
    revision: str,
    overrides: dict[str, object] | None = None,
) -> dict[str, object]:
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


def test_template_materialization_is_deterministic_source_preserving_and_read_only() -> None:
    definition = template_catalog()[0]
    profile = _profile("template", definition.id, definition.revision)
    original = copy.deepcopy(profile)

    first = materialize_profile(profile)
    second = materialize_profile(profile)

    assert profile == original
    assert first.source_payload == original
    assert first.source_payload is not profile
    assert first.source_payload["responsibility_map"]["mode"] == "template"
    assert first.effective_payload == second.effective_payload
    assert first.effective_map_digest == second.effective_map_digest
    assert first.source_mode == "template"
    assert first.template_id == definition.id
    assert first.template_revision == definition.revision
    assert first.effective_payload["responsibility_map"] == {"mode": "explicit"}
    assert first.payload is first.effective_payload
    assert [item["id"] for item in first.effective_payload["components"]] == [
        "package",
        "package-tests",
    ]
    assert first.effective_payload["relationships"][0]["type"] == "VERIFIES"
    assert first.provenance.components == {
        "package": TEMPLATE,
        "package-tests": TEMPLATE,
    }
    assert first.provenance.relationships == {
        "package-tests-verify-package": TEMPLATE,
    }


def test_hybrid_override_replaces_stable_ids_and_tracks_origin() -> None:
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

    resolved = materialize_profile(profile)
    components = resolved.effective_payload["components"]
    assert [item["id"] for item in components] == [
        "package",
        "package-tests",
        "repository-tools",
    ]
    assert components[0]["include"] == ["lib/**"]
    assert [item["id"] for item in resolved.effective_payload["relationships"]] == [
        "package-tests-verify-package",
        "tools-verify-package",
    ]
    assert resolved.provenance.components == {
        "package": PROJECT_OVERRIDE,
        "package-tests": TEMPLATE,
        "repository-tools": PROJECT_EXTENSION,
    }
    assert resolved.provenance.relationships == {
        "package-tests-verify-package": TEMPLATE,
        "tools-verify-package": PROJECT_EXTENSION,
    }


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


def test_hybrid_removal_is_preserved_as_derived_provenance() -> None:
    definition = template_catalog()[0]
    profile = _profile(
        "hybrid",
        definition.id,
        definition.revision,
        {
            "remove_component_ids": ["package-tests"],
            "remove_relationship_ids": ["package-tests-verify-package"],
        },
    )

    resolved = materialize_profile(profile)
    assert [item["id"] for item in resolved.effective_payload["components"]] == ["package"]
    assert resolved.effective_payload["relationships"] == []
    assert resolved.provenance.removals["components"] == ("package-tests",)
    assert resolved.provenance.removals["relationships"] == (
        "package-tests-verify-package",
    )


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


def test_explicit_mode_preserves_source_and_records_project_provenance() -> None:
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
    resolved = materialize_profile(profile)
    assert resolved.source_mode == "explicit"
    assert resolved.template_id is None
    assert resolved.source_payload == profile
    assert resolved.effective_payload == profile
    assert resolved.source_payload is not profile
    assert resolved.effective_payload is not profile
    assert resolved.provenance.components == {"product": PROJECT_EXPLICIT}
    assert resolved.effective_map_digest == calculate_effective_map_digest(profile)


def test_equal_effective_maps_have_equal_digest_across_source_modes() -> None:
    definition = template_catalog()[0]
    template_profile = _profile("template", definition.id, definition.revision)
    template_resolved = materialize_profile(template_profile)

    explicit_profile = copy.deepcopy(template_resolved.effective_payload)
    explicit_resolved = materialize_profile(explicit_profile)

    assert template_resolved.source_mode == "template"
    assert explicit_resolved.source_mode == "explicit"
    assert template_resolved.template_id == definition.id
    assert explicit_resolved.template_id is None
    assert template_resolved.effective_map_digest == explicit_resolved.effective_map_digest


def test_effective_map_digest_ignores_stable_id_and_known_set_order() -> None:
    definition = template_catalog()[2]
    template_profile = _profile("template", definition.id, definition.revision)
    resolved = materialize_profile(template_profile)

    reordered = copy.deepcopy(resolved.effective_payload)
    reordered["components"].reverse()
    reordered["relationships"].reverse()
    for component in reordered["components"]:
        if component.get("id") == "development-verification":
            component["roles"].reverse()
    reordered_resolved = materialize_profile(reordered)

    assert resolved.effective_map_digest == reordered_resolved.effective_map_digest


def test_effective_map_digest_changes_when_architecture_semantics_change() -> None:
    definition = template_catalog()[0]
    resolved = materialize_profile(_profile("template", definition.id, definition.revision))

    changed = copy.deepcopy(resolved.effective_payload)
    changed["components"][0]["purpose"] = "changed_architecture_purpose"

    assert calculate_effective_map_digest(changed) != resolved.effective_map_digest


def test_digest_does_not_depend_on_specification_revision_metadata() -> None:
    definition = template_catalog()[0]
    resolved = materialize_profile(_profile("template", definition.id, definition.revision))

    changed_binding = copy.deepcopy(resolved.effective_payload)
    changed_binding["ptsip"]["specification"]["revision"] = "different-immutable-binding"

    assert calculate_effective_map_digest(changed_binding) == resolved.effective_map_digest
