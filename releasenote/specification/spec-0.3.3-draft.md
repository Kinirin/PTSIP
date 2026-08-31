# PTSIP Specification 0.3.3-draft Release Notes

**Status:** Proposed experimental Specification family  
**Predecessor:** `0.2.0-draft`  
**Normative baseline:** `0.2.0-draft` revision `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`  
**Candidate normative snapshot:** To be assigned only by a coherent normative migration commit  
**Identity model:** draft family label + immutable Git revision

The `0.3.3-draft` label is a proposed Specification family derived from the current `0.2.0-draft` normative baseline.

This document is a release-note-level design record. It does not by itself activate `0.3.3-draft`, rebind a Tool, publish a Tool release, or make current `main` implementation behavior normative.

Reference Tool `0.3.3` is permanently source-only and will not receive a `tool-v0.3.3` tag, GitHub Release, or PyPI publication. Its completed adoption work is useful implementation evidence for this draft, but the Tool version and Specification family remain independently versioned concepts.

The GitHub-coordinated authority code currently present in `main` is not treated as a completed Tool `0.3.3` feature and is not used by this draft as proof of a complete distributed-authority contract. That source is precursor implementation for the separate Tool `0.3.4` Distributed Authority Consistency workstream.

## 1. Compatibility baseline

Unless explicitly changed below, `0.3.3-draft` inherits the normative semantics of `0.2.0-draft` revision `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`.

The following architectural invariants remain unchanged:

- PTSIP has exactly three architecture classifications: `PRODUCT`, `TOOLCHAIN`, and `NEUTRAL_CONTRACT`;
- `UNKNOWN`, `CONFLICT`, and `INCOMPLETE` remain decision/evaluation states rather than additional architecture planes;
- a Project Profile is a project-owned declaration of intended architecture and is not itself conformance truth;
- declaration and observed evidence remain distinct;
- Product Artifact owner and artifact producer remain distinct concepts;
- project-specific dependency policy may strengthen but may not weaken universal PTSIP rules;
- mandatory-rule violations are not waived by project governance metadata;
- external PTSIP tooling remains non-intrusive toward Consumer Repository structure;
- a coding agent must not invent missing architecture intent;
- Specification and Reference Tool release identities remain independent.

`0.3.3-draft` does not introduce a fourth plane and does not redefine the existing evidence, artifact, dependency, conformance, or diagnostic models merely because Tool `0.3.3` added workflow capabilities.

## 2. Why a new draft family is proposed

The predecessor Specification can describe declared components and evaluate architecture, but explicit first adoption exposes a representation gap between:

```text
repository evidence
    -> discovered component scope

project-owner intent
    -> explicit architecture facts

Project Profile
    -> durable project-owned declaration
```

A Tool can collect architecture facts during clarification or adoption, but those facts must not live only in transient Tool state when they are required to reconstruct and validate the durable architecture declaration.

The primary `0.3.3-draft` design objective is therefore:

> define a lossless, explicit, project-owner-controlled path from discovered component scope to durable Project Profile declaration without converting repository naming conventions or Tool heuristics into architecture authority.

This is intentionally narrower than Distributed Authority Consistency.

## 3. Explicit Project Adoption

`0.3.3-draft` introduces **Explicit Project Adoption** as a Specification-level concept.

Explicit Project Adoption is the controlled process by which a Consumer Repository establishes or extends its Project Profile from:

1. discovered repository/component scope; and
2. explicit architecture facts supplied or approved by the project owner.

Conceptually:

```text
inspect/discover candidate scope
        |
        v
collect explicit project-owner facts
        |
        v
validate architecture answer
        |
        v
prepare Project Profile projection
        |
        v
validate projected declaration
        |
        v
explicit apply
```

The Reference Tool command `ptsip adopt` is one implementation of this concept. The CLI spelling itself is not a universal protocol requirement.

## 4. Candidate evidence is not architecture authority

