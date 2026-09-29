"""Mechanical migration projections; these candidates never activate authority."""
from __future__ import annotations

import json
from importlib.resources import files
from pathlib import Path, PurePosixPath

import yaml
from jsonschema import Draft202012Validator


class CandidateValidationError(ValueError):
    pass


KINDS = {
    "specs": ("spec", "id"),
    "operations": ("operation", "operation_id"),
    "actions": ("action", "action_id"),
    "conditions": ("condition", "condition_id"),
    "gates": ("gate", "gate_id"),
    "io_schemas": ("io", "x-ptsip-io-id"),
    "vocabularies": ("vocabulary", "vocabulary_id"),
}


def resource(root, ref: str):
    path = PurePosixPath(ref)
    if not ref or path.is_absolute() or ".." in path.parts or "\\" in ref or ":" in ref:
        raise CandidateValidationError(f"unsafe resource ref: {ref!r}")
    return root.joinpath(*path.parts)


def _unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise CandidateValidationError(f"duplicate JSON key: {key}")
        result[key] = value
    return result


def read_json(root, ref: str):
    return json.loads(resource(root, ref).read_text(encoding="utf-8"), object_pairs_hook=_unique_object)


def pointer(document, value: str):
    if value == "":
        return document
    if not value.startswith("/"):
        raise CandidateValidationError(f"invalid pointer: {value}")
    try:
        for token in value[1:].split("/"):
            token = token.replace("~1", "/").replace("~0", "~")
            document = document[int(token)] if isinstance(document, list) else document[token]
        return document
    except (KeyError, IndexError, TypeError, ValueError) as exc:
        raise CandidateValidationError(f"unresolved pointer: {value}") from exc


def _load_source(root, ref):
    return yaml.safe_load(resource(root, ref).read_text(encoding="utf-8"))


def build_candidate_assets(root=None) -> dict[str, dict]:
    """Retain source values exactly, adding only non-authoritative routing metadata."""
    root = root or files("agent_contracts")
    source_index = _load_source(root, "index.yaml")
    payloads, sources, schemas = {}, {}, {}
    for kind, (schema_kind, identity_field) in KINDS.items():
        payloads[kind], sources[kind] = {}, {}
        schemas[kind] = source_index["schemas"][schema_kind]
        for entry in source_index[kind]:
            identity, ref = entry["id"], entry["ref"]
            if identity in payloads[kind]:
                raise CandidateValidationError(f"duplicate {kind} identity: {identity}")
            payload = read_json(root, ref) if kind == "io_schemas" else _load_source(root, ref)
            if payload[identity_field] != identity:
                raise CandidateValidationError(f"source identity mismatch: {ref}")
            payloads[kind][identity] = payload
            sources[kind][identity] = ref

    failures, primitives, vectors = {}, {}, {}
    for identity, operation in payloads["operations"].items():
        # No inference about which outcome is a success: retain declared failure
        # routes and the entire condition branch table exactly as the source does.
        action_ids = [s["action_ref"] for s in operation["steps"] if s["type"] == "ACTION"]
        gate_ids = [s["mutation_gate_ref"] for s in operation["steps"] if "mutation_gate_ref" in s]
        failures[identity] = {
            "preconditions": operation["preconditions"],
            "actions": {a: {key: payloads["actions"][a][key]
                             for key in ("preconditions", "postconditions", "failure_outcomes")}
                        for a in action_ids},
            "gates": {g: payloads["gates"][g]["requirements"] for g in gate_ids},
            "condition_branches": {s["step_id"]: s["branches"] for s in operation["steps"]
                                   if s["type"] == "CONDITION"},
        }
        primitives[identity] = {
            "primitive_id": identity,
            "input_schema_ref": operation["input_schema_ref"],
            "output_schema_ref": operation["output_schema_ref"],
            "operation_semantics_ref": f"contracts/current.json#/payloads/operations/{identity}",
            "failure_semantics_ref": f"contracts/current.json#/failures/{identity}",
            "conformance_vectors_ref": f"conformance/current.json#/operations/{identity}",
        }
        checks = []
        for key in ("input_schema_ref", "output_schema_ref", "preconditions", "entry_step", "steps", "outcomes"):
            checks.append({"ref": f"/payloads/operations/{identity}/{key}", "expected": operation[key]})
        for a in action_ids:
            for key in ("effect", "preconditions", "postconditions", "failure_outcomes"):
                checks.append({"ref": f"/payloads/actions/{a}/{key}", "expected": payloads["actions"][a][key]})
        for g in gate_ids:
            checks.append({"ref": f"/payloads/gates/{g}", "expected": payloads["gates"][g]})
        vectors[identity] = checks

    # Capability declarations are the existing active identity sets. Python
    # locators, CLI commands and package/resource adapters are not copied here.
    binding = _load_source(root, source_index["binding"]["current_ref"])
    capabilities = {key: value for key, value in binding.items() if key.startswith("active_")}
    rule_index = {}
    for spec_id, spec in payloads["specs"].items():
        for position, rule in enumerate(spec["rules"]):
            if rule["rule_id"] in rule_index:
                raise CandidateValidationError(f"duplicate rule: {rule['rule_id']}")
            rule_index[rule["rule_id"]] = f"/payloads/specs/{spec_id}/rules/{position}"
    return {
        "index.json": {
            "format": "ptsip-machine-contract-candidate-index/v1",
            "status": "CANDIDATE", "normative_authority": False,
            "contract_set_id": source_index["contract_set"]["id"],
            "activation": "REQUIRES_VERIFIED_CUTOVER",
            "canonical_serialization": "UNRESOLVED", "canonical_digest": "UNRESOLVED",
            "schemas": schemas, "migration_sources": sources,
            "capabilities": capabilities, "primitives": primitives, "rule_index": rule_index,
            "contract_ref": "contracts/current.json", "conformance_ref": "conformance/current.json",
        },
        "contracts/current.json": {
            "format": "ptsip-machine-contract-candidate-group/v1",
            "status": "CANDIDATE", "normative_authority": False,
            "payloads": payloads, "failures": failures,
        },
        "conformance/current.json": {
            "format": "ptsip-machine-contract-candidate-vectors/v1",
            "status": "CANDIDATE", "normative_authority": False,
            "purpose": "MIGRATION_PARITY", "operations": vectors,
        },
    }


