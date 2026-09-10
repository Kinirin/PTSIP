from __future__ import annotations

import json
from pathlib import Path

import yaml


ROOT = Path(__file__).resolve().parents[2]


def _load_yaml(path: Path) -> dict[str, object]:
    payload = yaml.safe_load(path.read_text(encoding="utf-8"))
    assert isinstance(payload, dict)
    return payload


def _load_json(path: Path) -> dict[str, object]:
    payload = json.loads(path.read_text(encoding="utf-8"))
    assert isinstance(payload, dict)
    return payload


def test_current_sfp_mpd_corpus_has_no_legacy_source_provenance() -> None:
    policy_paths = [
        ROOT / "src" / "ptsip" / "specdata" / f"SFP-{number:04d}.yaml"
        for number in range(1, 22)
    ] + [
        ROOT / "developer" / "policy" / f"MPD-{number:04d}.yaml"
        for number in range(2, 10)
    ]

    assert len(policy_paths) == 29
    for path in policy_paths:
        payload = _load_yaml(path)
        assert "source_provenance" not in payload, path.relative_to(ROOT).as_posix()
        text = path.read_text(encoding="utf-8")
        assert "source_type: LEGACY_ADR" not in text, path.relative_to(ROOT).as_posix()
        assert "source_path: decisions/ADR-" not in text, path.relative_to(ROOT).as_posix()


def test_current_policy_schemas_do_not_define_legacy_source_provenance() -> None:
    sfp_schema_paths = (
        ROOT / "schemas" / "ptsip-support-feature-policy.schema.json",
        ROOT / "src" / "ptsip" / "specdata" / "ptsip-support-feature-policy.schema.json",
    )
    sfp_schemas = [_load_json(path) for path in sfp_schema_paths]
    assert sfp_schemas[0] == sfp_schemas[1]
    for schema in sfp_schemas:
        assert "source_provenance" not in schema["properties"]
        assert schema.get("not") == {"required": ["source_provenance"]}

    mpd_schema = _load_json(
        ROOT / "developer" / "policy" / "schemas" / "management-policy.schema.json"
    )
    assert "source_provenance" not in mpd_schema["properties"]
    assert mpd_schema["additionalProperties"] is False


def test_legacy_policy_materializer_is_retired_from_current_validation() -> None:
    materializer = ROOT / "developer" / "automation" / "policy_materializer.py"
    assert not materializer.exists()

    validator_text = (
        ROOT / "developer" / "automation" / "policy_validator.py"
    ).read_text(encoding="utf-8")
    for forbidden in (
        "policy_materializer",
        "legacy-decisions-inventory.yaml",
        "policy-relation-migration.yaml",
        "decision_reference_migrator",
    ):
        assert forbidden not in validator_text
