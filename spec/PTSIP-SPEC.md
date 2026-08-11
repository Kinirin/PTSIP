# PTSIP Specification

**Name:** Product–Toolchain SDK Isolation Policy  
**Acronym:** PTSIP  
**Version:** 0.3.4-draft  
**Status:** Active draft normative specification

The exact identity of this mutable draft family is the family label plus an immutable Git revision. Reference Tool builds that claim this family MUST bind an exact revision.

## 1. Scope

PTSIP defines how a software project classifies, builds, packages, depends on, releases, validates, and evolves SDKs and SDK-like components whose responsibility belongs to either Product or development Toolchain lifecycles.

PTSIP governs architectural ownership and lifecycle, not programming language, repository topology, package manager, deployment platform, documentation layout, or a required directory naming convention.

PTSIP also defines requirements for PTSIP implementations that coordinate explicit architecture decisions across multiple environments. Distributed decision coordination does not create another architecture plane and does not replace repository conformance evaluation.

## 2. Core model

PTSIP has exactly three architectural classifications:

- `PRODUCT` — Product-owned executable/library/SDK/component responsibility;
- `TOOLCHAIN` — development-tooling-owned executable/library/SDK/component responsibility;
- `NEUTRAL_CONTRACT` — deliberately non-executable, non-owning contract responsibility with independent lifecycle ownership.

`UNKNOWN`, `CONFLICT`, `INCOMPLETE`, `PENDING`, and similar workflow/evaluation states are not additional classifications.

### 2.1 Product SDK Plane

The Product SDK Plane contains components whose primary responsibility belongs to Product lifecycle and whose artifacts may be distributed, embedded, loaded, or depended on by Product runtime or Product-facing consumers.

### 2.2 Toolchain SDK Plane

The Toolchain SDK Plane contains development-only SDKs and tools used for validation, migration, generation, build, testing, release preparation, inspection, static analysis, repository transformation, compatibility auditing, or other development activities.

### 2.3 Neutral Contract Artifact

A Neutral Contract Artifact is a declarative or generated contract representation that MAY be consumed by both planes without becoming a shared executable SDK dependency. Examples include schemas, IDLs, registries, protocol definitions, test vectors, and immutable manifests.

A Neutral Contract Artifact MUST NOT become a hidden shared runtime implementation. Neutrality is determined by non-executable/non-owning semantics and lifecycle independence, not by directory name or a fixed current consumer count.

### 2.4 External PTSIP Tooling

External PTSIP Tooling is installed or executed outside the Consumer Repository project-owned source tree. It is outside Product/Toolchain classification scope unless the project intentionally vendors, embeds, packages, or takes lifecycle ownership of it.

### 2.5 PTSIP Component

A PTSIP Component is the coherent architectural unit to which one PTSIP classification, primary purpose, and lifecycle ownership statement can apply. It MAY be a package, module group, executable, generated artifact group, script, protocol bundle, or another project-defined unit.

### 2.6 Product Artifact

A Product Artifact is a deployable, distributable, installable, loadable, or otherwise Product-owned output. Artifact owner and artifact producer are distinct. A Toolchain component MAY produce a Product Artifact without becoming Product-owned.

## 3. Foundational principle

### PTSIP-CORE-001 — Purpose Before Reuse

Every in-scope component MUST be classified by primary purpose and lifecycle ownership before code reuse is considered.

Code similarity, convenience, directory naming, or DRY pressure MUST NOT by itself justify crossing the Product/Toolchain boundary. When reuse conflicts with lifecycle isolation, lifecycle isolation SHOULD take precedence.

## 4. Consumer Repository normative rules

### PTSIP-CLS-001 — Mandatory classification

Every in-scope project-owned SDK or SDK-like component MUST ultimately be `PRODUCT`, `TOOLCHAIN`, or `NEUTRAL_CONTRACT`.

Ambiguous ownership MAY remain unresolved during inspection/migration, but MUST be resolved before strict conformance is claimed or before the component becomes a shared architecture dependency whose legality depends on ownership.

### PTSIP-CLS-002 — Coherent component boundary

