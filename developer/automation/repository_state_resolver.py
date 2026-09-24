from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Mapping

import yaml
from jsonschema import Draft202012Validator

from developer.automation.policy_loader import load_json, load_yaml, repository_root


INDEX = "developer/state/index.yaml"
SCHEMA = "developer/state/repository-state-index.schema.json"


class RepositoryStateResolutionError(RuntimeError):
    pass


def resolve_state(domain: str, root: str | Path | None = None) -> dict[str, object]:
    base = repository_root(root)
    index = load_yaml(INDEX, root=base)
    schema = load_json(SCHEMA, root=base)
    errors = sorted(Draft202012Validator(schema).iter_errors(index), key=lambda e: list(e.path))
    if errors:
        raise RepositoryStateResolutionError("; ".join(error.message for error in errors[:8]))
    domains = index.get("domains")
    if not isinstance(domains, Mapping):
        raise RepositoryStateResolutionError("state domains are missing")
    record = domains.get(domain)
    if not isinstance(record, Mapping):
        raise RepositoryStateResolutionError(f"unknown state domain: {domain}")
    ref = record.get("ref")
    if not isinstance(ref, str) or not (base / ref).is_file():
        raise RepositoryStateResolutionError(f"state ref is unresolved: {ref!r}")
    return {
        "schema_version": "ptsip-repository-state-resolution/v1",
        "domain": domain,
        "ref": ref,
        "role": record.get("role"),
        "authority": False,
    }


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Resolve one bounded repository state owner.")
    parser.add_argument("domain")
    parser.add_argument("--repository", default=".")
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args(argv)
    try:
        result = resolve_state(args.domain, args.repository)
    except (RepositoryStateResolutionError, OSError, ValueError, yaml.YAMLError) as exc:
        print(f"repository-state-resolver: {exc}")
        return 2
    if args.json:
        print(json.dumps(result, ensure_ascii=True, indent=2))
    else:
        print(f"{result['domain']}: {result['ref']} ({result['role']})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
