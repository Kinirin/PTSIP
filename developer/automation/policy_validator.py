from __future__ import annotations

from pathlib import Path
from typing import Mapping

from jsonschema import Draft202012Validator

from developer.automation.policy_loader import load_json, load_yaml, repository_root


INDEX = "developer/policy/index.yaml"
INDEX_SCHEMA = "developer/policy/schemas/developer-policy-index.schema.json"
MPD_SCHEMA = "developer/policy/schemas/management-policy.schema.json"

SFP_CANONICAL_SCHEMA = "schemas/ptsip-support-feature-policy.schema.json"
SFP_EMBEDDED_SCHEMA = "src/ptsip/specdata/ptsip-support-feature-policy.schema.json"
SFP_INDEX = "src/ptsip/specdata/support-policy-index.yaml"
SFP_INDEX_CANONICAL_SCHEMA = "schemas/ptsip-support-feature-policy-index.schema.json"
SFP_INDEX_EMBEDDED_SCHEMA = "src/ptsip/specdata/ptsip-support-feature-policy-index.schema.json"

SUPPORT_REGISTRY_SCHEMA = "schemas/ptsip-support-governance-registry.schema.json"
DEVELOPER_REGISTRY_SCHEMA = "developer/policy/schemas/developer-governance-registry.schema.json"
SUPPORT_SEMANTICS_SCHEMA = "schemas/ptsip-support-authority-semantics.schema.json"
SUPPORT_SEMANTICS_EMBEDDED_SCHEMA = "src/ptsip/specdata/ptsip-support-authority-semantics.schema.json"
DEVELOPER_SEMANTICS_SCHEMA = "developer/policy/schemas/developer-authority-semantics.schema.json"

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
RELATION_KINDS = ("supersedes", "amends", "extends", "depends_on")


def _mapping(value: object) -> Mapping[str, object] | None:
    return value if isinstance(value, Mapping) else None


