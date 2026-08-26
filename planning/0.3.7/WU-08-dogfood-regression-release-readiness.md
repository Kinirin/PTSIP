# WU-08 — Repository Self-Analysis, Fixture Regression, Package, and Release Readiness

> **Status:** ACTIVE  
> **Target Tool:** `0.3.7`  
> **Roadmap predecessor:** WU-07 — fixture-verified safe sequential apply and canonical promotion capability (`COMPLETE / FOCUSED TEST VERIFIED`)  
> **Repository self-adoption boundary:** ADR-0017 — Tool Version and Project Profile Contract Independence  
> **Project Profile versioning boundary:** ADR-0019 — Independent Project Profile Versioning  
> **Verification boundary:** real PTSIP Project Profiles remain read-only until an independently authorized PP contract transition; migration mutation/promotion regression runs only in controlled fixtures  
> **Exact entry baseline:** `874fa80a508f5901647d6d2df132a95f0eadda49`  
> **Inherited verification evidence:** workflow run `32825710974` at `68b29308e8531f93267acc2aec585f6e751021f1` produced `437 passed / 1 failed`; that repository self-profile regression was later resolved and verified by run `32923347579` at `fc1f479d34cb9531180bdf111bfe0695ba0fc48b` with `439 passed / 0 failed`  
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

ADR-0019 strengthens this boundary by introducing an independent Project Profile contract namespace. The intended current-generation PP identity is `pp.1.01`, but no real repository profile may be relabeled or promoted to that identity until the PP parser/schema/compatibility/migration implementation described in Section 15 is complete and verified.

## 1. Real repository self-analysis boundary

At Tool `0.3.7` release verification, until the separately authorized PP contract transition is implemented and verified, the PTSIP repository canonical profile remains:

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

- create `ptsip_0.3.7.yaml` merely because Tool `0.3.7` exists;
- rewrite `ptsip.yaml` to `0.3.7-draft` merely to match the Tool version;
- rewrite `ptsip.yaml` to `pp.1.01` before ADR-0019 implementation gates are satisfied;
- rewrite the public example profiles merely to match the Tool version or intended PP namespace;
- delete any real source profile;
- exercise canonical promotion against the real repository without separately authorized PP migration;
- treat Tool release verification as Project Profile migration authority.

WU-08 SHOULD capture before/after content identities for the actual Project Profile files and prove they did not change as a side effect of migration-focused verification.

## 2. Fixture-based migration regression

All write-path regression for the migration engine MUST use isolated controlled fixture repositories.

### Simple transition fixture

```text
older canonical draft -> newer selected target draft
```

Historical fixtures may intentionally use legacy labels such as:

```text
0.3.6-draft -> 0.3.7-draft
```

for transition mechanics. Such fixture labels do not imply that Tool version and PP contract version are coupled and do not imply that the real PTSIP repository adopts a matching Tool-numbered profile.

### Sequential transition fixture

```text
0.3.4-draft canonical
+ 0.3.6-draft temporary source
+ 0.3.7-draft temporary source
+ 0.4.0-draft Final Point
```

These are legacy-version fixture identities until the ADR-0019 PP identity transition is implemented.

### Sparse intermediate history fixture

```text
0.3.4-draft canonical
+ 0.3.7-draft temporary source
+ 0.4.0-draft Final Point
```

No synthetic intermediate temporary file should be required merely because a conceptual version existed.

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

WU-08 must explicitly verify the ADR-0017 and ADR-0019 boundaries:

```text
Tool version = 0.3.7
Tool normative Specification family = 0.3.7-draft
Project Profile contract identity = independent PP namespace
intended current PP contract = pp.1.01
current repository source label = 0.3.6-draft until PP transition implementation is verified
```

A Tool version or Tool Specification-family change must not be interpreted as automatic authority to rewrite `ptsip.version` in consumer profiles.

The Project Profile version is not Tool SemVer. Its canonical namespace is:

```text
pp.<major>.<minor>
```

The Project Profile major definition is normative:

> **Project Profile major version은 governing lifecycle classification의 집합 또는 그 classification semantics가 변경되어 기존 프로젝트의 lifecycle 재분류를 요구할 수 있을 때 상승한다.**

