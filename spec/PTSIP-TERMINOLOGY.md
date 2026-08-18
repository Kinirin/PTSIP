# PTSIP Terminology

This document is the canonical human-readable terminology registry for PTSIP `0.3.6-draft`. The exact draft snapshot is identified by immutable Git revision.

## Classification

The canonical PTSIP **primary lifecycle ownership** of an in-scope project-owned architectural responsibility.

For `0.3.6-draft`, the only canonical classifications are:

- `PRODUCT`;
- `DEVELOPMENT_TOOLING`;
- `DELIVERY`;
- `OPERATIONS`;
- `NEUTRAL_CONTRACT`.

Classification is not a file type, framework label, role, evidence state, confidence value, or VPMS Verification Purpose.

## Primary lifecycle ownership

The lifecycle that primarily owns the existence, change responsibility, compatibility obligations, release/deployment/operation implications, and consumer impact of a project-owned architectural responsibility.

For `0.3.6-draft`, `classification` is the canonical representation of primary lifecycle ownership.

A separate legacy `lifecycle_owner` field may be consumed by migration tooling as historical metadata, but it must not create a second competing ownership authority or contradict canonical `classification` semantics.

## Governing lifecycle obligation

The lifecycle-specific reason a coherent project-owned responsibility must exist, change, remain compatible, execute, or be retired.

PTSIP lifecycle determination uses the governing lifecycle obligation rather than the most common file type, workflow phase, activity verb, technology, invocation frequency, or confidence score.

## Delivery handoff

The semantic boundary where a selected release target has reached or been accepted by its intended publication/deployment destination and ordinary ongoing operation begins.

Responsibilities whose primary obligation ends at this boundary are `DELIVERY` candidates. Responsibilities whose primary obligation continues after this boundary to maintain deployed state are `OPERATIONS` candidates.

## Material mixed-lifecycle responsibility

A component state in which one classification cannot describe lifecycle truth without hiding an independently governable responsibility.

Material mixed-lifecycle evidence can include different lifecycle triggers, release/compatibility/permission/environment obligations, independently evolving dependencies, distinct failure meanings/owners, or independently changeable release/deploy/operate behavior.

When material responsibilities are separable, PTSIP prefers a component-split proposal over forcing one classification.

## Subordinate lifecycle activity

An activity from another lifecycle phase that exists only to complete one coherent primary lifecycle obligation and does not introduce independently governable lifecycle ownership, release, compatibility, permission, environment, or dependency obligations.

Subordinate activity does not by itself require a component split.

## `PRODUCT`

Primary lifecycle ownership for Product runtime, user-facing behavior, Product distribution content, runtime SDK responsibility, and Product-specific quality or verification responsibility.

Product-owned tests are valid `PRODUCT` responsibilities when their existence and change obligations are owned by the Product lifecycle.

## `DEVELOPMENT_TOOLING`

Primary lifecycle ownership for development support such as reusable verification infrastructure, test SDKs/frameworks, generators, migration tooling, linting, static analysis, developer CLI tooling, build helpers, repository transformation, and development-environment tooling.

It contains much of the responsibility previously represented by Tool `0.3.5` `TOOLCHAIN`, but it is not a universal one-to-one rename target for legacy `TOOLCHAIN`.

## `DELIVERY`

Primary lifecycle ownership for carrying a release target or Product through release, publication, promotion, distribution, or deployment to a destination environment.

Typical responsibilities include release workflows, package publication, container image build/publication, artifact signing, deployment automation, deployment manifests/configuration, and release promotion.

## `OPERATIONS`

Primary lifecycle ownership for post-deployment operation: maintaining availability, recoverability, production state, infrastructure health, backup/recovery, incident response support, and ongoing production maintenance.

The Delivery/Operations distinction is based on primary lifecycle purpose and the delivery handoff, not technology.

## `NEUTRAL_CONTRACT`

A deliberately non-executable, non-owning contract responsibility with lifecycle independence from Product, Development Tooling, Delivery, and Operations ownership.

Neutrality is determined by semantics and independent lifecycle, not by directory name, file extension, non-executable form, or current consumer count.

## Legacy `TOOLCHAIN`

A canonical Tool `0.3.5` classification retained only as a **legacy migration input concept** in Tool `0.3.6`.

`TOOLCHAIN` is not a canonical Tool `0.3.6` lifecycle classification and must not be preserved as a new-schema alias merely for compatibility.

