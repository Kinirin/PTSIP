<p align="right">
  English | <a href="docs/translated/README.ko.md">한국어</a>
</p>

# PTSIP — Primary Lifecycle Ownership and Responsibility Isolation Policy

**Status:** Tool `0.3.8a3` Context Plane migration-semantics emergency prerelease — publication pending<br>
**Tool/package version:** `0.3.8a3`<br>
**Project Profile contract:** `pp.1.02`<br>
**Specification family:** `0.3.7-draft`<br>
**Bound immutable Specification revision:** `3c47816770d194ae42f98faedc911d980db0e62a`<br>
**License authority:** [`License-Authority/`](License-Authority/) (closed authority boundary; references elsewhere are non-authoritative)<br>
**License effective time:** `2026-09-15T00:00:00Z` (UTC)<br>
**Localized documentation:** `README.md` is canonical. Localized README files are regenerated on `main` by the self-hosted Argos Translate workflow; if a translation conflicts with this file, this file governs.

PTSIP is a project-defined architecture policy for separating project responsibilities by **primary lifecycle ownership** while preserving explicit architecture intent, lifecycle isolation, reproducible conformance, verification-purpose separation, and multi-environment decision consistency.

> **Purpose precedes reuse.** Classify a coherent responsibility by why it exists and which lifecycle owns it before optimizing for code sharing.

Tool `0.3.8a3` is an emergency corrective prerelease over the published `0.3.8a2` repository-namespace release. Tool `0.3.8a2` established `.ptsip/` and the provider-neutral Context Plane at repository level, but its Context Projection implementation remained under `developer/automation/` and therefore was not shipped in the installed Consumer Tool. Tool `0.3.8a3` closes that packaging and migration gap by shipping the canonical Context Plane implementation as `ptsip.context_plane`, exposing `ptsip context status|migrate|repair|check`, and binding release verification to the installed wheel. The Specification and Project Profile identities remain unchanged.

## Primary lifecycle ownership

Canonical Tool `0.3.8a3` classifications remain exactly:

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

Tool `0.3.8a3` preserves the five-classification model established by Tool `0.3.6`. `TOOLCHAIN` is therefore **legacy Tool `0.3.5` input**, not a current canonical alias. A legacy Toolchain responsibility may become `DEVELOPMENT_TOOLING`, `DELIVERY`, `OPERATIONS`, or require a split depending on its actual lifecycle ownership. Blind `TOOLCHAIN -> DEVELOPMENT_TOOLING` rewriting is prohibited.

Tool `0.3.8a3` provides evidence-bound direct current-target migration for explicitly supported historical sources. Migration capability remains separate from repository adoption authority and never turns inference into project intent.

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

Tool `0.3.8a3` uses Responsibility Map v2 as the project-owned architecture declaration model. It keeps several axes independent:

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

Tool `0.3.8a3` is a prerelease. A normal `pip install PTSIP` may continue to select the latest stable release unless prereleases are explicitly requested.

After publication, install this prerelease explicitly with:

```powershell
python -m pip install "PTSIP==0.3.8a3"
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
ptsip context status . --json
ptsip context check . --json
```

New project-owned profiles are selected through `.ptsip/profiles/index.json`; the catalog's `default_profile` resolves the active `*.ptsip.yaml` resource. Repository-root `ptsip.yaml` remains a compatibility/migration input, and an explicit `--profile` path still takes precedence.

PTSIP developers use that same local catalog and `.ptsip/profiles/main.ptsip.yaml`.
Profile implementation modules and distributed examples live in
`src/ptsip/profiles/`. The public catalog is `src/ptsip/profiles/index.yaml`;
`registry/project-profile-contracts.yaml` explicitly binds this source layout.
Historical PP baselines under `src/ptsip/profiles/history/` are preserved in
source distributions for verification and are excluded from wheels. Moving
storage preserves PP identity and historical bytes; public profile or schema
semantic changes still require the native Go PP transition gates.

## Adoption and decision authority

Repository evidence is not architecture authority. Candidate discovery, path names, templates, heuristics, and agent confidence can support review but cannot manufacture project intent.

Canonical Tool `0.3.8a3` explicit adoption facts center on `classification` as lifecycle ownership authority. New canonical decisions use facts such as:

```text
classification
purpose
shipped
runtime_required
executable
```

The historical `lifecycle_owner` field is legacy migration evidence, not a second Tool `0.3.8a3` ownership authority.

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

A Decision Authority does not replace the Project Profile selected by `.ptsip/profiles/index.json` and does not prove conformance. Legacy root `ptsip.yaml` remains compatibility input only.

## Distributed decision coordination

The Reference Tool supports repository-distributed decision coordination through:

```text
refs/heads/ptsip-policy
```

GitHub is a Tool backend, not a universal Specification dependency. The coordination model preserves stable decision identity, first-valid-resolution-wins, stale-writer-safe conditional mutation, authority freshness, deterministic reconciliation, fail-closed behavior, and separation of global decision state from clone-local application state.

