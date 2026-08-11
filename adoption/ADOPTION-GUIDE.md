# PTSIP Adoption Guide

This guide describes a controlled migration from an unclassified repository to PTSIP.

> **Specification status note:** the active normative baseline remains `0.2.0-draft` until a coherent normative migration activates a later immutable snapshot. The published `spec-v0.3.4-draft` release records the proposed Explicit Project Adoption and Distributed Authority Consistency model. Reference Tool `0.3.4` provides implementation evidence for that design while remaining bound to the active `0.2.0-draft` revision.

## Phase 0 — Stable baseline

Record the Consumer Repository revision before interpreting architecture evidence.

For automated Pilot evidence, rerun the analysis if HEAD or observed tracked state changes during collection. Do not mix evidence from different revisions into one conformance claim.

## Phase 1 — Inventory and evidence coverage

Inventory project-owned SDKs, packages, validators, migration tools, generators, build helpers, shared/common modules, manifests, relevant schemas/contracts, build/release automation, and known Product artifacts.

Do not move code yet.

Record inaccessible paths, parser failures, unsupported dependency forms, unresolved dynamic behavior, uninspected artifacts, and unsupported language/build/package adapters rather than silently treating them as absent.

For every evidence gap, ask whether it can conceal the result of an applicable mandatory PTSIP rule:

```text
can hide mandatory-rule result -> blocking gap
cannot affect mandatory-rule result -> non-blocking gap
```

Do not use a global unresolved-count or arbitrary coverage-percentage threshold as a substitute for rule-relative evidence sufficiency.

## Phase 2 — Component discovery

Identify architectural component candidates using evidence such as:

- package/build manifests;
- independent release/build anchors;
- CI-invoked scripts;
- plugin or SDK project files;
- schema/protocol bundles;
- test/tool roots;
- artifact producers.

Directory names are hints, not ownership decisions. A directory called `tools`, `scripts`, or `build` may be a useful candidate anchor, but PTSIP does not classify it as `TOOLCHAIN` from its name alone. Likewise, a differently named directory may still be discovered through manifests, invocation edges, CI evidence, or other structural evidence.

For each candidate record:

- primary purpose;
- consumers;
- shipped/not shipped;
- executable/declarative nature;
- dependency manifest;
- release owner;
- compatibility owner;
- evidence IDs and counter-evidence.

If one broad candidate contains materially different shipped state, executable purpose, release owner, compatibility owner, or Product/Toolchain lifecycle responsibility, split it into coherent ownership candidates rather than hiding the difference behind one root classification.

## Phase 3 — Classification decision

Resolve each in-scope project-owned component to exactly one architecture classification:

- `PRODUCT`
- `TOOLCHAIN`
- `NEUTRAL_CONTRACT`

During investigation, use decision status rather than inventing another class:

- `UNKNOWN` — insufficient evidence;
- `CONFLICT` — material evidence/declarations disagree;
- `INCOMPLETE` — required analysis coverage is missing.

External libraries, standard-library/platform nodes, and unresolved dependency targets are evidence-node scope, not extra PTSIP classifications.

An unresolved decision that affects a boundary relevant to a mandatory rule is a blocking gap and should block structural migration or conformance conclusions that depend on that ownership decision.

A coding agent may propose a schema-constrained decision with evidence IDs, but it does not automatically approve the project profile or determine conformance.

Neutral Contract classification is not determined by a fixed number of current consumers. Evaluate non-executable/non-owning contract semantics and lifecycle ownership.

## Phase 4 — Dependency audit

Construct typed dependency evidence and identify:

- Product -> Toolchain edges;
- relationship type (`IMPORTS`, `LINKS`, `LOADS`, `INVOKES`, `READS`, `GENERATES`, `PACKAGES`, `TESTS`, `PUBLISHES`);
- lifecycle phase when known (`RUNTIME`, `BUILD`, `TEST`, `RELEASE`, `INSPECTION`);
- evidence provenance (`DECLARED`, `OBSERVED`, `INFERRED`);
- project/external/platform/unresolved target scope;
- unresolved/dynamic edges;
- shared executable packages used by both planes;
- common modules with no explicit owner.

Do not assume every Toolchain -> Product edge is allowed. Distinguish bounded inspection/build/test/analysis inputs from executable implementation reuse.

Do not guess phase or target merely to increase automation coverage.

## Phase 5 — Boundary declaration and explicit adoption

Create or update the project-owned PTSIP profile only when the project wants a persistent declaration.

The default Project Profile is repository-root `ptsip.yaml`. It is project-owned architecture state and is intended to be committed with the Consumer Repository. Do not add it to `.gitignore` merely because the Reference Tool generated it. Projects that own the declaration elsewhere may consistently select an explicit path with `--profile`.

Use exactly one reference ownership-declaration mode:

- `boundaries` when root ownership is uniform; or
- `components` when nested/mixed/file-level boundaries exist.

Do not combine the two reference ownership modes in one profile.

A component profile records intended ownership. It does not prove that the dependency graph or Product artifacts obey the declaration.

