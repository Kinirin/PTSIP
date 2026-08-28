# WU-03 — Evidence/Provenance Normalization

> **Status:** COMPLETE / FOCUSED TEST VERIFIED  
> **Target Tool:** `0.3.7`  
> **Roadmap predecessor:** WU-02 — candidate-discovery evidence expansion (`COMPLETE / FOCUSED TEST VERIFIED`)  
> **WU-03 exact entry baseline:** `ded216a91edd01aeff02864749708d226fa84724`  
> **Bound Specification at entry:** `0.3.7-draft @ b648d9e026f502b14481ba2d0606d9acc88a31fc`  
> **Accepted architecture decision:** `ADR-0012 — Universal Normalized Evidence Contract`  
> **WU-03 implementation content SHA:** `316fe2fc42d03b418f67bcfc80c30748f2f9f00e`  
> **Focused verification:** `10 passed / 0 failed` across isolated core and adapter harnesses  
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

The system distinguishes:

```text
semantic observation identity
    !=
source-generation evaluation identity
```

Equivalent underlying observations may deduplicate by `semantic_id`. The repository snapshot and source generation are bound separately by `evaluation_id`, so downstream source-specific migration conclusions remain independent.

## 4. Work tracks

### WU-03A — evidence contract inventory — COMPLETE

Reviewed the current evidence surfaces:

- `DependencyEdge` native scanner output;
- WU-02 `CandidateObservation` / `CandidateEvidence`;
- imported external dependency evidence;
- artifact evidence documents and revision bindings;
- agent classification/review documents;
- conformance conflict and lifecycle evidence behavior.

The inventory confirmed that producer-specific payloads use different identity and provenance shapes. WU-05 must therefore not be required to interpret each producer contract directly.

### WU-03B — canonical normalization — COMPLETE

Added producer-neutral package:

```text
src/ptsip/evidence/
    __init__.py
    contract.py
    normalization.py
    adapters.py
```

Canonical normalized form:

```text
NormalizedEvidenceSet
    -> EvidenceEvaluationContext
    -> EvidenceRecord[]
        -> semantic_id
        -> subject / predicate / qualifiers
        -> EvidenceAssertion[]
            -> assertion value
            -> EvidenceOrigin[]
        -> CONSISTENT | CONFLICT
        -> authority = EVIDENCE_ONLY
    -> EvidenceChannel[]
    -> EvidenceNormalizationIssue[]
```

Equivalent assertion values converge while all origins remain preserved. Incompatible values become explicit `CONFLICT`; no producer wins silently.

### WU-03C — boundary evidence — COMPLETE

Boundary evidence now has a reusable adapter rule based on:

```text
resolved dependency edge
    + shared selector/coverage resolution
    -> declared-scope boundary evidence
```

If source and target each resolve uniquely and belong to distinct declaration scopes, `crosses_declared_scope = true` is recorded as inferred evidence.

If either side is ambiguous or uncovered:

```text
crosses_declared_scope = null
```

and the underlying coverage state is retained. No lifecycle classification or owner guess is created.

### WU-03D — provenance preservation — COMPLETE

Each normalized origin can retain:

- `DECLARED`, `OBSERVED`, or `INFERRED` provenance;
- adapter/producer identity;
- original evidence ID;
- source path and line;
- evidence-document SHA-256;
- producer ID;
- basis evidence IDs;
- explanatory detail.

Imported external dependency notes are decoded into structured source-document/producer/SHA provenance where the existing external-evidence loader provides that metadata.

Explanatory metadata is separated from assertion values where it does not change the semantic claim. For example, two agent reviews agreeing on classification do not conflict merely because rationale or confidence differs.

### WU-03E — ambiguity/conflict/absence model — COMPLETE

The normalized contract distinguishes:

```text
explicit assertion value false
NO_MATCH       # analyzed, no matching evidence
NOT_ANALYZED   # channel not evaluated
FAILED         # channel evaluation incomplete/invalid
```

Incompatible assertion values are represented by multiple assertions under one semantic record with status `CONFLICT`.

No last-writer-wins rule exists.