Repository inspection may identify candidate scope using evidence such as:

- include selectors;
- manifests;
- source roots;
- dependency observations;
- build/release files;
- executable anchors;
- artifact evidence.

Those observations may identify **what scope needs a declaration**. They do not by themselves determine whether that scope is `PRODUCT`, `TOOLCHAIN`, or `NEUTRAL_CONTRACT`.

A conforming adoption implementation MUST NOT assign architectural ownership solely from:

- directory names;
- package names;
- file names;
- repository-local naming conventions;
- the presence of `tools`, `src`, `sdk`, `build`, `runtime`, or similar labels;
- an unconstrained LLM guess.

Observed scope and project-owner intent remain separate inputs.

## 5. Project-owner architecture facts

An explicit adoption or architecture-decision answer must be sufficient to validate the intended classification without relying on hidden Tool state.

The `0.3.3-draft` candidate fact set is:

- `classification`;
- `purpose`;
- `shipped`;
- `runtime_required`;
- `lifecycle_owner`;
- `executable`.

These facts correspond to distinctions already used by current PTSIP classification, dependency, lifecycle, and neutrality semantics.

The final normative migration MUST ensure that durable Project Profile representation and decision/adoption validation do not disagree about the meaning of these facts.

## 6. Durable `runtime_required`

The predecessor `0.2.0-draft` Project Profile schema does not contain a durable component-level `runtime_required` field even though the current decision/adoption validator requires the fact.

Tool `0.3.3` correctly avoids silently widening its immutable `0.2.0-draft` binding. It may retain `runtime_required` in Tool-owned workflow state, but that does not solve the Specification representation gap.

`0.3.3-draft` therefore proposes `runtime_required` as a durable Project Profile component fact:

```yaml
runtime_required: true | false
```

Candidate semantics:

- `true` means Product runtime requires the component's executable/runtime implementation;
- `false` means no such Product runtime requirement is declared;
- a `TOOLCHAIN` component MUST NOT declare `runtime_required: true`;
- migrated legacy profiles MUST NOT silently interpret an absent value as `false` when strict evaluation requires the fact;
- absence/migration semantics must be explicitly defined by the final schema migration.

This is a `SCHEMA_CHANGE` and may also change profile-validation or conformance evaluability where the fact is required by an applicable mandatory rule.

## 7. Durable lifecycle ownership

Reference Tool decision/adoption validation also requires `lifecycle_owner`, with the current constrained values:

```text
PRODUCT
DEVELOPMENT_TOOLING
INDEPENDENT
```

The predecessor Project Profile contains lifecycle-related ownership fields but does not currently preserve this decision fact under one canonical `lifecycle_owner` representation.

The `0.3.3-draft` normative migration MUST choose one of two coherent outcomes:

1. add a canonical durable `lifecycle_owner` component field; or
2. define a lossless normative mapping from the decision fact to existing Project Profile ownership fields.

The migration MUST NOT leave a situation where an answer can pass decision validation but cannot be reconstructed unambiguously from the resulting Project Profile.

## 8. Candidate decision-validation semantics

The following relationships currently enforced by the Reference Tool are candidates for normative adoption because they express architecture meaning rather than CLI behavior:

- `PRODUCT` requires Product lifecycle ownership;
- `TOOLCHAIN` requires development-tooling lifecycle ownership;
- `TOOLCHAIN` cannot be shipped as part of the Product;
- `TOOLCHAIN` cannot be required by the Product at runtime;
- `NEUTRAL_CONTRACT` must be non-executable;
- `NEUTRAL_CONTRACT` requires independent lifecycle ownership.

The final normative migration MUST reconcile these constraints with the existing `PTSIP-DEP-*`, `PTSIP-LCY-*`, Product Artifact, Neutral Contract, and coherent-component rules.

A second parallel classification system MUST NOT be created merely because adoption requires structured answers.

## 9. Safe adoption transaction

