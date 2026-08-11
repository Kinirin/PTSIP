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
pip install PTSIP
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
ptsip adopt --help
ptsip validate .
ptsip clarify .
ptsip gate .
ptsip resolve --help
ptsip conform .
```

For source development:

```powershell
pip install -e ".[dev]"
```

## Specification and Tool lifecycle

The **PTSIP Specification** and the **PTSIP Reference Tool** have independent release lifecycles. A Tool version does not imply a Specification release with the same version number.

This README intentionally does **not** duplicate the current Tool version, latest published Tool release number, or immutable active Specification revision. Those values have authoritative sources:

- [`pyproject.toml`](pyproject.toml) — Tool source version and package metadata;
- [GitHub Releases](https://github.com/kwaksinwoo01/ptsip/releases) — published Tool and Specification releases;
- `ptsip --version` — installed Tool version;
- `ptsip spec` — Specification identity bound to the installed Tool;
- [`spec/`](spec/) and [`registry/ptsip-registry.yaml`](registry/ptsip-registry.yaml) — canonical Specification content and machine-readable identity.

The published `spec-v0.3.4-draft` GitHub Release records the proposed **Explicit Project Adoption + Distributed Authority Consistency** design. It is a design release, not an automatic active Tool binding. The active normative baseline remains whatever exact immutable revision an installed Tool reports through `ptsip spec` until a coherent Specification migration and explicit Tool rebind occur.

This separation keeps the project overview readable and prevents a Specification design release from silently changing already published Tool semantics.

## Consumer Repository non-intrusion

PTSIP does not require adopting repositories to create PTSIP-specific `docs/`, `tools/`, `.ptsip/`, cache, or report directories.

External PTSIP inspection and Pilot tooling is read-only against the Consumer Repository by default. Tool-owned local state should remain outside that repository unless the user explicitly chooses otherwise.

The default project-owned architecture declaration is repository-root `ptsip.yaml`. It is intended to be committed with the Consumer Repository rather than placed in `.gitignore`. An explicit `--profile <path>` remains available for projects that own the profile elsewhere.

Local workflow state such as `control-plane.sqlite3` remains under `PTSIP_HOME` and is **not** a portable architecture source of truth. It should not be committed or shared through Git. GitHub-coordinated repositories instead use a dedicated remote Git ref for unresolved/resolved decision coordination; the normal worktree still receives the durable architecture declaration through `ptsip.yaml`.

## Explicit project adoption

`ptsip adopt` is the project-owner entry point for turning a discovered component candidate into an explicit PTSIP architecture declaration. PTSIP discovers candidate scope from repository evidence; the project owner supplies the architecture intent. Directory names such as `tools/` are never sufficient to auto-classify a component as `TOOLCHAIN`.

The command is a dry-run by default:

```powershell
ptsip adopt . `
  --component tools `
  --classification TOOLCHAIN `
  --purpose "Repository-local generation tooling" `
  --shipped no `
  --runtime-required no `
  --lifecycle-owner DEVELOPMENT_TOOLING `
  --executable yes `
  --json
```

Apply only after reviewing the plan:

```powershell
ptsip adopt . `
  --component tools `
  --classification TOOLCHAIN `
  --purpose "Repository-local generation tooling" `
  --shipped no `
  --runtime-required no `
  --lifecycle-owner DEVELOPMENT_TOOLING `
  --executable yes `
  --apply `
  --json
```

For a non-GitHub repository, adoption is local and does not require a distributed Decision Authority. For a GitHub repository, a new shared architecture decision is coordinated through the repository's PTSIP GitHub authority before local mutation. If a valid local declaration already exists and no distributed decision exists for that scope, PTSIP does not fabricate remote decision history solely for bookkeeping.

`adopt`, `resolve`, `validate`, `conform`, `clarify`, and `gate` all accept the same explicit `--profile` location where applicable.

## Human clarification without speculative inference

