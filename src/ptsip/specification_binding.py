from __future__ import annotations

import re
from dataclasses import dataclass
from enum import Enum
from urllib.parse import urlparse


_FAMILY_PATTERN = re.compile(r"^[0-9]+\.[0-9]+\.[0-9]+(?:-[A-Za-z0-9][A-Za-z0-9.-]*)?$")
_REVISION_PATTERN = re.compile(r"^[0-9a-f]{40}$")

SPECIFICATION_TARGET_TOOL_VERSION = "0.3.7"
SPECIFICATION_SOURCE = "https://github.com/Kinirin/PTSIP"
SPECIFICATION_036_FAMILY = "0.3.6-draft"
SPECIFICATION_036_REVISION = "d6995ed232e845b88d8235b851e80ab54b7804ea"
SPECIFICATION_037_FAMILY = "0.3.7-draft"
# Final WU-12 immutable normative Specification snapshot.
SPECIFICATION_037_REVISION = "3c47816770d194ae42f98faedc911d980db0e62a"


class SpecificationBindingError(ValueError):
    """Stable fail-closed error for Specification identity and capability checks."""

    def __init__(self, code: str, message: str, value: object = None) -> None:
        super().__init__(message)
        self.code = code
        self.value = value


@dataclass(frozen=True, order=True)
class SpecificationBinding:
    """Exact Specification identity independent from Tool and Project Profile versions."""

    family: str
    source: str
    revision: str

    def __post_init__(self) -> None:
        family = self.family
        source = self.source
        revision = self.revision
        if not isinstance(family, str) or not family or family != family.strip():
            raise SpecificationBindingError(
                "SPEC_BINDING_FAMILY_INVALID",
                "Specification family must be a non-empty canonical string without surrounding whitespace.",
                family,
            )
        if _FAMILY_PATTERN.fullmatch(family) is None:
            raise SpecificationBindingError(
                "SPEC_BINDING_FAMILY_MALFORMED",
                f"Specification family {family!r} is not a canonical version-family identity.",
                family,
            )
        if not isinstance(source, str) or not source or source != source.strip():
            raise SpecificationBindingError(
                "SPEC_BINDING_SOURCE_INVALID",
                "Specification source must be a non-empty absolute URI without surrounding whitespace.",
                source,
            )
        parsed = urlparse(source)
        if not parsed.scheme or not parsed.netloc:
            raise SpecificationBindingError(
                "SPEC_BINDING_SOURCE_MALFORMED",
                f"Specification source {source!r} must be an absolute URI.",
                source,
            )
        if not isinstance(revision, str) or not revision or revision != revision.strip():
            raise SpecificationBindingError(
                "SPEC_BINDING_REVISION_INVALID",
                "Specification revision must be a non-empty immutable revision without surrounding whitespace.",
                revision,
            )
        if _REVISION_PATTERN.fullmatch(revision) is None:
            raise SpecificationBindingError(
                "SPEC_BINDING_REVISION_MALFORMED",
                "Specification revision must be a 40-character lowercase hexadecimal immutable Git revision.",
                revision,
            )

    @classmethod
    def from_mapping(cls, value: object) -> "SpecificationBinding":
        if not isinstance(value, dict):
            raise SpecificationBindingError(
                "SPEC_BINDING_TYPE",
                "Specification binding must be a mapping with family, source, and revision fields.",
                value,
            )
        required = {"family", "source", "revision"}
        actual = set(value)
        missing = sorted(required - actual)
        extra = sorted(actual - required)
        if missing:
            raise SpecificationBindingError(
                "SPEC_BINDING_FIELD_MISSING",
                "Specification binding is missing required field(s): " + ", ".join(missing),
                value,
            )
        if extra:
            raise SpecificationBindingError(
                "SPEC_BINDING_FIELD_UNEXPECTED",
                "Specification binding contains unsupported field(s): " + ", ".join(extra),
                value,
            )
        return cls(
            family=value["family"],
            source=value["source"],
            revision=value["revision"],
        )

    def as_dict(self) -> dict[str, str]:
        return {
            "family": self.family,
            "source": self.source,
            "revision": self.revision,
        }


class SpecificationOperation(str, Enum):
    IDENTIFY = "IDENTIFY"
    VALIDATE = "VALIDATE"
    ANALYZE = "ANALYZE"
    CONFORM = "CONFORM"
    CREATE_TARGET = "CREATE_TARGET"


@dataclass(frozen=True)
class SpecificationSupport:
    tool_version: str
    binding: SpecificationBinding
    operations: frozenset[SpecificationOperation]

    def supports(self, operation: SpecificationOperation) -> bool:
        return operation in self.operations


SPECIFICATION_036 = SpecificationBinding(
    SPECIFICATION_036_FAMILY,
    SPECIFICATION_SOURCE,
    SPECIFICATION_036_REVISION,
)
SPECIFICATION_037 = SpecificationBinding(
    SPECIFICATION_037_FAMILY,
    SPECIFICATION_SOURCE,
    SPECIFICATION_037_REVISION,
)


_TOOL_SPECIFICATION_SUPPORT: dict[tuple[str, SpecificationBinding], SpecificationSupport] = {
    (
        SPECIFICATION_TARGET_TOOL_VERSION,
        SPECIFICATION_036,
    ): SpecificationSupport(
        tool_version=SPECIFICATION_TARGET_TOOL_VERSION,
        binding=SPECIFICATION_036,
        operations=frozenset(
            {
                SpecificationOperation.IDENTIFY,
                SpecificationOperation.VALIDATE,
                SpecificationOperation.ANALYZE,
                SpecificationOperation.CONFORM,
            }
        ),
    ),
    (
        SPECIFICATION_TARGET_TOOL_VERSION,
        SPECIFICATION_037,
    ): SpecificationSupport(
        tool_version=SPECIFICATION_TARGET_TOOL_VERSION,
        binding=SPECIFICATION_037,
        operations=frozenset(SpecificationOperation),
    ),
}


def specification_support(
    tool_version: str,
    binding: SpecificationBinding | object,
) -> SpecificationSupport | None:
    parsed = binding if isinstance(binding, SpecificationBinding) else SpecificationBinding.from_mapping(binding)
    return _TOOL_SPECIFICATION_SUPPORT.get((tool_version, parsed))


def require_specification_support(
    tool_version: str,
    binding: SpecificationBinding | object,
    operation: SpecificationOperation,
) -> SpecificationSupport:
    parsed = binding if isinstance(binding, SpecificationBinding) else SpecificationBinding.from_mapping(binding)
    support = _TOOL_SPECIFICATION_SUPPORT.get((tool_version, parsed))
    if support is None or not support.supports(operation):
        raise SpecificationBindingError(
            "SPEC_BINDING_UNSUPPORTED",
            f"Tool {tool_version!r} does not support Specification {parsed.family!r} "
            f"at {parsed.revision!r} for {operation.value}.",
            parsed.as_dict(),
        )
    return support


def require_current_specification_support(
    binding: SpecificationBinding | object,
    operation: SpecificationOperation,
) -> SpecificationSupport:
    return require_specification_support(
        SPECIFICATION_TARGET_TOOL_VERSION,
        binding,
        operation,
    )


def current_target_specification_binding() -> SpecificationBinding:
    return SPECIFICATION_037
