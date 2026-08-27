# PTSIP Project Profile Transition Specification

**Specification family:** `0.3.7-draft`  
**Status:** Active normative companion / WU-11 freeze candidate  
**Base lifecycle semantics:** the mature `0.3.6-draft` primary lifecycle ownership and Responsibility Map v2 rules remain in force except where this companion adds Project Profile identity, compatibility, migration, and transition requirements.  
**Architecture authorities:** ADR-0010, ADR-0017, ADR-0019, ADR-0020, ADR-0021, ADR-0022

## 1. Purpose

This companion specifies how PTSIP identifies, reads, analyzes, migrates, and safely transitions Project Profiles without coupling Project Profile contract identity to the PTSIP Tool package version.

The canonical lifecycle classifications remain exactly:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

This companion does not add a lifecycle classification and does not change the governing classification semantics established by the base `0.3.6-draft` Specification family.

The Project Profile transition model exists because a Project Profile is a revision-bound architecture declaration. A source profile must remain interpretable while migration obligations are analyzed, owner decisions are preserved, target state is prepared, and final writes are validated.

## 2. Independent identity axes

PTSIP treats the following identities as distinct:

```text
PTSIP Tool Version
Project Profile Contract Version
Project Profile Instance Revision
PTSIP Specification family + immutable revision
```

The following implication is invalid:

```text
Tool 0.3.7
    => Project Profile 0.3.7-draft
```

Likewise, a Project Profile contract transition does not automatically authorize a Tool release, and a Tool release does not automatically authorize a repository Project Profile write.

### 2.1 Project Profile contract identity

Current-generation Project Profile contracts use the independent namespace:

```text
pp.<major>.<minor>
```

Canonical serialization uses a two-digit minor segment. For Tool `0.3.7`, the current supported canonical Project Profile target is:

```text
pp.1.01
```

The corresponding current-generation temporary-profile filename token is:

```text
pp1.01
```

and the current-generation temporary target path is:

```text
ptsip_pp1.01.yaml
```

The canonical adopted Project Profile path remains:

```text
ptsip.yaml
```

### 2.2 Project Profile instance revision

A Project Profile Contract Version identifies contract semantics. It does not identify one concrete repository declaration instance.

One concrete profile instance has its own immutable content/revision identity. Tool Version, Project Profile Contract Version, Project Profile Instance Revision, and Specification revision MUST NOT be collapsed into one version field or inferred from one another.

## 3. Historical source compatibility

Historical Tool-numbered Project Profile labels remain historical facts. A conforming implementation MUST NOT rewrite history by claiming they were originally published under the `pp.*` namespace.

Tool `0.3.7` recognizes the following revision-bound historical source families for direct convergence:

```text
0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e
    compatibility generation: pp.0.00
    current target:           pp.1.01
    transition:               SEMANTIC_MIGRATION

0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea
    compatibility generation: pp.1.01
    current target:           pp.1.01
    transition:               IDENTITY_ONLY
```

Compatibility-generation identity is interpretation metadata. It is not a mandatory intermediate repository state.

A historical source is supported only when its declared family and immutable Specification revision match an explicit compatibility authority. Unsupported or ambiguous source identity MUST fail closed.

Historical `TOOLCHAIN` semantics remain historical source semantics. A compatibility reader MUST NOT automatically convert `TOOLCHAIN` to `DEVELOPMENT_TOOLING` merely because the current PTSIP lifecycle vocabulary contains `DEVELOPMENT_TOOLING`.

## 4. `0.3.6-draft -> pp.1.01` identity-only bridge

ADR-0021 defines:

```text
0.3.6-draft
    ↓ IDENTITY_ONLY
pp.1.01
```

For this bridge, the Project Profile contract-content delta is zero:

```text
components delta:               NONE
relationships delta:            NONE
associated_artifacts delta:     NONE
policies delta:                 NONE
Responsibility Map delta:       NONE
lifecycle classification delta: NONE
```

Therefore an otherwise valid `0.3.6-draft` Project Profile MUST NOT require lifecycle redesign, component reclassification, relationship redesign, policy redesign, or a synthetic semantic migration merely because its canonical Project Profile identity becomes `pp.1.01`.

