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
0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea
```

WU-04G G0 froze `PTSIP-RMAP-017`, which defines accepted project-owned clarification/adoption decisions as authority for the exact safe-apply declaration delta and permits `template -> hybrid` only to represent that accepted decision while preserving the selected immutable template identity. Tool constants and the repository root profile are bound to this immutable Specification revision.

WU-04G remains ACTIVE. G1 effective read + selector coverage is VERIFIED; G2 clarification-answer/v2 is VERIFIED; G3 hybrid safe apply is VERIFIED; G4 exact profile-path control plane is VERIFIED; G5 recovery + integration + migration audit is next and has not yet been entered.

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
WU-04G  clarification/adoption integration               ACTIVE — G0 COMPLETE / G1 VERIFIED / G2 VERIFIED / G3 VERIFIED / G4 VERIFIED / G5 NEXT
WU-04H  VPMS narrow read-only integration                LOCKED
WU-04I  regression + WU-04 completion                    LOCKED
```

Active WU-04G document:

```text
planning/0.3.6/WU-04G-clarification-adoption-effective-map.md
entry baseline 52a455115d191123504c2fd690ffe499caf0ff6a
current normative revision d6995ed232e845b88d8235b851e80ab54b7804ea
```

The earlier `planning/0.3.6/pre-entry-WU-04G-decision-review.md` is historical review context and no longer the implementation authority.

## WU-04G accepted scope

The maintainer accepted:

```text
D1-B  ResolvedProfile.effective_payload as clarification/adoption read authority
D2-B  fail closed, then continue with deterministic remediation/retry information
D3-B  associated-artifact coverage suppresses duplicate component questions
D4-B  canonical shared selector/candidate coverage primitive
D5-B  create ptsip-clarification-answer/v2 and remove lifecycle_owner from new canonical answers
D6-B  accepted decisions may convert template -> hybrid when needed
D7-C  perform eligible hybrid conversion/application automatically
D8-B  carry exact selected profile path through decision protocol and remote CAS
D9-B  bounded optimization/migration of WU-04G-owned tests only
```

### G0 normative result

G0 is COMPLETE. The canonical/embedded Specification and registry define that an **accepted project-owned clarification/adoption decision** authorizes only the exact project extension/replacement and, when required to represent that decision, `template -> hybrid` source-mode transition.

This does **not** grant architecture authority to discovery or materialization. Repository evidence finds candidates only. Materialization remains deterministic reproduction only. The safe-apply layer may encode the accepted project decision and must retain exact template ID/revision and write only the minimum project-owned delta.

### G1 verified effective read and selector coverage

G1 clarification/adoption read authority is:

```text
selected profile
    -> validate_profile()
    -> ValidationResult.resolved_profile
    -> ResolvedProfile.effective_payload
```

Clarification no longer treats raw YAML as a second architecture interpretation. Effective component and associated-artifact coverage use the shared validation/selector coverage primitive, including ambiguity review rather than ownership guessing. Partial-declaration coverage remains a distinct valid-profile contract rather than relying on schema-invalid raw source.

The G1 participating existing files are:

```text
tests/ptsip/test_clarification.py
tests/ptsip/test_adoption_033.py
tests/ptsip/test_repository_self_profile_035.py
```

Shared stateless support is in `tests/ptsip/_wu04g_support.py`, and focused G1 coverage is in `TestG1EffectiveReadCoverage` within `tests/ptsip/test_clarification_adoption_effective_map_036.py`.

Maintainer-provided OpenAI Local Bridge G1 verification:

```text
repository_id:                  Kinirin/PTSIP
task_id:                        wu04g-g1-files
run_id:                         fa46e93a3d7545b7bf793e3df21263c0
status:                         completed
exit_code:                      0
failure_kind:                   null
protocol_version:               2025-11-25
remote_task_execution_verified: true
log_available:                  true
log_truncated:                  false
capability_url_exposed:         false
```

Therefore G1 files-mode verification is PASS. This does not replace the final G5 exact-SHA complete repository regression.

### G2 verified answer-v2 decision protocol

G2 is VERIFIED. New canonical decisions use:

```text
ptsip-clarification-answer/v2

classification
purpose
shipped
runtime_required
executable
```

`lifecycle_owner` is absent from canonical `DecisionAnswer`, new rendered clarification replies, new local/GitHub resolution payloads, and newly stored canonical answers. Existing v1 data is accepted only through an explicit compatibility reader/canonicalization boundary. Legacy `TOOLCHAIN` is not restored as a Tool 0.3.6 classification and is never silently translated to `DEVELOPMENT_TOOLING`.

G2 preserves first-valid-resolution, retry comparison, store isolation, and GitHub authority semantics. Persisted v1 answers may be normalized to canonical v2 only for read-time semantic comparison; the Tool does not implicitly rewrite historical persisted data merely by reading it.

Maintainer-provided OpenAI Local Bridge G2 verification:

