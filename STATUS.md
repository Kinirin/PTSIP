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

Tool `0.3.5` remains the published compatibility baseline. Its historical PTSIP semantics must not be rewritten by Tool `0.3.6` development.

## Active Tool 0.3.6 development

Development branch:

```text
tool-0.3.6-lifecycle-ownership
```

Development Tool identity is `0.3.6`; Tool `0.3.6` is **not published**.

Current bound Specification:

```text
0.3.6-draft @ 82abd09360df09a95fbbfb516855fa9ffb49f050
```

The WU-04C snapshot added normative declaration-authority/materialization semantics (`PTSIP-RMAP-013` through `PTSIP-RMAP-016`). WU-04D is implementing those frozen semantics; no new Specification revision is selected merely for runtime/test changes.

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
WU-04D  ResolvedProfile + digest + provenance            ACTIVE
WU-04E  validation consumes effective map                LOCKED
WU-04F  conformance consumes effective map               LOCKED
WU-04G  clarification/adoption read integration          LOCKED
WU-04H  VPMS narrow read-only integration                LOCKED
WU-04I  regression + WU-04 completion                    LOCKED
```

WU-04D exact entry baseline:

```text
d8713ac4e684852f3e6cf67a68165f82ae0b80aa
```

Active stage document:

```text
planning/0.3.6/WU-04D-resolved-profile-digest-provenance.md
```

WU-04E and later sub-documents must not be created until WU-04D is completed and the next stage is explicitly entered after a fresh branch-HEAD read.

## Tool 0.3.6 canonical lifecycle model

Canonical classifications are exactly:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

`TOOLCHAIN` is not a Tool `0.3.6` PTSIP classification alias. It remains historical/migration input for the planned legacy reader.

Classification is primary lifecycle ownership and remains distinct from component roles, typed relationships, associated artifacts, declaration source, materialization provenance, and VPMS Verification Purpose.

## WU-04 declaration/materialization authority

Frozen responsibility boundary:

```text
classification
    = primary lifecycle ownership

source_mode / derived entity origin
    = declaration authority provenance

materializer
    = deterministic non-authoritative resolution
```

Responsibility Map source modes:

```text
explicit
template
hybrid
```

Template selection is explicit and exact-revision-bound. Hybrid project replacement/extension/removal outranks the selected immutable template declaration, using stable-ID whole-entity authority rather than implicit field-level inheritance.

Derived runtime/review origin vocabulary:

```text
PROJECT_EXPLICIT
TEMPLATE
PROJECT_OVERRIDE
PROJECT_EXTENSION
PROJECT_REMOVAL
```

The materializer must not infer missing architecture, auto-select templates, rewrite lifecycle classifications, fabricate responsibilities, silently repair dangling relationships, cascade project removals, or mutate source declarations merely to produce a valid effective map.

## WU-04D active implementation boundary

WU-04D introduces one source-preserving resolved view:

```text
source declaration
      |
      v
deterministic materialization
      |
      v
ResolvedProfile
      +--> source_payload
      +--> effective_payload
      +--> source_mode + exact template identity
      +--> effective_map_digest
      +--> derived entity/removal provenance
```

Current implementation work is centered in:

```text
src/ptsip/validation/templates.py
```

Focused regression contracts are in:

```text
tests/ptsip/test_template_materialization_036.py
```

The effective-map digest represents effective architecture semantics rather than source mechanism. It excludes Specification binding, source mode, template identity, and materialization provenance, while normalizing stable-ID collections and known set-valued fields deterministically.

WU-04D remains **ACTIVE**. Validation, conformance, clarification/adoption, and VPMS consumer integration have not been entered and belong to WU-04E through WU-04H.

## Repository self-profile

Root `ptsip.yaml` structurally self-adopts Responsibility Map v2:

- release automation: `DELIVERY`;
- repository verification/CI/maintenance: `DEVELOPMENT_TOOLING`;
- canonical lifecycle-independent contracts: `NEUTRAL_CONTRACT` where applicable;
- subordinate specification governance support: associated-artifact semantics where independent component ownership is unnecessary.

## Tool 0.3.5 compatibility direction

Tool `0.3.6` compatibility means assisted migration, not preservation of obsolete ontology:

```text
Tool 0.3.5 profile
    -> legacy reader
    -> repository evidence
    -> lifecycle/role/relationship/split/artifact proposals
    -> project-owner confirmation
    -> canonical Tool 0.3.6 Responsibility Map
```

Blind `TOOLCHAIN -> DEVELOPMENT_TOOLING` translation is prohibited.

## VPMS boundary

PTSIP and VPMS remain independent concerns. Current VPMS Verification Purpose vocabulary remains:

```text
PRODUCT
TOOLCHAIN
```

VPMS `TOOLCHAIN` is VPMS vocabulary, not a Tool `0.3.6` PTSIP classification. PTSIP core must not depend on VPMS. VPMS effective-map read integration is reserved for WU-04H.

## Tool 0.3.6 verification status

Focused WU-04D regression cases have been added, but **no test execution success is claimed yet**.

No self-hosted workflow has been dispatched for WU-04D. Full repository regression/package verification remains a later WU-12/release-candidate responsibility unless a maintainer explicitly requests earlier self-hosted verification.

Historical Tool `0.3.5` maintainer-local regression remains:

```text
python -m pytest -q
244 passed in 571.90s
```

That is historical Tool `0.3.5` evidence only and must not be represented as Tool `0.3.6` verification.

## Workflow resource policy

Approved Windows self-hosted runner:

```text
DESKTOP-5HCCQIR
```

Before dispatching `tooling-test.yml` or `release.yml`, the maintainer must be told that this runner will be used and must explicitly confirm that the host and PowerShell environment are ready.

The narrow GNU/Linux PyPI Trusted Publishing job remains the only approved GitHub-hosted compute exception. It must not absorb tests, compilation, package building, or release preparation.

## Tool lineage

- Tool `0.3.0`: published
- Tool `0.3.1`: published
- Tool `0.3.2`: source-only migration
- Tool `0.3.3`: permanently source-only
- Tool `0.3.4`: published historical Tool release
- Tool `0.3.5`: **published; first VPMS-capable Tool release**
- Tool `0.3.6`: **active development; WU-04D ACTIVE**

Current next-version work remains consolidated under `planning/0.3.6.md`.
