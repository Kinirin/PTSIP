from __future__ import annotations

from dataclasses import asdict, dataclass, field
import json
import os
from pathlib import Path, PurePosixPath
import subprocess
import sysconfig

from ..constants import SPEC_ACRONYM, SPEC_NAME, SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION, TOOL_VERSION


class ProfileNativeError(RuntimeError):
    """The registered native Profile operation could not return a current result."""


_PROFILE_OPERATIONS = frozenset({
    "discovery.profile.contract", "discovery.profile.catalog", "discovery.profile.rules",
    "validation.profile.identity", "validation.profile.transition", "validation.profile.rules",
    "projection.profile.metadata",
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
    """Prepare the registered envelope and route through the installed Go CLI."""
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


@dataclass(frozen=True)
class SpecIdentity:
    name: str = SPEC_NAME
    acronym: str = SPEC_ACRONYM
    version: str = SPEC_VERSION
    source: str = SPEC_SOURCE
    revision: str = SPEC_REVISION
    tool_version: str = TOOL_VERSION
    project_profile_contract_version: str = field(default_factory=current_project_profile_version)

    def as_dict(self) -> dict[str, str]:
        return asdict(self)


def current_spec_identity() -> SpecIdentity:
    return SpecIdentity()
