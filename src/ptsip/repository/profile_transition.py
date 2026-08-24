from __future__ import annotations

import hashlib
import re
from dataclasses import asdict, dataclass
from pathlib import Path

import yaml

from .profile_path import DEFAULT_PROFILE_PATH, normalize_profile_path, profile_path_on_disk
from .snapshot import RepositorySnapshot, capture_snapshot, compare_snapshots

_TEMPORARY_PROFILE_RE = re.compile(r"^ptsip_(\d+)\.(\d+)\.(\d+)\.yaml$")
_DRAFT_VERSION_RE = re.compile(r"^(\d+)\.(\d+)\.(\d+)-draft$")
_TEMPORARY_PROFILE_PREFIX = "ptsip_"
_TEMPORARY_PROFILE_SUFFIX = ".yaml"


@dataclass(frozen=True, order=True)
class DraftVersion:
    major: int
    minor: int
    micro: int

    @classmethod
    def from_draft_label(cls, value: object) -> DraftVersion | None:
        if not isinstance(value, str):
            return None
        match = _DRAFT_VERSION_RE.fullmatch(value.strip())
        if match is None:
            return None
        return cls(*(int(item) for item in match.groups()))

    @property
    def semantic(self) -> str:
        return f"{self.major}.{self.minor}.{self.micro}"

    @property
    def draft_label(self) -> str:
        return f"{self.semantic}-draft"


@dataclass(frozen=True)
class TransitionDiagnostic:
    code: str
    message: str
    path: str | None = None

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


@dataclass(frozen=True)
class ProfileGenerationIdentity:
    path: str
    version: DraftVersion
    declared_version: str
    specification_revision: str
    specification_source: str | None
    content_sha256: str
    temporary: bool

    def as_dict(self) -> dict[str, object]:
        result = asdict(self)
        result["version"] = self.version.semantic
        return result


@dataclass(frozen=True)
class TransitionSnapshot:
    repository_root: str
    repository: RepositorySnapshot
    profiles: tuple[ProfileGenerationIdentity, ...]

    def as_dict(self) -> dict[str, object]:
        return {
            "repository_root": self.repository_root,
            "repository": self.repository.as_dict(),
            "profiles": [item.as_dict() for item in self.profiles],
        }


@dataclass(frozen=True)
class ProfileTransitionState:
    mode: str
    canonical_source: ProfileGenerationIdentity
    temporary_profiles: tuple[ProfileGenerationIdentity, ...]
    final_point: ProfileGenerationIdentity | None
    ordered_sources: tuple[ProfileGenerationIdentity, ...]
    snapshot: TransitionSnapshot

    def as_dict(self) -> dict[str, object]:
        return {
            "mode": self.mode,
            "canonical_source": self.canonical_source.as_dict(),
            "temporary_profiles": [item.as_dict() for item in self.temporary_profiles],
            "final_point": self.final_point.as_dict() if self.final_point else None,
            "ordered_sources": [item.as_dict() for item in self.ordered_sources],
            "snapshot": self.snapshot.as_dict(),
        }


@dataclass(frozen=True)
class ProfileTransitionDiscovery:
    state: ProfileTransitionState | None
    diagnostics: tuple[TransitionDiagnostic, ...]

    @property
    def valid(self) -> bool:
        return self.state is not None and not self.diagnostics

    def as_dict(self) -> dict[str, object]:
        return {
            "valid": self.valid,
            "state": self.state.as_dict() if self.state else None,
            "diagnostics": [item.as_dict() for item in self.diagnostics],
        }


def _diagnostic(code: str, message: str, path: str | None = None) -> TransitionDiagnostic:
    return TransitionDiagnostic(code=code, message=message, path=path)


def _temporary_filename_version(path: Path) -> tuple[DraftVersion, str] | None:
    match = _TEMPORARY_PROFILE_RE.fullmatch(path.name)
    if match is None:
        return None
    token = ".".join(match.groups())
    return DraftVersion(*(int(item) for item in match.groups())), token


