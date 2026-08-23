# WU-07 — Final Specification Freeze / Release Contract Strengthening / Release Preparation

> **Status:** ACTIVE — Strategy B release-contract strengthening implemented; final exact-SHA verification pending  
> **Parent release:** Tool `0.3.6` — Primary Lifecycle Ownership Foundation  
> **Entry branch:** `tool-0.3.6-lifecycle-ownership`  
> **Entry predecessor:** WU-06 — COMPLETE / EXACT-SHA VERIFIED  
> **WU-06 exact verification authority:** `a94c50130a694dba937708403520417719aec1e1`  
> **WU-06 verification run/job:** `32598727026 / 97093629017`  
> **WU-07 exact entry baseline:** `8b2c0819e10b58902a780a094a0f52c603c39fba`  
> **WU-07 entry-document commit:** `0e438e7544ebbfee1f1692086d31d9d6f539d297`  
> **Bound Specification at entry/current:** `0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea`  
> **Maintainer-selected strategy:** **B — Release Contract Strengthening**  
> **Release successor:** Tool `0.3.6` main-branch release preparation / publication boundary

## 0. Decision authority

Before WU-07 entry, three release-closure strategies were reviewed:

```text
A. Minimal Release Closure
   - smallest change set
   - fastest release preparation
   - rely mostly on the existing release contract

B. Release Contract Strengthening
   - preserve the existing release architecture
   - close identified release-contract gaps
   - verify bound Specification content, release-document consistency,
     and the actual publication artifact boundary

C. Build Once / Release Evidence Chain
   - redesign the release pipeline around one immutable built artifact,
     artifact manifest/attestation, and stronger end-to-end reuse
   - largest and longest-running change set
```

The maintainer explicitly selected **B**.

Therefore WU-07 MUST strengthen the current release contract without turning Tool `0.3.6` release closure into a broad release-infrastructure redesign. Strategy C remains a possible future improvement but is not a Tool `0.3.6` completion requirement.

## 1. Purpose

WU-07 closes Tool `0.3.6` development by converting the already verified implementation candidate into a release-ready, explicitly frozen Tool/Specification/release-contract state.

WU-06 proved the implementation/package boundary on exact SHA `a94c50130a694dba937708403520417719aec1e1`, including complete regression, distribution construction, Product Artifact evidence, exact artifact snapshot binding, installed-wheel smoke, VPMS compatibility, and successful `self-hosted/tooling-test` status.

WU-07 does **not** repeat WU-06 merely for ceremony. It owns the remaining release-preparation questions:

```text
Is the final Specification binding still normatively correct?
Does current release source contain exactly the normative content claimed by SPEC_REVISION?
Are release-facing repository documents internally consistent?
Does Tool 0.3.6 have a reviewed release note before release preparation?
Does the actual publication build receive the required Product Artifact boundary verification?
Does the existing exact-SHA release chain still reject stale/mismatched release candidates?
Is the final release source ready for main-branch exact-SHA verification and draft release creation?
```

## 2. Entry authority

WU-06 completed technical and administrative closure with exact verification authority:

```text
WU-06 entry baseline:        bdd77ef3f026a83bb0fd0099144aebea1de55365
WU-06 verification SHA:      a94c50130a694dba937708403520417719aec1e1
workflow:                    tooling-test
run/job:                     32598727026 / 97093629017
pytest:                      327 passed
self-hosted/tooling-test:    success
Product Artifact evidence:   PASS
artifact snapshot binding:   PASS
PTSIP-PKG-001 violations:    0
```

WU-06 closure documentation was then propagated through the release-level plan/status records. A fresh development-branch HEAD read immediately before creating this WU-07 document returned:

```text
8b2c0819e10b58902a780a094a0f52c603c39fba
```

That SHA is the **WU-07 exact entry baseline**.

This WU-07 stage document is a descendant planning record and MUST NOT replace that entry baseline.

## 3. Entry review observations

The WU-07 pre-entry review established the following release-facing facts.

### 3.1 Existing release preparation is already exact-SHA gated

`.github/workflows/release.yml` already:

```text
requires manual workflow_dispatch
requires dispatch from main
checks out exact github.sha
requires checked-out SHA == dispatched SHA
refreshes origin/main
requires current origin/main == candidate SHA
derives package version and tag from repository state
requires releasenote/<version>.md
requires exact self-hosted/tooling-test success on candidate SHA
runs verify_release_contract.py
reconfirms main has not moved
creates a draft GitHub Release targeting the same exact SHA
```

