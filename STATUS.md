# PTSIP Status

## Current published release

- Canonical repository: `Kinirin/PTSIP`
- Maturity: Experimental
- Current published Tool/package version: **`0.3.5`**
- Current Tool release tag: **`tool-v0.3.5`**
- Tool `0.3.5` release commit: `79bc4c2daf695e8462a02f2a7c4b1bb1a88846e1`
- Tool `0.3.5` GitHub Release: **PUBLISHED** on 2026-08-17
- Tool `0.3.5` PyPI publication: **COMPLETE** through Trusted Publishing
- Bound published Specification: `0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e`
- Supported Python metadata: Python 3.11–3.14
- Tool release namespace: `tool-v*`
- Specification release/design namespace: `spec-v*`
- License: Apache License 2.0

Tool `0.3.5` remains the published compatibility baseline and retains its historical PTSIP classifications:

```text
PRODUCT
TOOLCHAIN
NEUTRAL_CONTRACT
```

Tool `0.3.5` is also the first published Tool containing VPMS. Current VPMS Verification Purpose vocabulary remains `PRODUCT | TOOLCHAIN`; that VPMS `TOOLCHAIN` token is independent from the Tool `0.3.6` PTSIP lifecycle ontology.

## Active Tool 0.3.6 development

Development branch:

```text
tool-0.3.6-lifecycle-ownership
```

Development package/runtime identity on that branch is `0.3.6`. This does **not** mean Tool `0.3.6` has been published.

Active Specification family:

```text
0.3.6-draft
```

Current bound immutable WU-04C Specification snapshot:

```text
82abd09360df09a95fbbfb516855fa9ffb49f050
```

The earlier WU-03 activation snapshot was `12e2ccd15634ecb3d0a4195b0f61ac3f620e7540`. The binding advanced because WU-04C changed normative declaration-authority/materialization semantics.

### Work-unit progress

```text
WU-00  0.3.6-draft normative baseline                    COMPLETE
WU-01  lifecycle ontology/boundary rules                  COMPLETE
WU-02  roles + typed relationships + associated artifacts COMPLETE
WU-03  canonical Responsibility Map v2 activation         COMPLETE
WU-04  template/materialization/effective-map pipeline    IN PROGRESS
```

WU-04 staged progress:

```text
WU-04A  template catalog identity                         COMPLETE
WU-04B  deterministic materializer core                  COMPLETE
WU-04C  declaration authority/source_mode boundary       COMPLETE
WU-04D  ResolvedProfile + digest + provenance            NEXT / NOT ENTERED
WU-04E  validation consumes effective map                LOCKED
WU-04F  conformance consumes effective map               LOCKED
WU-04G  clarification/adoption read integration          LOCKED
WU-04H  VPMS narrow read-only integration                LOCKED
WU-04I  regression + WU-04 completion                    LOCKED
```

The only WU-04 sub-document created so far is the completed WU-04C record:

```text
planning/0.3.6/WU-04C-declaration-authority-effective-map.md
```

There is intentionally no WU-04D sub-document yet. Future sub-documents are created only when the preceding stage gate is complete and that next stage is actually entered after a fresh branch-HEAD read.

## Tool 0.3.6 canonical lifecycle model

Canonical classifications are:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

`TOOLCHAIN` is not a canonical Tool `0.3.6` classification alias. It remains historical/migration input for the planned legacy reader.

Responsibility Map v2 supports:

- explicit/template/hybrid declaration modes;
- optional multi-valued closed component roles;
- typed directed responsibility relationships;
- associated artifacts subordinate to exactly one classified anchor;
- map-wide component/associated-artifact endpoint identity;
- lifecycle-neutral canonical policy keys; and
- exact revision-bound template references.

## WU-04 declaration authority boundary

WU-04C freezes these separate responsibilities:

```text
classification
    = primary lifecycle ownership

source_mode / derived entity origin
    = declaration authority provenance

materializer
    = deterministic non-authoritative resolution
```

`source_mode` values are:

```text
explicit
template
hybrid
```

They are not lifecycle classifications.

Authority semantics:

- `explicit`: project owns the complete declaration;
- `template`: project explicitly adopts one exact template ID + immutable revision;
- `hybrid`: project owns exact template selection plus stable-ID replacements/extensions/removals;
- unchanged template declarations retain template origin;
- project replacement/extension/removal outranks selected template declaration;
- hybrid replacement is whole-entity by stable ID, not implicit field-level inheritance.

The materializer must not infer missing architecture, auto-select templates, rewrite lifecycle classifications, fabricate responsibilities, silently repair/delete dangling relationships, cascade project removals, or mutate the source declaration merely to produce a valid map.

