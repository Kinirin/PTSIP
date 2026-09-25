from __future__ import annotations

from pathlib import Path

import pytest

from ptsip.agent_contracts import (
    AgentOperationLoadError,
    load_operation_contract,
    validate_agent_contract_plane,
)


OPERATIONS = (
    "adopt",
    "validate",
    "conform",
    "reconcile-authority",
    "migrate-profile",
)


def _all_strings(value: object):
    if isinstance(value, str):
        yield value
    elif isinstance(value, dict):
        for item in value.values():
            yield from _all_strings(item)
    elif isinstance(value, (list, tuple)):
        for item in value:
            yield from _all_strings(item)


def test_active_machine_plane_is_self_consistent_and_current() -> None:
    assert validate_agent_contract_plane() == {
        "specs": 8,
        "operations": 5,
        "vocabularies": 4,
        "bindings": 1,
    }


@pytest.mark.parametrize("operation_key", OPERATIONS)
def test_operation_entry_loads_only_machine_resources_within_budget(
    operation_key: str,
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.chdir(tmp_path)
    bundle = load_operation_contract(operation_key)

    assert bundle.binding["status"] == "CURRENT"
    assert bundle.binding["activation"] == {
        "status": "ACTIVE",
        "basis": "EXPLICIT_PROJECT_OWNER_APPROVAL",
    }
    assert bundle.binding["loading"]["strategy"] == "OPERATION_SCOPED"
    assert bundle.binding["loading"]["markdown_dependency"] == "FORBIDDEN"
    assert bundle.context_bytes <= bundle.context_budget_bytes
    assert bundle.operation_ref in bundle.resource_refs
    assert set(bundle.operation["spec_refs"]) <= set(bundle.resource_refs)
    assert set(bundle.operation["vocabulary_refs"]) <= set(bundle.resource_refs)

    strings = list(_all_strings(bundle.as_dict()))
    assert not [value for value in strings if ".md" in value.lower()]


def test_reconcile_authority_entry_does_not_load_unrelated_spec_domains() -> None:
    bundle = load_operation_contract("reconcile-authority")

    assert bundle.operation["spec_refs"] == ["spec/authority.yaml"]
    assert [spec["id"] for spec in bundle.specs] == ["PTSIP-AGENT-SPEC-AUTHORITY"]


def test_unknown_operation_fails_closed() -> None:
    with pytest.raises(AgentOperationLoadError, match="Unknown agent operation"):
        load_operation_contract("unknown-operation")
