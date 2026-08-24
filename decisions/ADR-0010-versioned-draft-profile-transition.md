# ADR-0010 — Versioned Draft Profile Transition and Sequential Work Policy

**Status:** Accepted  
**Decision:** Establish a controlled `ptsip.yaml` draft-version transition model using versioned temporary profile files, independent migration eligibility evaluation, Final PTSIP Point selection, and ordered Sequential Work  
**Applies to:** Tool `0.3.7` planning and every future PTSIP workflow that changes the draft family bound by `ptsip.version` / `ptsip.specification.revision`

## Context

A repository normally exposes one canonical root profile:

```text
ptsip.yaml
```

The profile is not version-neutral. It binds at least:

```yaml
ptsip:
  version: "<major>.<minor>.<micro>-draft"
  specification:
    revision: "<immutable revision>"
```

A draft-family change can therefore invalidate the assumption that the same physical `ptsip.yaml` may simply be edited in place. Existing classifications, relationships, associated artifacts, selectors, or other declarations may need to be re-evaluated under the target draft semantics. In-place replacement also removes the stable source needed to prove that required prior declarations were handled before promotion.

The prior Tool `0.3.6.1` migration plan protected evidence/authority boundaries, but it did not define a first-class lifecycle for multiple profile generations. In particular, it did not answer how `ptsip.yaml` should coexist with a target-draft profile, how completion is measured before promotion, or how work is ordered when more than one newer draft appears before earlier migration has completed.

Tool `0.3.7` therefore treats profile-generation transition as a migration-control problem, not a filename convenience.

## Decision

### 1. Canonical and temporary file identities

The repository's active canonical profile remains:

```text
ptsip.yaml
```

When a newer draft must be prepared while `ptsip.yaml` still represents an older draft, PTSIP creates or manages a versioned temporary profile:

```text
ptsip_<major>.<minor>.<micro>.yaml
```

Example:

```text
ptsip.yaml             # source: 0.3.4-draft
ptsip_0.3.6.yaml       # target working profile: 0.3.6-draft
```

A `ptsip_<major>.<minor>.<micro>.yaml` file is canonically called a **Temporary PTSIP Profile File** (`임시 ptsip.yaml 파일`).

The filename version excludes the `-draft` suffix, while the internal version remains the full draft identity. Therefore:

```text
ptsip_0.3.6.yaml
    <-> ptsip.version == 0.3.6-draft
```

Filename/internal-version mismatch is invalid and MUST fail closed.

There may be at most one Temporary PTSIP Profile File for one target semantic version. Two logical `ptsip_0.3.7.yaml` candidates are not a supported state; duplicate target identity MUST be rejected rather than merged implicitly.

### 2. Migration element categories are source-specific

For every migration source profile, PTSIP evaluates the repository again and classifies relevant files/documents into exactly these migration-control categories.

#### PTSIP Required Work Element (`PTSIP 필수작업요소`)

An element represented by the source draft's classified/domain structure that still exists validly in the current repository and therefore must be handled by the target profile migration.

A source migration is not complete until every Required Work Element has been represented, explicitly transformed, or explicitly resolved into the target state according to target-draft semantics.

#### PTSIP Removal Migration Element (`PTSIP 제거이전요소`)

An element mentioned by the source draft but no longer validly used/present in the repository because it was removed, retired, or otherwise ceased to be an active migration obligation.

It MUST NOT be copied merely for preservation. Handling it does not contribute to source-migration completion. There is no preservation obligation.

#### PTSIP Asynchronous Work Target (`PTSIP 비동기작업대상`)

An existing file/document that the source draft did not directly classify or include as an active migration obligation.

It is not required for migration completion. Required Work Elements have priority. After Required Work Elements are complete, an Asynchronous Work Target MAY be newly added if the project owner requests or accepts that additional target architecture work.

Asynchronous work never increases the completion score or substitutes for Required Work Elements.

### 3. Completion is obligation-based, not copy-based

Migration completion for one source profile is defined only by its source-specific Required Work Elements.

```text
source migration complete
    == all source-specific Required Work Elements handled in target
```

The following do not contribute to completion:

```text
Removal Migration Elements
Asynchronous Work Targets
```

This prevents stale declarations from becoming preservation obligations and prevents optional new architecture work from hiding unfinished required migration.

### 4. Single-step promotion

For a simple transition such as:

```text
0.3.4-draft -> 0.3.6-draft
```

PTSIP uses:

```text
ptsip.yaml
    -> ptsip_0.3.6.yaml
```

The canonical `ptsip.yaml` remains intact until its Required Work Elements have been completely migrated and validated against the target draft.

Only after completion:

```text
remove old ptsip.yaml
rename ptsip_0.3.6.yaml -> ptsip.yaml
```

Promotion MUST be treated as one guarded finalization boundary. A partially migrated temporary file MUST NOT become canonical.

### 5. PTSIP Draft Sequential Work

When a newer draft appears while one or more previous target migrations are still incomplete, the repository enters **PTSIP Draft Sequential Work** (`ptsip draft Sequential work`).

Example:

```text
canonical:  ptsip.yaml          = 0.3.4-draft
temporary:  ptsip_0.3.6.yaml    = 0.3.6-draft
new target: ptsip_0.4.0.yaml    = 0.4.0-draft
```

The highest selected target draft is the **Final PTSIP Point File** (`Final ptsip point file`).

In the example:

```text
Final PTSIP Point File = ptsip_0.4.0.yaml
```

The existence of a conceptual version path does not require every intermediate version file to exist. Only files actually created for active/incomplete target work participate as migration sources.

