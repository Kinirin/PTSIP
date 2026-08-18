# PTSIP Responsibility Map v2

**Specification family:** `0.3.6-draft`  
**Status:** Active normative companion specification  
**Scope:** Component roles, typed responsibility relationships, associated artifacts, declaration authority, and materialization

This document is part of the canonical PTSIP `0.3.6-draft` Specification family. It refines the Responsibility Map requirements introduced by `PTSIP-RMAP-001` through `PTSIP-RMAP-003` in `PTSIP-SPEC.md`.

It does not weaken or override universal PTSIP lifecycle, dependency, packaging, conformance, migration, or authority rules. Where an implementation cannot satisfy both this document and another mandatory PTSIP rule, the architecture is not made valid merely by using Responsibility Map metadata.

The design rationale is recorded in `decisions/ADR-0008-responsibility-roles-relationships-associated-artifacts.md` and `decisions/ADR-0009-responsibility-map-declaration-authority.md`.

## 1. Semantic axes

Responsibility Map v2 MUST keep these axes distinct:

```text
classification
    = primary lifecycle ownership

roles
    = coarse responsibility characteristics inside that lifecycle

relationships
    = typed directed semantics between declared endpoints

VPMS Verification Purpose
    = what a Verification Case protects/verifies
```

A Tool MUST NOT derive one axis as a lossy alias of another unless a separate normative rule explicitly permits that derivation.

## 2. Component roles

### PTSIP-RMAP-004 — Roles are optional, multi-valued, and non-classifying

A component MAY declare zero, one, or multiple responsibility roles.

Roles MUST NOT replace `classification` or `purpose`. A role MUST NOT by itself decide primary lifecycle ownership.

A role MUST NOT automatically imply the existence, target, or type of a relationship.

When several role characteristics apply, implementations MUST preserve them as separate role values rather than manufacture composite role names.

For example:

```yaml
roles:
  - VERIFICATION
  - AUTOMATION
```

is valid conceptual meaning, while a generated token such as `VERIFICATION_AUTOMATION` is not part of the canonical Tool `0.3.6` vocabulary.

### PTSIP-RMAP-005 — Canonical role vocabulary

The canonical Tool `0.3.6` component role vocabulary is exactly:

- `IMPLEMENTATION`;
- `VERIFICATION`;
- `AUTOMATION`;
- `CONFIGURATION`;
- `DOCUMENTATION`;
- `GOVERNANCE`.

Their meanings are:

#### `IMPLEMENTATION`

The component directly realizes its declared responsibility through application, SDK/library, runtime, developer-tool, delivery, operational, or other implementation logic.

#### `VERIFICATION`

The component's own responsibility includes checking, validating, testing, auditing, or establishing correctness, conformance, acceptance, or policy satisfaction.

`VERIFICATION` does not identify what is verified and does not determine VPMS Verification Purpose.

#### `AUTOMATION`

The component automates or orchestrates repeatable lifecycle work. `AUTOMATION` is valid under any lifecycle classification whose governing obligation supports that responsibility.

#### `CONFIGURATION`

The component's independently governable responsibility is primarily configuration/state declaration. File format or declarative syntax alone MUST NOT create this role.

#### `DOCUMENTATION`

The component's independently governable responsibility is maintaining descriptive/explanatory documentation.

Documentation that is merely subordinate to another component and lacks an independent lifecycle SHOULD be represented as an associated artifact rather than promoted solely to obtain this role.

#### `GOVERNANCE`

The component's independently governable responsibility is project authority, policy, architecture governance, lifecycle decision control, or equivalent governance responsibility.

Governance/authority material that is merely subordinate to one component and lacks independent lifecycle ownership SHOULD be represented as an associated artifact.

The canonical role set is intentionally small. Tool `0.3.6` MUST NOT add lifecycle-specific composite roles merely to encode relationship combinations.

## 3. Typed responsibility relationships

### PTSIP-RMAP-006 — Directed typed relationship semantics

A Responsibility Map typed relationship is a project-declared directed semantic edge:

```text
source --TYPE--> target
```

The source is the responsibility/artifact that depends on, acts on, derives, verifies, delivers, operates, documents, specifies, or governs the target according to `TYPE`.

A relationship MUST identify a valid declared source endpoint, valid declared target endpoint, and one canonical relationship type.

Relationship direction MUST be stable and MUST NOT be inferred from YAML/JSON nesting or endpoint classification.

