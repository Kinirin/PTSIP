# WU-06 — Evidence/Provenance Normalization and Stronger Boundary Discovery

> **Status:** PRE-CREATED / LOCKED  
> **Roadmap predecessor:** WU-05 — candidate-discovery evidence expansion  
> **Planning baseline:** `7ade7a392bc43749bae6da0e2f0cdf4e1d7818d3`  
> **Entry baseline:** not assigned; capture fresh branch HEAD on actual entry  
> **Successor:** WU-07 — Tool 0.3.5 legacy reader

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
- distinguish observed, declared, derived, and historical/migration evidence where required;
- strengthen component/boundary discovery using dependency, packaging, execution, release, CI, schema, and artifact signals;
- preserve source locations and adapter provenance sufficient for review;
- represent contradictory or incomplete evidence explicitly;
- provide deterministic serialization/comparison for tests and migration analysis.

## 3. Work tracks

### WU-06A — evidence contract inventory
Review all current evidence payloads, schemas, internal dataclasses, reports, and pilot artifacts.

### WU-06B — canonical normalization
Define the narrow normalized representation and conversion boundaries from existing adapters.

### WU-06C — boundary evidence
Improve detection of likely lifecycle boundaries from objective relationships without assigning final lifecycle ownership.

### WU-06D — provenance preservation
Every normalized record must retain enough origin information to explain why it exists and where it came from.

### WU-06E — ambiguity/conflict model
Multiple incompatible observations must remain reviewable rather than being silently collapsed.

## 4. Non-goals

WU-06 does not perform Tool 0.3.5 profile migration, write target profiles, or authorize lifecycle classification based solely on evidence. It must not create a second Responsibility Map authority.

## 5. Expected deliverables

- normalized evidence/provenance contract;
- adapter conversion layer with compatibility handling where needed;
- deterministic evidence identity/deduplication rules;
- stronger but non-authoritative boundary discovery;
- tests for provenance retention, duplicate normalization, conflict/ambiguity, and deterministic output.

## 6. Completion gate

WU-06 is complete when normalized evidence is deterministic, provenance-preserving, reviewable, compatible with WU-05 discovery, suitable as input to WU-07/WU-08, and does not silently mutate architecture or classification authority.

## 7. Entry discipline

Pre-created roadmap document only. Assign the real entry SHA and change to `ACTIVE` only after WU-05 completion review.
