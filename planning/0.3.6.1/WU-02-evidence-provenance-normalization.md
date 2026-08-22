# WU-02 — Evidence/Provenance Normalization and Stronger Boundary Discovery

> **Status:** PRE-CREATED / LOCKED  
> **Target Tool:** `0.3.6.1`  
> **Roadmap predecessor:** WU-01 — candidate-discovery evidence expansion  
> **Planning relocation baseline:** `3529a32c862e1d43a91f732ced36358b4e13e1d9`  
> **Entry baseline:** not assigned; capture fresh branch HEAD on actual entry  
> **Successor:** WU-03 — Tool 0.3.5 legacy reader

## 0. Purpose

Normalize the evidence/provenance model produced by discovery so later migration analysis can reason about repository observations consistently without mistaking observations for architecture authority.

## 1. Architecture boundary

```text
observed fact
    -> normalized evidence record
    -> provenance + confidence/ambiguity metadata
    -> boundary discovery / migration analysis

observed fact
    -X-> authoritative classification
```

Evidence answers **what was observed and where**. The Responsibility Map and accepted project decisions answer **what the architecture means**.

## 2. Initial scope

- define stable evidence identity and provenance fields;
- normalize equivalent evidence emitted by different adapters;
- distinguish `DECLARED`, `OBSERVED`, and `INFERRED` provenance while keeping provenance separate from architecture authority;
- strengthen component/boundary discovery using dependency, packaging, execution, release, CI, schema, and artifact signals;
- preserve source locations and adapter provenance sufficient for review;
- represent contradictory or incomplete evidence explicitly;
- distinguish false, not-observed, and not-analyzed states where absence matters;
- provide deterministic serialization/comparison for tests and migration analysis;
- bind evidence to the repository/worktree snapshot from which it was produced.

## 3. Work tracks

### WU-02A — evidence contract inventory
Review all current evidence payloads, schemas, internal dataclasses, reports, and pilot artifacts.

### WU-02B — canonical normalization
Define the narrow normalized representation and conversion boundaries from existing adapters.

### WU-02C — boundary evidence
Improve detection of likely lifecycle boundaries from objective relationships without assigning final lifecycle ownership.

### WU-02D — provenance preservation
Every normalized record must retain enough origin information to explain why it exists and where it came from.

### WU-02E — ambiguity/conflict model
Multiple incompatible observations must remain reviewable rather than being silently collapsed.

## 4. Non-goals

WU-02 does not perform Tool `0.3.5` profile migration, write target profiles, or authorize lifecycle classification based solely on evidence. It must not create a second Responsibility Map authority.

## 5. Expected deliverables

- normalized evidence/provenance contract;
- adapter conversion layer with compatibility handling where needed;
- deterministic evidence identity/deduplication rules;
- snapshot-bound evidence identity;
- stronger but non-authoritative boundary discovery;
- tests for provenance retention, duplicate normalization, conflict/ambiguity, absence states, and deterministic output.

## 6. Completion gate

WU-02 is complete when normalized evidence is deterministic, provenance-preserving, reviewable, compatible with WU-01 discovery, suitable as input to WU-03/WU-04, and does not silently mutate architecture or classification authority.

## 7. Entry discipline

Pre-created roadmap document only. Assign the real entry SHA and change to `ACTIVE` only after WU-01 completion review.
