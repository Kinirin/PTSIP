from __future__ import annotations

import importlib.util
import json
import shutil
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

    original_dimensions = {
        "establish_canonical_identity": True,
        "establish_decision_criterion_priority": True,
        "define_classification_basis": True,
        "require_architectural_boundary": True,
        "require_versioned_normative_contract": True,
    }
    effects = rows["ADR-0001"]["role_effect_analysis"]
    assert {name: effects[name] for name in original_dimensions} == original_dimensions


@pytest.fixture
def p03_analyzer():
    spec = importlib.util.spec_from_file_location("p03_authority_role_matrix", P03_ROLE_ANALYZER)
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def _write_yaml(path: Path, payload: dict[str, object]) -> None:
    path.write_text(yaml.safe_dump(payload, sort_keys=False, allow_unicode=True), encoding="utf-8")


def _p03_dimension(path: str, operator: str, **operands: object) -> dict[str, object]:
    return {
        "status": "PROVISIONAL",
        "introduced_by": "ADR-0002",
        "expression": {"path": f"authority_semantics.{path}", "operator": operator, **operands},
    }


@pytest.fixture
def p03_repo(tmp_path: Path) -> Path:
    # Two reviewed records isolate engine behavior from the growing corpus and
    # ensure generation does not need to open any unreviewed ADR source.
    (tmp_path / "schemas").mkdir()
    (tmp_path / "decisions").mkdir()
    for path in (ADR_SCHEMA_PATH, INDEX_SCHEMA_PATH, REGISTRY_SCHEMA_PATH, SEMANTICS_SCHEMA_PATH):
        shutil.copyfile(path, tmp_path / "schemas" / path.name)
    shutil.copyfile(REGISTRY_PATH, tmp_path / "decisions" / REGISTRY_PATH.name)
    index = _yaml(INDEX_PATH)
    index["topics"] = {
        topic: route for topic, route in index["topics"].items()
        if route["current_adr"] in {"ADR-0001", "ADR-0002"}
    }
    _write_yaml(tmp_path / "decisions" / INDEX_PATH.name, index)
    for route in index["topics"].values():
        shutil.copyfile(ROOT / route["path"], tmp_path / route["path"])
    _write_yaml(tmp_path / "dimensions.yaml", {
        "schema_version": "ptsip-p03-authority-role-provisional-dimensions/v1",
        "analysis": {
            "index_path": "decisions/INDEX.yaml",
            "first_adr": "ADR-0001",
            "last_adr": "ADR-0002",
            "reviewed_through": "ADR-0002",
            "vocabulary_registration": False,
            "runtime_authority": "NONE",
        },
        "dimensions": {
            "identity": {
                **_p03_dimension("canonical_term", "NON_EMPTY"),
                "introduced_by": "ADR-0001",
            },
        },
    })
    return tmp_path


def _p03_run(analyzer, repo: Path, mode: str = "--write") -> int:
    return analyzer.main([
        "--repo-root", str(repo), "--registry", "dimensions.yaml",
        "--output", "matrix.yaml", mode,
    ])


def test_p03_new_provisional_dimension_automatically_backfills_reviewed_rows(
    p03_analyzer, p03_repo: Path,
) -> None:
    assert _p03_run(p03_analyzer, p03_repo) == 0
    registry_path = p03_repo / "dimensions.yaml"
    registry = _yaml(registry_path)
    registry["dimensions"]["versioned_contract"] = _p03_dimension(
        "versioned_specification", "EQUALS", value=True
    )
    registry["dimensions"]["read_only_default"] = _p03_dimension(
        "inspection_default_mutation_mode", "EQUALS", value="READ_ONLY"
    )
    _write_yaml(registry_path, registry)

    assert _p03_run(p03_analyzer, p03_repo) == 0
    generated = _yaml(p03_repo / "matrix.yaml")
    assert generated["rows"]["ADR-0001"]["role_effect_analysis"] == {
        "identity": True, "versioned_contract": True, "read_only_default": False,
    }
    assert generated["rows"]["ADR-0002"]["role_effect_analysis"] == {
        "identity": False, "versioned_contract": False, "read_only_default": True,
    }
    assert generated["validation"] == {
        "rectangular": True, "reviewed_row_count": 2, "dimension_count": 3,
    }
    first_output = (p03_repo / "matrix.yaml").read_bytes()
    assert _p03_run(p03_analyzer, p03_repo) == 0
    assert (p03_repo / "matrix.yaml").read_bytes() == first_output
    assert _p03_run(p03_analyzer, p03_repo, "--check") == 0


