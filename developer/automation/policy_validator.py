from __future__ import annotations

from pathlib import Path
import re
from typing import Mapping

from jsonschema import Draft202012Validator
from referencing import Registry, Resource

from developer.automation.policy_loader import load_json, load_yaml, repository_root
from developer.automation.policy_responsibility_gate import (
    FAMILIES,
    validate_analysis_semantics,
)


INDEX = "developer/policy/index.yaml"
INDEX_SCHEMA = "developer/policy/schemas/developer-policy-index.schema.json"
MPD_SCHEMA = "developer/policy/schemas/management-policy.schema.json"
GOVERNANCE_SOURCE_REGISTRY = "developer/policy/registries/governance-source-registry.yaml"
GOVERNANCE_SOURCE_REGISTRY_SCHEMA = "developer/policy/schemas/governance-source-registry.schema.json"
SOURCE_APPLICATION_REVIEW = "developer/policy/source-application-review.yaml"
SOURCE_APPLICATION_REVIEW_SCHEMA = "developer/policy/schemas/source-application-review.schema.json"
APPROVAL_PROVENANCE_SCHEMA = "developer/policy/schemas/policy-approval-provenance.schema.json"
APPROVAL_PROVENANCE_ROOT = "developer/policy/approvals"
RESPONSIBILITY_ANALYSIS_ROOT = "developer/policy/analysis"
RESPONSIBILITY_ANALYSIS_SCHEMA = "developer/policy/schemas/policy-responsibility-analysis.schema.json"
RESPONSIBILITY_ANALYSIS_REGISTRY = "developer/policy/analysis/registry.yaml"
RESPONSIBILITY_ANALYSIS_REGISTRY_SCHEMA = "developer/policy/schemas/policy-materialization-analysis-registry.schema.json"
NEUTRAL_CATALOG_CONTRACTS = "developer/policy/registries/developer-policy-catalog-contracts.json"
NEUTRAL_CATALOG_CONTRACTS_SCHEMA = "developer/policy/schemas/developer-policy-catalog-contracts.schema.json"

SUPPORT_POLICY_ROOT = "src/policy"
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
_FAMILY_POLICY_ID_RE = re.compile(
    r"^MPD-(SPEC|PLAN|WORK|VERI|MIGR|RELS)-[0-9]{4}$"
)
_POLICY_VERSION_RE = re.compile(r"^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$")


def _canonical_mpd_path(policy_id: str) -> str:
    match = _FAMILY_POLICY_ID_RE.fullmatch(policy_id)
    if match is not None:
        return f"developer/policy/{match.group(1)}/{policy_id}.yaml"
    return f"developer/policy/{policy_id}.yaml"


def _mapping(value: object) -> Mapping[str, object] | None:
    return value if isinstance(value, Mapping) else None



def _validate_policy_version_semantics(
    policy_id: str,
    payload: Mapping[str, object],
) -> list[str]:
    """Validate explicit Policy version/status integrity without inferring either field."""

    policy = _mapping(payload.get("policy"))
    if policy is None:
        return []

    version = policy.get("version")
    status = policy.get("status")
    if not isinstance(version, str):
        return [f"{policy_id}: policy.version must be an explicit string"]
    match = _POLICY_VERSION_RE.fullmatch(version)
    if match is None:
        return [f"{policy_id}: invalid policy.version {version!r}"]

    major = int(match.group(1))
    expected = (
        major == 0 if status == "DRAFT"
        else major == 1 if status == "APPROVED"
        else major >= 2 if status in {"ACTIVE", "SUPERSEDED", "RETIRED"}
        else False
    )
    if not expected:
        return [
            f"{policy_id}: POLICY_VERSION_STATUS_MISMATCH "
            f"version={version!r} status={status!r}"
        ]
    return []


