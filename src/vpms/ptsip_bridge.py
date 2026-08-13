from __future__ import annotations

from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Mapping

import yaml

from .model import TargetRef


class PtsipMetadataError(ValueError):
    """Raised when the minimal PTSIP metadata contract cannot be read safely."""


@dataclass(frozen=True)
class PtsipTargetMetadata:
    component_id: str
    classification: str

    def as_dict(self) -> dict[str, str]:
        return asdict(self)


@dataclass(frozen=True)
class PtsipMetadataSnapshot:
    targets: tuple[PtsipTargetMetadata, ...]

    def get_target(self, component_id: str) -> PtsipTargetMetadata | None:
        for target in self.targets:
            if target.component_id == component_id:
                return target
        return None

    def as_dict(self) -> dict[str, object]:
        return {"targets": [target.as_dict() for target in self.targets]}


def _is_identifier(value: object) -> bool:
    return isinstance(value, str) and bool(value) and value.strip() == value


def load_ptsip_metadata(profile_path: str | Path) -> PtsipMetadataSnapshot:
    """Read the minimal stable component metadata VPMS consumes from a PTSIP profile.

    This bridge intentionally does not perform PTSIP conformance validation and
    exposes no write path into the project profile or Decision Authority.
    """

    path = Path(profile_path)
    try:
        payload = yaml.safe_load(path.read_text(encoding="utf-8-sig"))
    except Exception as exc:
        raise PtsipMetadataError(f"Unable to read PTSIP project profile: {exc}") from exc

    if not isinstance(payload, Mapping):
        raise PtsipMetadataError("PTSIP project profile root must be a mapping.")

    raw_components = payload.get("components")
    if raw_components is None:
        return PtsipMetadataSnapshot(targets=())
    if not isinstance(raw_components, list):
        raise PtsipMetadataError("PTSIP project profile components must be a list.")

    targets: list[PtsipTargetMetadata] = []
    seen: set[str] = set()
    for index, raw_component in enumerate(raw_components):
        if not isinstance(raw_component, Mapping):
            raise PtsipMetadataError(f"PTSIP component at index {index} must be a mapping.")

        component_id = raw_component.get("id")
        classification = raw_component.get("classification")
        if not _is_identifier(component_id):
            raise PtsipMetadataError(
                f"PTSIP component at index {index} has an invalid id."
            )
        if not _is_identifier(classification):
            raise PtsipMetadataError(
                f"PTSIP component {component_id!r} has an invalid classification."
            )
        if component_id in seen:
            raise PtsipMetadataError(f"Duplicate PTSIP component id: {component_id}.")

        seen.add(component_id)
        targets.append(
            PtsipTargetMetadata(
                component_id=component_id,
                classification=classification,
            )
        )

    targets.sort(key=lambda target: target.component_id)
    return PtsipMetadataSnapshot(targets=tuple(targets))


def resolve_target_metadata(
    target: TargetRef,
    metadata: PtsipMetadataSnapshot,
) -> PtsipTargetMetadata | None:
    """Resolve one VPMS TargetRef against read-only PTSIP component metadata."""

    return metadata.get_target(target.component_id)
