# PTSIP Status

- Specification family: `0.2.0-draft`
- Latest canonical normative snapshot: `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`
- Current Tool/package source version: `0.3.1`
- Tool 0.2.3 historical bound specification revision: `14a0c2f54bb486de6a109979224f998b04fd04a3`
- Tool 0.3.x bound specification revision: `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`
- Specification identity model: draft family + immutable Git revision
- Maturity: Experimental
- Canonical repository: `kwaksinwoo01/ptsip`
- External tooling model: Defined
- Reference Tool package name: `ptsip`
- Latest verified PyPI publication: `0.2.0`
- Tool 0.2.3 publication policy: source-only migration; intentionally not published/tagged
- Tool 0.3.0 publication status: conformance-capability source merged; not yet published/tagged
- Tool 0.3.1 source target: on-demand coding-agent decision gate plus GitHub App/Webhook decision control plane
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
- read-only clarification analysis unless publication or an explicit Tool 0.3.1 resolution operation is requested.

Tool 0.3.1 extends this boundary with structured Issue answers and explicit user-authorized chat resolution. Free-form Issue prose is still not interpreted by an LLM.

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
- unresolved gates create/reuse a GitHub clarification Issue through a GitHub App-backed control plane;
- a user may resolve the decision in the active coding-agent chat through explicit `ptsip resolve` or asynchronously through a structured GitHub Issue answer;
- the authoritative store uses compare-and-set semantics so the first valid resolution wins;
- late Issue/chat answers never replace an already resolved decision;
- Issue answers require repository write/maintain/admin authority and the fixed `ptsip-clarification-answer/v1` structure;
- fixed deterministic resolution rules reject contradictory Product/Toolchain/Neutral Contract facts instead of interpreting prose;
- chat-originated profile writes validate a temporary projected profile before replacing `ptsip.yaml`;
- Issue-originated profile writes are bound to the exact recorded branch revision and use a non-force Git ref update so stale decisions are not silently applied;
- the GitHub Issue is an asynchronous interaction surface; the control-plane decision state is authoritative for workflow state, while `ptsip.yaml` remains the architecture declaration;
- the existing explicit `ptsip clarify --publish github-issue` path remains available as a manual/offline fallback.

See [`reference/DECISION-CONTROL-PLANE.md`](reference/DECISION-CONTROL-PLANE.md) for the Tool-level workflow contract.

## Release blockers for PTSIP Specification 1.0

- stabilize core rule and profile semantics through multiple real Pilots;
- exercise automated validator rules against real dependency and packaging boundaries;
- validate the constrained Agent Contract and deterministic Human Clarification against real ownership questions;
- exercise rule-relative evidence coverage and Product Artifact evidence;
- publish tagged stable specification releases;
- exercise Tool 0.3.1 or later against real Consumer Repositories with component declarations, artifact evidence, review/import evidence, on-demand human decision gates, and repeatable conformance evaluation.
