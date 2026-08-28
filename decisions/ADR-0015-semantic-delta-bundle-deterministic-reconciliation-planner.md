# ADR-0015 — Semantic Delta Bundle and Deterministic Reconciliation Planner

**Status:** Accepted  
**Decision date:** 2026-08-24  
**Target Tool:** `0.3.7`  
**Governing work unit:** WU-06 — Target Proposals and Final PTSIP Point Plan  
**Bound Specification:** `0.3.7-draft @ b648d9e026f502b14481ba2d0606d9acc88a31fc`

## Context

WU-05 produces source-specific migration analysis in which Required/Removal/Async obligation taxonomy, lifecycle findings, normalized evidence, and accepted-target compatibility remain separate typed axes. WU-06 must transform those findings into reviewable target changes and a deterministic Final Point convergence plan without turning analysis into architecture authority.

The critical authority boundary is:

```text
WU-05 finding
    != project-owner decision

proposal
    != accepted target delta

accepted target delta
    != applied Final Point state
```

WU-07 is the only mutation boundary. Therefore WU-06 must produce a plan that is rich enough for deterministic preview and stale-state validation while remaining non-mutating and incapable of silently promoting suggestions into executable changes.

Sequential Work adds another requirement: accepted target state is cumulative. A later source must be reconciled against the semantic Final Point state that would result from already accepted earlier-source deltas, not merely against the initial on-disk Final Point snapshot.

## Considered architectures

### Option A — Target Operation Algebra + Constraint/Plan Graph

Represent every architecture change as a generic semantic operation and build a dependency/conflict graph across operations.

**Architectural strengths**

- very broad applicability beyond PTSIP;
- rich representation of dependencies, alternatives, split/merge operations, and generic constraints;
- natural basis for future solver or visualization systems.

**Architectural weaknesses**

- PTSIP entity semantics are less strongly represented by the core operation type;
- architecture dependencies and plan-operation dependencies can become conflated;
- forbidden PTSIP combinations require a separate constraint layer rather than being apparent in the semantic delta itself.

### Option B — Strongly Typed Proposal Domain + Acceptance Boundary + Typed Execution Plan

Define a distinct proposal and accepted-delta type for every architecture change class, then feed only accepted types into a strongly typed execution plan.

**Architectural strengths**

- strongest proposal-specific invariants;
- very strong separation between suggested, accepted, unresolved, and executable states;
- direct architectural continuity with the strongly typed WU-05 analyzer.

**Architectural weaknesses**

- cross-type reconciliation becomes a separate combinatorial domain;
- every new target entity/change shape expands the proposal type union;
- compound semantic changes need an additional composition layer anyway.

### Option C — Semantic Delta Bundle + Deterministic Reconciliation Planner

Use one canonical semantic `TargetDelta` representation for entity-level changes, group related deltas into proposal bundles, and separate authority at the bundle-container boundary.

The project owner selected **Option C**.

## Decision

### 1. Canonical semantic delta

WU-06 uses a common target-change contract:

```text
TargetDelta
  entity_kind
  entity_id
  change_kind
  before
  after
  source obligation / finding references
  evidence references
```

The supported atomic change vocabulary is deliberately small:

```text
ADD
REMOVE
REPLACE
```

Component split/merge is represented as a bundle of explicit atomic deltas rather than introducing an opaque mutation primitive. For example, a split may contain removal/replacement of the old component plus additions of the new components and required relationships.

### 2. Semantic identity

Delta identity is deterministic over canonical semantic content. Mapping key order and set-like PTSIP fields such as selectors/roles do not change delta identity when their meaning is equivalent.

A delta identity includes its source-obligation/finding references so semantically similar changes derived from different migration obligations remain auditable rather than being silently inherited.

### 3. Authority is separated by bundle container

WU-06 distinguishes:

```text
ProposalBundle
AcceptedDeltaBundle
UnresolvedBundle
```

`ProposalBundle` has proposal-only authority. It is never executable merely because it is first, highest-confidence, mechanically convenient, or conflict-free.

`AcceptedDeltaBundle` exists only when the proposal is bound to a non-empty project-owned decision identity. WU-07 may consume accepted bundles as execution candidates, subject to its own stale-state and completion checks.

`UnresolvedBundle` retains questions, alternatives, or missing target intent without inventing a target decision.

### 4. Proposal purposes remain explicit

Bundles distinguish at least:

```text
REQUIRED_MIGRATION
TARGET_VALIDITY
ASYNC_OPTIONAL
ADVISORY
```

Required migration and target-validity proposals block an executable WU-07 preview until accepted or otherwise explicitly resolved.

Async proposals are optional and non-contributing to source completion. They are emitted only when explicitly requested/accepted and are ordered after required deltas.

