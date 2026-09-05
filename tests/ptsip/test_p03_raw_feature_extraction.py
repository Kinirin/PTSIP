from __future__ import annotations

import importlib.util
import subprocess
import sys
from pathlib import Path

import yaml


ROOT = Path(__file__).resolve().parents[2]
RAW_COLLECTOR = ROOT / ".github" / "scripts" / "p03_authority_role_raw_features.py"
RAW_SNAPSHOT = (
    ROOT / "planning" / "0.4.0" / "WU-02" / "p03-authority-role-raw-features.generated.yaml"
)
LEGACY_ANALYZER = ROOT / ".github" / "scripts" / "p03_authority_role_matrix.py"
LEGACY_REGISTRY = (
    ROOT / "planning" / "0.4.0" / "WU-02" / "p03-authority-role-provisional-dimensions.yaml"
)


def _yaml(path: Path) -> dict[str, object]:
    value = yaml.safe_load(path.read_text(encoding="utf-8"))
    assert isinstance(value, dict)
    return value


def _load_raw_collector():
    spec = importlib.util.spec_from_file_location("p03_authority_role_raw_features", RAW_COLLECTOR)
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def test_p03_raw_feature_snapshot_is_current() -> None:
    result = subprocess.run(
        [
            sys.executable,
            str(RAW_COLLECTOR),
            "--repo-root",
            str(ROOT),
            "--check",
        ],
        text=True,
        encoding="utf-8",
        errors="replace",
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )

    assert result.returncode == 0, result.stderr


def test_p03_raw_collection_covers_full_corpus_without_semantic_review_promotion() -> None:
    snapshot = _yaml(RAW_SNAPSHOT)
    source = snapshot["source"]
    assert source["first_adr"] == "ADR-0001"
    assert source["raw_collection_through"] == "ADR-0023"
    assert source["semantic_review_through"] == "ADR-0010"
    assert source["runtime_authority"] == "NONE"
    assert source["vocabulary_registration"] is False
    assert source["semantic_grouping_performed"] is False

    rows = snapshot["rows"]
    assert list(rows) == [f"ADR-{number:04d}" for number in range(1, 24)]

    feature_model = snapshot["feature_model"]
    assert feature_model["unit"] == "EXACT_TOP_LEVEL_AUTHORITY_SEMANTIC_PATH_TYPED_VALUE"
    assert feature_model["binary_projection"] == "SPARSE_MACHINE_DERIVABLE"
    assert feature_model["false_is_default_fill"] is False
    assert feature_model["natural_language_consumption"] == "FORBIDDEN"
    assert feature_model["semantic_effect_name_generation"] == "FORBIDDEN"

    for feature in snapshot["features"]:
        assert set(feature) == {"path", "value_type", "value", "occurs_in"}
        assert feature["path"].startswith("authority_semantics.")
        assert feature["occurs_in"] == sorted(set(feature["occurs_in"]))


def test_p03_raw_feature_extraction_preserves_exact_machine_value_types() -> None:
    collector = _load_raw_collector()
    features = collector.raw_features_from_semantics(
        {
            "flag": True,
            "count": 1,
            "name": "1",
            "items": ["A", "B"],
            "mapping": {"b": 2, "a": 1},
        }
    )
    by_path = {feature["path"]: feature for feature in features}

    assert by_path["authority_semantics.flag"]["value_type"] == "BOOLEAN"
    assert by_path["authority_semantics.count"]["value_type"] == "INTEGER"
    assert by_path["authority_semantics.name"]["value_type"] == "STRING"
    assert by_path["authority_semantics.items"]["value_type"] == "ARRAY"
    assert by_path["authority_semantics.mapping"]["value_type"] == "OBJECT"
    assert by_path["authority_semantics.mapping"]["value"] == {"a": 1, "b": 2}


def test_p03_legacy_semantic_dimension_path_is_frozen_at_adr_0010(tmp_path: Path) -> None:
    registry = _yaml(LEGACY_REGISTRY)
    registry["analysis"]["reviewed_through"] = "ADR-0011"
    registry_path = tmp_path / "legacy-registry.yaml"
    registry_path.write_text(
        yaml.safe_dump(registry, sort_keys=False, allow_unicode=True),
        encoding="utf-8",
    )

    result = subprocess.run(
        [
            sys.executable,
            str(LEGACY_ANALYZER),
            "--repo-root",
            str(ROOT),
            "--registry",
            str(registry_path),
            "--output",
            str(tmp_path / "legacy-matrix.yaml"),
            "--write",
        ],
        text=True,
        encoding="utf-8",
        errors="replace",
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )

    assert result.returncode == 1
    assert "legacy semantic dimension analysis is frozen" in result.stderr
