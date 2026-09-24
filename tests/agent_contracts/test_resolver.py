from __future__ import annotations

import pytest

from agent_contracts.resolver import resolve_operation


OPERATIONS = (
    "PTSIP-OP-ADOPT-001",
    "PTSIP-OP-VALIDATE-001",
    "PTSIP-OP-CONFORM-001",
    "PTSIP-OP-RECONCILE-AUTHORITY-001",
    "PTSIP-OP-MIGRATE-PROFILE-001",
)


@pytest.mark.parametrize("operation_id", OPERATIONS)
def test_operation_resolution_is_bounded_and_exact(operation_id: str) -> None:
    result = resolve_operation(operation_id)

    assert result["operation_id"] == operation_id
    assert result["markdown_dependency"] is False
    assert [item["rule_id"] for item in result["rules"]] == result["operation"]["rule_refs"]
    assert set(result["vocabularies"]) == set(result["operation"]["vocabulary_refs"])
    assert result["io_schemas"]
    assert result["selected_specs"]


def test_unknown_operation_fails_closed() -> None:
    with pytest.raises(Exception, match="unknown operation"):
        resolve_operation("PTSIP-OP-UNKNOWN-999")
