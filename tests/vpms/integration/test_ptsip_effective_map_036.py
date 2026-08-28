from __future__ import annotations

import ast
import copy
import subprocess
from dataclasses import FrozenInstanceError
from pathlib import Path

import pytest
import yaml

from ptsip.app.store import DecisionStore
from ptsip.constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION
from ptsip.profile_identity import CURRENT_PROJECT_PROFILE_VERSION
from ptsip.validation.profile import validate_profile
from ptsip.validation.templates import template_catalog
from vpms.domain.model import (
    FormulaRef,
    PolicyRef,
    RunnerRef,
    TargetRef,
    VariablesRef,
    VerificationCase,
    VerificationOutcome,
    VerificationPurpose,
)
from vpms.execution.runner import RunnerExecution, run_case
from vpms.integration import ptsip_bridge
from vpms.integration.ptsip_bridge import (
    PtsipMetadataError,
    PtsipMetadataSnapshot,
    load_ptsip_metadata,
    metadata_from_effective_map,
    resolve_target_metadata,
)


_REPO_ROOT = Path(__file__).resolve().parents[3]
_POLICIES = {
    "product_to_nonproduct_runtime_dependency": "deny",
    "nonproduct_in_product_package": "deny",
    "independent_build_resolution": "required",
}


class _PassExecutor:
    def execute(self, case: VerificationCase) -> RunnerExecution:
        return RunnerExecution(outcome=VerificationOutcome.PASS)


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
            "version": CURRENT_PROJECT_PROFILE_VERSION,
            "specification": {
                "family": SPEC_VERSION,
                "source": SPEC_SOURCE,
                "revision": SPEC_REVISION,
            },
        },
        "responsibility_map": responsibility_map,
        "policies": copy.deepcopy(_POLICIES),
    }


def _verification_case(
    *,
    component_id: str,
    purpose: VerificationPurpose,
) -> VerificationCase:
    return VerificationCase(
        id=f"h.vpms.{component_id}",
        purpose=purpose,
        target=TargetRef(component_id=component_id),
        formula=FormulaRef(ref="h.formula"),
        variables=VariablesRef(ref="h.variables"),
        policy=PolicyRef(ref="h.policy"),
        runner=RunnerRef(ref="in-memory.pass"),
    )


def _seed_authority(store: DecisionStore, *, component_id: str) -> dict[str, object]:
    record, stale = store.gate(
        {
            "id": "clr-vpms-h7",
            "repository": "example/project",
            "branch": "main",
            "subject_revision": "abc123",
            "component_id": component_id,
            "request": {
                "id": "clr-vpms-h7",
                "component_id": component_id,
                "status": "INCOMPLETE",
            },
        }
    )
    assert stale == ()
    return record.as_dict()


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


@pytest.mark.parametrize(
    ("classification", "purpose"),
    [
        ("DEVELOPMENT_TOOLING", VerificationPurpose.PRODUCT),
        ("PRODUCT", VerificationPurpose.TOOLCHAIN),
    ],
)
def test_ptsip_classification_does_not_determine_vpms_verification_purpose(
    classification: str,
    purpose: VerificationPurpose,
) -> None:
    snapshot = metadata_from_effective_map(
        {
            "components": [
                {
                    "id": "verifier-sdk",
                    "classification": classification,
                }
            ]
        }
    )
    case = _verification_case(component_id="verifier-sdk", purpose=purpose)

    metadata = resolve_target_metadata(case.target, snapshot)

    assert metadata is not None
    assert metadata.classification == classification
    assert case.purpose is purpose
    assert set(VerificationPurpose) == {
        VerificationPurpose.PRODUCT,
        VerificationPurpose.TOOLCHAIN,
    }


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


def test_hybrid_removal_is_absent_from_vpms_target_metadata(tmp_path: Path) -> None:
    definition = template_catalog()[0]
    repo = _repo(tmp_path)
    _git(repo, "rm", "tests/test_package.py")
    _git(repo, "commit", "-m", "remove test fixture for hybrid removal")

    hybrid_payload = _base_profile(
        "hybrid",
        template_id=definition.id,
        revision=definition.revision,
    )
    responsibility_map = hybrid_payload["responsibility_map"]
    assert isinstance(responsibility_map, dict)
    responsibility_map["overrides"] = {
        "remove_component_ids": ["package-tests"],
        "remove_relationship_ids": ["package-tests-verify-package"],
    }

    snapshot, source_mode = _resolved_snapshot(repo, hybrid_payload)

    assert source_mode == "hybrid"
    assert snapshot.get_target("package") is not None
    assert snapshot.get_target("package-tests") is None


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


