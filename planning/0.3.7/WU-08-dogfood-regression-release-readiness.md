# WU-08 — Repository Dogfood, Regression, Package, and Release Readiness

> **Status:** PRE-CREATED / LOCKED  
> **Target Tool:** `0.3.7`  
> **Roadmap predecessor:** WU-07 — safe sequential apply and canonical promotion  
> **Entry baseline:** not assigned; capture fresh branch HEAD on actual entry  
> **Successor:** release handoff only after exact-SHA verification and Specification freeze

## 0. Purpose

Close Tool `0.3.7` by proving the new profile-generation transition policy works on focused fixtures and on the PTSIP repository itself, then run the full regression/package/distribution/release-readiness chain on exact branch state.

## 1. Repository dogfood transition

At `dev/0.3.7` branch creation, the repository canonical profile is:

```text
ptsip.yaml
ptsip.version = 0.3.6-draft
ptsip.specification.revision = d6995ed232e845b88d8235b851e80ab54b7804ea
```

The repository must dogfood the new rule rather than editing those fields in place.

Expected development transition:

```text
ptsip.yaml          # canonical 0.3.6-draft source
ptsip_0.3.7.yaml    # temporary 0.3.7-draft Final Point
```

The old canonical file remains until all source-specific Required Work Elements are handled and the target profile validates under the final 0.3.7 draft/revision.

Only then may the Final Point become canonical.

## 2. Required fixture families

WU-08 must exercise at least:

### Simple transition

```text
0.3.6-draft -> 0.3.7-draft
```

### Sequential transition

```text
0.3.4-draft canonical
+ 0.3.6-draft temporary
+ 0.3.7-draft temporary
+ 0.4.0-draft Final Point
```

### Sparse intermediate history

```text
0.3.4-draft canonical
+ 0.3.7-draft temporary
+ 0.4.0-draft Final Point
```

No synthetic 0.3.6 temporary file should be required merely because a conceptual version existed.

### Invalid states

- filename/internal version mismatch;
- malformed/missing revision;
- duplicate logical target identity;
- non-monotonic target selection;
- stale source snapshot;
- stale Final Point snapshot;
- unresolved required obligations;
- conflict with accepted Final Point state;
- interrupted transition after one source completion;
- attempted early canonical promotion.

## 3. Obligation semantics verification

Tests and dogfood evidence must prove:

```text
Required Work Element
    -> contributes to completion
    -> unresolved blocks source deletion/promotion

Removal Migration Element
    -> no preservation obligation
    -> no completion contribution

Asynchronous Work Target
    -> optional after Required work
    -> no completion contribution
```

No score/heuristic may allow optional work to compensate for incomplete Required work.

## 4. Independent source evaluation verification

Sequential fixtures must prove that each source independently re-evaluates the three obligation categories.

The system must not copy the previous source's category assignment as authority.

At the same time, already accepted Final Point target declarations remain target state. A later source may recognize them as satisfying an obligation only through explicit target-semantic analysis, not through inherited category metadata.

## 5. Documentation and contract surfaces

Before closure, review whether the completed behavior requires synchronized updates in:

```text
spec/
releasenote/spec-0.3.7-draft.md
profiles/
schemas/
src/ptsip/specdata/
adoption/ADOPTION-GUIDE.md
agents/AGENT-CONTRACT.md
README.md / README.ko.md
STATUS.md
MEMORY.md / AGENTS.md
```

Only surfaces that actually own or expose the new behavior should change. Do not add duplicate policy documents or parallel schemas without necessity.

## 6. Focused test structure

New tests should be placed under the existing test organization and should reuse existing helpers where sensible. The 0.3.6 WU-04G test-tracking migration direction remains relevant: when touching or adding tests, incrementally improve purpose-based test organization rather than performing an unrelated all-at-once repository test migration.

Minimum focused areas:

- transition state discovery;
- source ordering;
- source readers;
- obligation evaluation;
- proposal/Final Point planning;
- safe apply;
- deletion eligibility;
- interruption/resume;
- canonical promotion;
- repository self-adoption.

## 7. Full regression

After focused tests pass, run the repository's complete regression path appropriate to Tool `0.3.7`.

At minimum:

```text
python -m pytest
```

and the exact repository/package checks already required by current release policy.

If `tooling-test.yml` is part of the required closure contract, its exact-SHA run must be successful. The workflow should be modified only if 0.3.7 behavior genuinely requires new verification coverage; do not change it merely to force a run.

## 8. Package/distribution verification

Verify at least:

- package build succeeds;
- wheel/sdist contents contain synchronized runtime specdata assets;
- install/reinstall smoke succeeds;
- `python -m ptsip --version` reports the intended Tool version at the appropriate release-preparation stage;
- packaged CLI can validate/adopt the target profile contract as intended;
- VPMS boundary remains intact;
- no development-only temporary profile artifact is accidentally packaged unless explicitly part of the public contract.

## 9. Release Specification freeze

Tool `0.3.7` release cannot close against a floating or unverified draft identity.

Required final boundary:

```text
implemented behavior
    -> final 0.3.7-draft normative text
    -> immutable SPEC_REVISION
    -> exact Tool/Specification binding
    -> exact-SHA regression/package/workflow evidence
    -> release handoff
```

The repository's promoted canonical `ptsip.yaml` must bind the correct final draft/revision before release.

## 10. Completion gate

WU-08 is complete only when:

- simple and Sequential Work fixtures pass;
- independent obligation evaluation is proven;
- Required/Removal/Async completion semantics are proven;
- conflict/stale/interruption cases fail safely;
- PTSIP repository dogfood transition completes under the new policy;
- canonical promotion is proven guarded and deterministic;
- all focused tests pass;
- full regression passes;
- package/distribution verification passes;
- required exact-SHA workflow verification succeeds;
- documentation/schema/spec assets are synchronized where required;
- final Tool `0.3.7` / `0.3.7-draft` immutable revision binding is frozen for release.

## 11. Entry discipline

Pre-created roadmap document only. Actual entry requires WU-07 completion and a fresh exact branch HEAD. WU-08 verification evidence must record the exact source SHA it verifies; later documentation-only commits must not silently replace that verification authority.
