# PTSIP Repository Working Memory

This file is durable repository-operational context for maintainers and coding agents. It is **not** a normative Specification and does not replace `ptsip.yaml`, `spec/`, schemas, registry data, ADRs, or the active plan.

## Published baseline

- Canonical repository: `Kinirin/PTSIP`
- Published Tool: `0.3.5`
- Tag: `tool-v0.3.5`
- Release commit: `79bc4c2daf695e8462a02f2a7c4b1bb1a88846e1`
- Published bound Specification: `0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e`
- Historical Tool `0.3.5` classifications: `PRODUCT | TOOLCHAIN | NEUTRAL_CONTRACT`

Do not rewrite Tool `0.3.5` history. Tool `0.3.6` understands valid `0.3.5` profiles only through the planned legacy-reader/migration path.

## Active Tool 0.3.6 development

Development branch:

```text
tool-0.3.6-lifecycle-ownership
```

Master plan:

```text
planning/0.3.6.md
```

Active sub-stage:

```text
planning/0.3.6/WU-04D-resolved-profile-digest-provenance.md
```

WU-04D exact entry baseline:

```text
d8713ac4e684852f3e6cf67a68165f82ae0b80aa
```

Published PyPI Tool remains `0.3.5`; the development branch Tool identity is `0.3.6`.

## Specification snapshots

```text
WU-00 baseline:              654e41d49600fc091f9a6cb6b1c60bbc7da4e301
WU-03 canonical activation:  12e2ccd15634ecb3d0a4195b0f61ac3f620e7540
WU-04C authority snapshot:   82abd09360df09a95fbbfb516855fa9ffb49f050
```

Current Tool/root-profile binding is:

```text
Specification 0.3.6-draft
SPEC_REVISION 82abd09360df09a95fbbfb516855fa9ffb49f050
```

The WU-04C snapshot added normative `PTSIP-RMAP-013` through `PTSIP-RMAP-016`. WU-04D implements those frozen semantics and does not by itself justify moving `SPEC_REVISION`. Select another immutable snapshot only when normative Specification assets actually change.

Decision records:

- `ADR-0007` — primary lifecycle boundary determination;
- `ADR-0008` — roles, typed relationships, associated artifacts;
- `ADR-0009` — declaration authority, source mode, and non-authoritative materialization.

## Canonical Tool 0.3.6 lifecycle model

`classification` is primary lifecycle ownership. Canonical values are exactly:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

`TOOLCHAIN` is Tool `0.3.5` migration input only, not a Tool `0.3.6` alias.

Classification reasoning order:

```text
project-owned scope
    -> coherent responsibility boundary
    -> evidence + provenance
    -> NEUTRAL_CONTRACT qualification
    -> governing lifecycle obligation
    -> material mixed-lifecycle split test
    -> classification / split / unresolved
    -> project-owner confirmation for inferred architecture
```

Important boundaries:

- Product-owned tests may be `PRODUCT`.
- Reusable verification/test SDK/framework/harness may be `DEVELOPMENT_TOOLING`.
- Development/intermediate build support is normally `DEVELOPMENT_TOOLING`.
- Authoritative release-unit assembly/signing/packaging/publication/deployment-to-destination is normally `DELIVERY`.
- `DELIVERY` ends at delivery handoff; ongoing deployed-state health/recovery/reconciliation/maintenance is `OPERATIONS`.
- `NEUTRAL_CONTRACT` requires non-executable, non-owning, lifecycle-independent contract governance.
- Mixed independently governable lifecycles should split rather than select a majority lifecycle.

Path, technology, framework, language, file kind, test status, workflow provider, executable status, compilation behavior, invocation frequency, and confidence score are not classification authority.

## Responsibility Map v2

Keep semantic axes separate:

