# PTSIP Repository Working Memory

This file is durable repository-operational context for maintainers and coding agents. It is **not** a normative Specification and does not replace `ptsip.yaml`, `spec/`, schemas, registry data, ADRs, or the active plan.

## Published baseline

- Canonical repository: `Kinirin/PTSIP`
- Published Tool: `0.3.5`
- Tag: `tool-v0.3.5`
- Release commit: `79bc4c2daf695e8462a02f2a7c4b1bb1a88846e1`
- Published bound Specification: `0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e`
- Historical Tool `0.3.5` classifications: `PRODUCT | TOOLCHAIN | NEUTRAL_CONTRACT`

Do not rewrite Tool `0.3.5` history. Tool `0.3.6` compatibility is assisted migration, not preservation of obsolete ontology in canonical state.

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
planning/0.3.6/WU-04E-validation-effective-map.md
```

WU-04E exact entry baseline:

```text
2f0c7db2d20fb6d88ab5c4ab10707f50d486351f
```

Published PyPI Tool remains `0.3.5`; the development branch Tool identity is `0.3.6`.

## Specification snapshots

```text
WU-00 baseline:              654e41d49600fc091f9a6cb6b1c60bbc7da4e301
WU-03 canonical activation:  12e2ccd15634ecb3d0a4195b0f61ac3f620e7540
WU-04C authority snapshot:   82abd09360df09a95fbbfb516855fa9ffb49f050
```

Current Tool/root-profile binding:

```text
Specification 0.3.6-draft
SPEC_REVISION 82abd09360df09a95fbbfb516855fa9ffb49f050
```

WU-04C added normative `PTSIP-RMAP-013` through `PTSIP-RMAP-016`. WU-04D implemented those frozen materialization semantics and completed without a Specification revision move. WU-04E consumes that resolved view in validation and likewise does not independently justify moving `SPEC_REVISION`.

Decision records:

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

Derived origin vocabulary:

```text
PROJECT_EXPLICIT
TEMPLATE
PROJECT_OVERRIDE
PROJECT_EXTENSION
PROJECT_REMOVAL
```

The materializer MUST NOT infer architecture, auto-select templates, change classifications, fabricate responsibilities, silently repair dangling relationships, cascade removals, resolve conflicts by heuristics, or mutate source declarations merely to produce a valid effective map.

## WU-04 stage state

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

WU-04D completion record:

```text
planning/0.3.6/WU-04D-resolved-profile-digest-provenance.md
completion baseline 6b1aa963b11681242088527c5723ad5cd57b4ea1
```

WU-04E owns this validation pipeline:

```text
source ptsip.yaml
    -> source schema validation
    -> Tool/Specification binding validation
    -> source declaration mechanics
    -> materialize_profile()
    -> ResolvedProfile.effective_payload
    -> common semantic Responsibility Map validation
    -> common component/artifact selector partition + coverage
    -> resolution identity/provenance details
```

Primary implementation target:

```text
src/ptsip/validation/profile.py
```

Focused regression target:

```text
tests/ptsip/test_profile_validation_036.py
```

Current WU-04E implementation already:

- imports and calls `materialize_profile()` after source schema/binding checks;
- converts `TemplateMaterializationError` into fail-closed profile validation errors;
- validates effective explicit-form schema and Responsibility Map semantics for all source modes;
- applies component/associated-artifact partition and coverage to effective components/artifacts;
- exposes `resolution` identity and `resolution_provenance` in validation details;
- removes the old template/hybrid “materialization layer required” bypass path;
- adds focused template/hybrid/equivalence/fail-closed tests.

These WU-04E changes are **not yet self-hosted verified**. Do not claim WU-04E complete or enter WU-04F until focused validation behavior is verified and the WU-04E completion gate is explicitly closed.

## Latest available self-hosted regression evidence before WU-04E implementation

GitHub Actions run:

```text
run 32218181598
job 95963505443
source SHA bf62202507ee7b83d72cefb5cf01243675ffd062
```

Observed execution facts:

```text
self-hosted runner scheduling       PASS
exact SHA checkout                  PASS
exact source identity               PASS
host Python 3.14 preparation        PASS
isolated .venv creation             PASS
verification tooling installation   PASS
complete pytest execution           RAN
result                              219 passed / 43 failed
```

The earlier `actions/setup-python` infrastructure failure is resolved by using host Python through `py -3.14` and a per-run `.venv`.

The 43 failures included stale earlier-version expectations and later-stage effective-map consumer gaps. `tests/ptsip/test_template_materialization_036.py` did not appear in the failure summary. After that run, stale Spec-revision and VPMS nodeid test expectations were aligned with the current WU-04C snapshot/workflow contract, then WU-04D was closed and WU-04E implementation began.

Do not claim full Tool `0.3.6` regression success. The current WU-04E branch state has no exact-SHA self-hosted result yet.

## PTSIP / VPMS boundary

PTSIP classification and VPMS Verification Purpose remain independent.
Current VPMS purposes remain:

```text
PRODUCT
TOOLCHAIN
```

VPMS `TOOLCHAIN` is VPMS vocabulary, not Tool `0.3.6` PTSIP classification. PTSIP core MUST NOT depend on VPMS. VPMS effective-map read integration is reserved for WU-04H.

## GitHub Actions resource policy

Self-hosted workflows are capability-bound, not machine-name-bound.
Eligible build/test hosts must satisfy:

```text
self-hosted
Windows
X64
PowerShell
Python 3.14 available through `py -3.14`
```

Do not hard-code `DESKTOP-*` names into workflows or operational instructions.
Do not add `host_ready` inputs: an unavailable matching runner naturally leaves the Actions job queued.
Do not require maintainers to type exact SHAs into workflow inputs when the workflow can pin and verify `${{ github.sha }}` itself.

Current pipeline:

```text
tooling-test.yml
    manual workflow_dispatch; no custom inputs
    checkout exact github.sha
    host Python 3.14 + isolated .venv
    full regression/build/smoke
    -> self-hosted/tooling-test status on exact SHA

release.yml
    manual workflow_dispatch from main; no custom inputs
    derives github.sha and package version
    requires current origin/main == dispatched SHA
    requires self-hosted/tooling-test success on exact SHA
    verifies release contract
    creates draft release for same SHA

tooling-release.yml
    release: published
    self-hosted Windows distribution build/verification
    minimal ubuntu-latest Trusted Publishing only
```

The narrow GNU/Linux PyPI Trusted Publishing job is the only approved GitHub-hosted compute exception.

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

## Repository write discipline

Before repository changes, read `AGENTS.md`, this `MEMORY.md`, `ptsip.yaml`, `src/ptsip/constants.py`, applicable Specification, `planning/0.3.6.md`, and the current ACTIVE sub-stage document.
Re-read the remote target branch HEAD immediately before **every GitHub write**, merge, release preparation, or evidence claim.
