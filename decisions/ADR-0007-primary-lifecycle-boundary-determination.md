# ADR-0007 — Freeze Primary Lifecycle Boundary Determination

**Status:** Accepted  
**Decision:** Freeze deterministic boundary-determination rules for Tool `0.3.6` primary lifecycle ownership  
**Design lineage:** Tool `0.3.6` five-classification ontology + Responsibility Map v2 migration requirements

## Context

Tool `0.3.6` expands PTSIP classification from the Tool `0.3.5` three-classification model into five canonical primary lifecycle ownership classifications:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

The expansion solves the overloading of legacy `TOOLCHAIN`, but it creates a new requirement: a project, coding agent, migration analyzer, or validator needs a repeatable way to distinguish lifecycle ownership without falling back to path names, framework names, file types, or generic activity labels such as `test`, `build`, `deploy`, or `infra`.

The central problem is that the same artifact kind can legitimately belong to different lifecycles. Product-specific tests may be `PRODUCT`; reusable test infrastructure may be `DEVELOPMENT_TOOLING`; release acceptance automation may be `DELIVERY`. Likewise, Docker, Terraform, GitHub Actions, FastAPI, Cloudflare Workers, Python, Markdown, and YAML are technologies or forms, not ownership classifications.

A simple confidence score or majority-of-lines heuristic is insufficient because one legacy component may contain several materially different lifecycle responsibilities. In those cases, choosing the highest score would preserve the old classification-collapse problem under a new name.

## Decision

Primary lifecycle ownership is determined from the **governing lifecycle obligation** of a coherent responsibility.

The governing lifecycle obligation is the lifecycle-specific reason the responsibility must exist, change, remain compatible, execute, or be retired.

The determination procedure is ordered as follows:

```text
1. establish project-owned scope
2. identify a coherent responsibility boundary
3. collect lifecycle evidence
4. test NEUTRAL_CONTRACT qualification
5. determine the governing owning lifecycle
6. test for material mixed-lifecycle responsibility
7. keep one component, split, or remain unresolved
8. project owner confirms inferred/migrated architecture
```

No step may be replaced by a path-name, technology-name, executable-status, line-count, job-count, invocation-frequency, or confidence-score shortcut.

## Governing-lifecycle questions

For one candidate responsibility, analysis should ask:

```text
Why does this responsibility exist?
Which lifecycle obligation fails if it disappears?
What type of change normally requires it to change?
Who owns its compatibility consequences?
When is it normally invoked or enforced?
Does its obligation end when a release target reaches a destination?
Does its obligation continue after deployment as ongoing service management?
Can it evolve independently from another responsibility currently grouped with it?
```

These questions produce evidence and candidate ownership. They do not transfer project architecture authority from the maintainer to tooling.

## `PRODUCT` boundary

`PRODUCT` is selected when the governing obligation is Product existence or Product-owned quality, including runtime behavior, user-facing behavior, Product distribution content, runtime SDK responsibility, Product compatibility, or Product-specific verification responsibility.

Typical positive evidence includes:

- the responsibility is required for Product runtime or user-visible behavior;
- the responsibility defines Product-distributed contents or Product-owned compatibility;
- the responsibility changes because Product behavior, API, protocol use, or Product contract changes;
- a test/verification responsibility exists as part of the Product's own quality lifecycle rather than as reusable development infrastructure.

The following do not by themselves make a responsibility `PRODUCT`:

- being compiled or packaged;
- being located under `src/` or `tests/`;
- using FastAPI, Cloudflare Workers, Docker, or another runtime/platform technology;
- producing an artifact that is Product-owned when the producer itself belongs to another lifecycle.

A Product Artifact and the mechanism that builds, packages, publishes, or deploys it may therefore have different lifecycle classifications.

## `DEVELOPMENT_TOOLING` boundary

`DEVELOPMENT_TOOLING` is selected when the governing obligation is developer work before or outside release delivery: authoring, inspection, validation, transformation, migration, generation, static analysis, local/development build support, reusable test infrastructure, repository transformation, or development-environment support.

Typical positive evidence includes:

