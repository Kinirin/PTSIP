# PTSIP Status

- Specification family: `0.2.0-draft`
- Current Tool/package source version on `main`: `0.2.2`
- Tool 0.2.2 bound specification revision: `895e12d27230af2bb99ad17a96e8df8ef41bc3e0`
- Normative forward-port baseline before this snapshot: `main@3aa5c0b0c91841b2a5eabba4aff1335703b832b1`
- Specification identity model: draft family + immutable Git revision
- Maturity: Experimental
- Canonical repository: `kwaksinwoo01/ptsip`
- External tooling model: Defined
- Reference Tool package name: `ptsip`
- Latest verified PyPI publication in this migration context: `0.2.0`
- Tool CI target: Python 3.11–3.14
- Tool release namespace: `tool-v*`
- Reuse license: Apache License 2.0

## Tool 0.2.2 state

Tool 0.2.2 preserves the evidence-v2 behavior of Tool 0.2.0 and adds deterministic Human Clarification:

- `ptsip clarify`;
- deterministic clarification with zero LLM/model API calls;
- English/Korean fixed questions via `--lang` / `PTSIP_LANG`;
- Consumer Repository GitHub identity from Git `origin`;
- explicit `--publish github-issue` with `--repo owner/repository` override;
- duplicate-publication state under external `PTSIP_HOME/clarifications`;
- read-only clarification analysis unless publication is explicitly requested;
- no Issue-answer collection, free-form interpretation, or automatic classification.

This capability does not change PTSIP architectural classifications, DecisionStatus values, or normative Specification semantics. Tool 0.2.2 remains bound to the immutable specification revision above.

## Next normative snapshot work

The next immutable snapshot remains inside the `0.2.0-draft` family and is driven by the turbo-system and Simple Connection Consumer Pilots. Primary design areas are:

- rule-relative evidence completeness;
- Product Artifact owner/producer/contents/derivation semantics;
- deterministic `CONFORMANT` / `NON_CONFORMANT` / `INCOMPLETE` evaluation;
- mandatory-rule remediation rather than PTSIP waiver/exception semantics;
- coherent component split criteria;
- typed edge/lifecycle-phase/provenance semantics;
- Profile Validation versus Conformance Evaluation separation;
- stable diagnostic identity;
- optional project-specific component dependency constraints that may strengthen but not weaken universal PTSIP rules.

The normative snapshot does not retroactively change Tool 0.2.2. A later Tool release must explicitly bind the newer immutable revision it implements.

## Tool roadmap after the normative snapshot

### Tool 0.2.3 — evidence correctness

Forward-port the work previously planned as Tool 0.2.1:

- Python BOM-aware decoding;
- deterministic relative-import resolution where evidence permits it;
- project/external/platform/unresolved node-scope distinction;
- GitHub Actions working-directory and false-positive handling;
- explicit evaluator-run/evaluator-blocked reporting;
- regression coverage proving Tool 0.2.2 Human Clarification remains deterministic and intact.

### Tool 0.3.0 — conformance capability expansion

Planned capability work includes JavaScript/TypeScript and npm adapters, artifact evidence adapters, stable diagnostics, rule-relative coverage evaluation, `conform`, agent-decision ingestion, external evidence import, extended .NET analysis, and Go coverage.

## Release blockers for PTSIP Specification 1.0

- stabilize core rule and profile semantics through multiple real Pilots;
- exercise automated validator rules against real dependency and packaging boundaries;
- validate the constrained Agent Contract and deterministic Human Clarification against real ownership questions;
- exercise rule-relative evidence coverage and Product Artifact evidence;
- publish tagged stable specification releases;
- exercise a later Reference Tool release against real Consumer Repositories with component declarations, artifact evidence, and repeatable conformance evaluation.
