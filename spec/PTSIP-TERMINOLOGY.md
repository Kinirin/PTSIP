# PTSIP Terminology

This document is the canonical human-readable terminology registry for PTSIP 0.2.0-draft.

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

A version label such as `0.2.0-draft` that identifies an experimental specification family whose exact normative snapshot is identified by immutable repository revision until a stable specification release is declared.

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

## Project component dependency policy

An optional repository-specific policy declaring stricter component-to-component dependency constraints, including constraints within one PTSIP plane. Such policy may strengthen but MUST NOT weaken universal PTSIP requirements.

## Profile Validation

The operation that checks whether a PTSIP Project Profile or equivalent declaration is structurally and semantically valid. Profile Validation does not determine whether the repository conforms to PTSIP.

## Conformance Evaluation

The operation that combines applicable declarations with observed dependency, artifact, lifecycle, coverage, and other evidence to evaluate PTSIP rules.

## Conformance outcome

The result of a completed PTSIP Conformance Evaluation. Canonical outcomes are `CONFORMANT`, `NON_CONFORMANT`, and `INCOMPLETE`.

`NOT_EVALUATED` may be used as a tooling state but is not a conformance outcome.

## PTSIP diagnostic

A machine-readable finding produced by PTSIP evaluation. A diagnostic has its own unique instance identity and references the stable PTSIP rule ID it concerns. One rule may produce multiple diagnostic instances.

## Conformance

A claim that a project satisfies a defined PTSIP conformance level and outcome according to `PTSIP-CONFORMANCE.md`.
