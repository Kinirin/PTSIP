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

Development package/runtime identity on that branch is now `0.3.6`. This does **not** mean Tool `0.3.6` has been published.

Active Specification family:

```text
0.3.6-draft
```

Current bound immutable WU-03 Specification snapshot:

```text
12e2ccd15634ecb3d0a4195b0f61ac3f620e7540
```

### Work-unit progress

```text
WU-00  0.3.6-draft normative baseline                    COMPLETE
WU-01  lifecycle ontology/boundary rules                  COMPLETE
WU-02  roles + typed relationships + associated artifacts COMPLETE
WU-03  canonical Responsibility Map v2 activation         COMPLETE
WU-04  template catalog + deterministic materialization   NEXT
```

WU-03 activates the canonical five-classification lifecycle model:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

`TOOLCHAIN` is not a canonical Tool `0.3.6` classification alias. It remains readable only as historical/migration input once the WU-07 legacy reader is implemented.

Responsibility Map v2 now has canonical schema/runtime support for:

- explicit/template/hybrid declaration modes;
- optional multi-valued closed component roles;
- typed directed responsibility relationships;
- associated artifacts subordinate to exactly one classified anchor;
- map-wide component/associated-artifact endpoint identity;
- lifecycle-neutral canonical policy keys; and
- fail-closed template/hybrid behavior until exact template materialization is available.

The root repository profile structurally self-adopts Responsibility Map v2. Release automation is represented as `DELIVERY`; repository verification/CI/maintenance are represented as `DEVELOPMENT_TOOLING`; specification governance support uses associated-artifact semantics where independent component ownership is not required.

### WU-03 implementation boundary

Canonical activation has been applied across:

- `schemas/` profile, agent-classification, and artifact-evidence contracts;
- `registry/ptsip-registry.yaml`;
- embedded `src/ptsip/specdata/**` contract copies;
- `src/ptsip/**` lifecycle models, profile validation, dependency/build/lifecycle/conformance evaluation, adoption/projection, CLI, and clarification behavior;
- `ptsip.yaml` repository self-profile;
- narrow `src/vpms/` PTSIP metadata bridge behavior without renaming VPMS purposes;
- Tool version/constants and immutable Specification binding;
- the release Specification contract; and
- current regression contracts directly affected by Responsibility Map v2 activation.

Full self-hosted repository regression has **not** been claimed for WU-03. It remains part of WU-12/release-candidate verification.

## Compatibility direction

Tool `0.3.6` compatibility with Tool `0.3.5` means assisted migration, not preservation of obsolete ontology in the new canonical schema:

```text
Tool 0.3.5 profile
    -> legacy reader
    -> repository evidence
    -> lifecycle/role/relationship/split proposals
    -> project-owner confirmation
    -> canonical Tool 0.3.6 Responsibility Map
```

A legacy `TOOLCHAIN` responsibility may become `PRODUCT`, `DEVELOPMENT_TOOLING`, `DELIVERY`, `OPERATIONS`, a component split, or unresolved clarification. Blind `TOOLCHAIN -> DEVELOPMENT_TOOLING` translation is prohibited.

## VPMS boundary

PTSIP and VPMS remain sibling concerns:

```text
PTSIP
    Who owns this project responsibility across its lifecycle?

VPMS
    Why does this verification exist / what does it protect?
```

A PTSIP `DEVELOPMENT_TOOLING` verifier may still carry VPMS Verification Purpose `PRODUCT`. PTSIP classification must not be inferred from VPMS purpose and VPMS purpose must not be renamed merely because PTSIP classification changed.

## Verification and publication evidence for published Tool 0.3.5

The final maintainer-local Tool `0.3.5` repository regression after self-profile correction completed successfully:

```text
python -m pytest -q
244 passed in 571.90s
```

Earlier Tool `0.3.5` package-boundary verification also established successful wheel/sdist build, `twine check`, installed-wheel smoke, VPMS package inclusion, Tool identity `0.3.5`, and Specification identity `0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e`.

Tool `0.3.5` release preparation and publication are historical evidence only and must not be represented as Tool `0.3.6` verification.

## Current workflow resource policy

Repository verification, release preparation, and distribution build compute default to the approved self-hosted Windows runner:

```text
DESKTOP-5HCCQIR
```

`tooling-test.yml` and `release.yml` are manually dispatched and require explicit host readiness. Release-candidate verification is bound to the exact candidate SHA.

`tooling-release.yml` performs its build/distribution verification on the self-hosted Windows runner. The narrow GitHub-hosted GNU/Linux PyPI Trusted Publishing job is the only current hosted-compute exception because the publishing action is Docker-based; it must not absorb tests, compilation, release preparation, or package building.

No Tool `0.3.6` release verification or publication is claimed until the required self-hosted exact-SHA gates have actually completed.

## Tool lineage

- Tool `0.3.0`: published
- Tool `0.3.1`: published
- Tool `0.3.2`: source-only migration
- Tool `0.3.3`: permanently source-only
- Tool `0.3.4`: published historical Tool release
- Tool `0.3.5`: **published; first VPMS-capable Tool release**
- Tool `0.3.6`: **active development; Responsibility Map v2 canonical activation complete through WU-03**

Current next-version work remains consolidated under `planning/0.3.6.md`. Do not create a separate Tool `0.4.0` plan unless an independently consequential scope emerges beyond the current `0.3.6` contract.
