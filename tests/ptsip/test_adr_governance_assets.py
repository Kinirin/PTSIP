from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

import pytest
import yaml
from jsonschema import Draft202012Validator, ValidationError


ROOT = Path(__file__).resolve().parents[2]
DECISIONS = ROOT / "decisions"
SCHEMAS = ROOT / "schemas"

ADR_SCHEMA_PATH = SCHEMAS / "ptsip-adr.schema.json"
TEMPLATE_SCHEMA_PATH = SCHEMAS / "ptsip-adr-template.schema.json"
INDEX_SCHEMA_PATH = SCHEMAS / "ptsip-adr-index.schema.json"
REGISTRY_SCHEMA_PATH = SCHEMAS / "ptsip-governance-authority-registry.schema.json"
SEMANTICS_SCHEMA_PATH = SCHEMAS / "ptsip-governance-authority-semantics.schema.json"

TEMPLATE_PATH = DECISIONS / "ADR-TEMPLATE.yaml"
INDEX_PATH = DECISIONS / "INDEX.yaml"
REGISTRY_PATH = DECISIONS / "AUTHORITY-SCHEMA-REGISTRY.yaml"
P03_ROLE_ANALYZER = ROOT / ".github" / "scripts" / "p03_authority_role_matrix.py"
P03_ROLE_REGISTRY = ROOT / "planning" / "0.4.0" / "WU-02" / "p03-authority-role-provisional-dimensions.yaml"
P03_ROLE_MATRIX = ROOT / "planning" / "0.4.0" / "WU-02" / "p03-authority-role-matrix.generated.yaml"


def _json(path: Path) -> dict[str, object]:
    return json.loads(path.read_text(encoding="utf-8"))


def _yaml(path: Path) -> dict[str, object]:
    value = yaml.safe_load(path.read_text(encoding="utf-8"))
    assert isinstance(value, dict)
    return value


def _adr_paths() -> list[Path]:
    return sorted(DECISIONS.glob("ADR-[0-9][0-9][0-9][0-9]-*.yaml"))


@pytest.mark.parametrize(
    "path",
    [
        ADR_SCHEMA_PATH,
        TEMPLATE_SCHEMA_PATH,
        INDEX_SCHEMA_PATH,
        REGISTRY_SCHEMA_PATH,
        SEMANTICS_SCHEMA_PATH,
    ],
)
def test_governance_json_schemas_are_valid_draft_2020_12(path: Path) -> None:
    Draft202012Validator.check_schema(_json(path))


def test_adr_template_validates_and_requires_yaml_single_semantic_contract() -> None:
    schema = _json(TEMPLATE_SCHEMA_PATH)
    template = _yaml(TEMPLATE_PATH)
    Draft202012Validator(schema).validate(template)

    assert template["output"]["format"] == "yaml"
    assert template["semantic_contract"]["authority_semantic_cardinality"] == "EXACTLY_ONE"
    assert template["semantic_contract"]["opaque_natural_language_in_authority_semantics"] == "FORBIDDEN"
    assert template["history_policy"]["representation_migration_exception"]["status"] == "CONSUMED_BY_P03"


def test_current_tree_contains_no_markdown_adr_records() -> None:
    assert list(DECISIONS.glob("ADR-[0-9][0-9][0-9][0-9]-*.md")) == []


def test_registry_is_valid_and_contract_identities_are_unique() -> None:
    registry = _yaml(REGISTRY_PATH)
    Draft202012Validator(_json(REGISTRY_SCHEMA_PATH)).validate(registry)

    entries = registry["entries"]
    identity = [
        (entry["authority_type"], entry["schema_id"], entry["schema_version"])
        for entry in entries
    ]
    assert len(identity) == len(set(identity))
    assert len({entry["authority_type"] for entry in entries}) == len(entries)
    assert len({entry["schema_id"] for entry in entries}) == len(entries)


def test_every_machine_adr_resolves_exact_registered_semantics() -> None:
    adr_schema = _json(ADR_SCHEMA_PATH)
    semantics_schema = _json(SEMANTICS_SCHEMA_PATH)
    registry = _yaml(REGISTRY_PATH)

    by_identity = {
        (entry["authority_type"], entry["schema_id"], entry["schema_version"]): entry
        for entry in registry["entries"]
    }

    paths = _adr_paths()
    assert paths

    for path in paths:
        record = _yaml(path)
        Draft202012Validator(adr_schema).validate(record)

        contract = record["authority_contract"]
        key = (
            contract["authority_type"],
            contract["schema_id"],
            contract["schema_version"],
        )
        assert key in by_identity

        definition = by_identity[key]["schema_definition"]
        assert definition in semantics_schema["$defs"]
        Draft202012Validator(semantics_schema["$defs"][definition]).validate(
            record["authority_semantics"]
        )

        migration = record["representation_migration"]
        assert migration["from_format"] == "MARKDOWN"
        assert migration["semantic_change"] is False
        assert len(migration["source_blob_sha"]) == 40


