# PTSIP Draft Profile Transition Specification

**Specification family:** `0.3.7-draft`  
**Status:** Active normative companion  
**Parent semantics:** Tool `0.3.6` primary lifecycle ownership and Responsibility Map v2 remain unchanged except where this companion adds draft-profile transition requirements.  
**Architectural decision:** `decisions/ADR-0010-versioned-draft-profile-transition.md`

## 1. Purpose

This companion specifies how a Consumer Repository transitions a canonical `ptsip.yaml` from one PTSIP draft family to another without destroying the prior declaration before migration obligations are evaluated and completed.

The purpose is not to introduce a new lifecycle classification. The canonical PTSIP classifications remain:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

The transition model exists because a Project Profile is revision-bound architecture declaration. A change such as:

```text
0.3.4-draft -> 0.3.6-draft
```

can require re-evaluation of classifications, relationships, associated artifacts, selectors, or other architecture declarations. Rewriting the same physical `ptsip.yaml` in place can destroy the stable source needed to establish what must be migrated.

## 2. Canonical profile identity

The active canonical Project Profile remains:

```text
ptsip.yaml
```

The canonical file represents the architecture declaration currently adopted by the repository/worktree under its declared PTSIP draft family and immutable specification revision.

A canonical profile MUST NOT be treated as version-neutral merely because its filename is stable.

At minimum, the profile draft identity includes:

```yaml
ptsip:
  version: "<major>.<minor>.<micro>-draft"
  specification:
    revision: "<immutable revision>"
```

## 3. Temporary PTSIP Profile File

When a newer draft must be prepared while an older canonical `ptsip.yaml` remains the migration source, the target working profile uses:

```text
ptsip_<major>.<minor>.<micro>.yaml
```

This file is canonically called a **Temporary PTSIP Profile File**.

Example:

```text
ptsip.yaml             # source: 0.3.4-draft
ptsip_0.3.6.yaml       # target working profile: 0.3.6-draft
```

The filename omits the `-draft` suffix. The internal version MUST match the filename semantic version plus `-draft` exactly.

```text
ptsip_0.3.6.yaml
    <-> ptsip.version == 0.3.6-draft
```

A filename/internal-version mismatch MUST fail closed.

There MUST be at most one Temporary PTSIP Profile File for one target semantic version. Two logical candidates for the same target identity, such as two competing `ptsip_0.3.7.yaml` states, MUST NOT be merged or selected implicitly.

## 4. Source-specific migration-control categories

For every migration source profile, PTSIP evaluates the current repository snapshot again. Migration-control categories belong to that source evaluation only and are not inherited from another profile generation.

### 4.1 PTSIP Required Work Element (`PTSIP 필수작업요소`)

A **PTSIP Required Work Element** is a file or document represented by the source draft's active classified/domain structure that still exists validly in the current repository and therefore remains a mandatory migration obligation for that source.

A source migration is not complete until every Required Work Element has been represented, transformed, or explicitly resolved into target semantics.

### 4.2 PTSIP Removal Migration Element (`PTSIP 제거이전요소`)

A **PTSIP Removal Migration Element** is an element mentioned by the source draft that is no longer validly active in the current repository because it was removed, retired, or otherwise ceased to be a current migration obligation.

It MUST NOT be copied merely for preservation. There is no preservation obligation. Handling or preserving such an element does not contribute to source-migration completion.

### 4.3 PTSIP Asynchronous Work Target (`PTSIP 비동기작업대상`)

A **PTSIP Asynchronous Work Target** is a repository file or document that was not part of the source draft's active migration obligation, including material that was not directly classified or otherwise included by the source profile.

It MAY be added to the target profile when the project owner requests it, but Required Work Elements SHOULD be completed first. Asynchronous Work Targets do not contribute to completion of the source migration and MUST NOT substitute for unresolved Required Work Elements.

## 5. Completion semantics

### PTSIP-MIG-004 — Versioned target coexistence

When the canonical `ptsip.yaml` is bound to an older draft and migration to a newer draft requires re-evaluation, a conforming write workflow MUST preserve the canonical source while preparing the target through the target version's Temporary PTSIP Profile File.