A component MUST be narrow enough that one classification, purpose, and lifecycle ownership statement coherently describes it. Materially contradictory shipped state, executable purpose, release responsibility, compatibility responsibility, build ownership, or Product/Toolchain lifecycle responsibility requires the project to split the component or explicitly redesign the boundary.

Physical co-location is permitted when declared component boundaries still represent the actual ownership.

### PTSIP-DEP-001 — Product-to-Toolchain runtime dependency prohibition

A Product component MUST NOT import, link, load, vendor, or otherwise depend on Toolchain implementation as a Product runtime or shipped dependency.

### PTSIP-DEP-002 — Toolchain inspection is permitted

A Toolchain component MAY inspect, parse, validate, transform, generate, test, migrate, or package Product source/artifacts when the purpose and lifecycle phase are bounded development activities and no Product-to-Toolchain runtime/package coupling is created.

Toolchain-to-Product direction alone does not automatically authorize executable implementation reuse.

### PTSIP-DEP-003 — Cross-boundary executable sharing denied by default

A project-local executable package used directly by both Product and Toolchain SHOULD NOT be introduced unless an explicit architecture decision demonstrates lifecycle independence and absence of release/runtime coupling. Otherwise separate implementations or a Neutral Contract SHOULD be preferred.

### PTSIP-PKG-001 — Packaging isolation

Toolchain implementation and Toolchain-only dependencies MUST NOT be included in Product deployable/distributable artifacts unless their ownership has been explicitly changed to Product through a valid architecture decision.

### PTSIP-ART-001 — Artifact ownership and derivation evidence

Artifact owner MUST be evaluated independently from producer ownership. Automated packaging evidence used for strict `PTSIP-PKG-001` evaluation MUST be sufficient to establish Product Artifact contents or equivalent packaging truth and SHOULD preserve producer/derivation relationships.

### PTSIP-BLD-001 — Independently resolvable build environments

Product and Toolchain planes MUST have independently resolvable dependency/build environments. A project MUST be able to determine each plane's direct dependencies without relying on accidental undeclared state from the other plane.

### PTSIP-BLD-002 — Independent buildability

A Product build MUST NOT require development-only Toolchain packages merely because both planes share a repository. Toolchain build/inspection MAY consume declared Product source/contracts as bounded inputs.

### PTSIP-LCY-001 — Lifecycle independence

Product and Toolchain versioning/release decisions MUST be independently governable. A Toolchain-only change SHOULD NOT force a Product release unless it changes a release-relevant Product Artifact or Product compatibility obligation.

A CI/workflow trigger alone is not proof of Product lifecycle coupling.

### PTSIP-LCY-002 — Compatibility ownership

Backward-compatibility obligations MUST be owned by the lifecycle whose consumers require them. Product compatibility requirements MUST NOT automatically freeze Toolchain-only interfaces with no Product consumer.

### PTSIP-CMN-001 — No unclassified common package

A package named `common`, `shared`, `core`, `utils`, or equivalent MUST NOT be exempt from PTSIP classification. Generic naming does not create neutral ownership.

### PTSIP-CNT-001 — Contract-first cross-boundary reuse

When Product and Toolchain need shared semantics, projects SHOULD prefer a Neutral Contract or generated separate implementations over one shared executable project-local implementation.

### PTSIP-POL-001 — Project policy may strengthen but not weaken PTSIP

A project MAY declare stricter component-to-component dependency policy, including same-plane constraints. Project policy MUST NOT authorize behavior prohibited by a universal PTSIP `MUST`/`MUST NOT`.

### Mandatory-rule violations require remediation

PTSIP defines no waiver that converts a real mandatory-rule violation into conformance. A confirmed violation remains `NON_CONFORMANT` until architecture is remediated and reevaluated. Project debt, approvals, or migration plans MAY be tracked externally but do not waive PTSIP.

`PTSIP-EXC-001` is retired and applies only to historical immutable snapshots where it existed.

## 5. PTSIP implementation and evidence rules

### PTSIP-INT-001 — Consumer Repository Non-Intrusion

External PTSIP Tooling MUST NOT require PTSIP-specific repository directories merely to operate. Inspection/Pilot operations MUST be read-only by default. Repository writes require explicit user action or explicit write-enabled mode. Tool-owned cache, reports, and local operational state SHOULD live outside the Consumer Repository by default.