Within one major generation, PP minor changes represent Project Profile contract-semantic/structural evolution such as `components`, `relationships`, `associated_artifacts`, `policies`, and Responsibility Map declaration semantics. Ordinary project-specific declaration edits remain Project Profile Instance revisions and do not bump the PP contract version.

The public examples remain on their actual current source identity until an independently justified and implemented PP contract migration exists.

## 6. Documentation and contract surfaces

Before closure, review whether completed Tool behavior or ADR-0019 PP versioning requires synchronized updates in:

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

`profiles/` MUST NOT be version-bumped solely because Tool `0.3.7` exists. They also MUST NOT be mechanically relabeled to `pp.1.01` before the PP compatibility transition is implemented and verified.

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
PP identity/compatibility tests
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
- repository self-analysis without Project Profile mutation;
- Tool/PP independent version identity;
- PP compatibility and unsupported-version fail-closed behavior.

## 8. Full regression

After focused tests pass, run the repository's complete regression path appropriate to Tool `0.3.7`.

At minimum:

```text
python -m pytest
```

and the exact repository/package checks already required by current release policy.

The full regression may execute fixture mutation tests, but it MUST NOT modify the actual repository Project Profile files.

If `tooling-test.yml` is part of the required closure contract, its exact-SHA run must be successful. The workflow should be modified only if Tool `0.3.7` behavior genuinely requires new verification coverage; do not change it merely to force a run.

Workflow run `32923347579` verified exact SHA `fc1f479d34cb9531180bdf111bfe0695ba0fc48b` with:

```text
439 passed
0 failed
repository validation warnings: []
responsibility_map_coverage.unassigned_count: 0
self-hosted/tooling-test: success
```

This closes the inherited 18-file repository self-profile coverage regression. It predates the later workflow artifact-evidence ownership synchronization and ADR-0019 implementation work, so it is not the final WU-08 release-readiness authority.

## 9. Package/distribution verification

Verify at least:

- package build succeeds;
- wheel/sdist contents contain synchronized runtime specdata assets required by the actual Tool contract;
- wheel artifact evidence attributes `ptsip-evidence`, `ptsip-source-compat`, and `ptsip-migration` to their ADR-0018 PRODUCT components rather than collapsing them into `ptsip-core`;
- install/reinstall smoke succeeds;
- `python -m ptsip --version` reports Tool `0.3.7` at the appropriate release-preparation stage;
- packaged CLI reports Tool version and Project Profile contract identity as separate concepts where both are relevant;
- packaged CLI continues to validate/adopt explicitly supported Project Profile contracts as intended;
- transition/migration functionality is available without forcing a Project Profile contract bump solely because of Tool version;
- VPMS boundary remains intact;
- no development-only Temporary PTSIP Profile File is accidentally packaged unless explicitly part of a public fixture/example contract;
- no test ledger/checkpoint artifact is included in a release package.

The wheel artifact-evidence ownership logic was synchronized with ADR-0018 on `dev/0.3.7` in commit `4329e48127003151a6f256320da236943e674911`. A later exact-SHA workflow run must verify this change before WU-08 completion.

## 10. Release Specification freeze

Tool `0.3.7` release cannot close against a floating or unverified Tool Specification identity.

Required final boundary:

```text
implemented Tool behavior
    -> final Tool 0.3.7 normative text
    -> immutable SPEC_REVISION
    -> exact Tool/Specification binding
    -> explicit Tool 0.3.7 <-> supported PP compatibility declaration
    -> exact-SHA focused/full regression/package/workflow evidence
    -> release handoff
```

This Tool/Specification binding does **not** require the repository's canonical Project Profile to declare the same semantic version.

ADR-0019 removes the expectation that the Project Profile should ever use a matching Tool `0.3.7` identity merely for numeric symmetry.

## 11. Required verification evidence

WU-08 closure evidence must distinguish at least:

```text
A. real repository read-only self-analysis evidence
B. fixture mutation/promotion regression evidence
C. full exact-SHA regression evidence
D. package/distribution evidence
E. exact-SHA workflow/release-gate evidence where required
F. Tool/PP compatibility and independent-version evidence
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
- real repository self-analysis succeeds without unauthorized Project Profile mutation;
- Tool `0.3.7` and PP contract identity are implemented as independent version axes;
- `pp.<major>.<minor>` parsing/serialization and supported-version compatibility behavior are verified before `pp.1.01` is represented as active;
- actual `ptsip.yaml` changes PP contract identity only through an independently authorized and verified PP migration;
- no `ptsip_0.3.7.yaml` is created in the PTSIP repository solely for Tool `0.3.7`;
- existing public profile examples are not version-bumped without a semantic contract reason;
- focused tests pass;
- full regression passes;
- package/distribution verification passes;
- ADR-0018 wheel artifact ownership is verified by an exact-SHA workflow run;
- required exact-SHA workflow verification succeeds;
- documentation/schema/spec assets are synchronized where actually required;
- final Tool `0.3.7` immutable Specification binding and supported PP compatibility declaration are frozen for release.

## 13. Entry state and inherited regression

WU-08 entered automatically under the project owner's standing successor-entry authorization after WU-07 was documented `COMPLETE / FOCUSED TEST VERIFIED` and exact `dev/0.3.7` HEAD `874fa80a508f5901647d6d2df132a95f0eadda49` was freshly revalidated.

This ACTIVE state is entry only. It does not by itself authorize release, Specification freeze, profile migration, or unrelated architecture changes.

The first known WU-08 regression came from owner-dispatched `tooling-test` workflow run `32825710974`, which verified exact source SHA `68b29308e8531f93267acc2aec585f6e751021f1` and ran the complete pytest suite:

```text
437 passed
1 failed
```

The sole failure was:

```text
tests/ptsip/test_repository_self_profile_035.py::
test_repository_self_profile_is_valid_complete_and_revision_pinned
```

because repository validation returned:

```text
18 tracked file(s) are outside declared component and associated-artifact selectors;
this is not automatically a PTSIP violation.
```

The repository owner selected the long-term-maintainability architecture: `ptsip-evidence`, `ptsip-source-compat`, and `ptsip-migration` became separate PRODUCT components under ADR-0018. Workflow run `32923347579` then verified exact SHA `fc1f479d34cb9531180bdf111bfe0695ba0fc48b` with `439 passed / 0 failed`, no validation warnings, and `unassigned_count == 0`. The inherited 18-file blocker is therefore CLOSED / VERIFIED.

## 14. Entry discipline

WU-08 verification evidence must record the exact source SHA it verifies; later documentation-only commits must not silently replace that verification authority.

ADR-0017 and ADR-0019 are mandatory WU-08 boundaries: Tool release verification must not silently reintroduce the superseded assumption that Tool `0.3.7` requires a repository-local Tool-numbered Project Profile.

## 15. ADR-0019 PP transition implementation boundary

The accepted Project Profile version model is:

```text
Tool Version
    0.3.7

Project Profile Contract Version
    pp.<major>.<minor>
    intended current generation: pp.1.01

Project Profile Instance Revision
    immutable revision/content identity of one concrete project declaration
```

The PP major rule is fixed by ADR-0019:

> **Project Profile major version은 governing lifecycle classification의 집합 또는 그 classification semantics가 변경되어 기존 프로젝트의 lifecycle 재분류를 요구할 수 있을 때 상승한다.**

`pp.0.00` names the legacy compatibility generation whose primary governing lifecycle boundary was `PRODUCT` / `TOOLCHAIN`; historical `NEUTRAL_CONTRACT` vocabulary remains recognized as the neutral/non-governing contract classification rather than being erased.

The current five-class governing model is:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

and the intended first formally numbered current PP contract is `pp.1.01`. WU-08 must not invent a historical published `pp.1.00` contract merely to make the numbering contiguous.

Before any real profile may adopt `pp.1.01`, WU-08 must implement and verify:

- canonical `pp` parser/serializer and numeric ordering;
- runtime/schema identity for `pp.1.01`;
- source-family compatibility mapping for historical labels;
- explicit Tool `0.3.7` compatibility matrix;
- migration source/target identity independent of Tool SemVer;
- PP-aware temporary/Final Point identity rules;
- fail-closed unsupported/unknown PP diagnostics;
- tests separating Tool release changes from PP migration responsibility;
- tests separating PP contract changes from ordinary Project Profile Instance revisions.

The existing `<major>.<minor>.<micro>-draft` transition parser and `ptsip_<major>.<minor>.<micro>.yaml` temporary naming remain historical implementation constraints until this boundary is intentionally replaced. Accepting ADR-0019 does not authorize a partial relabel that would make current validators reject the repository profile.