# Developer automation Go migration: local PC checkpoint

This is a non-authoritative execution handoff. Canonical policies, contracts,
registered state owners and exact machine resolution remain authoritative.
This document does not complete a work unit, approve a release, retire Support
policy, or turn the migration counters into full-equivalence evidence.

## Resume point

- Branch: `dev/0.3.8a4`.
- Implementation/test base: `6a1e814b82f8f3cc4ad4d8de25b330dfcbb3ce2e`.
- The commit containing this file saves the current implementation and evidence.
  Resolve its identity with `git log -1 --format=%H -- developer/verification/evidence/HANDOFF-GO-AUTOMATION-20261006-PC.md`.
- The user explicitly authorized committing and pushing this checkpoint.
- The user selected `policy/binding`, `policy/lifecycle`, `planning`, Agent and
  IWP responsibilities. While a canonical domain path is unresolved, the user
  explicitly directed temporary implementation under
  `developer/automation/internal/machine`. Preserve already selected domain
  roots; do not interpret this temporary location as a new canonical owner.
- The current branch has no active exact planning entry. Do not infer an old WU
  or change its completion state merely to route the migration.

## Implemented in this checkpoint

- Policy-Plan registry storage, CAS, relation resolution, materialization and
  movement tracking now live under `policy/binding`. Plan-file identity parsing
  lives under `planning`; mixed-package entrypoints delegate to their owners.
- Root relation keys include the sorted exact `policy_sections` set. Disjoint
  sections may reference the same logical plan; duplicates and unknown sections
  fail closed. The live 19-binding registry validates as `CURRENT` without repair.
- Policy task-context joining is native Go. It checks exact scope/operation,
  runtime branch, planning/rule/test references and delegated native selector
  validation. Runtime task registration remains absent when the canonical
  resolver contract contains a null `task_context_ref`.
- Markdown cleanup uses the exact Root GOV owner, native Go commands and
  registered repeated scopes. Simulation can report `BLOCKED` without mutation;
  its nonfatal transport status is declared in the command contract. Actual apply
  remains gated. The original Python cleanup implementation was retired.
- `current-dependency-gate` dispatches to the existing native implementation.
  Its actual repository result remains `FAIL` for two `ADR-0021` schema references.
- `developer/tests` is one Go module, including the existing Root Family package.
  Seven Python test files were replaced by native Go coverage: binding schema,
  policy version, stage/extension finalization, Markdown cleanup, workflow routing
  and H3 hook activation/delegation.
- CI, release verification, pre-commit and both startup scripts invoke Go
  Developer automation. Go setup precedes those CI consumers.
- PP release assets compare committed blob identities through Git's configured
  clean conversion. CRLF checkout bytes are accepted when their committed blob
  is identical; actual content edits are still rejected.
- The Branch Control package move had removed a shared Root-section adapter and
  left private-package tests calling retired symbols. The adapter was restored in
  the Policy bridge and those tests now call the native Branch package externally.

## Verification and evidence identity

The full runs below tested the working-tree/index snapshot on the stated base,
not a later checkpoint commit. The handoff and copied logs were added afterward.
No exact checkpoint-SHA remote CI success is claimed.

- Automation Go: **67 top-level tests + 78 subtests passed**, zero failed/skipped.
- Developer Go: **50 top-level tests + 156 subtests passed**, zero failed/skipped.
- Both Go module `vet` checks passed.
- Policy validation, resolver validation, Test Mode Registry (15 modes), workflow
  YAML parsing, binding validation and Git diff checks passed.
- Remaining Developer Python: **408 passed / 34 failed / 2 skipped**. Every failed
  node ID reproduced in an isolated checkout of the base commit; failure-ID delta
  is zero. Failures concern historical family/binding fixtures and Support/VPMS
  provenance. Do not change approval fingerprints or weaken guards to hide them.
- During unstaged deletion, repository snapshot collection reported missing
  tracked files and `INVALIDATED`. Only the ten task-owned deletion paths were
  staged to make the snapshot complete; the affected self-profile test then passed.
- Final source-manifest verification found no drift in the 59 implementation
  change paths. Its SHA-256 is
  `3e82280e771c6c0b4034a90b168c2a8f5551a41a929b57ce5a56855cbdddcd64`.

Portable [checkpoint metadata](go-automation-pc-20261006/checkpoint.json) and the
[execution evidence archive](go-automation-pc-20261006/execution-evidence.zip)
are saved beside this document. The ZIP contains `final-source-manifest.json`,
`implementation-verification-summary.json`, both Go JSONL logs, the remaining
Python log/JUnit XML and the baseline failure-recheck log/JUnit XML. Every entry
was verified against its original SHA-256. Archiving preserves raw stdout and
failure-message bytes across Git line-ending conversion without editing logs.
Original local paths remain provenance, not required paths on another PC.

## Remaining work

- Migration remains `IN_PROGRESS`: **51 Python automation files**, **38 Python
  test files**, **8 registered Go implementations**, **46 registered pending
  modules**. These are registry/projection counts, not a verified completion ratio.
- Continue native coverage and caller cutover for policy/binding/lifecycle,
  planning, Agent/IWP/context, PP and audit/verification consumers. Existing Go
  source is broader than the registered migration count; verify each capability
  before reconciling the inventory or removing its Python source.
- Resolve the 34 existing Python regression failures through their actual
  authority/fixture/provenance boundaries. Preserve immutable receipts.
- Retire remaining Python implementations only after meaningful Go equivalence
  and real consumer cutover. Do not relocate Python business logic or add Go
  wrappers that execute Developer Python automation.
- Developer legacy removal is not ready. Support-plane retirement does not
  inherit authorization from this migration.
- Verify CI for the actual pushed checkpoint SHA separately; functional local
  PASS does not establish release/WU acceptance.

## Destination PC entry

Read `AGENTS.md`, inspect Git state and run canonical resolution before broader
context. This PC used Go **1.26.5 / windows amd64** and the checkout `.venv`
Python **3.14.6** for remaining Product/legacy regression comparisons.

```powershell
git status --short --branch
git rev-parse HEAD
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev policy-resolver resolve --scope developer/automation --operation MODIFY
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev policy-resolver resolve --scope developer/tests --operation MODIFY
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev policy-validator validate
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev automation-migration inspect
```

When required by further changes, rerun:

```powershell
go -C developer/automation test -tags grammar_subset,grammar_subset_python -count=1 -json ./...
go -C developer/tests test -tags grammar_subset,grammar_subset_python -count=1 -json ./...
go -C developer/automation vet -tags grammar_subset,grammar_subset_python ./...
go -C developer/tests vet -tags grammar_subset,grammar_subset_python ./...
```
