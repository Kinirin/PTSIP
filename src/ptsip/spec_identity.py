from __future__ import annotations

from dataclasses import asdict, dataclass

from .constants import SPEC_ACRONYM, SPEC_NAME, SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION, TOOL_VERSION


@dataclass(frozen=True)
class SpecIdentity:
    name: str = SPEC_NAME
    acronym: str = SPEC_ACRONYM
    version: str = SPEC_VERSION
    source: str = SPEC_SOURCE
    revision: str = SPEC_REVISION
    tool_version: str = TOOL_VERSION

    def as_dict(self) -> dict[str, str]:
        return asdict(self)


def current_spec_identity() -> SpecIdentity:
    return SpecIdentity()
