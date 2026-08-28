<p align="right">
  English | <a href="README.ko.md">한국어</a>
</p>

# PTSIP — Primary Lifecycle Ownership and Responsibility Isolation Policy

**Status:** Tool `0.3.7` WU-12 implementation integrated — exact-SHA verification pending<br>
**Tool/package version:** `0.3.7`<br>
**Project Profile contract:** `pp.1.01`<br>
**Specification family:** `0.3.7-draft`<br>
**Bound immutable Specification revision:** `3c47816770d194ae42f98faedc911d980db0e62a`<br>
**License:** Apache License 2.0

PTSIP is a project-defined architecture policy for separating project responsibilities by **primary lifecycle ownership** while preserving explicit architecture intent, lifecycle isolation, reproducible conformance, verification-purpose separation, and multi-environment decision consistency.

> **Purpose precedes reuse.** Classify a coherent responsibility by why it exists and which lifecycle owns it before optimizing for code sharing.

Tool `0.3.7` now carries the WU-12 implementation and frozen Specification binding on `dev/0.3.7`. It is not yet verified or published: a fresh self-hosted workflow must pass for the final exact source SHA before release preparation. The latest published PyPI package remains Tool `0.3.5` until a later publication boundary succeeds.

## Primary lifecycle ownership

Canonical Tool `0.3.7` classifications remain exactly:

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

Tool `0.3.7` preserves the five-classification model established by Tool `0.3.6`. `TOOLCHAIN` is therefore **legacy Tool `0.3.5` input**, not a current canonical alias. A legacy Toolchain responsibility may become `DEVELOPMENT_TOOLING`, `DELIVERY`, `OPERATIONS`, or require a split depending on its actual lifecycle ownership. Blind `TOOLCHAIN -> DEVELOPMENT_TOOLING` rewriting is prohibited.

Tool `0.3.7` provides evidence-bound direct current-target migration for explicitly supported historical sources. Migration capability remains separate from repository adoption authority and never turns inference into project intent.

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

Tool `0.3.7` uses Responsibility Map v2 as the project-owned architecture declaration model. It keeps several axes independent:

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

Until Tool `0.3.7` is published, those commands may still install Tool `0.3.5` from PyPI. For source development on this release-candidate line:

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

The default project-owned profile is repository-root `ptsip.yaml`; projects may consistently use another explicit path through `--profile`.

## Adoption and decision authority

Repository evidence is not architecture authority. Candidate discovery, path names, templates, heuristics, and agent confidence can support review but cannot manufacture project intent.

Canonical Tool `0.3.7` explicit adoption facts center on `classification` as lifecycle ownership authority. New canonical decisions use facts such as:

```text
classification
purpose
shipped
runtime_required
executable
```

The historical `lifecycle_owner` field is legacy migration evidence, not a second Tool `0.3.7` ownership authority.

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

A Decision Authority does not replace `ptsip.yaml` and does not prove conformance.

## Distributed decision coordination

The Reference Tool supports repository-distributed decision coordination through:

```text
refs/heads/ptsip-policy
```

GitHub is a Tool backend, not a universal Specification dependency. The coordination model preserves stable decision identity, first-valid-resolution-wins, stale-writer-safe conditional mutation, authority freshness, deterministic reconciliation, fail-closed behavior, and separation of global decision state from clone-local application state.

PTSIP uses action-time synchronization rather than continuous background polling.

## Product Artifact boundary

Artifact ownership is independent from producer ownership. A `DEVELOPMENT_TOOLING` or `DELIVERY` component may validly build a `PRODUCT` artifact, but the resulting artifact must still satisfy the Product package boundary.

Tool `0.3.7` supports snapshot-bound Product Artifact evidence. Release verification checks actual built distribution content rather than treating packaging configuration as proof. Product distribution verification rejects definite non-Product implementation leakage under `PTSIP-PKG-001`.

## VPMS — Verification Purpose Management System

PTSIP and VPMS answer different questions:

```text
PTSIP
    Who owns this responsibility across its lifecycle?

VPMS
    Why does this Verification Case exist, and what does it protect?
```

PTSIP classification and VPMS Verification Purpose remain separate axes. PTSIP core does not depend on VPMS. VPMS consumes only a narrow read-only projection of already-resolved PTSIP metadata.

The current VPMS compatibility vocabulary may still contain `PRODUCT | TOOLCHAIN`. VPMS `TOOLCHAIN` is not a canonical Tool `0.3.7` PTSIP classification.

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

Tool `0.3.7` is bound to independent PP and Specification identities:

```text
Project Profile pp.1.01
Specification 0.3.7-draft
SPEC_REVISION 3c47816770d194ae42f98faedc911d980db0e62a
```

A new immutable revision is required only for a genuine normative change. Release workflow, test, planning, status, or documentation-only changes do not move `SPEC_REVISION` by themselves.

## Tool 0.3.7 verification and release state

The WU-12 implementation and immutable Specification freeze are present locally, but WU-12 is not `COMPLETE` or `VERIFIED`. The required final exact-SHA `tooling-test` run has not occurred for this source state.

```text
Tool:              0.3.7
Project Profile:   pp.1.01
Specification:     0.3.7-draft @ 3c47816770d194ae42f98faedc911d980db0e62a
exact source SHA:  pending final integration commit
tooling-test:      NOT RUN for the final exact SHA
publication:       NOT RUN
```

Earlier Tool `0.3.6` workflow evidence remains historical evidence for its own SHA and does not verify Tool `0.3.7`.

The remaining release boundary is:

```text
final dev/0.3.7 exact SHA
    -> tooling-test.yml on that exact SHA
    -> require self-hosted/tooling-test success
    -> full regression, independent identity, distribution, artifact, and wheel smoke PASS
    -> later exact-main release handoff
    -> release.yml and reviewed draft publication from the same source identity
    -> tooling-release.yml publication verification and PyPI Trusted Publishing
```

See [`STATUS.md`](STATUS.md), [`planning/0.3.7/WU-12-specification-binding-capability-registry-release-readiness.md`](planning/0.3.7/WU-12-specification-binding-capability-registry-release-readiness.md), and [`releasenote/tool/0.3.7.md`](releasenote/tool/0.3.7.md) for the current implementation and handoff boundary.

## Consumer Repository non-intrusion

PTSIP does not require Consumer Repositories to create PTSIP-specific `.ptsip/`, cache, report, or hidden state directories merely to use the Tool. External inspection and Pilot operations are read-only by default. Tool-owned local state belongs outside the Consumer Repository unless a repository path is explicitly chosen.

## Project status

PTSIP remains experimental. Tool `0.3.7` is an unverified, unpublished release candidate until the final exact-SHA completion gate succeeds. Historical Tool releases and Specification notes are preserved under [`releasenote/`](releasenote/).
