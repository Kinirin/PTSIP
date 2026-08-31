# PTSIP Specification 0.2.0-draft Release Notes

**Status:** Current experimental Specification family  
**Latest canonical normative snapshot:** `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`  
**Identity model:** draft family label + immutable Git revision

The `0.2.0-draft` label is a mutable draft family. The exact normative meaning of any Tool binding or historical evaluation is identified by the immutable Git revision bound to that Tool or evaluation.

## Initial 0.2.0-draft snapshot — 2026-08-09

- defined **Consumer Repository** and **External PTSIP Tooling**;
- added `PTSIP-INT-001` Consumer Repository Non-Intrusion;
- made external inspection and Pilot operations read-only against Consumer Repositories by default;
- prohibited mandatory PTSIP-specific `docs/`, `tools/`, cache, report, or equivalent repository hierarchies solely for tooling operation;
- added `PTSIP-SPC-001` Specification Binding;
- extended the project-profile schema with canonical Specification source binding;
- updated the Agent Contract and conformance rules for external tooling;
- separated PTSIP Specification and Reference Tool release lifecycles;
- added ADR-0002.

## Draft-family identity and component evolution

- defined the `0.2.0-draft` label as a mutable draft family whose exact normative snapshot is identified by immutable Git revision;
- clarified that PTSIP Specification and Reference Tool use independent version numbers;
- defined the PTSIP Component as the architectural classification unit;
- preserved exactly three architecture classifications and separated unresolved decision states `UNKNOWN`, `CONFLICT`, and `INCOMPLETE`;
- added component declarations to the reference project profile while retaining boundary roots as shorthand;
- defined deterministic nested-selector precedence and equal-specificity conflict handling;
- changed reference profile ownership declarations to exactly one mode: `boundaries` XOR `components`;
- clarified Neutral Contract semantics so current consumer count alone neither grants nor denies neutrality;
- added `PTSIP-CLS-002` Coherent Component Boundary so mixed ownership cannot be hidden behind an over-broad component.

## Evidence model

- added `PTSIP-EVD-001` Evidence Snapshot Integrity;
- added `PTSIP-EVD-002` Declaration and Observation Are Distinct;
- added `PTSIP-EVD-003` rule-relative Applicable Evidence Coverage so zero findings cannot imply conformance when blocking evidence is missing;
- added `PTSIP-EVD-004` Evidence Provenance and Dependency Semantics;
- separated evidence-node scope (`PROJECT_COMPONENT`, `EXTERNAL_DEPENDENCY`, `PLATFORM`, `UNRESOLVED_TARGET`) from the three PTSIP architecture classifications;
- standardized adapter-independent dependency relationship types, lifecycle phases, resolution semantics, and evidence provenance (`DECLARED`, `OBSERVED`, `INFERRED`);
- added schema-constrained coding-agent classification decisions;
- updated conformance requirements for stable snapshot evidence and visible coverage gaps.

## Product Artifact, dependency, and lifecycle semantics

- added `PTSIP-ART-001` and Product Artifact semantics separating artifact owner from artifact producer and requiring packaging/content evidence for strict packaging conclusions;
- clarified `PTSIP-DEP-002` to distinguish bounded Toolchain analysis/build/test use of Product implementation from convenient executable implementation reuse;
- clarified `PTSIP-LCY-001` so a workflow trigger alone is not treated as proof of Product release coupling;
- added the reference `ptsip-artifact-evidence/v1` schema.

## Conformance and diagnostics

- separated **Profile Validation** from **Conformance Evaluation**;
- defined conformance outcomes `CONFORMANT`, `NON_CONFORMANT`, and `INCOMPLETE`;
- retained `NOT_EVALUATED` as a Tool execution state rather than a conformance outcome;
- added `PTSIP-DIA-001` Stable Diagnostic Identity and the reference `ptsip-diagnostic/v1` schema;
- retired/superseded `PTSIP-EXC-001` for the current draft family;
- established that an applicable `MUST`/`MUST NOT` violation is `NON_CONFORMANT` until the architecture is remediated and reevaluated;
- removed the canonical reference-profile waiver surface (`exceptions` and `exception_required`) so project governance records cannot alter PTSIP conformance.

## Project-specific policy

- added `PTSIP-POL-001` so project-specific component dependency policies may strengthen but never weaken universal PTSIP rules;
- extended the reference profile with optional same-plane/cross-component dependency constraints using `default`, `allow`, and `deny` relationships.

## Repository identity maintenance

During this draft family the canonical repository moved from an earlier repository location to a unified Tool and Specification repository.

- updated canonical Specification Binding metadata, profile schema identity, registry source, examples, and governance references to the new repository URL;
- preserved immutable historical Specification revisions across the repository rename;
- corrected the Conformance example canonical source after the rename.

## Governance and documentation synchronization

- adopted Apache License 2.0 for the PTSIP Specification repository;
- added ADR-0003 for the first Pilot-driven evidence-v2 migration;
- added ADR-0004 documenting the two-Pilot normative decisions;
- synchronized Terminology, Registry, Agent Contract, Reference Architecture, and Adoption Guide with the evidence/artifact/conformance semantics above.

## Compatibility and historical Tool bindings

The mutable draft-family label does not retroactively change already published Tool behavior. Historical Tools remain bound to their recorded immutable Specification revisions.

In particular, Tool `0.2.0` remained bound to revision `895e12d27230af2bb99ad17a96e8df8ef41bc3e0`, while later Tool `0.3.x` source is bound to canonical revision `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`.

Change categories represented in this family include `CLARIFICATION`, `NORMATIVE_ADDITION`, `SCHEMA_CHANGE`, `CONFORMANCE_CHANGE`, and limited `NORMATIVE_BREAKING` changes within the explicitly experimental draft family.
