from __future__ import annotations

import json
from copy import deepcopy
from pathlib import Path

import pytest
import yaml
from jsonschema import Draft202012Validator, ValidationError


ROOT = Path(__file__).resolve().parents[2]
SCHEMA_PATH = ROOT / "schemas" / "ptsip-adr-template.schema.json"
TEMPLATE_PATH = ROOT / "decisions" / "ADR-TEMPLATE.yaml"


def _schema() -> dict[str, object]:
    return json.loads(SCHEMA_PATH.read_text(encoding="utf-8"))


def _template() -> dict[str, object]:
    return yaml.safe_load(TEMPLATE_PATH.read_text(encoding="utf-8"))


def test_adr_template_schema_is_valid_draft_2020_12() -> None:
    Draft202012Validator.check_schema(_schema())


def test_adr_template_validates_against_schema() -> None:
    Draft202012Validator(_schema()).validate(_template())


def test_adr_template_schema_requires_rejected_alternatives_contract() -> None:
    invalid = deepcopy(_template())
    del invalid["sections"]["rejected_alternatives"]

    with pytest.raises(ValidationError):
        Draft202012Validator(_schema()).validate(invalid)
