<p align="right">
  English | <a href="README.ko.md">한국어</a>
</p>

# PTSIP — Primary Lifecycle Ownership and Responsibility Isolation Policy

**Status:** Draft project-defined specification / Tool `0.3.6` development in progress  
**Tool version:** `0.3.6`  
**Specification family:** `0.3.6-draft`  
**Active normative snapshot:** `d6995ed232e845b88d8235b851e80ab54b7804ea`  
**License:** Apache License 2.0

PTSIP is an architecture policy for separating project-owned responsibilities by **primary lifecycle ownership** while preserving explicit architecture intent, lifecycle isolation, reproducible conformance, verification-purpose separation, and multi-environment architecture-decision consistency.

> **Purpose precedes reuse.** Classify a coherent responsibility by why it exists and which lifecycle owns it before considering code sharing.

> **Development status:** Tool `0.3.6` is still under development. This README describes the `0.3.6` foundations already implemented on this branch. Evidence-driven candidate discovery and assisted Tool `0.3.5` migration are still being developed and are not presented as completed release capabilities.

## Architecture model

Tool `0.3.6` evolves PTSIP from the earlier Product/Toolchain-centered model into a **primary lifecycle ownership model**.

The canonical lifecycle classifications are:

| Classification | Meaning |
| --- | --- |
| `PRODUCT` | Responsibility owned directly by the Product lifecycle. |
| `DEVELOPMENT_TOOLING` | Development-lifecycle responsibility used to create, inspect, validate, transform, generate, migrate, analyze, test, or otherwise support development work. |
| `DELIVERY` | Responsibility for carrying a release target through release preparation, publication, promotion, distribution, or deployment to its destination. |
| `OPERATIONS` | Ongoing post-deployment responsibility for keeping deployed systems healthy, recoverable, maintained, and operational. |
| `NEUTRAL_CONTRACT` | Deliberately non-executable, non-owning contract responsibility with independent lifecycle ownership. |

`UNKNOWN`, `CONFLICT`, `INCOMPLETE`, `PENDING`, confidence values, and migration states are workflow/evaluation states, not additional architecture classifications.

### From Tool 0.3.5 to Tool 0.3.6

Tool `0.3.5` used:

```text
PRODUCT
TOOLCHAIN
NEUTRAL_CONTRACT
```

Tool `0.3.6` uses:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

The transition is intentionally asymmetric:

```text
PRODUCT
    -> PRODUCT
       canonical name retained

TOOLCHAIN
    -> DEVELOPMENT_TOOLING
       canonical Tool 0.3.6 development-tooling name

       or, when the former component actually contains
       a different lifecycle responsibility:

       -> DELIVERY
       -> OPERATIONS
       -> another appropriate lifecycle ownership

NEUTRAL_CONTRACT
    -> NEUTRAL_CONTRACT
       when independent-contract semantics still hold
```

`TOOLCHAIN` is therefore a **Tool `0.3.5` legacy classification**, not a canonical Tool `0.3.6` classification or compatibility alias.

`DEVELOPMENT_TOOLING` carries the core development-tool responsibility previously represented by `TOOLCHAIN`, while `DELIVERY` and `OPERATIONS` allow responsibilities that were previously grouped too broadly under Toolchain to be represented by their actual lifecycle ownership.

## Classification is about lifecycle ownership

Classification is not determined by filename, directory, language, framework, executable status, compilation behavior, workflow provider, or artifact kind.

For example:

```text
Product-specific tests
    -> PRODUCT

Reusable test SDK / verification infrastructure
    -> DEVELOPMENT_TOOLING

Product runtime implementation
    -> PRODUCT

Release / publication automation
    -> DELIVERY

Post-deployment maintenance automation
    -> OPERATIONS
```

A path such as:

```text
tests/
src/
tools/
deploy/
ops/
.github/workflows/
```

is evidence context, not architecture authority.

FastAPI, Cloudflare Workers, GitHub Actions, Docker, Terraform, Python, PowerShell, Markdown, YAML, and other technologies likewise do not have universal PTSIP classifications.

## Governing lifecycle responsibility

PTSIP asks which lifecycle obligation primarily explains why a responsibility exists.

Typical questions include:

```text
Why does this responsibility exist?

Which lifecycle obligation fails if it disappears?

What kind of change normally requires it to change?

Who owns its compatibility consequences?

Does its responsibility end when a release reaches its destination?

Or does it continue after deployment as ongoing operational responsibility?
```