When PTSIP detects a component candidate but the Consumer Repository does not declare enough architectural intent to classify it safely, `ptsip clarify` can stop at the missing facts and ask the project owner instead of expanding speculative inference.

Clarification generation is deterministic: it uses repository evidence, fixed completeness rules, and fixed question templates. It does **not** call an LLM or model API, and JSON output explicitly reports `llm_calls: 0` and `speculative_classification: false`.

The clarification interface supports English and Korean prompts only. Language selection follows `--lang en|ko`, then `PTSIP_LANG`, then the operating-system locale, with English as the fallback.

```powershell
ptsip clarify . --lang ko
ptsip clarify . --json
ptsip clarify . --component tools
ptsip clarify . --component tools --profile config/ptsip.yaml
```

Clarification is read-only by default. The older explicit Issue publisher remains available as a manual/offline fallback:

```powershell
ptsip clarify . --publish github-issue
```

PTSIP reads the inspected Git repository's `origin` and, for GitHub HTTPS or SSH remotes, derives the default `owner/repository`. An explicit override is available when needed:

```powershell
ptsip clarify . --publish github-issue --repo owner/repository
```

The manual publisher requires an authenticated `gh` CLI only for the explicit publish operation. Its duplicate-publication state is stored under `PTSIP_HOME/clarifications`, outside the Consumer Repository. Free-form Issue replies are not interpreted by an LLM.

## Coding-agent decision gate

The Reference Tool provides an **on-demand** human architecture-decision workflow for coding-agent sessions. It does not run a reminder timer or continuous background poll. A coding agent calls `ptsip gate` when its current boundary-sensitive task needs authoritative architecture state. In distributed mode, this includes checking whether an apparently complete local Project Profile is stale relative to an existing repository-global winner.

```powershell
ptsip gate . --component tools --json
```

Decision coordination is selected as follows:

- GitHub origin detected: use the GitHub-coordinated authority by default;
- no GitHub origin: use the embedded Local DecisionStore under `PTSIP_HOME`;
- `--coordination local`: explicitly use local-only coordination;
- `--coordination github`: explicitly require GitHub coordination;
- `--control-plane <URL>`: explicitly use the hosted HTTP Control Plane instead of either built-in mode.

GitHub coordination stores unresolved/resolved decision records under the dedicated `refs/heads/ptsip-policy` authority ref. Mutations are serialized with non-force Git ref compare-and-swap semantics. A stale writer cannot overwrite a newer authority HEAD, and the global decision key is based on repository identity plus normalized component include scope rather than one clone's local clarification ID.

A read-only authority lookup does not create `refs/heads/ptsip-policy` or fabricate a pending decision merely to prove that no distributed decision exists.

Cloud environments may authenticate GitHub coordination with `GH_TOKEN` or `GITHUB_TOKEN`; interactive developer machines may use an authenticated `gh` CLI. The credential must provide the repository permissions required by the operation. If GitHub coordination is selected but required authority freshness or mutation cannot be established, PTSIP fails the coordinated operation instead of silently falling back to a separate Local DecisionStore and creating split-brain authority.

### Authority freshness and reconciliation

For the relevant component scope, GitHub-coordinated `ptsip gate` compares the current local Project Profile with the current distributed authority state before returning a coordination-sensitive result.

| Local Project Profile | GitHub authority | Result |
| --- | --- | --- |
| declaration absent | no decision | create/reuse `DECISION_REQUIRED` state only when the active task actually needs a decision |
| declaration absent | resolved winner | validate and safely project the winner into the selected local profile |
| declaration present | no authority decision | use the project declaration and return without fabricating authority history |
| declaration present and semantically equivalent | resolved equivalent winner | return a resolved/consistent result without rewriting equivalent profile content |
| declaration present and semantically conflicting | resolved different winner | return `AUTHORITY_PROFILE_CONFLICT`; do not silently overwrite/reclassify the Project Profile |
| repository/profile changed during reconciliation | any authority state | refuse stale application and require re-analysis |

Semantic equivalence is based on architecture meaning, not YAML formatting, key ordering, whitespace, or Tool-generated formatting.

