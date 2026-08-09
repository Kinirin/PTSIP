# PTSIP Status

- Specification family: `0.2.0-draft`
- Specification identity model: draft family + immutable Git revision
- Maturity: Experimental
- Canonical repository: `kwaksinwoo01/ptsip`
- Public standard status: None; project-defined specification
- External tooling model: Defined
- Reference Python tooling source: `0.2.0`
- Reference Tool package name: `ptsip`
- Latest PyPI publication: `0.1.0a1` via Trusted Publishing
- Tool 0.2.0 publication: Pending migration merge, immutable specification binding, and release
- Tool CI target: Python 3.11–3.14
- Tool release namespace: `tool-v*`
- First Tool release: `tool-v0.1.0a1`, published 2026-08-09
- First real Pilot: completed as evidence-collection validation; strict conformance not claimed
- First external conformance implementation: Not yet recorded
- Reuse license: Apache License 2.0

## First Pilot conclusions incorporated

The first real Consumer Repository Pilot established that Tool 0.1.0a1 successfully performed installation, read-only evidence collection, and external report storage, while exposing gaps that prevented a strict conformance claim.

Tool 0.2.0 migration addresses the highest-priority gaps:

- stable before/after repository snapshot evidence;
- observable non-intrusion status instead of a hard-coded boolean;
- tracked-file inventory with explicit scan/parser coverage failures;
- component candidates and component-aware profile declarations;
- nested selector specificity/conflict validation;
- typed dependency evidence for Python, .NET ProjectReference, and GitHub Actions local-script invocation;
- declaration-versus-observation separation;
- exception schema alignment;
- constrained agent classification decisions;
- Python 3.14 added to the CI verification target.

## Remaining blockers for full Enforced Conformance tooling

- product artifact build/content inspection;
- stronger lifecycle-phase resolution for dependency edges;
- JavaScript/TypeScript dependency adapter;
- Go dependency adapter;
- broader build/release manifest analysis;
- repeatable rule evaluation against at least one real Consumer Repository profile;
- explicit conformance claim generation only after required evidence coverage is complete.

## Release blockers for PTSIP Specification 1.0

- stabilize core rule and profile semantics through additional real Pilots;
- exercise automated validator rules against real dependency and packaging boundaries;
- validate the constrained Agent Contract against real coding-agent classification tasks;
- publish tagged stable specification releases;
- exercise a published Reference Tool release against at least one real Consumer Repository with component declarations and artifact evidence.
