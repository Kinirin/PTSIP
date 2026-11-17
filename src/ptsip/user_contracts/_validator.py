"""Validate constraints declared by the adopting project."""

import os
import re
import stat
from dataclasses import dataclass
from importlib.resources import files
from pathlib import Path
from tempfile import NamedTemporaryFile
from urllib.parse import urljoin

from jsonschema import Draft202012Validator
from jsonschema.exceptions import SchemaError
from referencing import Registry
from referencing.exceptions import Unresolvable
from referencing.jsonschema import DRAFT202012

from ._schema import CATEGORY_TARGET, DIALECT, VERSION_TARGET, Issue, check_references, parse_json, targets_exist

_BINDINGS = {
    "evaluation_state.json": ("Policy_Rules.evaluation.state",),
    "policy_content_id.json": ("Policy_Content.items[].id", "Policy_Rules.rules[].content.content_id[]"),
    "rules_patterns.json": ("Policy_Rules.rules[].id",),
}
_SEGMENT = re.compile(r"([A-Za-z_]\w*)((?:\[\])*)")
_ANNOTATION = "x-ptsip-target"
_RA_ID_TARGET = "Policy_Rules.rules[].RA_id"
_CATEGORY_ROLES = "x-ptsip-category-roles"
_REQUIRED_ROLES = ("CATEGORY_NAMESPACE", "NAMESPACE_DEFINITION")
_OPTIONAL_ROLES = {"OTHER_NAMESPACE_BOUNDARIES", "CATEGORY_RESPONSIBILITIES", "EXCLUDED_RESPONSIBILITIES"}
_VALUE_ANNOTATIONS = {"title", "description", "default", "deprecated", "examples", "readOnly", "writeOnly"}


@dataclass(frozen=True)
class ContractResult:
    name: str | None
    targets: tuple[str, ...]
    status: str
    issues: tuple[Issue, ...] = ()


def _contract_root(project_root):
    project = Path(project_root).resolve(strict=True)
    if not project.is_dir():
        raise NotADirectoryError(project)
    for ancestor in (project, *project.parents):
        if (ancestor / "src/ptsip/CLI.go").is_file() and (ancestor / "src/ptsip/schemas/config.json").is_file():
            raise ValueError("User contracts require an adopting project outside the PTSIP development checkout")

    root = project
    for part in (".ptsip", "contract", "user"):
        root /= part
        try:
            info = root.lstat()
        except FileNotFoundError:
            continue
        if _is_link(info):
            raise ValueError(f"User contract path must stay in the adopting project: {root}")
        if not stat.S_ISDIR(info.st_mode):
            raise NotADirectoryError(root)
    return root


def _is_link(info):
    tag = getattr(info, "st_reparse_tag", None)
    return stat.S_ISLNK(info.st_mode) or (
        tag is not None and tag == stat.IO_REPARSE_TAG_MOUNT_POINT
    )


def _create_template(path):
    template = files("ptsip.user_contracts").joinpath("templates", path.name).read_bytes()
    path.parent.mkdir(parents=True, exist_ok=True)
    # Publish a complete template while preserving any file another caller created.
    with NamedTemporaryFile(dir=path.parent, delete=False) as staged:
        temporary = Path(staged.name)
        try:
            staged.write(template)
            staged.close()
            os.link(temporary, path)
        except FileExistsError:
            pass
        finally:
            temporary.unlink()


