from .analysis import (
    AmbiguousSourceElement, ArchitectureFinding, ArchitectureFindingKind, AsynchronousWorkTarget,
    DirectConvergenceAnalysis, DirectConvergenceAnalysisIssue, EvidenceCorrelation, LifecycleFinding,
    LifecycleFindingKind, MigrationAnalysis, MigrationAnalysisIssue, ObligationCategory,
    RemovalMigrationElement, RequiredWorkElement, SourceCoverageProjection, SourceMigrationCompletion,
    SourceProjectionKind, TargetArchitectureState, TargetAssociatedArtifact, TargetCompatibility,
    TargetComponent, TargetRelationship, TargetSemantics, analyze_direct_profile_convergence,
    analyze_source_migration, current_pp_target_semantics, default_target_semantics, target_state_from_mapping,
)
from .execution.authorization.binding import BoundExecutionPlan, SourceExecutionBinding, bind_execution_plan
from .execution.authorization.preconditions import verify_source_preconditions
from .execution.authorization.proof import AuthorizationProof, AuthorizedExecutionPlan, AuthorityHeadStore, authorize_execution, build_authorization_proof
from .execution.direct import (
    IdentityRewriteAuthorization, IdentityRewriteError, IdentityRewritePlan, IdentityRewriteResult,
    authorize_identity_rewrite, build_identity_rewrite_plan, build_legacy_target_identity_rewrite_plan,
    execute_identity_rewrite, prepare_direct_promotion, verify_direct_post_promotion,
)
from .execution.error import ExecutionStateError
from .execution.guard.mutation import MutationGuardExpectation, capture_mutation_guard
from .execution.guard.repository_snapshot import RepositorySnapshotExpectation
from .execution.ledger import CheckpointLedger, CheckpointRecord, LedgerIntegrityError, default_ledger_root
from .execution.mutation import CompletionCallback, apply_optional_async_deltas, apply_required_deltas, complete_source, finalize_source, reanalyze_source
from .execution.promotion import prepare_promotion, promote_canonical, verify_post_promotion
from .execution.recovery import inspect_recovery
from .execution.state.phase import ExecutionPhase
from .execution.state.promotion import PostPromotionVerifiedState, PromotedState, PromotionReadyState
from .execution.state.recovery import RecoveryInspection, RecoveryRequiredState
from .execution.state.source_steps import (
    AppliedSourceStep, AsyncAppliedSourceStep, CanonicalSourceComplete, CompletedSourceStep,
    ReanalyzedSourceStep, RemovedTemporarySourceStep, SourceCompletionProof, VerifiedSourceStep,
)
from .planning import (
    DeletionGate, ExecutionPreview, FinalPointConvergencePlan, FinalPointEntity, FinalPointKind,
    FinalPointReference, FinalPointStateSnapshot, PlanningIssue, ReconciliationResult,
    ReconciliationStatus, SourceConvergencePlan, build_final_point_convergence_plan,
    derive_source_proposals, final_point_state_from_mapping, reconcile_delta,
)
from .planning.direct import DirectFinalPointReference, build_direct_final_point_convergence_plan
from .proposal import (
    AcceptedDeltaBundle, DeltaChangeKind, ProposalAuthority, ProposalBundle, ProposalPurpose,
    SourceProposalSet, TargetDelta, TargetEntityKind, UnresolvedBundle,
)

ExecutionState = (
    BoundExecutionPlan | AuthorizedExecutionPlan | VerifiedSourceStep | AppliedSourceStep |
    ReanalyzedSourceStep | CompletedSourceStep | AsyncAppliedSourceStep | RemovedTemporarySourceStep |
    CanonicalSourceComplete | PromotionReadyState | PromotedState | PostPromotionVerifiedState | RecoveryRequiredState
)

__all__ = [
    "AcceptedDeltaBundle",
    "AmbiguousSourceElement",
    "AppliedSourceStep",
    "ArchitectureFinding",
    "ArchitectureFindingKind",
    "AsynchronousWorkTarget",
    "AsyncAppliedSourceStep",
    "AuthorizationProof",
    "AuthorizedExecutionPlan",
    "AuthorityHeadStore",
    "BoundExecutionPlan",
    "CanonicalSourceComplete",
    "CheckpointLedger",
    "CheckpointRecord",
    "CompletedSourceStep",
    "CompletionCallback",
    "DeletionGate",
    "DeltaChangeKind",
    "DirectConvergenceAnalysis",
    "DirectConvergenceAnalysisIssue",
    "DirectFinalPointReference",
    "EvidenceCorrelation",
    "ExecutionPhase",
    "ExecutionPreview",
    "ExecutionState",
    "ExecutionStateError",
    "FinalPointConvergencePlan",
    "FinalPointEntity",
    "FinalPointKind",
    "FinalPointReference",
    "FinalPointStateSnapshot",
    "IdentityRewriteAuthorization",
    "IdentityRewriteError",
    "IdentityRewritePlan",
    "IdentityRewriteResult",
    "LedgerIntegrityError",
    "LifecycleFinding",
    "LifecycleFindingKind",
    "MigrationAnalysis",
    "MigrationAnalysisIssue",
    "MutationGuardExpectation",
    "ObligationCategory",
    "PlanningIssue",
    "PostPromotionVerifiedState",
    "PromotedState",
    "PromotionReadyState",
    "ProposalAuthority",
    "ProposalBundle",
    "ProposalPurpose",
    "ReanalyzedSourceStep",
    "ReconciliationResult",
    "ReconciliationStatus",
    "RecoveryInspection",
    "RecoveryRequiredState",
    "RemovalMigrationElement",
    "RemovedTemporarySourceStep",
    "RepositorySnapshotExpectation",
    "RequiredWorkElement",
    "SourceCompletionProof",
    "SourceConvergencePlan",
    "SourceCoverageProjection",
    "SourceExecutionBinding",
    "SourceMigrationCompletion",
    "SourceProjectionKind",
    "SourceProposalSet",
    "TargetArchitectureState",
    "TargetAssociatedArtifact",
    "TargetCompatibility",
    "TargetComponent",
    "TargetDelta",
    "TargetEntityKind",
    "TargetRelationship",
    "TargetSemantics",
    "UnresolvedBundle",
    "VerifiedSourceStep",
    "analyze_direct_profile_convergence",
    "analyze_source_migration",
    "apply_optional_async_deltas",
    "apply_required_deltas",
    "authorize_execution",
    "authorize_identity_rewrite",
    "bind_execution_plan",
    "build_authorization_proof",
    "build_direct_final_point_convergence_plan",
    "build_identity_rewrite_plan",
    "build_legacy_target_identity_rewrite_plan",
    "capture_mutation_guard",
    "complete_source",
    "current_pp_target_semantics",
    "default_ledger_root",
    "default_target_semantics",
    "derive_source_proposals",
    "execute_identity_rewrite",
    "final_point_state_from_mapping",
    "finalize_source",
    "inspect_recovery",
    "prepare_direct_promotion",
    "prepare_promotion",
    "promote_canonical",
    "reanalyze_source",
    "reconcile_delta",
    "target_state_from_mapping",
    "verify_direct_post_promotion",
    "verify_post_promotion",
    "verify_source_preconditions",
]
