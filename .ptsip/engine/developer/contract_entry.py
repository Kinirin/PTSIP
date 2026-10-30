"""Temporary, read-only AGENTS.md contract entry adapter (no policy authority)."""

from __future__ import annotations

import argparse
import json
import os
from pathlib import Path, PurePosixPath
import subprocess
import sys


class RoutingError(Exception):
    pass


def _unique_pairs(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise RoutingError(f"DUPLICATE_JSON_KEY: {key}")
        result[key] = value
    return result


def _path(root: Path, relative: str) -> Path:
    if not isinstance(relative, str) or not relative or "\\" in relative or ":" in relative:
        raise RoutingError("INVALID_REGISTERED_PATH")
    path = PurePosixPath(relative)
    if path.is_absolute() or any(p in ("", ".", "..") for p in relative.split("/")):
        raise RoutingError("INVALID_REGISTERED_PATH")
    absolute = root.joinpath(*path.parts)
    # Canonical directory stays inside the caller-selected repository, including symlinks.
    if not absolute.resolve().is_relative_to(root.resolve()):
        raise RoutingError("PATH_ESCAPES_REPOSITORY")
    if not absolute.is_file():
        raise RoutingError(f"MISSING_REGISTERED_FILE: {relative}")
    return absolute


def _json(root: Path, relative: str) -> dict:
    try:
        result = json.loads(_path(root, relative).read_text(encoding="utf-8-sig"), object_pairs_hook=_unique_pairs)
    except (OSError, UnicodeError, ValueError) as exc:
        raise RoutingError(f"INVALID_JSON_CONTRACT: {relative}: {exc}") from exc
    if not isinstance(result, dict):
        raise RoutingError(f"INVALID_JSON_CONTRACT: {relative}")
    return result


def _yaml(root: Path, relative: str) -> dict:
    # Reuse PyYAML, already required by the existing Go developer verifier.
    try:
        import yaml
    except ImportError as exc:
        raise RoutingError("YAML_READER_UNAVAILABLE: install PyYAML for the existing developer verifier") from exc

    class UniqueLoader(yaml.SafeLoader):
        pass

    def mapping(loader, node):
        result = {}
        for key_node, value_node in node.value:
            key = loader.construct_object(key_node)
            if not isinstance(key, str) or key in result:
                raise RoutingError("INVALID_OR_DUPLICATE_YAML_KEY")
            result[key] = loader.construct_object(value_node)
        return result

    UniqueLoader.add_constructor(yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG, mapping)
    try:
        data = yaml.load(_path(root, relative).read_text(encoding="utf-8-sig"), Loader=UniqueLoader)
    except yaml.YAMLError as exc:
        raise RoutingError(f"INVALID_TARGET_DOCUMENT: {exc}") from exc
    if not isinstance(data, dict):
        raise RoutingError("TARGET_DOCUMENT_MUST_BE_OBJECT")
    return data


def _lookup(obj: dict, *keys):
    for key in keys:
        if not isinstance(obj, dict) or key not in obj:
            raise RoutingError(f"MISSING_CONTRACT_BINDING: {'.'.join(keys)}")
        obj = obj[key]
    return obj


def _within(path: str, directory: str) -> bool:
    return path.startswith(directory.rstrip("/") + "/")


def resolve(root: Path, target: str) -> dict:
    """Exact path-based routing only; no filename/family inference or mutation."""
    _path(root, target)
    repository = _json(root, ".ptsip/index.json")
    ns = _lookup(repository, "namespaces", "contracts")
    if _lookup(ns, "status") != "ACTIVE":
        raise RoutingError("CONTRACT_NAMESPACE_NOT_ACTIVE")
    catalog_path = ".ptsip/" + _lookup(ns, "index")
    if catalog_path != ".ptsip/contracts/index.json":
        raise RoutingError("UNEXPECTED_CONTRACT_ENTRY")
    catalog = _json(root, catalog_path)
    engine_path = _lookup(catalog, "authoring", "entrypoint")
    engine = _json(root, engine_path + "/index.json")
    if engine.get("policy_authority") != "NONE" or engine.get("execution_authorized") is not False:
        raise RoutingError("TEMPORARY_ENGINE_AUTHORITY_BOUNDARY_CHANGED")
    if _lookup(catalog, "authoring", "execution_authorized") is not False:
        raise RoutingError("CONTRACT_EXECUTION_BOUNDARY_CHANGED")
    bindings = _lookup(catalog, "authoring", "validators")
    if not isinstance(bindings, list):
        raise RoutingError("INVALID_AUTHORING_VALIDATORS")
    registered = _lookup(engine, "validators")
    if not isinstance(registered, dict):
        raise RoutingError("INVALID_ENGINE_REGISTRY")

    policy_path = ".ptsip/contracts/" + _lookup(catalog, "policy")
    boundary = _json(root, policy_path)
    routes = _json(root, _lookup(boundary, "routing_binding", "data"))
    route_map = _lookup(routes, "routes")
    matches = []

    for kind, definition in route_map.items():
        output_root = _lookup(definition, "output_root")
        if not _within(target, output_root):
            continue
        doc = _yaml(root, target)
        identity = _lookup(doc, "Policy_Identity")
        if _lookup(identity, "policy_kind") != kind:
            raise RoutingError("POLICY_KIND_DOES_NOT_MATCH_ROUTE")
        family = _lookup(identity, "root_family")
        family_path = _lookup(boundary, "rules_binding", "directory") + "/" + family + "_rules.json"
        allowed_families = _lookup(_json(root, _lookup(boundary, "rules_binding", "family_source")), "enum")
        allowed_kinds = _lookup(_json(root, _lookup(routes, "identity_sources", "policy_kind")), "enum")
        if kind not in allowed_kinds or family not in allowed_families:
            raise RoutingError("UNREGISTERED_POLICY_IDENTITY")
        family_contract = _json(root, family_path)
        if family_contract.get("root_family") != family:
            raise RoutingError("FAMILY_CONTRACT_IDENTITY_MISMATCH")
        # Full document validity is deliberately delegated to the existing verifier.
        matches.append({"domain": "policy", "validator": "policy-check", "target": target,
                        "policy_kind": kind, "root_family": family,
                        "contract": policy_path, "family_contract": family_path,
                        "document_status": doc.get("Policy_Status")})

    responsibility_target = _lookup(boundary, "rules_binding", "responsibility_profile")
    if target == responsibility_target:
        _json(root, ".ptsip/contracts/" + _lookup(catalog, "responsibility_authority"))
        matches.append({"domain": "responsibility_authority", "validator": "responsibility-authority",
                        "target": target, "contract": _lookup(catalog, "responsibility_authority")})

    release_note = _json(root, _lookup(catalog, "release_documentation"))
    if target == _lookup(release_note, "canonical_readme", "path"):
        matches.append({"domain": "release_documentation", "validator": "release-documentation",
                        "target": target, "contract": _lookup(catalog, "release_documentation")})

    release_path = _lookup(catalog, "release")
    release = _json(root, release_path)
    # The release domain is selected exclusively by an exact declared contract,
    # never by file extension or a broad release directory prefix.
    release_contracts = {_lookup(release, "contract_ref"), _lookup(release, "validation_rules_ref")}
    for platform in _lookup(release, "platforms"):
        release_contracts.update(_lookup(platform, "contracts").values())
        release_contracts.add(_lookup(platform, "validation_ref"))
    if target in release_contracts:
        matches.append({"domain": "release", "validator": _lookup(release, "validator_name"),
                        "target": target, "contract": release_path})

    if len(matches) != 1:
        raise RoutingError("UNREGISTERED_OR_AMBIGUOUS_CONTRACT_TARGET")
    selected = matches[0]
    validator = selected["validator"]
    if validator not in registered:
        raise RoutingError("UNREGISTERED_EXECUTABLE_VALIDATOR")
    if (selected["domain"] == "policy" and "policy-rules" not in bindings) or (
            selected["domain"] == "responsibility_authority" and validator not in bindings):
        raise RoutingError("VALIDATOR_NOT_ADMITTED_BY_AUTHORING_CATALOG")
    args = ["go", "-C", engine_path, "run", ".", "--root", "../../..", "--validator", validator]
    if selected["domain"] in ("policy", "responsibility_authority"):
        args += ["--operation", "check", "--input", target]
    elif selected["domain"] == "release_documentation":
        args += ["--operation", "check"]
    selected["argv"] = args
    selected["execution_authorized"] = False
    selected["mutation_authorized"] = False
    selected["policy_conformance"] = "NOT_ASSERTED"
    return selected


def run(root: Path, target: str, *, dry_run: bool = False, runner=subprocess.run):
    try:
        selected = resolve(root, target)
    except (RoutingError, TypeError, KeyError) as exc:
        return {"status": "UNRESOLVED", "problem": str(exc), "execution_authorized": False}, 2
    if dry_run:
        return {"status": "SELECTED_NOT_EXECUTED", **selected}, 0
    try:
        result = runner(selected["argv"], cwd=str(root), capture_output=True, text=True, check=False)
    except (OSError, ValueError) as exc:
        return {"status": "INVOCATION_FAILED", "problem": str(exc), **selected}, 2
    try:
        output = json.loads(result.stdout)
    except (ValueError, TypeError):
        output = {"raw_stdout": result.stdout[-4000:]}
    status = "CHECK_COMPLETED" if result.returncode == 0 else "CHECK_UNVERIFIED" if result.returncode == 4 else "CHECK_FAILED"
    return {"status": status, "exit_code": result.returncode, "result": output,
            "stderr": result.stderr[-4000:], **selected}, result.returncode


def main(argv=None):
    parser = argparse.ArgumentParser(description="Temporary registry-driven, non-mutating contract entry")
    parser.add_argument("--root", default=".", help="repository root")
    parser.add_argument("--target", required=True, help="explicit repository-relative canonical file path")
    parser.add_argument("--dry-run", action="store_true")
    opts = parser.parse_args(argv)
    report, code = run(Path(opts.root).resolve(), opts.target, dry_run=opts.dry_run)
    print(json.dumps(report, ensure_ascii=False, indent=2, default=str))
    return code


if __name__ == "__main__":
    sys.exit(main())
