# ADR-0011 — Activate PTSIP Specification 0.3.7-draft

**Status:** Accepted  
**Decision:** Activate `0.3.7-draft` as the normative draft family for Tool `0.3.7` development  
**Immutable normative snapshot:** `b648d9e026f502b14481ba2d0606d9acc88a31fc`  
**Primary new normative companion:** `spec/PTSIP-DRAFT-PROFILE-TRANSITION.md`  
**Design authority:** `ADR-0010 — Versioned Draft Profile Transition and Sequential Work Policy`

## Context

Tool `0.3.6` established the five-class primary lifecycle ownership model, Responsibility Map v2, explicit/template/hybrid declaration authority, deterministic Effective Responsibility Map materialization, and accepted-decision safe-apply boundaries.

The cancelled Tool `0.3.6.1` planning family proposed evidence-driven migration work, but it did not establish a first-class lifecycle for the Project Profile itself when the bound draft family changes.

A root `ptsip.yaml` is not version-neutral. Its architecture declaration is bound by at least:

```yaml
ptsip:
  version: "<major>.<minor>.<micro>-draft"
  specification:
    revision: "<immutable revision>"
```

Changing the draft family can require classifications and other Responsibility Map declarations to be re-evaluated. Rewriting the same physical `ptsip.yaml` in place before migration completion destroys the stable source needed to establish what must be preserved, transformed, removed, or newly considered.

ADR-0010 therefore established versioned Temporary PTSIP Profile Files, source-specific migration obligations, PTSIP Draft Sequential Work, and Final PTSIP Point promotion. The project owner has now explicitly confirmed that these semantics are the Specification direction for Tool `0.3.7`.

## Decision

PTSIP `0.3.7-draft` is activated for Tool `0.3.7` development at immutable normative snapshot:

```text
b648d9e026f502b14481ba2d0606d9acc88a31fc
```

The `0.3.7-draft` family preserves the canonical Tool `0.3.6` lifecycle and Responsibility Map semantics and adds the normative draft-profile transition requirements frozen in:

```text
spec/PTSIP-DRAFT-PROFILE-TRANSITION.md
```

The new normative rule IDs are:

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

## Preserved 0.3.6 semantics

Activation of `0.3.7-draft` does not create a new lifecycle ontology.

Canonical PTSIP classifications remain exactly:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

`TOOLCHAIN` remains historical PTSIP migration vocabulary and a separate current VPMS purpose token where applicable. It does not become a canonical PTSIP `0.3.7-draft` classification.

The following established boundaries remain unchanged:

- evidence is not architecture authority;
- candidate discovery is not project intent;
- migration proposals are non-authoritative until accepted;
- Responsibility Map declaration source and lifecycle classification are distinct axes;
- distributed Decision Authority and Project Profile are distinct responsibilities;
- VPMS Verification Purpose remains a separate axis;
- stale or conflicting architecture mutation fails closed.

## New profile-transition semantics

### Temporary PTSIP Profile File

A newer target draft prepared while the canonical source remains active uses:

```text
ptsip_<major>.<minor>.<micro>.yaml
```

The filename target identity and internal `ptsip.version` must match. A target semantic version has at most one Temporary PTSIP Profile File.

### Source-specific migration obligations

Every source profile independently evaluates:

```text
PTSIP Required Work Element
PTSIP Removal Migration Element
PTSIP Asynchronous Work Target
```

These categories are not inherited by another source generation.

Required Work Elements determine completion. Removal Migration Elements have no preservation obligation and no completion credit. Asynchronous Work Targets are optional with respect to source migration completion and do not substitute for Required Work Elements.

### PTSIP Draft Sequential Work

When multiple newer draft targets overlap before migration completion, the highest selected target is the Final PTSIP Point File.

Still-incomplete source generations converge directly into that Final Point. Completed/deleted intermediate profiles are not migration hops.

The strongly recommended order is:

```text
Final-Point-nearest incomplete temporary source first
    -> progressively older temporary sources
    -> canonical ptsip.yaml last
```

This ordering is recommended to reduce duplicate re-analysis, repeated classification judgments, and the risk of contradictory reinterpretation.

### Final Point promotion

Only after source-specific Required Work obligations and target validation are complete may the old canonical `ptsip.yaml` be removed and the Final PTSIP Point File promoted to `ptsip.yaml`.

Promotion must preserve accepted Final Point semantics and must not become a hidden architecture rewrite.

## Specification composition

`0.3.7-draft` is an additive family over the frozen Tool `0.3.6` normative model.

The Tool `0.3.6` specification body and Responsibility Map v2 semantics remain the inherited base unless explicitly superseded by a `0.3.7-draft` rule. The new profile-transition companion is normative for Tool `0.3.7` migration behavior.

This avoids rewriting the mature lifecycle ontology merely to add Project Profile generation-management semantics.

## Tool-binding consequence

This ADR activates the Specification family for development; it does not by itself rewrite the repository's canonical Project Profile or claim that the current Tool runtime has already implemented every `0.3.7-draft` rule.

The repository's existing:

```text
ptsip.yaml = 0.3.6-draft
```

remains the canonical migration source until Tool `0.3.7` WU-01 enters and creates/manages the `0.3.7-draft` Temporary PTSIP Profile File under the newly activated rules.

Likewise, Reference Tool constants, schemas, embedded registry/specdata, profiles, tests, and CLI behavior are implementation surfaces to be migrated under the Tool `0.3.7` work units. They MUST NOT be declared conforming to this family before the corresponding work is implemented and verified.

## Immutable revision rule

The normative snapshot was created before this activation record so that this ADR can point backward to an exact immutable commit.

This intentionally follows the established PTSIP two-step binding shape:

```text
1. normative snapshot commit
2. later activation/binding record points to snapshot SHA
```

A Git commit cannot safely contain a literal self-reference to its own SHA.

## Rejected alternatives

### Rewrite the current `ptsip.yaml` immediately to `0.3.7-draft`

Rejected. That would violate the newly accepted source-preservation rule before WU-01 has established migration obligations and the Temporary PTSIP Profile state model.

### Treat the 0.3.7 change as a filename-only convention

Rejected. The problem is architecture-generation migration control, not naming convenience.

### Preserve every historical declaration for compatibility

Rejected. Removal Migration Elements explicitly have no preservation obligation, and obsolete source semantics must not be carried forward merely to avoid change.

### Migrate every source through every intermediate draft

Rejected. Once a Final PTSIP Point is selected, remaining sources converge directly to it. Deleted/completed intermediates are not migration hops.
