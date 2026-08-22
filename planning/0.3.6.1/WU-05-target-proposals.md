# WU-05 — Split / Relationship / Associated-Artifact Target Proposals

> **Status:** PRE-CREATED / LOCKED  
> **Target Tool:** `0.3.6.1`  
> **Roadmap predecessor:** WU-04 — lifecycle migration analyzer  
> **Planning relocation baseline:** `3529a32c862e1d43a91f732ced36358b4e13e1d9`  
> **Entry baseline:** not assigned; capture fresh branch HEAD on actual entry  
> **Successor:** WU-06 — preview + project-owner confirmation + safe apply

## 0. Purpose

Transform WU-04 migration findings into explicit, reviewable target-architecture proposals for Tool `0.3.6.1` without applying those proposals to the project.

WU-05 is the proposal-construction stage. It may describe a candidate target Responsibility Map delta, but project-owned architecture authority remains with the project owner and the WU-06 confirmation/apply flow.

## 1. Proposal categories

WU-05 may propose:

- lifecycle classification for a target component;
- component split when one historical component spans distinct lifecycle ownership;
- component merge only when evidence and project semantics support a single target responsibility;
- typed relationships between proposed components;
- associated artifacts anchored to a component instead of inventing unnecessary components;
- selector/include changes needed to express the proposed responsibility boundary;
- explicit unresolved questions where a safe target cannot be proposed.

## 2. Proposal contract

Every proposal must be:

```text
source-aware
+ evidence-linked
+ deterministic for the same inputs
+ explainable
+ non-mutating
+ individually reviewable
```

A proposal is not an accepted decision and MUST NOT be treated as write authority.

## 3. Work tracks

### WU-05A — target-delta model
Define stable proposal identities and the difference between source state, suggested target state, and unresolved questions.

### WU-05B — split proposals
Represent one-to-many lifecycle separation without losing traceability to the legacy component.

### WU-05C — relationship proposals
Propose typed directed semantics only where target components require explicit project-owned relationships.

### WU-05D — associated-artifact proposals
Prefer associated-artifact representation for files/contracts/docs that do not justify an independent lifecycle component.

### WU-05E — rationale and alternatives
Where multiple plausible targets exist, preserve alternatives or require owner clarification rather than selecting one silently.

## 4. Non-goals

WU-05 does not collect owner confirmation, mutate profiles, perform CAS writes, or declare a proposed classification authoritative.

## 5. Completion gate

WU-05 is complete when migration findings can be converted into deterministic non-mutating target proposals; splits, relationships, and associated-artifact cases are represented explicitly; every proposal is traceable to evidence and source state; ambiguous cases remain explicit; and the proposal contract is ready for WU-06 preview/confirmation.

## 6. Entry discipline

Pre-created roadmap document only. Actual WU-05 entry requires WU-04 completion and a fresh exact branch HEAD recorded here.