PTSIP uses action-time synchronization rather than continuous background polling.

## Product Artifact boundary

Artifact ownership is independent from producer ownership. A `DEVELOPMENT_TOOLING` or `DELIVERY` component may validly build a `PRODUCT` artifact, but the resulting artifact must still satisfy the Product package boundary.

Tool `0.3.8a3` supports snapshot-bound Product Artifact evidence. Release verification checks actual built distribution content rather than treating packaging configuration as proof. Product distribution verification rejects definite non-Product implementation leakage under `PTSIP-PKG-001`.

## VPMS — Verification Protocol Management System

PTSIP and VPMS answer different questions:

```text
PTSIP
    Who owns this responsibility across its lifecycle?

VPMS
    How are explicit Verification Cases bound, executed, and reported?
```

PTSIP classification and the historical VPMS Verification Purpose remain separate axes. The Case `purpose` field is compatibility-only: it does not decide selection, runner binding, or PTSIP classification. PTSIP core does not depend on VPMS.

The repository's three product contracts are ACTIVE and separately own the Case/reference/runner protocol, explicit Case selection, and execution composition. The public API is:

```python
from vpms import load_registry_snapshot, resolve_selection, run_cases

loaded = load_registry_snapshot(case_definitions, references=reference_registrations)
if not loaded.ok:
    raise ValueError(loaded.diagnostics)
selection = resolve_selection(
    loaded.snapshot, {"kind": "CASE_IDS", "case_ids": ["explicit.case.id"]}
)
if not selection.ok:
    raise ValueError(selection.diagnostics)
results = run_cases(loaded.snapshot, selection, executors=runner_adapters)
```

`case_definitions`, `reference_registrations`, and `runner_adapters` are explicit caller-owned inputs. Selection fails closed on invalid, duplicate, or unknown IDs. Execution accepts selection bound to the same validated snapshot and preflights every selected adapter before executing the first Case. `SelectionScope`, `select_cases`, and `run_selected_cases` are retired, not compatibility aliases.

Optional PTSIP integration uses `ptsip.validation.handoff.load_validated_effective_map` followed by `vpms.integration.ptsip_bridge.metadata_from_effective_map`. ACTIVE `SFP-0023` owns only this read-only boundary: the projection exposes exact component ID and classification from a genuinely validated immutable effective map. A raw dict, a `validated=True` assertion, and the historical `load_ptsip_metadata` compatibility reader are not canonical validation proof or implicit fallbacks. `SFP-0006` is RETIRED with its historical meaning preserved; VPMS product protocol authority resides in the separate product contracts.

The current VPMS compatibility vocabulary may still contain `PRODUCT | TOOLCHAIN`. VPMS `TOOLCHAIN` is not a canonical Tool `0.3.8a3` PTSIP classification.

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

## Dependency analysis and bounded review

The WU-13 development CLI reconciles dependency evidence before presenting items for review:

```powershell
ptsip dependency analyze .
ptsip dependency analyze . --component ptsip-core --json
ptsip dependency review-pack . --max-items 8 --max-context-bytes 12000
ptsip dependency validation-plan . --changed src/ptsip/dependency_analysis.py --json
```

`--component` requires a component ID declared in the selected Project Profile. All three commands accept `--profile`; omit `--component` to analyze the repository. The examples using `ptsip-core` and `src/ptsip/dependency_analysis.py` apply to this repository and must be replaced with consumer-owned component IDs and tracked paths elsewhere.

Analysis reports four advisory actionability buckets:

| Bucket | Next action |
| --- | --- |
| `AUTO_RESOLVED` | Reuse the recorded mechanical evidence. |
| `REPOSITORY_DEFECT` | Review the evidence-backed remediation candidate; application remains explicit. |
| `REVIEW_REQUIRED` | Review bounded source context and the unresolved support or ownership question. |
| `RESOLVER_LIMITATION` | Retain incomplete evidence and address the resolver or supply authoritative evidence. |

The default human output is concise. `--json` preserves the detailed machine report; keep large reports in an external working directory instead of placing the entire report in an AI prompt. A Review Pack includes only `REVIEW_REQUIRED` items, defaults to at most eight items and 12,000 UTF-8 bytes per item, and limits each item to four source files. Its summary distinguishes selected and deferred items. An item is deferred when its import snippet cannot fit the context budget. Generation does not perform AI review; record actual review counts separately from the generator's `ai_reviewed_items: 0`.

Review Packs and reusable Python source evidence are stored in external Tool-owned state by default. `review-pack --output <new-report.json>` chooses an explicit report path; it rejects overwriting tracked repository content. Source evidence is reused only when its cache identity matches, while repository snapshots are checked freshly. Corrupt or stale cache entries are recomputed.

`validation-plan` proposes focused and component commands from declared verification ownership and observed import reachability. Repeat `--changed` for multiple tracked paths. It does not execute the commands or replace the repository's full regression and exact-SHA verification requirements.

