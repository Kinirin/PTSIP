# ADR-0008 — Freeze Responsibility Roles, Typed Relationships, and Associated Artifacts

**Status:** Accepted  
**Decision:** Freeze Tool `0.3.6` Responsibility Map v2 semantics for component roles, typed responsibility relationships, and associated artifacts  
**Design lineage:** Tool `0.3.6` five-lifecycle ontology + WU-01 governing-lifecycle boundary rules + external project feedback on Toolchain automation/documentation representation

## Context

Tool `0.3.6` already separates PTSIP `classification` from VPMS Verification Purpose and defines classification as primary lifecycle ownership:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

That solves lifecycle ownership, but lifecycle classification alone cannot preserve all repository architecture semantics.

Real repositories commonly contain several components with the same lifecycle classification but materially different responsibilities. For example:

```text
tools/turbo_sdk/**
    DEVELOPMENT_TOOLING

architecture verification workflow
    DEVELOPMENT_TOOLING

release/publish workflow
    DELIVERY
```

The SDK implementation, verification automation, and release automation must not be collapsed merely because two happen to share one lifecycle. Likewise, project-owned documentation or authority material may evolve with and govern one component without being executable implementation and without satisfying `NEUTRAL_CONTRACT` lifecycle independence.

The Tool `0.3.5` Project Profile cannot preserve this distinction losslessly. Its component declaration has `purpose`, metadata such as `consumers`/`analysis_inputs`, and an optional component dependency policy whose relation objects contain only `from` and `to`. The profile has no canonical typed component relationship and no model for non-component support artifacts.

A naive extension would create new lifecycle classifications such as `CI`, `BUILD`, `RELEASE`, `DOCUMENTATION`, or `GOVERNANCE`, or create composite role labels such as `BUILD_RELEASE_AUTOMATION`. Both would repeat the same semantic collapse under a larger vocabulary.

## Decision overview

Responsibility Map v2 keeps four semantic axes separate:

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

Associated artifacts form a separate endpoint kind for project-owned support surfaces that are subordinate to one classified component and do not have an independently governable lifecycle responsibility.

## Component role model

Component roles are optional, multi-valued, and closed by the Specification/schema vocabulary.

The Tool `0.3.6` canonical role vocabulary is:

```text
IMPLEMENTATION
VERIFICATION
AUTOMATION
CONFIGURATION
DOCUMENTATION
GOVERNANCE
```

### `IMPLEMENTATION`

The component contains or provides the implementation that directly realizes its declared responsibility, such as an application, SDK/library implementation, runtime service, developer tool implementation, or operational implementation.

### `VERIFICATION`

The component's own responsibility includes checking, validating, testing, auditing, or establishing correctness/conformance/acceptance of another responsibility or artifact.

This role does not determine VPMS Verification Purpose and does not imply that the component is `DEVELOPMENT_TOOLING`.

### `AUTOMATION`

The component automates or orchestrates repeatable lifecycle work. This may occur under `DEVELOPMENT_TOOLING`, `DELIVERY`, `OPERATIONS`, or another classification when the governing lifecycle obligation supports it.

### `CONFIGURATION`

The component's independently governable responsibility is primarily configuration/state declaration rather than executable implementation. A file being YAML/TOML/JSON or declarative does not automatically create this role.

### `DOCUMENTATION`

The component's independently governable responsibility is maintaining descriptive/explanatory documentation. Subordinate documentation that has no independent lifecycle should normally remain an associated artifact instead of being promoted merely to obtain this role.

### `GOVERNANCE`

The component's independently governable responsibility is project authority, policy, lifecycle decision control, or architecture governance. Subordinate authority material that exists only with one anchor component may remain an associated artifact.

## Role cardinality and anti-proliferation rule

A component MAY have zero, one, or multiple canonical roles.

Multiple responsibilities are represented compositionally:

```text
verification workflow
roles:
  - VERIFICATION
  - AUTOMATION
```

rather than inventing:

```text
VERIFICATION_AUTOMATION
```

Likewise, Tool `0.3.6` does not standardize role tokens such as:

```text
SDK_IMPLEMENTATION
BUILD_RELEASE_AUTOMATION
PRODUCTION_VERIFICATION_AUTOMATION
```

Those distinctions belong in `purpose`, lifecycle classification, and typed relationships.