A conforming adoption implementation MUST preserve project ownership of the declaration and MUST avoid partial or stale mutation.

Before applying a new or extended declaration, it MUST perform equivalent checks for:

- selected candidate/component identity;
- repository evidence freshness;
- explicit architecture-fact validity;
- conflict with an existing declaration;
- projected Project Profile validity;
- concurrent Project Profile modification.

Mutation MUST occur only after the projected declaration is valid.

If the repository/profile changes after analysis but before application, the implementation MUST refuse stale application and require re-analysis rather than silently applying an answer to a different repository state.

Equivalent adoption of an already equivalent declaration SHOULD be deterministic and idempotent.

## 10. Dry-run and explicit mutation

Adoption tooling SHOULD support a read-only planning mode before repository mutation.

The Reference Tool `0.3.3` uses dry-run by default and requires explicit `--apply`. The exact flag is Tool-specific, but the Specification-level principle is retained:

> architecture declaration mutation must be distinguishable from inspection/planning and must not occur merely because a candidate was discovered.

Inspection alone MUST NOT mutate the Consumer Repository Project Profile.

## 11. Project Profile location and ownership

The Project Profile remains project-owned state.

The default Reference Tool location remains:

```text
<repository-root>/ptsip.yaml
```

A Tool MAY support an explicitly selected alternative profile path, but all operations participating in one adoption/decision workflow MUST operate against the same selected Project Profile identity.

Profile-path selection MUST NOT change classification semantics.

A Project Profile intended to describe the repository revision is normally Git-tracked project state. Tool-owned operational databases are not substitutes for it.

## 12. Tool-owned decision state remains non-authoritative for project reconstruction

Local operational state such as:

```text
%LOCALAPPDATA%\PTSIP\decisions\<repository-fingerprint>\control-plane.sqlite3
```

is Tool-owned workflow state.

It MUST NOT be required to reconstruct the durable project architecture declared by a repository revision.

Deleting or losing a local DecisionStore must not erase the meaning of an otherwise complete committed Project Profile.

This rule does not standardize a particular Local DecisionStore schema or backend.

## 13. Decision workflow and Specification boundary

PTSIP Tools may provide workflows that create pending decisions, collect explicit answers, enforce first-valid-resolution behavior inside a selected backend, and apply accepted answers to a Project Profile.

`0.3.3-draft` does not standardize every control-plane transport, persistence mechanism, issue integration, hosted service, or backend-selection rule as core architecture semantics.

The normative focus of this family is the correctness of the resulting architecture declaration and the explicit separation of:

```text
observed repository evidence
project-owner architecture intent
durable Project Profile declaration
Tool-owned workflow state
```

Tool implementation details remain non-normative unless separately adopted into the Specification.

## 14. Distributed Authority Consistency is not completed by 0.3.3

The current `main` source contains early GitHub-coordinated authority code created during Tool `0.3.3` development.

That implementation includes useful precursor primitives such as:

- repository-scoped authority state;
- `refs/heads/ptsip-policy` bootstrap;
- authority manifest checks;
- stable `gdec-*` component-scope identity;
- non-force Git ref compare-and-swap writes;
- first-winner protection for some concurrent contradictory mutations;
- basic stale-clone projection when the local declaration is absent;
- fail-closed behavior in selected GitHub-coordinated paths.

These primitives are **not** treated by `0.3.3-draft` as proof that repository-global distributed authority is complete.

In particular, the `0.3.3` source does not establish the finished consistency contract for:

- authority freshness when a local Project Profile is already complete but stale;
- deterministic comparison of local declaration versus remote winner;
- explicit equivalent/missing/conflicting reconciliation semantics;
- final machine-readable authority/profile conflict states;
- separation of global decision resolution from clone-local projection/application state;
- complete multi-clone consistency verification across architecture-sensitive operation boundaries.

Those requirements belong to the separate Tool `0.3.4` **Distributed Authority Consistency** workstream.