```text
classification
    = primary lifecycle ownership

roles
    = coarse responsibility characteristics

relationships
    = project-owned typed directed semantics

source_mode / derived origin
    = declaration authority provenance

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

Evidence `TESTS` may support a `VERIFIES` proposal but does not silently create a project-owned relationship.

### Associated artifacts

An associated artifact is a project-owned non-component support surface subordinate to exactly one classified anchor component. It has stable identity, selectors/purpose, no classification or component roles, no anchor-classification inheritance, and at least one typed relationship to its anchor.

Promote/re-evaluate as a component if independent lifecycle/release/compatibility/executable/Delivery/Operations responsibility emerges. Independently governed non-executable/non-owning lifecycle-independent contract semantics require `NEUTRAL_CONTRACT` evaluation.

Component IDs and associated-artifact IDs share one map-wide endpoint namespace.

## WU-04 authority model

Responsibility Map source modes:

```text
explicit
template
hybrid
```

Frozen boundary:

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
    project owns exact template ID + immutable revision selection
    selected revision supplies adopted declaration content

hybrid
    project owns exact selection + stable-ID replacement/extension/removal
    selected template supplies only unchanged entities
```

Hybrid precedence:

```text
project replacement / extension / removal
    > selected immutable template declaration
```

Derived runtime/review origin vocabulary:

```text
PROJECT_EXPLICIT
TEMPLATE
PROJECT_OVERRIDE
PROJECT_EXTENSION
PROJECT_REMOVAL
```

This provenance is not lifecycle classification and is not required canonical Project Profile serialization.

Materializer MUST NOT infer architecture, auto-select templates, change classifications, fabricate responsibilities, silently repair dangling relationships, cascade removals, resolve conflicts by confidence/path heuristics, or mutate source declarations merely to produce a valid effective map.

## WU-04 stage state

```text
WU-04A  template catalog identity                         COMPLETE
WU-04B  deterministic materializer core                  COMPLETE
WU-04C  declaration authority/source_mode boundary       COMPLETE
WU-04D  ResolvedProfile + digest + provenance            ACTIVE
WU-04E  validation consumes effective map                LOCKED
WU-04F  conformance consumes effective map               LOCKED
WU-04G  clarification/adoption read integration          LOCKED
WU-04H  VPMS narrow read-only integration                LOCKED
WU-04I  regression + WU-04 completion                    LOCKED
```

WU-04D currently owns only:

```text
source declaration
      |
      v
deterministic materialization
      |
      v
ResolvedProfile
      +--> preserved source_payload
      +--> effective_payload
      +--> source_mode + exact template identity
      +--> effective_map_digest
      +--> derived entity/removal provenance
```

Primary implementation target:

```text
src/ptsip/validation/templates.py
```

Focused regression target:

```text
tests/ptsip/test_template_materialization_036.py
```

WU-04D has introduced the `ResolvedProfile` runtime view and focused regression cases on the development branch, but WU-04D remains **ACTIVE** until its completion gate is explicitly closed. Do not claim focused tests or full regression passed unless execution evidence exists.

Do not migrate validation, conformance, clarification/adoption, or VPMS consumers to the resolved view during WU-04D. Those integrations belong to WU-04E through WU-04H.

Future sub-stage documents MUST NOT be created before their entry gate. WU-04E and later sub-documents do not belong in the repository while WU-04D is ACTIVE.

## Tool 0.3.5 compatibility boundary

Compatibility means:

```text
legacy profile reader
    -> evidence collection
    -> lifecycle/role/relationship/split/artifact proposals
    -> project-owner confirmation
    -> canonical Tool 0.3.6 Responsibility Map
```

Compatibility does not mean retaining `TOOLCHAIN` as an alias. Legacy `TOOLCHAIN` may resolve to Product, Development Tooling, Delivery, Operations, split, or unresolved clarification.

Legacy `lifecycle_owner`, `consumers`, `analysis_inputs`, old boundary roots, and untyped dependency-policy entries are migration evidence, not automatic canonical mappings.

Migration remains preview-first, evidence-backed, loss-preserving, and project-owner-confirmed.

## VPMS boundary

PTSIP classification and VPMS Verification Purpose remain independent.

Current VPMS purposes remain:

