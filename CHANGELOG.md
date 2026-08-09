# Changelog

## Unreleased — next 0.2.0-draft normative snapshot

Pilot-driven specification evolution based on materially different Consumer Repository exercises (`turbo-system` and Simple Connection):

- added `PTSIP-CLS-002` Coherent Component Boundary so mixed ownership cannot be hidden behind an over-broad component;
- clarified Neutral Contract classification so consumer count alone neither grants nor denies neutrality;
- separated evidence-node scope (`PROJECT_COMPONENT`, `EXTERNAL_DEPENDENCY`, `PLATFORM`, `UNRESOLVED_TARGET`) from the three PTSIP architecture classifications;
- standardized adapter-independent dependency relationship types, lifecycle phases, resolution semantics, and evidence provenance (`DECLARED`, `OBSERVED`, `INFERRED`);
- clarified `PTSIP-DEP-002` to distinguish bounded Toolchain analysis/build/test use of Product implementation from convenient executable implementation reuse;
- added `PTSIP-ART-001` and Product Artifact semantics separating artifact owner from artifact producer and requiring packaging/contents evidence for strict packaging conclusions;
- clarified `PTSIP-LCY-001` so a workflow trigger alone is not treated as proof of Product release coupling;
- added `PTSIP-EVD-003` rule-relative Applicable Evidence Coverage so zero findings cannot imply conformance when blocking evidence is missing;
- added `PTSIP-EVD-004` Evidence Provenance and Dependency Semantics;
- separated Profile Validation from Conformance Evaluation;
- defined conformance outcomes `CONFORMANT`, `NON_CONFORMANT`, and `INCOMPLETE`; `NOT_EVALUATED` remains a tooling state rather than a conformance outcome;
- added `PTSIP-DIA-001` Stable Diagnostic Identity and reference `ptsip-diagnostic/v1` schema;
- added reference `ptsip-artifact-evidence/v1` schema;
- added `PTSIP-POL-001` so project-specific component dependency policies may strengthen but never weaken universal PTSIP rules;
- extended the reference profile with optional same-plane/cross-component dependency constraints using `default`, `allow`, and `deny` relationships;
- changed reference profile ownership declarations to exactly one mode: `boundaries` XOR `components`;
- retired/superseded `PTSIP-EXC-001`; an established `MUST`/`MUST NOT` violation is `NON_CONFORMANT` until remediated and reevaluated;
- removed the canonical reference-profile waiver surface (`exceptions` and `exception_required`) so project governance records cannot alter PTSIP conformance;
- added ADR-0004 documenting the two-Pilot normative decisions;
- synchronized Terminology, Registry, Agent Contract, Reference Architecture, and Adoption Guide with the new evidence/artifact/conformance semantics;
- intentionally left Tool `0.2.2` source and embedded normative resources unchanged because Tool `0.2.2` remains bound to specification revision `895e12d27230af2bb99ad17a96e8df8ef41bc3e0`.

Change categories: `CLARIFICATION`, `NORMATIVE_ADDITION`, `SCHEMA_CHANGE`, `CONFORMANCE_CHANGE`, and limited `NORMATIVE_BREAKING` within the explicitly experimental draft family.

## Unreleased — 0.2.0-draft family evolution

- defined the `0.2.0-draft` label as a mutable draft family whose exact normative snapshot is identified by immutable Git revision;
- clarified that PTSIP Specification and Reference Tool use independent version numbers;
- defined PTSIP Component as the architectural classification unit;
- preserved exactly three architecture classifications and separated unresolved decision statuses (`UNKNOWN`, `CONFLICT`, `INCOMPLETE`);
- added component declarations to the reference project profile while retaining boundary roots as shorthand;
- defined deterministic nested-selector precedence and equal-specificity conflict handling;
- added `PTSIP-EVD-001` Evidence Snapshot Integrity;
- added `PTSIP-EVD-002` Declaration and Observation Are Distinct;
- aligned the exception schema with all fields already required by `PTSIP-EXC-001`;
- added schema-constrained coding-agent classification decisions;
- updated conformance requirements for stable snapshot evidence and visible coverage gaps;
- added ADR-0003 documenting the first Pilot-driven evidence-v2 migration;
- corrected the Conformance example canonical source after the repository rename.

Repository identity maintenance in this draft family:

- renamed the canonical repository from `kwaksinwoo01/ptsip-spec` to `kwaksinwoo01/ptsip`;
- updated canonical Specification Binding metadata, profile schema identity, registry source, examples, and governance references to the new repository URL;
- preserved immutable historical specification revisions across the repository rename.

## 0.2.0-draft — 2026-08-09

- defined Consumer Repository and External PTSIP Tooling;
- added `PTSIP-INT-001` Consumer Repository Non-Intrusion;
- made external inspection and Pilot operations read-only against Consumer Repositories by default;
- prohibited mandatory PTSIP-specific `docs/`, `tools/`, cache, report, or equivalent repository hierarchies solely for tooling operation;
- added `PTSIP-SPC-001` Specification Binding;
- extended the project-profile schema with canonical specification source binding;
- updated Agent Contract and conformance rules for external tooling;
- separated PTSIP specification and reference-tooling release lifecycles;
- added ADR-0002.

## Unreleased history

- adopted Apache License 2.0 for the PTSIP specification repository.

## 0.1.0-draft — 2026-08-07

Initial public-specification draft package:

- defined PTSIP name and scope;
- defined Product, Toolchain, and Neutral Contract classifications;
- established Purpose Before Reuse;
- defined dependency, packaging, build, lifecycle, common-package, contract, and exception rules;
- added governance and conformance model;
- added reference architecture and adoption guide;
- added machine-readable registry and project-profile schema;
- added coding-agent contract;
- added ADR-0001.
