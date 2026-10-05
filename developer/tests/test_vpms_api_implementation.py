from __future__ import annotations

import json
from pathlib import Path

from developer.automation.vpms_api_implementation import verify

ROOT = Path(__file__).resolve().parents[2]


def test_successor_activation_does_not_rewrite_implementation_only_approval():
    result = verify(ROOT)
    assert result["status"] == "IMPLEMENTED_ACTIVE"
    assert result["commit_push_authorized"] is True
    assert result["runtime_activation"] is True
    record = json.loads((ROOT / "developer/policy/registries/vpms-api-implementation.json").read_text(encoding="utf-8"))
    assert record["approval"]["decision_source"] == "USER_EXPLICIT"
    assert record["approval"]["automatic_commit_push_authorized"]
    assert not record["approval"]["selector_retirement_authorized"]
    assert not record["approval"]["sfp_0006_retirement_authorized"]
    assert record["task_context_status"] == "UNREGISTERED"


def test_new_user_policy_writes_use_source_policy_not_retired_docs_tree():
    record = json.loads((ROOT / "developer/policy/registries/vpms-api-implementation.json").read_text(encoding="utf-8"))
    assert record["support_policy_generation_root"] == "src/policy"
    assert not (ROOT / "docs/Support_policy").exists()
    assert (ROOT / "developer/automation/support_contract_registration.py").is_file()
    retired = {item["path"] for item in record["subsequent_retirements"]}
    assert all(not path.startswith("docs/Support_policy/")
               for path in record["mutation_targets"]
               if path not in retired)
    assert verify(ROOT)["retired_target_count"] == 1