The goal is not to classify individual technologies.

The goal is to identify a coherent project-owned responsibility and its primary lifecycle owner.

## Classification, role, relationship, and Verification Purpose

Tool `0.3.6` keeps several architecture concepts separate:

```text
classification
    = primary lifecycle ownership

role
    = responsibility performed inside that lifecycle

relationship
    = typed semantic relationship to another responsibility/artifact

VPMS Verification Purpose
    = what a Verification Case protects or verifies
```

For example, both of these are valid:

```text
PRODUCT
    role = VERIFICATION

DEVELOPMENT_TOOLING
    role = VERIFICATION
```

A Product-owned test can therefore remain `PRODUCT`, while a reusable verification framework can be `DEVELOPMENT_TOOLING`.

The fact that both perform verification does not collapse their lifecycle ownership.

## Responsibility Map v2

Tool `0.3.6` introduces **Responsibility Map v2** as the project-owned architecture declaration model.

It can represent:

- lifecycle-owned components;
- responsibility roles;
- typed component relationships;
- associated artifacts;
- component dependency policy;
- Product packaging/runtime policy;
- explicit, template-backed, and hybrid architecture declarations.

### Explicit mode

The repository directly declares its complete Responsibility Map.

```yaml
responsibility_map:
  mode: explicit
```

### Template mode

The repository explicitly selects an immutable, revision-bound Responsibility Map template.

```yaml
responsibility_map:
  mode: template
  template:
    id: python-package-library
    revision: "sha256:..."
```

### Hybrid mode

The repository selects a template and adds project-owned overrides, extensions, or removals.

```yaml
responsibility_map:
  mode: hybrid
  template:
    id: python-package-library
    revision: "sha256:..."
  overrides:
    components:
      - ...
```

Template selection is explicit.

PTSIP does not automatically choose a template from directory names, framework detection, or discovery confidence.

## Initial Responsibility Map templates

The current Tool `0.3.6` template catalog includes:

```text
python-package-library
python-cli-application
mixed-product-development-delivery
```

Template identities are revision-bound.

Changing template semantics requires a different immutable template revision rather than silently changing the meaning of an already selected template.

Templates provide useful starting declarations.

They do not prescribe one mandatory repository layout.

## Source declaration and Effective Responsibility Map

Tool `0.3.6` distinguishes the repository's original declaration from the architecture resolved for downstream evaluation.

```text
Source Project Profile
        |
        v
explicit / template / hybrid
        |
        v
deterministic materialization
        |
        v
Effective Responsibility Map
```

The **Source Project Profile** remains the project-owned declaration.

The **Effective Responsibility Map** is the deterministic resolved architecture consumed by validation and downstream operations.

Materialization does not become architecture authority.

It must not independently:

- select a template;
- infer lifecycle ownership;
- repair architecture;
- create components from repository heuristics;
- silently rewrite the project's source declaration.

For hybrid declarations, explicit project-owned overrides take precedence over the selected immutable template within the defined override boundary.

## Effective Map identity and provenance

Tool `0.3.6` maintains deterministic Effective Map identity so equivalent resolved architecture can be recognized independently from incidental formatting differences.

The resolved view also retains origin information such as whether an effective responsibility came from:

```text
PROJECT_EXPLICIT
TEMPLATE
PROJECT_OVERRIDE
PROJECT_EXTENSION
PROJECT_REMOVAL
```

This is **declaration-resolution provenance**.

It describes where Effective Map architecture came from and is distinct from repository evidence provenance used by discovery and migration analysis.

## Roles

The current schema supports a deliberately small role vocabulary:

```text
IMPLEMENTATION
VERIFICATION
AUTOMATION
CONFIGURATION
DOCUMENTATION
GOVERNANCE
```

Roles describe responsibility performed within a lifecycle.

They do not create additional lifecycle classifications.

## Typed relationships

Responsibility Map v2 can explicitly describe project-owned semantic relationships such as:

```text
IMPORTS
LINKS
LOADS
INVOKES
READS
GENERATES
BUILDS
PACKAGES
PUBLISHES
DEPLOYS
VERIFIES
MANAGES
DOCUMENTS
SPECIFIES
GOVERNS
```

For example:

```text
Development verification
        |
      VERIFIES
        |
        v
     Product

Delivery automation
        |
      BUILDS
        |
        v
     Product

Delivery automation
        |
     PUBLISHES
        |
        v
     Product
```

