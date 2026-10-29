from __future__ import annotations

import json
from copy import deepcopy
from pathlib import Path

import pytest
import yaml
from jsonschema import Draft202012Validator

from ptsip.profiles.rule_records import (
    RuleProfileError,
    discover_rule_records,
    load_rule_records,
    resolve_rule_record,
)

REPOSITORY = Path(__file__).resolve().parents[4]
CONTRACT_PATH = ".ptsip/contracts/schemas/Policy/Policy_Rules/rule_record.schema.json"


def records_fixture() -> list[dict]:
    root = "rule_sha256:" + "a" * 64
    child = "rule_ABsha256:" + "b" * 64
    grandchild = "rule_AB.CDsha256:" + "c" * 64
    sibling = "rule_ABsha256:" + "d" * 64
    return [
        {"ID": root, "rule_value": {"constant": "AB", "root_id": child}},
        {"ID": child, "parent_id": root, "rule_value": {"constant": "CD", "root_id": grandchild}},
        {"ID": grandchild, "parent_id": child, "rule_value": {"constant": "EF", "explanation": "leaf"}},
        {"ID": sibling, "parent_id": root, "rule_value": {"constant": "GH", "explanation": "sibling"}},
    ]


def project_fixture(tmp_path: Path, records: list[dict]) -> tuple[Path, dict]:
    root = tmp_path
    directory = root / ".ptsip" / "profiles"
    directory.mkdir(parents=True)
    schema_target = root / CONTRACT_PATH
    schema_target.parent.mkdir(parents=True)
    schema_target.write_text((REPOSITORY / CONTRACT_PATH).read_text(encoding="utf-8"), encoding="utf-8")
    (directory / "index.json").write_text(json.dumps({
        "schema_version": "ptsip-local-profile-catalog/v1",
        "default_profile": "main",
        "profiles": [
            {"id": "main", "resource": "main.ptsip.yaml"},
            {"id": "rules", "resource": "rules.ptsip.yaml"}
        ]
    }), encoding="utf-8")
    (directory / "rules.ptsip.yaml").write_text(
        yaml.safe_dump({"rules": records}, sort_keys=False), encoding="utf-8"
    )
    main = {"ptsip": {"rule_profile": "rules.ptsip.yaml"}}
    return root, main


def test_record_schema_and_five_allowed_shapes() -> None:
    schema = json.loads((REPOSITORY / CONTRACT_PATH).read_text(encoding="utf-8"))
    Draft202012Validator.check_schema(schema)
    records = records_fixture()
    records.append({"ID": "rule_sha256:" + "e" * 64, "rule_value": {"constant": "HI", "explanation": "root leaf"}})
    assert not list(Draft202012Validator(schema).iter_errors({"rules": records}))
    defs = schema["$defs"]
    assert len(defs["record"]["oneOf"]) == 5
    assert set(defs["record"]["properties"]["rule_value"]["properties"]) == {
        "constant", "root_id", "explanation"
    }
    assert "constant2" not in json.dumps(schema)
    assert "constant3" not in json.dumps(schema)


def test_discovery_and_resolution_of_full_three_level_graph(tmp_path: Path) -> None:
    records = records_fixture()
    root, main = project_fixture(tmp_path, records)
    assert load_rule_records(root, main_payload=main) == records
    assert discover_rule_records(root, main_payload=main) == sorted(r["ID"] for r in records)
    resolved = resolve_rule_record(root, records[2]["ID"], main_payload=main)
    assert resolved["status"] == "RESOLVED"
    assert resolved["record"] == records[2]
    assert resolved["normative_authority"] is False
    assert resolved["execution_authorized"] is False
    assert resolved["identity_rewritten"] is False
    assert resolve_rule_record(root, "missing", main_payload=main)["status"] == "UNRESOLVED"


