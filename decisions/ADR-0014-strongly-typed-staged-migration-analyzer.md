# ADR-0014 — Strongly Typed Staged Migration Analyzer

**Status:** Accepted  
**Decision:** Implement WU-05 as a strongly typed staged migration analyzer that separates source coverage, repository resolution, obligation taxonomy, lifecycle compatibility, evidence correlation, accepted-target compatibility, and completion accounting.  
**Applies to:** Tool `0.3.7` WU-05 and downstream WU-06/WU-07 consumers of source-specific migration analysis.

## Context

WU-01 established source-generation and repository snapshot identity. WU-03 established a normalized evidence contract whose authority is explicitly `EVIDENCE_ONLY`. WU-04 established read-only historical source models that preserve Tool `0.3.5` / `0.3.4-draft` and Tool `0.3.6` semantics without silently canonicalizing historical vocabulary.

WU-05 must combine those inputs to determine, independently for each source generation:

```text
PTSIP Required Work Element
PTSIP Removal Migration Element
PTSIP Asynchronous Work Target
```

while also surfacing lifecycle compatibility and accepted Final Point compatibility.

The analyzer must not allow any one of those axes to silently overwrite another. In particular:

- `Required` does not imply a target lifecycle classification;
- evidence does not become project-owned architecture authority;
- accepted Final Point state does not rewrite source-specific obligation taxonomy;
- historical `TOOLCHAIN` does not automatically become `DEVELOPMENT_TOOLING`.

## Decision

### 1. Use a staged analysis pipeline

WU-05 uses the following ordered stages:

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

Each stage produces immutable typed output. Later stages may attach findings or resolution state but do not mutate the meaning of earlier stages.

### 2. Source Coverage Projection is source-semantic, not target-semantic

The projection stage answers only what the source profile actively covered.

For Tool `0.3.5` / `0.3.4-draft`, both component declarations and historical boundary roots are supported.

For Tool `0.3.6`:

- explicit declarations are projected directly;
- immutable template/hybrid declarations are materialized only inside this source-projection stage using the frozen template revision;
- the WU-04 source object and raw declaration remain unchanged;
- effective template projection is not target authority and is not reused as a target declaration.

Historical classifications are preserved verbatim. `TOOLCHAIN` remains `TOOLCHAIN` in the projected source coverage.

### 3. Repository resolution has four typed outcomes

A current repository element is resolved into one of:

```text
ExistingSourceElement
RemovedSourceElement
UncoveredRepositoryElement
AmbiguousSourceElement
```

Ambiguous source coverage is not coerced into Required, Removal, or Async. It remains fail-closed analysis state.

### 4. Obligation taxonomy is mechanically restricted

The taxonomy stage maps only unambiguous repository-resolution types:

```text
ExistingSourceElement
    -> RequiredWorkElement

RemovedSourceElement
    -> RemovalMigrationElement

UncoveredRepositoryElement
    -> AsynchronousWorkTarget
```

`AmbiguousSourceElement` produces no obligation category until ambiguity is resolved.

This protects the ADR-0010 rule that Required/Removal/Async are source-specific migration-control categories rather than confidence-weighted labels.

### 5. Lifecycle compatibility is a separate axis

Lifecycle findings are stored independently from obligation categories.

Examples:

```text
category = REQUIRED
lifecycle = EXACT_SEMANTIC_PRESERVATION
```

or:

```text
category = REQUIRED
lifecycle = HISTORICAL_TOOLCHAIN_AMBIGUITY
```

The second example remains unresolved until project-owned target intent is established. Required status itself cannot choose the target lifecycle.

### 6. Evidence correlation is non-authoritative

Normalized evidence is correlated to repository elements through stable evidence IDs, subjects, qualifiers, and origin source paths.

Evidence conflicts and incomplete channels are preserved as findings:

```text
EVIDENCE_CONFLICT
EVIDENCE_INCOMPLETE
```

They do not change Required/Removal/Async category assignment.

The analyzer verifies that normalized evidence is bound to the same source generation and repository snapshot before treating the analysis as valid.

### 7. Source and repository freshness are fail-closed

Before analysis the analyzer verifies:

- current source-profile bytes still match the WU-04 `content_sha256` binding;
- normalized evidence source-generation identity matches the WU-04 source;
- normalized evidence repository snapshot matches the repository snapshot being analyzed.

The repository snapshot is captured again after analysis. Any relevant change invalidates the analysis.

### 8. Accepted target state is a separate typed input

Accepted target architecture, when available, is represented separately from source architecture.

WU-05 recognizes:

```text
NOT_EVALUATED
ALREADY_SATISFIED
COMPATIBLE_TARGET_STATE
CONFLICTING_TARGET_STATE
TARGET_REVIEW_REQUIRED
```

These statuses do not alter the source obligation category.

For historical `TOOLCHAIN`, target coverage by `DEVELOPMENT_TOOLING`, `DELIVERY`, or `OPERATIONS` is not automatically considered resolved. It remains `TARGET_REVIEW_REQUIRED`.

### 9. Completion is Required-only

The completion contract is exactly:

```text
required_total
required_resolved
required_unresolved
removal_count
async_count
```

and:

```text
complete == (required_unresolved == 0)
```

Removal and Async never compensate for unresolved Required Work Elements.

### 10. Deterministic output

Analysis output is sorted before serialization and exposes a deterministic SHA-256 digest. Equivalent source, repository, evidence, target semantics, and accepted target state produce equivalent analysis identity.

## Consequences

### Positive

- category, lifecycle, evidence, and target compatibility cannot silently overwrite one another;
- invalid intermediate states are represented explicitly and can fail closed;
- historical vocabulary is preserved through migration analysis;
- accepted Final Point state can satisfy an obligation without becoming the source of that obligation;
- future lifecycle rules can evolve without rewriting Required/Removal/Async taxonomy;
- source-generation and snapshot freshness are explicit analyzer invariants;
- downstream WU-06 receives deterministic typed findings rather than loosely structured mixed analysis.

### Functional tradeoffs

- new migration-subject kinds must be added deliberately to the relevant stage types instead of appearing dynamically;
- a future Specification that makes relationships or associated artifacts independent completion obligations will require explicit obligation-domain extension;
- cross-stage rules must have a clear owning stage, so new semantics cannot be added as unrestricted cross-cutting inference.

## Rejected alternatives

### Declarative fact/rule/proof engine

Not selected for WU-05 because rule interaction would make correctness depend on rule-conflict semantics rather than typed stage invariants.

### Common obligation core with family-specific strategies

Not selected because it gives the family strategy/common-engine boundary more semantic responsibility than desired for long-term maintenance. WU-05 instead projects all supported source families into typed source coverage first and keeps the later stages family-independent.

### Automatic historical vocabulary conversion

Rejected. `TOOLCHAIN -> DEVELOPMENT_TOOLING` is not a valid reader or analyzer default.

## Verification boundary

Focused WU-05 verification covers at minimum:

- Required / Removal / Async separation;
- unresolved Required without accepted target state;
- historical `TOOLCHAIN` ambiguity;
- exact and compatible target satisfaction;
- conflicting lifecycle state;
- ambiguous source coverage fail-closed;
- evidence conflict and incomplete-channel preservation;
- evidence source-generation mismatch;
- source bytes stale after WU-04 read;
- legacy boundary projection;
- immutable template source projection;
- relationship findings;
- Required-only completion;
- deterministic output.

Full repository regression and exact-SHA workflow verification remain WU-08 responsibilities.
