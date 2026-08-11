# PTSIP Conformance

**Version:** 0.2.0-draft

> **0.3.4-draft alignment note:** `spec-v0.3.4-draft` is a published proposed Specification design record. This document remains part of the active `0.2.0-draft` normative baseline until a coherent migration updates all affected normative and machine-readable assets at one immutable revision. Section 15 records the proposed distributed-authority/conformance separation without silently activating that draft.

PTSIP conformance is defined so that a human reviewer or automated implementation can make a reproducible claim about a project without confusing declaration validity, evidence collection, decision coordination, or absence of detected findings with actual conformance.

## 1. Conformance levels

### 1.1 PTSIP Core Conformant

A project is **PTSIP Core Conformant** when it satisfies all applicable normative `MUST`/`MUST NOT` requirements in `PTSIP-SPEC.md`, including:

- SDK/component classification;
- coherent component boundaries;
- Product-to-Toolchain runtime dependency prohibition;
- packaging isolation;
- independently resolvable build environments;
- lifecycle ownership separation;
- evidence obligations required to support the claim;
- Consumer Repository Non-Intrusion for external PTSIP tooling; and
- remediation of established mandatory-rule violations before a conformant claim.

A project does not need to add PTSIP-specific documentation or tooling directories merely to claim Core Conformance.

Core Conformance MAY be established through a reproducible human review, automated tooling, or a combination, but the evidence used for the claim must be sufficient for the applicable mandatory rules.

### 1.2 PTSIP Enforced Conformant

A project is **PTSIP Enforced Conformant** when it is Core Conformant and additionally provides:

- a machine-readable PTSIP Project Profile or equivalent declaration;
- a Specification Binding identifying canonical source, draft family/version, and immutable specification revision when the specification is mutable;
- automated dependency-boundary validation;
- product artifact inspection or an equivalent packaging check sufficient for `PTSIP-PKG-001` / `PTSIP-ART-001`;
- stable machine-readable diagnostics that report PTSIP rule IDs and evidence references;
- stable evidence tied to one repository snapshot;
- rule-relative evidence coverage sufficient for the mandatory rules being claimed; and
- CI or equivalent repeatable validation.

## 2. Profile Validation is not Conformance Evaluation

A PTSIP implementation SHOULD distinguish these operations explicitly.

### 2.1 Profile Validation

Profile Validation asks whether the project declaration is structurally and semantically well-formed.

It may validate:

- JSON/YAML schema structure;
- specification-binding syntax;
- component IDs;
- ownership declaration mode;
- selector conflicts;
- referenced component existence; and
- project-specific dependency-policy consistency.

A valid profile is **not** proof that repository dependencies, artifacts, build behavior, or lifecycle behavior conform to the declaration or to PTSIP.

### 2.2 Conformance Evaluation

Conformance Evaluation combines, as applicable:

- the validated project declaration;
- observed repository evidence;
- dependency evidence;
- artifact/packaging evidence;
- lifecycle/release evidence;
- evidence coverage; and
- deterministic PTSIP rule evaluation.

A tool MAY expose separate operations such as `validate` and `conform`. Those command names are informative; the semantic separation is normative.

## 3. Conformance outcomes

A completed PTSIP conformance evaluation has one of these outcomes:

### 3.1 `CONFORMANT`

`CONFORMANT` means:

- evidence is sufficient for the applicable mandatory rule set;
- no applicable `MUST`/`MUST NOT` violation is established; and
- no blocking uncertainty remains that could conceal an applicable mandatory-rule result.

An empty finding list alone is insufficient to produce `CONFORMANT`.

### 3.2 `NON_CONFORMANT`

`NON_CONFORMANT` means sufficient evidence establishes at least one applicable PTSIP `MUST`/`MUST NOT` violation.

A definite mandatory violation is sufficient to establish `NON_CONFORMANT` even if additional unrelated evidence gaps also exist. Those gaps remain reportable and may limit additional conclusions, but they do not erase the established violation.

### 3.3 `INCOMPLETE`

`INCOMPLETE` means no definite violation is sufficient to settle the result, but one or more blocking conditions prevent a conformant result, including for example:

- evidence gaps that can conceal an applicable mandatory-rule violation;
- unresolved `UNKNOWN`, `CONFLICT`, or `INCOMPLETE` component ownership relevant to a mandatory boundary;
- unsupported language/build/package analysis that affects an applicable rule;
- unresolved dynamic dependency relevant to a mandatory rule;
- required Product Artifact evidence not inspected;
- unstable/mixed repository snapshot; or
- missing required Enforced Conformance declaration/binding evidence.

### 3.4 `NOT_EVALUATED`

