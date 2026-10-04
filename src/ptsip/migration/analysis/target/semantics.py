from __future__ import annotations

from dataclasses import dataclass

from ptsip.model import Classification, ResponsibilityRelationshipType
from ptsip.profile_identity import CURRENT_PROJECT_PROFILE_VERSION

@dataclass(frozen=True)
class TargetSemantics:
    draft_version: str
    classifications: tuple[str, ...]
    relationship_types: tuple[str, ...]

    def as_dict(self) -> dict[str, object]:
        return {
            "draft_version": self.draft_version,
            "classifications": list(self.classifications),
            "relationship_types": list(self.relationship_types),
        }

@dataclass(frozen=True)
class TargetComponent:
    id: str
    classification: str
    include: tuple[str, ...]
    exclude: tuple[str, ...] = ()

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "classification": self.classification,
            "include": list(self.include),
            "exclude": list(self.exclude),
        }

@dataclass(frozen=True)
class TargetAssociatedArtifact:
    id: str
    anchor: str
    include: tuple[str, ...]
    exclude: tuple[str, ...] = ()

    def as_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "anchor": self.anchor,
            "include": list(self.include),
            "exclude": list(self.exclude),
        }

@dataclass(frozen=True)
class TargetRelationship:
    id: str
    source: str
    target: str
    relationship_type: str

    def as_dict(self) -> dict[str, str]:
        return {
            "id": self.id,
            "from": self.source,
            "to": self.target,
            "type": self.relationship_type,
        }

@dataclass(frozen=True)
class TargetArchitectureState:
    draft_version: str
    specification_revision: str
    components: tuple[TargetComponent, ...]
    associated_artifacts: tuple[TargetAssociatedArtifact, ...] = ()
    relationships: tuple[TargetRelationship, ...] = ()

    def as_dict(self) -> dict[str, object]:
        return {
            "draft_version": self.draft_version,
            "specification_revision": self.specification_revision,
            "components": [item.as_dict() for item in self.components],
            "associated_artifacts": [item.as_dict() for item in self.associated_artifacts],
            "relationships": [item.as_dict() for item in self.relationships],
        }

def default_target_semantics() -> TargetSemantics:
    """Return semantics for the current canonical Project Profile target.

    The target identity comes from the independent PP capability namespace.  It
    must not be derived from the Tool version or Specification family label.
    """

    return TargetSemantics(
        draft_version=CURRENT_PROJECT_PROFILE_VERSION,
        classifications=tuple(item.value for item in Classification),
        relationship_types=tuple(item.value for item in ResponsibilityRelationshipType),
    )

def target_state_from_mapping(payload: dict[str, object]) -> TargetArchitectureState:
    ptsip = payload.get("ptsip")
    if not isinstance(ptsip, dict):
        raise ValueError("Target state requires ptsip metadata.")
    version = ptsip.get("version")
    specification = ptsip.get("specification")
    revision = specification.get("revision") if isinstance(specification, dict) else None
    if not isinstance(version, str) or not isinstance(revision, str) or not revision:
        raise ValueError("Target state requires Project Profile contract identity and immutable specification revision.")

    responsibility_map = payload.get("responsibility_map")
    if not isinstance(responsibility_map, dict) or responsibility_map.get("mode") != "explicit":
        raise ValueError(
            "WU-05 target-state mapping accepts explicit target state only; "
            "template/hybrid target materialization belongs to the target-contract runtime boundary."
        )

    components = tuple(
        sorted(
            (
                TargetComponent(
                    id=str(item.get("id", "")),
                    classification=str(item.get("classification", "")),
                    include=tuple(str(value) for value in item.get("include", [])),
                    exclude=tuple(str(value) for value in item.get("exclude", [])),
                )
                for item in payload.get("components", [])
                if isinstance(item, dict)
            ),
            key=lambda item: item.id,
        )
    )
    artifacts = tuple(
        sorted(
            (
                TargetAssociatedArtifact(
                    id=str(item.get("id", "")),
                    anchor=str(item.get("anchor", "")),
                    include=tuple(str(value) for value in item.get("include", [])),
                    exclude=tuple(str(value) for value in item.get("exclude", [])),
                )
                for item in payload.get("associated_artifacts", [])
                if isinstance(item, dict)
            ),
            key=lambda item: item.id,
        )
    )
    relationships = tuple(
        sorted(
            (
                TargetRelationship(
                    id=str(item.get("id", "")),
                    source=str(item.get("from", "")),
                    target=str(item.get("to", "")),
                    relationship_type=str(item.get("type", "")),
                )
                for item in payload.get("relationships", [])
                if isinstance(item, dict)
            ),
            key=lambda item: item.id,
        )
    )
    return TargetArchitectureState(version, revision, components, artifacts, relationships)

