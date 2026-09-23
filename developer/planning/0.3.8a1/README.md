# PTSIP Tool 0.3.8a1 — Emergency Implementation Entry Point

## Read this first

`dev/0.3.8a1` is **not** an implementation branch for the complete Tool 0.4.0 policy set.

It is a narrowly scoped emergency prerelease branch created from `dev/0.4.0` so that one release-blocking capability can be implemented and shipped without waiting for the remaining 0.4.0 architecture to be implemented.

## What is implemented here

The implementation target for this branch is limited to:

```text
not-yet-existing component
        ↓
explicit proposed candidate registration
        ↓
machine-readable stable proposal identity
        ↓
existing Decision Control Plane resolution
        ↓
PROPOSAL_APPROVED
        ↓
no physical component creation
no active components[] projection before materialization
```

Primary runtime surfaces:

- `ptsip propose-component`
- `ptsip resolve` for an `EXPLICIT_PROPOSED_COMPONENT`
- terminal application state `PROPOSAL_APPROVED`

## What is NOT implemented by this branch

The presence of 0.4.0 policy documents in this branch does **not** mean their runtime implementation is complete.

In particular, Tool 0.3.8a1 does not claim implementation completion for:

- the final S1 CandidateProvenance-based candidate engine;
- the complete CanonicalCapability runtime model;
- Capability Satisfaction runtime composition;
- MachineAssertionContract registry/runtime evaluation;
- the complete RecoveryPlan / RecoveryPath runtime;
- CapabilityRecoveryAssessment / RecoveryAttempt final contracts;
- the complete S3 Resolution Classification runtime;
- the complete Tool 0.4.0 work-unit set.

Those remain Tool 0.4.0 development work unless a later implementation record explicitly states otherwise.

## Why 0.4.0 policy documents are present

This branch was forked from `dev/0.4.0`.

The 0.4.0 documents are retained as **semantic design authority and implementation constraints** so that the emergency implementation does not contradict the future architecture.

They are not evidence that all described 0.4.0 structures exist in runtime code.

Use this rule:

```text
0.4.0 policy present
        ≠
0.4.0 runtime implemented
```

## Document navigation

For Tool 0.3.8a1 work, use this order:

1. `developer/planning/0.3.8a1/README.md` — branch scope and implementation boundary.
2. `developer/planning/0.3.8a1/emergency-implementation-overlay.yaml` — machine-readable implementation scope.
3. `planning/0.3.8/0.3.8a1-emergency-release-gate.yaml` — emergency feature acceptance and verification gate.
4. `releasenote/tool/0.3.8a1.md` — user-visible prerelease scope.

Consult `developer/planning/0.4.0/**` only for inherited semantic constraints or future 0.4.0 work. Do not infer implementation completion from those documents.

## Integration boundary

After the 0.3.8a1 emergency implementation is accepted:

```text
dev/0.3.8a1
        ↓
dev/0.4.0
        ↓
restore active planning context to Tool 0.4.0
        ↓
continue unfinished 0.4.0 implementation
```

Release/publication is a separate later boundary. Merging the emergency implementation does not itself assert that Tool 0.4.0 is implemented or release-ready.
