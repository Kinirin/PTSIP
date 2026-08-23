# PTSIP Repository Working Memory

This file is durable repository-operational context for maintainers and coding agents. It is **not** a normative Specification and does not replace `ptsip.yaml`, `spec/`, schemas, registry data, ADRs, or the active plan.

## Published baseline

- Canonical repository: `Kinirin/PTSIP`
- Published Tool: `0.3.5`
- Tag: `tool-v0.3.5`
- Release commit: `79bc4c2daf695e8462a02f2a7c4b1bb1a88846e1`
- Published bound Specification: `0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e`
- Historical Tool `0.3.5` classifications: `PRODUCT | TOOLCHAIN | NEUTRAL_CONTRACT`

Do not rewrite Tool `0.3.5` history. Tool `0.3.6` compatibility means understand/migrate obsolete state, not preserve obsolete ontology as canonical Tool `0.3.6` authority.

## Active Tool 0.3.6 release closure

Development branch:

```text
tool-0.3.6-lifecycle-ownership
```

Master plan:

```text
planning/0.3.6.md
```

Active stage:

```text
planning/0.3.6/WU-07-final-specification-freeze-release-preparation.md
strategy B — Release Contract Strengthening
exact entry baseline 8b2c0819e10b58902a780a094a0f52c603c39fba
```

Current ordered state:

```text
WU-00  0.3.6-draft normative baseline                     COMPLETE
WU-01  lifecycle ontology/boundary rules                   COMPLETE
WU-02  roles + typed relationships + associated artifacts  COMPLETE
WU-03  canonical Responsibility Map v2 activation          COMPLETE
WU-04  template/materialization/effective-map pipeline     COMPLETE / EXACT-SHA VERIFIED
WU-05  repository dogfood / self-evaluation                COMPLETE / DOGFOOD REVIEWED
WU-06  full regression/package/distribution verification   COMPLETE / EXACT-SHA VERIFIED
WU-07  final Specification freeze/release preparation      ACTIVE
```

Published PyPI Tool remains `0.3.5`; the development branch Tool identity is `0.3.6` until the exact-main release pipeline publishes Tool `0.3.6`.

Historical WU-04G/H/I documents remain implementation/evidence history. They do not override the current WU-07 plan, current Specification binding, or later exact-SHA verification evidence.

## Current Specification binding

Current Tool/root-profile release-candidate binding:

```text
Specification 0.3.6-draft
SPEC_REVISION d6995ed232e845b88d8235b851e80ab54b7804ea
```

Important historical checkpoints include:

```text
WU-00 baseline:              654e41d49600fc091f9a6cb6b1c60bbc7da4e301
WU-03 canonical activation:  12e2ccd15634ecb3d0a4195b0f61ac3f620e7540
WU-04C authority snapshot:   82abd09360df09a95fbbfb516855fa9ffb49f050
current bound snapshot:      d6995ed232e845b88d8235b851e80ab54b7804ea
```

The final WU-07 audit found no pre-established genuine normative defect and therefore retains `d6995ed...` unless later evidence proves an actual normative correction is necessary. Release workflow/test/documentation changes MUST NOT move `SPEC_REVISION` by themselves.

WU-07 release-contract strengthening binds the release-source normative asset set to the exact immutable revision. Current release-bound assets include the five canonical Specification Markdown documents, canonical machine-readable schemas/registry, and their embedded package copies. Canonical/embedded equality and bound-revision equality are separate required properties.

Decision records remain:

- `ADR-0007` — primary lifecycle boundary determination;
- `ADR-0008` — roles, typed relationships, associated artifacts;
- `ADR-0009` — declaration authority, source mode, and non-authoritative materialization.

## Canonical Tool 0.3.6 model

Canonical classifications exactly:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

`TOOLCHAIN` is Tool `0.3.5` migration input only, not a Tool `0.3.6` alias.
Classification is primary lifecycle ownership and remains distinct from roles, typed relationships, source mode/materialization provenance, and VPMS Verification Purpose.

Canonical roles:

