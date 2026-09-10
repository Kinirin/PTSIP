"""Machine-readable Support Feature authority runtime.

Product runtime consumes shipped SFP policies, shipped support registries, and
current solve inputs. Developer policy, planning artifacts, owner grants, and
legacy decisions are not product-runtime authority inputs.
"""

from .authority import AuthorityCatalog, ProjectAuthorityRuntime
from .authorization import AuthorizationTransitionEvaluator
from .model import (
    AuthorizationState,
    AuthorizationTransitionResult,
    CheckStatus,
    EligibilityCheck,
    EligibilityResult,
    EligibilityStatus,
    GovernanceAuthorityError,
    LifecycleState,
    ProjectAuthorityRecord,
    RepositoryBinding,
    SolveSubject,
    SubjectIdentity,
    SubjectMatchKind,
)
from .subject_matching import match_subject_binding

__all__ = [
    "AuthorityCatalog",
    "AuthorizationState",
    "AuthorizationTransitionEvaluator",
    "AuthorizationTransitionResult",
    "CheckStatus",
    "EligibilityCheck",
    "EligibilityResult",
    "EligibilityStatus",
    "GovernanceAuthorityError",
    "LifecycleState",
    "ProjectAuthorityRecord",
    "ProjectAuthorityRuntime",
    "RepositoryBinding",
    "SolveSubject",
    "SubjectIdentity",
    "SubjectMatchKind",
    "match_subject_binding",
]
