from __future__ import annotations

import argparse
import json
import subprocess
from pathlib import Path
from typing import Mapping, Sequence

import yaml

from developer.automation.policy_loader import repository_root
from ptsip.local_profile_catalog import default_catalog_text
from ptsip.profile_identity import ProjectProfileVersion


SOURCE_PP = "pp.1.01"
USER_BASELINE_REVISION = "Rev.0001"
DISTRIBUTED_EXAMPLE = "DISTRIBUTED_EXAMPLE"
PROJECT = "PROJECT"
MATERIALIZATION_REQUIRED = "PROJECT_PATH_RESOLUTION_REQUIRED"

AUTOMATION_OWNED_FORBIDDEN = (
    "registry/project-profile-contracts.yaml",
    "src/ptsip/specdata/project-profile-contracts.yaml",
    "profiles/history/",
    "schemas/ptsip-profile-pp-1.02.schema.json",
    "src/ptsip/specdata/ptsip-profile-pp-1.02.schema.json",
)


class SeedError(RuntimeError):
    pass


def _git(root: Path, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", *args],
        cwd=root,
        check=False,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )


def _require_clean(root: Path) -> None:
    result = _git(root, "status", "--porcelain")
    if result.returncode != 0:
        raise SeedError(result.stderr.strip() or "Unable to inspect repository status.")
    if result.stdout.strip():
        raise SeedError(
            "Working tree/index must be clean before seeding the pp.1.02 transition."
        )