### PTSIP-SPC-001 — Specification Binding

A machine-readable Project Profile used for automated conformance MUST identify the canonical Specification source and family/version. Enforced Conformance against a mutable draft MUST bind an exact immutable Specification revision.

A Reference Tool implementing a mutable draft MUST report the exact revision it implements and MUST NOT silently evaluate against a different normative revision.

### PTSIP-EVD-001 — Evidence snapshot integrity

Automated conformance evidence MUST identify the Consumer Repository revision or equivalent snapshot. If relevant repository state changes during collection, the evidence MUST be invalidated, marked incomplete, or otherwise prevented from supporting a strict conformance claim.

### PTSIP-EVD-002 — Declaration and observation are distinct

A Project Profile is intended architecture declaration, not proof of actual repository behavior. Validators MUST distinguish declaration from observed evidence and report material contradictions.

Agent/heuristic decisions MAY assist review but MUST NOT silently override project declaration or transform unresolved evidence into conformance fact.

### PTSIP-EVD-003 — Applicable evidence coverage

A strict conformance claim MUST NOT be derived solely from absence of detected violations. A coverage gap is blocking when it can conceal an applicable mandatory-rule result. Blocking gaps prevent `CONFORMANT` unless a definite violation already establishes `NON_CONFORMANT`.

PTSIP defines no universal unresolved-count or coverage-percentage threshold; sufficiency is rule-relative.

### PTSIP-EVD-004 — Evidence provenance and dependency semantics

Automated dependency evidence MUST preserve enough information to distinguish relationship type, lifecycle phase when known, resolution state, provenance, source/target identity when resolvable, and unresolved uncertainty.

Canonical provenance is `DECLARED`, `OBSERVED`, `INFERRED`. Unknown/dynamic/multi-phase evidence MUST NOT be forced into guessed values merely to complete evaluation.

### PTSIP-DIA-001 — Stable diagnostic identity

Automated diagnostics MUST distinguish diagnostic instance identity from stable PTSIP rule identity. Diagnostics SHOULD preserve rule ID, outcome effect, severity, evidence references, component endpoints where applicable, evaluator/provenance data, and reviewable message text.

## 6. Explicit Project Adoption

### PTSIP-ADP-001 — Explicit project adoption preserves architecture intent losslessly

Candidate discovery is not architecture authority. A write-enabled adoption/resolution workflow MUST obtain explicit project-owner/user architecture intent rather than infer missing intent from directory names, detected tooling behavior, or agent confidence.

When the following structured architecture facts are supplied for a component, an adoption/resolution workflow MUST preserve them losslessly in durable Project Profile semantics or refuse mutation when the selected profile representation cannot preserve them:

- `classification`;
- `purpose`;
- `shipped`;
- `runtime_required`;
- `lifecycle_owner`;
- `executable`.

Canonical lifecycle owners are:

- `PRODUCT`;
- `DEVELOPMENT_TOOLING`;
- `INDEPENDENT`.

Required relationships are:

- `PRODUCT` requires `lifecycle_owner = PRODUCT` when lifecycle ownership is represented;
- `TOOLCHAIN` requires `lifecycle_owner = DEVELOPMENT_TOOLING`;
- `TOOLCHAIN` MUST NOT be `shipped = true`;
- `TOOLCHAIN` MUST NOT be `runtime_required = true`;
- `NEUTRAL_CONTRACT` MUST be non-executable;
- `NEUTRAL_CONTRACT` requires `lifecycle_owner = INDEPENDENT` when lifecycle ownership is represented.

`release_owner` and `compatibility_owner` are separate optional project metadata. They MUST NOT be used as lossy aliases for canonical `lifecycle_owner`.

A boundary-root shorthand profile MAY remain structurally valid, but a write-enabled structured adoption workflow MUST NOT silently discard architecture facts merely to preserve shorthand. It MUST use a lossless component representation or stop for explicit migration.

Dry-run/planning MUST remain non-mutating. Before applying a prepared profile mutation, the implementation MUST re-check relevant repository/profile freshness and MUST refuse stale or concurrently changed content.

## 7. Decision Authority model

A **Decision Authority** coordinates unresolved/resolved explicit architecture decisions for a defined **coordination domain**.

