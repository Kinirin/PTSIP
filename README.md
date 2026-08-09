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
ptsip clarify .
ptsip conform .
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

## Human clarification without speculative inference

When PTSIP detects a component candidate but the Consumer Repository does not declare enough architectural intent to classify it safely, `ptsip clarify` can stop at the missing facts and ask the project owner instead of expanding speculative inference.

Clarification generation is deterministic: it uses repository evidence, fixed completeness rules, and fixed question templates. It does **not** call an LLM or model API, and JSON output explicitly reports `llm_calls: 0` and `speculative_classification: false`.

The clarification interface supports English and Korean prompts only. Language selection follows `--lang en|ko`, then `PTSIP_LANG`, then the operating-system locale, with English as the fallback.

```powershell
ptsip clarify . --lang ko
ptsip clarify . --json
ptsip clarify . --component tools
```

Clarification is read-only by default. To explicitly publish unresolved questions as GitHub Issues in the Consumer Repository:

```powershell
ptsip clarify . --publish github-issue
```

PTSIP reads the inspected Git repository's `origin` and, for GitHub HTTPS or SSH remotes, derives the default `owner/repository`. An explicit override is available when needed:

```powershell
ptsip clarify . --publish github-issue --repo owner/repository
```

GitHub publishing requires an authenticated `gh` CLI only for the explicit publish operation. Publication state used to prevent duplicate clarification Issues is stored under `PTSIP_HOME/clarifications`, outside the Consumer Repository. The clarification workflow does not collect Issue answers, interpret free-form replies, or automatically convert them into an architectural classification.

## Enforced conformance evaluation

`ptsip conform` combines the Project Profile with observed repository evidence and explicitly supplied artifact/review evidence. It reports only `CONFORMANT`, `NON_CONFORMANT`, or `INCOMPLETE`; `NOT_EVALUATED` remains an execution state used by workflows such as Pilot evidence collection.

A basic machine-readable run is:

```powershell
ptsip conform . --artifact-evidence path/to/artifact-evidence.json --json
```

Explicit evidence inputs are repeatable and read-only:

```powershell
ptsip conform . `
  --artifact-evidence product-artifact.json `
  --agent-decision component-review.json `
  --external-evidence validator-evidence.json `
  --json
```

- `--artifact-evidence` accepts `ptsip-artifact-evidence/v1` and evaluates Product packaging without treating the artifact producer as the artifact owner.
- `--agent-decision` accepts the bound `ptsip-agent-classification` decision contract. Agent decisions are review evidence and never silently overwrite the Project Profile.
- `--external-evidence` accepts the Reference Tool `ptsip-external-evidence/v1` input envelope. The producer, Consumer Repository identity, exact repository revision, evidence provenance, and imported-file SHA-256 are preserved. Stale or contradictory evidence blocks a strict claim instead of overriding native evidence.

The Tool collects dependency evidence from Python, JavaScript/TypeScript and npm, Go source/modules, .NET project/source metadata, and GitHub Actions local-script invocations. It also evaluates declared component manifests for independent build resolution and uses path-scoped release automation plus declared release/compatibility ownership as bounded lifecycle evidence.

`CONFORMANT` is emitted only when applicable mandatory-rule evidence is sufficient for the supported evaluation scope and the final diagnostic/coverage contract audit passes. A definite mandatory violation produces `NON_CONFORMANT`; an unresolved target, invalid/stale evidence input, incomplete artifact evidence, ambiguous build/lifecycle evidence, or another gap capable of hiding a mandatory violation produces `INCOMPLETE`.

CLI exit codes for `ptsip conform` are:

| Exit code | Outcome |
| --- | --- |
| `0` | `CONFORMANT` |
| `5` | `NON_CONFORMANT` |
| `6` | `INCOMPLETE` |

The Tool does not restructure the Consumer Repository, auto-approve architecture exceptions, or convert Human Clarification answers into classifications.

## Reference Tool

The independently versioned Reference Tool is implemented under [`src/ptsip/`](src/ptsip/). The repository is shared with the Specification, but their release lifecycles remain separate.

The current tooling focuses on:

- read-only repository inspection;
- Pilot evidence collection;
- repository snapshot and non-intrusion evidence;
- component and multi-language dependency evidence;
- deterministic human clarification for missing architectural intent;
- project-profile validation;
- explicit Product Artifact evidence ingestion and packaging evaluation;
- independent build-resolution and bounded lifecycle evidence evaluation;
- constrained coding-agent decision ingestion as review evidence;
- revision-bound external dependency evidence import with provenance;
- deterministic Enforced Conformance evaluation and diagnostic/coverage auditing.

Conformance is evidence-relative rather than detection-relative: unsupported, unresolved, contradictory, stale, or incomplete evidence that can conceal an applicable `MUST`/`MUST NOT` result prevents `CONFORMANT` even when no violation has been detected.

Pilot state is stored outside the repository by default (`%LOCALAPPDATA%\PTSIP` on Windows and the platform-equivalent user state directory elsewhere). `PTSIP_HOME` can override that location.

Tool releases use the `tool-v*` tag/release namespace. Specification releases may use a separate `spec-v*` namespace.

## Repository map

| Area | Location | Purpose |
| --- | --- | --- |
| Normative Specification | [`spec/`](spec/) | Architecture rules, terminology, governance, and conformance requirements. |
| Machine-readable rules | [`registry/`](registry/) | Canonical terminology and rule registry. |
| Schemas | [`schemas/`](schemas/) | Project profile, diagnostics, artifact evidence, and coding-agent decision schemas. |
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