A role MUST NOT determine classification by itself. A role also MUST NOT automatically create a relationship. For example, `VERIFICATION` does not identify what is verified; that requires an explicit/proposed `VERIFIES` relationship.

## Typed relationship model

A Responsibility Map relationship is a project-declared, directed semantic edge.

Direction is always interpreted as:

```text
source --TYPE--> target
```

where the source depends on, acts on, derives, verifies, delivers, operates, documents, specifies, or governs the target according to the relationship type.

A relationship endpoint is identified by stable Responsibility Map identity. Tool `0.3.6` relationship endpoints may be:

- a classified component; or
- an associated artifact.

Component IDs and associated-artifact IDs must therefore share one unambiguous map-wide identity namespace.

## Canonical Responsibility Map relationship vocabulary

The Tool `0.3.6` project-declared relationship vocabulary is:

### Dependency/use relationships

- `IMPORTS` — source imports target implementation/module/package semantics.
- `LINKS` — source links target as build/link implementation dependency.
- `LOADS` — source dynamically loads target implementation/artifact.
- `INVOKES` — source executes or triggers target responsibility/process.
- `READS` — source consumes target as data/configuration/contract/input rather than executable implementation reuse.

### Derivation/build/delivery relationships

- `GENERATES` — source derives generated source/data/metadata/configuration/artifact from or for target.
- `BUILDS` — source materializes/assembles a build output or authoritative build unit for target.
- `PACKAGES` — source includes/assembles target content into a package/distribution unit.
- `PUBLISHES` — source publishes/distributes a release target associated with target responsibility.
- `DEPLOYS` — source places or activates a release target associated with target at a destination environment.

### Verification/operations relationships

- `VERIFIES` — source checks or establishes correctness, conformance, acceptance, or policy satisfaction of target.
- `MANAGES` — source performs ongoing post-handoff operational management of target/deployed state.

### Documentation/authority relationships

- `DOCUMENTS` — source provides descriptive/explanatory information about target.
- `SPECIFIES` — source defines normative technical semantics, required contract meaning, or authoritative interface/behavior expectations for target.
- `GOVERNS` — source carries project authority, lifecycle policy, decision control, or governance rules over target.

## Relationship normalization decisions

`PRODUCES` is not introduced as a canonical relationship because it is too broad. Tool `0.3.6` uses the more specific `GENERATES`, `BUILDS`, `PACKAGES`, or `PUBLISHES` semantics where applicable.

`SUPPORTS`, `USES`, and `DEPENDS_ON` are not introduced as generic escape-hatch relationships because they hide materially different dependency/authority semantics.

The legacy evidence relationship `TESTS` remains valid in the evidence vocabulary. Responsibility Map v2 uses `VERIFIES` for the broader project-declared architecture relation because verification may be established by tests, static analysis, policy checks, schema validation, compatibility auditing, or other mechanisms. An observed `TESTS` edge may support a `VERIFIES` proposal but does not automatically create that declaration.

`BUILDS` is distinct from `GENERATES` and `PACKAGES`:

- `GENERATES` describes derivation/generation;
- `BUILDS` describes materialization/assembly of a build unit;
- `PACKAGES` describes inclusion/assembly into a package/distribution boundary.

One automation may declare several relationships when those semantics are all materially true. The model does not encode them into a composite role name.

## Declared relationship versus evidence edge

Project-declared Responsibility Map relationships and evidence edges are related but distinct semantic namespaces.

```text
repository observation
    -> evidence edge
    -> may support/contradict a relationship proposal

project-owner confirmation/declaration
    -> Responsibility Map typed relationship
```

An `OBSERVED` or `INFERRED` evidence edge MUST NOT silently become project-owned architecture declaration merely because the relationship token has the same spelling.

Conversely, a declared relationship does not prove the observed repository actually behaves that way. Validation/conformance may report contradictions.

## Relationship versus dependency policy

A typed relationship declares architecture semantics. It does not grant permission.

`component_dependency_policy` or its Responsibility Map v2 successor remains a policy/constraint surface, and universal PTSIP `MUST`/`MUST NOT` rules remain controlling.

For example:

```text
product --IMPORTS--> delivery-helper
```

may accurately declare/observe architecture and still be prohibited by Product runtime isolation rules. The presence of an explicit typed edge does not waive the violation.

Likewise, an allow-policy entry does not create an architectural relationship that is not otherwise declared/observed.

## Associated artifact model