Legacy `TOOLCHAIN` responsibility may migrate to `PRODUCT`, `DEVELOPMENT_TOOLING`, `DELIVERY`, `OPERATIONS`, a component split, or unresolved clarification depending on repository evidence and project-owner confirmation.

## PTSIP Component

A coherent project-owned architectural unit to which one primary lifecycle classification and one primary purpose can apply.

A component may be a package, module group, executable, generated artifact group, script, protocol bundle, workflow group, infrastructure responsibility, verification unit, or another project-defined unit.

## Component boundary

The declared scope that keeps one component's primary lifecycle ownership and purpose coherent.

If one old component contains materially different Product, Development Tooling, Delivery, Operations, or independent-contract responsibilities, migration may require a component-split proposal rather than a classification rename.

## Component-split decision

The lifecycle-boundary result used when one candidate contains multiple materially independent lifecycle responsibilities that can be represented separately.

The decision test is whether independently governable lifecycle responsibility exists, not whether several activity verbs or workflow steps appear in one file.

## Responsibility role

The responsibility performed by a component **inside** its lifecycle ownership.

Role is orthogonal to classification. A `PRODUCT` component and a `DEVELOPMENT_TOOLING` component may both perform verification roles without sharing lifecycle ownership.

The canonical closed role vocabulary is defined by Responsibility Map v2 schema/rule work and must remain separate from lifecycle classification.

## Typed responsibility relationship

A declared semantic relationship between project responsibilities or associated artifacts, such as verification, generation, packaging, publication, documentation, specification, governance, or another schema-defined relationship.

Typed relationships preserve semantics that cannot be represented losslessly by an untyped `from`/`to` edge.

Evidence relationship vocabulary and project-declared Responsibility Map relationships are related but are not automatically identical namespaces.

## Associated artifact

A project-owned documentation/authority/support artifact that is explicitly associated with a component without being promoted to an independent architecture component merely to express the relationship.

Associated-artifact representation is not a classification escape hatch. Executable responsibility, independent lifecycle/release/compatibility ownership, or independently owned cross-lifecycle contract semantics require component/classification evaluation.

## Product-owned test

A test or verification responsibility whose existence, change responsibility, compatibility expectations, and quality obligation belong primarily to the Product lifecycle.

The directory name `tests/` does not force `DEVELOPMENT_TOOLING` ownership.

## Reusable verification infrastructure

Verification implementation, test SDK/framework, or shared verification mechanism whose primary ownership belongs to the development lifecycle rather than to one Product's own lifecycle.

It is typically a `DEVELOPMENT_TOOLING` responsibility, even when it verifies `PRODUCT` behavior.

## Development build

Build responsibility whose governing obligation is coding, local compilation, inspection, testing, or creation of non-release intermediate outputs.

It is normally a `DEVELOPMENT_TOOLING` candidate.

## Delivery build

Build responsibility whose governing obligation is to materialize, assemble, sign, verify for handoff, package, or otherwise prepare an authoritative release/distribution/deployment unit for delivery to a destination.

It is normally a `DELIVERY` candidate. The resulting Product Artifact may remain `PRODUCT`-owned.

## Artifact-kind neutrality rule

The principle that file type, language, framework, path, executable status, compilation behavior, or workflow provider alone does not determine PTSIP classification.

FastAPI, Cloudflare Workers, GitHub Actions, Docker, Terraform, Python, PowerShell, Markdown, YAML, tests, and workflows do not have universal PTSIP classifications by themselves.

## Product Artifact

A Product-owned deployable, distributable, installable, loadable, or otherwise release-relevant output.

Artifact ownership is independent from producer ownership. A `DEVELOPMENT_TOOLING` or `DELIVERY` producer may create/package/publish a Product Artifact without becoming Product-owned.

## Artifact producer

The component/process that generates, assembles, packages, signs, publishes, or otherwise produces an artifact.

Producer lifecycle ownership does not automatically determine artifact ownership.

## Artifact derivation

Evidence describing how an artifact was generated, assembled, packaged, signed, published, promoted, or deployed from source components or other artifacts.

## Responsibility Map

A project-owned machine-readable declaration of intended lifecycle ownership, component boundaries, and related responsibility semantics.

Discovery output, migration proposals, templates, and coding-agent confidence are not architecture authority.

## Responsibility Map v2

The Tool `0.3.6` Responsibility Map generation whose canonical schema is based on the five primary lifecycle classifications and can preserve role/typed-relationship/associated-artifact semantics without retaining legacy `TOOLCHAIN` as a canonical alias.

