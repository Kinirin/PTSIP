# 0.4.0 — Development Branch, Core Path, and Optional Sub-branch Strategy

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

`dev/0.4.0` is the primary development and integration surface for the approved Tool `0.4.0` core.

The mandatory 0.4.0 work does **not** need to be decomposed into a chain of sub-branches merely because the architecture has several responsibilities. The core itself is already a large integration effort, and the default path may remain directly on `dev/0.4.0` with narrow commits, explicit WU scopes, and focused verification.

## 2. Two development lanes

The intended development model has two lanes.

### Lane A — 0.4.0 core / release-blocking work

```text
dev/0.4.0
    ↓
approved core work
    ↓
focused verification
    ↓
integration checkpoints
    ↓
0.4.0 release qualification
```

This is the main path. Work in this lane is included because the approved 0.4.0 release boundary requires it.

A WU in the core lane may be implemented directly on `dev/0.4.0`. WU boundaries are still used to control responsibility and verification, but they do not require one Git branch per WU.

### Lane B — optional / deferred / experimental work

Some previously discussed ideas do not need to block the core release. If such work is worth exploring during the 0.4.0 cycle, it may be isolated on a separate branch from the current `dev/0.4.0` baseline.

Conceptually:

```text
dev/0.4.0
    ├─ optional/<experimental-scope>
    ├─ optional/<deferred-extension>
    └─ optional/<independent-investigation>
```

These branches are **not a mandatory completion chain**.

An optional branch may:

- be completed and merged into `dev/0.4.0` if explicitly approved for the release;
- remain open while 0.4.0 core continues;
- be deferred to a later Tool version;
- be abandoned if the experiment is not useful.

None of those outcomes should block Tool `0.4.0` unless the project owner later explicitly promotes that scope into the release-blocking core.

## 3. Sub-branches are optional, not required

The earlier discussion did not establish a rule that every WU must receive a sub-branch.

The governing relationship is:

```text
planning topic
    ≠ implementation requirement

WU boundary
    ≠ mandatory Git branch

sub-branch exists
    ≠ release prerequisite
```

A sub-branch should be created only when isolation provides practical value, especially when the work is:

- optional for the current release;
- experimental;
- likely to be deferred;
- independently discardable;
- large enough to create unnecessary instability in the main integration lane;
- dependent on an unresolved architecture decision that should not block core progress.

## 4. Why optional isolation can still be useful

Optional sub-branches can provide:

- an exact experiment entry baseline;
- isolation from the release-blocking core;
- independent focused tests;
- easier abandonment without reverting core work;
- a clear boundary for work that may move to a later version;
- protection against an experimental capability accidentally becoming a hidden 0.4.0 dependency.

Their value is therefore **decoupling**, not mandatory sequential integration.

## 5. WU strategy

WU decomposition remains useful because 0.4.0 contains authority-sensitive responsibilities. A WU should still define, as applicable:

- one primary responsibility;
- exact entry baseline or integration context;
- explicit non-goals;
- authority boundary;
- input/output contracts;
- focused verification targets;
- completion conditions;
- any unresolved later architecture decisions that must remain outside the WU.

However, a WU plan must also identify whether it belongs to:

```text
CORE / RELEASE-BLOCKING
```

or:

```text
OPTIONAL / NON-BLOCKING
```

This classification is a planning/release property. It must not be inferred from the file path or branch name.

## 6. Vertical-slice preference for core work

The core should still prefer meaningful vertical slices over one broad rewrite.

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

This is a responsibility and verification strategy. It does not imply that every layer must live on a separate branch.

## 7. Candidate implementation layers

The consolidated discussion identified these layers as architecture/decomposition candidates:

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

These are design responsibilities, not an approved one-to-one WU list and not a one-to-one branch list.

Some responsibilities may be required by the final 0.4.0 core; others may be implemented only to the extent necessary for the approved representative framework. Experimental extensions can remain outside the release boundary.

## 8. Integration and release rule

For core work directly on `dev/0.4.0`, meaningful integration checkpoints should satisfy:

```text
focused tests pass
    ↓
relevant Test Mode(s) pass
    ↓
architecture/profile ownership remains valid
    ↓
approved core postconditions are satisfied
```

Full exact-SHA qualification should remain a major checkpoint/release boundary rather than the feedback loop for every small edit.

For optional sub-branches, successful focused verification only demonstrates that the optional work is internally viable. It does **not** make that branch a release requirement. Merge into `dev/0.4.0` remains a separate explicit decision.

## 9. Planning-document structure

The planning hierarchy is:

```text
planning/0.4.0.md
    consolidated direction / release-boundary discussion

planning/0.4.0/*.md
    topic-specific planning references

planning/0.4.0/WU-xx-*.md
    work-unit plans when a specific WU needs a durable plan
```

This hierarchy is for readability and responsibility tracking only.

It must **not** be interpreted as:

```text
all documents
    ↓
all WUs
    ↓
all sub-branches
    ↓
all must complete
    ↓
0.4.0 can release
```

That interpretation is explicitly rejected.

## 10. Optional-work candidates

The discussions already contain examples of work that can remain optional/non-blocking unless explicitly promoted into core, including:

- concrete AI advisory provider integrations beyond a provider-neutral boundary;
- autonomous LLM architect experiments;
- natural-language Specification synthesis;
- remediation-history optimization storage;
- other future-facing convenience capabilities not required to prove the approved remediation core.

These are natural candidates for optional sub-branches if explored during the 0.4.0 cycle.

## 11. Release dependency rule

A piece of work blocks Tool `0.4.0` only when the approved release plan explicitly says it is part of the 0.4.0 core.

The following are never sufficient by themselves to make work release-blocking:

```text
it has a planning document
it has a WU number
it has a sub-branch
it was discussed during the 0.4.0 cycle
it would be useful eventually
```

Release dependency must be explicit, not inferred.

## 12. Current branch policy

At this planning stage:

```text
dev/0.4.0
    = active integration branch for 0.4.0 core

optional sub-branches
    = create only when a specific optional/deferred scope is selected for exploration
```

No collection of pre-created WU sub-branches is required. The repository should create such branches only when there is an actual optional scope worth isolating.
