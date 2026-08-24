# WU-07 — Preview, Confirmation, Safe Sequential Apply, and Canonical Promotion

> **Status:** PRE-CREATED / LOCKED  
> **Target Tool:** `0.3.7`  
> **Roadmap predecessor:** WU-06 — target proposals and Final Point plan  
> **Entry baseline:** not assigned; capture fresh branch HEAD on actual entry  
> **Successor:** WU-08 — dogfood, regression, package and release readiness

## 0. Purpose

Own the only architecture-mutating boundary of the Tool `0.3.7` migration pipeline.

WU-07 applies accepted source-specific target deltas directly into the Final PTSIP Point File, removes completed temporary sources in the required order, migrates canonical `ptsip.yaml` last, and performs guarded final promotion.

## 1. Mutation pipeline

```text
WU-06 deterministic plan
    -> project-owner preview
    -> explicit confirmation where architecture decisions are required
    -> exact source/final-point snapshot validation
    -> apply minimum accepted delta to Final Point
    -> deterministic post-apply validation
    -> prove current source Required Work Elements complete
    -> delete current temporary source when eligible
    -> advance to next source
    -> canonical ptsip.yaml last
    -> global completion validation
    -> remove old canonical source
    -> rename Final Point -> ptsip.yaml
```

Mutation must stop at the first stale, conflicting, or incomplete boundary.

## 2. Source order enforcement

WU-07 must use the WU-01/WU-06 ordered source list and must not invent another sequence.

For Sequential Work:

```text
temporary sources: highest to lowest, excluding Final Point
canonical ptsip.yaml: always last
```

A removed temporary source may never become an intermediate migration destination or hop.

## 3. Per-source completion gate

Before deleting a temporary source, WU-07 must prove:

```text
source identity unchanged since accepted analysis
AND Final Point identity is the expected current state
AND source required_unresolved == 0 after apply
AND target validation for represented source obligations passes
AND no unresolved accepted-target conflict remains
```

Removal Migration Elements and Asynchronous Work Targets do not affect this completion gate.

If any Required Work Element is unresolved, source deletion is forbidden.

## 4. Canonical source special rule

`ptsip.yaml` is migrated last and is not deleted immediately after its local source completion if global Final Point validation has not yet succeeded.

The final boundary is:

```text
all temporary sources completed/removed
canonical source Required Work Elements completed
Final Point validates against target draft/revision
no unresolved transition conflicts
expected snapshots still current
    -> canonical promotion allowed
```

Only then:

```text
remove old ptsip.yaml
rename Final Point -> ptsip.yaml
```

The implementation must minimize the interval where the canonical path is absent and must fail safely if filesystem operations cannot complete as planned.

## 5. Project-owner authority

Evidence and deterministic migration mechanics do not own architecture decisions.

WU-07 requires explicit project-owner confirmation for target deltas that change architecture intent, including ambiguous lifecycle reclassification, component split/merge, conflict replacement, or other semantics not already authorized by a prior accepted decision.

Purely mechanical application of an already accepted exact delta does not require inventing an additional architecture decision.

## 6. Stale-state and conflict rules

Before every mutation stage, validate the exact expected identities from WU-06.

Fail closed for at least:

- changed source profile;
- changed repository snapshot where the accepted analysis depends on it;
- changed Final Point content not represented by the accepted plan;
- target draft/revision mismatch;
- unresolved required obligation;
- conflict with accepted Final Point target state;
- source ordering mismatch;
- missing expected source;
- attempted promotion while another participating source remains.

Automatic broad re-analysis after a stale-state failure must not silently apply a new plan. Re-analysis returns to preview/confirmation where target intent can change.

## 7. Optional Async additions

If the owner requested Asynchronous Work Targets after Required Work Elements were handled, WU-07 may apply those separately and visibly.

They must never:

- make an incomplete source appear complete;
- delay source completion without explicit owner choice;
- be required for source deletion;
- be silently inherited into another source's obligation taxonomy.

## 8. Work tracks

### WU-07A — preview and authorization contract

Render source order, required completion, proposed deltas, optional work, conflicts, and promotion effects deterministically.

### WU-07B — stale-state guard

Reuse existing snapshot/digest/CAS-style validation primitives where possible. Do not create a parallel weak write mechanism.

### WU-07C — minimum-delta Final Point apply

Apply only the accepted target delta. Avoid unrelated profile rewriting.

### WU-07D — source completion and deletion

Prove required completion before deleting a temporary source.

### WU-07E — canonical-last migration

Handle root `ptsip.yaml` only after every temporary source has converged.

### WU-07F — guarded promotion

Validate global completion and perform Final Point -> canonical promotion safely.

### WU-07G — interruption/retry behavior

Ensure interruption at any stage produces a discoverable, deterministic transition state on the next run instead of corrupting source ordering.

## 9. Required tests

At minimum:

- accepted simple transition apply;
- multi-source ordered apply;
- wrong source order rejected;
- incomplete Required Work Element blocks deletion;
- Removal element does not block deletion;
- Async work does not affect completion;
- stale source blocks mutation;
- stale Final Point blocks mutation;
- accepted target conflict blocks silent overwrite;
- interruption after one temporary source deletion resumes from remaining sources;
- canonical source remains last;
- promotion blocked while any source remains;
- successful final promotion produces exactly one canonical `ptsip.yaml` for the target draft.

## 10. Non-goals

WU-07 does not define new lifecycle ontology, invent source obligation semantics, or bypass owner authority for convenience. It does not treat promotion success as release readiness; WU-08 owns full verification.

## 11. Completion gate

WU-07 is complete only when the full ordered mutation path is deterministic, snapshot-guarded, owner-authorized where required, source-completion aware, interruption-safe, and incapable of promoting a partial Final Point.

## 12. Entry discipline

Pre-created roadmap document only. Actual entry requires WU-06 completion and a fresh exact branch HEAD.
