# AGENTS.md

These instructions apply to coding agents working anywhere in this repository.

## Developer Policy vs Support Feature Policy

PTSIP has two non-interchangeable policy classes.

- `PTSIP_DEVELOPER_POLICY` uses IDs `MPD-####`, lives under `developer/policy/`, is automated by `developer/automation/`, and must not ship as a consumer runtime contract.
- `PTSIP_SUPPORT_FEATURE` uses IDs `SFP-####`. Canonical repository authority lives under `docs/Support_policy/policy/`; repository-side Support Policy automation lives under `docs/Support_policy/automation/`. Installed distributions receive the deterministic `ptsip/support/` projection. This boundary is separate from `developer/`.
- PTSIP repository self-management profiles belong under `developer/profiles/`. The canonical repository self-profile is `developer/profiles/ptsip-repository.yaml`; the former root `ptsip.yaml` compatibility bridge was retired by the 0.4.0 P01-F migration.
- The legacy mixed-policy `decisions/` tree was retired and removed after migration to current SFP/MPD authorities. Do not recreate it as an active policy source.
- Product runtime under `src/ptsip/**` or `src/vpms/**` must not depend on MPD documents. Current product governance consumes shipped SFP contracts only.
- Developer planning authority is `developer/planning/index.yaml` and version control planes under `developer/planning/<version>/index.yaml`. `current_gate` must match `^WU-[0-9]{2}(?:-P[0-9]{2})?$`.
- `Pxx` means Plan Extension. WU files may be detailed; automatable state must remain in structured machine fields. Use `WU-xx-explanation.yaml` only for definitions that cannot yet be represented by supported machine fields.

## Mandatory developer policy entry

Before broadly reading repository policy or planning prose, resolve the developer-policy context mechanically:

```text
python -m developer.automation.policy_resolver resolve --scope <repository-path> --operation <READ|MODIFY|PLAN|VERIFY|RELEASE>
```

The Policy Resolver performs exact bounded lookup through `developer/policy/policy-resolver-bindings.yaml` and `developer/policy/index.yaml`. Use only the returned canonical policy IDs and rule sections as the normal policy-loading path.

For a returned section, prefer the bounded lookup:

```text
python -m developer.automation.policy_resolver get <MPD-ID> --section <rule-section>
```

Rules:

- Do not scan `developer/policy/**`, planning trees, ADR/history, or prose documentation merely to discover potentially relevant policy.
- Do not substitute semantic similarity, filename similarity, nearby documents, or natural-language inference for a resolver binding.
- A resolver failure is fail-closed for the dependent policy lookup. Do not choose an alternative policy manually.
- The resolver is a routing/projection control plane only. Canonical MPD records remain authority.
- `explain` is optional human-facing metadata and is not the normal coding-agent policy path.
- Re-run resolution after changing task scope, operation class, or branch context.


## Mandatory PTSIP Agent Contract entry

For PTSIP product/consumer operations, resolve the bounded machine contract before reading any narrative specification:

```text
python -m agent_contracts.resolver <PTSIP-OP-ID> --json
```

Current operation IDs are `PTSIP-OP-ADOPT-001`, `PTSIP-OP-VALIDATE-001`, `PTSIP-OP-CONFORM-001`, `PTSIP-OP-RECONCILE-AUTHORITY-001`, and `PTSIP-OP-MIGRATE-PROFILE-001`. Use only the returned exact rules, actions, conditions, gates, I/O schemas, vocabularies, and implementation bindings.

For current repository state, resolve one exact owner instead of reading repository history or status prose:

```text
python -m developer.automation.repository_state_resolver <domain> --json
```

Do not preload `adoption/`, `agents/AGENT-CONTRACT.md`, `STATUS.md`, `MEMORY.md`, `spec/*.md`, or `reference/*.md` as normative or default coding-agent context. Human-readable history/reference material is optional and cannot override the current machine contract or canonical state owner. Resolver failure is fail-closed.

## Mandatory MPD identity lifecycle preflight

Do not choose a new `MPD-####` ID or change an MPD lifecycle status by repository scan or conversational inference.

For an existing ID, inspect it mechanically first:

```text
python -m developer.automation.policy_identity_lifecycle inspect <MPD-ID>
```

