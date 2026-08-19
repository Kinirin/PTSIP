# AGENTS.md

These instructions apply to coding agents working anywhere in this repository.

## Required context before work

Read, in order:

1. `MEMORY.md`
2. `ptsip.yaml`
3. `src/ptsip/constants.py`
4. applicable Specification files under `spec/`
5. `planning/0.3.6.md`
6. the WU-04 sub-stage document(s) identified by the master plan for the current implementation/verification tranche

`MEMORY.md` and planning documents are operational context. Normative claims come from the applicable bound Specification and canonical machine-readable contracts.

## Repository-state discipline

- Re-read the remote target branch HEAD immediately before **every GitHub write**, merge, release preparation, or evidence claim.
- Preserve maintainer commits and never force-update `main`.
- Do not claim tests, builds, releases, tags, or publication succeeded without evidence for the exact relevant SHA.
- Do not create future WU-04 sub-stage documents before their entry gate is reached.
- Do not finalize the Tool release note before the explicit release boundary.

## Tool 0.3.6 lifecycle reasoning

PTSIP classification answers:

```text
Who primarily owns this project responsibility across its lifecycle?
```

Canonical Tool `0.3.6` classifications are exactly:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

`TOOLCHAIN` is historical Tool `0.3.5` migration input only and MUST NOT be emitted or preserved as a canonical Tool `0.3.6` alias.

Classification is determined from governing lifecycle obligation, not file type, path, framework, language, executable status, workflow provider, compilation behavior, test status, majority of files/jobs/steps, runtime duration, invocation frequency, or confidence score.

Important boundaries:

- Product-owned tests may be `PRODUCT`.
- Reusable verification/test SDK/framework/harness infrastructure may be `DEVELOPMENT_TOOLING`.
- Development/intermediate build support is normally `DEVELOPMENT_TOOLING`.
- Authoritative release-unit assembly/signing/packaging/publication/deployment-to-destination is normally `DELIVERY`.
- `DELIVERY` ends at delivery handoff; ongoing health/recovery/reconciliation/maintenance is `OPERATIONS`.
- `NEUTRAL_CONTRACT` requires non-executable, non-owning, lifecycle-independent contract responsibility.
- Material mixed-lifecycle responsibilities should split when independently governable; do not choose a majority lifecycle.
- Material unresolved ownership fails closed.

## Responsibility Map axes

Keep these distinct:

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

Direction is always `source --TYPE--> target`.
Observed evidence, project-declared relationship, and dependency policy remain separate. Evidence `TESTS` may support `VERIFIES` but must not silently create a project declaration.

## Associated artifacts

An associated artifact is a project-owned non-component support surface subordinate to exactly one classified anchor component.

Rules:

- stable map identity;
- explicit selectors and purpose;
- exactly one anchor component;
- no classification or component roles of its own;
- no anchor-classification inheritance;
- non-executable architectural role;
- no independently governable lifecycle/release/compatibility responsibility;
- at least one typed relationship connecting it to its anchor;
- component IDs and associated-artifact IDs share one map-wide endpoint namespace.

Promote/re-evaluate as a component when independent lifecycle responsibility emerges. Independently governed non-executable/non-owning lifecycle-independent contract semantics require `NEUTRAL_CONTRACT` evaluation.

## Declaration authority and materialization

Responsibility Map source modes are:

```text
explicit
template
hybrid
```

ADR-0009 and `PTSIP-RMAP-013` through `PTSIP-RMAP-016` freeze the authority boundary:

```text
classification
    = lifecycle responsibility

source_mode / derived entity origin
    = declaration authority provenance

materializer
    = deterministic non-authoritative resolution
```

Authority rules:

- `explicit`: project owns the complete declaration;
- `template`: project explicitly adopts exact template `id + immutable revision`; that revision supplies adopted declaration content;
- `hybrid`: project owns exact template selection plus stable-ID whole-entity replacement, extension, and removal decisions;
- project replacement/extension/removal outranks selected immutable template declaration;
- template selection MUST NOT be inferred from repository evidence, language, framework, manifest, package manager, path, or confidence.

Derived runtime/review origin vocabulary:

```text
PROJECT_EXPLICIT
TEMPLATE
PROJECT_OVERRIDE
PROJECT_EXTENSION
PROJECT_REMOVAL
```

This provenance is not lifecycle classification and is not required canonical profile serialization.
The materializer MUST NOT infer architecture, change classifications, fabricate responsibilities, silently repair dangling relationships, cascade removals, resolve semantic conflicts by heuristics, or mutate source declarations merely to obtain a valid map.

## Current WU-04 stage

Current ordered state:

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

Verified E/F documents:

```text
planning/0.3.6/WU-04E-validation-effective-map.md
planning/0.3.6/WU-04F-conformance-effective-map.md
```

Combined exact-SHA evidence:

```text
workflow run  32240740753
job           96030499443
source SHA    48b75e699a592703e4e03a8462131e4932103677
Python        3.14.3
pytest        260 passed / 13 failed
```

