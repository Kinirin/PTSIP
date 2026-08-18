# PTSIP Repository Working Memory

This file is durable repository-operational context for maintainers and coding agents. It is **not** a normative Specification and must not replace `ptsip.yaml`, `spec/`, schemas, registry data, or ADRs.

## Current published baseline

- Canonical repository: `Kinirin/PTSIP`
- Current published Tool: `0.3.5`
- Current Tool tag: `tool-v0.3.5`
- Tool `0.3.5` release commit: `79bc4c2daf695e8462a02f2a7c4b1bb1a88846e1`
- Published Tool `0.3.5` bound Specification: `0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e`
- Historical Tool `0.3.5` canonical classifications: `PRODUCT | TOOLCHAIN | NEUTRAL_CONTRACT`

Do not rewrite historical Tool `0.3.5` semantics. Tool `0.3.6` understands valid `0.3.5` profiles only through the planned legacy-reader/migration path.

## Active Tool 0.3.6 development

Development branch:

```text
tool-0.3.6-lifecycle-ownership
```

Master plan:

```text
planning/0.3.6.md
```

Published PyPI Tool remains `0.3.5`; the development branch package/runtime identity is `0.3.6`.

## Specification snapshots

WU-00 first `0.3.6-draft` baseline:

```text
654e41d49600fc091f9a6cb6b1c60bbc7da4e301
```

WU-03 canonical activation snapshot:

```text
12e2ccd15634ecb3d0a4195b0f61ac3f620e7540
```

WU-04C declaration-authority/materialization snapshot now bound by Tool `0.3.6`:

```text
82abd09360df09a95fbbfb516855fa9ffb49f050
```

`src/ptsip/constants.py` and root `ptsip.yaml` bind `0.3.6-draft` to the WU-04C snapshot above.

The binding moved from the WU-03 snapshot because WU-04C changed normative Responsibility Map semantics (`PTSIP-RMAP-013` through `PTSIP-RMAP-016`). Do not move `SPEC_REVISION` merely because later implementation/tests exist; select a new immutable snapshot only when normative Specification assets themselves change.

Decision records:

- `ADR-0007` — lifecycle boundary determination;
- `ADR-0008` — roles, typed relationships, associated artifacts;
- `ADR-0009` — declaration authority, source mode, and non-authoritative materialization.

## Tool 0.3.6 lifecycle model

`classification` is **primary lifecycle ownership**. Canonical classifications are exactly:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

- `PRODUCT` — Product runtime/user behavior/distribution content/runtime SDK/Product-owned verification.
- `DEVELOPMENT_TOOLING` — development authoring/inspection/validation/transformation/migration/generation/reusable verification/support.
- `DELIVERY` — authoritative release-unit assembly, release, signing, publication, promotion, distribution, deployment-to-destination.
- `OPERATIONS` — post-delivery production health, recovery, reconciliation, maintenance, ongoing operational state.
- `NEUTRAL_CONTRACT` — non-executable, non-owning contract responsibility with lifecycle-independent governance.

Artifact kind, path, framework, language, workflow provider, compilation status, and test status do not determine classification.

Classification reasoning order:

```text
project-owned scope
    -> coherent responsibility boundary
    -> evidence + provenance
    -> NEUTRAL_CONTRACT qualification
    -> governing owning lifecycle
    -> mixed-lifecycle split test
    -> classification / split / unresolved
    -> project-owner confirmation for inferred architecture
```

Product-specific tests may be `PRODUCT`; reusable test SDK/framework/harness infrastructure may be `DEVELOPMENT_TOOLING`. Development/intermediate build support is normally `DEVELOPMENT_TOOLING`; authoritative release-unit assembly/signing/packaging for handoff is normally `DELIVERY`. Delivery ends at the delivery handoff; ongoing state/health/recovery/maintenance is Operations.

Do not choose majority files/jobs/steps or confidence score when independently governable lifecycle responsibilities are mixed.

## Responsibility Map v2

Keep semantic axes separate:

```text
classification
    = primary lifecycle ownership

roles
    = coarse responsibility characteristics

relationships
    = project-owned typed directed semantic edges

source_mode / declaration provenance
    = where architecture declaration authority came from

VPMS Verification Purpose
    = what verification protects/verifies
```

Canonical component roles:

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

Evidence `TESTS` remains evidence vocabulary and may support a `VERIFIES` proposal; evidence is not silently promoted into project-owned declaration.

### Associated artifacts

An associated artifact is a project-owned non-component support surface subordinate to exactly one classified anchor component. It has stable identity, explicit selectors/purpose, no classification or roles of its own, and at least one typed relationship connecting it to the anchor.

Associated artifacts do not inherit anchor classification. Promote/re-evaluate as components if independent executable/lifecycle/release/compatibility/Delivery/Operations responsibility emerges. Independently governed non-executable/non-owning lifecycle-independent contract semantics require `NEUTRAL_CONTRACT` evaluation.

Component IDs and associated-artifact IDs share one map-wide endpoint namespace.

## WU-03 canonical activation

WU-03 is complete.

