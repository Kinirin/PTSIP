from __future__ import annotations

from pathlib import Path

from jsonschema import Draft202012Validator

from developer.automation.policy_loader import load_json, load_yaml, repository_root
from developer.automation.decision_reference_migrator import (
    UnroutedDecisionReferenceError,
    project_relation,
)
from developer.automation.registry_split_validator import validate_registry_split
from developer.automation.policy_materializer import (
    expected_materialized_policies,
    expected_mpd_index_entries,
    expected_support_policy_index,
)


INDEX = "developer/policy/index.yaml"
INDEX_SCHEMA = "developer/policy/schemas/developer-policy-index.schema.json"
MPD_SCHEMA = "developer/policy/schemas/management-policy.schema.json"
LEGACY_INVENTORY = "developer/policy/legacy-decisions-inventory.yaml"
LEGACY_INVENTORY_SCHEMA = "developer/policy/schemas/legacy-decision-inventory.schema.json"
LEGACY_ROUTING = "developer/policy/legacy-decision-reference-routing.yaml"
LEGACY_ROUTING_SCHEMA = "developer/policy/schemas/legacy-decision-reference-routing.schema.json"
SPLIT_TEXTUAL_REVIEW = "developer/policy/split-textual-reference-review.yaml"
SPLIT_TEXTUAL_REVIEW_SCHEMA = "developer/policy/schemas/split-textual-reference-review.schema.json"
RELATION_MIGRATION = "developer/policy/policy-relation-migration.yaml"
RELATION_MIGRATION_SCHEMA = "developer/policy/schemas/policy-relation-migration.schema.json"
SFP_CANONICAL_SCHEMA = "schemas/ptsip-support-feature-policy.schema.json"
SFP_EMBEDDED_SCHEMA = "src/ptsip/specdata/ptsip-support-feature-policy.schema.json"
SFP_INDEX = "src/ptsip/specdata/support-policy-index.yaml"
SFP_INDEX_CANONICAL_SCHEMA = "schemas/ptsip-support-feature-policy-index.schema.json"
SFP_INDEX_EMBEDDED_SCHEMA = "src/ptsip/specdata/ptsip-support-feature-policy-index.schema.json"


