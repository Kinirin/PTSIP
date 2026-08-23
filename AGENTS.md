# AGENTS.md

These instructions apply to coding agents working anywhere in this repository.

## Required context before work

Read, in order:

1. `MEMORY.md`
2. `ptsip.yaml`
3. `src/ptsip/constants.py`
4. applicable Specification files under `spec/`
5. `planning/0.3.6.md`
6. `planning/0.3.6/WU-07-final-specification-freeze-release-preparation.md` when reviewing Tool `0.3.6` release closure or exact-main handoff

`MEMORY.md` and planning documents are operational context. Normative claims come from the applicable bound Specification and canonical machine-readable contracts.

## Repository-state discipline

- Re-read the remote target branch HEAD immediately before every GitHub write, merge, release preparation, or exact-SHA evidence claim.
- Preserve maintainer commits and never force-update `main`.
- Do not claim tests, builds, releases, tags, or publication succeeded without evidence for the exact relevant SHA.
- Documentation descendants after a successful verification run record results; they do not replace the exact source verification authority.
- Do not enter future Tool `0.3.6.1` implementation merely because its planning documents exist.
- Historical release notes, ADRs, and completed WU evidence are not rewritten to make current-version wording uniform.

## Tool 0.3.6 completion state

Tool `0.3.6` development work is complete. Current ordered state:

```text
WU-00  0.3.6-draft normative baseline                     COMPLETE
WU-01  lifecycle ontology/boundary rules                   COMPLETE
WU-02  roles + typed relationships + associated artifacts  COMPLETE
WU-03  canonical Responsibility Map v2 activation          COMPLETE
WU-04  template/materialization/effective-map pipeline     COMPLETE / EXACT-SHA VERIFIED
WU-05  repository dogfood / self-evaluation                COMPLETE / DOGFOOD REVIEWED
WU-06  full regression/package/distribution verification   COMPLETE / EXACT-SHA VERIFIED
WU-07  final Specification freeze/release preparation      COMPLETE / EXACT-SHA VERIFIED
```

WU-07 used **Strategy B — Release Contract Strengthening**.

Exact WU-07 verification authority:

```text
source SHA:       452d0f8b0c78bdebb180ceb2b9994485f59eb43a
workflow run/job: 32640319047 / 97196299107
Python:           3.14.6
pytest:           331 passed / 0 failed
Specification:    0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea
profile coverage: unassigned_count=0
Product Artifact: PASS / exact snapshot binding
PTSIP-PKG-001:    0 definite violations
wheel/VPMS smoke: PASS
commit status:    self-hosted/tooling-test = success
```

The exact WU-07 entry baseline remains:

```text
8b2c0819e10b58902a780a094a0f52c603c39fba
```

The completion/verification authority is the later exact candidate `452d0f8...`; closure/documentation commits after it do not replace that verification authority.

The next release boundary is exact-main handoff:

```text
approved Tool 0.3.6 state -> main
    -> fresh exact main SHA
    -> tooling-test.yml on that exact SHA
    -> require self-hosted/tooling-test success
    -> release.yml from the same current main SHA
    -> release contract PASS
    -> draft GitHub Release targeting the same SHA
    -> maintainer publication
    -> tooling-release.yml publication build/verification
    -> PyPI Trusted Publishing
```

Do not describe Tool `0.3.6` as published until that publication boundary actually succeeds.

## Current Specification binding

Tool `0.3.6` is bound to:

```text
Specification 0.3.6-draft
SPEC_REVISION d6995ed232e845b88d8235b851e80ab54b7804ea
```

A new immutable Specification revision is required only for a genuine normative change. Workflow, test, planning, status, release-note, or other documentation-only changes do not move `SPEC_REVISION` by themselves.

## Tool 0.3.6 lifecycle reasoning

PTSIP classification answers:

```text
Who primarily owns this project responsibility across its lifecycle?
```

Canonical Tool `0.3.6` classifications are exactly:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

`TOOLCHAIN` is historical Tool `0.3.5` migration input only. It must not be emitted or preserved as a canonical Tool `0.3.6` alias.

Classification is determined from governing lifecycle obligation, not file type, path, framework, language, executable status, workflow provider, compilation behavior, test status, majority of files/jobs/steps, runtime duration, invocation frequency, or confidence score.

Important boundaries:

- Product-specific verification may be `PRODUCT`.
- Reusable verification/test SDK/framework/harness infrastructure may be `DEVELOPMENT_TOOLING`.
- Local/intermediate development build support is normally `DEVELOPMENT_TOOLING`.
- Authoritative release-unit assembly/signing/packaging/publication/deployment-to-destination is normally `DELIVERY`.
- `DELIVERY` ends at delivery handoff; ongoing health/recovery/reconciliation/maintenance is `OPERATIONS`.
- `NEUTRAL_CONTRACT` requires non-executable, non-owning, lifecycle-independent contract responsibility.
- Material mixed-lifecycle responsibilities should split when independently governable; do not choose a majority lifecycle.
- Material unresolved ownership fails closed.

## Responsibility Map v2 axes

Keep these distinct:

```text
classification
    = primary lifecycle ownership

roles
    = coarse responsibility characteristics

relationships
    = project-owned typed directed semantics

source_mode / derived origin
    = declaration/materialization provenance

VPMS Verification Purpose
    = what verification protects/verifies
```

Canonical roles:

```text
IMPLEMENTATION
VERIFICATION
AUTOMATION
CONFIGURATION
DOCUMENTATION
GOVERNANCE
```

Canonical project-declared relationship types:

```text
IMPORTS
LINKS
LOADS
INVOKES
READS
GENERATES
BUILDS
PACKAGES
PUBLISHES
DEPLOYS
VERIFIES
MANAGES
DOCUMENTS
SPECIFIES
GOVERNS
```

Do not derive project-owned relationships automatically from observed evidence. Evidence can support a proposal; explicit project declaration remains authority.

## Responsibility Map declaration modes

Canonical source modes are:

```text
explicit
template
hybrid
```

Template selection is explicit and immutable-revision-bound. Never select a template from repository layout, language, framework, manifest presence, or confidence.

Materialization is deterministic and non-authoritative. It must not infer lifecycle ownership, repair architecture, create project intent, or silently rewrite the Source Project Profile.

All source modes resolve to a validated Canonical Effective Responsibility Map for downstream Tool behavior. Preserve source declaration and materialized provenance separately.

## Adoption and migration discipline

Canonical Tool `0.3.6` Project Profiles use `classification` as primary lifecycle ownership authority. A second canonical `lifecycle_owner` field must not compete with it.

When explicit adoption/resolution records architecture facts, preserve applicable facts such as:

```text
classification
roles
purpose
shipped
runtime_required
executable
associated artifacts
typed relationships
explicit release/compatibility metadata
```

Legacy Tool `0.3.5` `TOOLCHAIN`, `lifecycle_owner`, old boundary roots, consumers, analysis inputs, or untyped policy edges are migration evidence only. Do not blindly convert them into Tool `0.3.6` authority.

Evidence-backed Tool `0.3.5 -> 0.3.6` assisted migration is owned by Tool `0.3.6.1`. Tool `0.3.6` release closure does not authorize implementing that continuation.

## Decision Authority / Project Profile / evidence

Keep these responsibilities separate:

```text
Specification
    -> normative rules

Decision Authority
    -> which explicit coordinated architecture answer won

Project Profile / Responsibility Map
    -> durable project-owned architecture declaration

Observed evidence
    -> what repository/artifacts actually do

Conformance Evaluation
    -> whether declaration + evidence satisfy applicable rules
```

A Decision Authority does not replace `ptsip.yaml` and does not prove conformance.

For GitHub coordination, the Reference Tool uses:

```text
refs/heads/ptsip-policy
```

Preserve stable decision identity, first-valid-resolution-wins, stale-writer-safe conditional mutation, read-side authority freshness, deterministic reconciliation, fail-closed behavior, and global/local state separation.

A complete local declaration is not sufficient reason to skip relevant distributed authority reads. Semantic equivalence is architecture meaning, not YAML formatting.

## Read-only default and mutation safety

Inspection and Pilot behavior are read-only by default. Tool-owned caches, reports, and local decision databases stay outside the Consumer Repository unless explicitly directed otherwise.

Prepared profile writes must reject stale repository/profile state. Do not combine evidence from different revisions into one stable claim.

## Product Artifact boundary

Artifact owner and producer are different concepts. Development Tooling or Delivery may build a Product Artifact, but Product distribution contents still have to satisfy the Product lifecycle boundary.

Release verification must inspect actual built artifacts, not packaging configuration as proof. Preserve snapshot-bound Product Artifact evidence and fail closed on definite `PTSIP-PKG-001` violations.

## VPMS boundary

PTSIP asks who owns a responsibility across its lifecycle. VPMS asks why a Verification Case exists and what it protects.

PTSIP classification and VPMS Verification Purpose remain independent. PTSIP core must not depend on VPMS. VPMS consumes only a narrow read-only projection of validated effective PTSIP metadata.

Current VPMS compatibility vocabulary may retain `PRODUCT | TOOLCHAIN`; VPMS `TOOLCHAIN` is not a Tool `0.3.6` PTSIP classification.

VPMS PASS != PTSIP CONFORMANT, and PTSIP CONFORMANT != functional verification PASS.

## Conformance behavior

Completed Consumer Repository outcomes are only:

```text
CONFORMANT
NON_CONFORMANT
INCOMPLETE
```

Do not equate zero findings with conformance. Blocking evidence gaps remain `INCOMPLETE` unless a definite mandatory violation already establishes `NON_CONFORMANT`.

Do not let project-local policy weaken universal PTSIP requirements.

## Release and CI resource policy

Primary regression/release verification uses capability-bound self-hosted Windows X64 execution with PowerShell and Python 3.14 available through `py -3.14`.

Do not hard-code a machine name. Preserve the existing exact-SHA checkout/status model. The narrow GNU/Linux PyPI Trusted Publishing job may remain the GitHub-hosted exception.

Do not create a parallel workflow where an existing release/test workflow can be narrowly maintained.

## Instruction priority

Use this order:

1. bound canonical Specification and normative companion assets from the same immutable revision;
2. relevant Decision Authority winner when distributed coordination applies;
3. repository Project Profile / Responsibility Map;
4. observed repository/dependency/artifact evidence;
5. imported external evidence with provenance;
6. project ADR/history;
7. this repository-operational contract;
8. informal examples.

This priority does not make Decision Authority a conformance oracle. Authority governs which explicit answer won; observed evidence still governs what the repository actually does.
