<p align="right">English | <a href="docs/translated/README.ko.md">한국어</a></p>

# PTSIP — Primary Lifecycle Ownership and Responsibility Isolation Policy

**PTSIP records architecture responsibility as explicit, machine-readable project intent, then verifies it against repository evidence.** It separates Product, development, delivery, and operations lifecycles without forcing a specific folder layout or programming language.

> **Purpose precedes reuse.** Determine why a responsibility exists and which lifecycle owns it before optimizing how code is shared.

## Component versions

| Identity | Source-tree value |
| --- | --- |
| PTSIP_Package | `0.3.8.<SHA6>` — composition assigned after successful publication |
| PTSIP_Tool | `0.3.8b1` |
| Project Profile | `pp.1.02` |
| Engine | Context/Projection `0.2.0`; Storage/Validation/Discovery/Planning/Execution `0.3.0` |

Package composition, Tool, each engine and Project Profile retain independent versions. The immutable Specification binding is recorded in the canonical composition and shipping manifest. Read [what changed in `0.3.8b1`](docs/releasenote/tool/0.3.8b1.md), the [Tool release history](docs/releasenote/tool/README.md), or [the release workflow](.github/workflows/tooling-release.yml). Release verification is selected from `.github/test_modes.yaml` and runs exclusively in `tooling-test.yml`; publication consumes that exact-source evidence in `tooling-release.yml`. Actual publication state and exact Package composition are recorded in the version-specific release assets.

## Quick start

Requires Python **3.11 or newer**. These commands use PowerShell; the basic CLI commands also work in other shells.

Install the latest normally selected published release, then inspect its identities:

```powershell
python -m pip install --upgrade PTSIP
ptsip --version
ptsip spec
```

To install Tool `0.3.8b1` specifically (ordinary pip resolution might select a stable version instead):

```powershell
python -m pip install "PTSIP==0.3.8b1"
```

To develop this repository directly:

```powershell
git clone https://github.com/Kinirin/PTSIP.git
cd PTSIP
python -m pip install -e ".[dev]"
```

Start with read-only or diagnostic operations before changing a Project Profile:

```powershell
ptsip doctor .
ptsip inspect .
ptsip pilot .
ptsip validate .
ptsip conform .
```

`validate` checks declarations. `conform` evaluates applicable architecture requirements against observed evidence. A valid profile is not proof of conformance; consult the installed CLI help for exact supported options.

## What the system keeps separate

| Authority/evidence plane | Responsibility |
| --- | --- |
| Specification | Normative requirements and their immutable revision |
| Decision Authority | Explicit approved architectural answers and their coordination |
| Source Project Profile / Responsibility Map | Project-owned architecture declarations |
| Observations and conformance | Actual repository/artifact behavior and rule evaluation |

**Evidence is not architecture authority.** File paths, packages, heuristics, templates, AI confidence and observed imports can inform review; none may silently create lifecycle ownership. The tool never treats an `INCOMPLETE` result as a pass.

## Primary lifecycle ownership

Canonical Tool `0.3.8a5` classifications remain exactly:

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

Tool `0.3.8a5` preserves the five-classification model established by Tool `0.3.6`. `TOOLCHAIN` is therefore **legacy Tool `0.3.5` input**, not a current canonical alias. A legacy Toolchain responsibility may become `DEVELOPMENT_TOOLING`, `DELIVERY`, `OPERATIONS`, or require a split depending on its actual lifecycle ownership. Blind `TOOLCHAIN -> DEVELOPMENT_TOOLING` rewriting is prohibited.

Tool `0.3.8a5` provides evidence-bound direct current-target migration for explicitly supported historical sources. Migration capability remains separate from repository adoption authority and never turns inference into project intent.

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

Tool `0.3.8a5` uses Responsibility Map v2 as the project-owned architecture declaration model. It keeps several axes independent:

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

## Profile selection and adoption

When repository-local PTSIP configuration is persisted, [`.ptsip/index.json`](.ptsip/index.json) routes the canonical `.ptsip/` namespaces and [`.ptsip/profiles/index.json`](.ptsip/profiles/index.json) selects the active `*.ptsip.yaml` Project Profile through `default_profile`. Repository-root `ptsip.yaml` is a historical migration/compatibility input, not a second active declaration authority.

`explicit`, `template`, and `hybrid` declarations all materialize to a Canonical Effective Responsibility Map. The materializer may resolve registered structures and provenance; it must **not** auto-select a template, invent a component owner, or fix conflicting project intent.

For an existing component, inspect the declaration change without applying it:

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

Apply only an owner-reviewed change by adding `--apply`. This is an illustrative command, not a rule classifying every directory named `tools`. Canonical ownership is `classification`; the historical `lifecycle_owner` field is migration evidence only.

Optional GitHub-coordinated decision authority uses `refs/heads/ptsip-policy` **when configured**. It relies on fresh authority reads and conditional mutations to reject stale writers. Coordination failures block new architecture-authority writes rather than silently falling back to an unrelated local decision store; GitHub is not a universal Specification dependency.

## Native foundation and consumer boundaries

Seven independently built **Rust engines** have versioned, language-neutral contracts and registered entrypoint bindings under [`src/ptsip/engines/`](src/ptsip/engines/). See the [official engine index](src/ptsip/engines/index.json).