`NOT_EVALUATED` MAY be used by tooling to indicate that conformance evaluation was not attempted or could not start. It is an execution/evaluation state, not a PTSIP conformance outcome.

## 4. Required claim identity

A conformance claim SHOULD identify:

- PTSIP canonical specification source;
- PTSIP specification version/family;
- exact immutable specification revision;
- project commit or release;
- conformance level (`Core` or `Enforced`);
- conformance outcome;
- project profile path or configuration source for Enforced Conformance;
- validation/evaluation result;
- evidence snapshot status;
- blocking and non-blocking evidence gaps; and
- diagnostic/evidence format version when automated tooling is used.

For Enforced Conformance against a mutable draft family, the exact immutable specification revision is required even when the generic Project Profile schema permits a profile to exist without one.

Example:

```text
PTSIP Conformance Level: Enforced
PTSIP Conformance Outcome: CONFORMANT
Specification source: https://github.com/kwaksinwoo01/ptsip
Specification version: 0.2.0-draft
Specification revision: <commit-or-release>
Project revision: <commit>
Profile: <project-defined location>
Evidence snapshot: STABLE
Blocking coverage gaps: none
```

## 5. Evidence sufficiency and coverage

Evidence coverage is **rule-relative**, not a universal percentage.

A coverage gap is **blocking** when it can conceal whether an applicable mandatory PTSIP rule is satisfied. Blocking coverage prevents `CONFORMANT` and produces `INCOMPLETE` unless a definite violation already establishes `NON_CONFORMANT`.

A coverage gap is **non-blocking** when it cannot materially change the result for the applicable mandatory rule set.

Examples:

- inability to resolve a Product runtime dynamic import target may block a `PTSIP-DEP-001` conclusion;
- lack of a parser for unrelated documentation does not by itself block dependency conformance;
- lack of Product Artifact content evidence blocks a strict packaging-isolation conclusion when `PTSIP-PKG-001` applies.

A validator SHOULD report inaccessible files, parser failures, unresolved dynamic dependencies, unsupported languages/adapters, artifact-inspection gaps, and other evidence gaps together with whether each gap is blocking and which rule(s) it may affect.

Unresolved evidence MUST NOT be silently converted into an absence-of-violation claim.

## 6. Active violations, remediation, and unresolved decisions

When sufficient evidence establishes a violation of an applicable PTSIP `MUST` or `MUST NOT` rule, the conformance outcome is `NON_CONFORMANT`.

PTSIP does not define a mandatory-rule waiver that changes this result. A repository may track architectural debt or migration approval in an external governance system, but that record does not erase the observed violation and is not an input that can produce `CONFORMANT`.

The path back to conformance is remediation of the violating architecture followed by reevaluation on a stable evidence snapshot.

A project MAY describe itself as `PTSIP-adopting` or `PTSIP-transitioning` while remediation is in progress, but adoption/transition state is not a conformance outcome.

A component decision status of `UNKNOWN`, `CONFLICT`, or `INCOMPLETE` is not itself a fourth classification. Such unresolved status is blocking when it can conceal the result of an applicable mandatory PTSIP boundary.

## 7. Declaration versus observed evidence

A Project Profile records intended ownership and policy. It does not prove that dependencies, artifacts, or build behavior comply with the declaration.

Automated conformance evidence SHOULD distinguish:

- declared component ownership/policy (`DECLARED` evidence);
- direct repository/artifact/runtime facts (`OBSERVED` evidence);
- bounded derived conclusions (`INFERRED` evidence);
- agent or heuristic candidate decisions; and
- deterministic rule findings.

A contradiction between declaration and observed behavior MUST NOT be hidden by treating the declaration as authoritative proof of compliance.

Imported external evidence is analysis input. Unless a future specification explicitly grants authority to a specific evidence source, imported evidence MUST NOT silently override contradictory native observed evidence or project declaration.

## 8. Product Artifact evidence

A packaging-isolation conclusion must distinguish:

- artifact architectural owner;
- artifact producer;
- contained components/paths or equivalent content evidence;
- derivation/generation relationship; and
- shipping/distribution scope.

A Toolchain producer creating a Product Artifact is not itself a violation. The relevant question is whether the resulting Product Artifact contains Toolchain-owned implementation or Toolchain-only dependencies contrary to `PTSIP-PKG-001`.

If Product Artifact evidence is required for the claim but no equivalent packaging/content evidence is available, the packaging conclusion is `INCOMPLETE`, not conformant by absence of detected inclusion.

## 9. Lifecycle evidence

A workflow or CI trigger is evidence of automation behavior, not by itself proof of Product release coupling.

Lifecycle evaluation SHOULD distinguish at least:

- pipeline/workflow trigger;
- Product artifact modification;
- Product version/release decision;
- Product publication/deployment; and
- Product compatibility obligation.

