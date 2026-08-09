# ADR-0003: Pilot Evidence v2 and Independent Tool Versioning

- **Status:** Accepted for draft specification
- **Date:** 2026-08-09
- **Specification family:** `0.2.0-draft`
- **Reference Tool target:** `0.2.0`

## Context

The first real PTSIP Pilot exercised the published `ptsip 0.1.0a1` package against a large mixed-language Consumer Repository. Installation, command execution, external report storage, and the intended read-only implementation behaved successfully. The Pilot also exposed limits that prevent the 0.1 tool from being treated as a conformance validator:

- inventory counted Python imports without preserving dependency edges;
- root-based hints could not represent nested Product/Toolchain/Neutral Contract ownership;
- repository HEAD could change during a scan without invalidating the report;
- `consumer_repository_modified: false` was emitted as an implementation assertion rather than observed evidence;
- scan/parser failures were not first-class coverage evidence;
- the project-profile exception schema required less information than `PTSIP-EXC-001`;
- agent classification needed a constrained output contract;
- supported Python versions and CI-verified versions needed to be distinguished.

The Pilot also demonstrated that unresolved ownership is useful during analysis. Treating `UNKNOWN` as a fourth architecture classification, however, would weaken `PTSIP-CLS-001` and risk creating an unintended permanent plane.

## Decision

1. Keep the PTSIP Specification family label at `0.2.0-draft` while it remains explicitly experimental.
2. Treat the immutable Git revision as the normative identity of a particular snapshot within that draft family.
3. Version the Reference Tool independently and migrate the next implementation to `0.2.0`.
4. Introduce Pilot report format `ptsip-pilot-report/v2` with repository snapshot, non-intrusion observation, coverage, component candidates, typed dependency evidence, profile state, artifact-inspection state, and conformance-evaluation state.
5. Keep exactly three architecture classifications: `PRODUCT`, `TOOLCHAIN`, and `NEUTRAL_CONTRACT`.
6. Represent unresolved analysis as decision status: `UNKNOWN`, `CONFLICT`, or `INCOMPLETE`.
7. Extend the project profile with optional component declarations while preserving boundary roots as shorthand for uniform repositories.
8. Resolve overlapping component selectors deterministically by specificity and reject equal-specificity ownership conflicts.
9. Treat the project profile as a declaration of intended ownership, not as proof that runtime/build/package evidence conforms to that declaration.
10. Preserve dependency evidence as typed edges and lifecycle phase where known; unresolved phase or dynamic resolution remains explicit.
11. Align the machine-readable exception schema with all information required by `PTSIP-EXC-001`.
12. Constrain coding-agent classification output with JSON Schema and evidence IDs; agent decisions do not approve exceptions or declare conformance.
13. Mark a Pilot/inspection evidence set invalid if repository revision or observed tracked content changes during collection.
14. Replace hard-coded non-intrusion booleans with before/after observable repository-state evidence.

## Consequences

### Positive

- Pilot reports can detect mixed-snapshot evidence caused by concurrent repository changes.
- Nested ownership such as Product runtime, Toolchain build scripts, tests, and Neutral Contracts under one source tree can be declared explicitly.
- AI assistance becomes bounded and reviewable rather than an unstructured architecture authority.
- Dependency evidence can evolve adapter-by-adapter without silently claiming unsupported coverage.
- Specification and Tool versions no longer appear artificially coupled.

### Costs

- Report v2 is not shape-compatible with report v1.
- Component selectors and semantic validation add implementation complexity.
- Dependency phases remain incomplete until language/build adapters gain more context.
- Artifact inspection is still required before the Reference Tool can support a full Enforced Conformance claim.
- Draft-family consumers must record immutable revisions for reproducibility.

## Compatibility

The Specification remains `0.2.0-draft` under the draft-family revision policy. Existing root-based profiles remain supported when they do not rely on incomplete exception records. The Reference Tool moves from prerelease `0.1.0a1` to independently versioned `0.2.0`.

## Affected rules and assets

- `PTSIP-CLS-001` — clarified component granularity and unresolved decision status
- `PTSIP-SPC-001` — strengthened draft revision binding
- `PTSIP-EVD-001` — new evidence snapshot integrity rule
- `PTSIP-EVD-002` — new declaration/observation separation rule
- `PTSIP-EXC-001` — schema aligned to existing normative requirements
- `schemas/ptsip-profile.schema.json` — component model and exception alignment
- `schemas/ptsip-agent-classification.schema.json` — new constrained agent decision schema
- `agents/AGENT-CONTRACT.md` — constrained classification behavior
