# WU-01 — Draft Profile Transition State Model

> **Status:** PRE-CREATED / LOCKED  
> **Target Tool:** `0.3.7`  
> **Roadmap predecessor:** Tool `0.3.6` release + ADR-0010 acceptance  
> **Branch baseline:** `1aaa6868fe9b423b1fa536404115820cc4736ac4`  
> **Entry baseline:** not assigned; capture fresh `dev/0.3.7` HEAD on actual entry  
> **Successor:** WU-02 — candidate-discovery evidence expansion

## 0. Purpose

Introduce the profile-generation state model required before evidence-driven migration can safely mutate any draft-bound `ptsip.yaml`.

WU-01 converts ADR-0010 from policy terminology into deterministic, inspectable transition state. It does not migrate architecture yet.

## 1. Required repository state model

The implementation must distinguish:

```text
CanonicalProfile
    path = ptsip.yaml

TemporaryProfile
    path = ptsip_<major>.<minor>.<micro>.yaml

TransitionSet
    canonical source
    temporary sources
    final point
    execution order
    snapshot identity
```

The model must not treat the filename alone as draft identity. Each profile requires coherent internal `version` and immutable `specification.revision` identity.

## 2. Deterministic discovery rules

WU-01 must provide one canonical discovery mechanism for repository-root PTSIP profile generations.

Rules:

1. `ptsip.yaml` is the canonical active source when present.
2. `ptsip_<major>.<minor>.<micro>.yaml` is a Temporary PTSIP Profile File candidate.
3. The filename version must map exactly to internal `<major>.<minor>.<micro>-draft`.
4. A target semantic version may have only one logical temporary candidate.
5. Unknown similarly named files must not be silently adopted as transition state.
6. Profiles with malformed/missing `version` or `specification.revision` fail closed for mutation.
7. Discovery result ordering must be stable.

## 3. Simple transition state

Example:

```text
ptsip.yaml          = 0.3.6-draft
ptsip_0.3.7.yaml    = 0.3.7-draft
```

Expected state:

```text
mode = SIMPLE
canonical_source = ptsip.yaml
final_point = ptsip_0.3.7.yaml
ordered_sources = [ptsip.yaml]
```

The target file existing does not mean migration is complete or authoritative.

## 4. Sequential transition state

Example:

```text
ptsip.yaml          = 0.3.4-draft
ptsip_0.3.6.yaml    = 0.3.6-draft
ptsip_0.3.7.yaml    = 0.3.7-draft
ptsip_0.4.0.yaml    = 0.4.0-draft
```

Expected state:

```text
mode = SEQUENTIAL
final_point = ptsip_0.4.0.yaml
ordered_sources = [
    ptsip_0.3.7.yaml,
    ptsip_0.3.6.yaml,
    ptsip.yaml,
]
```

The canonical profile is always last. Temporary sources are descending by version, excluding the Final Point itself.

## 5. Final Point selection

Final Point selection must be deterministic and monotonic.

Default rule:

```text
highest valid selected temporary target version
    -> Final PTSIP Point File
```

A caller may request creation of a newer target draft; after creation/validation it becomes the Final Point if it is higher than all participating target versions.

The implementation must reject ambiguous identity rather than guessing.

## 6. Snapshot identity

Transition planning must be bound to exact file/repository facts sufficient to detect stale mutation.

At minimum, preserve:

- repository snapshot/worktree identity used by the caller;
- canonical source content identity;
- each temporary source content identity;
- Final Point content identity;
- source and target draft/revision identity.

The exact digest/SHA representation may reuse existing repository snapshot/profile digest facilities if suitable. WU-01 should extend existing reusable identity primitives rather than create a second unrelated snapshot system.

## 7. Diagnostics

WU-01 needs structured diagnostics for at least:

```text
NO_CANONICAL_PROFILE
INVALID_TEMPORARY_FILENAME
PROFILE_VERSION_FILENAME_MISMATCH
MISSING_PROFILE_VERSION
MISSING_SPEC_REVISION
DUPLICATE_TARGET_IDENTITY
NON_MONOTONIC_TARGET
AMBIGUOUS_FINAL_POINT
STALE_TRANSITION_SNAPSHOT
```

Diagnostic names may be adjusted to existing conventions during implementation, but semantics must remain explicit and machine-testable.

## 8. Work tracks

### WU-01A — existing profile-path inventory

Review current `src/ptsip/repository/profile_path.py`, validation entrypoints, CLI profile resolution, adoption projection, and tests. Reuse the existing path contract where possible.

### WU-01B — generation discovery model

Add the narrow data model and parser for canonical/temporary generation identities.

### WU-01C — ordering and Final Point resolver

Implement simple/sequential classification, deterministic target selection, and source ordering.

### WU-01D — snapshot/staleness contract

Bind transition plans to exact profile/repository identity using existing snapshot/digest primitives where possible.

### WU-01E — focused tests

Add isolated tests for discovery, identity mismatch, ordering, ambiguity, malformed state, and stale plan detection.

## 9. Non-goals

WU-01 does not:

- classify repository files into migration elements;
- migrate lifecycle classifications;
- build target proposals;
- ask the owner architecture questions;
- delete any profile;
- rename the Final Point to `ptsip.yaml`;
- change `.github/workflows/tooling-test.yml` unless focused verification truly requires workflow integration.

## 10. Completion gate

WU-01 is complete only when:

- canonical/temporary profile discovery is deterministic;
- filename/internal draft identity is enforced;
- duplicate logical target identity fails closed;
- Final Point selection is deterministic;
- source ordering exactly follows ADR-0010;
- canonical `ptsip.yaml` is always last;
- transition state is snapshot-bound;
- no architecture mutation occurs;
- focused tests pass;
- WU-02 has not been entered early.

## 11. Entry discipline

This document is planning authority only. Actual WU-01 entry requires an established `0.3.7-draft` Specification family/revision and a fresh exact `dev/0.3.7` entry SHA recorded here.
