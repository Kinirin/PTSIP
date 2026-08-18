# ADR-0009 — Freeze Responsibility Map Declaration Authority and Materialization Boundary

**Status:** Accepted  
**Decision:** Freeze Tool `0.3.6` declaration-source authority and Canonical Effective Responsibility Map semantics  
**Applies to:** Responsibility Map v2 `explicit | template | hybrid` modes and deterministic materialization

## Context

WU-03 established three canonical Responsibility Map declaration modes:

```text
explicit
template
hybrid
```

WU-04 then introduced a version/revision-bound template catalog and deterministic materializer.

Without an explicit authority boundary, `source_mode` could be misread as lifecycle ownership, template defaults could silently become project architecture authority, field-level hybrid inheritance could fragment ownership of one entity, and materialization code could drift into inference or conflict repair.

That would conflict with existing PTSIP principles:

- `classification` is primary lifecycle ownership;
- candidate discovery/evidence does not own architecture;
- project-owned declarations remain architecture authority;
- ambiguous architecture changes fail closed;
- deterministic tooling may reproduce declared intent but must not invent it.

## Decision

PTSIP separates three responsibility boundaries.

### 1. Lifecycle responsibility

Lifecycle responsibility is represented by canonical `classification`:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

It answers which lifecycle primarily owns a coherent architectural responsibility.

Declaration source does not change lifecycle ownership.

A component classified `DELIVERY` remains `DELIVERY` whether its declaration originated from an explicit profile, a selected template revision, or a hybrid project override.

### 2. Declaration authority

`source_mode` describes the source/authority structure of the Responsibility Map declaration, not lifecycle ownership.

Canonical source modes are:

```text
explicit
template
hybrid
```

#### Explicit

The project directly owns the complete declared map.

#### Template

The project owns the explicit decision to adopt one exact template identity (`id + immutable revision`).

The selected immutable template revision supplies the declaration content the project adopted.

Template selection is never inferred from path, repository shape, framework, language, manifest, package manager, or confidence score.

#### Hybrid

The project owns:

- the exact template selection;
- explicit stable-ID replacements;
- explicit stable-ID extensions; and
- explicit stable-ID removals.

The selected template owns only declarations that remain unchanged after those project-owned decisions are applied.

Precedence is:

```text
project override / extension / removal
    > selected immutable template declaration
```

### 3. Materialization responsibility

The PTSIP materializer is non-authoritative.

Its sole architectural responsibility is to reproduce declared authority deterministically as one Canonical Effective Responsibility Map.

It MUST NOT:

- select a template automatically;
- infer missing architecture;
- change lifecycle classification;
- silently repair broken relationships;
- silently delete dangling declarations;
- invent replacement entities;
- collapse conflicting project/template facts by confidence;
- mutate the source profile merely to obtain a valid map.

Invalid effective architecture fails closed and is reported for project-owner action.

## Whole-entity hybrid replacement

Tool `0.3.6` hybrid overrides replace complete entities by stable ID.

Field-level patch inheritance is intentionally not introduced.

For example, when a template contains component `package` and the project supplies its own component with ID `package`, the project replacement owns the entire resulting entity.

This avoids a fragmented authority model such as:

```text
classification -> template
roles          -> template
include        -> project
purpose        -> template
```

Instead:

```text
entity absent from project override
    -> TEMPLATE origin

existing template entity replaced by same project ID
    -> PROJECT_OVERRIDE origin

new project ID
    -> PROJECT_EXTENSION origin

selected template ID explicitly removed
    -> PROJECT_REMOVAL decision
```

Removed entities do not exist in the effective map.

## Derived provenance

Materialization may expose runtime/review provenance using the following planning vocabulary:

```text
PROJECT_EXPLICIT
TEMPLATE
PROJECT_OVERRIDE
PROJECT_EXTENSION
PROJECT_REMOVAL
```

This provenance explains how an effective entity was resolved.

It is not a lifecycle classification and is not required to be persisted into the canonical Project Profile.

## Canonical Effective Responsibility Map

All source modes resolve to one downstream semantic form:

```text
explicit ----+
template ----+--> Canonical Effective Responsibility Map
hybrid ------+
```

The effective map provides the same semantic collections used by downstream logic:

- components;
- associated artifacts;
- typed relationships;
- component dependency policy where applicable; and
- project policies.

Downstream validation, conformance, clarification, build/lifecycle/dependency evaluation, and narrow VPMS metadata consumption should use the effective view rather than reimplement declaration-mode logic independently.

The original source declaration remains separately available for provenance, review, migration diff, and safe apply.

## Effective-map identity

A deterministic semantic digest should identify effective-map equivalence:

```text
effective_map_digest = sha256(canonical semantic effective-map representation)
```

The digest supports reproducibility and comparison but does not replace the source declaration, template identity, or project authority.

## Fail-closed materialization

The materializer must not automatically repair architecture conflicts.

Examples that require explicit failure include:

- removing a component while retaining a relationship to it;
- replacing an associated artifact with an invalid/missing anchor;
- component/associated-artifact ID collision;
- remove of an unknown template ID;
- replace and remove of the same ID;
- effective selector or endpoint conflicts that violate canonical validation.

Project-owned declarations must be corrected explicitly rather than silently normalized into different architecture intent.

## Consequences

### Positive

- `source_mode` cannot be confused with PTSIP lifecycle classification.
- Template-backed and explicit projects can be compared through one effective semantic map.
- Hybrid precedence is deterministic and explainable.
- Template upgrades cannot silently change projects bound to older revisions.
- Migration preview can distinguish template defaults, project overrides, extensions, and removals.
- Downstream evaluators do not need separate explicit/template/hybrid architecture semantics.

### Tradeoffs

- Whole-entity overrides are more verbose than field-level patches.
- Project overrides must repeat fields that remain semantically unchanged when replacing a template entity.
- Invalid removals or dangling references require explicit project correction rather than automatic repair.

These tradeoffs are accepted because they keep authority provenance and architecture meaning deterministic.

## Rejected alternatives

### Treat `source_mode` as another responsibility classification

Rejected because declaration source and lifecycle ownership are orthogonal axes.

### Infer templates from repository evidence

Rejected because evidence/candidate discovery is not project architecture authority.

### Field-level hybrid merge by default

Rejected because it fragments authority provenance inside one responsibility entity and complicates migration/CAS/semantic review.

### Silent cascading removal

Rejected because deleting related relationships/artifacts automatically would make the materializer an architecture decision-maker.

### Confidence-based conflict resolution

Rejected because confidence is review metadata, not architecture authority.

## Implementation sequencing

This ADR is the accepted WU-04C decision boundary.

WU-04D and later integration stages must not reinterpret these semantics for convenience.

The next implementation steps must preserve:

```text
source declaration
    + exact template identity where selected
    + project override decisions
        |
        v
non-authoritative deterministic materializer
        |
        v
Canonical Effective Responsibility Map
        |
        v
downstream consumers
```
