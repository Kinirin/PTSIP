from __future__ import annotations

from pathlib import Path

from jsonschema import Draft202012Validator

from developer.automation.policy_loader import load_json, load_yaml, repository_root


ROOT_INDEX = "docs/planning/index.yaml"
ROOT_SCHEMA = "developer/policy/schemas/planning-root-index.schema.json"
PLAN_SCHEMA = "developer/policy/schemas/planning-index.schema.json"
WU_SCHEMA = "developer/policy/schemas/work-unit.schema.json"
EXTENSION_SCHEMA = "developer/policy/schemas/plan-extension.schema.json"


def _errors(payload: dict[str, object], schema: dict[str, object], label: str) -> list[str]:
    return [f"{label}: {error.message}" for error in Draft202012Validator(schema).iter_errors(payload)]


def validate_planning(root: str | Path | None = None) -> tuple[str, ...]:
    base = repository_root(root)
    errors: list[str] = []
    root_index = load_yaml(ROOT_INDEX, root=base)
    root_schema = load_json(ROOT_SCHEMA, root=base)
    plan_schema = load_json(PLAN_SCHEMA, root=base)
    wu_schema = load_json(WU_SCHEMA, root=base)
    extension_schema = load_json(EXTENSION_SCHEMA, root=base)
    for schema in (root_schema, plan_schema, wu_schema, extension_schema):
        Draft202012Validator.check_schema(schema)
    errors.extend(_errors(root_index, root_schema, ROOT_INDEX))
    for plan_entry in root_index.get("plans", []):
        path = plan_entry.get("path")
        if not isinstance(path, str):
            continue
        plan = load_yaml(path, root=base)
        errors.extend(_errors(plan, plan_schema, path))

        root_routing = plan_entry.get("entry_routing")
        plan_routing = plan.get("responsibility_routing")
        if isinstance(root_routing, dict):
            if not isinstance(plan_routing, dict):
                errors.append(f"{path}: root entry_routing exists but responsibility_routing is missing")
            else:
                if root_routing.get("model") != plan_routing.get("model"):
                    errors.append(f"{path}: responsibility routing model does not match root entry routing")
                if root_routing.get("canonical_plan") != plan_routing.get("canonical_plan"):
                    errors.append(f"{path}: canonical planning document does not match root entry routing")
                if root_routing.get("responsibility_control_plane") != path:
                    errors.append(f"{ROOT_INDEX}: responsibility_control_plane does not match plan path")
                if root_routing.get("branch_creation_parent") != plan_routing.get("branch_creation_parent"):
                    errors.append(f"{path}: branch creation parent does not match root entry routing")
                if plan_routing.get("branch_creation_parent") != plan.get("plan", {}).get("integration_branch"):
                    errors.append(f"{path}: branch creation parent does not match integration_branch")

                root_convergence = root_routing.get("dependency_bearing_convergence", {})
                plan_convergence = plan_routing.get("dependency_bearing_convergence", {})
                if (
                    root_convergence.get("id"),
                    root_convergence.get("responsibility"),
                ) != (
                    plan_convergence.get("id"),
                    plan_convergence.get("responsibility"),
                ):
                    errors.append(f"{path}: dependency-bearing convergence routing does not match root entry routing")

                root_leafs = {
                    (entry.get("id"), entry.get("branch"), entry.get("responsibility"))
                    for entry in root_routing.get("independent_leaf_work_units", [])
                    if isinstance(entry, dict)
                }
                plan_leafs = {
                    (entry.get("id"), entry.get("branch"), entry.get("responsibility"))
                    for entry in plan_routing.get("independent_leaf_work_units", [])
                    if isinstance(entry, dict)
                }
                if root_leafs != plan_leafs:
                    errors.append(f"{path}: independent leaf routing does not match root entry routing")

                indexed_wus = {
                    wu.get("id"): wu
                    for wu in plan.get("work_units", [])
                    if isinstance(wu, dict) and isinstance(wu.get("id"), str)
                }
                convergence_id = plan_convergence.get("id")
                convergence_wu = indexed_wus.get(convergence_id)
                if convergence_wu is None:
                    errors.append(f"{path}: dependency-bearing convergence WU is not indexed")
                elif convergence_wu.get("depends_on", []) != plan_convergence.get("expected_depends_on", []):
                    errors.append(f"{path}: convergence WU depends_on does not match responsibility routing")

                leaf_ids = {
                    entry.get("id")
                    for entry in plan_routing.get("independent_leaf_work_units", [])
                    if isinstance(entry, dict)
                }
                for leaf in plan_routing.get("independent_leaf_work_units", []):
                    if not isinstance(leaf, dict):
                        continue
                    leaf_id = leaf.get("id")
                    indexed = indexed_wus.get(leaf_id)
                    if indexed is None:
                        errors.append(f"{path}: independent leaf {leaf_id!r} is not indexed")
                        continue
                    declared_dependencies = indexed.get("depends_on", [])
                    if declared_dependencies:
                        errors.append(f"{path}: independent leaf {leaf_id!r} must have empty depends_on")
                    forbidden = set(leaf.get("forbidden_dependencies", []))
                    if forbidden.intersection(declared_dependencies):
                        errors.append(f"{path}: independent leaf {leaf_id!r} has a forbidden dependency")
                    if any(other != leaf_id and other in declared_dependencies for other in leaf_ids):
                        errors.append(f"{path}: independent leaf {leaf_id!r} depends on another independent leaf")

        gate = plan.get("plan", {}).get("current_gate")
        gate_resolved = False
        for wu in plan.get("work_units", []):
            if wu.get("id") == gate:
                gate_resolved = True
            wu_path = wu.get("path")
            if isinstance(wu_path, str):
                payload = load_yaml(wu_path, root=base)
                errors.extend(_errors(payload, wu_schema, wu_path))
                work_unit = payload.get("work_unit", {})
                if work_unit.get("id") != wu.get("id"):
                    errors.append(f"{wu_path}: work_unit.id does not match version index")
                if work_unit.get("lifecycle", {}).get("status") != wu.get("lifecycle", {}).get("status"):
                    errors.append(f"{wu_path}: lifecycle.status does not match version index")
                if work_unit.get("approval", {}).get("status") != wu.get("approval", {}).get("status"):
                    errors.append(f"{wu_path}: approval.status does not match version index")
                if work_unit.get("implementation_authorization") != wu.get("implementation_authorization", {}).get("status"):
                    errors.append(f"{wu_path}: implementation_authorization does not match version index")
                if work_unit.get("depends_on", []) != wu.get("depends_on", []):
                    errors.append(f"{wu_path}: depends_on does not match version index")
                for extension in payload.get("extensions", []):
                    ext_path = extension.get("path")
                    if not isinstance(ext_path, str):
                        continue
                    ext_payload = load_yaml(ext_path, root=base)
                    errors.extend(_errors(ext_payload, extension_schema, ext_path))
                    if ext_payload.get("extension", {}).get("id") == gate:
                        gate_resolved = True
        if not gate_resolved:
            errors.append(f"{path}: current_gate {gate!r} does not resolve to a current WU or Plan Extension")
    return tuple(errors)


if __name__ == "__main__":
    failures = validate_planning()
    if failures:
        raise SystemExit("\n".join(failures))
    print("Developer planning validation: PASS")