When selectors overlap, exact/more-specific ownership wins. Equal-specificity ownership conflicts must be resolved explicitly.

A project may optionally declare stricter component-to-component dependency constraints, including same-plane constraints. These project-specific policies may strengthen but cannot weaken universal PTSIP rules.

### `ptsip adopt`

Reference Tool `0.3.4` retains the explicit project-owner adoption workflow introduced in Tool `0.3.3`. It reuses discovered candidate scope but requires the architecture facts to be supplied explicitly:

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

The default is a dry-run. It validates the candidate, decision facts, repository snapshot, profile projection, and schema without changing the Consumer Repository. Apply only after review:

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

`ptsip adopt` does not invent architecture intent and does not create a second classification algorithm. It reuses the same deterministic architecture-answer validation and Project Profile projection used by the decision workflow.

The Tool currently collects structured facts such as `runtime_required` and `lifecycle_owner` for deterministic decision/adoption validation. Under the active `0.2.0-draft` binding, Tool-owned workflow state does not silently widen the Project Profile schema. The published `0.3.4-draft` design proposes a future lossless durable representation for any such facts that remain normatively required; that representation is not active until the coherent schema migration is completed.

### Multi-environment decision coordination

A local SQLite DecisionStore is intentionally not Git-shared. It is suitable for local-only coordination, but two different clones cannot use separate SQLite files as a repository-global first-winner lock.

For a repository with a GitHub origin, Tool `0.3.4` coordinates relevant architecture decisions through a dedicated remote authority ref by default:

```text
refs/heads/ptsip-policy
```

The ref is bootstrapped only by a write-enabled coordinated operation. A read-only authority check does not create the ref or fabricate a pending decision merely to prove that no authority history exists. The authority stores JSON authority/decision records, not `control-plane.sqlite3`.

GitHub authority mutations use exact-parent commits and non-force ref updates. A stale environment cannot overwrite a newer authority HEAD. The global decision key is derived from repository identity and normalized component include scope so temporarily different local clarification IDs do not create separate winners for the same component boundary.

PTSIP uses **action-time synchronization**, not continuous polling. A coding agent calls `ptsip gate` when the active task reaches a boundary where authoritative coordinated architecture state matters. In GitHub-coordinated mode, a complete local declaration does not by itself allow the gate to skip the relevant authority freshness check.

The required reconciliation behavior is:

| Local Project Profile | GitHub authority | Tool 0.3.4 behavior |
| --- | --- | --- |
| declaration absent | no decision | create/reuse a pending decision only when the active gate actually needs one |
| declaration absent | resolved winner | validate and safely project the winner into the selected local profile |
| declaration present | no authority decision | use the project declaration; do not fabricate remote decision history solely for bookkeeping |
| declaration present and semantically equivalent | resolved equivalent winner | return a consistent/resolved result without rewriting equivalent profile content |
| declaration present and semantically conflicting | resolved different winner | return `AUTHORITY_PROFILE_CONFLICT`; do not silently overwrite or reclassify the profile |
| local repository/profile changes during reconciliation | any authority state | refuse stale application and require re-analysis |

Semantic equivalence is based on architecture meaning, not YAML formatting, key ordering, or other incidental serialization differences.

If GitHub coordination is selected but unavailable because of network, authentication, permissions, malformed authority state, or inability to establish required freshness, the affected coordinated architecture operation fails closed with a coordination error such as `COORDINATION_UNAVAILABLE`. PTSIP does not silently fall back to a separate Local DecisionStore, because that would reintroduce split-brain authority.

Cloud environments may use `GH_TOKEN` or `GITHUB_TOKEN`; interactive developer environments may use an authenticated `gh` CLI. The credential must have enough repository write authority for write-enabled authority operations.

A non-GitHub repository continues to use the embedded Local DecisionStore for coding-agent gates. A GitHub repository can deliberately opt into isolated local coordination with `--coordination local`, but that is intentionally not repository-global distributed coordination. An explicit hosted Control Plane may be selected with `--control-plane <URL>`.

### Global decision state is not local application state

Keep these two states separate during adoption and reconciliation:

```text
GLOBAL DECISION STATE
    PENDING / RESOLVED

LOCAL PROJECTION STATE
    missing / equivalent / locally applied / stale / failed
```

A globally `RESOLVED` decision means one architecture answer won within the coordination domain. It does not mean every clone has already written that answer into its local Project Profile.

Tool `0.3.4` therefore treats clone-local projection/application reporting separately from the global winner. A local projection receipt cannot change which architecture answer won and must not be interpreted as proof that every worktree is synchronized.

## Phase 6 — Profile Validation

Validate the declaration itself before using it for automated Enforced Conformance.

Profile Validation may check:

- schema structure;
- specification-binding syntax;
- component IDs;
- selector conflicts;
- referenced components;
- exception fields; and
- optional project-policy consistency.

A successful Profile Validation result does **not** mean the repository is conformant.

Under distributed coordination, Profile Validation also remains distinct from authority reconciliation. A structurally valid local profile can still be stale relative to a repository-global authority winner; conversely, a resolved Decision Authority record is not proof that observed repository behavior conforms to the declaration.

