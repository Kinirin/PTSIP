# PTSIP Tool 0.4.0 — CORE Planning Map

> **Status:** CORE CONTRACT FROZEN / WU PLANNING READY  
> **Integration branch:** `dev/0.4.0`  
> **Integration-branch baseline:** `f2fbbb5175eafde0ff8a3b7c5e7ca31a8224189d`  
> **Top-level plan:** `planning/0.4.0.md`  
> **Release-boundary authority:** `Pre-WU-00A-release-boundary-classification.md`  
> **Core contract authority:** `Pre-WU-00B-core-contract-freeze.md`

## 1. Directory responsibility

`planning/0.4.0/` is the planning namespace for:

```text
CORE / RELEASE-BLOCKING
```

The parallel namespace is:

```text
planning/0.4.0-op/
    = OPTIONAL / NON-BLOCKING
```

Directory placement follows classification; it does not create classification.

```text
UNCLASSIFIED
    ↓
Pre-WU-00A classification review
    ↓
CORE
    → planning/0.4.0/

OPTIONAL
    → planning/0.4.0-op/
```

## 2. Core document map

| Document | Primary responsibility | State |
| --- | --- | --- |
| `Pre-WU-00A-release-boundary-classification.md` | release-dependency intake and `CORE / OPTIONAL / DEFERRED` boundary | **COMPLETE / APPROVED** |
| `Pre-WU-00B-core-contract-freeze.md` | exact 0.4.0 remediation contracts, release acceptance, ownership, verification, WU decomposition | **COMPLETE / CORE CONTRACT FROZEN** |
| `01-authority-and-remediation-domain.md` | Evidence/fact/constraint/authority/decision separation and typed remediation domain | CORE design reference |
| `02-solution-space-and-planning.md` | candidate generation, elimination, cardinality, semantic planning, planner priorities | CORE design reference |
| `03-operational-rule-registry.md` | Specification-to-executable-rule bridge and supported remediation-family model | CORE design reference |
| `04-escalation-and-advisory.md` | Escalation Proof, owner-intent and external-fact boundaries | CORE boundary reference |
| `05-authority-gate-and-safe-apply.md` | mutation authorization, exact snapshot, stale-state rejection, apply/verify | CORE design reference |
| `06-verification-and-test-mode-strategy.md` | verification responsibility, Test Mode inheritance, selective CI and full exact-SHA qualification | CORE verification reference |
| `07-development-branch-and-wu-strategy.md` | `dev/0.4.0` integration and WU/isolation rules | CORE development-process reference |
| `08-representative-rule-families.md` | representative deterministic/owner-intent/unsatisfiable/capability-gap proof | CORE representative coverage reference |
| `09-core-open-decisions-and-non-goals.md` | resolved CORE decision queue and non-goals | **RESOLVED BY PRE-WU-00B** |

Optional planning index:

```text
planning/0.4.0-op/README.md
```

## 3. Planning sequence

```text
Top-level 0.4.0 direction
        ↓
Pre-WU-00A release-boundary classification
        = COMPLETE
        ↓
Pre-WU-00B Core Contract Freeze
        = COMPLETE
        ↓
WU-01 planning document
        ↓
WU-01 implementation
        ↓
WU-02 ... WU-06
        ↓
full release acceptance + exact-SHA qualification
```

The architecture dependency remains:

```text
Authority/domain model
        ↓
Operational rule contract
        ↓
Solution Space / semantic planning
        ↓
Escalation or Authority Gate
        ↓
Safe physical execution
        ↓
Postcondition / release verification
```

## 4. Frozen 0.4.0 CORE responsibilities introduced by Pre-WU-00B

Generic remediation is an independent Product responsibility:

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

The first mutation-capable representative family is packaging isolation under `PTSIP-PKG-001`.

Canonical outcomes:

```text
DETERMINISTIC
OWNER_INTENT_REQUIRED
EXTERNAL_FACT_REQUIRED
UNSATISFIABLE
TOOL_CAPABILITY_GAP
```

The approved Issue #31-derived safety invariant is:

```text
Coverage Gap
    ≠ Remediation Candidate
    ≠ Accepted Architecture Decision
```

Therefore `component-ownership:unassigned-relevant-files` cannot be flattened into automatic `ptsip.yaml` assignment work.

## 5. Approved CORE WU sequence

```text
WU-01 — Remediation Responsibility + Typed Foundation
WU-02 — Solution Space + Deterministic Rule / Coverage Routing Slice
WU-03 — Escalation + Fresh Solve Boundaries
WU-04 — Authority Gate + RepositoryChangePlan + Safe Apply
WU-05 — Boundary Outcomes + Remediation Verification Closure
WU-06 — Repository Dogfood + Full Release Qualification
```

A WU is a responsibility/completion boundary, not a mandatory Git sub-branch.

## 6. Promotion firewall

OPTIONAL work cannot become a release dependency by implementation accident.

```text
OPTIONAL
    ↓
Promotion Review
    ↓
prove which approved 0.4.0 objective / invariant / release proof fails without it
    ↓
project-owner approval
    ↓
CORE
```

A branch, WU number, implementation already written, usefulness, or experimental success is not enough.

## 7. Verification boundary

```text
Verification Responsibility
    ≠ Physical Test Location
    ≠ CI Execution Mode
```

and:

```text
Selective Test Mode success
    ≠ full repository qualification
```

Full release qualification remains complete repository regression + release contract + package/distribution + Product Artifact verification + exact-SHA self-hosted `tooling-test` success.

## 8. Current planning gate

```text
Pre-WU-00A — Release Boundary Classification
    = COMPLETE / APPROVED

Pre-WU-00B — Core Contract Freeze
    = COMPLETE / APPROVED

Next CORE planning action
    = create WU-01 planning document

Repository implementation
    = only under an approved WU execution boundary
```
