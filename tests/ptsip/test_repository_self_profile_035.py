from __future__ import annotations

from pathlib import Path

import yaml

from ptsip.clarification.generator import analyze_clarifications
from ptsip.validation.profile import validate_profile
from vpms.integration.ptsip_bridge import load_ptsip_metadata


REPO_ROOT = Path(__file__).resolve().parents[2]
PROFILE_PATH = REPO_ROOT / "ptsip.yaml"
SPEC_REVISION = "b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e"


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
    assert partition["unassigned_count"] == 0
    assert partition["unmatched_selectors"] == []

    payload = _profile()
    ptsip = payload["ptsip"]
    assert isinstance(ptsip, dict)
    specification = ptsip["specification"]
    assert isinstance(specification, dict)
    assert ptsip["version"] == "0.3.4-draft"
    assert specification["revision"] == SPEC_REVISION


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
    assert classifications["ptsip-embedded-contracts"] == "NEUTRAL_CONTRACT"
    assert classifications["ptsip-canonical-contracts"] == "NEUTRAL_CONTRACT"
    assert classifications["repository-architecture"] == "NEUTRAL_CONTRACT"
    assert classifications["repository-verification"] == "TOOLCHAIN"
    assert classifications["repository-release-automation"] == "TOOLCHAIN"
    assert classifications["repository-ci"] == "TOOLCHAIN"


def test_vpms_self_adoption_targets_resolve_against_repository_self_profile() -> None:
    metadata = load_ptsip_metadata(PROFILE_PATH)

    distribution = metadata.get_target("ptsip-distribution")
    release_automation = metadata.get_target("repository-release-automation")

    assert distribution is not None
    assert distribution.classification == "PRODUCT"
    assert release_automation is not None
    assert release_automation.classification == "TOOLCHAIN"
