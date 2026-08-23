# PTSIP Tool 0.3.6 Adoption Guide

This guide describes controlled adoption of PTSIP Tool `0.3.6` and Specification family `0.3.6-draft`. The exact governing snapshot is the immutable Specification revision bound by the installed Tool or Consumer Repository profile.

Current Tool `0.3.6` binding:

```text
Specification 0.3.6-draft
SPEC_REVISION d6995ed232e845b88d8235b851e80ab54b7804ea
```

Tool `0.3.6` establishes the canonical Primary Lifecycle Ownership model and Responsibility Map v2. Evidence-driven assisted migration from Tool `0.3.5` declarations is intentionally deferred to Tool `0.3.6.1`; this guide does not pretend that migration is automatic.

## Phase 0 — Stable repository snapshot

Before interpreting architecture evidence, record the Consumer Repository revision and relevant tracked-state fingerprint.

If repository state changes during evidence collection or before an authorized profile write, stop and re-analyze rather than mixing snapshots.

PTSIP treats exact repository state as part of evidence integrity. A stale prepared mutation must fail closed.

## Phase 1 — Identify coherent project responsibilities

Inventory project-owned responsibilities such as:

```text
Product runtime/library code
reusable development tooling
verification frameworks and test SDKs
release/package/publication automation
post-deployment operational automation
contracts and schemas
associated documentation/governance artifacts
Product distributions and other release artifacts
```

Do not classify from path names alone. `src/`, `tests/`, `tools/`, `.github/workflows/`, `deploy/`, and `ops/` are evidence context, not architecture authority.

Split broad candidates when one lifecycle ownership statement cannot coherently describe the whole scope.

## Phase 2 — Determine primary lifecycle ownership

Resolve each in-scope project-owned component to exactly one canonical Tool `0.3.6` classification:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

Use the governing lifecycle obligation:

- `PRODUCT` — Product functionality, Product-owned verification, or other responsibility primarily owned by the Product lifecycle;
- `DEVELOPMENT_TOOLING` — development-time creation, inspection, validation, transformation, generation, migration, analysis, reusable verification infrastructure, or local/intermediate development support;
- `DELIVERY` — release-unit assembly, signing, packaging, publication, promotion, distribution, or deployment until delivery handoff;
- `OPERATIONS` — ongoing health, recovery, maintenance, reconciliation, or operation after delivery handoff;
- `NEUTRAL_CONTRACT` — deliberately non-executable, non-owning, lifecycle-independent contract responsibility.

`UNKNOWN`, `CONFLICT`, and `INCOMPLETE` are decision/evaluation states, not classifications.

If mixed lifecycle responsibilities are independently governable, split them rather than choosing the majority of files, jobs, steps, duration, or confidence.

## Phase 3 — Keep architecture axes separate

Responsibility Map v2 separates:

```text
classification
    = primary lifecycle ownership

roles
    = coarse responsibility characteristics

relationships
    = typed project-owned semantic edges

source/derived provenance
    = declaration/materialization origin

VPMS Verification Purpose
    = why verification exists and what it protects
```

Canonical roles are:

```text
IMPLEMENTATION
VERIFICATION
AUTOMATION
CONFIGURATION
DOCUMENTATION
GOVERNANCE
```

Canonical project-declared relationships are:

```text
IMPORTS
LINKS
LOADS
INVOKES
READS
GENERATES
BUILDS
PACKAGES
PUBLISHES
DEPLOYS
VERIFIES
MANAGES
DOCUMENTS
SPECIFIES
GOVERNS
```

A relationship is not a substitute for classification. Observed evidence may support a relationship proposal but must not silently become project-owned declaration authority.

## Phase 4 — Choose a Responsibility Map source mode

Tool `0.3.6` supports three canonical source modes:

### Explicit

The repository directly declares the complete Responsibility Map.

```yaml
responsibility_map:
  mode: explicit
```

### Template

The repository explicitly selects an immutable revision-bound template.

```yaml
responsibility_map:
  mode: template
  template:
    id: python-package-library
    revision: "sha256:..."
```

### Hybrid

The repository selects a template and records project-owned overrides, extensions, or removals.

```yaml
responsibility_map:
  mode: hybrid
  template:
    id: python-package-library
    revision: "sha256:..."
  overrides:
    components:
      - id: product
        # project-owned replacement facts
```

The current template catalog includes:

```text
python-package-library
python-cli-application
mixed-product-development-delivery
```

Template selection is explicit. Do not auto-select a template from layout, framework, language, manifests, or similarity confidence.

## Phase 5 — Materialize the Effective Responsibility Map

