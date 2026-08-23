<p align="right">
  English | <a href="README.ko.md">한국어</a>
</p>

# PTSIP — Primary Lifecycle Ownership and Responsibility Isolation Policy

**Status:** Tool `0.3.6` development complete — pre-publication release candidate  
**Tool/package version:** `0.3.6`  
**Specification family:** `0.3.6-draft`  
**Bound immutable Specification revision:** `d6995ed232e845b88d8235b851e80ab54b7804ea`  
**License:** Apache License 2.0

PTSIP is a project-defined architecture policy for separating project responsibilities by **primary lifecycle ownership** while preserving explicit architecture intent, lifecycle isolation, reproducible conformance, verification-purpose separation, and multi-environment decision consistency.

> **Purpose precedes reuse.** Classify a coherent responsibility by why it exists and which lifecycle owns it before optimizing for code sharing.

Tool `0.3.6` development is complete on the release-candidate line. The remaining boundary is release operations: merge the approved state to `main`, verify that exact `main` SHA on the self-hosted gate, then prepare and publish the release from the same source identity. The latest PyPI package remains Tool `0.3.5` until Tool `0.3.6` is actually published.

## Primary lifecycle ownership

Canonical Tool `0.3.6` classifications are exactly:

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

Tool `0.3.6` uses the five-classification model above. `TOOLCHAIN` is therefore **legacy Tool `0.3.5` input**, not a Tool `0.3.6` alias. A legacy Toolchain responsibility may become `DEVELOPMENT_TOOLING`, `DELIVERY`, `OPERATIONS`, or require a split depending on its actual lifecycle ownership. Blind `TOOLCHAIN -> DEVELOPMENT_TOOLING` rewriting is prohibited.

Evidence-driven assisted migration from Tool `0.3.5` declarations is intentionally deferred to Tool `0.3.6.1`; it is not claimed as a Tool `0.3.6` release capability.

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

Tool `0.3.6` uses Responsibility Map v2 as the project-owned architecture declaration model. It keeps several axes independent:

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

Until Tool `0.3.6` is published, those commands may still install Tool `0.3.5` from PyPI. For source development on this release-candidate line:

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

Canonical Tool `0.3.6` explicit adoption facts center on `classification` as lifecycle ownership authority. New canonical decisions use facts such as:

```text
classification
purpose
shipped
runtime_required
executable
```

The historical `lifecycle_owner` field is legacy migration evidence, not a second Tool `0.3.6` ownership authority.

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

Tool `0.3.6` supports snapshot-bound Product Artifact evidence. Release verification checks actual built distribution content rather than treating packaging configuration as proof. Product distribution verification rejects definite non-Product implementation leakage under `PTSIP-PKG-001`.

## VPMS — Verification Purpose Management System

PTSIP and VPMS answer different questions:

```text
PTSIP
    Who owns this responsibility across its lifecycle?

VPMS
    Why does this Verification Case exist, and what does it protect?
```

PTSIP classification and VPMS Verification Purpose remain separate axes. PTSIP core does not depend on VPMS. VPMS consumes only a narrow read-only projection of already-resolved PTSIP metadata.

The current VPMS compatibility vocabulary may still contain `PRODUCT | TOOLCHAIN`. VPMS `TOOLCHAIN` is not a canonical Tool `0.3.6` PTSIP classification.

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

Tool `0.3.6` is bound to:

```text
Specification 0.3.6-draft
SPEC_REVISION d6995ed232e845b88d8235b851e80ab54b7804ea
```

A new immutable revision is required only for a genuine normative change. Release workflow, test, planning, status, or documentation-only changes do not move `SPEC_REVISION` by themselves.

## Tool 0.3.6 verification and release state

Development-branch WU-00 through WU-07 are complete. Final WU-07 exact-SHA verification authority is:

```text
source SHA:       452d0f8b0c78bdebb180ceb2b9994485f59eb43a
workflow run/job: 32640319047 / 97196299107
runner:           self-hosted Windows X64
Python:           3.14.6
pytest:           331 passed / 0 failed
Specification:    0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea
profile coverage: unassigned_count=0
build/twine:      PASS
Product Artifact: PASS / exact snapshot binding
PTSIP-PKG-001:    0 definite violations
wheel smoke/VPMS: PASS
commit status:    self-hosted/tooling-test = success
```

Documentation commits after that successful run record closure and do not replace the exact verification authority.

The remaining release boundary is:

```text
approved 0.3.6 state -> main
    -> read exact main SHA
    -> tooling-test.yml on that exact main SHA
    -> require self-hosted/tooling-test success
    -> release.yml from the same current main SHA
    -> exact Tool / Specification / release-document contract verification
    -> draft GitHub Release targeting the same SHA
    -> maintainer publishes reviewed draft
    -> tooling-release.yml verifies distributions from the published tag
    -> PyPI Trusted Publishing
```

See [`STATUS.md`](STATUS.md), [`planning/0.3.6.md`](planning/0.3.6.md), and [`releasenote/0.3.6.md`](releasenote/0.3.6.md) for the release evidence and handoff state.

## Consumer Repository non-intrusion

PTSIP does not require Consumer Repositories to create PTSIP-specific `.ptsip/`, cache, report, or hidden state directories merely to use the Tool. External inspection and Pilot operations are read-only by default. Tool-owned local state belongs outside the Consumer Repository unless a repository path is explicitly chosen.

## Project status

PTSIP remains experimental. Tool `0.3.6` is development-complete but not yet published at the time of this release-candidate documentation update. Historical Tool releases and Specification notes are preserved under [`releasenote/`](releasenote/).
