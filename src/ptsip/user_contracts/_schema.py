"""Load the distributed Policy v4 and RBAC schema resources."""

import json
from dataclasses import dataclass
from functools import lru_cache
from importlib.resources import files
from pathlib import PurePosixPath

from jsonschema import Draft202012Validator
from referencing import Registry
from referencing.jsonschema import DRAFT202012

DIALECT = "https://json-schema.org/draft/2020-12/schema"
POLICY_KEY = "ptsip.policy.v4"
VERSION_TARGET = "Policy_Identity.version"
CATEGORY_TARGET = "Policy_Identity.Policy_Category"
_COMPONENTS = (
    "policy/identity/PTSIP_policy_identity.schema.json",
    "policy/lifecycle/PTSIP_policy_lifecycle.schema.json",
    "policy/rules/PTSIP_policy_rules.schema.json",
    "policy/content/PTSIP_policy_content.schema.json",
)


@dataclass(frozen=True)
class Issue:
    code: str
    path: tuple[str | int, ...]
    message: str


def parse_json(text):
    def object_pairs(pairs):
        result = {}
        for key, value in pairs:
            if key in result:
                raise ValueError(f"Duplicate JSON key: {key}")
            result[key] = value
        return result

    def reject_constant(value):
        raise ValueError(f"Invalid JSON constant: {value}")

    return json.loads(text, object_pairs_hook=object_pairs, parse_constant=reject_constant)


@lru_cache(maxsize=1)
def catalog():
    root = files("ptsip").joinpath("schemas")
    config = parse_json(root.joinpath("config.json").read_text(encoding="utf-8"))
    groups = (config["Policy"], config["plan"], config["storage"]["Responsibility-Authority"])
    schemas = {}
    paths = set()
    resources = {}

    def load(path):
        parts = PurePosixPath(path)
        if parts.is_absolute() or ".." in parts.parts or path in paths:
            raise ValueError(f"Invalid or duplicate schema path: {path}")
        paths.add(path)
        schema = parse_json(root.joinpath(*parts.parts).read_text(encoding="utf-8"))
        Draft202012Validator.check_schema(schema)
        uri = schema["$id"]
        if uri in resources:
            raise ValueError(f"Duplicate schema resource ID: {uri}")
        resources[uri] = schema
        return schema

    for group in groups:
        for name, entry in group.items():
            if name in schemas or not entry.get("version"):
                raise ValueError(f"Invalid or duplicate schema registration: {name}")
            schema = load(entry["path"])
            if schema["$id"] != entry["$id"]:
                raise ValueError(f"Schema resource ID mismatch: {name}")
            schemas[name] = schema
    for path in _COMPONENTS:
        load(path)
    registry = Registry().with_resources(
        (uri, DRAFT202012.create_resource(schema)) for uri, schema in resources.items()
    )
    for uri in resources:
        check_references(registry, uri)
    return schemas, resources, registry


def check_references(registry, uri):
    """Check the declared references, including ones an instance might not visit."""
    root = registry[uri]
    pending = [(root, registry.resolver(uri).in_subresource(root))]
    visited = set()
    while pending:
        resource, resolver = pending.pop()
        node = resource.contents
        if not isinstance(node, dict) or id(node) in visited:
            continue
        visited.add(id(node))
        for keyword in ("$ref", "$dynamicRef"):
            if keyword in node:
                target = resolver.lookup(node[keyword])
                if not isinstance(target.contents, (bool, dict)):
                    raise ValueError(f"{keyword} does not point to a schema: {node[keyword]}")
                pending.append((DRAFT202012.create_resource(target.contents), target.resolver))
        pending.extend((child, resolver.in_subresource(child)) for child in resource.subresources())


def structure_issues(name, instance, prefix=()):
    schemas, _, registry = catalog()
    validator = Draft202012Validator(schemas[name], registry=registry)
    return tuple(
        Issue("SCHEMA", (*prefix, *error.absolute_path), error.message)
        for error in validator.iter_errors(instance)
    )


def targets_exist(parts):
    schemas, resources, _ = catalog()
    nodes = [schemas[POLICY_KEY]]
    for key, steps in parts:
        parents = (resources[node["$ref"]] if "$ref" in node else node for node in nodes)
        nodes = [parent["properties"][key] for parent in parents if key in parent.get("properties", {})]
        for _ in range(steps):
            nodes = [
                item
                for node in nodes
                for item in [node.get("items"), *node.get("prefixItems", [])]
                if isinstance(item, dict)
            ]
        if not nodes:
            return False
    return bool(nodes)


def official_version_valid(version):
    schemas, resources, registry = catalog()
    identity_ref = schemas[POLICY_KEY]["properties"]["Policy_Identity"]["$ref"]
    schema = resources[identity_ref]["$defs"]["officialVersion"]
    return Draft202012Validator(schema, registry=registry).is_valid(version)
