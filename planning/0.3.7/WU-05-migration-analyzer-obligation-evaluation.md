# WU-05 — Migration Analyzer and Source-Specific Obligation Evaluation

> **Status:** PRE-CREATED / LOCKED  
> **Target Tool:** `0.3.7`  
> **Roadmap predecessor:** WU-04 — source-draft compatibility readers  
> **Entry baseline:** not assigned; capture fresh branch HEAD on actual entry  
> **Successor:** WU-06 — target proposals and Final Point plan

## 0. Purpose

Analyze one source profile at a time against the current repository and target draft semantics, producing both lifecycle migration findings and the migration-completion obligation categories frozen by ADR-0010.

This strengthens the cancelled Tool `0.3.6.1` WU-04 migration analyzer. The analyzer remains non-mutating and non-authoritative.

## 1. Inputs

```text
WU-04 source read model
+ WU-02/WU-03 normalized repository evidence
+ WU-01 source/final-point transition identity
+ target 0.3.7 draft semantics
    -> source-specific migration analysis
```

## 2. Mandatory obligation categories

Every relevant source evaluation must distinguish:

### PTSIP Required Work Element

A validly existing repository element represented by the source draft's active classified/domain structure and mandatory for that source migration's completion.

### PTSIP Removal Migration Element

A source-mentioned element no longer validly active in the repository. It has no preservation obligation and contributes nothing to completion.

### PTSIP Asynchronous Work Target

A repository element not directly included in the source draft's active obligation. It is optional for the migration and contributes nothing to source completion.

The analyzer must expose why an element received its category, including source declaration facts and current repository evidence.

## 3. Independent evaluation rule

Sequential Work must run this categorization independently for each participating source.

Example:

```text
source 0.3.7-draft -> evaluate taxonomy A
source 0.3.6-draft -> evaluate taxonomy B
canonical 0.3.4-draft -> evaluate taxonomy C
```

A/B/C are not inherited. Previous source conclusions may be available as audit/history but MUST NOT be copied as authoritative input to the next source classification.

## 4. Lifecycle migration findings

In addition to obligation status, the analyzer must retain the distinctions from the cancelled plan:

- exact semantic preservation;
- terminology-only compatibility cases;
- ambiguous historical `TOOLCHAIN` cases;
- possible lifecycle separation;
- missing relationships or associated artifacts;
- stale source declarations;
- new repository candidates absent from the source profile;
- conflicts with already accepted Final Point target state.

No automatic `TOOLCHAIN -> DEVELOPMENT_TOOLING` rule is authoritative.

## 5. Completion model

The analyzer must produce a machine-evaluable source completion contract.

```text
required_total
required_resolved
required_unresolved
removal_count
async_count
```

Completion is:

```text
required_unresolved == 0
```

Removal and Async counts are informational/optional and do not improve completion.

There should be no weighted score that allows optional work to compensate for unresolved Required Work Elements.

## 6. Final Point conflict awareness

The analyzer may compare a source finding with already accepted Final Point declarations to identify:

```text
ALREADY_SATISFIED
COMPATIBLE_TARGET_STATE
CONFLICTING_TARGET_STATE
TARGET_REVIEW_REQUIRED
```

These are analysis findings only. The analyzer MUST NOT overwrite the Final Point.

A source-specific obligation may be considered resolved by existing Final Point state only when the analyzer can demonstrate that the target declaration satisfies the source obligation under the target draft semantics. Mere presence of a similarly named component is insufficient.

## 7. Work tracks

### WU-05A — migration comparison model

Separate source architecture, observed repository state, target semantics, and existing Final Point state.

### WU-05B — obligation taxonomy engine

Implement explicit Required/Removal/Async evaluation with rationale and source identity.

### WU-05C — lifecycle compatibility analysis

Retain historical semantics and surface ambiguous changes instead of guessing.

### WU-05D — evidence correlation

Attach normalized evidence while keeping evidence non-authoritative.

### WU-05E — completion contract

Produce deterministic required-resolution counts and unresolved identities.

### WU-05F — conflict analysis

Detect compatible/conflicting accepted target state without mutating it.

## 8. Non-goals

WU-05 does not:

- construct the final target delta;
- decide owner intent in ambiguous cases;
- write/delete/rename profile files;
- route migration through intermediate sources;
- promote a Final Point;
- count Async work as migration completion.

## 9. Completion gate

WU-05 is complete only when:

- source/current/evidence/target states are separated deterministically;
- Required/Removal/Async categories are explicit and source-specific;
- each category decision is explainable;
- only Required Work Elements contribute to source completion;
- historical vocabulary is not silently canonicalized;
- Final Point compatibility/conflict is reviewable;
- repeated equivalent inputs produce stable analysis;
- focused tests pass;
- WU-06 has not been entered early.

## 10. Entry discipline

Pre-created roadmap document only. Actual entry requires WU-04 completion plus a fresh exact branch HEAD.
