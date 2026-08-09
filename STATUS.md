# PTSIP Status

- Specification family: `0.2.0-draft`
- Latest canonical normative snapshot: `14a0c2f54bb486de6a109979224f998b04fd04a3`
- Current Tool/package source version on this migration branch: `0.2.3`
- Tool 0.2.3 bound specification revision: `14a0c2f54bb486de6a109979224f998b04fd04a3`
- Specification identity model: draft family + immutable Git revision
- Maturity: Experimental
- Canonical repository: `kwaksinwoo01/ptsip`
- External tooling model: Defined
- Reference Tool package name: `ptsip`
- Latest verified PyPI publication in this migration context: `0.2.0`
- Tool CI target: Python 3.11–3.14
- Tool release namespace: `tool-v*`
- Reuse license: Apache License 2.0

## Tool 0.2.3 evidence-correctness state

Tool 0.2.3 implements the low-risk correctness work defined after the two-Pilot normative snapshot while preserving the deterministic Human Clarification capability introduced in Tool 0.2.2.

Implemented in this migration:

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

Tool 0.2.3 still does **not** claim complete automated PTSIP Conformance Evaluation. `ptsip pilot` remains evidence collection plus bounded rule findings, and `conformance.status` remains `NOT_EVALUATED` until rule-relative coverage and Product Artifact evaluation are implemented.

## Human Clarification regression boundary

The following Tool 0.2.2 behavior remains required and is regression-tested:

- `ptsip clarify`;
- deterministic clarification with zero LLM/model API calls;
- English/Korean fixed questions via `--lang` / `PTSIP_LANG`;
- Consumer Repository GitHub identity from Git `origin`;
- explicit `--publish github-issue` with `--repo owner/repository` override;
- duplicate-publication state under external `PTSIP_HOME/clarifications`;
- read-only clarification analysis unless publication is explicitly requested;
- no Issue-answer collection, free-form interpretation, or automatic classification.

## Tool 0.3.0 — conformance capability expansion

Planned capability work remains separate from this correctness release:

- JavaScript/TypeScript source dependency adapter;
- npm manifest adapter;
- Product Artifact evidence adapters;
- stable diagnostic emission using `ptsip-diagnostic/v1`;
- rule-relative evidence coverage evaluation;
- `ptsip conform`;
- agent-decision ingestion;
- external evidence import;
- extended .NET analysis;
- Go adapter.

## Release blockers for PTSIP Specification 1.0

- stabilize core rule and profile semantics through multiple real Pilots;
- exercise automated validator rules against real dependency and packaging boundaries;
- validate the constrained Agent Contract and deterministic Human Clarification against real ownership questions;
- exercise rule-relative evidence coverage and Product Artifact evidence;
- publish tagged stable specification releases;
- exercise a later Reference Tool release against real Consumer Repositories with component declarations, artifact evidence, and repeatable conformance evaluation.
