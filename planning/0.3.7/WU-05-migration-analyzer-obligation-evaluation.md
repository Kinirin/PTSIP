# WU-05 — Migration Analyzer and Source-Specific Obligation Evaluation

> **Status:** COMPLETE / FOCUSED TEST VERIFIED  
> **Target Tool:** `0.3.7`  
> **Roadmap predecessor:** WU-04 — source-draft compatibility readers (`COMPLETE / FOCUSED TEST VERIFIED`)  
> **WU-05 exact entry baseline:** `be04ac2733870aaf890ccd6bbfad1b4de174f321`  
> **Bound Specification at entry:** `0.3.7-draft @ b648d9e026f502b14481ba2d0606d9acc88a31fc`  
> **Accepted architecture decision:** `ADR-0014 — Strongly Typed Staged Migration Analyzer`  
> **WU-05 implementation content SHA:** `2b832590ff52cb0d939207e957dbec1de5bb4e2d`  
> **Focused verification:** `17 passed / 0 failed` in the isolated staged-analyzer harness  
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

The analyzer exposes why an element received its category, including source declaration identity, selector, source classification, current repository path, correlated evidence, and accepted-target state where supplied.

## 3. Independent evaluation rule

Sequential Work runs this categorization independently for each participating source.

Example:

```text
source 0.3.7-draft -> evaluate taxonomy A
source 0.3.6-draft -> evaluate taxonomy B
canonical 0.3.4-draft -> evaluate taxonomy C
```

A/B/C are not inherited. Previous source conclusions may be available as audit/history but MUST NOT be copied as authoritative input to the next source classification.

WU-05 enforces this structurally by binding one `MigrationAnalysis` to exactly one WU-04 `SourceGenerationBinding` and one repository/evidence snapshot.

## 4. Lifecycle migration findings

In addition to obligation status, the analyzer retains lifecycle findings separately from the obligation category.

Implemented lifecycle findings include:

```text
EXACT_SEMANTIC_PRESERVATION
HISTORICAL_TOOLCHAIN_AMBIGUITY
POSSIBLE_LIFECYCLE_SEPARATION
TARGET_REVIEW_REQUIRED
```

This means a result can validly be:

```text
category = REQUIRED
lifecycle = HISTORICAL_TOOLCHAIN_AMBIGUITY
```

without choosing a target lifecycle.

No automatic `TOOLCHAIN -> DEVELOPMENT_TOOLING` rule is authoritative.

## 5. Completion model

The analyzer produces a machine-evaluable source completion contract.

```text
required_total
required_resolved
required_unresolved
removal_count
async_count
```

Completion remains exactly:

```text
required_unresolved == 0
```

Removal and Async counts are informational/optional and do not improve completion.

There is no weighted score that allows optional work to compensate for unresolved Required Work Elements.

## 6. Final Point / accepted target-state awareness

WU-05 represents accepted target architecture as a separate typed input. The analyzer may produce:

```text
NOT_EVALUATED
ALREADY_SATISFIED
COMPATIBLE_TARGET_STATE
CONFLICTING_TARGET_STATE
TARGET_REVIEW_REQUIRED
```

These values are stored independently from Required/Removal/Async.

A source-specific Required Work Element becomes `resolved` only when accepted target coverage is provably `ALREADY_SATISFIED` or `COMPATIBLE_TARGET_STATE` under the target lifecycle vocabulary.

Historical `TOOLCHAIN` remains unresolved for owner review even when a target component is classified `DEVELOPMENT_TOOLING`, `DELIVERY`, or `OPERATIONS`.

## 7. Implemented staged architecture

The project owner selected the long-term-maintainability option. ADR-0014 freezes the pipeline:

```text
WU-04 Compatibility Source
    -> Source Coverage Projection
    -> Repository Element Resolution
    -> Obligation Taxonomy
    -> Lifecycle Compatibility
    -> WU-03 Evidence Correlation
    -> Accepted Target-State Compatibility
    -> Source Migration Completion
```

### WU-05A — migration comparison model — COMPLETE

Implemented under:

```text
src/ptsip/migration/model.py
```

Source architecture, repository resolution, obligation category, lifecycle finding, evidence correlation, target compatibility, and completion are independent immutable types.

### WU-05B — obligation taxonomy engine — COMPLETE

Repository resolution has four typed outcomes:

```text
ExistingSourceElement
RemovedSourceElement
UncoveredRepositoryElement
AmbiguousSourceElement
```

Only the first three map to Required / Removal / Async. Ambiguous source coverage remains fail-closed and receives no category.

### WU-05C — lifecycle compatibility analysis — COMPLETE

