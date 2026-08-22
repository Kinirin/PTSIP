# WU-06 — Full Regression / Package / Distribution Verification

> **Status:** ENTRY DOCUMENT CREATED — activation pending release-level/status propagation  
> **Parent release:** Tool `0.3.6` — Primary Lifecycle Ownership Foundation  
> **Entry branch:** `tool-0.3.6-lifecycle-ownership`  
> **Entry predecessor:** WU-05 — COMPLETE / DOGFOOD REVIEWED  
> **WU-05 exact dogfood verification snapshot:** `5010fa81ebd104eb329f29217d4c5fecec51cacb`  
> **WU-06 exact entry baseline:** `bdd77ef3f026a83bb0fd0099144aebea1de55365`  
> **Bound Specification:** `0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea`  
> **Successor:** WU-07 — final Specification freeze / release preparation; locked until WU-06 completion

## 0. Purpose

WU-06 verifies that the Tool `0.3.6` release candidate is regression-clean, distributable, installable, and reproducibly tied to one exact repository SHA.

WU-05 established that the repository can consume the Tool `0.3.6` architecture foundation without project-owned architecture reinterpretation or mutation. WU-06 is different: it closes the release-verification gap by exercising the complete repository regression and the actual package/distribution boundary.

WU-06 does not change architecture merely to obtain a green result. It verifies the implementation and built Product Artifact against the already-bound Tool/Specification contract, fixes only WU-06-owned release-verification defects when necessary, and records exact-SHA evidence suitable for WU-07 entry.

## 1. Entry authority

WU-05 completed repository dogfood against exact snapshot:

```text
5010fa81ebd104eb329f29217d4c5fecec51cacb
```

with the following reviewed result:

```text
repository declaration:             valid
Effective Responsibility Map:       coherent / unassigned_count=0
conformance:                         ENFORCED / INCOMPLETE / reviewed
definite architecture violations:   0
conformance report audit:            PASS
clarification:                       NO_CLARIFICATION_REQUIRED / STABLE
local gate:                          NO_DECISION_REQUIRED / LOCAL
post-gate working tree:              CLEAN
confirmed WU-05 A-C defects:         NONE
normative Specification defect:      NONE
```

WU-05 then recorded closure documentation. A fresh development-branch HEAD read immediately before this WU-06 stage document was created returned:

```text
bdd77ef3f026a83bb0fd0099144aebea1de55365
```

That SHA is the exact WU-06 entry baseline. This stage document is a descendant planning record and does not replace the entry baseline.

## 2. WU-05 handoff owned by WU-06

WU-05 intentionally did not manufacture Product Artifact evidence merely to force Enforced Conformance to green.

The release-closure handoff is:

```text
Product Artifact evidence unavailable during repository dogfood
    -> actual sdist / wheel build in WU-06
    -> inspect distribution contents and metadata
    -> establish exact artifact/repository snapshot binding
    -> evaluate release-relevant conformance evidence
```

The following WU-05 observations are **not** pre-classified as WU-06 implementation defects:

```text
OBS-01
Python distribution/import-name mismatch such as PyYAML -> yaml and PyJWT -> jwt
may remain unresolved by native dependency evidence and correctly fail closed.

OBS-02
Independent-build evaluation requires sufficient component-declared build/dependency evidence;
missing evidence may correctly keep strict conformance INCOMPLETE.
```

WU-06 must not rewrite `ptsip.yaml`, weaken fail-closed behavior, or broaden dependency inference merely to remove those observations unless separate evidence establishes an actual Tool `0.3.6` correctness defect.

## 3. Scope

WU-06 owns release-candidate verification for the Tool `0.3.6` source tree and its built Python distributions.

The stage must cover at least:

1. **exact source identity** — every authoritative verification run is tied to one exact commit SHA;
2. **complete repository regression** — the full pytest collection passes with zero failures on the verification candidate;
3. **Tool / package / Specification identity** — Tool `0.3.6`, package version `0.3.6`, `0.3.6-draft`, and immutable `SPEC_REVISION` remain coherent;
4. **repository self-profile validation** — root `ptsip.yaml` remains valid and complete under the canonical Effective Responsibility Map;
5. **canonical/embedded contract coherence** — packaged schema/registry/specdata assets remain synchronized with their canonical bound sources where the existing tests/contracts require equality;
6. **distribution construction** — both sdist and wheel build successfully from the exact candidate source;
7. **distribution-content review** — required Product package content is present and unintended repository-only / lifecycle-only implementation is not silently included;
8. **Product Artifact evidence** — actual built Product distribution evidence is sufficient to evaluate the release-relevant Product Artifact boundary;
9. **artifact snapshot binding** — artifact evidence is tied reproducibly to the exact repository revision/content snapshot used to build it;
10. **metadata validation** — `twine check` succeeds for produced distributions;
11. **installed-wheel verification** — the built wheel can be force-reinstalled independently of editable source installation and public surfaces still work;
12. **CLI/package smoke** — installed `ptsip` identity, Specification reporting, and required public conformance surfaces remain callable;
13. **VPMS compatibility smoke** — packaged `vpms` remains importable and its compatibility vocabulary remains intact without becoming PTSIP classification authority;
14. **self-hosted exact-SHA verification** — the final WU-06 candidate receives successful existing `self-hosted/tooling-test` verification on the exact SHA.

## 4. Non-goals

WU-06 does **not** own:

```text
Tool 0.3.6.1 migration/discovery implementation
Tool 0.3.5 legacy-reader implementation
lifecycle ontology redesign
Responsibility Map v2 redesign
template auto-selection or evidence-to-authority promotion
blind TOOLCHAIN -> DEVELOPMENT_TOOLING migration
final Specification freeze
release tag creation
GitHub Release publication
PyPI publication
main-branch release execution
WU-07 release preparation
```

WU-06 also does not require an unrelated refactor merely because a verification surface could be made cleaner.

A new Specification revision is not created for ordinary regression, package, workflow, fixture, or documentation corrections. `SPEC_REVISION` may move only after a separately established genuine normative defect and explicit Specification review.

## 5. Verification model

The intended release-candidate flow is:

```text
WU-06 exact entry baseline
    -> establish candidate SHA
    -> verify Tool/package/Specification identity
    -> validate repository self-profile
    -> run complete repository regression
    -> build sdist + wheel
    -> inspect distribution contents / metadata
    -> establish Product Artifact evidence + exact snapshot binding
    -> run release-relevant conformance review
    -> twine check
    -> reinstall built wheel
    -> CLI + Specification + VPMS smoke
    -> exact-SHA self-hosted tooling-test
    -> review exact commit status
    -> classify/remediate only WU-06-owned defects
    -> repeat affected checks on a new candidate SHA if source changes
    -> record WU-06 completion authority
```

An entry SHA, a local candidate SHA, and a final exact verification SHA are distinct concepts. A source correction after a failed check creates a new candidate and requires the affected verification evidence to be regenerated against that new exact SHA.

## 6. Required invariants

### R1 — source and verification identity

Authoritative evidence must identify the exact source revision it verifies.

Expected final condition:

```text
workflow-dispatch github.sha
    == checked-out repository HEAD
    == recorded WU-06 verification SHA
    == SHA receiving self-hosted/tooling-test success
```

A successful run against another SHA does not satisfy WU-06.

### R2 — regression cleanliness

The complete repository pytest collection must finish with:

```text
failures = 0
exit code = 0
```

Focused tests may assist diagnosis but cannot replace the final full-repository regression.

### R3 — identity coherence

The release candidate must remain coherent across at least:

```text
pyproject.toml project.version = 0.3.6
PTSIP Tool version             = 0.3.6
Specification family           = 0.3.6-draft
SPEC_REVISION                  = d6995ed232e845b88d8235b851e80ab54b7804ea
```

No release verification correction may silently change the normative binding.

### R4 — self-profile remains valid

The exact verification candidate must preserve the canonical repository declaration contract.

Expected minimum:

```text
valid = true
errors = []
warnings = []
materialized = true
responsibility_map_coverage.unassigned_count = 0
```

If a verification correction changes repository structure, profile coverage must be reevaluated rather than assumed.

### R5 — built distributions represent the intended Product Artifact

At least one sdist and one wheel must be produced from the exact candidate source.

The review must distinguish:

```text
required packaged Product implementation / metadata
required embedded PTSIP contract assets
required VPMS compatibility package
repository-only verification/planning/governance surfaces
Development Tooling / Delivery / Operations implementation
```

Artifact ownership and artifact producer ownership remain separate. A build or delivery mechanism may produce a Product Artifact without itself becoming Product-owned.

### R6 — artifact evidence is snapshot-bound

Product Artifact evidence used for strict release-relevant conformance must identify the artifact contents sufficiently and bind those facts to the exact source snapshot from which the artifact was built.

