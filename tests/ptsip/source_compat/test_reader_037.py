from __future__ import annotations

import hashlib
from pathlib import Path

import yaml

from ptsip.repository.profile_transition import DraftVersion, ProfileGenerationIdentity
from ptsip.source_compat import (
    SourceDeclarationScope,
    SourceFamily,
    V034_REVISION,
    V036_REVISION,
    read_source_profile,
    supported_source_families,
    validate_source_read_binding,
)


def _write_profile(root: Path, payload: dict[str, object], *, path: str = "ptsip.yaml") -> ProfileGenerationIdentity:
    raw = yaml.safe_dump(payload, sort_keys=False).encode()
    (root / path).write_bytes(raw)
    version = str(payload["ptsip"]["version"])
    semantic = version.removesuffix("-draft")
    major, minor, micro = map(int, semantic.split("."))
    return ProfileGenerationIdentity(
        path=path,
        version=DraftVersion(major, minor, micro),
        declared_version=version,
        specification_revision=str(payload["ptsip"]["specification"]["revision"]),
        specification_source=str(payload["ptsip"]["specification"]["source"]),
        content_sha256=hashlib.sha256(raw).hexdigest(),
        temporary=path != "ptsip.yaml",
    )


def _profile_034() -> dict[str, object]:
    return {
        "ptsip": {
            "version": "0.3.4-draft",
            "specification": {
                "source": "https://github.com/Kinirin/PTSIP",
                "revision": V034_REVISION,
            },
        },
        "components": [
            {
                "id": "tests",
                "classification": "TOOLCHAIN",
                "include": ["tests/**"],
                "purpose": "verification",
                "shipped": False,
                "runtime_required": False,
                "lifecycle_owner": "DEVELOPMENT_TOOLING",
                "executable": True,
                "analysis_inputs": ["src/**"],
            }
        ],
        "policies": {
            "product_to_toolchain_runtime_dependency": "deny",
            "toolchain_in_product_package": "deny",
            "independent_build_resolution": "required",
            "shared_executable_cross_boundary": "deny",
            "neutral_contract_sharing": "allow",
        },
    }


def _profile_036() -> dict[str, object]:
    return {
        "ptsip": {
            "version": "0.3.6-draft",
            "specification": {
                "source": "https://github.com/Kinirin/PTSIP",
                "revision": V036_REVISION,
            },
        },
        "responsibility_map": {"mode": "explicit"},
        "components": [
            {
                "id": "delivery",
                "classification": "DELIVERY",
                "roles": ["AUTOMATION"],
                "include": ["release/**"],
                "purpose": "release",
                "shipped": False,
                "runtime_required": False,
            }
        ],
        "associated_artifacts": [
            {"id": "gov", "anchor": "delivery", "include": ["docs/**"], "purpose": "docs"}
        ],
        "relationships": [
            {"id": "r1", "from": "gov", "to": "delivery", "type": "DOCUMENTS"}
        ],
        "policies": {
            "product_to_nonproduct_runtime_dependency": "deny",
            "nonproduct_in_product_package": "deny",
            "independent_build_resolution": "required",
            "shared_executable_cross_lifecycle": "deny",
            "neutral_contract_sharing": "allow",
        },
    }


def test_supported_families_are_revision_frozen() -> None:
    rows = supported_source_families()
    assert [(version, revision, family.value) for version, revision, family in rows] == [
        ("0.3.4-draft", V034_REVISION, SourceFamily.TOOL_035_PROFILE.value),
        ("0.3.6-draft", V036_REVISION, SourceFamily.TOOL_036_PROFILE.value),
    ]


def test_034_preserves_toolchain_and_family_attributes(tmp_path: Path) -> None:
    payload = _profile_034()
    result = read_source_profile(tmp_path, _write_profile(tmp_path, payload))
    assert result.complete and result.profile is not None
    assert result.profile.family is SourceFamily.TOOL_035_PROFILE
    assert result.profile.components[0].source_classification == "TOOLCHAIN"
    attributes = {item.name: item.as_dict()["value"] for item in result.profile.components[0].attributes}
    assert attributes["lifecycle_owner"] == "DEVELOPMENT_TOOLING"
    assert attributes["analysis_inputs"] == ["src/**"]
    assert result.profile.as_dict()["authority"] == "SOURCE_DECLARATION_ONLY"


def test_034_boundary_form_is_preserved_without_component_inference(tmp_path: Path) -> None:
    payload = _profile_034()
    payload.pop("components")
    payload["boundaries"] = {
        "product": {"roots": ["src"]},
        "toolchain": {"roots": ["tests"]},
        "neutral_contract": {"roots": ["spec"]},
    }
    result = read_source_profile(tmp_path, _write_profile(tmp_path, payload))
    assert result.complete and result.profile is not None
    assert result.profile.components == ()
    semantics = result.profile.family_semantics
    assert semantics.declaration_form == "BOUNDARIES"
    assert [item.source_classification for item in semantics.boundaries] == [
        "PRODUCT",
        "TOOLCHAIN",
        "NEUTRAL_CONTRACT",
    ]


