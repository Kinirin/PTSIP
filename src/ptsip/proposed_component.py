from __future__ import annotations

import hashlib
import re
from dataclasses import dataclass

from .clarification.resolution.model import CANONICAL_ANSWER_FIELDS
from .repository.profile_path import normalize_profile_path
from .validation.components import normalize_selector


class ProposedComponentError(ValueError):
    """Fail-closed error for malformed explicit proposed-component registration."""


def _require_text(name: str, value: object) -> str:
    if not isinstance(value, str) or not value.strip():
        raise ProposedComponentError(f"{name} must be a non-empty string")
    return value.strip()


def _component_id(value: object) -> str:
    text = _require_text("component_id", value)
    if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]*", text):
        raise ProposedComponentError(
            "component_id must match the canonical Project Profile component ID grammar"
        )
    return text


def _selectors(values: object) -> tuple[str, ...]:
    if not isinstance(values, (tuple, list)) or not values:
        raise ProposedComponentError("include selectors must be a non-empty sequence")
    normalized: list[str] = []
    for raw in values:
        text = normalize_selector(_require_text("include selector", raw))
        if not text:
            raise ProposedComponentError("include selector must not normalize to an empty value")
        normalized.append(text)
    if len(set(normalized)) != len(normalized):
        raise ProposedComponentError("include selectors must be unique after canonical normalization")
    return tuple(sorted(normalized))


@dataclass(frozen=True)
class ProposedComponentCandidate:
    """Non-authoritative proposal for a component that need not exist yet.

    The proposal records explicit user-supplied target identity and selectors.
    It never discovers, creates, or claims the physical component exists.
    """

    proposal_id: str
    component_id: str
    include: tuple[str, ...]
    profile_path: str

    def as_dict(self) -> dict[str, object]:
        return {
            "format": "ptsip-proposed-component-candidate/v1",
            "proposal_id": self.proposal_id,
            "component_id": self.component_id,
            "include": list(self.include),
            "profile_path": self.profile_path,
            "status": "PROPOSED",
            "materialized": False,
            "authoritative": False,
        }

    def request_payload(self) -> dict[str, object]:
        return {
            "id": self.proposal_id,
            "component_id": self.component_id,
            "include": list(self.include),
            "anchors": [],
            "evidence_ids": [],
            "missing_fields": list(CANONICAL_ANSWER_FIELDS),
            "reason_codes": ["PROPOSED_COMPONENT_REQUIRES_PROJECT_DECISION"],
            "status": "INCOMPLETE",
            "origin": "EXPLICIT_PROPOSED_COMPONENT",
            "materialized": False,
            "authoritative": False,
        }

    def gate_payload(
        self,
        *,
        repository: str,
        branch: str,
        subject_revision: str,
    ) -> dict[str, object]:
        return {
            "id": self.proposal_id,
            "repository": _require_text("repository", repository),
            "branch": _require_text("branch", branch),
            "subject_revision": _require_text("subject_revision", subject_revision),
            "profile_path": self.profile_path,
            "component_id": self.component_id,
            "request": self.request_payload(),
        }


def build_proposed_component_candidate(
    repository_identity: str,
    component_id: str,
    include: tuple[str, ...] | list[str],
    profile_path: str | None = None,
) -> ProposedComponentCandidate:
    repository = _require_text("repository_identity", repository_identity)
    canonical_component = _component_id(component_id)
    canonical_include = _selectors(include)
    canonical_profile = normalize_profile_path(profile_path)
    identity = (
        repository
        + "\0"
        + canonical_profile
        + "\0"
        + canonical_component
        + "\0"
        + "\0".join(canonical_include)
    )
    digest = hashlib.sha256(identity.encode("utf-8")).hexdigest()[:20]
    return ProposedComponentCandidate(
        proposal_id=f"pcand-{digest}",
        component_id=canonical_component,
        include=canonical_include,
        profile_path=canonical_profile,
    )
