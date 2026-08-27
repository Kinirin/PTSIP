from __future__ import annotations

import hashlib
from dataclasses import dataclass
from enum import StrEnum
from pathlib import Path

import yaml

from ..profile_compatibility import (
    HistoricalProjectProfileBridge,
    current_project_profile_target,
    historical_project_profile_bridge,
)
from ..profile_identity import (
    PP_COMPATIBILITY_TARGET_TOOL_VERSION,
    ProjectProfileIdentityError,
    ProjectProfileOperation,
    ProjectProfileTransitionKind,
    ProjectProfileVersion,
    require_project_profile_support,
)
from .profile_path import DEFAULT_PROFILE_PATH, normalize_profile_path, profile_path_on_disk
from .snapshot import RepositorySnapshot, capture_snapshot, compare_snapshots


class DirectConvergenceMode(StrEnum):
    CURRENT = "CURRENT"
    IDENTITY_ONLY = "IDENTITY_ONLY"
    DIRECT_SEMANTIC_MIGRATION = "DIRECT_SEMANTIC_MIGRATION"


@dataclass(frozen=True)
class ConvergenceDiagnostic:
    code: str
    message: str
    path: str | None = None

    def as_dict(self) -> dict[str, object]:
        return {"code": self.code, "message": self.message, "path": self.path}


@dataclass(frozen=True)
class ConvergenceProfileBinding:
    path: str
    declared_version: str
    specification_revision: str
    specification_source: str | None
    content_sha256: str
    temporary: bool

    def as_dict(self) -> dict[str, object]:
        return {
            "path": self.path,
            "declared_version": self.declared_version,
            "specification_revision": self.specification_revision,
            "specification_source": self.specification_source,
            "content_sha256": self.content_sha256,
            "temporary": self.temporary,
        }


@dataclass(frozen=True)
class DirectConvergenceState:
    mode: DirectConvergenceMode
    source: ConvergenceProfileBinding
    source_compatibility_contract: ProjectProfileVersion
    target_contract: ProjectProfileVersion
    transition_kind: ProjectProfileTransitionKind | None
    target_path: str
    target: ConvergenceProfileBinding | None
    target_is_legacy_alias: bool
    requires_temporary_target: bool
    snapshot: RepositorySnapshot

    @property
    def intermediate_profiles(self) -> tuple[object, ...]:
        return ()

    def as_dict(self) -> dict[str, object]:
        return {
            "mode": self.mode.value,
            "source": self.source.as_dict(),
            "source_compatibility_contract": self.source_compatibility_contract.canonical,
            "target_contract": self.target_contract.canonical,
            "transition_kind": self.transition_kind.value if self.transition_kind else None,
            "target_path": self.target_path,
            "target": self.target.as_dict() if self.target else None,
            "target_is_legacy_alias": self.target_is_legacy_alias,
            "requires_temporary_target": self.requires_temporary_target,
            "intermediate_profiles": [],
            "snapshot": self.snapshot.as_dict(),
        }


@dataclass(frozen=True)
class DirectConvergenceDiscovery:
    state: DirectConvergenceState | None
    diagnostics: tuple[ConvergenceDiagnostic, ...]

    @property
    def valid(self) -> bool:
        return self.state is not None and not self.diagnostics

    def as_dict(self) -> dict[str, object]:
        return {
            "valid": self.valid,
            "state": self.state.as_dict() if self.state else None,
            "diagnostics": [item.as_dict() for item in self.diagnostics],
        }


def _diagnostic(code: str, message: str, path: str | None = None) -> ConvergenceDiagnostic:
    return ConvergenceDiagnostic(code, message, path)