def validate_developer_policy(root: str | Path | None = None) -> tuple[str, ...]:
    base = repository_root(root)
    errors: list[str] = []
    index = load_yaml(INDEX, root=base)
    index_schema = load_json(INDEX_SCHEMA, root=base)
    mpd_schema = load_json(MPD_SCHEMA, root=base)
    inventory = load_yaml(LEGACY_INVENTORY, root=base)
    inventory_schema = load_json(LEGACY_INVENTORY_SCHEMA, root=base)
    routing = load_yaml(LEGACY_ROUTING, root=base)
    routing_schema = load_json(LEGACY_ROUTING_SCHEMA, root=base)
    split_review = load_yaml(SPLIT_TEXTUAL_REVIEW, root=base)
    split_review_schema = load_json(SPLIT_TEXTUAL_REVIEW_SCHEMA, root=base)
    relation_migration = load_yaml(RELATION_MIGRATION, root=base)
    relation_migration_schema = load_json(RELATION_MIGRATION_SCHEMA, root=base)
    Draft202012Validator.check_schema(index_schema)
    Draft202012Validator.check_schema(mpd_schema)
    Draft202012Validator.check_schema(inventory_schema)
    Draft202012Validator.check_schema(routing_schema)
    Draft202012Validator.check_schema(split_review_schema)
    Draft202012Validator.check_schema(relation_migration_schema)
    for error in Draft202012Validator(index_schema).iter_errors(index):
        errors.append(f"developer/policy/index.yaml: {error.message}")
    for error in Draft202012Validator(inventory_schema).iter_errors(inventory):
        errors.append(f"{LEGACY_INVENTORY}: {error.message}")
    for error in Draft202012Validator(routing_schema).iter_errors(routing):
        errors.append(f"{LEGACY_ROUTING}: {error.message}")
    for error in Draft202012Validator(split_review_schema).iter_errors(split_review):
        errors.append(f"{SPLIT_TEXTUAL_REVIEW}: {error.message}")
    for error in Draft202012Validator(relation_migration_schema).iter_errors(relation_migration):
        errors.append(f"{RELATION_MIGRATION}: {error.message}")
    inventory_path = index.get("legacy_decisions_migration", {}).get("inventory_path")
    if inventory_path != LEGACY_INVENTORY:
        errors.append("developer/policy/index.yaml: legacy decision inventory_path is not canonical")
    inventory_entries = inventory.get("entries", [])
    expected_ids = [f"ADR-{number:04d}" for number in range(1, 24)]
    actual_ids = [item.get("source_id") for item in inventory_entries if isinstance(item, dict)]
    if actual_ids != expected_ids:
        errors.append("legacy decision inventory must contain ADR-0001 through ADR-0023 in order")
    target_ids: list[str] = []
    class_counts = {"MPD": 0, "SFP": 0, "SPLIT": 0, "RETIRE": 0}
    retired_fragments = 0
    for item in inventory_entries:
        if not isinstance(item, dict):
            continue
        classification = item.get("classification")
        if classification in class_counts:
            class_counts[classification] += 1
        source_path = item.get("source_path")
        if isinstance(source_path, str):
            source = load_yaml(source_path, root=base)
            decision = source.get("decision", {})
            if decision.get("id") != item.get("source_id"):
                errors.append(f"{source_path}: decision.id does not match inventory")
            if decision.get("topic_id") != item.get("topic_id"):
                errors.append(f"{source_path}: decision.topic_id does not match inventory")
        outputs = item.get("outputs", [])
        if classification == "SPLIT" and len(outputs) < 2:
            errors.append(f"{item.get('source_id')}: SPLIT requires at least two outputs")
        if classification in {"MPD", "SFP"} and len(outputs) != 1:
            errors.append(f"{item.get('source_id')}: pure classification requires exactly one output")
        for output in outputs:
            if not isinstance(output, dict):
                continue
            target_id = output.get("target_id")
            if isinstance(target_id, str):
                target_ids.append(target_id)
            if output.get("disposition") == "RETIRE_FRAGMENT":
                retired_fragments += 1
    if len(target_ids) != len(set(target_ids)):
        errors.append("legacy decision inventory target IDs must be unique")
    inventory_routes = {
        item.get("source_id"): tuple(
            output.get("target_id")
            for output in item.get("outputs", [])
            if isinstance(output, dict) and isinstance(output.get("target_id"), str)
        )
        for item in inventory_entries
        if isinstance(item, dict)
    }
    routing_routes = {
        adr_id: tuple(route.get("targets", []))
        for adr_id, route in routing.get("id_routes", {}).items()
        if isinstance(route, dict)
    }
    if inventory_routes != routing_routes:
        errors.append("legacy decision reference routing must exactly match inventory target allocation")
    for adr_id, targets in routing_routes.items():
        route = routing.get("id_routes", {}).get(adr_id, {})
        mode = route.get("textual_reference_mode") if isinstance(route, dict) else None
        expected_mode = "MANUAL_CONTEXT_REVIEW" if len(targets) > 1 else "AUTO_SINGLE_TARGET"
        if mode != expected_mode:
            errors.append(f"{adr_id}: textual_reference_mode must be {expected_mode}")
    summary = inventory.get("summary", {})
    if any(summary.get(key) != value for key, value in class_counts.items()):
        errors.append("legacy decision inventory classification summary does not match entries")
    if summary.get("planned_sfp_targets") != sum(item.startswith("SFP-") for item in target_ids):
        errors.append("legacy decision inventory SFP target count does not match entries")
    if summary.get("planned_mpd_targets") != sum(item.startswith("MPD-") for item in target_ids):
        errors.append("legacy decision inventory MPD target count does not match entries")
    if summary.get("retired_fragments") != retired_fragments:
        errors.append("legacy decision inventory retired fragment count does not match entries")

    manifest_source_relations = {
        (
            item["source_relation"]["source_adr"],
            item["source_relation"]["relation"],
            item["source_relation"]["target_adr"],
            item["source_relation"]["scope"],
        )
        for item in relation_migration.get("relations", [])
        if isinstance(item, dict) and isinstance(item.get("source_relation"), dict)
    }
    actual_source_relations: set[tuple[str, str, str, str | None]] = set()

    # Every machine relation in the legacy ADR corpus must project deterministically.
    # Unique-to-unique relations may project automatically; any relation touching a
    # SPLIT source or target must have a predeclared relation route.
    relation_kinds = ("depends_on", "amends", "extends", "supersedes")
    for item in inventory_entries:
        if not isinstance(item, dict):
            continue
        source_id = item.get("source_id")
        source_path = item.get("source_path")
        if not isinstance(source_id, str) or not isinstance(source_path, str):
            continue
        source = load_yaml(source_path, root=base)
        relations = source.get("relations", {})
        if not isinstance(relations, dict):
            continue
        for relation_kind in relation_kinds:
            values = relations.get(relation_kind, [])
            if not isinstance(values, list):
                continue
            for relation in values:
                if isinstance(relation, str):
                    target_id = relation
                    scope = None
                elif isinstance(relation, dict):
                    target_id = relation.get("adr")
                    scope = relation.get("scope")
                else:
                    errors.append(f"{source_path}: invalid relation entry in {relation_kind}")
                    continue
                if not isinstance(target_id, str):
                    errors.append(f"{source_path}: relation {relation_kind} is missing ADR target")
                    continue
                relation_key = (
                    source_id,
                    relation_kind,
                    target_id,
                    scope if isinstance(scope, str) else None,
                )
                actual_source_relations.add(relation_key)
                try:
                    edges = project_relation(
                        source_id,
                        relation_kind,
                        target_id,
                        scope=scope if isinstance(scope, str) else None,
                        root=base,
                    )
                except UnroutedDecisionReferenceError as exc:
                    errors.append(f"{source_path}: {exc}")
                    continue
                if not edges:
                    errors.append(
                        f"{source_path}: relation {relation_kind} to {target_id} projected no edges"
                    )

    if actual_source_relations != manifest_source_relations:
        errors.append(
            "policy relation migration manifest must exactly cover every legacy ADR machine relation"
        )

    allowed_boundaries = {
        ("SFP", "SFP"),
        ("MPD", "MPD"),
        ("MPD", "SFP"),
    }
    for item in relation_migration.get("relations", []):
        if not isinstance(item, dict):
            continue
        for edge in item.get("projected_edges", []):
            if not isinstance(edge, dict):
                continue
            source = str(edge.get("source_policy", ""))
            target = str(edge.get("target_policy", ""))
            boundary = (source[:3], target[:3])
            if boundary not in allowed_boundaries:
                errors.append(
                    f"{item.get('migration_id')}: forbidden policy relation boundary {source} -> {target}"
                )

    for entry in index.get("policies", []):
        path = entry.get("path")
        if not isinstance(path, str):
            continue
        payload = load_yaml(path, root=base)
        for error in Draft202012Validator(mpd_schema).iter_errors(payload):
            errors.append(f"{path}: {error.message}")
        if payload.get("policy", {}).get("id") != entry.get("id"):
            errors.append(f"{path}: policy.id does not match index id")
    expected_policies = expected_materialized_policies(base)
    actual_migrated_mpd_entries = [
        entry for entry in index.get("policies", [])
        if isinstance(entry, dict) and entry.get("id") != "MPD-0001"
    ]
    if actual_migrated_mpd_entries != expected_mpd_index_entries(base):
        errors.append("developer policy index does not match materialized MPD corpus")

    sfp_index = load_yaml(SFP_INDEX, root=base)
    sfp_index_schema = load_json(SFP_INDEX_CANONICAL_SCHEMA, root=base)
    sfp_index_embedded_schema = load_json(SFP_INDEX_EMBEDDED_SCHEMA, root=base)
    Draft202012Validator.check_schema(sfp_index_schema)
    Draft202012Validator.check_schema(sfp_index_embedded_schema)
    if sfp_index_schema != sfp_index_embedded_schema:
        errors.append("Support Feature Policy index canonical and embedded schemas differ")
    for error in Draft202012Validator(sfp_index_schema).iter_errors(sfp_index):
        errors.append(f"{SFP_INDEX}: {error.message}")
    if sfp_index != expected_support_policy_index(base):
        errors.append("Support Feature Policy index does not match materialized SFP corpus")

    canonical = load_json(SFP_CANONICAL_SCHEMA, root=base)
    embedded = load_json(SFP_EMBEDDED_SCHEMA, root=base)
    Draft202012Validator.check_schema(canonical)
    Draft202012Validator.check_schema(embedded)
    if canonical != embedded:
        errors.append("Support Feature Policy canonical and embedded schemas differ")

    sfp_validator = Draft202012Validator(canonical)
    for path, expected in expected_policies.items():
        actual = load_yaml(path, root=base)
        if actual != expected:
            errors.append(f"{path}: materialized policy differs from deterministic materializer")
        if path.startswith("src/ptsip/specdata/SFP-"):
            for error in sfp_validator.iter_errors(actual):
                errors.append(f"{path}: {error.message}")
            raw_text = (base / path).read_text(encoding="utf-8")
            forbidden = ("subject_binding:", "authority_role:", "repository_binding:")
            for token in forbidden:
                if token in raw_text:
                    errors.append(f"{path}: forbidden legacy developer wrapper {token}")

    errors.extend(validate_registry_split(base))

    return tuple(errors)


if __name__ == "__main__":
    failures = validate_developer_policy()
    if failures:
        raise SystemExit("\n".join(failures))
    print("Developer policy validation: PASS")
