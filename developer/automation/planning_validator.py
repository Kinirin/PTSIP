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
