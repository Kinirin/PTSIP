# WU-06 — Target Proposals and Final PTSIP Point Plan

> **Status:** COMPLETE / FOCUSED TEST VERIFIED  
> **Target Tool:** `0.3.7`  
> **Roadmap predecessor:** WU-05 — migration analyzer and obligation evaluation (`COMPLETE / FOCUSED TEST VERIFIED`)  
> **WU-06 exact entry baseline:** `d21b499495465e637cc64a528d9b40afedb314ff`  
> **Bound Specification at entry:** `0.3.7-draft @ b648d9e026f502b14481ba2d0606d9acc88a31fc`  
> **Accepted architecture decision:** `ADR-0015 — Semantic Delta Bundle and Deterministic Reconciliation Planner`  
> **WU-06 implementation content SHA:** `c4e036aec913cdd8dc75f71ba555fccb60bf5a49`  
> **Focused verification:** `21 passed / 0 failed` in the isolated semantic-delta/planner contract harness  
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

Every proposal retains:

```text
source profile identity
source obligation/finding identity
WU-05 deterministic analysis digest
target draft/revision identity
Final Point identity
rationale/evidence links
```

Identical-looking changes from different source generations remain auditable and are not silently treated as inherited decisions.

## 3. Accepted Option C architecture

WU-06 uses the balanced architecture selected by the project owner:

```text
Semantic Delta Bundle
    +
Deterministic Reconciliation Planner
```

The authority boundary is explicit:

```text
ProposalBundle
    != AcceptedDeltaBundle
    != applied Final Point state

UnresolvedBundle
    != target decision
```

Only a proposal bound to a non-empty project-owned decision identity becomes an `AcceptedDeltaBundle`. WU-06 itself does not infer owner acceptance from proposal ordering, evidence strength, compatibility, or convenience.

### Semantic TargetDelta

Every atomic target change is one of:

```text
ADD
REMOVE
REPLACE
```

and retains:

```text
entity kind
stable entity identity
reviewed before semantics
proposed after semantics
source obligation/finding references
evidence references
```

Compound split/merge work is represented as a bundle of explicit atomic deltas rather than an opaque mutation primitive.

Semantic identity is deterministic. Mapping key order and set-like PTSIP fields such as `include`, `exclude`, and `roles` do not change the delta identity when their meaning is equivalent.

## 4. Proposal-purpose boundary

Proposal bundles distinguish:

```text
REQUIRED_MIGRATION
TARGET_VALIDITY
ASYNC_OPTIONAL
ADVISORY
```

Required-migration and target-validity bundles are blocking until accepted or explicitly resolved.

Optional Async work is not generated as an executable delta merely because an Asynchronous Work Target exists. It is included only when explicitly requested/accepted and never contributes to source completion.

Evidence conflicts/incomplete channels remain advisory context and do not become architecture authority.

## 5. No automatic target design from unresolved analysis

If WU-05 reports an unresolved Required Work Element, WU-06 does not invent:

- a lifecycle classification;
- a component grouping;
- a split/merge;
- a relationship stable ID;
- an associated-artifact shape;
- or another target declaration.

An explicit proposal/accepted delta may cover the obligation. Otherwise WU-06 emits a blocking `UnresolvedBundle`.

This preserves historical `TOOLCHAIN` ambiguity and other project-intent decisions for owner review.

## 6. Final Point identity

WU-06 distinguishes:

```text
EXISTING Final Point
PLANNED Final Point
```

When an existing Final Point is discovered by WU-01, reconciliation requires an exact semantic snapshot matching its path, draft identity, immutable Specification revision, and content SHA.

When no Final Point exists, WU-06 may plan:

```text
ptsip_<target-version>.yaml
```

without creating it. File creation remains WU-07 mutation.

For the repository's expected first self-migration this allows planning:

```text
ptsip_0.3.7.yaml
```

while the canonical `ptsip.yaml` remains untouched.

## 7. Deterministic reconciliation

Each semantic delta resolves to one of:

```text
NO_CHANGE_REQUIRED
ADD_TARGET_DECLARATION
REPLACE_WITH_EXPLICIT_OWNER_DECISION
CONFLICT_REQUIRES_CONFIRMATION
UNRESOLVED
```

Rules include:

- ADD against absent identity -> `ADD_TARGET_DECLARATION`;
- ADD against identical semantic state -> `NO_CHANGE_REQUIRED`;
- ADD against different state with the same stable identity -> conflict;
- REMOVE/REPLACE require the Final Point to match the exact reviewed `before` semantics;
- an accepted REMOVE/REPLACE against that exact state -> `REPLACE_WITH_EXPLICIT_OWNER_DECISION`;
- changed/unexpected target state -> fail closed rather than silent overwrite.

## 8. Cumulative Final Point reconciliation

The Final Point is cumulative target state.

WU-06 processes WU-01 `ordered_sources` exactly. Accepted deltas for one source are applied only to an **in-memory semantic projection** of the Final Point. No repository file is changed.

The next source is reconciled against that cumulative projected target state.

Therefore:

```text
source A accepted delta -> projected Final Point
source B accepted delta -> reconcile against projected Final Point
```

can detect stable-ID or semantic conflicts before WU-07 mutation.

This in-memory state is not an intermediate migration hop. Every source still targets the same Final PTSIP Point directly.

## 9. Required-before-Async execution preview

For each source, accepted execution candidates are ordered:

```text
Required / target-validity accepted deltas
    -> Async accepted deltas
```

Async work cannot substitute for unresolved Required Work Elements and cannot improve source-deletion eligibility.