| Engine | Owned capability |
| --- | --- |
| Context | Load validated, identity/source-bound Context snapshots |
| Storage | Mediate provider operations without taking provider ownership |
| Validation | Check schemas, contracts, typed I/O, preconditions and Cases |
| Discovery | Resolve registered source closures and invocation bindings |
| Planning | Order dependencies, assess changes and prepare native invocation |
| Execution | Perform bounded caller-admitted requests and record effects/stops |
| Projection | Render validated Context into declared output formats |

`0.3.8a5` verifies the native foundation on Linux, Windows and macOS; it **does not** implicitly grant consumer production installation, authority admission, activation, or versioned cutover. The Python wheel carries neutral engine contract data, **not** the Rust sources, native binaries or build outputs. Existing consumer APIs remain available; see the [release-scope boundary](docs/releasenote/tool/0.3.8a5.md).

PTSIP controls **who owns lifecycle responsibilities**. The independent [`src/vpms/`](src/vpms/) Verification Protocol Management System (VPMS) controls **how explicit Verification Cases are registered, selected, run and reported**. VPMS purpose/verification status is separate from PTSIP classification and conformance.

VPMS public composition uses `load_registry_snapshot`, `resolve_selection`, and `run_cases`, with explicit caller-owned definitions, references and adapters. The optional PTSIP bridge consumes only validated immutable Effective Responsibility Map data; it does not establish project authority. VPMS PASS does not imply PTSIP `CONFORMANT`, or vice versa.

PTSIP conformance completes with `CONFORMANT` (exit `0`), `NON_CONFORMANT` (`5`), or `INCOMPLETE` (`6`). Missing required evidence is fail-closed. Product distribution verification inspects actual snapshot-bound built artifacts; the producer component owning a build step is not automatically the owner of the resulting Product artifact.

## Developer workflow and repository map

| Path | Responsibility |
| --- | --- |
| [`.ptsip/`](.ptsip/) | Consumer-repository indexes, profiles, Context; Task/runtime are reserved |
| [`src/policy/index.yaml`](src/policy/index.yaml) | Current Support Feature policy registry (`ACTIVE`/`DRAFT`/`RETIRED`) |
| [`developer/policy/`](developer/policy/) | Developer machine policy and bindings |
| [`developer/planning/`](developer/planning/) and [`planning/`](planning/) | Policy-derived execution plans, work units and historical records |
| [`developer/automation/`](developer/automation/) | Native Go developer tooling, policy/planning/release controls |
| [`src/ptsip/`](src/ptsip/) | Python Consumer Tool, conformance, profile materialization and engines contracts |
| [`src/ptsip/engines/`](src/ptsip/engines/) | Seven independently owned native engine implementations/contracts |
| [`src/vpms/`](src/vpms/) | Independent verification subsystem |
| [`src/tests/`](src/tests/) and [`developer/tests/`](developer/tests/) | Consumer, developer and native verification |
| [`registry/`](registry/) and [`schemas/`](schemas/) | Specification registries and schemas |
| [`.github/workflows/`](.github/workflows/) | Test, release and README translation workflows |
| [`docs/releasenote/`](docs/releasenote/) | Tool, Project Profile and Specification release records |

For coding agents, [`AGENTS.md`](AGENTS.md) is an entry projection **not** an alternative policy authority. Policies govern plans, plans govern implementation; lower levels must not silently change an authoritative decision. See [CONTRIBUTING.md](CONTRIBUTING.md) and the [developer automation guide](developer/automation/README.md).

Bounded dependency analysis and validation planning are advisory and do not replace current architecture declarations or exact-source release validation:

```powershell
ptsip dependency analyze .
ptsip dependency review-pack . --max-items 8 --max-context-bytes 12000
ptsip dependency validation-plan . --changed src/ptsip/dependency_analysis.py --json
```

Release PREPARE and PUBLISH are independent, exact-source-verification phases in [`.github/workflows/tooling-release.yml`](.github/workflows/tooling-release.yml). Documentation, tests, workflow or Tool-only changes do not themselves change the immutable Specification revision.

## Release notes and background

- [Current `0.3.8a5` release candidate](docs/releasenote/tool/0.3.8a5.md): current-policy validation, registered Check coverage and seven-engine foundation; **publication pending**.
- [Published `0.3.8a4` baseline](docs/releasenote/tool/0.3.8a4.md): Go-native developer automation, Project Profile and release verification.
- [Historical Tool notes](docs/releasenote/tool/README.md): earlier releases, namespace and Context Plane migrations.
- [Specification/Project Profile/Tool release records](docs/releasenote/README.md): independently versioned lifecycle histories.

## License authority

PTSIP licensing authority is contained exclusively under [`License-Authority/`](License-Authority/). The machine-readable entry point is [`License-Authority/license-authority.yaml`](License-Authority/license-authority.yaml), and the controlling legal text is [`License-Authority/LICENSE.md`](License-Authority/LICENSE.md).

The canonical effective time currently declared by the License Authority is `2026-09-15T00:00:00Z` (UTC).

License-related wording elsewhere in this repository is informational or contextual only. It does not become part of the License, modify it, or become incorporated by reference.

Normal Tool releases, Specification validation, Tool version changes, documentation mentions, keyword matches, and AI semantic guesses do not authorize License Authority entry. Entry uses only the triggers declared by `License-Authority/license-authority.yaml`.

## Language and project status

This English README is the canonical project overview. [Localized documentation](docs/translated/) is regenerated on `main` using the registered Argos Translate workflow; translations do not override the English overview or machine-readable policy. PTSIP remains experimental and release claims are bounded to their exact source and verification evidence.