Post-write identity/schema validation remains mandatory.

When canonical `ptsip.yaml` is the supported `0.3.6-draft` source and no independent semantic change is required, the identity transition is performed in place:

```text
ptsip.yaml
ptsip.version: 0.3.6-draft
        ↓
ptsip.yaml
ptsip.version: pp.1.01
```

No temporary Project Profile is required solely for this identity-only rewrite.

## 5. Direct latest-target convergence

Project Profile migration is **source-to-current-target convergence**, not mandatory historical-version traversal.

A conforming implementation answers:

```text
What does the actual supported source mean,
and what must change for it to satisfy the current canonical PP target?
```

It MUST NOT require:

```text
source -> every historical intermediate -> current target
```

merely because intermediate Project Profile generations existed.

Conceptually:

```text
actual supported source
        ↓ source-family compatibility reader
normalized source semantics
        ↓ direct reconciliation
current canonical PP target selected by explicit Tool/PP compatibility authority
        ↓ authorized execution
validated target profile
```

Intermediate generations MAY contribute compatibility/history knowledge. They MUST NOT become mandatory execution hops or be materialized merely to replay version history.

For Tool `0.3.7`, the selected current target is `pp.1.01`. A future Tool may select a later canonical target through explicit compatibility authority without requiring repositories to materialize every intervening PP generation.

Target selection MUST NOT be inferred solely from numeric ordering when more than one target contract is intentionally supported.

## 6. Temporary Project Profile targets and continuity aliases

A temporary target is required only when the migration semantics actually require a separate working target while the source remains authoritative.

The current-generation temporary path for `pp.1.01` is:

```text
ptsip_pp1.01.yaml
```

Historical migration continuity is preserved where an explicit compatibility bridge declares a legacy physical path equivalent to the current logical target.

For Tool `0.3.7`, an existing:

```text
ptsip_0.3.6.yaml
```

may continue as the physical target for logical Project Profile `pp.1.01` when the applicable source bridge explicitly authorizes that equivalence. Its physical filename is continuity state, not independent target-contract authority.

If no authorized legacy target already exists, a new current target MUST use the current PP identity rather than fabricate an obsolete Tool-numbered intermediate:

```text
ptsip_pp1.01.yaml
```

A conforming Tool MUST NOT create `ptsip_0.3.7.yaml` merely because the Tool package version is `0.3.7`.

## 7. Equivalent-target ambiguity

If multiple physical paths represent the same logical current PP target, tooling MUST fail closed rather than select one implicitly.

For the accepted `0.3.6-draft -> pp.1.01` continuity bridge, simultaneous existence of:

```text
ptsip_0.3.6.yaml
ptsip_pp1.01.yaml
```

is an equivalent-target collision and MUST produce the stable diagnostic:

```text
DUPLICATE_EQUIVALENT_TARGET
```

Repository-state correction or explicit authority is required before migration continues.

Unrecognized temporary Project Profile files that imply synthetic intermediate traversal MUST also fail closed during direct-convergence discovery.

## 8. Source-specific migration-control categories

For every semantic migration source, PTSIP evaluates the current repository snapshot again. Migration-control categories belong to that source evaluation only and are not inherited from another profile generation.

### 8.1 PTSIP Required Work Element (`PTSIP 필수작업요소`)

A **PTSIP Required Work Element** is an element represented by the source profile's active architecture that still validly exists in the current repository and therefore must be handled before that source migration can complete.

Required Work Elements determine source-migration completion.

### 8.2 PTSIP Removal Migration Element (`PTSIP 제거이전요소`)

A **PTSIP Removal Migration Element** is represented by the source but no longer validly active in the current repository.

It has no preservation obligation and contributes no migration-completion credit.

### 8.3 PTSIP Asynchronous Work Target (`PTSIP 비동기작업대상`)

A **PTSIP Asynchronous Work Target** is repository material not directly part of the source profile's active migration obligation.

It MAY be added when separately authorized, but it does not contribute to source-migration completion and MUST NOT substitute for unresolved Required Work Elements.