def _read_binding(
    root: Path,
    relative_path: str,
    *,
    temporary: bool,
) -> tuple[ConvergenceProfileBinding | None, list[ConvergenceDiagnostic]]:
    normalized = normalize_profile_path(relative_path)
    try:
        path = profile_path_on_disk(root, normalized)
        raw = path.read_bytes()
    except (OSError, ValueError) as exc:
        return None, [_diagnostic("PP_CONVERGENCE_PROFILE_READ_ERROR", str(exc), normalized)]

    try:
        payload = yaml.safe_load(raw.decode("utf-8-sig"))
    except (UnicodeError, yaml.YAMLError) as exc:
        return None, [_diagnostic("PP_CONVERGENCE_INVALID_YAML", str(exc), normalized)]
    if not isinstance(payload, dict):
        return None, [_diagnostic("PP_CONVERGENCE_INVALID_PROFILE", "Profile root must be a mapping.", normalized)]

    ptsip = payload.get("ptsip")
    if not isinstance(ptsip, dict):
        return None, [_diagnostic("PP_CONVERGENCE_MISSING_IDENTITY", "Profile has no ptsip metadata.", normalized)]

    declared = ptsip.get("version")
    specification = ptsip.get("specification")
    revision = specification.get("revision") if isinstance(specification, dict) else None
    source_value = specification.get("source") if isinstance(specification, dict) else None
    source = source_value.strip() if isinstance(source_value, str) and source_value.strip() else None

    diagnostics: list[ConvergenceDiagnostic] = []
    if not isinstance(declared, str) or not declared.strip():
        diagnostics.append(_diagnostic("PP_CONVERGENCE_MISSING_IDENTITY", "Profile has no ptsip.version.", normalized))
    if not isinstance(revision, str) or not revision.strip():
        diagnostics.append(_diagnostic("PP_CONVERGENCE_MISSING_SPEC_REVISION", "Profile has no immutable specification revision.", normalized))
    if diagnostics:
        return None, diagnostics

    return ConvergenceProfileBinding(
        path=normalized,
        declared_version=declared.strip(),
        specification_revision=revision.strip(),
        specification_source=source,
        content_sha256=hashlib.sha256(raw).hexdigest(),
        temporary=temporary,
    ), []


def _temporary_candidates(root: Path) -> tuple[str, ...]:
    try:
        entries = root.iterdir()
    except OSError:
        return ()
    return tuple(
        sorted(
            item.name
            for item in entries
            if item.is_file() and item.name.startswith("ptsip_") and item.name.endswith(".yaml")
        )
    )


def _historical_source_state(
    root: Path,
    source: ConvergenceProfileBinding,
    bridge: HistoricalProjectProfileBridge,
) -> DirectConvergenceDiscovery:
    current_target = current_project_profile_target()
    if bridge.target_contract != current_target.contract:
        return DirectConvergenceDiscovery(
            None,
            (
                _diagnostic(
                    "PP_CONVERGENCE_TARGET_MISMATCH",
                    (
                        f"Historical source {source.declared_version!r} targets "
                        f"{bridge.target_contract.canonical!r}, but Tool "
                        f"{current_target.tool_version!r} selects {current_target.contract.canonical!r}."
                    ),
                    source.path,
                ),
            ),
        )

    candidates = _temporary_candidates(root)
    equivalent_names = set(bridge.equivalent_target_filenames)
    equivalent_present = [name for name in candidates if name in equivalent_names]
    unsupported = [name for name in candidates if name not in equivalent_names]

    diagnostics: list[ConvergenceDiagnostic] = []
    for name in unsupported:
        diagnostics.append(
            _diagnostic(
                "PP_CONVERGENCE_SYNTHETIC_INTERMEDIATE",
                (
                    f"Temporary profile {name!r} is not an authorized representation of current "
                    f"target {current_target.contract.canonical!r}; direct convergence does not replay intermediate versions."
                ),
                name,
            )
        )
    if len(equivalent_present) > 1:
        diagnostics.append(
            _diagnostic(
                "DUPLICATE_EQUIVALENT_TARGET",
                (
                    f"Equivalent target {current_target.contract.canonical!r} is represented by multiple paths: "
                    + ", ".join(equivalent_present)
                ),
            )
        )
    if diagnostics:
        return DirectConvergenceDiscovery(None, tuple(diagnostics))

    target: ConvergenceProfileBinding | None = None
    target_path = current_target.temporary_filename
    target_is_legacy_alias = False
    if equivalent_present:
        target_path = equivalent_present[0]
        target, target_diagnostics = _read_binding(root, target_path, temporary=True)
        if target_diagnostics:
            return DirectConvergenceDiscovery(None, tuple(target_diagnostics))
        assert target is not None
        target_is_legacy_alias = target_path != current_target.temporary_filename
        allowed_declared = {current_target.contract.canonical}
        if target_is_legacy_alias and target_path == "ptsip_0.3.6.yaml":
            allowed_declared.add("0.3.6-draft")
        if target.declared_version not in allowed_declared:
            return DirectConvergenceDiscovery(
                None,
                (
                    _diagnostic(
                        "PP_CONVERGENCE_TARGET_IDENTITY_MISMATCH",
                        (
                            f"Target path {target_path!r} represents {current_target.contract.canonical!r} "
                            f"but declares {target.declared_version!r}."
                        ),
                        target_path,
                    ),
                ),
            )

    if (
        bridge.transition_kind is ProjectProfileTransitionKind.IDENTITY_ONLY
        and source.path == DEFAULT_PROFILE_PATH
        and target is None
    ):
        return DirectConvergenceDiscovery(
            DirectConvergenceState(
                mode=DirectConvergenceMode.IDENTITY_ONLY,
                source=source,
                source_compatibility_contract=bridge.compatibility_contract,
                target_contract=current_target.contract,
                transition_kind=bridge.transition_kind,
                target_path=DEFAULT_PROFILE_PATH,
                target=None,
                target_is_legacy_alias=False,
                requires_temporary_target=False,
                snapshot=capture_snapshot(root),
            ),
            (),
        )

    return DirectConvergenceDiscovery(
        DirectConvergenceState(
            mode=DirectConvergenceMode.DIRECT_SEMANTIC_MIGRATION,
            source=source,
            source_compatibility_contract=bridge.compatibility_contract,
            target_contract=current_target.contract,
            transition_kind=bridge.transition_kind,
            target_path=target_path,
            target=target,
            target_is_legacy_alias=target_is_legacy_alias,
            requires_temporary_target=target is None,
            snapshot=capture_snapshot(root),
        ),
        (),
    )