Evidence from an earlier source revision must not be reused after source or package-content changes.

### R7 — installed-wheel independence

The final smoke test must run against the built wheel after force reinstall, not merely against an editable source checkout.

The installed distribution must expose the required public entry points and packaged data without relying on repository-local accidental state.

### R8 — existing workflow remains the primary verification path

The repository already provides `.github/workflows/tooling-test.yml` with exact `github.sha` checkout, self-hosted Windows X64 execution, Python 3.14 preparation, full pytest, distribution build, `twine check`, wheel reinstall smoke, VPMS smoke, and exact-SHA success status recording.

WU-06 should extend or narrow-correct that existing workflow only if the stage review establishes that a required WU-06 verification contract is missing. A new parallel workflow is not created merely to duplicate the same verification path.

### R9 — failure classification precedes modification

Any WU-06 failure must be classified before code/profile/spec/workflow modification.

```text
A. production implementation defect
B. package/distribution configuration defect
C. verification/test/fixture defect
D. workflow/exact-SHA evidence defect
E. repository declaration / Responsibility Map defect
F. artifact-evidence generation/binding defect
G. WU-07-owned final freeze/release-preparation concern
H. deferred Tool 0.3.6.1 migration concern
I. genuine normative Specification defect
```

Only defects necessary to satisfy the WU-06 verification contract are corrected in this stage.

## 7. Baseline verification surfaces

The existing `tooling-test.yml` already provides these release-candidate checks:

```text
exact dispatched SHA checkout
self-hosted Windows X64
Python 3.14 isolated environment
editable verification install
complete repository pytest
Tool / Specification identity smoke
repository validate / clarify / local gate smoke
sdist + wheel build
twine check
force reinstall built wheel
installed CLI smoke
VPMS compatibility smoke
self-hosted/tooling-test success status on exact SHA
```

WU-06 must review, not assume, whether this is sufficient for the stage completion gate. In particular, WU-05 handed off explicit Product Artifact / artifact snapshot-binding evidence, while the current workflow's ordinary build/smoke steps alone do not automatically prove that conformance evidence contract.

If a workflow change is required, it should be the smallest reusable change to the existing verification path and remain capability-bound to eligible self-hosted runners rather than hard-coding a machine name.

## 8. Distribution-content review

The package configuration at WU-06 entry declares:

```text
project name: PTSIP
project version: 0.3.6
packages: ptsip*, vpms*
ptsip package data: specdata/*.json, specdata/*.yaml
console scripts: ptsip, ptsip-app
```

WU-06 must verify the actual built sdist/wheel rather than treating packaging configuration as proof of artifact contents.

The review should establish at least:

- `ptsip` implementation required by the released Tool is present;
- `vpms` compatibility package expected by Tool `0.3.6` is present;
- embedded `ptsip/specdata` contract data required at runtime is present;
- package metadata reports version `0.3.6`;
- console entry points are present and executable after wheel installation;
- repository planning, GitHub workflow, and unrelated verification-source trees are not silently turned into installed Product implementation;
- no non-Product implementation is included in a way that contradicts the declared package boundary.

A source-tree path not present in the wheel is not automatically a success; the classification and runtime requirement of the omitted or included responsibility must remain the authority.

## 9. Product Artifact evidence review

WU-06 owns the release-closure evidence that WU-05 deliberately deferred.

The stage must establish evidence sufficient to answer at least:

```text
Which built distribution is the Product Artifact?
Which exact repository SHA produced it?
What paths/components are included?
Is the content description complete enough for PTSIP-PKG-001 / PTSIP-ART-001?
Is the artifact evidence bound to the same repository snapshot being claimed?
Does any non-Product implementation appear inside the Product distribution?
```

The evidence mechanism must preserve the distinction between:

```text
project-owned architecture declaration
observed artifact contents
derivation / producer evidence
exact snapshot binding
conformance finding
```

The Tool must not infer a new lifecycle classification from package membership alone.

## 10. Conformance handling in WU-06

WU-06 must not equate `INCOMPLETE` with `NON_CONFORMANT`.

After actual Product Artifact evidence and binding are available, conformance is rerun/reviewed against the exact candidate as appropriate to determine which WU-05 release-closure gaps are resolved.

A remaining `INCOMPLETE` outcome is not automatically a WU-06 failure if the remaining blocking gaps are demonstrated to be outside the WU-06 release contract and are explicitly classified. Conversely, a definite mandatory architecture/package violation or a missing evidence condition required by the WU-06 completion gate cannot be waived merely to proceed to WU-07.