def _read_profile_identity(
    repository_root: Path,
    relative_path: str,
    *,
    temporary: bool,
) -> tuple[ProfileGenerationIdentity | None, list[TransitionDiagnostic]]:
    normalized = normalize_profile_path(relative_path)
    diagnostics: list[TransitionDiagnostic] = []
    try:
        path = profile_path_on_disk(repository_root, normalized)
    except ValueError as exc:
        return None, [_diagnostic("PROFILE_PATH_ERROR", str(exc), normalized)]

    try:
        raw = path.read_bytes()
    except OSError as exc:
        return None, [_diagnostic("PROFILE_READ_ERROR", f"Unable to read profile: {exc}", normalized)]

    try:
        payload = yaml.safe_load(raw.decode("utf-8-sig"))
    except Exception as exc:
        return None, [_diagnostic("INVALID_PROFILE_YAML", f"Unable to parse profile YAML: {exc}", normalized)]

    if not isinstance(payload, dict):
        return None, [_diagnostic("INVALID_PROFILE_YAML", "Profile root must be a mapping.", normalized)]

    ptsip = payload.get("ptsip")
    if not isinstance(ptsip, dict):
        return None, [_diagnostic("MISSING_PROFILE_VERSION", "Profile has no ptsip.version identity.", normalized)]

    declared_version = ptsip.get("version")
    version = DraftVersion.from_draft_label(declared_version)
    if declared_version is None or declared_version == "":
        diagnostics.append(_diagnostic("MISSING_PROFILE_VERSION", "Profile has no ptsip.version identity.", normalized))
    elif version is None:
        diagnostics.append(
            _diagnostic(
                "INVALID_PROFILE_VERSION",
                f"ptsip.version must be a <major>.<minor>.<micro>-draft label, got {declared_version!r}.",
                normalized,
            )
        )

    specification = ptsip.get("specification")
    revision: object | None = None
    source: str | None = None
    if isinstance(specification, dict):
        revision = specification.get("revision")
        source_value = specification.get("source")
        source = source_value if isinstance(source_value, str) and source_value.strip() else None
    if not isinstance(revision, str) or not revision.strip():
        diagnostics.append(
            _diagnostic("MISSING_SPEC_REVISION", "Profile has no immutable ptsip.specification.revision.", normalized)
        )

    if temporary:
        filename_identity = _temporary_filename_version(path)
        if filename_identity is None:
            diagnostics.append(
                _diagnostic(
                    "INVALID_TEMPORARY_FILENAME",
                    "Temporary PTSIP Profile File must be named ptsip_<major>.<minor>.<micro>.yaml.",
                    normalized,
                )
            )
        elif isinstance(declared_version, str):
            _filename_version, filename_token = filename_identity
            expected = f"{filename_token}-draft"
            if declared_version.strip() != expected:
                diagnostics.append(
                    _diagnostic(
                        "PROFILE_VERSION_FILENAME_MISMATCH",
                        f"Filename requires ptsip.version {expected!r}, got {declared_version!r}.",
                        normalized,
                    )
                )

    if diagnostics or version is None or not isinstance(revision, str):
        return None, diagnostics

    return (
        ProfileGenerationIdentity(
            path=normalized,
            version=version,
            declared_version=declared_version.strip(),
            specification_revision=revision.strip(),
            specification_source=source,
            content_sha256=hashlib.sha256(raw).hexdigest(),
            temporary=temporary,
        ),
        diagnostics,
    )


def _root_temporary_candidates(repository_root: Path) -> list[Path]:
    try:
        entries = list(repository_root.iterdir())
    except OSError:
        return []
    return sorted(
        (
            item
            for item in entries
            if item.is_file()
            and item.name.startswith(_TEMPORARY_PROFILE_PREFIX)
            and item.name.endswith(_TEMPORARY_PROFILE_SUFFIX)
        ),
        key=lambda item: item.name,
    )


