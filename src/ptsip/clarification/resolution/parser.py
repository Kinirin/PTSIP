from __future__ import annotations

import re

import yaml

from .model import DecisionAnswer

ANSWER_FORMAT = "ptsip-clarification-answer/v1"


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


def _mapping(text: str) -> dict[str, object]:
    blocks = re.findall(r"```(?:ya?ml)?\s*\n(.*?)```", text, flags=re.IGNORECASE | re.DOTALL)
    candidates = [*reversed(blocks), text]
    for candidate in candidates:
        try:
            payload = yaml.safe_load(candidate)
        except yaml.YAMLError:
            continue
        if isinstance(payload, dict):
            if payload.get("format") == ANSWER_FORMAT or isinstance(payload.get("decision"), dict):
                return payload
    raise ValueError("No structured PTSIP clarification answer was found")


def parse_answer(text: str) -> DecisionAnswer:
    payload = _mapping(text)
    if payload.get("format") not in (None, ANSWER_FORMAT):
        raise ValueError(f"format must be {ANSWER_FORMAT!r}")
    raw = payload.get("decision", payload)
    if not isinstance(raw, dict):
        raise ValueError("decision must be a mapping")
    classification = str(raw.get("classification", "")).strip().upper()
    purpose = str(raw.get("purpose", "")).strip()
    lifecycle_owner = str(raw.get("lifecycle_owner", "")).strip().upper()
    executable = _boolean(raw.get("executable"), "executable")
    return DecisionAnswer(
        classification=classification,
        purpose=purpose,
        shipped=_boolean(raw.get("shipped"), "shipped"),
        runtime_required=_boolean(raw.get("runtime_required"), "runtime_required"),
        lifecycle_owner=lifecycle_owner,
        executable=executable,
    )
