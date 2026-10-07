from __future__ import annotations

import json
import shutil
from importlib.resources import files
from pathlib import Path

import pytest
from jsonschema import ValidationError

from agent_contracts.candidate import (
    INDEX_METADATA_FIELDS,
    CandidateValidationError,
    build_candidate_assets,
    read_json,
    validate_candidate_assets,
)
from agent_contracts.machine_resolver import MachineContractResolver


@pytest.fixture
def contract_root(tmp_path: Path) -> Path:
    target = tmp_path / "contracts"
    shutil.copytree(
        Path(str(files("agent_contracts"))), target,
        ignore=shutil.ignore_patterns("__pycache__", "*.pyc"),
    )
    return target


def _write(root: Path, ref: str, payload: dict) -> None:
    (root / ref).write_text(json.dumps(payload, ensure_ascii=False) + "\n", encoding="utf-8")


def test_materialized_candidates_match_current_bound_assets() -> None:
    root = files("agent_contracts")
    generated = build_candidate_assets(root)
    for ref, payload in generated.items():
        assert payload == read_json(root, ref)
    assert validate_candidate_assets(root)["operations"] == 5
    index = generated["index.json"]
    schema = read_json(root, "schemas/candidate-index.schema.json")
    for field in INDEX_METADATA_FIELDS:
        assert index[field] == schema["properties"][field]["const"]
    assert index["normative_authority"] is False
    assert index["canonical_digest"] == "UNRESOLVED"


@pytest.mark.parametrize("field", INDEX_METADATA_FIELDS)
def test_missing_metadata_binding_is_not_inferred(contract_root: Path, field: str) -> None:
    ref = "schemas/candidate-index.schema.json"
    schema = read_json(contract_root, ref)
    schema["properties"][field].pop("const")
    _write(contract_root, ref, schema)
    with pytest.raises(CandidateValidationError, match="unresolved required index metadata binding"):
        build_candidate_assets(contract_root)


def test_failure_projection_drift_still_blocks_parity(contract_root: Path) -> None:
    ref = "contracts/current.json"
    group = read_json(contract_root, ref)
    failure = next(iter(group["failures"].values()))
    action = next(iter(failure["actions"].values()))
    action["failure_outcomes"].append("UNDECLARED_FAILURE")
    _write(contract_root, ref, group)
    with pytest.raises(CandidateValidationError, match="candidate differs from current migration source"):
        validate_candidate_assets(contract_root)


@pytest.mark.parametrize("ref", [
    "canonicalization/ptsip-canonical-json-v1.json",
    "digests/policy-v1.json",
    "digests/projection-v1.json",
])
def test_serialization_and_digest_reference_drift_requires_revalidation(
    contract_root: Path, ref: str,
) -> None:
    resolver = MachineContractResolver(contract_root)
    operation = "PTSIP-OP-ADOPT-001"
    assert resolver.resolve(operation)["status"] == "RESOLVED"
    target = contract_root / ref
    target.write_bytes(target.read_bytes() + b"\n")
    result = resolver.resolve(operation)
    assert result["status"] == "UNRESOLVED"
    assert result["reason"] == "REVALIDATION_REQUIRED"
    assert result["mutation_authorized"] is False
    assert result["normative_authority"] is False


def test_unknown_identity_does_not_gain_resolution_authority() -> None:
    result = MachineContractResolver().resolve("PTSIP-OP-NOT-REGISTERED-001")
    assert result["status"] == "UNRESOLVED"
    assert result["reason"] == "UNKNOWN_EXACT_IDENTITY"
    assert result["mutation_authorized"] is False


@pytest.mark.parametrize("field,value", [
    ("canonical_digest", "SHA-256"),
    ("normative_authority", True),
    ("status", "ACTIVE"),
])
def test_unresolved_digest_cannot_be_promoted_by_candidate_edits(
    contract_root: Path, field: str, value: object,
) -> None:
    index = read_json(contract_root, "index.json")
    index[field] = value
    _write(contract_root, "index.json", index)
    with pytest.raises(ValidationError):
        validate_candidate_assets(contract_root)