The canonical source MUST NOT be destructively rewritten solely to change `ptsip.version` or `ptsip.specification.revision` before migration completion has been established.

### PTSIP-MIG-005 — Source-specific obligation evaluation

Every source profile migration MUST independently evaluate the current repository snapshot and determine its Required Work Elements, Removal Migration Elements, and Asynchronous Work Targets.

A prior profile generation's category decisions MUST NOT be inherited as authority for a later source evaluation.

### PTSIP-MIG-006 — Required Work completion gate

A source migration is complete only when every PTSIP Required Work Element for that source has been handled in the target state under the target draft semantics.

A migration workflow MUST NOT declare completion because optional work was performed, because the target profile validates structurally, or because some high-confidence classifications were produced while Required Work Elements remain unresolved.

### PTSIP-MIG-007 — Removal elements do not carry forward

PTSIP Removal Migration Elements MUST NOT be migrated merely to preserve obsolete source declarations. Their omission is not migration loss and their handling does not increase completion credit.

### PTSIP-MIG-008 — Asynchronous work is non-blocking and non-crediting

PTSIP Asynchronous Work Targets are not immediate migration obligations. They MAY be added after Required Work Elements are complete when explicitly requested or otherwise authorized by the project owner.

Adding them does not increase source-migration completion and MUST NOT mask incomplete Required Work Elements.

## 6. PTSIP Draft Sequential Work

When more than one newer draft target exists or becomes necessary before earlier migration work has completed, the repository is in **PTSIP Draft Sequential Work**.

The highest selected target draft is the **Final PTSIP Point File**.

Example:

```text
ptsip.yaml          = 0.3.4-draft
ptsip_0.3.6.yaml    = 0.3.6-draft
ptsip_0.3.7.yaml    = 0.3.7-draft
ptsip_0.4.0.yaml    = 0.4.0-draft  # Final PTSIP Point File
```

The existence of multiple relevant draft generations does not require every possible intermediate Temporary PTSIP Profile File to exist physically. Sequential Work is determined by the migration state and target generations that actually exist or are required.

### PTSIP-MIG-009 — Final Point direct convergence

Once a Final PTSIP Point File is selected, still-incomplete source generations MUST converge directly into that Final Point. A source MUST NOT be routed through a Temporary PTSIP Profile File that has already completed and been removed.

The strongly recommended processing order is the incomplete Temporary PTSIP Profile File closest to the Final Point first, continuing toward older temporary sources, with the canonical `ptsip.yaml` migrated last.

For the example above:

```text
0.3.7 temporary -> 0.4.0 Final Point -> delete completed 0.3.7 temporary
0.3.6 temporary -> 0.4.0 Final Point -> delete completed 0.3.6 temporary
canonical ptsip.yaml -> 0.4.0 Final Point
```

The recommendation exists to reduce repeated classification and migration judgments, avoid revisiting already-decided intermediate semantics, and reduce the risk that repeated evaluation reverses or inconsistently reinterprets prior judgments.

### PTSIP-MIG-010 — Completed intermediate source removal

After one Temporary PTSIP Profile File has completed its source-specific migration into the Final Point and post-apply checks establish that completion, that source Temporary PTSIP Profile File SHOULD be removed from the active repository state.

Later sources MUST NOT migrate into or through that removed intermediate generation.

### PTSIP-MIG-011 — No migration-category inheritance

Required Work Element, Removal Migration Element, and Asynchronous Work Target classifications are local to one source evaluation.

They MUST NOT be inherited, copied, or preserved as authoritative category metadata when another profile generation is evaluated. Each source is evaluated anew against the current repository snapshot and the target draft semantics.

### PTSIP-MIG-012 — Final Point target-state accumulation

Declarations already accepted into the Final PTSIP Point File are target state. They are not inherited source-category metadata.

When a later source proposes a target delta semantically equivalent to existing Final Point state, tooling SHOULD preserve the existing target state without unnecessary rewrite.

When a later source conflicts materially with already accepted Final Point state, tooling MUST fail closed or obtain explicit project-owner resolution. It MUST NOT silently choose the newest source, oldest source, largest source, or highest-confidence proposal.

## 7. Canonical promotion

