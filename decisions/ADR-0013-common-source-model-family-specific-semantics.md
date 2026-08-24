# ADR-0013 — Common Source Model with Family-Specific Semantics

- **Status:** Accepted
- **Date:** 2026-08-24
- **Tool target:** 0.3.7
- **Work unit:** WU-04 — Source-Draft Compatibility Readers
- **Decision owner:** Project owner
- **Bound Specification:** `0.3.7-draft @ b648d9e026f502b14481ba2d0606d9acc88a31fc`

## Context

Tool 0.3.7 migration must read source Project Profiles without reinterpreting historical vocabulary as current architecture authority.

The two source families required for Tool 0.3.7 are genuinely different:

- Tool 0.3.5 shipped against `0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e` and used `PRODUCT`, `TOOLCHAIN`, and `NEUTRAL_CONTRACT`, with either boundary-root or component declaration forms.
- Tool 0.3.6 shipped against `0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea` and introduced Responsibility Map v2, five lifecycle classifications, roles, typed relationships, associated artifacts, and explicit/template/hybrid declaration modes.

The canonical Tool 0.3.6 validator is intentionally version-bound. Broadening it to accept legacy source profiles would weaken the distinction between compatibility input and current runtime authority.

The project owner selected the balanced architecture option: a common source model for genuinely shared concepts plus family-specific semantic extensions for version-specific meaning.

## Decision

Tool 0.3.7 SHALL implement a read-only compatibility boundary under `src/ptsip/source_compat/`.

### Common source model

The handoff model SHALL expose common source concepts without converting them into current canonical model types:

- exact source-generation binding;
- source component ID;
- source classification token as source vocabulary;
- include/exclude selectors;
- purpose;
- source declaration scope and source location;
- source attributes preserved without reinterpretation;
- associated artifacts and typed relationships where the family actually declares them;
- source policies;
- an immutable representation of the original validated payload.

Compatibility source objects MUST identify themselves as `SOURCE_DECLARATION_ONLY`. They are not canonical effective profiles and do not grant architecture-write authority.

### Family-specific semantics

The common model SHALL be extended by distinct semantic records:

- `V034SourceSemantics` preserves the historical declaration form and boundary-root declarations.
- `V036SourceSemantics` preserves Responsibility Map mode, template identity, hybrid override scope, and whole-entity removal IDs.

The reader MUST NOT materialize a current template while reading an historical template/hybrid source.

### Frozen support units

Support is frozen by both source draft label and immutable Specification revision, not by version label alone.

Tool 0.3.7 supports exactly:

```text
0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e
0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea
```

A known version with a different revision is `UNSUPPORTED_SOURCE_REVISION`. An unknown family is `UNSUPPORTED_SOURCE_FAMILY`.

This prevents a mutable historical draft label from silently acquiring semantics from an unreviewed revision.

### Historical schema boundary

Historical validation SHALL use packaged compatibility schema snapshots separate from the canonical Project Profile validator.

The compatibility reader SHALL NOT call canonical `validate_profile()` or canonical template materialization to reinterpret source input.

### Source-generation binding

The reader receives a `ProfileGenerationIdentity` produced by WU-01 and verifies before projection:

- exact profile path;
- exact file SHA-256;
- declared source draft version;
- specification source;
- immutable specification revision.

A changed source file fails closed as stale evidence.

### No semantic translation in WU-04

WU-04 SHALL NOT perform mappings such as:

```text
TOOLCHAIN -> DEVELOPMENT_TOOLING
```

It SHALL NOT assign target classification, Required Work Element, Removal Migration Element, or Asynchronous Work Target status.

Those are downstream WU-05 responsibilities and require source-specific evaluation against repository evidence and target semantics.

## Consequences

### Positive

- WU-05 receives one stable source-profile interface for shared selectors, IDs, purpose, policies, relationships, and artifacts.
- Historical vocabulary remains visible and reviewable instead of being normalized into current authority.
- 0.3.4 boundary form and 0.3.6 Responsibility Map forms retain their distinct semantics.
- Cross-version comparison can use common fields directly while family-specific analysis remains explicit.
- New source families can reuse the common model when their concepts are genuinely shared without forcing existing families into a single historical type.

### Trade-offs

- The boundary between common fields and family-specific semantics must remain disciplined; superficially similar historical fields must not be promoted into common semantics unless their meaning is actually equivalent.
- Downstream analyzers must inspect family semantics when a migration rule depends on historical declaration mechanics.
- Compatibility schema snapshots are intentionally separate from the canonical current-profile schema so the source reader and current validator can evolve independently.

## Rejected alternatives

### Universal schema-neutral AST with capability model

Rejected for WU-04 because it would make arbitrary producer/family extensibility the primary abstraction. Tool 0.3.7 currently needs strong PTSIP source semantics, and the project owner selected a more PTSIP-aware balanced compatibility boundary.

### Fully versioned typed domains

Rejected because duplicating all shared component/selector/policy concepts per source family would make cross-version migration analysis unnecessarily version-dispatched. Family-specific semantics are retained only where meaning genuinely differs.

## Non-goals

This ADR does not authorize:

- legacy acceptance by canonical runtime validation;
- source-file mutation;
- target profile creation;
- migration obligation categorization;
- lifecycle-classification translation;
- template auto-selection or current-template materialization;
- source-migration completion claims.
