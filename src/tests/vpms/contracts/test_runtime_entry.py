from __future__ import annotations

import copy
import pytest

from vpms import contract_runtime as contracts
from vpms.domain.snapshot import load_registry_snapshot
from vpms.execution.composition import run_cases
from vpms.selection import resolve_selection


@pytest.mark.parametrize("role", ["protocol", "selection", "execution_composition"])
def test_registered_active_contract_grants_execution_but_is_immutable(role):
    contract = contracts.load_registered_contract(role)
    assert contract["status"] == "ACTIVE" and contract["runtime_enabled"]
    assert contracts.require_active_contract(role) == contract
    with pytest.raises(TypeError):
        contract["runtime_enabled"] = True


def test_active_runtime_still_rejects_invalid_handoffs():
    assert resolve_selection(True, {"kind": "CASE_IDS", "case_ids": ["a"]}).state == "REJECTED"
    with pytest.raises(ValueError):
        run_cases(True, True, executors={})


def test_unknown_role_or_escaping_resource_path_fails_closed():
    with pytest.raises(contracts.ContractUnavailable, match="UNKNOWN_CONTRACT_ROLE"):
        contracts.load_registered_contract("VPMS")
    for path in ("../policy/index.yaml", "C:/escape", "/absolute", "schema\\escape"):
        with pytest.raises(contracts.ContractUnavailable, match="UNSAFE_CONTRACT_PATH"):
            contracts._read(path)


def test_index_and_file_status_mismatch_is_not_accepted(monkeypatch):
    read = contracts._read
    def changed(relative):
        payload = copy.deepcopy(read(relative))
        if relative == "selection.json": payload["status"] = "APPROVED"
        return payload
    monkeypatch.setattr(contracts, "_read", changed)
    with pytest.raises(contracts.ContractUnavailable, match="CONTRACT_INDEX_MISMATCH"):
        contracts.load_registered_contract("selection")


def test_matching_active_flags_do_not_bypass_non_active_catalog_capability(monkeypatch):
    read = contracts._read
    def changed(relative):
        payload = copy.deepcopy(read(relative))
        if relative == "selection.json": payload.update(status="ACTIVE", runtime_enabled=True)
        if relative == "index.json":
            payload["capability"] = "REGISTERED_NON_ACTIVE_ONLY"
            identity = payload["entrypoints"]["selection"]
            payload["contracts"][identity].update(status="ACTIVE", runtime_enabled=True)
        return payload
    monkeypatch.setattr(contracts, "_read", changed)
    with pytest.raises(contracts.ContractUnavailable, match="CONTRACT_NOT_ACTIVE"):
        contracts.require_active_contract("selection")


@pytest.mark.parametrize("status,enabled", [("APPROVED", False), ("RETIRED", False), ("ACTIVE", False)])
def test_active_catalog_cannot_override_inactive_contract(monkeypatch, status, enabled):
    read = contracts._read
    def changed(relative):
        payload = copy.deepcopy(read(relative))
        if relative == "selection.json": payload.update(status=status, runtime_enabled=enabled)
        if relative == "index.json":
            identity = payload["entrypoints"]["selection"]
            payload["contracts"][identity].update(status=status, runtime_enabled=enabled)
        return payload
    monkeypatch.setattr(contracts, "_read", changed)
    with pytest.raises(contracts.ContractUnavailable, match="CONTRACT_NOT_ACTIVE"):
        contracts.require_active_contract("selection")