WU-07 MUST preserve these properties.

### 3.2 Existing publication workflow builds again after GitHub Release publication

`.github/workflows/tooling-release.yml` is triggered by:

```text
release: published
```

It does not create the release note. It checks out the published Tool tag, verifies Tool/Specification release identity, builds sdist/wheel on self-hosted Windows, checks distribution metadata/content, uploads the distributions, and uses a narrow GitHub-hosted GNU/Linux job only for PyPI Trusted Publishing.

Therefore the Tool release note MUST exist and be reviewed/committed before `release.yml` creates the draft GitHub Release.

### 3.3 Tool `0.3.6` release note is not yet present at entry

At WU-07 entry, `releasenote/0.3.6.md` is absent.

The repository release policy allows `git-cliff` to generate an initial Tool release-note draft, but the generated result is only a starting point. The final Tool release note MUST be reviewed and curated before release preparation.

### 3.4 Operational repository guidance contains stale development state

At entry, `MEMORY.md` and `AGENTS.md` still contain historical WU-04G-era active-stage statements and older Tool `0.3.6` work-unit ordering/binding context that no longer describes the actual release stage.

Those documents are not normative Specification authority, but stale instructions are release-readiness defects because coding agents and maintainers are explicitly told to read them before work.

WU-07 therefore owns a bounded operational-context refresh.

### 3.5 Release-note index is stale for the current development family

At entry, `releasenote/README.md` records Tool history through `0.3.5` and still identifies `0.3.4-draft` as the current active Specification family.

WU-07 owns updating release-history/index text to accurately describe the Tool `0.3.6` pre-publication state and bound `0.3.6-draft` family without rewriting published historical release documents.

### 3.6 Existing Specification release contract proves identity/existence but not complete content freeze

`.github/scripts/verify_release_contract.py` already verifies:

```text
package version -> expected <version>-draft Specification family
SPEC_REVISION full commit SHA shape
ptsip.yaml Tool/Specification binding consistency
Specification release-note presence
SPEC_REVISION resolves to a commit
SPEC_REVISION is an ancestor of release source HEAD
required normative Specification paths exist at SPEC_REVISION
```

However, existence-at-revision plus ancestor relationship does not by itself prove that the corresponding normative content in release source HEAD is byte/content-identical to the bound immutable snapshot.

WU-07 Strategy B closes this release-contract gap.

### 3.7 No current normative drift was established during pre-entry review

A `d6995ed232e845b88d8235b851e80ab54b7804ea -> WU-07 entry baseline` repository comparison did not show current changes to the canonical `spec/` normative documents, canonical schemas, or registry files. Therefore WU-07 begins with **no pre-established normative defect**.

The release contract is strengthened to prevent future unbound drift; it MUST NOT manufacture a new `SPEC_REVISION` merely because the guard is being improved.

## 4. Selected Strategy B boundary

WU-07 Strategy B combines the necessary release closure with focused contract hardening.

It includes:

1. final normative review of the bound `0.3.6-draft` snapshot;
2. release-facing document closure;
3. stronger immutable Specification content verification;
4. stronger actual publication-artifact verification;
5. release-readiness regression and exact-SHA evidence;
6. final release-preparation handoff.

It explicitly does **not** include a Build-Once/Publish-the-Same-Bits redesign.

The existing release architecture remains:

```text
development branch closure
    -> merge to main
    -> tooling-test.yml on exact main SHA
    -> self-hosted/tooling-test success on exact main SHA
    -> release.yml on same current main SHA
    -> release contract verification
    -> draft GitHub Release targeting same SHA
    -> maintainer publishes reviewed draft
    -> tooling-release.yml on published tag
    -> self-hosted distribution build/verification
    -> minimal GitHub-hosted PyPI Trusted Publishing
```

## 5. WU-07 scope

WU-07 owns the following work.

### 5.1 Final Specification review and freeze decision

Review the bound normative family:

```text
Specification: 0.3.6-draft
SPEC_REVISION: d6995ed232e845b88d8235b851e80ab54b7804ea
```

The review MUST distinguish:

```text
normative correctness defect
implementation/release-contract defect
documentation drift
release-note/index drift
future migration concern
```

If no genuine normative defect exists, WU-07 MUST retain `d6995ed...` as the Tool `0.3.6` release binding.

