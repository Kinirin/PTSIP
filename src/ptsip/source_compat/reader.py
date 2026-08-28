from __future__ import annotations

import hashlib
import json
from importlib.resources import files
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator

from ..repository.profile_path import normalize_profile_path, profile_path_on_disk
from ..repository.profile_transition import ProfileGenerationIdentity
from .model import (
    CompatibilitySourceProfile,
    SourceAssociatedArtifact,
    SourceAttribute,
    SourceBoundary,
    SourceComponent,
    SourceDeclarationScope,
    SourceFamily,
    SourceGenerationBinding,
    SourceLocation,
    SourcePolicy,
    SourceReadIssue,
    SourceReadResult,
    SourceRelationship,
    V034SourceSemantics,
    V036SourceSemantics,
    freeze_json,
)

SPEC_SOURCE = "https://github.com/Kinirin/PTSIP"
V034_REVISION = "b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e"
V036_REVISION = "d6995ed232e845b88d8235b851e80ab54b7804ea"

_SUPPORTED = {
    ("0.3.4-draft", V034_REVISION): SourceFamily.TOOL_035_PROFILE,
    ("0.3.6-draft", V036_REVISION): SourceFamily.TOOL_036_PROFILE,
}

_SCHEMA_ASSET = {
    SourceFamily.TOOL_035_PROFILE: "specdata/ptsip-source-profile-0.3.4.schema.json",
    SourceFamily.TOOL_036_PROFILE: "specdata/ptsip-source-profile-0.3.6.schema.json",
}

_COMMON_COMPONENT_FIELDS = {"id", "classification", "include", "exclude", "purpose"}


def supported_source_families() -> tuple[tuple[str, str, SourceFamily], ...]:
    return tuple((version, revision, family) for (version, revision), family in sorted(_SUPPORTED.items()))


def _issue(code: str, message: str, generation: ProfileGenerationIdentity, pointer: str | None = None) -> SourceReadIssue:
    return SourceReadIssue(code, message, generation.path, pointer)


def _schema(family: SourceFamily) -> dict[str, object]:
    path = files("ptsip").joinpath(_SCHEMA_ASSET[family])
    return json.loads(path.read_text(encoding="utf-8"))


def _schema_issues(payload: dict[str, object], family: SourceFamily, generation: ProfileGenerationIdentity) -> list[SourceReadIssue]:
    validator = Draft202012Validator(_schema(family))
    result: list[SourceReadIssue] = []
    for error in sorted(validator.iter_errors(payload), key=lambda item: list(item.absolute_path)):
        pointer = "/" + "/".join(str(part).replace("~", "~0").replace("/", "~1") for part in error.absolute_path)
        if pointer == "/":
            pointer = ""
        result.append(_issue("SOURCE_SCHEMA_ERROR", error.message, generation, pointer))
    return result


def _binding(generation: ProfileGenerationIdentity) -> SourceGenerationBinding:
    return SourceGenerationBinding(
        profile_path=generation.path,
        declared_version=generation.declared_version,
        specification_revision=generation.specification_revision,
        specification_source=generation.specification_source,
        content_sha256=generation.content_sha256,
        temporary=generation.temporary,
    )


def _location(generation: ProfileGenerationIdentity, pointer: str) -> SourceLocation:
    return SourceLocation(generation.path, pointer)


def _attributes(item: dict[str, object]) -> tuple[SourceAttribute, ...]:
    return tuple(
        SourceAttribute(name, freeze_json(value))
        for name, value in sorted(item.items())
        if name not in _COMMON_COMPONENT_FIELDS
    )


def _component(item: dict[str, object], generation: ProfileGenerationIdentity, pointer: str, scope: SourceDeclarationScope) -> SourceComponent:
    return SourceComponent(
        id=str(item["id"]),
        source_classification=str(item["classification"]),
        include=tuple(str(value) for value in item["include"]),
        exclude=tuple(str(value) for value in item.get("exclude", [])),
        purpose=str(item["purpose"]),
        scope=scope,
        location=_location(generation, pointer),
        attributes=_attributes(item),
    )


def _artifact(item: dict[str, object], generation: ProfileGenerationIdentity, pointer: str, scope: SourceDeclarationScope) -> SourceAssociatedArtifact:
    return SourceAssociatedArtifact(
        id=str(item["id"]),
        anchor=str(item["anchor"]),
        include=tuple(str(value) for value in item["include"]),
        exclude=tuple(str(value) for value in item.get("exclude", [])),
        purpose=str(item["purpose"]),
        scope=scope,
        location=_location(generation, pointer),
    )


def _relationship(item: dict[str, object], generation: ProfileGenerationIdentity, pointer: str, scope: SourceDeclarationScope) -> SourceRelationship:
    return SourceRelationship(
        id=str(item["id"]),
        source=str(item["from"]),
        target=str(item["to"]),
        relationship_type=str(item["type"]),
        scope=scope,
        location=_location(generation, pointer),
    )


