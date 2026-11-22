from __future__ import annotations

import hashlib
import json
import sys
from pathlib import Path, PurePosixPath
from shutil import copy2, copytree, rmtree

import yaml
from setuptools import setup
from setuptools.command.build_py import build_py as _build_py


ROOT = Path(__file__).resolve().parent
CANONICAL_SUPPORT_POLICY = ROOT / "src" / "policy"
PP200_CANDIDATE_ASSET = "ptsip-profile-pp-2.00.candidate.schema.json"


def _load_yaml(path: Path) -> dict[str, object]:
    value = yaml.safe_load(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise RuntimeError(f"canonical YAML asset must be a mapping: {path}")
    return value


def _project_support_policy(build_lib: Path) -> None:
    source = CANONICAL_SUPPORT_POLICY
    required = (source / "index.yaml", source / "schemas", source / "registries")
    if not all(path.exists() for path in required):
        missing = [str(path) for path in required if not path.exists()]
        raise RuntimeError(f"canonical Support Policy source is incomplete: {missing}")

    target = build_lib / "ptsip" / "support"
    if target.exists():
        rmtree(target)

    policy_target = target / "policy"
    policy_target.mkdir(parents=True, exist_ok=True)
    copy2(source / "index.yaml", policy_target / "index.yaml")

    sfp_files = sorted(source.rglob("SFP-*.yaml"))
    if not sfp_files:
        raise RuntimeError("canonical Support Policy source contains no SFP records")
    # Setuptools can invoke this in a Git-less sdist with no installed PTSIP.
    sys.path.insert(0, str(ROOT / "src"))
    try:
        from ptsip._common_contract import PolicySourceError, source_result
    finally:
        sys.path.pop(0)
    for path in sfp_files:
        relative = path.relative_to(source)
        destination = policy_target / relative
        destination.parent.mkdir(parents=True, exist_ok=True)
        try:
            projected = source_result("project-support-source", "source", raw=path.read_bytes().decode("utf-8-sig"),
                                      path=relative.as_posix())
        except PolicySourceError as error:
            raise RuntimeError(f"Support Policy source {relative.as_posix()}: {error}") from error
        if projected is not None:
            destination.write_text(yaml.safe_dump(projected, allow_unicode=True, sort_keys=False), encoding="utf-8")
        else:
            copy2(path, destination)

    copytree(source / "schemas", target / "schemas")
    copytree(source / "registries", target / "registries")


def _project_profile_contracts(build_lib: Path) -> None:
    """Stage registered Profile schemas from the central schema catalog."""
    schema_root = ROOT / "src" / "ptsip" / "schemas"
    config_bytes = (schema_root / "config.json").read_bytes()
    config = json.loads(config_bytes)
    profiles = config.get("storage", {}).get("profile")
    expected = {
        "ptsip.project-profile": ("pp.3.00", "storage/profile/project-profile.schema.json"),
        **{
            f"ptsip.project-profile.{version}": (
                version, f"storage/profile/history/project-profile-{version}.schema.json"
            )
            for version in ("pp.1.01", "pp.1.02", "pp.2.00")
        },
    }
    if not isinstance(profiles, dict) or set(profiles) != set(expected):
        raise RuntimeError("central Profile schema catalog must contain exactly four registered resources")

    assets: list[tuple[str, bytes]] = []
    ids: set[str] = set()
    selected_id = None
    for key, (version, relative) in expected.items():
        row = profiles[key]
        lifecycle = "ACTIVE" if key == "ptsip.project-profile" else "DEPRECATED"
        if (
            not isinstance(row, dict)
            or row.get("path") != relative
            or row.get("version") != version
            or row.get("registered_lifecycle") != lifecycle
        ):
            raise RuntimeError(f"invalid central Profile schema registration: {key}")
        data = (schema_root / relative).read_bytes()
        schema = json.loads(data)
        identity = schema.get("$id")
        digest = hashlib.sha256(data.replace(b"\r\n", b"\n")).hexdigest()
        if (
            not isinstance(identity, str)
            or not identity
            or identity != row.get("$id")
            or identity in ids
            or digest != row.get("schema_sha256")
        ):
            raise RuntimeError(f"central Profile schema identity or immutable digest mismatch: {key}")
        ids.add(identity)
        if key == "ptsip.project-profile":
            selected_id = identity
            if schema.get("$ref") is not None:
                raise RuntimeError("official current Profile must be a direct schema")
        if schema.get("properties", {}).get("ptsip", {}).get("properties", {}).get("version", {}).get("const") != version:
            raise RuntimeError(f"central Profile schema version mismatch: {key}")
        assets.append((relative, data))
    if selected_id != "urn:ptsip:storage:contract:profile:official":
        raise RuntimeError("current Profile must use the approved official schema identity")

    target = build_lib / "ptsip" / "schemas"
    target.mkdir(parents=True, exist_ok=True)
    (target / "config.json").write_bytes(config_bytes)
    for relative, data in assets:
        destination = target / relative
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_bytes(data)
    # Incremental build staging can still contain the retired packages and catalog.
    package_build_root = build_lib.resolve() / "ptsip"
    for name in ("profiles", "repository"):
        legacy = build_lib / "ptsip" / name
        if legacy.is_symlink() or legacy.resolve() != package_build_root / name:
            raise RuntimeError("Retired Profile or Repository build output is an alias")
        if legacy.exists():
            rmtree(legacy)
    catalog = build_lib / "ptsip" / "specdata" / "project-profile-contracts.yaml"
    if catalog.is_symlink() or catalog.resolve() != package_build_root / "specdata" / catalog.name:
        raise RuntimeError("Retired Profile catalog build output is an alias")
    catalog.unlink(missing_ok=True)


def _project_rbac_contract_assets(build_lib: Path) -> None:
    """Stage the three product RBAC structure schemas selected by config.json."""
    schema_root = ROOT / "src" / "ptsip" / "schemas"
    config_bytes = (schema_root / "config.json").read_bytes()
    registrations = json.loads(config_bytes).get("storage", {}).get("Responsibility-Authority")
    if not isinstance(registrations, dict) or set(registrations) != {"RA_list", "RAC_list", "AEC_list"}:
        raise RuntimeError("central RBAC catalog must register RA_list, RAC_list and AEC_list")

    assets: dict[PurePosixPath, bytes] = {}
    identities: set[str] = set()
    rbac_root = PurePosixPath("storage/rbac")
    for key, entry in registrations.items():
        if not isinstance(entry, dict):
            raise RuntimeError(f"invalid central RBAC schema registration: {key}")
        identity, relative, version = (entry.get(name) for name in ("$id", "path", "version"))
        if not all(isinstance(value, str) and value for value in (identity, relative, version)):
            raise RuntimeError(f"incomplete central RBAC schema registration: {key}")
        path = PurePosixPath(relative)
        if (
            path.parent != rbac_root or path.as_posix() != relative
            or not path.name.endswith(".schema.json") or "\\" in relative or ":" in relative
            or path in assets or identity in identities
        ):
            raise RuntimeError(f"invalid or repeated registered RBAC schema path or identity: {key}")
        source = schema_root.joinpath(*path.parts)
        if source.resolve() != schema_root.resolve().joinpath(*path.parts):
            raise RuntimeError(f"registered RBAC schema is an alias: {key}")
        data = source.read_bytes()
        schema = json.loads(data)
        if not isinstance(schema, dict) or schema.get("$id") != identity:
            raise RuntimeError(f"central RBAC schema identity mismatch: {key}")
        assets[path] = data
        identities.add(identity)

    package_root = build_lib / "ptsip"
    stale = package_root / "specdata" / "compile-sealed-rbac"
    if stale.resolve() != build_lib.resolve() / "ptsip" / "specdata" / stale.name or stale.is_symlink():
        raise RuntimeError("retired RBAC build output is an alias")
    if stale.exists():
        rmtree(stale)

    target = package_root / "schemas"
    target.joinpath(*rbac_root.parts).mkdir(parents=True, exist_ok=True)
    (target / "config.json").write_bytes(config_bytes)
    for path, data in assets.items():
        target.joinpath(*path.parts).write_bytes(data)


def _project_profile_candidate(build_lib: Path) -> None:
    # This opt-in package asset retains its draft role after retirement of the
    # byte-identical root schemas/ copy. It is not a profile validation source.
    candidate = ROOT / "src" / "ptsip" / "specdata" / PP200_CANDIDATE_ASSET
    if not candidate.is_file():
        raise RuntimeError("pp.2.00 candidate profile schema is missing")
    comment = _load_yaml(candidate).get("$comment")
    if not isinstance(comment, str) or not comment.startswith("DRAFT_NON_NORMATIVE:"):
        raise RuntimeError("packaged pp.2.00 profile schema candidate must remain non-normative")
    target = build_lib / "ptsip" / "specdata"
    target.mkdir(parents=True, exist_ok=True)
    copy2(candidate, target / candidate.name)


class build_py(_build_py):
    """Project canonical repository assets into the built ptsip package."""

    def run(self) -> None:
        super().run()
        build_lib = Path(self.build_lib)
        # Incremental builds can retain the replaced Python feature from an
        # earlier checkout even though it is absent from the source package.
        for name in ("unified.py", "__init__.py"):
            stale = build_lib / "ptsip" / "policy_contracts" / name
            if stale.exists():
                if not stale.resolve().is_relative_to(build_lib.resolve()):
                    raise RuntimeError("retired contract output escapes build directory")
                stale.unlink()
        _project_support_policy(build_lib)
        _project_profile_contracts(build_lib)
        _project_rbac_contract_assets(build_lib)
        _project_profile_candidate(build_lib)


setup(cmdclass={"build_py": build_py})