Therefore this Specification draft MUST NOT claim any of the following merely from the presence of the current GitHub CAS implementation:

```text
GitHub CAS exists
    != Distributed Authority Consistency is complete

ptsip-policy ref exists
    != all coordinated reads are fresh

one global winner can be written
    != every clone reconciles that winner correctly
```

## 15. GitHub authority prototype is non-normative in this family

`refs/heads/ptsip-policy`, `authority.json`, `decisions/<gdec-id>.json`, and the current GitHub CAS implementation are Reference Tool precursor implementation details for the next distributed-consistency workstream.

`0.3.3-draft` does not require PTSIP implementations to use:

- GitHub;
- a Git branch as an authority database;
- `gdec-*` encoding;
- GitHub Contents/Git Database APIs;
- continuous remote synchronization.

No claim in this draft should be interpreted as making the current `main` GitHub authority prototype a release-quality interoperability contract.

## 16. Coding-agent contract implications

A coding agent operating under the proposed `0.3.3-draft` adoption semantics MUST:

- inspect the selected Project Profile before inventing architecture intent;
- stop affected architecture-sensitive work when required intent is unresolved;
- request explicit project-owner/user facts rather than infer ownership from repository names;
- preserve explicit classification facts exactly when applying a declaration;
- refuse stale or conflicting profile application;
- not treat Tool-owned local state as a replacement for the Project Profile;
- distinguish observed scope from project-owner architectural intent.

Distributed multi-environment authority freshness and reconciliation requirements are deliberately not claimed as completed `0.3.3-draft` semantics by this release-note design record.

## 17. Machine-readable adoption expectations

Adoption interfaces SHOULD expose stable machine-readable states sufficient for coding agents and automation to distinguish at least:

- dry-run/adoption plan;
- adopted/applied;
- already equivalently declared;
- unknown component/candidate;
- stale evidence;
- conflicting existing declaration;
- validation/application error.

The Reference Tool `ptsip-adoption/v1` result family is implementation evidence for this requirement.

The exact CLI JSON envelope is not automatically normative merely because it exists in Tool `0.3.3`.

Distributed authority-specific result states are deferred to the consistency workstream that completes those semantics.

## 18. Preserved predecessor semantics

Except for the explicit additions and schema questions above, `0.3.3-draft` preserves the predecessor family's established semantics for:

- exactly three architecture classifications;
- coherent component boundaries;
- nested-selector precedence;
- declaration versus observation separation;
- evidence provenance and coverage;
- Product Artifact owner/producer distinction;
- dependency edge and lifecycle-phase semantics;
- Product-to-Toolchain runtime prohibition;
- lifecycle independence requirements;
- Neutral Contract non-executable/independent semantics;
- profile validation versus conformance evaluation;
- stable diagnostic identity;
- project-specific dependency policy layering;
- mandatory-rule remediation without waiver;
- Consumer Repository non-intrusion;
- immutable Specification revision binding.

## 19. Explicit non-goals

`0.3.3-draft` does not introduce or standardize:

- a fourth architecture classification;
- automatic LLM architecture classification;
- directory-name-based ownership inference;
- shared SQLite through Git;
- a mandatory `.PTSIP/` or `.ptsip/` Consumer Repository directory;
- continuous background polling;
- GitHub as a universal PTSIP dependency;
- the current GitHub authority prototype as a completed distributed consistency contract;
- silent overwrite/reclassification of an existing conflicting Project Profile;
- mandatory conformance waivers;
- organization-wide profile composition;
- unrelated expansion of evidence adapters or conformance rules;
- a requirement that Specification and Tool version numbers match.

## 20. Required normative migration assets

Before `0.3.3-draft` can become an active Specification binding, one coherent normative migration must update all affected normative and machine-readable assets together.

At minimum this is expected to include:

- `spec/PTSIP-SPEC.md`;
- `spec/PTSIP-CONFORMANCE.md` where new durable facts affect validation/evaluability;
- `spec/PTSIP-TERMINOLOGY.md` for Explicit Project Adoption and any new durable architecture-fact terminology;
- `schemas/ptsip-profile.schema.json`;
- `registry/ptsip-registry.yaml` where rule/schema metadata changes;
- `agents/AGENT-CONTRACT.md` for agent-visible normative behavior;
- reference/adoption documentation affected by the new contract;
- embedded `src/ptsip/specdata/*` resources;
- an ADR recording the normative transition from `0.2.0-draft` to `0.3.3-draft`.

Tool binding constants must be changed only by a Tool version that intentionally adopts the completed immutable `0.3.3-draft` snapshot.

The migration commit that makes all normative assets coherent becomes the first immutable normative revision of the family.

## 21. Change classification

Expected change categories are:

- `CLARIFICATION` — separate discovered scope, explicit project-owner intent, durable Project Profile state, and Tool workflow state;
- `NORMATIVE_ADDITION` — Explicit Project Adoption and safe declaration-application requirements;
- `SCHEMA_CHANGE` — durable `runtime_required` and canonical/lossless lifecycle ownership representation;
- `CONFORMANCE_CHANGE` — only where new durable facts alter profile validity or applicable mandatory-rule evaluability;
- `NORMATIVE_BREAKING` — only where predecessor profile meaning cannot be preserved and the migration explicitly documents the incompatibility.

Distributed Authority Consistency is not classified as a completed `0.3.3-draft` normative addition by this document.

Existing rule IDs MUST NOT be silently repurposed for incompatible meanings.

## 22. Tool 0.3.3 historical and publication boundary

Reference Tool `0.3.3` was implemented and verified while bound to:

```text
Specification family: 0.2.0-draft
Specification revision: a877b2f66a7f94c1b844c979e1b08fb08a9a8e45
```

It remains permanently source-only.

The following Tool `0.3.3` publication artifacts are intentionally absent and must not be treated as pending work:

```text
tool-v0.3.3
GitHub Release 0.3.3
PyPI PTSIP==0.3.3
```

The presence of a proposed Specification family named `0.3.3-draft` does not imply that Reference Tool `0.3.3` implements or publishes that Specification family.

Likewise, Tool `0.3.4` does not automatically bind this Specification merely because it follows Tool `0.3.3`. Tool/Specification binding must always be explicit and revision-pinned.

## 23. Acceptance criteria for the first 0.3.3-draft snapshot

The first active `0.3.3-draft` normative snapshot is ready only when all of the following are true:

- every changed normative asset agrees on Specification family `0.3.3-draft`;
- the immutable migration revision is recorded consistently;
- exactly three architecture classifications remain;
- Explicit Project Adoption has implementation-independent normative meaning;
- discovered repository scope cannot become architecture ownership without explicit project-owner intent;
- adoption validates candidate identity, architecture facts, evidence freshness, projected profile validity, and concurrent profile state before mutation;
- Project Profile representation is lossless for the architecture facts required by normative decision/adoption validation;
- `runtime_required` has explicit durable schema and migration semantics;
- lifecycle ownership has a canonical or explicitly lossless durable representation;
- Tool-owned Local DecisionStore state is not required to reconstruct a complete durable Project Profile;
- Project Profile location selection does not change architecture semantics;
- stale or conflicting Project Profile mutation is refused rather than silently applied;
- the current GitHub authority precursor is not misrepresented as completed Distributed Authority Consistency;
- Distributed Authority Consistency requirements are not silently standardized from incomplete Tool `0.3.3` behavior;
- Tool `0.3.3` remains historically bound to `0.2.0-draft@a877b2f66a7f94c1b844c979e1b08fb08a9a8e45` and permanently unpublished;
- the final migration has an ADR and complete schema/registry/agent-contract synchronization.

Until those conditions are satisfied, `0.2.0-draft` remains the active Specification family.
