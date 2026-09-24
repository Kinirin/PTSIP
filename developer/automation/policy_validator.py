from __future__ import annotations

from pathlib import Path
from typing import Mapping

from jsonschema import Draft202012Validator

from developer.automation.policy_loader import load_json, load_yaml, repository_root


INDEX = "developer/policy/index.yaml"
INDEX_SCHEMA = "developer/policy/schemas/developer-policy-index.schema.json"
MPD_SCHEMA = "developer/policy/schemas/management-policy.schema.json"
GOVERNANCE_SOURCE_REGISTRY = "developer/policy/registries/governance-source-registry.yaml"
GOVERNANCE_SOURCE_REGISTRY_SCHEMA = "developer/policy/schemas/governance-source-registry.schema.json"
SOURCE_APPLICATION_REVIEW = "developer/policy/source-application-review.yaml"
SOURCE_APPLICATION_REVIEW_SCHEMA = "developer/policy/schemas/source-application-review.schema.json"
APPROVAL_PROVENANCE_SCHEMA = "developer/policy/schemas/policy-approval-provenance.schema.json"
APPROVAL_PROVENANCE_ROOT = "developer/policy/approvals"

SUPPORT_POLICY_ROOT = "docs/Support_policy/policy"
SFP_CANONICAL_SCHEMA = f"{SUPPORT_POLICY_ROOT}/schemas/ptsip-support-feature-policy.schema.json"
SFP_INDEX = f"{SUPPORT_POLICY_ROOT}/index.yaml"
SFP_INDEX_CANONICAL_SCHEMA = f"{SUPPORT_POLICY_ROOT}/schemas/ptsip-support-feature-policy-index.schema.json"

SUPPORT_REGISTRY_SCHEMA = f"{SUPPORT_POLICY_ROOT}/schemas/ptsip-support-governance-registry.schema.json"
DEVELOPER_REGISTRY_SCHEMA = "developer/policy/schemas/developer-governance-registry.schema.json"
SUPPORT_SEMANTICS_SCHEMA = f"{SUPPORT_POLICY_ROOT}/schemas/ptsip-support-authority-semantics.schema.json"
DEVELOPER_SEMANTICS_SCHEMA = "developer/policy/schemas/developer-authority-semantics.schema.json"