def test_effective_metadata_consumption_does_not_mutate_ptsip_source(
    tmp_path: Path,
) -> None:
    definition = template_catalog()[0]
    repo = _repo(tmp_path)
    payload = _base_profile(
        "template",
        template_id=definition.id,
        revision=definition.revision,
    )
    profile = repo / "ptsip.yaml"
    profile.write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8")
    profile_before = profile.read_bytes()

    validation = validate_profile(repo)
    assert validation.valid, validation.errors
    assert validation.resolved_profile is not None
    resolved = validation.resolved_profile

    selected_profile_path_before = validation.profile_path
    source_mode_before = resolved.source_mode
    template_identity_before = (resolved.template_id, resolved.template_revision)
    source_payload_before = copy.deepcopy(resolved.source_payload)
    effective_payload_before = copy.deepcopy(resolved.effective_payload)

    snapshot = metadata_from_effective_map(resolved.effective_payload)
    target_before = resolve_target_metadata(TargetRef(component_id="package"), snapshot)
    assert target_before is not None
    classification_before = target_before.classification

    result = run_case(
        _verification_case(
            component_id="package",
            purpose=VerificationPurpose.PRODUCT,
        ),
        _PassExecutor(),
    )

    target_after = resolve_target_metadata(TargetRef(component_id="package"), snapshot)
    assert result.outcome is VerificationOutcome.PASS
    assert profile.read_bytes() == profile_before
    assert validation.profile_path == selected_profile_path_before
    assert resolved.source_mode == source_mode_before == "template"
    assert (resolved.template_id, resolved.template_revision) == template_identity_before
    assert resolved.source_payload == source_payload_before
    assert resolved.effective_payload == effective_payload_before
    assert target_after == target_before
    assert target_after is not None
    assert target_after.classification == classification_before

    with pytest.raises(FrozenInstanceError):
        snapshot.targets[0].classification = "OPERATIONS"  # type: ignore[misc]


def test_effective_metadata_consumption_does_not_mutate_decision_authority(
    tmp_path: Path,
) -> None:
    snapshot = metadata_from_effective_map(
        {
            "components": [
                {
                    "id": "verifier-sdk",
                    "classification": "DEVELOPMENT_TOOLING",
                }
            ]
        }
    )
    target_before = resolve_target_metadata(TargetRef(component_id="verifier-sdk"), snapshot)
    assert target_before is not None

    store = DecisionStore(tmp_path / "authority.sqlite3")
    authority_before = _seed_authority(store, component_id="verifier-sdk")

    result = run_case(
        _verification_case(
            component_id="verifier-sdk",
            purpose=VerificationPurpose.PRODUCT,
        ),
        _PassExecutor(),
    )

    authority_after = store.get("clr-vpms-h7")
    target_after = resolve_target_metadata(TargetRef(component_id="verifier-sdk"), snapshot)

    assert result.outcome is VerificationOutcome.PASS
    assert result.purpose is VerificationPurpose.PRODUCT
    assert target_after == target_before
    assert target_after is not None
    assert target_after.classification == "DEVELOPMENT_TOOLING"
    assert authority_after is not None
    assert authority_after.as_dict() == authority_before


def test_legacy_raw_bridge_preserves_toolchain_without_translation(tmp_path: Path) -> None:
    profile = tmp_path / "ptsip.yaml"
    profile.write_text(
        """
components:
  - id: verifier-sdk
    classification: TOOLCHAIN
""".strip(),
        encoding="utf-8",
    )
    before = profile.read_bytes()

    snapshot = load_ptsip_metadata(profile)
    target = snapshot.get_target("verifier-sdk")

    assert target is not None
    assert target.classification == "TOOLCHAIN"
    assert target.classification != "DEVELOPMENT_TOOLING"
    assert profile.read_bytes() == before


def test_canonical_036_path_does_not_repair_legacy_toolchain_profile(
    tmp_path: Path,
) -> None:
    repo = _repo(tmp_path)
    payload = _base_profile("explicit")
    payload["components"] = [
        {
            "id": "package",
            "classification": "TOOLCHAIN",
            "include": ["src/**"],
            "purpose": "legacy tooling classification",
        }
    ]
    profile = repo / "ptsip.yaml"
    profile.write_text(yaml.safe_dump(payload, sort_keys=False), encoding="utf-8")
    before = profile.read_bytes()

    validation = validate_profile(repo)

    assert not validation.valid
    assert validation.resolved_profile is None
    with pytest.raises(PtsipMetadataError, match="resolved effective Responsibility Map is required"):
        metadata_from_effective_map(None)
    assert profile.read_bytes() == before


def test_legacy_boundaries_profile_remains_outside_canonical_036_handoff(
    tmp_path: Path,
) -> None:
    repo = _repo(tmp_path)
    profile = repo / "ptsip.yaml"
    profile.write_text(
        """
ptsip:
  version: 0.3.4-draft
boundaries:
  PRODUCT:
    roots: [src]
  TOOLCHAIN:
    roots: [tests]
""".strip(),
        encoding="utf-8",
    )
    before = profile.read_bytes()

    validation = validate_profile(repo)
    compatibility_snapshot = load_ptsip_metadata(profile)

    assert not validation.valid
    assert validation.resolved_profile is None
    assert compatibility_snapshot.targets == ()
    assert profile.read_bytes() == before