def _policies(payload: dict[str, object], generation: ProfileGenerationIdentity) -> tuple[SourcePolicy, ...]:
    result: list[SourcePolicy] = []
    policy_map = payload.get("policies")
    if isinstance(policy_map, dict):
        for name, value in sorted(policy_map.items()):
            result.append(SourcePolicy(str(name), freeze_json(value), _location(generation, f"/policies/{name}")))
    if "component_dependency_policy" in payload:
        result.append(
            SourcePolicy(
                "component_dependency_policy",
                freeze_json(payload["component_dependency_policy"]),
                _location(generation, "/component_dependency_policy"),
            )
        )
    return tuple(result)


def _parse_v034(payload: dict[str, object], generation: ProfileGenerationIdentity) -> CompatibilitySourceProfile:
    components: list[SourceComponent] = []
    raw_components = payload.get("components")
    if isinstance(raw_components, list):
        for index, item in enumerate(raw_components):
            if isinstance(item, dict):
                components.append(_component(item, generation, f"/components/{index}", SourceDeclarationScope.TOP_LEVEL))

    boundaries: list[SourceBoundary] = []
    raw_boundaries = payload.get("boundaries")
    boundary_map = (("product", "PRODUCT"), ("toolchain", "TOOLCHAIN"), ("neutral_contract", "NEUTRAL_CONTRACT"))
    if isinstance(raw_boundaries, dict):
        for source_key, classification in boundary_map:
            item = raw_boundaries.get(source_key)
            if isinstance(item, dict):
                boundaries.append(
                    SourceBoundary(
                        classification,
                        tuple(str(value) for value in item.get("roots", [])),
                        _location(generation, f"/boundaries/{source_key}"),
                    )
                )

    semantics = V034SourceSemantics(
        declaration_form="COMPONENTS" if components else "BOUNDARIES",
        boundaries=tuple(boundaries),
    )
    return CompatibilitySourceProfile(
        family=SourceFamily.TOOL_035_PROFILE,
        generation=_binding(generation),
        components=tuple(components),
        associated_artifacts=(),
        relationships=(),
        policies=_policies(payload, generation),
        family_semantics=semantics,
        raw_payload=freeze_json(payload),
    )


def _parse_v036(payload: dict[str, object], generation: ProfileGenerationIdentity) -> CompatibilitySourceProfile:
    responsibility_map = payload["responsibility_map"]
    assert isinstance(responsibility_map, dict)
    mode = str(responsibility_map["mode"])
    scope = SourceDeclarationScope.TOP_LEVEL
    component_items = payload.get("components", [])
    artifact_items = payload.get("associated_artifacts", [])
    relationship_items = payload.get("relationships", [])
    base_pointer = ""

    template_id: str | None = None
    template_revision: str | None = None
    remove_component_ids: tuple[str, ...] = ()
    remove_artifact_ids: tuple[str, ...] = ()
    remove_relationship_ids: tuple[str, ...] = ()

    template = responsibility_map.get("template")
    if isinstance(template, dict):
        template_id = str(template["id"])
        template_revision = str(template["revision"])

    if mode == "hybrid":
        scope = SourceDeclarationScope.RESPONSIBILITY_MAP_OVERRIDE
        overrides = responsibility_map.get("overrides")
        assert isinstance(overrides, dict)
        component_items = overrides.get("components", [])
        artifact_items = overrides.get("associated_artifacts", [])
        relationship_items = overrides.get("relationships", [])
        base_pointer = "/responsibility_map/overrides"
        remove_component_ids = tuple(str(value) for value in overrides.get("remove_component_ids", []))
        remove_artifact_ids = tuple(str(value) for value in overrides.get("remove_associated_artifact_ids", []))
        remove_relationship_ids = tuple(str(value) for value in overrides.get("remove_relationship_ids", []))

    def pointer(label: str, index: int) -> str:
        return f"{base_pointer}/{label}/{index}" if base_pointer else f"/{label}/{index}"

    components = tuple(
        _component(item, generation, pointer("components", index), scope)
        for index, item in enumerate(component_items)
        if isinstance(item, dict)
    )
    artifacts = tuple(
        _artifact(item, generation, pointer("associated_artifacts", index), scope)
        for index, item in enumerate(artifact_items)
        if isinstance(item, dict)
    )
    relationships = tuple(
        _relationship(item, generation, pointer("relationships", index), scope)
        for index, item in enumerate(relationship_items)
        if isinstance(item, dict)
    )
    semantics = V036SourceSemantics(
        responsibility_map_mode=mode,
        template_id=template_id,
        template_revision=template_revision,
        remove_component_ids=remove_component_ids,
        remove_associated_artifact_ids=remove_artifact_ids,
        remove_relationship_ids=remove_relationship_ids,
    )
    return CompatibilitySourceProfile(
        family=SourceFamily.TOOL_036_PROFILE,
        generation=_binding(generation),
        components=components,
        associated_artifacts=artifacts,
        relationships=relationships,
        policies=_policies(payload, generation),
        family_semantics=semantics,
        raw_payload=freeze_json(payload),
    )