Relationship semantics and lifecycle ownership remain separate.

A `DELIVERY` component publishing a Product Artifact does not become `PRODUCT` merely because it produces or publishes that artifact.

## Associated artifacts

Responsibility Map v2 supports **associated artifacts** for project-owned documentation, governance, configuration, or support artifacts that belong with an architectural responsibility but do not independently require another lifecycle-owned component.

Associated artifacts are not a classification escape hatch.

An artifact still requires component evaluation when it has independently owned executable, compatibility, release, or lifecycle responsibility.

## Install and use

PTSIP requires Python 3.11 or newer.

For a new installation of the latest published release:

```powershell
python -m pip install PTSIP
```

To upgrade an existing installation to the latest published release:

```powershell
python -m pip install --upgrade PTSIP
```

The latest published package may differ from this in-development Tool `0.3.6` branch until Tool `0.3.6` is released.

For source development:

```powershell
python -m pip install -e ".[dev]"
```

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

## Specification and Tool lifecycle

The **PTSIP Specification** and **PTSIP Reference Tool** are independently versioned.

- `pyproject.toml` owns Tool/package source version;
- `ptsip --version` reports the installed Tool version;
- `ptsip spec` reports the exact Specification family and immutable revision bound to that Tool;
- `spec/`, `schemas/`, and `registry/` contain canonical Specification assets;
- `src/ptsip/specdata/` contains matching resources embedded in the Tool;
- GitHub Releases publish Tool and Specification release/design records.

This branch currently identifies itself as:

```text
PTSIP Tool
    0.3.6

Specification
    0.3.6-draft
    @ d6995ed232e845b88d8235b851e80ab54b7804ea
```

A Tool version number matching a Specification family number does not imply identity by itself.

## Consumer Repository non-intrusion

PTSIP does not require adopting repositories to create PTSIP-specific:

```text
docs/
tools/
.ptsip/
cache/
report/
hidden state directories
```

merely to use PTSIP.

External inspection and Pilot operations are read-only by default.

Tool-owned operational state belongs outside the Consumer Repository unless the user explicitly chooses a repository path.

The default project-owned architecture declaration remains:

```text
ptsip.yaml
```

at repository root.

Projects may consistently select another profile path with `--profile`.

Local operational state such as `control-plane.sqlite3` is not portable project architecture authority and must not be treated as Git-shared global decision state.

## Explicit project adoption

Candidate discovery is evidence, not architecture authority.

The project owner supplies architecture intent.

Canonical Tool `0.3.6` clarification/adoption decisions use:

```text
classification
purpose
shipped
runtime_required
executable
```

`lifecycle_owner` is no longer a separate canonical Tool `0.3.6` decision field.

It may still be accepted by some CLI paths as a deprecated compatibility input for legacy decision data, but new Tool `0.3.6` architecture state is represented by canonical `classification`.

Example development-tooling adoption dry-run:

```powershell
ptsip adopt . `
  --component tools `
  --classification DEVELOPMENT_TOOLING `
  --purpose "Repository-local generation tooling" `
  --shipped no `
  --runtime-required no `
  --executable yes `
  --json
```

Apply only after reviewing the plan:

```powershell
ptsip adopt . `
  --component tools `
  --classification DEVELOPMENT_TOOLING `
  --purpose "Repository-local generation tooling" `
  --shipped no `
  --runtime-required no `
  --executable yes `
  --apply `
  --json
```

Dry-run remains non-mutating.

Prepared writes must still reject stale repository/profile state rather than applying an architecture decision to changed content.

## VPMS — Verification Purpose Management System

PTSIP and VPMS answer different questions.

PTSIP asks:

```text
Who primarily owns this project responsibility
across its lifecycle?
```

VPMS asks:

```text
Why does this Verification Case exist,
and what does it protect?
```

A Verification Case separates concerns such as:

```text
Formula
Variables
Policy
Target
Runner
```

PTSIP lifecycle classification and VPMS Verification Purpose are separate axes.

PTSIP core does not depend on VPMS.

VPMS may consume stable PTSIP architecture metadata through a narrow read-only integration boundary.

VPMS PASS does not imply PTSIP `CONFORMANT`.

PTSIP `CONFORMANT` does not imply functional verification PASS.

## Decision Authority

PTSIP distinguishes:

```text
Specification
    -> normative architecture / conformance / coordination rules

Decision Authority
    -> which explicit architecture answer won