Decision Authority and Project Profile are distinct:

```text
Decision Authority
    -> which explicit architecture answer won

Project Profile
    -> which durable architecture declaration this repository revision/worktree represents
```

A Decision Authority is not a conformance oracle and does not replace observed evidence.

A backend MAY be local, repository-distributed, or hosted. A backend claiming distributed coordination MUST satisfy `PTSIP-AUT-001` through `PTSIP-AUT-007`.

### PTSIP-AUT-001 — Authority/Profile responsibility separation

A distributed Decision Authority MUST NOT replace the Project Profile. The authoritative global winner and clone-local profile application state MUST remain separately representable.

### PTSIP-AUT-002 — Stable distributed decision identity

The same architectural component scope within one coordination domain MUST map to the same distributed decision identity across participating environments.

Identity MUST NOT depend solely on clone-local clarification IDs, branch-local missing-field lists, Local DecisionStore IDs, temporary operation IDs, or other incidental local state.

The exact encoding/hash is an interoperability choice unless separately standardized.

### PTSIP-AUT-003 — First valid resolution wins

For one distributed decision identity, the first valid accepted resolution wins. A later contradictory resolution MUST NOT silently replace it.

Distributed authority mutation MUST use ordered conditional-write semantics such as compare-and-swap, transaction, consensus, generation/ETag matching, immutable-ref ancestry, or equivalent stale-writer protection.

After a conditional-write race, an implementation MUST reread the current authority. If the same-scope winner is already resolved, it MUST accept that winner rather than retrying a contradictory answer.

### PTSIP-AUT-004 — Authority freshness

When an architecture-sensitive operation is using distributed coordination, the implementation MUST account for relevant current authority state before returning a result that relies on the local Project Profile as authoritative for that coordinated scope.

A complete local declaration MUST NOT cause an early success path that skips a relevant distributed authority check.

A read-only absence check SHOULD be non-mutating and MUST NOT fabricate pending decision history merely to prove that no authority decision exists.

### PTSIP-AUT-005 — Safe authority/profile reconciliation

Distributed coordination MUST distinguish at least these semantics:

| Local Project Profile | Distributed Authority | Required behavior |
| --- | --- | --- |
| declaration absent | no decision | create/reuse pending state only when active operation actually requires a decision |
| declaration absent | resolved winner | validate and safely project/reconcile winner locally |
| declaration present | no authority decision | use project declaration; do not fabricate authority history solely for bookkeeping |
| declaration present + semantically equivalent | resolved equivalent winner | report consistency/resolution without rewriting equivalent text |
| declaration present + semantically conflicting | resolved different winner | expose explicit authority/profile conflict; do not silently overwrite either side |
| repository/profile changed during reconciliation | any authority state | refuse stale application and require re-analysis |

Semantic equivalence MUST compare architecture meaning, not incidental YAML formatting, key ordering, whitespace, or Tool-generated serialization.

Automatic local projection of a remote winner MUST validate the authoritative answer, repository scope, repository/profile freshness, projected profile validity, concurrent-content safety, and local write result.

### PTSIP-AUT-006 — Fail-closed distributed coordination

When distributed coordination is selected and the operation requires authority freshness or safe mutation, failure to safely read/mutate authority MUST NOT silently fall back to an isolated Local winner.

Authentication failure, permission failure, network unavailability, malformed authority data, incompatible authority ownership, conditional-write failure, or inability to establish required freshness MUST stop the affected coordinated operation explicitly.

### PTSIP-AUT-007 — Global decision state versus local projection state

Global decision state such as `PENDING`/`RESOLVED` MUST remain distinct from clone/worktree-local states such as missing, consistent, locally applied, stale, or failed.

A global `RESOLVED` decision MUST NOT imply that every clone has applied the declaration. Local application receipts MUST NOT alter which architecture answer won.

## 8. Action-time synchronization

PTSIP does not require continuous background polling. A conforming distributed implementation MAY consult relevant authority at architecture-sensitive action boundaries.

Correctness MUST NOT depend on every clone immediately receiving another clone's `ptsip.yaml` commit.

A coding agent MUST stop only work that actually depends on unresolved/conflicting coordinated architecture state. It MUST NOT invent an answer to avoid the gate.