### PTSIP-MIG-013 — Guarded Final Point promotion

The canonical `ptsip.yaml` is migrated last in Sequential Work.

After all source-specific Required Work Elements are complete and the Final Point passes required target-profile validation and migration consistency checks, the old canonical profile is removed and the Final PTSIP Point File is promoted to the canonical filename:

```text
remove old ptsip.yaml
rename ptsip_<final-major>.<final-minor>.<final-micro>.yaml -> ptsip.yaml
```

Promotion MUST preserve the Final Point content semantically. Promotion itself MUST NOT trigger a fresh implicit architecture rewrite.

A workflow MUST NOT delete the old canonical source before it has sufficient evidence that the Final Point is valid, complete for required source obligations, and safe to promote.

### PTSIP-MIG-014 — Snapshot and stale-state safety

Migration analysis, completion decisions, apply operations, source deletion, and canonical promotion MUST be bound to sufficiently exact repository/profile state.

If a relevant source profile, target profile, or repository state changes after analysis but before apply/promotion, the operation MUST fail closed or re-analyze the changed state before mutation.

No migration workflow may use a stale completion judgment merely because a prior version of the file validated successfully.

### PTSIP-MIG-015 — Final Point ordering is preferred governance, not compatibility preservation

PTSIP Draft Sequential Work SHOULD use the Final-Point-nearest-first order described by `PTSIP-MIG-009` and SHOULD process the canonical `ptsip.yaml` last.

The reason is governance correctness and efficiency, not preservation of obsolete compatibility. Implementations MUST NOT retain an intermediate profile merely because older tooling might depend on it when that intermediate profile has completed its migration and is no longer an active source obligation.

If a project intentionally departs from the recommended order, tooling SHOULD surface that the departure can cause duplicate re-analysis and repeated architecture judgments. Mandatory source-specific completion, direct Final Point convergence, conflict safety, and guarded canonical promotion still apply.

## 8. Evidence and authority boundary

This transition model does not convert evidence into architecture authority.

```text
repository observation
    -> evidence
    -> source-specific migration analysis
    -> target proposal
    -> project-owner decision/authorized state
    -> safe apply
```

Evidence, path conventions, confidence scores, repository topology, or historical classification alone MUST NOT authorize a target architecture rewrite.

A Temporary PTSIP Profile File is working target architecture state; its filename does not itself authorize its declarations.

## 9. Relationship to distributed Decision Authority

Distributed Decision Authority and draft-profile migration solve different problems.

Decision Authority coordinates which explicit architecture answer won for a decision identity. Draft-profile transition controls how one revision-bound Project Profile generation is transformed into another.

A resolved distributed winner MAY be an input to a migration target proposal or accepted target state, but it does not waive Required Work completion, target validation, stale-state protection, or Final Point conflict handling.

## 10. Tooling obligations

A conforming Tool `0.3.7` implementation that automates draft-profile migration SHOULD provide machine-checkable state for at least:

- canonical source draft identity;
- Temporary PTSIP Profile File identities;
- selected Final PTSIP Point File;
- source-specific Required Work Element completion;
- source-specific Removal Migration Elements;
- source-specific Asynchronous Work Targets;
- migration order;
- stale/conflicting state;
- promotion readiness.

The implementation MAY choose internal data structures and CLI surfaces freely, but it MUST preserve the normative semantics above.

## 11. Non-goals

This companion does not:

- add a sixth PTSIP lifecycle classification;
- make filenames architecture authority;
- require every historical intermediate draft to have a physical file;
- require Asynchronous Work Targets to be completed before promotion;
- preserve removed source declarations for backward compatibility;
- permit blind `TOOLCHAIN -> DEVELOPMENT_TOOLING` migration;
- permit Final Point conflicts to be resolved by confidence or file recency;
- require continuous background migration or polling.

## 12. Normative lineage

The architecture rationale is recorded by:

```text
decisions/ADR-0010-versioned-draft-profile-transition.md
```

Tool `0.3.7` planning consumes this companion through:

```text
planning/0.3.7.md
planning/0.3.7/*
```

The immutable Git revision containing this normative companion is the revision to which the `0.3.7-draft` activation record binds.
