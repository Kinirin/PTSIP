# WU-03 — Evidence/Provenance Normalization

> **Status:** ACTIVE  
> **Target Tool:** `0.3.7`  
> **Roadmap predecessor:** WU-02 — candidate-discovery evidence expansion (`COMPLETE / FOCUSED TEST VERIFIED`)  
> **WU-03 exact entry baseline:** `ded216a91edd01aeff02864749708d226fa84724`  
> **Bound Specification at entry:** `0.3.7-draft @ b648d9e026f502b14481ba2d0606d9acc88a31fc`  
> **Successor:** WU-04 — source-draft compatibility readers

## 0. Purpose

Normalize discovery evidence so later source-specific migration analysis can reason consistently without mistaking observations, previous source evaluations, or Final Point contents for architecture authority.

This work retains the cancelled Tool `0.3.6.1` WU-02 intent and adds generation-aware provenance required by ADR-0010.

## 1. Architecture boundary

```text
observed fact
    -> normalized evidence record
    -> provenance + ambiguity metadata
    -> source-specific migration analysis

observed fact
    -X-> authoritative classification
previous source result
    -X-> inherited obligation category
Final Point content
    -X-> proof that a later source obligation is satisfied without evaluation
```

Evidence answers what was observed. Source-specific obligation evaluation answers what must migrate. Project-owned decisions answer what the target architecture means.

## 2. Initial scope

- define stable evidence identity and provenance fields;
- normalize equivalent observations from different adapters;
- preserve `DECLARED`, `OBSERVED`, and `INFERRED` provenance separately from authority;
- preserve source locations and adapter provenance;
- represent contradictory/incomplete evidence explicitly;
- distinguish false, not-observed, and not-analyzed states where absence matters;
- provide deterministic serialization/comparison;
- bind evidence to the repository/worktree snapshot;
- bind the evaluation context to the source profile generation without treating that context as reusable migration authority.

## 3. Generation provenance rule

Sequential Work can evaluate the same repository artifact against multiple source profiles. Normalization must not collapse those evaluations into one inherited conclusion.

The system should be able to distinguish conceptually:

```text
artifact X observed at snapshot S
    for source 0.3.7-draft

artifact X observed at snapshot S
    for source 0.3.6-draft
```

The underlying observation may deduplicate where semantically identical, but source-specific migration conclusions remain separate downstream.

## 4. Work tracks

### WU-03A — evidence contract inventory

Review current evidence payloads, schemas, dataclasses, reports, and pilot artifacts.

### WU-03B — canonical normalization

Define a narrow normalized representation and conversion boundaries from existing adapters.

### WU-03C — boundary evidence

Improve detection of likely lifecycle boundaries from objective relationships without assigning final ownership.

### WU-03D — provenance preservation

Retain enough origin information to explain every normalized record.

### WU-03E — ambiguity/conflict model

Incompatible observations remain reviewable rather than silently collapsed.

### WU-03F — transition-context binding

Represent source-generation and snapshot context using the smallest reusable extension compatible with WU-01 identity primitives.

## 5. Non-goals

WU-03 does not:

- read unsupported historical profiles into canonical runtime state;
- assign Required/Removal/Async categories;
- write Temporary PTSIP Profile Files;
- build Final Point deltas;
- authorize lifecycle classification based solely on evidence.

## 6. Expected deliverables

- normalized evidence/provenance contract;
- deterministic evidence identity/deduplication rules;
- snapshot and source-generation context;
- stronger but non-authoritative boundary discovery;
- tests for provenance retention, duplicate normalization, conflicts, absence states, generation separation, and deterministic output.

## 7. Completion gate

WU-03 is complete when normalized evidence is deterministic, provenance-preserving, reviewable, compatible with WU-02, correctly bound to transition context, suitable for WU-04/WU-05, and incapable of silently becoming architecture or migration-obligation authority.

## 8. Entry discipline

WU-03 entered automatically under the project owner's standing successor-entry authorization after WU-02 completion was recorded and exact `dev/0.3.7` HEAD `ded216a91edd01aeff02864749708d226fa84724` was freshly revalidated.

This ACTIVE state authorizes WU-03 implementation only. It does not authorize WU-04 implementation, Required/Removal/Async categorization, Temporary PTSIP Profile mutation, Final Point delta application, or bypass of any project-owner decision gate.
