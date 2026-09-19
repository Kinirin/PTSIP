from __future__ import annotations

from pathlib import Path
from typing import Mapping

from jsonschema import Draft202012Validator

from developer.automation.policy_loader import load_json, load_yaml, repository_root


PUBLIC_PROFILE_CATALOG = "profiles/index.yaml"
PUBLIC_PROFILE_CATALOG_SCHEMA = "developer/policy/schemas/public-profile-catalog.schema.json"
PP_CONTRACT_REGISTRY = "registry/project-profile-contracts.yaml"
EMBEDDED_PP_CONTRACT_REGISTRY = "src/ptsip/specdata/project-profile-contracts.yaml"
PP_CONTRACT_REGISTRY_SCHEMA = (
    "developer/policy/schemas/project-profile-contract-registry.schema.json"
)


def _mapping(value: object) -> Mapping[str, object] | None:
    return value if isinstance(value, Mapping) else None


def load_public_profile_catalog(root: str | Path | None = None) -> dict[str, object]:
    return load_yaml(PUBLIC_PROFILE_CATALOG, root=repository_root(root))


def load_project_profile_contract_registry(
    root: str | Path | None = None,
) -> dict[str, object]:
    return load_yaml(PP_CONTRACT_REGISTRY, root=repository_root(root))


def current_project_profile_contract(
    root: str | Path | None = None,
) -> Mapping[str, object]:
    registry = load_project_profile_contract_registry(root)
    current = registry.get("current")
    matches = [
        item
        for item in registry.get("contracts", [])
        if isinstance(item, Mapping) and item.get("version") == current
    ]
    if len(matches) != 1:
        raise ValueError(
            "project-profile contract registry current identity must resolve exactly once"
        )
    return matches[0]


