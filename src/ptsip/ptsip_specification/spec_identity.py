from __future__ import annotations

from dataclasses import asdict, dataclass, field
from enum import StrEnum
import json
import os
from pathlib import Path, PurePosixPath
import subprocess
import sysconfig
import tomllib


class ProfileNativeError(RuntimeError):
    """The registered native Profile operation could not return a current result."""


_PROFILE_OPERATIONS = frozenset({
    "discovery.profile.contract", "discovery.profile.catalog", "discovery.profile.rules",
    "validation.profile.identity", "validation.profile.transition", "validation.profile.rules",
    "projection.profile.metadata",
    "context.repository.snapshot", "discovery.repository.inspect", "discovery.profile.candidates",
    "validation.repository.snapshot", "validation.profile.path", "validation.profile.transition-state",
    "validation.profile.convergence",
    "projection.migration.source", "projection.migration.coverage",
    "projection.migration.compatibility",
})


def profile_package_root() -> Path:
    configured = os.environ.get("PTSIP_PROFILE_PACKAGE_ROOT")
    root = Path(configured) if configured is not None else Path(__file__).resolve().parents[1]
    if not root.is_absolute() or not root.is_dir():
        raise ProfileNativeError("Profile package root must be an absolute existing directory.")
    return root.resolve()


def _json_object(pairs: list[tuple[str, object]]) -> dict[str, object]:
    value: dict[str, object] = {}
    for key, item in pairs:
        if key in value:
            raise ProfileNativeError(f"Duplicate native JSON key: {key}")
        value[key] = item
    return value


def _json_payload(content: bytes) -> dict[str, object]:
    def invalid_constant(value: str) -> None:
        raise ProfileNativeError(f"Invalid native JSON constant: {value}")

    value = json.loads(content.decode("utf-8"), object_pairs_hook=_json_object,
                       parse_constant=invalid_constant)
    if not isinstance(value, dict):
        raise ProfileNativeError("Native JSON resource must be an object.")
    return value


def _native_resource(root: Path, reference: str) -> dict[str, object]:
    if not isinstance(reference, str) or not reference:
        raise ProfileNativeError("Native resource requires a non-empty relative path.")
    relative = PurePosixPath(reference.removeprefix("src/ptsip/"))
    if (relative.is_absolute() or ".." in relative.parts or not relative.parts
            or "\\" in reference or ":" in reference):
        raise ProfileNativeError("Native resource must be a package-relative path.")
    path = root.joinpath(*relative.parts).resolve(strict=True)
    if not path.is_relative_to(root) or not path.is_file():
        raise ProfileNativeError("Native resource escapes the selected package.")
    return _json_payload(path.read_bytes())


def native_profile_operation(operation: str, payload: dict[str, object]) -> dict[str, object]:
    """Route a registered Profile or Repository operation through the installed Go CLI."""
    if operation not in _PROFILE_OPERATIONS:
        raise ProfileNativeError("Unsupported native Profile operation.")
    try:
        root = profile_package_root()
        scripts = Path(os.environ.get("PTSIP_ENGINE_SCRIPTS", sysconfig.get_path("scripts")))
        executable = Path(os.environ.get("PTSIP_GO_CLI", str(
            Path(sysconfig.get_path("scripts")) / ("ptsip-go.exe" if os.name == "nt" else "ptsip-go")
        )))
        if not scripts.is_absolute() or not scripts.is_dir():
            raise ProfileNativeError("PTSIP_ENGINE_SCRIPTS must be an absolute existing directory.")
        if not executable.is_absolute() or not executable.is_file():
            raise ProfileNativeError("A configured or installed absolute ptsip-go executable is required.")
        index = _native_resource(root, "engines/index.json")
        entries = index.get("official_engine_entries")
        if not isinstance(entries, list):
            raise ProfileNativeError("Registered native engine entries are required.")
        matches = [row for row in entries if isinstance(row, dict)
                   and isinstance(row.get("operations"), list) and operation in row["operations"]]
        if len(matches) != 1:
            raise ProfileNativeError("Profile operation must have exactly one registered native route.")
        entry = matches[0]
        manifest = _native_resource(root, entry["manifest_ref"])
        contract = _native_resource(root, entry["contract_ref"])
        request = {
            "engine_id": manifest["engine_id"],
            "engine_version": manifest["implementation_version"],
            "contract_id": contract["contract_id"],
            "contract_version": contract["version"],
            "operation_id": operation,
            "input": payload,
        }
        result = subprocess.run(
            [str(executable), "engine", str(root), str(scripts), operation],
            input=json.dumps(request, ensure_ascii=False, allow_nan=False).encode("utf-8"),
            capture_output=True, timeout=60, check=False,
        )
        if result.returncode != 0 and not result.stdout:
            raise ProfileNativeError(f"Native Go router failed: {result.stderr.decode('utf-8', errors='replace').strip()}")
        response = _json_payload(result.stdout)
        if result.returncode != 0 or response.get("status") != "PASS":
            raise ProfileNativeError(f"Native Profile operation failed: {response.get('error')}")
        if response.get("schema_version") != "ptsip-engine-operation-result/v1" or any(
            response.get(key) != request[key]
            for key in ("engine_id", "engine_version", "contract_id", "contract_version")
        ):
            raise ProfileNativeError("Native Profile response identity does not match its registered request.")
        output = response.get("output")
        if not isinstance(output, dict):
            raise ProfileNativeError("Native Profile response omitted an object output.")
        return output
    except ProfileNativeError:
        raise
    except (OSError, UnicodeError, ValueError, KeyError, TypeError, subprocess.TimeoutExpired) as exc:
        raise ProfileNativeError(f"Unable to invoke native Profile operation: {exc}") from exc


