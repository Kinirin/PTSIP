from __future__ import annotations

from dataclasses import dataclass

from .profile_identity import (
    PP_0_00,
    PP_1_01,
    PP_COMPATIBILITY_TARGET_TOOL_VERSION,
    ProjectProfileIdentityError,
    ProjectProfileOperation,
    ProjectProfileTransitionKind,
    ProjectProfileVersion,
    require_project_profile_support,
)

SPEC_SOURCE = "https://github.com/Kinirin/PTSIP"
V034_REVISION = "b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e"
V036_REVISION = "d6995ed232e845b88d8235b851e80ab54b7804ea"


@dataclass(frozen=True)
class HistoricalProjectProfileBridge:
    """Evidence-bound interpretation of one historical Project Profile source."""

    declared_version: str
    specification_revision: str
    compatibility_contract: ProjectProfileVersion
    target_contract: ProjectProfileVersion
    transition_kind: ProjectProfileTransitionKind
    source_family: str
    legacy_target_filename: str | None = None

    @property
    def direct_target_filename(self) -> str:
        return f"ptsip_{self.target_contract.filename_token}.yaml"

    @property
    def equivalent_target_filenames(self) -> tuple[str, ...]:
        values = [self.direct_target_filename]
        if self.legacy_target_filename is not None:
            values.insert(0, self.legacy_target_filename)
        return tuple(values)

    def as_dict(self) -> dict[str, object]:
        return {
            "declared_version": self.declared_version,
            "specification_revision": self.specification_revision,
            "compatibility_contract": self.compatibility_contract.canonical,
            "target_contract": self.target_contract.canonical,
            "transition_kind": self.transition_kind.value,
            "source_family": self.source_family,
            "legacy_target_filename": self.legacy_target_filename,
            "direct_target_filename": self.direct_target_filename,
        }


@dataclass(frozen=True)
class ProjectProfileTarget:
    tool_version: str
    contract: ProjectProfileVersion
    schema_resource: str
    temporary_filename: str

    def as_dict(self) -> dict[str, str]:
        return {
            "tool_version": self.tool_version,
            "contract": self.contract.canonical,
            "schema_resource": self.schema_resource,
            "temporary_filename": self.temporary_filename,
        }


_HISTORICAL_BRIDGES: dict[tuple[str, str], HistoricalProjectProfileBridge] = {
    (
        "0.3.4-draft",
        V034_REVISION,
    ): HistoricalProjectProfileBridge(
        declared_version="0.3.4-draft",
        specification_revision=V034_REVISION,
        compatibility_contract=PP_0_00,
        target_contract=PP_1_01,
        transition_kind=ProjectProfileTransitionKind.SEMANTIC_MIGRATION,
        source_family="TOOL_0.3.5_PROFILE_0.3.4_DRAFT",
    ),
    (
        "0.3.6-draft",
        V036_REVISION,
    ): HistoricalProjectProfileBridge(
        declared_version="0.3.6-draft",
        specification_revision=V036_REVISION,
        compatibility_contract=PP_1_01,
        target_contract=PP_1_01,
        transition_kind=ProjectProfileTransitionKind.IDENTITY_ONLY,
        source_family="TOOL_0.3.6_PROFILE_0.3.6_DRAFT",
        legacy_target_filename="ptsip_0.3.6.yaml",
    ),
}


def historical_project_profile_bridges() -> tuple[HistoricalProjectProfileBridge, ...]:
    return tuple(_HISTORICAL_BRIDGES[key] for key in sorted(_HISTORICAL_BRIDGES))


def historical_project_profile_bridge(
    declared_version: object,
    specification_revision: object,
) -> HistoricalProjectProfileBridge | None:
    if not isinstance(declared_version, str) or not isinstance(specification_revision, str):
        return None
    return _HISTORICAL_BRIDGES.get((declared_version.strip(), specification_revision.strip()))


def require_historical_project_profile_bridge(
    declared_version: object,
    specification_revision: object,
) -> HistoricalProjectProfileBridge:
    bridge = historical_project_profile_bridge(declared_version, specification_revision)
    if bridge is None:
        raise ProjectProfileIdentityError(
            "PP_COMPAT_UNSUPPORTED_HISTORICAL_SOURCE",
            "Historical Project Profile source is not registered for direct convergence.",
            {
                "declared_version": declared_version,
                "specification_revision": specification_revision,
            },
        )
    return bridge


def current_project_profile_target(
    tool_version: str = PP_COMPATIBILITY_TARGET_TOOL_VERSION,
) -> ProjectProfileTarget:
    support = require_project_profile_support(
        tool_version,
        PP_1_01,
        ProjectProfileOperation.CREATE_TARGET,
    )
    if not support.schema_resource:
        raise ProjectProfileIdentityError(
            "PP_COMPAT_TARGET_SCHEMA_MISSING",
            f"Project Profile target {PP_1_01.canonical!r} has no schema resource.",
            PP_1_01.canonical,
        )
    return ProjectProfileTarget(
        tool_version=tool_version,
        contract=PP_1_01,
        schema_resource=support.schema_resource,
        temporary_filename=f"ptsip_{PP_1_01.filename_token}.yaml",
    )


def require_direct_historical_transition(
    declared_version: object,
    specification_revision: object,
    target_contract: ProjectProfileVersion | str = PP_1_01,
) -> HistoricalProjectProfileBridge:
    bridge = require_historical_project_profile_bridge(
        declared_version,
        specification_revision,
    )
    target = (
        ProjectProfileVersion.parse(target_contract, require_canonical=True)
        if isinstance(target_contract, str)
        else target_contract
    )
    if target != bridge.target_contract:
        raise ProjectProfileIdentityError(
            "PP_COMPAT_UNSUPPORTED_DIRECT_TARGET",
            (
                f"Historical source {bridge.declared_version!r} is registered for direct "
                f"convergence to {bridge.target_contract.canonical!r}, not {target.canonical!r}."
            ),
            {
                "declared_version": bridge.declared_version,
                "target_contract": target.canonical,
            },
        )
    return bridge