def read_source_profile(repository_root: str | Path, generation: ProfileGenerationIdentity) -> SourceReadResult:
    root = Path(repository_root).expanduser().resolve()
    try:
        normalized = normalize_profile_path(generation.path)
        path = profile_path_on_disk(root, normalized)
    except ValueError as exc:
        return SourceReadResult(None, (_issue("SOURCE_PATH_ERROR", str(exc), generation),))

    try:
        raw = path.read_bytes()
    except OSError as exc:
        return SourceReadResult(None, (_issue("SOURCE_READ_ERROR", f"Unable to read source profile: {exc}", generation),))

    digest = hashlib.sha256(raw).hexdigest()
    if digest != generation.content_sha256:
        return SourceReadResult(
            None,
            (_issue("SOURCE_CONTENT_STALE", "Source profile bytes no longer match the WU-01 generation identity.", generation),),
        )

    try:
        payload = yaml.safe_load(raw.decode("utf-8-sig"))
    except (UnicodeError, yaml.YAMLError) as exc:
        return SourceReadResult(None, (_issue("SOURCE_PARSE_ERROR", f"Unable to parse source profile: {exc}", generation),))
    if not isinstance(payload, dict):
        return SourceReadResult(None, (_issue("SOURCE_SHAPE_ERROR", "Source profile root must be a mapping.", generation, ""),))

    ptsip = payload.get("ptsip")
    specification = ptsip.get("specification") if isinstance(ptsip, dict) else None
    declared_version = ptsip.get("version") if isinstance(ptsip, dict) else None
    revision = specification.get("revision") if isinstance(specification, dict) else None
    source = specification.get("source") if isinstance(specification, dict) else None

    identity_issues: list[SourceReadIssue] = []
    if declared_version != generation.declared_version:
        identity_issues.append(_issue("SOURCE_VERSION_MISMATCH", "ptsip.version does not match the WU-01 generation identity.", generation, "/ptsip/version"))
    if revision != generation.specification_revision:
        identity_issues.append(_issue("SOURCE_REVISION_MISMATCH", "Specification revision does not match the WU-01 generation identity.", generation, "/ptsip/specification/revision"))
    if source != generation.specification_source:
        identity_issues.append(_issue("SOURCE_SPECIFICATION_SOURCE_MISMATCH", "Specification source does not match the WU-01 generation identity.", generation, "/ptsip/specification/source"))
    if identity_issues:
        return SourceReadResult(None, tuple(identity_issues))

    family = _SUPPORTED.get((str(declared_version), str(revision)))
    if family is None:
        same_version = any(version == declared_version for version, _revision in _SUPPORTED)
        code = "UNSUPPORTED_SOURCE_REVISION" if same_version else "UNSUPPORTED_SOURCE_FAMILY"
        message = (
            f"Source family {declared_version!r} is recognized but revision {revision!r} is not frozen for Tool 0.3.7 migration."
            if same_version
            else f"Source profile family {declared_version!r} is not supported by Tool 0.3.7 migration readers."
        )
        return SourceReadResult(None, (_issue(code, message, generation, "/ptsip"),))

    if source != SPEC_SOURCE:
        return SourceReadResult(None, (_issue("UNSUPPORTED_SPECIFICATION_SOURCE", f"Unsupported PTSIP specification source {source!r}.", generation, "/ptsip/specification/source"),))

    issues = _schema_issues(payload, family, generation)
    if issues:
        return SourceReadResult(None, tuple(issues))

    profile = _parse_v034(payload, generation) if family == SourceFamily.TOOL_035_PROFILE else _parse_v036(payload, generation)
    return SourceReadResult(profile, ())


def validate_source_read_binding(repository_root: str | Path, result: SourceReadResult) -> tuple[SourceReadIssue, ...]:
    if result.profile is None:
        return result.issues or (SourceReadIssue("NO_SOURCE_PROFILE", "No compatibility source profile is available.", "<unknown>"),)
    binding = result.profile.generation
    root = Path(repository_root).expanduser().resolve()
    try:
        path = profile_path_on_disk(root, normalize_profile_path(binding.profile_path))
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
    except (OSError, ValueError) as exc:
        return (SourceReadIssue("SOURCE_BINDING_INVALID", str(exc), binding.profile_path),)
    if digest != binding.content_sha256:
        return (SourceReadIssue("SOURCE_CONTENT_STALE", "Source profile bytes changed after compatibility read.", binding.profile_path),)
    return ()
