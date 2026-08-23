# PTSIP Status

## Current published release

- Canonical repository: `Kinirin/PTSIP`
- Maturity: Experimental
- Published Tool/package version: **`0.3.5`**
- Published tag: **`tool-v0.3.5`**
- Release commit: `79bc4c2daf695e8462a02f2a7c4b1bb1a88846e1`
- GitHub Release: **PUBLISHED** on 2026-08-17
- PyPI publication: **COMPLETE** through Trusted Publishing
- Bound published Specification: `0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e`
- Historical Tool `0.3.5` classifications: `PRODUCT | TOOLCHAIN | NEUTRAL_CONTRACT`

Tool `0.3.5` remains the published compatibility baseline. Its historical PTSIP semantics are not rewritten by Tool `0.3.6` development.

## Tool 0.3.6 development closure

Development branch:

```text
tool-0.3.6-lifecycle-ownership
```

Development Tool identity is `0.3.6`; Tool `0.3.6` is **not published**.

Tool `0.3.6` development work units are complete. The next boundary is a separate exact-main release handoff.

Final bound Specification:

```text
0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea
```

Canonical Tool `0.3.6` classifications are exactly:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

`TOOLCHAIN` is not a canonical Tool `0.3.6` PTSIP classification alias.

## Current work-unit state

```text
WU-00  0.3.6-draft normative baseline                    COMPLETE
WU-01  lifecycle ontology/boundary rules                  COMPLETE
WU-02  roles + typed relationships + associated artifacts COMPLETE
WU-03  canonical Responsibility Map v2 activation         COMPLETE
WU-04  template/materialization/effective-map pipeline    COMPLETE / EXACT-SHA VERIFIED
WU-05  repository dogfood / self-evaluation               COMPLETE / DOGFOOD REVIEWED
WU-06  full regression / package/distribution verification COMPLETE / EXACT-SHA VERIFIED
WU-07  final Specification freeze / release preparation   COMPLETE / EXACT-SHA VERIFIED
```

WU-05 exact entry baseline:

```text
c87e597409592f0794a5d404d621b6f73dd00cc6
```

WU-05 exact dogfood verification snapshot:

```text
5010fa81ebd104eb329f29217d4c5fecec51cacb
```

WU-05 execution/closure authority:

```text
planning/0.3.6/WU-05-repository-dogfood-self-evaluation.md
```

WU-06 exact entry baseline:

```text
bdd77ef3f026a83bb0fd0099144aebea1de55365
```

WU-06 detailed stage document was initially created at:

```text
59224b9df25db34f4af58e48eaeac0b1db8bf187
```

WU-06 exact verification authority:

```text
a94c50130a694dba937708403520417719aec1e1
```

WU-06 verification run/job:

```text
32598727026 / 97093629017
```

WU-06 execution/closure authority:

```text
planning/0.3.6/WU-06-full-regression-package-distribution-verification.md
```

WU-07 exact entry baseline:

```text
8b2c0819e10b58902a780a094a0f52c603c39fba
```

WU-07 detailed stage document was initially created at:

```text
0e438e7544ebbfee1f1692086d31d9d6f539d297
```

WU-07 selected strategy:

```text
B — Release Contract Strengthening
```

WU-07 exact verification authority:

```text
452d0f8b0c78bdebb180ceb2b9994485f59eb43a
```

WU-07 verification run/job:

```text
32640319047 / 97196299107
```

WU-07 execution/closure authority:

```text
planning/0.3.6/WU-07-final-specification-freeze-release-preparation.md
```

WU-06 entry/activation/closure records do not replace WU-06 exact verification authority `a94c501...`. WU-07 entry, activation, implementation, and closure-documentation descendants do not replace exact WU-07 entry baseline `8b2c0819...` or exact verification authority `452d0f8...`.

## WU-04 completion authority

WU-04 staged status:

```text
WU-04A  template catalog identity                         COMPLETE
WU-04B  deterministic materializer core                  COMPLETE
WU-04C  declaration authority/source_mode boundary       COMPLETE
WU-04D  ResolvedProfile + digest + provenance            COMPLETE
WU-04E  validation consumes effective map                COMPLETE / EXACT-SHA VERIFIED
WU-04F  conformance consumes effective map               COMPLETE / EXACT-SHA VERIFIED
WU-04G  clarification/adoption integration               COMPLETE / EXACT-SHA VERIFIED
WU-04H  VPMS narrow read-only integration                COMPLETE / VERIFIED
WU-04I  final regression + WU-04 completion              COMPLETE / EXACT-SHA VERIFIED
```

