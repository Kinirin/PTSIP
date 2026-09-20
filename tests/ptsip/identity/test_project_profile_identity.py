from __future__ import annotations

import copy
import json
from pathlib import Path

import pytest
import yaml
from jsonschema import Draft202012Validator

from ptsip.constants import SPEC_VERSION, TOOL_VERSION
from ptsip.project_profile_contracts import current_runtime_project_profile_contract
from ptsip.profile_identity import (
    CURRENT_PROJECT_PROFILE_VERSION,
    PP_0_00,
    PP_1_01,
    ProjectProfileIdentityError,
    ProjectProfileInstanceRevision,
    ProjectProfileUserRevision,
    ProjectProfileOperation,
    ProjectProfileTransitionKind,
    ProjectProfileVersion,
    project_profile_support,
    require_project_profile_support,
)
from ptsip.spec_identity import current_spec_identity


ROOT = Path(__file__).resolve().parents[3]


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
    assert current.schema_resource == "ptsip-profile-pp-1.01.schema.json"
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


def test_spec_identity_exposes_tool_spec_and_pp_as_separate_axes() -> None:
    identity = current_spec_identity()

    assert identity.tool_version == TOOL_VERSION
    assert identity.version == SPEC_VERSION
    assert identity.project_profile_contract_version == CURRENT_PROJECT_PROFILE_VERSION
    assert identity.version != identity.project_profile_contract_version


def test_legacy_and_pp_schemas_are_separate_canonical_embedded_pairs() -> None:
    legacy_public = json.loads(
        (ROOT / "schemas" / "ptsip-profile.schema.json").read_text(encoding="utf-8")
    )
    legacy_embedded = json.loads(
        (ROOT / "src" / "ptsip" / "specdata" / "ptsip-profile.schema.json").read_text(
            encoding="utf-8"
        )
    )
    pp_public = json.loads(
        (ROOT / "schemas" / "ptsip-profile-pp-1.01.schema.json").read_text(encoding="utf-8")
    )
    pp_embedded = json.loads(
        (
            ROOT
            / "src"
            / "ptsip"
            / "specdata"
            / "ptsip-profile-pp-1.01.schema.json"
        ).read_text(encoding="utf-8")
    )

    assert legacy_public == legacy_embedded
    assert pp_public == pp_embedded
    assert legacy_public["properties"]["ptsip"]["properties"]["version"]["const"] == "0.3.6-draft"
    assert pp_public["properties"]["ptsip"]["properties"]["version"]["const"] == "pp.1.01"


def test_pp_1_01_schema_is_identity_only_structural_peer_of_legacy_schema() -> None:
    legacy = json.loads((ROOT / "schemas" / "ptsip-profile.schema.json").read_text(encoding="utf-8"))
    current = json.loads(
        (ROOT / "schemas" / "ptsip-profile-pp-1.01.schema.json").read_text(encoding="utf-8")
    )

    legacy_normalized = copy.deepcopy(legacy)
    current_normalized = copy.deepcopy(current)
    for schema in (legacy_normalized, current_normalized):
        schema.pop("$id", None)
        schema.pop("title", None)
        schema["properties"].pop("ptsip")

    assert legacy_normalized == current_normalized
    legacy_binding = legacy["properties"]["ptsip"]["properties"]
    current_binding = current["properties"]["ptsip"]["properties"]
    assert legacy_binding["version"]["const"] == "0.3.6-draft"
    assert legacy_binding["specification"]["required"] == ["source"]
    assert current_binding["version"]["const"] == "pp.1.01"
    assert current_binding["specification"]["required"] == [
        "family",
        "source",
        "revision",
    ]


def test_pp_1_01_historical_baseline_remains_valid_and_immutable() -> None:
    schema = json.loads(
        (ROOT / "schemas" / "ptsip-profile-pp-1.01.schema.json").read_text(encoding="utf-8")
    )
    validator = Draft202012Validator(schema)

    for resource in (
        "example.ptsip.yaml",
        "hybrid-python-package.ptsip.yaml",
        "template-python-package.ptsip.yaml",
    ):
        profile = yaml.safe_load(
            (ROOT / "profiles" / "history" / "pp.1.01" / resource).read_text(
                encoding="utf-8"
            )
        )
        assert profile["ptsip"]["version"] == "pp.1.01"
        validator.validate(profile)


def test_pp_1_01_release_note_discloses_identity_only_bridge() -> None:
    note = (ROOT / "releasenote" / "project-profile" / "pp.1.01.md").read_text(encoding="utf-8")

    assert "0.3.6-draft" in note
    assert "pp.1.01" in note
    assert "IDENTITY_ONLY" in note
    assert "does **not** need lifecycle redesign" in note


def test_current_pp_identity_is_loaded_from_embedded_registry() -> None:
    runtime = current_runtime_project_profile_contract()

    assert CURRENT_PROJECT_PROFILE_VERSION == runtime.version
    expected_schema = (
        "ptsip-profile-pp-"
        + runtime.version.removeprefix("pp.")
        + ".schema.json"
    )
    assert runtime.schema_resource == expected_schema
    assert runtime.operations == frozenset(
        {"IDENTIFY", "VALIDATE", "ANALYZE", "CREATE_TARGET"}
    )


def test_user_revision_uses_canonical_rev_generation() -> None:
    baseline = ProjectProfileUserRevision.parse("Rev.0001")

    assert baseline.canonical == "Rev.0001"
    assert baseline.next().canonical == "Rev.0002"

    with pytest.raises(ProjectProfileIdentityError) as exc_info:
        ProjectProfileUserRevision.parse("RevX0001")
    assert exc_info.value.code == "PP_USER_REVISION_MALFORMED"
