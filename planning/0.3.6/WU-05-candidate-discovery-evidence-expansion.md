# WU-05 — Candidate-Discovery Evidence Expansion

> **Status:** PRE-CREATED / LOCKED  
> **Roadmap predecessor:** WU-04 — complete effective-map pipeline, currently closing through WU-04I  
> **Planning baseline:** `f6e010c2c9519e894de79a91feef4fa7b46326e1`  
> **Entry baseline:** not assigned; must be captured from fresh branch HEAD when WU-05 is actually entered  
> **Bound Specification family:** `0.3.6-draft`  
> **Successor:** WU-06 — evidence/provenance normalization and stronger boundary discovery

## 0. Purpose

Expand repository candidate discovery so PTSIP can observe a broader and more precise set of architecture-relevant evidence without converting evidence into project-owned architecture authority.

WU-05 starts from the WU-04 rule that repository evidence may discover candidates, while the effective Responsibility Map and accepted project decisions remain authoritative.

## 1. Core boundary

```text
repository observation
    -> candidate evidence
    -> candidate identity / selector evidence
    -> clarification or migration analysis

repository observation
    -X-> automatic lifecycle classification
    -X-> automatic architecture mutation
```

Discovery MUST remain descriptive and provenance-bearing. It MUST NOT infer lifecycle ownership merely because a file is executable, packaged, tested, CI-invoked, or referenced by another component.

## 2. Initial scope

Candidate discovery should evaluate the evidence classes already present in the repository and extend them only where the signal is deterministic and reviewable. Candidate sources may include:

- manifests and package metadata;
- source/package roots;
- test roots and verification entrypoints;
- CI/workflow-invoked scripts;
- release/package assembly inputs;
- schema and neutral-contract groups;
- generated or embedded contract copies;
- operational/maintenance scripts where repository structure supplies objective evidence;
- dependency edges that identify architecture-relevant boundaries without deciding ownership.

## 3. Work tracks

### WU-05A — discovery inventory

Inventory every current discovery adapter and candidate type. Record duplicated logic and current blind spots.

### WU-05B — deterministic evidence expansion

Add narrowly justified candidate discovery rules with stable IDs and explicit evidence provenance.

### WU-05C — selector and coverage integration

Ensure new candidates participate in the shared selector/coverage mechanism rather than introducing a parallel matching system.

### WU-05D — ambiguity handling

Where evidence supports multiple plausible ownership interpretations, produce ambiguity for later clarification/migration analysis rather than guessing.

### WU-05E — repository fixtures and regression

Add representative repositories/fixtures for each new evidence source and verify deterministic discovery across repeated runs.

## 4. Non-goals

WU-05 does not authorize:

- lifecycle migration of Tool 0.3.5 profiles;
- automatic classification changes;
- template selection by discovery evidence;
- safe-apply writes;
- evidence/provenance schema redesign beyond what is strictly needed to represent newly discovered evidence;
- release workflow changes.

These belong to later work units, especially WU-06 through WU-10.

## 5. Expected deliverables

- documented discovery inventory;
- deterministic candidate IDs for each supported evidence source;
- explicit provenance for why each candidate exists;
- shared selector coverage integration;
- focused tests for positive, negative, duplicate, and ambiguous discovery cases;
- no new architecture write authority.

## 6. Completion gate

WU-05 is complete only when:

- expanded discovery is deterministic on a stable repository snapshot;
- every candidate can identify its evidence provenance;
- discovery does not silently assign lifecycle classification or ownership;
- duplicate observations converge on stable candidate identity where appropriate;
- selector/coverage behavior uses the canonical shared mechanism;
- ambiguous evidence remains explicit rather than guessed;
- WU-05 focused tests and participating existing discovery tests pass;
- WU-06 implementation has not been entered early.

## 7. Entry discipline

This document is pre-created for roadmap visibility by maintainer direction. Its `Planning baseline` is not an implementation entry SHA. When WU-04I closes and WU-05 is actually entered, update this document with a fresh exact branch HEAD and change status to `ACTIVE` before production work begins.
