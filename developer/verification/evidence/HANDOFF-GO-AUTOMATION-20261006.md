# Developer automation Go migration handoff

This file is a non-authoritative execution handoff. Canonical policies, contracts,
catalogs and registered state owners win over this note. Do not infer authority
from it or copy its mutable state into AGENTS.md.

## Human-authorized scope

- Replace all existing implementation under `developer/automation` with Go.
- Preserve regression semantic coverage. Go wrappers must not execute or embed
  Python automation. Product runtime calls in regression tests are distinct
  verification capabilities; they do not implement the Go automation algorithm.
- Use exact Root policy IDs and sections. Do not rebuild old-ID interfaces or
  read Developer legacy policy bodies as runtime authority.
- Eventually remove `developer/policy/legacy` after responsibility coverage,
  consumer cutover and verification. Support-plane authority cannot inherit
  Developer authority.
- New root/src modules require language-neutral machine definitions, governed by
  `MPD-REAL-0004`; the Go mandate is scoped to Developer automation.
- Commit/push was authorized. The user requested this handoff because this PC
  suffers from lag and resource limits. Stop implementation here after saving.

## Branch and checkpoints

- Branch: `dev/0.3.8a4`.
- Base before the resumed work: `2e228e17de82f5d33584a59c827bb27aed47a976`.
- Native implementation integration checkpoint:
  `49030198d0e44a75500526c839ea8f6340b65f83`.
- The commit containing this handoff additionally saves the pending native
  Markdown cleanup implementation. Find the final handoff commit with Git;
  this document does not claim a hash for its own commit.

## Exact entry and initial checks on the destination

Read `AGENTS.md`. Use the canonical Go entry before broader context:

```powershell
git status --short
git branch --show-current
git rev-parse HEAD
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev policy-resolver resolve --scope developer/automation --operation MODIFY
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev policy-validator validate
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev automation-migration inspect
```

The installed toolchains on the source PC were Go 1.26.5 and Python 3.14.3.
Go dependencies are pinned in `developer/automation/go.mod` and `go.sum`.
The Python parser is pure Go `github.com/odvcencio/gotreesitter v0.55.1`.
Selective grammar build tags reduce memory and compilation time. Python source
selection uses strict complete parsing and rejects error/missing nodes; Go source
selection uses the standard Go AST parser. No compiler/Python parser subprocess
is required for these implementations.

## Implementations saved in the checkpoint

New `developer/automation/internal/machine` files contain native implementations
for lifecycle/responsibility gates, policy validation, catalog migration,
policy-plan consistency, pp.1.02 seeding, agent classification/materialization/
progressive entry/activation/integration, neutral context projection, Agent
Contract graph and operation checks, IWP packet/check/brief/context/failure/
execution, planning validators/finalizers/merge, Policy-Plan binding storage/CAS/
resolution/management, PP snapshot/delta/reconciliation/remote/release checks,
branch approval guard/GitHub API/recreation rollback, hook installation,
dependency gate, WU-02 lane validation, legacy decision retirement evaluation,
profile registry validation and VPMS contract/activation/provenance audits.

The previous five Go implementations remain: policy loading/resolution, Root
Family ID entry, repository state and planning entry. The package init/error-only
Python files need removal during caller cutover; their names are not independent
functional ports.

`markdown_cleanup.go` was written after checkpoint 4903019. It compiles, but its
operation is not yet admitted in the canonical command vocabulary and the
existing workflow still points to an old policy ID. It needs the direct Root
workflow reference and meaningful tests before being called complete.

## Metadata already converted

- `go-automation-cutover.v1.json` contains 92 registered command signatures.
- The policy, planning and agent `*_commands.json` files under automation are
  integration proposals, not independent authority. The admitted canonical
  command list owns invocation eligibility. Keep them synchronized or retire
  them after admission.
- Five old Policy-Plan bindings became 19 exact Root bindings with unit section
  lists. New binding IDs are `PPB-0006` through `PPB-0024`; original binding
  identities and mappings are preserved in `go-binding-root-cutover.json`.
- Binding schema now accepts 14 Root IDs and optional `policy_sections`.
- Developer authority-schema/role registry references and source-review queries
  were switched to exact Root owner sections. Source-review query count is 26.
- `go-support-audit-refs.v1.json` identifies class-local Support Root fields and
  state groups for read-only audits. It grants no Developer ownership of Support
  semantics.
- Policy validator on the working candidate returned `PASS` after these
  metadata changes. Legacy catalog rows remain audit metadata; the native
  validator does not use their policy bodies.

## Known integration gaps — do not mark full migration complete

