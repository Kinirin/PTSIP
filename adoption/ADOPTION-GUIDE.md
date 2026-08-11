# PTSIP Adoption Guide

This guide describes controlled migration to PTSIP `0.3.4-draft`. The exact governing snapshot is the immutable Specification revision bound by the Tool/profile.

## Phase 0 — Stable baseline

Record the Consumer Repository revision and relevant tracked-state fingerprint before interpreting architecture evidence. If repository state changes during collection or before a prepared profile write, re-analyze rather than mixing snapshots.

## Phase 1 — Inventory and evidence coverage

Inventory project-owned SDKs, packages, validators, migration tools, generators, build helpers, shared/common modules, manifests, contracts, build/release automation, and known Product artifacts.

Do not move code yet. Record inaccessible paths, parser failures, unsupported dependency forms, unresolved dynamic behavior, uninspected artifacts, and unsupported adapters.

A gap is blocking when it can hide the result of an applicable mandatory PTSIP rule. Do not use a global unresolved-count or arbitrary coverage percentage as a substitute for rule-relative sufficiency.

## Phase 2 — Component discovery

Discover component candidates from evidence such as manifests, release/build anchors, CI-invoked scripts, SDK/plugin projects, schema/protocol bundles, and artifact producers.

Directory names such as `tools/`, `scripts/`, or `build/` are hints, not architecture authority.

For each candidate record enough facts to support review, including purpose, consumers, shipping scope, executable/declarative nature, manifests, release/compatibility responsibility, evidence IDs, and counter-evidence.

Split broad candidates when one classification/purpose/lifecycle statement cannot coherently describe the whole scope.

## Phase 3 — Explicit architecture decision

Resolve each in-scope project-owned component to exactly one classification:

- `PRODUCT`;
- `TOOLCHAIN`;
- `NEUTRAL_CONTRACT`.

During investigation preserve `UNKNOWN`, `CONFLICT`, and `INCOMPLETE` as decision states rather than inventing another classification.

The project owner/user supplies architecture intent. A coding agent may surface evidence and ask for a decision but must not manufacture missing intent.

## Phase 4 — Durable adoption facts

`0.3.4-draft` defines this explicit component fact set for structured adoption/resolution:

- `classification`;
- `purpose`;
- `shipped`;
- `runtime_required`;
- `lifecycle_owner`;
- `executable`.

Canonical lifecycle owners are `PRODUCT`, `DEVELOPMENT_TOOLING`, and `INDEPENDENT`.

Required relationships include:

- `PRODUCT` -> `lifecycle_owner: PRODUCT`;
- `TOOLCHAIN` -> `lifecycle_owner: DEVELOPMENT_TOOLING`;
- `TOOLCHAIN` -> `shipped: false`;
- `TOOLCHAIN` -> `runtime_required: false`;
- `NEUTRAL_CONTRACT` -> `executable: false`;
- `NEUTRAL_CONTRACT` -> `lifecycle_owner: INDEPENDENT` when lifecycle ownership is represented.

`release_owner` and `compatibility_owner` remain separate optional project metadata; they do not substitute for canonical `lifecycle_owner`.

## Phase 5 — Project Profile declaration

The default profile is repository-root `ptsip.yaml`, although a project may consistently select another explicit path.

The reference schema supports either:

- `boundaries` for uniform-root shorthand; or
- `components` for precise nested/mixed/file-level ownership.

Do not combine both modes in one profile.

Boundary shorthand may remain structurally valid, but it cannot preserve the full structured adoption fact set. A write-enabled adoption/resolution workflow must not silently discard supplied facts. Migrate to component declarations before structured mutation when lossless representation is required.

## Phase 6 — `ptsip adopt`

`ptsip adopt` is dry-run by default. It uses discovered candidate scope but requires explicit architecture facts.

```powershell
ptsip adopt . `
  --component tools `
  --classification TOOLCHAIN `
  --purpose "Repository-local generation tooling" `
  --shipped no `
  --runtime-required no `
  --lifecycle-owner DEVELOPMENT_TOOLING `
  --executable yes `
  --json
```

Review the plan, then apply explicitly:

```powershell
ptsip adopt . `
  --component tools `
  --classification TOOLCHAIN `
  --purpose "Repository-local generation tooling" `
  --shipped no `
  --runtime-required no `
  --lifecycle-owner DEVELOPMENT_TOOLING `
  --executable yes `
  --apply `
  --json
