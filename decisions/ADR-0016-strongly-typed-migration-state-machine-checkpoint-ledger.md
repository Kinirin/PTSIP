# ADR-0016 — Strongly Typed Migration State Machine and Append-Only Checkpoint Ledger

**Status:** Accepted  
**Decision date:** 2026-08-24  
**Target Tool:** `0.3.7`  
**Governing work unit:** WU-07 — Preview, Confirmation, Safe Sequential Apply, and Canonical Promotion  
**Bound Specification:** `0.3.7-draft @ b648d9e026f502b14481ba2d0606d9acc88a31fc`

## Context

WU-06 produces a deterministic, non-mutating convergence plan that separates proposal authority from accepted target deltas and records exact source identities, accepted delta identities, deletion gates, and cumulative projected Final Point semantics.

WU-07 is the first and only Tool `0.3.7` boundary that may mutate profile files. It must therefore prevent a broad class of unsafe states:

- applying a proposal that was never accepted;
- applying an accepted delta against a stale `before` state;
- deleting a source before its Required Work Elements are re-proven complete;
- allowing expected WU-07 profile mutations to hide unrelated repository drift;
- resuming after interruption by guessing which mutation completed;
- migrating canonical `ptsip.yaml` before temporary sources;
- promoting a Final Point that differs from the accepted WU-06 projection;
- treating filesystem write success as equivalent to successful promotion.

The project owner selected the long-term-maintainability architecture: a strongly typed migration state machine plus an append-only checkpoint ledger.

## Considered architectures

### Option A — Transaction Graph Executor + Recoverable Mutation Journal

Represent every mutation as a generic operation graph with dependencies, preconditions, postconditions, and rollback metadata.

**Strengths**

- broad applicability beyond PTSIP;
- fine-grained operation recovery;
- expressive dependency modeling.

**Weaknesses**

- generic operation legality is weaker than PTSIP domain-state legality;
- architecture dependencies and executor dependencies can become conflated;
- PTSIP-specific deletion/promotion constraints require a parallel constraint layer.

### Option B — Strongly Typed Migration State Machine + Append-Only Checkpoint Ledger

Model the legal migration lifecycle as explicit states and transitions. Persist every completed transition as an immutable, digest-chained checkpoint.

**Strengths**

- source deletion and promotion legality are explicit state invariants;
- interruption recovery can distinguish verified states from merely observed filesystem state;
- PTSIP-specific sequencing is structurally enforced;
- audit and retry behavior are deterministic.

**Weaknesses**

- future mutation stages require explicit state-graph evolution;
- filesystem writes and checkpoint persistence are not one ACID transaction, so crash-window reconciliation is still required.

### Option C — Guarded Apply Session + CAS Checkpoints + Deterministic Recovery

Use a smaller phase-oriented session around WU-06 semantic deltas and source-level checkpoints.

**Strengths**

- direct continuity with WU-06;
- strong semantic CAS behavior with a compact execution model.

**Weaknesses**

- audit granularity is weaker than a typed checkpoint chain;
- complex recovery can gradually recreate the state-machine model implicitly.

## Decision

### 1. Typed execution states

WU-07 defines explicit states including at least:

```text
PLAN_BOUND
AUTHORIZED
PRECONDITIONS_VERIFIED
FINAL_POINT_APPLIED
SOURCE_REANALYZED
SOURCE_COMPLETE
ASYNC_APPLIED
SOURCE_REMOVED
CANONICAL_SOURCE_COMPLETE
GLOBAL_VALIDATION
PROMOTION_READY
PROMOTED
POST_PROMOTION_VERIFIED
RECOVERY_REQUIRED
```

The executor exposes typed state objects so invalid transitions are not represented as normal control flow.

Examples:

```text
FINAL_POINT_APPLIED
    -X-> SOURCE_REMOVED

FINAL_POINT_APPLIED
    -> SOURCE_REANALYZED
    -> SOURCE_COMPLETE
    -> SOURCE_REMOVED
```

and:

```text
CANONICAL_SOURCE_COMPLETE
    -> GLOBAL_VALIDATION
    -> PROMOTION_READY
    -> PROMOTED
    -> POST_PROMOTION_VERIFIED
```

### 2. WU-06 plan binding is exact

Before mutation, WU-07 binds to one immutable WU-06 plan digest.

Binding requires:

- `ExecutionPreview.ready_for_wu07 == true`;
- no WU-06 blocking IDs or planning issues;
- exact source order agreement;
- exact source content SHA and WU-05 analysis digest;
- accepted bundle identity agreement;
- current repository snapshot agreement with the accepted source analyses.

If any input is stale, WU-07 does not rewrite or repair the plan automatically.

### 3. Accepted decisions only

Only `AcceptedDeltaBundle` objects may become execution deltas.

An authorization proof binds:

```text
WU-06 plan digest
accepted project decision IDs
authority revision
```

The accepted decision set must exactly match the execution plan's accepted decision identities.

Where a coordinated authority store is supplied, its current HEAD must still equal the authorization revision. Existing GitHub non-force CAS semantics are reused rather than creating a weaker parallel authority mechanism.

### 4. Controlled mutation scope

The WU-05 analysis snapshot is validated at session bind time.

After mutation begins, expected profile edits would intentionally change the repository-wide snapshot. WU-07 therefore establishes a session mutation guard over all repository content **except** the exact controlled profile paths:

```text
all participating source profile paths
Final Point path
```

Every stage separately verifies exact source and Final Point SHAs. The scoped guard therefore detects unrelated repository drift without falsely rejecting expected WU-07 profile changes.

The local checkpoint ledger is stored outside observed repository content: under `.git/ptsip/migration/` for Git repositories and under the user's PTSIP state directory for non-Git repositories.

