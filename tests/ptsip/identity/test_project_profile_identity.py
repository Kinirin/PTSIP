from __future__ import annotations

import json
from pathlib import Path

import pytest
import yaml
from jsonschema import Draft202012Validator

from ptsip.constants import SPEC_VERSION, TOOL_VERSION
from ptsip.profile_identity import (
    CURRENT_PROJECT_PROFILE_VERSION,
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
    assert current.schema_resource == "ptsip-profile.schema.json"
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
    assert identity.project_profile_contract_version == "pp.1.01"
    assert identity.version != identity.project_profile_contract_version


def test_public_and_embedded_profile_schemas_are_identical_and_accept_identity_bridge() -> None:
    public_path = ROOT / "schemas" / "ptsip-profile.schema.json"
    embedded_path = ROOT / "src" / "ptsip" / "specdata" / "ptsip-profile.schema.json"
    public_schema = json.loads(public_path.read_text(encoding="utf-8"))
    embedded_schema = json.loads(embedded_path.read_text(encoding="utf-8"))

    assert public_schema == embedded_schema
    version_contract = public_schema["properties"]["ptsip"]["properties"]["version"]
    assert version_contract["enum"] == ["0.3.6-draft", "pp.1.01"]


def test_maintained_examples_use_pp_1_01_without_structural_redesign() -> None:
    schema = json.loads((ROOT / "schemas" / "ptsip-profile.schema.json").read_text(encoding="utf-8"))
    validator = Draft202012Validator(schema)

    for relative_path in (
        "profiles/example.ptsip.yaml",
        "profiles/hybrid-python-package.ptsip.yaml",
        "profiles/template-python-package.ptsip.yaml",
    ):
        profile = yaml.safe_load((ROOT / relative_path).read_text(encoding="utf-8"))
        assert profile["ptsip"]["version"] == "pp.1.01"
        validator.validate(profile)


def test_pp_1_01_release_note_discloses_identity_only_bridge() -> None:
    note = (ROOT / "releasenote" / "project-profile" / "pp.1.01.md").read_text(encoding="utf-8")

    assert "0.3.6-draft" in note
    assert "pp.1.01" in note
    assert "IDENTITY_ONLY" in note
    assert "does **not** need lifecycle redesign" in note