A Tool MUST NOT fabricate unresolved endpoints merely to preserve an edge.

### PTSIP-RMAP-007 — Canonical project-declared relationship vocabulary

The canonical Tool `0.3.6` Responsibility Map relationship vocabulary is exactly:

#### Dependency/use

- `IMPORTS` — source imports target implementation/module/package semantics.
- `LINKS` — source links target as build/link implementation dependency.
- `LOADS` — source dynamically loads target implementation/artifact.
- `INVOKES` — source executes or triggers target responsibility/process.
- `READS` — source consumes target as data, configuration, contract, or input rather than executable implementation reuse.

#### Derivation/build/delivery

- `GENERATES` — source derives generated source, data, metadata, configuration, or artifact from or for target.
- `BUILDS` — source materializes or assembles a build output/build unit for target.
- `PACKAGES` — source includes or assembles target content into a package/distribution boundary.
- `PUBLISHES` — source publishes or distributes a release target associated with target responsibility.
- `DEPLOYS` — source places or activates a release target associated with target at a destination environment.

#### Verification/operations

- `VERIFIES` — source checks or establishes correctness, conformance, acceptance, compatibility, or policy satisfaction of target.
- `MANAGES` — source performs ongoing post-delivery-handoff operational management of target/deployed state.

#### Documentation/authority

- `DOCUMENTS` — source provides descriptive/explanatory information about target.
- `SPECIFIES` — source defines normative technical semantics, required contract meaning, or authoritative interface/behavior expectations for target.
- `GOVERNS` — source carries project authority, lifecycle policy, decision control, or governance rules over target.

`PRODUCES`, `SUPPORTS`, `USES`, and generic `DEPENDS_ON` are not canonical Tool `0.3.6` Responsibility Map relationship types. Implementations MUST prefer the more specific canonical meaning when one is available.

### Relationship normalization

`GENERATES`, `BUILDS`, and `PACKAGES` are distinct:

```text
GENERATES
    derivation/generation

BUILDS
    materialization/assembly of a build unit

PACKAGES
    inclusion/assembly into a package/distribution boundary
```

One source MAY have several relationships to the same target when several semantics are materially true.

For example, a Delivery automation may both `BUILDS` and `PUBLISHES` one Product/SDK responsibility. Implementations MUST preserve those as separate edges rather than collapsing them into a composite role.

## 4. Evidence relationships remain distinct

### PTSIP-RMAP-008 — Project declaration, evidence, and policy are separate

Responsibility Map relationships are project-owned architecture declaration. Evidence edges are observed/declared/inferred facts collected for analysis. Dependency policy is permission/constraint metadata.

These MUST remain separate:

```text
observed/inferred evidence
    -> may support or contradict relationship proposal

project-owner declaration
    -> typed Responsibility Map relationship

policy
    -> constrains what is allowed
```

An evidence edge MUST NOT silently become a project-owned relationship solely because both vocabularies contain related terms.

A declared relationship MUST NOT be treated as proof that observed repository behavior agrees.

A typed relationship MUST NOT be interpreted as permission to violate a PTSIP mandatory rule.

An allow-policy entry MUST NOT be interpreted as proof that the relationship exists.

The evidence vocabulary retains `TESTS`. Responsibility Map v2 uses the broader declared relationship `VERIFIES`. An observed/declared evidence `TESTS` edge MAY support a `VERIFIES` migration proposal, but MUST NOT automatically create that project-owned edge.

## 5. Associated artifacts

### PTSIP-RMAP-009 — Associated artifact eligibility

An **Associated Artifact** is a project-owned, non-component support surface subordinate to exactly one classified anchor component.

An associated artifact MUST:

1. have stable Responsibility Map identity;
2. have explicit project-owned scope/selectors and purpose;
3. have exactly one anchor component;
4. be non-executable in its architectural role;
5. have no independently governable primary lifecycle responsibility;
6. have no independent release/compatibility lifecycle that must be governed separately from its anchor;
7. participate in at least one typed relationship connecting it to its anchor;
8. not hide a mandatory dependency, packaging, delivery, operations, or conformance responsibility.

An associated artifact has no PTSIP `classification` of its own and no component `roles` of its own.

It MUST NOT inherit its anchor's classification merely to make validation convenient.

File format, path, extension, declarative form, or lack of executable permission is insufficient to establish associated-artifact eligibility.