def validate_project_profile_registry_plane(
    root: str | Path | None = None,
) -> tuple[str, ...]:
    base = repository_root(root)
    errors: list[str] = []

    catalog = load_yaml(PUBLIC_PROFILE_CATALOG, root=base)
    registry = load_yaml(PP_CONTRACT_REGISTRY, root=base)
    embedded_registry = load_yaml(EMBEDDED_PP_CONTRACT_REGISTRY, root=base)
    catalog_schema = load_json(PUBLIC_PROFILE_CATALOG_SCHEMA, root=base)
    registry_schema = load_json(PP_CONTRACT_REGISTRY_SCHEMA, root=base)

    for schema in (catalog_schema, registry_schema):
        Draft202012Validator.check_schema(schema)

    for error in Draft202012Validator(catalog_schema).iter_errors(catalog):
        errors.append(f"{PUBLIC_PROFILE_CATALOG}: {error.message}")
    for error in Draft202012Validator(registry_schema).iter_errors(registry):
        errors.append(f"{PP_CONTRACT_REGISTRY}: {error.message}")

    if registry != embedded_registry:
        errors.append(
            "embedded Project Profile contract registry must equal canonical registry"
        )

    contracts = [
        item
        for item in registry.get("contracts", [])
        if isinstance(item, Mapping)
    ]
    contract_ids = [str(item.get("version")) for item in contracts]
    if len(contract_ids) != len(set(contract_ids)):
        errors.append("project-profile contract registry versions must be unique")

    current = registry.get("current")
    current_contracts = [item for item in contracts if item.get("version") == current]
    if len(current_contracts) != 1:
        errors.append("project-profile contract registry current identity must resolve exactly once")
    else:
        current_contract = current_contracts[0]
        if current_contract.get("lifecycle") != "CURRENT":
            errors.append("current project-profile contract must have lifecycle CURRENT")
        baseline_path = current_contract.get("baseline")
        if not isinstance(baseline_path, str):
            errors.append("current project-profile contract must bind an immutable baseline")
        else:
            baseline_root = base / baseline_path
            if not baseline_root.is_dir():
                errors.append(
                    f"current project-profile baseline is missing: {baseline_path}"
                )

        schema_path = current_contract.get("schema")
        if not isinstance(schema_path, str):
            errors.append("current project-profile contract must bind a canonical schema")
        else:
            candidate = base / schema_path
            if not candidate.is_file():
                errors.append(f"current project-profile schema is missing: {schema_path}")
            else:
                schema_payload = load_json(schema_path, root=base)
                declared = (
                    schema_payload.get("properties", {})
                    .get("ptsip", {})
                    .get("properties", {})
                    .get("version", {})
                    .get("const")
                )
                if declared != current:
                    errors.append(
                        "current project-profile schema version const does not match registry current"
                    )

    transition_pairs: set[tuple[str, str]] = set()
    for item in registry.get("transitions", []):
        if not isinstance(item, Mapping):
            continue
        source = item.get("from")
        target = item.get("to")
        pair = (str(source), str(target))
        if pair in transition_pairs:
            errors.append(f"duplicate project-profile transition: {pair[0]} -> {pair[1]}")
        transition_pairs.add(pair)
        if source not in contract_ids or target not in contract_ids:
            errors.append(
                f"project-profile transition references an unregistered contract: {source!r} -> {target!r}"
            )

    profile_entries = [
        item for item in catalog.get("profiles", []) if isinstance(item, Mapping)
    ]
    profile_ids = [str(item.get("id")) for item in profile_entries]
    resources = [str(item.get("resource")) for item in profile_entries]
    if len(profile_ids) != len(set(profile_ids)):
        errors.append("public profile catalog ids must be unique")
    if len(resources) != len(set(resources)):
        errors.append("public profile catalog resources must be unique")

    profile_root = base / str(catalog.get("root", "profiles"))
    discovered = sorted(path.name for path in profile_root.glob("*.ptsip.yaml"))
    if sorted(resources) != discovered:
        errors.append(
            "public profile catalog resources must cover canonical root *.ptsip.yaml exactly"
        )

    for item in profile_entries:
        resource = item.get("resource")
        contract = item.get("contract")
        expected_mode = item.get("responsibility_mode")
        if not isinstance(resource, str):
            continue
        path = profile_root / resource
        if not path.is_file():
            errors.append(f"registered public profile resource is missing: {resource}")
            continue
        payload = load_yaml(path.relative_to(base).as_posix(), root=base)
        ptsip = _mapping(payload.get("ptsip"))
        responsibility_map = _mapping(payload.get("responsibility_map"))
        declared_contract = None if ptsip is None else ptsip.get("version")
        declared_mode = None if responsibility_map is None else responsibility_map.get("mode")
        if contract not in contract_ids:
            errors.append(
                f"public profile {resource} references unregistered contract {contract!r}"
            )
        if declared_contract != contract:
            errors.append(
                f"public profile {resource} contract does not match catalog binding"
            )
        if declared_mode != expected_mode:
            errors.append(
                f"public profile {resource} responsibility mode does not match catalog binding"
            )

    current_profile_contracts = {
        item.get("contract") for item in profile_entries if isinstance(item.get("contract"), str)
    }

    for contract in contracts:
        baseline_path = contract.get("baseline")
        if not isinstance(baseline_path, str):
            continue
        baseline_root = base / baseline_path
        if not baseline_root.is_dir():
            errors.append(
                f"project-profile baseline is missing for {contract.get('version')}: {baseline_path}"
            )
            continue
        baseline_resources = sorted(
            path.name for path in baseline_root.glob("*.ptsip.yaml")
        )
        if contract.get("version") == current and baseline_resources != sorted(resources):
            errors.append(
                "current project-profile baseline must cover current public profile resources exactly"
            )
        for resource in baseline_resources:
            payload = load_yaml(
                (baseline_root / resource).relative_to(base).as_posix(),
                root=base,
            )
            ptsip = _mapping(payload.get("ptsip"))
            declared = None if ptsip is None else ptsip.get("version")
            if declared != contract.get("version"):
                errors.append(
                    f"baseline {baseline_path}/{resource} does not declare contract {contract.get('version')!r}"
                )
    if current_profile_contracts != {current}:
        errors.append(
            "current public profile catalog must bind every distributed profile to registry current"
        )

    return tuple(errors)


if __name__ == "__main__":
    failures = validate_project_profile_registry_plane()
    if failures:
        raise SystemExit("\n".join(failures))
    print("Project Profile registry plane validation: PASS")
