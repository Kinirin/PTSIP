<p align="right">
  English | <a href="README.ko.md">한국어</a>
</p>

# PTSIP — Product–Toolchain SDK Isolation Policy

**Status:** Draft project-defined specification  
**Specification family:** `0.3.4-draft`  
**Active normative snapshot:** `b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e`<br>
**License:** Apache License 2.0

PTSIP is an architecture policy for keeping Product SDK responsibility separate from development Toolchain SDK responsibility while preserving explicit contracts, reproducible conformance, and multi-environment architecture-decision consistency.

> **Purpose precedes reuse.** Classify a component by why it exists and which lifecycle owns it before considering code sharing.

## Architecture model

PTSIP has exactly three architectural classifications:

| Classification | Meaning |
| --- | --- |
| `PRODUCT` | Product-owned runtime/library/SDK/component responsibility. |
| `TOOLCHAIN` | Development-tooling-owned SDK/component responsibility. |
| `NEUTRAL_CONTRACT` | Deliberately non-executable, independently governed contract responsibility. |

`UNKNOWN`, `CONFLICT`, `INCOMPLETE`, `PENDING`, and similar states are workflow/evaluation states, not additional architecture planes.

The core boundary is:

```text
Toolchain SDK  --->  Product source / artifacts
      inspect / validate / generate / migrate / package

Product SDK    -X->  Toolchain implementation
      runtime / shipped dependency prohibited
```

Shared semantics should prefer a Neutral Contract over one project-local executable package owned by both planes.

## Install and use

PTSIP requires Python 3.11 or newer.

```powershell
pip install PTSIP
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

For source development:

```powershell
pip install -e ".[dev]"
```

## Specification and Tool lifecycle

The **PTSIP Specification** and **PTSIP Reference Tool** are independently versioned.

- `pyproject.toml` owns Tool/package source version;
- `ptsip --version` reports installed Tool version;
- `ptsip spec` reports the exact Specification family + immutable revision bound to that Tool;
- `spec/`, `schemas/`, and `registry/` contain canonical Specification assets;
- `src/ptsip/specdata/` contains matching embedded resources used by the Tool;
- GitHub Releases publish Tool and Specification release/design records.

The existing `spec-v0.3.4-draft` GitHub Release records the earlier design proposal. It remains an immutable historical checkpoint and is not moved. The active repository-identity-migration snapshot is `0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e`.

A Tool version number matching a Specification family number does not imply identity by itself.

## Consumer Repository non-intrusion

PTSIP does not require adopting repositories to create PTSIP-specific `docs/`, `tools/`, `.ptsip/`, cache, report, or hidden directories.

External inspection/Pilot tooling is read-only by default. Tool-owned operational state belongs outside the Consumer Repository unless the user explicitly chooses a repository path.

The default project-owned architecture declaration is repository-root `ptsip.yaml`; projects may consistently select another path with `--profile`.

Local state such as `control-plane.sqlite3` is not portable architecture authority and must not be Git-shared as repository-global coordination state.

## Explicit project adoption

Candidate discovery is evidence, not architecture authority. The project owner supplies architecture intent.

`0.3.4-draft` defines this structured adoption fact set:

```text
classification
purpose
shipped
runtime_required
lifecycle_owner
executable
```

Canonical lifecycle owners are:

```text
PRODUCT
DEVELOPMENT_TOOLING
INDEPENDENT
```

Example dry-run:

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

Structured adoption preserves those facts losslessly in component declarations. `runtime_required` is not discarded, and canonical `lifecycle_owner` is not aliased to `release_owner`.

Boundary-root shorthand remains available for simple declarations, but a write-enabled structured adoption/resolution must refuse mutation when shorthand cannot preserve the full fact set.

## Decision Authority

PTSIP distinguishes:

```text
Specification
    -> normative architecture / conformance / coordination rules

Decision Authority
    -> which explicit architecture answer won

Project Profile
    -> durable project-owned architecture declaration

Observed evidence
    -> what repository/artifacts actually do