## 9. Project Profile

A Project Profile is project-owned machine-readable architecture declaration. The reference schema is `schemas/ptsip-profile.schema.json`.

For `0.3.4-draft`, the profile binds:

- `ptsip.version: 0.3.4-draft`;
- canonical Specification source;
- immutable Specification revision for reproducible Enforced Conformance.

The reference schema supports two mutually exclusive ownership modes:

1. `boundaries` — uniform-root shorthand;
2. `components` — precise component declarations.

A component declaration identifies at least a stable ID, classification, selectors, and purpose. It MAY record shipped/runtime/lifecycle/executable facts and project metadata such as release owner, compatibility owner, manifests, consumers, or analysis inputs.

`runtime_required` is a boolean architecture fact. `lifecycle_owner` is canonical and uses `PRODUCT`, `DEVELOPMENT_TOOLING`, or `INDEPENDENT`.

When selectors overlap, implementations MUST use deterministic specificity rules and MUST reject equal-specificity conflicting ownership rather than choosing arbitrarily.

A Project Profile is declaration, not observed conformance truth.

## 10. Evidence vocabulary

Canonical relationship types are:

- `IMPORTS`, `LINKS`, `LOADS`, `INVOKES`, `READS`, `GENERATES`, `PACKAGES`, `TESTS`, `PUBLISHES`.

Canonical lifecycle phases are:

- `RUNTIME`, `BUILD`, `TEST`, `RELEASE`, `INSPECTION`.

Evidence-node scopes MAY include:

- `PROJECT_COMPONENT`, `EXTERNAL_DEPENDENCY`, `PLATFORM`, `UNRESOLVED_TARGET`.

These scopes are not architecture classifications.

## 11. Validation, authority reconciliation, and conformance

These are distinct operations:

**Profile Validation** — checks Project Profile structure and semantics.

**Authority reconciliation** — when distributed coordination is selected, checks whether the relevant local declaration is consistent with the current coordinated winner and applies safe reconciliation rules.

**Conformance Evaluation** — combines validated declaration with observed dependency, artifact, build, lifecycle, snapshot, and coverage evidence against Consumer Repository PTSIP rules.

A valid profile does not prove conformance. A resolved authority winner does not prove conformance. A repository may be `NON_CONFORMANT` even when declaration and authority are perfectly synchronized.

Completed Consumer Repository conformance outcomes are only:

- `CONFORMANT`;
- `NON_CONFORMANT`;
- `INCOMPLETE`.

`NOT_EVALUATED` MAY be used as Tool execution state but is not a conformance outcome.

Distributed-coordination implementation rules (`PTSIP-AUT-*`) apply only to an implementation claiming distributed coordination semantics. A Consumer Repository is not required to use distributed coordination merely to satisfy Product/Toolchain architecture rules.

## 12. Reuse policy

Acceptable strategies include shared declarative contracts, generation of separate implementations, independent implementations validated against common conformance vectors, and separately governed third-party/platform dependencies.

Risky strategies include project-local executable `shared` packages imported by both planes, Toolchain validators imported into Product runtime, unbounded Product implementation reuse by Toolchain merely for convenience, and one accidental dependency environment that makes both planes build.

## 13. Repository topology

PTSIP does not require a monorepo or multirepo and does not prescribe `docs/`, `tools/`, `.ptsip/`, `product/`, `toolchain/`, or another directory hierarchy.

Directory names are evidence hints only. Purpose and lifecycle ownership determine architecture.

## 14. Non-goals

PTSIP does not prescribe programming paradigm, build system, package manager, test framework, CI provider, service topology, repository count, documentation hierarchy, continuous authority polling, GitHub as a universal dependency, or automatic LLM architecture classification.

PTSIP does not require shared SQLite through Git and does not define a fourth architecture plane for distributed authority state.

## 15. Status and novelty statement

PTSIP is a project-defined draft architecture policy. It does not claim invention of host/target separation, build-time/runtime separation, toolchain isolation, dependency isolation, or independent lifecycle management. Its contribution is a reproducible SDK-oriented governance model combining those ideas with explicit classification, conformance, evidence, adoption, and distributed architecture-decision consistency semantics.