def test_unregistered_or_mismatched_semantics_are_rejected() -> None:
    paths = _adr_paths()
    assert paths

    semantics_schema = _json(SEMANTICS_SCHEMA_PATH)
    registry = _yaml(REGISTRY_PATH)
    record = _yaml(paths[0])
    contract = record["authority_contract"]

    entry = next(
        item
        for item in registry["entries"]
        if (
            item["authority_type"],
            item["schema_id"],
            item["schema_version"],
        )
        == (
            contract["authority_type"],
            contract["schema_id"],
            contract["schema_version"],
        )
    )
    invalid = dict(record["authority_semantics"])
    invalid["UNREGISTERED_FIELD"] = "FORBIDDEN"

    with pytest.raises(ValidationError):
        Draft202012Validator(
            semantics_schema["$defs"][entry["schema_definition"]]
        ).validate(invalid)


def test_index_routes_every_current_topic_to_matching_machine_adr() -> None:
    index = _yaml(INDEX_PATH)
    Draft202012Validator(_json(INDEX_SCHEMA_PATH)).validate(index)

    routed_ids: set[str] = set()
    for topic_id, route in index["topics"].items():
        path = ROOT / route["path"]
        assert path.exists()
        assert path.suffix == ".yaml"

        record = _yaml(path)
        assert record["decision"]["id"] == route["current_adr"]
        assert record["decision"]["topic_id"] == topic_id
        routed_ids.add(record["decision"]["id"])

    all_ids = {_yaml(path)["decision"]["id"] for path in _adr_paths()}
    assert routed_ids == all_ids


def test_p03_authority_role_matrix_is_current_rectangular_and_semantically_evaluated() -> None:
    result = subprocess.run(
        [
            sys.executable,
            str(P03_ROLE_ANALYZER),
            "--repo-root",
            str(ROOT),
            "--check",
        ],
        text=True,
        encoding="utf-8",
        errors="replace",
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )

    assert result.returncode == 0, result.stderr

    matrix = _yaml(P03_ROLE_MATRIX)
    dimensions = matrix["dimensions"]
    rows = matrix["rows"]
    assert matrix["source"]["vocabulary_registration"] is False
    assert matrix["source"]["runtime_authority"] == "NONE"
    assert matrix["validation"]["rectangular"] is True
    assert matrix["validation"]["dimension_count"] == len(dimensions)
    assert matrix["validation"]["reviewed_row_count"] == len(rows)

    for adr_id, row in rows.items():
        effects = row["role_effect_analysis"]
        assert list(effects) == dimensions, adr_id
        assert all(type(value) is bool for value in effects.values())

    assert rows["ADR-0001"]["role_effect_analysis"] == {
        "establish_canonical_identity": True,
        "establish_decision_criterion_priority": True,
        "define_classification_basis": True,
        "require_architectural_boundary": True,
        "require_versioned_normative_contract": True,
    }


def test_p03_new_provisional_dimension_automatically_backfills_reviewed_rows(
    tmp_path: Path,
) -> None:
    registry = _yaml(P03_ROLE_REGISTRY)
    dimensions = registry["dimensions"]
    dimensions["probe_absent_semantic"] = {
        "status": "PROVISIONAL",
        "introduced_by": "ADR-0002",
        "expression": {
            "path": "authority_semantics.__p03_absent_probe__",
            "operator": "PRESENT",
        },
    }

    registry_path = tmp_path / "dimensions.yaml"
    output_path = tmp_path / "matrix.yaml"
    registry_path.write_text(
        yaml.safe_dump(registry, sort_keys=False, allow_unicode=True),
        encoding="utf-8",
    )

    result = subprocess.run(
        [
            sys.executable,
            str(P03_ROLE_ANALYZER),
            "--repo-root",
            str(ROOT),
            "--registry",
            str(registry_path),
            "--output",
            str(output_path),
            "--write",
        ],
        text=True,
        encoding="utf-8",
        errors="replace",
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )

    assert result.returncode == 0, result.stderr

    generated = _yaml(output_path)
    assert generated["dimensions"][-1] == "probe_absent_semantic"
    assert generated["rows"]["ADR-0001"]["role_effect_analysis"]["probe_absent_semantic"] is False
    assert set(generated["rows"]["ADR-0001"]["role_effect_analysis"]) == set(
        generated["dimensions"]
    )
