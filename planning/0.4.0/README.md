# PTSIP Tool 0.4.0 — CORE Planning Map

> **Status:** DRAFT / CORE PLANNING INDEX  
> **Integration branch:** `dev/0.4.0`  
> **Integration-branch baseline:** `f2fbbb5175eafde0ff8a3b7c5e7ca31a8224189d`  
> **Top-level plan:** `planning/0.4.0.md`  
> **Release-boundary authority:** `Pre-WU-00A-release-boundary-classification.md`  
> **Current planning gate:** `Pre-WU-00B-core-contract-freeze.md`

## 1. Directory responsibility

`planning/0.4.0/` is the planning namespace for work classified as:

```text
CORE / RELEASE-BLOCKING
```

Session documents, Pre-WUs, WUs, contract freezes, and release-blocking design decisions belong here only after their responsibility has been classified through the approved Pre-WU-00A release-boundary procedure.

The parallel namespace is:

```text
planning/0.4.0-op/
    = OPTIONAL / NON-BLOCKING planning and session documents
```

Deferred/out-of-release ideas may also be retained there as non-blocking planning context when useful, but they do not become 0.4.0 implementation work merely by being documented.

The physical directory is a routing consequence of classification, not the authority that creates classification:

```text
UNCLASSIFIED
    ↓
Pre-WU-00A classification review
    ↓
CORE / RELEASE-BLOCKING
    → planning/0.4.0/

OPTIONAL / NON-BLOCKING
    → planning/0.4.0-op/
```

A file must never become CORE merely because it was accidentally placed in `planning/0.4.0/`.

## 2. Core document map

| Document | Primary responsibility | Release relationship |
| --- | --- | --- |
| `Pre-WU-00A-release-boundary-classification.md` | approved release-dependency intake procedure and initial `CORE / OPTIONAL / DEFERRED` boundary | **completed prerequisite for Core Contract Freeze** |
| `Pre-WU-00B-core-contract-freeze.md` | owner review and freeze of representative rules, ownership, typed contracts, authorization, Fresh Solve, release proof, verification, and WU decomposition | **ACTIVE / owner decision review; implementation remains unauthorized** |
| `01-authority-and-remediation-domain.md` | Evidence/fact/constraint/authority/decision separation and typed remediation domain | CORE design reference |
| `02-solution-space-and-planning.md` | candidate generation, elimination, cardinality, semantic planning, planner priorities | CORE design reference |
| `03-operational-rule-registry.md` | Specification-to-executable-rule bridge and supported remediation-family model | CORE design reference |
| `04-escalation-and-advisory.md` | CORE Escalation Proof, owner-intent and external-fact boundaries; optional provider work is routed to `0.4.0-op` | CORE boundary reference |
| `05-authority-gate-and-safe-apply.md` | mutation authorization, irreversibility, exact snapshot, stale-state rejection, apply/verify | CORE design reference |
| `06-verification-and-test-mode-strategy.md` | release-blocking verification responsibility, Test Mode inheritance, selective CI, full exact-SHA qualification | CORE verification reference |
| `07-development-branch-and-wu-strategy.md` | `dev/0.4.0` CORE integration path and isolation rules | CORE development-process reference |
| `08-representative-rule-families.md` | deterministic / owner-intent / unsatisfiable / capability-gap representative release proof | CORE representative coverage planning |
| `09-core-open-decisions-and-non-goals.md` | unresolved release-blocking contracts, inherited safety limits, CORE non-goals | CORE decision queue consumed by Pre-WU-00B |

Optional planning index:

```text
planning/0.4.0-op/README.md
```

## 3. Reading and planning order

The release/planning intake order is:

```text
Top-level 0.4.0 direction
        ↓
Pre-WU-00A release-boundary classification
        ↓
Pre-WU-00B Core Contract Freeze
        ↓
approved CORE WU decomposition
        ↓
implementation
```

The architecture can be read in this dependency order:

```text
Authority/domain model
        ↓
Operational rule contract
        ↓
Solution-space / semantic planning
        ↓
Escalation or Authority Gate
        ↓
Safe physical execution
        ↓
Verification
```

This is an architecture-reading order, not permission to start implementation before the applicable planning gate is approved.

## 4. CORE integration-branch model

The repository has one 0.4.0 CORE integration branch:

```text
main
  ↓
dev/0.4.0
```

The default CORE implementation path is:

```text
dev/0.4.0
  ↓
approved CORE implementation
  ↓
focused verification
  ↓
meaningful checkpoint
  ↓
full exact-SHA release qualification
```

A WU responsibility boundary does not require a Git sub-branch:

```text
WU responsibility boundary
    ≠ mandatory sub-branch
```

Optional experimentation may use isolated branches, but those branches and their session documents are governed from `planning/0.4.0-op/` and cannot become release prerequisites without explicit promotion.

## 5. Promotion firewall

`OPTIONAL / NON-BLOCKING` work cannot silently enter this directory or the release dependency graph.

Required promotion sequence:

```text
OPTIONAL
    ↓
Promotion Review
    ↓
prove which approved 0.4.0 objective / invariant / release proof fails without it
    ↓
project-owner approval
    ↓
reclassify as CORE
    ↓
create or move the controlling CORE planning document here
```

Implementation already written, branch existence, usefulness, or successful experiments are not sufficient promotion reasons.

## 6. Planning drift rule

When a topic document, top-level plan, optional document, or implementation appears to conflict with the approved release boundary, do not silently choose one. Reconcile the conflict explicitly.

For release dependency, Pre-WU-00A remains controlling unless a later explicit owner-approved classification decision supersedes it.

## 7. Current planning gate

```text
Pre-WU-00A — Release Boundary Classification
    = COMPLETE

Directory routing policy
    = CORE → planning/0.4.0
      OPTIONAL → planning/0.4.0-op

Pre-WU-00B — Core Contract Freeze
    = ACTIVE / OWNER DECISION REVIEW
    = D01–D08 pending approval

0.4.0 remediation implementation
    = NOT YET AUTHORIZED
```

Pre-WU-00B may be declared complete only after its eight decisions are owner-approved and the open-decision source is reconciled. Only then may release-blocking implementation WU documents be created.