def discover_direct_profile_convergence(
    repository_root: str | Path,
) -> DirectConvergenceDiscovery:
    root = Path(repository_root).expanduser().resolve()
    canonical_path = root / DEFAULT_PROFILE_PATH
    if not canonical_path.is_file():
        return DirectConvergenceDiscovery(
            None,
            (_diagnostic("NO_CANONICAL_PROFILE", f"Repository has no canonical {DEFAULT_PROFILE_PATH}.", DEFAULT_PROFILE_PATH),),
        )

    source, diagnostics = _read_binding(root, DEFAULT_PROFILE_PATH, temporary=False)
    if diagnostics or source is None:
        return DirectConvergenceDiscovery(None, tuple(diagnostics))

    bridge = historical_project_profile_bridge(
        source.declared_version,
        source.specification_revision,
    )
    if bridge is not None:
        return _historical_source_state(root, source, bridge)

    try:
        contract = ProjectProfileVersion.parse(source.declared_version, require_canonical=True)
        require_project_profile_support(
            PP_COMPATIBILITY_TARGET_TOOL_VERSION,
            contract,
            ProjectProfileOperation.ANALYZE,
        )
    except ProjectProfileIdentityError as exc:
        return DirectConvergenceDiscovery(
            None,
            (_diagnostic(exc.code, str(exc), source.path),),
        )

    current_target = current_project_profile_target()
    if contract != current_target.contract:
        return DirectConvergenceDiscovery(
            None,
            (
                _diagnostic(
                    "PP_CONVERGENCE_UNSUPPORTED_SOURCE_TARGET",
                    (
                        f"Tool {current_target.tool_version!r} cannot directly converge canonical "
                        f"{contract.canonical!r} to {current_target.contract.canonical!r}."
                    ),
                    source.path,
                ),
            ),
        )

    candidates = _temporary_candidates(root)
    if candidates:
        return DirectConvergenceDiscovery(
            None,
            tuple(
                _diagnostic(
                    "PP_CONVERGENCE_UNEXPECTED_TEMPORARY",
                    (
                        f"Canonical profile is already at current target {contract.canonical!r}; "
                        f"temporary profile {name!r} is ambiguous."
                    ),
                    name,
                )
                for name in candidates
            ),
        )

    return DirectConvergenceDiscovery(
        DirectConvergenceState(
            mode=DirectConvergenceMode.CURRENT,
            source=source,
            source_compatibility_contract=contract,
            target_contract=contract,
            transition_kind=None,
            target_path=DEFAULT_PROFILE_PATH,
            target=None,
            target_is_legacy_alias=False,
            requires_temporary_target=False,
            snapshot=capture_snapshot(root),
        ),
        (),
    )


def validate_direct_convergence_snapshot(
    repository_root: str | Path,
    state: DirectConvergenceState,
) -> tuple[ConvergenceDiagnostic, ...]:
    root = Path(repository_root).expanduser().resolve()
    comparison = compare_snapshots(state.snapshot, capture_snapshot(root))
    reasons = list(comparison.reasons)

    for binding in (state.source, state.target):
        if binding is None:
            continue
        try:
            path = profile_path_on_disk(root, binding.path)
            digest = hashlib.sha256(path.read_bytes()).hexdigest()
        except (OSError, ValueError) as exc:
            reasons.append(f"{binding.path}: unavailable ({exc})")
            continue
        if digest != binding.content_sha256:
            reasons.append(f"{binding.path}: profile content changed")

    if not reasons:
        return ()
    return (
        _diagnostic(
            "STALE_DIRECT_CONVERGENCE_SNAPSHOT",
            "Direct convergence snapshot is stale: " + "; ".join(dict.fromkeys(reasons)),
        ),
    )
