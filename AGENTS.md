# AGENTS.md

These instructions apply to coding agents working anywhere in this repository.

## Required context before work

Read these files before planning or modifying code:

1. `MEMORY.md`
2. `ptsip.yaml`
3. `src/ptsip/constants.py`
4. applicable Specification files under `spec/`
5. `planning/0.3.6.md`
6. the sub-document for the currently ACTIVE sub-stage, if one exists

For Responsibility Map work also read:

- `spec/PTSIP-RESPONSIBILITY-MAP.md`;
- `decisions/ADR-0007-primary-lifecycle-boundary-determination.md`;
- `decisions/ADR-0008-responsibility-roles-relationships-associated-artifacts.md`;
- `decisions/ADR-0009-responsibility-map-declaration-authority.md` when template/materialization authority is relevant.

`MEMORY.md`, `AGENTS.md`, and planning documents are operational guidance. Normative claims come from the applicable Specification revision and project-owned machine-readable contracts.

The published Tool remains `0.3.5` until an explicit release boundary. Development on `tool-0.3.6-lifecycle-ownership` uses Tool `0.3.6` and the bound `0.3.6-draft` Specification snapshot recorded in constants/root profile.

## Repository-state discipline

- Re-read the remote target branch HEAD immediately before **every GitHub write**, merge, release preparation, or evidence claim.
- Preserve maintainer commits and never force-update `main`.
- Do not claim a test, build, release, tag, workflow, or publication succeeded unless evidence for the exact relevant SHA was observed.
- Do not create/finalize the Tool release note early; finalize `releasenote/X.Y.Z.md` at the explicit release boundary before exact release-candidate SHA verification.
- Published tags/version documents are immutable historical evidence.

## PTSIP lifecycle reasoning

Tool `0.3.6` classification answers:

```text
Which lifecycle primarily owns this coherent project responsibility?
```

Canonical classifications are exactly:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

Rules:

- classification is primary lifecycle ownership, not file type, path, framework, language, workflow provider, executable status, compilation behavior, activity verb, or test status;
- Product-owned tests may be `PRODUCT`;
- reusable verification/test-SDK infrastructure may be `DEVELOPMENT_TOOLING`;
- release/publication/deployment-to-destination responsibility may be `DELIVERY`;
- post-handoff production health/recovery/maintenance may be `OPERATIONS`;
- `NEUTRAL_CONTRACT` requires non-executable, non-owning, lifecycle-independent contract semantics;
- mixed independently governable lifecycle responsibilities require split/redesign or unresolved clarification rather than majority classification;
- technology names are evidence context, not classification authority.

`TOOLCHAIN` is canonical Tool `0.3.5` input vocabulary only. Tool `0.3.6` may understand it through the legacy migration path but must not emit it as a canonical classification or alias.

## Classification, roles, relationships, artifacts, and VPMS

Keep these axes separate:

```text
classification
    = primary lifecycle ownership

roles
    = coarse responsibility characteristics within a lifecycle

relationships
    = project-owned typed directed semantic edges

associated artifact
    = non-component support surface subordinate to one classified anchor

VPMS Verification Purpose
    = what a Verification Case protects/verifies
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

Do not manufacture composite role tokens. Multiple applicable characteristics are multiple role values.

Canonical Responsibility Map relationship direction is always:

```text
source --TYPE--> target
```

Canonical relationship types:

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

Do not invent generic escape-hatch relations such as `PRODUCES`, `SUPPORTS`, `USES`, or `DEPENDS_ON` where canonical semantics apply.

Evidence `TESTS` may support a project-owned `VERIFIES` proposal but must not silently become that declaration. Typed relationships do not grant permission or waive dependency/conformance rules.

Associated artifacts:

- have stable ID, selectors, purpose, and exactly one anchor component;
- have no classification or component roles;
- do not inherit anchor classification;
- must be non-executable in their architectural role and lack independently governable lifecycle/release/compatibility responsibility;
- require at least one typed relationship connecting them to the anchor;
- share one endpoint namespace with components;
- must be promoted/re-evaluated as components when independent responsibility emerges;
- must be evaluated for `NEUTRAL_CONTRACT` when independently governed non-executable/non-owning lifecycle-independent contract semantics emerge.

## Responsibility Map declaration authority

Tool `0.3.6` source modes are:

```text
explicit
template
hybrid
```

`source_mode` is **declaration-source/authority provenance**, not lifecycle ownership.

The responsibility boundaries frozen by ADR-0009 and `PTSIP-RMAP-013` through `PTSIP-RMAP-016` are:

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
- `template`: project owns explicit adoption of exact template `id + immutable revision`; selected revision supplies adopted declaration content;
- `hybrid`: project owns exact selection plus stable-ID replacements, extensions, and removals; unchanged template entities retain template origin;
- project replacement/extension/removal outranks selected template declaration;
- Tool `0.3.6` hybrid replacement is whole-entity by stable ID, not implicit field-level inheritance.

Derived runtime/review origin vocabulary may include:

```text
PROJECT_EXPLICIT
TEMPLATE
PROJECT_OVERRIDE
PROJECT_EXTENSION
PROJECT_REMOVAL
```

This provenance is not lifecycle classification and need not be persisted in canonical `ptsip.yaml`.

The materializer MUST NOT auto-select templates, infer missing architecture, change classification, fabricate responsibilities, silently repair/delete dangling relations, cascade project removals, resolve conflicts by confidence/path heuristics, or mutate source declarations merely to produce a valid map.

All three modes resolve to one Canonical Effective Responsibility Map for downstream consumers while retaining source declaration and template identity separately.

## VPMS boundary

PTSIP and VPMS remain independent.

Current VPMS Verification Purpose vocabulary remains:

```text
PRODUCT
TOOLCHAIN
```

VPMS `TOOLCHAIN` is VPMS vocabulary, not a Tool `0.3.6` PTSIP classification. Do not rename it merely because PTSIP lifecycle ontology changed.

PTSIP core must not acquire a VPMS dependency. VPMS may consume stable PTSIP metadata through a narrow read-only boundary after PTSIP resolves the effective map. VPMS must not implement template-selection/materialization authority itself.

VPMS PASS does not imply PTSIP CONFORMANT and PTSIP CONFORMANT does not imply VPMS PASS.

## Tool 0.3.6 staged development sequencing

Follow `planning/0.3.6.md` exactly.

Current release work-unit state:

```text
WU-00  normative 0.3.6 baseline                         COMPLETE
WU-01  lifecycle boundary rules                         COMPLETE
WU-02  roles/relationships/associated artifacts         COMPLETE
WU-03  canonical Responsibility Map v2 activation       COMPLETE
WU-04  template/materialization/effective-map pipeline  IN PROGRESS
```

Current WU-04 sub-stage state:

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

### Sub-document entry rule

Future WU-04 sub-documents MUST NOT be created in advance.

To enter the next stage:

```text
complete current stage gate
    -> fresh-read branch HEAD
    -> verify predecessor remains complete
    -> create only the next stage sub-document
    -> record exact entry baseline
    -> mark that stage ACTIVE
    -> perform only that stage scope
