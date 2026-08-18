from __future__ import annotations

from pathlib import Path

import yaml

from ptsip.clarification.generator import analyze_clarifications
from ptsip.validation.profile import validate_profile
from vpms.integration.ptsip_bridge import load_ptsip_metadata


REPO_ROOT = Path(__file__).resolve().parents[2]
PROFILE_PATH = REPO_ROOT / "ptsip.yaml"
SPEC_REVISION = "12e2ccd15634ecb3d0a4195b0f61ac3f620e7540"


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
    assert ptsip["version"] == "0.3.6-draft"
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
    classifications = {
        target.component_id: target.classification
        for target in metadata.targets
    }

    assert classifications["ptsip-core"] == "PRODUCT"
    assert classifications["vpms"] == "PRODUCT"
    assert classifications["ptsip-distribution"] == "PRODUCT"
    assert classifications["ptsip-package-assembly"] == "DELIVERY"
    assert classifications["ptsip-embedded-contracts"] == "NEUTRAL_CONTRACT"
    assert classifications["ptsip-canonical-contracts"] == "NEUTRAL_CONTRACT"
    assert classifications["repository-architecture"] == "DEVELOPMENT_TOOLING"
    assert classifications["repository-verification"] == "DEVELOPMENT_TOOLING"
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


def test_vpms_self_adoption_targets_resolve_against_repository_self_profile() -> None:
    metadata = load_ptsip_metadata(PROFILE_PATH)

    distribution = metadata.get_target("ptsip-distribution")
    release_automation = metadata.get_target("repository-release-automation")

    assert distribution is not None
    assert distribution.classification == "PRODUCT"
    assert release_automation is not None
    assert release_automation.classification == "DELIVERY"
