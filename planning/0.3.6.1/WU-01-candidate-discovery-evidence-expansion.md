# WU-01 — Candidate-Discovery Evidence Expansion

> **Status:** PRE-CREATED / LOCKED  
> **Target Tool:** `0.3.6.1`  
> **Roadmap predecessor:** Tool `0.3.6` release  
> **Planning relocation baseline:** `3529a32c862e1d43a91f732ced36358b4e13e1d9`  
> **Entry baseline:** not assigned; capture fresh released/main baseline when WU-01 is actually entered  
> **Required Specification family before release:** `0.3.6.1-draft`  
> **Successor:** WU-02 — evidence/provenance normalization and stronger boundary discovery

## 0. Purpose

Expand repository candidate discovery so PTSIP can observe a broader and more precise set of architecture-relevant evidence without converting evidence into project-owned architecture authority.

WU-01 starts from the Tool `0.3.6` rule that repository evidence may discover candidates, while the effective Responsibility Map and accepted project decisions remain authoritative.

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

### WU-01A — discovery inventory

Inventory every current discovery adapter and candidate type. Record duplicated logic and current blind spots.

### WU-01B — deterministic evidence expansion

Add narrowly justified candidate discovery rules with stable IDs and explicit evidence provenance.

### WU-01C — selector and coverage integration

Ensure new candidates participate in the shared selector/coverage mechanism rather than introducing a parallel matching system.

### WU-01D — ambiguity handling

Where evidence supports multiple plausible ownership interpretations, produce ambiguity for later clarification/migration analysis rather than guessing.

### WU-01E — repository fixtures and regression

Add representative repositories/fixtures for each new evidence source and verify deterministic discovery across repeated runs.

## 4. Non-goals

WU-01 does not authorize:

- lifecycle migration of Tool `0.3.5` profiles;
- automatic classification changes;
- template selection by discovery evidence;
- safe-apply writes;
- evidence/provenance schema redesign beyond what is strictly needed to represent newly discovered evidence;
- release workflow changes.

These belong to later Tool `0.3.6.1` work units, especially WU-02 through WU-06.

## 5. Expected deliverables

- documented discovery inventory;
- deterministic candidate IDs for each supported evidence source;
- explicit provenance for why each candidate exists;
- shared selector coverage integration;
- focused tests for positive, negative, duplicate, and ambiguous discovery cases;
- no new architecture write authority.

## 6. Completion gate

WU-01 is complete only when:

- expanded discovery is deterministic on a stable repository snapshot;
- every candidate can identify its evidence provenance;
- discovery does not silently assign lifecycle classification or ownership;
- duplicate observations converge on stable candidate identity where appropriate;
- selector/coverage behavior uses the canonical shared mechanism;
- ambiguous evidence remains explicit rather than guessed;
- WU-01 focused tests and participating existing discovery tests pass;
- WU-02 implementation has not been entered early.

## 7. Entry discipline

This document is pre-created for roadmap continuity after the Tool `0.3.6` planning split. It is not an implementation entry authority. WU-01 may become `ACTIVE` only after Tool `0.3.6` is released, the Tool `0.3.6.1` Specification binding is confirmed, and a fresh exact entry baseline is recorded.
