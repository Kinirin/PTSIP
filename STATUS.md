# PTSIP Status

- Active Specification family: `0.2.0-draft`
- Latest canonical normative snapshot: `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`
- Proposed Specification family design record: `0.3.3-draft` (not yet an active Tool binding)
- Current Tool/package source version: `0.3.4`
- Tool 0.2.3 historical bound specification revision: `14a0c2f54bb486de6a109979224f998b04fd04a3`
- Current Tool 0.3.4 bound specification revision: `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`
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
- Tool 0.3.3 completed scope: explicit `ptsip adopt`, profile-path symmetry, deterministic validation/profile-projection reuse, and Local DecisionStore continuity
- Tool 0.3.3 publication policy: permanently source-only; never create `tool-v0.3.3`, a GitHub Release, or PyPI `0.3.3`
- Tool 0.3.4 completed scope: GitHub-coordinated repository-global authority freshness, reconciliation, conflict semantics, clone-local application receipts, and fail-closed distributed consistency
- Tool 0.3.4 implementation merge: `555c528593f700a348d8da84545a62ce61291cae` (PR `#24`)
- Tool 0.3.4 verification completion: `8cd0ddf16dc9b56f27f694138a37caae1c49bb4f` (PR `#25`)
- Tool 0.3.4 verification: GitHub Actions run `31471025526`, Python 3.14.6, `134 passed`, build/twine/installed-wheel checks successful
- Tool 0.3.4 publication status: verified publication candidate; no tag, GitHub Release, or PyPI `0.3.4` publication created
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

## Tool 0.3.3 — explicit adoption source-only migration

Tool `0.3.3` completes the explicit project-adoption workstream while remaining bound to the active immutable `0.2.0-draft` Specification revision.

Completed Tool `0.3.3` scope:

- new `ptsip adopt` command with deterministic candidate selection, explicit architecture facts, read-only dry-run by default, explicit `--apply`, stale-evidence protection, profile prevalidation, concurrent-profile protection, and post-write validation;
- adoption reuses `DecisionAnswer`, `validate_answer()`, and the existing profile projection/write path instead of adding a second classification model;
- `ptsip clarify --profile` and `ptsip gate --profile` complete explicit profile-location symmetry with `adopt`, `resolve`, `validate`, and `conform`;
- Local DecisionStore continuity for local-only/explicit local coding-agent decision workflows;
- explicit hosted HTTP Control Plane compatibility remains available;
- local `control-plane.sqlite3` remains outside the Consumer Repository and is not Git-shared;
- repository-root `ptsip.yaml` remains the default project-owned declaration and is intended to be committed, not ignored;
- the `runtime_required` decision fact is not silently added to the bound predecessor Project Profile schema.

The `0.3.3` source tree also contains an early GitHub-coordinated authority prototype from a subsequently withdrawn Tool `0.3.3` scope amendment. The prototype includes CAS/write-side and basic stale-missing-profile behavior, but its full repository-global contract is **not** considered completed Tool `0.3.3` behavior.

The authority-consistency work moved to Tool `0.3.4` and is now completed there.

Pre-merge Tool `0.3.3` source verification completed in GitHub Actions run `31465581071` on Python 3.14 with `128 passed`, CLI/package checks, build, and `twine check` success.

Tool `0.3.3` is permanently source-only. The project will never create `tool-v0.3.3`, a GitHub Release for Tool `0.3.3`, or PyPI `PTSIP==0.3.3`.

See [`planning/PTSIP-TOOL-0.3.3-EXPLICIT-PROJECT-ADOPTION-PLAN.md`](planning/PTSIP-TOOL-0.3.3-EXPLICIT-PROJECT-ADOPTION-PLAN.md) and [`releasenote/0.3.3.md`](releasenote/0.3.3.md).

## Tool 0.3.4 — GitHub-coordinated distributed authority completion

Tool `0.3.4` completes the GitHub-coordinated repository-global authority contract begun experimentally in the `0.3.3` source tree.

Completed behavior includes:

- authority freshness checks at relevant `ptsip gate` boundaries even when the local Project Profile already contains a complete declaration;
- read-only authority lookup that does not bootstrap `refs/heads/ptsip-policy` or fabricate decision history merely to observe absence;
- deterministic local/remote reconciliation:
  - local missing + remote winner -> safe validated local projection;
  - local equivalent + remote winner -> `RESOLVED` / `CONSISTENT` without rewrite;
  - local conflicting + remote winner -> `AUTHORITY_PROFILE_CONFLICT` without silent overwrite;
  - complete local declaration + no remote record -> `NO_DECISION_REQUIRED` without fabricated authority state;
- preserved global `gdec-*` identity based on repository identity plus normalized include scope;
- preserved non-force Git ref CAS and first-valid-resolution-wins;
- selected GitHub coordination fails closed and does not silently create a Local winner;
- global decision resolution is separate from clone-local application receipts reported with `scope = LOCAL_PROJECTION`;
- coordinated `adopt`, `gate`, and `resolve` behavior while preserving explicit Local and hosted backends;
- Tool/package source identity `0.3.4` while remaining bound to `0.2.0-draft` revision `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`.

Implementation merged in PR `#24` as `555c528593f700a348d8da84545a62ce61291cae`.

A first merged verification run found only three stale tests that still expected Tool `0.3.3`; those expectations were corrected in PR `#25` as `8cd0ddf16dc9b56f27f694138a37caae1c49bb4f`.

Final release-boundary verification completed in GitHub Actions run `31471025526` on Python `3.14.6`:

- complete pytest suite: `134 passed`;
- Tool identity and exact Specification binding: successful;
- CLI smoke: successful;
- wheel/sdist build: successful;
- `twine check`: successful;
- built-wheel reinstall and installed-wheel Tool/spec/CLI smoke: successful.

Tool `0.3.4` is now a verified publication candidate. No `tool-v0.3.4` tag, GitHub Release, or PyPI `0.3.4` publication has been created; publication remains a separate explicit release decision.

See [`planning/PTSIP-TOOL-0.3.4-GITHUB-COORDINATED-AUTHORITY-PLAN.md`](planning/PTSIP-TOOL-0.3.4-GITHUB-COORDINATED-AUTHORITY-PLAN.md) and [`releasenote/0.3.4.md`](releasenote/0.3.4.md).

## Release blockers for PTSIP Specification 1.0

- stabilize core rule and profile semantics through multiple real Pilots;
- exercise automated validator rules against real dependency and packaging boundaries;
- validate the constrained Agent Contract and deterministic Human Clarification against real ownership questions;
- exercise rule-relative evidence coverage and Product Artifact evidence;
- publish tagged stable specification releases;
- exercise Tool 0.3.1 or later against real Consumer Repositories with component declarations, artifact evidence, review/import evidence, on-demand human decision gates, topology migrations, distributed decision coordination, and repeatable conformance evaluation.
