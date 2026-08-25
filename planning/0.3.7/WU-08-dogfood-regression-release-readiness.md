# WU-08 — Repository Self-Analysis, Fixture Regression, Package, and Release Readiness

> **Status:** PRE-CREATED / LOCKED  
> **Target Tool:** `0.3.7`  
> **Roadmap predecessor:** WU-07 — fixture-verified safe sequential apply and canonical promotion capability  
> **Repository self-adoption boundary:** ADR-0017 — Tool Version and Project Profile Contract Independence  
> **Verification boundary:** real PTSIP Project Profiles remain read-only; migration mutation/promotion regression runs only in controlled fixtures  
> **Entry baseline:** not assigned; capture fresh branch HEAD on actual entry  
> **Successor:** release handoff only after exact-SHA verification and Specification freeze

## 0. Purpose

Close Tool `0.3.7` by combining three distinct verification layers without conflating them:

```text
real PTSIP repository
    -> read-only self-analysis / non-mutating preview / regression

controlled fixture repositories
    -> mutation / deletion / interruption / recovery / promotion regression

exact Tool branch state
    -> full pytest / package / distribution / workflow / release readiness
```

Tool version and Project Profile contract version are independent axes. WU-08 therefore does **not** require the PTSIP repository to create or promote a `0.3.7-draft` Project Profile merely because Tool `0.3.7` is being released.

## 1. Real repository self-analysis boundary

At Tool `0.3.7` release verification, unless a separate Project Profile migration decision is explicitly accepted, the PTSIP repository canonical profile remains:

```text
ptsip.yaml
ptsip.version = 0.3.6-draft
ptsip.specification.revision = d6995ed232e845b88d8235b851e80ab54b7804ea
```

The following public example profiles likewise remain on their actual supported Project Profile contract unless their semantics independently change:

```text
profiles/example.ptsip.yaml
profiles/hybrid-python-package.ptsip.yaml
profiles/template-python-package.ptsip.yaml
```

Repository self-analysis may exercise:

- transition discovery against the current root state;
- source compatibility reading;
- repository/candidate evidence collection;
- normalized evidence behavior;
- migration obligation analysis;
- proposal derivation;
- deterministic planning and preview;
- stale-state detection that does not mutate the profile;
- confirmation that no repository-local target Project Profile was selected.

Repository self-analysis MUST NOT:

- create `ptsip_0.3.7.yaml`;
- rewrite `ptsip.yaml` to `0.3.7-draft`;
- rewrite the public example profiles to `0.3.7-draft` merely to match the Tool version;
- delete any real source profile;
- exercise canonical promotion against the real repository;
- treat Tool release verification as Project Profile migration authority.

WU-08 SHOULD capture before/after content identities for the actual Project Profile files and prove they did not change as a side effect of migration-focused verification.

## 2. Fixture-based migration regression

All write-path regression for the migration engine MUST use isolated controlled fixture repositories.

### Simple transition fixture

```text
older canonical draft -> newer selected target draft
```

A fixture may intentionally use:

```text
0.3.6-draft -> 0.3.7-draft
```

for transition mechanics. This does not imply that the real PTSIP repository adopts `0.3.7-draft`.

### Sequential transition fixture

```text
0.3.4-draft canonical
+ 0.3.6-draft temporary source
+ 0.3.7-draft temporary source
+ 0.4.0-draft Final Point
```

### Sparse intermediate history fixture

```text
0.3.4-draft canonical
+ 0.3.7-draft temporary source
+ 0.4.0-draft Final Point
```

No synthetic `0.3.6` temporary file should be required merely because a conceptual version existed.

### Required mutation-path cases

Fixture regression must include at least:

- planned Final Point creation;
- accepted semantic delta apply;
- ordered multi-source convergence;
- temporary source deletion after completion proof;
- canonical-last behavior;
- guarded Final Point promotion;
- interruption after mutation but before checkpoint;
- interruption after source deletion;
- deterministic resume where state matches the ledger;
- fail-closed recovery where ledger/filesystem state disagree;
- append-only checkpoint integrity failure;
- stale source snapshot;
- stale Final Point snapshot;
- target `before` semantic CAS mismatch;
- unresolved required obligations;
- conflict with accepted cumulative Final Point state;
- attempted early canonical promotion;
- filename/internal version mismatch;
- malformed/missing revision;
- duplicate logical target identity;
- non-monotonic target selection.

Fixture mutation MUST be confined to the fixture root. A test that escapes and changes the real repository Project Profile is a verification failure.

## 3. Obligation semantics verification

Tests and self-analysis evidence must prove:

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

No score, heuristic, evidence count, or optional work may compensate for incomplete Required Work Elements.

## 4. Independent source evaluation verification

Sequential fixtures must prove that each source independently re-evaluates Required/Removal/Async categories against its current snapshot.

The system must not copy a previous source's category assignment as authority.

At the same time, already accepted Final Point declarations remain cumulative target state. A later source may recognize them as satisfying an obligation only through explicit target-semantic analysis, not through inherited source-category metadata.

## 5. Project Profile contract/version verification

WU-08 must explicitly verify the ADR-0017 boundary:

```text
Tool version = 0.3.7
Tool normative Specification family = 0.3.7-draft
Project Profile contract = may remain 0.3.6-draft
```

A Tool version or Tool Specification-family change must not be interpreted as automatic authority to rewrite `ptsip.version` in consumer profiles.

The public examples remain `0.3.6-draft` unless a real Project Profile semantic change independently justifies another contract version.

## 6. Documentation and contract surfaces

Before closure, review whether completed Tool behavior requires synchronized updates in:

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

`profiles/` MUST NOT be version-bumped solely because Tool `0.3.7` exists.

If migration tests require fixture profiles, keep them under test/fixture ownership rather than introducing production root temporary profiles.

## 7. Focused test structure

New tests should follow the existing test organization and reuse existing helpers where sensible. The 0.3.6 WU-04G direction remains active: when touching or adding tests, incrementally improve purpose/role-based organization rather than performing an unrelated wholesale test migration.

Focused verification should remain clearly separated by purpose:

```text
transition/discovery tests
source compatibility tests
evidence/analysis tests
proposal/planning tests
execution-state/ledger tests
fixture apply/deletion/promotion tests
repository read-only self-analysis tests
```

Minimum focused areas:

- transition state discovery;
- source ordering;
- source readers;
- obligation evaluation;
- proposal/Final Point planning;
- authorization and exact-plan binding;
- append-only checkpoint ledger;
- safe fixture apply;
- deletion eligibility;
- interruption/recovery;
- fixture canonical promotion;
- repository self-analysis without Project Profile mutation.

## 8. Full regression

After focused tests pass, run the repository's complete regression path appropriate to Tool `0.3.7`.

At minimum:

```text
python -m pytest
```

and the exact repository/package checks already required by current release policy.

The full regression may execute fixture mutation tests, but it MUST NOT modify the actual repository Project Profile files.

If `tooling-test.yml` is part of the required closure contract, its exact-SHA run must be successful. The workflow should be modified only if Tool `0.3.7` behavior genuinely requires new verification coverage; do not change it merely to force a run.

## 9. Package/distribution verification

Verify at least:

- package build succeeds;
- wheel/sdist contents contain synchronized runtime specdata assets required by the actual Tool contract;
- install/reinstall smoke succeeds;
- `python -m ptsip --version` reports the intended Tool version at the appropriate release-preparation stage;
- packaged CLI continues to validate/adopt supported Project Profile contracts as intended;
- transition/migration functionality is available without forcing a Project Profile contract bump;
- VPMS boundary remains intact;
- no development-only Temporary PTSIP Profile File is accidentally packaged unless explicitly part of a public fixture/example contract;
- no test ledger/checkpoint artifact is included in a release package.

## 10. Release Specification freeze

Tool `0.3.7` release cannot close against a floating or unverified Tool Specification identity.

Required final boundary:

```text
implemented Tool behavior
    -> final Tool 0.3.7 normative text
    -> immutable SPEC_REVISION
    -> exact Tool/Specification binding
    -> exact-SHA focused/full regression/package/workflow evidence
    -> release handoff
```

This Tool/Specification binding does **not** require the repository's canonical Project Profile to declare the same semantic version.

For Tool `0.3.7`, repository `ptsip.yaml` may remain bound to the supported `0.3.6-draft` Project Profile contract unless a separate Project Profile migration decision is accepted.

## 11. Required verification evidence

WU-08 closure evidence must distinguish at least:

```text
A. real repository read-only self-analysis evidence
B. fixture mutation/promotion regression evidence
C. full exact-SHA regression evidence
D. package/distribution evidence
E. exact-SHA workflow/release-gate evidence where required
```

Evidence from a fixture migration MUST NOT be represented as evidence that the real repository itself migrated Project Profile generations.

Likewise, successful repository self-analysis MUST NOT be represented as proof that mutation/promotion paths were executed on the real repository.

## 12. Completion gate

WU-08 is complete only when:

- simple and Sequential Work fixture regressions pass;
- independent source evaluation is proven;
- Required/Removal/Async completion semantics are proven;
- conflict/stale/interruption/recovery cases fail safely;
- guarded mutation/deletion/promotion are proven in controlled fixtures;
- real repository self-analysis succeeds without Project Profile mutation;
- actual `ptsip.yaml` remains on its independently selected Project Profile contract;
- `ptsip_0.3.7.yaml` is not created in the PTSIP repository solely for Tool `0.3.7`;
- existing public profile examples are not version-bumped without a semantic contract reason;
- focused tests pass;
- full regression passes;
- package/distribution verification passes;
- required exact-SHA workflow verification succeeds;
- documentation/schema/spec assets are synchronized where actually required;
- final Tool `0.3.7` immutable Specification binding is frozen for release.

## 13. Entry discipline

Pre-created roadmap document only. Actual entry requires WU-07 completion and a fresh exact branch HEAD.

WU-08 verification evidence must record the exact source SHA it verifies; later documentation-only commits must not silently replace that verification authority.

ADR-0017 is a mandatory WU-08 boundary: Tool release verification must not silently reintroduce the superseded assumption that Tool `0.3.7` requires a repository-local `0.3.7-draft` Project Profile.
