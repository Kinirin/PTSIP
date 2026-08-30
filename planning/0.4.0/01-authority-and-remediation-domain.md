# 0.4.0 — Authority and Remediation Domain

> **Status:** DRAFT / DESIGN SOURCE  
> **Parent:** `planning/0.4.0.md`

## 1. Core authority separation

0.4.0 must preserve a hard separation among repository observation, deterministic derivation, normative rules, project-owned intent, and executable authorization.

```text
Evidence
    repository observation
        ↓
Derived Fact
    deterministic fact derived from evidence
        ↓
Normative Constraint
    Specification rule / executable obligation
        ↓
Architectural Intent
    project-owner authority
        ↓
Decision / Authorization
    explicit permitted target state or mutation authority
```

The governing invariant is:

```text
Evidence ≠ Derived Fact ≠ Normative Constraint ≠ Architecture Authority
```

Repository evidence must never silently become architecture authority.

## 2. Typed domain objective

The implementation should make authority-boundary violations structurally difficult. The exact class names remain subject to WU design, but the semantic types should remain distinct.

Candidate conceptual types:

```text
Evidence
DerivedFact
Constraint
Authority
Decision
SemanticCandidate
SemanticRemediationPlan
AuthorityDecision
RepositoryChangePlan
EscalationProof
```

The implementation must not provide shortcuts equivalent to:

```text
Authority.from_evidence(...)
```

Evidence may establish facts. Facts may make normative constraints applicable. Constraints may eliminate illegal solutions. None of those steps may invent lifecycle purpose, project intent, or permission to mutate.

## 3. Tool-owned responsibility

PTSIP should automatically own the reasoning that is deterministic under current Specification, current repository state, and current explicit authority:

- collect repository evidence;
- normalize provenance and derived facts;
- determine rule applicability;
- evaluate definitive violations;
- enumerate modeled legal remediation candidates;
- eliminate candidates that violate hard constraints or explicit authority;
- reduce provably equivalent or dominated candidates;
- generate mechanical profile/schema transformations when target semantics are already decided;
- verify exact pre-state and post-state conditions;
- re-run conformance after authorized application.

The objective is to reduce the amount of Specification reasoning that a coding agent must reconstruct manually.

## 4. Project-owner responsibility

Project-owner authority is required only when a remaining decision is genuinely semantic and cannot be derived from current authority, for example:

- lifecycle purpose is genuinely ambiguous;
- multiple architecture-valid semantic target states remain after all provable eliminations;
- destructive or architecture-semantic mutation requires confirmation;
- current declarations conflict and no existing authority determines precedence.

The owner should not be asked to repeat evidence collection or rule interpretation that PTSIP can perform itself.

## 5. Outcome taxonomy

The remediation domain must distinguish at least these outcomes:

```text
DETERMINISTIC
OWNER_INTENT_REQUIRED
EXTERNAL_FACT_REQUIRED
UNSATISFIABLE
TOOL_CAPABILITY_GAP
```

Meanings:

- `DETERMINISTIC`: exactly one semantic target remains among the Tool's modeled legal remediation families. This does not itself authorize mutation.
- `OWNER_INTENT_REQUIRED`: more than one legal semantic target remains because current project authority does not determine the semantic choice.
- `EXTERNAL_FACT_REQUIRED`: a required technical/environmental fact is missing and should be acquired as evidence rather than as project intent.
- `UNSATISFIABLE`: no legal semantic target remains under current constraints and authority.
- `TOOL_CAPABILITY_GAP`: a legal resolution may exist, but the Tool does not model the required remediation family or operational rule.

These outcomes must not collapse into a generic "uncertain" state.

## 6. Fresh Solve

Every remediation attempt must start from:

```text
Current Specification
+ Current Authority
+ Current Repository State
        ↓
Fresh Solve
```

History is not authority. Previous remediation choices, old repository states, or optimization databases must not eliminate current candidates.

History may later be used as a non-authoritative optimization hint only after the current solution space has been constructed independently.

## 7. Architecture preservation

Coarse discovery evidence cannot override more specific validated Project Profile declarations.

```text
coarse discovery candidate
    = evidence / candidate

explicit validated Project Profile partition
    = architecture authority
```

This rule generalizes the earlier test-architecture correction where a broad `tests/**` discovery candidate could not collapse already-declared verification responsibility partitions.

## 8. Determinacy versus authorization

These are separate dimensions:

```text
Determinacy ≠ Mutation Authorization
```

A remediation can be mathematically/deterministically unique among modeled legal candidates while still requiring project-owner authorization before any repository mutation.

That separation is consumed by the Authority Gate rather than hidden in candidate selection.

## 9. Domain-level safety expectations

Any implementation slice touching this domain must verify that:

- evidence cannot be converted directly into authority;
- heuristic or AI confidence cannot create authority;
- unknown authority state fails closed;
- external facts remain evidence/facts unless explicitly materialized as owner intent;
- semantic target determination is inspectable;
- mutation authorization is separately inspectable;
- current explicit architecture is preserved over coarse discovery convenience.