## 9. Normative migration rules

### PTSIP-MIG-004 — Versioned target coexistence

When genuine semantic migration requires a separate target while the canonical source must remain stable, the target MAY coexist through the selected target contract's authorized temporary path.

Target path identity MUST come from Project Profile compatibility authority, not Tool package version symmetry.

An identity-only canonical rewrite does not require a temporary target merely to reuse the semantic migration state machine.

### PTSIP-MIG-005 — Source-specific obligation evaluation

Every semantic source migration MUST independently evaluate the current repository snapshot and determine Required Work Elements, Removal Migration Elements, and Asynchronous Work Targets.

A prior source generation's category decisions MUST NOT be inherited as authority.

### PTSIP-MIG-006 — Required Work completion gate

A semantic source migration is complete only when every Required Work Element for that source has been represented, transformed, or explicitly resolved into the accepted current-target state.

Structural target validation, optional work, or high-confidence inference MUST NOT substitute for unresolved Required Work.

### PTSIP-MIG-007 — Removal elements do not carry forward

Removal Migration Elements MUST NOT be migrated merely to preserve obsolete source declarations. Their omission is not migration loss and their handling does not increase completion credit.

### PTSIP-MIG-008 — Asynchronous work is non-blocking and non-crediting

Asynchronous Work Targets are not immediate source-migration obligations. They MAY be handled after Required Work when separately authorized, but they do not increase completion and MUST NOT mask unresolved Required Work.

### PTSIP-MIG-009 — Direct current-target convergence

A supported historical source MUST reconcile directly against the current canonical PP target selected by explicit Tool/PP compatibility authority.

Completed or historical intermediate Project Profile generations MUST NOT be required as migration hops. No synthetic intermediate profile may be created solely for version traversal.

### PTSIP-MIG-010 — Completed source retirement

After a temporary source has completed its source-specific semantic migration into the current target and post-apply checks establish completion, that source MAY be removed according to the accepted execution plan.

Later sources MUST NOT be routed through the removed intermediate state.

### PTSIP-MIG-011 — No migration-category inheritance

Required Work, Removal Migration, and Asynchronous Work categories are local to one source evaluation. They MUST NOT be copied into another source evaluation as architecture authority.

### PTSIP-MIG-012 — Current-target state accumulation

Declarations already accepted into the current target are target state, not inherited source-category metadata.

Semantically equivalent later proposals SHOULD preserve existing accepted target state without unnecessary rewrite.

Material conflict with accepted target state MUST fail closed or require explicit project-owner resolution. File recency, source age, numeric version order, confidence, or migration order MUST NOT silently choose a winner.

### PTSIP-MIG-013 — Guarded canonical promotion

For genuine semantic migration using a separate target, canonical promotion MAY occur only after required source obligations are complete and the selected current target passes required validation and consistency checks.

Promotion MUST preserve accepted target semantics and MUST NOT become an implicit architecture rewrite.

When the accepted bridge is `IDENTITY_ONLY`, an in-place canonical identity rewrite MAY be used instead of synthetic target creation/promotion, subject to the same freshness and post-write validation requirements.

### PTSIP-MIG-014 — Snapshot, binding, and stale-state safety

Migration discovery, analysis, accepted plan, apply operations, source deletion, identity rewrite, and promotion MUST be bound to sufficiently exact repository/profile state.

If a relevant source profile, target profile, accepted delta, or repository state changes after analysis but before mutation, the operation MUST fail closed or re-analyze the changed state before mutation.

Semantic compare-and-swap or equivalent stale-writer protection MUST preserve the exact accepted mutation boundary.

### PTSIP-MIG-015 — Processing order is governance, not compatibility preservation

When several active source states require convergence to one current target, tooling SHOULD process them in an order that minimizes repeated analysis and conflicting architecture judgments, while preserving source-specific completion and canonical-last safety where applicable.

The ordering recommendation MUST NOT be interpreted as a requirement to preserve obsolete intermediate versions or traverse historical Project Profile generations sequentially.

## 10. Analyzer, planner, and executor boundaries

The migration pipeline remains responsibility-segmented:

