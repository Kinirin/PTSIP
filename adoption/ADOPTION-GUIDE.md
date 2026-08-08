# PTSIP Adoption Guide

This guide describes a controlled migration from an unclassified repository to PTSIP.

## Phase 1 — Inventory

Inventory all project-owned SDKs, packages, validators, migration tools, generators, build helpers, and shared/common modules.

Do not move code yet.

For each component record:

- primary purpose;
- consumers;
- shipped/not shipped;
- dependency manifest;
- release owner;
- compatibility owner.

## Phase 2 — Classification

Classify each component as:

- `PRODUCT`
- `TOOLCHAIN`
- `NEUTRAL_CONTRACT`

Any `UNKNOWN` item blocks structural migration until ownership is decided.

## Phase 3 — Dependency audit

Construct the dependency graph and identify:

- Product -> Toolchain edges;
- shared executable packages used by both planes;
- Toolchain dependencies accidentally packaged with Product;
- common modules with no explicit owner.

## Phase 4 — Boundary design

Define Product roots, Toolchain roots, Contract roots, and build contexts in `ptsip.yaml`.

Prefer contract extraction over cross-plane executable sharing where semantics must remain common.

## Phase 5 — Structural migration

Move or repackage components so ownership is visible.

Do not treat renaming alone as architectural migration. A directory move is incomplete if dependency and packaging behavior remain coupled.

## Phase 6 — Build isolation

Make Product and Toolchain dependencies independently resolvable.

Verify that a clean Product build does not need Toolchain-only packages.

## Phase 7 — Packaging validation

Inspect Product artifacts and prove Toolchain-only code is absent.

## Phase 8 — Enforcement

Add automated PTSIP checks and rule-ID diagnostics.

## Phase 9 — Conformance claim

Only after evidence exists should the project claim `PTSIP Core Conformant` or `PTSIP Enforced Conformant`.

## Migration principle

PTSIP migration should optimize for **ownership correctness and future independent evolution**, not for minimizing the number of files changed in the first migration.