All source modes resolve through deterministic materialization:

```text
Source Project Profile
    -> validate source declaration and binding
    -> deterministic materialization
    -> ResolvedProfile
    -> Canonical Effective Responsibility Map
```

The Source Project Profile remains architecture authority.

The materializer must not:

- infer lifecycle ownership;
- choose a template;
- repair invalid architecture;
- invent missing project intent;
- silently rewrite the source declaration.

Downstream validation, conformance, clarification/adoption, and narrow VPMS integration consume the validated resolved map rather than each reinterpreting raw YAML independently.

## Phase 6 — Preserve declaration provenance

Effective architecture may retain declaration provenance such as:

```text
PROJECT_EXPLICIT
TEMPLATE
PROJECT_OVERRIDE
PROJECT_EXTENSION
PROJECT_REMOVAL
```

This provenance describes where effective architecture came from. It is distinct from evidence provenance and does not replace project authority.

Evidence provenance remains:

```text
DECLARED
OBSERVED
INFERRED
```

Keep these concepts separate.

## Phase 7 — Explicit adoption facts

Canonical Tool `0.3.6` uses `classification` itself as primary lifecycle ownership authority. Do not introduce a second canonical `lifecycle_owner` field that competes with it.

When an explicit adoption/resolution workflow records structured facts, preserve the applicable project decision losslessly, including facts such as:

```text
classification
roles
purpose
shipped
runtime_required
executable
associated artifacts
typed relationships
explicit release/compatibility metadata
```

Canonical Tool `0.3.6` clarification/adoption answers use the v2 fact model centered on:

```text
classification
purpose
shipped
runtime_required
executable
```

Legacy Tool `0.3.5` facts such as `TOOLCHAIN`, `lifecycle_owner`, old boundary roots, consumers, analysis inputs, and untyped dependency edges are migration evidence only.

## Phase 8 — Use `ptsip adopt` safely

`ptsip adopt` is dry-run by default.

Example development-tooling decision:

```powershell
ptsip adopt . `
  --component tools `
  --classification DEVELOPMENT_TOOLING `
  --purpose "Repository-local generation tooling" `
  --shipped no `
  --runtime-required no `
  --executable yes `
  --json
```

Review the planned declaration change, then apply explicitly:

```powershell
ptsip adopt . `
  --component tools `
  --classification DEVELOPMENT_TOOLING `
  --purpose "Repository-local generation tooling" `
  --shipped no `
  --runtime-required no `
  --executable yes `
  --apply `
  --json
```

If the selected profile or repository snapshot changes after analysis, prepared mutation must be rejected and recomputed.

## Phase 9 — Decision Authority is not Project Profile

Keep these responsibilities separate:

```text
Specification
    -> normative rules

Decision Authority
    -> which explicit coordinated architecture answer won

Project Profile / Responsibility Map
    -> durable project-owned architecture declaration

Observed evidence
    -> what repository/artifacts actually do

Conformance Evaluation
    -> deterministic rule evaluation
```

A resolved Decision Authority winner does not automatically mean every clone has applied the declaration and does not prove conformance.

Local SQLite state is Tool-owned operational state, not portable repository-global authority.

## Phase 10 — Multi-environment decision coordination

For GitHub-coordinated repositories, the Reference Tool uses a dedicated authority ref:

```text
refs/heads/ptsip-policy
```

The Git representation is a Tool backend detail; the important semantics are backend-neutral:

- stable coordination domain and component scope identity;
- first-valid-resolution-wins;
- stale-writer-safe conditional mutation/CAS;
- current authority reads at sensitive boundaries;
- deterministic reconciliation;
- fail-closed behavior;
- global decision state separated from clone-local projection state.

A read-only authority check must not fabricate a pending decision merely to prove no decision exists.

If distributed freshness or safe mutation cannot be established, do not silently fall back to an isolated local winner.

## Phase 11 — Reconcile local declaration and distributed authority

Use semantic architecture meaning rather than YAML formatting.

| Local Project Profile | Distributed Authority | Required behavior |
| --- | --- | --- |
| declaration absent | no decision | create/reuse pending only when the active operation really requires a decision |
| declaration absent | resolved winner | validate and safely project/reconcile the winner locally |
| declaration present | no decision | use the project declaration; do not fabricate authority history solely for bookkeeping |
| declaration present + equivalent | resolved equivalent winner | report consistency without rewriting equivalent formatting |
| declaration present + conflicting | resolved different winner | expose conflict and do not silently overwrite either side |
| repository/profile changed | any authority state | refuse stale application and re-analyze |

Global state and local projection remain separate:

```text
GLOBAL
    PENDING / RESOLVED