def test_036_explicit_preserves_roles_relationships_and_artifacts(tmp_path: Path) -> None:
    payload = _profile_036()
    result = read_source_profile(tmp_path, _write_profile(tmp_path, payload))
    assert result.complete and result.profile is not None
    assert result.profile.family is SourceFamily.TOOL_036_PROFILE
    assert result.profile.components[0].source_classification == "DELIVERY"
    attributes = {item.name: item.as_dict()["value"] for item in result.profile.components[0].attributes}
    assert attributes["roles"] == ["AUTOMATION"]
    assert result.profile.relationships[0].relationship_type == "DOCUMENTS"
    assert result.profile.associated_artifacts[0].anchor == "delivery"


def test_036_hybrid_preserves_override_scope_and_removals(tmp_path: Path) -> None:
    payload = _profile_036()
    payload.pop("components")
    payload.pop("associated_artifacts")
    payload.pop("relationships")
    payload["responsibility_map"] = {
        "mode": "hybrid",
        "template": {"id": "base", "revision": "abc"},
        "overrides": {
            "components": [
                {
                    "id": "ops",
                    "classification": "OPERATIONS",
                    "include": ["ops/**"],
                    "purpose": "ops",
                    "shipped": False,
                    "runtime_required": False,
                }
            ],
            "remove_component_ids": ["old"],
            "remove_relationship_ids": ["r-old"],
        },
    }
    result = read_source_profile(tmp_path, _write_profile(tmp_path, payload))
    assert result.complete and result.profile is not None
    assert result.profile.components[0].scope is SourceDeclarationScope.RESPONSIBILITY_MAP_OVERRIDE
    semantics = result.profile.family_semantics
    assert semantics.responsibility_map_mode == "hybrid"
    assert semantics.template_id == "base"
    assert semantics.template_revision == "abc"
    assert semantics.remove_component_ids == ("old",)
    assert semantics.remove_relationship_ids == ("r-old",)


def test_036_template_does_not_materialize_current_template(tmp_path: Path) -> None:
    payload = _profile_036()
    payload.pop("components")
    payload.pop("associated_artifacts")
    payload.pop("relationships")
    payload["responsibility_map"] = {
        "mode": "template",
        "template": {"id": "base", "revision": "abc"},
    }
    result = read_source_profile(tmp_path, _write_profile(tmp_path, payload))
    assert result.complete and result.profile is not None
    assert result.profile.components == ()
    assert result.profile.relationships == ()
    assert result.profile.family_semantics.template_id == "base"


def test_same_version_unknown_revision_fails_closed(tmp_path: Path) -> None:
    payload = _profile_036()
    payload["ptsip"]["specification"]["revision"] = "other"
    result = read_source_profile(tmp_path, _write_profile(tmp_path, payload))
    assert not result.complete
    assert result.issues[0].code == "UNSUPPORTED_SOURCE_REVISION"


def test_unsupported_family_fails_closed(tmp_path: Path) -> None:
    payload = _profile_036()
    payload["ptsip"]["version"] = "0.3.5-draft"
    result = read_source_profile(tmp_path, _write_profile(tmp_path, payload))
    assert not result.complete
    assert result.issues[0].code == "UNSUPPORTED_SOURCE_FAMILY"


def test_generation_identity_mismatch_fails_before_projection(tmp_path: Path) -> None:
    payload = _profile_036()
    generation = _write_profile(tmp_path, payload)
    mismatched = ProfileGenerationIdentity(
        generation.path,
        generation.version,
        "0.3.4-draft",
        generation.specification_revision,
        generation.specification_source,
        generation.content_sha256,
        generation.temporary,
    )
    result = read_source_profile(tmp_path, mismatched)
    assert not result.complete
    assert result.issues[0].code == "SOURCE_VERSION_MISMATCH"


def test_schema_rejects_toolchain_as_036_classification(tmp_path: Path) -> None:
    payload = _profile_036()
    payload["components"][0]["classification"] = "TOOLCHAIN"
    result = read_source_profile(tmp_path, _write_profile(tmp_path, payload))
    assert not result.complete
    assert any(item.code == "SOURCE_SCHEMA_ERROR" for item in result.issues)


def test_binding_detects_source_change_after_read(tmp_path: Path) -> None:
    payload = _profile_034()
    result = read_source_profile(tmp_path, _write_profile(tmp_path, payload))
    assert result.complete
    path = tmp_path / "ptsip.yaml"
    path.write_text(path.read_text() + "\n# changed\n")
    issues = validate_source_read_binding(tmp_path, result)
    assert issues and issues[0].code == "SOURCE_CONTENT_STALE"


def test_read_is_deterministic_and_does_not_mutate_source(tmp_path: Path) -> None:
    payload = _profile_036()
    generation = _write_profile(tmp_path, payload)
    path = tmp_path / "ptsip.yaml"
    before = path.read_bytes()
    first = read_source_profile(tmp_path, generation)
    second = read_source_profile(tmp_path, generation)
    assert first.as_dict() == second.as_dict()
    assert path.read_bytes() == before
    assert first.profile is not None
    assert first.profile.as_dict()["raw_payload"] == payload