def current_project_profile_contract() -> dict[str, object]:
    return native_profile_operation("discovery.profile.contract", {"root": str(profile_package_root())})


def current_project_profile_version() -> str:
    version = current_project_profile_contract().get("version")
    if not isinstance(version, str) or not version:
        raise ProfileNativeError("Selected native Profile contract has no version.")
    return version


def current_tool_version() -> str:
    package = Path(__file__).resolve().parents[1]
    source = package / "VERSION"
    if not source.is_file() and (package.parent / "VERSION").is_file():
        source = package.parent / "VERSION"
    with source.open("rb") as stream:
        version = tomllib.load(stream).get("tool_version")
    if not isinstance(version, str) or not version:
        raise ProfileNativeError("Tool version is missing in VERSION.")
    return version


@dataclass(frozen=True)
class SpecIdentity:
    name: str = "Primary Lifecycle Ownership and Responsibility Isolation Policy"
    acronym: str = "PTSIP"
    version: str = "0.3.7-draft"
    source: str = "https://github.com/Kinirin/PTSIP"
    revision: str = "3c47816770d194ae42f98faedc911d980db0e62a"
    tool_version: str = field(default_factory=current_tool_version)
    project_profile_contract_version: str = field(default_factory=current_project_profile_version)

    def as_dict(self) -> dict[str, str]:
        return asdict(self)


def current_spec_identity() -> SpecIdentity:
    return SpecIdentity()


DEFAULT_PROFILE_PATH = "ptsip.yaml"


@dataclass(frozen=True)
class RepositorySnapshot:
    captured_at: str
    is_git: bool
    head: str | None
    status_fingerprint: str
    tracked_content_fingerprint: str
    tracked_files: int
    observation_errors: tuple[str, ...]

    @classmethod
    def from_native(cls, value: dict) -> RepositorySnapshot:
        return cls(**{**value, "observation_errors": tuple(value["observation_errors"])})

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


@dataclass(frozen=True)
class SnapshotComparison:
    status: str
    stable: bool
    reasons: tuple[str, ...]


def capture_snapshot(root: str | Path) -> RepositorySnapshot:
    value = native_profile_operation("context.repository.snapshot", {"root": str(Path(root).expanduser())})
    return RepositorySnapshot.from_native(value)


def compare_snapshots(before: RepositorySnapshot, after: RepositorySnapshot) -> SnapshotComparison:
    value = native_profile_operation("validation.repository.snapshot", {"before": before.as_dict(), "after": after.as_dict()})
    return SnapshotComparison(value["status"], value["stable"], tuple(value["reasons"]))


def repository_files(root: str | Path) -> tuple[str, list[str], list[str]]:
    value = native_profile_operation("discovery.repository.inspect", {"path": str(Path(root).expanduser())})
    return value["file_mode"], value["files"], value["observation_errors"]