def test_duplicate_identity_rejected_even_if_json_schema_accepts_it(tmp_path: Path) -> None:
    records = records_fixture()
    records.append(deepcopy(records[3]))
    root, main = project_fixture(tmp_path, records)
    with pytest.raises(RuleProfileError, match="duplicate rule ID"):
        load_rule_records(root, main_payload=main)


def test_parent_prefix_and_pointer_mismatch_fail_closed(tmp_path: Path) -> None:
    records = records_fixture()
    records[0]["rule_value"]["constant"] = "XY"
    root, main = project_fixture(tmp_path, records)
    with pytest.raises(RuleProfileError, match="child prefix"):
        load_rule_records(root, main_payload=main)

    records = records_fixture()
    records[0]["rule_value"]["root_id"] = "rule_ABsha256:" + "e" * 64
    root, main = project_fixture(tmp_path, records)
    with pytest.raises(RuleProfileError, match="direct child"):
        load_rule_records(root, main_payload=main)


def test_missing_and_wrong_parent_rejected(tmp_path: Path) -> None:
    records = records_fixture()
    records[2]["parent_id"] = "rule_ABsha256:" + "e" * 64
    root, main = project_fixture(tmp_path, records)
    with pytest.raises(RuleProfileError, match="missing parent"):
        load_rule_records(root, main_payload=main)

    records = records_fixture()
    records[2]["parent_id"] = records[0]["ID"]
    root, main = project_fixture(tmp_path, records)
    with pytest.raises(RuleProfileError, match="rule record schema rejected"):
        load_rule_records(root, main_payload=main)


def test_hash_segments_stay_unchanged_after_declared_prefix_change(tmp_path: Path) -> None:
    records = records_fixture()
    initial_root_id = records[0]["ID"]
    initial_child_hash = records[1]["ID"].split("sha256:", 1)[1]
    initial_grandchild_hash = records[2]["ID"].split("sha256:", 1)[1]
    initial_sibling_hash = records[3]["ID"].split("sha256:", 1)[1]

    records[0]["rule_value"]["constant"] = "XY"
    new_child = "rule_XYsha256:" + initial_child_hash
    new_grandchild = "rule_XY.CDsha256:" + initial_grandchild_hash
    records[0]["rule_value"]["root_id"] = new_child
    records[1]["ID"] = new_child
    records[1]["rule_value"]["root_id"] = new_grandchild
    records[2]["ID"] = new_grandchild
    records[2]["parent_id"] = new_child
    records[3]["ID"] = "rule_XYsha256:" + initial_sibling_hash

    root, main = project_fixture(tmp_path, records)
    assert load_rule_records(root, main_payload=main) == records
    assert records[0]["ID"] == initial_root_id
    assert records[1]["ID"].endswith("sha256:" + initial_child_hash)
    assert records[2]["ID"].endswith("sha256:" + initial_grandchild_hash)


def test_unsafe_link_and_unregistered_catalog_rejected(tmp_path: Path) -> None:
    root, main = project_fixture(tmp_path, records_fixture())
    main["ptsip"]["rule_profile"] = "../rules.ptsip.yaml"
    with pytest.raises(RuleProfileError, match="relative"):
        load_rule_records(root, main_payload=main)

    main["ptsip"]["rule_profile"] = "rules.ptsip.yaml"
    index_path = root / ".ptsip/profiles/index.json"
    index = json.loads(index_path.read_text(encoding="utf-8"))
    index["profiles"].pop()
    index_path.write_text(json.dumps(index), encoding="utf-8")
    with pytest.raises(RuleProfileError, match="missing, ambiguous"):
        load_rule_records(root, main_payload=main)


def test_extra_constants_and_unregistered_shapes_fail_schema(tmp_path: Path) -> None:
    records = records_fixture()
    records[2]["rule_value"]["constant3"] = "XYZ"
    root, main = project_fixture(tmp_path, records)
    with pytest.raises(RuleProfileError, match="schema rejected"):
        load_rule_records(root, main_payload=main)
