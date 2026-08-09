<p align="right">
  English | <a href="README.ko.md">한국어</a>
</p>

# PTSIP — Product–Toolchain SDK Isolation Policy

**Status:** Draft project-defined specification  
**Specification family:** Software architecture / SDK governance / development toolchain isolation  
**License:** Apache License 2.0

PTSIP (Product–Toolchain SDK Isolation Policy) is a project-defined architecture policy for managing Software Development Kits (SDKs) according to their **purpose, packaging responsibility, dependency boundary, build environment, and lifecycle**.

> **Purpose precedes reuse.** Classify a component by why it exists and which lifecycle owns it before considering code-sharing opportunities.

## Why PTSIP exists

In large or long-lived codebases, validators, schema helpers, generators, migration modules, and generic utilities can gradually become shared by both product runtime code and development tooling. That creates hidden coupling:

- development-only dependencies leak into product packaging;
- toolchain changes force product releases;
- product compatibility concerns block toolchain evolution;
- generic `common` packages erase architectural ownership;
- code reuse becomes more important than lifecycle independence.

PTSIP makes the opposite trade-off: **lifecycle and responsibility boundaries are primary; reuse is conditional.**

## Architecture at a glance

PTSIP distinguishes two primary SDK planes:

| Plane | Responsibility |
| --- | --- |
| **Product SDK Plane** | SDKs and libraries that are part of, support, or are distributed with the product. |
| **Toolchain SDK Plane** | SDKs and development tools used to build, validate, migrate, test, generate, inspect, release, or otherwise develop the product. |

PTSIP defines exactly three architectural classifications:

| Classification | Meaning |
| --- | --- |
| `PRODUCT` | Product-owned runtime, library, SDK, or distributed component. |
| `TOOLCHAIN` | Development-only tooling or SDK component owned by the development lifecycle. |
| `NEUTRAL_CONTRACT` | A deliberately neutral contract that may be shared without collapsing Product and Toolchain ownership. |

Inspection states such as `UNKNOWN`, `CONFLICT`, and `INCOMPLETE` describe unresolved decisions. They are not additional architectural classifications or SDK planes.

PTSIP does not claim that host/target separation, build-time/runtime separation, toolchain isolation, or independent lifecycle management are new ideas. It combines these established ideas into an explicit SDK-governance boundary with conformance rules and machine-readable project metadata.

## Install and use

PTSIP requires Python 3.11 or newer.

Install the latest published Reference Tool from PyPI:

```powershell
pip install ptsip
```

For a project dependency, prefer a **minimum compatible version** over an exact release pin unless reproducibility requires exact pinning. For example, a project that requires the 0.2+ interface can declare:

```text
# requirements.txt
ptsip>=0.2.0
```

That requirement is a compatibility floor, not a declaration of the latest PTSIP release.

Common commands:

```powershell
ptsip --version
ptsip spec
ptsip doctor .
ptsip inspect .
ptsip pilot .
ptsip validate .
```

For source development:

```powershell
pip install -e ".[dev]"
```

## Specification and Tool lifecycle

The **PTSIP Specification** and the **PTSIP Reference Tool** have independent release lifecycles. A Tool version does not imply a Specification release with the same version number.

This README intentionally does **not** duplicate the current Tool version, latest published release number, or immutable Specification revision. Those values have authoritative sources:

