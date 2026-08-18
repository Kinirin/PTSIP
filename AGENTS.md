# AGENTS.md

These instructions apply to coding agents working anywhere in this repository.

## Required context before work

Read, in order:

1. `MEMORY.md`
2. `ptsip.yaml`
3. `src/ptsip/constants.py`
4. applicable Specification files under `spec/`
5. `planning/0.3.6.md`
6. only the currently ACTIVE sub-stage document under `planning/0.3.6/`

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

Classification is determined from governing lifecycle obligation, not file type, path, framework, language, executable status, workflow provider, compilation behavior, test status, majority of files/jobs/steps, runtime duration, or confidence score.

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

Roles are optional, multi-valued, non-classifying, and must not be replaced by composite tokens.

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

## Active WU-04 stage

Current ordered state:

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

Active stage document:

```text
planning/0.3.6/WU-04D-resolved-profile-digest-provenance.md
```

WU-04D entry baseline:

```text
d8713ac4e684852f3e6cf67a68165f82ae0b80aa
```

WU-04D scope is limited to:

- source-preserving `ResolvedProfile`;
- deterministic Canonical Effective Responsibility Map digest;
- derived entity/removal provenance;
- focused materialization tests.

Do **not** migrate validation, conformance, clarification/adoption, or VPMS consumers to the resolved view during WU-04D. Those remain WU-04E through WU-04H.

Future sub-stage documents are created only after predecessor completion and a fresh branch-HEAD read. WU-04E and later sub-documents must not be created early.

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

WU-04D implements semantics already permitted/frozen by WU-04C. Do not move `SPEC_REVISION` merely because implementation/tests change. If a genuinely new normative rule is required, freeze the Specification first and then choose a new immutable snapshot.

Do not weaken `.github/scripts/verify_release_contract.py` to make a release pass.

## Exact release gate

At Tool release boundary:

```text
merged main + reviewed releasenote/X.Y.Z.md
    -> exact immutable main SHA
    -> Tool / Specification / SPEC_REVISION contract
    -> tooling-test.yml
         host_ready=true
         release_candidate=true
         source_sha=<exact SHA>
    -> self-hosted/release-verification on exact SHA
    -> release.yml with same SHA
    -> draft release without main mutation
    -> publish reviewed draft
    -> tooling-release.yml self-hosted build
    -> minimal GitHub-hosted GNU/Linux PyPI Trusted Publishing only
```

No repository mutation may occur between successful exact-SHA release verification and draft release creation.

## Self-hosted workflow policy

Approved Windows self-hosted runner:

```text
DESKTOP-5HCCQIR
```

Before dispatching `tooling-test.yml` or `release.yml`:

1. tell the user `DESKTOP-5HCCQIR` will be used;
2. wait for explicit confirmation that the host and PowerShell environment are ready;
3. only then dispatch with `host_ready=true`.

Do not automatically dispatch self-hosted workflows merely because a test would be useful.

`tooling-release.yml` build/distribution verification also runs on the approved Windows runner. The narrow GNU/Linux PyPI Trusted Publishing job is the only current GitHub-hosted compute exception; do not move tests, compilation, package building, or release preparation into it.