def _parts(target):
    parts = []
    for segment in target.split("."):
        match = _SEGMENT.fullmatch(segment)
        if match is None:
            return ()
        parts.append((match[1], len(match[2]) // 2))
    return tuple(parts)


def _values(instance, parts, path=()):
    if not parts:
        yield path, instance
        return
    key, depth = parts[0]
    if isinstance(instance, dict) and key in instance:
        for child_path, child in _array_values(instance[key], depth, (*path, key)):
            yield from _values(child, parts[1:], child_path)


def _array_values(instance, depth, path):
    if depth == 0:
        yield path, instance
    elif isinstance(instance, list):
        for index, child in enumerate(instance):
            yield from _array_values(child, depth - 1, (*path, index))


def _reported(name, targets, status, code, message):
    return ContractResult(name, targets, status, (Issue(code, (), message),))


def _read(path):
    info = path.lstat()
    if _is_link(info) or not stat.S_ISREG(info.st_mode):
        raise ValueError("Contract must be a regular file under the user contract root")
    return parse_json(path.read_text(encoding="utf-8-sig"))


def _schema_problem(schema):
    unsupported = "UNSUPPORTED", "CONTRACT_DIALECT", "Expected JSON Schema Draft 2020-12"
    if isinstance(schema, dict) and schema.get("$schema", DIALECT) not in (DIALECT, DIALECT + "#"):
        return unsupported
    try:
        Draft202012Validator.check_schema(schema)
        pending = [DRAFT202012.create_resource(schema)]
        while pending:
            resource = pending.pop()
            node = resource.contents
            if isinstance(node, dict) and node.get("$schema", DIALECT) not in (DIALECT, DIALECT + "#"):
                return unsupported
            pending.extend(resource.subresources())
    except (SchemaError, TypeError, OverflowError, RecursionError) as error:
        return "INVALID", "CONTRACT_SCHEMA", getattr(error, "message", str(error))
    return None


def _declares_constraints(schema):
    if isinstance(schema, bool):
        return not schema
    return any(
        key in Draft202012Validator.VALIDATORS and key not in _VALUE_ANNOTATIONS
        for key in schema
    )


def _category_fields(schema):
    """Return the user-specified field names for the required category roles."""
    roles = schema.get(_CATEGORY_ROLES) if isinstance(schema, dict) else None
    required = set(_REQUIRED_ROLES)
    if (
        not isinstance(roles, dict)
        or not required <= roles.keys() <= (required | _OPTIONAL_ROLES)
        or any(not isinstance(field, str) or not field for field in roles.values())
        or len(set(roles.values())) != len(roles)
    ):
        raise ValueError(
            f"{_CATEGORY_ROLES} must bind CATEGORY_NAMESPACE and NAMESPACE_DEFINITION "
            "to distinct user-defined fields"
        )
    return tuple(roles[role] for role in _REQUIRED_ROLES)


def _resource_ids(resource, base_uri):
    declared = resource.id()
    if declared is not None:
        base_uri = urljoin(base_uri, declared)
        yield base_uri
    for child in resource.subresources():
        yield from _resource_ids(child, base_uri)


def validate_user_contracts(policy, project_root=None, *, bootstrap=False):
    """Return the configured constraints and their results for a Policy v4 document.

    Extra contracts bind existing fields using x-ptsip-target. Category contracts
    use x-ptsip-category-roles to associate approved roles with user-owned keys.
    Pass an adopting project root and bootstrap=True to materialize only missing
    standard contracts whose targets are used in this document.
    """
    if bootstrap and project_root is None:
        raise ValueError("Bootstrapping requires an explicit adopting project root")
    root = _contract_root(project_root) if project_root is not None else None
    required = {
        name: targets
        for name, targets in _BINDINGS.items()
        if any(next(_values(policy, _parts(target)), None) is not None for target in targets)
    }
    if bootstrap:
        for name in required:
            path = root / name
            if not path.exists() and not path.is_symlink():
                _create_template(path)

    paths = (
        {path.name: path for path in root.iterdir() if path.name.endswith(".json")}
        if root is not None and root.is_dir() else {}
    )
    names = required.keys() | (paths.keys() - _BINDINGS.keys())
    results = {}
    selected = {}
    resources = {}
    claimed_ids = {}
    version_present = next(_values(policy, _parts(VERSION_TARGET)), None) is not None
    for name in sorted(paths):
        targets = required.get(name, ())
        try:
            schema = _read(paths[name])
        except (OSError, ValueError, RecursionError) as error:
            if name in names:
                results[name] = _reported(name, targets, "INVALID", "CONTRACT_SYNTAX", str(error))
            continue
        annotation = schema.get(_ANNOTATION) if isinstance(schema, dict) else None
        known = name in _BINDINGS
        if known and annotation is not None and annotation not in _BINDINGS[name]:
            if name in required:
                results[name] = _reported(
                    name, targets, "UNBOUND", "CONTRACT_BINDING", "Standard contract target cannot be reassigned"
                )
            continue
        if not known and isinstance(annotation, str):
            parts = _parts(annotation)
            if parts and targets_exist(parts) and annotation != _RA_ID_TARGET:
                targets = (annotation,)
        if targets == (VERSION_TARGET,) and not version_present:
            names.discard(name)
            targets = ()
        problem = _schema_problem(schema)
        if problem is not None:
            if name in names:
                results[name] = _reported(name, targets, *problem)
            continue
        uri = paths[name].as_uri()
        resource = DRAFT202012.create_resource(schema)
        for absolute_id in _resource_ids(resource, uri):
            claimed_ids.setdefault(absolute_id, []).append(name)
        resources[uri] = resource
        if targets:
            selected[name] = (uri, schema, targets)
        elif name in names:
            results[name] = _reported(
                name, (), "UNBOUND", "CONTRACT_BINDING", "Declare an existing policy target with x-ptsip-target"
            )

    # Reject ambiguous resources before resolving any reference.
    for uri, owners in claimed_ids.items():
        if len(owners) > 1 or (uri in resources and paths[owners[0]].as_uri() != uri):
            for name in set(owners):
                resources.pop(paths[name].as_uri(), None)
                entry = selected.pop(name, None)
                if name in names:
                    results[name] = _reported(
                        name, entry[2] if entry else (), "INVALID", "CONTRACT_REFERENCE",
                        f"Ambiguous contract resource ID: {uri}",
                    )
    registry = Registry().with_resources(resources.items()).crawl()
    for name, (uri, schema, targets) in selected.items():
        schema_uri = urljoin(uri, resources[uri].id() or "")
        validator = Draft202012Validator({"$ref": schema_uri}, registry=registry)
        issues = []
        applied = False
        category_fields = ()
        if targets == (CATEGORY_TARGET,) and next(_values(policy, _parts(CATEGORY_TARGET)), None):
            try:
                category_fields = _category_fields(schema)
            except ValueError as error:
                results[name] = _reported(name, targets, "INVALID", "CONTRACT_BINDING", str(error))
                continue
        try:
            for target in targets:
                for path, value in _values(policy, _parts(target)):
                    if not applied:
                        check_references(registry, uri)
                    applied = True
                    issues.extend(
                        Issue("CONTRACT_VALUE", (*path, *error.absolute_path), error.message)
                        for error in validator.iter_errors(value)
                    )
                    for field in category_fields:
                        if not isinstance(value, dict) or field not in value:
                            issues.append(Issue("CATEGORY_ROLE", path, f"Missing category role field: {field!r}"))
        except (Unresolvable, ValueError, RecursionError) as error:
            results[name] = _reported(name, targets, "INVALID", "CONTRACT_REFERENCE", str(error))
            continue
        if issues:
            results[name] = ContractResult(name, targets, "INVALID", tuple(issues))
        else:
            status = "VALID" if applied and (category_fields or _declares_constraints(schema)) else "UNCONFIGURED"
            results[name] = ContractResult(name, targets, status)

    for name, targets in required.items():
        if name not in paths:
            results[name] = ContractResult(name, targets, "UNCONFIGURED")
    if next(_values(policy, _parts(CATEGORY_TARGET)), None) and not any(
        CATEGORY_TARGET in result.targets for result in results.values()
    ):
        results[None] = ContractResult(None, (CATEGORY_TARGET,), "UNCONFIGURED")
    return tuple(results[name] for name in sorted(results, key=lambda name: name or ""))