```

At present **WU-04D has not been entered and no WU-04D sub-document should exist**. Do not implement `ResolvedProfile`, digest/provenance integration, validation consumption, conformance consumption, clarification integration, or VPMS integration out of sequence.

WU-04C's completed record is:

```text
planning/0.3.6/WU-04C-declaration-authority-effective-map.md
```

## Migration boundary

Tool `0.3.5` compatibility means understand and migrate, not retain obsolete ontology in the canonical schema.

Migration must be preview-first, evidence-backed, and loss-preserving. Candidate discovery/migration analysis may propose lifecycle mappings, component splits, roles, relationships, associated artifacts, and templates but are not project architecture authority.

Never silently replace a project-owned map. If the target representation cannot preserve confirmed facts losslessly, report and stop.

Legacy `consumers`, `analysis_inputs`, `lifecycle_owner`, old boundaries, and untyped dependency-policy entries are migration evidence; do not blindly translate them into canonical relationships/ownership.

## Mandatory Specification release contract

For Tool `X.Y.Z`:

```text
SPEC_VERSION = X.Y.Z-draft
```

Release preparation requires an immutable `SPEC_REVISION`, matching root `ptsip.yaml` binding, required canonical Specification files at that revision, and `releasenote/spec-X.Y.Z-draft.md`.

Do not bypass or weaken `.github/scripts/verify_release_contract.py` to make a release pass. Fix missing/inconsistent Specification work.

The current Tool `0.3.6` development binding is recorded in `src/ptsip/constants.py` and root `ptsip.yaml`. Move `SPEC_REVISION` only when normative Specification assets change, not for ordinary implementation/test commits.

### Exact merge-to-release sequence

```text
merged main + reviewed releasenote/X.Y.Z.md
    -> exact immutable main SHA
    -> Tool/Specification/binding contract
    -> tooling-test.yml
         host_ready=true
         release_candidate=true
         source_sha=<exact SHA>
    -> self-hosted/release-verification on same SHA
    -> release.yml
         host_ready=true
         source_sha=<same exact SHA>
    -> draft release targets same SHA without mutating main
    -> publish reviewed draft
    -> tooling-release.yml self-hosted distribution build
    -> minimal GitHub-hosted GNU/Linux PyPI Trusted Publishing only
```

`release.yml` must fail if the candidate is no longer current `origin/main` HEAD. No repository mutation is inserted between exact-SHA verification and draft-release creation.

## Self-hosted verification and Actions cost boundary

Approved runner:

```text
DESKTOP-5HCCQIR
```

Before dispatching `tooling-test.yml` or `release.yml`:

1. tell the user this self-hosted runner will be used;
2. wait for explicit confirmation host and PowerShell environment are ready;
3. only then dispatch with `host_ready=true`.

Do not automatically dispatch self-hosted verification merely because a test is convenient.

- `tooling-test.yml`: manual self-hosted full verification/exact-SHA candidate status.
- `release.yml`: manual self-hosted release preparation.
- `tooling-release.yml` build: self-hosted Windows distribution build/verification.
- GitHub-hosted exception: minimal GNU/Linux PyPI Trusted Publishing that only downloads already verified distributions and publishes them.

Do not add GitHub-hosted push/PR/schedule full-suite matrices or move tests, compilation, packaging, release preparation, or other avoidable compute into the publish exception without explicit maintainer approval.
