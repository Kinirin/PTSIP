from __future__ import annotations

import json
from pathlib import Path

from developer.automation.vpms_api_implementation import verify

ROOT = Path(__file__).resolve().parents[2]


def test_explicit_api_implementation_and_automatic_commit_push_do_not_activate_contracts():
    result = verify(ROOT)
    assert result["status"] == "IMPLEMENTED_INACTIVE"
    assert result["commit_push_authorized"] is True
    assert result["runtime_activation"] is False
    record = json.loads((ROOT / "developer/policy/registries/vpms-api-implementation.json").read_text(encoding="utf-8"))
    assert record["approval"]["decision_source"] == "USER_EXPLICIT"
    assert record["approval"]["automatic_commit_push_authorized"]
    assert not record["approval"]["selector_retirement_authorized"]
    assert not record["approval"]["sfp_0006_retirement_authorized"]
    assert record["task_context_status"] == "UNREGISTERED"


def test_new_user_policy_writes_use_source_policy_not_retired_docs_tree():
    record = json.loads((ROOT / "developer/policy/registries/vpms-api-implementation.json").read_text(encoding="utf-8"))
    assert record["support_policy_generation_root"] == "src/policy"
    assert not (ROOT / "docs/Support_policy/automation/contract_registration.py").exists()
    assert (ROOT / "developer/automation/support_contract_registration.py").is_file()
    assert all(not path.startswith("docs/Support_policy/")
               for path in record["mutation_targets"]
               if path != "docs/Support_policy/automation/README.md")
    retired_notice = (ROOT / "docs/Support_policy/automation/README.md").read_text(encoding="utf-8")
    assert "is retired" in retired_notice and "must be created under `src/policy/`" in retired_notice
