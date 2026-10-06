from __future__ import annotations

import argparse
import json
import re
from pathlib import Path
from typing import Mapping, Sequence

from jsonschema import Draft202012Validator

from developer.automation.policy_loader import load_json, load_yaml, repository_root


INDEX = "developer/policy/index.yaml"
ANALYSIS_ROOT = Path("developer/policy/analysis")
ANALYSIS_RECORD_ROOT = ANALYSIS_ROOT / "records"
ANALYSIS_REGISTRY = ANALYSIS_ROOT / "registry.yaml"
ANALYSIS_SCHEMA = "developer/policy/analysis/schemas/policy-responsibility-analysis.schema.json"
ANALYSIS_REGISTRY_SCHEMA = "developer/policy/analysis/schemas/policy-materialization-analysis-registry.schema.json"
_ANALYSIS_ID_RE = re.compile(r"^PRA-[0-9]{4}$")

ROOT_FAMILIES = ("NORM", "GOV", "INTENT", "ARCH", "INFO", "CNTR", "RISK", "SUPPLY", "REAL", "ASSURE", "CTRL", "CHANGE", "OPS", "RECORD")
LEGACY_FAMILIES = ("SPEC", "PLAN", "WORK", "VERI", "MIGR", "RELS")
FAMILIES = ROOT_FAMILIES + LEGACY_FAMILIES
AUTHORITY_RELATIONS = ("OWN", "REFERENCE", "CONSUME", "VERIFY", "TRANSFORM", "EXECUTE")
_FAMILY_ID_RE = re.compile(r"^MPD-(NORM|GOV|INTENT|ARCH|INFO|CNTR|RISK|SUPPLY|REAL|ASSURE|CTRL|CHANGE|OPS|RECORD|SPEC|PLAN|WORK|VERI|MIGR|RELS)-[0-9]{4}$")

COLLISION_RESOLUTION = {
    "EXACT_DUPLICATE": {"REFERENCE_EXISTING"},
    "SEMANTIC_EQUIVALENT": {"REFERENCE_EXISTING"},
    "EXISTING_SUBSUMES_CANDIDATE": {"DROP_REDUNDANT_CANDIDATE"},
    "CANDIDATE_EXTENDS_EXISTING": {"MERGE_INTO_EXISTING", "CREATE_NEW_SIBLING_POLICY"},
    "PARTIAL_OVERLAP": {"SPLIT_RESPONSIBILITY_REQUIRED"},
    "DISTINCT_SCOPED_AUTHORITY": {"COEXIST_SCOPED"},
    "CONFLICT": {"FAIL_CLOSED_OWNER_DECISION_REQUIRED"},
}
BLOCKING_COLLISIONS = {"PARTIAL_OVERLAP", "CONFLICT"}
NEW_POLICY_RESOLUTIONS = {"CREATE_NEW_SIBLING_POLICY", "COEXIST_SCOPED"}