A new immutable Specification revision is permitted only if the review finds a real normative defect requiring a normative change. Release workflow improvements, release-note creation, stale operational-document cleanup, test additions, or publication-artifact checks MUST NOT move `SPEC_REVISION` by themselves.

### 5.2 Tool `0.3.6` release note

Create and review:

```text
releasenote/0.3.6.md
```

The release note SHOULD be generated initially from final Tool history when the repository's existing `git-cliff` path is usable, but generated commit lists MUST be curated into a human-readable release record.

The final note MUST explain at least:

```text
Primary Lifecycle Ownership five-classification model
Responsibility Map v2
explicit/template/hybrid declaration modes
Canonical Effective Responsibility Map
clarification/adoption integration
VPMS narrow read-only compatibility boundary
repository dogfood result
full regression/package/distribution verification
Product Artifact evidence and exact binding
important compatibility/migration boundary
Tool 0.3.6.1 migration continuation
release verification evidence appropriate for users/maintainers
```

The note MUST NOT claim that Tool `0.3.6.1` migration functionality ships in Tool `0.3.6`.

The release note MUST be committed before `release.yml` release preparation.

### 5.3 Release-facing repository context cleanup

Update only stale release/current-state information in:

```text
MEMORY.md
AGENTS.md
releasenote/README.md
```

Objectives:

```text
remove obsolete WU-04G-as-active instructions
record current Tool 0.3.6 release-stage context
record current bound Specification revision
remove or clearly mark retired WU numbering/state
preserve published Tool 0.3.5 history
preserve normative/non-normative authority boundaries
preserve self-hosted/exact-SHA release policy
preserve Tool 0.3.6.1 migration as post-0.3.6 work
```

This is not authorization to rewrite the repository's architecture policy or published history.

### 5.4 Strengthen immutable Specification content verification

Strengthen `.github/scripts/verify_release_contract.py` so release readiness proves not only that `SPEC_REVISION` exists and contains required paths, but that the release source does not silently claim a bound snapshot while carrying modified normative content.

The strengthened guard MUST compare the release-source normative asset set against the exact bound revision.

At minimum the review must cover the canonical normative family used by Tool `0.3.6`, including the required Specification documents already recognized by the release contract:

```text
spec/PTSIP-SPEC.md
spec/PTSIP-CONFORMANCE.md
spec/PTSIP-TERMINOLOGY.md
spec/PTSIP-GOVERNANCE.md
spec/PTSIP-RESPONSIBILITY-MAP.md
```

The WU-07 implementation review MUST also enumerate the canonical machine-readable normative assets that participate in the Tool/Specification binding and determine which of them must be exact-content-bound to `SPEC_REVISION`, including applicable canonical schemas/registry resources and their embedded package copies.

Required behavior:

```text
for each release-bound normative asset
    -> verify asset exists at SPEC_REVISION
    -> verify release-source content matches the bound revision
    -> fail closed on missing/mismatched content
```

Canonical-vs-embedded equality remains a separate check from bound-revision equality; both may be required.

No guard may infer that a changed normative file is harmless. If a normative asset differs from `SPEC_REVISION`, release preparation MUST stop until either:

```text
A. unintended drift is reverted
or
B. a genuine normative change is reviewed and a new immutable revision is explicitly selected
```

### 5.5 Strengthen release-document consistency checks

Extend release-readiness verification so obvious release-state contradictions fail before publication.

The gate SHOULD verify deterministic facts such as:

```text
releasenote/<package-version>.md exists and is non-empty
Tool release note contains meaningful categorized sections
Specification release note exists
release-note index recognizes the current Tool/Specification development family where required
current package/Tool/Specification identities agree
release-facing current-state documents do not still declare a retired active WU
```

The implementation MUST avoid brittle prose-policing. It should test stable markers/identities rather than require exact paragraphs.

### 5.6 Verify the actual publication artifact boundary

WU-06 verified a wheel produced by `tooling-test.yml`. The publication workflow later rebuilds distributions after GitHub Release publication.

Strategy B therefore requires the actual publication build in `.github/workflows/tooling-release.yml` to receive a Product Artifact boundary/evidence check equivalent in release meaning to the WU-06 requirement.

The publication build MUST establish at least:

```text
published tag matches package version
checked-out commit is exactly pointed to by published tag
Tool/Specification release contract passes
sdist + wheel build succeeds
wheel metadata Name/Version are correct
required console entry points are present
required ptsip package content is present
required vpms compatibility package is present
required embedded specdata is present
LICENSE is present
unexpected Product-distribution boundary content fails closed
Product Artifact evidence is generated from the actual publication wheel
artifact evidence is bound to the exact tagged repository snapshot
PTSIP-PKG-001 definite Product-package violation count is zero
twine check passes
```

If the existing WU-06 artifact-verification logic would otherwise be duplicated unsafely, WU-07 MAY extract only that bounded verification logic into a reusable repository script used by both `tooling-test.yml` and `tooling-release.yml`.

Such an extraction is allowed only when it preserves behavior and remains narrower than Strategy C. It MUST NOT redesign the release pipeline into a one-build artifact transport/attestation system.

### 5.7 Release workflow exact-SHA contract review

Review `.github/workflows/release.yml` and preserve at least:

```text
manual workflow_dispatch
no manual source_sha/version/host_ready inputs
main-only release preparation
exact github.sha checkout
origin/main freshness check
exact self-hosted/tooling-test status requirement
release contract verification
second origin/main freshness check before draft creation
draft release targets same source SHA
no repository mutation by release.yml
```

Only narrow corrections established by WU-07 findings are authorized.

### 5.8 Release-readiness tests

Extend focused tests, primarily the existing release-readiness contract tests, to pin the strengthened behavior.

Tests must cover at least:

```text
bound normative content mismatch -> release contract FAIL
bound normative content match -> PASS path remains possible
release contract still requires exact immutable revision
publication workflow performs strengthened artifact verification
release workflow still requires exact self-hosted status
release note/current-version markers are required without brittle full-text matching
```

Do not replace final complete repository regression with focused release-readiness tests.

## 6. Explicit non-goals

WU-07 does **not** own:

```text
Tool 0.3.6.1 candidate discovery implementation
Tool 0.3.5 legacy reader implementation
migration analyzer / proposal / apply work
new lifecycle classifications
Responsibility Map v2 redesign
VPMS vocabulary migration
Build Once / publish identical previously-built bits redesign
SLSA/provenance platform rollout
new external signing infrastructure
new release orchestration service
automatic GitHub Release publication without maintainer review
PyPI publication itself during development-branch WU execution
rewriting historical tagged release notes
rewriting Tool 0.3.5 history
```

Strategy C remains explicitly deferred.

## 7. Release-note generation boundary

`tooling-release.yml` MUST NOT be treated as a release-note generator. It runs only after a GitHub Release is published.

The Tool `0.3.6` release-note flow is:

```text
final Tool history
    -> optional git-cliff initial draft
    -> maintainer/release-stage curation
    -> releasenote/0.3.6.md committed
    -> release.yml reads the reviewed file
    -> release.yml creates draft GitHub Release using that file as body
    -> maintainer publishes reviewed draft
    -> tooling-release.yml builds/verifies/publishes package distributions
```

WU-07 MAY automate initial draft generation where existing repository tooling already supports it, but the final reviewed note remains repository-owned release evidence.

## 8. Specification freeze model

WU-07 uses a fail-closed freeze decision:

```text
review current bound Specification
    |
    +-- no genuine normative defect
    |      -> retain 0.3.6-draft @ d6995ed...
    |      -> strengthen content-equality release guard
    |
    +-- genuine normative defect
           -> stop ordinary release closure
           -> document exact normative defect
           -> modify canonical + embedded normative assets coherently
           -> select new immutable Specification revision
           -> update Tool/root binding
           -> rerun all affected verification
```

The second path is exceptional and requires explicit evidence. It is not the default WU-07 objective.

## 9. Failure classification

Every WU-07 finding MUST be classified before modification:

```text
A. normative Specification defect
B. Specification-binding / content-freeze guard defect
C. Tool release-note defect
D. operational current-state documentation drift
E. release workflow exact-SHA contract defect
F. publication artifact verification defect
G. release-readiness test/fixture defect
H. package/distribution defect discovered during final verification
I. Tool 0.3.6 implementation regression
J. deferred Strategy-C release-infrastructure improvement
K. deferred Tool 0.3.6.1 migration concern
```

A, B, C, D, E, F, G and any genuine H/I release blocker are WU-07-owned.

J/K are explicitly deferred unless a discovered problem proves release correctness is impossible without them.

