from __future__ import annotations

from pathlib import Path

import yaml

from ptsip.clarification.generator import analyze_clarifications
from ptsip.constants import SPEC_REVISION, SPEC_SOURCE, SPEC_VERSION
from ptsip.profile_identity import CURRENT_PROJECT_PROFILE_VERSION
from ptsip.validation.profile import validate_profile
from vpms.integration.ptsip_bridge import load_ptsip_metadata


REPO_ROOT = Path(__file__).resolve().parents[3]
PROFILE_PATH = REPO_ROOT / "ptsip.yaml"


def _profile() -> dict[str, object]:
    payload = yaml.safe_load(PROFILE_PATH.read_text(encoding="utf-8-sig"))
    assert isinstance(payload, dict)
    return payload


def test_repository_self_profile_is_valid_complete_and_revision_pinned() -> None:
    result = validate_profile(REPO_ROOT)

    assert result.valid, result.errors
    assert result.errors == []
    assert result.warnings == []
    assert result.details is not None

    partition = result.details["component_partition"]
    assert isinstance(partition, dict)
    assert partition["conflict_count"] == 0
    assert partition["unmatched_selectors"] == []

    artifact_partition = result.details["associated_artifact_partition"]
    assert isinstance(artifact_partition, dict)
    assert artifact_partition["conflict_count"] == 0
    assert artifact_partition["unmatched_selectors"] == []

    map_coverage = result.details["responsibility_map_coverage"]
    assert isinstance(map_coverage, dict)
    assert map_coverage["unassigned_count"] == 0

    payload = _profile()
    ptsip = payload["ptsip"]
    assert isinstance(ptsip, dict)
    specification = ptsip["specification"]
    assert isinstance(specification, dict)
    assert ptsip["version"] == CURRENT_PROJECT_PROFILE_VERSION
    assert specification["family"] == SPEC_VERSION
    assert specification["source"] == SPEC_SOURCE
    assert specification["revision"] == SPEC_REVISION
    assert payload["responsibility_map"] == {"mode": "explicit"}


def test_repository_self_profile_resolves_all_discovered_candidates() -> None:
    analysis = analyze_clarifications(REPO_ROOT)

    assert analysis.comparison.stable
    assert analysis.candidate_ids
    assert analysis.status == "NO_CLARIFICATION_REQUIRED"
    assert analysis.requests == ()
    assert analysis.profile_path == str(PROFILE_PATH)
    assert analysis.profile_parse_error is None


def test_repository_self_profile_declares_expected_responsibility_axes() -> None:
    metadata = load_ptsip_metadata(PROFILE_PATH)
    classifications = {target.component_id: target.classification for target in metadata.targets}

    assert classifications["ptsip-core"] == "PRODUCT"
    assert classifications["ptsip-evidence"] == "PRODUCT"
    assert classifications["ptsip-source-compat"] == "PRODUCT"
    assert classifications["ptsip-migration"] == "PRODUCT"
    assert classifications["vpms"] == "PRODUCT"
    assert classifications["ptsip-distribution"] == "PRODUCT"
    assert classifications["ptsip-package-assembly"] == "DELIVERY"
    assert classifications["ptsip-embedded-contracts"] == "NEUTRAL_CONTRACT"
    assert classifications["ptsip-canonical-contracts"] == "NEUTRAL_CONTRACT"
    assert classifications["repository-architecture"] == "DEVELOPMENT_TOOLING"

    assert classifications["ptsip-core-verification"] == "PRODUCT"
    assert classifications["ptsip-evidence-verification"] == "PRODUCT"
    assert classifications["ptsip-source-compat-verification"] == "PRODUCT"
    assert classifications["ptsip-migration-verification"] == "PRODUCT"
    assert classifications["vpms-verification"] == "PRODUCT"
    assert classifications["ptsip-contract-verification"] == "DEVELOPMENT_TOOLING"
    assert classifications["repository-architecture-verification"] == "DEVELOPMENT_TOOLING"
    assert classifications["repository-release-verification"] == "DELIVERY"
    assert classifications["repository-verification-support"] == "DEVELOPMENT_TOOLING"
    assert classifications["repository-test-mode-control-plane"] == "DEVELOPMENT_TOOLING"

    assert classifications["repository-release-automation"] == "DELIVERY"
    assert classifications["repository-ci"] == "DEVELOPMENT_TOOLING"
    assert classifications["repository-maintenance"] == "DEVELOPMENT_TOOLING"

    payload = _profile()
    artifacts = {
        item["id"]: item
        for item in payload.get("associated_artifacts", [])
        if isinstance(item, dict) and item.get("id")
    }
    assert artifacts["ptsip-governance-support"]["anchor"] == "ptsip-canonical-contracts"

    components = {
        item["id"]: item
        for item in payload.get("components", [])
        if isinstance(item, dict) and item.get("id")
    }
    assert components["repository-ci"]["roles"] == ["AUTOMATION"]
    assert components["repository-verification-support"]["roles"] == ["CONFIGURATION"]
    assert components["repository-test-mode-control-plane"]["roles"] == [
        "IMPLEMENTATION",
        "VERIFICATION",
        "CONFIGURATION",
    ]