class ResponsibilityGateError(RuntimeError):
    def __init__(self, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code


def _mapping(value: object, *, label: str) -> Mapping[str, object]:
    if not isinstance(value, Mapping):
        raise ResponsibilityGateError(
            "INVALID_RESPONSIBILITY_ANALYSIS",
            f"{label} must be a mapping",
        )
    return value


def _load_analysis_records(base: Path) -> tuple[Mapping[str, object], ...]:
    registry = load_yaml(ANALYSIS_REGISTRY.as_posix(), root=base)
    schema = load_json(ANALYSIS_REGISTRY_SCHEMA, root=base)
    from developer.automation.policy_validator import developer_contract_validator

    errors = tuple(developer_contract_validator(schema, base).iter_errors(registry))
    if errors:
        raise ResponsibilityGateError(
            "INVALID_ANALYSIS_REGISTRY",
            "; ".join(error.message for error in errors),
        )
    records = registry.get("records")
    if not isinstance(records, list):
        raise ResponsibilityGateError(
            "INVALID_ANALYSIS_REGISTRY",
            "analysis registry records must be a list",
        )
    return tuple(_mapping(item, label="analysis registry record") for item in records)


def resolve_analysis_record(
    analysis_id: str | None = None,
    *,
    subject_type: str | None = None,
    subject_id: str | None = None,
    analysis_kind: str | None = None,
    root: str | Path | None = None,
) -> dict[str, object]:
    base = repository_root(root)
    records = _load_analysis_records(base)
    subject_key_supplied = any(
        value is not None for value in (subject_type, subject_id, analysis_kind)
    )

    if analysis_id is not None:
        if subject_key_supplied:
            raise ResponsibilityGateError(
                "ANALYSIS_LOOKUP_KEY_CONFLICT",
                "analysis_id and subject lookup keys are mutually exclusive",
            )
        if _ANALYSIS_ID_RE.fullmatch(analysis_id) is None:
            raise ResponsibilityGateError("INVALID_ANALYSIS_ID", analysis_id)
        matches = [item for item in records if item.get("analysis_id") == analysis_id]
    else:
        if not all(
            isinstance(value, str) and value
            for value in (subject_type, subject_id, analysis_kind)
        ):
            raise ResponsibilityGateError(
                "ANALYSIS_LOOKUP_KEY_REQUIRED",
                "use analysis_id or subject_type + subject_id + analysis_kind",
            )
        matches = [
            item for item in records
            if item.get("subject_type") == subject_type
            and item.get("subject_id") == subject_id
            and item.get("analysis_kind") == analysis_kind
        ]

    if not matches:
        raise ResponsibilityGateError(
            "RESPONSIBILITY_ANALYSIS_NOT_FOUND",
            "analysis registry has no exact matching record",
        )
    if len(matches) != 1:
        raise ResponsibilityGateError(
            "RESPONSIBILITY_ANALYSIS_AMBIGUOUS",
            "analysis registry lookup must resolve exactly one record",
        )

    record = dict(matches[0])
    resolved_id = record.get("analysis_id")
    analysis_ref = record.get("analysis_ref")
    if not isinstance(resolved_id, str) or not isinstance(analysis_ref, str):
        raise ResponsibilityGateError("INVALID_ANALYSIS_REGISTRY", "record identity/ref missing")
    expected = (ANALYSIS_RECORD_ROOT / f"{resolved_id}.yaml").as_posix()
    if analysis_ref != expected:
        raise ResponsibilityGateError(
            "NONCANONICAL_ANALYSIS_REF",
            f"{resolved_id}: expected {expected}, got {analysis_ref}",
        )
    if not (base / analysis_ref).is_file():
        raise ResponsibilityGateError(
            "RESPONSIBILITY_ANALYSIS_NOT_FOUND",
            f"registered analysis does not exist: {analysis_ref}",
        )
    return record

def _family_from_policy_id(policy_id: str) -> str | None:
    match = _FAMILY_ID_RE.fullmatch(policy_id)
    return None if match is None else match.group(1)


def _active_family_ids(base: Path, policy_class: str, family: str) -> tuple[str, ...]:
    from developer.automation.policy_validator import load_neutral_policy_index

    classes = load_json("developer/policy/registries/developer-policy-catalog-contracts.json", root=base)["$defs"]["developer_policy_class"]["enum"]
    if policy_class not in classes or family not in FAMILIES:
        raise ResponsibilityGateError("INVALID_AUTHORITY_FAMILY_KEY", "explicit registered policy_class and readable Family are required")
    try:
        index = load_neutral_policy_index(base)
    except (OSError, ValueError) as exc:
        raise ResponsibilityGateError("INVALID_POLICY_INDEX", str(exc)) from exc
    ids: list[str] = []
    for entry in index["policies"]:
        if _family_from_policy_id(entry["id"]) != family:
            continue
        target = load_yaml(entry["path"], root=base)
        policy = _mapping(target.get("policy"), label=entry["id"])
        if target.get("policy_class") != entry["policy_class"] or policy.get("status") != entry["status"] or policy.get("id") != entry["id"]:
            raise ResponsibilityGateError("AUTHORITY_METADATA_MISMATCH", f"{entry['id']}: class/status/identity projection mismatch")
        if entry["status"] == "ACTIVE" and entry["policy_class"] == policy_class:
            ids.append(entry["id"])
    return tuple(sorted(ids))


def validate_analysis_semantics(payload: Mapping[str, object], *, root: str | Path | None = None) -> tuple[str, ...]:
    classes = load_json("developer/policy/registries/developer-policy-catalog-contracts.json", root=repository_root(root))["$defs"]["developer_policy_class"]["enum"]
    errors: list[str] = []
    analysis = payload.get("analysis")
    if not isinstance(analysis, Mapping):
        return ("analysis must be a mapping",)

    responsibilities = analysis.get("responsibilities")
    decision = analysis.get("decision")
    if not isinstance(responsibilities, list) or not isinstance(decision, Mapping):
        return ("analysis responsibilities/decision must be present",)

    by_id: dict[str, Mapping[str, object]] = {}
    owned_keys: set[tuple[str, str]] = set()
    create_ids: set[str] = set()
    blocked_ids: set[str] = set()
    partial_overlap_present = False

    for raw in responsibilities:
        if not isinstance(raw, Mapping):
            errors.append("responsibility entry must be a mapping")
            continue

        responsibility_id = raw.get("responsibility_id")
        if not isinstance(responsibility_id, str):
            errors.append("responsibility_id must be a string")
            continue
        if responsibility_id in by_id:
            errors.append(f"{responsibility_id}: duplicate responsibility_id")
            continue
        by_id[responsibility_id] = raw

        relation = raw.get("authority_relation")
        family = raw.get("family")
        policy_class = raw.get("policy_class")
        referenced_family = raw.get("referenced_family")
        referenced_policy_class = raw.get("referenced_policy_class")
        lookup = raw.get("existing_authority_lookup")
        action = raw.get("materialization_action")
        target_group = raw.get("target_group_id")

        if relation == "OWN":
            if family not in FAMILIES or policy_class not in classes:
                errors.append(f"{responsibility_id}: OWN responsibility requires explicit policy_class and one Family")
                continue
            owned_keys.add((str(policy_class), str(family)))
            if referenced_family is not None or referenced_policy_class is not None:
                errors.append(f"{responsibility_id}: OWN responsibility must not use referenced authority identity")
            if not isinstance(lookup, Mapping):
                errors.append(f"{responsibility_id}: OWN responsibility requires existing_authority_lookup")
                continue
            if lookup.get("searched_family") != family:
                errors.append(
                    f"{responsibility_id}: lookup searched_family must equal owned Family"
                )
            if lookup.get("searched_policy_class") != policy_class:
                errors.append(f"{responsibility_id}: lookup searched_policy_class must equal owned policy_class")

            searched = lookup.get("searched_policy_ids")
            comparisons = lookup.get("candidate_comparisons")
            if not isinstance(searched, list) or len(searched) != len(set(searched)):
                errors.append(
                    f"{responsibility_id}: searched_policy_ids must be a unique list"
                )
                searched = []
            if not isinstance(comparisons, list):
                errors.append(
                    f"{responsibility_id}: candidate_comparisons must be a list"
                )
                comparisons = []

            comparison_ids: set[str] = set()
            has_blocker = False
            has_new_resolution = False
            for comparison in comparisons:
                if not isinstance(comparison, Mapping):
                    errors.append(f"{responsibility_id}: comparison must be a mapping")
                    continue
                policy_id = comparison.get("policy_id")
                collision = comparison.get("collision_class")
                resolution = comparison.get("resolution_action")
                scope_relation = comparison.get("scope_relation")

                if not isinstance(policy_id, str):
                    errors.append(
                        f"{responsibility_id}: comparison policy_id must be a string"
                    )
                    continue
                if policy_id in comparison_ids:
                    errors.append(
                        f"{responsibility_id}: duplicate comparison for {policy_id}"
                    )
                comparison_ids.add(policy_id)
                if policy_id not in searched:
                    errors.append(
                        f"{responsibility_id}: compared policy {policy_id} was not searched"
                    )

                allowed = COLLISION_RESOLUTION.get(str(collision), set())
                if resolution not in allowed:
                    errors.append(
                        f"{responsibility_id}: {collision} does not allow {resolution}"
                    )

                if collision == "DISTINCT_SCOPED_AUTHORITY":
                    if scope_relation != "DISTINCT_SCOPE":
                        errors.append(
                            f"{responsibility_id}: DISTINCT_SCOPED_AUTHORITY requires DISTINCT_SCOPE"
                        )
                elif collision in COLLISION_RESOLUTION and scope_relation != "SAME_SCOPE":
                    errors.append(
                        f"{responsibility_id}: {collision} requires SAME_SCOPE"
                    )

                if collision in BLOCKING_COLLISIONS:
                    has_blocker = True
                    blocked_ids.add(responsibility_id)
                if collision == "PARTIAL_OVERLAP":
                    partial_overlap_present = True
                if resolution in NEW_POLICY_RESOLUTIONS:
                    has_new_resolution = True

            expected_outcome = "MATCHES_FOUND" if comparisons else "NO_MATCH"
            if lookup.get("lookup_outcome") != expected_outcome:
                errors.append(
                    f"{responsibility_id}: lookup_outcome must be {expected_outcome}"
                )

            expected_action = (
                "BLOCKED"
                if has_blocker
                else "CREATE_NEW_POLICY"
                if not comparisons or has_new_resolution
                else "USE_EXISTING_AUTHORITY"
            )
            if action != expected_action:
                errors.append(
                    f"{responsibility_id}: materialization_action must be {expected_action}"
                )

            if expected_action == "CREATE_NEW_POLICY":
                create_ids.add(responsibility_id)
                if not isinstance(target_group, str):
                    errors.append(
                        f"{responsibility_id}: CREATE_NEW_POLICY requires target_group_id"
                    )
            elif target_group is not None:
                errors.append(
                    f"{responsibility_id}: only CREATE_NEW_POLICY may declare target_group_id"
                )
        else:
            if policy_class is not None:
                errors.append(f"{responsibility_id}: non-OWN responsibility must not own a policy_class")
            if (referenced_family is None) != (referenced_policy_class is None) or (
                referenced_family is not None and (referenced_family not in FAMILIES or referenced_policy_class not in classes)
            ):
                errors.append(f"{responsibility_id}: referenced authority requires an explicit registered policy_class and Family pair")
            if relation not in AUTHORITY_RELATIONS:
                errors.append(
                    f"{responsibility_id}: unknown authority_relation {relation!r}"
                )
            if family is not None:
                errors.append(
                    f"{responsibility_id}: non-OWN responsibility must not own a Family"
                )
            if lookup is not None:
                errors.append(
                    f"{responsibility_id}: non-OWN responsibility must not perform owner authority lookup"
                )
            if action != "USE_EXISTING_AUTHORITY":
                errors.append(
                    f"{responsibility_id}: non-OWN responsibility must USE_EXISTING_AUTHORITY"
                )
            if target_group is not None:
                errors.append(
                    f"{responsibility_id}: non-OWN responsibility must not target a new group"
                )

    expected_keys = [{"policy_class": policy_class, "family": family} for policy_class in classes for family in FAMILIES if (policy_class, family) in owned_keys]
    if decision.get("owned_authority_family_set") != expected_keys:
        errors.append(
            f"decision.owned_authority_family_set must equal canonical owned authority Family set {expected_keys}"
        )

    groups = decision.get("materialization_groups")
    if not isinstance(groups, list):
        errors.append("decision.materialization_groups must be a list")
        groups = []

    group_ids: set[str] = set()
    grouped_responsibilities: set[str] = set()
    for group in groups:
        if not isinstance(group, Mapping):
            errors.append("materialization group must be a mapping")
            continue
        group_id = group.get("group_id")
        family = group.get("family")
        policy_class = group.get("policy_class")
        if policy_class not in classes or family not in FAMILIES:
            errors.append(f"{group.get('group_id')}: materialization group requires explicit policy_class and Family")
        cohesion_key = group.get("cohesion_key")
        responsibility_ids = group.get("responsibility_ids")

        if not isinstance(group_id, str):
            errors.append("materialization group_id must be a string")
            continue
        if group_id in group_ids:
            errors.append(f"{group_id}: duplicate materialization group")
        group_ids.add(group_id)

        if not isinstance(responsibility_ids, list) or not responsibility_ids:
            errors.append(f"{group_id}: responsibility_ids must be non-empty")
            continue
        if len(responsibility_ids) != len(set(responsibility_ids)):
            errors.append(f"{group_id}: responsibility_ids must be unique")

        for responsibility_id in responsibility_ids:
            responsibility = by_id.get(str(responsibility_id))
            if responsibility is None:
                errors.append(f"{group_id}: unknown responsibility {responsibility_id}")
                continue
            if str(responsibility_id) in grouped_responsibilities:
                errors.append(
                    f"{responsibility_id}: responsibility appears in multiple groups"
                )
            grouped_responsibilities.add(str(responsibility_id))
            if responsibility.get("materialization_action") != "CREATE_NEW_POLICY":
                errors.append(
                    f"{group_id}: {responsibility_id} is not a new-policy responsibility"
                )
            if responsibility.get("target_group_id") != group_id:
                errors.append(
                    f"{group_id}: {responsibility_id} target_group_id mismatch"
                )
            if responsibility.get("family") != family:
                errors.append(f"{group_id}: {responsibility_id} Family mismatch")
            if responsibility.get("policy_class") != policy_class:
                errors.append(f"{group_id}: {responsibility_id} policy_class mismatch")
            if responsibility.get("cohesion_key") != cohesion_key:
                errors.append(
                    f"{group_id}: {responsibility_id} cohesion_key mismatch"
                )

    if grouped_responsibilities != create_ids:
        errors.append(
            "materialization_groups must cover every and only CREATE_NEW_POLICY responsibility"
        )

    expected_split = (
        len(expected_keys) > 1
        or len(groups) > 1
        or partial_overlap_present
    )
    if decision.get("split_required") is not expected_split:
        errors.append(f"decision.split_required must be {expected_split}")

    expected_allowed = not blocked_ids
    if decision.get("materialization_allowed") is not expected_allowed:
        errors.append(
            f"decision.materialization_allowed must be {expected_allowed}"
        )

    return tuple(errors)


def validate_responsibility_analysis(
    analysis_id: str,
    *,
    root: str | Path | None = None,
    enforce_current_lookup: bool = True,
) -> dict[str, object]:
    base = repository_root(root)
    record = resolve_analysis_record(analysis_id=analysis_id, root=base)
    relative = str(record["analysis_ref"])
    payload = load_yaml(relative, root=base)
    payload_analysis = payload.get("analysis")
    if not isinstance(payload_analysis, Mapping) or payload_analysis.get("analysis_id") != analysis_id:
        raise ResponsibilityGateError(
            "ANALYSIS_ID_MISMATCH",
            f"{relative}: payload analysis_id must equal registry identity {analysis_id}",
        )
    schema = load_json(ANALYSIS_SCHEMA, root=base)
    from developer.automation.policy_validator import developer_contract_validator
    schema_errors = tuple(developer_contract_validator(schema, base).iter_errors(payload))
    errors = [error.message for error in schema_errors]
    errors.extend(validate_analysis_semantics(payload, root=base))

    analysis = payload.get("analysis")
    if isinstance(analysis, Mapping) and enforce_current_lookup:
        # One validation observes one corpus; repeated units share this exact lookup.
        # The cache is invocation-local so a later registration always re-reads state.
        current_family_ids: dict[tuple[str, str], tuple[str, ...]] = {}
        responsibilities = analysis.get("responsibilities", [])
        if isinstance(responsibilities, list):
            for raw in responsibilities:
                if not isinstance(raw, Mapping) or raw.get("authority_relation") != "OWN":
                    continue
                responsibility_id = str(raw.get("responsibility_id"))
                family = raw.get("family")
                policy_class = raw.get("policy_class")
                lookup = raw.get("existing_authority_lookup")
                if family not in FAMILIES or not isinstance(policy_class, str) or not isinstance(lookup, Mapping):
                    continue

                key = (policy_class, str(family))
                if key not in current_family_ids:
                    current_family_ids[key] = _active_family_ids(base, *key)
                expected = list(current_family_ids[key])
                searched = lookup.get("searched_policy_ids")
                if searched != expected:
                    errors.append(
                        f"{responsibility_id}: searched_policy_ids must exactly cover current ACTIVE "
                        f"{policy_class} + {family} policies {expected}"
                    )

                comparisons = lookup.get("candidate_comparisons", [])
                if not isinstance(comparisons, list):
                    continue
                for comparison in comparisons:
                    if not isinstance(comparison, Mapping):
                        continue
                    policy_id = comparison.get("policy_id")
                    section = comparison.get("section")
                    if not isinstance(policy_id, str) or policy_id not in expected:
                        errors.append(
                            f"{responsibility_id}: comparison target {policy_id!r} is not current ACTIVE {family} authority"
                        )
                        continue
                    if isinstance(section, str):
                        index = load_yaml(INDEX, root=base)
                        entry = next(
                            (
                                item
                                for item in index.get("policies", [])
                                if isinstance(item, Mapping)
                                and item.get("id") == policy_id
                            ),
                            None,
                        )
                        if not isinstance(entry, Mapping):
                            errors.append(
                                f"{responsibility_id}: unresolved comparison policy {policy_id}"
                            )
                            continue
                        target = load_yaml(str(entry["path"]), root=base)
                        rules = target.get("rules")
                        if not isinstance(rules, Mapping) or section not in rules:
                            errors.append(
                                f"{responsibility_id}: {policy_id} has no compared rules section {section!r}"
                            )

    if errors:
        raise ResponsibilityGateError(
            "RESPONSIBILITY_ANALYSIS_BLOCKED",
            "; ".join(errors),
        )

    analysis_map = _mapping(payload.get("analysis"), label="analysis")
    decision = _mapping(analysis_map.get("decision"), label="analysis.decision")
    return {
        "status": "PASS",
        "analysis_id": analysis_map.get("analysis_id"),
        "analysis_ref": relative,
        "owned_authority_family_set": decision.get("owned_authority_family_set"),
        "split_required": decision.get("split_required"),
        "materialization_allowed": decision.get("materialization_allowed"),
        "materialization_groups": decision.get("materialization_groups"),
    }


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Validate Policy Analysis through the canonical registry."
    )
    parser.add_argument("--root")
    parser.add_argument("--analysis-id")
    parser.add_argument("--subject-type")
    parser.add_argument("--subject-id")
    parser.add_argument("--analysis-kind")
    parser.add_argument("--current-authority-lookup", action="store_true")
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        if args.analysis_id is not None:
            record = resolve_analysis_record(analysis_id=args.analysis_id, root=args.root)
        else:
            record = resolve_analysis_record(
                subject_type=args.subject_type,
                subject_id=args.subject_id,
                analysis_kind=args.analysis_kind,
                root=args.root,
            )
        result = validate_responsibility_analysis(
            str(record["analysis_id"]),
            root=args.root,
            enforce_current_lookup=args.current_authority_lookup,
        )
    except ResponsibilityGateError as exc:
        print(json.dumps({"status": "BLOCKED", "code": exc.code, "message": str(exc)}, sort_keys=True))
        return 2
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
