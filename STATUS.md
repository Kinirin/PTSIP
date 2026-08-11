# PTSIP Status

- Specification family: `0.2.0-draft`
- Latest canonical normative snapshot: `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`
- Current Tool/package source version: `0.3.3`
- Tool 0.2.3 historical bound specification revision: `14a0c2f54bb486de6a109979224f998b04fd04a3`
- Tool 0.3.x bound specification revision: `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`
- Specification identity model: draft family + immutable Git revision
- Maturity: Experimental
- Canonical repository: `kwaksinwoo01/PTSIP`
- External tooling model: Defined
- Reference Tool package name: `PTSIP`
- Latest verified PyPI publication: `0.3.1`
- Tool 0.2.3 publication policy: source-only migration; intentionally not published/tagged
- Tool 0.3.0 publication status: published to PyPI
- Tool 0.3.1 publication status: published as `tool-v0.3.1` and available from PyPI
- Tool 0.3.2 source status: merged source baseline; not published/tagged
- Tool 0.3.2 verification: GitHub Actions run `31365614038`, Python 3.14, `115 passed`, CLI/package checks successful
- Tool 0.3.3 source target: explicit `ptsip adopt`, profile-path symmetry, Local DecisionStore continuity, and GitHub-coordinated multi-environment decision authority
- Tool 0.3.3 publication status: migration source only; not published/tagged
- Supported Python metadata: Python 3.11–3.14
- Routine hosted Tool CI: Python 3.14 to conserve GitHub Actions usage; release-readiness compatibility is verified separately before publication
- Tool release namespace: `tool-v*`
- Reuse license: Apache License 2.0

## Tool 0.2.3 evidence-correctness baseline

Tool 0.2.3 implements the low-risk correctness work defined after the two-Pilot normative snapshot while preserving the deterministic Human Clarification capability introduced in Tool 0.2.2.

Implemented in that source-only migration:

- Python source decoding through Python encoding detection so UTF-8 BOM and valid source encoding declarations do not become false read failures;
- deterministic relative-import resolution when package/repository evidence identifies the target;
- dependency evidence-node scope separated from architectural classification using `PROJECT_COMPONENT`, `EXTERNAL_DEPENDENCY`, `PLATFORM`, and `UNRESOLVED_TARGET`;
- dependency provenance recorded separately from target scope;
- dynamic Python imports represented as `LOADS`, retaining `DYNAMIC` resolution when the target is not statically known;
- deterministic external-dependency recognition from direct Python dependency declarations where package/import naming evidence matches;
- GitHub Actions local-script resolution from the effective workflow/job/step `working-directory`;
- arbitrary `run:` commands without local-script evidence no longer become synthetic unresolved `INVOKES` edges;
- declared dependency evaluator state reports `RAN` versus `BLOCKED` with a reason and does not use an empty finding list to imply that evaluation ran;
- embedded profile schema and registry synchronized to the Tool-bound immutable Specification snapshot;
- Project Profile validation aligned with `boundaries XOR components`, retired mandatory-rule waiver semantics, exact bound revision checks, and component-policy reference validation.

Tool 0.2.3 is intentionally not released to PyPI. It is the source baseline for Tool 0.3.0.

## Human Clarification regression boundary

The following Tool 0.2.2 behavior remains required and regression-tested:

- `ptsip clarify`;
- deterministic clarification with zero LLM/model API calls;
- English/Korean fixed questions via `--lang` / `PTSIP_LANG`;
- Consumer Repository GitHub identity from Git `origin`;
- explicit `--publish github-issue` with `--repo owner/repository` override as a manual/offline fallback;
- duplicate-publication state under external `PTSIP_HOME/clarifications` for that manual transport;
- read-only clarification analysis unless publication or an explicit Tool 0.3.1+ resolution/adoption operation is requested.

Tool 0.3.1 extends this boundary with structured Issue answers and explicit user-authorized chat resolution. Free-form Issue prose is still not interpreted by an LLM. Tool 0.3.3 additionally makes explicit profile selection symmetric across clarification/gate and declaration workflows.

## Tool 0.3.0 — conformance capability expansion

Tool 0.3.0 implements:

