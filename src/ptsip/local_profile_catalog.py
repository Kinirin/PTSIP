from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path, PurePosixPath
from typing import Mapping

import yaml


LOCAL_PROFILE_CATALOG_SCHEMA_VERSION = "ptsip-local-profile-catalog/v1"
LOCAL_PROFILE_ROOT = PurePosixPath(".ptsip/profiles")
LOCAL_PROFILE_CATALOG = LOCAL_PROFILE_ROOT / "index.yaml"
DEFAULT_PROFILE_ID = "main"
DEFAULT_PROFILE_RESOURCE = "main.ptsip.yaml"
LEGACY_ROOT_PROFILE = "ptsip.yaml"


class LocalProfileCatalogError(ValueError):
    """Fail-closed error for repository-local Project Profile selection."""


@dataclass(frozen=True)
class LocalProfileSelection:
    profile_id: str
    resource: str
    path: Path


def _profile_resource(value: object) -> str:
    if not isinstance(value, str) or not value:
        raise LocalProfileCatalogError("Local profile resource must be a non-empty string.")
    resource = PurePosixPath(value)
    if (
        resource.is_absolute()
        or ".." in resource.parts
        or len(resource.parts) != 1
        or not resource.name.endswith(".ptsip.yaml")
    ):
        raise LocalProfileCatalogError(
            f"Local profile resource must be one *.ptsip.yaml filename: {value!r}."
        )
    return resource.name


def load_local_profile_selection(
    repository_root: str | Path,
) -> LocalProfileSelection | None:
    root = Path(repository_root).resolve()
    catalog_path = root / LOCAL_PROFILE_CATALOG.as_posix()
    if not catalog_path.is_file():
        return None

    try:
        payload = yaml.safe_load(catalog_path.read_text(encoding="utf-8"))
    except Exception as exc:
        raise LocalProfileCatalogError(
            f"Unable to parse {LOCAL_PROFILE_CATALOG.as_posix()}: {exc}"
        ) from exc
    if not isinstance(payload, Mapping):
        raise LocalProfileCatalogError("Local profile catalog must be a mapping.")
    if payload.get("schema_version") != LOCAL_PROFILE_CATALOG_SCHEMA_VERSION:
        raise LocalProfileCatalogError(
            "Local profile catalog schema_version is unsupported."
        )

    default_profile = payload.get("default_profile")
    profiles = payload.get("profiles")
    if not isinstance(default_profile, str) or not default_profile:
        raise LocalProfileCatalogError(
            "Local profile catalog default_profile must be a non-empty id."
        )
    if not isinstance(profiles, list) or not profiles:
        raise LocalProfileCatalogError("Local profile catalog profiles must be non-empty.")

    by_id: dict[str, str] = {}
    resources: set[str] = set()
    for row in profiles:
        if not isinstance(row, Mapping):
            raise LocalProfileCatalogError("Local profile catalog entries must be mappings.")
        profile_id = row.get("id")
        if not isinstance(profile_id, str) or not profile_id:
            raise LocalProfileCatalogError("Local profile catalog entry id is invalid.")
        if profile_id in by_id:
            raise LocalProfileCatalogError(
                f"Duplicate local profile id: {profile_id!r}."
            )
        resource = _profile_resource(row.get("resource"))
        if resource in resources:
            raise LocalProfileCatalogError(
                f"Duplicate local profile resource: {resource!r}."
            )
        by_id[profile_id] = resource
        resources.add(resource)

    resource = by_id.get(default_profile)
    if resource is None:
        raise LocalProfileCatalogError(
            f"default_profile {default_profile!r} does not resolve to a catalog entry."
        )

    path = root / LOCAL_PROFILE_ROOT.as_posix() / resource
    if not path.is_file():
        raise LocalProfileCatalogError(
            f"Default local profile resource is missing: {path.relative_to(root).as_posix()}."
        )
    return LocalProfileSelection(default_profile, resource, path)


def canonical_new_profile_path(repository_root: str | Path) -> Path:
    root = Path(repository_root).resolve()
    return root / LOCAL_PROFILE_ROOT.as_posix() / DEFAULT_PROFILE_RESOURCE


def default_catalog_payload() -> dict[str, object]:
    return {
        "schema_version": LOCAL_PROFILE_CATALOG_SCHEMA_VERSION,
        "default_profile": DEFAULT_PROFILE_ID,
        "profiles": [
            {
                "id": DEFAULT_PROFILE_ID,
                "resource": DEFAULT_PROFILE_RESOURCE,
            }
        ],
    }


def default_catalog_text() -> str:
    return yaml.safe_dump(default_catalog_payload(), sort_keys=False, allow_unicode=True)