### PTSIP-RMAP-010 — Anchor and identity semantics

Every associated artifact has exactly one semantic **anchor component**.

The anchor identifies the coherent classified responsibility whose lifecycle explains why the associated artifact exists and changes. It does not make the associated artifact a component and does not copy classification onto it.

Component IDs and associated-artifact IDs MUST share one map-wide unambiguous endpoint identity namespace.

A component ID and associated-artifact ID MUST NOT collide.

A relationship endpoint MUST resolve to one declared component or one declared associated artifact.

The exact serialization of the anchor relationship is defined by the Responsibility Map v2 JSON Schema; the one-anchor semantic is mandatory.

### PTSIP-RMAP-011 — Promotion and reclassification boundary

An associated artifact MUST be promoted/re-evaluated as a PTSIP Component when it acquires independently governable lifecycle responsibility under the WU-01 mixed-lifecycle criteria.

Promotion/re-evaluation is required when, for example:

- the artifact becomes executable responsibility in its own architectural role;
- it has independent release or compatibility obligations;
- it can evolve/govern independently from its anchor in a lifecycle-relevant way;
- it carries independently owned Delivery or Operations responsibility;
- it carries cross-lifecycle authority that cannot coherently remain subordinate to one anchor.

If the responsibility becomes an independently governed, non-executable, non-owning, lifecycle-independent cross-lifecycle contract, it MUST be evaluated for `NEUTRAL_CONTRACT`.

If the responsibility becomes independently owned by Product, Development Tooling, Delivery, or Operations, it MUST be evaluated as a classified component under that lifecycle.

An associated artifact MAY have typed relationships to components other than its anchor only while it remains subordinate to the anchor and does not acquire independent cross-lifecycle authority.

## 6. Documentation and authority example

A project-owned SDK authority/documentation tree may be represented conceptually as:

```yaml
components:
  - id: turbo-sdk
    classification: DEVELOPMENT_TOOLING
    roles: [IMPLEMENTATION]
    purpose: reusable_sdk_implementation

associated_artifacts:
  - id: turbo-sdk-authority
    anchor: turbo-sdk
    include:
      - docs_4/SDK/**
    purpose: sdk_normative_authority_and_lifecycle_documentation

relationships:
  - from: turbo-sdk-authority
    to: turbo-sdk
    type: SPECIFIES

  - from: turbo-sdk-authority
    to: turbo-sdk
    type: GOVERNS
```

The associated artifact is not `DEVELOPMENT_TOOLING` implementation and is not forced into `NEUTRAL_CONTRACT`. Its subordinate lifecycle attachment is expressed through the anchor; its semantic authority is expressed through typed relationships.

## 7. Automation example

A repository may represent SDK implementation, verification automation, and release automation conceptually as:

```yaml
components:
  - id: turbo-sdk
    classification: DEVELOPMENT_TOOLING
    roles: [IMPLEMENTATION]

  - id: turbo-sdk-ci
    classification: DEVELOPMENT_TOOLING
    roles: [VERIFICATION, AUTOMATION]

  - id: turbo-sdk-release
    classification: DELIVERY
    roles: [AUTOMATION]

relationships:
  - from: turbo-sdk-ci
    to: turbo-sdk
    type: VERIFIES

  - from: turbo-sdk-release
    to: turbo-sdk
    type: BUILDS

  - from: turbo-sdk-release
    to: turbo-sdk
    type: PUBLISHES
```

The role vocabulary, relationship meanings, relationship direction, one-anchor semantics, and promotion rules in this document are normative.

## 8. Migration

### PTSIP-RMAP-012 — Legacy fields are evidence, not automatic typed-edge mappings

Tool `0.3.5` fields such as `consumers`, `analysis_inputs`, and untyped `component_dependency_policy` relations MAY be used as migration evidence.

They MUST NOT be blindly translated into canonical typed relationships.

In particular:

- `consumers` does not distinguish `IMPORTS`, `READS`, `LOADS`, `INVOKES`, or another consumption mechanism;
- `analysis_inputs` may support `READS`, `VERIFIES`, or another relation depending on stronger evidence;
- an allow/deny `from`/`to` dependency-policy entry declares policy, not necessarily an actual architecture edge.

Migration tooling SHOULD combine legacy declaration with observed/inferred repository evidence and produce typed relationship/associated-artifact proposals for project-owner confirmation.