```

A Decision Authority is not a conformance oracle and does not replace `ptsip.yaml`.

## GitHub-coordinated Reference Tool profile

Reference Tool `0.3.4` demonstrates distributed coordination through a dedicated Git ref:

```text
refs/heads/ptsip-policy
```

GitHub storage details are implementation-specific. The Specification requires the semantics, not GitHub itself:

- stable coordination-domain + component-scope decision identity;
- first-valid-resolution-wins;
- ordered conditional mutation / stale-writer protection;
- authority freshness at architecture-sensitive boundaries;
- non-mutating absence lookup;
- deterministic reconciliation;
- fail-closed distributed behavior; and
- separation of global decision state from clone-local application state.

## Coding-agent decision gate

```powershell
ptsip gate . --component tools --json
```

In distributed mode, a complete local profile does **not** automatically bypass the authority check. The relevant local declaration is compared with current authority state.

| Local Project Profile | Distributed Authority | Required result |
| --- | --- | --- |
| declaration absent | no decision | pending only when the active operation actually needs a decision |
| declaration absent | resolved winner | validate and safely project winner locally |
| declaration present | no authority decision | use project declaration; do not fabricate authority history |
| declaration present + equivalent | resolved equivalent winner | resolved/consistent; no formatting rewrite required |
| declaration present + conflicting | resolved different winner | explicit authority/profile conflict; no silent overwrite |
| repository/profile changed during reconciliation | any authority state | stale; refuse application and re-analyze |

Semantic equivalence means architecture meaning, not YAML key order or whitespace.

If distributed coordination is selected but required freshness/safe mutation cannot be established, PTSIP fails the affected operation instead of silently creating a separate Local winner.

## Global decision state versus local projection

```text
GLOBAL
    PENDING / RESOLVED

LOCAL CLONE / WORKTREE
    missing / consistent / locally applied / stale / failed
```

A global winner does not mean every clone is already synchronized. A clone-local application receipt cannot change the winner.

PTSIP uses **action-time synchronization** rather than continuous polling.

## Enforced conformance

`ptsip conform` evaluates declaration + observed evidence + Product Artifact/build/lifecycle evidence + snapshot/coverage against Consumer Repository PTSIP rules.

Completed outcomes are only:

| Exit code | Outcome |
| --- | --- |
| `0` | `CONFORMANT` |
| `5` | `NON_CONFORMANT` |
| `6` | `INCOMPLETE` |

A valid Project Profile does not prove conformance. A resolved Decision Authority winner does not prove conformance. A zero-finding scan does not prove conformance when blocking evidence gaps remain.

For Enforced Conformance against a mutable draft, bind the exact immutable Specification revision.

## Reference Tool focus

The Reference Tool provides:

- read-only repository inspection and Pilot evidence;
- multi-language dependency/artifact evidence;
- deterministic clarification for missing intent;
- explicit project-owner adoption;
- on-demand decision gating and explicit resolution;
- GitHub-coordinated first-winner authority for multi-environment agents;
- gate-time authority freshness and reconciliation;
- fail-closed distributed coordination;
- local-only DecisionStore mode when intentionally selected;
- project-profile validation;
- Product Artifact evidence ingestion;
- evidence-relative Enforced Conformance and stable diagnostics.

## Repository map

| Area | Location | Purpose |
| --- | --- | --- |
| Normative Specification | [`spec/`](spec/) | Architecture, terminology, and conformance rules. |
| Machine-readable registry | [`registry/`](registry/) | Canonical terms/rule IDs/metadata. |
| Schemas | [`schemas/`](schemas/) | Project Profile and interoperability schemas. |
| Agent contract | [`agents/AGENT-CONTRACT.md`](agents/AGENT-CONTRACT.md) | Coding-agent operational contract. |
| Adoption guide | [`adoption/ADOPTION-GUIDE.md`](adoption/ADOPTION-GUIDE.md) | Controlled adoption sequence. |
| Reference architecture | [`reference/REFERENCE-ARCHITECTURE.md`](reference/REFERENCE-ARCHITECTURE.md) | Informative architecture guidance. |
| ADRs | [`decisions/`](decisions/) | Normative architecture decisions. |
| Embedded spec data | [`src/ptsip/specdata/`](src/ptsip/specdata/) | Tool-packaged schema/registry copies. |
| Reference Tool | [`src/ptsip/`](src/ptsip/) | Installable Python implementation. |
| Tests | [`tests/`](tests/) | Tool and contract verification. |
| Release notes | [`releasenote/`](releasenote/) | Tool/Specification release history. |

## Key Specification documents

- [`spec/PTSIP-SPEC.md`](spec/PTSIP-SPEC.md)
- [`spec/PTSIP-CONFORMANCE.md`](spec/PTSIP-CONFORMANCE.md)
- [`spec/PTSIP-TERMINOLOGY.md`](spec/PTSIP-TERMINOLOGY.md)
- [`schemas/ptsip-profile.schema.json`](schemas/ptsip-profile.schema.json)
- [`registry/ptsip-registry.yaml`](registry/ptsip-registry.yaml)
- [`decisions/ADR-0005-activate-spec-0.3.4-draft.md`](decisions/ADR-0005-activate-spec-0.3.4-draft.md)

## Release namespaces

Tool releases use `tool-v*` tags. Specification releases/design records use the separate `spec-v*` namespace.

The exact normative identity of a mutable draft remains the immutable revision, not the tag string alone.

## Maturity

PTSIP is a draft project-defined specification, not an ISO, IEEE, IETF, CNCF, or other external industry standard.

## License

This repository, including the PTSIP Specification and Reference Tool unless explicitly stated otherwise, is licensed under the Apache License, Version 2.0. See [`LICENSE`](LICENSE).