### 6. Sequential Work ordering

When Sequential Work is active, `ptsip.yaml` is intentionally the last migration source.

Among Temporary PTSIP Profile Files other than the Final PTSIP Point File, migration order is descending by draft version: the source closest to the Final Point is migrated first.

Example:

```text
ptsip.yaml          = 0.3.4-draft
ptsip_0.3.6.yaml    = 0.3.6-draft
ptsip_0.3.7.yaml    = 0.3.7-draft
ptsip_0.4.0.yaml    = 0.4.0-draft  # Final Point
```

Required order:

```text
1. ptsip_0.3.7.yaml -> ptsip_0.4.0.yaml
2. remove ptsip_0.3.7.yaml after its migration completes
3. ptsip_0.3.6.yaml -> ptsip_0.4.0.yaml
4. remove ptsip_0.3.6.yaml after its migration completes
5. ptsip.yaml       -> ptsip_0.4.0.yaml
6. remove old ptsip.yaml after its migration completes
7. rename ptsip_0.4.0.yaml -> ptsip.yaml
```

A deleted/intermediate Temporary PTSIP Profile File MUST NOT be used as a hop. In the example, `0.3.6-draft` MUST migrate directly to the Final Point after `0.3.7-draft` is removed; it MUST NOT be routed through the removed `0.3.7-draft` state.

This ordering is a strong recommendation and the default execution order because it minimizes repeated reclassification/review, reduces the chance that previously evaluated semantics are revisited inconsistently, and prevents work from being repeatedly re-projected through obsolete intermediate drafts.

### 7. Independent evaluation at every source boundary

Each canonical or Temporary PTSIP Profile File owns its own source-specific evaluation of:

```text
Required Work Elements
Removal Migration Elements
Asynchronous Work Targets
```

These categories are NOT inherited from another source profile and MUST NOT be preserved merely because an earlier migration assigned them.

Every source-to-Final-Point migration re-evaluates the current repository and the source draft semantics independently.

This does not erase already accepted target declarations in the Final PTSIP Point File. Those declarations are target state, not inherited source-category metadata. If a later source evaluation proposes a conflicting target delta, PTSIP MUST surface the conflict for project-owner resolution or fail closed; it MUST NOT silently overwrite an already accepted target decision.

### 8. Final Point is cumulative target state, not an intermediate authority chain

All incomplete sources migrate directly into the Final PTSIP Point File. The Final Point may accumulate accepted target deltas from multiple independently evaluated sources.

The system MUST distinguish:

```text
source obligation taxonomy  # recalculated per source
accepted Final Point state  # cumulative target architecture
```

Conflating them would either incorrectly inherit source obligations or incorrectly discard previously accepted target decisions.

### 9. Version and revision identity are mandatory transition inputs

A transition operation MUST inspect the source and target `version` and immutable `revision` identities. It MUST NOT choose migration behavior only from the filename.

A target file that cannot prove a coherent target draft family/revision MUST NOT be promoted.

### 10. Fail-closed states

At minimum, PTSIP MUST fail closed for profile-generation mutation when any of the following is true:

- filename version and internal draft version disagree;
- more than one logical temporary candidate claims the same target version;
- source or target version/revision identity is missing or malformed;
- the Final PTSIP Point cannot be selected deterministically;
- a source changed after its migration analysis snapshot was captured;
- Required Work Elements remain unresolved;
- an accepted Final Point declaration conflicts with a later proposed delta and no project-owned resolution exists;
- promotion would replace canonical `ptsip.yaml` before all participating source migrations complete.

## Relationship to the cancelled Tool 0.3.6.1 plan

The Tool `0.3.6.1` planning family is cancelled and replaced by Tool `0.3.7` planning.

Its evidence-driven migration principles remain useful, especially:

- evidence is not architecture authority;
- legacy input is read without silent reinterpretation;
- migration analysis is non-mutating;
- project-owner confirmation owns architecture-changing writes;
- stale/conflicting state fails closed.

However, those mechanisms now operate inside the profile-generation transition model defined by this ADR. They may not bypass Temporary PTSIP Profile Files, source-specific obligation evaluation, Sequential Work ordering, or guarded Final Point promotion.

## Consequences

### Positive

- old and new draft profiles can coexist without overwriting the migration source;
- migration completion has a precise obligation boundary;
- stale/removed declarations do not become mandatory preservation work;
- optional newly discovered work cannot mask incomplete required migration;
- multiple draft upgrades converge directly on one Final Point instead of repeatedly migrating through obsolete intermediates;
- each source is re-evaluated under its own semantics, reducing accidental inherited assumptions;
- canonical `ptsip.yaml` is replaced only after complete, validated convergence.

### Costs

- migration state becomes more explicit and requires deterministic file/version discovery;
- the tool must track source snapshots and per-source completion separately;
- conflict handling is required when independently evaluated sources propose incompatible Final Point deltas;
- tests must cover multi-generation repositories, interrupted transitions, and promotion failure modes.

## Rejected alternatives

### Edit `ptsip.yaml` in place

Rejected because it destroys the stable migration source before completion and makes rollback/review semantics ambiguous.

### Always migrate through every intermediate draft

Rejected because it repeats classification/review work, increases inconsistency risk, and makes removed intermediate states an unnecessary dependency.

### Copy every old declaration forward

Rejected because removed/stale source declarations have no preservation obligation and target-draft semantics may require re-analysis.

### Treat every repository file as required migration work

Rejected because source-unclassified Asynchronous Work Targets are not part of the source migration completion contract.