def validate_candidate_assets(root=None, *, compare_migration_source=True) -> dict[str, int]:
    root = root or files("agent_contracts")
    assets = {ref: read_json(root, ref) for ref in
              ("index.json", "contracts/current.json", "conformance/current.json")}
    for ref, schema in (("index.json", "index"), ("contracts/current.json", "group"),
                        ("conformance/current.json", "vectors")):
        Draft202012Validator(read_json(root, f"schemas/candidate-{schema}.schema.json")).validate(assets[ref])
    index, group, vectors = (assets[r] for r in ("index.json", "contracts/current.json", "conformance/current.json"))
    counts = {}
    for kind, (_, identity_field) in KINDS.items():
        schema = read_json(root, index["schemas"][kind])
        Draft202012Validator.check_schema(schema)
        for identity, payload in group["payloads"][kind].items():
            Draft202012Validator(schema).validate(payload)
            if payload[identity_field] != identity:
                raise CandidateValidationError(f"candidate identity mismatch: {identity}")
        counts[kind] = len(group["payloads"][kind])
    operations = group["payloads"]["operations"]
    if set(index["primitives"]) != set(operations) or set(vectors["operations"]) != set(operations):
        raise CandidateValidationError("primitive/conformance coverage mismatch")
    for identity, primitive in index["primitives"].items():
        if primitive["primitive_id"] != identity:
            raise CandidateValidationError(f"primitive identity mismatch: {identity}")
        for field in ("input_schema_ref", "output_schema_ref"):
            if primitive[field] != operations[identity][field] or primitive[field] not in group["payloads"]["io_schemas"]:
                raise CandidateValidationError(f"unresolved primitive I/O: {identity}")
        for field in ("operation_semantics_ref", "failure_semantics_ref", "conformance_vectors_ref"):
            ref, fragment = primitive[field].split("#", 1)
            pointer(assets[ref], fragment)
        for vector in vectors["operations"][identity]:
            if pointer(group, vector["ref"]) != vector["expected"]:
                raise CandidateValidationError(f"conformance mismatch: {vector['ref']}")
    if compare_migration_source and assets != build_candidate_assets(root):
        raise CandidateValidationError("candidate differs from current migration source; revalidation required")
    return counts


def write_candidate_assets(root: Path) -> None:
    for ref, payload in build_candidate_assets(root).items():
        target = resource(root, ref)
        target.parent.mkdir(parents=True, exist_ok=True)
        # This is reproducible file formatting, not an approved canonical digest.
        target.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