- the responsibility primarily exists to help developers create, inspect, transform, or verify project state;
- it remains useful as reusable verification/generation/migration infrastructure independently of one Product-specific quality obligation;
- it produces intermediate/development outputs rather than owning release-target transport to a destination;
- its compatibility consumers are developers, repository automation, or other development-time tools rather than Product runtime users.

A verifier may be `DEVELOPMENT_TOOLING` even when VPMS says the Verification Purpose is `PRODUCT`. Verification target/purpose does not determine implementation lifecycle ownership.

## Product tests versus Development Tooling tests

Tests are explicitly multi-classification artifacts.

A test or verification responsibility is a `PRODUCT` candidate when its own lifecycle is coupled to Product behavior/compatibility and it exists as Product-owned quality responsibility.

A test framework, reusable test SDK, shared harness, generic verification engine, or repository-wide reusable validation mechanism is a `DEVELOPMENT_TOOLING` candidate when its own lifecycle is development-support infrastructure.

The target being tested is insufficient to decide implementation ownership. Current consumer count is also insufficient: a reusable development tool does not become `PRODUCT` merely because only one Product currently consumes it.

## Development build versus Delivery build

The word `build` does not identify one lifecycle.

A build responsibility is normally `DEVELOPMENT_TOOLING` when it primarily supports coding, local compilation, inspection, testing, or creation of non-release intermediate outputs.

A build responsibility is normally `DELIVERY` when it materializes, assembles, signs, verifies for handoff, or otherwise prepares an authoritative release/distribution/deployment unit as part of carrying that target to a destination.

Examples:

```text
local compile/test helper
    -> DEVELOPMENT_TOOLING candidate

wheel/container build used to create the release unit
    -> DELIVERY candidate

Product files inside that wheel/container
    -> PRODUCT ownership may remain unchanged
```

Artifact ownership and producer ownership remain independent.

## `DELIVERY` boundary

`DELIVERY` is selected when the governing obligation is transition of a release target from prepared source/artifact state to a publication, promotion, distribution, or deployment destination.

Typical positive evidence includes:

- release artifact assembly intended for authoritative distribution;
- signing/attestation bound to release handoff;
- package or container publication;
- release promotion between channels/environments;
- deployment automation whose responsibility is to place or activate a release target at its destination;
- deployment configuration whose primary obligation exists for that handoff.

The defining boundary is the **delivery handoff**: the point at which a release target has reached/been accepted by its intended destination and ordinary ongoing operation begins.

A Product runtime implementation does not become `DELIVERY` merely because it is deployed. Conversely, deployment tooling does not become `PRODUCT` merely because it handles Product artifacts.

## `OPERATIONS` boundary

`OPERATIONS` is selected when the governing obligation is ongoing management of an already deployed system or environment after delivery handoff.

Typical positive evidence includes:

- continuous or recurring infrastructure reconciliation;
- production health/availability management;
- backup and recovery responsibility;
- incident-response or operational repair tooling;
- production maintenance;
- ongoing runtime infrastructure management, scaling, rotation, or state management.

The same technology may appear in Delivery and Operations. For example, Terraform may perform one-time/transition deployment in a `DELIVERY` responsibility or ongoing production-state reconciliation in an `OPERATIONS` responsibility.

Rollback is not universally classified by name. A release-channel rollback/promotion mechanism may be `DELIVERY`; an incident-recovery mechanism owned by ongoing production operations may be `OPERATIONS`.

## Delivery versus Operations handoff test

When Delivery and Operations are both plausible, use this boundary test:

```text
Does the primary obligation end when the selected release reaches/activates at its destination?
    yes -> DELIVERY candidate

Does the primary obligation persist because deployed state must remain healthy/recoverable/maintained?
    yes -> OPERATIONS candidate
```

If both are materially true and independently governable, the component should be split rather than classified by whichever phase has more steps or longer runtime.

## `NEUTRAL_CONTRACT` qualification

`NEUTRAL_CONTRACT` is not a default bucket for non-executable or shared files. A candidate must satisfy all of these semantic conditions:

1. the responsibility is non-executable in its architectural role;
2. it is non-owning with respect to Product, Development Tooling, Delivery, and Operations implementation responsibility;
3. its lifecycle/compatibility governance is meaningfully independent from those consuming lifecycles.

