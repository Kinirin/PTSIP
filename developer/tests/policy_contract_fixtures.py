"""Explicit current-contract fixtures; no compatibility fallback in production."""
from __future__ import annotations

import copy
import json
from pathlib import Path
import shutil

import yaml

from developer.automation.policy_validator import developer_contract_validator

SOURCE_ROOT = Path(__file__).resolve().parents[2]
RESOURCE_PATHS = (
    "developer/policy/registries/developer-policy-catalog-contracts.json",
    "developer/policy/schemas/developer-policy-catalog-contracts.schema.json",
    "developer/policy/schemas/developer-policy-catalog.schema.json",
    "developer/policy/schemas/developer-policy-subject-catalog.schema.json",
)


def copy_contract_resources(root: Path) -> None:
    for relative in RESOURCE_PATHS:
        target = root / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(SOURCE_ROOT / relative, target)


def neutralize_fixture_catalog(root: Path) -> None:
    copy_contract_resources(root)
    path = root / "developer/policy/index.yaml"
    index = yaml.safe_load(path.read_text(encoding="utf-8"))
    index.pop("policy_class", None)
    index["schema_version"] = "developer-policy-catalog/v1"
    index["artifact_class"] = "DEVELOPER_POLICY_CATALOG"
    for entry in index["policies"]:
        entry["policy_class"] = yaml.safe_load((root / entry["path"]).read_text(encoding="utf-8"))["policy_class"]
    path.write_text(yaml.safe_dump(index, sort_keys=False), encoding="utf-8")
    path = root / "developer/policy/registries/authority-subject-registry.yaml"
    subject = copy.deepcopy(yaml.safe_load((SOURCE_ROOT / path.relative_to(root)).read_text(encoding="utf-8")))
    subject["subject_identity_schemes"]["MANAGEMENT_POLICY_ID"]["registered_values"] = [entry["id"] for entry in index["policies"]]
    path.write_text(yaml.safe_dump(subject, sort_keys=False), encoding="utf-8")


def write_catalog_fixture(root: Path, policy_ids: list[str]) -> None:
    copy_contract_resources(root)
    payload = {
        "schema_version": "developer-policy-catalog/v1",
        "artifact_class": "DEVELOPER_POLICY_CATALOG",
        "policies": [{"id": policy_id, "path": f"developer/policy/{policy_id}.yaml", "policy_class": "PTSIP_DEVELOPER_POLICY", "status": "ACTIVE"} for policy_id in policy_ids],
    }
    (root / "developer/policy/index.yaml").write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8")


def contract_validator(schema: dict[str, object]):
    return developer_contract_validator(schema, SOURCE_ROOT)