def _profile_path(root: str | Path, value: object | None) -> dict[str, object]:
    try:
        return native_profile_operation("validation.profile.path", {
            "root": str(Path(root).expanduser()), "path": None if value is None else str(value),
        })
    except ProfileNativeError as exc:
        raise ValueError(str(exc)) from exc


def normalize_profile_path(root: str | Path, value: object | None) -> str:
    return _profile_path(root, value)["normalized_path"]


def profile_path_on_disk(root: str | Path, value: object | None) -> Path:
    return Path(_profile_path(root, value)["path_on_disk"])


@dataclass(frozen=True)
class DraftVersion:
    major: int
    minor: int
    micro: int
    semantic: str
    draft_label: str

    @classmethod
    def from_draft_label(cls, value: object) -> DraftVersion | None:
        if not isinstance(value, str) or not value.strip():
            return None
        result = native_profile_operation("validation.profile.transition-state", {
            "source": {"declared_version": value}, "candidates": [], "snapshot": None, "root": None,
        })
        return cls(**result["state"]) if result["valid"] else None


@dataclass(frozen=True)
class ProfileBinding:
    path: str
    declared_version: str
    specification_revision: str
    specification_source: str | None
    content_sha256: str
    temporary: bool

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


@dataclass(frozen=True)
class ProfileGenerationIdentity(ProfileBinding):
    version: DraftVersion

    @classmethod
    def from_native(cls, value: dict) -> ProfileGenerationIdentity:
        return cls(**{**value, "version": DraftVersion(**value["version"])})

    def as_dict(self) -> dict[str, object]:
        return {**asdict(self), "version": self.version.semantic}


@dataclass(frozen=True)
class ProfileDiagnostic:
    code: str
    message: str
    path: str | None = None

    def as_dict(self) -> dict[str, object]:
        return asdict(self)


@dataclass(frozen=True)
class TransitionSnapshot:
    repository_root: str
    repository: RepositorySnapshot
    profiles: tuple[ProfileGenerationIdentity, ...]

    def as_dict(self) -> dict[str, object]:
        return {"repository_root": self.repository_root, "repository": self.repository.as_dict(),
                "profiles": [profile.as_dict() for profile in self.profiles]}


@dataclass(frozen=True)
class ProfileTransitionState:
    mode: str
    canonical_source: ProfileGenerationIdentity
    temporary_profiles: tuple[ProfileGenerationIdentity, ...]
    final_point: ProfileGenerationIdentity | None
    ordered_sources: tuple[ProfileGenerationIdentity, ...]
    snapshot: TransitionSnapshot

    @classmethod
    def from_native(cls, value: dict) -> ProfileTransitionState:
        snapshot = value["snapshot"]
        generation = ProfileGenerationIdentity.from_native
        return cls(value["mode"], generation(value["canonical_source"]),
                   tuple(map(generation, value["temporary_profiles"])),
                   generation(value["final_point"]) if value["final_point"] else None,
                   tuple(map(generation, value["ordered_sources"])),
                   TransitionSnapshot(snapshot["repository_root"], RepositorySnapshot.from_native(snapshot["repository"]),
                                      tuple(map(generation, snapshot["profiles"]))))

    def as_dict(self) -> dict[str, object]:
        return {"mode": self.mode, "canonical_source": self.canonical_source.as_dict(),
                "temporary_profiles": [profile.as_dict() for profile in self.temporary_profiles],
                "final_point": self.final_point.as_dict() if self.final_point else None,
                "ordered_sources": [profile.as_dict() for profile in self.ordered_sources],
                "snapshot": self.snapshot.as_dict()}


class DirectConvergenceMode(StrEnum):
    CURRENT = "CURRENT"
    IDENTITY_ONLY = "IDENTITY_ONLY"
    DIRECT_SEMANTIC_MIGRATION = "DIRECT_SEMANTIC_MIGRATION"


@dataclass(frozen=True)
class SelectedProfileVersion:
    canonical: str