Project Profile / Responsibility Map
    -> durable project-owned architecture declaration

Observed evidence
    -> what repository/artifacts actually do
```

A Decision Authority is not a conformance oracle and does not replace the Project Profile.

## GitHub-coordinated Reference Tool profile

The Reference Tool supports distributed architecture-decision coordination through a dedicated Git ref:

```text
refs/heads/ptsip-policy
```

GitHub is a Reference Tool backend, not a universal Specification dependency.

The coordination semantics include:

- stable coordination-domain + component-scope decision identity;
- first-valid-resolution-wins;
- conditional mutation / stale-writer protection;
- authority freshness at architecture-sensitive boundaries;
- deterministic reconciliation;
- fail-closed distributed behavior;
- separation of global decision state from clone-local application state.

## Coding-agent decision gate

```powershell
ptsip gate . --component tools --json
```

In distributed mode, a complete local declaration does not automatically bypass relevant authority checks.

The local declaration and current coordinated winner are reconciled semantically.

Equivalent architecture does not require rewriting YAML merely because key ordering or formatting differs.

A conflicting authority winner must not silently overwrite a conflicting local project declaration.

If required distributed authority freshness cannot be established, the affected coordinated operation fails closed instead of silently creating an isolated local winner.

## Global decision state versus local projection

PTSIP keeps global authority state distinct from clone-local application state.

```text
GLOBAL AUTHORITY

PENDING
RESOLVED


LOCAL CLONE / WORKTREE

missing
consistent
locally applied
stale
failed
```

A global `RESOLVED` state does not mean every clone has already applied the declaration.

A clone-local application receipt cannot change which architecture answer won.

PTSIP uses **action-time synchronization** rather than requiring continuous background polling.

## Enforced conformance

`ptsip conform` evaluates declared architecture and observed evidence against applicable PTSIP rules.

Tool `0.3.6` resolves explicit, template, and hybrid source declarations into the Effective Responsibility Map before downstream conformance logic evaluates architecture ownership.

Completed Consumer Repository outcomes are:

| Exit code | Outcome |
| --- | --- |
| `0` | `CONFORMANT` |
| `5` | `NON_CONFORMANT` |
| `6` | `INCOMPLETE` |

A valid Project Profile does not prove conformance.

A resolved Decision Authority winner does not prove conformance.

A zero-finding scan does not prove conformance when blocking evidence gaps remain.

Enforced Conformance against a mutable draft must bind the exact immutable Specification revision.

## Tool 0.3.6 implemented foundation

The current development branch already establishes major Tool `0.3.6` foundations including:

- `PRODUCT` retained as the canonical Product lifecycle classification;
- canonical `TOOLCHAIN` replaced by `DEVELOPMENT_TOOLING`;
- new `DELIVERY` lifecycle classification;
- new `OPERATIONS` lifecycle classification;
- preserved `NEUTRAL_CONTRACT` classification;
- primary lifecycle ownership semantics;
- Responsibility Map v2;
- explicit, template, and hybrid declaration modes;
- immutable template identities;
- roles;
- typed responsibility relationships;
- associated artifacts;
- Source Profile versus Effective Responsibility Map separation;
- deterministic Effective Map identity;
- declaration-resolution provenance;
- Effective Map-aware profile validation;
- Effective Map-aware conformance;
- Effective Map-aware clarification/adoption;
- safe template-to-hybrid project extension when explicitly authorized;
- Effective Map integration boundary for VPMS;
- distributed Decision Authority and reconciliation adapted to Tool `0.3.6` architecture semantics.

## Tool 0.3.6 work still in development

The following areas remain development work and are not claimed here as completed Tool `0.3.6` release capabilities:

```text
evidence/provenance model stabilization
        |
        v
stronger candidate and boundary discovery
        |
        v
Tool 0.3.5 legacy migration analysis
        |
        v
migration proposal / preview
        |
        v
