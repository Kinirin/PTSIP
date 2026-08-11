# PTSIP Tool 0.3.3 — Explicit Project Adoption Plan

> **Status:** COMPLETED / PARTIALLY SUPERSEDED  
> **Target Tool version:** `0.3.3`  
> **Original planning baseline:** `ba620456cd510cf1a056073647969b908697795b`  
> **Completed migration merge:** `2ab93a63a78391c64f3f715d415314e5f28e2d98`  
> **Bound Specification family:** `0.2.0-draft`  
> **Bound Specification revision:** `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`  
> **Superseding scope amendment:** [`PTSIP-TOOL-0.3.3-GITHUB-COORDINATED-AUTHORITY-AMENDMENT.md`](PTSIP-TOOL-0.3.3-GITHUB-COORDINATED-AUTHORITY-AMENDMENT.md)

## Document authority

This document records the **original Tool 0.3.3 adoption plan**. Its explicit-adoption, profile-path-consistency, validation-reuse, dry-run, and Project Profile projection requirements were implemented.

During implementation, the project identified a multi-environment coordination gap in the Local DecisionStore-only model and explicitly expanded Tool 0.3.3 to include a GitHub-coordinated authority. That later decision is authoritative where it differs from this original plan.

Therefore:

```text
Original adoption requirements
    -> this document

GitHub coordination / distributed authority changes
    -> GITHUB-COORDINATED-AUTHORITY-AMENDMENT.md

Implemented Tool 0.3.3 behavior
    -> current main source + releasenote/0.3.3.md
```

If this document conflicts with the scope amendment, **the scope amendment wins**.

## 1. Original goal

Tool 0.3.3 was created to close the first-adoption gap for existing Consumer Repositories:

```text
Component discovery
    -> explicit project-owner classification
    -> boundary declaration
    -> Project Profile validation
```

The architecture classifications remain exactly:

- `PRODUCT`
- `TOOLCHAIN`
- `NEUTRAL_CONTRACT`

PTSIP observes candidate scope from repository evidence. The project owner supplies architecture intent. The Tool must not automatically classify a candidate from directory names, package names, or anchors.

## 2. `ptsip adopt`

Tool 0.3.3 adds the explicit project-owner adoption command:

```powershell
ptsip adopt . `
  --component tools-turbo-sdk `
  --classification TOOLCHAIN `
  --purpose "Repository-local Turbo SDK tooling" `
  --shipped no `
  --runtime-required no `
  --lifecycle-owner DEVELOPMENT_TOOLING `
  --executable yes `
  --profile ptsip.yaml `
  --json
```

The default operation is a read-only dry-run. Mutation requires explicit `--apply`.

The implementation reuses the existing:

- `DecisionAnswer` model;
- `validate_answer()` semantic validation;
- Project Profile projection;
- schema/profile validation;
- concurrent profile-content guard;
- atomic Project Profile writer.

No parallel adoption-specific classification rule system is introduced.

## 3. Candidate and authority boundary

A selected candidate contributes observed facts such as:

- candidate ID;
- include selectors;
- anchors;
- evidence IDs.

The project owner contributes explicit architecture facts such as:

- classification;
- purpose;
- shipped state;
- runtime requirement;
- lifecycle owner;
- executable state.

The original plan distinguished `ptsip adopt` from `ptsip resolve`: adoption establishes or extends project-owned architecture, while resolve answers a pending coding-agent decision.

That distinction remains, but the original assumption that GitHub adoption would not participate in a repository-global authority was later superseded. For GitHub repositories, the final Tool 0.3.3 behavior is defined by the scope amendment.

## 4. Profile-path consistency

Tool 0.3.3 requires the same explicit profile location to be usable throughout the declaration and decision workflow.

The migration therefore added explicit profile support to clarification/gate analysis so these commands can operate consistently against the same Project Profile:

```text
adopt
resolve
validate
conform
clarify
gate
```

Repository-root `ptsip.yaml` remains the default.

## 5. Adoption transaction requirements

The adoption workflow preserves these original requirements:

### Dry-run

- discover and validate the selected candidate;
- capture repository evidence;
- load the selected Project Profile;
- validate supplied architecture facts;
- prepare and validate a projected profile;
- return structured adoption output;
- do not modify the Consumer Repository.

### `--apply`

- refuse stale repository evidence;
- refuse conflicting existing declarations;
- preserve concurrent profile-content protection;
- atomically write the validated profile;
- rerun profile validation;
- remain idempotent for equivalent declarations.

The result family is `ptsip-adoption/v1`, including outcomes such as `ADOPTION_PLAN`, `ADOPTED`, `ALREADY_DECLARED`, `CONFLICT`, `UNKNOWN_COMPONENT`, and `STALE_EVIDENCE`. Coordinated GitHub adoption may additionally return authority-related resolution outcomes defined by the amendment.

## 6. Preserved non-goals

The following original exclusions remain valid for Tool 0.3.3:

- global `gate --json` failure-envelope redesign;
- aggregate gate-precondition diagnostics;
- a new Specification revision;
- a fourth architecture classification;
- automatic/LLM ownership classification;
- directory-name-based ownership decisions;
- batch adoption manifests;
- topology migration semantic changes;
- conformance-rule changes;
- automatic reclassification of existing declarations.

The original exclusion of a new distributed authority **does not remain valid**. It was explicitly superseded by the GitHub-coordinated authority amendment.

## 7. Completion result

The original adoption plan is complete as part of Tool 0.3.3 migration merge:

```text
2ab93a63a78391c64f3f715d415314e5f28e2d98
```

The implemented migration includes:

- `ptsip adopt` dry-run / `--apply`;
- explicit profile-path consistency;
- deterministic semantic validation reuse;
- validated atomic profile projection;
- Local DecisionStore continuity;
- GitHub-coordinated authority and stale-clone reconciliation as defined by the amendment;
- Tool/package source identity `0.3.3` while retaining the bound `0.2.0-draft` Specification revision.

For the final implemented behavior, use `releasenote/0.3.3.md`, `reference/DECISION-CONTROL-PLANE.md`, `adoption/ADOPTION-GUIDE.md`, and the current `main` source as the implementation record.