### WU-03F — transition-context binding — COMPLETE

The WU-02 generation-aware context is adapted into the producer-neutral evidence contract without making that contract depend on WU-02 candidate internals.

`semantic_id` excludes source generation. `evaluation_id` binds:

- repository root;
- revision/HEAD where available;
- status fingerprint;
- tracked-content fingerprint;
- source-profile path/version/specification revision/content SHA where applicable.

This preserves observation deduplication while preventing source-specific migration conclusions from being inherited across Sequential Work.

## 5. Non-goals preserved

WU-03 did not:

- read unsupported historical profiles into canonical runtime state;
- assign Required/Removal/Async categories;
- write Temporary PTSIP Profile Files;
- build Final Point deltas;
- authorize lifecycle classification based solely on evidence;
- replace the existing conformance engine wholesale.

Existing producer-specific classes remain valid. WU-03 adds adapters so migration work can adopt the universal contract progressively.

## 6. Interchange schema

The accepted universality-centered option includes a versioned producer-neutral interchange format:

```text
ptsip-normalized-evidence/v1
```

Synchronized schema copies:

```text
schemas/ptsip-normalized-evidence.schema.json
src/ptsip/specdata/ptsip-normalized-evidence.schema.json
```

Both copies resolve to the same Git blob SHA:

```text
474016c5f54bae85b17cd59701a30fee32e7e983
```

The embedded schema is already covered by existing `pyproject.toml` package-data rules for `ptsip/specdata/*.json`.

## 7. Focused verification

Public clone execution was attempted but the available container could not resolve `github.com`.

Verification therefore used isolated execution with the exact normalized-core semantics and repository-compatible adapter signatures.

### Core harness

```text
5 passed / 0 failed
```

Covered:

- semantic ID remains stable across different source generations;
- evaluation ID changes with source generation;
- equivalent producer assertions deduplicate while retaining both origins;
- incompatible values become conflict;
- explicit false differs from `NO_MATCH` / `NOT_ANALYZED`;
- input reordering preserves deterministic output/digest.

### Adapter harness

```text
5 passed / 0 failed
```

Covered:

- equivalent native/external dependency observations converge;
- contradictory dependency resolution becomes conflict;
- agent rationale/confidence differences do not create false conflict when the semantic classification claim agrees;
- artifact set-like content ordering normalizes deterministically;
- resolved dependency + selector coverage reports real cross-scope boundary evidence;
- ambiguous ownership remains unresolved rather than guessed;
- external evidence document path/producer/SHA provenance is retained structurally.

The existing WU-02 implementation modules were not modified by WU-03. WU-02 remains the producer-side discovery layer; the new adapter consumes its public evidence/context objects.

Full repository regression and exact-SHA workflow verification remain owned by WU-08.

## 8. Completion gate

WU-03 completion gate is satisfied at the focused-verification boundary:

- normalized evidence is deterministic;
- provenance is preserved through deduplication;
- conflicting assertions remain reviewable;
- explicit false and analysis absence states are distinct;
- WU-02 candidate evidence is consumable through an adapter without becoming the canonical contract itself;
- snapshot and source-generation context remain bound separately from semantic evidence identity;
- boundary evidence uses objective resolved relationships and shared selector semantics;
- every normalized record remains `EVIDENCE_ONLY`;
- the representation is suitable for WU-04/WU-05 consumption;
- no Required/Removal/Async or architecture authority was introduced.

## 9. Entry and successor discipline

WU-03 entered automatically under the project owner's standing successor-entry authorization after WU-02 completion was recorded and exact `dev/0.3.7` HEAD `ded216a91edd01aeff02864749708d226fa84724` was freshly revalidated.

The project owner subsequently selected the **universality-centered architecture option**, now recorded as ADR-0012. Implementation followed that approved decision.

After this completion is recorded, WU-04 may be entered automatically only after a fresh exact `dev/0.3.7` HEAD is captured and no unresolved project-owner decision, architecture conflict, Specification boundary, or explicit confirmation gate blocks entry. Auto-entry authorizes WU-04 entry state only, not WU-04 implementation.
