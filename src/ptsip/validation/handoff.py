"""PTSIP-owned validated, immutable effective-map handoff for read-only consumers."""
from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from types import MappingProxyType
from weakref import WeakKeyDictionary

from .profile import validate_profile


class EffectiveMapValidationError(ValueError):
    """No usable validated effective-map handoff could be produced."""


def _freeze(value):
    if isinstance(value, dict):
        return MappingProxyType({key: _freeze(item) for key, item in value.items()})
    if isinstance(value, list):
        return tuple(_freeze(item) for item in value)
    return value


@dataclass(frozen=True)
class _EffectiveMapData:
    payload: object
    source_mode: str
    profile_path: str


_EFFECTIVE_MAPS: WeakKeyDictionary = WeakKeyDictionary()


class ValidatedEffectiveMap:
    """Opaque factory provenance, not a sandbox against hostile Python code.

    Neither a public boolean nor direct construction can assert validation.
    This is profile-validation evidence, never a PTSIP conformance verdict.
    """

    __slots__ = ("__weakref__",)

    def __init__(self):
        raise TypeError("Use load_validated_effective_map; direct construction is not validation proof.")

    @property
    def effective_payload(self):
        return _effective_map_data(self).payload

    @property
    def source_mode(self):
        return _effective_map_data(self).source_mode

    @property
    def profile_path(self):
        return _effective_map_data(self).profile_path


def _effective_map_data(value):
    if type(value) is not ValidatedEffectiveMap or value not in _EFFECTIVE_MAPS:
        raise EffectiveMapValidationError("UNVALIDATED_EFFECTIVE_MAP: Validated PTSIP provider handoff is required.")
    return _EFFECTIVE_MAPS[value]


def load_validated_effective_map(repository_root: str | Path, explicit: str | Path | None = None) -> ValidatedEffectiveMap:
    """Validate one source snapshot and freeze that exact resolved effective map."""
    result = validate_profile(repository_root, explicit)
    if not result.valid or result.errors or result.resolved_profile is None:
        raise EffectiveMapValidationError("PTSIP profile validation failed: " + "; ".join(result.errors))
    resolved = result.resolved_profile
    handoff = object.__new__(ValidatedEffectiveMap)
    _EFFECTIVE_MAPS[handoff] = _EffectiveMapData(
        _freeze(resolved.effective_payload), resolved.source_mode, result.profile_path,
    )
    return handoff
