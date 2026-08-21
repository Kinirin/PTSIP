from __future__ import annotations

from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Mapping

import yaml

from ..domain.model import TargetRef


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


def _metadata_from_payload(
    payload: object,
    *,
    source_label: str,
) -> PtsipMetadataSnapshot:
    if not isinstance(payload, Mapping):
        raise PtsipMetadataError(f"{source_label} root must be a mapping.")

    raw_components = payload.get("components")
    if raw_components is None:
        return PtsipMetadataSnapshot(targets=())
    if not isinstance(raw_components, list):
        raise PtsipMetadataError(f"{source_label} components must be a list.")

    targets: list[PtsipTargetMetadata] = []
    seen: set[str] = set()
    for index, raw_component in enumerate(raw_components):
        if not isinstance(raw_component, Mapping):
            raise PtsipMetadataError(
                f"{source_label} component at index {index} must be a mapping."
            )

        component_id = raw_component.get("id")
        classification = raw_component.get("classification")
        if not _is_identifier(component_id):
            raise PtsipMetadataError(
                f"{source_label} component at index {index} has an invalid id."
            )
        if not _is_identifier(classification):
            raise PtsipMetadataError(
                f"{source_label} component {component_id!r} has an invalid classification."
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


def metadata_from_effective_map(effective_payload: object) -> PtsipMetadataSnapshot:
    """Project canonical Tool 0.3.6 VPMS metadata from a resolved effective map.

    The caller is responsible for obtaining this payload from the validated PTSIP
    ``ResolvedProfile.effective_payload`` handoff. VPMS only projects the narrow
    read-only target metadata it owns; it does not read a profile file, interpret
    declaration mode, materialize templates, or perform PTSIP validation here.

    A missing resolved handoff fails closed. Canonical Tool 0.3.6 consumption
    never falls back to the historical raw-profile reader.
    """

    if effective_payload is None:
        raise PtsipMetadataError(
            "Validated PTSIP resolved effective Responsibility Map is required; "
            "canonical VPMS metadata does not fall back to a raw project profile."
        )

    return _metadata_from_payload(
        effective_payload,
        source_label="PTSIP effective Responsibility Map",
    )


def load_ptsip_metadata(profile_path: str | Path) -> PtsipMetadataSnapshot:
    """Read the historical raw-profile metadata boundary used before Tool 0.3.6.

    This compatibility bridge intentionally does not perform PTSIP conformance
    validation and exposes no write path into the project profile or Decision
    Authority. Tool 0.3.6 canonical consumption uses
    :func:`metadata_from_effective_map` after the PTSIP layer has produced a
    validated ``ResolvedProfile.effective_payload``.

    Template/hybrid source profiles remain rejected here so this compatibility
    reader cannot become a second materializer or architecture authority.
    """

    path = Path(profile_path)
    try:
        payload = yaml.safe_load(path.read_text(encoding="utf-8-sig"))
    except Exception as exc:
        raise PtsipMetadataError(f"Unable to read PTSIP project profile: {exc}") from exc

    if not isinstance(payload, Mapping):
        raise PtsipMetadataError("PTSIP project profile root must be a mapping.")

    responsibility_map = payload.get("responsibility_map")
    if isinstance(responsibility_map, Mapping):
        mode = responsibility_map.get("mode")
        if mode in {"template", "hybrid"}:
            raise PtsipMetadataError(
                "PTSIP template/hybrid Responsibility Map must be materialized before VPMS metadata consumption."
            )

    return _metadata_from_payload(payload, source_label="PTSIP project profile")


def resolve_target_metadata(
    target: TargetRef,
    metadata: PtsipMetadataSnapshot,
) -> PtsipTargetMetadata | None:
    """Resolve one VPMS TargetRef against read-only PTSIP component metadata."""

    return metadata.get_target(target.component_id)