If a confirmed legacy fact cannot be represented without semantic loss, migration MUST stop rather than dropping it or replacing it with a generic relationship.

## 9. Responsibility Map v2 schema obligations

The Responsibility Map v2 schema MUST support the frozen semantics in this document, including:

- zero-or-more canonical component roles;
- no composite-role extension mechanism that silently bypasses the canonical vocabulary;
- map-wide stable endpoint identity across components and associated artifacts;
- associated artifacts with exactly one anchor component;
- typed directed relationships whose endpoints resolve to declared identities;
- the canonical Tool `0.3.6` relationship vocabulary;
- clear separation between actual/intended relationships and dependency policy;
- no canonical `TOOLCHAIN` classification alias;
- lossless preservation of confirmed migration facts;
- explicit, template, and hybrid declaration modes with exact template identity where a template is selected.

## 10. Declaration authority and materialization

### PTSIP-RMAP-013 — Declaration source does not alter lifecycle ownership

Responsibility Map declaration source and PTSIP lifecycle classification are independent semantic axes.

`source_mode` identifies how the project declaration is sourced:

```text
explicit
template
hybrid
```

It MUST NOT be interpreted as a lifecycle classification, responsibility role, relationship type, VPMS Verification Purpose, evidence provenance value, or substitute for `classification`.

A component's canonical lifecycle ownership MUST have the same meaning regardless of whether that component declaration originated from an explicit profile, an explicitly selected template revision, or a hybrid project override.

### PTSIP-RMAP-014 — Exact template selection is project-owned architecture authority

Template selection MUST be an explicit project-owned architecture decision.

A template-backed declaration MUST identify an exact supported template identity using stable template ID plus immutable/versioned revision.

The Tool MUST NOT select or change the selected template solely from repository path, language, framework, manifest, package manager, workflow provider, candidate confidence, or other discovery evidence.

For `template` mode, the project owns the decision to adopt the exact selected template revision, and that immutable revision supplies the adopted declaration content.

For `hybrid` mode, the project owns both exact template selection and its explicit stable-ID replacement, extension, and removal decisions.

A later template revision MUST NOT silently change the Canonical Effective Responsibility Map of a project still bound to an earlier revision.

### PTSIP-RMAP-015 — Hybrid precedence is stable-ID whole-entity authority

Tool `0.3.6` hybrid materialization MUST apply project-owned stable-ID declarations over the selected immutable template declaration.

Precedence is:

```text
project replacement / extension / removal
    > selected immutable template declaration
```

For a template entity replaced by the same project-owned stable ID, the project declaration owns the entire resulting entity. Tool `0.3.6` MUST NOT perform implicit field-level inheritance that fragments declaration authority inside that entity.

A project-owned new stable ID is an extension. A project-owned removal is an explicit decision not to include the selected template entity in the effective map.

Materialization MAY expose derived review provenance such as `PROJECT_EXPLICIT`, `TEMPLATE`, `PROJECT_OVERRIDE`, `PROJECT_EXTENSION`, and `PROJECT_REMOVAL`, but that provenance MUST remain distinct from lifecycle classification and need not be persisted into the canonical Project Profile.

### PTSIP-RMAP-016 — Materialization is deterministic and non-authoritative

The materializer MUST resolve an already-declared `explicit`, `template`, or `hybrid` source into one **Canonical Effective Responsibility Map** without acquiring architecture authority.

The Canonical Effective Responsibility Map is the common downstream semantic representation of:

- components;
- associated artifacts;
- typed relationships;
- applicable component dependency policy; and
- project policies.

The original source declaration and exact template identity, where applicable, MUST remain distinguishable from the derived effective view.

The materializer MUST NOT:

- infer missing architecture;
- auto-select a template;
- change lifecycle classification to make validation pass;
- fabricate components, artifacts, roles, or relationships;
- silently repair or delete dangling relationships;
- silently cascade removals into additional project architecture changes;
- resolve project/template semantic conflicts using confidence or path heuristics;
- mutate the source profile merely to obtain a valid effective map.

If materialization produces an invalid effective endpoint, anchor, identity, selector, or other canonical Responsibility Map condition, the operation MUST fail closed and report the inconsistency for project-owner action.

Implementations MAY calculate a deterministic semantic digest of the Canonical Effective Responsibility Map for reproducibility, comparison, stale-state detection, or migration preview. Such a digest MUST NOT replace declaration provenance or become architecture authority.
