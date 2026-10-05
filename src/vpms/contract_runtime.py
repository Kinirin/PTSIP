"""Bounded shipped-contract reads; inspection never grants runtime activation."""
from __future__ import annotations

from functools import cache
from importlib.resources import files
import json
from pathlib import PurePosixPath
from types import MappingProxyType

from jsonschema import Draft202012Validator
from referencing import Registry, Resource


class ContractUnavailable(ValueError):
    """A dependent operation has no usable, active product contract."""


_RESPONSIBILITIES = {
    "protocol": "VPMS_PRODUCT_PROTOCOL",
    "selection": "VPMS_CASE_SELECTION",
    "execution_composition": "VPMS_EXECUTION_COMPOSITION",
}
_SCHEMAS = (
    "schemas/catalog.schema.json", "schemas/product-contract.schema.json",
    "schemas/selection-request.schema.json", "schemas/selection-result.schema.json",
    "schemas/selection-rule.schema.json",
)


def _unique_pairs(items):
    result = {}
    for key, value in items:
        if key in result:
            raise ContractUnavailable("DUPLICATE_CONTRACT_KEY")
        result[key] = value
    return result


def _read(relative: str):
    path = PurePosixPath(relative)
    if (not relative or path.is_absolute() or ".." in path.parts
            or "\\" in relative or ":" in relative):
        raise ContractUnavailable("UNSAFE_CONTRACT_PATH")
    try:
        resource = files("vpms").joinpath("contracts", *path.parts)
        return json.loads(resource.read_text(encoding="utf-8"), object_pairs_hook=_unique_pairs)
    except (OSError, ValueError) as exc:
        raise ContractUnavailable("INVALID_CONTRACT_RESOURCE") from exc


def _freeze(value):
    if isinstance(value, dict):
        return MappingProxyType({key: _freeze(item) for key, item in value.items()})
    if isinstance(value, list):
        return tuple(_freeze(item) for item in value)
    return value


@cache
def _validators():
    # Cache shape resources only, not lifecycle state or business contract data.
    identity = _read("identity-registry.json")
    if (identity.get("format") != "vpms-product-identity-registry/v1"
            or tuple(identity.get("schema_resources", ())) != _SCHEMAS):
        raise ContractUnavailable("UNSUPPORTED_IDENTITY_REGISTRY")
    try:
        registry = Registry().with_resource(identity["$id"], Resource.from_contents(identity))
        schemas = {}
        for relative in _SCHEMAS:
            schema = _read(relative)
            Draft202012Validator.check_schema(schema)
            registry = registry.with_resource(schema["$id"], Resource.from_contents(schema))
            schemas[relative] = schema
        return {
            relative: Draft202012Validator(schema, registry=registry)
            for relative, schema in schemas.items()
        }
    except Exception as exc:
        raise ContractUnavailable("INVALID_CONTRACT_SCHEMA") from exc


def document_errors(kind: str, payload):
    if kind not in {"request", "rule", "result"}:
        raise ContractUnavailable("UNKNOWN_SELECTION_DOCUMENT")
    validator = _validators()[f"schemas/selection-{kind}.schema.json"]
    try:
        return tuple(validator.iter_errors(payload))
    except Exception as exc:
        raise ContractUnavailable("UNRESOLVED_CONTRACT_SCHEMA") from exc


def load_registered_contract(role: str):
    """Inspect an exact registered contract; APPROVED is not execution authority."""
    if role not in _RESPONSIBILITIES:
        raise ContractUnavailable("UNKNOWN_CONTRACT_ROLE")
    try:
        catalog = _read("index.json")
        validators = _validators()
        validators["schemas/catalog.schema.json"].validate(catalog)
        identity = catalog["entrypoints"][role]
        entry = catalog["contracts"][identity]
        contract = _read(entry["path"])
        validators["schemas/product-contract.schema.json"].validate(contract)
        for field in ("status", "responsibility", "runtime_enabled"):
            if contract[field] != entry[field]:
                raise ContractUnavailable("CONTRACT_INDEX_MISMATCH")
        if (contract["id"] != identity
                or contract["contract_class"] != catalog["contract_class"]
                or contract["responsibility"] != _RESPONSIBILITIES[role]):
            raise ContractUnavailable("CONTRACT_IDENTITY_MISMATCH")
        return _freeze(contract)
    except ContractUnavailable:
        raise
    except Exception as exc:
        raise ContractUnavailable("INVALID_REGISTERED_CONTRACT") from exc


def require_active_contract(role: str):
    contract = load_registered_contract(role)
    catalog = _read("index.json")
    if (catalog["capability"] != "ACTIVE"
            or contract["status"] != "ACTIVE" or not contract["runtime_enabled"]):
        raise ContractUnavailable("CONTRACT_NOT_ACTIVE")
    return contract
