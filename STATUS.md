# PTSIP Status

- Specification family: `0.2.0-draft`
- Current published Tool 0.2.0 bound specification revision: `895e12d27230af2bb99ad17a96e8df8ef41bc3e0`
- Specification identity model: draft family + immutable Git revision
- Maturity: Experimental
- Canonical repository: `kwaksinwoo01/ptsip`
- Public standard status: None; project-defined specification
- External tooling model: Defined
- Reference Python tooling source: `0.2.0`
- Reference Tool package name: `ptsip`
- Latest PyPI publication: `0.2.0` via Trusted Publishing
- Tool 0.2.0 publication: Published and installable as `pip install ptsip==0.2.0`
- Tool CI target: Python 3.11–3.14
- Tool release namespace: `tool-v*`
- First Tool release: `tool-v0.1.0a1`, published 2026-08-09
- Current published Tool release: `tool-v0.2.0`
- First real Pilot: completed as evidence-collection validation; strict conformance not claimed
- Multi-Consumer Pilot evidence: turbo-system plus Simple Connection feedback now drive the next `0.2.0-draft` normative snapshot design
- First external conformance implementation: Not yet recorded
- Reuse license: Apache License 2.0

## Tool 0.2.0 state

Tool 0.2.0 addresses the highest-priority gaps from the first Pilot:

- stable before/after repository snapshot evidence;
- observable non-intrusion status instead of a hard-coded boolean;
- tracked-file inventory with explicit scan/parser coverage failures;
- component candidates and component-aware profile declarations;
- nested selector specificity/conflict validation;
- typed dependency evidence for Python, .NET ProjectReference, and GitHub Actions local-script invocation;
- declaration-versus-observation separation;
- constrained agent classification decisions;
- Python 3.14 in the CI verification target.

Tool 0.2.0 is intentionally bound to specification revision `895e12d27230af2bb99ad17a96e8df8ef41bc3e0` and remains a bounded evidence/validation implementation rather than a complete Enforced Conformance engine.

## Next normative snapshot work

Two materially different Consumer Pilot exercises now drive the next immutable snapshot inside the same `0.2.0-draft` family.

Primary design areas:

- rule-relative evidence completeness and strict-claim gating;
- Product Artifact owner/producer/contents/derivation semantics;
- exception/conformance-effect consistency;
- coherent component split criteria;
- typed edge/lifecycle-phase/provenance semantics;
- Profile Validation versus Conformance Evaluation separation;
- stable diagnostic identity;
- optional project-specific component dependency constraints that may strengthen but not weaken universal PTSIP rules.

The next normative snapshot does not retroactively change published Tool 0.2.0. A later Tool release must explicitly bind the newer merge revision it implements.

## Remaining blockers for full Enforced Conformance tooling

- Product Artifact content inspection/adapters;
- stronger lifecycle-phase and evidence-provenance resolution;
- JavaScript/TypeScript dependency and npm-manifest adapters;
- extended .NET dependency/build/publish analysis;
- Go dependency adapter;
- agent decision ingestion;
- stable diagnostic contract implementation;
- rule-relative evidence coverage gate and conformance evaluation;
- broader build/release manifest analysis;
- repeatable rule evaluation against real Consumer Repository profiles.

## Release blockers for PTSIP Specification 1.0

- stabilize core rule and profile semantics through multiple real Pilots;
- exercise automated validator rules against real dependency and packaging boundaries;
- validate the constrained Agent Contract against real coding-agent classification tasks;
- exercise rule-relative evidence coverage and Product Artifact evidence;
- publish tagged stable specification releases;
- exercise a later Reference Tool release against at least one real Consumer Repository with component declarations, artifact evidence, and repeatable conformance evaluation.
