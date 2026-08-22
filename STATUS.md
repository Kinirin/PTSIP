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

## Active Tool 0.3.6 development

Development branch:

```text
tool-0.3.6-lifecycle-ownership
```

Development Tool identity is `0.3.6`; Tool `0.3.6` is **not published**.

Current bound Specification:

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
WU-05  repository dogfood / self-evaluation               NEXT / ENTRY REVIEW PENDING
WU-06  full regression / package verification             PLANNED
WU-07  final Specification freeze / release preparation   PLANNED
```

WU-05 is not ACTIVE yet. It requires a fresh branch-HEAD read, predecessor completion review, and an exact entry baseline.

## WU-04 completion

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

Primary records:

```text
planning/0.3.6/WU-04C-declaration-authority-effective-map.md
planning/0.3.6/WU-04D-resolved-profile-digest-provenance.md
planning/0.3.6/WU-04E-validation-effective-map.md
planning/0.3.6/WU-04F-conformance-effective-map.md
planning/0.3.6/WU-04G-clarification-adoption-effective-map.md
planning/0.3.6/WU-04G-G5-verification-closure.md
planning/0.3.6/WU-04H-vpms-effective-map-integration.md
planning/0.3.6/WU-04I-final-regression-wu04-completion.md
```

### WU-04G checkpoint

WU-04G's earlier exact-SHA full regression at:

```text
3ee6bb1d8ecff3bbd6b2e63f50e4c9cde3fcd667
```

reported:

```text
303 passed / 8 failed
```

That run was correctly recorded as a failed repository-wide workflow while WU-04G itself was stage-complete after its own G contracts passed and all eight failures were classified.

WU-04I subsequently reviewed and remediated all eight failures as stale/current-contract regression expectations without changing production architecture semantics. The remediation snapshot was:

```text
5f7bfdce120650db30d247a686d50a93f91bb3e4
```

### Final WU-04 exact-SHA verification

Final implementation verification authority:

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

The workflow explicitly checked out `eadb07f...` and verified `git rev-parse HEAD` against the dispatched SHA before execution.

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

The successful workflow recorded:

```text
context:     self-hosted/tooling-test
state:       success
description: Self-hosted Windows verification passed for the exact dispatched SHA
target:      https://github.com/Kinirin/PTSIP/actions/runs/32588090216
```

Therefore:

```text
WU-04I COMPLETE / EXACT-SHA VERIFIED
WU-04  COMPLETE
remaining repository regression failures: 0
```

The documentation commits that record this closure occur after `eadb07f...`; they are closure records and do not replace that exact implementation verification authority.

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

The final WU-04 exact-SHA workflow verified:

```text
valid: true
errors: []
warnings: []
source_mode: explicit
materialized: true
responsibility_map_coverage.unassigned_count: 0
```

The final Effective Responsibility Map digest at the verified SHA was:

```text
sha256:c009d4fb98daa8a3dc0fd2dfbdd974b467cf107c8c1e7814e460059f4c0c4a01
```

Repository responsibilities include:

- release automation: `DELIVERY`;
- repository verification/CI/maintenance: `DEVELOPMENT_TOOLING`;
- canonical lifecycle-independent contracts: `NEUTRAL_CONTRACT` where applicable;
- subordinate specification governance support: associated-artifact semantics where independent component ownership is unnecessary.

## VPMS boundary

PTSIP lifecycle classification and VPMS Verification Purpose remain independent.

Current VPMS purpose compatibility vocabulary remains:

```text
PRODUCT
TOOLCHAIN
```

VPMS `TOOLCHAIN` is VPMS vocabulary, not a Tool `0.3.6` PTSIP classification.

PTSIP core does not depend on VPMS. VPMS consumes only a narrow read-only projection of validated effective PTSIP metadata and does not own template/hybrid semantics, PTSIP classification authority, or Tool `0.3.5` migration.

WU-04H's focused VPMS isolation/integration evidence and WU-04I's final full regression both passed.

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

Blind `TOOLCHAIN -> DEVELOPMENT_TOOLING` migration remains prohibited. Legacy responsibility must be re-evaluated against actual lifecycle ownership and explicit project authority.

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

Do not hard-code a `DESKTOP-*` machine name into workflow policy.

Current verification/release path:

```text
tooling-test.yml
    workflow_dispatch with no custom inputs
    -> selected ref's exact github.sha
    -> self-hosted regression/build/smoke
    -> self-hosted/tooling-test status on exact SHA only after complete workflow success

release.yml
    workflow_dispatch from main with no custom inputs
    -> derive exact github.sha and package version
    -> require current origin/main == dispatched SHA
    -> require self-hosted/tooling-test success on exact SHA
    -> verify release contract
    -> create draft release for the same SHA

tooling-release.yml
    release published event
    -> self-hosted Windows distribution build/verification
    -> minimal GitHub-hosted GNU/Linux Trusted Publishing only
```

Manual `host_ready`, `source_sha`, `version`, and `release_candidate` inputs are not part of the current pipeline.

The narrow GNU/Linux PyPI Trusted Publishing job remains the approved GitHub-hosted compute exception.

## Next stage

Current release sequence is:

```text
WU-04 COMPLETE
    -> WU-05 repository dogfood / self-evaluation
    -> WU-06 full regression / package / distribution verification
    -> WU-07 final Specification freeze / release preparation
    -> Tool 0.3.6 release
```

WU-05 has **not** been entered by the WU-04 closure commits. Its entry requires a fresh branch-HEAD read and an explicitly recorded entry baseline.

## Tool lineage

- Tool `0.3.0`: published
- Tool `0.3.1`: published
- Tool `0.3.2`: source-only migration
- Tool `0.3.3`: permanently source-only
- Tool `0.3.4`: published historical Tool release
- Tool `0.3.5`: **published; first VPMS-capable Tool release**
- Tool `0.3.6`: **active development; WU-04 COMPLETE / EXACT-SHA VERIFIED; WU-05 NEXT / ENTRY REVIEW PENDING**

Current Tool `0.3.6` release planning authority is `planning/0.3.6.md`.