An **Associated Artifact** is a project-owned, non-component support surface that is subordinate to exactly one classified **anchor component**.

Its purpose is to preserve meaningful documentation, specification, governance, configuration, or similar support artifacts without falsely declaring them as executable implementation, forcing them into `NEUTRAL_CONTRACT`, or creating a separate classified component solely to express an association.

An associated artifact:

- has stable map identity;
- has project-owned selectors/scope and purpose;
- has exactly one anchor component;
- has no PTSIP `classification` of its own;
- has no component `roles` of its own;
- MUST participate in at least one typed relationship that connects it to its anchor component;
- MAY participate in additional typed relationships when doing so does not create independent lifecycle ownership.

The exact YAML/JSON nesting (`associated_artifacts` under a component versus a map-level collection with `anchor`) is deferred to WU-03 schema design. The one-anchor semantic is frozen by this ADR.

## Why associated artifacts do not inherit classification

An associated artifact is not a component and therefore does not receive a classification merely by inheritance.

For example:

```text
turbo-sdk
classification: DEVELOPMENT_TOOLING
roles: [IMPLEMENTATION]

sdk-authority-docs
associated artifact anchored to turbo-sdk
sdk-authority-docs --SPECIFIES--> turbo-sdk
sdk-authority-docs --GOVERNS--> turbo-sdk
```

The documentation/authority material is not declared as `DEVELOPMENT_TOOLING` implementation, but it is also not misclassified as `NEUTRAL_CONTRACT`. Its lifecycle subordination is represented by its anchor and its semantics by typed relationships.

## Associated artifact eligibility

A candidate may remain an associated artifact only while all of these are true:

1. it is non-executable in its architectural role;
2. it has no independently governable primary lifecycle responsibility;
3. it has no independent release/compatibility lifecycle that must be governed separately from the anchor;
4. one anchor component can coherently explain why the artifact exists and changes;
5. representing it as associated does not hide a mandatory dependency, packaging, delivery, operations, or conformance responsibility.

A file being Markdown, YAML, JSON, declarative, or non-executable is insufficient by itself.

## Promotion rule

An associated artifact MUST be promoted/re-evaluated as a classified component when it acquires an independently governable responsibility under the WU-01 split criteria, including executable responsibility or independent lifecycle/release/compatibility obligations.

If the artifact becomes an independently governed cross-lifecycle non-executable contract, it MUST be evaluated for `NEUTRAL_CONTRACT` instead of remaining associated merely to avoid classification.

If it becomes an independently owned Product, Development Tooling, Delivery, or Operations responsibility, it MUST be promoted to a component with the corresponding classification proposal/decision.

## Cross-component associated-artifact use

An associated artifact may document/read/specify another component in addition to its anchor only when the artifact remains subordinate to its single anchor and does not acquire independent cross-lifecycle authority.

If one artifact normatively specifies or governs several independently owned lifecycles in its own right, its lifecycle independence/ownership must be re-evaluated. It may require promotion to `NEUTRAL_CONTRACT` or to another classified governance/documentation component.

This prevents `associated_artifact` from becoming a hiding place for shared authority.

## Stable identity and endpoint rules

Tool `0.3.6` Responsibility Map v2 must provide stable unique identity for both components and associated artifacts.

The map must reject ambiguous endpoint resolution. A component ID and associated-artifact ID must not collide.

Relationships must reference declared endpoints. A migration tool must not silently fabricate an endpoint merely to preserve an unresolvable edge.

Self-relationships that carry no distinct architectural meaning should be rejected by the schema/tooling rather than used as generic annotations.

## Example: SDK verification, release, and authority documentation

Conceptual representation:

```yaml
components:
  - id: turbo-sdk
    classification: DEVELOPMENT_TOOLING
    roles: [IMPLEMENTATION]
    purpose: reusable_sdk_implementation

  - id: turbo-sdk-ci
    classification: DEVELOPMENT_TOOLING
    roles: [VERIFICATION, AUTOMATION]
    purpose: architecture_and_sdk_verification

  - id: turbo-sdk-release
    classification: DELIVERY
    roles: [AUTOMATION]
    purpose: sdk_release_build_and_publication

associated_artifacts:
  - id: turbo-sdk-authority
    anchor: turbo-sdk
    include:
      - docs_4/SDK/**
    purpose: sdk_normative_authority_and_lifecycle_documentation

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

  - from: turbo-sdk-authority
    to: turbo-sdk
    type: SPECIFIES

  - from: turbo-sdk-authority
    to: turbo-sdk
    type: GOVERNS
```