Both focused contracts passed at that SHA because neither `test_profile_validation_036.py` nor `test_conformance_effective_map_036.py` appears in the exhaustive 13-failure summary. The remaining failures are later-stage debt in clarification/decision, pilot/evidence, topology/legacy migration, plus one stale lifecycle-evidence expectation. The repository-wide workflow still failed, so do not claim global Tool regression success or a `self-hosted/tooling-test` success status for that SHA.

WU-04G is **not entered yet**. Do not create its sub-document or modify clarification/adoption consumers until a later session starts with a fresh branch-HEAD read and the master plan records the WU-04G entry baseline.

WU-04E validation owns source validation -> materialization -> common effective semantic/selector validation and retains `ValidationResult.resolved_profile` as an in-process handoff.
WU-04F conformance consumes `validation.resolved_profile.effective_payload`; conformance must not reload raw YAML architecture, implement template/hybrid resolution, or restore a materialization-required branch.

## Migration boundary

Tool `0.3.5` compatibility means understand and migrate, not retain obsolete ontology in canonical Tool `0.3.6` state.
Migration is preview-first, evidence-backed, loss-preserving, and project-owner-confirmed. Blind `TOOLCHAIN -> DEVELOPMENT_TOOLING` mapping is prohibited.
Legacy `lifecycle_owner`, `consumers`, `analysis_inputs`, old boundary roots, and untyped dependency-policy entries are migration evidence, not automatic canonical relationships or ownership facts.
If confirmed architecture cannot be represented losslessly, stop and report the conflict.

## PTSIP / VPMS boundary

PTSIP lifecycle classification and VPMS Verification Purpose are independent.
Current VPMS Verification Purpose remains:

```text
PRODUCT
TOOLCHAIN
```

VPMS `TOOLCHAIN` is VPMS vocabulary, not Tool `0.3.6` PTSIP classification. Do not rename VPMS vocabulary as an accidental ontology migration.
PTSIP core MUST NOT depend on VPMS. VPMS may consume a narrow read-only effective PTSIP view only after WU-04H is entered. VPMS PASS does not imply PTSIP CONFORMANT and vice versa.

## Specification binding

For Tool `X.Y.Z`:

```text
Tool X.Y.Z
    -> Specification X.Y.Z-draft
    -> immutable SPEC_REVISION
```

Current Tool `0.3.6` development binding:

```text
SPEC_VERSION  = 0.3.6-draft
SPEC_REVISION = 82abd09360df09a95fbbfb516855fa9ffb49f050
```

WU-04E/F consume semantics already frozen by WU-04C and materialized by WU-04D. Do not move `SPEC_REVISION` merely because implementation/tests change. If a genuinely new normative rule is required, freeze the Specification first and then choose a new immutable snapshot.
Do not weaken `.github/scripts/verify_release_contract.py` to make a release pass.

## Exact release gate

At Tool release boundary:

```text
merged main + reviewed releasenote/X.Y.Z.md
    -> dispatch tooling-test.yml from main
    -> workflow pins checkout to github.sha
    -> self-hosted Windows + Python 3.14 full regression/build/smoke
    -> self-hosted/tooling-test status on that exact SHA
    -> dispatch release.yml from main
    -> release.yml derives github.sha + package version automatically
    -> require current origin/main == dispatched SHA
    -> require self-hosted/tooling-test success on that SHA
    -> verify Tool / Specification / SPEC_REVISION release contract
    -> create draft release targeting the same SHA without mutating main
    -> publish reviewed draft
    -> tooling-release.yml self-hosted distribution build
    -> minimal GitHub-hosted GNU/Linux PyPI Trusted Publishing only
```

No manual `source_sha`, `version`, `release_candidate`, or `host_ready` workflow inputs are part of this pipeline.
No repository mutation may occur between successful exact-SHA tooling verification and draft release creation.

## Self-hosted workflow policy

Self-hosted verification is capability-bound, not machine-name-bound.
Eligible build/test runners must satisfy:

```text
self-hosted
Windows
X64
PowerShell
Python 3.14 available through `py -3.14`
```

Do not hard-code a Windows computer name such as `DESKTOP-*` into workflow logic or operational instructions. If a matching runner is offline, GitHub Actions may remain queued until one becomes available; do not add a `host_ready` checkbox merely to duplicate that scheduler state.

`tooling-test.yml` and `release.yml` are manual workflows. The maintainer selects the target branch/ref in GitHub Actions; the workflow derives the immutable execution SHA from `github.sha` and validates it after checkout.

`tooling-release.yml` build/distribution verification also runs on self-hosted Windows and uses the host-provided Python 3.14 interpreter through an isolated per-run virtual environment.

The narrow GNU/Linux PyPI Trusted Publishing job is the only current GitHub-hosted compute exception; do not move tests, compilation, package building, or release preparation into it.
