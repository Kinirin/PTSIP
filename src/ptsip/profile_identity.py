from __future__ import annotations

import hashlib
import re
from dataclasses import dataclass
from enum import Enum

from .project_profile_contracts import (
    ProjectProfileContractRegistryError,
    current_runtime_project_profile_contract,
    current_runtime_project_profile_version,
)


_PP_PATTERN = re.compile(r"^pp\.(\d+)\.(\d+)$")
_PP_FILENAME_TOKEN_PATTERN = re.compile(r"^pp(\d+)\.(\d+)$")
_USER_REVISION_PATTERN = re.compile(r"^Rev\.(\d{4})$")


class ProjectProfileIdentityError(ValueError):
    """Stable fail-closed error for Project Profile contract identity."""

    def __init__(self, code: str, message: str, value: object = None) -> None:
        super().__init__(message)
        self.code = code
        self.value = value


@dataclass(frozen=True, order=True)
class ProjectProfileVersion:
    """Typed Project Profile contract identity independent from Tool SemVer."""

    major: int
    minor: int

    def __post_init__(self) -> None:
        if self.major < 0 or self.minor < 0:
            raise ProjectProfileIdentityError(
                "PP_IDENTITY_NEGATIVE_SEGMENT",
                "Project Profile major/minor segments must be non-negative integers.",
                (self.major, self.minor),
            )

    @classmethod
    def parse(cls, value: object, *, require_canonical: bool = False) -> ProjectProfileVersion:
        if not isinstance(value, str):
            raise ProjectProfileIdentityError(
                "PP_IDENTITY_TYPE",
                "Project Profile identity must be a string in pp.<major>.<minor> form.",
                value,
            )
        text = value.strip()
        match = _PP_PATTERN.fullmatch(text)
        if match is None:
            raise ProjectProfileIdentityError(
                "PP_IDENTITY_MALFORMED",
                f"Project Profile identity must use pp.<major>.<minor>, got {value!r}.",
                value,
            )
        version = cls(int(match.group(1)), int(match.group(2)))
        if require_canonical and text != version.canonical:
            raise ProjectProfileIdentityError(
                "PP_IDENTITY_NON_CANONICAL",
                f"Project Profile identity {text!r} is non-canonical; use {version.canonical!r}.",
                value,
            )
        return version

    @classmethod
    def from_filename_token(
        cls,
        value: object,
        *,
        require_canonical: bool = False,
    ) -> ProjectProfileVersion:
        if not isinstance(value, str):
            raise ProjectProfileIdentityError(
                "PP_FILENAME_TOKEN_TYPE",
                "Project Profile filename token must be a string in pp<major>.<minor> form.",
                value,
            )
        text = value.strip()
        match = _PP_FILENAME_TOKEN_PATTERN.fullmatch(text)
        if match is None:
            raise ProjectProfileIdentityError(
                "PP_FILENAME_TOKEN_MALFORMED",
                f"Project Profile filename token must use pp<major>.<minor>, got {value!r}.",
                value,
            )
        version = cls(int(match.group(1)), int(match.group(2)))
        if require_canonical and text != version.filename_token:
            raise ProjectProfileIdentityError(
                "PP_FILENAME_TOKEN_NON_CANONICAL",
                f"Project Profile filename token {text!r} is non-canonical; use {version.filename_token!r}.",
                value,
            )
        return version

    @property
    def canonical(self) -> str:
        return f"pp.{self.major}.{self.minor:02d}"

    @property
    def filename_token(self) -> str:
        return f"pp{self.major}.{self.minor:02d}"

    def is_canonical_text(self, value: object) -> bool:
        return isinstance(value, str) and value.strip() == self.canonical

    def __str__(self) -> str:
        return self.canonical


@dataclass(frozen=True, order=True)
class ProjectProfileUserRevision:
    """User-owned authoritative generation within one Project Profile contract."""

    number: int

    def __post_init__(self) -> None:
        if self.number < 1 or self.number > 9999:
            raise ProjectProfileIdentityError(
                "PP_USER_REVISION_RANGE",
                "Project Profile user revision must be between Rev.0001 and Rev.9999.",
                self.number,
            )

    @classmethod
    def parse(cls, value: object) -> "ProjectProfileUserRevision":
        if not isinstance(value, str):
            raise ProjectProfileIdentityError(
                "PP_USER_REVISION_TYPE",
                "Project Profile user revision must use Rev.#### form.",
                value,
            )
        match = _USER_REVISION_PATTERN.fullmatch(value)
        if match is None:
            raise ProjectProfileIdentityError(
                "PP_USER_REVISION_MALFORMED",
                "Project Profile user revision must use canonical Rev.#### form.",
                value,
            )
        return cls(int(match.group(1)))

    @property
    def canonical(self) -> str:
        return f"Rev.{self.number:04d}"

    def next(self) -> "ProjectProfileUserRevision":
        return ProjectProfileUserRevision(self.number + 1)


DEVELOPER_BASELINE_USER_REVISION = ProjectProfileUserRevision(1)


