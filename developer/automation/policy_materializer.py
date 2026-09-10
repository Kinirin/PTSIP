from __future__ import annotations

from collections import defaultdict
from pathlib import Path
from typing import Any

from developer.automation.policy_loader import load_yaml, repository_root


INVENTORY = "developer/policy/legacy-decisions-inventory.yaml"
RELATION_MIGRATION = "developer/policy/policy-relation-migration.yaml"


def _policy_status(source_status: str, target_class: str) -> str:
    if target_class == "SFP" and source_status == "PROPOSED_FOR_NEXT_NORMATIVE_SNAPSHOT":
        return "DRAFT"
    return "ACTIVE"


def _project_semantics(source: dict[str, Any], output: dict[str, Any]) -> dict[str, Any]:
    semantics = source["authority_semantics"]
    projection = output["semantic_scope"]
    mode = projection["mode"]
    if mode == "FULL_AUTHORITY_SEMANTICS":
        return dict(semantics)
    fields = projection["include_fields"]
    missing = [field for field in fields if field not in semantics]
    if missing:
        raise ValueError(
            f"{source['decision']['id']} -> {output['target_id']}: "
            f"semantic fields missing from source: {missing}"
        )
    return {field: semantics[field] for field in fields}


def _projected_relations(root: Path) -> dict[str, dict[str, list[dict[str, Any]]]]:
    manifest = load_yaml(RELATION_MIGRATION, root=root)
    result: dict[str, dict[str, list[dict[str, Any]]]] = defaultdict(
        lambda: {kind: [] for kind in ("supersedes", "amends", "extends", "depends_on")}
    )
    for item in manifest["relations"]:
        for edge in item["projected_edges"]:
            result[edge["source_policy"]][edge["relation"]].append(
                {"policy": edge["target_policy"], "scope": edge["scope"]}
            )
    return result


def _empty_relations() -> dict[str, list[dict[str, Any]]]:
    return {kind: [] for kind in ("supersedes", "amends", "extends", "depends_on")}


def expected_materialized_policies(
    root: str | Path | None = None,
) -> dict[str, dict[str, Any]]:
    base = repository_root(root)
    inventory = load_yaml(INVENTORY, root=base)
    relation_map = _projected_relations(base)
    result: dict[str, dict[str, Any]] = {}

    for entry in inventory["entries"]:
        source = load_yaml(entry["source_path"], root=base)
        decision = source["decision"]
        for output in entry["outputs"]:
            if output["disposition"] != "MATERIALIZE":
                continue
            target_id = output["target_id"]
            target_class = output["target_class"]
            relations = relation_map.get(target_id, _empty_relations())
            semantics = _project_semantics(source, output)

            if target_class == "SFP":
                payload = {
                    "schema_version": "ptsip-support-feature-policy/v1",
                    "policy_class": "PTSIP_SUPPORT_FEATURE",
                    "policy": {
                        "id": target_id,
                        "title": output["title"],
                        "status": _policy_status(decision["status"], target_class),
                    },
                    "feature_contract": {
                        "shipped": True,
                        "runtime_surface": (
                            ["src/vpms/**"] if target_id == "SFP-0006" else ["src/ptsip/**"]
                        ),
                    },
                    "authority_contract": source["authority_contract"],
                    "authority_semantics": semantics,
                    "relations": relations,
                }
                result[f"src/ptsip/specdata/{target_id}.yaml"] = payload
            elif target_class == "MPD":
                payload = {
                    "schema_version": "ptsip-developer-policy/v1",
                    "policy_class": "PTSIP_DEVELOPER_POLICY",
                    "policy": {
                        "id": target_id,
                        "title": output["title"],
                        "status": _policy_status(decision["status"], target_class),
                    },
                    "rules": {"authority_semantics": semantics},
                    "relations": relations,
                }
                result[f"developer/policy/{target_id}.yaml"] = payload
            else:
                raise ValueError(f"Unsupported materialization target class: {target_class}")

    return result


def expected_support_policy_index(
    root: str | Path | None = None,
) -> dict[str, Any]:
    policies = expected_materialized_policies(root)
    entries = []
    for path, payload in sorted(policies.items()):
        if not path.startswith("src/ptsip/specdata/SFP-"):
            continue
        entries.append(
            {
                "id": payload["policy"]["id"],
                "path": path,
                "status": payload["policy"]["status"],
            }
        )
    return {
        "schema_version": "ptsip-support-feature-policy-index/v1",
        "policy_class": "PTSIP_SUPPORT_FEATURE",
        "policies": entries,
    }


def expected_mpd_index_entries(
    root: str | Path | None = None,
) -> list[dict[str, str]]:
    policies = expected_materialized_policies(root)
    entries = []
    for path, payload in sorted(policies.items()):
        if not path.startswith("developer/policy/MPD-"):
            continue
        entries.append(
            {
                "id": payload["policy"]["id"],
                "path": path,
                "status": payload["policy"]["status"],
            }
        )
    return entries