Final WU-04 implementation verification authority:

```text
source SHA:   eadb07f78f9690ca0180bfbba194c5c9602e1838
workflow:     tooling-test
run:          32588090216
job:          97067421324
runner:       self-hosted Windows X64
Python:       3.14.6
pytest:       327 passed / 0 failed
wall time:    259.67s (0:04:19)
job result:   success
commit status: self-hosted/tooling-test = success
```

Post-regression verification also passed:

```text
PTSIP Tool identity                 0.3.6
Specification family               0.3.6-draft
SPEC_REVISION                       d6995ed232e845b88d8235b851e80ab54b7804ea
repository profile valid            true
repository profile warnings         []
responsibility-map unassigned_count 0
clarification status                NO_CLARIFICATION_REQUIRED
clarification snapshot              STABLE
local gate status                   NO_DECISION_REQUIRED
sdist build                          PASS
wheel build                          PASS
twine check                          PASS
built-wheel reinstall smoke          PASS
VPMS compatibility smoke             PASS
```

Therefore:

```text
WU-04I COMPLETE / EXACT-SHA VERIFIED
WU-04  COMPLETE
remaining repository regression failures: 0
```

The WU-04 closure documentation commits occur after `eadb07f...`; they record the verified result and do not replace that exact implementation verification authority.

## WU-05 repository dogfood completion

WU-05 treated the PTSIP repository as a real Tool `0.3.6` consumer and exercised:

```text
Tool / Specification identity
    -> root ptsip.yaml validation
    -> Effective Responsibility Map / coverage review
    -> ptsip conform . --json
    -> ptsip clarify . --json
    -> ptsip gate . --coordination local --json
    -> read-only stability / non-mutation review
    -> finding ownership classification
```

Repository declaration remained architecture authority throughout. Discovery, paths, workflow names, and confidence remained evidence only and did not override project-owned lifecycle ownership.

Exact reviewed dogfood snapshot:

```text
5010fa81ebd104eb329f29217d4c5fecec51cacb
```

Dogfood evidence summary:

```text
validate                                PASS
valid                                   true
errors                                  []
warnings                                []
source_mode                             explicit
materialized                            true
component selector conflicts            0
associated-artifact conflicts           0
scan_errors                             []
responsibility_map_coverage.unassigned  0
Effective Map digest                    sha256:c009d4fb98daa8a3dc0fd2dfbdd974b467cf107c8c1e7814e460059f4c0c4a01

conform                                 ENFORCED / INCOMPLETE / REVIEWED
definite architecture violations        0
blocking evidence gaps                  20
conformance audit                       PASS
conformance snapshot                    STABLE

clarify                                 NO_CLARIFICATION_REQUIRED
clarification_count                     0
llm_calls                               0
speculative_classification              false
clarification snapshot                  STABLE

local gate                              NO_DECISION_REQUIRED
backend                                 LOCAL
decisions                               []

post-gate git diff -- ptsip.yaml         clean / no output
post-gate git status --short             clean / no output
```

Final finding ownership classification:

```text
A. repository declaration defect                  NONE
B. Tool 0.3.6 implementation defect               NONE CONFIRMED
C. test/fixture/documentation drift               NONE IDENTIFIED
D. Tool 0.3.6.1 migration/discovery concern       NONE ENTERED
E. WU-06/WU-07 release-closure concern            Product Artifact evidence / exact artifact binding
F. genuine normative Specification defect         NONE
```

Two observations initially considered possible Tool implementation defects were reviewed and retained as fail-closed evidence limitations instead:

```text
OBS-01  PyYAML -> yaml and PyJWT -> jwt distribution/import-name mismatches remain unresolved by native dependency evidence.
        The Tool does not turn that uncertainty into a false architecture violation; strict conformance remains INCOMPLETE.

OBS-02  independent-build evaluation requires component-declared, component-owned manifest evidence and deliberately blocks shared cross-lifecycle manifest evidence.
        `manifests` remains optional in the canonical profile schema; insufficient evidence therefore keeps strict conformance INCOMPLETE rather than making the profile invalid or the repository NON_CONFORMANT.
```