Before allocating a new ID, create an approved provenance record under `developer/policy/approvals/` and run:

```text
python -m developer.automation.policy_identity_lifecycle preflight --approval-ref developer/policy/approvals/<record>.yaml
```

After the policy file is materialized with the exact allocated ID and explicit approved status, register it through:

```text
python -m developer.automation.policy_identity_lifecycle register --approval-ref developer/policy/approvals/<record>.yaml --policy-file developer/policy/<MPD-ID>.yaml
```

For a lifecycle status change on an existing MPD, run `status-preflight` first. Temporary implementation approval does not imply `ACTIVE` or `DRAFT`; the approval provenance must state the target status explicitly. Any ID collision, index/file/subject-registry mismatch, missing approval provenance, or status mismatch is fail-closed.

## Implementation Work Packet

For implementation work with a registered task context, prepare a bounded developer-only work packet before editing:

```text
python -m developer.automation.implementation_work_packet prepare --scope <repository-path> --operation MODIFY --output <temporary-json-path> --json
```

Use the packet's exact edit targets, required tests, and verification stages. Run `check` before verification or commit. Branch, HEAD, context, or unlisted-path changes invalidate the packet and require regeneration.

## Deterministic Test Mode verification

Normal development verification uses automatic Test Mode resolution. The agent does not infer the affected Test Mode from prose, semantic similarity, confidence, or uncertainty.

```text
python .github/scripts/resolve_test_modes.py automatic --head HEAD
```

Rules:

- Project Profile verification `analysis_inputs` and owned test `include` paths are the selection authority.
- Run only the Test Modes returned by the resolver.
- `all` is not an uncertainty fallback and is not a Test Mode.
- Use `full` only for a release boundary, an explicit policy requirement, or an explicit maintainer request.
- An unmapped changed path is fail-closed; do not replace it with full verification.
- Manual mode selection is for an explicit targeted rerun or debugging request, not normal post-change verification.

## Mandatory branch command plane

Branch creation and remote branch inspection are governed by `MPD-0014` and must enter through the canonical command plane:

```text
python -m developer.automation.branch_control commands
python -m developer.automation.branch_control list --repository Kinirin/PTSIP
python -m developer.automation.branch_control inspect --repository Kinirin/PTSIP --branch <exact-branch-name>
python -m developer.automation.branch_control create --repository Kinirin/PTSIP --branch <exact-branch-name> --approved-name <exact-user-approved-branch-name> --base-ref <exact-base-ref>
```

The registered v1 command vocabulary is exactly `commands | list | inspect | create`. An unregistered branch operation fails closed. A coding agent that is about to create, query, rename, delete, or otherwise mutate a branch outside this command plane must treat that path as a management-intent deviation signal and stop rather than invent an alternative branch-management mechanism.

The `create` command internally applies `branch_creation_guard` and the only authorized mutation backend is the GitHub `create_branch` API. Do not create branches with `git push`, `git switch -c`, `git checkout -b`, `git update-ref`, the GitHub `update_ref` API, or a workflow that writes a new ref.

## Mandatory branch creation preflight

Branch creation is governed by `MPD-0014`. Coding agents must not invent a branch name from planning state, version context, a Project Profile change, a test need, or a temporary verification need.

Before creating a branch, the canonical command plane performs the exact-name preflight mechanically. Direct guard invocation is diagnostic/internal and is not the normal branch-management entry:

```text
python -m developer.automation.branch_creation_guard validate --candidate <exact-branch-name> --approved-name <exact-user-approved-branch-name> --authorization-source USER_EXPLICIT --request-kind DEVELOPMENT_VERSION_BRANCH --creation-mechanism GITHUB_CREATE_BRANCH_API
```

Rules:

- Current v1 creation authority recognizes only development-version branches matching `^dev/[0-9]+\\.[0-9]+\\.[0-9]+# AGENTS.md

These instructions apply to coding agents working anywhere in this repository.

## Developer Policy vs Support Feature Policy

PTSIP has two non-interchangeable policy classes.

- `PTSIP_DEVELOPER_POLICY` uses IDs `MPD-####`, lives under `developer/policy/`, is automated by `developer/automation/`, and must not ship as a consumer runtime contract.
- `PTSIP_SUPPORT_FEATURE` uses IDs `SFP-####`. Canonical repository authority lives under `docs/Support_policy/policy/`; repository-side Support Policy automation lives under `docs/Support_policy/automation/`. Installed distributions receive the deterministic `ptsip/support/` projection. This boundary is separate from `developer/`.
- PTSIP repository self-management profiles belong under `developer/profiles/`. The canonical repository self-profile is `developer/profiles/ptsip-repository.yaml`; the former root `ptsip.yaml` compatibility bridge was retired by the 0.4.0 P01-F migration.
- The legacy mixed-policy `decisions/` tree was retired and removed after migration to current SFP/MPD authorities. Do not recreate it as an active policy source.
- Product runtime under `src/ptsip/**` or `src/vpms/**` must not depend on MPD documents. Current product governance consumes shipped SFP contracts only.
- Developer planning authority is `developer/planning/index.yaml` and version control planes under `developer/planning/<version>/index.yaml`. `current_gate` must match `^WU-[0-9]{2}(?:-P[0-9]{2})?$`.
- `Pxx` means Plan Extension. WU files may be detailed; automatable state must remain in structured machine fields. Use `WU-xx-explanation.yaml` only for definitions that cannot yet be represented by supported machine fields.

## Mandatory developer policy entry

Before broadly reading repository policy or planning prose, resolve the developer-policy context mechanically:

```text
python -m developer.automation.policy_resolver resolve --scope <repository-path> --operation <READ|MODIFY|PLAN|VERIFY|RELEASE>
```

The Policy Resolver performs exact bounded lookup through `developer/policy/policy-resolver-bindings.yaml` and `developer/policy/index.yaml`. Use only the returned canonical policy IDs and rule sections as the normal policy-loading path.

For a returned section, prefer the bounded lookup:

```text
python -m developer.automation.policy_resolver get <MPD-ID> --section <rule-section>
```

Rules:

- Do not scan `developer/policy/**`, planning trees, ADR/history, or prose documentation merely to discover potentially relevant policy.
- Do not substitute semantic similarity, filename similarity, nearby documents, or natural-language inference for a resolver binding.
- A resolver failure is fail-closed for the dependent policy lookup. Do not choose an alternative policy manually.
- The resolver is a routing/projection control plane only. Canonical MPD records remain authority.
- `explain` is optional human-facing metadata and is not the normal coding-agent policy path.
- Re-run resolution after changing task scope, operation class, or branch context.


## Mandatory PTSIP Agent Contract entry

For PTSIP product/consumer operations, resolve the bounded machine contract before reading any narrative specification:

```text
python -m agent_contracts.resolver <PTSIP-OP-ID> --json
```

Current operation IDs are `PTSIP-OP-ADOPT-001`, `PTSIP-OP-VALIDATE-001`, `PTSIP-OP-CONFORM-001`, `PTSIP-OP-RECONCILE-AUTHORITY-001`, and `PTSIP-OP-MIGRATE-PROFILE-001`. Use only the returned exact rules, actions, conditions, gates, I/O schemas, vocabularies, and implementation bindings.

For current repository state, resolve one exact owner instead of reading repository history or status prose:

```text
python -m developer.automation.repository_state_resolver <domain> --json
```

Do not preload `adoption/`, `agents/AGENT-CONTRACT.md`, `STATUS.md`, `MEMORY.md`, `spec/*.md`, or `reference/*.md` as normative or default coding-agent context. Human-readable history/reference material is optional and cannot override the current machine contract or canonical state owner. Resolver failure is fail-closed.

## Mandatory MPD identity lifecycle preflight

Do not choose a new `MPD-####` ID or change an MPD lifecycle status by repository scan or conversational inference.

For an existing ID, inspect it mechanically first:

```text
python -m developer.automation.policy_identity_lifecycle inspect <MPD-ID>
```

Before allocating a new ID, create an approved provenance record under `developer/policy/approvals/` and run:

```text
python -m developer.automation.policy_identity_lifecycle preflight --approval-ref developer/policy/approvals/<record>.yaml
```

After the policy file is materialized with the exact allocated ID and explicit approved status, register it through:

```text
python -m developer.automation.policy_identity_lifecycle register --approval-ref developer/policy/approvals/<record>.yaml --policy-file developer/policy/<MPD-ID>.yaml
```

For a lifecycle status change on an existing MPD, run `status-preflight` first. Temporary implementation approval does not imply `ACTIVE` or `DRAFT`; the approval provenance must state the target status explicitly. Any ID collision, index/file/subject-registry mismatch, missing approval provenance, or status mismatch is fail-closed.

## Implementation Work Packet

For implementation work with a registered task context, prepare a bounded developer-only work packet before editing:

```text
python -m developer.automation.implementation_work_packet prepare --scope <repository-path> --operation MODIFY --output <temporary-json-path> --json
```

Use the packet's exact edit targets, required tests, and verification stages. Run `check` before verification or commit. Branch, HEAD, context, or unlisted-path changes invalidate the packet and require regeneration.

## Deterministic Test Mode verification

Normal development verification uses automatic Test Mode resolution. The agent does not infer the affected Test Mode from prose, semantic similarity, confidence, or uncertainty.

```text
python .github/scripts/resolve_test_modes.py automatic --head HEAD
```

Rules:

- Project Profile verification `analysis_inputs` and owned test `include` paths are the selection authority.
- Run only the Test Modes returned by the resolver.
- `all` is not an uncertainty fallback and is not a Test Mode.
- Use `full` only for a release boundary, an explicit policy requirement, or an explicit maintainer request.
- An unmapped changed path is fail-closed; do not replace it with full verification.
- Manual mode selection is for an explicit targeted rerun or debugging request, not normal post-change verification.

## Mandatory branch creation preflight

Branch creation is governed by `MPD-0014`. Coding agents must not invent a branch name from planning state, version context, a Project Profile change, a test need, or a temporary verification need.

Before creating a branch, validate the exact user-approved name mechanically:

```text
python -m developer.automation.branch_creation_guard validate --candidate <exact-branch-name> --approved-name <exact-user-approved-branch-name> --authorization-source USER_EXPLICIT --request-kind DEVELOPMENT_VERSION_BRANCH --creation-mechanism GITHUB_CREATE_BRANCH_API
```

Rules:

.
- The exact candidate branch name must equal the exact branch name supplied by the user request. Missing or inferred names fail closed.
- Changes to `ptsip-public-profile-catalog/v1` or a `pp.[0-9].[0-9]{2}` identity do not authorize branch creation and do not require a development branch rename or version change.
- Existing `tool-0.3.[0-9]-*` branches are grandfathered for retention only; that pattern is not new branch-creation authority.
- The only authorized creation mechanism is the GitHub `create_branch` API after a successful guard result. Do not create branches with `git push`, `git switch -c`, `git checkout -b`, `git update-ref`, the GitHub `update_ref` API, or a workflow that writes a new ref.
- Merge and retirement policy are separate from creation. Do not use merge eligibility or branch age as a substitute for creation authorization.

## Mandatory branch-aware planning entry

Before interpreting any version-specific planning document, `current_gate`, WU number, or branch name, resolve the planning entry mechanically:

```text
python -m developer.automation.planning.planning_entry_resolver
```

The resolver reads the current Git branch and performs an exact lookup against `developer/planning/index.yaml -> plans[].entry_routing.branch_entrypoints`. Treat its `entry_document` as the planning entry point for the current branch.

Rules:

- Exact mapping only. Do not infer a WU from branch prefixes, suffixes, naming similarity, `current_gate`, nearby files, or historical context.
- A nonzero resolver result is fail-closed. Do not choose another planning document manually.
- Re-run the resolver after every branch switch before continuing version-specific work.
- On `INDEPENDENT_LEAF`, the returned WU document is the branch entry point; do not substitute the integration plan's current gate.
- On `INTEGRATION_CONTROL_PLANE`, enter through the returned version index and follow its machine-readable routing.
- The resolver selects context only. It does not grant implementation authorization or expand the selected WU's scope.

For branches covered by this resolver, this section supersedes any fixed historical version-specific planning paths elsewhere in this file. Historical `planning/0.3.6...` references remain relevant only to explicit 0.3.6 release-history or handoff work and must not override the resolved entry document.

## Canonical Project Authority runtime

For Project Authority work, use the canonical shipped Support Feature machine surface only:

1. select current support policies through canonical `docs/Support_policy/policy/index.yaml` when working from repository source, or the shipped `ptsip/support/policy/index.yaml` projection when installed;
2. resolve authority contracts through the corresponding Support `registries/` boundary;
3. resolve Authority Role effects through the corresponding Support `registries/` boundary;
4. resolve repository/subject applicability through the corresponding Support `registries/` boundary;
5. use `src/ptsip/governance/` for fresh eligibility, projection, and preauthorized transition evaluation.

Design-time `planning/**` artifacts are not runtime authority and must not be used to infer missing machine semantics. Subject matching is exact first and may relax only through an explicitly registered machine relationship. AI confidence, prose similarity, and unregistered aliases cannot create authority.

Completed generated review artifacts are retained through Git history rather than the active planning surface once their canonical result has been materialized and machine validation exists.

## Context loading after policy resolution

Do not unconditionally preload repository-wide natural-language context.

Normal entry order:

1. run the Policy Resolver for the exact task scope and operation;
2. load only the returned canonical MPD rule sections;
3. when the operation is planning/version-specific, run the branch-aware Planning Entry Resolver and enter through its exact returned document;
4. load Specification, Project Profile, evidence, history, release notes, or other prose only when the resolved task actually requires them.

Repository state and memory use the provider-neutral Context Plane under `.ptsip/context/`. The single semantic write target is `.ptsip/context/source/context.source.json`; `context.json`, `context.jsonl`, and `context.schema.json` are deterministic generated projections and are never independent authority.
Choose `context.json` or `context.jsonl` according to the consuming agent's supported machine-input shape. `context.schema.json` is their shared machine contract. No projection format is privileged by provider. Do not edit generated projections directly.
Use `python -m developer.automation.context_projection write --input <source.json>` for a single semantic write, `python -m developer.automation.context_projection sync` after an authorized source edit, and `python -m developer.automation.context_projection check` to fail closed on source-binding, byte, schema, or semantic-equivalence drift. Historical memory is targeted context; do not replay the complete memory set unless an explicitly resolved operation requires full-history analysis.
Planning documents and context memory records are operational or historical context. Normative claims come from the applicable bound Specification and canonical machine-readable contracts.

## License Authority entry discipline

`License-Authority/` is a closed License Authority Boundary and is not part of the default coding-agent read set. Default behavior is `DO_NOT_ENTER`.

Entry is permitted only for `EXPLICIT_USER_REQUEST`, `LICENSE_AUTHORITY_PATH_CHANGED`, `LICENSE_PROJECTION_MISMATCH`, or `REPORTED_LICENSE_PROBLEM`.

Normal Tool releases, normal Specification validation, Tool version changes, documentation mentions, keyword detection, and AI semantic guesses do not authorize License Authority entry.

When entry is permitted:

1. evaluate the trigger without reading License content;
2. read only `License-Authority/license-authority.yaml`;
3. resolve the requested operation from its closed operation map;
4. read only documents declared for that operation;
5. read external projections only when that operation declares them;
6. fail closed on unknown operations, undeclared files, or paths outside `License-Authority/`.

Files outside `License-Authority/` may mention or describe licensing, but those references are non-authoritative and are not incorporated into the License by reference. Do not scan the repository for license-like prose to determine License Authority.

## Repository-state discipline

- Re-read the remote target branch HEAD immediately before every GitHub write, merge, release preparation, or exact-SHA evidence claim.
- Preserve maintainer commits and never force-update `main`.
- Do not claim tests, builds, releases, tags, or publication succeeded without evidence for the exact relevant SHA.
- Documentation descendants after a successful verification run record results; they do not replace the exact source verification authority.
- Do not enter future Tool `0.3.6.1` implementation merely because its planning documents exist.
- Historical release notes, ADRs, and completed WU evidence are not rewritten to make current-version wording uniform.

## Tool 0.3.6 completion state

Tool `0.3.6` development work is complete. Current ordered state:

