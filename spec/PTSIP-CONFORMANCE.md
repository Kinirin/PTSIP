# PTSIP Conformance

**Version:** 0.3.4-draft

PTSIP conformance distinguishes Consumer Repository architecture conformance from PTSIP implementation capability requirements. A Decision Authority may coordinate architecture decisions, but it does not itself prove that a Consumer Repository conforms.

## 1. Consumer Repository conformance levels

### 1.1 PTSIP Core Conformant

A project is **PTSIP Core Conformant** when it satisfies all applicable Consumer Repository `MUST`/`MUST NOT` requirements in `PTSIP-SPEC.md`, including classification, coherent boundaries, Product-to-Toolchain runtime isolation, packaging isolation, independently resolvable build environments, lifecycle independence, and other applicable universal rules.

A project does not need to use distributed decision coordination merely to be Core Conformant.

### 1.2 PTSIP Enforced Conformant

A project is **PTSIP Enforced Conformant** when it is Core Conformant and additionally provides enough machine-readable declaration and reproducible evidence for automated enforcement, including:

- a PTSIP Project Profile or equivalent declaration;
- canonical Specification source/family and exact immutable revision for a mutable draft;
- automated dependency-boundary evaluation;
- Product Artifact evidence when required by `PTSIP-PKG-001` / `PTSIP-ART-001`;
- stable diagnostics with PTSIP rule IDs and evidence references;
- stable evidence tied to one repository snapshot;
- rule-relative evidence coverage sufficient for applicable mandatory rules; and
- CI or equivalent repeatable validation.

## 2. Profile Validation, authority reconciliation, and Conformance Evaluation

These operations are distinct.

### 2.1 Profile Validation

Profile Validation asks whether the selected declaration is structurally and semantically valid. It may check schema structure, specification binding, component IDs, selector conflicts, lifecycle facts, referenced components, and project-specific dependency policy.

A valid profile is not proof that observed dependencies, artifacts, build behavior, or lifecycle behavior conform.

### 2.2 Authority reconciliation

When distributed coordination is selected, authority reconciliation determines whether the relevant local Project Profile declaration is consistent with current coordinated authority state.

A local profile may be structurally valid but stale relative to a resolved distributed winner. Conversely, a resolved distributed winner may be perfectly synchronized with the profile while the repository remains architecturally non-conformant.

Authority reconciliation therefore MUST NOT be treated as Conformance Evaluation.

### 2.3 Conformance Evaluation

Conformance Evaluation combines, as applicable:

- validated project declaration;
- observed repository evidence;
- dependency evidence;
- Product Artifact evidence;
- build/lifecycle evidence;
- evidence coverage;
- snapshot integrity; and
- deterministic Consumer Repository PTSIP rules.

## 3. Conformance outcomes

A completed Consumer Repository Conformance Evaluation has exactly one of these outcomes:

### 3.1 `CONFORMANT`

`CONFORMANT` means evidence is sufficient for the applicable mandatory rule set, no applicable mandatory violation is established, and no blocking uncertainty remains.

An empty finding list alone is insufficient.

### 3.2 `NON_CONFORMANT`

`NON_CONFORMANT` means sufficient evidence establishes at least one applicable PTSIP `MUST`/`MUST NOT` violation.

A definite mandatory violation settles the result even if unrelated evidence gaps also exist. Those gaps remain reportable but do not erase the violation.

### 3.3 `INCOMPLETE`

`INCOMPLETE` means no definite violation has already settled the result, but one or more blocking conditions prevent a conformant conclusion. Examples include:

- evidence gaps capable of hiding an applicable violation;
- unresolved `UNKNOWN`, `CONFLICT`, or `INCOMPLETE` ownership relevant to a mandatory boundary;
- unsupported relevant language/build/package analysis;
- unresolved dynamic dependency relevant to a mandatory rule;
- missing required Product Artifact evidence;
- unstable/mixed repository snapshot;
- missing required Enforced Conformance binding; or
- missing durable component facts when a mandatory rule cannot be evaluated without them.

### 3.4 `NOT_EVALUATED`

`NOT_EVALUATED` MAY be used as a Tool execution state when evaluation was not attempted or could not start. It is not a PTSIP conformance outcome.

## 4. Required claim identity

A reproducible conformance claim SHOULD identify:

- canonical PTSIP Specification source;
- Specification family/version;
- exact immutable Specification revision;
- project commit/release;
- conformance level;
- conformance outcome;
- selected Project Profile path/configuration source;
- validation/evaluation status;
- evidence snapshot status;
- blocking/non-blocking evidence gaps; and
- diagnostic/evidence format versions where applicable.

For Enforced Conformance against `0.3.4-draft`, the immutable Specification revision is required.

## 5. Evidence sufficiency and coverage

Evidence coverage is rule-relative, not a universal percentage.

A gap is **blocking** when it can conceal whether an applicable mandatory PTSIP rule is satisfied. A blocking gap prevents `CONFORMANT` unless a definite violation already establishes `NON_CONFORMANT`.

A gap is **non-blocking** when it cannot materially change the result for the applicable mandatory rule set.

Unresolved evidence MUST NOT be converted into absence-of-violation proof.

## 6. Durable Project Profile facts