No WU-05 source/profile correction was required. No Tool `0.3.6.1` migration work was entered. The bound Specification remains:

```text
0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea
```

Therefore:

```text
WU-05 COMPLETE / DOGFOOD REVIEWED
exact repository-consumer verification authority:
5010fa81ebd104eb329f29217d4c5fecec51cacb
```

Closure-documentation commits after `5010fa81...` record the reviewed result and do not replace the exact dogfood verification authority.

## WU-06 completion authority

WU-06 started from exact entry baseline:

```text
bdd77ef3f026a83bb0fd0099144aebea1de55365
```

and completed release-candidate verification on exact source SHA:

```text
a94c50130a694dba937708403520417719aec1e1
```

Exact verification evidence:

```text
workflow                              tooling-test
run/job                               32598727026 / 97093629017
runner                                self-hosted Windows X64
Python                                3.14.6
complete repository pytest            327 passed / 0 failed / 257.31s
Tool                                  0.3.6
Specification                         0.3.6-draft
SPEC_REVISION                         d6995ed232e845b88d8235b851e80ab54b7804ea
repository profile                    valid=true / errors=[] / warnings=[]
source_mode                           explicit
materialized                          true
responsibility_map_coverage           unassigned_count=0
clarify                               NO_CLARIFICATION_REQUIRED / STABLE
local gate                            NO_DECISION_REQUIRED / LOCAL
sdist                                 PASS
wheel                                 PASS
twine check                           PASS
Product Artifact evidence             PASS
artifact exact-snapshot binding       PASS
artifact PTSIP-PKG-001 violations     0
artifact-aware conform                INCOMPLETE / reviewed
built-wheel force reinstall           PASS
installed Tool/Specification smoke    PASS
VPMS compatibility smoke              PASS
self-hosted/tooling-test               success
```

The Product Artifact wheel verified in the exact run had SHA-256:

```text
b5ed0a7fca13181af9a0e1bc92dc9cceb382fe96a762fbf0504216e923880726
```

WU-05's Product Artifact / exact artifact-binding handoff is therefore closed. The remaining artifact-aware `INCOMPLETE` outcome does not represent a Product Artifact failure: artifact-evidence blocking gaps were cleared and no definite `PTSIP-PKG-001` violation remained. Previously reviewed Python distribution/import-name and independent-build manifest evidence limitations remain fail-closed evidence sufficiency observations.

WU-06 found and corrected a release-verification workflow defect while establishing Product Artifact evidence. The failed precursor run did not replace authority; after the narrow correction, exact candidate `a94c501...` passed the full workflow and received `self-hosted/tooling-test = success`.

Therefore:

```text
WU-06 COMPLETE / EXACT-SHA VERIFIED
exact verification authority:
a94c50130a694dba937708403520417719aec1e1
```

Closure documentation after `a94c501...` records this result and does not replace the exact source verification authority.

## WU-07 completion authority

WU-07 started from exact entry baseline:

```text
8b2c0819e10b58902a780a094a0f52c603c39fba
```

with detailed stage-document commit:

```text
0e438e7544ebbfee1f1692086d31d9d6f539d297
```

Selected strategy:

```text
B — Release Contract Strengthening
```

Final Specification audit result:

```text
genuine normative Specification defect     NONE
bound Specification family                  0.3.6-draft
SPEC_REVISION                               d6995ed232e845b88d8235b851e80ab54b7804ea
revision movement                            NONE
release-bound normative assets              15
```

WU-07 strengthened the release contract while preserving the existing release architecture. It added exact Git-blob equality between release-bound normative assets at `HEAD` and `SPEC_REVISION`, canonical/embedded machine-readable equality, reviewed `releasenote/0.3.6.md`, release-note index consistency, bounded `MEMORY.md` / `AGENTS.md` current-state cleanup, publication-wheel Product Artifact evidence, exact tagged-snapshot binding, and regression tests for these requirements.