safe confirmed migration application
```

In particular, Tool `0.3.6` migration must not be implemented as a blind:

```text
TOOLCHAIN -> DEVELOPMENT_TOOLING
```

text replacement.

`DEVELOPMENT_TOOLING` is the canonical successor for actual development-tooling ownership, but a legacy `TOOLCHAIN` component may contain responsibilities that must instead be separated into `DELIVERY`, `OPERATIONS`, `PRODUCT`, or another coherent architecture boundary.

Migration analysis therefore remains evidence-driven and project-owner controlled.

## Reference Tool focus

The Tool `0.3.6` direction includes:

- read-only repository inspection and Pilot evidence;
- primary lifecycle ownership classification;
- Responsibility Map v2 validation and resolution;
- explicit/template/hybrid architecture declarations;
- typed responsibility relationships and associated artifacts;
- deterministic clarification for missing architecture intent;
- explicit project-owner adoption;
- on-demand architecture-decision gating and resolution;
- GitHub-coordinated first-winner authority for multi-environment agents;
- gate-time authority freshness and reconciliation;
- fail-closed distributed coordination;
- local-only DecisionStore mode when intentionally selected;
- Product Artifact evidence ingestion;
- evidence-relative Enforced Conformance;
- stable diagnostics;
- VPMS Verification Purpose separation.

Stronger evidence-driven lifecycle discovery and assisted Tool `0.3.5` migration remain under development.

## Repository map

| Area | Location | Purpose |
| --- | --- | --- |
| Normative Specification | [`spec/`](spec/) | Lifecycle ownership, architecture, terminology, Responsibility Map, and conformance rules. |
| Responsibility Map | [`spec/PTSIP-RESPONSIBILITY-MAP.md`](spec/PTSIP-RESPONSIBILITY-MAP.md) | Responsibility Map v2 declaration/materialization semantics. |
| Machine-readable registry | [`registry/`](registry/) | Canonical terms, rule IDs, and metadata. |
| Schemas | [`schemas/`](schemas/) | Project Profile and interoperability schemas. |
| Agent contract | [`agents/AGENT-CONTRACT.md`](agents/AGENT-CONTRACT.md) | Coding-agent operational contract. |
| Adoption guide | [`adoption/ADOPTION-GUIDE.md`](adoption/ADOPTION-GUIDE.md) | Controlled project-adoption sequence. |
| Reference architecture | [`reference/REFERENCE-ARCHITECTURE.md`](reference/REFERENCE-ARCHITECTURE.md) | Informative architecture guidance. |
| ADRs | [`decisions/`](decisions/) | Architecture decisions. |
| Embedded Specification data | [`src/ptsip/specdata/`](src/ptsip/specdata/) | Tool-packaged schema/registry resources. |
| Reference Tool | [`src/ptsip/`](src/ptsip/) | Installable Python PTSIP implementation. |
| VPMS | [`src/vpms/`](src/vpms/) | Verification Purpose Management implementation. |
| Tests | [`tests/`](tests/) | Tool, contract, integration, and VPMS verification. |
| Release notes | [`releasenote/`](releasenote/) | Tool/Specification release and draft history. |
| Development plan | [`planning/0.3.6.md`](planning/0.3.6.md) | Tool `0.3.6` Work Unit plan and development sequence. |

## Key Specification documents

- [`spec/PTSIP-SPEC.md`](spec/PTSIP-SPEC.md)
- [`spec/PTSIP-RESPONSIBILITY-MAP.md`](spec/PTSIP-RESPONSIBILITY-MAP.md)
- [`spec/PTSIP-CONFORMANCE.md`](spec/PTSIP-CONFORMANCE.md)
- [`spec/PTSIP-TERMINOLOGY.md`](spec/PTSIP-TERMINOLOGY.md)
- [`schemas/ptsip-profile.schema.json`](schemas/ptsip-profile.schema.json)
- [`registry/ptsip-registry.yaml`](registry/ptsip-registry.yaml)
- [`decisions/ADR-0007-primary-lifecycle-boundary-determination.md`](decisions/ADR-0007-primary-lifecycle-boundary-determination.md)
- [`decisions/ADR-0008-responsibility-roles-relationships-associated-artifacts.md`](decisions/ADR-0008-responsibility-roles-relationships-associated-artifacts.md)
- [`decisions/ADR-0009-responsibility-map-declaration-authority.md`](decisions/ADR-0009-responsibility-map-declaration-authority.md)

## Release namespaces

Tool releases use:

```text
tool-v*
```

Specification releases and design records use:

```text
spec-v*
```

The exact normative identity of a mutable draft remains its immutable Specification revision, not the tag or family string alone.

## Maturity

PTSIP is a draft project-defined architecture policy.

It is not an ISO, IEEE, IETF, CNCF, or other external industry standard.

Tool `0.3.6` remains under active development.

## License

This repository, including the PTSIP Specification and Reference Tool unless explicitly stated otherwise, is licensed under the Apache License, Version 2.0.

See [`LICENSE`](LICENSE).