@dataclass(frozen=True)
class DirectConvergenceState:
    mode: DirectConvergenceMode
    source: ProfileBinding
    source_compatibility_contract: SelectedProfileVersion
    target_contract: SelectedProfileVersion
    transition_kind: str | None
    target_path: str
    target: ProfileBinding | None
    target_is_legacy_alias: bool
    requires_temporary_target: bool
    intermediate_profiles: tuple[object, ...]
    snapshot: RepositorySnapshot

    @classmethod
    def from_native(cls, value: dict) -> DirectConvergenceState:
        return cls(**{
            **value, "mode": DirectConvergenceMode(value["mode"]),
            "source": ProfileBinding(**value["source"]),
            "source_compatibility_contract": SelectedProfileVersion(value["source_compatibility_contract"]),
            "target_contract": SelectedProfileVersion(value["target_contract"]),
            "target": ProfileBinding(**value["target"]) if value["target"] else None,
            "intermediate_profiles": tuple(value["intermediate_profiles"]),
            "snapshot": RepositorySnapshot.from_native(value["snapshot"]),
        })

    def as_dict(self) -> dict[str, object]:
        return {**asdict(self), "mode": self.mode.value,
                "source_compatibility_contract": self.source_compatibility_contract.canonical,
                "target_contract": self.target_contract.canonical}


@dataclass(frozen=True)
class ProfileDiscovery:
    valid: bool
    state: ProfileTransitionState | DirectConvergenceState | None
    diagnostics: tuple[ProfileDiagnostic, ...]

    def as_dict(self) -> dict[str, object]:
        return {"valid": self.valid, "state": self.state.as_dict() if self.state else None,
                "diagnostics": [item.as_dict() for item in self.diagnostics]}


def _observe_profiles(root: str | Path, **selection: object) -> tuple[dict, RepositorySnapshot, tuple[ProfileDiagnostic, ...]]:
    before = capture_snapshot(root)
    observed = native_profile_operation("discovery.profile.candidates", {
        "root": str(Path(root).expanduser()), **selection,
    })
    after = capture_snapshot(root)
    comparison = compare_snapshots(before, after)
    diagnostics = tuple(ProfileDiagnostic(**item) for item in observed["diagnostics"])
    diagnostics += tuple(ProfileDiagnostic("REPOSITORY_OBSERVATION_CHANGED", reason) for reason in comparison.reasons)
    return {"source": observed["source"], "candidates": observed["candidates"]}, after, diagnostics


def discover_profile_transition(root: str | Path) -> ProfileDiscovery:
    observed, snapshot, diagnostics = _observe_profiles(root)
    if diagnostics:
        return ProfileDiscovery(False, None, diagnostics)
    result = native_profile_operation("validation.profile.transition-state", {
        **observed, "snapshot": snapshot.as_dict(), "root": str(Path(root).expanduser()),
    })
    state = ProfileTransitionState.from_native(result["state"]) if result["valid"] else None
    return ProfileDiscovery(result["valid"], state, tuple(ProfileDiagnostic(**item) for item in result["diagnostics"]))


def discover_direct_profile_convergence(root: str | Path) -> ProfileDiscovery:
    observed, snapshot, diagnostics = _observe_profiles(root)
    if diagnostics:
        return ProfileDiscovery(False, None, diagnostics)
    result = native_profile_operation("validation.profile.convergence", {
        **observed, "snapshot": snapshot.as_dict(), "selected_contract": current_project_profile_contract(),
    })
    state = DirectConvergenceState.from_native(result["state"]) if result["valid"] else None
    return ProfileDiscovery(result["valid"], state, tuple(ProfileDiagnostic(**item) for item in result["diagnostics"]))


def validate_direct_convergence_snapshot(root: str | Path, state: DirectConvergenceState) -> tuple[ProfileDiagnostic, ...]:
    target_paths = [state.target.path] if state.target else []
    observed, snapshot, diagnostics = _observe_profiles(root, source_path=state.source.path, candidate_paths=target_paths)
    reasons = list(compare_snapshots(state.snapshot, snapshot).reasons)
    reasons.extend(item.message for item in diagnostics)
    if observed["source"] != state.source.as_dict():
        reasons.append(f"{state.source.path}: Profile binding changed")
    if state.target and observed["candidates"] != [state.target.as_dict()]:
        reasons.append(f"{state.target.path}: Profile binding changed")
    if reasons:
        return (ProfileDiagnostic("STALE_DIRECT_CONVERGENCE_SNAPSHOT", "; ".join(dict.fromkeys(reasons))),)
    return ()
