from __future__ import annotations

from dataclasses import dataclass
from importlib.resources import files
from pathlib import PurePosixPath
from typing import Mapping

import yaml


class ProjectProfileContractRegistryError(RuntimeError):
    """Fail-closed runtime error for the embedded Project Profile contract registry."""


@dataclass(frozen=True)
class RuntimeProjectProfileContract:
    version: str
    operations: frozenset[str]
    schema_resource: str


def _embedded_registry_payload() -> dict[str, object]:
    resource = files("ptsip").joinpath("specdata", "project-profile-contracts.yaml")
    try:
        payload = yaml.safe_load(resource.read_text(encoding="utf-8"))
    except Exception as exc:
        raise ProjectProfileContractRegistryError(
            f"Unable to load embedded Project Profile contract registry: {exc}"
        ) from exc
    if not isinstance(payload, dict):
        raise ProjectProfileContractRegistryError(
            "Embedded Project Profile contract registry must be a mapping."
        )
    return payload


def _schema_resource(value: object) -> str:
    if not isinstance(value, str):
        raise ProjectProfileContractRegistryError(
            "Current Project Profile contract must bind a schema path."
        )
    path = PurePosixPath(value)
    if (
        path.is_absolute()
        or ".." in path.parts
        or len(path.parts) != 2
        or path.parts[0] != "schemas"
        or not path.name.startswith("ptsip-profile-pp-")
        or not path.name.endswith(".schema.json")
    ):
        raise ProjectProfileContractRegistryError(
            f"Invalid current Project Profile schema path: {value!r}."
        )
    return path.name


def current_runtime_project_profile_contract() -> RuntimeProjectProfileContract:
    payload = _embedded_registry_payload()
    current = payload.get("current")
    contracts = payload.get("contracts")
    if not isinstance(current, str) or not isinstance(contracts, list):
        raise ProjectProfileContractRegistryError(
            "Embedded Project Profile contract registry is missing current/contracts."
        )

    matches = [
        item
        for item in contracts
        if isinstance(item, Mapping) and item.get("version") == current
    ]
    if len(matches) != 1:
        raise ProjectProfileContractRegistryError(
            "Embedded Project Profile current identity must resolve exactly once."
        )
    contract = matches[0]
    if contract.get("lifecycle") != "CURRENT":
        raise ProjectProfileContractRegistryError(
            "Embedded Project Profile current contract must have lifecycle CURRENT."
        )

    operations = contract.get("operations")
    if (
        not isinstance(operations, list)
        or not operations
        or any(not isinstance(item, str) for item in operations)
        or len(operations) != len(set(operations))
    ):
        raise ProjectProfileContractRegistryError(
            "Embedded Project Profile current contract operations are invalid."
        )

    return RuntimeProjectProfileContract(
        version=current,
        operations=frozenset(operations),
        schema_resource=_schema_resource(contract.get("schema")),
    )


def current_runtime_project_profile_version() -> str:
    return current_runtime_project_profile_contract().version