The exact property names are illustrative until WU-03 freezes the schema. The lifecycle/role/relationship/anchor semantics are not illustrative; they are the accepted WU-02 design.

## Migration implications

Tool `0.3.5` fields such as `consumers`, `analysis_inputs`, and untyped `component_dependency_policy` edges contain useful but incomplete semantics.

Migration MUST NOT guess a canonical typed relationship solely from one of those legacy fields.

Examples:

- `consumers: [x]` does not reveal whether `x` `IMPORTS`, `READS`, `LOADS`, or otherwise consumes the component;
- `analysis_inputs` may support `READS`, `VERIFIES`, or another relationship proposal depending on evidence;
- a legacy `from`/`to` dependency-policy entry describes allowed/denied policy, not necessarily an actual architecture edge.

Migration may use those fields as declared evidence and combine them with stronger repository evidence, then present typed relationship proposals for project-owner confirmation.

## WU-03 schema requirements created by this decision

Responsibility Map v2 schema work must be able to represent at least:

- zero-or-more canonical roles per component, without composite-role proliferation;
- map-wide stable unique identity across components and associated artifacts;
- associated artifacts with exactly one anchor component;
- typed directed relationships whose endpoints resolve to declared components/artifacts;
- the canonical relationship vocabulary frozen here;
- separation of declared relationships from dependency-policy permissions;
- no canonical `TOOLCHAIN` classification alias;
- migration without silently losing `purpose`, selectors, ownership facts, associated artifacts, or relationships.

WU-03 decides exact JSON Schema layout, required/optional field shape, template/hybrid override representation, and compatibility parsing boundaries. It must not reopen these WU-02 semantic decisions merely for serialization convenience.

## Consequences

1. Components in the same lifecycle can preserve materially different roles without expanding classification.
2. Verification, build, publication, deployment, operations, documentation, specification, and governance relations become machine-readable instead of being compressed into `purpose` strings or untyped `from`/`to` edges.
3. Project-owned SDK documentation/authority can remain lifecycle-subordinate support material without being falsely treated as Development Tooling implementation or Neutral Contract.
4. A single component can express several coarse roles without combinatorial role tokens.
5. A single automation can express several relationships without a composite `BUILD_RELEASE_AUTOMATION` role.
6. Migration automation must do more semantic work because legacy `consumers`/`analysis_inputs`/policy edges cannot be blindly translated.
7. WU-03 has a constrained schema target rather than an open-ended ontology design task.

## Rejected alternatives

### Add more lifecycle classifications for CI/build/release/docs/governance

Rejected. These are roles/relationships/support-surface semantics, not primary lifecycle ownership.

### Use one singular `role` value

Rejected. Real components often combine verification and automation or documentation and governance. A singular value would either lose information or force composite taxonomy growth.

### Standardize composite role names

Rejected. Tokens such as `BUILD_RELEASE_AUTOMATION` produce combinatorial growth and duplicate information better represented by lifecycle + roles + typed relationships.

### Infer relationships from roles

Rejected. `VERIFICATION` does not identify its target and `AUTOMATION` does not identify what is built/published/deployed. The edge must be explicit or a separately reviewed proposal.

### Treat all documentation/authority as `NEUTRAL_CONTRACT`

Rejected. Neutral Contract requires independent lifecycle/non-owning semantics. Documentation or authority coupled to one component may be subordinate associated material instead.

### Make associated artifacts inherit their anchor classification

Rejected. That would recreate the original feedback problem by making support documentation appear to be lifecycle implementation. Associated artifacts are intentionally unclassified non-component endpoints.

### Allow associated artifacts without a single anchor

Rejected. Without one coherent anchor, the model becomes an unclassified shared bucket and can hide independently governed cross-lifecycle authority.

### Add generic `PRODUCES`, `SUPPORTS`, or `USES` relationships

Rejected for Tool `0.3.6`. They are too broad for lossless migration and make conformance/agent reasoning less precise than `GENERATES`, `BUILDS`, `PACKAGES`, `PUBLISHES`, and the specific dependency/authority relationships.

### Merge evidence edges and project-declared relationships

Rejected. Observed/inferred repository behavior and project-owned architecture declaration have different authority/provenance semantics even when they use related vocabulary.