A Toolchain-only change that triggers a Product workflow but does not require or cause release-relevant Product changes is not automatically a `PTSIP-LCY-001` violation.

## 10. Dependency evidence and external nodes

Automated dependency evidence SHOULD preserve relationship type, lifecycle phase(s), resolution state, provenance, and evidence-graph scope.

External libraries, standard-library modules, platforms, and unresolved targets are evidence nodes and are not additional PTSIP architectural classifications.

A project-owned target must not be mislabeled external merely to avoid PTSIP ownership or boundary evaluation.

## 11. Stable diagnostics

Automated conformance diagnostics SHOULD conform to the versioned reference contract `ptsip-diagnostic/v1` defined by `schemas/ptsip-diagnostic.schema.json`.

A diagnostic MUST distinguish:

- unique diagnostic instance ID; and
- stable PTSIP rule ID.

One PTSIP rule may produce multiple diagnostic instances.

Diagnostics SHOULD include:

- outcome effect;
- severity;
- source and target component when applicable;
- evidence IDs;
- message; and
- evaluator/provenance metadata.

An empty diagnostic collection MUST NOT be interpreted as conformance when evaluation was blocked or coverage was incomplete.

## 12. External validator independence

An external PTSIP validator is architecture-governance tooling and is not part of the Consumer Repository's Product or project-owned Toolchain plane merely because it is installed in a developer virtual environment, user-level tool environment, CI image, or equivalent external environment.

If a project vendors or takes lifecycle ownership of the validator, that copy becomes subject to normal PTSIP classification.

A Product build MUST NOT require a PTSIP validator at runtime.

## 13. Non-intrusion evidence

For external PTSIP tooling, inspection and pilot validation SHOULD compare observable repository state before and after analysis rather than emit a constant assertion.

At minimum, a tool SHOULD identify which observation methods were used. Examples include repository revision, Git status/index state, tracked-content fingerprints, and untracked/ignored state.

A change observed during analysis does not by itself prove that PTSIP caused the change. The report SHOULD mark the evidence unstable or indeterminate and require rerun or investigation.

Tool-owned caches and pilot reports SHOULD be placed outside the Consumer Repository by default. A user-selected output path inside the repository is an explicit write and SHOULD be reported as such rather than described as non-intrusive.

## 14. False-positive handling

A validator suppression MUST NOT silently disable a PTSIP rule.

A suppression MAY be used only when review establishes that the reported evidence is a false positive and therefore does not actually establish the referenced rule violation. The suppression record SHOULD identify the evidence and rationale.

A real architecture violation MUST NOT be relabeled as a false positive or suppression; it remains `NON_CONFORMANT` until remediated.

## 15. Proposed 0.3.4-draft Decision Authority alignment

This section records the `spec-v0.3.4-draft` design relationship between distributed architecture-decision coordination and conformance. It is **not active 0.2.0-draft normative text** until the coherent `0.3.4-draft` migration is completed.

The proposed model keeps these responsibilities separate:

```text
Decision Authority
    -> which explicit architecture answer won for a coordinated scope

Project Profile
    -> durable project-owned architecture declaration for a repository/worktree

Conformance Evaluation
    -> declaration + observed evidence + artifacts + coverage + normative rules
```

A Decision Authority is therefore not a conformance oracle. A globally resolved architecture decision does not establish that repository dependencies, Product Artifacts, build behavior, or lifecycle behavior conform to PTSIP.

For an Enforced Conformance evaluation claiming a future `0.3.4-draft` binding, the selected Project Profile must provide one unambiguous declaration for the evaluated scope. If a relevant distributed authority winner and local declaration are in unresolved semantic conflict, the declaration basis is not stable enough for a strict conformant claim. The conflict must be explicitly reconciled before that scope can support `CONFORMANT`.

The proposed distributed coordination model distinguishes:

- **authority freshness** from evidence freshness;
- **global decision resolution** from clone-local Project Profile projection;
- **authority/profile semantic conflict** from declaration-versus-observed-evidence conflict; and
- **coordination failure** from a PTSIP architecture-rule violation.

These distinctions matter because their remedies differ:

```text
authority/profile conflict
    -> explicit architecture-declaration reconciliation

coordination unavailable
    -> fail the affected coordinated decision operation; do not create a second local winner

declaration vs observed architecture violation
    -> conformance remediation under the applicable PTSIP rule

blocking evidence gap
    -> collect sufficient evidence or remain INCOMPLETE
```

A future `0.3.4-draft` conformance claim must identify the immutable Specification revision that actually contains the coherent normative migration. The `spec-v0.3.4-draft` design release by itself is not that binding revision.
