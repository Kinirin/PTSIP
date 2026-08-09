# PTSIP Adoption Guide

This guide describes a controlled migration from an unclassified repository to PTSIP.

## Phase 0 — Stable baseline

Record the Consumer Repository revision before interpreting architecture evidence.

For automated Pilot evidence, rerun the analysis if HEAD or observed tracked state changes during collection. Do not mix evidence from different revisions into one conformance claim.

## Phase 1 — Inventory and evidence coverage

Inventory project-owned SDKs, packages, validators, migration tools, generators, build helpers, shared/common modules, manifests, relevant schemas/contracts, build/release automation, and known Product artifacts.

Do not move code yet.

Record inaccessible paths, parser failures, unsupported dependency forms, unresolved dynamic behavior, uninspected artifacts, and unsupported language/build/package adapters rather than silently treating them as absent.

For every evidence gap, ask whether it can conceal the result of an applicable mandatory PTSIP rule:

```text
can hide mandatory-rule result -> blocking gap
cannot affect mandatory-rule result -> non-blocking gap
```

Do not use a global unresolved-count or arbitrary coverage-percentage threshold as a substitute for rule-relative evidence sufficiency.

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

If one broad candidate contains materially different shipped state, executable purpose, release owner, compatibility owner, or Product/Toolchain lifecycle responsibility, split it into coherent ownership candidates rather than hiding the difference behind one root classification.

## Phase 3 — Classification decision

Resolve each in-scope project-owned component to exactly one architecture classification:

- `PRODUCT`
- `TOOLCHAIN`
- `NEUTRAL_CONTRACT`

During investigation, use decision status rather than inventing another class:

- `UNKNOWN` — insufficient evidence;
- `CONFLICT` — material evidence/declarations disagree;
- `INCOMPLETE` — required analysis coverage is missing.

External libraries, standard-library/platform nodes, and unresolved dependency targets are evidence-node scope, not extra PTSIP classifications.

An unresolved decision that affects a boundary relevant to a mandatory rule is a blocking gap and should block structural migration or conformance conclusions that depend on that ownership decision.

A coding agent may propose a schema-constrained decision with evidence IDs, but it does not automatically approve the project profile or exception governance.

Neutral Contract classification is not determined by a fixed number of current consumers. Evaluate non-executable/non-owning contract semantics and lifecycle ownership.

## Phase 4 — Dependency audit

Construct typed dependency evidence and identify:

- Product -> Toolchain edges;
- relationship type (`IMPORTS`, `LINKS`, `LOADS`, `INVOKES`, `READS`, `GENERATES`, `PACKAGES`, `TESTS`, `PUBLISHES`);
- lifecycle phase when known (`RUNTIME`, `BUILD`, `TEST`, `RELEASE`, `INSPECTION`);
- evidence provenance (`DECLARED`, `OBSERVED`, `INFERRED`);
- project/external/platform/unresolved target scope;
- unresolved/dynamic edges;
- shared executable packages used by both planes;
- common modules with no explicit owner.

Do not assume every Toolchain -> Product edge is allowed. Distinguish bounded inspection/build/test/analysis inputs from executable implementation reuse.

Do not guess phase or target merely to increase automation coverage.

## Phase 5 — Boundary declaration

Create or update the project-owned PTSIP profile only when the project wants a persistent declaration.

Use exactly one reference ownership-declaration mode:

- `boundaries` when root ownership is uniform; or
- `components` when nested/mixed/file-level boundaries exist.

Do not combine the two reference ownership modes in one profile.

A component profile records intended ownership. It does not prove that the dependency graph or Product artifacts obey the declaration.

When selectors overlap, exact/more-specific ownership wins. Equal-specificity ownership conflicts must be resolved explicitly.

A project may optionally declare stricter component-to-component dependency constraints, including same-plane constraints. These project-specific policies may strengthen but cannot weaken universal PTSIP rules.

## Phase 6 — Profile Validation

Validate the declaration itself before using it for automated Enforced Conformance.

Profile Validation may check:

- schema structure;
- specification-binding syntax;
- component IDs;
- selector conflicts;
- referenced components;
- exception fields; and
- optional project-policy consistency.

A successful Profile Validation result does **not** mean the repository is conformant.

## Phase 7 — Structural migration

Move or repackage components only after ownership decisions and dependency evidence are sufficiently stable.

Do not treat renaming alone as architectural migration. A directory move is incomplete if dependency and packaging behavior remain coupled.

