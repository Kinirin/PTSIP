# ADR-0004: Pilot-Driven Evidence, Artifact, and Conformance Semantics

- **Status:** Proposed for the next `0.2.0-draft` normative snapshot
- **Date:** 2026-08-09
- **Specification family:** `0.2.0-draft`
- **Baseline normative revision:** `895e12d27230af2bb99ad17a96e8df8ef41bc3e0`
- **Planning baseline:** `planning/PTSIP-0.2.0-DRAFT-NEXT-NORMATIVE-SNAPSHOT-PLAN.md`

## Context

Two materially different Consumer Pilot exercises exposed gaps that cannot be solved only by adding repository-specific adapters.

The `turbo-system` Pilot stressed mixed-language dependency evidence, nested Product/Toolchain ownership, dynamic imports, build/release scripts, Product artifact generation, lifecycle triggers, exception semantics, and evidence completeness.

The Simple Connection Pilot feedback stressed JavaScript/TypeScript and npm evidence, explicit same-plane SDK dependency constraints, separation of profile validation from repository conformance evaluation, Electron/package artifact boundaries, stable diagnostics, external evidence, and future profile composition.

The repeated problem is not merely missing language support. PTSIP needs a stable adapter-independent pipeline:

```text
Declaration
    +
Observed / imported evidence
    ↓
Coverage determination
    ↓
Deterministic rule evaluation
    ↓
Stable diagnostics
    ↓
Conformance outcome
```

## Decision

### 1. Preserve exactly three architectural classifications

PTSIP continues to define only:

- `PRODUCT`;
- `TOOLCHAIN`;
- `NEUTRAL_CONTRACT`.

Dependency targets outside project-owned PTSIP scope are represented with evidence-node scope/type such as `PROJECT_COMPONENT`, `EXTERNAL_DEPENDENCY`, `PLATFORM`, or `UNRESOLVED_TARGET`. These are not architectural classifications.

### 2. Add an objective coherent-component boundary obligation

A broad component boundary is insufficient when one classification/purpose/lifecycle statement cannot coherently describe the included material.

A component MUST be split, or an explicit architecture decision MUST justify keeping it unified, when material evidence would otherwise require contradictory ownership conclusions because of differences such as:

- Product shipping responsibility;
- executable purpose;
- release ownership;
- compatibility ownership;
- build/manifest ownership; or
- Product versus Toolchain lifecycle responsibility.

This does not require one physical directory per component.

### 3. Standardize dependency evidence semantics before adding more adapters

Automated dependency evidence used for conformance records:

- relationship type;
- lifecycle phase or phases;
- resolution state;
- evidence provenance;
- source/target identity when resolvable; and
- uncertainty without guessing.

Canonical evidence provenance is:

- `DECLARED` — present in a manifest/profile/configuration declaration;
- `OBSERVED` — directly observed from repository/artifact/runtime/static evidence;
- `INFERRED` — derived from bounded deterministic or review reasoning and identified as such.

Canonical node scope is separate from classification.

Multiple relationship types or lifecycle phases MAY describe one real relationship when supported by distinct evidence. Implementations MUST NOT force one guessed phase merely to make evaluation possible.

### 4. Make evidence sufficiency rule-relative

A strict PTSIP conformance result cannot be derived merely from zero detected violations.

A gap is **blocking** when it can conceal whether an applicable normative `MUST`/`MUST NOT` rule is satisfied. A blocking gap produces an `INCOMPLETE` result unless a definite violation already proves `NON_CONFORMANT`.

A gap is **non-blocking** when it cannot materially affect the evaluated mandatory rule set. A global unresolved-count or coverage-percentage threshold is not normative.

### 5. Separate Product Artifact owner from producer

A Product Artifact is classified by what is shipped/distributed and which lifecycle owns that artifact, not by which component produced it.

A Toolchain component MAY generate/package a Product Artifact without becoming Product-owned.

Artifact evidence used for packaging conformance identifies, as applicable:

- artifact identity;
- architectural owner/classification;
- producer component;
- artifact format/type;
- contained component/path evidence;
- derivation/generation relationship;
- shipping/distribution scope; and
- evidence provenance.

### 6. Mandatory-rule violations require remediation; PTSIP defines no waiver

A PTSIP `MUST`/`MUST NOT` violation is an architecture result, not an approval workflow branch. Once sufficient evidence establishes the violation, conformance is `NON_CONFORMANT` until the architecture is changed to satisfy the rule and reevaluated.

PTSIP therefore does not define an approved-exception or waiver mechanism that can transform an active mandatory-rule violation into conformance. Projects remain free to track debt, approvals, owners, review conditions, and target migrations in repository- or organization-specific governance, but those records are outside the PTSIP conformance decision.

The earlier `PTSIP-EXC-001` rule is retired/superseded in the new immutable snapshot rather than silently repurposed. Historical snapshots that contained it remain immutable and interpretable by revision.

### 7. Separate Profile Validation from Conformance Evaluation

**Profile Validation** determines whether a declaration is syntactically and semantically well-formed: schema validity, component IDs, selector conflicts, references, policy declarations, and binding syntax.

**Conformance Evaluation** combines the valid declaration when required with observed evidence, artifact evidence, lifecycle evidence, evidence coverage and PTSIP rules.

