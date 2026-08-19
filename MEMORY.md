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

Active stage:

```text
planning/0.3.6/WU-04G-clarification-adoption-effective-map.md
entry baseline 52a455115d191123504c2fd690ffe499caf0ff6a
```

The earlier `planning/0.3.6/pre-entry-WU-04G-decision-review.md` is historical decision-analysis context. The official WU-04G document owns implementation after the maintainer selected D1-B through D9-B/C options.

Published PyPI Tool remains `0.3.5`; the development branch Tool identity is `0.3.6`.

## Specification snapshots

```text
WU-00 baseline:              654e41d49600fc091f9a6cb6b1c60bbc7da4e301
WU-03 canonical activation:  12e2ccd15634ecb3d0a4195b0f61ac3f620e7540
WU-04C authority snapshot:   82abd09360df09a95fbbfb516855fa9ffb49f050
```

Current Tool/root-profile binding at WU-04G entry:

```text
Specification 0.3.6-draft
SPEC_REVISION 82abd09360df09a95fbbfb516855fa9ffb49f050
```

WU-04C froze `PTSIP-RMAP-013` through `PTSIP-RMAP-016`. WU-04D implemented the frozen materialization semantics. WU-04E/F consume those semantics in validation and conformance.

WU-04G accepted D6-B/D7-C adds a genuine normative authority rule: an accepted project-owned clarification/adoption decision may authorize the exact hybrid project extension/replacement and, when required to represent that accepted decision, a `template -> hybrid` source-mode transition. Therefore G0 must freeze this rule in canonical/embedded Specification/registry data and select a new immutable `SPEC_REVISION` before runtime implementation depends on automatic conversion. Do not invent or predeclare the future SHA.

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

Frozen base boundary:

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

Verified E/F pipeline:

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

`ValidationResult.resolved_profile` is an in-process handoff and is deliberately omitted from `ValidationResult.as_dict()`.

## WU-04G accepted decision package

```text
D1-B  ResolvedProfile.effective_payload is clarification/adoption read authority
D2-B  invalid/unresolvable profiles fail closed, then expose deterministic remediation/retry work
D3-B  effective associated-artifact coverage suppresses duplicate component questions
D4-B  selector/candidate coverage uses one shared canonical primitive
D5-B  create ptsip-clarification-answer/v2 and remove lifecycle_owner from new canonical answers
D6-B  accepted decisions may extend hybrid and convert template -> hybrid when required
D7-C  eligible accepted decisions automatically perform that conversion/application
D8-B  exact selected profile path travels through complete decision protocol and remote CAS
D9-B  optimize/migrate only WU-04G-owned tests with coverage-preservation proof
```

### D2 recovery extension

Fail closed does not end with a dead-end error. The expected G behavior is:

```text
profile resolution failure
    -> stop architecture-dependent clarification/adoption
    -> preserve validation/materialization errors + failure stage + exact profile path
    -> provide non-authoritative remediation/retry information
    -> project/source is corrected
    -> take fresh repository/profile snapshot
    -> re-run validate_profile()
    -> resume only after a valid ResolvedProfile exists
```

No raw-profile fallback, partial effective-map reuse, inferred architecture, or silent repair is allowed.

### D5 answer-v2 migration

New canonical answers use `ptsip-clarification-answer/v2` with:

```text
classification
purpose
shipped
runtime_required
executable
```

`lifecycle_owner` is removed from new canonical decisions and Tool 0.3.6 Project Profile serialization. Existing v1 data may be handled only through explicit compatibility/migration logic for already-persisted decisions; v1 facts cannot restore `TOOLCHAIN` as canonical authority or silently map it to `DEVELOPMENT_TOOLING`.

### D6/D7 safe-apply authority

Automatic `template -> hybrid` is allowed only because an accepted project decision authorizes the exact project-owned change. Repository evidence merely discovers candidates; the materializer merely resolves declarations. Safe apply must retain exact template ID/revision and write only the minimum accepted project delta. It must not materialize the full effective map to explicit form or introduce unrelated overrides/removals.

