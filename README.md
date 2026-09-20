<p align="right">
  English | <a href="README.ko.md">한국어</a>
</p>

# PTSIP — Primary Lifecycle Ownership and Responsibility Isolation Policy

**Status:** Tool `0.3.8a1` emergency prerelease — publication pending<br>
**Tool/package version:** `0.3.8a1`<br>
**Project Profile contract:** `pp.1.02`<br>
**Specification family:** `0.3.7-draft`<br>
**Bound immutable Specification revision:** `3c47816770d194ae42f98faedc911d980db0e62a`<br>
**License authority:** [`License-Authority/`](License-Authority/) (closed authority boundary; references elsewhere are non-authoritative)<br>
**License effective time:** `2026-09-15T00:00:00Z` (UTC)<br>
**Localized documentation:** `README.md` is canonical. Localized README files are regenerated on `main` by the self-hosted Argos Translate workflow; if a translation conflicts with this file, this file governs.

PTSIP is a project-defined architecture policy for separating project responsibilities by **primary lifecycle ownership** while preserving explicit architecture intent, lifecycle isolation, reproducible conformance, verification-purpose separation, and multi-environment decision consistency.

> **Purpose precedes reuse.** Classify a coherent responsibility by why it exists and which lifecycle owns it before optimizing for code sharing.

Tool `0.3.8a1` is a narrow emergency prerelease bridge built on the Tool `0.3.7` baseline and the same frozen Specification binding. It adds a supported way to represent a component that does not yet exist as an explicit proposed candidate, resolve that proposal through the existing Decision Control Plane, and retain an approved proposal as `PROPOSAL_APPROVED` without projecting it into active `components[]` before materialization. Tool `0.3.8a1` is intentionally not the completion of Tool `0.4.0`.

## Primary lifecycle ownership

Canonical Tool `0.3.8a1` classifications remain exactly:

| Classification | Meaning |
| --- | --- |
| `PRODUCT` | Responsibility primarily owned by the Product lifecycle. |
| `DEVELOPMENT_TOOLING` | Development-lifecycle responsibility used to create, inspect, validate, transform, generate, migrate, analyze, test, or otherwise support development. |
| `DELIVERY` | Responsibility for release preparation, packaging, publication, promotion, distribution, or deployment to the delivery destination. |
| `OPERATIONS` | Ongoing post-delivery responsibility for health, recovery, reconciliation, maintenance, and operation. |
| `NEUTRAL_CONTRACT` | Deliberately non-executable, non-owning contract responsibility with lifecycle-independent governance. |

`UNKNOWN`, `CONFLICT`, `INCOMPLETE`, `PENDING`, confidence values, and migration states are workflow/evaluation states, not additional classifications.

### Tool 0.3.5 compatibility boundary

Tool `0.3.5` historically used:

```text
PRODUCT
TOOLCHAIN
NEUTRAL_CONTRACT
```

Tool `0.3.8a1` preserves the five-classification model established by Tool `0.3.6`. `TOOLCHAIN` is therefore **legacy Tool `0.3.5` input**, not a current canonical alias. A legacy Toolchain responsibility may become `DEVELOPMENT_TOOLING`, `DELIVERY`, `OPERATIONS`, or require a split depending on its actual lifecycle ownership. Blind `TOOLCHAIN -> DEVELOPMENT_TOOLING` rewriting is prohibited.

Tool `0.3.8a1` provides evidence-bound direct current-target migration for explicitly supported historical sources. Migration capability remains separate from repository adoption authority and never turns inference into project intent.

## Classification is not path or technology

Classification follows the governing lifecycle obligation, not filename, directory, language, framework, executable status, workflow provider, compilation behavior, runtime duration, test status, or confidence score.

Examples:

```text
Product-specific verification responsibility       -> PRODUCT
Reusable verification framework / test SDK         -> DEVELOPMENT_TOOLING
Product runtime implementation                      -> PRODUCT
Release-unit assembly / publication automation      -> DELIVERY
Post-deployment health or recovery automation       -> OPERATIONS
Independent non-executable shared contract          -> NEUTRAL_CONTRACT
```

Paths such as `tests/`, `tools/`, `deploy/`, `ops/`, or `.github/workflows/` are evidence context only. They do not become architecture authority.

## Responsibility Map v2

Tool `0.3.8a1` uses Responsibility Map v2 as the project-owned architecture declaration model. It keeps several axes independent:

```text
classification
    = primary lifecycle ownership

roles
    = coarse responsibility characteristics

relationships
    = project-owned typed directed semantics

source/derived provenance
    = where declaration/materialized architecture came from

VPMS Verification Purpose
    = why verification exists and what it protects
```

