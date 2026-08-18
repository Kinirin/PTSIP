# PTSIP Repository Working Memory

This file is durable repository-operational context for maintainers and coding agents. It is **not** a normative Specification and must not replace `ptsip.yaml`, `spec/`, schemas, registry data, or ADRs.

## Current published baseline

- Canonical repository: `Kinirin/PTSIP`
- Current published Tool: `0.3.5`
- Current Tool tag: `tool-v0.3.5`
- Tool `0.3.5` release commit: `79bc4c2daf695e8462a02f2a7c4b1bb1a88846e1`
- Published Tool `0.3.5` bound Specification: `0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e`
- Tool `0.3.5` canonical PTSIP classifications are historical `PRODUCT | TOOLCHAIN | NEUTRAL_CONTRACT`.

Do not rewrite the historical meaning of Tool `0.3.5`. Tool `0.3.6` must understand valid `0.3.5` profiles only through the planned legacy migration path.

## Active Tool 0.3.6 development

Development branch:

```text
tool-0.3.6-lifecycle-ownership
```

Active plan:

```text
planning/0.3.6.md
```

Published PyPI Tool remains `0.3.5`; the development branch package/runtime identity is now `0.3.6`.

### Specification snapshots

WU-00 first `0.3.6-draft` baseline:

```text
654e41d49600fc091f9a6cb6b1c60bbc7da4e301
```

WU-03 canonical activation snapshot bound by Tool `0.3.6`:

```text
12e2ccd15634ecb3d0a4195b0f61ac3f620e7540
```

`src/ptsip/constants.py` and root `ptsip.yaml` bind `0.3.6-draft` to the WU-03 snapshot above. Do not move `SPEC_REVISION` to a later commit merely because later implementation/tests exist; select a new immutable Specification snapshot only when normative assets themselves need a new bound revision.

WU-01 lifecycle boundary determination is recorded in `decisions/ADR-0007-primary-lifecycle-boundary-determination.md`.
WU-02 role/relationship/associated-artifact semantics are recorded in `decisions/ADR-0008-responsibility-roles-relationships-associated-artifacts.md` and `spec/PTSIP-RESPONSIBILITY-MAP.md`.

## Tool 0.3.6 lifecycle model

`classification` is **primary lifecycle ownership**. Canonical classifications are exactly:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

- `PRODUCT` — Product runtime/user behavior/distribution content/runtime SDK/Product-owned quality and verification.
- `DEVELOPMENT_TOOLING` — developer authoring/inspection/validation/transformation/migration/generation/static analysis/reusable verification/development support.
- `DELIVERY` — authoritative release-unit assembly, release, signing, publication, promotion, distribution, and deployment-to-destination.
- `OPERATIONS` — post-delivery production health, recovery, reconciliation, maintenance, and ongoing operational state management.
- `NEUTRAL_CONTRACT` — deliberately non-executable, non-owning contract responsibility with independent lifecycle governance.

Artifact kind, path, framework, language, workflow provider, compilation status, and test status do not determine classification.

### Governing lifecycle obligation

Classification reasoning order is:

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

Product-specific tests may be `PRODUCT`; reusable test SDK/framework/harness infrastructure may be `DEVELOPMENT_TOOLING`. Development/intermediate build support is normally `DEVELOPMENT_TOOLING`; authoritative release-unit assembly/signing/packaging for handoff is normally `DELIVERY`. `DELIVERY` ends at the delivery handoff; ongoing post-handoff state/health/recovery/maintenance is `OPERATIONS`.

Do not choose the majority of files/jobs/steps or the highest confidence score when independently governable lifecycles are mixed.

## Responsibility Map v2

The axes are separate:

```text
classification
    = primary lifecycle ownership

roles
    = coarse responsibility characteristics

relationships
    = project-owned typed directed semantic edges

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

An associated artifact is a project-owned **non-component** support surface subordinate to exactly one classified anchor component. It has stable identity, explicit selectors/purpose, no classification or roles of its own, and at least one typed relationship connecting it to the anchor.

Associated artifacts do not inherit anchor classification. Promote/re-evaluate one as a component if independent executable/lifecycle/release/compatibility/Delivery/Operations responsibility emerges. Independently governed non-executable/non-owning lifecycle-independent contract semantics require `NEUTRAL_CONTRACT` evaluation.

Component IDs and associated-artifact IDs share one map-wide endpoint namespace. Their file scopes must not overlap in canonical explicit profiles.

## WU-03 canonical activation

WU-03 is complete at the implementation-contract level.

Canonical profile schema now supports:

```text
responsibility_map.mode:
  explicit
  template
  hybrid