### D8 path identity

The normalized repository-relative selected profile path must persist through gate creation, decision storage, resolution context, subject revision, remote file read, stale check, projection validation, and CAS write. Never silently substitute root `ptsip.yaml` for a non-root selected profile.

### D9 bounded test migration

Only G-owned/touched tests may be optimized/migrated. Collect duration evidence first; consolidate only duplicated immutable setup/semantically equivalent variants; keep mutable CAS/snapshot/stale-state isolation; record replacement coverage for any merged/removed historical test; do not change xdist/workflow execution; final authority remains exact-SHA complete repository regression.

## WU-04 stage state

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

WU-04G implementation order is G0 normative freeze, then effective-read/selector integration, answer-v2 migration, hybrid safe apply, exact profile-path control plane, recovery/test integration. Parallel work may begin only after shared normative meaning is fixed and file ownership is partitioned.

WU-04H must not be entered or pre-created before the WU-04G completion gate is reviewed.

## OpenAI Local Bridge operational evidence

Canonical registry:

```text
.codex/openai-local-bridge/tasks.toml
```

Current Bridge task selected by the PTSIP repository policy:

```text
repository_id   Kinirin/PTSIP
task_id         wu04g-precheck
executable      py
args            -3.14 .github/scripts/wu04g_guard.py precheck
cwd             .
access          read
```

The task is deliberately the existing WU-04G repository-native precheck rather than the currently known-red complete pytest suite. It is a Bridge connectivity/task-execution probe only and does not replace G1-G5 focused verification or the post-G5 exact-SHA complete repository regression.

Maintainer-reported remote E2E success on 2026-08-19:

```text
run_id                         a4441c401c994cf6bc09ec539a56ffa7
status                         completed
exit_code                      0
failure_kind                   null
protocol_version               2025-11-25
remote_task_execution_verified true
log_available                  true
log_truncated                  false
capability_url_exposed         false
```

This proves Bridge repository discovery + task discovery + read-only remote execution + run evidence/log retrieval for `wu04g-precheck`. Do not interpret it as WU-04G implementation completion, full regression success, or release readiness.

## Latest self-hosted regression evidence

GitHub Actions run:

```text
run 32240740753
job 96030499443
source SHA 48b75e699a592703e4e03a8462131e4932103677
```

Observed execution facts:

```text
self-hosted runner scheduling       PASS
exact SHA checkout                  PASS
exact source identity               PASS
host Python 3.14.3 preparation      PASS
isolated .venv creation             PASS
verification tooling installation   PASS
complete pytest execution           RAN
result                              260 passed / 13 failed
```

`tests/ptsip/test_profile_validation_036.py` and `tests/ptsip/test_conformance_effective_map_036.py` did not appear in the exhaustive 13-failure summary, so both focused E/F contracts passed at the exact verification SHA.

Remaining failures were classified as:

```text
3  decision/clarification-control historical expectations
1  pilot/evidence fixture using pre-0.3.6 profile state
1  stale lifecycle-evidence expectation for an unscoped release workflow
1  agent-decision conflict expectation for later decision integration
7  topology/legacy-profile migration fixtures
```

WU-04G may migrate only the decision/clarification tests that fall within its accepted scope. The overall workflow concluded failure, so no `self-hosted/tooling-test` success status was recorded for `48b75e6...`. Do not claim full Tool `0.3.6` regression success or release readiness from this run.

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
    -> self-hosted/tooling-test status on exact SHA only after complete workflow success

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

Before repository changes, read `AGENTS.md`, this `MEMORY.md`, `ptsip.yaml`, `src/ptsip/constants.py`, applicable Specification, `planning/0.3.6.md`, and `planning/0.3.6/WU-04G-clarification-adoption-effective-map.md` while G is active.
Re-read the remote target branch HEAD immediately before **every GitHub write**, merge, release preparation, or evidence claim.
