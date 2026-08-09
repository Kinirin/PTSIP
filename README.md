# PTSIP — Product–Toolchain SDK Isolation Policy

**Status:** Draft Specification 0.2.0  
**Canonical repository:** `https://github.com/kwaksinwoo01/ptsip`  
**Specification family:** Software architecture / SDK governance / development toolchain isolation  
**Reference Tool source:** `0.2.0`  
**Latest published Tool:** `0.1.0a1` until `tool-v0.2.0` is published  
**License:** Apache License 2.0

PTSIP (Product–Toolchain SDK Isolation Policy) is a project-defined architecture policy for managing Software Development Kits (SDKs) according to their **purpose, packaging responsibility, dependency boundary, build environment, and lifecycle**.

PTSIP distinguishes two primary SDK planes:

- **Product SDK Plane** — SDKs and libraries that are part of, support, or are distributed with the product.
- **Toolchain SDK Plane** — SDKs and development tools used to build, validate, migrate, test, generate, inspect, release, or otherwise develop the product.

The central rule is:

> **Purpose precedes reuse.** A component is classified by why it exists and which lifecycle owns it before code-sharing opportunities are considered.

PTSIP defines exactly three architectural classifications: `PRODUCT`, `TOOLCHAIN`, and `NEUTRAL_CONTRACT`. Inspection states such as `UNKNOWN`, `CONFLICT`, and `INCOMPLETE` describe unresolved decisions and are not additional planes.

PTSIP does not claim that host/target separation, build-time/runtime separation, toolchain isolation, or independent lifecycle management are new ideas. PTSIP is a named policy that combines these established ideas into a stronger SDK-governance boundary with explicit conformance and machine-readable project rules.

## Specification and Tool versions

Specification and Tool versions are independent.

The current specification remains the experimental **`0.2.0-draft` family**. Because a draft family may evolve, an automated evaluation should also record the immutable Git revision that identifies the exact normative snapshot.

The Reference Tool is independently moving to **`0.2.0`**. A Tool version does not imply a same-numbered stable Specification release.

## Consumer Repository non-intrusion

PTSIP does not require adopting repositories to create PTSIP-specific `docs/`, `tools/`, `.ptsip/`, cache, or report directories. External PTSIP inspection and Pilot tooling is read-only against the Consumer Repository by default, and tool-owned state should remain outside that repository unless the user explicitly chooses otherwise.

Tool 0.2.0 replaces the previous hard-coded non-intrusion boolean with before/after repository observations. If HEAD, Git status, or tracked content changes during evidence collection, the snapshot is invalidated rather than silently reported as stable.

A project may voluntarily provide a machine-readable profile for enforced conformance, but the profile location remains a project/configuration concern rather than a required repository topology.

## Reference Tool

This canonical repository contains the independently versioned **PTSIP Reference Tool** under `src/ptsip/`. The repository is shared; the Specification and Tool release lifecycles are not.

The Python distribution name and CLI command are both `ptsip`. The published `0.1.0a1` release remains installable until Tool `0.2.0` is published:

```powershell
pip install ptsip==0.1.0a1
```

Source development of Tool `0.2.0` uses:

```powershell
pip install -e ".[dev]"
ptsip --version
ptsip spec
ptsip doctor .
ptsip inspect .
ptsip pilot .
ptsip validate .
```

Tool 0.2.0 introduces `ptsip-pilot-report/v2` with:

- repository snapshot integrity evidence;
- observed non-intrusion status;
- tracked-file inventory and scan coverage/errors;
- evidence-based component candidates without automatic ownership claims;
- typed Python, .NET ProjectReference, and GitHub Actions invocation edges;
- component-aware project-profile validation;
- constrained coding-agent decision schema;
- initial declaration-to-dependency rule findings.

Product artifact inspection and complete Enforced Conformance evaluation are still future work. Tool 0.2.0 must not be described as a complete PTSIP conformance engine.

Pilot state is stored outside the repository by default (`%LOCALAPPDATA%\PTSIP` on Windows and the platform-equivalent user state directory elsewhere). `PTSIP_HOME` can override that location.