## 10. Change discipline

WU-07 is a release-closure stage, not an open-ended cleanup tranche.

Rules:

- modify only files necessary for the selected Strategy B contract;
- retain exact lifecycle architecture semantics already verified by WU-06;
- do not change `ptsip.yaml` merely to satisfy prose/current-state cleanup;
- do not move `SPEC_REVISION` for non-normative changes;
- do not weaken fail-closed conformance or release checks;
- do not replace exact-SHA verification with branch-name trust;
- do not hard-code a self-hosted machine name;
- do not move package build/test work to GitHub-hosted runners except the existing narrow PyPI Trusted Publishing boundary;
- do not create a parallel release workflow where the existing workflow can be narrowly strengthened;
- do not create Tool `0.3.6.1` implementation commits during WU-07.

## 11. Verification model

The intended WU-07 flow is:

```text
WU-07 exact entry baseline
    -> final normative review
    -> classify findings
    -> retain d699... unless a genuine normative defect exists
    -> create/curate releasenote/0.3.6.md
    -> refresh stale release-facing repository context
    -> strengthen Specification content freeze guard
    -> strengthen release-document consistency checks
    -> strengthen publication-artifact verification
    -> focused release-readiness tests
    -> complete repository pytest
    -> Tool/package/Specification identity verification
    -> repository self-profile verification
    -> build/twine/package smoke as affected
    -> exact-SHA tooling-test on final development candidate
    -> review exact status
    -> record WU-07 completion evidence
    -> prepare main-branch release handoff
```

Any source-changing correction creates a new candidate SHA. Evidence from an earlier candidate MUST NOT be reused as final WU-07 authority.

## 12. Development-branch completion gate

WU-07 development-branch work is complete only when all applicable conditions below hold:

- final normative review records whether `d6995ed...` remains the correct Tool `0.3.6` binding;
- no unresolved genuine normative defect exists;
- release contract rejects bound normative content drift;
- canonical/embedded release-bound contracts remain coherent;
- `releasenote/0.3.6.md` exists and is reviewed;
- `MEMORY.md`, `AGENTS.md`, and `releasenote/README.md` no longer carry materially stale release-stage state;
- actual publication workflow contains the required Product Artifact evidence/binding verification contract;
- release workflow retains exact-SHA/main freshness/self-hosted status requirements;
- focused release-readiness tests pass;
- complete repository pytest passes with zero failures;
- Tool/package/Specification identities remain coherent;
- repository self-profile remains valid and coverage-complete;
- package/distribution checks affected by WU-07 pass;
- final development candidate receives successful exact-SHA `self-hosted/tooling-test` status;
- all WU-07 findings/corrections are recorded;
- no Strategy C redesign or Tool `0.3.6.1` implementation is entered.

## 13. Main-branch release handoff

WU-07 completion on the development branch does not itself publish Tool `0.3.6`.

After WU-07 completion, release preparation follows the repository's exact-SHA boundary:

```text
merge approved Tool 0.3.6 release state to main
    -> read fresh exact main HEAD
    -> dispatch tooling-test.yml from main
    -> require self-hosted/tooling-test success on that exact main SHA
    -> ensure no repository mutation after exact verification
    -> dispatch release.yml from the same current main SHA
    -> require release contract PASS
    -> create draft GitHub Release targeting that SHA
    -> maintainer reviews/publishes draft
    -> tooling-release.yml runs on published tag
    -> self-hosted publication distribution verification
    -> minimal GitHub-hosted PyPI Trusted Publishing
```

A successful development-branch WU-07 run MUST NOT be substituted for the final exact `main` release gate.

## 14. Completion evidence to record

The WU-07 completion record must capture at least:

```text
exact WU-07 entry baseline
final development verification SHA
selected strategy B authority
final Specification family/revision decision
normative content-freeze verification result
Tool release-note path/review state
release-facing documentation cleanup result
release workflow exact-SHA review result
publication artifact verification result
focused release-readiness test result
complete pytest result
Tool/package/Specification identity
repository self-profile result
build/twine/distribution smoke as applicable
self-hosted workflow run/job
exact self-hosted/tooling-test status
all discovered findings and ownership classifications
all WU-07 corrections
remaining deferred Strategy C / Tool 0.3.6.1 concerns, if any
```

Documentation commits after successful exact-SHA verification record the result and do not replace the exact source verification authority.