```text
wu04g-scope
run_id:                         c57d0fd7adab45c39739685a6fca581d
status:                         completed
exit_code:                      0
failure_kind:                   null
protocol_version:               2025-11-25
remote_task_execution_verified: true
log_available:                  true
log_truncated:                  false
capability_url_exposed:         false

wu04g-g2-files
run_id:                         95f081cc5f2140cea719afea67a33b2a
status:                         completed
exit_code:                      0
failure_kind:                   null
protocol_version:               2025-11-25
remote_task_execution_verified: true
log_available:                  true
log_truncated:                  false
capability_url_exposed:         false
```

An earlier files-mode run `97def58217ec44fb8b1725ef7299d693` produced `41 passed / 1 failed`; the sole failure was the frozen historical `test_topology_legacy_boundary_profile_preserves_historical_plane_classification` contract. Because `test_topology_032.py` is a mixed file whose topology semantics are explicitly outside G2 ownership, that test was preserved unchanged and the G2 selector now runs only its G2-owned `DecisionAnswer`/projection node. This does not erase or resolve the legacy topology debt; the later all-G/full regression remains authoritative for it.

### G3 verified hybrid safe apply

G3 is VERIFIED. Accepted project decisions now preserve the explicit projection baseline and may encode only the exact accepted project-owned delta for template/hybrid source. A template source changes to hybrid only when that decision requires a new declaration; the exact selected template ID/revision is preserved. Existing hybrid unrelated overrides/removals remain intact, materialize-to-explicit writeback is prohibited, conflicts fail closed without partial mutation, and semantic no-op decisions do not alter source.

Focused G3 coverage is in `TestG3HybridSafeApply`, and the participating G3 files include `test_adoption_033.py`, `test_decision_control_plane.py`, and the G3-owned explicit projection node from mixed `test_topology_032.py`. The topology migration behavior in that mixed file remains semantically frozen.

Maintainer-provided OpenAI Local Bridge G3 verification on final implementation SHA `cf663949bb7005616574db62a2d850bbc1d80bce`:

```text
wu04g-scope
run_id:                         fc9a315beafd485a88fa6d69772cc18a
status:                         completed
exit_code:                      0
failure_kind:                   null
protocol_version:               2025-11-25
remote_task_execution_verified: true
log_available:                  true
log_truncated:                  false
capability_url_exposed:         false

wu04g-g3-files
run_id:                         7993c2bb283b481e9775ea7a08ad45b2
status:                         completed
exit_code:                      0
failure_kind:                   null
protocol_version:               2025-11-25
remote_task_execution_verified: true
log_available:                  true
log_truncated:                  false
capability_url_exposed:         false
```

An earlier G3 files-mode run `38988b39ba70466396697c350e214af7` produced `21 passed / 1 failed`. The sole failure was not a production safe-apply defect: the newly added template-adoption integration fixture selected `python-package-library` while omitting tracked `src/**` and `tests/**` files required by that template, so existing profile validation correctly failed closed before projection. The fixture was corrected by adding tracked template-owned files at `cf663949...`; production safe-apply code was unchanged. Final scope and files-mode verification then passed.

### G4 verified exact profile-path control plane

G4 is VERIFIED. Production/test implementation completed at:

```text
471baba7656dbbb25d1a976d53bc20193b731def
```

Repository-local Bridge invocation hardening then advanced the branch without changing G4 production semantics. Direct repository-native verification was performed at:

```text
c7f8060514a4814e5e6ccd9fd75280b2f931d54d
```

Verification result:

```text
wu04g_guard.py scope
    PASS
    scope base:    52a455115d191123504c2fd690ffe499caf0ff6a
    changed files: 44
    changed tests: 10

wu04g_track_tests.py G4 --mode files
    37 passed in 79.23s (0:01:19)
```

The exact selected profile path is now a normalized repository-relative control-plane identity. It is persisted with decisions, preserved across retry/rebind, used for exact local and hosted file projection/read/write, and included in non-root GitHub authority identity. Historical root `ptsip.yaml` identity remains compatible. A resolve attempt against a different path fails with `PROFILE_PATH_MISMATCH`; stale subject revisions and branch-head/CAS races fail closed without applying the selected profile.

Focused G4 coverage remains separated across non-root projection, local reconciliation, persistence, retry/rebind, exact hosted read/write, wrong-path protection, stale revision, and branch-head race contracts. The mixed topology file contributes only its G4-owned explicit path/projection node; historical topology semantics remain frozen.

After pytest had already reported `37 passed`, Windows emitted an ignored pytest temporary-directory cleanup callback `PermissionError` for `%TEMP%\pytest-of-rhkrt\pytest-current`. This is post-test environment cleanup noise and not a G4 product/test failure.

G5 recovery + cross-track integration + final migration audit is now the next WU-04G track and has not yet been entered.

For accepted project decisions:

- explicit profiles use canonical explicit safe apply;
- template profiles may become hybrid while preserving exact selected template ID/revision;
- hybrid profiles preserve unrelated overrides/removals;
- only the accepted project delta may be written;
- materialize-to-explicit writeback is prohibited;
- conflicts remain fail-closed and must not partially mutate source.

### Exact profile-path control plane