def _load_yaml(path: Path) -> dict[str, object]:
    value = yaml.safe_load(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise SeedError(f"YAML asset must be a mapping: {path}")
    return value


def _dump_yaml(path: Path, payload: Mapping[str, object]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        yaml.safe_dump(dict(payload), sort_keys=False, allow_unicode=True, width=120),
        encoding="utf-8",
        newline="\n",
    )


def _next_minor(value: str) -> str:
    version = ProjectProfileVersion.parse(value, require_canonical=True)
    return ProjectProfileVersion(version.major, version.minor + 1).canonical


def _profile_header(payload: dict[str, object], *, version: str, role: str) -> None:
    ptsip = payload.get("ptsip")
    if not isinstance(ptsip, dict):
        raise SeedError("Profile has no ptsip mapping.")
    specification = ptsip.get("specification")
    if not isinstance(specification, Mapping):
        raise SeedError("Profile has no specification mapping.")
    source = specification.get("source")
    revision = specification.get("revision")
    if not isinstance(source, str) or not isinstance(revision, str):
        raise SeedError("Profile specification source/revision are invalid.")

    ptsip.clear()
    ptsip.update(
        {
            "version": version,
            "revision": USER_BASELINE_REVISION,
            "profile_role": role,
            "specification": {
                "source": source,
                "revision": revision,
            },
        }
    )


def _placeholderize_example(payload: dict[str, object]) -> None:
    replacements = {
        "product-runtime": ["${PRODUCT_RUNTIME_ROOT}/**"],
        "product-sdk": ["${PRODUCT_SDK_ROOT}/**"],
        "product-tests": ["${PRODUCT_TEST_ROOT}/**"],
        "development-toolkit": ["${DEVELOPMENT_TOOLING_ROOT}/**"],
        "release-automation": ["${RELEASE_AUTOMATION_FILE}"],
        "production-operations": ["${OPERATIONS_ROOT}/**"],
        "shared-contracts": ["${CONTRACT_ROOT}/**"],
    }
    for component in payload.get("components", []):
        if not isinstance(component, dict):
            continue
        component_id = component.get("id")
        if component_id in replacements:
            component["include"] = replacements[str(component_id)]
        if component_id == "development-toolkit":
            component["analysis_inputs"] = [
                "${PRODUCT_ROOT}/**",
                "${CONTRACT_ROOT}/**",
            ]

    for artifact in payload.get("associated_artifacts", []):
        if isinstance(artifact, dict) and artifact.get("id") == "development-toolkit-docs":
            artifact["include"] = ["${DEVELOPMENT_TOOLING_DOCS_ROOT}/**"]


def _placeholderize_hybrid(payload: dict[str, object]) -> None:
    responsibility = payload.get("responsibility_map")
    if not isinstance(responsibility, dict):
        return
    overrides = responsibility.get("overrides")
    if not isinstance(overrides, dict):
        return
    for component in overrides.get("components", []):
        if not isinstance(component, dict):
            continue
        if component.get("id") == "package":
            component["include"] = ["${PACKAGE_ROOT}/**"]
        elif component.get("id") == "package-tests":
            component["include"] = ["${PRODUCT_TEST_ROOT}/**"]


def _update_current_schema(path: Path) -> None:
    payload = json.loads(path.read_text(encoding="utf-8"))
    ptsip = payload["properties"]["ptsip"]
    ptsip["required"] = ["version", "revision", "profile_role", "specification"]
    properties = ptsip["properties"]
    properties["revision"] = {
        "type": "string",
        "pattern": r"^Rev\.[0-9]{4}$",
        "description": (
            "User-owned authoritative Project Profile generation within one PP contract. "
            "Developer-distributed baselines begin at Rev.0001."
        ),
    }
    properties["profile_role"] = {
        "enum": ["PROJECT", "DISTRIBUTED_EXAMPLE"],
        "description": (
            "PROJECT is repository authority. DISTRIBUTED_EXAMPLE is non-authoritative "
            "and requires project-path materialization before use."
        ),
    }

    specification = properties["specification"]
    specification["required"] = ["source", "revision"]
    specification["properties"].pop("family", None)
    specification["properties"]["revision"]["description"] = (
        "Immutable Specification Git revision. This identity is independent from "
        "ptsip.revision, which is user-owned Project Profile lineage."
    )
    path.write_text(
        json.dumps(payload, indent=2, ensure_ascii=False) + "\n",
        encoding="utf-8",
        newline="\n",
    )


def _add_ptsip_control_root(payload: dict[str, object]) -> None:
    for component in payload.get("components", []):
        if not isinstance(component, dict):
            continue
        if component.get("id") != "repository-architecture":
            continue
        include = component.get("include")
        if isinstance(include, list) and ".ptsip/**" not in include:
            include.append(".ptsip/**")


def seed(root: str | Path | None = None) -> tuple[str, ...]:
    repo = repository_root(root)
    _require_clean(repo)

    registry = _load_yaml(repo / "registry" / "project-profile-contracts.yaml")
    current = registry.get("current")
    if current != SOURCE_PP:
        raise SeedError(
            f"Seed requires current {SOURCE_PP}, found {current!r}; refusing replay."
        )
    target = _next_minor(SOURCE_PP)
    if target != "pp.1.02":
        raise SeedError(f"Unexpected target identity: {target}")

    catalog_path = repo / "profiles" / "index.yaml"
    catalog = _load_yaml(catalog_path)
    rows = catalog.get("profiles")
    if not isinstance(rows, list) or not rows:
        raise SeedError("Public Profile catalog is empty.")

    changed: list[str] = []

    schema_path = repo / "schemas" / "ptsip-profile-pp-1.01.schema.json"
    _update_current_schema(schema_path)
    changed.append(schema_path.relative_to(repo).as_posix())

    for row in rows:
        if not isinstance(row, dict):
            raise SeedError("Public Profile catalog entry is invalid.")
        resource = row.get("resource")
        if not isinstance(resource, str):
            raise SeedError("Public Profile resource is invalid.")
        profile_path = repo / "profiles" / resource
        profile = _load_yaml(profile_path)
        _profile_header(profile, version=SOURCE_PP, role=DISTRIBUTED_EXAMPLE)
        if row.get("id") == "example":
            _placeholderize_example(profile)
        elif row.get("id") == "hybrid-python-package":
            _placeholderize_hybrid(profile)
        _dump_yaml(profile_path, profile)
        row["profile_role"] = DISTRIBUTED_EXAMPLE
        row["materialization"] = MATERIALIZATION_REQUIRED
        changed.append(profile_path.relative_to(repo).as_posix())

    _dump_yaml(catalog_path, catalog)
    changed.append(catalog_path.relative_to(repo).as_posix())

    root_profile = _load_yaml(repo / "ptsip.yaml")
    _profile_header(root_profile, version=target, role=PROJECT)
    _add_ptsip_control_root(root_profile)
    local_root = repo / ".ptsip" / "profiles"
    local_profile = local_root / "main.ptsip.yaml"
    _dump_yaml(local_profile, root_profile)
    (local_root / "index.yaml").write_text(
        default_catalog_text(),
        encoding="utf-8",
        newline="\n",
    )
    changed.extend(
        [
            local_profile.relative_to(repo).as_posix(),
            (local_root / "index.yaml").relative_to(repo).as_posix(),
        ]
    )

    developer_profile_path = repo / "developer" / "profiles" / "ptsip-repository.yaml"
    developer_profile = _load_yaml(developer_profile_path)
    _profile_header(developer_profile, version=target, role=PROJECT)
    _add_ptsip_control_root(developer_profile)
    _dump_yaml(developer_profile_path, developer_profile)
    changed.append(developer_profile_path.relative_to(repo).as_posix())

    forbidden = [
        path
        for path in changed
        if any(path == item or path.startswith(item) for item in AUTOMATION_OWNED_FORBIDDEN)
    ]
    if forbidden:
        raise SeedError(
            "Seed attempted to mutate automation-owned transition output: "
            + ", ".join(forbidden)
        )

    return tuple(sorted(set(changed)))


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Seed only the approved semantic inputs for the pp.1.02 T2 dry-run."
    )
    parser.add_argument("--repository", default=".")
    parser.add_argument("--apply", action="store_true")
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    if not args.apply:
        print(
            "Refusing to mutate without --apply. This command seeds semantic inputs only; "
            "the pre-commit hook must generate the PP transition outputs."
        )
        return 2
    try:
        changed = seed(args.repository)
    except SeedError as exc:
        print(f"pp.1.02 transition seed: FAIL: {exc}")
        return 2

    print("pp.1.02 transition seed: PASS")
    print("Stage these semantic/source paths only:")
    for path in changed:
        print(path)
    print(
        "Do not manually stage or create registry/project-profile-contracts.yaml, "
        "profiles/history/pp.1.02, pp.1.02 schemas, or embedded PP projections."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