The completion record must preserve the exact remaining gaps, if any, rather than reducing them to a binary pass/fail label.

## 11. Self-hosted exact-SHA gate

Final WU-06 authority requires an exact-SHA self-hosted verification result using the existing verification workflow or a narrowly corrected descendant of it.

Expected environment contract:

```text
runs-on: [self-hosted, Windows, X64]
PowerShell-capable
Python 3.14 available through py -3.14
```

Runner identity is capability-based. A specific machine name is observation only and must not become workflow authority.

The final completion evidence must record at least:

```text
verification SHA
workflow run ID
job ID
runner capability / observed identity where useful
Python version
pytest result
build result
twine result
wheel reinstall result
Tool/Specification/profile result
artifact/conformance result relevant to WU-06
exact self-hosted/tooling-test commit status
```

## 12. Change discipline during WU-06

If the first candidate fails:

```text
failure
    -> classify ownership
    -> make the narrowest authorized correction
    -> obtain new branch HEAD / candidate SHA
    -> rerun affected local checks
    -> rerun final exact-SHA verification
```

Do not preserve stale verification authority after a source-changing correction.

WU-06 may modify an existing workflow, packaging configuration, implementation, test, or fixture only when the observed failure is classified as WU-06-owned. It must not opportunistically enter WU-07 or Tool `0.3.6.1` work.

## 13. Completion evidence

The WU-06 completion record must capture at least:

```text
exact entry baseline
final verification candidate SHA
Tool/package/Specification identity
complete pytest result
repository self-profile result
canonical/embedded contract result
sdist + wheel identities
artifact content review
Product Artifact evidence + exact binding result
release-relevant conform result and remaining gaps
build result
twine result
wheel reinstall / CLI smoke result
VPMS compatibility result
self-hosted workflow run/job
exact self-hosted/tooling-test status
all discovered failures and ownership classification
all WU-06 corrections, if any
```

Documentation commits made after successful exact-SHA verification record the result but do not replace the exact source verification authority.

## 14. Completion gate

WU-06 is complete only when:

- the exact entry baseline is recorded;
- final verification is performed on a documented descendant candidate SHA;
- complete repository pytest passes with zero failures;
- Tool/package/Specification identity is coherent;
- repository self-profile remains valid and coverage-complete;
- canonical/embedded contract checks required by the repository pass;
- sdist and wheel build successfully;
- actual distribution contents are reviewed against the Product boundary;
- Product Artifact evidence and exact snapshot binding required for WU-06 release closure are established or any remaining non-WU-06 gap is explicitly classified without hiding a mandatory violation;
- `twine check` passes;
- the built wheel is force-reinstalled and required public CLI/Specification/VPMS smoke checks pass;
- every failure/findings is classified before correction;
- no Tool `0.3.6.1` migration work is entered;
- no WU-07 final Specification freeze/release publication work is entered;
- no new Specification revision is created without separately established normative necessity;
- the exact final candidate SHA receives successful `self-hosted/tooling-test` status;
- completion evidence is recorded in this stage document, `planning/0.3.6.md`, and `STATUS.md`.

After completion:

```text
WU-06 COMPLETE / EXACT-SHA VERIFIED
    -> fresh branch HEAD
    -> create WU-07 detailed stage document with exact entry baseline
    -> WU-07 ACTIVE
```

## 15. Initial execution sequence

Once release-level/status activation is propagated, execute in this order:

```text
1. confirm WU-06 exact entry baseline and current branch descendant
2. establish Tool/package/Specification identity
3. validate repository self-profile and Effective Map coverage
4. run complete repository pytest
5. build sdist + wheel
6. inspect actual distribution contents and metadata
7. establish Product Artifact evidence and exact snapshot binding
8. rerun/review release-relevant conformance with that evidence
9. run twine check
10. force reinstall built wheel and run installed CLI/Specification/VPMS smoke
11. classify/fix only WU-06-owned failures
12. repeat affected checks on any new candidate SHA
13. run exact-SHA self-hosted tooling-test
14. verify exact commit status and collect run/job evidence
15. record WU-06 completion evidence
```

This document fixes the WU-06 exact entry baseline at `bdd77ef3f026a83bb0fd0099144aebea1de55365`. The stage is not represented as ACTIVE at the release/status level until `planning/0.3.6.md` and `STATUS.md` are explicitly propagated.