The normalized repository-relative selected profile path flows through gate creation, persisted decision state, subject revision, remote file read, projection validation, stale check, and CAS write. A non-root selected profile never silently becomes root `ptsip.yaml`.

### G-owned test migration/optimization

WU-04G may optimize/migrate only tests it directly owns or changes. It may consolidate duplicated immutable setup and parameterize equivalent semantic variants, but must preserve mutable CAS/snapshot/stale-state isolation, record replacement coverage for migrated historical tests, avoid repository-wide test restructuring/xdist/workflow changes, and retain full exact-SHA repository regression as the final authority.

## OpenAI Local Bridge integration

Repository-local task registry:

```text
.codex/openai-local-bridge/tasks.toml
repository_id: Kinirin/PTSIP
```

Current read-only tasks include:

```text
wu04g-precheck
    py -3.14 .github/scripts/wu04g_guard.py precheck

wu04g-scope
    py -3.14 .github/scripts/wu04g_guard.py scope

wu04g-g1-files
    .venv/Scripts/python.exe .github/scripts/wu04g_track_tests.py G1 --mode files

wu04g-g2-files
    .venv/Scripts/python.exe .github/scripts/wu04g_track_tests.py G2 --mode files

wu04g-g3-files
    .venv/Scripts/python.exe .github/scripts/wu04g_track_tests.py G3 --mode files

wu04g-g4-files
    .venv/Scripts/python.exe .github/scripts/wu04g_track_tests.py G4 --mode files
```

Repository-local Bridge evidence inspection is available through:

```text
.github/scripts/wu04g_bridge_log.ps1
```

This helper reads already-recorded Bridge run evidence under `%LOCALAPPDATA%\OpenAI_Local_Bridge\runs` and does not create a new task run.

The previous full-suite Bridge task was replaced because the active WU-04G branch is intentionally not yet globally regression-clean and complete repository regression is reserved for the post-G5 exact-SHA gate.

Maintainer-provided remote E2E precheck result on 2026-08-19:

```text
passed:                         true
repository_id:                  Kinirin/PTSIP
task_id:                        wu04g-precheck
run_id:                         a4441c401c994cf6bc09ec539a56ffa7
status:                         completed
exit_code:                      0
failure_kind:                   null
protocol_version:               2025-11-25
remote_task_execution_verified: true
log_available:                  true
log_truncated:                  false
capability_url_exposed:         false
```

The G1 files-mode task completed successfully in run `fa46e93a3d7545b7bf793e3df21263c0`. G2 scope and files-mode tasks completed successfully in runs `c57d0fd7adab45c39739685a6fca581d` and `95f081cc5f2140cea719afea67a33b2a`. G3 final scope and files-mode tasks completed successfully in runs `fc9a315beafd485a88fa6d69772cc18a` and `7993c2bb283b481e9775ea7a08ad45b2`. All cited successful historical Bridge stage-gate runs completed with exit code 0 and `remote_task_execution_verified=true`.

G4 was verified directly with the same checked-in repository-native commands because Bridge transport/acceptance instability was a development-tooling problem rather than a PTSIP semantic prerequisite. For WU-04G intermediate stages, the checked-in guard/track command executed against a recorded HEAD is authoritative; Bridge is an optional transport for those read tasks. This does not weaken G5: the final exact-SHA self-hosted complete repository regression remains mandatory.

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
- has focused coverage in `tests/ptsip/test_profile_validation_036.py`.

WU-04F:

- removes raw YAML architecture reload from `src/ptsip/conformance.py` and `src/ptsip/conformance_engine.py`;
- consumes effective components and `component_dependency_policy` from `ValidationResult.resolved_profile.effective_payload`;
- removes the old materialization-required branch;
- applies effective components to dependency, artifact, project-policy, language, build, lifecycle, and agent-decision comparison paths;
- has focused explicit/template/hybrid consumption coverage in `tests/ptsip/test_conformance_effective_map_036.py`.

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

Therefore WU-04E/F are COMPLETE / exact-SHA verified.

The 13 remaining failures were classified as:

```text
3  decision/clarification-control historical expectations
1  pilot/evidence fixture using pre-0.3.6 profile state
1  stale lifecycle-evidence expectation for an unscoped release workflow
1  agent-decision conflict expectation for later decision integration
7  topology/legacy-profile migration fixtures
```

WU-04G may migrate only the decision/clarification failures/tests within its accepted scope. The lifecycle-evidence failure is not an E/F regression.

The workflow still concluded failure. Consequently:

- no `self-hosted/tooling-test` success status was recorded for `48b75e6...`;
- Tool `0.3.6` is **not** globally regression-clean;
- release/build/smoke steps after pytest were skipped;
- this evidence closes E/F only and does not establish release readiness.

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

Blind `TOOLCHAIN -> DEVELOPMENT_TOOLING` translation is prohibited. WU-04G's v1 decision compatibility is a narrow decision-protocol migration and does not enter the WU-07 legacy architecture reader.

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
- Tool `0.3.6`: **active development; WU-04G ACTIVE — G4 VERIFIED / G5 NEXT**

Current next-version work remains consolidated under `planning/0.3.6.md`.
