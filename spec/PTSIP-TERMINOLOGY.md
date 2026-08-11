# PTSIP Terminology

This document is the canonical human-readable terminology registry for PTSIP `0.3.4-draft`. The exact draft snapshot is identified by immutable Git revision.

## PTSIP Component

The coherent architectural unit to which PTSIP ownership is assigned. A component may be a package, module group, executable, generated artifact group, script, protocol bundle, or another project-defined unit whose purpose and lifecycle can be evaluated coherently.

## Product SDK

A Product-owned SDK or SDK-like component whose lifecycle responsibility belongs to the Product and whose code/artifacts may be consumed by, embedded in, or distributed with the Product.

## Toolchain SDK

A development-tooling-owned SDK or SDK-like component used for validation, migration, generation, build, test, release preparation, inspection, repository automation, or related development activities.

## Product SDK Plane

The architectural ownership domain containing Product-owned components and Product-facing dependencies.

## Toolchain SDK Plane

The architectural ownership domain containing development-tooling-owned components and Toolchain-only dependencies.

## Neutral Contract Artifact

A deliberately non-executable, non-owning declarative contract that may be consumed by both planes without collapsing Product and Toolchain executable ownership. Neutrality depends on contract semantics and lifecycle independence, not current consumer count.

## Classification

The resolved PTSIP architectural ownership of a component. The only classifications are `PRODUCT`, `TOOLCHAIN`, and `NEUTRAL_CONTRACT`.

## Decision status

The state of an architecture/ownership decision during inspection or migration. Examples include `RESOLVED`, `UNKNOWN`, `CONFLICT`, and `INCOMPLETE`. A decision status is not a PTSIP classification.

## Consumer Repository

A repository being inspected, adopted, validated, coordinated, or governed using PTSIP. PTSIP does not own the repository's directory or documentation conventions.

## External PTSIP Tooling

PTSIP implementation operated outside the Consumer Repository project-owned source tree. It is outside repository Product/Toolchain classification scope unless intentionally vendored or adopted by the project.

## Consumer Repository Non-Intrusion

The property that external PTSIP tooling can operate without requiring PTSIP-specific repository directories or default repository mutation. Tool-owned operational state should remain outside the Consumer Repository unless the user explicitly chooses otherwise.

## PTSIP Project Profile

A project-owned machine-readable declaration of intended architecture ownership and policy. It is durable declaration state, not observed conformance truth.

The reference profile uses either boundary-root shorthand or component declarations, not both simultaneously.

## Project Profile fact

A durable structured architecture fact represented for a component. `0.3.4-draft` recognizes the explicit adoption fact set:

- `classification`;
- `purpose`;
- `shipped`;
- `runtime_required`;
- `lifecycle_owner`;
- `executable`.

## Runtime required

The boolean architecture fact `runtime_required`, indicating whether Product runtime operation requires the component. A Toolchain component cannot be Product-runtime-required.

## Lifecycle ownership

Responsibility for versioning, compatibility, release, deprecation, and consumer impact of a component.

Canonical `lifecycle_owner` values are:

- `PRODUCT`;
- `DEVELOPMENT_TOOLING`;
- `INDEPENDENT`.

`lifecycle_owner` is distinct from optional project metadata such as `release_owner` and `compatibility_owner`.

## Product Artifact

A Product-owned deployable, distributable, installable, loadable, or otherwise release-relevant output. Artifact ownership is independent from producer ownership.

## Artifact producer

The component/process that generates, assembles, packages, or publishes an artifact. A Toolchain producer may create a Product Artifact.

## Artifact derivation

Evidence describing how an artifact was generated, packaged, or published from source components or other artifacts.

## Specification Binding

The association between a PTSIP declaration/Tool and the canonical Specification source, draft family/version, and immutable revision used to interpret it.

## Draft family

A mutable family label such as `0.3.4-draft`. The label names the evolving family; an immutable Git revision identifies an exact normative snapshot.

## Evidence snapshot

The observable Consumer Repository state to which one automated evidence set is bound. For Git repositories this normally includes immutable revision identity and enough tracked-state information to detect instability during collection.

## Evidence node scope

Dependency/artifact graph scope distinct from PTSIP classification. Reference scopes include `PROJECT_COMPONENT`, `EXTERNAL_DEPENDENCY`, `PLATFORM`, and `UNRESOLVED_TARGET`.

## Observed evidence

Repository/artifact/runtime/static-analysis facts collected directly from observable state.

## Declared evidence

Facts explicitly declared by a profile, manifest, or configuration. Declaration is not proof that observed behavior agrees.

## Inferred evidence

A bounded conclusion derived from evidence by deterministic logic, human review, or constrained agent reasoning and explicitly marked as inferred.

## Evidence provenance

The origin category attached to evidence. Canonical values are `DECLARED`, `OBSERVED`, and `INFERRED`.

## Dependency relationship type

The semantic relationship represented by an evidence edge. Canonical values are `IMPORTS`, `LINKS`, `LOADS`, `INVOKES`, `READS`, `GENERATES`, `PACKAGES`, `TESTS`, and `PUBLISHES`.