If the repository uses explicit transition planning, keep these concepts separate:

```text
current observed architecture
!= target architecture
!= exception
!= conformance result
```

A migration target does not rewrite current observed conformance.

## Phase 8 — Build isolation

Make Product and Toolchain dependencies independently resolvable.

Verify that a clean Product build does not need Toolchain-only packages.

If Toolchain build/test/inspection uses Product implementation as an input, record the purpose and lifecycle phase instead of treating Toolchain -> Product direction as automatically allowed.

## Phase 9 — Product Artifact evidence

Identify the Product Artifact separately from the component that produced it.

For each artifact relevant to `PTSIP-PKG-001`, collect enough evidence to describe:

- artifact identity;
- Product/Toolchain/Neutral ownership when applicable;
- producer component;
- artifact type/format;
- shipping scope;
- contained paths/components or equivalent package manifest;
- derivation (`GENERATES`, `PACKAGES`, `PUBLISHES`); and
- evidence provenance.

A Toolchain build component may validly produce a Product Artifact. What matters for packaging isolation is the resulting artifact contents, not the producer's classification alone.

Source-path declarations alone are not sufficient evidence for `PTSIP-PKG-001` when actual Product artifact contents can differ from source topology.

The reference evidence shape is `ptsip-artifact-evidence/v1`.

## Phase 10 — Exception review

Record every intentional PTSIP `MUST`/`MUST NOT` violation as an explicit exception with rule ID, affected components, coupling, category/scope, ownership/approval authority, review/expiry condition, decision, and conformance effect.

An approved active PTSIP normative exception does not erase the violation. While the violation remains active, strict Core/Enforced PTSIP conformance is blocked.

Project-specific stricter dependency policy may have separate local governance, but local policy exceptions cannot waive universal PTSIP rules.

## Phase 11 — Conformance Evaluation

Conformance Evaluation combines declaration, observed dependencies, Product Artifact evidence, lifecycle evidence, coverage, exceptions, and deterministic PTSIP rules.

The completed evaluation uses:

- `CONFORMANT`;
- `NON_CONFORMANT`; or
- `INCOMPLETE`.

Decision sequence:

```text
Definite applicable mandatory violation?
    yes -> NON_CONFORMANT
    no
     |
     v
Blocking evidence gap?
    yes -> INCOMPLETE
    no  -> eligible for CONFORMANT when all other requirements are satisfied
```

A successful read-only Pilot or empty finding list is not by itself a conformance result.

For Enforced Conformance against a mutable draft family, bind the exact immutable PTSIP specification revision.

## Phase 12 — Stable diagnostics and repeatable enforcement

Automated checks should emit stable diagnostics that distinguish:

- diagnostic instance ID;
- PTSIP rule ID;
- outcome effect;
- severity;
- source/target component where applicable;
- evidence IDs;
- message; and
- evaluator/provenance information.

The reference diagnostic contract is `ptsip-diagnostic/v1`.

CI, developer tooling, coding agents, and external reports should consume the same diagnostic semantics rather than parse ad-hoc human messages.

## Phase 13 — External evidence integration (optional)

A project may use external repository-specific validators for security, license, governance, or other concerns.

PTSIP does not need to reimplement those validators merely to use their results as analysis input.

Imported evidence should carry provenance such as producer/version, subject repository/revision, scope, generation time, claims, and integrity information.

External evidence is input to PTSIP evaluation; it does not automatically override contradictory native observed evidence or universal PTSIP rules.

## Phase 14 — Lifecycle review

Distinguish these lifecycle events:

- workflow/pipeline trigger;
- Product artifact change;
- Product version/release decision;
- Product publication/deployment; and
- Product compatibility obligation.

A Toolchain-only change triggering a shared Product workflow is not automatically a lifecycle violation. The relevant question is whether it forces Product lifecycle obligations without a release-relevant Product change.

## Phase 15 — Conformance claim

Only after stable, sufficient evidence exists should the project claim `PTSIP Core Conformant` or `PTSIP Enforced Conformant`.

A project with active PTSIP normative exceptions may report those approved deviations and may describe itself as `PTSIP-adopting` or `PTSIP-transitioning`, but it must not call the active violation strict PTSIP conformance.

## Migration principle

PTSIP migration should optimize for **ownership correctness, evidence integrity, artifact truth, and future independent evolution**, not for minimizing the number of files changed in the first migration or maximizing automatic classification/coverage percentages.
