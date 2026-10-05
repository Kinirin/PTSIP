from __future__ import annotations

import json
from pathlib import Path
from typing import Any

import yaml


def repository_root(start: str | Path | None = None) -> Path:
    current = Path(start or __file__).resolve()
    if current.is_file():
        current = current.parent
    for candidate in (current, *current.parents):
        if (candidate / "pyproject.toml").is_file():
            return candidate
    raise RuntimeError("Unable to locate PTSIP repository root.")


def registered_policy_file(path: str | Path, *, root: str | Path) -> Path:
    """Locate an admitted immutable source file for audit or provenance checks."""
    from ptsip.governance.authority import migration_registry, safe_policy_path, source_route

    base = Path(root).resolve()
    candidate = Path(path)
    candidate = candidate.resolve() if candidate.is_absolute() else (base / candidate).resolve()
    for plane, policy_class in (("developer/policy", "PTSIP_DEVELOPER_POLICY"), ("src/policy", "PTSIP_SUPPORT_FEATURE")):
        policy_root = base / plane
        if not candidate.is_relative_to(policy_root):
            continue
        source = source_route(migration_registry(policy_root, policy_class, copy_result=False), path=candidate.relative_to(policy_root).as_posix())
        if source is not None:
            return safe_policy_path(policy_root, source["archive_path"])
    return candidate


def load_yaml(path: str | Path, *, root: str | Path | None = None) -> dict[str, Any]:
    base = repository_root(root)
    candidate = Path(path)
    if not candidate.is_absolute():
        candidate = base / candidate
    from ptsip.governance.authority import load_yaml_mapping, migration_registry, read_policy, source_route

    for relative_root, policy_class in (
        ("developer/policy", "PTSIP_DEVELOPER_POLICY"),
        ("src/policy", "PTSIP_SUPPORT_FEATURE"),
    ):
        policy_root = base / relative_root
        if candidate.resolve().is_relative_to(policy_root.resolve()):
            relative = candidate.resolve().relative_to(policy_root.resolve()).as_posix()
            registry = migration_registry(policy_root, policy_class, copy_result=False)
            if source_route(registry, path=relative):
                return read_policy(policy_root, relative, policy_class)
    return load_yaml_mapping(candidate)


def load_json(path: str | Path, *, root: str | Path | None = None) -> dict[str, Any]:
    base = repository_root(root)
    candidate = Path(path)
    if not candidate.is_absolute():
        candidate = base / candidate
    value = json.loads(candidate.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{candidate}: expected a JSON object")
    return value
