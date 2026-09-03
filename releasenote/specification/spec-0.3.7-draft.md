# PTSIP Specification 0.3.7-draft

## Bound normative snapshot

Tool `0.3.7` development activates the `0.3.7-draft` Specification family at the immutable normative snapshot:

```text
b648d9e026f502b14481ba2d0606d9acc88a31fc
```

Activation is recorded by:

```text
decisions/ADR-0011-activate-spec-0.3.7-draft.yaml
```

The architectural policy that led to this family is:

```text
decisions/ADR-0010-versioned-draft-profile-transition.yaml
```

The primary new normative companion is:

```text
spec/PTSIP-DRAFT-PROFILE-TRANSITION.md
```

The family label is mutable during draft evolution. Normative claims that require exact identity must use the immutable revision above or a later explicitly rebound immutable revision.

## Specification composition

`0.3.7-draft` is additive over the mature Tool `0.3.6` primary lifecycle ownership and Responsibility Map v2 model.

The canonical lifecycle classifications remain exactly:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

The family does not reintroduce legacy PTSIP `TOOLCHAIN` as a canonical classification and does not collapse PTSIP classification with VPMS Verification Purpose.

The principal normative change is the lifecycle of the Project Profile itself when a repository moves between revision-bound draft families.

## Why a new draft family is required

The stable root filename:

```text
ptsip.yaml
```

must not be interpreted as version-neutral. Its contents bind architecture declaration to a draft family and immutable Specification revision.

A change such as:

```text
version: 0.3.4-draft
    ->
version: 0.3.6-draft
```

may require existing classifications, relationships, associated artifacts, selectors, and other project-owned architecture declarations to be re-evaluated under the target draft semantics.

Rewriting the same physical file in place before that evaluation completes destroys the stable source needed to determine whether all migration obligations were handled.

`0.3.7-draft` therefore standardizes versioned Temporary PTSIP Profile Files and guarded canonical promotion.

## Temporary PTSIP Profile File

When the canonical profile remains an older source while a target draft is prepared, the target uses:

```text
ptsip_<major>.<minor>.<micro>.yaml
```

Example:

```text
ptsip.yaml             # 0.3.4-draft source
ptsip_0.3.6.yaml       # 0.3.6-draft temporary target
```

The filename target identity must match the internal `ptsip.version` exactly after adding the `-draft` suffix.

A target semantic version has at most one Temporary PTSIP Profile File. Duplicate logical target identities fail closed rather than being merged implicitly.

## Source-specific migration categories

Each migration source independently evaluates repository elements into the following migration-control categories.

### PTSIP Required Work Element (`PTSIP 필수작업요소`)

An element represented by the source draft's active architecture that still validly exists in the current repository and therefore must be handled before that source migration can complete.

Required Work Elements determine migration completion.

### PTSIP Removal Migration Element (`PTSIP 제거이전요소`)

An element represented by the source draft but no longer validly active in the current repository.

It has no preservation obligation and contributes no migration-completion credit.

### PTSIP Asynchronous Work Target (`PTSIP 비동기작업대상`)

A repository file/document not directly part of the source draft's active migration obligation.

It may be added when requested after Required Work is prioritized, but it does not contribute to completion and cannot substitute for Required Work Elements.

These categories are source-local. They are re-evaluated for every source generation and are never inherited as authority from another profile.

## PTSIP Draft Sequential Work

When multiple newer draft targets overlap before earlier migration completes, the repository enters **PTSIP Draft Sequential Work**.

The highest selected target draft becomes the **Final PTSIP Point File**.

Example:

```text
ptsip.yaml          = 0.3.4-draft
ptsip_0.3.6.yaml    = 0.3.6-draft
ptsip_0.3.7.yaml    = 0.3.7-draft
ptsip_0.4.0.yaml    = 0.4.0-draft  # Final PTSIP Point File
```

The strongly recommended order is:

```text
0.3.7 temporary -> 0.4.0 Final Point -> remove completed 0.3.7 temporary
0.3.6 temporary -> 0.4.0 Final Point -> remove completed 0.3.6 temporary
canonical ptsip.yaml -> 0.4.0 Final Point
validate
promote Final Point -> ptsip.yaml
```

Still-incomplete sources converge directly to the Final Point. A completed/deleted intermediate profile is never used as a migration hop.

The ordering recommendation exists to avoid repeated classification and migration judgments, reduce duplicate work, and lower the risk of contradictory reinterpretation.

## Final Point conflict semantics

Each source is re-evaluated independently, but declarations already accepted into the Final Point are target state rather than inherited source-category metadata.

Equivalent later source results should not cause unnecessary rewrites.

Material conflicts with already accepted Final Point state must fail closed or require explicit project-owner resolution. File recency, source age, confidence score, or migration order does not silently choose a winner.

## Guarded canonical promotion

The canonical `ptsip.yaml` is processed last.

Only after all source-specific Required Work Elements are complete and the Final Point passes target validation and consistency checks may the old canonical source be removed and the Final Point promoted:

```text
remove old ptsip.yaml
rename ptsip_<final-version>.yaml -> ptsip.yaml
```

Promotion itself must preserve the accepted Final Point semantics and must not become an implicit architecture rewrite.

Migration analysis, deletion, apply, and promotion must also fail closed on stale relevant repository/profile state.

## New normative rule IDs

`0.3.7-draft` adds:

- `PTSIP-MIG-004` — Versioned target coexistence.
- `PTSIP-MIG-005` — Source-specific obligation evaluation.
- `PTSIP-MIG-006` — Required Work completion gate.
- `PTSIP-MIG-007` — Removal elements do not carry forward.
- `PTSIP-MIG-008` — Asynchronous work is non-blocking and non-crediting.
- `PTSIP-MIG-009` — Final Point direct convergence.
- `PTSIP-MIG-010` — Completed intermediate source removal.
- `PTSIP-MIG-011` — No migration-category inheritance.
- `PTSIP-MIG-012` — Final Point target-state accumulation.
- `PTSIP-MIG-013` — Guarded Final Point promotion.
- `PTSIP-MIG-014` — Snapshot and stale-state safety.
- `PTSIP-MIG-015` — Final Point ordering is preferred governance, not compatibility preservation.

## Authority boundary

The existing PTSIP authority principle remains unchanged:

```text
Evidence != Authority
Inference != Authority
Proposal != Authority
```

Repository discovery, path conventions, historical classification, confidence, and migration analysis may support a proposal but cannot silently mutate project-owned architecture.

A Temporary PTSIP Profile File is working target state; its filename does not create architecture authority.

Distributed Decision Authority also remains distinct from draft-profile migration. A distributed winner may contribute accepted target state but does not waive migration completion, stale-state safety, or Final Point conflict handling.

## Current repository dogfood boundary

At activation time, the PTSIP repository itself still contains a canonical root profile bound to:

```text
0.3.6-draft
```

This is intentional.

Specification activation does not rewrite the canonical profile to `0.3.7-draft`. Tool `0.3.7` WU-01 must first establish the transition state model and create/manage the appropriate `ptsip_0.3.7.yaml` target under the new rules.

Likewise, Reference Tool constants, schemas, embedded registry/specdata, example profiles, CLI behavior, and tests remain implementation work. They are not declared implemented merely because the Specification family is now confirmed.

## Planning consequence

The cancelled:

```text
planning/0.3.6.1.md
planning/0.3.6.1/*
```

is replaced by:

```text
planning/0.3.7.md
planning/0.3.7/*
```

The `0.3.7-draft` Specification confirmation satisfies the Specification-family prerequisite for later WU-01 entry. WU-01 still requires an explicit fresh entry baseline/status transition before implementation begins.
