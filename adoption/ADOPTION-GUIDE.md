# PTSIP Adoption Guide

This guide describes a controlled migration from an unclassified repository to PTSIP.

## Phase 0 — Stable baseline

Record the Consumer Repository revision before interpreting architecture evidence.

For automated Pilot evidence, rerun the analysis if HEAD or observed tracked state changes during collection. Do not mix evidence from different revisions into one conformance claim.

## Phase 1 — Inventory and evidence coverage

Inventory project-owned SDKs, packages, validators, migration tools, generators, build helpers, shared/common modules, manifests, relevant schemas/contracts, and build/release automation.

Do not move code yet.

Record inaccessible paths, parser failures, unsupported dependency forms, and unresolved dynamic behavior rather than silently treating them as absent.

## Phase 2 — Component discovery

Identify architectural component candidates using evidence such as:

- package/build manifests;
- independent release/build anchors;
- CI-invoked scripts;
- plugin or SDK project files;
- schema/protocol bundles;
- test/tool roots;
- artifact producers.

Directory names are hints, not ownership decisions.

For each candidate record:

- primary purpose;
- consumers;
- shipped/not shipped;
- executable/declarative nature;
- dependency manifest;
- release owner;
- compatibility owner;
- evidence IDs and counter-evidence.

## Phase 3 — Classification decision

Resolve each in-scope component to exactly one architecture classification:

- `PRODUCT`
- `TOOLCHAIN`
- `NEUTRAL_CONTRACT`

During investigation, use decision status rather than inventing another class:

- `UNKNOWN` — insufficient evidence;
- `CONFLICT` — material evidence/declarations disagree;
- `INCOMPLETE` — required analysis coverage is missing.

An unresolved decision that affects a boundary relevant to a mandatory rule blocks strict conformance and should block structural migration that depends on that ownership decision.

A coding agent may propose a schema-constrained decision with evidence IDs, but it does not automatically approve the project profile or exception governance.

## Phase 4 — Dependency audit

Construct typed dependency evidence and identify:

- Product -> Toolchain edges;
- edge lifecycle phase when known (`RUNTIME`, `BUILD`, `TEST`, `RELEASE`, `INSPECTION`);
- unresolved/dynamic edges;
- shared executable packages used by both planes;
- Toolchain dependencies accidentally packaged with Product;
- common modules with no explicit owner.

Do not assume every Toolchain -> Product edge is allowed. Distinguish inspection/analysis inputs from executable implementation reuse.

## Phase 5 — Boundary declaration

Create or update the project-owned PTSIP profile only when the project wants a persistent declaration.

Use boundary roots as shorthand when ownership is uniform. Use `components` when nested or file-level boundaries exist.

A component profile records intended ownership. It does not prove that the dependency graph or Product artifacts obey the declaration.

When selectors overlap, exact/more-specific ownership wins. Equal-specificity ownership conflicts must be resolved explicitly.

## Phase 6 — Structural migration

Move or repackage components only after ownership decisions and dependency evidence are sufficiently stable.

Do not treat renaming alone as architectural migration. A directory move is incomplete if dependency and packaging behavior remain coupled.

## Phase 7 — Build isolation

Make Product and Toolchain dependencies independently resolvable.

Verify that a clean Product build does not need Toolchain-only packages.

## Phase 8 — Packaging validation

Inspect actual Product artifacts and prove Toolchain-only code/dependencies are absent.

Source-path declarations alone are not sufficient evidence for `PTSIP-PKG-001`.

## Phase 9 — Enforcement

Add repeatable PTSIP checks that compare declarations with observed dependency/build/package evidence and emit PTSIP rule IDs.

Suppressions or exceptions must remain explicit and reviewable.

## Phase 10 — Conformance claim

Only after stable evidence exists should the project claim `PTSIP Core Conformant` or `PTSIP Enforced Conformant`.

A successful `ptsip pilot` evidence collection is not by itself a conformance claim.

## Migration principle

PTSIP migration should optimize for **ownership correctness, evidence integrity, and future independent evolution**, not for minimizing the number of files changed in the first migration or maximizing automatic classification percentage.