```text
WU-00  0.3.6-draft normative baseline                     COMPLETE
WU-01  lifecycle ontology/boundary rules                   COMPLETE
WU-02  roles + typed relationships + associated artifacts  COMPLETE
WU-03  canonical Responsibility Map v2 activation          COMPLETE
WU-04  template/materialization/effective-map pipeline     COMPLETE / EXACT-SHA VERIFIED
WU-05  repository dogfood / self-evaluation                COMPLETE / DOGFOOD REVIEWED
WU-06  full regression/package/distribution verification   COMPLETE / EXACT-SHA VERIFIED
WU-07  final Specification freeze/release preparation      COMPLETE / EXACT-SHA VERIFIED
```

WU-07 used **Strategy B — Release Contract Strengthening**.

Exact WU-07 verification authority:

```text
source SHA:       452d0f8b0c78bdebb180ceb2b9994485f59eb43a
workflow run/job: 32640319047 / 97196299107
Python:           3.14.6
pytest:           331 passed / 0 failed
Specification:    0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea
profile coverage: unassigned_count=0
Product Artifact: PASS / exact snapshot binding
PTSIP-PKG-001:    0 definite violations
wheel/VPMS smoke: PASS
commit status:    self-hosted/tooling-test = success
```

The exact WU-07 entry baseline remains:

```text
8b2c0819e10b58902a780a094a0f52c603c39fba
```

The completion/verification authority is the later exact candidate `452d0f8...`; closure/documentation commits after it do not replace that verification authority.

The next release boundary is exact-main handoff:

```text
approved Tool 0.3.6 state -> main
    -> fresh exact main SHA
    -> tooling-test.yml on that exact SHA
    -> require self-hosted/tooling-test success
    -> release.yml from the same current main SHA
    -> release contract PASS
    -> draft GitHub Release targeting the same SHA
    -> maintainer publication
    -> tooling-release.yml publication build/verification
    -> PyPI Trusted Publishing
```

Do not describe Tool `0.3.6` as published until that publication boundary actually succeeds.

## Current Specification binding

Tool `0.3.6` is bound to:

```text
Specification 0.3.6-draft
SPEC_REVISION d6995ed232e845b88d8235b851e80ab54b7804ea
```

A new immutable Specification revision is required only for a genuine normative change. Workflow, test, planning, status, release-note, or other documentation-only changes do not move `SPEC_REVISION` by themselves.

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

`TOOLCHAIN` is historical Tool `0.3.5` migration input only. It must not be emitted or preserved as a canonical Tool `0.3.6` alias.

Classification is determined from governing lifecycle obligation, not file type, path, framework, language, executable status, workflow provider, compilation behavior, test status, majority of files/jobs/steps, runtime duration, invocation frequency, or confidence score.

Important boundaries:

- Product-specific verification may be `PRODUCT`.
- Reusable verification/test SDK/framework/harness infrastructure may be `DEVELOPMENT_TOOLING`.
- Local/intermediate development build support is normally `DEVELOPMENT_TOOLING`.
- Authoritative release-unit assembly/signing/packaging/publication/deployment-to-destination is normally `DELIVERY`.
- `DELIVERY` ends at delivery handoff; ongoing health/recovery/reconciliation/maintenance is `OPERATIONS`.
- `NEUTRAL_CONTRACT` requires non-executable, non-owning, lifecycle-independent contract responsibility.
- Material mixed-lifecycle responsibilities should split when independently governable; do not choose a majority lifecycle.
- Material unresolved ownership fails closed.

## Responsibility Map v2 axes

Keep these distinct:

```text
classification
    = primary lifecycle ownership

roles
    = coarse responsibility characteristics

relationships
    = project-owned typed directed semantics

source_mode / derived origin
    = declaration/materialization provenance

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

Do not derive project-owned relationships automatically from observed evidence. Evidence can support a proposal; explicit project declaration remains authority.

## Responsibility Map declaration modes

Canonical source modes are:

```text
explicit
template
hybrid
```

Template selection is explicit and immutable-revision-bound. Never select a template from repository layout, language, framework, manifest presence, or confidence.

Materialization is deterministic and non-authoritative. It must not infer lifecycle ownership, repair architecture, create project intent, or silently rewrite the Source Project Profile.

All source modes resolve to a validated Canonical Effective Responsibility Map for downstream Tool behavior. Preserve source declaration and materialized provenance separately.

## Adoption and migration discipline

Canonical Tool `0.3.6` Project Profiles use `classification` as primary lifecycle ownership authority. A second canonical `lifecycle_owner` field must not compete with it.

When explicit adoption/resolution records architecture facts, preserve applicable facts such as:

```text
classification
roles
purpose
shipped
runtime_required
executable
associated artifacts
typed relationships
explicit release/compatibility metadata
```

Legacy Tool `0.3.5` `TOOLCHAIN`, `lifecycle_owner`, old boundary roots, consumers, analysis inputs, or untyped policy edges are migration evidence only. Do not blindly convert them into Tool `0.3.6` authority.

Evidence-backed Tool `0.3.5 -> 0.3.6` assisted migration is owned by Tool `0.3.6.1`. Tool `0.3.6` release closure does not authorize implementing that continuation.

## Decision Authority / Project Profile / evidence

Keep these responsibilities separate:

```text
Specification
    -> normative rules

