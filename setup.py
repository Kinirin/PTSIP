from __future__ import annotations

from pathlib import Path
from shutil import copy2, copytree, rmtree

import yaml
from setuptools import setup
from setuptools.command.build_py import build_py as _build_py


ROOT = Path(__file__).resolve().parent
CANONICAL_SUPPORT_POLICY = ROOT / "docs" / "Support_policy" / "policy"
CANONICAL_PUBLIC_PROFILES = ROOT / "profiles"
CANONICAL_PUBLIC_PROFILE_CATALOG = CANONICAL_PUBLIC_PROFILES / "index.yaml"
CANONICAL_PP_CONTRACT_REGISTRY = ROOT / "registry" / "project-profile-contracts.yaml"
EMBEDDED_PP_CONTRACT_REGISTRY = (
    ROOT / "src" / "ptsip" / "specdata" / "project-profile-contracts.yaml"
)


def _load_yaml(path: Path) -> dict[str, object]:
    value = yaml.safe_load(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise RuntimeError(f"canonical YAML asset must be a mapping: {path}")
    return value


def _project_support_policy(build_lib: Path) -> None:
    source = CANONICAL_SUPPORT_POLICY
    required = (source / "index.yaml", source / "schemas", source / "registries")
    if not all(path.exists() for path in required):
        missing = [str(path) for path in required if not path.exists()]
        raise RuntimeError(f"canonical Support Policy source is incomplete: {missing}")

    target = build_lib / "ptsip" / "support"
    if target.exists():
        rmtree(target)

    policy_target = target / "policy"
    policy_target.mkdir(parents=True, exist_ok=True)
    copy2(source / "index.yaml", policy_target / "index.yaml")

    sfp_files = sorted(source.glob("SFP-*.yaml"))
    if not sfp_files:
        raise RuntimeError("canonical Support Policy source contains no SFP records")
    for path in sfp_files:
        copy2(path, policy_target / path.name)

    copytree(source / "schemas", target / "schemas")
    copytree(source / "registries", target / "registries")


def _registered_public_profile_resources() -> tuple[str, ...]:
    catalog = _load_yaml(CANONICAL_PUBLIC_PROFILE_CATALOG)
    if catalog.get("root") != "profiles":
        raise RuntimeError("public Profile catalog root must be 'profiles'")

    profiles = catalog.get("profiles")
    if not isinstance(profiles, list) or not profiles:
        raise RuntimeError("public Profile catalog must contain registered profiles")

    resources: list[str] = []
    for entry in profiles:
        if not isinstance(entry, dict):
            raise RuntimeError("public Profile catalog entries must be mappings")
        resource = entry.get("resource")
        if (
            not isinstance(resource, str)
            or Path(resource).name != resource
            or not resource.endswith(".ptsip.yaml")
        ):
            raise RuntimeError(
                f"public Profile catalog contains invalid resource identity: {resource!r}"
            )
        resources.append(resource)

    if len(resources) != len(set(resources)):
        raise RuntimeError("public Profile catalog resource identities must be unique")

    discovered = sorted(path.name for path in CANONICAL_PUBLIC_PROFILES.glob("*.ptsip.yaml"))
    if sorted(resources) != discovered:
        raise RuntimeError(
            "public Profile catalog must cover canonical root *.ptsip.yaml exactly"
        )
    return tuple(resources)


def _registered_profile_baselines() -> tuple[tuple[str, Path], ...]:
    registry = _load_yaml(CANONICAL_PP_CONTRACT_REGISTRY)
    if CANONICAL_PP_CONTRACT_REGISTRY.read_bytes() != EMBEDDED_PP_CONTRACT_REGISTRY.read_bytes():
        raise RuntimeError(
            "canonical and embedded Project Profile contract registries differ"
        )

    current = registry.get("current")
    contracts = registry.get("contracts")
    if not isinstance(current, str) or not isinstance(contracts, list):
        raise RuntimeError("Project Profile contract registry current/contracts are invalid")

    current_matches = [
        item
        for item in contracts
        if isinstance(item, dict) and item.get("version") == current
    ]
    if len(current_matches) != 1 or current_matches[0].get("lifecycle") != "CURRENT":
        raise RuntimeError("Project Profile registry current contract must resolve exactly once")

    baselines: list[tuple[str, Path]] = []
    for contract in contracts:
        if not isinstance(contract, dict):
            continue
        version = contract.get("version")
        baseline = contract.get("baseline")
        if baseline is None:
            continue
        if not isinstance(version, str) or not isinstance(baseline, str):
            raise RuntimeError("Project Profile baseline binding is invalid")
        expected = f"profiles/history/{version}"
        if baseline != expected:
            raise RuntimeError(
                f"Project Profile baseline {baseline!r} must equal {expected!r}"
            )
        source = ROOT / baseline
        if not source.is_dir():
            raise RuntimeError(f"registered Project Profile baseline is missing: {baseline}")
        baselines.append((version, source))

    current_baseline = current_matches[0].get("baseline")
    if current_baseline != f"profiles/history/{current}":
        raise RuntimeError("current Project Profile contract must bind its generation baseline")
    return tuple(baselines)


def _project_public_profiles(build_lib: Path) -> None:
    resources = _registered_public_profile_resources()
    baselines = _registered_profile_baselines()

    target = build_lib / "ptsip" / "profiles"
    if target.exists():
        rmtree(target)
    target.mkdir(parents=True, exist_ok=True)

    copy2(CANONICAL_PUBLIC_PROFILE_CATALOG, target / "index.yaml")
    for resource in resources:
        copy2(CANONICAL_PUBLIC_PROFILES / resource, target / resource)

    for version, source in baselines:
        history_target = target / "history" / version
        history_target.mkdir(parents=True, exist_ok=True)
        baseline_resources = sorted(source.glob("*.ptsip.yaml"))
        if not baseline_resources:
            raise RuntimeError(
                f"registered Project Profile baseline contains no profiles: {source}"
            )
        for path in baseline_resources:
            copy2(path, history_target / path.name)

    specdata_target = build_lib / "ptsip" / "specdata"
    specdata_target.mkdir(parents=True, exist_ok=True)
    copy2(
        CANONICAL_PP_CONTRACT_REGISTRY,
        specdata_target / "project-profile-contracts.yaml",
    )


class build_py(_build_py):
    """Project canonical repository assets into the built ptsip package."""

    def run(self) -> None:
        super().run()
        build_lib = Path(self.build_lib)
        _project_support_policy(build_lib)
        _project_public_profiles(build_lib)


setup(cmdclass={"build_py": build_py})
