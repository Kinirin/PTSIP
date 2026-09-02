# 0.4.0 — CORE Decisions and Non-goals

> **Status:** RESOLVED / PRE-WU-00B FROZEN  
> **Classification:** `CORE / RELEASE-BLOCKING`  
> **Parent:** `planning/0.4.0.md`  
> **Controlling freeze:** `planning/0.4.0/Pre-WU-00B-core-contract-freeze.md`  
> **Optional planning:** `planning/0.4.0-op/`

## 1. Purpose

This document now records the closed CORE decision queue for Tool `0.4.0`.

All release-blocking open decisions previously listed here were resolved by owner-approved Pre-WU-00B on 2026-09-02. The exact controlling contracts are in `Pre-WU-00B-core-contract-freeze.md`.

No item in `planning/0.4.0-op/` becomes CORE without the Pre-WU-00A Promotion Review procedure.

## 2. Frozen CORE non-goals

Tool `0.4.0` does not require or authorize:

- operationalizing every PTSIP rule;
- making AI mandatory runtime infrastructure;
- allowing AI, confidence, heuristics, history, repository evidence, or candidate dispositions to decide project architecture intent;
- treating a coverage gap as an accepted remediation decision;
- expanding `ptsip.yaml` with confidence, provider, remediation-history, optimization, or execution-authorization fields merely for Tool convenience;
- creating a giant secondary policy DSL;
- self-authorizing Specification changes;
- using historical remediation choices as current authority;
- optimizing the smallest textual diff over architecture correctness;
- presenting semantically equivalent physical edit orders as owner architecture choices;
- claiming global semantic uniqueness without a closed declared remediation-family model;
- treating `TOOL_CAPABILITY_GAP` as owner ambiguity;
- treating an externally knowable technical fact as owner intent;
- weakening exact-SHA/full qualification semantics;
- completing OPTIONAL / NON-BLOCKING work before 0.4.0 may release.

## 3. Required 0.3.7 inheritance

0.4.0 inherits rather than duplicates or weakens:

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

Generic remediation reuses neutral contracts where semantically valid and keeps migration-specific state machines separate.

## 4. Resolved Pre-WU-00B decisions

```text
D01 = A  Packaging-centered representative rules
D02 = A  Independent ptsip-remediation Product responsibility
D03 = A  Explicit semantic types + two-axis authorization
D04 = A  Explicit-apply conservative mutation authorization
D05 = A  Fresh Solve after owner intent, external fact, or stale-state drift
D06 = A  Explicit release acceptance matrix
D07 = A  One remediation verification responsibility + one Test Mode
D08 = A  Vertical responsibility WU sequence
```

The exact machine-readable/semantic contracts are controlled by the Pre-WU-00B freeze document rather than duplicated here.

## 5. Issue #31 reconciliation

Issue #31 exposed a real consumer-facing gap around `component-ownership:unassigned-relevant-files`.

The frozen CORE invariant is:

```text
Coverage Gap
    ≠ Remediation Candidate
    ≠ Accepted Architecture Decision
```

Therefore `UNASSIGNED` remains a coverage fact. Tool `0.4.0` must provide an agent-consumable, non-authoritative route through remediation disposition candidates and semantic resolution rather than encouraging automatic Project Profile assignment.

The exact UX surface (`triage`, `clarify --from-gap`, structured `next_actions`, or equivalent) is an implementation choice as long as the authority boundary and release acceptance proof are satisfied.

## 6. Frozen ownership and verification direction

```text
ptsip-remediation
    PRODUCT / IMPLEMENTATION
    src/ptsip/remediation/**

ptsip-remediation-verification
    PRODUCT / VERIFICATION
    tests/ptsip/remediation/**
        ↓ component_ref
Test Mode: ptsip-remediation
```

Internal module layers do not automatically become independent PTSIP components or Test Modes.

## 7. Frozen outcome and authorization separation

Canonical semantic outcomes:

```text
DETERMINISTIC
OWNER_INTENT_REQUIRED
EXTERNAL_FACT_REQUIRED
UNSATISFIABLE
TOOL_CAPABILITY_GAP
```

Mutation impact:

```text
MECHANICAL_REVERSIBLE
STRUCTURAL_SEMANTIC_PRESERVING
ARCHITECTURE_SEMANTIC
DESTRUCTIVE
```

Mutation authorization:

```text
AUTHORIZED
OWNER_CONFIRMATION_REQUIRED
NOT_AUTHORIZED
```

The impact and authorization axes remain separate.

## 8. Next CORE planning boundary

Pre-WU-00B is complete. The next action is not free-form implementation.

```text
create WU-01 planning document
    ↓
freeze exact WU-01 scope / entry baseline / non-goals / verification
    ↓
execute WU-01
```

The approved high-level WU sequence is WU-01 through WU-06 as defined by `Pre-WU-00B-core-contract-freeze.md`.
