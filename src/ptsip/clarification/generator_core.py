from __future__ import annotations

import hashlib
from typing import Iterable, Protocol

from .model import FIELD_ORDER, REASON_BY_FIELD, ClarificationRequest


class CandidateLike(Protocol):
    id: str
    include: tuple[str, ...]
    anchors: tuple[str, ...]
    evidence_ids: tuple[str, ...]


def _normalize_selector(value: str) -> str:
    text = value.replace("\\", "/").strip()
    while text.startswith("./"):
        text = text[2:]
    return text.strip("/")


def _selector_covers(declared: str, candidate: str) -> bool:
    declared = _normalize_selector(declared)
    candidate = _normalize_selector(candidate)
    if declared == candidate:
        return True
    if declared.endswith("/**"):
        base = declared[:-3].rstrip("/")
        candidate_base = candidate[:-3].rstrip("/") if candidate.endswith("/**") else candidate
        return candidate_base == base or candidate_base.startswith(base + "/")
    return False


def _covering_components(candidate: CandidateLike, components: list[dict[str, object]]) -> list[dict[str, object]]:
    found: list[dict[str, object]] = []
    for component in components:
        selectors = [str(item) for item in component.get("include", [])]
        if any(
            _selector_covers(selector, candidate_selector)
            for selector in selectors
            for candidate_selector in candidate.include
        ):
            found.append(component)
    return found


def build_requests(
    repository_identity: str,
    candidates: Iterable[CandidateLike],
    declared_components: list[dict[str, object]],
) -> tuple[ClarificationRequest, ...]:
    requests: list[ClarificationRequest] = []
    for candidate in candidates:
        covering = _covering_components(candidate, declared_components)
        if len(covering) == 1 and str(covering[0].get("purpose", "")).strip():
            continue
        missing_fields = ("purpose",) if len(covering) == 1 else FIELD_ORDER
        reasons = tuple(REASON_BY_FIELD[field] for field in missing_fields)
        digest = hashlib.sha256(
            (repository_identity + "\0" + candidate.id + "\0" + ",".join(missing_fields)).encode("utf-8")
        ).hexdigest()[:16]
        requests.append(
            ClarificationRequest(
                id=f"clr-{digest}",
                component_id=candidate.id,
                include=tuple(candidate.include),
                anchors=tuple(candidate.anchors),
                evidence_ids=tuple(candidate.evidence_ids),
                missing_fields=tuple(missing_fields),
                reason_codes=reasons,
            )
        )
    return tuple(requests)
