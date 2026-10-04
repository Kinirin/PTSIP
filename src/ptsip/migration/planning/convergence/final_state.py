from __future__ import annotations

from dataclasses import dataclass
from enum import StrEnum
from typing import Mapping

from ptsip.migration.proposal.semantic_identity import canonical_semantics, semantic_digest
from ptsip.migration.proposal.target_delta import TargetEntityKind

class FinalPointKind(StrEnum):
    EXISTING = "EXISTING"
    PLANNED = "PLANNED"

@dataclass(frozen=True)
class FinalPointReference:
    kind: FinalPointKind
    path: str
    draft_version: str
    specification_revision: str
    content_sha256: str | None = None

    def as_dict(self) -> dict[str, object]:
        return {
            "kind": self.kind.value,
            "path": self.path,
            "draft_version": self.draft_version,
            "specification_revision": self.specification_revision,
            "content_sha256": self.content_sha256,
        }

@dataclass(frozen=True)
class FinalPointEntity:
    kind: TargetEntityKind
    id: str
    payload: object

    def as_dict(self) -> dict[str, object]:
        return {
            "kind": self.kind.value,
            "id": self.id,
            "payload": canonical_semantics(self.payload),
        }

@dataclass(frozen=True)
class FinalPointStateSnapshot:
    path: str
    draft_version: str
    specification_revision: str
    content_sha256: str | None
    entities: tuple[FinalPointEntity, ...]

    @property
    def semantic_digest(self) -> str:
        return semantic_digest(
            {
                "draft_version": self.draft_version,
                "specification_revision": self.specification_revision,
                "entities": [
                    item.as_dict()
                    for item in sorted(self.entities, key=lambda item: (item.kind.value, item.id))
                ],
            }
        )

    def as_dict(self) -> dict[str, object]:
        return {
            "path": self.path,
            "draft_version": self.draft_version,
            "specification_revision": self.specification_revision,
            "content_sha256": self.content_sha256,
            "entities": [
                item.as_dict()
                for item in sorted(self.entities, key=lambda item: (item.kind.value, item.id))
            ],
            "semantic_digest": self.semantic_digest,
        }

def final_point_state_from_mapping(
    payload: Mapping[str, object],
    *,
    path: str,
    content_sha256: str | None = None,
) -> FinalPointStateSnapshot:
    ptsip = payload.get("ptsip")
    specification = ptsip.get("specification") if isinstance(ptsip, Mapping) else None
    version = ptsip.get("version") if isinstance(ptsip, Mapping) else None
    revision = specification.get("revision") if isinstance(specification, Mapping) else None
    if not isinstance(version, str) or not isinstance(revision, str) or not revision:
        raise ValueError("Final Point state requires draft version and immutable specification revision.")

    rows: list[FinalPointEntity] = []
    for key, kind in (
        ("components", TargetEntityKind.COMPONENT),
        ("associated_artifacts", TargetEntityKind.ASSOCIATED_ARTIFACT),
        ("relationships", TargetEntityKind.RELATIONSHIP),
    ):
        raw = payload.get(key, [])
        if not isinstance(raw, list):
            raise ValueError(f"{key} must be a list.")
        for item in raw:
            if not isinstance(item, Mapping) or not isinstance(item.get("id"), str) or not item.get("id"):
                raise ValueError(f"{key} entries require stable string ids.")
            rows.append(FinalPointEntity(kind, str(item["id"]), canonical_semantics(dict(item))))

    if "component_dependency_policy" in payload:
        rows.append(
            FinalPointEntity(
                TargetEntityKind.COMPONENT_DEPENDENCY_POLICY,
                "component_dependency_policy",
                canonical_semantics(payload["component_dependency_policy"]),
            )
        )
    if "policies" in payload:
        rows.append(
            FinalPointEntity(
                TargetEntityKind.POLICIES,
                "policies",
                canonical_semantics(payload["policies"]),
            )
        )

    keys = [(item.kind.value, item.id) for item in rows]
    if len(keys) != len(set(keys)):
        raise ValueError("Final Point contains duplicate stable entity identities.")
    return FinalPointStateSnapshot(
        path,
        version,
        revision,
        content_sha256,
        tuple(sorted(rows, key=lambda item: (item.kind.value, item.id))),
    )

