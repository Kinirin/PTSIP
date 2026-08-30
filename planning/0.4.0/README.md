# PTSIP Tool 0.4.0 — Planning Document Map

> **Status:** DRAFT / PLANNING INDEX  
> **Integration branch:** `dev/0.4.0`  
> **Integration-branch baseline:** `f2fbbb5175eafde0ff8a3b7c5e7ca31a8224189d`  
> **Top-level plan:** `planning/0.4.0.md`

## Purpose

`planning/0.4.0.md` remains the consolidated top-level plan. It records the complete direction discussed before implementation authorization, but it is not intended to carry every detailed architecture note, optional experiment, implementation boundary, and verification concern by itself.

The documents in this directory split that planning material by topic so later work can refer to a narrow source instead of reconstructing intent from one 1,000+ line file or from chat history.

**Document separation does not create implementation dependency.** These files are parallel planning references, not a mandatory sequential session chain. The existence of a sub-document does not mean that its entire subject must be implemented before Tool `0.4.0` can be released.

No document in this directory becomes Specification authority or Project Profile architecture authority merely by existing. They are development planning records under the authority boundaries already defined by PTSIP.

## Document map

| Document | Primary responsibility | Release relationship |
| --- | --- | --- |
| `01-authority-and-remediation-domain.md` | Evidence/fact/constraint/authority/decision separation and typed remediation domain | core design reference |
| `02-solution-space-and-planning.md` | candidate generation, elimination, cardinality, semantic planning, planner priorities | core design reference |
| `03-operational-rule-registry.md` | Specification-to-executable-rule bridge and supported remediation-family model | core design reference |
| `04-escalation-and-advisory.md` | Escalation Proof, owner-intent boundary, external facts, optional AI advisory | mixed: core escalation boundary + optional advisory |
| `05-authority-gate-and-safe-apply.md` | mutation authorization, irreversibility, exact snapshot, stale-state rejection, apply/verify | core design reference |
| `06-verification-and-test-mode-strategy.md` | verification responsibility, Test Modes, selective CI, full exact-SHA qualification | development/release verification reference |
| `07-development-branch-and-wu-strategy.md` | `dev/0.4.0` core path and optional/deferred sub-branch policy | development process reference |
| `08-representative-rule-families.md` | deterministic / owner-intent / unsatisfiable / capability-gap representative slices | representative coverage planning |
| `09-experimental-non-goals-and-open-decisions.md` | experimental ideas, explicit non-goals, unresolved planning decisions | non-blocking unless explicitly promoted into core |

## Reading order

The architecture can be read in this dependency order:

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

This is an **architecture-reading order**, not a branch-completion order and not a release checklist. A topic can remain deferred if the approved 0.4.0 release boundary does not require it.

## Development branch model

The repository has one 0.4.0 integration branch:

```text
main
  ↓
dev/0.4.0
```

The default 0.4.0 implementation path is the integration branch itself:

```text
dev/0.4.0
  ↓
approved core implementation
  ↓
focused verification
  ↓
release qualification
```

Because the mandatory 0.4.0 core is already large, creating a sub-branch for every WU or every planning document is **not required**.

Sub-branches are optional isolation surfaces for work that is useful to explore independently and may or may not be included in the `0.4.0` release, for example:

```text
dev/0.4.0
  ├─ optional/<experimental-or-deferred-scope>
  └─ optional/<non-blocking-extension>
```

A sub-branch may later be merged into `dev/0.4.0` if its scope is approved and ready. It may also remain unmerged, be deferred to a later version, or be abandoned without blocking `0.4.0` release.

**No optional sub-branch is a release prerequisite merely because it exists.**

## WU relationship

WU decomposition is still useful for defining responsibility, authorization, and verification boundaries, but WU boundaries and Git branch boundaries are not required to be one-to-one.

```text
WU responsibility boundary
    ≠ mandatory sub-branch
```

A core WU may be implemented directly on `dev/0.4.0` when that is the approved development path. A sub-branch is chosen only when isolation has practical value, especially for optional, experimental, deferred, or independently discardable work.

## Planning rule

When a sub-document and the top-level plan appear to differ, do not silently choose one. Treat the difference as planning drift that must be reconciled explicitly before implementation.

Most importantly, do not infer release dependency from document layout or branch layout. Whether an item blocks Tool `0.4.0` is determined only by the explicitly approved 0.4.0 core/release boundary, not by the existence of a planning file or sub-branch.