LOCAL
    missing / consistent / locally applied / stale / failed
```

PTSIP uses action-time synchronization rather than continuous background polling.

## Phase 12 — Profile Validation

Validate the selected/materialized Project Profile before using it for downstream evaluation.

Profile Validation checks declaration structure and semantics: schema, binding, source mode, selectors, component/associated-artifact identity, relationship endpoints, template binding, and project policy.

Profile Validation does not prove conformance.

## Phase 13 — Dependency and lifecycle evidence

Collect typed repository evidence with provenance. Identify cross-lifecycle edges, external/platform/unresolved targets, build/release behavior, runtime requirements, and coverage gaps.

Evidence is not authority. A Tool may infer that an edge exists, but it must not use that observation to silently rewrite project-owned lifecycle ownership.

Make lifecycle environments independently resolvable. Product runtime/build requirements must not depend on Development Tooling, Delivery, or Operations implementation merely because one developer environment contains everything.

## Phase 14 — Product Artifact evidence

Artifact owner and producer are separate.

A `DEVELOPMENT_TOOLING` or `DELIVERY` component may validly build/package/publish a `PRODUCT` artifact. Product packaging conformance depends on the resulting artifact contents and declared Product boundary, not producer classification alone.

For release-relevant evaluation collect:

```text
artifact identity/type
classification
producer component
shipping scope
actual included paths/components
complete-content assertion
provenance
derivation
exact repository/snapshot binding
```

Tool `0.3.6` supports `ptsip-artifact-evidence/v1` plus exact snapshot binding. Actual distribution inspection is stronger evidence than packaging configuration alone.

Definite non-Product implementation leakage into a Product distribution is governed by `PTSIP-PKG-001`.

## Phase 15 — Conformance Evaluation

Conformance combines project declaration with observed dependency, artifact, lifecycle, snapshot, and coverage evidence against applicable PTSIP rules.

Completed outcomes are only:

```text
CONFORMANT
NON_CONFORMANT
INCOMPLETE
```

A zero-finding report does not prove `CONFORMANT` when a blocking evidence gap can hide an applicable mandatory rule.

Decision Authority is not a conformance oracle. A synchronized authority/profile pair can still describe a non-conformant repository.

## Phase 16 — Structural remediation

For an established mandatory violation, change the architecture that causes the violation rather than weakening the rule.

Possible remediation patterns include:

- remove a forbidden dependency;
- split mixed lifecycle responsibilities;
- correct explicit project-owned classification;
- extract a valid `NEUTRAL_CONTRACT`;
- isolate Product packaging from non-Product implementation;
- separate development, delivery, and operations environments;
- fix stale or incomplete explicit relationships/associated-artifact anchors.

Migration/debt acknowledgement does not waive a real violation.

## Phase 17 — Tool 0.3.5 migration boundary

Do not blindly transform Tool `0.3.5`:

```text
TOOLCHAIN -> DEVELOPMENT_TOOLING
```

The legacy category may contain multiple Tool `0.3.6` lifecycle responsibilities.

Tool `0.3.6.1` owns the assisted migration continuation:

```text
facts
    -> candidate discovery
    -> normalized evidence + provenance
    -> Tool 0.3.5 legacy reader
    -> migration analysis
    -> target proposals
    -> owner preview / confirmation
    -> safe apply
```

Until that work is entered and implemented, migrate manually through explicit project decisions using Tool `0.3.6` canonical concepts.

## Optional VPMS adoption

VPMS is optional. PTSIP classification, conformance, adoption, authority, and decision behavior must remain usable when VPMS is absent.

PTSIP asks:

```text
Who owns this project responsibility across its lifecycle?
```

VPMS asks:

```text
Why does this Verification Case exist, and what does it protect?
```

The two axes remain independent. The current VPMS compatibility purpose set may remain:

```text
PRODUCT
TOOLCHAIN
```

VPMS `TOOLCHAIN` is not a canonical Tool `0.3.6` PTSIP classification.

A reusable verification implementation may be PTSIP `DEVELOPMENT_TOOLING` while a Verification Case has VPMS purpose `PRODUCT`.

Use VPMS purpose metadata to select verification; do not derive purpose solely from file paths or test frameworks.

VPMS PASS != PTSIP CONFORMANT. PTSIP CONFORMANT != functional verification PASS.

## Migration principle

Optimize for **ownership correctness, durable project intent, lifecycle separation, decision consistency, evidence integrity, artifact truth, and independent lifecycle evolution** rather than minimizing the number of changed files or maximizing automatic classification.