def test_p03_predicate_change_recomputes_every_reviewed_row(p03_analyzer, p03_repo: Path) -> None:
    assert _p03_run(p03_analyzer, p03_repo) == 0
    registry_path = p03_repo / "dimensions.yaml"
    registry = _yaml(registry_path)
    registry["dimensions"]["identity"]["expression"] = _p03_dimension(
        "inspection_default_mutation_mode", "EQUALS", value="READ_ONLY"
    )["expression"]
    _write_yaml(registry_path, registry)

    old_output = (p03_repo / "matrix.yaml").read_bytes()
    assert _p03_run(p03_analyzer, p03_repo, "--check") == 1
    assert (p03_repo / "matrix.yaml").read_bytes() == old_output
    assert _p03_run(p03_analyzer, p03_repo) == 0
    rows = _yaml(p03_repo / "matrix.yaml")["rows"]
    assert rows["ADR-0001"]["role_effect_analysis"]["identity"] is False
    assert rows["ADR-0002"]["role_effect_analysis"]["identity"] is True


@pytest.mark.parametrize("corruption", ["missing", "null", "incomplete", "wrong_type", "unregistered", "mismatched"])
def test_p03_invalid_authority_source_never_writes_false_defaults(
    p03_analyzer, p03_repo: Path, corruption: str, capsys,
) -> None:
    assert _p03_run(p03_analyzer, p03_repo) == 0
    output_path = p03_repo / "matrix.yaml"
    old_output = output_path.read_bytes()
    source_path = next((p03_repo / "decisions").glob("ADR-0001-*.yaml"))
    record = _yaml(source_path)
    if corruption == "missing":
        del record["authority_semantics"]
    elif corruption == "null":
        record["authority_semantics"] = None
    elif corruption == "incomplete":
        del record["authority_semantics"]["canonical_term"]
    elif corruption == "wrong_type":
        record["authority_semantics"]["requires_product_toolchain_boundary"] = 1
    elif corruption == "unregistered":
        record["authority_contract"]["schema_version"] = 99
    elif corruption == "mismatched":
        other = _yaml(next((p03_repo / "decisions").glob("ADR-0002-*.yaml")))
        record["authority_contract"] = other["authority_contract"]
    _write_yaml(source_path, record)

    assert _p03_run(p03_analyzer, p03_repo) == 1
    assert "ADR-0001" in capsys.readouterr().err
    assert output_path.read_bytes() == old_output
    output_path.unlink()
    assert _p03_run(p03_analyzer, p03_repo) == 1
    assert not output_path.exists()


@pytest.mark.parametrize("expression", [
    {"path": "authority_semantics.typo_field", "operator": "PRESENT"},
    {"path": "decision.id", "operator": "NON_EMPTY"},
    {"path": "authority_semantics.classifies_by", "operator": "CONTAINS_ALL", "value": "PURPOSE"},
    {"path": "authority_semantics.classifies_by", "operator": "CONTAINS_ALL", "value": []},
    {"path": "authority_semantics.requires_product_toolchain_boundary", "operator": "EQUALS", "value": 1},
    {"path": "authority_semantics.canonical_term", "operator": "CONTAINS", "value": True},
    {"path": "authority_semantics.canonical_term", "operator": "UNREGISTERED"},
    {"path": "authority_semantics.canonical_term", "operator": "EQUALS"},
    {"path": "authority_semantics.canonical_term", "operator": "PRESENT", "value": True},
    {"all": [
        {"path": "authority_semantics.inspection_default_mutation_mode", "operator": "PRESENT"},
        {"path": "authority_semantics.classifies_by", "operator": "CONTAINS_ALL", "value": False},
    ]},
])
def test_p03_invalid_predicate_rejected_even_when_semantic_path_is_absent(
    p03_analyzer, p03_repo: Path, expression: dict[str, object],
) -> None:
    registry_path = p03_repo / "dimensions.yaml"
    registry = _yaml(registry_path)
    registry["dimensions"]["identity"]["expression"] = expression
    _write_yaml(registry_path, registry)
    assert _p03_run(p03_analyzer, p03_repo) == 1
    assert not (p03_repo / "matrix.yaml").exists()