### 5. Append-only checkpoint ledger

Each completed state transition is persisted as an immutable checkpoint.

A checkpoint records at least:

```text
sequence
phase
plan digest
source path / source SHA
Final Point before / after SHA
analysis digest
decision IDs
repository snapshot where relevant
phase payload
previous checkpoint digest
checkpoint digest
```

Checkpoint files are sequence-numbered and digest-chained. Existing checkpoints are never overwritten.

A corrupt, missing, reordered, or modified checkpoint makes recovery fail closed.

### 6. Per-source atomic Final Point write

For one source, accepted Required/target-validity deltas are applied to an in-memory target mapping first.

Every delta enforces semantic compare-and-set behavior:

```text
current target entity == accepted delta.before
    -> replace/remove allowed

ADD target identity absent
    -> add allowed

otherwise
    -> stale/conflict; no silent overwrite
```

After all required deltas are prepared, WU-07 writes the Final Point once using a sibling temporary file plus `os.replace`.

This creates one filesystem mutation boundary per source rather than exposing intermediate per-delta states.

### 7. Planned Final Point creation requires an explicit seed

If WU-06 planned a Final Point that does not yet exist, WU-07 does not invent the target profile scaffold.

The caller must provide a target-draft seed mapping whose:

```text
ptsip.version
ptsip.specification.revision
```

exactly match the WU-06 Final Point reference.

The seed is then modified only by accepted semantic deltas.

### 8. Post-apply source completion proof

A successful Final Point write does not authorize source deletion.

WU-07 requires a post-apply completion proof for the current source. The proof must establish:

```text
required_unresolved == 0
AND target-validity checks succeeded
```

Removal Migration Elements and Asynchronous Work Targets do not contribute to this proof.

If Required work remains unresolved, the state machine cannot enter `SOURCE_COMPLETE`.

### 9. Async work is a separate state

Accepted `ASYNC_OPTIONAL` deltas may be applied only after `SOURCE_COMPLETE`.

They are checkpointed separately and are explicitly marked as non-contributing to source completion.

A source may be deletion-eligible regardless of whether optional Async work exists, subject to the exact accepted execution plan.

### 10. Temporary source deletion and canonical-last rule

A temporary source may be removed only after:

- post-apply source completion proof;
- exact source SHA revalidation;
- exact Final Point semantic digest validation against the WU-06 per-source projected state.

After deletion, WU-01 transition rediscovery must remain valid.

Canonical `ptsip.yaml` is never deleted through the temporary-source transition. Its completed state is represented separately as `CANONICAL_SOURCE_COMPLETE`.

### 11. Promotion is an independent state boundary

Promotion requires:

```text
all temporary participating sources removed
canonical source is the final source
canonical source still matches its accepted SHA
Final Point still matches its accepted SHA
Final Point semantic digest == WU-06 global projected digest
Final Point draft/revision == WU-06 target identity
WU-01 rediscovery reports canonical as the only remaining source
no unrelated repository drift
```

Only then may the state enter `PROMOTION_READY`.

Promotion uses same-filesystem `os.replace(Final Point, ptsip.yaml)` to minimize the interval in which the canonical path could be absent. This is the physical implementation of guarded replacement; it does not weaken the canonical-last semantic rule.

### 12. Promotion success requires rediscovery

`PROMOTED` is not terminal success.

After replacement, WU-07 must prove:

- canonical `ptsip.yaml` exists with the target content;
- the old Final Point path no longer exists;
- WU-01 transition rediscovery is valid;
- no Final Point remains selected;
- canonical draft/revision exactly match the promoted target.

Only then is `POST_PROMOTION_VERIFIED` reached.

Failure enters `RECOVERY_REQUIRED` rather than silently restoring, retrying, or re-planning.

### 13. Deterministic recovery

On restart, WU-07 reads the digest-chained ledger and compares it with actual source/Final Point/canonical hashes and the scoped repository guard.

If physical state matches the latest checkpoint, the session is safe to resume from the next legal state.

If physical state is ahead of, behind, or incompatible with the latest checkpoint, WU-07 returns `RECOVERY_REQUIRED`.

In particular, a crash after filesystem replacement but before checkpoint persistence is detected as a mismatch rather than guessed to be successful.

Automatic broad re-analysis never applies a new target plan. If re-analysis changes architecture intent, control returns to WU-06 preview/owner confirmation.

## Consequences

### Positive

- illegal source-deletion and promotion paths are structurally visible;
- every accepted mutation remains traceable to exact WU-06 and project-owner identities;
- unrelated repository drift is distinguished from planned profile mutation;
- one-source atomic Final Point writes avoid per-delta partial target states;
- interruption recovery is deterministic and auditable;
- canonical promotion has a separate precondition and postcondition boundary;
- the existing GitHub authority CAS model can participate without making migration core depend on the app/control-plane implementation.

### Tradeoffs

- state evolution is explicit and future execution phases require deliberate model changes;
- the checkpoint ledger adds durable local transition state that must be integrity-checked;
- filesystem mutation and ledger append cannot be a single transaction, so `RECOVERY_REQUIRED` remains necessary for crash windows;
- text serialization may rewrite YAML presentation while semantic delta validation prevents unrelated target-architecture changes; byte-preserving YAML editing is not made an architecture authority requirement.

## Non-authority statement

This ADR approves the WU-07 executor architecture only.

It does **not** authorize any concrete target delta, create or modify `ptsip_0.3.7.yaml`, delete a source profile, alter canonical `ptsip.yaml`, or perform canonical promotion in the PTSIP repository. Concrete architecture-changing execution still requires the accepted project-owned decision identities carried by WU-06/WU-07 authorization.