```text
IMPLEMENTATION
VERIFICATION
AUTOMATION
CONFIGURATION
DOCUMENTATION
GOVERNANCE
```

Canonical declared relationship types:

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

Associated artifacts are project-owned non-component support surfaces subordinate to exactly one classified anchor component. They have no classification or component roles and cannot hide independently governable lifecycle responsibility.

## Responsibility Map declaration and effective-map boundary

Responsibility Map source modes:

```text
explicit
template
hybrid
```

Authority boundary:

```text
classification
    = lifecycle responsibility

source_mode / derived entity origin
    = declaration authority provenance

materializer
    = deterministic non-authoritative resolution
```

Derived origin vocabulary:

```text
PROJECT_EXPLICIT
TEMPLATE
PROJECT_OVERRIDE
PROJECT_EXTENSION
PROJECT_REMOVAL
```

The materializer MUST NOT infer architecture, auto-select templates, change classifications, fabricate responsibilities, silently repair dangling relationships, cascade removals, resolve conflicts by heuristics, or mutate source declarations merely to produce a valid effective map.

Canonical downstream flow is:

```text
source declaration
    -> validate source/binding
    -> deterministic materialization
    -> validated ResolvedProfile
    -> Canonical Effective Responsibility Map
        -> validation / conformance
        -> clarification / adoption
        -> narrow VPMS read-only projection
```

An accepted project-owned clarification/adoption decision may authorize only the minimum project delta needed to represent that accepted decision. Where required, the accepted decision may authorize `template -> hybrid` while retaining the exact selected template ID/revision. Repository discovery and deterministic materialization remain non-authoritative.

## Clarification / decision boundary

New canonical answers use `ptsip-clarification-answer/v2` with:

```text
classification
purpose
shipped
runtime_required
executable
```

`lifecycle_owner` is removed from new Tool `0.3.6` canonical decisions and Project Profile serialization. Historical v1 data may be handled only through explicit compatibility logic and cannot restore `TOOLCHAIN` authority.

Exact repository-relative selected profile path identity persists through gate creation, decision storage, resolution context, subject revision, remote read, stale check, projection validation, and CAS write. Never silently substitute root `ptsip.yaml` for an explicitly selected non-root profile.

Dedicated distributed decision authority remains distinct from project architecture declaration and observed evidence. Evidence is not authority; materialization is not authority; accepted project-owned decisions authorize only their exact permitted write delta.

## WU-04 / WU-05 / WU-06 completion evidence

WU-04 architecture-foundation completion is backed by exact-SHA verification authority:

```text
source SHA                    eadb07f78f9690ca0180bfbba194c5c9602e1838
workflow run/job              32588090216 / 97067421324
Python                        3.14.6
pytest                        327 passed / 0 failed
self-hosted/tooling-test      success
```

WU-05 repository dogfood exact review authority:

```text
5010fa81ebd104eb329f29217d4c5fecec51cacb
```

Reviewed WU-05 result included valid self-profile, `unassigned_count=0`, stable clarification/local gate, no confirmed Tool `0.3.6` implementation defect, and no normative Specification defect. Strict conformance remained fail-closed `INCOMPLETE`; it was not forced to a false `CONFORMANT` result.

WU-06 exact verification authority:

```text
entry baseline                  bdd77ef3f026a83bb0fd0099144aebea1de55365
verification SHA                a94c50130a694dba937708403520417719aec1e1
workflow run/job                32598727026 / 97093629017
runner                          self-hosted Windows X64
Python                          3.14.6
pytest                          327 passed / 0 failed / 257.31s
Tool                            0.3.6
Specification                   0.3.6-draft
SPEC_REVISION                   d6995ed232e845b88d8235b851e80ab54b7804ea
self-profile                    valid / errors=[] / warnings=[] / unassigned_count=0
sdist + wheel                   PASS
twine                           PASS
Product Artifact evidence       PASS
artifact snapshot binding       PASS
PTSIP-PKG-001 definite failures 0
wheel reinstall / CLI smoke     PASS
VPMS compatibility smoke        PASS
self-hosted/tooling-test        success
```