Dependency analysis and Review Packs do not evaluate conformance or establish architecture authority. Run `ptsip conform .` for the strict outcome. Unresolved verification dependencies remain blocking in a separate verification bucket; dynamic or ambiguous local imports remain unresolved when their targets cannot be established mechanically. A support-contract decision and any consumer change remain explicit.

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

## Tool 0.3.8a3 release identity

Tool `0.3.8a3` is an emergency Context Plane migration-semantics patch over the published `0.3.8a2` repository-namespace release.

```text
Tool:             0.3.8a3
Project Profile:  pp.1.02
Specification:    0.3.7-draft
SPEC_REVISION:    3c47816770d194ae42f98faedc911d980db0e62a
Release scope:    deterministic MEMORY.md -> Context Plane migration and projection repair
```

The canonical semantic write target is `.ptsip/context/source/context.source.json`. Generated `context.json`, `context.jsonl`, and `context.schema.json` are repaired from that source. Semantic ambiguity remains user-owned; deterministic projection repair is preauthorized for coding agents. Unlike Tool `0.3.8a2`, Tool `0.3.8a3` ships this implementation inside the Consumer Tool as `ptsip/context_plane.py`; a release build is invalid if that module or the `ptsip context` CLI surface is absent from the built wheel.

## Tool 0.3.8a2 release identity

Tool `0.3.8a2` is a repository-namespace stabilization prerelease with a deliberately narrow compatibility goal.

```text
Tool:             0.3.8a2
Project Profile:  pp.1.02
Specification:    0.3.7-draft
SPEC_REVISION:    3c47816770d194ae42f98faedc911d980db0e62a
Release scope:    canonical .ptsip repository namespace + JSON index routing
```

This Tool release does not introduce a new Specification family or Project Profile contract. It establishes `.ptsip/` as the canonical repository-local PTSIP ownership boundary and `.ptsip/index.json` as the root machine-routing entry point. `.ptsip/profiles/index.json` is the canonical local Project Profile catalog representation. The former `.ptsip/profiles/index.yaml` form is accepted only as compatibility input for migration.

`.ptsip/tasks/` and `.ptsip/runtime/` are reserved namespaces in this release. Reservation does not claim a complete generic Repository Task Engine or finalized runtime persistence semantics, and the absence of those capabilities does not authorize a coding agent to establish another PTSIP control-plane root under `tools/`, `scripts/`, or another repository path.

Release scope is recorded in [`releasenote/tool/0.3.8a2.md`](releasenote/tool/0.3.8a2.md).

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

Release scope and verification history are recorded in [`releasenote/tool/0.3.8a1.md`](releasenote/tool/0.3.8a1.md). The release gate is defined by [`developer/planning/0.3.8/0.3.8a1/0.3.8a1-emergency-release-gate.yaml`](developer/planning/0.3.8/0.3.8a1/0.3.8a1-emergency-release-gate.yaml).

## License authority

PTSIP licensing authority is contained exclusively under [`License-Authority/`](License-Authority/). The machine-readable entry point is [`License-Authority/license-authority.yaml`](License-Authority/license-authority.yaml), and the controlling legal text is [`License-Authority/LICENSE.md`](License-Authority/LICENSE.md).

The canonical effective time currently declared by the License Authority is `2026-09-15T00:00:00Z` (UTC).

License-related wording elsewhere in this repository is informational or contextual only. It does not become part of the License, modify it, or become incorporated by reference.

Normal Tool releases, Specification validation, Tool version changes, documentation mentions, keyword matches, and AI semantic guesses do not authorize License Authority entry. Entry uses only the triggers declared by `License-Authority/license-authority.yaml`.
## Consumer Repository canonical namespace

Read-only inspection and Pilot operations remain read-only and do not create repository state merely by being invoked. When a Consumer Repository persists repository-local contracts, registries, configuration, task/operation metadata, or execution state whose semantic owner is PTSIP, `.ptsip/` is the canonical ownership boundary.

Repository tooling implementations may physically live under `tools/`, `scripts/`, `src/`, or another project-owned path. Their PTSIP-owned registration, policy bindings, task/operation contracts, indexes, and lifecycle state must not establish an alternative repository-local PTSIP control-plane root.

Canonical machine indexes under `.ptsip/` use JSON. Tool `0.3.8a3` uses `.ptsip/index.json` as the root router and `.ptsip/profiles/index.json` for local Project Profile selection. The Task and runtime namespaces are reserved; runtime persistence details remain future work. A reserved or unavailable Task capability must fail closed for the dependent operation rather than causing an agent to invent a new Task Engine elsewhere in the repository.

## Project status

PTSIP remains experimental. Tool `0.3.8a3` is the current namespace-stabilization prerelease candidate; its publication boundary is tracked in [`releasenote/tool/0.3.8a3.md`](releasenote/tool/0.3.8a3.md). Historical Tool releases, including `0.3.8a1`, and Specification notes are preserved under [`releasenote/`](releasenote/).