def discover_profile_transition(repository_root: str | Path) -> ProfileTransitionDiscovery:
    root = Path(repository_root).expanduser().resolve()
    diagnostics: list[TransitionDiagnostic] = []
    canonical_path = root / DEFAULT_PROFILE_PATH

    canonical: ProfileGenerationIdentity | None = None
    if not canonical_path.is_file():
        diagnostics.append(
            _diagnostic("NO_CANONICAL_PROFILE", f"Repository has no canonical {DEFAULT_PROFILE_PATH}.", DEFAULT_PROFILE_PATH)
        )
    else:
        canonical, identity_diagnostics = _read_profile_identity(
            root,
            DEFAULT_PROFILE_PATH,
            temporary=False,
        )
        diagnostics.extend(identity_diagnostics)

    temporaries: list[ProfileGenerationIdentity] = []
    candidates = _root_temporary_candidates(root)
    for candidate in candidates:
        filename_identity = _temporary_filename_version(candidate)
        relative = candidate.relative_to(root).as_posix()
        if filename_identity is None:
            diagnostics.append(
                _diagnostic(
                    "INVALID_TEMPORARY_FILENAME",
                    "Temporary PTSIP Profile File must be named ptsip_<major>.<minor>.<micro>.yaml.",
                    relative,
                )
            )
            continue
        identity, identity_diagnostics = _read_profile_identity(root, relative, temporary=True)
        diagnostics.extend(identity_diagnostics)
        if identity is not None:
            temporaries.append(identity)

    by_version: dict[DraftVersion, list[ProfileGenerationIdentity]] = {}
    for temporary in temporaries:
        by_version.setdefault(temporary.version, []).append(temporary)
    duplicate_versions = {version: items for version, items in by_version.items() if len(items) > 1}
    for version, items in sorted(duplicate_versions.items()):
        diagnostics.append(
            _diagnostic(
                "DUPLICATE_TARGET_IDENTITY",
                f"Target semantic version {version.semantic} is represented by multiple temporary profiles: "
                + ", ".join(item.path for item in items),
            )
        )

    if duplicate_versions:
        highest = max(by_version) if by_version else None
        if highest in duplicate_versions:
            diagnostics.append(
                _diagnostic(
                    "AMBIGUOUS_FINAL_POINT",
                    f"Highest target semantic version {highest.semantic} does not identify one Final PTSIP Point File.",
                )
            )

    if canonical is not None:
        non_monotonic = [item for item in temporaries if item.version <= canonical.version]
        for item in sorted(non_monotonic, key=lambda value: (value.version, value.path)):
            diagnostics.append(
                _diagnostic(
                    "NON_MONOTONIC_TARGET",
                    f"Temporary target {item.version.semantic} must be newer than canonical source {canonical.version.semantic}.",
                    item.path,
                )
            )

    if diagnostics or canonical is None:
        return ProfileTransitionDiscovery(state=None, diagnostics=tuple(diagnostics))

    ordered_temporaries = tuple(sorted(temporaries, key=lambda item: (item.version, item.path)))
    if not ordered_temporaries:
        final_point = None
        ordered_sources: tuple[ProfileGenerationIdentity, ...] = ()
        mode = "IDLE"
    else:
        final_point = ordered_temporaries[-1]
        remaining = [item for item in ordered_temporaries if item.path != final_point.path]
        ordered_sources = tuple(sorted(remaining, key=lambda item: item.version, reverse=True) + [canonical])
        mode = "SIMPLE" if len(ordered_temporaries) == 1 else "SEQUENTIAL"

    profile_snapshots = (canonical,) + ordered_temporaries
    snapshot = TransitionSnapshot(
        repository_root=str(root),
        repository=capture_snapshot(root),
        profiles=profile_snapshots,
    )
    return ProfileTransitionDiscovery(
        state=ProfileTransitionState(
            mode=mode,
            canonical_source=canonical,
            temporary_profiles=ordered_temporaries,
            final_point=final_point,
            ordered_sources=ordered_sources,
            snapshot=snapshot,
        ),
        diagnostics=(),
    )


def validate_transition_snapshot(
    repository_root: str | Path,
    state: ProfileTransitionState,
) -> tuple[TransitionDiagnostic, ...]:
    root = Path(repository_root).expanduser().resolve()
    reasons: list[str] = []
    if str(root) != state.snapshot.repository_root:
        reasons.append("repository root changed")
    repository_comparison = compare_snapshots(state.snapshot.repository, capture_snapshot(root))
    reasons.extend(repository_comparison.reasons)

    for expected in state.snapshot.profiles:
        path = profile_path_on_disk(root, expected.path)
        try:
            raw = path.read_bytes()
        except OSError as exc:
            reasons.append(f"{expected.path}: profile is unavailable ({exc})")
            continue
        current_sha256 = hashlib.sha256(raw).hexdigest()
        if current_sha256 != expected.content_sha256:
            reasons.append(f"{expected.path}: profile content changed")

    if not reasons:
        return ()
    unique_reasons = tuple(dict.fromkeys(reasons))
    return (
        _diagnostic(
            "STALE_TRANSITION_SNAPSHOT",
            "Transition snapshot is stale: " + "; ".join(unique_reasons),
        ),
    )