- [`pyproject.toml`](pyproject.toml) — Tool source version and package metadata;
- [GitHub Releases](https://github.com/kwaksinwoo01/ptsip/releases) — published Tool and Specification releases;
- `ptsip --version` — installed Tool version;
- `ptsip spec` — Specification identity bound to the installed Tool;
- [`spec/`](spec/) and [`registry/ptsip-registry.yaml`](registry/ptsip-registry.yaml) — canonical Specification content and machine-readable identity.

This keeps the project overview readable and prevents routine release work from requiring README version edits.

## Consumer Repository non-intrusion

PTSIP does not require adopting repositories to create PTSIP-specific `docs/`, `tools/`, `.ptsip/`, cache, or report directories.

External PTSIP inspection and Pilot tooling is read-only against the Consumer Repository by default. Tool-owned state should remain outside that repository unless the user explicitly chooses otherwise.

A project may voluntarily provide a machine-readable profile for enforced conformance, but the profile location remains a project/configuration concern rather than a required repository topology.

## Reference Tool

The independently versioned Reference Tool is implemented under [`src/ptsip/`](src/ptsip/). The repository is shared with the Specification, but their release lifecycles remain separate.

The current tooling focuses on:

- read-only repository inspection;
- Pilot evidence collection;
- repository snapshot and non-intrusion evidence;
- component and dependency evidence;
- project-profile validation;
- constrained coding-agent classification decisions.

Product artifact inspection and complete automated **Enforced Conformance** evaluation are not claimed unless explicitly documented by the applicable Specification and Tool release.

Pilot state is stored outside the repository by default (`%LOCALAPPDATA%\PTSIP` on Windows and the platform-equivalent user state directory elsewhere). `PTSIP_HOME` can override that location.

Tool releases use the `tool-v*` tag/release namespace. Specification releases may use a separate `spec-v*` namespace.

## Repository map

| Area | Location | Purpose |
| --- | --- | --- |
| Normative Specification | [`spec/`](spec/) | Architecture rules, terminology, governance, and conformance requirements. |
| Machine-readable rules | [`registry/`](registry/) | Canonical terminology and rule registry. |
| Schemas | [`schemas/`](schemas/) | Project profile and coding-agent decision schemas. |
| Reference architecture | [`reference/`](reference/) | Informative architecture guidance. |
| Adoption guidance | [`adoption/`](adoption/) | Migration and adoption sequence. |
| Agent contract | [`agents/`](agents/) | Concise rules for coding agents. |
| Example profiles | [`profiles/`](profiles/) | Example PTSIP project profiles. |
| Architecture decisions | [`decisions/`](decisions/) | Specification decision records. |
| Reference Tool | [`src/ptsip/`](src/ptsip/) | Installable Python implementation. |
| Tool tests | [`tests/`](tests/) | Reference Tool verification. |

Important repository files and automation:

- [`pyproject.toml`](pyproject.toml) — Python package/build metadata;
- [`CHANGELOG.md`](CHANGELOG.md) — Specification change history;
- [`TOOLING-CHANGELOG.md`](TOOLING-CHANGELOG.md) — Tool change history;
- [`.github/workflows/tooling-test.yml`](.github/workflows/tooling-test.yml) — Tool CI;
- [`.github/workflows/tooling-release.yml`](.github/workflows/tooling-release.yml) — PyPI Trusted Publishing for `tool-v*` releases;
- [`.github/workflows/readme-translation.yml`](.github/workflows/readme-translation.yml) — automatic Korean README synchronization;
- [`.github/scripts/sync_readme_ko.py`](.github/scripts/sync_readme_ko.py) — README translation and structural-validation helper;
- [`README.md`](README.md) — canonical English project overview;
- [`README.ko.md`](README.ko.md) — automatically synchronized Korean translation.

## README localization

`README.md` is the canonical project overview. `README.ko.md` is a translated view and should not become an independently maintained source of project facts.

When the English README changes on `main`, [`.github/workflows/readme-translation.yml`](.github/workflows/readme-translation.yml) regenerates the Korean README and commits the result. The workflow calls GitHub Models with the repository-provided `GITHUB_TOKEN`, so a separate model API secret is not required.

Before writing the translation, the helper at [`.github/scripts/sync_readme_ko.py`](.github/scripts/sync_readme_ko.py) checks Markdown heading structure, fenced code blocks, link destinations, normative keywords, and suspicious output length. A structurally unsafe model response fails the workflow instead of overwriting `README.ko.md`.

This avoids routine manual translation work while reducing the risk of the two documents drifting apart.

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

PTSIP is a **draft project-defined specification**, not an ISO, IEEE, IETF, CNCF, or other external industry standard.

The public specification is intended to make the policy reproducible: a person, coding agent, or external validator should be able to identify the governing Specification and independently evaluate a repository against it.

## License

This repository, including the PTSIP Specification and Reference Tool unless explicitly stated otherwise, is licensed under the **Apache License, Version 2.0**. See [`LICENSE`](LICENSE).