Its conceptual modes are `explicit`, `template`, and `hybrid`.

## Explicit Responsibility Map

A Responsibility Map in which the repository directly declares the complete canonical project-owned map without runtime template defaults.

## Template Responsibility Map

A Responsibility Map that explicitly selects a supported versioned/revision-bound template.

Template selection is project-owned and must not be inferred solely from repository shape or discovery confidence.

## Hybrid Responsibility Map

A Responsibility Map that combines an explicitly selected template with project-owned overrides/extensions.

Explicit repository declarations take precedence within the schema-defined override boundary.

## Legacy profile reader

Tool `0.3.6` functionality that recognizes and parses valid Tool `0.3.5` Project Profiles for inspection and migration without treating the old ontology as canonical `0.3.6` semantics.

## Assisted migration

A preview-first process that combines legacy declarations with repository evidence to propose lifecycle classification, role, relationship, associated-artifact, and component-boundary changes for project-owner confirmation.

Assisted migration shifts evidence-collection cost toward tooling while keeping final architecture authority with the project owner.

## Migration proposal

A non-authoritative proposed change from legacy/current representation to a target Responsibility Map.

A useful proposal may include supporting evidence, provenance, confidence/ambiguity, component-split effects, semantic diff, textual diff, and unresolved facts.

## Component-split proposal

A migration proposal that separates one legacy component into multiple components because one primary lifecycle classification can no longer describe its responsibilities coherently.

## PTSIP Project Profile

A project-owned machine-readable declaration of intended architecture ownership and policy. It is durable declaration state, not observed conformance truth.

For Tool `0.3.6`, the canonical profile representation is Responsibility Map v2 once the schema work is frozen. Valid Tool `0.3.5` profiles remain legacy migration inputs rather than automatically canonical `0.3.6` profiles.

## Decision status

The state of an architecture/ownership decision during inspection or migration. Examples include `RESOLVED`, `UNKNOWN`, `CONFLICT`, `PENDING`, and `INCOMPLETE`.

A decision status is not a PTSIP classification.

## Consumer Repository

A repository being inspected, adopted, validated, coordinated, migrated, or governed using PTSIP. PTSIP does not own the repository's directory, testing, development, or documentation conventions.

## External PTSIP Tooling

PTSIP implementation operated outside the Consumer Repository project-owned source tree. It is outside repository classification scope unless intentionally vendored or adopted by the project.

## Consumer Repository Non-Intrusion

The property that external PTSIP tooling can operate without requiring PTSIP-specific repository directories or default repository mutation. Tool-owned operational state should remain outside the Consumer Repository unless the user explicitly chooses otherwise.

## Specification Binding

The association between a PTSIP declaration/Tool and the canonical Specification source, draft family/version, and immutable revision used to interpret it.

## Draft family

A mutable family label such as `0.3.6-draft`. The label names the evolving family; an immutable Git revision identifies an exact normative snapshot.

## Evidence snapshot

The observable Consumer Repository state to which one automated evidence set is bound. For Git repositories this normally includes immutable revision identity and enough tracked-state information to detect instability during collection.

## Observed evidence

Repository/artifact/runtime/static-analysis facts collected directly from observable state.

## Declared evidence

Facts explicitly declared by a profile, manifest, configuration, Responsibility Map, or other project-owned declaration. Declaration is not proof that observed behavior agrees.

## Inferred evidence

A bounded conclusion derived from evidence by deterministic logic, human review, or constrained agent reasoning and explicitly marked as inferred.

## Evidence provenance

The origin category attached to evidence. Canonical values carried forward are `DECLARED`, `OBSERVED`, and `INFERRED`.

## Candidate discovery

Evidence acquisition that identifies likely project responsibilities, boundaries, roles, relationships, lifecycle ownership candidates, and migration candidates.

Candidate discovery is not project architecture authority.

## Dependency/evidence relationship type

The semantic relationship represented by an observed/declared evidence edge.

Canonical values carried forward from `0.3.4-draft` are `IMPORTS`, `LINKS`, `LOADS`, `INVOKES`, `READS`, `GENERATES`, `PACKAGES`, `TESTS`, and `PUBLISHES`.

Responsibility Map v2 may define additional project-declared relationship types after schema/rule normalization.

## Lifecycle phase

