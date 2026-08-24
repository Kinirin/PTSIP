# WU-06 — Target Proposals and Final PTSIP Point Plan

> **Status:** ACTIVE  
> **Target Tool:** `0.3.7`  
> **Roadmap predecessor:** WU-05 — migration analyzer and obligation evaluation (`COMPLETE / FOCUSED TEST VERIFIED`)  
> **WU-06 exact entry baseline:** `d21b499495465e637cc64a528d9b40afedb314ff`  
> **Bound Specification at entry:** `0.3.7-draft @ b648d9e026f502b14481ba2d0606d9acc88a31fc`  
> **Successor:** WU-07 — safe sequential apply and promotion

## 0. Purpose

Transform source-specific migration findings into explicit, reviewable, non-mutating target-architecture deltas and one deterministic plan for convergence into the Final PTSIP Point File.

This retains the cancelled Tool `0.3.6.1` WU-05 proposal boundary and adds Sequential Work planning.

## 1. Proposal categories

WU-06 may propose:

- target lifecycle classification;
- component split or justified merge;
- typed relationships;
- associated artifacts;
- selector/include changes;
- explicit removals required by target semantics;
- source obligations already satisfied by accepted Final Point state;
- unresolved questions/alternatives;
- optional Asynchronous Work Target additions requested by the project owner.

A proposal remains non-authoritative until accepted.

## 2. Source-specific proposal identity

Every proposal must retain:

```text
source profile identity
source obligation identity
repository snapshot identity
target draft/revision identity
Final Point identity
rationale/evidence links
```

This ensures identical-looking changes from different source generations remain auditable and are not silently treated as inherited decisions.

## 3. Final Point plan

WU-06 must build one deterministic, non-mutating execution plan from WU-01 transition state.

Example:

```text
Final Point: ptsip_0.4.0.yaml
Sources:
  1. ptsip_0.3.7.yaml
  2. ptsip_0.3.6.yaml
  3. ptsip.yaml
```

For each source, the plan records:

- exact source identity and expected content/snapshot identity;
- Required Work Elements and unresolved count;
- accepted/proposed target deltas;
- conflicts requiring owner resolution;
- whether optional Async additions are requested;
- post-source deletion eligibility;
- expected next source.

The plan must never insert a deleted/obsolete intermediate hop.

## 4. Final Point target-state rule

The Final Point is cumulative target state.

When a later source migration is planned, WU-06 compares its proposed delta with accepted target declarations already present in the Final Point.

Possible outcomes include:

```text
NO_CHANGE_REQUIRED
ADD_TARGET_DECLARATION
REPLACE_WITH_EXPLICIT_OWNER_DECISION
CONFLICT_REQUIRES_CONFIRMATION
UNRESOLVED
```

No silent last-writer-wins behavior is allowed.

## 5. Async-work ordering

Required Work Elements always precede Asynchronous Work Targets for the same source.

The default plan must not interleave optional Async work in a way that delays or obscures required migration completion.

Async additions are included only when explicitly requested/accepted and must be visibly marked non-contributing to source completion.

## 6. Work tracks

### WU-06A — target-delta model

Define stable proposal identity and difference between source obligation, suggested target state, accepted target state, and unresolved question.

### WU-06B — split/relationship/artifact proposals

Preserve the richer migration capabilities from the cancelled plan without inventing unnecessary components.

### WU-06C — Final Point convergence planner

Generate source order and direct-to-Final-Point target deltas from WU-01/WU-05 outputs.

### WU-06D — conflict and alternatives model

Preserve multiple plausible targets or require project-owner resolution.

### WU-06E — deterministic preview contract

Produce a stable plan representation suitable for WU-07 preview/confirmation and stale-state validation.

## 7. Non-goals

WU-06 does not:

- mutate any profile;
- delete completed sources;
- rename Final Point to canonical;
- auto-resolve target conflicts;
- infer project intent from proposal ordering;
- count optional Async additions as required completion.

## 8. Completion gate

WU-06 is complete only when:

- each source finding can become a deterministic non-mutating proposal;
- source identity and obligation traceability are preserved;
- Final Point convergence order follows ADR-0010 exactly;
- all sources target the Final Point directly;
- accepted target state is protected from silent overwrite;
- ambiguous/conflicting cases remain explicit;
- Required work precedes optional Async work;
- the execution preview is stable enough for WU-07;
- focused tests pass.

## 9. Entry discipline

WU-06 entered automatically under the project owner's standing successor-entry authorization after WU-05 completion was recorded and exact `dev/0.3.7` HEAD `d21b499495465e637cc64a528d9b40afedb314ff` was freshly revalidated.

This ACTIVE state authorizes WU-06 implementation only. It does not authorize owner acceptance of target proposals, WU-07 mutation, source deletion, Final Point mutation, or canonical promotion.
