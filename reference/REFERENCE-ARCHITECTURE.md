# PTSIP Tool 0.3.6 Reference Architecture

This document is **informative**. Normative requirements come from the Tool-bound Specification `0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea` and its canonical machine-readable assets.

## 1. Primary lifecycle topology

Tool `0.3.6` uses exactly five canonical lifecycle classifications:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

An illustrative repository might contain:

```text
repository/
├─ product/                     PRODUCT
├─ devtools/                    DEVELOPMENT_TOOLING
├─ release/                     DELIVERY
├─ operations/                  OPERATIONS
└─ contracts/                   NEUTRAL_CONTRACT
```

PTSIP does **not** require those names or that layout. Mixed and nested ownership is valid when the project-owned Responsibility Map expresses the real boundary.

For example:

```text
src/
├─ app/                         PRODUCT
├─ verification/shared/         DEVELOPMENT_TOOLING
├─ publish/                     DELIVERY
├─ ops/reconcile/               OPERATIONS
└─ protocol/schema/             NEUTRAL_CONTRACT
```

Path names, languages, frameworks, compilation boundaries, workflow providers, and executable status are evidence context, not lifecycle authority.

`TOOLCHAIN` is a historical Tool `0.3.5` classification only. It is not a Tool `0.3.6` alias. Legacy Toolchain responsibilities must be evaluated against the actual Tool `0.3.6` lifecycle model rather than blindly renamed.

## 2. Responsibility Map v2

Tool `0.3.6` keeps lifecycle ownership separate from other architecture axes:

```text
classification
    = primary lifecycle ownership

roles
    = coarse responsibility characteristics

relationships
    = typed project-owned directed semantics

source/derived provenance
    = declaration/materialization origin

VPMS Verification Purpose
    = why verification exists and what it protects
```

Canonical roles:

```text
IMPLEMENTATION
VERIFICATION
AUTOMATION
CONFIGURATION
DOCUMENTATION
GOVERNANCE
```

Canonical project-owned relationship types:

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

Relationship semantics do not change endpoint classification. A `DELIVERY --BUILDS--> PRODUCT` relationship means Delivery builds a Product responsibility/artifact; it does not make the Delivery implementation Product.

## 3. Project-owned source declaration

The default project-owned architecture declaration is repository-root:

```text
ptsip.yaml
```

A Consumer Repository may consistently use another explicit path through `--profile`.

Tool-owned local operational state such as caches, reports, and SQLite decision stores belongs outside the Consumer Repository unless the user explicitly chooses a repository path.

The Source Project Profile remains project-owned architecture authority.

## 4. Explicit, template, and hybrid modes

Canonical source modes are:

```text
explicit
    project declares the complete map directly

template
    project explicitly selects one immutable revision-bound template

hybrid
    project explicitly selects a template and adds project-owned
    overrides, extensions, or removals
```

Template selection is never inferred from path layout, language, framework, manifest type, or confidence.

Initial Tool `0.3.6` template examples include:

```text
python-package-library
python-cli-application
mixed-product-development-delivery
```

A template is a reusable declaration starting point, not an architecture oracle.

## 5. Canonical Effective Responsibility Map

All source modes resolve through deterministic, non-authoritative materialization:

```text
Source Project Profile
        |
        v
source validation / exact template binding
        |
        v
deterministic materialization
        |
        v
ResolvedProfile
        |
        v
Canonical Effective Responsibility Map
        |
        +--> validation / conformance
        +--> clarification / adoption
        +--> narrow VPMS read-only projection
```

Materialization may resolve explicitly authorized template/hybrid declarations. It may not infer lifecycle ownership, choose a template, repair invalid architecture, or silently rewrite project-owned source state.

Effective architecture can retain declaration provenance such as:

```text
PROJECT_EXPLICIT
TEMPLATE
PROJECT_OVERRIDE
PROJECT_EXTENSION
PROJECT_REMOVAL
```

That provenance is not evidence confidence and does not replace project authority.

## 6. Associated artifacts

An associated artifact is a project-owned non-component support surface subordinate to exactly one classified anchor component.

Examples may include documentation, governance records, or configuration that do not carry independently governable executable/lifecycle responsibility.

Associated artifacts are not a classification escape hatch. If a surface gains independently governed executable, release, compatibility, Delivery, Operations, or neutral-contract responsibility, evaluate it as a component instead.

## 7. Product runtime and package boundary

Product code and Product distributions must not depend on non-Product lifecycle implementation merely because a shared development environment contains it.

Typical forbidden direction at Product runtime/shipping scope:

```text
PRODUCT
   |
   +---- runtime/import/package requirement ----> DEVELOPMENT_TOOLING
   +---- runtime/import/package requirement ----> DELIVERY
   +---- runtime/import/package requirement ----> OPERATIONS
```

A valid shared `NEUTRAL_CONTRACT` may cross lifecycle boundaries when it satisfies the non-executable, non-owning, independently governed contract requirements.

## 8. Artifact owner and producer are separate

A Product Artifact may be created by a non-Product producer:

```text
DEVELOPMENT_TOOLING / DELIVERY producer
                |
              BUILDS
                |
                v
          Product Artifact
```

