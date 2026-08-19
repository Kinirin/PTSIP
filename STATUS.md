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

The WU-04C snapshot added normative declaration-authority/materialization semantics (`PTSIP-RMAP-013` through `PTSIP-RMAP-016`). WU-04D implemented those frozen semantics. WU-04E/F consume them in validation/conformance and do not independently move the Specification revision.

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
WU-04E  validation consumes effective map                COMPLETE / EXACT-SHA VERIFIED
WU-04F  conformance consumes effective map               COMPLETE / EXACT-SHA VERIFIED
WU-04G  clarification/adoption read integration          NEXT / NOT ENTERED
WU-04H  VPMS narrow read-only integration                LOCKED
WU-04I  regression + WU-04 completion                    LOCKED
```

Verified E/F documents and entry baselines:

```text
WU-04E  planning/0.3.6/WU-04E-validation-effective-map.md
        entry 2f0c7db2d20fb6d88ab5c4ab10707f50d486351f

WU-04F  planning/0.3.6/WU-04F-conformance-effective-map.md
        entry 5d52e452ce48de4d0e0d8251d906c1e0f15f82c2
```

WU-04G is the next stage but has not been entered. Its stage document must not be created until a later session begins with a fresh branch-HEAD read.

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

## WU-04E/F verified implementation boundary

Verified runtime flow:

```text
source ptsip.yaml
    -> validate_profile()
        -> source schema + Tool/Specification binding
        -> source declaration mechanics
        -> materialize_profile()
        -> effective schema/semantic/selector validation
        -> ValidationResult.resolved_profile
    -> ResolvedProfile.effective_payload
        -> base conformance
        -> complete conformance engine
```

WU-04E:

- materializes explicit/template/hybrid only after source validation;
- converts materialization defects into fail-closed validation errors;
- validates effective Responsibility Map semantics and selector coverage for all modes;
- retains `resolved_profile` as an in-process downstream handoff while omitting it from serialized `as_dict()` output;
- has focused coverage in `tests/ptsip/test_profile_validation_036.py`, including dangling effective endpoint/anchor and dependency-policy conflicts.

WU-04F:

- removes raw YAML architecture reload from `src/ptsip/conformance.py` and `src/ptsip/conformance_engine.py`;
- consumes effective components and `component_dependency_policy` from `ValidationResult.resolved_profile.effective_payload`;
- removes the old `ownership:materialization-required` / `MATERIALIZED_COMPONENT_DECLARATIONS_REQUIRED` branch;
- applies effective components to dependency, artifact, project-policy, language, build, lifecycle, and agent-decision comparison paths;
- has focused explicit/template/hybrid consumption coverage in `tests/ptsip/test_conformance_effective_map_036.py`;
- updates current-tool conformance fixtures to canonical `0.3.6-draft`, current `SPEC_REVISION`, `DEVELOPMENT_TOOLING`, Responsibility Map v2, and current policy keys.

Repository search after implementation found no remaining raw `yaml.safe_load` or materialization-required branch in the two conformance integration targets.

## Verification status

Combined WU-04E/F self-hosted verification:

```text
run        32240740753
job        96030499443
source SHA 48b75e699a592703e4e03a8462131e4932103677
Python     3.14.3
result     260 passed / 13 failed
```

Infrastructure/execution status:

```text
self-hosted Windows scheduling       PASS
exact SHA checkout                   PASS
exact source identity                PASS
host Python 3.14 selection           PASS
isolated per-run .venv creation      PASS
verification tooling installation    PASS
complete pytest execution            RAN
```

The exhaustive failure summary did not contain either:

```text
tests/ptsip/test_profile_validation_036.py
tests/ptsip/test_conformance_effective_map_036.py
```

Therefore the focused WU-04E and WU-04F contracts passed at exact source SHA `48b75e699a592703e4e03a8462131e4932103677`, and both stages are COMPLETE / exact-SHA verified.

The 13 remaining failures were classified as:

```text
3  decision/clarification-control historical expectations
1  pilot/evidence fixture using pre-0.3.6 profile state
1  stale lifecycle-evidence expectation for an unscoped release workflow
1  agent-decision conflict expectation for later decision integration
7  topology/legacy-profile migration fixtures
```

The lifecycle-evidence failure is not an E/F regression. Current lifecycle evaluation explicitly treats workflow naming/trigger scope as evidence rather than lifecycle authority; an unscoped release-like workflow is observed but does not itself create a classification failure.

The workflow still concluded failure because of those 13 later-stage regressions. Consequently:

- no `self-hosted/tooling-test` success status was recorded for `48b75e6...`;
- Tool `0.3.6` is **not** globally regression-clean;
- release/build/smoke steps after pytest were skipped;
- this run is stage-completion evidence for WU-04E/F only, not release-readiness evidence.

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

Do not hard-code `DESKTOP-*` computer names into workflows or operational instructions. If a matching runner is offline, GitHub Actions may remain queued until one becomes available; no `host_ready` checkbox is required.

Current manual verification flow:

```text
tooling-test.yml
    workflow_dispatch with no custom inputs
    -> selected ref's exact github.sha
    -> self-hosted regression/build/smoke
    -> self-hosted/tooling-test status on exact SHA only after complete workflow success

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

The narrow GNU/Linux PyPI Trusted Publishing job remains the only approved GitHub-hosted compute exception.

## Tool lineage

- Tool `0.3.0`: published
- Tool `0.3.1`: published
- Tool `0.3.2`: source-only migration
- Tool `0.3.3`: permanently source-only
- Tool `0.3.4`: published historical Tool release
- Tool `0.3.5`: **published; first VPMS-capable Tool release**
- Tool `0.3.6`: **active development; WU-04E/F COMPLETE, WU-04G next/not entered**

Current next-version work remains consolidated under `planning/0.3.6.md`.
