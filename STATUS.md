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

Current bound Specification at WU-04G entry:

```text
0.3.6-draft @ 82abd09360df09a95fbbfb516855fa9ffb49f050
```

The WU-04C snapshot added normative declaration-authority/materialization semantics (`PTSIP-RMAP-013` through `PTSIP-RMAP-016`). WU-04D implemented those frozen semantics. WU-04E/F consume them in validation/conformance and are exact-SHA verified.

WU-04G is now ACTIVE. Its accepted automatic `template -> hybrid` decision/application behavior introduces one new normative accepted-decision authority rule. The existing `82abd093...` remains the entry binding until WU-04G G0 freezes that rule and selects a new immutable Specification revision. No replacement revision has been selected yet.

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
WU-04G  clarification/adoption integration               ACTIVE
WU-04H  VPMS narrow read-only integration                LOCKED
WU-04I  regression + WU-04 completion                    LOCKED
```

Active WU-04G document:

```text
planning/0.3.6/WU-04G-clarification-adoption-effective-map.md
entry baseline 52a455115d191123504c2fd690ffe499caf0ff6a
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

### G0 normative precondition

Before runtime code can depend on D6-B/D7-C, the canonical/embedded Specification and registry must define that an **accepted project-owned clarification/adoption decision** authorizes the exact project extension/replacement and, when required to represent that decision, `template -> hybrid` source-mode transition.

This does **not** grant architecture authority to discovery or materialization. Repository evidence finds candidates only. Materialization remains deterministic reproduction only. The safe-apply layer may encode the accepted project decision and must retain exact template ID/revision and write only the minimum project-owned delta.

### Effective read and failure recovery

WU-04G will use only valid `ValidationResult.resolved_profile.effective_payload` for clarification/adoption coverage. Invalid/unresolvable profiles do not fall back to raw source.

D2-B is extended so failure does not become a dead-end:

```text
resolution failure
    -> block architecture-dependent clarification/adoption
    -> expose validation/materialization errors + failure stage + exact profile path
    -> expose non-authoritative remediation/retry information
    -> project/source correction
    -> fresh repository/profile snapshot
    -> re-run validation
    -> resume only when a valid ResolvedProfile exists
```

No partial failed effective map may become authority.

### Answer-v2 and write behavior

New canonical decision format is `ptsip-clarification-answer/v2` and omits `lifecycle_owner`. Existing v1 data may be read only through an explicit compatibility/migration path for already-persisted decisions; it must not restore canonical `TOOLCHAIN` or silently translate it.

For accepted project decisions:

- explicit profiles use canonical explicit safe apply;
- template profiles may become hybrid while preserving exact selected template ID/revision;
- hybrid profiles preserve unrelated overrides/removals;
- only the accepted project delta may be written;
- materialize-to-explicit writeback is prohibited;
- conflicts remain fail-closed and must not partially mutate source.

### Exact profile-path control plane

The normalized repository-relative selected profile path must flow through gate creation, persisted decision state, subject revision, remote file read, projection validation, stale check, and CAS write. A non-root selected profile must never silently become root `ptsip.yaml`.

### G-owned test migration/optimization

WU-04G may optimize/migrate only tests it directly owns or changes. It may consolidate duplicated immutable setup and parameterize equivalent semantic variants, but must preserve mutable CAS/snapshot/stale-state isolation, record replacement coverage for migrated historical tests, avoid repository-wide test restructuring/xdist/workflow changes, and retain full exact-SHA repository regression as the final authority.

## OpenAI Local Bridge integration

Repository-local task registry:

```text
.codex/openai-local-bridge/tasks.toml
repository_id: Kinirin/PTSIP
task_id:       wu04g-precheck
executable:    py
args:          -3.14 .github/scripts/wu04g_guard.py precheck
access:        read
```

The previous full-suite Bridge task was replaced because the active WU-04G branch is intentionally not yet globally regression-clean and complete repository regression is reserved for the post-G5 exact-SHA gate.

Maintainer-provided remote E2E result on 2026-08-19:

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

This establishes that OpenAI Local Bridge discovered the PTSIP repository and registered task, accepted the read-only remote execution request, launched the repository-defined `wu04g-precheck` process, produced run/log evidence, and completed that task with exit code 0.

This Bridge E2E result is **connection/task-execution evidence only**. It does not mark WU-04G G1-G5 complete, does not replace track-specific guard/test commands, and does not establish complete repository regression or release readiness.

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
- Tool `0.3.6`: **active development; WU-04G ACTIVE**

Current next-version work remains consolidated under `planning/0.3.6.md`.