Decision Authority
    -> which explicit coordinated architecture answer won

Project Profile / Responsibility Map
    -> durable project-owned architecture declaration

Observed evidence
    -> what repository/artifacts actually do

Conformance Evaluation
    -> whether declaration + evidence satisfy applicable rules
```

A Decision Authority does not replace the selected Project Profile and does not prove conformance.

For GitHub coordination, the Reference Tool uses:

```text
refs/heads/ptsip-policy
```

Preserve stable decision identity, first-valid-resolution-wins, stale-writer-safe conditional mutation, read-side authority freshness, deterministic reconciliation, fail-closed behavior, and global/local state separation.

A complete local declaration is not sufficient reason to skip relevant distributed authority reads. Semantic equivalence is architecture meaning, not YAML formatting.

## Read-only default and mutation safety

Inspection and Pilot behavior are read-only by default. Tool-owned caches, reports, and local decision databases stay outside the Consumer Repository unless explicitly directed otherwise.

Prepared profile writes must reject stale repository/profile state. Do not combine evidence from different revisions into one stable claim.

## Product Artifact boundary

Artifact owner and producer are different concepts. Development Tooling or Delivery may build a Product Artifact, but Product distribution contents still have to satisfy the Product lifecycle boundary.

Release verification must inspect actual built artifacts, not packaging configuration as proof. Preserve snapshot-bound Product Artifact evidence and fail closed on definite `PTSIP-PKG-001` violations.

## VPMS boundary

PTSIP asks who owns a responsibility across its lifecycle. VPMS asks why a Verification Case exists and what it protects.

PTSIP classification and VPMS Verification Purpose remain independent. PTSIP core must not depend on VPMS. VPMS consumes only a narrow read-only projection of validated effective PTSIP metadata.

Current VPMS compatibility vocabulary may retain `PRODUCT | TOOLCHAIN`; VPMS `TOOLCHAIN` is not a Tool `0.3.6` PTSIP classification.

VPMS PASS != PTSIP CONFORMANT, and PTSIP CONFORMANT != functional verification PASS.

## Conformance behavior

Completed Consumer Repository outcomes are only:

```text
CONFORMANT
NON_CONFORMANT
INCOMPLETE
```

Do not equate zero findings with conformance. Blocking evidence gaps remain `INCOMPLETE` unless a definite mandatory violation already establishes `NON_CONFORMANT`.

Do not let project-local policy weaken universal PTSIP requirements.

## Release and CI resource policy

Primary regression/release verification uses GitHub-hosted `ubuntu-latest` execution with Python 3.14 provisioned through `actions/setup-python`.

Preserve the existing exact-SHA checkout/status model. Do not reintroduce self-hosted runner assumptions, Windows-only Python launcher requirements, or runner-specific status-context names.

Do not create a parallel workflow where an existing release/test workflow can be narrowly maintained.

## Instruction priority

Use this order:

1. bound canonical Specification and normative companion assets from the same immutable revision;
2. relevant Decision Authority winner when distributed coordination applies;
3. repository Project Profile / Responsibility Map;
4. observed repository/dependency/artifact evidence;
5. imported external evidence with provenance;
6. project ADR/history;
7. this repository-operational contract;
8. informal examples.

This priority does not make Decision Authority a conformance oracle. Authority governs which explicit answer won; observed evidence still governs what the repository actually does.