`0.3.4-draft` defines canonical component-level representation for these explicit adoption facts:

- `classification`;
- `purpose`;
- `shipped`;
- `runtime_required`;
- `lifecycle_owner`;
- `executable`.

A manually authored profile may omit optional facts when they are not needed for the claim, but if a missing fact prevents evaluation of an applicable mandatory rule, the result cannot be `CONFORMANT` on that evidence alone.

A write-enabled structured adoption/resolution workflow must preserve supplied facts losslessly under `PTSIP-ADP-001`. Failure of a Tool to preserve those facts is an implementation-conformance problem, not proof of Consumer Repository non-conformance.

Boundary-root shorthand may remain structurally valid, but its lower fact precision may make a strict claim `INCOMPLETE` when rule evaluation needs facts the shorthand cannot represent.

## 7. Declaration versus observed evidence

A Project Profile records intended architecture. It does not prove observed repository behavior.

Automated evaluation SHOULD distinguish:

- declared ownership/policy (`DECLARED`);
- direct repository/artifact/runtime facts (`OBSERVED`);
- bounded derived conclusions (`INFERRED`);
- agent/review decisions; and
- deterministic findings.

A contradiction between declaration and observation MUST NOT be hidden by treating the declaration as proof of compliance.

## 8. Product Artifact evidence

Packaging-isolation evaluation must distinguish artifact owner, artifact producer, contained components/paths or equivalent content evidence, derivation relationship, and shipping scope.

A Toolchain producer creating a Product Artifact is not itself a violation. The relevant question is whether the Product Artifact contains Toolchain-owned implementation or Toolchain-only dependencies contrary to `PTSIP-PKG-001`.

If required artifact-content evidence is unavailable, the packaging conclusion is `INCOMPLETE`, not conformant by absence of a detected inclusion.

## 9. Lifecycle evidence

A workflow/CI trigger is evidence of automation behavior, not by itself proof of Product release coupling.

Lifecycle evaluation SHOULD distinguish:

- workflow trigger;
- Product Artifact modification;
- Product version/release decision;
- Product publication/deployment; and
- Product compatibility obligation.

## 10. Dependency evidence and external nodes

Dependency evidence SHOULD preserve relationship type, lifecycle phase(s), resolution state, provenance, and evidence-node scope.

External libraries, standard-library modules, platforms, and unresolved targets are evidence nodes, not PTSIP architecture classifications.

A project-owned target MUST NOT be mislabeled external merely to avoid PTSIP evaluation.

## 11. Stable diagnostics

Automated diagnostics SHOULD conform to `ptsip-diagnostic/v1` and MUST distinguish diagnostic instance ID from stable PTSIP rule ID.

One rule may produce multiple diagnostic instances. Diagnostics SHOULD include outcome effect, severity, component endpoints where relevant, evidence IDs, message, and evaluator/provenance metadata.

## 12. Mandatory violations and remediation

PTSIP defines no mandatory-rule waiver that changes a real violation into conformance.

A confirmed violation remains `NON_CONFORMANT` until remediated and reevaluated. A project MAY describe itself as `PTSIP-adopting` or `PTSIP-transitioning`, but those labels do not replace the current conformance outcome.

## 13. External validator independence and non-intrusion

An external PTSIP validator is architecture-governance tooling and is not part of the Consumer Repository Product/Toolchain planes merely because it is installed in a developer environment or CI image.

If the project vendors/takes lifecycle ownership of the validator, that copy becomes subject to normal PTSIP classification.

External inspection/Pilot tooling SHOULD compare observable repository state before/after analysis and SHOULD place Tool-owned state outside the Consumer Repository by default.

## 14. Distributed coordination implementation conformance

`PTSIP-AUT-001` through `PTSIP-AUT-007` are requirements on a PTSIP implementation that claims distributed coordination semantics. They do not add Consumer Repository architecture requirements when distributed coordination is not used.

A distributed implementation conforms to this capability only when it satisfies at least:

- Decision Authority remains distinct from Project Profile;
- stable coordination-domain/decision identity;
- first-valid-resolution-wins with ordered conditional mutation;
- authority freshness at coordination-sensitive boundaries;
- non-mutating absence observation where supported;
- deterministic missing/equivalent/conflicting reconciliation;
- no silent overwrite of conflicting local declaration;
- stale application refusal;
- fail-closed behavior rather than isolated Local fallback; and
- separation of global decision state from clone-local projection/application state.

A failure of these requirements is an implementation capability failure. It does not by itself mean the Consumer Repository violates Product/Toolchain architecture rules.

## 15. Authority conflict and Consumer Repository claims

An `AUTHORITY_PROFILE_CONFLICT`-equivalent state means distributed authority and the selected local declaration disagree about architecture intent for the same coordinated scope.

While that conflict is unresolved, an architecture-sensitive operation requiring one unambiguous declaration MUST stop. A strict Enforced Conformance run that depends on the conflicted declaration cannot claim `CONFORMANT` until the declaration basis is unambiguous.

This does not authorize the Tool to choose either side silently. Explicit reconciliation is required.

## 16. False-positive handling

A validator suppression MUST NOT silently disable a PTSIP rule. A suppression MAY document evidence established as false positive; a real architecture violation remains `NON_CONFORMANT` until remediated.
