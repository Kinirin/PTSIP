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

Current verification tranche:

```text
planning/0.3.6/WU-04E-validation-effective-map.md
planning/0.3.6/WU-04F-conformance-effective-map.md
```

Entry baselines:

```text
WU-04E  2f0c7db2d20fb6d88ab5c4ab10707f50d486351f
WU-04F  5d52e452ce48de4d0e0d8251d906c1e0f15f82c2
```

Both E and F implementation gates are complete. They intentionally share one exact-SHA self-hosted verification run. WU-04G remains locked until that run is reviewed.

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

WU-04C froze `PTSIP-RMAP-013` through `PTSIP-RMAP-016`. WU-04D implemented the frozen materialization semantics. WU-04E/F consume those semantics in validation and conformance and do not independently justify moving `SPEC_REVISION`.

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

## WU-04 authority and runtime pipeline

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

Current E/F pipeline:

```text
source ptsip.yaml
    -> validate_profile()
        -> source schema + Tool/Specification binding
        -> materialize_profile()
        -> effective semantic/selector validation
        -> ValidationResult.resolved_profile
    -> ResolvedProfile.effective_payload
        -> base conformance
        -> complete conformance engine
```

`ValidationResult.resolved_profile` is an in-process handoff and is deliberately omitted from `ValidationResult.as_dict()`. Public reports expose serializable resolution identity/provenance details rather than duplicating the full effective payload.

Conformance must not reload raw YAML architecture, implement template/hybrid semantics, or restore a `materialization-required` branch.

## WU-04 stage state

```text
WU-04A  template catalog identity                         COMPLETE
WU-04B  deterministic materializer core                  COMPLETE
WU-04C  declaration authority/source_mode boundary       COMPLETE
WU-04D  ResolvedProfile + digest + provenance            COMPLETE
WU-04E  validation consumes effective map                IMPLEMENTATION COMPLETE / SHARED VERIFICATION PENDING
WU-04F  conformance consumes effective map               IMPLEMENTATION COMPLETE / SHARED VERIFICATION PENDING
WU-04G  clarification/adoption read integration          LOCKED
WU-04H  VPMS narrow read-only integration                LOCKED
WU-04I  regression + WU-04 completion                    LOCKED
```

Focused E/F regression targets:

```text
tests/ptsip/test_profile_validation_036.py
tests/ptsip/test_conformance_effective_map_036.py
```

WU-04F also aligned current-tool conformance fixtures in:

```text
tests/ptsip/test_profile_correctness_023.py
tests/ptsip/test_conformance_030.py
tests/ptsip/test_conformance_engine_030.py
tests/ptsip/test_merge_gate_remediation_030.py
tests/ptsip/test_remaining_030.py
```

Historical/intentionally conflicting input remains historical where the test purpose requires it; e.g. an agent decision may still claim `TOOLCHAIN` to verify that it cannot override canonical project declaration.

## Latest self-hosted regression evidence

GitHub Actions run:

```text
run 32227306170
job 95989558038
source SHA 33e26cebd845970c6e52fb0a9194a272f0e50228
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
result                              230 passed / 37 failed
```

At that SHA, the initial five tests in `tests/ptsip/test_profile_validation_036.py` did not appear in the failure summary. That run therefore provided useful WU-04E evidence, but it predates the final E completion cases, the resolved-profile handoff, all WU-04F runtime integration, F focused tests, and canonicalized conformance fixtures.

Do not claim the final E/F tranche is verified yet. The next self-hosted `tooling-test.yml` run must target the exact current branch SHA after all E/F implementation and operational-context commits.

If that shared run exposes an E regression, reopen E. If it exposes an F regression, reopen/continue F. Failures belonging to clarification/adoption, pilot, topology, VPMS, or legacy migration must be classified against their owning future stage rather than weakening E/F contracts. Do not enter WU-04G before this review.

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

Before repository changes, read `AGENTS.md`, this `MEMORY.md`, `ptsip.yaml`, `src/ptsip/constants.py`, applicable Specification, `planning/0.3.6.md`, and the sub-stage document(s) named by the current verification/implementation tranche.
Re-read the remote target branch HEAD immediately before **every GitHub write**, merge, release preparation, or evidence claim.
