from __future__ import annotations

from pathlib import Path
from shutil import copy2, copytree, rmtree

import yaml
from setuptools import setup
from setuptools.command.build_py import build_py as _build_py


ROOT = Path(__file__).resolve().parent
CANONICAL_SUPPORT_POLICY = ROOT / "src" / "policy"
CANONICAL_PUBLIC_PROFILES = ROOT / "src" / "ptsip" / "profiles"
CANONICAL_PUBLIC_PROFILE_CATALOG = CANONICAL_PUBLIC_PROFILES / "index.yaml"
CANONICAL_PP_CONTRACT_REGISTRY = ROOT / "registry" / "project-profile-contracts.yaml"
CANONICAL_RBAC_SCHEMA_ROOT = ROOT / ".ptsip" / "contracts" / "schemas" / "compile-sealed-rbac"
CANONICAL_RBAC_FIELDS = ROOT / "registry" / "compile-sealed-rbac-registry-fields.json"
CANONICAL_PP200_CANDIDATE = ROOT / "schemas" / "ptsip-profile-pp-2.00.candidate.schema.json"
RBAC_SCHEMA_FILES = (
    "responsibility.schema.json", "condition-atom.schema.json",
    "permission-atom.schema.json", "binding.schema.json",
    "sealed-contract.schema.json", "activation.schema.json",
)
# Draft validation input: independently packaged, never counted as an approved
# sixth-schema contract or registered as an executable RBAC capability.
RBAC_DRAFT_SCHEMA_FILES = (
    "central-registry.draft.schema.json",
    "responsibility.candidate.schema.json",
    "condition-atom.candidate.schema.json",
    "permission-atom.candidate.schema.json",
    "binding.candidate.schema.json",
    "sealed-contract.candidate.schema.json",
    "activation.candidate.schema.json",
)
RBAC_CANDIDATE_INDEX = "candidate-index.json"
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

    sfp_files = sorted(source.rglob("SFP-*.yaml"))
    if not sfp_files:
        raise RuntimeError("canonical Support Policy source contains no SFP records")
    for path in sfp_files:
        relative = path.relative_to(source)
        destination = policy_target / relative
        destination.parent.mkdir(parents=True, exist_ok=True)
        copy2(path, destination)

    copytree(source / "schemas", target / "schemas")
    copytree(source / "registries", target / "registries")


def _registered_public_profile_resources() -> tuple[str, ...]:
    catalog = _load_yaml(CANONICAL_PUBLIC_PROFILE_CATALOG)
    if catalog.get("root") != "src/ptsip/profiles":
        raise RuntimeError("public Profile catalog root must be 'src/ptsip/profiles'")

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
        expected = f"src/ptsip/profiles/history/{version}"
        if baseline != expected:
            raise RuntimeError(
                f"Project Profile baseline {baseline!r} must equal {expected!r}"
            )
        source = ROOT / baseline
        if not source.is_dir():
            raise RuntimeError(f"registered Project Profile baseline is missing: {baseline}")
        if not any(source.glob("*.ptsip.yaml")):
            raise RuntimeError(f"registered Project Profile baseline is empty: {baseline}")
        baselines.append((version, source))

    current_baseline = current_matches[0].get("baseline")
    if current_baseline != f"src/ptsip/profiles/history/{current}":
        raise RuntimeError("current Project Profile contract must bind its generation baseline")
    return tuple(baselines)