Canonical roles are:

```text
IMPLEMENTATION
VERIFICATION
AUTOMATION
CONFIGURATION
DOCUMENTATION
GOVERNANCE
```

Canonical typed relationships are:

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

An associated artifact is a project-owned non-component support surface subordinate to one classified anchor component. It must not be used to hide independently governable executable or lifecycle responsibility.

## Explicit, template, and hybrid declarations

Canonical source modes are:

```text
explicit
    repository directly declares the complete map

template
    repository explicitly selects one immutable revision-bound template

hybrid
    repository explicitly selects a template and adds project-owned
    overrides, extensions, or removals
```

The initial template catalog contains:

```text
python-package-library
python-cli-application
mixed-product-development-delivery
```

Template selection is explicit. PTSIP does not automatically select a template from repository layout, language, framework detection, manifests, or confidence.

## Source declaration and Effective Responsibility Map

All source modes resolve through deterministic, non-authoritative materialization:

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
Canonical Effective Responsibility Map
        |
        +--> validation / conformance
        +--> clarification / adoption
        +--> narrow VPMS read-only projection
```

The Source Project Profile remains project-owned architecture authority. Materialization must not select templates, infer ownership, repair invalid architecture, or silently rewrite the source declaration.

The resolved view retains declaration provenance such as:

```text
PROJECT_EXPLICIT
TEMPLATE
PROJECT_OVERRIDE
PROJECT_EXTENSION
PROJECT_REMOVAL
```

and has deterministic digest identity for reproducibility. Neither provenance nor digest becomes a replacement architecture authority.

## Install and use

PTSIP requires Python 3.11 or newer.

Install the latest **published** release:

```powershell
python -m pip install PTSIP
```

Upgrade to the latest **published** release:

```powershell
python -m pip install --upgrade PTSIP
```

Tool `0.3.8a1` is a prerelease. A normal `pip install PTSIP` may continue to select the latest stable release unless prereleases are explicitly requested.

After publication, install this prerelease explicitly with:

```powershell
python -m pip install "PTSIP==0.3.8a1"
```

For source development on this release line:

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

New project-owned profiles are selected through `.ptsip/profiles/index.yaml`; the catalog's `default_profile` resolves the active `*.ptsip.yaml` resource. Repository-root `ptsip.yaml` remains a compatibility/migration input, and an explicit `--profile` path still takes precedence.

## Adoption and decision authority

Repository evidence is not architecture authority. Candidate discovery, path names, templates, heuristics, and agent confidence can support review but cannot manufacture project intent.

Canonical Tool `0.3.8a1` explicit adoption facts center on `classification` as lifecycle ownership authority. New canonical decisions use facts such as:

```text
classification
purpose
shipped
runtime_required
executable
```

The historical `lifecycle_owner` field is legacy migration evidence, not a second Tool `0.3.8a1` ownership authority.

Example dry-run:

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

Apply only after reviewing the planned declaration change:

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

Prepared writes must reject stale repository/profile state.

PTSIP distinguishes four things that must not be collapsed:

```text
Specification
    -> normative rules

Decision Authority
    -> which explicit coordinated architecture answer won

Project Profile / Responsibility Map
    -> durable project-owned declaration

Observed evidence
    -> what the repository and artifacts actually do
```

A Decision Authority does not replace the Project Profile selected by `.ptsip/profiles/index.yaml` and does not prove conformance. Legacy root `ptsip.yaml` remains compatibility input only.

## Distributed decision coordination

The Reference Tool supports repository-distributed decision coordination through:

```text
refs/heads/ptsip-policy
```

GitHub is a Tool backend, not a universal Specification dependency. The coordination model preserves stable decision identity, first-valid-resolution-wins, stale-writer-safe conditional mutation, authority freshness, deterministic reconciliation, fail-closed behavior, and separation of global decision state from clone-local application state.

PTSIP uses action-time synchronization rather than continuous background polling.

## Product Artifact boundary

Artifact ownership is independent from producer ownership. A `DEVELOPMENT_TOOLING` or `DELIVERY` component may validly build a `PRODUCT` artifact, but the resulting artifact must still satisfy the Product package boundary.

Tool `0.3.8a1` supports snapshot-bound Product Artifact evidence. Release verification checks actual built distribution content rather than treating packaging configuration as proof. Product distribution verification rejects definite non-Product implementation leakage under `PTSIP-PKG-001`.

## VPMS — Verification Purpose Management System

PTSIP and VPMS answer different questions:

```text
PTSIP
    Who owns this responsibility across its lifecycle?

