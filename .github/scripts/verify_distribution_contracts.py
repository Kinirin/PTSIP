from __future__ import annotations

import tarfile
import zipfile
from pathlib import Path, PurePosixPath
from typing import Iterable

import yaml


ROOT = Path(__file__).resolve().parents[2]
DIST = ROOT / "dist"


def _load_yaml(path: Path) -> dict[str, object]:
    value = yaml.safe_load(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise SystemExit(f"YAML contract must be a mapping: {path.relative_to(ROOT)}")
    return value


def _single_distribution(pattern: str) -> Path:
    matches = sorted(DIST.glob(pattern))
    if len(matches) != 1:
        raise SystemExit(
            f"Expected exactly one distribution matching {pattern!r}, found {len(matches)}."
        )
    return matches[0]


def _catalog_resources() -> tuple[str, ...]:
    catalog = _load_yaml(ROOT / "profiles" / "index.yaml")
    if catalog.get("root") != "profiles":
        raise SystemExit("Public Profile catalog root must be 'profiles'.")

    rows = catalog.get("profiles")
    if not isinstance(rows, list) or not rows:
        raise SystemExit("Public Profile catalog must contain profile entries.")

    resources: list[str] = []
    for row in rows:
        if not isinstance(row, dict):
            raise SystemExit("Public Profile catalog entries must be mappings.")
        resource = row.get("resource")
        if (
            not isinstance(resource, str)
            or PurePosixPath(resource).name != resource
            or not resource.endswith(".ptsip.yaml")
        ):
            raise SystemExit(
                f"Invalid Public Profile catalog resource identity: {resource!r}"
            )
        resources.append(resource)

    if len(resources) != len(set(resources)):
        raise SystemExit("Public Profile catalog resource identities must be unique.")

    discovered = sorted(
        path.name for path in (ROOT / "profiles").glob("*.ptsip.yaml")
    )
    if sorted(resources) != discovered:
        raise SystemExit(
            "Public Profile catalog does not exactly cover profiles/*.ptsip.yaml."
        )
    return tuple(resources)


def _registry_contracts() -> tuple[str, dict[str, object], tuple[dict[str, object], ...]]:
    canonical_path = ROOT / "registry" / "project-profile-contracts.yaml"
    embedded_path = ROOT / "src" / "ptsip" / "specdata" / "project-profile-contracts.yaml"
    if canonical_path.read_bytes() != embedded_path.read_bytes():
        raise SystemExit(
            "Canonical and embedded Project Profile contract registries differ."
        )

    registry = _load_yaml(canonical_path)
    current = registry.get("current")
    contracts = registry.get("contracts")
    if not isinstance(current, str) or not isinstance(contracts, list):
        raise SystemExit("Project Profile contract registry current/contracts are invalid.")

    normalized = tuple(item for item in contracts if isinstance(item, dict))
    matches = [item for item in normalized if item.get("version") == current]
    if len(matches) != 1 or matches[0].get("lifecycle") != "CURRENT":
        raise SystemExit(
            "Project Profile registry current contract must resolve exactly once."
        )
    return current, matches[0], normalized


def _baseline_pairs(
    contracts: Iterable[dict[str, object]],
) -> tuple[tuple[Path, str], ...]:
    pairs: list[tuple[Path, str]] = []
    for contract in contracts:
        version = contract.get("version")
        baseline = contract.get("baseline")
        if baseline is None:
            continue
        if not isinstance(version, str) or not isinstance(baseline, str):
            raise SystemExit("Project Profile baseline binding is invalid.")
        expected = f"profiles/history/{version}"
        if baseline != expected:
            raise SystemExit(
                f"Project Profile baseline {baseline!r} must equal {expected!r}."
            )
        root = ROOT / baseline
        files = sorted(root.glob("*.ptsip.yaml"))
        if not files:
            raise SystemExit(f"Registered Project Profile baseline is empty: {baseline}")
        for source in files:
            pairs.append(
                (
                    source,
                    f"ptsip/profiles/history/{version}/{source.name}",
                )
            )
    return tuple(pairs)


def _require_members(names: set[str], required: Iterable[str], *, label: str) -> None:
    missing = [name for name in required if name not in names]
    if missing:
        raise SystemExit(f"{label} is missing embedded contracts: {missing}")


def _assert_wheel_bytes(
    archive: zipfile.ZipFile,
    wheel_names: set[str],
    pairs: Iterable[tuple[Path, str]],
) -> None:
    for source, target in pairs:
        if target not in wheel_names:
            continue
        if archive.read(target) != source.read_bytes():
            raise SystemExit(
                "wheel projection differs from canonical source: "
                f"{source.relative_to(ROOT)} -> {target}"
            )


def main() -> int:
    resources = _catalog_resources()
    current, current_contract, contracts = _registry_contracts()

    schema = current_contract.get("schema")
    if not isinstance(schema, str):
        raise SystemExit("Current Project Profile contract has no canonical schema.")
    schema_source = ROOT / schema
    if not schema_source.is_file():
        raise SystemExit(f"Current Project Profile schema is missing: {schema}")

    public_pairs = tuple(
        (ROOT / "profiles" / resource, f"ptsip/profiles/{resource}")
        for resource in resources
    )
    baseline_pairs = _baseline_pairs(contracts)
    canonical_pairs = (
        (ROOT / "profiles" / "index.yaml", "ptsip/profiles/index.yaml"),
        (
            ROOT / "registry" / "project-profile-contracts.yaml",
            "ptsip/specdata/project-profile-contracts.yaml",
        ),
        (
            schema_source,
            f"ptsip/specdata/{schema_source.name}",
        ),
        *public_pairs,
        *baseline_pairs,
    )

    required = (
        "ptsip/specdata/ptsip-profile.schema.json",
        f"ptsip/specdata/{schema_source.name}",
        "ptsip/specdata/project-profile-contracts.yaml",
        "ptsip/specdata/ptsip-normalized-evidence.schema.json",
        "ptsip/specdata/ptsip-registry.yaml",
        "ptsip/profiles/index.yaml",
        *(target for _, target in public_pairs),
        *(target for _, target in baseline_pairs),
    )
    support_required = (
        "ptsip/support/policy/index.yaml",
        "ptsip/support/policy/SFP-0001.yaml",
        "ptsip/support/policy/SFP-0021.yaml",
        "ptsip/support/schemas/ptsip-support-feature-policy.schema.json",
        "ptsip/support/registries/ptsip-support-authority-schema-registry.yaml",
    )

    wheel = _single_distribution("*.whl")
    with zipfile.ZipFile(wheel) as archive:
        wheel_names = set(archive.namelist())
        _require_members(
            wheel_names,
            (*required, *support_required),
            label="wheel",
        )
        _assert_wheel_bytes(archive, wheel_names, canonical_pairs)

    sdist = _single_distribution("*.tar.gz")
    with tarfile.open(sdist, "r:gz") as archive:
        sdist_names = {member.name for member in archive.getmembers()}

    sdist_required = (
        "profiles/index.yaml",
        "registry/project-profile-contracts.yaml",
        "src/ptsip/specdata/project-profile-contracts.yaml",
        *(f"profiles/{resource}" for resource in resources),
        *(
            source.relative_to(ROOT).as_posix()
            for source, _ in baseline_pairs
        ),
        "docs/Support_policy/policy/index.yaml",
        "docs/Support_policy/policy/SFP-0001.yaml",
        "docs/Support_policy/policy/SFP-0021.yaml",
        "docs/Support_policy/policy/schemas/ptsip-support-feature-policy.schema.json",
        "docs/Support_policy/policy/registries/ptsip-support-authority-schema-registry.yaml",
    )
    missing_sdist = [
        required_path
        for required_path in sdist_required
        if not any(path.endswith(f"/{required_path}") for path in sdist_names)
    ]
    if missing_sdist:
        raise SystemExit(f"sdist is missing canonical contracts: {missing_sdist}")

    print("PTSIP distribution contract verification: PASS")
    print(f"Project Profile: {current}")
    print(f"Public Profiles: {len(resources)}")
    print(f"Historical baseline assets: {len(baseline_pairs)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