def _project_public_profiles(build_lib: Path) -> None:
    resources = _registered_public_profile_resources()
    _registered_profile_baselines()

    target = build_lib / "ptsip" / "profiles"
    target.mkdir(parents=True, exist_ok=True)
    # Preserve modules copied by build_py. History is source-only, including
    # stale output left by an earlier build that shipped historical baselines.
    history_target = target / "history"
    if history_target.exists():
        if not history_target.resolve().is_relative_to(build_lib.resolve()):
            raise RuntimeError("historical build output escapes build directory")
        rmtree(history_target)
    for name in (
        "profile_identity", "profile_metadata", "profile_compatibility",
        "local_profile_catalog", "project_profile_contracts",
    ):
        retired = target.parent / f"{name}.py"
        if retired.exists():
            if not retired.resolve().is_relative_to(build_lib.resolve()):
                raise RuntimeError("retired profile output escapes build directory")
            retired.unlink()
    for stale in target.glob("*.ptsip.yaml"):
        if stale.name not in resources:
            if not stale.resolve().is_relative_to(build_lib.resolve()):
                raise RuntimeError("stale profile output escapes build directory")
            stale.unlink()

    copy2(CANONICAL_PUBLIC_PROFILE_CATALOG, target / "index.yaml")
    for resource in resources:
        copy2(CANONICAL_PUBLIC_PROFILES / resource, target / resource)

    specdata_target = build_lib / "ptsip" / "specdata"
    specdata_target.mkdir(parents=True, exist_ok=True)
    copy2(
        CANONICAL_PP_CONTRACT_REGISTRY,
        specdata_target / "project-profile-contracts.yaml",
    )


def _project_rbac_contract_assets(build_lib: Path) -> None:
    """Package approved RBAC identities; incomplete schema scaffolds stay fail-closed."""
    embedded_src = ROOT / "src" / "ptsip" / "specdata"
    target = build_lib / "ptsip" / "specdata"
    target.mkdir(parents=True, exist_ok=True)
    packaged_fields = embedded_src / "compile-sealed-rbac-registry-fields.json"
    if not CANONICAL_RBAC_FIELDS.is_file() or not packaged_fields.is_file():
        raise RuntimeError("canonical and embedded RBAC field requirements are missing")
    if CANONICAL_RBAC_FIELDS.read_bytes() != packaged_fields.read_bytes():
        raise RuntimeError("canonical and embedded RBAC field requirements diverged")
    copy2(CANONICAL_RBAC_FIELDS, target / packaged_fields.name)
    candidate = embedded_src / CANONICAL_PP200_CANDIDATE.name
    if not CANONICAL_PP200_CANDIDATE.is_file() or not candidate.is_file():
        raise RuntimeError("pp.2.00 candidate profile schema is missing")
    if CANONICAL_PP200_CANDIDATE.read_bytes() != candidate.read_bytes():
        raise RuntimeError("canonical and embedded pp.2.00 candidate schemas diverged")
    copy2(CANONICAL_PP200_CANDIDATE, target / candidate.name)
    index_source = CANONICAL_RBAC_SCHEMA_ROOT / RBAC_CANDIDATE_INDEX
    index_embedded = embedded_src / "compile-sealed-rbac" / RBAC_CANDIDATE_INDEX
    if not index_source.is_file() or not index_embedded.is_file():
        raise RuntimeError("RBAC candidate index missing")
    if index_source.read_bytes() != index_embedded.read_bytes():
        raise RuntimeError("canonical/embedded RBAC candidate index mismatch")
    index_dest = target / "compile-sealed-rbac" / RBAC_CANDIDATE_INDEX
    index_dest.parent.mkdir(parents=True, exist_ok=True)
    copy2(index_source, index_dest)
    for name in (*RBAC_SCHEMA_FILES, *RBAC_DRAFT_SCHEMA_FILES):
        source = CANONICAL_RBAC_SCHEMA_ROOT / name
        embedded = embedded_src / "compile-sealed-rbac" / name
        if not source.is_file() or not embedded.is_file():
            raise RuntimeError(f"RBAC schema asset missing: {name}")
        if source.read_bytes() != embedded.read_bytes():
            raise RuntimeError(f"canonical/embedded RBAC schema mismatch: {name}")
        destination = target / "compile-sealed-rbac" / name
        destination.parent.mkdir(parents=True, exist_ok=True)
        copy2(source, destination)


class build_py(_build_py):
    """Project canonical repository assets into the built ptsip package."""

    def run(self) -> None:
        super().run()
        build_lib = Path(self.build_lib)
        _project_support_policy(build_lib)
        _project_public_profiles(build_lib)
        _project_rbac_contract_assets(build_lib)


setup(cmdclass={"build_py": build_py})
