from __future__ import annotations

from pathlib import Path

from jsonschema import Draft202012Validator

from developer.automation.policy_loader import load_json, load_yaml, repository_root


INDEX = "developer/policy/index.yaml"
INDEX_SCHEMA = "developer/policy/schemas/developer-policy-index.schema.json"
MPD_SCHEMA = "developer/policy/schemas/management-policy.schema.json"
SFP_CANONICAL_SCHEMA = "schemas/ptsip-support-feature-policy.schema.json"
SFP_EMBEDDED_SCHEMA = "src/ptsip/specdata/ptsip-support-feature-policy.schema.json"


def validate_developer_policy(root: str | Path | None = None) -> tuple[str, ...]:
    base = repository_root(root)
    errors: list[str] = []
    index = load_yaml(INDEX, root=base)
    index_schema = load_json(INDEX_SCHEMA, root=base)
    mpd_schema = load_json(MPD_SCHEMA, root=base)
    Draft202012Validator.check_schema(index_schema)
    Draft202012Validator.check_schema(mpd_schema)
    for error in Draft202012Validator(index_schema).iter_errors(index):
        errors.append(f"developer/policy/index.yaml: {error.message}")
    for entry in index.get("policies", []):
        path = entry.get("path")
        if not isinstance(path, str):
            continue
        payload = load_yaml(path, root=base)
        for error in Draft202012Validator(mpd_schema).iter_errors(payload):
            errors.append(f"{path}: {error.message}")
        if payload.get("policy", {}).get("id") != entry.get("id"):
            errors.append(f"{path}: policy.id does not match index id")
    canonical = load_json(SFP_CANONICAL_SCHEMA, root=base)
    embedded = load_json(SFP_EMBEDDED_SCHEMA, root=base)
    Draft202012Validator.check_schema(canonical)
    Draft202012Validator.check_schema(embedded)
    if canonical != embedded:
        errors.append("Support Feature Policy canonical and embedded schemas differ")
    return tuple(errors)


if __name__ == "__main__":
    failures = validate_developer_policy()
    if failures:
        raise SystemExit("\n".join(failures))
    print("Developer policy validation: PASS")
