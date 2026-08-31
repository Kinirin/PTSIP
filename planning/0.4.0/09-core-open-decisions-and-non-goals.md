# 0.4.0 — CORE Open Decisions and Non-goals

> **Status:** DRAFT / CORE DECISION QUEUE  
> **Classification:** `CORE / RELEASE-BLOCKING`  
> **Parent:** `planning/0.4.0.md`  
> **Optional planning:** `planning/0.4.0-op/`

## 1. Purpose

This document contains only unresolved contracts, safety boundaries, and non-goals that affect the approved 0.4.0 CORE.

Experimental, advisory-provider, workflow-convenience, history, and other non-blocking work belongs under `planning/0.4.0-op/` and must not be pulled into this CORE decision queue without an explicit Promotion Review.

## 2. CORE non-goals

The release-blocking remediation core does not require or authorize:

- operationalizing every PTSIP rule in Tool 0.4.0;
- making AI mandatory runtime infrastructure;
- allowing AI, confidence, heuristics, history, or repository evidence to decide project architecture intent;
- expanding `ptsip.yaml` with confidence, provider, remediation-history, or optimization fields merely for Tool convenience;
- creating a giant secondary policy DSL;
- self-authorizing Specification changes;
- using historical remediation choices as current authority;
- optimizing the smallest textual diff over architecture correctness;
- presenting semantically equivalent physical edit orders as owner architecture choices;
- claiming global semantic uniqueness without a closed declared remediation-family model;
- treating `TOOL_CAPABILITY_GAP` as owner ambiguity;
- treating an externally knowable technical fact as owner intent;
- weakening exact-SHA/full qualification semantics inherited before 0.4.0;
- completing any OPTIONAL / NON-BLOCKING work before 0.4.0 can release.

## 3. Required 0.3.7 inheritance

0.4.0 must inherit rather than duplicate or weaken these established foundations:

```text
Tool Version independence
Project Profile Contract Version independence
Project Profile Instance Revision independence
Typed Specification Binding
Evidence / provenance normalization
Source compatibility boundaries
Exact snapshot validation
Safe apply / recovery behavior
Architecture authority separation
```

Generic remediation should reuse these contracts where semantically valid. Migration-specific state machines or assumptions must not be copied into generic remediation merely because they already exist.

The Core Contract Freeze must identify reusable neutral primitives and migration-specific responsibilities explicitly.

## 4. CORE open decisions for Pre-WU-00B

The following decisions remain release-blocking planning work and must be resolved before implementation WUs are authorized.

### Decision 1 — exact representative operational rule families

Select the initial rule families that prove the 0.4.0 framework without implying universal Specification coverage.

The selected set must exercise, at minimum, the approved representative outcome boundaries and must be small enough that the modeled remediation families can be stated honestly.

### Decision 2 — package, module, and Project Profile ownership

Fix the responsibility boundary for generic remediation code and verification.

This includes deciding whether a durable independent `ptsip-remediation` Product responsibility is declared or whether another explicit ownership arrangement is used. New paths must not rely on accidental wildcard ownership.

### Decision 3 — canonical typed state and diagnostics contracts

Freeze the machine-readable shapes and canonical vocabulary for:

```text
Evidence
DerivedFact
Normative Constraint
Architecture Authority / Project Intent
SemanticCandidate
ResolutionOutcome
SemanticRemediationPlan
EscalationProof
Mutation authorization
RepositoryChangePlan
Postcondition result
```

The freeze must prevent evidence, external facts, architecture intent, and mutation permission from collapsing into one generic authority object.

### Decision 4 — authorization defaults and irreversibility classes

Define the exact authorization treatment for reversible mechanical changes, structural semantic-preserving changes, architecture-semantic changes, and destructive/irreversible changes.

Semantic determinacy must remain separate from permission to mutate.

### Decision 5 — Fresh Solve resume lifecycle

Freeze the lifecycle for owner-intent and external-fact suspension/resume:

```text
Fresh Solve
    ↓
missing intent or fact
    ↓
materialize owner authority OR provenance-bound external evidence
    ↓
Fresh Solve again
    ↓
Authority Gate
```

No supplied answer may bypass a fresh evaluation against current Specification, authority, and repository state.

### Decision 6 — exact release acceptance matrix

Define the release proof that permits Tool 0.4.0 to claim the generic remediation framework is complete enough for release.

The matrix must cover the canonical outcome taxonomy, safe refusal, authorization, stale-state rejection, postcondition verification, inherited 0.3.7 contracts, package/distribution checks, and exact-SHA self-hosted qualification.

### Decision 7 — CORE verification responsibility and Test Mode coverage

Decide durable verification ownership before introducing any new Test Mode.

Correct direction:

```text
durable verification responsibility
    ↓
Project Profile declaration
    ↓
component_ref
    ↓
Test Mode when execution selection benefits from one
```

Physical test directories and watch paths remain implementation/execution evidence, not architecture authority.

### Decision 8 — final CORE WU decomposition

After Decisions 1–7 are frozen, decompose the release-blocking work into narrow vertical slices with explicit completion and verification boundaries.

Layer names from the consolidated plan are design responsibilities, not an automatically approved one-layer-per-WU schedule.

## 5. Tool capability gap is a CORE safety result

`TOOL_CAPABILITY_GAP` is not a failure to hide. It prevents partial operationalization from being mistaken for semantic determinacy.

It does not automatically mean:

```text
architecture ambiguous
owner must decide
repository unsatisfiable
```

It means the current Tool cannot complete the required modeled reasoning/remediation path with its declared capability.

## 6. Project Profile surface stability

Before adding a new `ptsip.yaml` field, the CORE design must answer:

```text
Does the project itself need to declare new truth, intent, or authority?
```

If no, internal execution, optimization, provider, history, or Tool-capability state should remain outside Project Profile authority.

## 7. Automation debt boundary

0.4.0 should reduce repeated manual PTSIP reasoning by coding agents, but automation cannot cross authority boundaries.

A deterministic supported rule should encode deterministic reasoning in executable Tool behavior. An unsupported semantic guess must remain an explicit gap or escalation instead of being hidden behind automation.

## 8. Relationship to OPTIONAL planning

Non-blocking work is tracked under:

```text
planning/0.4.0-op/
```

If an optional experiment reveals a missing CORE contract, the experiment itself does not become CORE automatically. The missing contract must be separately reviewed under Pre-WU-00A promotion rules and, if approved, represented by a controlling document in `planning/0.4.0/`.