Artifact classification follows artifact ownership and shipping responsibility, not producer classification alone.

Tool `0.3.6` release verification therefore inspects actual built distribution contents and can bind `ptsip-artifact-evidence/v1` to the exact repository snapshot with an adjacent artifact-evidence binding.

Definite non-Product implementation leakage into a Product distribution remains a `PTSIP-PKG-001` concern.

## 9. External PTSIP Tool topology

Preferred operational topology:

```text
External development / verification environment
│
├─ PTSIP Tool installation
├─ Tool-owned cache/report/local decision state
│
└──────── read / explicit authorized write ────────>
                    Consumer Repository
                    └─ ptsip.yaml
```

Inspection and Pilot operations are read-only by default. Consumer Repositories do not need a Tool-owned `.ptsip/` directory merely to use PTSIP.

## 10. Specification / Decision Authority / Project Profile

Keep these distinct:

```text
PTSIP Specification
    -> normative rules and semantics

Decision Authority
    -> which explicit coordinated architecture answer won

Consumer Repository Project Profile
    -> durable project-owned declaration for this repository revision/worktree

Observed evidence
    -> what repository/artifacts actually do

Conformance Evaluation
    -> whether declaration + evidence satisfy applicable rules
```

Decision Authority is not a conformance oracle. A valid authority winner may still describe architecture that is non-conformant with observed evidence.

## 11. GitHub-coordinated authority profile

The Reference Tool supports distributed decision coordination through:

```text
refs/heads/ptsip-policy
```

The GitHub storage representation is a Tool backend detail, not a universal Specification requirement.

Important properties are:

- stable coordination-domain and normalized component-scope identity;
- first-valid-resolution-wins;
- ordered stale-writer-safe conditional mutation;
- read-side authority freshness;
- deterministic reconciliation;
- fail-closed behavior;
- separation of global decision state from local projection/application state.

Another backend may provide equivalent guarantees using transactions, ETag/generation checks, consensus logs, or other ordered conditional mutation mechanisms.

## 12. Authority freshness

Write serialization is not enough. Before a distributed coordination-sensitive result relies on local declaration state, the Tool must account for relevant current authority state.

```text
analyze repository/profile
        |
        v
resolve coordination domain + scope
        |
        v
read current authority
        |
        v
compare authority vs local declaration
        |
        v
consistent / reconcile / conflict / fail closed
```

A complete local profile can still be stale relative to coordinated authority.

PTSIP uses action-time synchronization, not continuous background polling.

## 13. Global and local states

Do not conflate authority state with clone-local projection state.

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

A global `RESOLVED` winner does not imply every clone has already written the corresponding declaration. A local receipt cannot redefine the global winner.

## 14. Conformance topology

Conformance consumes the validated Effective Responsibility Map plus relevant observed evidence:

```text
project-owned declaration
        |
        v
Effective Responsibility Map
        |
        +--------+
        |        |
        v        v
 dependency   artifact/lifecycle/snapshot evidence
        |        |
        +---+----+
            |
            v
     deterministic PTSIP rules
            |
            v
CONFORMANT / NON_CONFORMANT / INCOMPLETE
```

A valid Project Profile does not prove conformance. A zero-finding result does not prove conformance when blocking evidence gaps can hide an applicable mandatory rule.

## 15. VPMS integration boundary

PTSIP and VPMS answer different questions:

```text
PTSIP
    Who owns this responsibility across its lifecycle?

VPMS
    Why does this Verification Case exist, and what does it protect?
```

The PTSIP core must not depend on VPMS. VPMS consumes only a narrow read-only projection of already-resolved PTSIP metadata.

Current VPMS compatibility vocabulary may retain `PRODUCT | TOOLCHAIN`. VPMS `TOOLCHAIN` is not a Tool `0.3.6` PTSIP classification.

A verification implementation may therefore be PTSIP `DEVELOPMENT_TOOLING` while a VPMS case has purpose `PRODUCT`.

## 16. Tool 0.3.6 release architecture

Tool `0.3.6` development closure is complete. Release architecture remains exact-SHA based:

```text
approved release candidate -> main
    -> self-hosted tooling-test on exact main SHA
    -> exact status success
    -> release.yml verifies current main + bound Specification/content contract
    -> draft GitHub Release targets the same SHA
    -> maintainer publishes draft
    -> tooling-release.yml builds/verifies actual publication distributions
    -> PyPI Trusted Publishing
```

The publication build remains self-hosted for distribution construction/verification. The narrow GNU/Linux Trusted Publishing boundary may remain GitHub-hosted.

Strategy C-style build-once/attestation redesign was deliberately not required for Tool `0.3.6`.

## 17. Tool 0.3.6.1 migration continuation

Tool `0.3.6` establishes the canonical lifecycle architecture. Assisted Tool `0.3.5 -> 0.3.6` migration continues under Tool `0.3.6.1`:

```text
facts
    -> candidate discovery
    -> normalized evidence/provenance
    -> legacy Tool 0.3.5 reader
    -> migration analysis
    -> proposals
    -> owner preview/confirmation
    -> safe apply
```

Evidence, inference, and proposals remain non-authoritative until a project owner accepts an explicit architecture decision.
