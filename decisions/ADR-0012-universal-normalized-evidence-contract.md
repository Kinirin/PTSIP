# ADR-0012 — Universal Normalized Evidence Contract

- **Status:** Accepted
- **Decision date:** 2026-08-24
- **Target Tool:** 0.3.7
- **Governing work unit:** WU-03 — Evidence/Provenance Normalization
- **Specification family:** `0.3.7-draft @ b648d9e026f502b14481ba2d0606d9acc88a31fc`
- **Related decisions:** ADR-0009, ADR-0010, ADR-0011

## Context

PTSIP evidence is produced through multiple independent surfaces:

- repository candidate discovery;
- native dependency scanners;
- imported external dependency evidence;
- artifact evidence documents;
- agent classification/review documents;
- lifecycle and selector-boundary analysis.

These producers do not share one payload shape. If downstream migration analysis consumes every producer-specific representation directly, WU-04/WU-05 would need to reimplement identity, provenance, conflict, absence, and generation-binding semantics repeatedly.

Tool 0.3.7 additionally requires the same repository observation to be reusable across source-profile evaluations without inheriting a prior source's migration conclusion.

## Decision options considered

### Option A — universality-centered contract

Define a producer-neutral canonical evidence contract. Existing and future evidence producers adapt into that contract without making the canonical contract depend on one scanner or candidate model.

Functional characteristics:

- supports repository-native and external producers through the same semantic representation;
- separates semantic observation identity from evaluation/source-generation identity;
- preserves multiple origins for equivalent assertions;
- represents incompatible values as explicit conflicts;
- exposes analyzed-no-match, not-analyzed, and failed channels separately from explicit assertion values;
- provides a versioned JSON interchange schema.

### Option B — long-term domain-centered evidence subsystem

Model evidence as a PTSIP-specific bounded domain with dedicated subdomains for candidate, dependency, artifact, agent, conflict, provenance, and transition context.

Functional characteristics:

- maximizes PTSIP-internal responsibility separation;
- provides strong domain-specific types and invariants;
- isolates changes between evidence subdomains;
- may require cross-domain composition when one analysis spans multiple evidence kinds.

### Option C — balanced candidate-centered normalization

Use the generation-aware WU-02 candidate model as the main integration object and adapt other evidence forms into candidate-compatible structures.

Functional characteristics:

- closely aligns normalization with current repository discovery semantics;
- reuses WU-02 generation and selector concepts directly;
- provides one migration-facing surface while retaining current inspection conventions;
- makes non-candidate evidence progressively depend on candidate-domain concepts.

## Decision

The project owner selected **Option A — universality-centered contract**.

WU-03 therefore establishes a producer-neutral normalized evidence layer under `src/ptsip/evidence/` and a versioned interchange schema.

The canonical contract is not a new architecture authority. Every normalized record remains:

```text
authority = EVIDENCE_ONLY
```

Project Profile declarations and accepted project-owned decisions retain their existing authority semantics.

## Identity model

Two identities are deliberately separate.

### Semantic evidence identity

`semantic_id` identifies the meaning of an observation and does not include source-profile generation identity.

Equivalent observations from different producers or different source evaluations may therefore share one semantic identity.

### Evaluation identity

`evaluation_id` binds normalized output to:

- repository/worktree snapshot;
- repository revision where available;
- source-profile generation where applicable.

The same semantic observation evaluated for two source generations has the same semantic identity but a different evaluation identity.

This prevents source-specific migration conclusions from becoming inherited authority.

## Assertion and provenance model

A normalized semantic record contains one or more assertions.

Equivalent assertion values converge into one assertion while retaining all producer origins. Origin metadata can include:

- provenance (`DECLARED`, `OBSERVED`, `INFERRED`);
- adapter/producer;
- original evidence ID;
- source path and line;
- evidence document SHA;
- producer identity;
- basis evidence IDs;
- explanatory detail.

Explanatory metadata does not participate in the assertion value when it does not change the semantic claim. For example, two agents agreeing on classification do not conflict merely because their rationale or confidence differs.

## Conflict model

If one semantic record contains incompatible assertion values, the record is `CONFLICT`.

There is no last-writer-wins behavior and no producer priority that silently discards contradictory evidence.

## Absence and analysis-state model

The contract distinguishes:

- an explicit assertion whose value is `false`;
- `NO_MATCH` — the channel was analyzed and produced no matching evidence;
- `NOT_ANALYZED` — the channel was not evaluated;
- `FAILED` — the channel evaluation was incomplete or invalid.

Absence therefore cannot silently become false evidence.

## Boundary evidence rule

WU-03 boundary evidence uses objective resolved dependency relationships together with the existing shared selector/coverage mechanism.

A boundary may report that two uniquely resolved declaration scopes differ. If either side is ambiguous or uncovered, the result remains unresolved and no owner or lifecycle classification is guessed.

Boundary evidence remains `INFERRED` and `EVIDENCE_ONLY`.

## Interchange contract

The canonical interchange format is:

```text
ptsip-normalized-evidence/v1
```

Its schema is maintained in synchronized canonical and embedded copies:

```text
schemas/ptsip-normalized-evidence.schema.json
src/ptsip/specdata/ptsip-normalized-evidence.schema.json
```

The normalized contract is intended to accept future evidence producers without requiring them to adopt WU-02 candidate internals.

## Compatibility boundary

Existing producer-specific classes and conformance behavior are not replaced by this ADR.

They remain valid producer interfaces and are adapted into the normalized layer. This permits progressive adoption by WU-04/WU-05 without silently changing existing conformance authority or profile semantics.

## Consequences

- WU-04/WU-05 can consume one deterministic evidence representation across multiple producers.
- External evidence producers can target a documented interchange shape rather than internal Python candidate classes.
- Provenance survives deduplication instead of being reduced to one winning source.
- Source-generation evaluation remains separate from semantic observation identity.
- Conflicts and missing analysis states remain machine-reviewable.
- Candidate, artifact, dependency, and agent evidence can evolve independently behind adapters while preserving the normalized contract.
- Lifecycle ownership and Required/Removal/Async obligation classification remain outside WU-03 authority.