@pytest.mark.parametrize("field,value", [
    ("reviewed_through", "ADR-0003"),
    ("reviewed_through", "ADR-0000"),
    ("vocabulary_registration", True),
    ("runtime_authority", "GOVERNANCE"),
])
def test_p03_invalid_analysis_registry_is_rejected(
    p03_analyzer, p03_repo: Path, field: str, value: object,
) -> None:
    registry_path = p03_repo / "dimensions.yaml"
    registry = _yaml(registry_path)
    registry["analysis"][field] = value
    _write_yaml(registry_path, registry)
    assert _p03_run(p03_analyzer, p03_repo) == 1


def test_p03_dimension_cannot_be_introduced_by_unreviewed_adr(p03_analyzer, p03_repo: Path) -> None:
    registry_path = p03_repo / "dimensions.yaml"
    registry = _yaml(registry_path)
    registry["dimensions"]["identity"]["introduced_by"] = "ADR-0003"
    _write_yaml(registry_path, registry)
    assert _p03_run(p03_analyzer, p03_repo) == 1


@pytest.mark.parametrize("corruption", ["missing_row", "extra_row", "missing_cell", "null", "numeric", "stale", "range"])
def test_p03_incomplete_or_stale_matrix_is_rejected(
    p03_analyzer, p03_repo: Path, corruption: str,
) -> None:
    assert _p03_run(p03_analyzer, p03_repo) == 0
    output_path = p03_repo / "matrix.yaml"
    matrix = _yaml(output_path)
    if corruption == "missing_row":
        del matrix["rows"]["ADR-0002"]
        matrix["validation"]["reviewed_row_count"] = 1
    elif corruption == "extra_row":
        matrix["rows"]["ADR-0003"] = matrix["rows"]["ADR-0002"]
        matrix["validation"]["reviewed_row_count"] = 3
    elif corruption == "missing_cell":
        del matrix["rows"]["ADR-0001"]["role_effect_analysis"]["identity"]
    elif corruption in {"null", "numeric", "stale"}:
        matrix["rows"]["ADR-0001"]["role_effect_analysis"]["identity"] = {
            "null": None, "numeric": 1, "stale": False,
        }[corruption]
    elif corruption == "range":
        matrix["source"]["first_adr"] = "ADR-0003"
    _write_yaml(output_path, matrix)
    invalid_output = output_path.read_bytes()
    assert _p03_run(p03_analyzer, p03_repo, "--check") == 1
    assert output_path.read_bytes() == invalid_output


@pytest.mark.parametrize("expression,expected", [
    ({"path": "authority_semantics.canonical_term", "operator": "PRESENT"}, True),
    ({"path": "authority_semantics.canonical_term", "operator": "NOT_EQUALS", "value": "OTHER"}, True),
    ({"path": "authority_semantics.classifies_by", "operator": "CONTAINS", "value": "PURPOSE"}, True),
    ({"path": "authority_semantics.classifies_by", "operator": "CONTAINS_ALL", "value": ["PURPOSE", "LIFECYCLE_OWNERSHIP"]}, True),
    ({"not": {"path": "authority_semantics.canonical_term", "operator": "PRESENT"}}, False),
    ({"any": [
        {"path": "authority_semantics.inspection_default_mutation_mode", "operator": "PRESENT"},
        {"path": "authority_semantics.canonical_term", "operator": "PRESENT"},
    ]}, True),
    ({"all": [
        {"path": "authority_semantics.inspection_default_mutation_mode", "operator": "PRESENT"},
        {"path": "authority_semantics.canonical_term", "operator": "PRESENT"},
    ]}, False),
])
def test_p03_predicate_operators_preserve_boolean_semantics(
    p03_analyzer, p03_repo: Path, expression: dict[str, object], expected: bool,
) -> None:
    registry_path = p03_repo / "dimensions.yaml"
    registry = _yaml(registry_path)
    registry["dimensions"]["identity"]["expression"] = expression
    _write_yaml(registry_path, registry)
    generated = p03_analyzer.build_matrix(p03_repo, registry_path)
    assert generated["rows"]["ADR-0001"]["role_effect_analysis"]["identity"] is expected