Tool releases use the `tool-v*` tag/release namespace. Specification releases may use a separate `spec-v*` namespace, so both lifecycles can remain independently governed inside one Git repository.

## Why PTSIP exists

In large or long-lived codebases, a validator, schema helper, generator, migration module, or common utility can gradually become shared by both product runtime code and development tooling. That often creates hidden coupling:

- development-only dependencies leak into product packaging;
- toolchain changes force product releases;
- product compatibility concerns block toolchain evolution;
- generic `common` packages erase architectural ownership;
- code reuse becomes more important than lifecycle independence.

PTSIP makes the opposite trade-off: **lifecycle and responsibility boundaries are primary; reuse is conditional.**

## Repository map

### Specification ownership

- [`spec/PTSIP-SPEC.md`](spec/PTSIP-SPEC.md) — normative architecture specification.
- [`spec/PTSIP-TERMINOLOGY.md`](spec/PTSIP-TERMINOLOGY.md) — canonical terms and meanings.
- [`spec/PTSIP-GOVERNANCE.md`](spec/PTSIP-GOVERNANCE.md) — specification change and exception governance.
- [`spec/PTSIP-CONFORMANCE.md`](spec/PTSIP-CONFORMANCE.md) — requirements for claiming PTSIP conformance.
- [`registry/ptsip-registry.yaml`](registry/ptsip-registry.yaml) — machine-readable terminology and rule registry.
- [`schemas/ptsip-profile.schema.json`](schemas/ptsip-profile.schema.json) — project-profile schema.
- [`schemas/ptsip-agent-classification.schema.json`](schemas/ptsip-agent-classification.schema.json) — constrained coding-agent classification decision schema.
- [`reference/REFERENCE-ARCHITECTURE.md`](reference/REFERENCE-ARCHITECTURE.md) — informative reference architecture.
- [`adoption/ADOPTION-GUIDE.md`](adoption/ADOPTION-GUIDE.md) — migration/adoption sequence.
- [`agents/AGENT-CONTRACT.md`](agents/AGENT-CONTRACT.md) — concise rules for coding agents.
- [`profiles/example.ptsip.yaml`](profiles/example.ptsip.yaml) — component-oriented example project profile.
- [`decisions/`](decisions/) — specification architecture decision records.
- [`CHANGELOG.md`](CHANGELOG.md) — Specification change history.

### Reference Tool ownership

- [`src/ptsip/`](src/ptsip/) — installable Python Reference Tool implementation.
- [`tests/`](tests/) — Reference Tool tests.
- [`pyproject.toml`](pyproject.toml) — PyPI distribution/build metadata for `ptsip`.
- [`.github/workflows/tooling-test.yml`](.github/workflows/tooling-test.yml) — Python 3.11–3.14 Tool CI.
- [`.github/workflows/tooling-release.yml`](.github/workflows/tooling-release.yml) — `tool-v*` PyPI Trusted Publishing workflow.
- [`TOOLING-CHANGELOG.md`](TOOLING-CHANGELOG.md) — independently versioned Tool change history.

### Shared repository assets

- [`LICENSE`](LICENSE) — Apache License 2.0 terms for this repository.
- [`README.md`](README.md) — project overview and ownership map.

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

PTSIP 0.2.0-draft is a **draft project-defined specification**, not an ISO, IEEE, IETF, CNCF, or other external industry standard. The public specification is intended to make the term reproducible: a person, coding agent, or external validator should be able to identify the governing specification family and immutable revision and independently evaluate a repository against it.

Reference Tool `0.2.0` is a bounded evidence/validation implementation. It improves snapshot integrity, component/profile semantics, dependency evidence, and agent constraints, but it does not yet claim complete automated PTSIP Enforced Conformance enforcement.

## License

This repository, including the PTSIP specification and Reference Tool unless explicitly stated otherwise, is licensed under the **Apache License, Version 2.0**. See [`LICENSE`](LICENSE).
