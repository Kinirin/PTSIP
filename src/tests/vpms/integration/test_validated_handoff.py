from __future__ import annotations

from pathlib import Path

import pytest

from ptsip.validation.handoff import ValidatedEffectiveMap, load_validated_effective_map
from vpms.domain.model import TargetRef
from vpms.integration.ptsip_bridge import (
    PtsipMetadataError, metadata_from_effective_map, resolve_target_metadata,
)

ROOT = Path(__file__).resolve().parents[4]


def test_active_boundary_projects_real_validated_ptsip_metadata_without_mutation():
    source = ROOT / "developer/profiles/ptsip-repository.yaml"
    before = source.read_bytes()
    handoff = load_validated_effective_map(ROOT, source)
    snapshot = metadata_from_effective_map(handoff)
    assert source.read_bytes() == before
    assert [row.component_id for row in snapshot.targets] == sorted(row.component_id for row in snapshot.targets)
    assert resolve_target_metadata(TargetRef("vpms"), snapshot).classification == "PRODUCT"
    assert resolve_target_metadata(TargetRef("unregistered"), snapshot) is None
    assert all(set(row.as_dict()) == {"component_id", "classification"} for row in snapshot.targets)


@pytest.mark.parametrize("value", [True, False, None, {}, {"validated": True, "components": []}])
def test_raw_values_never_gain_the_canonical_boundary(value, monkeypatch):
    import vpms.integration.ptsip_bridge as bridge
    monkeypatch.setattr(bridge, "load_ptsip_metadata", lambda *args: pytest.fail("raw-profile fallback invoked"))
    with pytest.raises(PtsipMetadataError, match="provider handoff"):
        metadata_from_effective_map(value)


def test_forged_provider_instance_is_rejected():
    with pytest.raises(PtsipMetadataError):
        metadata_from_effective_map(object.__new__(ValidatedEffectiveMap))


@pytest.mark.parametrize("bad", ["inactive_index", "inactive_policy", "unregistered_provider"])
def test_runtime_boundary_requires_active_registered_support_provider(monkeypatch, bad):
    import copy
    from ptsip.governance import AuthorityCatalog
    original = AuthorityCatalog.load_current_record
    def changed(self, identity):
        path, route, policy = original(self, identity)
        route, policy = copy.deepcopy(route), copy.deepcopy(policy)
        if identity == "SFP-0023":
            if bad == "inactive_index": route["status"] = "DRAFT"
            if bad == "inactive_policy": policy["policy"]["status"] = "DRAFT"
            if bad == "unregistered_provider": policy["authority_semantics"]["input"]["concrete_provider_binding"] = "unregistered:provider"
        return path, route, policy
    monkeypatch.setattr(AuthorityCatalog, "load_current_record", changed)
    with pytest.raises(PtsipMetadataError, match="not active|Unsupported PTSIP provider"):
        metadata_from_effective_map(True)