The analyzer preserves source vocabulary and never canonicalizes historical `TOOLCHAIN` automatically. Canonical lifecycle differences become reviewable conflicts/separation findings rather than silent conversions.

### WU-05D — evidence correlation — COMPLETE

WU-03 normalized evidence is attached by semantic ID, source path, subject, and qualifier path. Evidence conflict and incomplete channels remain findings only:

```text
EVIDENCE_CONFLICT
EVIDENCE_INCOMPLETE
```

Evidence cannot change the obligation category.

### WU-05E — completion contract — COMPLETE

`SourceMigrationCompletion` computes Required-only completion deterministically.

### WU-05F — target conflict analysis — COMPLETE

Accepted explicit target state can be compared without mutation. Exact/compatible/conflicting/review-required results remain separate from source taxonomy.

Typed source relationships missing from accepted target state become reviewable `MISSING_RELATIONSHIP` findings rather than automatic target writes.

## 8. Source-family handling

### Tool 0.3.5 / 0.3.4-draft

Both component form and historical boundary-root form project into source coverage while preserving:

```text
PRODUCT
TOOLCHAIN
NEUTRAL_CONTRACT
```

### Tool 0.3.6 / 0.3.6-draft

- explicit Responsibility Map declarations project directly;
- immutable template/hybrid source declarations are materialized only inside the source-coverage projection stage;
- WU-04 source objects/raw payload remain unchanged;
- template-effective source projection does not become target authority.

## 9. Freshness and fail-closed rules

WU-05 validates before accepting an analysis as valid:

1. current source-profile bytes still match the WU-04 `content_sha256`;
2. normalized evidence source generation matches the WU-04 source generation;
3. normalized evidence snapshot matches the repository snapshot under analysis;
4. target state uses the selected target draft/classification vocabulary;
5. repository snapshot remains stable from analysis start to finish;
6. source coverage is not ambiguous.

Stale or mismatched state is recorded as an explicit `MigrationAnalysisIssue` and makes the analysis invalid.

## 10. Non-goals preserved

WU-05 does not:

- construct the final target delta;
- decide owner intent in ambiguous lifecycle cases;
- write/delete/rename profile files;
- route migration through intermediate sources;
- promote a Final Point;
- count Async work as migration completion;
- silently materialize a target template/hybrid profile as accepted architecture.

`target_state_from_mapping()` therefore accepts explicit accepted target state only. Target-profile planning/materialization belongs to later target-draft boundaries.

## 11. Verification

Role-scoped test file:

```text
tests/ptsip/migration/test_analyzer_037.py
```

Focused isolated verification result:

```text
17 passed / 0 failed
```

Covered behaviors include:

- Required / Removal / Async separation;
- Required unresolved without accepted target state;
- historical `TOOLCHAIN` ambiguity;
- exact target satisfaction;
- semantically compatible target satisfaction with different stable ID;
- lifecycle conflict without silent conversion;
- equal-specificity source ambiguity fail-closed;
- evidence conflict preservation without authority escalation;
- evidence source-generation mismatch;
- missing source relationship finding;
- historical boundary-root projection;
- failed / not-analyzed evidence channels remaining distinct from false;
- Async non-contribution to completion;
- source bytes stale after WU-04 read;
- explicit-only accepted-target mapping boundary;
- immutable template source projection;
- deterministic analysis digest.

The isolated harness reproduces the WU-01/WU-03/WU-04 public contracts required by this analyzer. Full repository regression and exact-SHA self-hosted workflow verification remain WU-08 responsibilities.

## 12. Completion gate

WU-05 completion criteria are satisfied:

- source/current/evidence/target states are separated deterministically;
- Required/Removal/Async categories are explicit and source-specific;
- each category decision is explainable;
- only Required Work Elements contribute to source completion;
- historical vocabulary is not silently canonicalized;
- accepted target compatibility/conflict is reviewable and non-mutating;
- repeated equivalent inputs produce stable analysis identity;
- focused tests pass;
- WU-06 implementation has not been entered early.

## 13. Entry discipline

WU-05 entered automatically under the project owner's standing successor-entry authorization after WU-04 completion was recorded and exact `dev/0.3.7` HEAD `be04ac2733870aaf890ccd6bbfad1b4de174f321` was freshly revalidated.

WU-05 implementation was performed only after the project owner explicitly selected the long-term-maintainability staged-analyzer option.

Completion of WU-05 authorizes only the standing successor entry rule. It does not itself authorize WU-06 implementation, target-delta acceptance, Temporary Profile mutation, Final Point mutation, or profile promotion.