VPMS
    Why does this Verification Case exist, and what does it protect?
```

PTSIP classification and VPMS Verification Purpose remain separate axes. PTSIP core does not depend on VPMS. VPMS consumes only a narrow read-only projection of already-resolved PTSIP metadata.

The current VPMS compatibility vocabulary may still contain `PRODUCT | TOOLCHAIN`. VPMS `TOOLCHAIN` is not a canonical Tool `0.3.8a1` PTSIP classification.

VPMS verification PASS does not imply PTSIP `CONFORMANT`, and PTSIP `CONFORMANT` does not imply functional verification PASS.

## Conformance

`ptsip conform` evaluates declared architecture and observed evidence against applicable PTSIP rules after source declarations are resolved to the Effective Responsibility Map.

Completed outcomes are:

| Exit code | Outcome |
| --- | --- |
| `0` | `CONFORMANT` |
| `5` | `NON_CONFORMANT` |
| `6` | `INCOMPLETE` |

A valid profile does not prove conformance. Missing evidence that could hide an applicable mandatory rule remains fail-closed as `INCOMPLETE`; the Tool does not force an uncertain repository green.

## Tool and Specification lifecycle

The PTSIP Tool and PTSIP Specification are independently versioned.

- `pyproject.toml` owns Tool/package source version;
- `ptsip --version` reports installed Tool version;
- `ptsip spec` reports the exact Specification family and immutable revision bound to the Tool;
- `spec/`, `schemas/`, and `registry/` contain canonical Specification assets;
- `src/ptsip/specdata/` contains matching embedded machine-readable assets.

The current source tree exposes independent PP and Specification identities:

```text
Project Profile pp.1.02
Specification 0.3.7-draft
SPEC_REVISION 3c47816770d194ae42f98faedc911d980db0e62a
```

The already-published Tool `0.3.8a1` release retains its historical PP binding
in its Tool release note; advancing the PP contract does not rewrite that Tool history.

A new immutable revision is required only for a genuine normative change. Release workflow, test, planning, status, or documentation-only changes do not move `SPEC_REVISION` by themselves.

## Tool 0.3.8a1 release identity

Tool `0.3.8a1` is an emergency bridge prerelease with an intentionally narrow scope.

```text
Tool:             0.3.8a1
Project Profile:  pp.1.01
Specification:    0.3.7-draft
SPEC_REVISION:    3c47816770d194ae42f98faedc911d980db0e62a
Release scope:    explicit proposed candidate bridge
```

This Tool release does not introduce a new Specification family or Project Profile contract.

It adds support for representing a not-yet-existing component as an explicit proposed candidate and resolving that proposal without silently materializing it as an active component. It does not claim completion of the broader Tool `0.4.0` capability-recovery architecture.

Release scope and verification history are recorded in [`releasenote/tool/0.3.8a1.md`](releasenote/tool/0.3.8a1.md). The release gate is defined by [`planning/0.3.8/0.3.8a1-emergency-release-gate.yaml`](planning/0.3.8/0.3.8a1-emergency-release-gate.yaml).

## License authority

PTSIP licensing authority is contained exclusively under [`License-Authority/`](License-Authority/). The machine-readable entry point is [`License-Authority/license-authority.yaml`](License-Authority/license-authority.yaml), and the controlling legal text is [`License-Authority/LICENSE.md`](License-Authority/LICENSE.md).

The canonical effective time currently declared by the License Authority is `2026-09-15T00:00:00Z` (UTC).

License-related wording elsewhere in this repository is informational or contextual only. It does not become part of the License, modify it, or become incorporated by reference.

Normal Tool releases, Specification validation, Tool version changes, documentation mentions, keyword matches, and AI semantic guesses do not authorize License Authority entry. Entry uses only the triggers declared by `License-Authority/license-authority.yaml`.
## Consumer Repository non-intrusion

PTSIP does not require Consumer Repositories to create PTSIP-specific `.ptsip/`, cache, report, or hidden state directories merely to use the Tool. External inspection and Pilot operations are read-only by default. Tool-owned local state belongs outside the Consumer Repository unless a repository path is explicitly chosen.

## Project status

PTSIP remains experimental. Tool `0.3.8a1` is an emergency prerelease whose publication boundary is tracked in [`releasenote/tool/0.3.8a1.md`](releasenote/tool/0.3.8a1.md). Historical Tool releases and Specification notes are preserved under [`releasenote/`](releasenote/).
