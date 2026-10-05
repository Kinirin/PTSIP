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
    if not isinstance(raw_components, (list, tuple)):
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


def metadata_from_effective_map(handoff: object) -> PtsipMetadataSnapshot:
    """Project only a real PTSIP-owned validated immutable provider handoff.

    Raw dictionaries, materialization results and caller booleans are not
    validation proof. This boundary neither validates profiles itself nor
    falls back to the historical raw-profile compatibility reader.
    """
    from ptsip.governance import AuthorityCatalog
    from ptsip.validation.handoff import EffectiveMapValidationError, _effective_map_data

    support = AuthorityCatalog(Path(__file__).resolve().parents[3])
    _, route, policy = support.load_current_record("SFP-0023")
    if route["status"] != "ACTIVE" or policy["policy"]["status"] != "ACTIVE":
        raise PtsipMetadataError("Read-only PTSIP integration contract is not active.")
    if policy["authority_semantics"]["input"]["concrete_provider_binding"] != "ptsip.validation.handoff:load_validated_effective_map":
        raise PtsipMetadataError("Unsupported PTSIP provider binding.")
    try:
        payload = _effective_map_data(handoff).payload
    except EffectiveMapValidationError as exc:
        raise PtsipMetadataError(str(exc)) from exc
    return _metadata_from_payload(
        payload,
        source_label="PTSIP effective Responsibility Map",
    )


def load_ptsip_metadata(profile_path: str | Path) -> PtsipMetadataSnapshot:
    """Read the historical raw-profile metadata boundary used before Tool 0.3.6.

    This compatibility bridge intentionally does not perform PTSIP conformance
    validation and exposes no write path into the project profile or Decision
    Authority. Tool 0.3.6 canonical consumption uses
    :func:`metadata_from_effective_map` after the PTSIP layer has produced a
    factory-validated immutable provider handoff. This compatibility reader
    is never canonical validation proof or an implicit fallback.

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
