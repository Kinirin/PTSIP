from __future__ import annotations

from pathlib import Path

from jsonschema import Draft202012Validator

from developer.automation.planning_extension_finalizer import (
    extension_is_machine_ready,
    extension_parent_consistency_errors,
)
from developer.automation.planning_merge_reconciler import (
    PlanningStateReconciliationError,
    build_materialized_state,
)
from developer.automation.policy_loader import load_json, load_yaml, repository_root


ROOT_INDEX = "docs/planning/index.yaml"
ROOT_SCHEMA = "developer/policy/schemas/planning-root-index.schema.json"
PLAN_SCHEMA = "developer/policy/schemas/planning-index.schema.json"
WU_SCHEMA = "developer/policy/schemas/work-unit.schema.json"
EXTENSION_SCHEMA = "developer/policy/schemas/plan-extension.schema.json"
_TERMINAL_EXTENSION_STATUSES = {"COMPLETE", "SUPERSEDED", "CANCELLED"}


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

        if isinstance(root_routing, dict) and isinstance(plan_routing, dict):
            resolver = root_routing.get("resolver", {})
            entry_resolution = plan.get("entry_resolution", {})
            if resolver.get("module") != entry_resolution.get("resolver_module"):
                errors.append(f"{path}: resolver module does not match root entry routing")
            if resolver.get("match_mode") != entry_resolution.get("match_mode"):
                errors.append(f"{path}: resolver match mode does not match root entry routing")
            if resolver.get("unmatched_behavior") != entry_resolution.get("unknown_branch_behavior"):
                errors.append(f"{path}: resolver unknown-branch behavior does not match root entry routing")

            merge_reconciliation = root_routing.get("merge_reconciliation", {})
            if merge_reconciliation.get("target_branch") != plan_entry.get("integration_branch"):
                errors.append(f"{path}: merge reconciliation target must equal integration_branch")
            if merge_reconciliation.get("leaf_shared_index_mutation") != "FORBIDDEN":
                errors.append(f"{path}: leaf shared planning-index mutation must be FORBIDDEN")
            if merge_reconciliation.get("state_source") != "WORK_UNIT_DOCUMENTS":
                errors.append(f"{path}: merge reconciliation state source must be WORK_UNIT_DOCUMENTS")

            entrypoints = root_routing.get("branch_entrypoints", [])
            branches = [
                entry.get("branch")
                for entry in entrypoints
                if isinstance(entry, dict)
            ]
            if len(branches) != len(set(branches)):
                errors.append(f"{ROOT_INDEX}: branch entrypoints must use unique exact branch names")

            expected_branches = {plan_entry.get("integration_branch")}
            expected_branches.update(
                entry.get("branch")
                for entry in root_routing.get("independent_leaf_work_units", [])
                if isinstance(entry, dict)
            )
            if set(branches) != expected_branches:
                errors.append(
                    f"{path}: branch entrypoint set does not match declared integration/leaf branches"
                )

            integration_entries = [
                entry
                for entry in entrypoints
                if isinstance(entry, dict)
                and entry.get("branch") == plan_entry.get("integration_branch")
            ]
            if len(integration_entries) != 1:
                errors.append(f"{path}: integration branch must have exactly one planning entrypoint")
            else:
                integration_entry = integration_entries[0]
                if integration_entry.get("entry_document") != path:
                    errors.append(f"{path}: integration branch entrypoint must resolve to version index")
                if integration_entry.get("role") != "INTEGRATION_CONTROL_PLANE":
                    errors.append(f"{path}: integration branch entrypoint role is invalid")
                if integration_entry.get("state") != "ACTIVE":
                    errors.append(f"{path}: integration branch entrypoint must remain ACTIVE")
                if integration_entry.get("work_unit") is not None:
                    errors.append(f"{path}: integration branch entrypoint must not bind a work unit")

            indexed_wus = {
                wu.get("id"): wu
                for wu in plan.get("work_units", [])
                if isinstance(wu, dict) and isinstance(wu.get("id"), str)
            }
            leaf_routes = {
                entry.get("id"): entry
                for entry in root_routing.get("independent_leaf_work_units", [])
                if isinstance(entry, dict) and isinstance(entry.get("id"), str)
            }
            for leaf_id, leaf_route in leaf_routes.items():
                leaf_entries = [
                    entry
                    for entry in entrypoints
                    if isinstance(entry, dict)
                    and entry.get("branch") == leaf_route.get("branch")
                ]
                if len(leaf_entries) != 1:
                    errors.append(f"{path}: leaf {leaf_id!r} must have exactly one planning entrypoint")
                    continue
                leaf_entry = leaf_entries[0]
                indexed = indexed_wus.get(leaf_id)
                expected_document = indexed.get("path") if isinstance(indexed, dict) else None
                if leaf_entry.get("work_unit") != leaf_id:
                    errors.append(f"{path}: leaf entrypoint work_unit does not match routed leaf id")
                if leaf_entry.get("entry_document") != expected_document:
                    errors.append(f"{path}: leaf {leaf_id!r} entrypoint does not match indexed WU path")
                if leaf_entry.get("role") != "INDEPENDENT_LEAF":
                    errors.append(f"{path}: leaf {leaf_id!r} entrypoint role is invalid")
                if leaf_entry.get("state") == "MERGED":
                    if leaf_entry.get("merged_into") != plan_entry.get("integration_branch"):
                        errors.append(f"{path}: merged leaf {leaf_id!r} must point to integration_branch")
                elif leaf_entry.get("state") != "ACTIVE":
                    errors.append(f"{path}: leaf {leaf_id!r} entrypoint state is invalid")
                entry_document = leaf_entry.get("entry_document")
                if isinstance(entry_document, str) and not (base / entry_document).is_file():
                    errors.append(f"{path}: leaf {leaf_id!r} entry document does not exist")

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

                if work_unit.get("lifecycle", {}).get("status") == "COMPLETE":
                    completion_evidence = payload.get("completion_evidence")
                    if not isinstance(completion_evidence, list) or not completion_evidence:
                        errors.append(f"{wu_path}: COMPLETE work unit requires completion_evidence")
                    elif completion_evidence != wu.get("completion_evidence"):
                        errors.append(f"{wu_path}: completion_evidence does not match version index")

                for extension in payload.get("extensions", []):
                    ext_path = extension.get("path")
                    ext_id = extension.get("id")
                    if not isinstance(ext_path, str) or not isinstance(ext_id, str):
                        continue
                    ext_payload = load_yaml(ext_path, root=base)
                    errors.extend(_errors(ext_payload, extension_schema, ext_path))
                    errors.extend(
                        extension_parent_consistency_errors(
                            payload,
                            ext_payload,
                            extension_id=ext_id,
                            extension_path=ext_path,
                        )
                    )
                    ext_status = (
                        ext_payload.get("extension", {})
                        .get("lifecycle", {})
                        .get("status")
                    )
                    if extension_is_machine_ready(ext_payload) and ext_status != "COMPLETE":
                        errors.append(
                            f"{ext_path}: machine-ready Plan Extension remains {ext_status!r}; "
                            "run planning_stage_finalizer to close and reconcile it"
                        )
                    if ext_payload.get("extension", {}).get("id") == gate:
                        gate_resolved = True
                        if ext_status in _TERMINAL_EXTENSION_STATUSES:
                            errors.append(
                                f"{path}: current_gate {gate!r} points to terminal Plan Extension; "
                                "planning reconciliation is required"
                            )
        if not gate_resolved:
            errors.append(f"{path}: current_gate {gate!r} does not resolve to a current WU or Plan Extension")

        if isinstance(root_routing, dict) and root_routing.get("merge_reconciliation"):
            try:
                expected_materialized_state = build_materialized_state(
                    plan_entry,
                    plan,
                    base=base,
                )
            except PlanningStateReconciliationError as exc:
                errors.append(f"{path}: materialized planning state cannot be derived: {exc}")
            else:
                if plan_entry.get("materialized_state") != expected_materialized_state:
                    errors.append(
                        f"{ROOT_INDEX}: materialized_state is stale or does not match canonical planning state"
                    )

    return tuple(errors)


if __name__ == "__main__":
    failures = validate_planning()
    if failures:
        raise SystemExit("\n".join(failures))
    print("Developer planning validation: PASS")
