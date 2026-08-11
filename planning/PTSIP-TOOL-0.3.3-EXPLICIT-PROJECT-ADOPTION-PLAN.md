# PTSIP Tool 0.3.3 — Explicit Project Adoption Plan

> **Status:** COMPLETED  
> **Target Tool version:** `0.3.3`  
> **Original planning baseline:** `ba620456cd510cf1a056073647969b908697795b`  
> **Completed migration merge:** `2ab93a63a78391c64f3f715d415314e5f28e2d98`  
> **Bound Specification family:** `0.2.0-draft`  
> **Bound Specification revision:** `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`  
> **Successor workstream:** [`PTSIP-TOOL-0.3.4-GITHUB-COORDINATED-AUTHORITY-PLAN.md`](PTSIP-TOOL-0.3.4-GITHUB-COORDINATED-AUTHORITY-PLAN.md)  
> **Publication policy:** Tool `0.3.3` is permanently source-only. No `tool-v0.3.3` tag, GitHub Release, or PyPI `0.3.3` publication will be created.

## 1. Document authority

This document is the completed Tool `0.3.3` plan.

Tool `0.3.3` owns the explicit first-adoption workstream:

```text
Component discovery
    -> explicit project-owner architecture facts
    -> reviewable adoption plan
    -> explicit application
    -> validated Project Profile
```

A GitHub-coordinated distributed authority was explored during the same source migration and an initial implementation was merged. The previously approved Tool `0.3.3` authority amendment is **withdrawn from the Tool 0.3.3 completion scope**.

The distributed-authority contract requires additional read-side freshness, existing-declaration reconciliation, conflict semantics, and application-state design. That work is transferred to Tool `0.3.4` as an independent release objective.

Therefore:

```text
Tool 0.3.3 completed contract
    -> explicit project adoption
    -> profile-path symmetry
    -> deterministic validation/profile projection reuse
    -> Local DecisionStore continuity

Experimental GitHub authority source already present on main
    -> precursor implementation only
    -> not a completed Tool 0.3.3 release contract
    -> audited/completed under Tool 0.3.4 planning
```

## 2. Explicit project adoption

Tool `0.3.3` adds the project-owner adoption command:

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

The default operation is a read-only dry-run. Repository mutation requires explicit `--apply`.

The Tool must not infer `PRODUCT`, `TOOLCHAIN`, or `NEUTRAL_CONTRACT` merely from directory names, package names, file names, or candidate anchors.

The architecture classifications remain exactly:

- `PRODUCT`
- `TOOLCHAIN`
- `NEUTRAL_CONTRACT`

## 3. Reused validation and projection boundary

Adoption reuses the existing:

- `DecisionAnswer` model;
- `validate_answer()` semantic validation;
- Project Profile projection;
- schema/profile validation;
- concurrent profile-content guard;
- atomic Project Profile writer.

No second adoption-specific architecture model or classification validator is introduced.

Candidate discovery contributes observed repository scope such as candidate ID, include selectors, anchors, and evidence IDs. The project owner contributes explicit architecture intent such as classification, purpose, shipped state, runtime requirement, lifecycle owner, and executable state.

## 4. Profile-path symmetry

Tool `0.3.3` completes explicit Project Profile path consistency across:

```text
adopt
resolve
validate
conform
clarify
gate
```

The migration adds explicit profile support to clarification/gate analysis while repository-root `ptsip.yaml` remains the default when no alternate profile is selected.

## 5. Adoption transaction requirements

### Dry-run

The adoption workflow must:

- discover and validate the selected candidate;
- capture repository evidence;
- load the selected Project Profile;
- validate supplied architecture facts;
- prepare and validate a projected profile;
- return structured adoption output;
- avoid modifying the Consumer Repository.

### `--apply`

The adoption workflow must:

- refuse stale repository evidence;
- refuse conflicting existing declarations;
- preserve concurrent profile-content protection;
- atomically write the validated profile;
- rerun profile validation;
- remain idempotent for equivalent declarations.

The result family is `ptsip-adoption/v1`, including outcomes such as `ADOPTION_PLAN`, `ADOPTED`, `ALREADY_DECLARED`, `CONFLICT`, `UNKNOWN_COMPONENT`, and `STALE_EVIDENCE`.

## 6. Tool 0.3.3 coordination boundary

Tool `0.3.3` preserves the existing Local DecisionStore and explicit hosted Control Plane compatibility required by the coding-agent decision workflow.

The source tree also contains early GitHub-coordinated authority code created while the withdrawn amendment was being evaluated. That code is not used as evidence that repository-global authority consistency is complete in Tool `0.3.3`.

In particular, Tool `0.3.3` does **not** claim completion of all of the following as a stable release contract:

- authority freshness when a local Project Profile already contains a declaration;
- deterministic reconciliation when local declaration and remote winner disagree;
- a final machine-readable conflict/reconciliation state model;
- a final global-versus-local application-state model;
- complete distributed multi-clone consistency verification.

Those requirements belong to Tool `0.3.4`.

## 7. Preserved non-goals

The following are outside the completed Tool `0.3.3` adoption contract:

- global `gate --json` failure-envelope redesign;
- aggregate gate-precondition diagnostics;
- a new active Specification binding;
- a fourth architecture classification;
- automatic/LLM ownership classification;
- directory-name-based ownership decisions;
- batch adoption manifests;
- topology migration semantic changes;
- conformance-rule changes;
- automatic reclassification of existing declarations;
- completion of repository-global GitHub Decision Authority semantics.

A separately proposed Specification `0.3.3-draft` design record does not automatically rebind Tool `0.3.3`; Specification and Tool identities remain independent.

## 8. Completion result

The explicit-adoption migration is complete in source at:

```text
2ab93a63a78391c64f3f715d415314e5f28e2d98
```

The completed Tool `0.3.3` adoption scope includes:

- `ptsip adopt` dry-run / `--apply`;
- explicit profile-path consistency;
- deterministic semantic-validation reuse;
- validated atomic profile projection;
- Local DecisionStore continuity;
- Tool/package source identity `0.3.3` while retaining the bound `0.2.0-draft` Specification revision.

Pre-merge verification was completed in GitHub Actions run `31465581071` on Python 3.14 with `128 passed`, CLI smoke checks, package build, and `twine check` success.

## 9. Permanent publication decision

Tool `0.3.3` is intentionally and permanently a source-only migration version.

The project will not create:

```text
tool-v0.3.3
GitHub Release 0.3.3
PyPI PTSIP==0.3.3
```

The Tool release line skips directly from the last published `0.3.1` line to a future `0.3.4` publication candidate after the Tool `0.3.4` workstream satisfies its own acceptance and release-readiness requirements.

This publication decision is intentional and must not later be interpreted as an unfinished release task.