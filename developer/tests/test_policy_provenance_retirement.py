from __future__ import annotations

import json
from pathlib import Path

import yaml
from ptsip.governance.authority import AuthorityCatalog


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
    support_root = ROOT / "src" / "policy"
    support_index = _load_yaml(support_root / "index.yaml")
    developer_index = _load_yaml(ROOT / "developer" / "policy" / "index.yaml")
    policy_paths = [
        support_root / entry["path"] for entry in support_index["policies"]
    ] + [
        ROOT / entry["path"] for entry in developer_index["policies"]
    ]

    assert len(policy_paths) == len(support_index["policies"]) + len(developer_index["policies"])
    assert len(policy_paths) == len(set(policy_paths))
    for path in policy_paths:
        payload = _load_yaml(path)
        assert "source_provenance" not in payload, path.relative_to(ROOT).as_posix()
        text = path.read_text(encoding="utf-8")
        assert "source_type: LEGACY_ADR" not in text, path.relative_to(ROOT).as_posix()
        assert "source_path: decisions/ADR-" not in text, path.relative_to(ROOT).as_posix()


def test_current_policy_schemas_do_not_define_legacy_source_provenance() -> None:
    catalog = AuthorityCatalog(ROOT)
    canonical_schema = _load_json(
        ROOT / "src" / "policy" / "schemas" / catalog.SUPPORT_POLICY_SCHEMA
    )
    assert canonical_schema == catalog.policy_schema
    sfp_schemas = [canonical_schema, catalog.policy_schema]
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