## 10. Source deletion gate

WU-06 exposes one of:

```text
ALREADY_ELIGIBLE
REQUIRES_POST_APPLY_VERIFICATION
BLOCKED
```

`REQUIRES_POST_APPLY_VERIFICATION` is not deletion authorization. It means WU-07 may apply the accepted exact delta and must then re-prove source Required Work completion before deletion.

A source remains `BLOCKED` when, for example:

- a Required Work Element has no accepted delta or proven no-change resolution;
- a blocking proposal remains merely suggested;
- a blocking unresolved decision remains;
- an accepted delta conflicts with cumulative Final Point state;
- proposal/analysis identity binding is invalid.

## 11. WU-07 preview contract

WU-06 produces a deterministic `ExecutionPreview` containing:

- Final Point kind/path/draft/revision/content identity;
- exact WU-01 ordered source list;
- exact source content identity;
- WU-05 analysis digest;
- accepted/suggested/unresolved bundle IDs;
- Required unresolved identities;
- accepted execution delta IDs;
- reconciliation results;
- next source;
- deletion gate;
- cumulative projected Final Point semantic digest;
- blocking IDs;
- `ready_for_wu07`.

`ready_for_wu07` means only that WU-06 has no unresolved planning blocker. It does not authorize mutation or canonical promotion.

Promotion remains:

```text
WU07_GLOBAL_VALIDATION_REQUIRED
```

## 12. Work tracks

### WU-06A — target-delta model — COMPLETE

Implemented stable `TargetDelta` identity and explicit separation of suggested, accepted, and unresolved bundle authority.

### WU-06B — split/relationship/artifact proposals — COMPLETE

The semantic delta contract supports component, associated-artifact, relationship, dependency-policy, and policy entities. Compound architecture changes are modeled as bundles of atomic ADD/REMOVE/REPLACE deltas. The planner does not invent missing target intent.

### WU-06C — Final Point convergence planner — COMPLETE

Implemented WU-01 ordered source consumption, existing/planned Final Point identity, direct-to-Final-Point source steps, and cumulative in-memory target-state reconciliation.

### WU-06D — conflict and alternatives model — COMPLETE

Suggested proposals remain proposal-only, accepted deltas require a project-owned decision identity, unresolved questions remain explicit, and target-state conflicts fail closed.

### WU-06E — deterministic preview contract — COMPLETE

Implemented stable source steps, execution-delta ordering, deletion gates, blocking identities, projected Final Point digest, and deterministic plan digest for WU-07 consumption.

## 13. Implementation surfaces

WU-06 added/changed:

```text
src/ptsip/migration/proposal.py
src/ptsip/migration/planner.py
src/ptsip/migration/__init__.py
tests/ptsip/migration/test_planner_037.py
decisions/ADR-0015-semantic-delta-bundle-deterministic-reconciliation-planner.md
```

WU-06 did not modify:

```text
ptsip.yaml
ptsip_0.3.7.yaml
canonical profile schema
registry
GitHub Actions workflows
```

No Final Point file was created or mutated.

## 14. Focused verification

An isolated contract harness exercised 21 focused scenarios with:

```text
21 passed / 0 failed
```

Covered behavior includes:

- semantic delta identity independent of mapping/set-like ordering;
- acceptance requires project-owned decision identity;
- unresolved Required work is blocking;
- suggested Required proposal does not become accepted implicitly;
- accepted proposal is separated from suggested state;
- already satisfied Required work records no-change;
- missing relationship/artifact remains target-validity review;
- evidence conflict remains non-authoritative advisory context;
- Async ignored until explicitly requested;
- requested Async without delta remains optional/non-blocking;
- ADD/NO_CHANGE reconciliation;
- accepted REPLACE exact-before behavior;
- stale reviewed-before conflict;
- planned Final Point identity without file creation;
- blocking proposal prevents WU-07 readiness;
- Required accepted deltas precede Async deltas;
- existing Final Point requires exact state snapshot;
- WU-01 multi-source order preservation;
- deterministic plan digest under input collection reordering;
- direct Final Point convergence;
- cumulative cross-source stable-ID conflict detection.

This is not a claim of full repository pytest or GitHub Actions verification. Full exact-SHA regression remains WU-08 responsibility.

## 15. Non-goals preserved

WU-06 did not:

- mutate any profile;
- delete completed sources;
- rename Final Point to canonical;
- auto-resolve target conflicts;
- infer project intent from proposal ordering;
- accept lifecycle/split/merge decisions on behalf of the project owner;
- count optional Async additions as required completion.

## 16. Completion gate

WU-06 completion criteria are satisfied:

- source findings can be represented by deterministic proposals or explicit unresolved bundles;
- source identity and WU-05 analysis traceability are preserved;
- Final Point convergence order follows WU-01/ADR-0010;
- all sources converge directly to the Final Point;
- cumulative accepted target state is protected from silent overwrite;
- ambiguous/conflicting cases remain explicit;
- Required work precedes optional Async work;
- WU-07 receives a stable preview contract;
- focused contract verification passed.

WU-07 mutation has not begun.

## 17. Entry discipline

WU-06 entered automatically under the project owner's standing successor-entry authorization after WU-05 completion was recorded and exact `dev/0.3.7` HEAD `d21b499495465e637cc64a528d9b40afedb314ff` was freshly revalidated.

Completion of WU-06 does not itself authorize any proposal acceptance, Final Point mutation, source deletion, or canonical promotion. Those remain subject to WU-07's explicit authority and confirmation boundaries.
