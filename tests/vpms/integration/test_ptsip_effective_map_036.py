from __future__ import annotations

import ast
import copy
import subprocess
from pathlib import Path

import pytest
import yaml

from ptsip.constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION
from ptsip.validation.profile import validate_profile
from ptsip.validation.templates import template_catalog
from vpms.integration import ptsip_bridge
from vpms.integration.ptsip_bridge import (
    PtsipMetadataError,
    PtsipMetadataSnapshot,
    load_ptsip_metadata,
    metadata_from_effective_map,
)


_REPO_ROOT = Path(__file__).resolve().parents[3]
_POLICIES = {
    "product_to_nonproduct_runtime_dependency": "deny",
    "nonproduct_in_product_package": "deny",
    "independent_build_resolution": "required",
}


def _git(repo: Path, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", "-C", str(repo), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )


def _repo(tmp_path: Path) -> Path:
    repo = tmp_path / "repo"
    repo.mkdir(parents=True)
    _git(repo, "init")
    _git(repo, "config", "user.email", "ptsip-test@example.invalid")
    _git(repo, "config", "user.name", "PTSIP Test")
    files = {
        "src/package.py": "VALUE = 1\n",
        "tests/test_package.py": "def test_value():\n    assert True\n",
    }
    for relative, content in files.items():
        path = repo / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")
    _git(repo, "add", ".")
    _git(repo, "commit", "-m", "fixture")
    return repo


def _base_profile(
    mode: str,
    *,
    template_id: str | None = None,
    revision: str | None = None,
) -> dict[str, object]:
    responsibility_map: dict[str, object] = {"mode": mode}
    if template_id is not None and revision is not None:
        responsibility_map["template"] = {"id": template_id, "revision": revision}
    return {
        "ptsip": {
            "version": SPEC_VERSION,
            "specification": {
                "source": SPEC_SOURCE,
                "revision": SPEC_REVISION,
            },
        },
        "responsibility_map": responsibility_map,
        "policies": copy.deepcopy(_POLICIES),
    }


def _resolved_snapshot(
    repo: Path,
    payload: dict[str, object],
) -> tuple[PtsipMetadataSnapshot, str]:
    profile = repo / "ptsip.yaml"
    profile.write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8")

    validation = validate_profile(repo)

    assert validation.valid, validation.errors
    assert validation.resolved_profile is not None
    resolved = validation.resolved_profile
    return metadata_from_effective_map(resolved.effective_payload), resolved.source_mode


def test_explicit_effective_map_projects_vpms_target_metadata() -> None:
    effective_payload = {
        "components": [
            {
                "id": "z-verifier",
                "classification": "DEVELOPMENT_TOOLING",
                "purpose": "verification implementation",
                "roles": ["VERIFICATION"],
                "shipped": False,
            },
            {
                "id": "a-product",
                "classification": "PRODUCT",
                "runtime_required": True,
            },
        ],
        "associated_artifacts": [],
        "relationships": [],
        "component_dependency_policy": {},
        "policies": {},
    }

    snapshot = metadata_from_effective_map(effective_payload)

    assert snapshot.as_dict() == {
        "targets": [
            {"component_id": "a-product", "classification": "PRODUCT"},
            {
                "component_id": "z-verifier",
                "classification": "DEVELOPMENT_TOOLING",
            },
        ]
    }


def test_vpms_effective_map_bridge_does_not_import_ptsip_runtime() -> None:
    bridge_path = _REPO_ROOT / "src" / "vpms" / "integration" / "ptsip_bridge.py"
    tree = ast.parse(bridge_path.read_text(encoding="utf-8"), filename=str(bridge_path))
    imports: list[str] = []

    for node in ast.walk(tree):
        if isinstance(node, ast.Import):
            imports.extend(alias.name for alias in node.names)
        elif isinstance(node, ast.ImportFrom) and node.module:
            imports.append(node.module)

    assert not any(name == "ptsip" or name.startswith("ptsip.") for name in imports)


def test_effective_map_projection_exposes_only_narrow_vpms_metadata_contract() -> None:
    snapshot = metadata_from_effective_map(
        {
            "components": [
                {
                    "id": "verifier-sdk",
                    "classification": "DEVELOPMENT_TOOLING",
                    "purpose": "ignored",
                    "roles": ["VERIFICATION", "AUTOMATION"],
                    "include": ["src/vpms/**"],
                    "shipped": False,
                    "runtime_required": False,
                    "executable": True,
                    "release_owner": "ignored-owner",
                    "compatibility_owner": "ignored-compatibility",
                }
            ],
            "associated_artifacts": [
                {
                    "id": "vpms-docs",
                    "anchor": "verifier-sdk",
                    "include": ["docs/**"],
                    "purpose": "ignored",
                }
            ],
            "relationships": [],
            "component_dependency_policy": {"default": "deny"},
            "policies": {"ignored": True},
        }
    )

    assert snapshot.as_dict() == {
        "targets": [
            {
                "component_id": "verifier-sdk",
                "classification": "DEVELOPMENT_TOOLING",
            }
        ]
    }
    assert set(snapshot.as_dict()["targets"][0]) == {"component_id", "classification"}


def test_template_effective_map_projects_same_vpms_target_metadata(tmp_path: Path) -> None:
    definition = template_catalog()[0]
    repo = _repo(tmp_path)

    explicit_payload = _base_profile("explicit")
    explicit_payload.update(copy.deepcopy(definition.map_payload))
    explicit_snapshot, explicit_mode = _resolved_snapshot(repo, explicit_payload)

    template_payload = _base_profile(
        "template",
        template_id=definition.id,
        revision=definition.revision,
    )
    template_snapshot, template_mode = _resolved_snapshot(repo, template_payload)

    assert explicit_mode == "explicit"
    assert template_mode == "template"
    assert template_snapshot == explicit_snapshot


def test_hybrid_effective_map_projects_effective_override_metadata(tmp_path: Path) -> None:
    definition = template_catalog()[0]
    repo = _repo(tmp_path)
    template_components = copy.deepcopy(definition.map_payload["components"])
    assert isinstance(template_components, list)

    package_tests = next(
        component
        for component in template_components
        if isinstance(component, dict) and component.get("id") == "package-tests"
    )
    package_tests["classification"] = "OPERATIONS"
    package_tests["purpose"] = "project-owned verification operations"
    package_tests["shipped"] = False
    package_tests["runtime_required"] = False

    explicit_map = copy.deepcopy(definition.map_payload)
    explicit_components = explicit_map["components"]
    assert isinstance(explicit_components, list)
    explicit_map["components"] = [
        copy.deepcopy(package_tests)
        if isinstance(component, dict) and component.get("id") == "package-tests"
        else component
        for component in explicit_components
    ]
    explicit_payload = _base_profile("explicit")
    explicit_payload.update(explicit_map)
    explicit_snapshot, explicit_mode = _resolved_snapshot(repo, explicit_payload)

    hybrid_payload = _base_profile(
        "hybrid",
        template_id=definition.id,
        revision=definition.revision,
    )
    responsibility_map = hybrid_payload["responsibility_map"]
    assert isinstance(responsibility_map, dict)
    responsibility_map["overrides"] = {"components": [copy.deepcopy(package_tests)]}
    hybrid_snapshot, hybrid_mode = _resolved_snapshot(repo, hybrid_payload)

    assert explicit_mode == "explicit"
    assert hybrid_mode == "hybrid"
    assert hybrid_snapshot == explicit_snapshot
    projected = hybrid_snapshot.get_target("package-tests")
    assert projected is not None
    assert projected.classification == "OPERATIONS"


def test_invalid_profile_produces_no_vpms_metadata_snapshot(tmp_path: Path) -> None:
    definition = template_catalog()[0]
    repo = _repo(tmp_path)
    payload = _base_profile(
        "template",
        template_id=definition.id,
        revision="sha256:" + "0" * 64,
    )
    profile = repo / "ptsip.yaml"
    profile.write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8")

    validation = validate_profile(repo)

    assert not validation.valid
    assert validation.resolved_profile is None
    effective_payload = (
        validation.resolved_profile.effective_payload
        if validation.resolved_profile is not None
        else None
    )
    with pytest.raises(
        PtsipMetadataError,
        match="resolved effective Responsibility Map is required",
    ):
        metadata_from_effective_map(effective_payload)


def test_vpms_metadata_projection_does_not_fallback_to_raw_profile(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    repo = _repo(tmp_path)
    payload = _base_profile("explicit")
    payload["components"] = [
        {
            "id": "package",
            "classification": "PRODUCT",
            "include": ["src/**"],
            "purpose": "product package",
            "unexpected_contract_field": True,
        }
    ]
    profile = repo / "ptsip.yaml"
    profile.write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8")

    validation = validate_profile(repo)
    assert not validation.valid
    assert validation.resolved_profile is None

    legacy_snapshot = load_ptsip_metadata(profile)
    assert legacy_snapshot.get_target("package") is not None

    fallback_calls: list[Path] = []

    def _forbidden_raw_fallback(profile_path: str | Path) -> PtsipMetadataSnapshot:
        fallback_calls.append(Path(profile_path))
        return legacy_snapshot

    monkeypatch.setattr(ptsip_bridge, "load_ptsip_metadata", _forbidden_raw_fallback)

    with pytest.raises(
        PtsipMetadataError,
        match="does not fall back to a raw project profile",
    ):
        ptsip_bridge.metadata_from_effective_map(None)

    assert fallback_calls == []


def test_invalid_effective_map_does_not_return_partial_target_list() -> None:
    with pytest.raises(PtsipMetadataError, match="component at index 1 must be a mapping"):
        metadata_from_effective_map(
            {
                "components": [
                    {"id": "valid-product", "classification": "PRODUCT"},
                    "invalid-partial-row",
                ]
            }
        )