This is **action-time synchronization**: agents do not need continuous polling, but stale clones cannot create a second winner for the same component scope.

When a decision is unresolved, `ptsip gate` reports `DECISION_REQUIRED`. The coding agent should stop only the affected work and ask the user to decide. The active coding-agent chat can then record the explicit facts with `ptsip resolve`.

Example resolution:

```powershell
ptsip resolve . `
  --decision gdec-example `
  --classification TOOLCHAIN `
  --purpose "Repository migration tooling" `
  --shipped no `
  --runtime-required no `
  --lifecycle-owner DEVELOPMENT_TOOLING `
  --executable yes
```

The first valid resolution wins within the selected authority. A later contradictory answer cannot replace it. `ptsip.yaml` remains the project-owned architecture declaration; the GitHub authority or Local DecisionStore records workflow/decision state needed to coordinate how an undeclared architecture fact was resolved.

Global decision state and clone-local application state are separate. A global `RESOLVED` decision means one architecture answer won; it does not imply that every clone has already written the declaration locally. Local projection/application reporting cannot change which answer won.

The older hosted GitHub App / webhook Control Plane remains available through explicit `--control-plane <URL>` selection. Its GitHub Issue interface can still accept the fixed `ptsip-clarification-answer/v1` structure from an authorized repository writer. See [`reference/DECISION-CONTROL-PLANE.md`](reference/DECISION-CONTROL-PLANE.md) for the Tool-level workflow and authority contract.

The GitHub App runtime is optional:

```powershell
pip install "ptsip[github-app]"
ptsip-app --help
```

## Enforced conformance evaluation

`ptsip conform` combines the Project Profile with observed repository evidence and explicitly supplied artifact/review evidence. It reports only `CONFORMANT`, `NON_CONFORMANT`, or `INCOMPLETE`; `NOT_EVALUATED` remains an execution state used by workflows such as Pilot evidence collection.

A Decision Authority is architecture-decision coordination state, not a conformance oracle. A resolved distributed winner does not prove that dependencies, Product Artifacts, build behavior, or lifecycle behavior satisfy PTSIP.

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

- `--artifact-evidence` accepts `ptsip-artifact-evidence/v1` and evaluates Product packaging without treating the artifact producer as the artifact owner. Strict use also requires a Tool-level, non-normative `<artifact-path>.binding.json` sidecar with format `ptsip-artifact-evidence-binding/v1`, the artifact SHA-256, and a `subject` containing the exact Consumer Repository identity, revision, and tracked-content fingerprint. Missing, stale, or mismatched binding remains `INCOMPLETE`; the canonical artifact evidence schema is unchanged.
- `--agent-decision` accepts the bound `ptsip-agent-classification` decision contract. Agent decisions are review evidence and never silently overwrite the Project Profile.
- `--external-evidence` accepts the Reference Tool `ptsip-external-evidence/v1` input envelope. The producer, Consumer Repository identity, exact repository revision, evidence provenance, and imported-file SHA-256 are preserved. Stale or contradictory evidence blocks a strict claim instead of overriding native evidence.

The Tool collects dependency evidence from Python, JavaScript/TypeScript and npm, Go source/modules, .NET project/source metadata, and GitHub Actions local-script invocations. Unsupported executable source in Product/Toolchain ownership is a blocking coverage gap, while documentation and ordinary non-source files are not blocked merely by extension. It also evaluates ownership-compatible declared component manifests for independent build resolution and uses path-scoped release automation plus declared release/compatibility ownership as bounded lifecycle evidence.

`CONFORMANT` is emitted only when applicable mandatory-rule evidence is sufficient for the supported evaluation scope and the final diagnostic/coverage contract audit passes. A definite mandatory violation produces `NON_CONFORMANT`; an unresolved target, invalid/stale evidence input, incomplete artifact evidence, ambiguous build/lifecycle evidence, or another gap capable of hiding a mandatory violation produces `INCOMPLETE`.

CLI exit codes for `ptsip conform` are:

| Exit code | Outcome |
| --- | --- |
| `0` | `CONFORMANT` |
| `5` | `NON_CONFORMANT` |
| `6` | `INCOMPLETE` |

The Tool does not restructure the Consumer Repository or auto-approve architecture exceptions. A Project Profile is written only through an explicit project-owner adoption or user-authorized resolution workflow; conformance evaluation treats the resulting profile as a declaration that must be checked against observed evidence.

## Reference Tool

The independently versioned Reference Tool is implemented under [`src/ptsip/`](src/ptsip/). The repository is shared with the Specification, but their release lifecycles remain separate.

The current tooling focuses on:

- read-only repository inspection;
- Pilot evidence collection;
- repository snapshot and non-intrusion evidence;
- component and multi-language dependency evidence;
- deterministic human clarification for missing architectural intent;
- explicit project-owner adoption with dry-run/application guards;
- on-demand coding-agent decision gating and explicit human resolution;
- GitHub-coordinated first-winner authority for multi-environment agents;
- gate-time authority freshness even when a local declaration is complete;
- deterministic missing/equivalent/conflicting authority/Profile reconciliation;
- explicit `AUTHORITY_PROFILE_CONFLICT` without silent profile overwrite;
- fail-closed distributed coordination without implicit Local DecisionStore fallback;
- global decision state separated from clone-local projection/application state;
- embedded Local DecisionStore coordination for intentionally local-only repositories;
- optional hosted GitHub App/Webhook decision synchronization without scheduled reminders;
- project-profile validation;
- explicit Product Artifact evidence ingestion and packaging evaluation;
- independent build-resolution and bounded lifecycle evidence evaluation;
- constrained coding-agent decision ingestion as review evidence;
- revision-bound external dependency evidence import with provenance;
- deterministic Enforced Conformance evaluation and diagnostic/coverage auditing.

Conformance is evidence-relative rather than detection-relative: unsupported, unresolved, contradictory, stale, or incomplete evidence that can conceal an applicable `MUST`/`MUST NOT` result prevents `CONFORMANT` even when no violation has been detected.

Pilot and local decision state is stored outside the repository by default (`%LOCALAPPDATA%\PTSIP` on Windows and the platform-equivalent user state directory elsewhere). `PTSIP_HOME` can override that location. The GitHub-coordinated authority is a dedicated remote Git ref, not a worktree SQLite file.

Tool releases use the `tool-v*` tag/release namespace. Specification releases use the separate `spec-v*` namespace.

## Repository map

| Area | Location | Purpose |
| --- | --- | --- |
| Normative Specification | [`spec/`](spec/) | Architecture rules, terminology, governance, and conformance requirements. |
| Machine-readable rules | [`registry/`](registry/) | Canonical terminology and rule registry. |
| Schemas | [`schemas/`](schemas/) | Project profile, diagnostics, artifact evidence, and coding-agent decision schemas. |
| Reference architecture | [`reference/`](reference/) | Informative architecture guidance. |
| Adoption guidance | [`adoption/`](adoption/) | Migration and adoption sequence. |
| Agent contract | [`agents/AGENT-CONTRACT.md`](agents/AGENT-CONTRACT.md) | Concise rules for coding agents. |
| Example profiles | [`profiles/`](profiles/) | Example PTSIP project profiles. |
| Architecture decisions | [`decisions/`](decisions/) | Specification decision records. |
| Reference Tool | [`src/ptsip/`](src/ptsip/) | Installable Python implementation. |
| Tool tests | [`tests/`](tests/) | Reference Tool verification. |

Important repository files and automation:

- [`pyproject.toml`](pyproject.toml) — Python package/build metadata;
- [`releasenote/`](releasenote/) — versioned Reference Tool and Specification release/history notes;
- [`releasenote/spec-0.3.4-draft.md`](releasenote/spec-0.3.4-draft.md) — published `0.3.4-draft` design record and activation boundary;
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
