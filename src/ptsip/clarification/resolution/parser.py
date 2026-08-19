from __future__ import annotations

import re

import yaml

from .model import (
    CANONICAL_ANSWER_FIELDS,
    LEGACY_V1_ANSWER_FIELDS,
    DecisionAnswer,
    LegacyDecisionAnswerV1,
)

ANSWER_FORMAT = "ptsip-clarification-answer/v2"
LEGACY_ANSWER_FORMAT = "ptsip-clarification-answer/v1"


def _boolean(value: object, field: str) -> bool:
    if isinstance(value, bool):
        return value
    if isinstance(value, str):
        normalized = value.strip().upper()
        if normalized in {"YES", "TRUE"}:
            return True
        if normalized in {"NO", "FALSE"}:
            return False
    raise ValueError(f"{field} must be YES or NO")


def _mapping_for_format(text: str, expected_format: str) -> dict[str, object]:
    blocks = re.findall(r"```(?:ya?ml)?\s*\n(.*?)```", text, flags=re.IGNORECASE | re.DOTALL)
    candidates = [*reversed(blocks), text]
    for candidate in candidates:
        try:
            payload = yaml.safe_load(candidate)
        except yaml.YAMLError:
            continue
        if isinstance(payload, dict) and payload.get("format") == expected_format:
            return payload
    raise ValueError(f"No structured PTSIP clarification answer with format {expected_format!r} was found")


def _decision_mapping(payload: dict[str, object]) -> dict[str, object]:
    raw = payload.get("decision")
    if not isinstance(raw, dict):
        raise ValueError("decision must be a mapping")
    return raw


def _require_exact_fields(raw: dict[str, object], expected: tuple[str, ...], format_name: str) -> None:
    actual = set(raw)
    expected_set = set(expected)
    missing = sorted(expected_set - actual)
    extra = sorted(actual - expected_set)
    if missing or extra:
        details: list[str] = []
        if missing:
            details.append("missing: " + ", ".join(missing))
        if extra:
            details.append("unexpected: " + ", ".join(extra))
        raise ValueError(f"{format_name} decision fields must match the canonical shape exactly ({'; '.join(details)})")


def parse_answer(text: str) -> DecisionAnswer:
    """Parse only the canonical ptsip-clarification-answer/v2 contract."""

    payload = _mapping_for_format(text, ANSWER_FORMAT)
    raw = _decision_mapping(payload)
    _require_exact_fields(raw, CANONICAL_ANSWER_FIELDS, ANSWER_FORMAT)
    return DecisionAnswer(
        classification=str(raw["classification"]).strip().upper(),
        purpose=str(raw["purpose"]).strip(),
        shipped=_boolean(raw["shipped"], "shipped"),
        runtime_required=_boolean(raw["runtime_required"], "runtime_required"),
        executable=_boolean(raw["executable"], "executable"),
    )


def parse_legacy_answer(text: str) -> LegacyDecisionAnswerV1:
    """Explicitly parse the historical v1 compatibility contract.

    This function never translates classification vocabulary. Callers must pass
    its result through the explicit legacy compatibility validator before using
    the canonical DecisionAnswer produced by ``to_canonical()``.
    """

    payload = _mapping_for_format(text, LEGACY_ANSWER_FORMAT)
    raw = _decision_mapping(payload)
    _require_exact_fields(raw, LEGACY_V1_ANSWER_FIELDS, LEGACY_ANSWER_FORMAT)
    return LegacyDecisionAnswerV1(
        classification=str(raw["classification"]).strip().upper(),
        purpose=str(raw["purpose"]).strip(),
        shipped=_boolean(raw["shipped"], "shipped"),
        runtime_required=_boolean(raw["runtime_required"], "runtime_required"),
        lifecycle_owner=str(raw["lifecycle_owner"]).strip().upper(),
        executable=_boolean(raw["executable"], "executable"),
    )