@dataclass(frozen=True)
class ProjectProfileInstanceRevision:
    """Immutable identity of one concrete Project Profile declaration instance."""

    value: str

    def __post_init__(self) -> None:
        if not isinstance(self.value, str) or not self.value.strip():
            raise ProjectProfileIdentityError(
                "PP_INSTANCE_REVISION_INVALID",
                "Project Profile instance revision must be a non-empty immutable identity.",
                self.value,
            )
        object.__setattr__(self, "value", self.value.strip())

    @classmethod
    def from_content(cls, content: bytes) -> ProjectProfileInstanceRevision:
        return cls(f"sha256:{hashlib.sha256(content).hexdigest()}")


class ProjectProfileTransitionKind(str, Enum):
    IDENTITY_ONLY = "IDENTITY_ONLY"
    SEMANTIC_MIGRATION = "SEMANTIC_MIGRATION"


class ProjectProfileOperation(str, Enum):
    IDENTIFY = "IDENTIFY"
    VALIDATE = "VALIDATE"
    ANALYZE = "ANALYZE"
    MIGRATE_SOURCE = "MIGRATE_SOURCE"
    CREATE_TARGET = "CREATE_TARGET"


@dataclass(frozen=True)
class ProjectProfileSupport:
    tool_version: str
    contract: ProjectProfileVersion
    operations: frozenset[ProjectProfileOperation]
    schema_resource: str | None = None

    def supports(self, operation: ProjectProfileOperation) -> bool:
        return operation in self.operations


PP_0_00 = ProjectProfileVersion(0, 0)
PP_1_01 = ProjectProfileVersion(1, 1)
try:
    CURRENT_PROJECT_PROFILE_VERSION = current_runtime_project_profile_version()
except ProjectProfileContractRegistryError as exc:
    raise ProjectProfileIdentityError(
        "PP_RUNTIME_REGISTRY_INVALID",
        str(exc),
    ) from exc
PP_COMPATIBILITY_TARGET_TOOL_VERSION = "0.3.7"


# This registry is intentionally operation-specific. It does not map historical
# Tool-numbered labels to PP generations; that semantic bridge belongs to WU-10.
_TOOL_PP_SUPPORT: dict[tuple[str, ProjectProfileVersion], ProjectProfileSupport] = {
    (
        PP_COMPATIBILITY_TARGET_TOOL_VERSION,
        PP_0_00,
    ): ProjectProfileSupport(
        tool_version=PP_COMPATIBILITY_TARGET_TOOL_VERSION,
        contract=PP_0_00,
        operations=frozenset({ProjectProfileOperation.IDENTIFY}),
    ),
    (
        PP_COMPATIBILITY_TARGET_TOOL_VERSION,
        PP_1_01,
    ): ProjectProfileSupport(
        tool_version=PP_COMPATIBILITY_TARGET_TOOL_VERSION,
        contract=PP_1_01,
        operations=frozenset(
            {
                ProjectProfileOperation.IDENTIFY,
                ProjectProfileOperation.VALIDATE,
                ProjectProfileOperation.ANALYZE,
                ProjectProfileOperation.CREATE_TARGET,
            }
        ),
        schema_resource="ptsip-profile-pp-1.01.schema.json",
    ),
}


def project_profile_support(
    tool_version: str,
    contract: ProjectProfileVersion | str,
) -> ProjectProfileSupport | None:
    version = (
        ProjectProfileVersion.parse(contract, require_canonical=True)
        if isinstance(contract, str)
        else contract
    )
    return _TOOL_PP_SUPPORT.get((tool_version, version))


def require_project_profile_support(
    tool_version: str,
    contract: ProjectProfileVersion | str,
    operation: ProjectProfileOperation,
) -> ProjectProfileSupport:
    version = (
        ProjectProfileVersion.parse(contract, require_canonical=True)
        if isinstance(contract, str)
        else contract
    )
    support = _TOOL_PP_SUPPORT.get((tool_version, version))
    if support is None or not support.supports(operation):
        raise ProjectProfileIdentityError(
            "PP_IDENTITY_UNSUPPORTED",
            f"Tool {tool_version!r} does not support Project Profile {version.canonical!r} for {operation.value}.",
            version.canonical,
        )
    return support


def require_current_project_profile_support(
    contract: ProjectProfileVersion | str,
    operation: ProjectProfileOperation,
) -> ProjectProfileSupport:
    """Validate PP support from the embedded current-contract registry."""

    version = (
        ProjectProfileVersion.parse(contract, require_canonical=True)
        if isinstance(contract, str)
        else contract
    )
    try:
        runtime = current_runtime_project_profile_contract()
        current = ProjectProfileVersion.parse(
            runtime.version,
            require_canonical=True,
        )
        operations = frozenset(ProjectProfileOperation(item) for item in runtime.operations)
    except (ProjectProfileContractRegistryError, ValueError) as exc:
        raise ProjectProfileIdentityError(
            "PP_RUNTIME_REGISTRY_INVALID",
            f"Embedded Project Profile contract registry is invalid: {exc}",
            CURRENT_PROJECT_PROFILE_VERSION,
        ) from exc

    if version != current or operation not in operations:
        raise ProjectProfileIdentityError(
            "PP_IDENTITY_UNSUPPORTED",
            (
                f"Current Tool runtime does not support Project Profile "
                f"{version.canonical!r} for {operation.value}."
            ),
            version.canonical,
        )

    return ProjectProfileSupport(
        tool_version=PP_COMPATIBILITY_TARGET_TOOL_VERSION,
        contract=current,
        operations=operations,
        schema_resource=runtime.schema_resource,
    )