def test_repository_self_profile_keeps_migration_subsystems_as_explicit_product_components() -> None:
    payload = _profile()
    components = {
        item["id"]: item
        for item in payload.get("components", [])
        if isinstance(item, dict) and item.get("id")
    }

    assert "src/ptsip/**" not in components["ptsip-core"]["include"]
    assert components["ptsip-evidence"]["include"] == ["src/ptsip/evidence/**"]
    assert components["ptsip-source-compat"]["include"] == ["src/ptsip/source_compat/**"]
    assert components["ptsip-migration"]["include"] == ["src/ptsip/migration/**"]

    for component_id in ("ptsip-evidence", "ptsip-source-compat", "ptsip-migration"):
        component = components[component_id]
        assert component["classification"] == "PRODUCT"
        assert component["roles"] == ["IMPLEMENTATION"]
        assert component["shipped"] is True
        assert component["runtime_required"] is False
        assert component["executable"] is False
        assert component["release_owner"] == "tool"
        assert component["compatibility_owner"] == "tool"

    relationships = {
        (item.get("from"), item.get("to"), item.get("type"))
        for item in payload.get("relationships", [])
        if isinstance(item, dict)
    }
    expected = {
        ("ptsip-evidence", "ptsip-core", "IMPORTS"),
        ("ptsip-source-compat", "ptsip-core", "IMPORTS"),
        ("ptsip-source-compat", "ptsip-embedded-contracts", "READS"),
        ("ptsip-migration", "ptsip-core", "IMPORTS"),
        ("ptsip-migration", "ptsip-evidence", "IMPORTS"),
        ("ptsip-migration", "ptsip-source-compat", "IMPORTS"),
        ("ptsip-canonical-contracts", "ptsip-evidence", "SPECIFIES"),
        ("ptsip-canonical-contracts", "ptsip-source-compat", "SPECIFIES"),
        ("ptsip-canonical-contracts", "ptsip-migration", "SPECIFIES"),
        ("ptsip-evidence-verification", "ptsip-evidence", "VERIFIES"),
        ("ptsip-source-compat-verification", "ptsip-source-compat", "VERIFIES"),
        ("ptsip-migration-verification", "ptsip-migration", "VERIFIES"),
    }
    assert expected <= relationships


def test_repository_self_profile_declares_target_oriented_verification_relationships() -> None:
    payload = _profile()
    relationships = {
        (item.get("from"), item.get("to"), item.get("type"))
        for item in payload.get("relationships", [])
        if isinstance(item, dict)
    }

    expected = {
        ("ptsip-core-verification", "ptsip-core", "VERIFIES"),
        ("ptsip-evidence-verification", "ptsip-evidence", "VERIFIES"),
        ("ptsip-source-compat-verification", "ptsip-source-compat", "VERIFIES"),
        ("ptsip-migration-verification", "ptsip-migration", "VERIFIES"),
        ("vpms-verification", "vpms", "VERIFIES"),
        ("ptsip-contract-verification", "ptsip-canonical-contracts", "VERIFIES"),
        ("ptsip-contract-verification", "ptsip-embedded-contracts", "VERIFIES"),
        ("repository-architecture-verification", "repository-architecture", "VERIFIES"),
        ("repository-architecture-verification", "ptsip-governance-support", "VERIFIES"),
        ("repository-release-verification", "ptsip-distribution", "VERIFIES"),
        ("repository-release-verification", "repository-release-automation", "VERIFIES"),
        ("repository-release-verification", "repository-ci", "VERIFIES"),
        ("repository-release-verification", "ptsip-embedded-contracts", "VERIFIES"),
    }
    assert expected <= relationships


def test_repository_test_mode_control_plane_reads_architecture_authority() -> None:
    payload = _profile()
    relationships = {
        (item.get("from"), item.get("to"), item.get("type"))
        for item in payload.get("relationships", [])
        if isinstance(item, dict)
    }
    assert (
        "repository-test-mode-control-plane",
        "repository-architecture",
        "READS",
    ) in relationships


def test_vpms_self_adoption_targets_resolve_against_repository_self_profile() -> None:
    metadata = load_ptsip_metadata(PROFILE_PATH)

    distribution = metadata.get_target("ptsip-distribution")
    release_automation = metadata.get_target("repository-release-automation")

    assert distribution is not None
    assert distribution.classification == "PRODUCT"
    assert release_automation is not None
    assert release_automation.classification == "DELIVERY"