WU-06's Product Artifact evidence closes the release-verification handoff from WU-05. Remaining artifact-aware `INCOMPLETE` is not a Product Artifact failure and does not justify weakening fail-closed behavior.

## WU-07 Strategy B release contract

WU-07 strengthens the existing release architecture rather than redesigning it around Build Once / publish-identical-bits attestation.

Current WU-07 work owns:

```text
final Specification audit and freeze decision
exact bound-revision content verification
Tool 0.3.6 release-note closure
release-note/index consistency
stale operational current-state cleanup
publication Product Artifact verification
focused release-readiness tests
full repository regression
final exact-SHA self-hosted verification
main-branch release handoff preparation
```

Current implementation direction:

- `.github/scripts/verify_release_contract.py` fails closed when a release-bound normative asset at `HEAD` differs from the exact `SPEC_REVISION` Git blob;
- canonical/embedded machine-readable contract copies must match;
- `releasenote/0.3.6.md`, Specification release note, and release-note index must carry stable current Tool/Specification markers;
- `.github/workflows/tooling-release.yml` rebuilds distributions from the published tag on self-hosted Windows and verifies the actual publication wheel's metadata/content, Product Artifact evidence, exact tagged snapshot binding, and zero definite `PTSIP-PKG-001` Product-package violations before PyPI upload;
- Strategy C release-infrastructure redesign remains deferred.

Do not claim WU-07 complete until the final development candidate receives successful exact-SHA `self-hosted/tooling-test` status and completion evidence is recorded.

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

The later preview/confirmation/safe-apply stage remains Tool `0.3.6.1` work. Blind `TOOLCHAIN -> DEVELOPMENT_TOOLING` migration remains prohibited.

## PTSIP / VPMS boundary

PTSIP classification and VPMS Verification Purpose are independent.
Current VPMS purposes remain:

```text
PRODUCT
TOOLCHAIN
```

VPMS `TOOLCHAIN` is VPMS vocabulary, not Tool `0.3.6` PTSIP classification. PTSIP core MUST NOT depend on VPMS. VPMS consumes a narrow read-only projection of already-resolved effective PTSIP metadata and does not own PTSIP classification, template/hybrid semantics, or migration authority.

## Exact release gate

At the Tool release boundary:

```text
approved Tool 0.3.6 state merged to main
    -> read fresh exact main HEAD
    -> dispatch tooling-test.yml from main
    -> exact github.sha checkout
    -> self-hosted Windows + Python 3.14 full regression/build/smoke
    -> Product Artifact evidence + exact snapshot binding
    -> self-hosted/tooling-test success on that exact main SHA
    -> no repository mutation after exact verification
    -> dispatch release.yml from the same current main SHA
    -> require origin/main == dispatched SHA
    -> verify Tool / Specification / exact bound content / release documents
    -> create draft GitHub Release targeting the same SHA
    -> maintainer reviews and publishes draft
    -> tooling-release.yml checks out published tag
    -> self-hosted publication distribution build/verification
    -> publication Product Artifact evidence + exact tagged-snapshot binding
    -> minimal GitHub-hosted GNU/Linux PyPI Trusted Publishing only
```

No manual `source_sha`, `version`, `release_candidate`, or `host_ready` workflow inputs are part of this pipeline.

Self-hosted execution is capability-bound:

```text
self-hosted
Windows
X64
PowerShell
Python 3.14 via py -3.14
```

Never hard-code a runner machine name. The narrow GNU/Linux PyPI Trusted Publishing job remains the approved GitHub-hosted compute exception.

## Repository write discipline

Before repository changes, read `AGENTS.md`, this `MEMORY.md`, `ptsip.yaml`, `src/ptsip/constants.py`, applicable bound Specification, `planning/0.3.6.md`, and the active WU-07 stage document.

Re-read the remote target branch HEAD immediately before **every GitHub write**, merge, release preparation, or evidence claim. Preserve exact entry and verification SHAs as historical authority; descendant planning/status commits record evidence but do not replace the exact verified source SHA.
