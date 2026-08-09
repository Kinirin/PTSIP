# PTSIP Status

- Specification family: `0.2.0-draft`
- Latest canonical normative snapshot: `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`
- Current Tool/package source version on `main`: `0.2.3`
- Tool 0.2.3 historical bound specification revision: `14a0c2f54bb486de6a109979224f998b04fd04a3`
- Current Tool 0.3.0 migration branch source version: `0.3.0`
- Tool 0.3.0 bound specification revision: `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`
- Specification identity model: draft family + immutable Git revision
- Maturity: Experimental
- Canonical repository: `kwaksinwoo01/ptsip`
- External tooling model: Defined
- Reference Tool package name: `ptsip`
- Latest verified PyPI publication: `0.2.0`
- Tool 0.2.3 publication policy: source-only migration; intentionally not published/tagged
- Tool 0.3.0 publication status: code review gate passed; post-rebind release-readiness verification pending; not yet published/tagged; PR merge remains pending
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
- explicit `--publish github-issue` with `--repo owner/repository` override;
- duplicate-publication state under external `PTSIP_HOME/clarifications`;
- read-only clarification analysis unless publication is explicitly requested;
- no Issue-answer collection, free-form interpretation, or automatic classification.

## Tool 0.3.0 — conformance capability expansion

The current Tool 0.3.0 migration branch implements:

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

The earlier release-readiness run `31328764175` passed Python 3.11–3.14 plus package build/twine/wheel smoke before the final merge-gate remediation and Specification rebind. A final post-rebind release-readiness run is required before Tool 0.3.0 can be merged for release preparation. No Tool 0.3.0 tag, GitHub Release, or PyPI publication has been created yet.

## Release blockers for PTSIP Specification 1.0

- stabilize core rule and profile semantics through multiple real Pilots;
- exercise automated validator rules against real dependency and packaging boundaries;
- validate the constrained Agent Contract and deterministic Human Clarification against real ownership questions;
- exercise rule-relative evidence coverage and Product Artifact evidence;
- publish tagged stable specification releases;
- exercise Tool 0.3.0 or later against real Consumer Repositories with component declarations, artifact evidence, review/import evidence, and repeatable conformance evaluation.