def _validate_current_registry_planes(
    base: Path,
    *,
    sfp_ids: tuple[str, ...],
    mpd_ids: tuple[str, ...],
) -> list[str]:
    errors: list[str] = []

    support_registry_schema = load_json(SUPPORT_REGISTRY_SCHEMA, root=base)
    developer_registry_schema = load_json(DEVELOPER_REGISTRY_SCHEMA, root=base)
    support_semantics = load_json(SUPPORT_SEMANTICS_SCHEMA, root=base)
    embedded_support_semantics = load_json(SUPPORT_SEMANTICS_EMBEDDED_SCHEMA, root=base)
    developer_semantics = load_json(DEVELOPER_SEMANTICS_SCHEMA, root=base)

    for schema in (
        support_registry_schema,
        developer_registry_schema,
        support_semantics,
        developer_semantics,
    ):
        Draft202012Validator.check_schema(schema)

    if support_semantics != embedded_support_semantics:
        errors.append("support authority semantics canonical and embedded schemas differ")

    support_payloads: list[dict[str, object]] = []
    for path in SUPPORT_REGISTRIES:
        payload = load_yaml(path, root=base)
        support_payloads.append(payload)
        for error in Draft202012Validator(support_registry_schema).iter_errors(payload):
            errors.append(f"{path}: {error.message}")
        if "decisions/" in (base / path).read_text(encoding="utf-8"):
            errors.append(f"{path}: shipped support registry must not depend on legacy decisions paths")

    developer_payloads: list[dict[str, object]] = []
    for path in DEVELOPER_REGISTRIES:
        payload = load_yaml(path, root=base)
        developer_payloads.append(payload)
        for error in Draft202012Validator(developer_registry_schema).iter_errors(payload):
            errors.append(f"{path}: {error.message}")

    schema_registry = support_payloads[0]
    support_schema_entries = schema_registry.get("entries", [])
    if not isinstance(support_schema_entries, list):
        errors.append("support authority schema registry entries must be a list")
        support_schema_entries = []
    if [
        item.get("policy_id")
        for item in support_schema_entries
        if isinstance(item, Mapping)
    ] != list(sfp_ids):
        errors.append("support authority schema registry must cover support-policy-index exactly")

    role_registry = support_payloads[1]
    support_role_entries = role_registry.get("policy_roles", [])
    if not isinstance(support_role_entries, list):
        errors.append("support authority role registry policy_roles must be a list")
        support_role_entries = []
    if [
        item.get("policy_id")
        for item in support_role_entries
        if isinstance(item, Mapping)
    ] != list(sfp_ids):
        errors.append("support authority role registry must cover support-policy-index exactly")

    effect_vocabulary = _mapping(role_registry.get("effect_vocabulary"))
    if effect_vocabulary is None:
        errors.append("support authority role registry effect_vocabulary must be a mapping")
    else:
        tokens = effect_vocabulary.get("tokens")
        count = effect_vocabulary.get("count")
        if not isinstance(tokens, list) or count != len(tokens) or len(tokens) != len(set(tokens)):
            errors.append("support authority effect vocabulary count/uniqueness mismatch")

    subject_registry = support_payloads[2]
    if "current_repository_bindings" in subject_registry:
        errors.append("support subject registry must not ship PTSIP repository bindings")
    subject_schemes = _mapping(subject_registry.get("subject_identity_schemes"))
    if subject_schemes is None or set(subject_schemes) != {"SUPPORT_POLICY_ID"}:
        errors.append("support subject registry must use SUPPORT_POLICY_ID only")
    else:
        support_identity = _mapping(subject_schemes.get("SUPPORT_POLICY_ID"))
        registered = None if support_identity is None else support_identity.get("registered_values")
        if registered != list(sfp_ids):
            errors.append("support subject registry SUPPORT_POLICY_ID values must match support-policy-index")

    auth_registry = support_payloads[3]
    for forbidden in ("authorization_provenance", "rules", "held_scopes"):
        if forbidden in auth_registry:
            errors.append(f"support authorization registry must not ship {forbidden}")

    developer_schema_registry = developer_payloads[0]
    developer_schema_entries = developer_schema_registry.get("entries", [])
    if not isinstance(developer_schema_entries, list):
        errors.append("developer authority schema registry entries must be a list")
        developer_schema_entries = []
    developer_registry_ids = [
        str(item.get("policy_id"))
        for item in developer_schema_entries
        if isinstance(item, Mapping)
    ]
    expected_developer_registry_ids = [policy_id for policy_id in mpd_ids if policy_id != "MPD-0001"]
    if developer_registry_ids != expected_developer_registry_ids:
        errors.append(
            "developer authority schema registry must cover current migrated MPD policies exactly"
        )

    developer_role_registry = developer_payloads[1]
    developer_role_entries = developer_role_registry.get("policy_roles", [])
    if not isinstance(developer_role_entries, list):
        errors.append("developer authority role registry policy_roles must be a list")
        developer_role_entries = []
    if [
        item.get("policy_id")
        for item in developer_role_entries
        if isinstance(item, Mapping)
    ] != expected_developer_registry_ids:
        errors.append(
            "developer authority role registry must cover current migrated MPD policies exactly"
        )

    for entry in support_schema_entries:
        if not isinstance(entry, Mapping):
            continue
        policy_id = entry.get("policy_id")
        definition_name = entry.get("schema_definition")
        definition = support_semantics.get("$defs", {}).get(definition_name)
        if not isinstance(policy_id, str) or not isinstance(definition, Mapping):
            errors.append(f"support authority schema registry entry is unresolved: {entry!r}")
            continue
        policy = load_yaml(f"src/ptsip/specdata/{policy_id}.yaml", root=base)
        semantics = policy.get("authority_semantics")
        for error in Draft202012Validator(definition).iter_errors(semantics):
            errors.append(f"{policy_id}: {error.message}")

    for entry in developer_schema_entries:
        if not isinstance(entry, Mapping):
            continue
        policy_id = entry.get("policy_id")
        definition_name = entry.get("schema_definition")
        definition = developer_semantics.get("$defs", {}).get(definition_name)
        if not isinstance(policy_id, str) or not isinstance(definition, Mapping):
            errors.append(f"developer authority schema registry entry is unresolved: {entry!r}")
            continue
        policy = load_yaml(f"developer/policy/{policy_id}.yaml", root=base)
        rules = _mapping(policy.get("rules"))
        semantics = None if rules is None else rules.get("authority_semantics")
        for error in Draft202012Validator(definition).iter_errors(semantics):
            errors.append(f"{policy_id}: {error.message}")

    return errors