## 15. Initial execution sequence

Execute Strategy B in this order:

```text
1. confirm WU-07 exact entry baseline and current descendant
2. audit bound Specification for genuine normative defects
3. enumerate release-bound normative asset set
4. determine/implement exact bound-revision content checks
5. create and curate releasenote/0.3.6.md
6. update stale MEMORY.md / AGENTS.md / releasenote/README.md state
7. strengthen deterministic release-document consistency checks
8. extend publication workflow Product Artifact evidence/binding verification
9. add/update focused release-readiness tests
10. run focused release-contract tests
11. run complete repository pytest
12. validate Tool/package/Specification identity and repository self-profile
13. build/check distributions as affected by WU-07 changes
14. classify and correct only WU-07-owned failures
15. repeat affected checks on each new candidate SHA
16. run exact-SHA self-hosted tooling-test on final development candidate
17. verify exact commit status and collect run/job evidence
18. record WU-07 completion
19. only after completion, perform separate exact-main release handoff
```

## 16. Entry state

This document fixes the WU-07 exact entry baseline at:

```text
8b2c0819e10b58902a780a094a0f52c603c39fba
```

and records the maintainer-selected strategy as:

```text
B — Release Contract Strengthening
```

WU-06 remains `COMPLETE / EXACT-SHA VERIFIED` with technical verification authority `a94c50130a694dba937708403520417719aec1e1`.

WU-07 activation has now been propagated to `planning/0.3.6.md` and `STATUS.md`. Descendant activation and implementation records do not replace the exact WU-07 entry baseline.

## 17. Executed audit and implementation progress

### 17.1 Final Specification audit

WU-07 compared the bound immutable revision against the release-entry/current development line and found no established modification to the canonical `spec/`, `schemas/`, or `registry/` normative assets requiring a new normative snapshot.

Classification:

```text
A. normative Specification defect: NONE FOUND
```

Final freeze decision at this stage:

```text
Specification: 0.3.6-draft
SPEC_REVISION: d6995ed232e845b88d8235b851e80ab54b7804ea
revision movement: NONE
```

The release-bound asset set was explicitly enumerated as 15 Git objects:

```text
5 canonical Specification Markdown documents
5 canonical machine-readable schema/registry assets
5 embedded package copies of those machine-readable assets
```

At audit time the applicable bound machine-readable assets were blob-identical to their `d6995ed...` counterparts and canonical/embedded pairs were coherent.

### 17.2 Specification content-freeze guard

Finding classification:

```text
B. Specification-binding / content-freeze guard defect
```

Corrections:

```text
2862b9547fcb62292619aa8c189a4cacc9e5d62e
release: bind specification content to exact revision

1978b374d3a88001c38537b39efa943cb6d75e67
release: verify release-note consistency
```

`verify_release_contract.py` now uses Git object identity to prove that every release-bound asset exists both at `SPEC_REVISION` and release `HEAD` and that `HEAD:path` equals `SPEC_REVISION:path`. Git blob identity is used rather than checkout byte comparison so the guard is not weakened by platform line-ending behavior.

Canonical/embedded machine-readable pairs are also compared independently at release `HEAD`.

The contract additionally requires stable Tool/Specification release-document facts: a non-empty categorized Tool release note with exact Tool/Specification binding, an exact-bound Specification note, and Tool/Specification markers in the release-note index.

### 17.3 Tool release note and release-note index

Finding classifications:

```text
C. Tool release-note defect
D. operational current-state documentation drift
```

Corrections include:

```text
ae2e7e2dee7b5a14fc44592553e46a26ee13e9d9
docs: add Tool 0.3.6 release note

63e8286f401bb62e61374aae6d69ca9987327800
docs: update 0.3.6 release-note index

32f9f3a155c17466b1bfd97fa82918fe619e30c5
docs: stabilize Tool 0.3.6 pre-publication note

6bc4d5ccd901242b91f7945e1f92e48980ae1d22
docs: stabilize 0.3.6 release-note index state
```

`releasenote/0.3.6.md` is now a curated pre-publication release record covering the five-classification model, Responsibility Map v2, explicit/template/hybrid materialization, effective-map consumption, clarification/adoption, VPMS boundary, dogfood and WU-06 exact verification, Product Artifact evidence, release-contract strengthening, and Tool `0.3.6.1` migration continuation.

