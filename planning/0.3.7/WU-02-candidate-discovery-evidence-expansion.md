# WU-02 — Candidate-Discovery Evidence Expansion

> **Status:** ACTIVE  
> **Target Tool:** `0.3.7`  
> **Roadmap predecessor:** WU-01 — draft profile transition state model (`COMPLETE / FOCUSED TEST VERIFIED`)  
> **WU-02 exact entry baseline:** `6d364055532c592e8c6f778f5a145148e7f7e29a`  
> **Bound Specification at entry:** `0.3.7-draft @ b648d9e026f502b14481ba2d0606d9acc88a31fc`  
> **Successor:** WU-03 — evidence/provenance normalization

## 0. Purpose

Expand deterministic repository candidate discovery so PTSIP can observe a broader and more precise set of architecture-relevant evidence without converting evidence into project-owned architecture authority.

This work retains the cancelled Tool `0.3.6.1` WU-01 intent, but all evidence must now be attributable to the exact source-profile generation and repository snapshot being evaluated.

## 1. Core boundary

```text
repository observation
    -> candidate evidence
    -> candidate identity / selector evidence
    -> source-specific migration analysis

repository observation
    -X-> automatic lifecycle classification
    -X-> automatic architecture mutation
    -X-> automatic Required/Removal/Async obligation status without source evaluation
```

Discovery is descriptive. It does not own migration-completion semantics.

## 2. Initial scope

Candidate discovery should evaluate existing evidence classes and extend them only where the signal is deterministic and reviewable. Sources may include:

- manifests and package metadata;
- source/package roots;
- test roots and verification entrypoints;
- CI/workflow-invoked scripts;
- release/package assembly inputs;
- schema and neutral-contract groups;
- generated or embedded contract copies;
- operational/maintenance scripts where objective evidence exists;
- dependency edges identifying architecture-relevant boundaries without deciding ownership.

## 3. Generation-aware evidence identity

Equivalent repository observations may participate in more than one source migration during Sequential Work. Discovery must therefore preserve enough context to answer:

```text
what was observed?
where was it observed?
under which repository snapshot?
for which source-profile evaluation was it collected?
```

Source-generation context does not make evidence authoritative; it prevents evidence from being accidentally reused as if a previous source evaluation had already decided a later source's obligations.

## 4. Work tracks

### WU-02A — discovery inventory

Inventory every current discovery adapter and candidate type. Record duplicated logic and blind spots.

### WU-02B — deterministic evidence expansion

Add narrowly justified discovery rules with stable IDs and explicit evidence provenance.

### WU-02C — selector and coverage integration

Ensure new candidates participate in the shared selector/coverage mechanism instead of introducing a parallel matcher.

### WU-02D — ambiguity handling

Where evidence supports multiple plausible ownership interpretations, produce ambiguity for later analysis rather than guessing.

### WU-02E — generation/snapshot binding

Attach the exact transition/repository context necessary for WU-03/WU-05 without embedding migration decisions into discovery.

### WU-02F — repository fixtures and regression

Add representative fixtures for positive, negative, duplicate, ambiguous, and multi-generation discovery cases.

## 5. Non-goals

WU-02 does not authorize:

- creation or promotion of Temporary PTSIP Profile Files;
- lifecycle migration writes;
- Required/Removal/Async final categorization;
- template selection by evidence;
- target proposals;
- safe-apply writes;
- release workflow changes unrelated to actual verification needs.

## 6. Expected deliverables

- documented discovery inventory;
- deterministic candidate IDs;
- explicit evidence provenance;
- exact repository/source-generation context where needed;
- shared selector/coverage integration;
- focused tests for positive, negative, duplicate, ambiguous, and multi-generation cases;
- no new architecture write authority.

## 7. Completion gate

WU-02 is complete only when:

- expanded discovery is deterministic on a stable repository snapshot;
- every candidate can identify why and where it exists;
- evidence can be associated with the correct source migration without inheriting prior source decisions;
- duplicate observations converge on stable candidate identity where appropriate;
- ambiguity remains explicit;
- discovery does not silently assign lifecycle ownership or migration obligation category;
- focused and participating existing discovery tests pass;
- WU-03 has not been entered early.

## 8. Entry discipline

WU-02 entered on exact `dev/0.3.7` baseline `6d364055532c592e8c6f778f5a145148e7f7e29a` after WU-01 completion was recorded and the branch HEAD was freshly revalidated.

This ACTIVE state authorizes WU-02 implementation only. It does not authorize WU-03 implementation, lifecycle migration writes, Temporary PTSIP Profile promotion, or bypass of any project-owner decision gate.

## 9. Successor auto-entry authorization

The project owner has approved automatic successor-WU entry for the remainder of Tool `0.3.7` work.

After an active WU reaches its documented completion gate, the next WU MAY be changed from `PRE-CREATED / LOCKED` to `ACTIVE` without a separate approval message, provided that:

1. the predecessor completion is recorded;
2. a fresh exact `dev/0.3.7` HEAD is captured for the successor entry baseline;
3. no unresolved project-owner decision, architecture conflict, or explicit confirmation gate blocks entry;
4. only the successor entry state is automated — implementation beyond that WU's approved scope is not implicitly authorized by this rule.

If any blocking decision or conflict remains, the successor MUST stay locked until it is resolved.
