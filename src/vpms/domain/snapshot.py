"""Factory-validated immutable registry handoff, separate from legacy Registry."""
from __future__ import annotations

from dataclasses import dataclass
from types import MappingProxyType
from typing import Mapping
from weakref import WeakKeyDictionary

from .. import contract_runtime as contracts
from .registry import (
    Registry, RegistryDiagnostic, RegistryDiagnosticCode, RegistryReferenceIndex,
    load_registry,
)


@dataclass(frozen=True)
class _SnapshotData:
    registry: Registry
    case_index: Mapping


_SNAPSHOTS: WeakKeyDictionary = WeakKeyDictionary()
_NAMESPACES = ("targets", "formulas", "variables", "policies", "runners")


class ValidatedRegistrySnapshot:
    """Opaque factory provenance plus an owned immutable Case/reference snapshot.

    This is a handoff validation boundary, not a sandbox against hostile Python
    code that can replace module internals.
    """

    __slots__ = ("__weakref__",)

    def __init__(self):
        raise TypeError("Use load_registry_snapshot; direct construction is not validation proof.")

    @property
    def cases(self):
        return _snapshot_data(self).registry.cases

    @property
    def references(self):
        return _snapshot_data(self).registry.references

    def get_case(self, identity: str):
        return _snapshot_data(self).case_index.get(identity)

    def as_dict(self):
        return _snapshot_data(self).registry.as_dict()


def _snapshot_data(snapshot) -> _SnapshotData:
    if type(snapshot) is not ValidatedRegistrySnapshot or snapshot not in _SNAPSHOTS:
        raise ValueError("UNVALIDATED_REGISTRY")
    return _SNAPSHOTS[snapshot]


@dataclass(frozen=True)
class SnapshotLoadResult:
    snapshot: ValidatedRegistrySnapshot | None
    diagnostics: tuple[RegistryDiagnostic, ...]

    @property
    def ok(self):
        return self.snapshot is not None and not self.diagnostics


def load_registry_snapshot(definitions: object, *, references: object) -> SnapshotLoadResult:
    """Validate raw explicit registrations before producing a usable handoff."""
    contract = contracts.require_active_contract("protocol")
    handoff = contract["semantics"]["registry_handoff"]
    if not handoff["immutable_snapshot_required"] or handoff["direct_construction_is_validation_proof"]:
        raise contracts.ContractUnavailable("UNSUPPORTED_REGISTRY_HANDOFF")
    diagnostics = []
    if not isinstance(references, Mapping):
        return SnapshotLoadResult(None, (RegistryDiagnostic(
            RegistryDiagnosticCode.MALFORMED_DEFINITIONS, "$.references",
            "Reference registrations must be an explicit namespace mapping.",
        ),))
    for name in references:
        if name not in _NAMESPACES:
            diagnostics.append(RegistryDiagnostic(
                RegistryDiagnosticCode.UNKNOWN_FIELD, "$.references",
                "Unknown reference namespace.",
            ))
    bound = {}
    for name in _NAMESPACES:
        values = references.get(name, ())
        if not isinstance(values, (list, tuple)):
            diagnostics.append(RegistryDiagnostic(
                RegistryDiagnosticCode.INVALID_FIELD, f"$.references.{name}",
                "Reference registrations must be a list or tuple of explicit ids.",
            ))
            continue
        valid = tuple(values)
        if any(not isinstance(value, str) or not value or value.strip() != value for value in valid):
            diagnostics.append(RegistryDiagnostic(
                RegistryDiagnosticCode.INVALID_FIELD, f"$.references.{name}",
                "Reference identities must be non-empty, unpadded strings.",
            ))
        elif len(set(valid)) != len(valid):
            diagnostics.append(RegistryDiagnostic(
                RegistryDiagnosticCode.INVALID_FIELD, f"$.references.{name}",
                "Duplicate reference registrations are not usable.",
            ))
        else:
            bound[name] = tuple(sorted(valid))
    if diagnostics:
        return SnapshotLoadResult(None, tuple(sorted(diagnostics, key=lambda d: (d.location, d.code.value))))
    loaded = load_registry(definitions, references=RegistryReferenceIndex(**bound))
    if not loaded.ok:
        return SnapshotLoadResult(None, loaded.diagnostics)
    snapshot = object.__new__(ValidatedRegistrySnapshot)
    _SNAPSHOTS[snapshot] = _SnapshotData(
        loaded.registry, MappingProxyType({case.id: case for case in loaded.registry.cases}),
    )
    return SnapshotLoadResult(snapshot, ())
