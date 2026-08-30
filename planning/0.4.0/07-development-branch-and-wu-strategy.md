# 0.4.0 — Development Branch and WU Strategy

> **Status:** DRAFT / DEVELOPMENT PROCESS SOURCE  
> **Parent:** `planning/0.4.0.md`

## 1. Integration branch

The 0.4.0 integration branch is:

```text
dev/0.4.0
```

It was created from the post-0.3.7 `main` baseline:

```text
f2fbbb5175eafde0ff8a3b7c5e7ca31a8224189d
```

`dev/0.4.0` is the integration surface for approved 0.4.0 work. It is not intended to become a single long-running implementation workspace where unrelated responsibilities are mixed directly.

## 2. Sub-branch rule

Once a WU is approved, implementation should normally occur on a short-lived sub-branch created from the exact current `dev/0.4.0` integration baseline.

Provisional pattern:

```text
dev/0.4.0
  ├─ dev/0.4.0/wu-01-<scope>
  ├─ dev/0.4.0/wu-02-<scope>
  ├─ dev/0.4.0/wu-03-<scope>
  └─ ...
```

The exact WU numbering and branch suffixes are not authorized merely by this document. A sub-branch should be created only after its responsibility, entry baseline, expected outputs, and verification scope are approved.

## 3. Why sub-branches are required

0.4.0 introduces several authority-sensitive layers. Mixing them into one broad branch would make it difficult to determine which change owns a regression or semantic decision.

Sub-branches provide:

- exact WU entry baseline;
- narrow implementation authority;
- independent focused tests;
- easier rollback/review;
- cleaner integration checkpoints;
- less accidental coupling between domain, rules, planner, authority, execution, and advisory work.

This follows the 0.3.7 principle that responsibility boundaries are more important than minimizing WU count.

## 4. Vertical-slice preference

WU decomposition should not simply map one branch per abstract class or module. Prefer narrow vertical slices that prove meaningful responsibility end-to-end.

Representative shape:

```text
repository state
    ↓
typed evidence/facts
    ↓
operational rule
    ↓
solution-space result
    ↓
authority outcome
    ↓
execution plan when authorized
    ↓
verification
```

Some foundational WUs may establish typed contracts before a full vertical slice, but later slices should consume those contracts rather than duplicate them.

## 5. Candidate implementation layers

The consolidated discussion identified these layers as decomposition candidates:

```text
Layer A — typed remediation domain model
Layer B — operational rule contract/registry
Layer C — rule evaluation + derived facts
Layer D — solution-space engine
Layer E — semantic remediation planner
Layer F — escalation proof / owner-intent boundary
Layer G — authority gate
Layer H — repository execution planner + safe apply
Layer I — postcondition/conformance verification
Layer J — optional advisory provider boundary
```

These layers are design responsibilities, not an approved one-to-one WU list.

## 6. Candidate WU decomposition criteria

A WU should have:

- one primary responsibility;
- exact entry baseline SHA;
- explicit non-goals;
- clear authority boundary;
- defined input/output contracts;
- focused verification targets;
- a completion condition that can be independently proven;
- no hidden requirement to decide a later architecture question.

If a WU cannot be explained without several unrelated authority domains, it is probably too broad.

## 7. Integration into `dev/0.4.0`

A WU should be integrated only after:

```text
focused tests pass
    ↓
relevant Test Mode(s) pass
    ↓
architecture/profile ownership remains valid
    ↓
approved WU postconditions are satisfied
```

Full exact-SHA qualification should be reserved for meaningful integration checkpoints rather than every tiny WU edit, while still remaining mandatory at the release/major checkpoint boundary.

## 8. Planning-document structure

The intended planning hierarchy is:

```text
planning/0.4.0.md
    consolidated direction / index

planning/0.4.0/*.md
    responsibility-specific design sources

planning/0.4.0/WU-xx-*.md
    later approved work-unit plans
```

A WU file should link back to the responsibility-specific sub-documents it implements.

## 9. Branch authority discipline

A WU branch authorizes only the approved WU scope. It does not implicitly authorize:

- unrelated Project Profile changes;
- Specification changes;
- new remediation semantics outside the WU;
- new AI authority;
- widening mutation permission;
- release publication.

Those remain separate explicit decisions.

## 10. Likely first-slice direction

The earlier discussion suggests that the first meaningful implementation should demonstrate the framework rather than attempt universal remediation coverage.

A likely sequence is:

```text
foundation contracts
    ↓
one deterministic representative rule
    ↓
one owner-intent representative rule
    ↓
unsatisfiable/capability-gap boundaries
    ↓
safe apply + verification
```

The exact first rule family and final WU order remain pending explicit approval.