`release.yml` retained its existing exact-main contract: main-only dispatch, exact `${{ github.sha }}` checkout, current `origin/main` freshness checks before and after verification, exact `self-hosted/tooling-test` success requirement, release-contract verification, and same-SHA draft release target. No WU-07 defect required a semantic change to this workflow.

`tooling-release.yml` remains release-published-triggered and self-hosted for build/verification. Its publication build now verifies the actual newly built wheel's Product Artifact contents/evidence and exact tagged source binding before uploading the distributions used by the narrow GNU/Linux PyPI Trusted Publishing boundary.

Final exact verification was completed on:

```text
452d0f8b0c78bdebb180ceb2b9994485f59eb43a
```

Exact verification evidence:

```text
workflow                              tooling-test
run/job                               32640319047 / 97196299107
runner                                self-hosted Windows X64
Python                                3.14.6
complete repository pytest            331 passed / 0 failed / 267.80s
Tool                                  0.3.6
Specification                         0.3.6-draft
SPEC_REVISION                         d6995ed232e845b88d8235b851e80ab54b7804ea
repository profile                    valid=true / errors=[] / warnings=[]
source_mode                           explicit
materialized                          true
Effective Map digest                  sha256:c009d4fb98daa8a3dc0fd2dfbdd974b467cf107c8c1e7814e460059f4c0c4a01
responsibility_map_coverage           unassigned_count=0
clarify                               NO_CLARIFICATION_REQUIRED / STABLE
local gate                            NO_DECISION_REQUIRED / LOCAL
sdist                                 PASS
wheel                                 PASS
twine check                           PASS
Product Artifact evidence             PASS
artifact exact-snapshot binding       PASS
artifact PTSIP-PKG-001 violations     0
artifact-aware conform                INCOMPLETE / reviewed
built-wheel force reinstall           PASS
installed Tool/Specification smoke    PASS
VPMS compatibility smoke              PASS
self-hosted/tooling-test               success
```

The Product Artifact wheel verified in the WU-07 exact run had SHA-256:

```text
b4c04d9aa78276d7d5d470f02d6ae52aef54c92b2ec7c4fa1fab6e85b0ab4644
```

The artifact-aware `INCOMPLETE` result does not represent a release failure. The successful workflow required Product Artifact boundary and snapshot-binding evaluators to run, valid exact binding, zero artifact-evidence blocking gaps, and zero definite `PTSIP-PKG-001` violations.

Final WU-07 finding ownership:

```text
A. normative Specification defect                         NONE
B. Specification-binding/content-freeze guard defect      CORRECTED / VERIFIED
C. Tool release-note defect                               CORRECTED / VERIFIED
D. operational current-state documentation drift          CORRECTED / VERIFIED
E. release workflow exact-SHA contract defect             NONE
F. publication artifact verification defect               CORRECTED / CONTRACT VERIFIED
G. release-readiness test/fixture defect                  CORRECTED / VERIFIED
H. package/distribution defect discovered at final gate   NONE
I. Tool 0.3.6 implementation regression                   NONE
J. Strategy-C release-infrastructure improvement          DEFERRED / NOT REQUIRED
K. Tool 0.3.6.1 migration concern                         DEFERRED / OUT OF SCOPE
```

Therefore:

```text
WU-07 COMPLETE / EXACT-SHA VERIFIED
exact verification authority:
452d0f8b0c78bdebb180ceb2b9994485f59eb43a
```

Closure documentation after `452d0f8...` records this result and does not replace the exact source verification authority.

WU-07 did not enter Tool `0.3.6.1` migration work and did not implement Strategy C Build Once / release-attestation architecture.

## Canonical Responsibility Map / Effective Map boundary

Tool `0.3.6` supports project-owned source declarations in:

```text
explicit
template
hybrid
```

Template selection is explicit and immutable-revision-bound. Materialization is deterministic and non-authoritative.

Canonical downstream flow is:

```text
source declaration
    -> validate source declaration/binding
    -> deterministic materialization
    -> validated ResolvedProfile
    -> Canonical Effective Responsibility Map
        -> validation/conformance
        -> clarification/adoption
        -> narrow VPMS read-only projection
```

Repository evidence, path/layout convention, workflow naming, template materialization, and confidence do not become architecture authority.

Classification, component role, typed relationship, declaration/materialization provenance, and VPMS Verification Purpose remain separate axes.