```

The Tool must preserve the complete supplied fact set in component declarations. It must not map `lifecycle_owner` into `release_owner` or drop `runtime_required`.

Prepared mutation must be refused if the repository/profile changed after validation.

## Phase 7 — Multi-environment decision coordination

Local SQLite is intentionally local-only and must not be Git-shared as repository-global authority.

For GitHub-coordinated repositories, Reference Tool `0.3.4` uses a dedicated remote authority ref:

```text
refs/heads/ptsip-policy
```

That representation is a Reference Tool backend detail. The Specification-level requirements are backend-neutral.

A read-only authority check must not fabricate a pending decision merely to prove no decision exists.

Distributed writes must preserve first-valid-resolution-wins with conditional mutation so a stale writer cannot replace a newer winner.

## Phase 8 — Authority freshness and reconciliation

PTSIP uses action-time synchronization, not continuous polling. When an architecture-sensitive operation uses distributed coordination, a complete local profile does not permit skipping the relevant authority check.

Required reconciliation semantics are:

| Local Project Profile | Distributed Authority | Required behavior |
| --- | --- | --- |
| declaration absent | no decision | create/reuse pending state only when the active operation actually requires a decision |
| declaration absent | resolved winner | validate and safely project/reconcile the winner locally |
| declaration present | no authority decision | use the project declaration; do not fabricate authority history solely for bookkeeping |
| declaration present + semantically equivalent | resolved equivalent winner | report consistency/resolution without rewriting equivalent profile text |
| declaration present + semantically conflicting | resolved different winner | expose an explicit authority/profile conflict and do not silently overwrite either side |
| repository/profile changed during reconciliation | any authority state | refuse stale application and re-analyze |

Semantic equivalence is architecture meaning, not YAML formatting, key order, whitespace, or Tool-generated serialization.

If distributed coordination is selected but required freshness/safe mutation cannot be established, fail closed. Do not silently fall back to an isolated Local winner.

## Phase 9 — Global decision state versus local projection

Keep these states separate:

```text
GLOBAL DECISION STATE
    PENDING / RESOLVED

LOCAL PROJECTION STATE
    missing / consistent / locally applied / stale / failed
```

A global `RESOLVED` decision does not imply every clone has already written the declaration locally. A local application receipt cannot change the accepted winner.

## Phase 10 — Profile Validation

Validate the declaration before using it for Enforced Conformance.

Profile Validation checks schema/binding/component/policy semantics. It does not prove the repository conforms.

A boundary-root profile can be valid yet insufficient for strict evaluation when mandatory rule evaluation requires facts that shorthand does not carry.

## Phase 11 — Dependency and build audit

Construct typed dependency evidence and identify Product -> Toolchain edges, relationship types, lifecycle phases, provenance, external/platform/unresolved targets, shared executable packages, and undeclared ownership.

Make Product and Toolchain dependencies independently resolvable. A clean Product build must not require Toolchain-only packages merely because one developer environment contains them.

## Phase 12 — Product Artifact evidence

Identify Product Artifact owner separately from producer.

Collect artifact identity, owner/classification, producer, format, shipping scope, contents or equivalent package manifest, derivation relationships, and provenance.

A Toolchain producer may validly produce a Product Artifact. Product packaging conformance depends on the resulting contents, not producer classification alone.

## Phase 13 — Structural remediation

For each established mandatory violation, define the architecture change that actually satisfies the rule: remove dependency, split component, correct ownership, extract Neutral Contract, isolate packaging/build, or separate lifecycle responsibility.

Migration/debt approval does not waive a real violation.

## Phase 14 — Conformance Evaluation

Conformance Evaluation combines declaration, observed dependencies, Product Artifact evidence, lifecycle/build evidence, snapshot integrity, coverage, and deterministic PTSIP rules.

Completed outcomes are only:

- `CONFORMANT`;
- `NON_CONFORMANT`;
- `INCOMPLETE`.

Decision Authority is not a conformance oracle. A synchronized authority/profile pair can still describe a non-conformant repository.

For Enforced Conformance against `0.3.4-draft`, bind the exact immutable Specification revision.

## Phase 15 — Stable diagnostics and repeatability

Automated checks should emit stable diagnostics that distinguish diagnostic instance ID from rule ID and preserve evidence IDs, outcome effect, severity, component endpoints where applicable, message, and evaluator/provenance information.

Rerun evaluation after remediation against a stable snapshot.

## Migration principle

Optimize for **ownership correctness, durable architecture intent, distributed decision consistency, evidence integrity, artifact truth, and independent lifecycle evolution** rather than minimizing the number of files changed or maximizing automatic classification.
