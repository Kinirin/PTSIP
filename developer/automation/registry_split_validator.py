from __future__ import annotations

import json
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator


SUPPORT_REGISTRIES = (
    "src/ptsip/specdata/ptsip-support-authority-schema-registry.yaml",
    "src/ptsip/specdata/ptsip-support-authority-role-registry.yaml",
    "src/ptsip/specdata/ptsip-support-authority-subject-registry.yaml",
    "src/ptsip/specdata/ptsip-support-authorization-registry.yaml",
)
DEVELOPER_REGISTRIES = (
    "developer/policy/registries/authority-schema-registry.yaml",
    "developer/policy/registries/authority-role-registry.yaml",
    "developer/policy/registries/authority-subject-registry.yaml",
    "developer/policy/registries/authorization-transition-registry.yaml",
)


def _yaml(root: Path, path: str) -> dict[str, object]:
    value = yaml.safe_load((root / path).read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{path}: registry root must be a mapping")
    return value


def _json(root: Path, path: str) -> dict[str, object]:
    value = json.loads((root / path).read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{path}: schema root must be a mapping")
    return value


def validate_registry_split(root: str | Path) -> tuple[str, ...]:
    base = Path(root).resolve()
    errors: list[str] = []
    inventory = _yaml(base, "developer/policy/registry-split-inventory.yaml")
    inventory_schema = _json(base, "developer/policy/schemas/registry-split-inventory.schema.json")
    support_schema = _json(base, "schemas/ptsip-support-governance-registry.schema.json")
    developer_schema = _json(base, "developer/policy/schemas/developer-governance-registry.schema.json")
    support_semantics = _json(base, "schemas/ptsip-support-authority-semantics.schema.json")
    embedded_support_semantics = _json(base, "src/ptsip/specdata/ptsip-support-authority-semantics.schema.json")
    developer_semantics = _json(base, "developer/policy/schemas/developer-authority-semantics.schema.json")

    for schema in (inventory_schema, support_schema, developer_schema, support_semantics, developer_semantics):
        Draft202012Validator.check_schema(schema)
    for error in Draft202012Validator(inventory_schema).iter_errors(inventory):
        errors.append(f"registry-split-inventory: {error.message}")
    if support_semantics != embedded_support_semantics:
        errors.append("support authority semantics canonical and embedded schemas differ")

    if inventory.get("classification") != "SPLIT" or inventory.get("source_registry_count") != 4:
        errors.append("registry split inventory must classify exactly four legacy registries as SPLIT")

    support_payloads = []
    for path in SUPPORT_REGISTRIES:
        payload = _yaml(base, path)
        support_payloads.append(payload)
        for error in Draft202012Validator(support_schema).iter_errors(payload):
            errors.append(f"{path}: {error.message}")
        if "decisions/" in (base / path).read_text(encoding="utf-8"):
            errors.append(f"{path}: shipped support registry must not depend on legacy decisions paths")

    developer_payloads = []
    for path in DEVELOPER_REGISTRIES:
        payload = _yaml(base, path)
        developer_payloads.append(payload)
        for error in Draft202012Validator(developer_schema).iter_errors(payload):
            errors.append(f"{path}: {error.message}")

    support_index = _yaml(base, "src/ptsip/specdata/support-policy-index.yaml")
    support_ids = [item["id"] for item in support_index["policies"]]
    schema_registry = support_payloads[0]
    if [item["policy_id"] for item in schema_registry["entries"]] != support_ids:
        errors.append("support authority schema registry must cover support-policy-index exactly")

    role_registry = support_payloads[1]
    if [item["policy_id"] for item in role_registry["policy_roles"]] != support_ids:
        errors.append("support authority role registry must cover support-policy-index exactly")
    tokens = role_registry["effect_vocabulary"]["tokens"]
    if role_registry["effect_vocabulary"]["count"] != len(tokens) or len(tokens) != len(set(tokens)):
        errors.append("support authority effect vocabulary count/uniqueness mismatch")

    subject_registry = support_payloads[2]
    if "current_repository_bindings" in subject_registry:
        errors.append("support subject registry must not ship PTSIP repository bindings")
    if set(subject_registry["subject_identity_schemes"]) != {"SUPPORT_POLICY_ID"}:
        errors.append("support subject registry must use SUPPORT_POLICY_ID only")

    auth_registry = support_payloads[3]
    for forbidden in ("authorization_provenance", "rules", "held_scopes"):
        if forbidden in auth_registry:
            errors.append(f"support authorization registry must not ship {forbidden}")

    developer_schema_registry = developer_payloads[0]
    expected_mpd = [f"MPD-{index:04d}" for index in range(2, 10)]
    if [item["policy_id"] for item in developer_schema_registry["entries"]] != expected_mpd:
        errors.append("developer authority schema registry must cover migrated MPD-0002..MPD-0009 exactly")

    developer_role = developer_payloads[1]
    if [item["policy_id"] for item in developer_role["policy_roles"]] != expected_mpd:
        errors.append("developer authority role registry must cover migrated MPD-0002..MPD-0009 exactly")

    legacy_role = _yaml(base, "decisions/AUTHORITY-ROLE-REGISTRY.yaml")
    legacy_tokens = set(legacy_role["effect_vocabulary"]["tokens"])
    support_tokens = set(role_registry["effect_vocabulary"]["tokens"])
    developer_tokens = set(developer_role["effect_vocabulary"]["tokens"])
    if len(legacy_tokens) != 99:
        errors.append("legacy frozen authority effect vocabulary must contain exactly 99 tokens")
    if support_tokens | developer_tokens != legacy_tokens:
        errors.append("SFP/MPD authority effect vocabularies must preserve the exact legacy 99-token union")
    if (support_tokens | developer_tokens) - legacy_tokens:
        errors.append("registry split introduced an authority effect token not present in the frozen legacy vocabulary")

    for entry in schema_registry["entries"]:
        policy = _yaml(base, f"src/ptsip/specdata/{entry['policy_id']}.yaml")
        definition = support_semantics["$defs"][entry["schema_definition"]]
        for error in Draft202012Validator(definition).iter_errors(policy["authority_semantics"]):
            errors.append(f"{entry['policy_id']}: {error.message}")

    for entry in developer_schema_registry["entries"]:
        policy = _yaml(base, f"developer/policy/{entry['policy_id']}.yaml")
        definition = developer_semantics["$defs"][entry["schema_definition"]]
        for error in Draft202012Validator(definition).iter_errors(policy["rules"]["authority_semantics"]):
            errors.append(f"{entry['policy_id']}: {error.message}")

    return tuple(errors)