```text
PRODUCT
TOOLCHAIN
```

VPMS `TOOLCHAIN` is VPMS vocabulary, not Tool `0.3.6` PTSIP classification.

PTSIP core MUST NOT depend on VPMS. During WU-04H, PTSIP resolves the effective map first and VPMS consumes only narrow read-only effective component metadata. VPMS must not implement template semantics itself.

VPMS PASS is not PTSIP CONFORMANT, and PTSIP CONFORMANT is not VPMS PASS.

## Overall Tool 0.3.6 order

```text
WU-00  0.3.6-draft normative baseline                    COMPLETE
WU-01  lifecycle ontology/boundary rules                  COMPLETE
WU-02  roles + typed relationships + associated artifacts COMPLETE
WU-03  canonical Responsibility Map v2 activation         COMPLETE
WU-04  templates/materialization/effective-map pipeline   IN PROGRESS
WU-05  candidate-discovery evidence expansion             PLANNED
WU-06  evidence/provenance normalization                  PLANNED
WU-07  Tool 0.3.5 legacy reader                           PLANNED
WU-08  lifecycle migration analyzer                       PLANNED
WU-09  target architecture proposals                      PLANNED
WU-10  preview/confirm/safe apply                          PLANNED
WU-11  repository dogfood                                 PLANNED
WU-12  full regression/package verification               PLANNED
WU-13  final Specification/release boundary               PLANNED
```

## Coding-agent read/write discipline

Before repository changes, read:

1. `AGENTS.md`
2. this `MEMORY.md`
3. `ptsip.yaml`
4. `src/ptsip/constants.py`
5. applicable Specification under `spec/`
6. `planning/0.3.6.md`
7. the currently ACTIVE sub-stage document

Re-read the remote target branch HEAD immediately before **every GitHub write**, merge, release preparation, or evidence claim.

## Specification release rule

For Tool `X.Y.Z`:

```text
Tool X.Y.Z
    -> Specification X.Y.Z-draft
    -> immutable SPEC_REVISION
```

At the explicit release boundary, root `ptsip.yaml`, Tool constants, canonical Specification family, embedded specdata, registry, release-contract evidence, and Specification release note must agree.

`spec/PTSIP-RESPONSIBILITY-MAP.md` is a required canonical `0.3.6-draft` companion.

Finalize and commit the reviewed Tool `releasenote/X.Y.Z.md` before exact release-candidate SHA verification.

Required sequence:

```text
merged main + reviewed releasenote/X.Y.Z.md
    -> exact immutable main SHA
    -> Tool / Specification / binding contract
    -> tooling-test.yml on approved self-hosted Windows
         host_ready=true
         release_candidate=true
         source_sha=<exact SHA>
    -> self-hosted/release-verification on exact SHA
    -> release.yml on approved self-hosted Windows with same SHA
    -> draft release targets same SHA without mutating main
    -> publish reviewed draft
    -> tooling-release.yml self-hosted distribution build
    -> minimal GitHub-hosted GNU/Linux PyPI Trusted Publishing only
```

## Workflow resource policy

Approved Windows self-hosted runner:

```text
DESKTOP-5HCCQIR
```

Before dispatching `tooling-test.yml` or `release.yml`, tell the user that `DESKTOP-5HCCQIR` will be used and wait for explicit confirmation that the host and PowerShell environment are ready.

Both manual workflows require `host_ready=true`; release-candidate verification additionally binds to exact `source_sha`.

`tooling-release.yml` performs build/distribution verification on the approved Windows runner.

The only approved GitHub-hosted compute exception is the narrow GNU/Linux PyPI Trusted Publishing job. It may only download already verified distributions and publish them. Do not move tests, compilation, package building, release preparation, or other avoidable compute into it.

## Release-note discipline

Generated changelog output is only a starting draft. Architecture, migration, compatibility, and verification claims require review.

Do not finalize Tool `releasenote/0.3.6.md` until the explicit release boundary. `releasenote/spec-0.3.6-draft.md` tracks Specification binding separately.
