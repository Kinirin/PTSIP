# Changelog

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
