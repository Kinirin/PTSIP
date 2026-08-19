from __future__ import annotations

import hashlib
from typing import Iterable, Protocol

from ..validation.components import (
    AMBIGUOUS,
    COMPONENT_COVERED,
    normalize_selector,
    resolve_candidate_coverage,
)
from .model import FIELD_ORDER, REASON_BY_FIELD, ClarificationRequest


class CandidateLike(Protocol):
    id: str
    include: tuple[str, ...]
    anchors: tuple[str, ...]
    evidence_ids: tuple[str, ...]


def covering_components(candidate: CandidateLike, components: list[dict[str, object]]) -> list[dict[str, object]]:
    """Return canonical best component coverage for compatibility callers.

    Selector interpretation is owned by ``ptsip.validation.components``.  This
    wrapper preserves the historical caller shape without maintaining a second
    clarification-specific selector dialect.
    """

    coverage = resolve_candidate_coverage(candidate, components)
    if coverage.status not in {COMPONENT_COVERED, AMBIGUOUS}:
        return []
    owner_ids = set(coverage.owner_ids)
    return [item for item in components if str(item.get("id", "")) in owner_ids]


def build_requests(
    repository_identity: str,
    candidates: Iterable[CandidateLike],
    declared_components: list[dict[str, object]],
) -> tuple[ClarificationRequest, ...]:
    requests: list[ClarificationRequest] = []
    for candidate in candidates:
        covering = covering_components(candidate, declared_components)
        target_component_id = candidate.id
        if len(covering) == 1:
            declared = covering[0]
            declared_id = str(declared.get("id", "")).strip()
            if declared_id:
                target_component_id = declared_id
            missing_required = tuple(
                field
                for field in ("classification", "purpose")
                if not str(declared.get(field, "")).strip()
            )
            if not missing_required:
                continue
            missing_fields = missing_required
        else:
            missing_fields = FIELD_ORDER
        reasons = tuple(REASON_BY_FIELD[field] for field in missing_fields)
        selector_identity = ",".join(sorted(normalize_selector(item) for item in candidate.include))
        digest = hashlib.sha256(
            (
                repository_identity
                + "\0"
                + candidate.id
                + "\0"
                + target_component_id
                + "\0"
                + selector_identity
                + "\0"
                + ",".join(missing_fields)
            ).encode("utf-8")
        ).hexdigest()[:16]
        requests.append(
            ClarificationRequest(
                id=f"clr-{digest}",
                component_id=target_component_id,
                include=tuple(candidate.include),
                anchors=tuple(candidate.anchors),
                evidence_ids=tuple(candidate.evidence_ids),
                missing_fields=tuple(missing_fields),
                reason_codes=reasons,
            )
        )
    return tuple(requests)