A schema, IDL, registry, protocol definition, test vector, or immutable manifest may qualify when these conditions hold.

The following are insufficient by themselves:

- Markdown/YAML/declarative format;
- no executable bit;
- multiple consumers;
- being described as a contract;
- living under `docs/`, `schemas/`, or `contracts/`.

Project-owned documentation or authority material that evolves with and governs one component's lifecycle is not neutral merely because it is non-executable. Such material may later be represented through Responsibility Map associated-artifact/typed-relationship semantics.

## Material mixed-lifecycle responsibility

A component has a material mixed-lifecycle problem when one classification cannot describe its lifecycle truth without hiding an independently governable responsibility.

Strong split evidence includes one or more of:

- different lifecycle triggers for change or execution;
- different release, compatibility, permission, secret, or environment obligations;
- different dependency environments that can evolve independently;
- different failure meanings or operational owners;
- one responsibility can change/ship/deploy/operate independently of another;
- one portion remains useful after another lifecycle responsibility is removed;
- migration evidence maps substantial portions of one legacy component to different lifecycle obligations.

Physical co-location alone does not require a split. A component may contain subordinate activities from another phase when those activities exist solely to complete one coherent primary lifecycle obligation and introduce no independent lifecycle governance.

The deciding question is not `does more than one activity occur?`; it is `does more than one independently governable lifecycle responsibility exist?`.

## Composite workflows

A workflow may contain test, build, package, deploy, and operational steps. The workflow filename or provider does not determine classification.

If every step is subordinate to one coherent lifecycle obligation, the workflow may remain one component under that primary lifecycle.

If the workflow combines independently governable development verification, release delivery, and ongoing operational responsibilities, PTSIP should propose a component/workflow split or explicit redesign. The Tool must not hide the conflict by assigning the whole workflow to the phase with the most jobs.

## Ambiguity and fail-closed behavior

Tooling may generate a candidate classification when evidence coherently supports one lifecycle. It may attach confidence for review, but confidence is not architecture authority.

When material evidence supports conflicting lifecycle ownership and a safe split cannot yet be represented or confirmed, the result remains unresolved for migration/adoption purposes. The Tool must not choose a winner merely to complete a profile.

The resolution order is:

```text
coherent single lifecycle
    -> propose that classification

material separable lifecycles
    -> propose component split

material but not safely resolvable
    -> explicit unresolved/clarification
```

## Consequences

1. Product-owned tests remain valid without forcing users into one testing style.
2. Reusable verification infrastructure can remain Development Tooling even when it protects Product behavior.
3. Build responsibility is no longer automatically Development Tooling; release-unit construction may belong to Delivery.
4. Delivery and Operations are separated by the handoff boundary rather than by technology names.
5. Neutral Contract remains a strict semantic classification rather than a non-executable catch-all.
6. Legacy `TOOLCHAIN` migration can propose Development Tooling, Delivery, Operations, Product, split, or unresolved states from the same deterministic boundary rules.
7. Candidate-discovery confidence can help review but cannot erase mixed-lifecycle architecture.
8. WU-02 and WU-03 may now define role/relationship/schema representations against a stable lifecycle ontology.

## Rejected alternatives

### Keep `TOOLCHAIN` as a broad compatibility bucket

Rejected. It would preserve the exact lifecycle collapse Tool `0.3.6` is intended to remove.

### Classify by directory or technology

Rejected. Repository layout and technology names are evidence context, not lifecycle authority.

### Put all tests in Development Tooling

Rejected. Product-owned verification is a legitimate Product lifecycle responsibility, and PTSIP must not impose one development/testing style.

### Put all build work in Development Tooling

Rejected. Build mechanisms that create authoritative release units as part of handoff have Delivery ownership distinct from local/development build support.

### Treat deployment and operations as one lifecycle

Rejected. Transition-to-destination and ongoing post-handoff service management have different change triggers, permissions, compatibility/availability obligations, and failure semantics.

### Use confidence scoring to pick one lifecycle

Rejected. Scores may rank proposals but cannot safely collapse independently governable responsibilities into one classification.

### Treat every multi-step workflow as multiple components

Rejected. Activity diversity alone is not lifecycle diversity; only independently governable lifecycle responsibility requires a split.