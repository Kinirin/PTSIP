"""Owner-intent/external-fact input boundaries and mandatory Fresh Solve restart contracts."""

from .contract import (
    EscalationChoice,
    EscalationContractError,
    ExternalFactRequirement,
    FreshSolveBinding,
    FreshSolveBindingComparison,
    FreshSolveHandoff,
    FreshSolveRestartPolicy,
    InputMaterializationTarget,
    ResolutionInputKind,
    ResolutionInputRequest,
)
from .engine import (
    build_resolution_input_request,
    compare_fresh_solve_bindings,
    prepare_fresh_solve_handoff,
    require_current_resolution_input_request,
)

__all__ = [
    "EscalationChoice",
    "EscalationContractError",
    "ExternalFactRequirement",
    "FreshSolveBinding",
    "FreshSolveBindingComparison",
    "FreshSolveHandoff",
    "FreshSolveRestartPolicy",
    "InputMaterializationTarget",
    "ResolutionInputKind",
    "ResolutionInputRequest",
    "build_resolution_input_request",
    "compare_fresh_solve_bindings",
    "prepare_fresh_solve_handoff",
    "require_current_resolution_input_request",
]