- `ptsip conform` with `CONFORMANT`, `NON_CONFORMANT`, and `INCOMPLETE` outcomes plus distinct exit codes;
- stable deterministic `ptsip-diagnostic/v1`-shaped diagnostics and final diagnostic/coverage contract audit;
- rule-relative evidence coverage gates;
- explicit `ptsip-artifact-evidence/v1` ingestion and Product packaging-isolation evaluation;
- independent build-resolution evaluation from declared component manifests;
- bounded lifecycle/release and compatibility ownership evidence;
- JavaScript/TypeScript source and npm manifest adapters;
- Go source/module dependency evidence;
- `.csproj` plus source-level .NET namespace dependency evidence;
- project-specific component dependency policy findings separated from universal PTSIP diagnostics;
- constrained agent-decision ingestion as review evidence that cannot overwrite profile declarations;
- revision-bound external dependency evidence import with producer/subject/provenance and imported-file SHA-256 preservation;
- one composite dependency evidence graph used by `inspect`, `pilot`, `clarify`, and `conform`;
- `ptsip pilot` remaining `NOT_EVALUATED` for conformance while `ptsip conform` owns strict outcome evaluation.

Tool 0.3.0 deliberately preserves uncertainty. Invalid or stale review evidence, contradictory evidence, unresolved Product dependency targets, incomplete Product Artifact evidence, ambiguous build/lifecycle evidence, unstable snapshots, or internal diagnostic-contract failures block `CONFORMANT` rather than being silently ignored.

Final post-rebind release-readiness verification completed in GitHub Actions run `31334084470`: Python 3.11, 3.12, 3.13, and 3.14 each passed the full pytest suite plus Tool identity, exact Specification revision, and `ptsip conform --help` checks; the Python 3.14 package job verified `tool-v0.3.0` version mapping, built wheel/sdist distributions, passed `twine check`, installed the built wheel, and passed built-wheel Tool identity/spec/conform smoke checks.

## Tool 0.3.1 — coding-agent decision control plane

Tool 0.3.1 adds a Tool-level, non-normative orchestration layer for unresolved human architecture decisions without changing the bound PTSIP Specification snapshot:

- `ptsip gate` is invoked only when an active coding-agent task needs a missing component decision;
- no timer, scheduled reminder, or background polling is part of the workflow;
- unresolved gates can use a GitHub clarification Issue through a GitHub App-backed hosted control plane;
- a user may resolve the decision in the active coding-agent chat through explicit `ptsip resolve` or asynchronously through a structured GitHub Issue answer;
- the authoritative store uses compare-and-set semantics so the first valid resolution wins;
- late Issue/chat answers never replace an already resolved decision;
- Issue answers require repository write/maintain/admin authority and the fixed `ptsip-clarification-answer/v1` structure;
- fixed deterministic resolution rules reject contradictory Product/Toolchain/Neutral Contract facts instead of interpreting prose;
- chat-originated profile writes validate a temporary projected profile before replacing the selected local profile;
- Issue-originated profile writes remain bound to the exact recorded branch revision and use a non-force Git ref update so stale decisions are not silently applied;
- the Project Profile remains the architecture declaration;
- the existing explicit `ptsip clarify --publish github-issue` path remains available as a manual/offline fallback.

Pre-merge Tool 0.3.1 verification completed in GitHub Actions run `31354690223`: Python 3.11 and 3.14 both passed the complete pytest suite and Tool/new-CLI smoke checks; Python 3.14 reported `106 passed` and successfully built `ptsip-0.3.1` wheel and sdist artifacts.

See [`reference/DECISION-CONTROL-PLANE.md`](reference/DECISION-CONTROL-PLANE.md) for the current Tool-level workflow contract.

## Tool 0.3.2 — topology migration and profile-path symmetry baseline

Tool 0.3.2 source adds Tool-level refactoring support without changing normative PTSIP classification or conformance semantics:

- `ptsip topology` produces a dry-run plan by default and requires explicit `--apply` for repository changes;
- component-root moves rewrite path declarations while preserving every declared architecture classification;
- component-based profiles require `--component`, preventing root selection from becoming an implicit ownership decision;
- tracked path-text references are combined with observed PTSIP dependency edges so imports/links/loads/invocations can be reported even when source text does not contain the literal old repository path;
- profile/import/build/CI/documentation references are reported by impact category, and dependency-adapter scan issues are surfaced for review instead of being hidden;
- non-profile references are not blindly rewritten; the apply result reports any remaining review work;
- Git-backed apply requires a clean working tree/index and uses `git mv` so tracked evidence follows the new root;
- the projected profile is validated after the move before the migration is accepted;
- `ptsip resolve --profile <path>` respects project-owned profile placement and leaves repository-root `ptsip.yaml` as the default only when no explicit path is supplied;
- the bound Specification remains `0.2.0-draft` revision `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`.

