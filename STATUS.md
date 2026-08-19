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

The WU-04C snapshot added normative declaration-authority/materialization semantics (`PTSIP-RMAP-013` through `PTSIP-RMAP-016`). WU-04D implements those frozen semantics; runtime/test/workflow changes do not independently move the Specification revision.

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

## Tool 0.3.6 verification status

Self-hosted workflow run `32211081862` / job `95943730594` executed source SHA:

```text
613b7aa887cb4c0aefade5b0095a5e2448bf9cd5
```

Observed execution evidence:

```text
self-hosted Windows scheduling       PASS
exact SHA checkout                   PASS
exact source identity                PASS
host Python 3.14 selection           PASS
isolated per-run .venv creation      PASS
verification tooling installation    PASS
complete pytest execution            RAN
result                               216 passed / 45 failed
```

The previous `actions/setup-python` installation failure is resolved. The self-hosted runner successfully used host Python 3.14 through `py -3.14` and created an isolated verification environment.

The full Tool `0.3.6` regression is **NOT passing**. The 45 failures include several known stale-contract groups:

- tests pinned to the earlier WU-03 Specification snapshot `12e2ccd15634ecb3d0a4195b0f61ac3f620e7540` instead of current `82abd09360df09a95fbbfb516855fa9ffb49f050`;
- historical `TOOLCHAIN`, `lifecycle_owner`, and Tool `0.3.5` expectations that are not canonical Tool `0.3.6` output;
- conformance/adoption/topology fixtures whose old source profile expectations now fail before later WU-04 effective-map integrations are entered;
- a workflow test that expected `actions/setup-python`, subsequently updated to the host-Python contract.

`tests/ptsip/test_template_materialization_036.py` did not appear in the failure list of that complete run. This is evidence that the run did not report failures from that focused file; it is not a claim that the full Tool regression passed.

Do not repeat a full self-hosted regression solely to re-prove the workflow infrastructure while these known repository failures remain. Resolve the applicable WU-aligned failures first.

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
- Tool `0.3.6`: **active development; WU-04D ACTIVE**

Current next-version work remains consolidated under `planning/0.3.6.md`.
