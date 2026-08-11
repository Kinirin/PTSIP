# PTSIP Terminology

This document is the canonical human-readable terminology registry for the currently active PTSIP `0.2.0-draft` normative baseline.

> **0.3.4-draft alignment note:** the terms in **Candidate 0.3.4-draft distributed-authority terminology** below are published design terminology from `spec-v0.3.4-draft`. They are not an active normative binding until a coherent Specification migration updates all affected normative, schema, registry, agent-contract, and embedded Tool assets at one immutable revision.

## PTSIP Component

The architectural unit to which PTSIP ownership is assigned. A component may be a package, module group, executable, generated artifact group, single build/release script, protocol bundle, or another project-defined unit whose purpose and lifecycle can be evaluated coherently.

A component boundary is coherent only when one classification, primary purpose, and lifecycle ownership statement can describe it without hiding contradictory ownership responsibilities.

## Product SDK

An SDK or SDK-like package whose primary lifecycle ownership belongs to the software product and whose code or artifacts may be consumed by, embedded in, or distributed with the product.

## Toolchain SDK

An SDK or SDK-like package whose primary lifecycle ownership belongs to software development activities such as validation, migration, generation, build, testing, release preparation, inspection, or repository automation.

## Product SDK Plane

The architectural ownership domain containing Product SDKs and their product-facing dependencies.

## Toolchain SDK Plane

The architectural ownership domain containing Toolchain SDKs and development-only dependencies.

## Neutral Contract Artifact

A non-owning declarative contract that may be consumed by both planes, such as schemas, IDLs, registries, immutable manifests, or conformance vectors. It is not a shared executable SDK.

Neutrality is determined by non-executable/non-owning contract semantics and lifecycle independence, not by a fixed number of currently observed consumers.

## Classification

The resolved architectural ownership of a PTSIP Component. The only PTSIP classifications are `PRODUCT`, `TOOLCHAIN`, and `NEUTRAL_CONTRACT`.

## Decision status

The state of an ownership decision during inspection or migration. Reference statuses include `RESOLVED`, `UNKNOWN`, `CONFLICT`, and `INCOMPLETE`. A decision status is not an architectural plane or classification.

## Consumer Repository

A repository being inspected, piloted, validated, or governed using PTSIP. PTSIP does not own the Consumer Repository's documentation hierarchy, tooling hierarchy, package layout, or other repository conventions.

## External PTSIP Tooling

A PTSIP implementation installed or executed outside the Consumer Repository's project-owned source tree for inspection, pilot analysis, validation, or reporting. External PTSIP Tooling is outside the Consumer Repository's Product/Toolchain classification scope unless the project intentionally vendors, embeds, packages, or takes lifecycle ownership of it.

## Consumer Repository Non-Intrusion

The property that PTSIP tooling can operate without requiring PTSIP-specific documentation, tooling, cache, report, or directory hierarchies inside the Consumer Repository, and without modifying repository content by default.

## Evidence snapshot

The observable Consumer Repository state to which one automated evidence set is bound. For Git repositories this normally includes an immutable commit/HEAD identity and sufficient working-tree observations to detect a mixed or changed snapshot during collection.

## Evidence node scope

The scope/type of a dependency or artifact evidence node, distinct from architectural classification. Reference scopes include `PROJECT_COMPONENT`, `EXTERNAL_DEPENDENCY`, `PLATFORM`, and `UNRESOLVED_TARGET`.

External/platform/unresolved node scope MUST NOT be represented as an additional PTSIP architectural classification.

## Observed evidence

Repository facts collected directly from files, manifests, dependency relationships, build/release definitions, artifacts, runtime behavior, or static analysis. Observed evidence is distinct from a project's declared intended ownership.

## Declared evidence

Evidence that a project, manifest, profile, or configuration explicitly declares a relationship, ownership intent, dependency, artifact, or policy. Declared evidence is a declaration, not proof that observed behavior agrees with it.

## Inferred evidence

A bounded conclusion derived from declared/observed evidence by deterministic logic, human review, or constrained agent reasoning and explicitly marked as inferred. Inferred evidence MUST NOT be presented as directly observed fact.

## Evidence provenance

The origin category attached to evidence. Canonical values are `DECLARED`, `OBSERVED`, and `INFERRED`.

## Dependency relationship type

The architectural/evidence relationship represented by an edge. Canonical relationship types are `IMPORTS`, `LINKS`, `LOADS`, `INVOKES`, `READS`, `GENERATES`, `PACKAGES`, `TESTS`, and `PUBLISHES`.

