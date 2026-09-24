from agent_contracts import validate_agent_contract_plane


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
