# PTSIP Terminology

This document is the canonical human-readable terminology registry for PTSIP 0.2.0-draft.

## PTSIP Component

The architectural unit to which PTSIP ownership is assigned. A component may be a package, module group, executable, generated artifact group, single build/release script, protocol bundle, or another project-defined unit whose purpose and lifecycle can be evaluated coherently.

## Product SDK

An SDK or SDK-like package whose primary lifecycle ownership belongs to the software product and whose code or artifacts may be consumed by, embedded in, or distributed with the product.

## Toolchain SDK

An SDK or SDK-like package whose primary lifecycle ownership belongs to software development activities such as validation, migration, generation, build, testing, release preparation, inspection, or repository automation.

## Product SDK Plane

The architectural ownership domain containing Product SDKs and their product-facing dependencies.

## Toolchain SDK Plane

The architectural ownership domain containing Toolchain SDKs and development-only dependencies.

## Neutral Contract Artifact

A non-owning declarative contract consumable by both planes, such as schemas, IDLs, registries, immutable manifests, or conformance vectors. It is not a shared executable SDK.

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

## Observed evidence

Repository facts collected from files, manifests, dependency relationships, build/release definitions, artifacts, or runtime/static analysis. Observed evidence is distinct from a project's declared intended ownership.

## Specification Binding

A machine-readable association between a Consumer Repository's PTSIP declaration and the canonical PTSIP specification source and version/family used to interpret that declaration. For a mutable draft family, an immutable revision should additionally be recorded for reproducibility.

## Draft family

A version label such as `0.2.0-draft` that identifies an experimental specification family whose exact normative snapshot is identified by immutable repository revision until a stable specification release is declared.

## Boundary

The set of dependency, packaging, build, and lifecycle constraints separating Product and Toolchain planes.

## Isolation

The property that one plane can evolve, resolve dependencies, and be governed without accidentally acquiring the other plane's runtime or release obligations.

## Lifecycle ownership

Responsibility for versioning, compatibility, release, deprecation, and consumer impact of a component.

## Packaging ownership

Responsibility for deciding whether a component is included in a product artifact.

## Build environment

The declared dependency and execution context needed to resolve and build a plane. PTSIP does not require a particular environment technology.

## Cross-boundary dependency

A dependency edge from a component in one PTSIP plane to a component in another plane.

## PTSIP exception

A documented, governed deviation from a normative PTSIP rule.

## PTSIP Project Profile

A machine-readable declaration of intended component/path ownership and policies for one repository or project. A profile may use boundary roots as shorthand or component declarations for precise nested ownership. The profile is declaration, not proof of conformance. It is required for Enforced Conformance but is not required merely to run read-only inspection or pilot tooling.

## Conformance

A claim that a project satisfies a defined PTSIP conformance level according to `PTSIP-CONFORMANCE.md`.
