"""Bounded, developer-only catalog and PRA identity backfill (no policy creation)."""
from __future__ import annotations

import argparse
import copy
import hashlib
import json
from pathlib import Path

from developer.automation.policy_loader import load_json, load_yaml, repository_root
from developer.automation.policy_identity_lifecycle import _atomic_write_yaml

CONTRACT = "developer/policy/registries/developer-policy-catalog-contracts.json"
ANALYSIS_REGISTRY = "developer/policy/analysis/registry.yaml"


def legacy_analysis_projection(payload: dict[str, object]) -> dict[str, object]:
    """Remove only explicitly backfilled identity fields, retaining all prior meaning."""
    result = copy.deepcopy(payload)
    result["schema_version"] = "ptsip-policy-responsibility-analysis/v1"
    analysis = result["analysis"]
    for responsibility in analysis["responsibilities"]:
        responsibility.pop("policy_class", None)
        responsibility.pop("referenced_policy_class", None)
        lookup = responsibility.get("existing_authority_lookup")
        if lookup is not None:
            lookup.pop("searched_policy_class", None)
    decision = analysis["decision"]
    keys = decision.pop("owned_authority_family_set")
    decision["owned_family_set"] = [key["family"] for key in keys]
    for group in decision["materialization_groups"]:
        group.pop("policy_class", None)
    return result


def semantic_digest(payload: object) -> str:
    return hashlib.sha256(json.dumps(payload, sort_keys=True, ensure_ascii=False, separators=(",", ":")).encode("utf-8")).hexdigest()


def backfill_legacy_analysis(original: dict[str, object]) -> dict[str, object]:
    """Apply the explicitly approved legacy PTSIP identity backfill, without inference."""
    if original["schema_version"] != "ptsip-policy-responsibility-analysis/v1":
        raise ValueError("BACKFILL_REQUIRES_EXACT_LEGACY_INPUT")
    updated = copy.deepcopy(original)
    updated["schema_version"] = "developer-policy-responsibility-analysis/v2"
    analysis = updated["analysis"]
    for responsibility in analysis["responsibilities"]:
        responsibility["policy_class"] = "PTSIP_DEVELOPER_POLICY" if responsibility["authority_relation"] == "OWN" else None
        responsibility["referenced_policy_class"] = "PTSIP_DEVELOPER_POLICY" if responsibility["referenced_family"] is not None else None
        lookup = responsibility.get("existing_authority_lookup")
        if lookup is not None:
            lookup["searched_policy_class"] = "PTSIP_DEVELOPER_POLICY"
    decision = analysis["decision"]
    decision["owned_authority_family_set"] = [{"policy_class": "PTSIP_DEVELOPER_POLICY", "family": family} for family in decision.pop("owned_family_set")]
    for group in decision["materialization_groups"]:
        group["policy_class"] = "PTSIP_DEVELOPER_POLICY"
    if legacy_analysis_projection(updated) != original:
        raise ValueError("BACKFILL_CHANGED_PRIOR_MEANING")
    return updated


def migrate(*, root: str | Path | None = None, apply: bool = False) -> dict[str, object]:
    base = repository_root(root)
    contract = load_json(CONTRACT, root=base)
    if not contract["application_gate"]["catalog_payload_migration_authorized"] or not contract["application_gate"]["m2_m8_implementation_authorized"]:
        raise ValueError("MIGRATION_NOT_AUTHORIZED")
    execution = contract["application_execution"]
    allowed = set(execution["targets"])
    source_index = load_yaml("developer/policy/index.yaml", root=base)
    source_subject = load_yaml("developer/policy/registries/authority-subject-registry.yaml", root=base)
    if source_index["schema_version"] != "ptsip-developer-policy-index/v1":
        raise ValueError("MIGRATION_REQUIRES_EXACT_LEGACY_CATALOG_INPUT")
    records = {entry["id"]: load_yaml(entry["path"], root=base) for entry in source_index["policies"]}
    catalog = copy.deepcopy(source_index)
    catalog.pop("policy_class")
    catalog["schema_version"] = contract["entrypoints"]["index"]
    catalog["artifact_class"] = contract["contracts"][catalog["schema_version"]]["artifact_class"]
    for entry in catalog["policies"]:
        entry["policy_class"] = records[entry["id"]]["policy_class"]
    subject = copy.deepcopy(source_subject)
    subject.pop("policy_class")
    subject["schema_version"] = contract["entrypoints"]["subject"]
    subject["artifact_class"] = contract["contracts"][subject["schema_version"]]["artifact_class"]
    from developer.automation.policy_validator import validate_neutral_catalog_snapshot, developer_contract_validator
    errors = validate_neutral_catalog_snapshot(catalog, subject, records, source_index=source_index, source_subject=source_subject, root=base)
    if errors:
        raise ValueError("; ".join(errors))
    registry = load_yaml(ANALYSIS_REGISTRY, root=base)
    migrated_registry = copy.deepcopy(registry)
    migrated_registry["schema_version"] = "developer-policy-materialization-analysis-registry/v2"
    updates = {"developer/policy/index.yaml": catalog, "developer/policy/registries/authority-subject-registry.yaml": subject}
    digests: dict[str, str] = {}
    for ref in sorted({binding["analysis_ref"] for binding in registry["bindings"]}):
        owner_classes = {records[binding["policy_id"]]["policy_class"] for binding in registry["bindings"] if binding["analysis_ref"] == ref}
        if owner_classes != {"PTSIP_DEVELOPER_POLICY"}:
            raise ValueError(f"{ref}: implicit legacy owner is not exactly the indexed PTSIP class")
        original = load_yaml(ref, root=base)
        updated = backfill_legacy_analysis(original)
        if legacy_analysis_projection(updated) != original:
            raise ValueError(f"{ref}: backfill changed prior normative analysis meaning")
        schema = load_json("developer/policy/schemas/policy-responsibility-analysis.schema.json", root=base)
        developer_contract_validator(schema, base).validate(updated)
        updates[ref] = updated
        digests[ref] = semantic_digest(original)
    for binding in migrated_registry["bindings"]:
        record = records[binding["policy_id"]]
        group = next(group for group in updates[binding["analysis_ref"]]["analysis"]["decision"]["materialization_groups"] if group["group_id"] == binding["group_id"])
        binding["policy_class"] = record["policy_class"]
        binding["family"] = group["family"]
        if binding["policy_class"] != group["policy_class"]:
            raise ValueError("legacy policy/group class mismatch")
    updates[ANALYSIS_REGISTRY] = migrated_registry
    developer_contract_validator(load_json("developer/policy/schemas/policy-materialization-analysis-registry.schema.json", root=base), base).validate(migrated_registry)
    if not set(updates).issubset(allowed):
        raise ValueError("UNREGISTERED_MIGRATION_WRITE_TARGET")
    if apply:
        originals = {path: load_yaml(path, root=base) for path in updates}
        try:
            for path, payload in updates.items():
                _atomic_write_yaml(base / path, payload)
        except Exception:
            for path, payload in originals.items():
                _atomic_write_yaml(base / path, payload)
            raise
    return {"status": "APPLIED" if apply else "READY", "paths": sorted(updates), "preserved_analysis_semantic_digests": digests}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root")
    parser.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    print(json.dumps(migrate(root=args.root, apply=args.apply), sort_keys=True))