A relationship may have more than one type when distinct evidence supports more than one semantic relationship.

## Lifecycle phase

The lifecycle context in which a dependency/evidence relationship is relevant. Canonical phases are `RUNTIME`, `BUILD`, `TEST`, `RELEASE`, and `INSPECTION`. A relationship may apply to multiple phases. Unknown phase remains unresolved rather than guessed.

## Product Artifact

A deployable, distributable, installable, loadable, or otherwise Product-owned output whose lifecycle responsibility belongs to the Product plane, such as an installer, application bundle, plugin archive, service bundle, container, or package.

Product Artifact classification is determined independently from the classification of the component that produced it.

## Artifact producer

The component or process that generates, assembles, packages, or publishes an artifact. A Toolchain component may be the producer of a Product Artifact without becoming Product-owned.

## Artifact derivation

Evidence describing how an artifact was generated, packaged, or published from source components or other artifacts. Reference relations include `GENERATES`, `PACKAGES`, and `PUBLISHES`.

## Specification Binding

A machine-readable association between a Consumer Repository's PTSIP declaration and the canonical PTSIP specification source and version/family used to interpret that declaration. For a mutable draft family, an immutable revision should additionally be recorded for reproducibility and is required for Enforced Conformance.

## Draft family

A version label such as `0.2.0-draft` or proposed `0.3.4-draft` that identifies an experimental specification family whose exact normative snapshot is identified by immutable repository revision until a stable specification release is declared.

A GitHub Specification Release or draft-family label does not by itself activate a new Tool binding. Activation requires an explicitly recorded coherent normative snapshot and, for a Tool, an explicit binding to that snapshot.

## Candidate 0.3.4-draft distributed-authority terminology

The following terms describe the published `spec-v0.3.4-draft` design. They are candidate normative terminology until the coherent `0.3.4-draft` migration is completed.

### Explicit Project Adoption

The controlled process by which a Consumer Repository establishes or extends a durable Project Profile from discovered component scope plus explicit project-owner architecture facts. Candidate discovery identifies **what scope needs a declaration**; it does not determine `PRODUCT`, `TOOLCHAIN`, or `NEUTRAL_CONTRACT` ownership by itself.

### Decision Authority

The authority used to coordinate unresolved and resolved architecture decisions for a defined coordination domain.

A Decision Authority determines which explicit architecture answer won for a coordinated decision. It does **not** replace the Project Profile, and it is not observed conformance truth.

### Coordination domain

The scope within which one distributed architecture-decision identity has one authoritative winner. Repository identity is normally part of the domain for repository-scoped coordination.

### Distributed decision identity

A deterministic identity for the same architectural component scope across participating clones and environments. It is derived from stable coordination-domain identity plus normalized architecture/component scope rather than clone-local clarification IDs, temporary missing-field state, local database IDs, or incidental process identifiers.

### Authority revision

An ordered state token that lets a participant distinguish the authority state it read from a later authority state and perform a safe conditional mutation. Examples include an immutable Git commit/ref revision, transaction version, ETag/generation, or consensus-log index.

### First-valid-resolution-wins

The distributed coordination rule that the first valid accepted resolution for one distributed decision identity becomes the winner. A later contradictory resolution cannot silently replace that winner. Implementations use compare-and-swap, transaction, consensus, or equivalent conditional mutation to reject stale overwrites.

### Authority freshness

The property that an architecture-sensitive operation using distributed coordination accounts for the relevant current Decision Authority state before returning a result that assumes a local declaration is sufficient for that coordinated scope.

A complete local Project Profile declaration is not, by itself, proof that no newer distributed winner exists.

### Read-only authority observation

A non-mutating check for existing authority state. Absence checking should not fabricate a pending decision, authority branch, database row, or other decision history merely to prove that no prior distributed decision exists.

### Authority/Profile reconciliation

The deterministic comparison and, where safe, convergence process between a selected local Project Profile declaration and a relevant Decision Authority state.

Candidate `0.3.4-draft` semantics distinguish at least:

- local declaration absent + no authority decision;
- local declaration absent + resolved winner;
- local declaration present + no authority decision;
- local declaration semantically equivalent + resolved equivalent winner;
- local declaration conflicting + resolved different winner; and
- local repository/profile mutation during reconciliation.

### Semantic equivalence

Equivalence of architecture meaning independent of incidental serialization differences such as YAML formatting, key order, insignificant whitespace, or Tool-generated versus manually formatted equivalent content.