def validate_developer_policy(root: str | Path | None = None) -> tuple[str, ...]:
    base = repository_root(root)
    errors: list[str] = []

    index = load_yaml(INDEX, root=base)
    index_schema = load_json(INDEX_SCHEMA, root=base)
    mpd_schema = load_json(MPD_SCHEMA, root=base)
    sfp_index = load_yaml(SFP_INDEX, root=base)
    sfp_index_schema = load_json(SFP_INDEX_CANONICAL_SCHEMA, root=base)
    sfp_index_embedded_schema = load_json(SFP_INDEX_EMBEDDED_SCHEMA, root=base)
    sfp_schema = load_json(SFP_CANONICAL_SCHEMA, root=base)
    sfp_embedded_schema = load_json(SFP_EMBEDDED_SCHEMA, root=base)

    for schema in (
        index_schema,
        mpd_schema,
        sfp_index_schema,
        sfp_index_embedded_schema,
        sfp_schema,
        sfp_embedded_schema,
    ):
        Draft202012Validator.check_schema(schema)

    if sfp_index_schema != sfp_index_embedded_schema:
        errors.append("Support Feature Policy index canonical and embedded schemas differ")
    if sfp_schema != sfp_embedded_schema:
        errors.append("Support Feature Policy canonical and embedded schemas differ")

    for error in Draft202012Validator(index_schema).iter_errors(index):
        errors.append(f"{INDEX}: {error.message}")
    for error in Draft202012Validator(sfp_index_schema).iter_errors(sfp_index):
        errors.append(f"{SFP_INDEX}: {error.message}")

    mpd_entries = index.get("policies", [])
    sfp_entries = sfp_index.get("policies", [])
    if not isinstance(mpd_entries, list):
        errors.append(f"{INDEX}: policies must be a list")
        mpd_entries = []
    if not isinstance(sfp_entries, list):
        errors.append(f"{SFP_INDEX}: policies must be a list")
        sfp_entries = []

    mpd_ids = tuple(
        str(entry.get("id"))
        for entry in mpd_entries
        if isinstance(entry, Mapping)
    )
    sfp_ids = tuple(
        str(entry.get("id"))
        for entry in sfp_entries
        if isinstance(entry, Mapping)
    )

    expected_mpd_ids = tuple(f"MPD-{number:04d}" for number in range(1, 10))
    expected_sfp_ids = tuple(f"SFP-{number:04d}" for number in range(1, 22))
    if mpd_ids != expected_mpd_ids:
        errors.append("developer policy index must contain MPD-0001 through MPD-0009 in order")
    if sfp_ids != expected_sfp_ids:
        errors.append("support policy index must contain SFP-0001 through SFP-0021 in order")

    all_policy_ids = set(mpd_ids) | set(sfp_ids)
    if len(all_policy_ids) != len(mpd_ids) + len(sfp_ids):
        errors.append("current policy IDs must be globally unique across MPD and SFP indexes")

    current_records: dict[str, dict[str, object]] = {}
    for entry in mpd_entries:
        if not isinstance(entry, Mapping):
            continue
        policy_id = entry.get("id")
        path = entry.get("path")
        if not isinstance(policy_id, str) or not isinstance(path, str):
            continue
        payload = load_yaml(path, root=base)
        current_records[policy_id] = payload
        for error in Draft202012Validator(mpd_schema).iter_errors(payload):
            errors.append(f"{path}: {error.message}")
        policy = _mapping(payload.get("policy"))
        if policy is None:
            errors.append(f"{path}: policy must be a mapping")
            continue
        if policy.get("id") != policy_id:
            errors.append(f"{path}: policy.id does not match index id")
        if policy.get("status") != entry.get("status"):
            errors.append(f"{path}: policy.status does not match index status")

    sfp_validator = Draft202012Validator(sfp_schema)
    for entry in sfp_entries:
        if not isinstance(entry, Mapping):
            continue
        policy_id = entry.get("id")
        path = entry.get("path")
        if not isinstance(policy_id, str) or not isinstance(path, str):
            continue
        payload = load_yaml(path, root=base)
        current_records[policy_id] = payload
        for error in sfp_validator.iter_errors(payload):
            errors.append(f"{path}: {error.message}")
        policy = _mapping(payload.get("policy"))
        if policy is None:
            errors.append(f"{path}: policy must be a mapping")
            continue
        if policy.get("id") != policy_id:
            errors.append(f"{path}: policy.id does not match index id")
        if policy.get("status") != entry.get("status"):
            errors.append(f"{path}: policy.status does not match index status")
        raw_text = (base / path).read_text(encoding="utf-8")
        for token in ("subject_binding:", "authority_role:", "repository_binding:"):
            if token in raw_text:
                errors.append(f"{path}: forbidden legacy developer wrapper {token}")

    for source_id, payload in current_records.items():
        relations = payload.get("relations")
        if relations is None:
            continue
        relation_map = _mapping(relations)
        if relation_map is None:
            errors.append(f"{source_id}: relations must be a mapping")
            continue
        for relation_kind in RELATION_KINDS:
            edges = relation_map.get(relation_kind, [])
            if not isinstance(edges, list):
                errors.append(f"{source_id}: relations.{relation_kind} must be a list")
                continue
            for edge in edges:
                edge_map = _mapping(edge)
                if edge_map is None:
                    errors.append(f"{source_id}: invalid {relation_kind} relation entry")
                    continue
                target_id = edge_map.get("policy")
                if not isinstance(target_id, str) or target_id not in all_policy_ids:
                    errors.append(
                        f"{source_id}: {relation_kind} targets unknown current policy {target_id!r}"
                    )
                    continue
                if source_id.startswith("SFP-") and target_id.startswith("MPD-"):
                    errors.append(
                        f"{source_id}: forbidden current policy relation boundary "
                        f"{source_id} -> {target_id}"
                    )

    errors.extend(
        _validate_current_registry_planes(
            base,
            sfp_ids=sfp_ids,
            mpd_ids=mpd_ids,
        )
    )

    return tuple(errors)


if __name__ == "__main__":
    failures = validate_developer_policy()
    if failures:
        raise SystemExit("\n".join(failures))
    print("Developer policy validation: PASS")