## Repository self-profile

Root `ptsip.yaml` self-adopts Responsibility Map v2.

WU-07 final exact verification reconfirmed the self-profile as:

```text
valid: true
errors: []
warnings: []
source_mode: explicit
materialized: true
responsibility_map_coverage.unassigned_count: 0
```

The Effective Responsibility Map digest remains:

```text
sha256:c009d4fb98daa8a3dc0fd2dfbdd974b467cf107c8c1e7814e460059f4c0c4a01
```

WU-07 changes remained inside existing `repository-release-automation`, `repository-verification`, `repository-maintenance`, and `product-documentation` selectors; no project-owned Responsibility Map change was required.

## VPMS boundary

PTSIP lifecycle classification and VPMS Verification Purpose remain independent.

Current VPMS purpose compatibility vocabulary remains:

```text
PRODUCT
TOOLCHAIN
```

VPMS `TOOLCHAIN` is VPMS vocabulary, not a Tool `0.3.6` PTSIP classification.

PTSIP core does not depend on VPMS. VPMS consumes only a narrow read-only projection of validated effective PTSIP metadata and does not own template/hybrid semantics, PTSIP classification authority, or Tool `0.3.5` migration.

## Tool 0.3.5 migration continuation

Evidence-backed Tool `0.3.5 -> 0.3.6` architecture migration is not a Tool `0.3.6` release prerequisite.

The continuation is owned by Tool `0.3.6.1`:

```text
planning/0.3.6.1.md
planning/0.3.6.1/WU-01-candidate-discovery-evidence-expansion.md
planning/0.3.6.1/WU-02-evidence-provenance-normalization.md
planning/0.3.6.1/WU-03-tool-035-legacy-reader.md
planning/0.3.6.1/WU-04-lifecycle-migration-analyzer.md
planning/0.3.6.1/WU-05-target-proposals.md
```

Blind `TOOLCHAIN -> DEVELOPMENT_TOOLING` migration remains prohibited.

## GitHub Actions resource policy

Self-hosted execution is capability-bound, not machine-name-bound.

Eligible Tool build/test runners must satisfy:

```text
self-hosted
Windows
X64
PowerShell
Python 3.14 available through `py -3.14`
```

Current verification/release path is:

```text
tooling-test.yml
    workflow_dispatch with no custom inputs
    -> selected ref's exact github.sha
    -> self-hosted regression/build/smoke
    -> Product Artifact evidence + exact snapshot binding
    -> self-hosted/tooling-test status on exact SHA only after complete workflow success

release.yml
    workflow_dispatch from main with no custom inputs
    -> derive exact github.sha and package version
    -> require current origin/main == dispatched SHA
    -> require self-hosted/tooling-test success on exact SHA
    -> verify exact Tool / Specification / bound-content / release-document contract
    -> reconfirm current origin/main
    -> create draft release for the same SHA

tooling-release.yml
    release published event
    -> self-hosted Windows distribution build
    -> verify actual publication Product Artifact evidence + exact tagged-snapshot binding
    -> verify distribution metadata
    -> upload only verified distributions
    -> minimal GitHub-hosted GNU/Linux Trusted Publishing only
```

## Next stage sequence

```text
WU-04 COMPLETE
    -> WU-05 COMPLETE / DOGFOOD REVIEWED
    -> WU-06 COMPLETE / EXACT-SHA VERIFIED
    -> WU-07 COMPLETE / EXACT-SHA VERIFIED
    -> exact-main release handoff
    -> Tool 0.3.6 release
```

The development-branch verification authority `452d0f8...` is not transferable to a different resulting `main` SHA. The main handoff must independently verify the exact merged `main` commit before release preparation.

## Tool lineage

- Tool `0.3.0`: published
- Tool `0.3.1`: published
- Tool `0.3.2`: source-only migration
- Tool `0.3.3`: permanently source-only
- Tool `0.3.4`: published historical Tool release
- Tool `0.3.5`: **published; first VPMS-capable Tool release**
- Tool `0.3.6`: **development complete; WU-07 COMPLETE / EXACT-SHA VERIFIED at `452d0f8...`; exact-main release handoff pending; not yet published**

Current Tool `0.3.6` release planning authority is `planning/0.3.6.md`.