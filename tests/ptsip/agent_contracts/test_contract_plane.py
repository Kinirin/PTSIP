from ptsip.agent_contracts import validate_agent_contract_plane


def test_embedded_agent_contract_plane_is_self_consistent() -> None:
    result = validate_agent_contract_plane()

    assert result == {
        "specs": 8,
        "operations": 5,
        "vocabularies": 4,
        "rules": 48,
        "bindings": 1,
    }
