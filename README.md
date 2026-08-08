# PTSIP — Product–Toolchain SDK Isolation Policy

**Status:** Draft Specification 0.2.0  
**Canonical repository:** `https://github.com/kwaksinwoo01/ptsip-spec`  
**Specification family:** Software architecture / SDK governance / development toolchain isolation  
**License:** Apache License 2.0

PTSIP (Product–Toolchain SDK Isolation Policy) is a project-defined architecture policy for managing Software Development Kits (SDKs) according to their **purpose, packaging responsibility, dependency boundary, build environment, and lifecycle**.

PTSIP distinguishes two primary SDK planes:

- **Product SDK Plane** — SDKs and libraries that are part of, support, or are distributed with the product.
- **Toolchain SDK Plane** — SDKs and development tools used to build, validate, migrate, test, generate, inspect, release, or otherwise develop the product.

The central rule is:

> **Purpose precedes reuse.** A component is classified by why it exists and which lifecycle owns it before code-sharing opportunities are considered.

PTSIP does not claim that host/target separation, build-time/runtime separation, toolchain isolation, or independent lifecycle management are new ideas. PTSIP is a named policy that combines these established ideas into a stronger SDK-governance boundary with explicit conformance and machine-readable project rules.

## Consumer Repository non-intrusion

PTSIP does not require adopting repositories to create PTSIP-specific `docs/`, `tools/`, `.ptsip/`, cache, or report directories. External PTSIP inspection and Pilot tooling is read-only against the Consumer Repository by default, and tool-owned state should remain outside that repository unless the user explicitly chooses otherwise.

A project may voluntarily provide a machine-readable profile for enforced conformance, but the profile location remains a project/configuration concern rather than a required repository topology.

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
- [`reference/REFERENCE-ARCHITECTURE.md`](reference/REFERENCE-ARCHITECTURE.md) — informative reference architecture.
- [`adoption/ADOPTION-GUIDE.md`](adoption/ADOPTION-GUIDE.md) — migration/adoption sequence.
- [`agents/AGENT-CONTRACT.md`](agents/AGENT-CONTRACT.md) — concise rules for coding agents.
- [`registry/ptsip-registry.yaml`](registry/ptsip-registry.yaml) — machine-readable terminology and rule registry.
- [`schemas/ptsip-profile.schema.json`](schemas/ptsip-profile.schema.json) — project-profile schema.
- [`profiles/example.ptsip.yaml`](profiles/example.ptsip.yaml) — example project profile.
- [`decisions/ADR-0001-establish-ptsip.md`](decisions/ADR-0001-establish-ptsip.md) — initial architecture decision record.
- [`decisions/ADR-0002-external-tooling-non-intrusion.md`](decisions/ADR-0002-external-tooling-non-intrusion.md) — external tooling/non-intrusion decision.
- [`LICENSE`](LICENSE) — Apache License 2.0 terms for this repository.

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

## Maturity

PTSIP 0.2.0-draft is a **draft project-defined specification**, not an ISO, IEEE, IETF, CNCF, or other external industry standard. The public specification is intended to make the term reproducible: a person, coding agent, or external validator should be able to identify the governing specification and independently evaluate a repository against it.

## License

This specification repository is licensed under the **Apache License, Version 2.0**. See [`LICENSE`](LICENSE).