Canonical schema/runtime supports:

```text
responsibility_map.mode:
  explicit
  template
  hybrid
```

Canonical Tool `0.3.6` profile schema has no `TOOLCHAIN` classification alias and no competing persisted `lifecycle_owner` field. Transitional authority/adoption inputs may still carry historical ownership facts, but canonical profile ownership is `classification`.

Canonical policy keys are lifecycle-neutral:

```text
product_to_nonproduct_runtime_dependency: deny
nonproduct_in_product_package: deny
independent_build_resolution: required
shared_executable_cross_lifecycle: deny   # optional
neutral_contract_sharing: allow|deny      # optional
```

Root `ptsip.yaml` structurally self-adopts Responsibility Map v2. Repository release automation is `DELIVERY`; repository verification/CI/maintenance are `DEVELOPMENT_TOOLING`; specification governance support is an associated artifact anchored to canonical contracts.

## WU-04 staged template/materialization development

WU-04 is **IN PROGRESS** and is now explicitly staged.

### WU-04A — template catalog identity — COMPLETE

Initial catalog:

```text
python-package-library
python-cli-application
mixed-product-development-delivery
```

Template identity is stable ID + immutable semantic revision. Revision is a SHA-256 of canonical JSON for the template semantic map.

### WU-04B — deterministic materializer core — COMPLETE

First materializer implementation commit:

```text
5be7623fa1b750a1c11e349fd7f00073233d9595
```

Implemented direction:

- exact `id + revision` resolution;
- read-only explicit/template/hybrid materialization;
- stable-ID whole-entity replacement;
- known-ID removal;
- new-ID extension;
- fail closed on unknown remove and replace/remove collision;
- no automatic template inference.

### WU-04C — declaration authority + Canonical Effective Map boundary — COMPLETE

Completed record:

```text
planning/0.3.6/WU-04C-declaration-authority-effective-map.md
```

Accepted decision:

```text
decisions/ADR-0009-responsibility-map-declaration-authority.md
```

Normative rules:

```text
PTSIP-RMAP-013  Declaration source does not alter lifecycle ownership
PTSIP-RMAP-014  Exact template selection is project-owned architecture authority
PTSIP-RMAP-015  Hybrid precedence is stable-ID whole-entity authority
PTSIP-RMAP-016  Materialization is deterministic and non-authoritative
```

Responsibility boundaries:

```text
classification
    = lifecycle responsibility

source_mode / derived entity origin
    = declaration authority provenance

materializer
    = deterministic non-authoritative resolution
```

Mode authority:

```text
explicit
    project owns complete declaration

template
    project owns exact template selection
    selected immutable revision supplies adopted declaration content

hybrid
    project owns exact selection + replace/extend/remove decisions
    template supplies only unchanged declarations
```

Precedence:

```text
project replacement / extension / removal
    > selected immutable template declaration
```

Derived runtime/review origins may use:

```text
PROJECT_EXPLICIT
TEMPLATE
PROJECT_OVERRIDE
PROJECT_EXTENSION
PROJECT_REMOVAL
```

This provenance is not lifecycle classification and need not be persisted into canonical `ptsip.yaml`.

Materializer MUST NOT infer architecture, auto-select templates, change classification, fabricate responsibilities, silently repair/delete dangling relationships, cascade removals, resolve conflicts by confidence/path heuristics, or mutate source declarations merely to obtain a valid map.

All modes resolve to one **Canonical Effective Responsibility Map** while retaining source declaration/template identity separately.

### WU-04D — NEXT / NOT ENTERED

Responsibility:

```text
ResolvedProfile
+ source declaration
+ Canonical Effective Responsibility Map
+ exact template identity
+ effective_map_digest
+ derived declaration provenance
```

WU-04D has **not been entered** and no WU-04D sub-document should exist yet.

Sub-document creation rule:

```text
complete predecessor
    -> fresh-read development branch HEAD
    -> create only next sub-stage document
    -> record exact entry baseline
    -> mark next stage ACTIVE
    -> perform only that stage scope
```

WU-04E through WU-04I remain LOCKED until entered in order.

Current WU-04 order:

```text
WU-04A  catalog identity                               COMPLETE
WU-04B  materializer core                              COMPLETE
WU-04C  declaration authority/effective-map boundary   COMPLETE
WU-04D  ResolvedProfile/digest/provenance              NEXT / NOT ENTERED
WU-04E  validation effective-map consumption           LOCKED
WU-04F  conformance effective-map consumption          LOCKED
WU-04G  clarification/adoption read integration        LOCKED
WU-04H  VPMS narrow read-only integration              LOCKED
WU-04I  regression/WU-04 completion                    LOCKED
```

## Tool 0.3.5 compatibility boundary

Compatibility means:

```text
legacy profile reader
    -> evidence collection
    -> lifecycle/role/relationship/split/artifact proposals
    -> project-owner confirmation
    -> canonical 0.3.6 profile
```

Compatibility does not mean retaining `TOOLCHAIN` as an alias. Legacy `TOOLCHAIN` may map to Product, Development Tooling, Delivery, Operations, split, or unresolved clarification.

