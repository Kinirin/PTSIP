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

# These fields have exact bindings in the current candidate index schema.
# Materialization copies those decisions; it does not choose digest semantics.
INDEX_METADATA_FIELDS = (
    "canonical_serialization",
    "canonical_serialization_status",
    "canonicalization_ref",
    "canonical_digest_management",
    "canonical_digest_policy_ref",
    "normative_semantic_projection_ref",
    "selected_hash_algorithm",
    "selected_digest_output_encoding",
)


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


def _index_metadata(root):
    schema = read_json(root, "schemas/candidate-index.schema.json")
    properties = schema.get("properties", {})
    if not isinstance(properties, dict):
        raise CandidateValidationError("candidate index metadata bindings must be a mapping")
    metadata = {}
    for field in INDEX_METADATA_FIELDS:
        binding = properties.get(field)
        if not isinstance(binding, dict) or not isinstance(binding.get("const"), str):
            raise CandidateValidationError(f"unresolved required index metadata binding: {field}")
        metadata[field] = binding["const"]
    return metadata


def validate_projection_bindings(root=None) -> dict[str, int]:
    """Validate exact candidate bindings, without hashing or activating authority."""
    root = root or files("agent_contracts")
    try:
        projection = read_json(root, "digests/projection-v1.json")
        policy = read_json(root, "digests/policy-v1.json")
        scope = projection["binding_scope"]
        if (scope["status"] != "BOUND_PENDING_REQUIRED_SEMANTIC_VERIFICATION"
                or projection["unresolved"] != [] or policy["active_scheme"] != "UNRESOLVED"):
            raise CandidateValidationError("unresolved binding or unverified digest scheme activation")
        for document in (projection, policy):
            if document["normative_authority"] is not False or document["status"] != "CANDIDATE":
                raise CandidateValidationError("unverified digest authority activation")
        expected_policy = {
            "projection_id": projection["id"], "status": scope["status"],
            "field_governance": projection["field_governance"],
            "collection_semantics": projection["collection_semantics"],
            "binding_scope": scope,
        }
        if policy["projection_policy"] != expected_policy:
            raise CandidateValidationError("digest/projection binding mismatch")
        asset_refs = scope["asset_refs"]
        if not isinstance(asset_refs, list) or len(asset_refs) != len(set(asset_refs)):
            raise CandidateValidationError("invalid binding asset registry")
        documents = {ref: read_json(root, ref) for ref in asset_refs}
        for ref, schema_kind in (("index.json", "index"), ("contracts/current.json", "group"),
                                 ("conformance/current.json", "vectors")):
            Draft202012Validator(read_json(root, f"schemas/candidate-{schema_kind}.schema.json")).validate(documents[ref])
        payloads = documents["contracts/current.json"]["payloads"]
        if set(payloads) != set(scope["registered_records"]) or set(payloads) != set(KINDS):
            raise CandidateValidationError("unregistered payload kind")
        for kind, records in payloads.items():
            registered = scope["registered_records"][kind]
            if len(registered) != len(set(registered)) or set(records) != set(registered):
                raise CandidateValidationError(f"unregistered payload identity: {kind}")
            identity_field = scope["record_identity_fields"][kind]
            if identity_field != KINDS[kind][1]:
                raise CandidateValidationError(f"identity binding mismatch: {kind}")
            for identity, record in records.items():
                schema = documents[documents["index.json"]["schemas"][kind]]
                Draft202012Validator(schema).validate(record)
                if record[identity_field] != identity:
                    raise CandidateValidationError(f"registry/record identity mismatch: {identity}")
        for vectors in documents["conformance/current.json"]["operations"].values():
            for vector in vectors:
                if pointer(documents["contracts/current.json"], vector["ref"]) != vector["expected"]:
                    raise CandidateValidationError("excluded migration expectation differs from included meaning")

        governance = projection["field_governance"]
        included = governance["exact_inclusion_list"]["exact_role_bindings"]
        excluded = governance["exact_exclusion_list"]["exact_role_bindings"]
        required = set(governance["binding_record_contract"]["required_fields"])
        seen = set()
        for mode, bindings in (("inclusion", included), ("exclusion", excluded)):
            roles = governance[f"exact_{mode}_list"]["exact_semantic_roles"]
            if not isinstance(bindings, list) or (mode == "inclusion" and not bindings):
                raise CandidateValidationError(f"unresolved {mode} bindings")
            for binding in bindings:
                if set(binding) != required or binding["semantic_role"] not in roles:
                    raise CandidateValidationError("invalid field role binding")
                if (binding["extraction_condition"] != "PRESENT_IN_VALIDATED_REGISTERED_ASSET"
                        or binding["required_cardinality"] != {"minimum": 1, "maximum": 1}):
                    raise CandidateValidationError("unsupported extraction/cardinality binding")
                key = binding["asset_ref"], binding["json_pointer"]
                if key in seen or key[0] not in documents:
                    raise CandidateValidationError("duplicate or unregistered field binding")
                pointer(documents[key[0]], key[1])
                seen.add(key)

        def covers(binding, asset, path):
            declared = binding["json_pointer"]
            return binding["asset_ref"] == asset and (
                declared == "" or path == declared or path.startswith(declared + "/")
            )

        for admitted in included:
            for veto in excluded:
                if covers(admitted, veto["asset_ref"], veto["json_pointer"]) or covers(
                    veto, admitted["asset_ref"], admitted["json_pointer"]
                ):
                    raise CandidateValidationError("include/exclude overlap: policy conflict")

        leaves, actual_arrays = [], set()

        def walk(value, asset, path):
            if isinstance(value, dict) and value:
                for key, child in value.items():
                    escaped = key.replace("~", "~0").replace("/", "~1")
                    walk(child, asset, path + "/" + escaped)
            elif isinstance(value, list):
                if not any(covers(b, asset, path) for b in excluded):
                    actual_arrays.add((asset, path))
                if value:
                    for position, child in enumerate(value):
                        walk(child, asset, path + "/" + str(position))
                else:
                    leaves.append((asset, path))
            else:
                leaves.append((asset, path))

        for asset, document in documents.items():
            walk(document, asset, "")
        for asset, path in leaves:
            if not any(covers(b, asset, path) for b in included + excluded):
                raise CandidateValidationError(f"unclassified field: {asset}#{path}")

        collections = projection["collection_semantics"]["exact_collection_bindings"]
        if projection["collection_semantics"]["set_like"]["exception"]["exact_exception_bindings"] != []:
            raise CandidateValidationError("unsupported collection exception binding")
        resolved_arrays = set()
        for binding in collections:
            expected_keys = {"asset_ref", "json_pointer", "collection_type"}
            if "member_identity" in binding:
                expected_keys |= {"member_identity", "member_json_pointer"}
            if binding["collection_type"] == "SET_LIKE":
                expected_keys.add("identity")
            if set(binding) != expected_keys:
                raise CandidateValidationError("undeclared collection binding field")
            asset, path = binding["asset_ref"], binding["json_pointer"]
            value = pointer(documents[asset], path)
            if "member_identity" in binding:
                member = binding["member_identity"]
                if scope["collection_identity_fields"].get(member["record_kind"]) != member["logical_id_pointer"]:
                    raise CandidateValidationError("invalid collection member identity binding")
                parents = [b for b in collections if b["asset_ref"] == asset
                           and b["json_pointer"] == path and "member_identity" not in b]
                if len(parents) != 1 or parents[0].get("identity") != {
                    "record_kind": member["record_kind"],
                    "logical_id_pointer": member["logical_id_pointer"],
                } or parents[0]["collection_type"] != "SET_LIKE":
                    raise CandidateValidationError("unregistered set member selection")
                matches = [(i, item) for i, item in enumerate(value)
                           if pointer(item, member["logical_id_pointer"]) == member["logical_id"]]
                if len(matches) != 1:
                    raise CandidateValidationError("unresolved collection member identity")
                position, value = matches[0]
                member_path = binding["member_json_pointer"]
                value = pointer(value, member_path)
                path += "/" + str(position) + member_path
            if not isinstance(value, list) or (asset, path) in resolved_arrays:
                raise CandidateValidationError("invalid or duplicate collection binding")
            collection_type = binding["collection_type"]
            if collection_type == "SET_LIKE":
                identity = binding["identity"]
                if scope["collection_identity_fields"].get(identity["record_kind"]) != identity["logical_id_pointer"]:
                    raise CandidateValidationError("invalid set-like identity binding")
                identities = [pointer(item, identity["logical_id_pointer"]) for item in value]
                if any(not isinstance(item, str) or not item for item in identities):
                    raise CandidateValidationError("missing set-like logical identity")
                if len(identities) != len(set(identities)):
                    raise CandidateValidationError("duplicate set-like logical identity")
            elif collection_type != "ORDERED":
                raise CandidateValidationError("unclassified collection type")
            resolved_arrays.add((asset, path))
        if resolved_arrays != actual_arrays:
            raise CandidateValidationError("unclassified or stale collection binding")
        return {"included": len(included), "excluded": len(excluded),
                "collections": len(collections), "assets": len(documents)}
    except (OSError, KeyError, TypeError, IndexError) as exc:
        raise CandidateValidationError(f"unresolved required projection binding: {exc}") from exc


def build_candidate_assets(root=None) -> dict[str, dict]:
    """Retain source values exactly, adding only non-authoritative routing metadata."""
    root = root or files("agent_contracts")
    metadata = _index_metadata(root)
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
            **metadata, "canonical_digest": "UNRESOLVED",
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
    validate_projection_bindings(root)
    return counts


def write_candidate_assets(root: Path) -> None:
    for ref, payload in build_candidate_assets(root).items():
        target = resource(root, ref)
        target.parent.mkdir(parents=True, exist_ok=True)
        # This is reproducible file formatting, not an approved canonical digest.
        target.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
