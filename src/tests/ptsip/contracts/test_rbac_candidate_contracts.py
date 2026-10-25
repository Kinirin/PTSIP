"""Non-normative RBAC candidate contract checks.

These tests validate draft shape and package mirrors, NOT actual RBAC approval,
Seal authenticity, Activation permission or final pp.2.00 registration.
"""
from __future__ import annotations

import json
from importlib.resources import files
from pathlib import Path

from jsonschema import Draft202012Validator

_ROOT = Path(__file__).resolve().parents[4]
_SCHEMA_ROOT = ".ptsip/contracts/schemas/compile-sealed-rbac"
_PACKAGE = files("ptsip").joinpath("specdata", "compile-sealed-rbac")
_FILES = (
    "responsibility",
    "condition-atom",
    "permission-atom",
    "binding",
    "sealed-contract",
    "activation",
)


def _schema(name: str) -> dict:
    filename = f"{name}.candidate.schema.json"
    canonical = json.loads(
        (_ROOT / _SCHEMA_ROOT / filename).read_text(encoding="utf-8")
    )
    embedded = json.loads(_PACKAGE.joinpath(filename).read_text(encoding="utf-8"))
    assert canonical == embedded
    assert "DRAFT_NON_NORMATIVE" in canonical["$comment"]
    Draft202012Validator.check_schema(canonical)
    return canonical


def _samples() -> dict[str, dict]:
    return {
        "responsibility": {
            "id": "RESP-001", "kind": "RESPONSIBILITY", "revision": "Rev.0001",
            "purpose": "Review deployment", "condition_refs": ["COND-001"],
        },
        "condition-atom": {
            "id": "COND-001", "kind": "CONDITION_ATOM", "revision": "Rev.0001",
            "description": "Change validated",
            "predicate": {"fact_ref": "verified_change", "operator": "EXISTS"},
        },
        "permission-atom": {
            "id": "PERM-001", "kind": "PERMISSION_ATOM", "revision": "Rev.0001",
            "action": "read", "target": "report", "scope": "project",
        },
        "binding": {
            "id": "BIND-001", "kind": "BINDING", "revision": "Rev.0001",
            "responsibility_ref": "RESP-001", "permission_refs": ["PERM-001"],
            "approval_ref": "POLICY-APPROVAL-001",
        },
        "sealed-contract": {
            "id": "SEAL-001", "kind": "SEALED_CONTRACT", "revision": "Rev.0001",
            "responsibility_ref": "RESP-001", "source_digest_sha256": "0" * 64,
            "compiler_id": "compiler-candidate", "sealed_digest_sha256": "1" * 64,
        },
        "activation": {
            "id": "ACT-001", "kind": "ACTIVATION", "revision": "Rev.0001",
            "responsibility_ref": "RESP-001", "seal_digest_sha256": "1" * 64,
            "assessment": "BLOCKED", "authority": "NOT_ASSERTED",
        },
    }


def test_six_independent_candidate_schemas_accept_only_expected_draft_shapes() -> None:
    samples = _samples()
    for name in _FILES:
        schema = _schema(name)
        validator = Draft202012Validator(schema)
        sample = samples[name]
        assert validator.is_valid(sample), name

        without_id = dict(sample)
        del without_id["id"]
        assert not validator.is_valid(without_id), f"{name}: no id"

        invalid_revision = dict(sample, revision=17)
        assert not validator.is_valid(invalid_revision), f"{name}: numeric revision"

        unknown_grant = dict(sample, execution_authorized=True)
        assert not validator.is_valid(unknown_grant), f"{name}: authority injection"

    for name in _FILES:
        # Strict kind identity must not be silently coerced.
        sample = dict(samples[name], kind="OTHER")
        assert not Draft202012Validator(_schema(name)).is_valid(sample)


def test_activation_candidate_can_never_claim_active_authority() -> None:
    validator = Draft202012Validator(_schema("activation"))
    base = _samples()["activation"]
    for invalid in [
        dict(base, assessment="ACTIVE"),
        dict(base, authority="AUTHORIZED"),
        dict(base, authority=True),
    ]:
        assert not validator.is_valid(invalid)


def test_candidate_index_mirrors_and_preserves_separate_artifact_plane() -> None:
    name = "candidate-index.json"
    source = json.loads(
        (_ROOT / _SCHEMA_ROOT / name).read_text(encoding="utf-8")
    )
    embedded = json.loads(_PACKAGE.joinpath(name).read_text(encoding="utf-8"))
    assert source == embedded
    assert source["status"] == "DRAFT_NON_NORMATIVE"
    assert source["effective_policy_override"] is False
    assert source["rbac_operation_registered"] is False
    assert source["rbac_authorization_permitted"] is False
    assert source["release_admission_evidence"] is False
    assert source["source_revision_verification_candidate"]["rule"] == (
        "PRESENT_IN_BOTH_OR_NEITHER_AND_EXACT_EQUAL"
    )
    kinds = source["independent_schema_candidates"]
    assert len(kinds) == 6
    assert [x["file"] for x in kinds] == [
        f"{name}.candidate.schema.json" for name in _FILES
    ]
    assert all(x["role"] == "DECLARATIVE_SOURCE" for x in kinds[:4])
    assert all(x["role"] == "DERIVED_ARTIFACT" for x in kinds[4:])
    assert all(x["authoritative"] is False for x in kinds)


def test_original_six_schema_scaffolds_remain_fail_closed() -> None:
    for name in _FILES:
        filename = name + ".schema.json"
        canonical = json.loads(
            (_ROOT / _SCHEMA_ROOT / filename).read_text(encoding="utf-8")
        )
        embedded = json.loads(_PACKAGE.joinpath(filename).read_text(encoding="utf-8"))
        assert canonical == embedded
        assert canonical["not"] == {}
        assert not Draft202012Validator(canonical).is_valid({})
        assert not Draft202012Validator(canonical).is_valid(_samples()[name])
