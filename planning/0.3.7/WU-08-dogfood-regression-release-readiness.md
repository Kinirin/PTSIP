# WU-08 — Repository Dogfood, Regression, Package, and Release Readiness

> **Status:** PRE-CREATED / LOCKED  
> **Target Tool:** `0.3.7`  
> **Roadmap predecessor:** WU-07 — safe sequential apply and canonical promotion  
> **Repository self-adoption boundary:** ADR-0017 — Tool Version and Project Profile Contract Independence  
> **Entry baseline:** not assigned; capture fresh branch HEAD on actual entry  
> **Successor:** release handoff only after exact-SHA verification and Specification freeze

## 0. Purpose

Close Tool `0.3.7` by proving the new profile-generation transition policy works on focused fixtures, by exercising repository self-analysis/dogfood without forcing an artificial Project Profile version change, and by running the full regression/package/distribution/release-readiness chain on exact branch state.

Tool version and Project Profile contract version are independent axes. WU-08 therefore does **not** require the PTSIP repository to create or promote a `0.3.7-draft` Project Profile merely because Tool `0.3.7` is being released.

## 1. Repository dogfood boundary

At `dev/0.3.7` branch creation, and throughout Tool `0.3.7` unless a separate Project Profile migration decision is accepted, the repository canonical profile remains:

```text
ptsip.yaml
ptsip.version = 0.3.6-draft
ptsip.specification.revision = d6995ed232e845b88d8235b851e80ab54b7804ea
```

The repository must dogfood Tool `0.3.7` by exercising discovery, compatibility reading, evidence, migration analysis, planning, stale-state guards, and non-mutating preview against the real repository state.

It must **not** manufacture this transition:

```text
ptsip.yaml          # 0.3.6-draft
ptsip_0.3.7.yaml    # artificial self-target — NOT created for Tool 0.3.7
```

Repository dogfood succeeds when the Tool can inspect and reason about the current repository deterministically without forcing a Project Profile version bump.

Mutation, source deletion, interruption recovery, and canonical promotion are verified in controlled fixture repositories that intentionally model real draft-family transitions.

## 2. Required fixture families

WU-08 must exercise at least:

### Simple transition fixture

```text
older canonical draft -> newer selected target draft
```

A fixture may use `0.3.6-draft -> 0.3.7-draft` to test transition mechanics, but that fixture does not imply that the PTSIP repository itself adopts `0.3.7-draft`.

### Sequential transition fixture

```text
0.3.4-draft canonical
+ 0.3.6-draft temporary
+ 0.3.7-draft temporary
+ 0.4.0-draft Final Point
```

### Sparse intermediate history fixture

```text
0.3.4-draft canonical
+ 0.3.7-draft temporary
+ 0.4.0-draft Final Point
```

No synthetic `0.3.6` temporary file should be required merely because a conceptual version existed.

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
- attempted early canonical promotion;
- repository self-run attempting to create `ptsip_0.3.7.yaml` without an accepted Project Profile migration decision.

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

## 5. Project Profile contract/version verification

WU-08 must explicitly verify the ADR-0017 boundary:

```text
Tool version = 0.3.7
Project Profile contract = may remain 0.3.6-draft
```

The following existing examples remain `0.3.6-draft` unless their actual Project Profile semantics change:

```text
profiles/example.ptsip.yaml
profiles/hybrid-python-package.ptsip.yaml
profiles/template-python-package.ptsip.yaml
```

A Tool version bump must not be treated as authority to rewrite their `ptsip.version` values.

## 6. Documentation and contract surfaces

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

`profiles/` must not be version-bumped solely because Tool `0.3.7` exists.

## 7. Focused test structure

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
- repository self-analysis without profile-version mutation.

## 8. Full regression

After focused tests pass, run the repository's complete regression path appropriate to Tool `0.3.7`.

At minimum:

```text
python -m pytest
```

and the exact repository/package checks already required by current release policy.

If `tooling-test.yml` is part of the required closure contract, its exact-SHA run must be successful. The workflow should be modified only if 0.3.7 behavior genuinely requires new verification coverage; do not change it merely to force a run.

## 9. Package/distribution verification

Verify at least:

- package build succeeds;
- wheel/sdist contents contain synchronized runtime specdata assets;
- install/reinstall smoke succeeds;
- `python -m ptsip --version` reports the intended Tool version at the appropriate release-preparation stage;
- packaged CLI continues to validate/adopt supported Project Profile contracts as intended;
- transition/migration functionality is available without forcing a profile contract bump;
- VPMS boundary remains intact;
- no development-only temporary profile artifact is accidentally packaged unless explicitly part of the public contract.

## 10. Release Specification freeze

Tool `0.3.7` release cannot close against a floating or unverified development Specification identity.

Required final boundary:

```text
implemented Tool behavior
    -> final Tool 0.3.7 normative text
    -> immutable SPEC_REVISION
    -> exact Tool/Specification binding
    -> exact-SHA regression/package/workflow evidence
    -> release handoff
```

This Tool/Specification binding does **not** require the repository's canonical Project Profile to declare the same semantic version.

For Tool `0.3.7`, the canonical repository `ptsip.yaml` may remain bound to the supported `0.3.6-draft` Project Profile contract unless a separate migration decision is accepted.

## 11. Completion gate

WU-08 is complete only when:

- simple and Sequential Work fixtures pass;
- independent obligation evaluation is proven;
- Required/Removal/Async completion semantics are proven;
- conflict/stale/interruption cases fail safely;
- repository self-analysis/dogfood succeeds without artificial Project Profile promotion;
- fixture-based canonical promotion is proven guarded and deterministic;
- `ptsip_0.3.7.yaml` is not created in the PTSIP repository solely for Tool `0.3.7`;
- existing profile examples are not version-bumped without a semantic contract reason;
- all focused tests pass;
- full regression passes;
- package/distribution verification passes;
- required exact-SHA workflow verification succeeds;
- documentation/schema/spec assets are synchronized where required;
- final Tool `0.3.7` immutable Specification binding is frozen for release.

## 12. Entry discipline

Pre-created roadmap document only. Actual entry requires WU-07 completion and a fresh exact branch HEAD. WU-08 verification evidence must record the exact source SHA it verifies; later documentation-only commits must not silently replace that verification authority.

ADR-0017 is a mandatory WU-08 boundary: Tool release verification must not silently reintroduce the superseded assumption that Tool `0.3.7` requires a repository-local `0.3.7-draft` Project Profile.
