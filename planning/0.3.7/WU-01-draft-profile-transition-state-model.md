# WU-01 — Draft Profile Transition State Model

> **Status:** COMPLETE / FOCUSED TEST VERIFIED  
> **Target Tool:** `0.3.7`  
> **Roadmap predecessor:** Tool `0.3.6` release + ADR-0010 acceptance  
> **Branch baseline:** `1aaa6868fe9b423b1fa536404115820cc4736ac4`  
> **WU-01 exact entry baseline:** `ebc2f7d03a18e740f81826a0fff543f4aebfbb93`  
> **Bound Specification at entry:** `0.3.7-draft @ b648d9e026f502b14481ba2d0606d9acc88a31fc`  
> **WU-01 implementation content SHA:** `db513a7bb8f5e081a538293d8396086f2540c43d`  
> **Focused verification:** `12 passed / 0 failed` on the WU-01 focused suite  
> **Successor:** WU-02 — candidate-discovery evidence expansion (`PRE-CREATED / LOCKED`)

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

WU-01 entered on exact `dev/0.3.7` baseline `ebc2f7d03a18e740f81826a0fff543f4aebfbb93` after establishment of the `0.3.7-draft` Specification family at immutable revision `b648d9e026f502b14481ba2d0606d9acc88a31fc`.

WU-01 completion does not authorize WU-02 entry, architecture migration, canonical `ptsip.yaml` replacement, or Final Point promotion. WU-02 remains locked until a separate explicit entry transition records a fresh branch SHA.

## 12. WU-01A~E execution record

### WU-01A — COMPLETE

Inventory confirmed that the existing repository already has two reusable primitives required by this WU:

- `src/ptsip/repository/profile_path.py` provides repository-relative path normalization, selected-profile identity, on-disk containment, and decision-ID path binding;
- `src/ptsip/repository/snapshot.py` provides Git/worktree observation, status fingerprinting, tracked-content fingerprinting, and snapshot comparison.

Existing validation/CLI/adoption behavior was intentionally not rewritten. WU-01 introduces a narrow sibling transition-state module so current Tool `0.3.6` profile selection and validation behavior remain stable.

### WU-01B — COMPLETE

Added:

```text
src/ptsip/repository/profile_transition.py
```

The module provides immutable draft-version/profile-generation identities, structured diagnostics, repository-root temporary-profile discovery, canonical profile discovery, filename/internal-version matching, immutable revision capture, duplicate semantic target detection, and fail-closed malformed-state handling.

### WU-01C — COMPLETE

The resolver now exposes deterministic states:

```text
IDLE       canonical only; no target generation exists
SIMPLE     one temporary target; canonical is the only migration source
SEQUENTIAL multiple temporary generations; highest target is Final Point
```

For Sequential Work, temporary sources are ordered by descending semantic version below the Final Point and canonical `ptsip.yaml` is appended last. Non-monotonic targets, duplicate semantic targets, and an ambiguous highest target fail closed.

### WU-01D — COMPLETE

`TransitionSnapshot` reuses the existing `RepositorySnapshot` and additionally binds:

- resolved repository root;
- canonical and temporary profile paths;
- each profile's declared draft version;
- each profile's immutable Specification revision;
- each profile's SHA-256 content identity.

`validate_transition_snapshot()` rejects repository-root changes, repository snapshot changes, missing profile files, and profile-content changes through `STALE_TRANSITION_SNAPSHOT`. This closes the untracked-temporary-file gap where Git status alone could remain unchanged while an existing untracked profile's bytes change.

### WU-01E — COMPLETE

Added the focused role-scoped test file:

```text
tests/ptsip/repository/test_profile_transition_037.py
```

Covered scenarios:

- canonical-only `IDLE` discovery;
- simple transition;
- multi-generation Sequential Work ordering;
- filename/internal-version mismatch;
- invalid similarly named temporary file;
- missing version/revision;
- duplicate semantic target identity and ambiguous Final Point;
- non-monotonic target;
- missing canonical profile;
- malformed YAML;
- stale profile content;
- repository-root mismatch.

Focused verification result:

```text
Python 3.13.5
12 passed / 0 failed
```

The focused suite was executed in an isolated local harness using the same WU-01 module/test content and the existing `profile_path`/`snapshot` contracts. This is intentionally recorded as focused verification, not as GitHub Actions exact-SHA release verification. Full self-hosted `tooling-test.yml` exact-SHA regression remains the WU-08 release-readiness gate.

## 13. Completion conclusion

WU-01A through WU-01E are complete. The implementation is read-only with respect to architecture declaration and does not create, delete, rename, or promote a PTSIP profile.

The canonical root `ptsip.yaml` therefore remains the existing `0.3.6-draft` source. No `ptsip_0.3.7.yaml` is created by WU-01 itself.

WU-02 remains `PRE-CREATED / LOCKED` and is not entered by this completion record.