Its state wording is deliberately stable across final verification: **pre-publication release candidate — not yet published**. Publication state is finalized only at the actual release boundary.

### 17.4 Operational current-state cleanup

Finding classification:

```text
D. operational current-state documentation drift
```

Corrections:

```text
46c1238bb43a04d1b4bd044bed0d27a94549603e
docs: refresh coding-agent WU-07 context

9e3f304adb70ae52e03e7d262ec26caac57af51c
docs: refresh WU-07 working memory
```

`AGENTS.md` and `MEMORY.md` no longer direct coding agents to treat WU-04G as the active stage or `82abd...` as the current Tool binding. Current operational context identifies WU-07 Strategy B and `d6995ed...` while preserving the canonical lifecycle/materialization/decision/VPMS/release policies and published Tool `0.3.5` history.

### 17.5 Publication Product Artifact verification

Finding classification:

```text
F. publication artifact verification defect
```

Correction:

```text
122754053d65b77ded02b4d25b833701c7343789
release: verify publication product artifact boundary
```

The published-tag build in `tooling-release.yml` now verifies the newly built publication wheel before it is uploaded for PyPI publishing. The step checks exact tagged source identity, package metadata, entry points, expected Product package content, embedded contracts, LICENSE, and fail-closed unexpected wheel content.

It force-installs the actual publication wheel, generates `ptsip-artifact-evidence/v1`, binds the evidence to the exact tagged repository snapshot using `ptsip-artifact-evidence-binding/v1`, runs artifact-aware conformance, requires `product_artifact_boundary` and `artifact_snapshot_binding` to run, requires valid binding, rejects artifact-evidence blocking gaps, and requires zero definite `PTSIP-PKG-001` Product-package violations.

The existing release architecture is preserved; Strategy C Build Once/attestation redesign was not entered.

### 17.6 Release-readiness regression contracts

Finding classification:

```text
G. release-readiness test/fixture defect
```

Corrections:

```text
6704371850789961d6b75b55e1c318faf463d365
test: pin WU-07 specification content freeze

2ed5caec790bcb110192bfcaa8acd734ff315954
test: require publication artifact verification

f709570ca69f909830b4495953dd7b06da515341
test: pin Tool 0.3.6 release documents

86fbb77e8ae60c3f72ca24c7033328b0a7afd24e
test: reject stale WU-04G release context

220ab34f2de9fb76607f2c9847aab0c4ad2857ee
test: pin exact main release preparation gate
```

The focused release-readiness contract now pins:

```text
15 release-bound Specification assets
exact bound asset equality PASS path
simulated bound-content drift FAIL path
canonical/embedded equality
Tool 0.3.6 release-note exact binding markers
release-note index Tool/Specification markers
WU-07 operational-context markers and retired WU-04G ACTIVE marker rejection
publication Product Artifact evidence/binding requirements
release.yml exact main freshness + same-SHA draft release invariants
```

### 17.7 Release workflow exact-SHA review

Classification:

```text
E. release workflow exact-SHA contract defect: NONE FOUND
```

No semantic change to `release.yml` was required. Its existing manual main-only dispatch, exact `${{ github.sha }}` checkout, first `origin/main` freshness check, exact `self-hosted/tooling-test` status requirement, release-contract invocation, second freshness check, and same-SHA draft release target remain the release-preparation authority.

### 17.8 Responsibility Map / architecture impact

No WU-07 implementation so far has required changes to:

```text
ptsip.yaml
canonical lifecycle classifications
Responsibility Map v2 semantics
production Tool architecture
VPMS vocabulary
Tool 0.3.6.1 migration implementation
canonical Specification assets
SPEC_REVISION
```

Changed paths are already covered by existing project-owned selectors for release automation, verification, maintenance/planning, and Product documentation.

### 17.9 Remaining gate

Repository execution evidence has not yet been generated for the post-WU-07 implementation candidate. Therefore this document does **not** claim focused tests, full pytest, package build, or final exact-SHA self-hosted verification have passed after these changes.

The next authority-producing action is:

```text
fresh final development candidate SHA
    -> tooling-test.yml workflow_dispatch on that exact SHA
    -> complete repository pytest
    -> Tool/package/Specification/self-profile checks
    -> distribution/Product Artifact checks
    -> exact self-hosted/tooling-test status
    -> review run/job evidence
```

WU-07 remains ACTIVE until that final exact-SHA gate succeeds and completion evidence is recorded.