Final pre-merge verification completed in GitHub Actions run `31365614038`: Python 3.14 passed the complete suite with `115 passed`, Tool/spec/new-CLI smoke checks succeeded, and both `ptsip-0.3.2` wheel and sdist passed `twine check`.

See [`releasenote/0.3.2.md`](releasenote/0.3.2.md) for the migration-source scope.

## Tool 0.3.3 — explicit adoption and multi-environment authority

Tool 0.3.3 migration source extends the Tool orchestration layer while keeping the same immutable Specification binding.

Implemented source scope:

- new `ptsip adopt` command with deterministic candidate selection, explicit architecture facts, read-only dry-run by default, explicit `--apply`, stale-evidence protection, profile prevalidation, concurrent-profile protection, and post-write validation;
- adoption reuses `DecisionAnswer`, `validate_answer()`, and the existing profile projection/write path instead of adding a second classification model;
- `ptsip clarify --profile` and `ptsip gate --profile` complete explicit profile-location symmetry with `adopt`, `resolve`, `validate`, and `conform`;
- non-GitHub repositories retain the embedded Local DecisionStore for active coding-agent decision gates;
- GitHub repositories use a GitHub-coordinated authority by default for unresolved architecture decisions, with explicit `--coordination local|github` overrides and the hosted HTTP `--control-plane` override retained;
- the GitHub authority is an automatically bootstrapped `refs/heads/ptsip-policy` Git ref containing JSON authority/decision records, not a shared SQLite database;
- GitHub authority writes use exact-parent commits and non-force ref updates so stale writers cannot replace a newer authority HEAD;
- global decision IDs are derived from repository identity plus normalized component include scope, avoiding separate winners when clones temporarily generate different local clarification IDs;
- resolved GitHub authority decisions can be reconciled into a stale clone's selected `ptsip.yaml` at `ptsip gate` time, without continuous background polling;
- GitHub coordination accepts `GH_TOKEN`/`GITHUB_TOKEN` for cloud environments or authenticated `gh` CLI credentials for interactive use;
- selected GitHub coordination fails closed if network/auth/permissions are unavailable instead of silently creating a separate Local DecisionStore winner;
- an existing non-PTSIP `ptsip-policy` branch is refused unless its authority manifest matches the PTSIP authority format/repository/ref identity;
- local `control-plane.sqlite3` remains outside the Consumer Repository and is not Git-shared;
- repository-root `ptsip.yaml` remains the default project-owned declaration and is intended to be committed, not ignored;
- the `runtime_required` decision fact is retained in decision authority data but is not added to the bound Project Profile schema, preserving the immutable Specification revision contract.

Tool 0.3.3 does not change the three PTSIP classifications, conformance outcomes, universal rule semantics, or the bound Specification revision. Global `gate --json` exception-envelope redesign and aggregate gate-precondition diagnostics remain separate follow-up work.

Final Tool 0.3.3 release-boundary verification is recorded after the migration branch passes its final Python 3.14 test/package job. No release tag, GitHub Release, or PyPI publication is implied by the source migration.

See [`releasenote/0.3.3.md`](releasenote/0.3.3.md) and [`reference/DECISION-CONTROL-PLANE.md`](reference/DECISION-CONTROL-PLANE.md).

## Release blockers for PTSIP Specification 1.0

- stabilize core rule and profile semantics through multiple real Pilots;
- exercise automated validator rules against real dependency and packaging boundaries;
- validate the constrained Agent Contract and deterministic Human Clarification against real ownership questions;
- exercise rule-relative evidence coverage and Product Artifact evidence;
- publish tagged stable specification releases;
- exercise Tool 0.3.1 or later against real Consumer Repositories with component declarations, artifact evidence, review/import evidence, on-demand human decision gates, topology migrations, multi-environment decision coordination, and repeatable conformance evaluation.
