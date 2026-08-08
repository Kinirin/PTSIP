# PTSIP — Product–Toolchain SDK Isolation Policy

**Status:** Draft Specification 0.1.0  
**Canonical repository name:** `ptsip-spec`  
**Specification family:** Software architecture / SDK governance / development toolchain isolation

PTSIP (Product–Toolchain SDK Isolation Policy) is a project-defined architecture policy for managing Software Development Kits (SDKs) according to their **purpose, packaging responsibility, dependency boundary, build environment, and lifecycle**.

PTSIP distinguishes two primary SDK planes:

- **Product SDK Plane** — SDKs and libraries that are part of, support, or are distributed with the product.
- **Toolchain SDK Plane** — SDKs and development tools used to build, validate, migrate, test, generate, inspect, release, or otherwise develop the product.

The central rule is:

> **Purpose precedes reuse.** A component is classified by why it exists and which lifecycle owns it before code-sharing opportunities are considered.

PTSIP does not claim that host/target separation, build-time/runtime separation, toolchain isolation, or independent lifecycle management are new ideas. PTSIP is a named policy that combines these established ideas into a stronger SDK-governance boundary with explicit conformance and machine-readable project rules.

## Why PTSIP exists

In large or long-lived codebases, a validator, schema helper, generator, migration module, or common utility can gradually become shared by both product runtime code and development tooling. That often creates hidden coupling:

- development-only dependencies leak into product packaging;
- toolchain changes force product releases;
- product compatibility concerns block toolchain evolution;
- generic `common` packages erase architectural ownership;
- code reuse becomes more important than lifecycle independence.

PTSIP makes the opposite trade-off: **lifecycle and responsibility boundaries are primary; reuse is conditional.**

## Repository map

- [`spec/PTSIP-SPEC.md`](spec/PTSIP-SPEC.md) — normative architecture specification.
- [`spec/PTSIP-TERMINOLOGY.md`](spec/PTSIP-TERMINOLOGY.md) — canonical terms and meanings.
- [`spec/PTSIP-GOVERNANCE.md`](spec/PTSIP-GOVERNANCE.md) — specification change and exception governance.
- [`spec/PTSIP-CONFORMANCE.md`](spec/PTSIP-CONFORMANCE.md) — requirements for claiming PTSIP conformance.
- [`reference/REFERENCE-ARCHITECTURE.md`](reference/REFERENCE-ARCHITECTURE.md) — reference directory and dependency architecture.
- [`adoption/ADOPTION-GUIDE.md`](adoption/ADOPTION-GUIDE.md) — migration/adoption sequence.
- [`agents/AGENT-CONTRACT.md`](agents/AGENT-CONTRACT.md) — concise rules for coding agents.
- [`registry/ptsip-registry.yaml`](registry/ptsip-registry.yaml) — machine-readable terminology and rule registry.
- [`schemas/ptsip-profile.schema.json`](schemas/ptsip-profile.schema.json) — project-profile schema.
- [`profiles/example.ptsip.yaml`](profiles/example.ptsip.yaml) — example project profile.
- [`decisions/ADR-0001-establish-ptsip.md`](decisions/ADR-0001-establish-ptsip.md) — initial architecture decision record.

## Normative language

The words **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT**, and **MAY** are used as normative requirement keywords in the sense of BCP 14 (RFC 2119 as updated by RFC 8174) when, and only when, they appear in uppercase.

References:

- RFC 2119: https://www.rfc-editor.org/info/rfc2119/
- RFC 8174: https://www.rfc-editor.org/info/rfc8174/

## Relationship to existing concepts

PTSIP is related to, but not identical with:

- host / execution / target separation;
- build-time / runtime dependency separation;
- toolchain isolation;
- dependency graph isolation;
- independent release lifecycle management;
- hermetic or reproducible build practices.

For example, Bazel distinguishes execution-platform and target-platform constraints for toolchains. PTSIP uses the same broad idea that the tools which build or validate software and the software being produced may have different architectural ownership, but PTSIP applies that distinction specifically as an SDK governance policy.

Bazel reference: https://bazel.build/reference/be/platforms-and-toolchains

## Maturity

PTSIP 0.1.0 is a **draft project-defined specification**, not an ISO, IEEE, IETF, CNCF, or other external industry standard. The public specification is intended to make the term reproducible: a person or coding agent should be able to read the specification, apply it to a repository, and independently determine whether the repository conforms.

## License status

A reuse license has not yet been selected for the first public release. Until a license is explicitly added, publication alone does not grant broad reuse rights. License selection is therefore a release-blocking governance item for the first stable public release.