## Lifecycle phase

The lifecycle context of a dependency/evidence relationship. Canonical phases are `RUNTIME`, `BUILD`, `TEST`, `RELEASE`, and `INSPECTION`. Unknown phase remains unresolved rather than guessed.

## Boundary

The dependency, packaging, build, and lifecycle constraints separating Product and Toolchain ownership.

## Isolation

The property that one plane can evolve, resolve dependencies, and be governed without accidentally acquiring the other plane's runtime or release obligations.

## Packaging ownership

Responsibility for deciding whether a component belongs in a Product Artifact or other distributable output.

## Build environment

The dependency and execution context needed to resolve/build a plane. PTSIP does not require a particular environment technology.

## Cross-boundary dependency

A dependency relationship from a component in one PTSIP plane to a component in another plane.

## Blocking evidence gap

An evidence gap capable of concealing whether an applicable PTSIP `MUST`/`MUST NOT` is satisfied. It prevents `CONFORMANT` unless a definite violation already establishes `NON_CONFORMANT`.

## Non-blocking evidence gap

An evidence gap that cannot materially change the applicable mandatory-rule result.

## Project component dependency policy

Optional repository-specific dependency constraints that may strengthen but never weaken universal PTSIP requirements.

## Profile Validation

Validation of Project Profile structure/semantics. It does not determine repository conformance.

## Conformance Evaluation

Deterministic evaluation combining declaration with observed dependency, artifact, lifecycle, snapshot, and coverage evidence against Consumer Repository PTSIP rules.

## Conformance outcome

The result of a completed Consumer Repository Conformance Evaluation. Canonical outcomes are `CONFORMANT`, `NON_CONFORMANT`, and `INCOMPLETE`. `NOT_EVALUATED` may be a Tool execution state but is not a conformance outcome.

## PTSIP diagnostic

A machine-readable finding instance that references a stable PTSIP rule ID and supporting evidence. Diagnostic instance identity and rule identity are different.

## Decision Authority

The authority used to coordinate unresolved/resolved explicit architecture decisions for a defined coordination domain.

A Decision Authority records **which explicit architecture answer won**. It does not replace the Project Profile and does not prove conformance.

## Coordination domain

The scope within which one distributed architecture decision identity has one authoritative winner. Repository identity is normally part of repository-scoped coordination.

## Distributed decision identity

A deterministic identity for one coordinated architecture decision derived from stable coordination-domain identity plus normalized architecture/component scope. It must not depend solely on clone-local temporary IDs or missing-field state.

## Authority revision

An ordered state token that lets participants distinguish one authority state from a later state and perform safe conditional mutation. Examples include Git commit/ref revision, transaction version, ETag/generation, or consensus log index.

## First-valid-resolution-wins

The distributed coordination rule that the first valid accepted resolution for one decision identity is the winner and later contradictory answers cannot silently replace it.

## Authority freshness

The requirement that a distributed coordination-sensitive operation account for relevant current authority state before relying on local Project Profile state as authoritative for the coordinated scope.

A complete local declaration does not by itself prove freshness.

## Read-only authority observation

A non-mutating lookup used to determine whether relevant authority state exists. Absence checking should not fabricate a pending decision or create history merely to prove absence.

## Authority/Profile reconciliation

Deterministic comparison and safe handling of local Project Profile declaration versus distributed authority state.

Required semantic distinctions include missing/no-decision, missing/resolved, present/no-authority, present/equivalent, present/conflicting, and stale-during-reconciliation states.

## Semantic equivalence

Equality of architecture meaning independent from incidental serialization differences such as YAML key order, whitespace, or Tool-generated formatting.

## Authority/Profile conflict

A state where a resolved distributed winner and an existing local Project Profile declaration disagree semantically for the same coordinated scope. The implementation must not silently overwrite either side.

Reference Tool `0.3.4` exposes this state as `AUTHORITY_PROFILE_CONFLICT`; the exact token is an interoperability detail unless separately standardized.

## Global decision state

Coordination-domain-wide decision state such as `PENDING` or `RESOLVED`.

## Local projection

Clone/worktree-local application/consistency state for a resolved architecture declaration. Local projection may be missing, consistent, locally applied, stale, or failed. It cannot change the global winner.

## Action-time synchronization

Synchronization performed when an active architecture-sensitive operation reaches a boundary where current authority matters. Continuous background polling is not required.

## Fail-closed distributed coordination

The rule that a selected distributed coordination operation must stop explicitly when required authority freshness or safe mutation cannot be established, rather than silently creating an isolated Local winner.

## PTSIP mandatory-rule waiver

PTSIP defines no waiver that authorizes a real `MUST`/`MUST NOT` violation. Project governance may record debt or migration approval, but confirmed violation remains `NON_CONFORMANT` until remediated.

## Conformance

A reproducible claim that a project satisfies a defined PTSIP conformance level/outcome under one exact bound Specification revision and evidence snapshot.
