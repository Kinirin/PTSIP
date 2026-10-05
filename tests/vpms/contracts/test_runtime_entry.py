from __future__ import annotations

import copy
import pytest

from vpms import contract_runtime as contracts
from vpms.domain.snapshot import load_registry_snapshot
from vpms.execution.composition import run_cases
from vpms.selection import resolve_selection


@pytest.mark.parametrize("role", ["protocol", "selection", "execution_composition"])
def test_approved_registered_contract_does_not_grant_execution(role):
    contract = contracts.load_registered_contract(role)
    assert contract["status"] == "APPROVED" and not contract["runtime_enabled"]
    with pytest.raises(contracts.ContractUnavailable, match="CONTRACT_NOT_ACTIVE"):
        contracts.require_active_contract(role)
    with pytest.raises(TypeError):
        contract["runtime_enabled"] = True


@pytest.mark.parametrize("call", [lambda: load_registry_snapshot([], references={}),
    lambda: resolve_selection(True, {"kind": "CASE_IDS", "case_ids": ["a"]}),
    lambda: run_cases(True, True, executors={})])
def test_successor_apis_are_installed_but_default_runtime_calls_are_blocked(call):
    with pytest.raises(contracts.ContractUnavailable, match="CONTRACT_NOT_ACTIVE"):
        call()


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
        if relative == "selection.json": payload["status"] = "ACTIVE"
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
            identity = payload["entrypoints"]["selection"]
            payload["contracts"][identity].update(status="ACTIVE", runtime_enabled=True)
        return payload
    monkeypatch.setattr(contracts, "_read", changed)
    with pytest.raises(contracts.ContractUnavailable, match="CONTRACT_NOT_ACTIVE"):
        contracts.require_active_contract("selection")
