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

The WU-04C snapshot added normative declaration-authority/materialization semantics (`PTSIP-RMAP-013` through `PTSIP-RMAP-016`). WU-04D implemented those frozen semantics and is complete. WU-04E consumes the resulting effective map in profile validation; runtime/test changes do not independently move the Specification revision.

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
WU-04D  ResolvedProfile + digest + provenance            COMPLETE
WU-04E  validation consumes effective map                ACTIVE
WU-04F  conformance consumes effective map               LOCKED
WU-04G  clarification/adoption read integration          LOCKED
WU-04H  VPMS narrow read-only integration                LOCKED
WU-04I  regression + WU-04 completion                    LOCKED
```

WU-04E exact entry baseline:

```text
2f0c7db2d20fb6d88ab5c4ab10707f50d486351f
```

Active stage document:

```text
planning/0.3.6/WU-04E-validation-effective-map.md
```

WU-04F and later sub-documents must not be created until WU-04E is completed and the next stage is explicitly entered after a fresh branch-HEAD read.

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

## WU-04E active implementation boundary

WU-04E validation pipeline is:

```text
source ptsip.yaml
    -> canonical source schema
    -> Tool / Specification binding
    -> source declaration mechanics
    -> materialize_profile()
    -> ResolvedProfile.effective_payload
    -> common semantic Responsibility Map validation
    -> common component/artifact selector partition + coverage
    -> resolution identity/provenance details
```

Current implementation changes are centered in:

```text
src/ptsip/validation/profile.py
```

Focused regression contracts are in:

```text
tests/ptsip/test_profile_validation_036.py
```

The implementation now:

- materializes explicit/template/hybrid profiles only after source schema and binding checks;
- reports `TemplateMaterializationError` as fail-closed profile validation errors;
- validates the materialized explicit-form effective payload with the canonical schema and common semantic rules;
- applies component/associated-artifact partition and coverage to the effective map for all source modes;
- exposes `resolution` identity and `resolution_provenance` in `ValidationResult.details`;
- removes the prior template/hybrid path that only emitted a “materialization layer required” warning;
- adds focused validation tests for template, hybrid override/removal, unknown revision, dangling effective endpoint, and explicit/template equivalence.

WU-04E remains **ACTIVE**. These changes have not yet been verified by an exact-SHA self-hosted run. Conformance, clarification/adoption, and VPMS consumer integration remain WU-04F through WU-04H.

## Tool 0.3.6 verification status

Latest available full self-hosted regression evidence before WU-04E implementation:

```text
run 32218181598
job 95963505443
source SHA bf62202507ee7b83d72cefb5cf01243675ffd062
result 219 passed / 43 failed
```

Infrastructure status in that run:

```text
self-hosted Windows scheduling       PASS
exact SHA checkout                   PASS
exact source identity                PASS
host Python 3.14 selection           PASS
isolated per-run .venv creation      PASS
verification tooling installation    PASS
complete pytest execution            RAN
```

The previous `actions/setup-python` installation failure is resolved. The workflow uses host Python 3.14 through `py -3.14` and an isolated per-run virtual environment.

After that run, stale current-Spec and VPMS workflow-test expectations were aligned, WU-04D was closed, and WU-04E implementation began. Therefore the current branch state does **not** yet have an exact-SHA test result.

Do not claim full Tool `0.3.6` regression success and do not mark WU-04E complete until the new validation behavior has been verified.

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

## GitHub Actions resource policy

Self-hosted execution is **capability-bound, not machine-name-bound**.

Eligible Tool build/test runners must satisfy:

```text
self-hosted
Windows
X64
PowerShell
Python 3.14 available through `py -3.14`
```

Do not hard-code `DESKTOP-*` computer names into workflows or operational instructions.
If a matching runner is offline, GitHub Actions may remain queued until one becomes available; no `host_ready` checkbox is required.

Current manual verification flow:

```text
tooling-test.yml
    workflow_dispatch with no custom inputs
    -> selected ref's exact github.sha
    -> self-hosted regression/build/smoke
    -> self-hosted/tooling-test status on exact SHA

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

The narrow GNU/Linux PyPI Trusted Publishing job remains the only approved GitHub-hosted compute exception. It must not absorb tests, compilation, package building, or release preparation.

## Tool lineage

- Tool `0.3.0`: published
- Tool `0.3.1`: published
- Tool `0.3.2`: source-only migration
- Tool `0.3.3`: permanently source-only
- Tool `0.3.4`: published historical Tool release
- Tool `0.3.5`: **published; first VPMS-capable Tool release**
- Tool `0.3.6`: **active development; WU-04E ACTIVE**

Current next-version work remains consolidated under `planning/0.3.6.md`.