### Authority/Profile conflict

A state where a resolved distributed winner and an existing local Project Profile declaration express different architecture meaning for the same coordinated scope.

The candidate `0.3.4-draft` contract requires explicit conflict reporting and forbids silent automatic overwrite or reclassification of the existing Project Profile.

### Global decision state

The Decision Authority state describing whether a coordinated decision is unresolved or resolved, for example `PENDING` or `RESOLVED`. Global resolution identifies the winning architecture answer; it does not prove that every clone has written that answer into its Project Profile.

### Local projection state

Clone-, worktree-, or revision-local state describing whether an authoritative winner has been reconciled/applied locally, is already semantically consistent, is stale, or failed to apply. Local projection state cannot alter which global architecture answer won.

### Action-time synchronization

Synchronization with the selected Decision Authority at an architecture-sensitive operation boundary rather than through continuous background polling. Correctness does not require every clone to continuously poll or immediately receive another clone's `ptsip.yaml` commit.

### Fail-closed distributed coordination

The property that a selected distributed authority does not silently degrade to an isolated local authority when required freshness or mutation cannot be established. Authentication, permission, network, malformed authority state, unsafe conditional mutation, or equivalent coordination failure stops the affected coordinated architecture operation rather than creating a second independent winner.

## Boundary

The set of dependency, packaging, build, and lifecycle constraints separating Product and Toolchain planes.

## Isolation

The property that one plane can evolve, resolve dependencies, and be governed without accidentally acquiring the other plane's runtime or release obligations.

## Lifecycle ownership

Responsibility for versioning, compatibility, release, deprecation, and consumer impact of a component.

## Packaging ownership

Responsibility for deciding whether a component is included in a Product Artifact or other distributable output.

## Build environment

The declared dependency and execution context needed to resolve and build a plane. PTSIP does not require a particular environment technology.

## Cross-boundary dependency

A dependency edge from a component in one PTSIP plane to a component in another plane.

## Blocking evidence gap

An evidence gap that can conceal whether an applicable PTSIP `MUST` or `MUST NOT` rule is satisfied. A blocking evidence gap prevents a conformant outcome unless a definite violation already establishes `NON_CONFORMANT`.

## Non-blocking evidence gap

A reported evidence gap that cannot materially affect the result of the applicable mandatory PTSIP rules being evaluated.

## PTSIP mandatory-rule waiver

PTSIP does not define a waiver that authorizes violation of a PTSIP `MUST`/`MUST NOT` rule. Project governance may record debt or migration approval, but a confirmed violation remains `NON_CONFORMANT` until remediated and reevaluated. `PTSIP-EXC-001` is a historical rule from earlier immutable draft snapshots and is retired/superseded in the new snapshot.

## PTSIP Project Profile

A machine-readable declaration of intended component/path ownership and policies for one repository or project. A reference profile uses exactly one ownership-declaration form: boundary roots for uniform ownership or component declarations for precise nested ownership. The profile is declaration, not proof of conformance. It is required for Enforced Conformance but is not required merely to run read-only inspection or pilot tooling.

Under the proposed `0.3.4-draft` model, the Project Profile remains the durable project-owned architecture declaration even when a separate Decision Authority is used to coordinate unresolved architecture decisions.

## Project component dependency policy

An optional repository-specific policy declaring stricter component-to-component dependency constraints, including constraints within one PTSIP plane. Such policy may strengthen but MUST NOT weaken universal PTSIP requirements.

## Profile Validation

The operation that checks whether a PTSIP Project Profile or equivalent declaration is structurally and semantically valid. Profile Validation does not determine whether the repository conforms to PTSIP.

## Conformance Evaluation

The operation that combines applicable declarations with observed dependency, artifact, lifecycle, coverage, and other evidence to evaluate PTSIP rules.

Decision Authority state is architecture-decision coordination state; it does not replace observed evidence or deterministic conformance-rule evaluation.

## Conformance outcome

The result of a completed PTSIP Conformance Evaluation. Canonical outcomes are `CONFORMANT`, `NON_CONFORMANT`, and `INCOMPLETE`.

`NOT_EVALUATED` may be used as a tooling state but is not a conformance outcome.

## PTSIP diagnostic

A machine-readable finding produced by PTSIP evaluation. A diagnostic has its own unique instance identity and references the stable PTSIP rule ID it concerns. One rule may produce multiple diagnostic instances.

## Conformance

A claim that a project satisfies a defined PTSIP conformance level and outcome according to `PTSIP-CONFORMANCE.md`.
