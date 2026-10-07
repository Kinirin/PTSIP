from __future__ import annotations

from pathlib import Path

import pytest

from ptsip.validation.handoff import (
    EffectiveMapValidationError, ValidatedEffectiveMap, _effective_map_data,
    load_validated_effective_map,
)

ROOT = Path(__file__).resolve().parents[4]


def test_real_validation_produces_an_immutable_owned_snapshot():
    value = load_validated_effective_map(ROOT, ROOT / ".ptsip/profiles/main.ptsip.yaml")
    assert value.source_mode == "explicit"
    assert isinstance(value.effective_payload["components"], tuple)
    with pytest.raises(TypeError):
        value.effective_payload["components"][0]["classification"] = "OPERATIONS"


def test_invalid_profile_does_not_produce_a_handoff(tmp_path):
    profile = tmp_path / "invalid.yaml"
    profile.write_text("components: []\n", encoding="utf-8")
    with pytest.raises(EffectiveMapValidationError, match="validation failed"):
        load_validated_effective_map(tmp_path, profile)


@pytest.mark.parametrize("value", [True, False, {}, {"validated": True}, None])
def test_caller_assertions_are_not_validation_proof(value):
    with pytest.raises(EffectiveMapValidationError):
        _effective_map_data(value)


def test_direct_or_forged_construction_is_not_validation_proof():
    with pytest.raises(TypeError):
        ValidatedEffectiveMap()
    with pytest.raises(EffectiveMapValidationError):
        _effective_map_data(object.__new__(ValidatedEffectiveMap))
