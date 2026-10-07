from importlib import import_module

import ptsip.migration as migration
from agent_contracts import validate_agent_contract_plane
from agent_contracts.resolver import resolve_operation


def test_embedded_agent_contract_plane_is_self_consistent() -> None:
    result = validate_agent_contract_plane()

    assert result == {
        "specs": 15,
        "operations": 5,
        "actions": 18,
        "conditions": 23,
        "gates": 3,
        "io_schemas": 15,
        "vocabularies": 5,
        "rules": 156,
        "bindings": 1,
    }


def test_migration_bindings_preserve_canonical_public_callables() -> None:
    resolved = resolve_operation("PTSIP-OP-MIGRATE-PROFILE-001")
    expected = {
        "PTSIP-ACT-ANALYZE-SOURCE-PROFILE-001": migration.analyze_source_migration,
        "PTSIP-ACT-MAP-PROFILE-TARGET-001": migration.build_direct_final_point_convergence_plan,
        "PTSIP-ACT-BUILD-TRANSITION-PLAN-001": migration.build_direct_final_point_convergence_plan,
        "PTSIP-ACT-APPLY-PROFILE-TRANSITION-001": migration.promote_canonical,
    }
    bindings = resolved["implementation_bindings"]["actions"]
    assert {item["action_id"] for item in bindings} == set(expected)
    for binding in bindings:
        module, attribute = binding["python_callable"].split(":")
        assert getattr(import_module(module), attribute) is expected[binding["action_id"]]