SUPPORT_REGISTRIES = (
    f"{SUPPORT_POLICY_ROOT}/registries/ptsip-support-authority-schema-registry.yaml",
    f"{SUPPORT_POLICY_ROOT}/registries/ptsip-support-authority-role-registry.yaml",
    f"{SUPPORT_POLICY_ROOT}/registries/ptsip-support-authority-subject-registry.yaml",
    f"{SUPPORT_POLICY_ROOT}/registries/ptsip-support-authorization-registry.yaml",
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
    developer_authority_ids: tuple[str, ...],
    mpd_ids: tuple[str, ...],
) -> list[str]:
    errors: list[str] = []

    support_registry_schema = load_json(SUPPORT_REGISTRY_SCHEMA, root=base)
    developer_registry_schema = load_json(DEVELOPER_REGISTRY_SCHEMA, root=base)
    support_semantics = load_json(SUPPORT_SEMANTICS_SCHEMA, root=base)
    developer_semantics = load_json(DEVELOPER_SEMANTICS_SCHEMA, root=base)

    for schema in (
        support_registry_schema,
        developer_registry_schema,
        support_semantics,
        developer_semantics,
    ):
        Draft202012Validator.check_schema(schema)

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
    expected_developer_registry_ids = list(developer_authority_ids)
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

    developer_effect_vocabulary = _mapping(developer_role_registry.get("effect_vocabulary"))
    if developer_effect_vocabulary is None:
        errors.append("developer authority role registry effect_vocabulary must be a mapping")
    else:
        tokens = developer_effect_vocabulary.get("tokens")
        count = developer_effect_vocabulary.get("count")
        if not isinstance(tokens, list) or count != len(tokens) or len(tokens) != len(set(tokens)):
            errors.append("developer authority effect vocabulary count/uniqueness mismatch")

    developer_subject_registry = developer_payloads[2]
    developer_subject_schemes = _mapping(
        developer_subject_registry.get("subject_identity_schemes")
    )
    if developer_subject_schemes is None:
        errors.append("developer authority subject registry subject_identity_schemes must be a mapping")
    else:
        management_policy_id = _mapping(
            developer_subject_schemes.get("MANAGEMENT_POLICY_ID")
        )
        registered_values = (
            None if management_policy_id is None else management_policy_id.get("registered_values")
        )
        if registered_values != list(mpd_ids):
            errors.append(
                "developer authority subject registry MANAGEMENT_POLICY_ID values must match developer policy index"
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
        policy = load_yaml(f"{SUPPORT_POLICY_ROOT}/{policy_id}.yaml", root=base)
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


def _validate_governance_source_plane(
    base: Path,
    *,
    current_records: Mapping[str, Mapping[str, object]],
    mpd_ids: tuple[str, ...],
) -> list[str]:
    errors: list[str] = []

    registry_schema = load_json(GOVERNANCE_SOURCE_REGISTRY_SCHEMA, root=base)
    review_schema = load_json(SOURCE_APPLICATION_REVIEW_SCHEMA, root=base)
    approval_schema = load_json(APPROVAL_PROVENANCE_SCHEMA, root=base)
    for schema in (registry_schema, review_schema, approval_schema):
        Draft202012Validator.check_schema(schema)

    registry = load_yaml(GOVERNANCE_SOURCE_REGISTRY, root=base)
    review = load_yaml(SOURCE_APPLICATION_REVIEW, root=base)

    for error in Draft202012Validator(registry_schema).iter_errors(registry):
        errors.append(f"{GOVERNANCE_SOURCE_REGISTRY}: {error.message}")
    for error in Draft202012Validator(review_schema).iter_errors(review):
        errors.append(f"{SOURCE_APPLICATION_REVIEW}: {error.message}")

    constants = _mapping(registry.get("constants"))
    expected_constants = ("USER_EXPLICIT", "AGENT_INFERRED", "AUTOMATION_DERIVED")
    if constants is None or tuple(constants) != expected_constants:
        errors.append(
            "governance source registry constants must be USER_EXPLICIT, "
            "AGENT_INFERRED, AUTOMATION_DERIVED in canonical order"
        )
        constants = {}

    user_explicit = _mapping(constants.get("USER_EXPLICIT"))
    agent_inferred = _mapping(constants.get("AGENT_INFERRED"))
    automation_derived = _mapping(constants.get("AUTOMATION_DERIVED"))

    if user_explicit is None or user_explicit.get("may_create_official_authority") is not True:
        errors.append("USER_EXPLICIT must be the only source allowed to create official authority")
    if agent_inferred is None or agent_inferred.get("may_create_official_authority") is not False:
        errors.append("AGENT_INFERRED must not create official authority")
    elif (
        agent_inferred.get("provisional_resolution_only") is not True
        or agent_inferred.get("explicit_user_opt_in_required") is not True
    ):
        errors.append(
            "AGENT_INFERRED must be provisional-resolution-only and require explicit user opt-in"
        )
    if automation_derived is None or automation_derived.get("may_create_official_authority") is not False:
        errors.append("AUTOMATION_DERIVED must not create official authority")
    elif automation_derived.get("may_propagate_existing_authority_only") is not True:
        errors.append("AUTOMATION_DERIVED may only propagate already registered authority")

    query_list = review.get("policy_query_list", [])
    if not isinstance(query_list, list):
        errors.append(f"{SOURCE_APPLICATION_REVIEW}: policy_query_list must be a list")
        query_list = []

    for item in query_list:
        item_map = _mapping(item)
        if item_map is None:
            continue
        policy_id = item_map.get("policy_id")
        expected_status = item_map.get("expected_status")
        sections = item_map.get("sections")
        if not isinstance(policy_id, str) or policy_id not in mpd_ids:
            errors.append(f"{SOURCE_APPLICATION_REVIEW}: unknown policy query {policy_id!r}")
            continue
        payload = current_records.get(policy_id)
        if payload is None:
            errors.append(f"{SOURCE_APPLICATION_REVIEW}: unresolved policy query {policy_id}")
            continue
        policy = _mapping(payload.get("policy"))
        rules = _mapping(payload.get("rules"))
        actual_status = None if policy is None else policy.get("status")
        if actual_status != expected_status:
            errors.append(
                f"{SOURCE_APPLICATION_REVIEW}: {policy_id} expected status "
                f"{expected_status!r}, got {actual_status!r}"
            )
        if actual_status != "ACTIVE" and item_map.get("review_role") != "REVIEW_ONLY_NOT_AUTHORITY":
            errors.append(
                f"{SOURCE_APPLICATION_REVIEW}: non-ACTIVE {policy_id} must be REVIEW_ONLY_NOT_AUTHORITY"
            )
        if not isinstance(sections, list) or rules is None:
            continue
        for section in sections:
            if section not in rules:
                errors.append(
                    f"{SOURCE_APPLICATION_REVIEW}: {policy_id} missing queried section {section!r}"
                )

    artifact_list = review.get("artifact_review_list", [])
    if isinstance(artifact_list, list):
        for item in artifact_list:
            item_map = _mapping(item)
            if item_map is None:
                continue
            path = item_map.get("path")
            if isinstance(path, str) and not (base / path).exists():
                errors.append(f"{SOURCE_APPLICATION_REVIEW}: review artifact does not exist: {path}")

    approval_validator = Draft202012Validator(approval_schema)
    valid_sources = set(constants)
    approval_root = base / APPROVAL_PROVENANCE_ROOT
    for path in sorted(approval_root.glob("*.yaml")):
        payload = load_yaml(path, root=base)
        relative = path.relative_to(base).as_posix()
        for error in approval_validator.iter_errors(payload):
            errors.append(f"{relative}: {error.message}")
        approval = _mapping(payload.get("approval"))
        if approval is None:
            continue
        source = approval.get("decision_source")
        if source not in valid_sources:
            errors.append(f"{relative}: unknown decision_source {source!r}")
        if source != "USER_EXPLICIT":
            errors.append(
                f"{relative}: policy approval provenance must use USER_EXPLICIT decision_source"
            )

    return errors


def validate_developer_policy(root: str | Path | None = None) -> tuple[str, ...]:
    base = repository_root(root)
    errors: list[str] = []

    index = load_yaml(INDEX, root=base)
    index_schema = load_json(INDEX_SCHEMA, root=base)
    mpd_schema = load_json(MPD_SCHEMA, root=base)
    sfp_index = load_yaml(SFP_INDEX, root=base)
    sfp_index_schema = load_json(SFP_INDEX_CANONICAL_SCHEMA, root=base)
    sfp_schema = load_json(SFP_CANONICAL_SCHEMA, root=base)

    for schema in (
        index_schema,
        mpd_schema,
        sfp_index_schema,
        sfp_schema,
    ):
        Draft202012Validator.check_schema(schema)

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

    indexed_mpd_paths = tuple(
        str(entry.get("path"))
        for entry in mpd_entries
        if isinstance(entry, Mapping)
    )
    discovered_mpd_paths = tuple(
        path.relative_to(base).as_posix()
        for path in sorted((base / "developer" / "policy").glob("MPD-*.yaml"))
    )
    if indexed_mpd_paths != discovered_mpd_paths:
        errors.append(
            "developer policy index must cover the current MPD corpus exactly in path order"
        )
    if mpd_ids != tuple(sorted(mpd_ids)):
        errors.append("developer policy index MPD identities must be in canonical ascending order")

    expected_sfp_ids = tuple(f"SFP-{number:04d}" for number in range(1, 22))
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
        payload = load_yaml(f"{SUPPORT_POLICY_ROOT}/{path}", root=base)
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
        raw_text = (base / SUPPORT_POLICY_ROOT / path).read_text(encoding="utf-8")
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
        _validate_governance_source_plane(
            base,
            current_records=current_records,
            mpd_ids=mpd_ids,
        )
    )

    developer_authority_ids: list[str] = []
    for policy_id in mpd_ids:
        payload = current_records.get(policy_id)
        if payload is None:
            continue
        policy = _mapping(payload.get("policy"))
        rules = _mapping(payload.get("rules"))
        if (
            policy is not None
            and policy.get("status") == "ACTIVE"
            and rules is not None
            and "authority_semantics" in rules
        ):
            developer_authority_ids.append(policy_id)

    errors.extend(
        _validate_current_registry_planes(
            base,
            sfp_ids=sfp_ids,
            developer_authority_ids=tuple(developer_authority_ids),
            mpd_ids=mpd_ids,
        )
    )

    return tuple(errors)


if __name__ == "__main__":
    failures = validate_developer_policy()
    if failures:
        raise SystemExit("\n".join(failures))
    print("Developer policy validation: PASS")