Evidence conflict/incompleteness may remain advisory. Evidence never becomes target architecture authority by itself.

### 5. WU-05 unresolved requirements are not auto-designed

When WU-05 reports an unresolved Required Work Element, WU-06 does not invent a lifecycle, component grouping, split, relationship stable ID, or target declaration merely to make a plan executable.

If an explicit proposal/accepted delta covers the obligation, it is used. Otherwise WU-06 emits a blocking `UnresolvedBundle`.

This preserves the project-owner authority boundary for historical `TOOLCHAIN`, split/merge, lifecycle change, and other architecture-intent decisions.

### 6. Final Point may be existing or planned

WU-06 distinguishes:

```text
EXISTING Final Point
PLANNED Final Point
```

If WU-01 already discovered a Final Point, reconciliation requires an exact semantic snapshot whose path, draft, immutable revision, and content SHA match the WU-01 identity.

If no Final Point exists, WU-06 may plan the required `ptsip_<version>.yaml` identity without creating the file. Creation remains WU-07 mutation.

### 7. Deterministic reconciliation statuses

Each delta reconciles to one of:

```text
NO_CHANGE_REQUIRED
ADD_TARGET_DECLARATION
REPLACE_WITH_EXPLICIT_OWNER_DECISION
CONFLICT_REQUIRES_CONFIRMATION
UNRESOLVED
```

An ADD against an identical existing entity becomes `NO_CHANGE_REQUIRED`. An ADD against a different entity with the same stable identity is a conflict.

REMOVE/REPLACE operations are safe only against the exact reviewed `before` state. If the Final Point changed, reconciliation fails closed rather than treating the accepted decision as permission to overwrite arbitrary newer state.

### 8. Cumulative planned Final Point semantics

For Sequential Work, the planner processes WU-01 `ordered_sources` exactly.

Accepted deltas from an earlier source are applied only to an in-memory semantic projection of the Final Point. The repository file is not mutated.

The next source is reconciled against that projected cumulative target state. Therefore two independently accepted source plans that collide on the same stable target identity are detected during WU-06 planning rather than being deferred to silent last-writer behavior.

Every source still targets the same Final Point directly. The in-memory projection is not an intermediate migration hop or a new authority chain.

### 9. Required work precedes Async work

Within each source execution preview:

```text
accepted required / target-validity deltas
    -> accepted Async deltas
```

Async work cannot satisfy an unresolved Required Work Element, affect source-completion counts, or improve deletion eligibility.

### 10. Source deletion remains a WU-07 proof

WU-06 exposes a deletion gate:

```text
ALREADY_ELIGIBLE
REQUIRES_POST_APPLY_VERIFICATION
BLOCKED
```

`REQUIRES_POST_APPLY_VERIFICATION` is not permission to delete. It means the accepted plan may be attempted by WU-07 and the source must then be re-proven complete before deletion.

### 11. Stable WU-07 preview contract

WU-06 produces an `ExecutionPreview` containing:

- exact Final Point identity;
- WU-01 ordered source paths;
- exact source content identities and WU-05 analysis digests;
- accepted/suggested/unresolved bundle identities;
- ordered accepted delta identities;
- reconciliation results;
- blocking identities;
- per-source deletion gate;
- cumulative projected Final Point semantic digest.

`ready_for_wu07` means only that the plan has no unresolved planning blocker. It does not mean profile mutation or canonical promotion has occurred.

Canonical promotion always remains guarded by WU-07 global validation.

## Consequences

### Positive

- target proposal representation remains compact while preserving PTSIP semantic identity;
- compound target changes can be represented as coherent bundles without an opaque split/merge mutation primitive;
- proposal, accepted decision, and applied state remain structurally distinct;
- later source plans see cumulative accepted target semantics and can detect cross-source conflicts early;
- Required and Async work remain visibly separate in execution ordering;
- planned Final Point creation can be previewed without mutating `ptsip.yaml` or creating a temporary profile;
- WU-07 receives a deterministic machine-readable plan instead of a prose proposal list.

### Tradeoffs

- entity-specific invariants are enforced by validation around the common semantic payload rather than by a unique Python type for every proposal kind;
- partial acceptance inside one semantic bundle should be represented by splitting the bundle before acceptance rather than assigning mixed authority inside one container;
- future target entities may require extending `TargetEntityKind` and semantic normalization rules;
- complex graph-like dependency solving remains outside this WU unless later requirements justify it.

## Non-authority statement

This ADR selects the WU-06 software architecture only. It does not approve any concrete lifecycle reclassification, split/merge, relationship, associated artifact, selector change, Async addition, Final Point write, source deletion, or canonical promotion.