The lifecycle context of an evidence relationship. Canonical phases carried forward are `RUNTIME`, `BUILD`, `TEST`, `RELEASE`, and `INSPECTION`.

Tool `0.3.6` evidence work may add delivery/operations phases. Unknown phase remains unresolved rather than guessed.

## Evidence node scope

Dependency/artifact graph scope distinct from PTSIP classification. Reference scopes include `PROJECT_COMPONENT`, `EXTERNAL_DEPENDENCY`, `PLATFORM`, and `UNRESOLVED_TARGET`.

## Boundary

The dependency, packaging, release, delivery, operations, compatibility, and lifecycle constraints separating primary lifecycle responsibilities.

## Isolation

The property that one lifecycle responsibility can evolve and be governed without accidentally acquiring incompatible runtime, release, delivery, operations, or compatibility obligations from another lifecycle.

## Packaging ownership

Responsibility for deciding whether a component belongs in a Product Artifact or another distributable output.

## Build environment

The dependency and execution context needed to resolve/build a lifecycle responsibility. PTSIP does not require a particular environment technology.

## Cross-lifecycle dependency

A dependency relationship from a component in one PTSIP lifecycle classification to a component in another lifecycle classification.

## Blocking evidence gap

An evidence gap capable of concealing whether an applicable PTSIP `MUST`/`MUST NOT` is satisfied. It prevents `CONFORMANT` unless a definite violation already establishes `NON_CONFORMANT`.

## Non-blocking evidence gap

An evidence gap that cannot materially change the applicable mandatory-rule result.

## Project component dependency policy

Optional repository-specific dependency constraints that may strengthen but never weaken universal PTSIP requirements.

## Profile Validation

Validation of Project Profile structure/semantics. It does not determine repository conformance.

## Migration analysis

Evidence-backed proposal generation for legacy/current architecture representation. Migration analysis does not itself authorize repository mutation or architecture decisions.

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

## Read-only authority observation

A non-mutating lookup used to determine whether relevant authority state exists. Absence checking should not fabricate a pending decision or create history merely to prove absence.

## Authority/Profile reconciliation

Deterministic comparison and safe handling of local Project Profile declaration versus distributed authority state.

Required semantic distinctions include missing/no-decision, missing/resolved, present/no-authority, present/equivalent, present/conflicting, and stale-during-reconciliation states.

## Semantic equivalence

Equality of architecture meaning independent from incidental serialization differences such as YAML key order, whitespace, or Tool-generated formatting.

## Authority/Profile conflict

A state where a resolved distributed winner and an existing local Project Profile declaration disagree semantically for the same coordinated scope. The implementation must not silently overwrite either side.

## Global decision state

Coordination-domain-wide decision state such as `PENDING` or `RESOLVED`.

## Local projection

Clone/worktree-local application/consistency state for a resolved architecture declaration. Local projection may be missing, consistent, locally applied, stale, or failed. It cannot change the global winner.

## Action-time synchronization

Synchronization performed when an active architecture-sensitive operation reaches a boundary where current authority matters. Continuous background polling is not required.

## Fail-closed distributed coordination

The rule that a selected distributed coordination operation must stop explicitly when required authority freshness or safe mutation cannot be established, rather than silently creating an isolated Local winner.

## VPMS

Verification Purpose Management System. VPMS manages why a Verification Case exists and what it protects. VPMS purpose is orthogonal to PTSIP lifecycle classification.

Tool `0.3.5` VPMS purpose vocabulary currently uses `PRODUCT | TOOLCHAIN`. The `TOOLCHAIN` token in that VPMS vocabulary is not a Tool `0.3.6` PTSIP lifecycle classification and must not be renamed accidentally as part of lifecycle-ontology migration.

## Verification Case

The purpose-bound VPMS execution unit that combines Formula, Variables, Policy, Target, and Runner concerns.

## Formula

Purpose-neutral reusable verification mechanism where possible. Formula reuse does not collapse Verification Purpose or PTSIP lifecycle ownership.

## Verification Purpose

The VPMS axis describing what a Verification Case protects/verifies. It is not PTSIP lifecycle classification.

## PTSIP mandatory-rule waiver

PTSIP defines no waiver that authorizes a real `MUST`/`MUST NOT` violation. Project governance may record debt or migration approval, but confirmed violation remains `NON_CONFORMANT` until remediated.

## Conformance

A reproducible claim that a project satisfies a defined PTSIP conformance outcome under one exact bound Specification revision and evidence snapshot.