Legacy `consumers`, `analysis_inputs`, `lifecycle_owner`, old boundaries, and untyped dependency-policy edges are migration evidence, not blind canonical mappings.

WU-07/WU-08 implement the actual legacy reader/migration analyzer. Do not jump to migration writes before WU-04 through WU-06 prerequisites are satisfied.

## VPMS boundary

PTSIP lifecycle classification and VPMS Verification Purpose are independent.

Current VPMS purposes remain:

```text
PRODUCT
TOOLCHAIN
```

VPMS `TOOLCHAIN` is VPMS vocabulary, not Tool `0.3.6` PTSIP classification.

PTSIP core must not depend on VPMS. During WU-04H, PTSIP must materialize first and VPMS should consume only narrow effective component metadata; VPMS must not implement template semantics itself.

VPMS PASS is not PTSIP CONFORMANT, and PTSIP CONFORMANT is not VPMS PASS.

## Tool 0.3.6 overall order

```text
WU-00  0.3.6-draft normative baseline                    COMPLETE
WU-01  lifecycle ontology/boundary rules                  COMPLETE
WU-02  roles + typed relationships + associated artifacts COMPLETE
WU-03  canonical Responsibility Map v2 activation         COMPLETE
WU-04  templates/materialization/effective map            IN PROGRESS
WU-05  candidate-discovery evidence expansion             PLANNED
WU-06  evidence/provenance normalization                  PLANNED
WU-07  Tool 0.3.5 legacy reader                           PLANNED
WU-08  lifecycle migration analyzer                       PLANNED
WU-09  target architecture proposals                      PLANNED
WU-10  preview/confirm/safe apply                          PLANNED
WU-11  repository dogfood                                 PLANNED
WU-12  tests/docs/package verification                    PLANNED
WU-13  final Specification/release boundary               PLANNED
```

WU-04 progress does not claim the full repository regression suite has been executed. Full verification remains a WU-12/release-candidate responsibility on the approved self-hosted runner.

## Specification release rule

For Tool `X.Y.Z`:

```text
Tool X.Y.Z
    -> Specification X.Y.Z-draft
    -> immutable SPEC_REVISION
```

Root `ptsip.yaml`, Tool constants, canonical Specification family, embedded specdata, registry, release-contract evidence, and Specification release note must agree at the release boundary.

`spec/PTSIP-RESPONSIBILITY-MAP.md` is a required canonical `0.3.6-draft` companion.

At explicit Tool release boundary, finalize and commit reviewed `releasenote/X.Y.Z.md` before exact-SHA verification.

Required release sequence:

```text
merged main + reviewed releasenote/X.Y.Z.md
    -> exact immutable main SHA
    -> Tool/Specification/binding contract
    -> tooling-test.yml on approved self-hosted Windows
         release_candidate=true
         source_sha=<exact SHA>
    -> self-hosted/release-verification on exact SHA
    -> release.yml on approved self-hosted Windows with same SHA
    -> draft release targets same SHA without mutating main
    -> publish reviewed draft
    -> tooling-release.yml self-hosted distribution build
    -> minimal GitHub-hosted GNU/Linux PyPI Trusted Publishing only
```

## Coding-agent read/write discipline

Before repository changes, read:

1. `AGENTS.md`
2. this `MEMORY.md`
3. `ptsip.yaml`
4. `src/ptsip/constants.py`
5. applicable Specification under `spec/`
6. `planning/0.3.6.md`
7. only the currently ACTIVE sub-stage document, when one exists

Re-read the remote target branch HEAD immediately before **every GitHub write**, merge, release preparation, or evidence claim. Do not rely on an older observed SHA.

## Workflow resource policy

GitHub-hosted Actions usage is constrained. Repository verification, release preparation, and distribution building default to self-hosted execution.

Approved Windows self-hosted runner:

```text
DESKTOP-5HCCQIR
```

Rules:

- Before dispatching `tooling-test.yml` or `release.yml`, tell the user that the self-hosted runner will be used and wait for explicit confirmation that host and PowerShell environment are ready.
- Both workflows require `host_ready=true` and validate `RUNNER_NAME == DESKTOP-5HCCQIR`.
- Release-candidate `tooling-test.yml` requires exact full `source_sha` and records `self-hosted/release-verification` for that SHA.
- `release.yml` requires the same SHA, verifies current `origin/main`, and creates the draft release without committing/pushing.
- `tooling-release.yml` performs build/distribution verification on approved self-hosted Windows.

The only approved GitHub-hosted compute exception is narrow GNU/Linux PyPI Trusted Publishing. It may only download already verified distributions and publish them. Do not move tests, compilation, package building, release preparation, or other avoidable compute into it.

## Release-note discipline

Generated changelog output is only a starting draft. Architecture/migration/compatibility/verification claims require review.

Do not finalize Tool `releasenote/0.3.6.md` until the explicit release boundary. `releasenote/spec-0.3.6-draft.md` tracks the active Specification binding and is distinct from the final Tool release note.