def _validate_policy_transition_semantics(
    policy_id: str,
    payload: Mapping[str, object],
) -> list[str]:
    """Validate machine transition dependencies and derived lifecycle state."""

    errors: list[str] = []
    policy = _mapping(payload.get("policy"))
    transition = _mapping(payload.get("transition"))
    if policy is None or transition is None:
        return errors

    requirements = transition.get("requirements")
    if not isinstance(requirements, list):
        return errors

    requirement_maps = [
        item for item in requirements if isinstance(item, Mapping)
    ]
    requirement_ids = [
        str(item.get("id"))
        for item in requirement_maps
        if isinstance(item.get("id"), str)
    ]
    if len(requirement_ids) != len(set(requirement_ids)):
        errors.append(f"{policy_id}: transition requirement IDs must be unique")

    states = {
        str(item["id"]): item.get("state")
        for item in requirement_maps
        if isinstance(item.get("id"), str)
    }
    known_ids = set(states)
    dependencies: dict[str, tuple[str, ...]] = {}

    for item in requirement_maps:
        requirement_id = item.get("id")
        if not isinstance(requirement_id, str):
            continue
        next_action = _mapping(item.get("next_action"))
        after = () if next_action is None else next_action.get("after", ())
        if not isinstance(after, list):
            continue
        dependency_ids = tuple(
            str(value) for value in after if isinstance(value, str)
        )
        dependencies[requirement_id] = dependency_ids

        for dependency_id in dependency_ids:
            if dependency_id not in known_ids:
                errors.append(
                    f"{policy_id}: {requirement_id} references unknown transition "
                    f"requirement {dependency_id}"
                )
            if dependency_id == requirement_id:
                errors.append(
                    f"{policy_id}: {requirement_id} must not depend on itself"
                )

        if item.get("state") in {"IN_PROGRESS", "FAILED", "SATISFIED"}:
            unsatisfied = [
                dependency_id
                for dependency_id in dependency_ids
                if states.get(dependency_id) != "SATISFIED"
            ]
            if unsatisfied:
                errors.append(
                    f"{policy_id}: {requirement_id} cannot be {item.get('state')} "
                    "before after-dependencies are SATISFIED: "
                    + ", ".join(unsatisfied)
                )

    visiting: set[str] = set()
    visited: set[str] = set()

    def visit(requirement_id: str, lineage: tuple[str, ...]) -> None:
        if requirement_id in visited:
            return
        if requirement_id in visiting:
            cycle = " -> ".join((*lineage, requirement_id))
            errors.append(f"{policy_id}: transition dependency cycle: {cycle}")
            return
        visiting.add(requirement_id)
        for dependency_id in dependencies.get(requirement_id, ()):
            if dependency_id in known_ids:
                visit(dependency_id, (*lineage, requirement_id))
        visiting.remove(requirement_id)
        visited.add(requirement_id)

    for requirement_id in requirement_ids:
        visit(requirement_id, ())

    all_satisfied = bool(requirement_maps) and all(
        item.get("state") == "SATISFIED"
        for item in requirement_maps
    )
    policy_status = policy.get("status")
    transition_state = transition.get("state")

    if policy_status == "APPROVED":
        expected_state = "READY" if all_satisfied else "PENDING"
        if transition_state != expected_state:
            errors.append(
                f"{policy_id}: APPROVED transition state must be {expected_state}, "
                f"got {transition_state!r}"
            )
    elif policy_status == "ACTIVE" and transition is not None:
        if not all_satisfied:
            errors.append(
                f"{policy_id}: ACTIVE transition history requires all requirements SATISFIED"
            )
        if transition_state != "COMPLETE":
            errors.append(
                f"{policy_id}: ACTIVE transition history must be COMPLETE"
            )

    return errors