```text
historical source reader
        ↓
normalized source semantics
        ↓
current target semantics
        ↓
source-specific migration analysis
        ↓
ProposalBundle
AcceptedDeltaBundle
UnresolvedBundle
        ↓
deterministic direct-convergence plan
        ↓
exact snapshot/binding validation
        ↓
authorized apply / identity rewrite / promotion
```

Evidence is not architecture authority. Proposal is not accepted delta. Accepted delta is not repository adoption authority. Execution MUST mutate only the accepted semantic or identity delta bound to the exact execution state.

Identity-only transitions MUST NOT manufacture semantic obligations merely to pass through analyzer/planner/executor stages designed for semantic migration.

## 11. Interruption, recovery, and promotion safety

Execution that performs semantic mutation MUST preserve typed state, append-only checkpoint evidence, exact source/target/repository bindings, Required-before-Async semantics, source completion before deletion, fail-closed recovery, and guarded final promotion.

Recovery MUST NOT infer success from partial filesystem state alone. It must reconcile the exact bound checkpoint/ledger state with current repository/profile content.

An interrupted identity-only rewrite likewise requires post-write validation and must not silently continue from stale pre-write assumptions.

## 12. Adoption authority

Migration capability and real-project adoption are separate authorities.

Tool `0.3.7` supporting `pp.1.01` does not by itself authorize mutation of a repository's canonical `ptsip.yaml`.

A write-enabled adoption or migration requires the applicable exact source state, accepted architecture/identity delta, required owner authorization, stale-state checks, and post-write validation.

The `0.3.6-draft -> pp.1.01` identity equivalence removes unnecessary semantic reclassification work. It does not remove write authorization.

## 13. User-visible disclosure requirements

A conforming Tool/release record that advertises `pp.1.01` support MUST distinguish:

```text
Tool version
Specification family + immutable revision
current Project Profile contract target
supported historical Project Profile source families
Project Profile instance/repository adoption state
```

User-facing transition documentation MUST state that the `0.3.6-draft -> pp.1.01` bridge is `IDENTITY_ONLY` and introduces no required changes solely to:

```text
components
relationships
associated_artifacts
policies
Responsibility Map semantics
lifecycle classifications
```

User-facing documentation MUST also state that supported historical/intermediate Project Profile generations may inform compatibility analysis but are not mandatory user-visible migration steps.

## 14. Fail-closed requirements

A conforming implementation MUST fail closed for at least:

- unsupported historical source family/revision;
- malformed or unsupported current PP identity;
- unsupported target selection;
- ambiguous target selection;
- duplicate equivalent target paths;
- synthetic/unrecognized intermediate target state;
- historical vocabulary with no deterministic current mapping;
- lifecycle reclassification requiring owner decision;
- stale source/target/repository snapshot;
- semantic CAS/binding mismatch;
- invalid post-write target;
- unsafe interruption/recovery state.

The Tool MUST NOT guess merely to complete migration or release preparation.

## 15. Non-goals

This companion does not:

- add a sixth PTSIP lifecycle classification;
- collapse Tool Version and Project Profile Contract Version;
- make filenames architecture authority;
- require every historical Project Profile generation to have a physical file;
- require mandatory intermediate-version traversal;
- permit blind `TOOLCHAIN -> DEVELOPMENT_TOOLING` migration;
- make migration capability equivalent to repository adoption authority;
- permit target conflict resolution by file recency, version order, or confidence;
- require continuous background migration or polling.

## 16. Normative lineage

Architecture rationale and responsibility boundaries are recorded by:

```text
decisions/ADR-0010-versioned-draft-profile-transition.md
decisions/ADR-0017-*.md
decisions/ADR-0019-*.md
decisions/ADR-0020-*.md
decisions/ADR-0021-project-profile-identity-bridge-and-release-note-namespaces.md
decisions/ADR-0022-*.md
```

Tool `0.3.7` planning consumes this companion through:

```text
planning/0.3.7.md
planning/0.3.7/*
```

The immutable Git revision selected during WU-11 Specification freeze is the normative `0.3.7-draft` revision to which the final Tool release binds.
