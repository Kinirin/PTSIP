# PTSIP Repository Working Memory

This file is durable repository-operational context for maintainers and coding agents. It is **not** a normative Specification and does not replace `ptsip.yaml`, `spec/`, schemas, registry data, ADRs, or the active release contract.

## Published baseline and current candidate

Published baseline:

```text
Tool:                0.3.5
Tag:                 tool-v0.3.5
Release commit:      79bc4c2daf695e8462a02f2a7c4b1bb1a88846e1
Published Spec:      0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e
Historical classes: PRODUCT | TOOLCHAIN | NEUTRAL_CONTRACT
```

Tool `0.3.5` history is immutable release history. Do not rewrite it to match Tool `0.3.6` terminology.

Current Tool `0.3.6` release candidate:

```text
Tool/package:        0.3.6
Specification:       0.3.6-draft
SPEC_REVISION:       d6995ed232e845b88d8235b851e80ab54b7804ea
Development status:  COMPLETE / PRE-PUBLICATION
```

The latest PyPI release may remain Tool `0.3.5` until the Tool `0.3.6` publication boundary actually succeeds.

## Tool 0.3.6 work-unit closure

Current canonical work-unit state:

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

WU-07 selected **Strategy B — Release Contract Strengthening**.

Exact WU-07 entry baseline:

```text
8b2c0819e10b58902a780a094a0f52c603c39fba
```

Exact WU-07 final verification authority:

```text
source SHA:       452d0f8b0c78bdebb180ceb2b9994485f59eb43a
workflow run/job: 32640319047 / 97196299107
runner:           self-hosted Windows X64
Python:           3.14.6
pytest:           331 passed / 0 failed
Tool:             0.3.6
Specification:    0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea
profile:          valid=true / errors=[] / warnings=[]
coverage:         unassigned_count=0
build/twine:      PASS
Product Artifact: PASS
artifact binding: PASS
PTSIP-PKG-001:    0 definite violations
wheel smoke:      PASS
VPMS smoke:       PASS
status:           self-hosted/tooling-test = success
```

Documentation descendants after `452d0f8...` record closure and release-facing state. They do not replace the exact verification authority.

Primary completion records:

```text
planning/0.3.6.md
planning/0.3.6/WU-07-final-specification-freeze-release-preparation.md
STATUS.md
releasenote/0.3.6.md
```

## Exact-main release handoff

Tool `0.3.6` development closure is complete. The next boundary is release preparation from `main`:

```text
approved 0.3.6 state -> main
    -> read fresh exact main SHA
    -> dispatch tooling-test.yml for that exact main SHA
    -> require self-hosted/tooling-test success on the same SHA
    -> ensure main has not moved
    -> dispatch release.yml from the same current main SHA
    -> verify Tool / Specification / bound-content / release-document contract
    -> reconfirm current main
    -> create draft GitHub Release targeting the same SHA
    -> maintainer reviews/publishes the draft
    -> tooling-release.yml builds/verifies from the published tag
    -> PyPI Trusted Publishing
```

A development-branch success does not automatically make a different final `main` SHA release-ready. Exact-main verification remains mandatory.

## Current Specification binding

Tool `0.3.6` is bound to:

```text
Specification 0.3.6-draft
SPEC_REVISION d6995ed232e845b88d8235b851e80ab54b7804ea
```

WU-07 final audit found no genuine normative defect requiring revision movement. Release-contract, workflow, test, release-note, planning, status, README, or other documentation changes do not move `SPEC_REVISION` by themselves.

Release-bound assets include the five canonical Specification Markdown documents, canonical machine-readable schemas/registry, and matching embedded package copies. The release contract checks exact bound-revision object identity and canonical/embedded coherence separately.

## Canonical Tool 0.3.6 lifecycle model

Canonical classifications are exactly:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

`TOOLCHAIN` is Tool `0.3.5` legacy migration input only and is not a canonical Tool `0.3.6` alias. Blind `TOOLCHAIN -> DEVELOPMENT_TOOLING` migration is prohibited.

Classification is **primary lifecycle ownership**. It remains separate from roles, typed relationships, declaration/materialization provenance, evidence provenance, and VPMS Verification Purpose.

Canonical roles:

```text
IMPLEMENTATION
VERIFICATION
AUTOMATION
CONFIGURATION
DOCUMENTATION
GOVERNANCE
```

Canonical project-declared relationships:

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

Classification follows governing lifecycle obligation, not path, filename, language, framework, executable status, workflow provider, test status, majority vote, or confidence.

## Responsibility Map declaration and effective-map boundary

Canonical source modes:

```text
explicit
template
hybrid
```

Template selection is explicit and immutable-revision-bound. Repository evidence, language/framework detection, layout, manifests, and confidence do not select architecture authority.

Canonical flow:

```text
Source Project Profile
    -> validate declaration/binding
    -> deterministic materialization
    -> validated ResolvedProfile
    -> Canonical Effective Responsibility Map
        -> validation/conformance
        -> clarification/adoption
        -> narrow VPMS read-only projection
```

The Source Project Profile remains project-owned authority. The materializer cannot infer ownership, select templates, repair architecture, or silently rewrite the source declaration.

Declaration provenance values include:

```text
PROJECT_EXPLICIT
TEMPLATE
PROJECT_OVERRIDE
PROJECT_EXTENSION
PROJECT_REMOVAL
```

These are declaration origins, not evidence confidence or architecture authority replacements.

## Associated artifacts

Associated artifacts are project-owned non-component support surfaces subordinate to exactly one classified anchor component. They carry no classification or component roles of their own.

Do not use associated artifacts to hide executable responsibility, independent release/compatibility ownership, independently governed Delivery/Operations responsibility, or a truly independent neutral contract.

## Decision authority model

Keep these separate:

```text
Specification
    -> normative rules

Decision Authority
    -> which explicit coordinated architecture answer won

Project Profile / Responsibility Map
    -> durable project-owned architecture declaration

Observed evidence
    -> repository/artifact facts

Conformance Evaluation
    -> deterministic evaluation against applicable rules
```

A Decision Authority does not replace `ptsip.yaml` and does not prove conformance.

The Reference Tool GitHub profile uses:

```text
refs/heads/ptsip-policy
```

Required distributed properties include stable coordination identity, first-valid-resolution-wins, stale-writer-safe conditional mutation, authority freshness, deterministic reconciliation, fail-closed behavior, and global/local state separation.

Local SQLite state is Tool-owned operational state, not Git-shared global architecture authority.

## Evidence and migration authority

Keep these distinctions explicit:

```text
Evidence != Authority
Inference != Authority
Proposal != Authority
```

The project owner supplies architecture intent. Candidate discovery and analysis may propose, but they may not silently create or rewrite project-owned architecture.

Evidence provenance remains:

```text
DECLARED
OBSERVED
INFERRED
```

Tool `0.3.6.1` owns the later assisted migration continuation:

```text
facts
    -> candidate discovery
    -> evidence/provenance normalization
    -> Tool 0.3.5 legacy reader
    -> migration analysis
    -> target proposals
    -> owner preview/confirmation
    -> safe apply
```

Do not enter Tool `0.3.6.1` implementation until the Tool `0.3.6` release boundary and its own entry authority permit it.

## Product Artifact evidence

Artifact owner and producer are separate. Development Tooling or Delivery may produce a Product Artifact without becoming Product.

Release verification inspects actual built wheel contents and creates Product Artifact evidence bound to the exact repository snapshot. Packaging configuration is not treated as sufficient proof.

Definite Product package boundary violations remain governed by `PTSIP-PKG-001`. Evidence insufficiency may keep overall conformance `INCOMPLETE`; it must not be forced to `CONFORMANT` merely for release cosmetics.

## VPMS boundary

PTSIP answers who owns responsibility across lifecycles. VPMS answers why a Verification Case exists and what it protects.

PTSIP core must not depend on VPMS. VPMS consumes stable PTSIP metadata through a narrow read-only boundary and does not own Responsibility Map materialization or classification authority.

The current VPMS purpose vocabulary may retain `PRODUCT | TOOLCHAIN` for compatibility. VPMS `TOOLCHAIN` is not a canonical PTSIP Tool `0.3.6` classification.

VPMS PASS != PTSIP CONFORMANT. PTSIP CONFORMANT != functional verification PASS.

## Repository self-profile

Root `ptsip.yaml` self-adopts Responsibility Map v2 in explicit mode. The WU-07 exact verification reconfirmed:

```text
valid=true
errors=[]
warnings=[]
source_mode=explicit
materialized=true
responsibility_map_coverage.unassigned_count=0
```

Release/documentation paths are already covered by project-owned selectors. Documentation completion does not authorize changing `ptsip.yaml` merely to update prose.

## CI and release resource policy

Primary regression, release preparation, and publication-build verification use capability-bound self-hosted Windows X64 execution with PowerShell and Python 3.14 via `py -3.14`.

Do not hard-code a runner machine name. Preserve exact-SHA checkout and exact status binding. The narrow GNU/Linux PyPI Trusted Publishing job is the intended GitHub-hosted exception.

Do not create parallel workflows when the existing `tooling-test.yml`, `release.yml`, or `tooling-release.yml` can carry the required contract.

## Historical authority

Historical WU-04, WU-05, WU-06, Tool `0.3.5`, earlier Specification families, and ADR records remain evidence/history. They are not rewritten just because Tool `0.3.6` is now the current release candidate.

Current release-state authority is determined from fresh repository state plus the completed Tool `0.3.6` planning/status/release records and the exact verification evidence above.