def _validate_current_registry_planes(
    base: Path,
    *,
    sfp_ids: tuple[str, ...],
    developer_authority_ids: tuple[str, ...],
    mpd_ids: tuple[str, ...],
    current_records: Mapping[str, Mapping[str, object]],
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
        policy = current_records.get(policy_id)
        if policy is None:
            errors.append(f"developer authority schema registry policy is unresolved: {policy_id}")
            continue
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

    authorization_registry = load_yaml(
        "developer/policy/registries/authorization-transition-registry.yaml",
        root=base,
    )
    authorization_provenance = _mapping(
        authorization_registry.get("authorization_provenance")
    )
    transition_semantics = _mapping(
        authorization_registry.get("transition_semantics")
    )
    if (
        authorization_provenance is None
        or authorization_provenance.get("authorization_source") != "USER_EXPLICIT"
    ):
        errors.append(
            "authorization transition registry must bind authorization_source to USER_EXPLICIT"
        )
    if (
        transition_semantics is None
        or transition_semantics.get("transition_source") != "AUTOMATION_DERIVED"
        or transition_semantics.get("may_create_official_authority") is not False
    ):
        errors.append(
            "authorization transition registry must use AUTOMATION_DERIVED "
            "for non-authority-creating transition derivation"
        )

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


def _validate_policy_responsibility_analysis_plane(
    base: Path,
    *,
    mpd_ids: tuple[str, ...],
    current_records: Mapping[str, Mapping[str, object]],
) -> list[str]:
    errors: list[str] = []

    gate_policy = current_records.get("MPD-WORK-0003")
    if gate_policy is None:
        errors.append(
            "MPD-WORK-0003: responsibility materialization gate policy is required"
        )
        return errors

    rules = _mapping(gate_policy.get("rules"))
    gate = (
        None
        if rules is None
        else _mapping(rules.get("policy_responsibility_materialization"))
    )
    static_enforcement = (
        None
        if gate is None
        else _mapping(gate.get("static_enforcement"))
    )
    baselines = (
        None
        if static_enforcement is None
        else _mapping(static_enforcement.get("grandfathered_family_maximums"))
    )
    if baselines is None or set(baselines) != set(FAMILIES):
        errors.append(
            "MPD-WORK-0003: grandfathered_family_maximums must define all Family vocabulary"
        )
        return errors

    for family in FAMILIES:
        value = baselines.get(family)
        if not isinstance(value, int) or value < 0 or value > 9999:
            errors.append(
                f"MPD-WORK-0003: invalid grandfathered maximum for {family}: {value!r}"
            )

    analysis_schema = load_json(RESPONSIBILITY_ANALYSIS_SCHEMA, root=base)
    registry_schema = load_json(
        RESPONSIBILITY_ANALYSIS_REGISTRY_SCHEMA,
        root=base,
    )
    Draft202012Validator.check_schema(analysis_schema)
    Draft202012Validator.check_schema(registry_schema)
    analysis_validator = Draft202012Validator(analysis_schema)

    analysis_payloads: dict[str, Mapping[str, object]] = {}
    analysis_root = base / RESPONSIBILITY_ANALYSIS_ROOT
    if analysis_root.is_dir():
        for path in sorted(analysis_root.glob("*.yaml")):
            if path.name == "registry.yaml":
                continue
            relative = path.relative_to(base).as_posix()
            payload = load_yaml(relative, root=base)
            for error in analysis_validator.iter_errors(payload):
                errors.append(f"{relative}: {error.message}")
            for error in validate_analysis_semantics(payload):
                errors.append(f"{relative}: {error}")
            analysis = _mapping(payload.get("analysis"))
            analysis_id = None if analysis is None else analysis.get("analysis_id")
            if isinstance(analysis_id, str):
                if analysis_id in analysis_payloads:
                    errors.append(f"{relative}: duplicate analysis_id {analysis_id}")
                analysis_payloads[analysis_id] = payload

    registry = load_yaml(RESPONSIBILITY_ANALYSIS_REGISTRY, root=base)
    for error in Draft202012Validator(registry_schema).iter_errors(registry):
        errors.append(f"{RESPONSIBILITY_ANALYSIS_REGISTRY}: {error.message}")

    bindings = registry.get("bindings", [])
    if not isinstance(bindings, list):
        errors.append(
            f"{RESPONSIBILITY_ANALYSIS_REGISTRY}: bindings must be a list"
        )
        bindings = []

    expected_policy_ids: list[str] = []
    for policy_id in mpd_ids:
        match = _FAMILY_POLICY_ID_RE.fullmatch(policy_id)
        if match is None:
            continue
        family = match.group(1)
        number = int(policy_id.rsplit("-", 1)[1])
        baseline = baselines.get(family)
        if isinstance(baseline, int) and number > baseline:
            expected_policy_ids.append(policy_id)

    bound_policy_ids = [
        str(item.get("policy_id"))
        for item in bindings
        if isinstance(item, Mapping)
    ]
    if bound_policy_ids != sorted(expected_policy_ids):
        errors.append(
            "policy responsibility analysis registry must cover every post-baseline "
            "Family policy exactly in policy-id order"
        )
    if len(bound_policy_ids) != len(set(bound_policy_ids)):
        errors.append(
            "policy responsibility analysis registry contains duplicate policy bindings"
        )

    for item in bindings:
        binding = _mapping(item)
        if binding is None:
            continue
        policy_id = binding.get("policy_id")
        analysis_ref = binding.get("analysis_ref")
        analysis_id = binding.get("analysis_id")
        group_id = binding.get("group_id")

        if not isinstance(policy_id, str) or policy_id not in mpd_ids:
            errors.append(
                f"{RESPONSIBILITY_ANALYSIS_REGISTRY}: unknown policy binding {policy_id!r}"
            )
            continue

        if not isinstance(analysis_ref, str) or not analysis_ref.startswith(
            f"{RESPONSIBILITY_ANALYSIS_ROOT}/"
        ):
            errors.append(
                f"{RESPONSIBILITY_ANALYSIS_REGISTRY}: invalid analysis_ref for {policy_id}"
            )
            continue
        if not (base / analysis_ref).is_file():
            errors.append(
                f"{RESPONSIBILITY_ANALYSIS_REGISTRY}: missing analysis for "
                f"{policy_id}: {analysis_ref}"
            )
            continue

        payload = load_yaml(analysis_ref, root=base)
        analysis = _mapping(payload.get("analysis"))
        if analysis is None:
            continue
        if analysis.get("analysis_id") != analysis_id:
            errors.append(
                f"{RESPONSIBILITY_ANALYSIS_REGISTRY}: {policy_id} analysis_id mismatch"
            )

        decision = _mapping(analysis.get("decision"))
        if decision is None:
            continue
        if decision.get("materialization_allowed") is not True:
            errors.append(
                f"{RESPONSIBILITY_ANALYSIS_REGISTRY}: {policy_id} binds blocked analysis"
            )

        groups = decision.get("materialization_groups", [])
        group = (
            next(
                (
                    entry
                    for entry in groups
                    if isinstance(entry, Mapping)
                    and entry.get("group_id") == group_id
                ),
                None,
            )
            if isinstance(groups, list)
            else None
        )
        if group is None:
            errors.append(
                f"{RESPONSIBILITY_ANALYSIS_REGISTRY}: {policy_id} "
                f"group {group_id!r} not found"
            )
            continue

        family_match = _FAMILY_POLICY_ID_RE.fullmatch(policy_id)
        if family_match is None or group.get("family") != family_match.group(1):
            errors.append(
                f"{RESPONSIBILITY_ANALYSIS_REGISTRY}: {policy_id} group Family mismatch"
            )

    return errors


def _neutral_catalog_resources(base: Path) -> tuple[dict[str, object], Registry]:
    record = load_json(NEUTRAL_CATALOG_CONTRACTS, root=base)
    schema = load_json(NEUTRAL_CATALOG_CONTRACTS_SCHEMA, root=base)
    Draft202012Validator.check_schema(schema)
    errors = [error.message for error in Draft202012Validator(schema).iter_errors(record)]
    if errors:
        raise ValueError("; ".join(errors))
    Draft202012Validator.check_schema(record)
    resources = Registry().with_resource(str(record["$id"]), Resource.from_contents(record))
    for contract in record["contracts"].values():
        path = base / contract["schema_ref"]
        if not path.resolve().is_relative_to(base.resolve()):
            raise ValueError("neutral catalog schema path escapes repository")
        target = load_json(contract["schema_ref"], root=base)
        Draft202012Validator.check_schema(target)
        pending: list[object] = [target]
        while pending:
            item = pending.pop()
            if isinstance(item, dict):
                reference = item.get("$ref")
                if reference is not None and not (
                    isinstance(reference, str)
                    and (reference.startswith("#/") or reference.startswith(str(record["$id"]) + "#/"))
                ):
                    raise ValueError("neutral catalog schema uses an unregistered external reference")
                pending.extend(item.values())
            elif isinstance(item, list):
                pending.extend(item)
        resources = resources.with_resource(str(target["$id"]), Resource.from_contents(target))
    return record, resources


def resolve_neutral_catalog_contract(
    contract_id: str, root: str | Path | None = None
) -> dict[str, object]:
    """Resolve an exact neutral artifact contract, not a policy or Task permission."""
    record, _ = _neutral_catalog_resources(repository_root(root))
    contract = record["contracts"].get(contract_id)
    if contract is None:
        raise ValueError(f"UNKNOWN_NEUTRAL_CATALOG_CONTRACT: {contract_id}")
    return {"contract_id": contract_id, **contract}


def validate_neutral_catalog_contract_registration(
    root: str | Path | None = None,
) -> tuple[str, ...]:
    """Validate the registered contract and bounded, non-executing change scope."""
    base = repository_root(root)
    errors: list[str] = []
    try:
        record, resources = _neutral_catalog_resources(base)
        contracts = record["contracts"]
        if set(record["entrypoints"].values()) != set(contracts):
            errors.append("neutral catalog entrypoints must cover each exact registered contract")
        for role, definition in (("index", "catalog"), ("subject", "subject")):
            contract_id = record["entrypoints"][role]
            contract = contracts[contract_id]
            definitions = record["$defs"]
            if contract_id != definitions[f"{definition}_schema_version"]["const"]:
                errors.append(f"neutral catalog {role} schema identity mismatch")
            if contract["artifact_class"] != definitions[f"{definition}_artifact_class"]["const"]:
                errors.append(f"neutral catalog {role} artifact identity mismatch")
            for reference in (contract["schema_ref"], contract["canonical_path"]):
                candidate = base / reference
                if not candidate.resolve().is_relative_to(base.resolve()) or not candidate.is_file():
                    errors.append(f"neutral catalog unresolved repository reference: {reference}")
            target = load_json(contract["schema_ref"], root=base)
            validator = Draft202012Validator(target, registry=resources)
            # Resolve all cross-layer identity references without enabling an artifact migration.
            expected = {"schema_version": contract_id, "artifact_class": contract["artifact_class"]}
            for field, value in expected.items():
                field_schema = target["properties"][field]
                errors.extend(error.message for error in validator.evolve(schema=field_schema).iter_errors(value))

        scope = record["change_scope"]
        targets = scope["materialization_targets"]
        paths = [target["path"] for target in targets]
        if len(paths) != len(set(paths)):
            errors.append("neutral catalog materialization targets must be unique")
        from developer.automation.policy_resolver import _validate_implementation_ref

        for target in targets + scope["deferred_application_targets"]:
            reference = target["path"]
            candidate = base / reference
            if not candidate.resolve().is_relative_to(base.resolve()) or not candidate.is_file():
                errors.append(f"neutral catalog unresolved scope target: {reference}")
                continue
            for name in target.get("python_functions", []):
                _validate_implementation_ref(
                    base, {"path": reference, "selector": {"kind": "PYTHON_FUNCTION", "name": name}}
                )
        for path in paths:
            if any(path.startswith(prefix) for prefix in scope["forbidden_write_roots"]):
                errors.append(f"neutral catalog forbidden materialization target: {path}")
        if not set(scope["verification_test_paths"]).issubset(paths):
            errors.append("neutral catalog verification paths must be registered materialization targets")
        for reference in scope["preserved_contracts"]:
            if not (base / reference).is_file():
                errors.append(f"neutral catalog preserved contract is missing: {reference}")
        gate = record["application_gate"]
        if not (base / gate["existing_policy_materialization_schema_ref"]).is_file():
            errors.append("neutral catalog policy materialization gate cannot be resolved")
    except (OSError, ValueError, KeyError, TypeError) as exc:
        errors.append(f"neutral catalog registration: {exc}")
    return tuple(errors)


def validate_neutral_catalog_snapshot(
    catalog: Mapping[str, object],
    subject: Mapping[str, object],
    policy_records: Mapping[str, Mapping[str, object]],
    *,
    source_index: Mapping[str, object],
    source_subject: Mapping[str, object],
    root: str | Path | None = None,
) -> tuple[str, ...]:
    """Check an in-memory migration snapshot; never write, register or activate policies.

    Callers supply the exact indexed source snapshot and its validated policy records.
    This is a catalog contract check, not the M3 authority resolver or M4 registrar.
    """
    base = repository_root(root)
    errors = list(validate_neutral_catalog_contract_registration(base))
    if errors:
        return tuple(errors)
    record, resources = _neutral_catalog_resources(base)
    for role, payload in (("index", catalog), ("subject", subject)):
        contract = record["contracts"][record["entrypoints"][role]]
        schema = load_json(contract["schema_ref"], root=base)
        errors.extend(
            f"neutral catalog {role}: {error.message}"
            for error in Draft202012Validator(schema, registry=resources).iter_errors(payload)
        )
    if errors:
        return tuple(errors)
    entries = catalog["policies"]
    ids = [entry["id"] for entry in entries]
    if ids != sorted(set(ids)):
        errors.append("neutral catalog IDs must be globally unique and canonically ordered")
    source_entries = source_index.get("policies", [])
    source_paths = {entry["id"]: entry["path"] for entry in source_entries}
    if ids != [entry["id"] for entry in source_entries] or set(ids) != set(policy_records):
        errors.append("neutral catalog membership must preserve the exact indexed source snapshot")
    definitions = record["$defs"]
    for entry in entries:
        policy_id = entry["id"]
        family = re.fullmatch(definitions["family_policy_id"]["pattern"], policy_id)
        expected_path = (
            f"developer/policy/{family.group(1)}/{policy_id}.yaml"
            if family else f"developer/policy/{policy_id}.yaml"
        )
        if entry["path"] != expected_path or source_paths.get(policy_id) != entry["path"]:
            errors.append(f"{policy_id}: neutral catalog canonical path mismatch")
        boundary = re.fullmatch(definitions["boundary_policy_id"]["pattern"], policy_id)
        boundary_class = definitions["boundary_policy_class"]["const"]
        if bool(boundary) != (entry["policy_class"] == boundary_class):
            errors.append(f"{policy_id}: neutral catalog boundary class/namespace mismatch")
        source = policy_records.get(policy_id)
        policy = None if source is None else _mapping(source.get("policy"))
        if source is None or policy is None or policy.get("id") != policy_id:
            errors.append(f"{policy_id}: neutral catalog unresolved source policy")
            continue
        if entry["policy_class"] != source.get("policy_class"):
            errors.append(f"{policy_id}: neutral catalog policy_class projection mismatch")
        if entry["status"] != policy.get("status"):
            errors.append(f"{policy_id}: neutral catalog status projection mismatch")
    registered = subject["subject_identity_schemes"]["MANAGEMENT_POLICY_ID"]["registered_values"]
    if registered != ids:
        errors.append("neutral subject registered policy IDs must exactly project catalog membership")
    for field in record["invariants"]["preserved_subject_fields"]:
        if subject.get(field) != source_subject.get(field):
            errors.append(f"neutral subject source semantics changed: {field}")
    return tuple(errors)


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
    expected_mpd_paths = tuple(_canonical_mpd_path(policy_id) for policy_id in mpd_ids)
    if indexed_mpd_paths != expected_mpd_paths:
        errors.append(
            "developer policy index paths must match canonical family materialization"
        )
    discovered_mpd_paths = tuple(
        path.relative_to(base).as_posix()
        for path in sorted((base / "developer" / "policy").rglob("MPD-*.yaml"))
    )
    if tuple(sorted(indexed_mpd_paths)) != discovered_mpd_paths:
        errors.append(
            "developer policy index must cover the current MPD corpus exactly"
        )
    if mpd_ids != tuple(sorted(mpd_ids)):
        errors.append("developer policy index MPD identities must be in canonical ascending order")

    expected_sfp_ids = tuple(f"SFP-{number:04d}" for number in range(1, 23))
    if sfp_ids != expected_sfp_ids:
        errors.append("support policy index must contain SFP-0001 through SFP-0022 in order")

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
        candidate = base / path
        if not candidate.is_file():
            errors.append(f"{path}: indexed policy file does not exist")
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
        for version_error in _validate_policy_version_semantics(policy_id, payload):
            errors.append(f"{path}: {version_error}")
        for transition_error in _validate_policy_transition_semantics(policy_id, payload):
            errors.append(f"{path}: {transition_error}")

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
    errors.extend(
        _validate_policy_responsibility_analysis_plane(
            base,
            mpd_ids=mpd_ids,
            current_records=current_records,
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
            current_records=current_records,
        )
    )

    errors.extend(validate_neutral_catalog_contract_registration(base))
    return tuple(errors)


if __name__ == "__main__":
    failures = validate_developer_policy()
    if failures:
        raise SystemExit("\n".join(failures))
    print("Developer policy validation: PASS")
