from __future__ import annotations

import hashlib
import json
from typing import Mapping

_SET_FIELDS = frozenset({"roles", "include", "exclude", "manifests", "consumers", "analysis_inputs"})

def _canonical(value: object, *, field_name: str | None = None) -> object:
    if isinstance(value, Mapping):
        return {
            str(key): _canonical(item, field_name=str(key))
            for key, item in sorted(value.items(), key=lambda pair: str(pair[0]))
        }
    if isinstance(value, (list, tuple)):
        items = [_canonical(item) for item in value]
        if field_name in _SET_FIELDS:
            items.sort(
                key=lambda item: json.dumps(
                    item,
                    sort_keys=True,
                    separators=(",", ":"),
                    ensure_ascii=False,
                )
            )
        return items
    return value

def canonical_semantics(value: object) -> object:
    return _canonical(value)

def semantic_digest(value: object) -> str:
    encoded = json.dumps(
        canonical_semantics(value),
        sort_keys=True,
        separators=(",", ":"),
        ensure_ascii=False,
    ).encode("utf-8")
    return hashlib.sha256(encoded).hexdigest()

