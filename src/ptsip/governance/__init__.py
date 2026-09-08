"""Machine-readable Project Authority runtime.

The governance runtime consumes canonical decisions/, schemas/, and current solve
inputs. Design-time planning artifacts are never runtime authority.
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