1. All 54 Python automation files still exist. The migration inventory still
   reports the previous 5 Go implementations / 49 pending. Update it only after
   reviewing each public capability and test coverage; current counters are
   stale projections, not completion evidence.
2. CI, `.githooks`, startup scripts, fixtures and other callers still contain
   `python -m developer.automation...` and imports. Convert real consumers,
   preserve regression meaning and then remove original Python implementations.
   Do not relocate the old business logic to another directory to hide it.
3. `current-dependency-gate validate` is admitted in the command list, but its
   alias handler still needs registration to the existing native dependency
   implementation (`dependency-gate`).
4. Binding duplicate-relation validation currently keys only `(policy_ref,
   resolved_plan_id)`. Review exact `policy_sections` in the relation key; the
   19 Root bindings can legitimately share a Root container while referencing
   disjoint sections. Preserve cardinality instead of treating the container as
   the complete old policy.
5. Go resolver still rejects non-null `task_context_ref`. Port its exact context
   join from the existing implementation using `ValidateImplementationRef` and
   registered branch/planning/rule/test references. Current admitted context is
   null; do not invent a planning route for this branch. IWP fixture parity needs
   this implementation, and missing context must stay fail closed.
6. VPMS audit receipts retain immutable fingerprints. Existing test-path edits
   in commit `43ce79a` changed prior materialization bytes and caused six extra
   provenance failures. Do not overwrite approval fingerprints merely to pass.
   Several current audit target paths refer to Python source files or historical
   policy paths. Register exact successor/audit relationships when retiring them
   rather than silently ignoring preserved-file guards.
7. Some Agent Contract callable validation was changed from Python import to
   strict static declaration validation. Review equivalence and register the
   execution binding; static existence alone cannot prove every runtime import
   condition. Product runtime regression tests can verify relevant semantics.
8. The Developer legacy directories, original-ID catalogs/projections and
   physical-archive verification assumptions still need a separate admitted
   retirement. Support legacy policy retirement is not authorized implicitly.
9. Generic old numeric policy registration and SFP preflight were intentionally
   made Root-only/fail-closed; review callers and approved class-owned creation
   routes before removing old entrypoints.
10. WU-02 control-branch literals were preserved from the original implementation.
    Review canonical state ownership before treating them as mutable routing
    authority in new Go code.

## Verification actually performed

- Initial native Go whole package run executed about 95 seconds and had one
  failure, `TestPlanningNativeStageTextAndMutationRollback`. Its expected string
  incorrectly omitted the preserved `automatic_completion` block. The assertion
  was corrected to require that block and the unchanged sibling stage.
- That test's focused recheck passed (about 3.35 seconds).
- Policy validator candidate returned `PASS`; profile registry returned `PASS`.
- Go source compiles. A focused native policy version test passed after saving
  the Markdown cleanup file (about 0.29 seconds).
- Whole native suite must be rerun after final integration; no final complete
  suite PASS is claimed here. Use:

```powershell
go -C developer/automation test -tags grammar_subset,grammar_subset_python -count=1 -v ./...
go -C developer/automation vet -tags grammar_subset,grammar_subset_python ./...
go -C developer/tests/rootfamily test -count=1 -v ./...
```

- The previous complete Python run was `1363 passed / 75 failed / 2 skipped`.
  Nine relocation/ownership failures were fixed; a focused recheck left 66
  failures. Sixty IDs matched failures reproduced at the prior base, and six were
  the VPMS provenance cases above. This is not a final whole-suite PASS.
- Previous latest successful CI belonged to commit 2e228e1, run
  `https://github.com/Kinirin/PTSIP/actions/runs/37362453586`; static validation and
  Go boundary step passed, `test-build` was skipped. Do not attribute that CI to
  the new integration/handoff commit.

Machine records: `developer/policy/analysis/go-automation-verification.json`,
`go-binding-root-cutover.json`, `committed-test-path-amendment.json`.
Source-PC-only logs are in `C:/Users/rhkrt/AppData/Local/Temp/ptsip-go-*` and are
not assumed available on the destination. Do not rely on source-local caches,
virtualenv, binaries or temp paths as repository authority.

## Suggested continuation order

Rerun native tests and fix integration failures; complete Markdown cleanup and
task-context support; review schema/ref/coverage differences; convert real
command callers and registered execution bindings; provide Go regression
coverage for removed Python APIs; retire Python files; reconcile audit/projection
contracts and remove Developer legacy files; run required selected modes and
whole regression checks; then commit/push and verify CI for the exact new SHA.

The three subagents (policy_go, agent_go, planning_go) stopped on an account
usage-limit error. Their source files are saved, but their final end-to-end
completion claims were not accepted. User explicitly authorized parallel agents;
reuse that authorization if resources permit, with disjoint file ownership.
