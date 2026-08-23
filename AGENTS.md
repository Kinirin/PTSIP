# AGENTS.md

These instructions apply to coding agents working anywhere in this repository.

## Required context before work

Read, in order:

1. `MEMORY.md`
2. `ptsip.yaml`
3. `src/ptsip/constants.py`
4. applicable Specification files under `spec/`
5. `planning/0.3.6.md`
6. `planning/0.3.6/WU-07-final-specification-freeze-release-preparation.md` while Tool `0.3.6` release closure is active

`MEMORY.md` and planning documents are operational context. Normative claims come from the applicable bound Specification and canonical machine-readable contracts.

## Repository-state discipline

- Re-read the remote target branch HEAD immediately before **every GitHub write**, merge, release preparation, or evidence claim.
- Preserve maintainer commits and never force-update `main`.
- Do not claim tests, builds, releases, tags, or publication succeeded without evidence for the exact relevant SHA.
- Enter or create future work-unit implementation only after its predecessor completion and exact entry gate are established.
- Tool release notes may be curated during the explicit release-preparation stage, but MUST NOT be described as published until the exact release boundary succeeds.

## Current Tool 0.3.6 stage

Current ordered state:

```text
WU-00  0.3.6-draft normative baseline                    COMPLETE
WU-01  lifecycle ontology/boundary rules                  COMPLETE
WU-02  roles + typed relationships + associated artifacts COMPLETE
WU-03  canonical Responsibility Map v2 activation         COMPLETE
WU-04  template/materialization/effective-map pipeline    COMPLETE / EXACT-SHA VERIFIED
WU-05  repository dogfood / self-evaluation               COMPLETE / DOGFOOD REVIEWED
WU-06  full regression / package/distribution verification COMPLETE / EXACT-SHA VERIFIED
WU-07  final Specification freeze / release preparation   ACTIVE
```

Current WU-07 authority:

```text
planning/0.3.6/WU-07-final-specification-freeze-release-preparation.md
strategy B — Release Contract Strengthening
entry baseline 8b2c0819e10b58902a780a094a0f52c603c39fba
```

WU-04G/H/I implementation is historical completed context. Do not treat their former entry baselines, active-state instructions, or predecessor verification failures as current release authority.

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

ADR-0009 and `PTSIP-RMAP-013` through `PTSIP-RMAP-016` freeze the base authority boundary. `PTSIP-RMAP-017` freezes accepted-decision safe-apply authority.

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
- template selection MUST NOT be inferred from repository evidence, language, framework, manifest, package manager, path, or confidence;
- an accepted project-owned clarification/adoption decision may authorize only the minimum project declaration delta needed to represent that accepted decision;
- a `template -> hybrid` transition is allowed only when such an accepted decision requires it, while retaining the exact selected template ID/revision.

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

## Clarification / adoption authority

Clarification and adoption consume a valid `ResolvedProfile.effective_payload`. Invalid/unresolvable profile state fails closed and exposes deterministic remediation/retry information without falling back to raw-profile architecture authority.

New canonical answers use `ptsip-clarification-answer/v2` with:

```text
classification
purpose
shipped
runtime_required
executable
```

`lifecycle_owner` is not part of new canonical Tool `0.3.6` decisions or Project Profile serialization. Historical v1 data may be handled only through explicit compatibility logic and cannot restore `TOOLCHAIN` as canonical authority.

Exact repository-relative selected profile path identity must survive gate creation, decision storage, resolution context, subject revision, remote file read, stale check, projection validation, and CAS write. Never silently substitute root `ptsip.yaml` for a selected non-root profile.

## Migration boundary

Tool `0.3.5` compatibility means understand and migrate, not retain obsolete ontology in canonical Tool `0.3.6` state.
Migration is preview-first, evidence-backed, loss-preserving, and project-owner-confirmed. Blind `TOOLCHAIN -> DEVELOPMENT_TOOLING` mapping is prohibited.
Legacy `lifecycle_owner`, `consumers`, `analysis_inputs`, old boundary roots, and untyped dependency-policy entries are migration evidence, not automatic canonical relationships or ownership facts.
If confirmed architecture cannot be represented losslessly, stop and report the conflict.

Evidence-driven Tool `0.3.5 -> 0.3.6` migration implementation belongs to Tool `0.3.6.1`; WU-07 MUST NOT enter that implementation work.

## PTSIP / VPMS boundary

PTSIP lifecycle classification and VPMS Verification Purpose are independent.
Current VPMS Verification Purpose remains:

```text
PRODUCT
TOOLCHAIN
```

VPMS `TOOLCHAIN` is VPMS vocabulary, not Tool `0.3.6` PTSIP classification. Do not rename VPMS vocabulary as an accidental ontology migration.
PTSIP core MUST NOT depend on VPMS. VPMS consumes only a narrow read-only projection of already-resolved effective PTSIP metadata and does not own PTSIP classification, template/hybrid semantics, or migration authority. VPMS PASS does not imply PTSIP CONFORMANT and vice versa.

## Specification binding

For Tool `X.Y.Z`:

```text
Tool X.Y.Z
    -> Specification X.Y.Z-draft
    -> immutable SPEC_REVISION
```

Current Tool `0.3.6` release-candidate binding is:

```text
SPEC_VERSION  = 0.3.6-draft
SPEC_REVISION = d6995ed232e845b88d8235b851e80ab54b7804ea
```

WU-07 final audit begins from this binding. Do not move `SPEC_REVISION` for release-note changes, workflow hardening, tests, planning/status updates, or operational-document cleanup. A new revision requires a separately established genuine normative defect.

`.github/scripts/verify_release_contract.py` MUST remain fail-closed. Release-bound normative source and machine-readable assets must match the exact bound revision; canonical/embedded machine-readable copies must remain coherent.

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
    -> publication Product Artifact evidence + exact tagged-snapshot binding
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