All source modes resolve to one **Canonical Effective Responsibility Map** for downstream semantic consumers while source declaration and template identity remain separately inspectable.

Normative rules are `PTSIP-RMAP-013` through `PTSIP-RMAP-016`, with the architecture decision recorded in `ADR-0009`.

## WU-04 catalog/materializer baseline

Initial templates:

```text
python-package-library
python-cli-application
mixed-product-development-delivery
```

Template revision is an immutable SHA-256 over canonical JSON of template semantic map content.

First WU-04 catalog/materializer implementation commit:

```text
5be7623fa1b750a1c11e349fd7f00073233d9595
```

WU-04D will introduce the common `ResolvedProfile` abstraction, deterministic effective-map digest, and derived declaration provenance. It has not been entered yet.

## Repository self-profile

Root `ptsip.yaml` structurally self-adopts Responsibility Map v2.

- release automation: `DELIVERY`;
- repository verification/CI/maintenance: `DEVELOPMENT_TOOLING`;
- canonical contracts: `NEUTRAL_CONTRACT` where lifecycle-independent contract semantics hold;
- subordinate specification governance support: associated artifact where separate component ownership is unnecessary.

## Compatibility direction

Tool `0.3.6` compatibility with Tool `0.3.5` means assisted migration, not preservation of obsolete ontology in the canonical schema:

```text
Tool 0.3.5 profile
    -> legacy reader
    -> repository evidence
    -> lifecycle/role/relationship/split/artifact proposals
    -> project-owner confirmation
    -> canonical Tool 0.3.6 Responsibility Map
```

A legacy `TOOLCHAIN` responsibility may become `PRODUCT`, `DEVELOPMENT_TOOLING`, `DELIVERY`, `OPERATIONS`, a split, or unresolved clarification. Blind `TOOLCHAIN -> DEVELOPMENT_TOOLING` translation is prohibited.

## VPMS boundary

PTSIP and VPMS remain sibling concerns:

```text
PTSIP
    Which lifecycle owns this project responsibility?

VPMS
    Why does this verification exist / what does it protect?
```

A PTSIP `DEVELOPMENT_TOOLING` verifier may still carry VPMS Verification Purpose `PRODUCT`. VPMS purpose must not be renamed merely because PTSIP lifecycle classification changed.

PTSIP core must not depend on VPMS. During WU-04H, PTSIP resolves the effective map first and VPMS consumes only narrow read-only effective component metadata.

## Verification status for Tool 0.3.6 development

Full self-hosted repository regression has **not** been claimed for WU-04. No self-hosted workflow was dispatched as part of WU-04A/B/C.

Full regression/package verification remains part of WU-12 and exact release-candidate verification.

## Verification and publication evidence for published Tool 0.3.5

Historical final Tool `0.3.5` maintainer-local repository regression:

```text
python -m pytest -q
244 passed in 571.90s
```

Earlier Tool `0.3.5` package-boundary verification also established successful wheel/sdist build, `twine check`, installed-wheel smoke, VPMS package inclusion, Tool identity `0.3.5`, and Specification identity `0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e`.

Tool `0.3.5` verification/publication is historical evidence only and must not be represented as Tool `0.3.6` verification.

## Current workflow resource policy

Repository verification, release preparation, and distribution build compute default to the approved self-hosted Windows runner:

```text
DESKTOP-5HCCQIR
```

`tooling-test.yml` and `release.yml` are manually dispatched and require explicit host readiness. Release-candidate verification is bound to the exact candidate SHA.

`tooling-release.yml` performs build/distribution verification on self-hosted Windows. The narrow GitHub-hosted GNU/Linux PyPI Trusted Publishing job is the only current hosted-compute exception because the publishing action is Docker-based; it must not absorb tests, compilation, release preparation, or package building.

No Tool `0.3.6` release verification or publication is claimed until the required self-hosted exact-SHA gates have actually completed.

## Tool lineage

- Tool `0.3.0`: published
- Tool `0.3.1`: published
- Tool `0.3.2`: source-only migration
- Tool `0.3.3`: permanently source-only
- Tool `0.3.4`: published historical Tool release
- Tool `0.3.5`: **published; first VPMS-capable Tool release**
- Tool `0.3.6`: **active development; WU-04C complete, WU-04D next/not entered**

Current next-version work remains consolidated under `planning/0.3.6.md`. Do not create a separate Tool `0.4.0` plan unless an independently consequential scope emerges beyond the current `0.3.6` contract.