```

- `explicit` directly declares components, associated artifacts, and typed relationships.
- `template` requires an explicit version/revision-bound template reference.
- `hybrid` requires an explicit template reference plus ID-addressed overrides/extensions/removals.
- Template/hybrid profiles are structurally valid before WU-04 materialization exists, but operations requiring concrete endpoints fail closed until materialized.

Canonical Tool `0.3.6` profile schema does not contain `TOOLCHAIN` and does not persist a competing `lifecycle_owner` ownership field. `DecisionAnswer.lifecycle_owner` and the current CLI option remain a transitional compatibility fact for authority/adoption inputs; after validation, canonical profile projection persists ownership only as `classification`.

Canonical policy keys are lifecycle-neutral:

```text
product_to_nonproduct_runtime_dependency: deny
nonproduct_in_product_package: deny
independent_build_resolution: required
shared_executable_cross_lifecycle: deny   # optional
neutral_contract_sharing: allow|deny      # optional
```

Old Tool `0.3.5` policy keys belong to the legacy reader/migration path, not the canonical Tool `0.3.6` schema.

### Runtime activation

WU-03 aligned:

- `schemas/ptsip-profile.schema.json`;
- agent-classification and artifact-evidence schemas;
- `registry/ptsip-registry.yaml`;
- embedded `src/ptsip/specdata/**` copies;
- `src/ptsip/model.py` five-classification/role/relationship types;
- profile semantic validation and map-level associated-artifact coverage;
- dependency, build-resolution, lifecycle, packaging, and conformance evaluators;
- adoption/resolution profile projection;
- CLI five-classification choices;
- deterministic clarification choices;
- root repository self-profile;
- VPMS read-only PTSIP bridge materialization guard;
- release Specification contract; and
- current contract/adoption/authority/conformance/topology/self-profile tests touched by the canonical activation.

Associated-artifact declarations suppress speculative component clarification for their declared scope. Explicit promotion/adoption remains a separate project-owner action.

Root `ptsip.yaml` structurally self-adopts Responsibility Map v2. Repository release automation is `DELIVERY`; repository verification/CI/maintenance are `DEVELOPMENT_TOOLING`; specification governance support is represented as an associated artifact anchored to canonical contracts.

## Tool 0.3.5 compatibility boundary

Compatibility means:

```text
legacy profile reader
    -> evidence collection
    -> lifecycle/role/relationship/split/associated-artifact proposals
    -> project-owner confirmation
    -> canonical 0.3.6 profile
```

Compatibility does **not** mean retaining `TOOLCHAIN` as a canonical alias. A legacy `TOOLCHAIN` component may map to `PRODUCT`, `DEVELOPMENT_TOOLING`, `DELIVERY`, `OPERATIONS`, a split, or unresolved clarification.

Legacy `consumers`, `analysis_inputs`, `lifecycle_owner`, old boundaries, and untyped dependency-policy edges are migration evidence. They are not blind canonical field/relationship mappings.

WU-07/WU-08 implement the actual legacy reader and migration analyzer. Direct canonical adoption intentionally refuses legacy boundary-root profiles instead of silently rewriting them.

## VPMS boundary

PTSIP classification and VPMS Verification Purpose are independent.

Current VPMS purposes remain:

```text
PRODUCT
TOOLCHAIN
```

VPMS `TOOLCHAIN` is VPMS vocabulary, not a Tool `0.3.6` PTSIP classification. Do not rename it as an accidental effect of PTSIP migration.

PTSIP core must not depend on VPMS. VPMS consumes only narrow read-only PTSIP metadata. An unmaterialized template/hybrid Responsibility Map must fail closed at the VPMS bridge rather than appear to contain zero targets.

VPMS PASS is not PTSIP CONFORMANT, and PTSIP CONFORMANT is not VPMS PASS.

## Tool 0.3.6 work-unit order

```text
WU-00  0.3.6-draft normative baseline                    COMPLETE
WU-01  lifecycle ontology/boundary rules                  COMPLETE
WU-02  roles + typed relationships + associated artifacts COMPLETE
WU-03  canonical Responsibility Map v2 activation         COMPLETE
WU-04  template catalog + deterministic materialization   NEXT
WU-05  candidate-discovery evidence expansion
WU-06  evidence/provenance/adapter normalization
WU-07  Tool 0.3.5 legacy reader
WU-08  lifecycle migration analyzer
WU-09  split/relationship/associated-artifact proposals
WU-10  preview/apply migration workflow
WU-11  repository self-adoption/migration dogfood
WU-12  tests/documentation/release completion
WU-13  Specification/release binding verification
```

WU-03 completion does not claim that the full repository regression suite has been executed. Full verification remains a WU-12/release-candidate responsibility on the approved self-hosted runner.

## Specification release rule

For Tool `X.Y.Z`, release preparation requires:

```text
Tool X.Y.Z
    -> Specification X.Y.Z-draft
    -> immutable SPEC_REVISION
```

Root `ptsip.yaml`, Tool constants, canonical Specification family, embedded specdata, registry, release-contract evidence, and Specification release note must agree at the release boundary.

`spec/PTSIP-RESPONSIBILITY-MAP.md` is a required canonical `0.3.6-draft` companion and `.github/scripts/verify_release_contract.py` must require it.

Normal development merges may occur before release candidacy. At the explicit release boundary, finalize and commit the reviewed `releasenote/X.Y.Z.md` before exact-SHA verification.

Required release sequence:

```text
merged main + reviewed releasenote/X.Y.Z.md
    -> exact immutable main SHA
    -> Tool X.Y.Z / Specification X.Y.Z-draft / immutable SPEC_REVISION check
    -> tooling-test.yml on approved self-hosted Windows
         release_candidate=true
         source_sha=<exact SHA>
    -> self-hosted/release-verification on that exact SHA
    -> release.yml on approved self-hosted Windows with the same SHA
    -> draft release targets the same SHA and does not mutate main
    -> publish reviewed draft
    -> tooling-release.yml self-hosted distribution build on published tag
    -> minimal GitHub-hosted GNU/Linux PyPI Trusted Publishing only
```

No repository mutation may be inserted between successful exact-SHA release verification and draft release creation.

## Coding-agent read order

Before repository changes, read:

1. `AGENTS.md`
2. this `MEMORY.md`
3. `ptsip.yaml`
4. `src/ptsip/constants.py`
5. applicable Specification under `spec/`
6. active version plan under `planning/`

Re-read the remote branch HEAD immediately before every write, merge, release preparation, or evidence claim. Do not rely on a previously observed SHA.

## Workflow resource policy

GitHub-hosted Actions usage is constrained. Repository verification, release preparation, and distribution building default to self-hosted execution.

Approved Windows self-hosted runner:

```text
DESKTOP-5HCCQIR
```

Rules:

- Before dispatching `tooling-test.yml` or `release.yml`, tell the user that the self-hosted runner will be used and wait for explicit confirmation that the host and PowerShell environment are ready.
- Both manual workflows require `host_ready=true` and validate `RUNNER_NAME == DESKTOP-5HCCQIR`.
- Release-candidate `tooling-test.yml` additionally requires the exact full `source_sha`.
- A successful candidate run records `self-hosted/release-verification` for that exact SHA.
- `release.yml` requires the same SHA, requires it still be current `origin/main`, and creates the draft release without committing/pushing.
- `tooling-release.yml` performs build/distribution verification on the approved self-hosted Windows runner.

### Minimal GitHub-hosted exception

The only approved GitHub-hosted compute exception is the narrow GNU/Linux PyPI publish job because `pypa/gh-action-pypi-publish` is Docker-based and cannot execute on the approved Windows runner.

That job may only download already verified distributions and perform Trusted Publishing. Do not move tests, compilation, package building, release preparation, or other avoidable compute into it.

New verification/regression/release-preparation/build-smoke workflows must default to self-hosted execution unless the maintainer explicitly approves an exception.

## Release-note discipline

Generated `git-cliff` output is only a starting draft. Architecture changes, migration semantics, compatibility boundaries, and verification evidence require human review.

Do not finalize Tool `releasenote/0.3.6.md` until the explicit release boundary. `releasenote/spec-0.3.6-draft.md` already records the bound Specification snapshot and is distinct from the final Tool release note.
