# PTSIP Terminology

This document is the canonical human-readable terminology registry for PTSIP 0.1.0-draft.

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

A machine-readable declaration of how one repository or project maps its paths and policies to PTSIP.

## Conformance

A claim that a project satisfies a defined PTSIP conformance level according to `PTSIP-CONFORMANCE.md`.
