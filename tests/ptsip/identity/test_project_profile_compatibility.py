from __future__ import annotations

import pytest

from ptsip.profile_compatibility import (
    V034_REVISION,
    V036_REVISION,
    current_project_profile_target,
    historical_project_profile_bridge,
    historical_project_profile_bridges,
    require_direct_historical_transition,
    require_historical_project_profile_bridge,
)
from ptsip.profile_identity import (
    PP_0_00,
    PP_1_01,
    ProjectProfileIdentityError,
    ProjectProfileTransitionKind,
    ProjectProfileVersion,
)


def test_historical_bridges_are_explicit_and_evidence_bound() -> None:
    rows = historical_project_profile_bridges()

    assert [(item.declared_version, item.specification_revision) for item in rows] == [
        ("0.3.4-draft", V034_REVISION),
        ("0.3.6-draft", V036_REVISION),
    ]

    legacy = rows[0]
    assert legacy.compatibility_contract == PP_0_00
    assert legacy.target_contract == PP_1_01
    assert legacy.transition_kind is ProjectProfileTransitionKind.SEMANTIC_MIGRATION

    current = rows[1]
    assert current.compatibility_contract == PP_1_01
    assert current.target_contract == PP_1_01
    assert current.transition_kind is ProjectProfileTransitionKind.IDENTITY_ONLY


def test_036_bridge_preserves_legacy_target_alias_without_making_it_a_hop() -> None:
    bridge = require_historical_project_profile_bridge("0.3.6-draft", V036_REVISION)

    assert bridge.legacy_target_filename == "ptsip_0.3.6.yaml"
    assert bridge.direct_target_filename == "ptsip_pp1.01.yaml"
    assert bridge.equivalent_target_filenames == (
        "ptsip_0.3.6.yaml",
        "ptsip_pp1.01.yaml",
    )


def test_current_target_is_pp_identity_not_tool_draft_identity() -> None:
    target = current_project_profile_target()

    assert target.contract == PP_1_01
    assert target.temporary_filename == "ptsip_pp1.01.yaml"
    assert target.schema_resource == "ptsip-profile-pp-1.01.schema.json"


def test_unregistered_old_source_fails_closed_instead_of_inferring_a_route() -> None:
    assert historical_project_profile_bridge("0.2.0-draft", "a" * 40) is None

    with pytest.raises(ProjectProfileIdentityError) as exc_info:
        require_historical_project_profile_bridge("0.2.0-draft", "a" * 40)

    assert exc_info.value.code == "PP_COMPAT_UNSUPPORTED_HISTORICAL_SOURCE"


def test_registered_source_cannot_compose_an_unregistered_future_target_route() -> None:
    with pytest.raises(ProjectProfileIdentityError) as exc_info:
        require_direct_historical_transition(
            "0.3.4-draft",
            V034_REVISION,
            ProjectProfileVersion.parse("pp.2.02", require_canonical=True),
        )

    assert exc_info.value.code == "PP_COMPAT_UNSUPPORTED_DIRECT_TARGET"


def test_direct_transition_returns_the_declared_bridge_only() -> None:
    bridge = require_direct_historical_transition(
        "0.3.4-draft",
        V034_REVISION,
        "pp.1.01",
    )

    assert bridge.target_contract == PP_1_01
    assert bridge.transition_kind is ProjectProfileTransitionKind.SEMANTIC_MIGRATION