Implementations MAY expose separate operations such as `validate` and `conform`; command names are not normative.

### 8. Standardize conformance outcomes

An evaluation outcome is one of:

- `CONFORMANT` — applicable evidence is sufficient and no applicable mandatory violation is established;
- `NON_CONFORMANT` — sufficient evidence establishes at least one applicable `MUST`/`MUST NOT` violation;
- `INCOMPLETE` — no definite violation is sufficient to settle the result, but blocking evidence, unresolved ownership, unsupported relevant analysis, or unstable snapshot prevents a conformance claim.

`NOT_EVALUATED` MAY be used as a tooling execution state but is not a conformance outcome.

A definite mandatory violation remains `NON_CONFORMANT` even if additional unrelated coverage gaps exist; those gaps are reported separately.

### 9. Make ownership declaration forms exclusive

A reference PTSIP Project Profile uses exactly one ownership-declaration mode:

1. `boundaries` for uniform root ownership; or
2. `components` for nested/mixed/file-level ownership.

A profile MUST NOT provide both simultaneously. `components` already supports broad selectors plus more-specific overrides and therefore does not require a second overlay ownership system.

### 10. Allow optional stricter component dependency constraints

A project MAY declare stricter component-to-component dependency policy, including same-plane constraints such as default deny plus explicit allows.

Such policy is project-specific and does not become a universal PTSIP topology requirement.

Project-specific allow rules MUST NOT authorize a dependency forbidden by a universal PTSIP rule.

Initial reference semantics use `default`, `allow`, and `deny` relationships between declared component IDs. Explicit allow/deny conflicts are invalid profile declarations. More detailed edge-type/phase constraints may be added after the shared edge/phase vocabulary is exercised further.

### 11. Define stable diagnostic identity

Automated conformance/evidence tooling uses a versioned diagnostic contract that distinguishes:

- unique diagnostic/finding instance identity; and
- stable PTSIP rule identity.

One normative rule may produce multiple diagnostic instances. Diagnostics include evidence references and sufficient source/target/provenance information for CI and agent consumers to reproduce or review the finding.

### 12. Clarify lifecycle trigger versus Product release coupling

A CI/workflow trigger caused by a Toolchain-only change is not by itself proof of a lifecycle violation.

Lifecycle coupling is evaluated from whether a Toolchain-only change requires or causes Product artifact change, Product version/release decision, Product publication, or Product compatibility obligation when no release-relevant Product artifact changed.

### 13. Clarify Neutral Contract neutrality

A Neutral Contract does not require a fixed minimum count of current Product and Toolchain consumers.

Neutrality is determined by non-executable/non-owning contract semantics, lifecycle independence from plane-specific executable implementation, and absence of hidden runtime implementation. Consumer evidence informs classification but consumer count alone does not determine it.

### 14. Published Tool 0.2.0 remains bound to its existing snapshot

This normative snapshot does not retroactively change published Tool `0.2.0`, which remains bound to `895e12d27230af2bb99ad17a96e8df8ef41bc3e0`.

Canonical specification/schema assets may advance in this PR while Tool 0.2.0 embedded resources remain intentionally unchanged. A later Tool release must explicitly bind the new merge revision it implements.

## Consequences

### Positive

- Multiple language/package adapters can emit interoperable evidence instead of ecosystem-specific semantics.
- Zero findings can no longer be confused with conformance when relevant coverage is missing.
- Packaging isolation can be evaluated against produced artifacts without misclassifying build producers.
- Profile syntax validation no longer masquerades as repository conformance.
- Same-plane project policy is expressible without turning PTSIP into a generic mandatory dependency-topology language.
- Diagnostics can be consumed consistently by CI, coding agents, and external reporters.
- Mandatory-rule violations have one deterministic conformance path: `NON_CONFORMANT` until remediated.

### Costs

- The profile schema becomes stricter by making `boundaries` and `components` mutually exclusive.
- Existing experimental profiles that declare PTSIP exceptions must migrate to remediation-oriented profiles without waiver semantics.
- Conformance implementations must model coverage per applicable rule rather than rely on simple pass/fail or global percentages.
- Artifact and diagnostic contracts add interoperability surface that future Tool versions must implement carefully.

## Compatibility

The family label remains `0.2.0-draft`. These changes are permitted under the draft-family revision policy but include `NORMATIVE_ADDITION`, `SCHEMA_CHANGE`, `CONFORMANCE_CHANGE`, and limited `NORMATIVE_BREAKING` semantics for experimental profiles/claims.

The next merge commit becomes a new immutable normative identity. Published Tool `0.2.0` is not rebound automatically.

## Affected rules and assets

Expected changes include:

- `PTSIP-CLS-001` clarification plus a coherent-component boundary rule;
- `PTSIP-DEP-002` clarification;
- dependency evidence vocabulary;
- `PTSIP-PKG-001` clarification plus Product Artifact semantics;
- `PTSIP-LCY-001` clarification;
- `PTSIP-EVD-*` evidence sufficiency/provenance additions;
- `PTSIP-EXC-001` strict-conformance semantics;
- Project Profile ownership-mode and optional component-policy schema;
- Conformance outcome and coverage semantics;
- stable diagnostic schema/registry vocabulary;
- terminology and registry synchronization.
