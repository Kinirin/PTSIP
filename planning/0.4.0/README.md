# PTSIP Tool 0.4.0 — Planning Document Map

> **Status:** DRAFT / PLANNING INDEX  
> **Integration branch:** `dev/0.4.0`  
> **Integration-branch baseline:** `f2fbbb5175eafde0ff8a3b7c5e7ca31a8224189d`  
> **Top-level plan:** `planning/0.4.0.md`

## Purpose

`planning/0.4.0.md` remains the consolidated top-level plan. It records the complete direction discussed before implementation authorization, but it is not intended to carry every implementation contract, work-unit boundary, branch rule, and verification detail by itself.

The documents in this directory split that consolidated direction by responsibility so that later WU approval and implementation can refer to a narrow source instead of reconstructing intent from a 1,000+ line plan or from chat history.

No document in this directory becomes Specification authority or Project Profile architecture authority merely by existing. They are development planning records under the authority boundaries already defined by PTSIP.

## Document map

| Document | Primary responsibility |
| --- | --- |
| `01-authority-and-remediation-domain.md` | Evidence/fact/constraint/authority/decision separation and typed remediation domain |
| `02-solution-space-and-planning.md` | candidate generation, elimination, cardinality, semantic planning, planner priorities |
| `03-operational-rule-registry.md` | Specification-to-executable-rule bridge and supported remediation-family model |
| `04-escalation-and-advisory.md` | Escalation Proof, owner-intent boundary, external facts, optional AI advisory |
| `05-authority-gate-and-safe-apply.md` | mutation authorization, irreversibility, exact snapshot, stale-state rejection, apply/verify |
| `06-verification-and-test-mode-strategy.md` | verification responsibility, Test Modes, selective CI, full exact-SHA qualification |
| `07-development-branch-and-wu-strategy.md` | `dev/0.4.0` integration branch, future WU sub-branches, vertical-slice decomposition |
| `08-representative-rule-families.md` | deterministic / owner-intent / unsatisfiable / capability-gap representative slices |
| `09-experimental-non-goals-and-open-decisions.md` | experimental ideas, explicit non-goals, unresolved planning decisions |

## Reading order

The intended dependency order is:

```text
Top-level 0.4.0 direction
        ↓
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

The development/WU strategy then maps those responsibility boundaries into implementation branches and focused tests.

## Branch model

The repository now has one 0.4.0 integration branch:

```text
main
  ↓
dev/0.4.0
```

Later implementation should not place every WU directly onto `dev/0.4.0` without isolation. Once a WU is explicitly approved, it should normally receive a short-lived sub-branch from the exact current integration-branch baseline, for example:

```text
dev/0.4.0
  ├─ dev/0.4.0/wu-01-<scope>
  ├─ dev/0.4.0/wu-02-<scope>
  └─ ...
```

The literal WU numbers and branch suffixes remain unapproved until the final WU decomposition is reviewed. Sub-branches are therefore a required development pattern, not yet a permission to invent implementation scopes.

## Planning rule

When a sub-document and the top-level plan appear to differ, do not silently choose one. Treat the difference as planning drift that must be reconciled explicitly before implementation. The top-level document remains the consolidated index of direction; narrow sub-documents should become the detailed implementation-planning source only after their scope is approved.
