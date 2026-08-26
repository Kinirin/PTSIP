from __future__ import annotations

import pytest

from ptsip.profile_identity import (
    PP_0_00,
    PP_1_01,
    ProjectProfileIdentityError,
    ProjectProfileInstanceRevision,
    ProjectProfileOperation,
    ProjectProfileTransitionKind,
    ProjectProfileVersion,
    project_profile_support,
    require_project_profile_support,
)


def test_pp_version_parses_and_serializes_canonically() -> None:
    assert ProjectProfileVersion.parse("pp.1.01").canonical == "pp.1.01"
    assert ProjectProfileVersion.parse("pp.2.100").canonical == "pp.2.100"
    assert ProjectProfileVersion.parse("pp.1.1").canonical == "pp.1.01"


def test_pp_version_rejects_noncanonical_text_when_required() -> None:
    with pytest.raises(ProjectProfileIdentityError) as exc_info:
        ProjectProfileVersion.parse("pp.1.1", require_canonical=True)
    assert exc_info.value.code == "PP_IDENTITY_NON_CANONICAL"


def test_pp_version_ordering_is_numeric_not_lexical() -> None:
    assert ProjectProfileVersion.parse("pp.1.02") < ProjectProfileVersion.parse("pp.1.10")
    assert ProjectProfileVersion.parse("pp.1.10") < ProjectProfileVersion.parse("pp.2.00")


def test_pp_filename_token_is_distinct_from_canonical_identity() -> None:
    version = ProjectProfileVersion.parse("pp.1.01", require_canonical=True)
    assert version.filename_token == "pp1.01"
    assert ProjectProfileVersion.from_filename_token("pp1.01", require_canonical=True) == version


def test_historical_tool_numbered_label_is_not_silently_parsed_as_pp_identity() -> None:
    with pytest.raises(ProjectProfileIdentityError) as exc_info:
        ProjectProfileVersion.parse("0.3.6-draft")
    assert exc_info.value.code == "PP_IDENTITY_MALFORMED"


def test_tool_pp_and_instance_revision_are_independent_types() -> None:
    tool_version = "0.3.7"
    pp_version = ProjectProfileVersion.parse("pp.1.01", require_canonical=True)
    instance_revision = ProjectProfileInstanceRevision.from_content(b"one concrete profile")

    assert tool_version == "0.3.7"
    assert pp_version == PP_1_01
    assert instance_revision.value.startswith("sha256:")
    assert pp_version.canonical not in instance_revision.value


def test_tool_037_support_is_operation_specific() -> None:
    current = project_profile_support("0.3.7", PP_1_01)
    assert current is not None
    assert current.supports(ProjectProfileOperation.IDENTIFY)
    assert current.supports(ProjectProfileOperation.VALIDATE)
    assert current.supports(ProjectProfileOperation.ANALYZE)
    assert current.supports(ProjectProfileOperation.CREATE_TARGET)
    assert not current.supports(ProjectProfileOperation.MIGRATE_SOURCE)

    historical_generation = project_profile_support("0.3.7", PP_0_00)
    assert historical_generation is not None
    assert historical_generation.operations == frozenset({ProjectProfileOperation.IDENTIFY})


def test_unknown_or_unsupported_pp_operation_fails_closed() -> None:
    with pytest.raises(ProjectProfileIdentityError) as exc_info:
        require_project_profile_support(
            "0.3.7",
            "pp.9.99",
            ProjectProfileOperation.VALIDATE,
        )
    assert exc_info.value.code == "PP_IDENTITY_UNSUPPORTED"

    with pytest.raises(ProjectProfileIdentityError) as exc_info:
        require_project_profile_support(
            "0.3.7",
            PP_0_00,
            ProjectProfileOperation.VALIDATE,
        )
    assert exc_info.value.code == "PP_IDENTITY_UNSUPPORTED"


def test_identity_transition_kind_is_not_semantic_migration() -> None:
    assert ProjectProfileTransitionKind.IDENTITY_ONLY != ProjectProfileTransitionKind.SEMANTIC_MIGRATION
