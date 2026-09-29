from __future__ import annotations

import copy
from dataclasses import replace

from developer.automation.pp.pp_transition_delta import (
    AuthorityState,
    PublicProfileEntry,
    evaluate_t2_authority_delta,
)


PROFILE_A = '{"policies":{},"ptsip":{"specification":{"family":"x"}},"responsibility_map":{"mode":"explicit"}}'
PROFILE_B = '{"policies":{"changed":true},"ptsip":{"specification":{"family":"x"}},"responsibility_map":{"mode":"explicit"}}'


def _state(
    *,
    label: str,
    current: str | None = "pp.1.01",
    catalog_present: bool = True,
    registry_present: bool = True,
    entries: dict[str, PublicProfileEntry] | None = None,
    semantic: str = PROFILE_A,
    raw: bytes = b"profile",
    schema: bytes | None = b"schema-v1",
    schema_semantic: str | None = "schema-v1",
) -> AuthorityState:
    selected = entries or {
        "example": PublicProfileEntry("example", "example.ptsip.yaml", current or "pp.1.01")
    }
    resources = tuple(sorted(entry.resource for entry in selected.values()))
    return AuthorityState(
        label=label,
        catalog_present=catalog_present,
        registry_present=registry_present,
        current=current,
        contract_schemas=(
            {current: "schemas/ptsip-profile-pp-1.01.schema.json"}
            if current is not None
            else {}
        ),
        entries=selected,
        discovered_resources=resources,
        raw_profiles={resource: raw for resource in resources},
        semantic_profiles={resource: semantic for resource in resources},
        declared_profile_versions={resource: current for resource in resources},
        current_schema_bytes=schema,
        current_schema_semantic=schema_semantic,
    )


def test_no_t2_delta_keeps_current_identity() -> None:
    base = _state(label="base")
    candidate = _state(label="candidate")

    result = evaluate_t2_authority_delta(
        base,
        candidate,
        base_schema_bytes_for_candidate_current=b"schema-v1",
    )

    assert result.valid
    assert not result.triggered
    assert result.classification == "NO_T2_AUTHORITY_DELTA"
    assert result.expected_next is None


def test_public_profile_semantic_content_change_triggers_adjacent_minor() -> None:
    base = _state(label="base")
    candidate = _state(label="candidate", semantic=PROFILE_B)

    result = evaluate_t2_authority_delta(
        base,
        candidate,
        base_schema_bytes_for_candidate_current=b"schema-v1",
    )

    assert result.valid
    assert result.triggered
    assert result.reasons == ("REGISTERED_CANONICAL_PUBLIC_PROFILE_CONTENT",)
    assert result.expected_next == "pp.1.02"
    assert not result.candidate_already_reconciled


def test_catalog_membership_change_is_t2_authority_delta() -> None:
    base = _state(label="base")
    entries = dict(base.entries)
    entries["second"] = PublicProfileEntry(
        "second", "second.ptsip.yaml", "pp.1.01"
    )
    candidate = _state(label="candidate", entries=entries)

    result = evaluate_t2_authority_delta(
        base,
        candidate,
        base_schema_bytes_for_candidate_current=b"schema-v1",
    )

    assert result.valid
    assert result.triggered
    assert "PUBLIC_PROFILE_CATALOG_MEMBERSHIP" in result.reasons


def test_catalog_resource_identity_change_is_t2_authority_delta() -> None:
    base = _state(label="base")
    candidate = _state(
        label="candidate",
        entries={
            "example": PublicProfileEntry(
                "example", "renamed.ptsip.yaml", "pp.1.01"
            )
        },
    )

    result = evaluate_t2_authority_delta(
        base,
        candidate,
        base_schema_bytes_for_candidate_current=b"schema-v1",
    )

    assert result.valid
    assert result.triggered
    assert "PUBLIC_PROFILE_CATALOG_RESOURCE_IDENTITY" in result.reasons


def test_current_canonical_schema_change_is_t2_authority_delta() -> None:
    base = _state(label="base", schema=b"schema-v1")
    candidate = _state(
        label="candidate",
        schema=b"schema-v2",
        schema_semantic="schema-v2",
    )

    result = evaluate_t2_authority_delta(
        base,
        candidate,
        base_schema_bytes_for_candidate_current=b"schema-v2",
    )

    assert result.valid
    assert result.triggered
    assert result.reasons == ("CURRENT_PP_CANONICAL_SCHEMA_CONTENT",)


def test_manual_current_pointer_change_without_authority_delta_fails_closed() -> None:
    base = _state(label="base")
    candidate = replace(
        _state(label="candidate"),
        current="pp.1.02",
        contract_schemas={"pp.1.02": "schemas/ptsip-profile-pp-1.02.schema.json"},
        current_schema_bytes=b"schema-v2",
        current_schema_semantic="schema-v1",
    )

    result = evaluate_t2_authority_delta(
        base,
        candidate,
        base_schema_bytes_for_candidate_current=b"schema-v1",
    )

    assert not result.valid
    assert not result.triggered
    assert result.classification == "MANUAL_PP_TRANSITION_WITHOUT_AUTHORITY"


def test_already_reconciled_candidate_does_not_request_second_bump() -> None:
    base = _state(label="base")
    candidate = replace(
        _state(label="candidate", semantic=PROFILE_B),
        current="pp.1.02",
        contract_schemas={"pp.1.02": "schemas/ptsip-profile-pp-1.02.schema.json"},
        current_schema_bytes=b"schema-v2",
        current_schema_semantic="schema-v1",
    )

    result = evaluate_t2_authority_delta(
        base,
        candidate,
        base_schema_bytes_for_candidate_current=b"schema-v1",
    )

    assert result.valid
    assert result.triggered
    assert result.expected_next == "pp.1.02"
    assert result.candidate_already_reconciled


def test_initial_catalog_and_registry_materialization_is_not_semantic_delta() -> None:
    candidate = _state(label="candidate")
    base = replace(
        candidate,
        label="base",
        catalog_present=False,
        registry_present=False,
        current=None,
        contract_schemas={},
        entries={},
        current_schema_bytes=None,
        declared_profile_versions={"example.ptsip.yaml": "pp.1.01"},
    )

    result = evaluate_t2_authority_delta(
        base,
        candidate,
        base_schema_bytes_for_candidate_current=b"schema-v1",
    )

    assert result.valid
    assert not result.triggered
    assert result.classification == "BASELINE_MATERIALIZATION_EXISTING_DISTRIBUTION"


def test_baseline_materialization_rejects_schema_drift() -> None:
    candidate = _state(
        label="candidate",
        schema=b"schema-v2",
        schema_semantic="schema-v2",
    )
    base = replace(
        candidate,
        label="base",
        catalog_present=False,
        registry_present=False,
        current=None,
        contract_schemas={},
        entries={},
        current_schema_bytes=None,
        declared_profile_versions={"example.ptsip.yaml": "pp.1.01"},
    )

    result = evaluate_t2_authority_delta(
        base,
        candidate,
        base_schema_bytes_for_candidate_current=b"schema-v1",
    )

    assert not result.valid
    assert result.classification == "INVALID_BASELINE_MATERIALIZATION"