## Phase 7 — Structural migration

Move or repackage components only after ownership decisions and dependency evidence are sufficiently stable.

Do not treat renaming alone as architectural migration. A directory move is incomplete if dependency and packaging behavior remain coupled.

If the repository uses explicit transition planning, keep these concepts separate:

```text
current observed architecture
!= target architecture
!= conformance result
```

A migration target does not rewrite current observed conformance.

## Phase 8 — Build isolation

Make Product and Toolchain dependencies independently resolvable.

Verify that a clean Product build does not need Toolchain-only packages.

If Toolchain build/test/inspection uses Product implementation as an input, record the purpose and lifecycle phase instead of treating Toolchain -> Product direction as automatically allowed.

## Phase 9 — Product Artifact evidence

Identify the Product Artifact separately from the component that produced it.

For each artifact relevant to `PTSIP-PKG-001`, collect enough evidence to describe:

- artifact identity;
- Product/Toolchain/Neutral ownership when applicable;
- producer component;
- artifact type/format;
- shipping scope;
- contained paths/components or equivalent package manifest;
- derivation (`GENERATES`, `PACKAGES`, `PUBLISHES`); and
- evidence provenance.

A Toolchain build component may validly produce a Product Artifact. What matters for packaging isolation is the resulting artifact contents, not the producer's classification alone.

Source-path declarations alone are not sufficient evidence for `PTSIP-PKG-001` when actual Product artifact contents can differ from source topology.

The reference evidence shape is `ptsip-artifact-evidence/v1`.

## Phase 10 — Remediation review

For every established PTSIP `MUST`/`MUST NOT` violation, define the concrete architecture change required to satisfy the rule. Typical remediation includes dependency removal, ownership correction, component splitting, Neutral Contract extraction, packaging exclusion, build-environment separation, or lifecycle separation.

Repository- or organization-specific governance MAY track owner, target state, review condition, and migration progress, but these records do not waive the PTSIP rule or change the current `NON_CONFORMANT` result.

After remediation, rerun evidence collection and conformance evaluation against a stable snapshot.

## Phase 11 — Conformance Evaluation

Conformance Evaluation combines declaration, observed dependencies, Product Artifact evidence, lifecycle evidence, coverage and deterministic PTSIP rules.

Decision Authority state is architecture-decision coordination state. It does not replace the Project Profile, observed evidence, artifact evidence, or deterministic conformance evaluation.

The completed evaluation uses:

- `CONFORMANT`;
- `NON_CONFORMANT`; or
- `INCOMPLETE`.

Decision sequence:

```text
Definite applicable mandatory violation?
    yes -> NON_CONFORMANT
    no
     |
     v
Blocking evidence gap?
    yes -> INCOMPLETE
    no  -> eligible for CONFORMANT when all other requirements are satisfied
```

A successful read-only Pilot, resolved distributed decision, or empty finding list is not by itself a conformance result.

For Enforced Conformance against a mutable draft family, bind the exact immutable PTSIP specification revision. The `spec-v0.3.4-draft` design release is not by itself an active binding revision; a future claim against `0.3.4-draft` must use the immutable revision produced by the completed coherent normative migration.

## Phase 12 — Stable diagnostics and repeatable enforcement

Automated checks should emit stable diagnostics that distinguish:

- diagnostic instance ID;
- PTSIP rule ID;
- outcome effect;
- severity;
- source/target component where applicable;
- evidence IDs;
- message; and
- evaluator/provenance information.

The reference diagnostic contract is `ptsip-diagnostic/v1`.

CI, developer tooling, coding agents, and external reports should consume the same diagnostic semantics rather than parse ad-hoc human messages.

## Phase 13 — External evidence integration (optional)

A project may use external repository-specific validators for security, license, governance, or other concerns.

PTSIP does not need to reimplement those validators merely to use their results as analysis input.

Imported evidence should carry provenance such as producer/version, subject repository/revision, scope, generation time, claims, and integrity information.

External evidence is input to PTSIP evaluation; it does not automatically override contradictory native observed evidence or universal PTSIP rules.

## Phase 14 — Lifecycle review

Distinguish these lifecycle events:

- workflow/pipeline trigger;
- Product artifact change;
- Product version/release decision;
- Product publication/deployment; and
- Product compatibility obligation.

A Toolchain-only change triggering a shared Product workflow is not automatically a lifecycle violation. The relevant question is whether it forces Product lifecycle obligations without a release-relevant Product change.

## Phase 15 — Conformance claim

Only after stable, sufficient evidence exists should the project claim `PTSIP Core Conformant` or `PTSIP Enforced Conformant`.

A project undergoing remediation may describe itself as `PTSIP-adopting` or `PTSIP-transitioning`, but those adoption labels do not replace the current conformance outcome.

## Migration principle

PTSIP migration should optimize for **ownership correctness, evidence integrity, artifact truth, authority consistency, and future independent evolution**, not for minimizing the number of files changed in the first migration or maximizing automatic classification/coverage percentages.
