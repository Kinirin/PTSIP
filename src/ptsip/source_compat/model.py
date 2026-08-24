from __future__ import annotations

from dataclasses import dataclass
from enum import StrEnum
from typing import TypeAlias

FrozenScalar: TypeAlias = str | int | float | bool | None
FrozenJson: TypeAlias = FrozenScalar | tuple["FrozenJson", ...] | tuple[tuple[str, "FrozenJson"], ...]


def freeze_json(value: object) -> FrozenJson:
    if value is None or isinstance(value, (str, int, float, bool)):
        return value
    if isinstance(value, dict):
        return tuple((str(key), freeze_json(item)) for key, item in sorted(value.items(), key=lambda pair: str(pair[0])))
    if isinstance(value, (list, tuple)):
        return tuple(freeze_json(item) for item in value)
    return str(value)


def thaw_json(value: FrozenJson) -> object:
    if isinstance(value, tuple):
        if all(isinstance(item, tuple) and len(item) == 2 and isinstance(item[0], str) for item in value):
            return {item[0]: thaw_json(item[1]) for item in value}  # type: ignore[index]
        return [thaw_json(item) for item in value]
    return value


class SourceFamily(StrEnum):
    TOOL_035_PROFILE = "TOOL_0.3.5_PROFILE_0.3.4_DRAFT"
    TOOL_036_PROFILE = "TOOL_0.3.6_PROFILE_0.3.6_DRAFT"


class SourceDeclarationScope(StrEnum):
    TOP_LEVEL = "TOP_LEVEL"
    RESPONSIBILITY_MAP_OVERRIDE = "RESPONSIBILITY_MAP_OVERRIDE"


@dataclass(frozen=True)
class SourceGenerationBinding:
    profile_path: str
    declared_version: str
    specification_revision: str
    specification_source: str | None
    content_sha256: str
    temporary: bool

    def as_dict(self) -> dict[str, object]:
        return {
            "profile_path": self.profile_path,
            "declared_version": self.declared_version,
            "specification_revision": self.specification_revision,
            "specification_source": self.specification_source,
            "content_sha256": self.content_sha256,
            "temporary": self.temporary,
        }


@dataclass(frozen=True)
class SourceLocation:
    profile_path: str
    pointer: str

    def as_dict(self) -> dict[str, str]:
        return {"profile_path": self.profile_path, "pointer": self.pointer}


@dataclass(frozen=True)
class SourceAttribute:
    name: str
    value: FrozenJson

    def as_dict(self) -> dict[str, object]:
        return {"name": self.name, "value": thaw_json(self.value)}


@dataclass(frozen=True)
class SourceComponent:
    id: str
    source_classification: str
    include: tuple[str, ...]
    exclude: tuple[str, ...]
    purpose: str
    scope: SourceDeclarationScope
    location: SourceLocation
    attributes: tuple[SourceAttribute, ...] = ()

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "source_classification": self.source_classification,
            "include": list(self.include),
            "exclude": list(self.exclude),
            "purpose": self.purpose,
            "scope": self.scope.value,
            "location": self.location.as_dict(),
            "attributes": [item.as_dict() for item in self.attributes],
        }


@dataclass(frozen=True)
class SourceAssociatedArtifact:
    id: str
    anchor: str
    include: tuple[str, ...]
    exclude: tuple[str, ...]
    purpose: str
    scope: SourceDeclarationScope
    location: SourceLocation

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "anchor": self.anchor,
            "include": list(self.include),
            "exclude": list(self.exclude),
            "purpose": self.purpose,
            "scope": self.scope.value,
            "location": self.location.as_dict(),
        }


@dataclass(frozen=True)
class SourceRelationship:
    id: str
    source: str
    target: str
    relationship_type: str
    scope: SourceDeclarationScope
    location: SourceLocation

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "from": self.source,
            "to": self.target,
            "type": self.relationship_type,
            "scope": self.scope.value,
            "location": self.location.as_dict(),
        }


@dataclass(frozen=True)
class SourcePolicy:
    name: str
    value: FrozenJson
    location: SourceLocation

    def as_dict(self) -> dict[str, object]:
        return {"name": self.name, "value": thaw_json(self.value), "location": self.location.as_dict()}


@dataclass(frozen=True)
class SourceBoundary:
    source_classification: str
    roots: tuple[str, ...]
    location: SourceLocation

    def as_dict(self) -> dict[str, object]:
        return {
            "source_classification": self.source_classification,
            "roots": list(self.roots),
            "location": self.location.as_dict(),
        }


@dataclass(frozen=True)
class V034SourceSemantics:
    declaration_form: str
    boundaries: tuple[SourceBoundary, ...] = ()

    def as_dict(self) -> dict[str, object]:
        return {
            "kind": "V034",
            "declaration_form": self.declaration_form,
            "boundaries": [item.as_dict() for item in self.boundaries],
        }


@dataclass(frozen=True)
class V036SourceSemantics:
    responsibility_map_mode: str
    template_id: str | None = None
    template_revision: str | None = None
    remove_component_ids: tuple[str, ...] = ()
    remove_associated_artifact_ids: tuple[str, ...] = ()
    remove_relationship_ids: tuple[str, ...] = ()

    def as_dict(self) -> dict[str, object]:
        return {
            "kind": "V036",
            "responsibility_map_mode": self.responsibility_map_mode,
            "template_id": self.template_id,
            "template_revision": self.template_revision,
            "remove_component_ids": list(self.remove_component_ids),
            "remove_associated_artifact_ids": list(self.remove_associated_artifact_ids),
            "remove_relationship_ids": list(self.remove_relationship_ids),
        }


SourceFamilySemantics: TypeAlias = V034SourceSemantics | V036SourceSemantics


@dataclass(frozen=True)
class CompatibilitySourceProfile:
    family: SourceFamily
    generation: SourceGenerationBinding
    components: tuple[SourceComponent, ...]
    associated_artifacts: tuple[SourceAssociatedArtifact, ...]
    relationships: tuple[SourceRelationship, ...]
    policies: tuple[SourcePolicy, ...]
    family_semantics: SourceFamilySemantics
    raw_payload: FrozenJson

    def as_dict(self) -> dict[str, object]:
        return {
            "kind": "PTSIP_COMPATIBILITY_SOURCE",
            "authority": "SOURCE_DECLARATION_ONLY",
            "family": self.family.value,
            "generation": self.generation.as_dict(),
            "components": [item.as_dict() for item in self.components],
            "associated_artifacts": [item.as_dict() for item in self.associated_artifacts],
            "relationships": [item.as_dict() for item in self.relationships],
            "policies": [item.as_dict() for item in self.policies],
            "family_semantics": self.family_semantics.as_dict(),
            "raw_payload": thaw_json(self.raw_payload),
        }


@dataclass(frozen=True)
class SourceReadIssue:
    code: str
    message: str
    profile_path: str
    pointer: str | None = None

    def as_dict(self) -> dict[str, object]:
        return {
            "code": self.code,
            "message": self.message,
            "profile_path": self.profile_path,
            "pointer": self.pointer,
        }


@dataclass(frozen=True)
class SourceReadResult:
    profile: CompatibilitySourceProfile | None
    issues: tuple[SourceReadIssue, ...]

    @property
    def complete(self) -> bool:
        return self.profile is not None and not self.issues

    def as_dict(self) -> dict[str, object]:
        return {
            "complete": self.complete,
            "profile": self.profile.as_dict() if self.profile else None,
            "issues": [item.as_dict() for item in self.issues],
        }
