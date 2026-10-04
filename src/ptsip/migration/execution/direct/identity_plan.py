from __future__ import annotations

import hashlib
import json
from dataclasses import dataclass

from ptsip.repository.profile_convergence import DirectConvergenceMode, DirectConvergenceState
from ptsip.repository.snapshot import RepositorySnapshot

class IdentityRewriteError(RuntimeError):
    def __init__(self, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code

@dataclass(frozen=True)
class IdentityRewritePlan:
    source_path: str
    source_sha256: str
    source_declared_version: str
    target_contract: str
    specification_family: str
    specification_revision: str
    repository_snapshot: RepositorySnapshot

    def content_payload(self) -> dict[str, object]:
        return {
            "kind": "PP_IDENTITY_ONLY_REWRITE",
            "source_path": self.source_path,
            "source_sha256": self.source_sha256,
            "source_declared_version": self.source_declared_version,
            "target_contract": self.target_contract,
            "specification_family": self.specification_family,
            "specification_revision": self.specification_revision,
            "repository_snapshot": self.repository_snapshot.as_dict(),
        }

    @property
    def deterministic_digest(self) -> str:
        encoded = json.dumps(
            self.content_payload(),
            sort_keys=True,
            separators=(",", ":"),
            ensure_ascii=False,
        ).encode("utf-8")
        return hashlib.sha256(encoded).hexdigest()

    def as_dict(self) -> dict[str, object]:
        payload = self.content_payload()
        payload["deterministic_digest"] = self.deterministic_digest
        return payload

@dataclass(frozen=True)
class IdentityRewriteAuthorization:
    plan_digest: str
    authority_revision: str
    authorization_id: str

    def as_dict(self) -> dict[str, str]:
        return {
            "plan_digest": self.plan_digest,
            "authority_revision": self.authority_revision,
            "authorization_id": self.authorization_id,
        }

@dataclass(frozen=True)
class IdentityRewriteResult:
    source_path: str
    before_sha256: str
    after_sha256: str
    source_declared_version: str
    target_contract: str
    specification_family: str
    specification_revision: str
    validation_warnings: tuple[str, ...]

    def as_dict(self) -> dict[str, object]:
        return {
            "source_path": self.source_path,
            "before_sha256": self.before_sha256,
            "after_sha256": self.after_sha256,
            "source_declared_version": self.source_declared_version,
            "target_contract": self.target_contract,
            "specification_family": self.specification_family,
            "specification_revision": self.specification_revision,
            "validation_warnings": list(self.validation_warnings),
        }

def build_identity_rewrite_plan(state: DirectConvergenceState) -> IdentityRewritePlan:
    if state.mode is not DirectConvergenceMode.IDENTITY_ONLY:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_NOT_APPLICABLE",
            "Identity rewrite planning requires an IDENTITY_ONLY direct-convergence state.",
        )
    if state.source.path != state.target_path or state.requires_temporary_target:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_NOT_IN_PLACE",
            "IDENTITY_ONLY canonical transition must be an in-place rewrite without a temporary target.",
        )
    if state.target is not None:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_AMBIGUOUS_TARGET",
            "In-place identity rewrite must not have a separate target binding.",
        )
    return IdentityRewritePlan(
        source_path=state.source.path,
        source_sha256=state.source.content_sha256,
        source_declared_version=state.source.declared_version,
        target_contract=state.target_contract.canonical,
        specification_family=state.source.declared_version,
        specification_revision=state.source.specification_revision,
        repository_snapshot=state.snapshot,
    )

def authorize_identity_rewrite(
    plan: IdentityRewritePlan,
    *,
    authority_revision: str,
) -> IdentityRewriteAuthorization:
    revision = authority_revision.strip()
    if not revision:
        raise IdentityRewriteError(
            "PP_IDENTITY_REWRITE_AUTHORITY_REQUIRED",
            "Real-project identity rewrite requires a non-empty owner authority revision.",
        )
    token = hashlib.sha256(
        f"{plan.deterministic_digest}\0{revision}".encode("utf-8")
    ).hexdigest()[:24]
    return IdentityRewriteAuthorization(
        plan_digest=plan.deterministic_digest,
        authority_revision=revision,
        authorization_id=f"pp-identity-rewrite:{token}",
    )

