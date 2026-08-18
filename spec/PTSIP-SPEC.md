# PTSIP Specification

**Name:** Primary Lifecycle Ownership and Responsibility Isolation Policy  
**Acronym:** PTSIP  
**Version:** 0.3.6-draft  
**Status:** Active draft normative specification

The exact identity of this mutable draft family is the family label plus an immutable Git revision. Reference Tool builds that claim this family MUST bind an exact revision.

## 1. Scope

PTSIP defines how a software project classifies, builds, packages, depends on, releases, delivers, operates, validates, and evolves project-owned responsibilities according to **primary lifecycle ownership**.

PTSIP governs architectural ownership and lifecycle. It does not prescribe programming language, framework, repository topology, package manager, deployment platform, documentation layout, test layout, or a required directory naming convention.

PTSIP also defines requirements for Project Profiles, evidence, explicit adoption/migration, distributed architecture-decision coordination, Responsibility Map evolution, and verification-purpose separation.

## 2. Core model

### 2.1 Classification means primary lifecycle ownership

For `0.3.6-draft`, `classification` is the canonical primary lifecycle ownership of an in-scope project-owned architectural responsibility.

The canonical classifications are exactly:

- `PRODUCT`;
- `DEVELOPMENT_TOOLING`;
- `DELIVERY`;
- `OPERATIONS`;
- `NEUTRAL_CONTRACT`.

`UNKNOWN`, `CONFLICT`, `INCOMPLETE`, `PENDING`, confidence values, migration states, and similar workflow/evaluation states are not classifications.

A filename, directory, language, framework, executable bit, compilation behavior, workflow provider, or the fact that an artifact is a test MUST NOT by itself determine classification.

### 2.2 `PRODUCT`

`PRODUCT` is responsibility owned directly by the Product lifecycle. It includes Product runtime, user-facing behavior, Product distribution content, runtime SDK responsibility, and Product-specific quality or verification responsibility.

Representative examples MAY include Product-owned tests, FastAPI applications, Cloudflare Worker applications, desktop applications, runtime SDKs, Product-specific validation, and Product package contents.

A test is not automatically Development Tooling. A test whose existence, compatibility expectations, and change responsibility are owned by a Product MAY be `PRODUCT`.

### 2.3 `DEVELOPMENT_TOOLING`

`DEVELOPMENT_TOOLING` is responsibility owned by the development lifecycle and used to create, transform, inspect, validate, migrate, generate, analyze, or otherwise support development work.

Representative examples MAY include reusable tests or verification infrastructure, test SDKs/frameworks, generators, migration tooling, linting, static analysis, developer CLI tooling, build helpers, repository transformation utilities, and development-environment tooling.

`DEVELOPMENT_TOOLING` carries the core development-tool responsibility represented by much of Tool `0.3.5` `TOOLCHAIN`, but it is not a one-to-one replacement for every legacy `TOOLCHAIN` component.

### 2.4 `DELIVERY`

`DELIVERY` is responsibility for carrying a release target or Product through release, publication, promotion, distribution, or deployment to its destination environment.

Representative examples MAY include release workflows, package publication, container image build/publication, artifact signing, deployment automation, deployment manifests/configuration, and release promotion.

The Product/Delivery distinction is ownership of **what exists as Product responsibility** versus ownership of **how a release target is carried to a publication/deployment destination**.

### 2.5 `OPERATIONS`

`OPERATIONS` is responsibility owned by the post-deployment operational lifecycle: keeping a deployed system healthy, recoverable, maintained, and operational over time.

Representative examples MAY include ongoing production provisioning, runtime infrastructure management, operational automation, backup/recovery, production maintenance, incident tooling, and production health management.

`DELIVERY` and `OPERATIONS` are separated by primary lifecycle purpose, not by technology. A deployment or infrastructure artifact that materially performs both responsibilities requires an explicit coherent boundary or project-owner resolution.

### 2.6 `NEUTRAL_CONTRACT`

`NEUTRAL_CONTRACT` is deliberately non-executable, non-owning contract responsibility with lifecycle independence from Product, Development Tooling, Delivery, and Operations ownership.

Examples MAY include schemas, IDLs, registries, protocol definitions, test vectors, and immutable contract manifests when those semantics are actually present.

A file does not become neutral merely because it is declarative, Markdown/YAML, shared by several consumers, or non-executable. Neutrality requires non-owning semantics and independent lifecycle ownership.

### 2.7 External dependencies and external PTSIP tooling

External frameworks, platforms, libraries, services, and external PTSIP Tooling are not project-owned classifications merely because a project uses them. FastAPI, Cloudflare Workers, GitHub Actions, Docker, Terraform, Python, PowerShell, Markdown, YAML, or another technology name is evidence context, not architecture authority.

An external dependency becomes in-scope only when the project intentionally vendors, embeds, packages, or takes lifecycle ownership of project-local responsibility around it.

### 2.8 PTSIP Component

A PTSIP Component is a coherent architectural unit to which one primary lifecycle classification and one primary purpose can apply. It MAY be a package, module group, executable, generated artifact group, script, protocol bundle, workflow group, infrastructure responsibility, verification unit, or another project-defined unit.

A component MUST be narrow enough that its primary lifecycle ownership is coherent. Materially contradictory lifecycle ownership requires a split or explicit redesign.

### 2.9 Product Artifact

A Product Artifact is a deployable, distributable, installable, loadable, or otherwise Product-owned output. Artifact owner and artifact producer are distinct. A Development Tooling or Delivery component MAY produce/package/publish a Product Artifact without becoming Product-owned.

## 3. Foundational principles

### PTSIP-CORE-001 — Purpose Before Reuse

Every in-scope component MUST be classified by primary purpose and lifecycle ownership before code reuse is considered.

Code similarity, convenience, directory naming, or DRY pressure MUST NOT by itself justify crossing lifecycle ownership boundaries. When reuse conflicts with lifecycle isolation, lifecycle isolation SHOULD take precedence.

### PTSIP-CORE-002 — Classification, role, relationship, and verification purpose are distinct axes

PTSIP classification, responsibility role, typed relationship, and VPMS Verification Purpose MUST NOT be collapsed into one field or inferred from one another without an explicit normative rule.

Conceptually:

```text
classification
    = primary lifecycle ownership

role
    = responsibility performed inside that lifecycle

relationship
    = typed semantic relationship to another responsibility/artifact

VPMS Verification Purpose
    = what a Verification Case protects/verifies
```

The exact closed role vocabulary and exact Responsibility Map v2 schema representation are standardized separately inside the `0.3.6-draft` family. Implementations MUST NOT create new lifecycle classifications merely to encode internal roles.

## 4. Consumer Repository normative rules

### PTSIP-CLS-001 — Mandatory classification

Every in-scope project-owned component MUST ultimately be `PRODUCT`, `DEVELOPMENT_TOOLING`, `DELIVERY`, `OPERATIONS`, or `NEUTRAL_CONTRACT`.

Ambiguous ownership MAY remain unresolved during inspection/migration, but MUST be resolved before strict conformance is claimed or before the component becomes a shared architecture dependency whose legality depends on ownership.

### PTSIP-CLS-002 — Coherent component boundary

A component MUST be narrow enough that one classification and primary purpose coherently describe it. Materially contradictory runtime responsibility, development responsibility, delivery responsibility, operational responsibility, independent-contract ownership, shipped state, release responsibility, compatibility responsibility, or dependency obligations require the project to split the component or explicitly redesign the boundary.

Physical co-location is permitted when declared component boundaries still represent actual ownership.

### PTSIP-CLS-003 — Artifact kind is not classification authority

A project MUST NOT be declared compliant/non-compliant or be automatically migrated solely because a path is named `tests`, `src`, `tools`, `deploy`, `ops`, `docs`, or because a technology/framework is detected.

Product-owned tests and Development-Tooling-owned tests are both valid patterns. PTSIP MUST NOT force one repository/testing style merely to satisfy classification.

### PTSIP-CLS-004 — Governing lifecycle obligation

Primary lifecycle ownership MUST be determined from the **governing lifecycle obligation** of a coherent responsibility: the lifecycle-specific reason that responsibility must exist, change, remain compatible, execute, or be retired.

Classification MUST NOT be selected by majority of files, lines, workflow jobs, runtime duration, invocation frequency, confidence score, or whichever lifecycle phase appears most often.

Relevant determination questions include:

```text
Why does this responsibility exist?
Which lifecycle obligation fails if it disappears?
What type of change normally requires it to change?
Who owns its compatibility consequences?
When is it normally invoked or enforced?
Does its obligation end when a release reaches its destination?
Does its obligation continue after deployment as ongoing service management?
Can it evolve independently from another responsibility currently grouped with it?
```

### PTSIP-CLS-005 — Ordered lifecycle determination procedure

A conforming adoption/migration analysis SHOULD apply lifecycle determination in this order:

1. establish that the candidate is in-scope project-owned responsibility;
2. identify a coherent responsibility boundary before assigning ownership;
3. collect declared/observed/inferred lifecycle evidence with provenance;
4. test whether the responsibility satisfies all `NEUTRAL_CONTRACT` semantics;
5. otherwise determine whether the governing owning lifecycle is Product, Development Tooling, Delivery, or Operations;
6. test whether materially independent lifecycle responsibilities have been collapsed into one candidate;
7. propose one classification, a component split, or unresolved clarification;
8. preserve project-owner authority for inferred/migrated architecture decisions.

A Tool MAY optimize how it collects evidence, but MUST NOT replace this semantic procedure with filename/framework/path inference.

### PTSIP-CLS-006 — Product versus Development Tooling boundary

A responsibility is a `PRODUCT` candidate when its governing obligation is Product runtime/user behavior, Product distribution content, Product compatibility, runtime SDK responsibility, or Product-owned quality/verification.

A responsibility is a `DEVELOPMENT_TOOLING` candidate when its governing obligation is developer authoring, inspection, validation, transformation, migration, generation, static analysis, reusable verification infrastructure, repository transformation, development-environment support, or non-delivery development build support.

A test target does not determine test implementation ownership. In particular:

- a Product-specific test whose lifecycle is owned with Product behavior/compatibility MAY be `PRODUCT`;
- a reusable test SDK, framework, harness, generic verification engine, or repository-wide reusable validation mechanism MAY be `DEVELOPMENT_TOOLING` even when it verifies Product behavior;
- current consumer count alone MUST NOT convert reusable Development Tooling into Product ownership.

VPMS Verification Purpose remains a separate axis and MUST NOT be used as a direct alias for PTSIP classification.

### PTSIP-CLS-007 — Development build versus Delivery build boundary

The activity name `build` MUST NOT determine classification.

Build responsibility is normally `DEVELOPMENT_TOOLING` when its governing obligation is coding, local compilation, inspection, testing, or creation of non-release intermediate outputs.

Build responsibility is normally `DELIVERY` when its governing obligation is to materialize, assemble, sign, verify for handoff, package, or otherwise prepare an authoritative release/distribution/deployment unit as part of carrying that target to a destination.

The Product Artifact created by such a process MAY remain `PRODUCT` even when the producing mechanism is `DELIVERY`. Artifact ownership and producer ownership are distinct.

### PTSIP-CLS-008 — Delivery versus Operations handoff boundary

`DELIVERY` governs transition of a release target through release, publication, promotion, distribution, or deployment to a destination.

`OPERATIONS` governs ongoing management of already deployed state after ordinary operation begins.

The **delivery handoff** is the semantic boundary where the selected release target has reached/been accepted by its intended destination and ongoing operational responsibility begins.

When both classifications appear plausible:

```text
Does the primary obligation end when the selected release reaches/activates at its destination?
    yes -> DELIVERY candidate

Does the primary obligation persist because deployed state must remain healthy/recoverable/maintained?
    yes -> OPERATIONS candidate
```

Technology does not decide the result. Terraform, GitHub Actions, shell/PowerShell automation, containers, or another mechanism MAY belong to either lifecycle depending on governing obligation.

Rollback is likewise not universally classified by name: release-channel rollback/promotion MAY be `DELIVERY`; incident-recovery responsibility owned by ongoing production operation MAY be `OPERATIONS`.

### PTSIP-CLS-009 — Neutral Contract qualification

`NEUTRAL_CONTRACT` MUST NOT be used as a default bucket for non-executable, declarative, documentation-like, or shared files.

A `NEUTRAL_CONTRACT` candidate MUST satisfy all of these conditions:

1. non-executable in its architectural role;
2. non-owning with respect to Product, Development Tooling, Delivery, and Operations implementation responsibility;
3. lifecycle/compatibility governance meaningfully independent from the consuming lifecycles.

Markdown/YAML form, a `docs/`/`schemas/`/`contracts/` path, multiple consumers, no executable bit, or a name containing `contract` are insufficient by themselves.

Project-owned documentation or authority material that evolves with and governs one component's lifecycle is not neutral merely because it is non-executable. Such material MAY instead require associated-artifact/typed-relationship representation under Responsibility Map v2.

### PTSIP-CLS-010 — Material mixed-lifecycle responsibility and split rule

A component has a material mixed-lifecycle problem when one classification cannot describe its lifecycle truth without hiding an independently governable responsibility.

Strong evidence for a required split/redesign includes one or more of:

- different lifecycle triggers for change or execution;
- different release, compatibility, permission, secret, or environment obligations;
- independently evolving dependency environments;
- different failure meanings or operational owners;
- one responsibility can change, release, deploy, or operate independently of another;
- one portion remains useful after another lifecycle responsibility is removed;
- migration evidence maps substantial portions of one legacy component to different governing lifecycle obligations.

Physical co-location, multiple workflow steps, or several activity verbs do not by themselves require a split. A component MAY contain subordinate activities from another phase when those activities exist solely to complete one coherent primary lifecycle obligation and introduce no independently governable lifecycle responsibility.

A workflow that combines independently governable development verification, release delivery, and ongoing operational responsibility SHOULD be split or explicitly redesigned rather than classified by whichever phase has more jobs or longer runtime.

### PTSIP-CLS-011 — Ambiguity and fail-closed classification proposals

Tooling MAY propose a classification when available evidence coherently supports one lifecycle, and MAY attach confidence for review. Confidence MUST NOT become architecture authority.

When material evidence supports conflicting lifecycle ownership:

```text
coherent single lifecycle
    -> propose one classification

material separable lifecycles
    -> propose component split

material but not safely resolvable
    -> remain unresolved / require clarification
```

A Tool MUST NOT select an arbitrary lifecycle merely to complete migration, validation, or profile generation.

The rationale for `PTSIP-CLS-004` through `PTSIP-CLS-011` is recorded in `decisions/ADR-0007-primary-lifecycle-boundary-determination.md`.

### PTSIP-DEP-001 — Product runtime isolation from non-Product implementation

A Product component MUST NOT import, link, load, vendor, or otherwise depend on `DEVELOPMENT_TOOLING`, `DELIVERY`, or `OPERATIONS` implementation as a Product runtime or shipped implementation dependency.

This rule does not prohibit a Product component from consuming a valid `NEUTRAL_CONTRACT`, external platform contract, generated Product implementation, or explicit Product-owned artifact.

### PTSIP-DEP-002 — Bounded lifecycle inspection and transformation are permitted

Development Tooling MAY inspect, parse, validate, transform, generate, test, or migrate Product source/artifacts when those are bounded development activities and do not create Product runtime/package coupling.

Delivery MAY build, package, sign, publish, promote, or deploy Product release targets without becoming Product-owned.

Operations MAY inspect and manage deployed Product state as bounded operational responsibility without becoming Product-owned.

Direction alone does not authorize executable implementation reuse.

### PTSIP-DEP-003 — Cross-lifecycle executable sharing denied by default

A project-local executable package used directly as implementation by multiple primary lifecycle classifications SHOULD NOT be introduced unless an explicit architecture decision demonstrates that the ownership and release/runtime obligations remain coherent. Prefer lifecycle-owned implementations, generated separate implementations, external/platform dependencies, or a `NEUTRAL_CONTRACT` when appropriate.

### PTSIP-PKG-001 — Product packaging isolation

`DEVELOPMENT_TOOLING`, `DELIVERY`, and `OPERATIONS` implementation and their lifecycle-only dependencies MUST NOT be included in Product deployable/distributable artifacts unless their ownership has explicitly changed to `PRODUCT` through a valid architecture decision.

### PTSIP-ART-001 — Artifact ownership and derivation evidence

Artifact owner MUST be evaluated independently from producer ownership. Automated packaging evidence used for strict packaging evaluation MUST be sufficient to establish Product Artifact contents or equivalent packaging truth and SHOULD preserve producer/derivation relationships.

### PTSIP-BLD-001 — Independently resolvable lifecycle environments

Each executable lifecycle responsibility MUST have determinable direct dependencies without relying on accidental undeclared state from another lifecycle. PTSIP does not require identical build mechanisms or physically separate repositories/environments.

### PTSIP-BLD-002 — Product buildability

A Product build MUST NOT require Development Tooling, Delivery, or Operations packages merely because those responsibilities share a repository. Bounded build/delivery tooling MAY consume declared Product source/contracts/artifacts as inputs.

### PTSIP-LCY-001 — Lifecycle independence

Versioning, release, delivery, operations, and compatibility decisions MUST be governable according to the lifecycle that owns them. A change in one lifecycle SHOULD NOT force an unrelated lifecycle release unless it changes a release-relevant artifact, compatibility obligation, or explicitly coupled contract.

A CI/workflow trigger alone is not proof of lifecycle ownership.

### PTSIP-LCY-002 — Compatibility ownership

Backward-compatibility obligations MUST be owned by the lifecycle whose consumers require them. Product compatibility requirements MUST NOT automatically freeze Development-Tooling-only, Delivery-only, or Operations-only interfaces with no Product consumer.

### PTSIP-CMN-001 — No unclassified common package

A package named `common`, `shared`, `core`, `utils`, or equivalent MUST NOT be exempt from PTSIP classification. Generic naming does not create neutral ownership.

### PTSIP-CNT-001 — Contract-first cross-lifecycle reuse

When multiple lifecycles need shared semantics, projects SHOULD prefer a valid `NEUTRAL_CONTRACT`, generated lifecycle-owned implementations, or separately governed external/platform contracts over one shared project-local executable implementation.

### PTSIP-POL-001 — Project policy may strengthen but not weaken PTSIP

A project MAY declare stricter component-to-component dependency policy. Project policy MUST NOT authorize behavior prohibited by a universal PTSIP `MUST`/`MUST NOT`.

### Mandatory-rule violations require remediation

PTSIP defines no waiver that converts a real mandatory-rule violation into conformance. A confirmed violation remains `NON_CONFORMANT` until architecture is remediated and reevaluated. Project debt, approvals, or migration plans MAY be tracked externally but do not waive PTSIP.

## 5. Responsibility Map v2 direction

### PTSIP-RMAP-001 — Project-owned responsibility declaration

A Responsibility Map is project-owned machine-readable architecture declaration. Discovery output, templates, migration proposals, and agent confidence are not architecture authority.

Tool `0.3.6` MUST support these conceptual declaration modes:

- **explicit** — the repository directly declares the complete canonical map;
- **template** — the repository explicitly selects one versioned/revision-bound supported template;
- **hybrid** — the repository selects a template and supplies project-owned overrides/extensions.

Template selection MUST be explicit. A Tool MUST NOT silently select a template from paths, language, framework, manifests, package manager, or discovery confidence.

### PTSIP-RMAP-002 — Role and typed relationship preservation

Responsibility Map v2 MUST be able to preserve materially relevant responsibility roles and typed relationships without creating additional lifecycle classifications merely to encode those semantics.

Implementations SHOULD reuse existing relationship concepts where semantics already exist. Existing evidence vocabulary includes `IMPORTS`, `LINKS`, `LOADS`, `INVOKES`, `READS`, `GENERATES`, `PACKAGES`, `TESTS`, and `PUBLISHES`.

Governance/documentation relationship concepts such as `DOCUMENTS`, `SPECIFIES`, and `GOVERNS` MAY be standardized when required to represent project-owned architecture truth. Exact canonical relationship IDs are frozen by the Responsibility Map v2 schema/rule work, not by filename inference.

### PTSIP-RMAP-003 — Associated artifacts

Responsibility Map v2 MUST be able to represent project-owned associated documentation/authority/support artifacts that relate to a component but do not themselves require promotion to an independent architecture component merely to express that relationship.

Associated-artifact representation MUST NOT become a classification escape hatch.

At minimum:

- executable responsibility requires component evaluation;
- independent release/compatibility/lifecycle ownership requires component evaluation;
- an independently owned cross-lifecycle contract requires `NEUTRAL_CONTRACT` evaluation;
- associated artifacts MUST NOT weaken mandatory dependency, packaging, lifecycle, or conformance rules.

## 6. Tool 0.3.5 compatibility and migration

### PTSIP-MIG-001 — Legacy profile readability without legacy ontology preservation

Tool `0.3.5` profiles use the canonical classifications `PRODUCT`, `TOOLCHAIN`, and `NEUTRAL_CONTRACT`.

A Tool `0.3.6` implementation MUST be able to recognize valid Tool `0.3.5` profiles as **legacy migration inputs**.

`TOOLCHAIN` MUST NOT be emitted as a canonical Tool `0.3.6` classification and MUST NOT remain in the canonical Tool `0.3.6` classification enum merely as a compatibility alias.

Legacy compatibility therefore means **understand and migrate**, not **silently reinterpret and preserve obsolete semantics**.

### PTSIP-MIG-002 — Evidence-driven migration proposals

Migration from Tool `0.3.5` MUST be preview-first and evidence-backed.

A legacy `TOOLCHAIN` component MUST NOT be blindly renamed to `DEVELOPMENT_TOOLING`. Evidence MAY support proposals for `PRODUCT`, `DEVELOPMENT_TOOLING`, `DELIVERY`, `OPERATIONS`, component splits, role changes, typed relationships, or unresolved clarification.

Legacy `PRODUCT` and `NEUTRAL_CONTRACT` components MAY have high-probability carry-forward mappings, but implementations MUST still be able to expose boundary splits or reclassification proposals when evidence contradicts the old coarse boundary.

A migration proposal SHOULD expose supporting evidence, provenance, confidence/ambiguity, component-boundary effects, semantic diff, textual diff, and unresolved facts.

### PTSIP-MIG-003 — Project-owner confirmation

Candidate discovery and migration analysis MUST NOT silently rewrite project-owned architecture intent.

A coding agent or CLI MAY perform repository-wide evidence collection and propose mappings so the maintainer does not bear all discovery cost. Final architecture confirmation remains project-owned.

If a target representation cannot preserve confirmed architecture intent losslessly, migration MUST stop instead of dropping or guessing facts.

## 7. PTSIP implementation and evidence rules

### PTSIP-INT-001 — Consumer Repository Non-Intrusion

External PTSIP Tooling MUST NOT require PTSIP-specific repository directories merely to operate. Inspection/Pilot operations MUST be read-only by default. Repository writes require explicit user action or explicit write-enabled mode. Tool-owned cache, reports, and local operational state SHOULD live outside the Consumer Repository by default.

### PTSIP-SPC-001 — Specification Binding

A machine-readable Project Profile used for automated conformance MUST identify the canonical Specification source and family/version. Enforced Conformance against a mutable draft MUST bind an exact immutable Specification revision.

A Reference Tool implementing a mutable draft MUST report the exact revision it implements and MUST NOT silently evaluate against a different normative revision.

### PTSIP-EVD-001 — Evidence snapshot integrity

Automated conformance or migration evidence MUST identify the Consumer Repository revision or equivalent snapshot. If relevant repository state changes during collection, evidence MUST be invalidated, marked incomplete, or otherwise prevented from supporting stale architecture/conformance claims.

### PTSIP-EVD-002 — Declaration and observation are distinct

A Project Profile is intended architecture declaration, not proof of actual repository behavior. Validators MUST distinguish declaration from observed evidence and report material contradictions.

Agent/heuristic decisions MAY assist review but MUST NOT silently override project declaration or transform unresolved evidence into architecture/conformance fact.

### PTSIP-EVD-003 — Applicable evidence coverage

A strict conformance claim MUST NOT be derived solely from absence of detected violations. A coverage gap is blocking when it can conceal an applicable mandatory-rule result. Blocking gaps prevent `CONFORMANT` unless a definite violation already establishes `NON_CONFORMANT`.

PTSIP defines no universal unresolved-count or coverage-percentage threshold; sufficiency is rule-relative.

### PTSIP-EVD-004 — Evidence provenance and relationship semantics

Automated evidence MUST preserve enough information to distinguish relationship type, lifecycle phase when known, resolution state, provenance, source/target identity when resolvable, and unresolved uncertainty.

Canonical provenance is `DECLARED`, `OBSERVED`, `INFERRED`. Unknown/dynamic/multi-phase evidence MUST NOT be forced into guessed values merely to complete evaluation.

### PTSIP-EVD-005 — Candidate discovery is evidence acquisition

Candidate discovery MAY use invocation, manifest, dependency, packaging, workflow/build, delivery, operations, and explicit project-declaration evidence.

Path/name heuristics MAY remain low-confidence evidence but MUST NOT dominate stronger evidence or become project architecture authority.

Discovery MAY propose lifecycle classification, role, relationship, associated-artifact, or component-split candidates. It MUST NOT silently finalize project ownership.

### PTSIP-DIA-001 — Stable diagnostic identity

Automated diagnostics MUST distinguish diagnostic instance identity from stable PTSIP rule identity. Diagnostics SHOULD preserve rule ID, outcome effect, severity, evidence references, component endpoints where applicable, evaluator/provenance data, and reviewable message text.

## 8. Explicit Project Adoption

### PTSIP-ADP-001 — Explicit project adoption preserves architecture intent losslessly

Candidate discovery is not architecture authority. A write-enabled adoption/resolution workflow MUST obtain explicit project-owner/user architecture intent rather than infer missing intent from directory names, detected tooling behavior, or agent confidence.

When structured architecture facts are supplied, an adoption/resolution workflow MUST preserve them losslessly in durable Project Profile semantics or refuse mutation when the selected representation cannot preserve them.

For `0.3.6-draft`, canonical lifecycle ownership is expressed by `classification` itself. A separate legacy `lifecycle_owner` field, if accepted by migration tooling, MUST NOT override or contradict canonical `classification` semantics. Whether Responsibility Map v2 retains such a compatibility field is a schema-level decision and MUST NOT create two competing ownership authorities.

Dry-run/planning MUST remain non-mutating. Before applying a prepared profile mutation, the implementation MUST re-check relevant repository/profile freshness and MUST refuse stale or concurrently changed content.

## 9. Decision Authority model

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

### PTSIP-AUT-007 — Global decision state versus local projection state

Global decision state such as `PENDING`/`RESOLVED` MUST remain distinct from clone/worktree-local states such as missing, consistent, locally applied, stale, or failed.

A global `RESOLVED` decision MUST NOT imply that every clone has applied the declaration. Local application receipts MUST NOT alter which architecture answer won.

## 10. Action-time synchronization

PTSIP does not require continuous background polling. A conforming distributed implementation MAY consult relevant authority at architecture-sensitive action boundaries.

Correctness MUST NOT depend on every clone immediately receiving another clone's `ptsip.yaml` commit.

A coding agent MUST stop only work that actually depends on unresolved/conflicting coordinated architecture state. It MUST NOT invent an answer to avoid the gate.

## 11. Project Profile and Responsibility Map schema

A Project Profile is project-owned machine-readable architecture declaration. Tool `0.3.6` introduces a Responsibility Map v2 schema direction that MUST represent the five canonical lifecycle classifications and MUST NOT retain legacy `TOOLCHAIN` as a canonical classification alias.

The exact v2 schema, supported template identities, override representation, role vocabulary, relationship vocabulary, and associated-artifact structure are frozen by the `0.3.6` schema work following this baseline.

A valid Tool `0.3.5` profile MAY remain readable by a Tool `0.3.6` legacy reader without becoming structurally canonical `0.3.6` input.

When selectors overlap, implementations MUST use deterministic specificity rules and MUST reject equal-specificity conflicting ownership rather than choosing arbitrarily.

A Project Profile is declaration, not observed conformance truth.

## 12. Evidence vocabulary

Canonical evidence relationship types carried forward from `0.3.4-draft` are:

- `IMPORTS`, `LINKS`, `LOADS`, `INVOKES`, `READS`, `GENERATES`, `PACKAGES`, `TESTS`, `PUBLISHES`.

Additional Responsibility Map typed relationships MAY be standardized in this draft family where they represent project-declared architecture semantics rather than observed dependency evidence.

Canonical lifecycle phases carried forward are:

- `RUNTIME`, `BUILD`, `TEST`, `RELEASE`, `INSPECTION`.

The `0.3.6` evidence-model work MAY extend lifecycle phases for delivery/operations evidence, but implementations MUST NOT guess a phase merely to complete evaluation.

Evidence-node scopes MAY include:

- `PROJECT_COMPONENT`, `EXTERNAL_DEPENDENCY`, `PLATFORM`, `UNRESOLVED_TARGET`.

These scopes are not architecture classifications.

## 13. VPMS boundary

VPMS (Verification Purpose Management System) and PTSIP solve different questions.

PTSIP asks:

```text
Who primarily owns this project responsibility across its lifecycle?
```

VPMS asks:

```text
Why does this Verification Case exist / what does it protect?
```

A Verification Case is a purpose-bound execution unit composed from Formula, Variables, Policy, Target, and Runner concerns.

Reusable Formula logic MUST NOT erase Verification Purpose. Product-owned tests and Development-Tooling-owned verification infrastructure are both valid PTSIP patterns.

PTSIP core MUST NOT acquire a runtime dependency on VPMS. VPMS MAY consume stable PTSIP metadata through a narrow read-only boundary.

VPMS PASS does not imply PTSIP `CONFORMANT`. PTSIP `CONFORMANT` does not imply functional verification PASS.

Tool `0.3.5` VPMS purpose vocabulary currently uses `PRODUCT | TOOLCHAIN`. That VPMS vocabulary is a separate axis and MUST NOT be renamed merely as an accidental side effect of removing `TOOLCHAIN` from PTSIP lifecycle classification. Any VPMS purpose-vocabulary change requires its own explicit design decision.

## 14. Validation, authority reconciliation, migration, and conformance

These are distinct operations:

**Profile Validation** — checks Project Profile structure and semantics.

**Candidate/Migration Analysis** — collects evidence and proposes lifecycle/role/relationship/boundary changes without becoming architecture authority.

**Authority Reconciliation** — when distributed coordination is selected, checks whether relevant local declaration is consistent with the current coordinated winner and applies safe reconciliation rules.

**Conformance Evaluation** — combines validated declaration with observed dependency, artifact, build, lifecycle, snapshot, and coverage evidence against Consumer Repository PTSIP rules.

A valid profile does not prove conformance. A migration proposal does not become architecture fact until confirmed. A resolved authority winner does not prove conformance.

Completed Consumer Repository conformance outcomes are only:

- `CONFORMANT`;
- `NON_CONFORMANT`;
- `INCOMPLETE`.

`NOT_EVALUATED` MAY be used as Tool execution state but is not a conformance outcome.

## 15. Reuse policy

Acceptable strategies include shared declarative contracts, generation of separate lifecycle-owned implementations, independent implementations validated against common conformance vectors, and separately governed third-party/platform dependencies.

Risky strategies include project-local executable `shared` packages imported as implementation across incompatible lifecycle ownership, Development Tooling imported into Product runtime, Delivery/Operations implementation included in Product packages merely for repository convenience, and one accidental dependency environment that makes unrelated lifecycle responsibilities inseparable.

## 16. Repository topology

PTSIP does not require a monorepo or multirepo and does not prescribe `docs/`, `tools/`, `tests/`, `deploy/`, `ops/`, `.ptsip/`, `product/`, or another directory hierarchy.

Directory names are evidence hints only. Primary lifecycle ownership and explicit project architecture determine classification.

## 17. Non-goals

PTSIP does not prescribe programming paradigm, build system, package manager, test framework, CI provider, service topology, repository count, documentation hierarchy, continuous authority polling, GitHub as a universal dependency, or automatic LLM architecture classification.

PTSIP does not force every test into one classification. It does not classify FastAPI, Cloudflare Workers, GitHub Actions, Docker, Terraform, Markdown, YAML, or another technology universally.

PTSIP does not preserve legacy `TOOLCHAIN` as a canonical Tool `0.3.6` alias merely to avoid migration.

PTSIP does not create a new lifecycle classification for every responsibility role.

## 18. Status and novelty statement

PTSIP is a project-defined draft architecture policy. It does not claim invention of host/target separation, build-time/runtime separation, toolchain isolation, delivery/operations separation, dependency isolation, or independent lifecycle management. Its contribution is a reproducible lifecycle-ownership governance model combining explicit classification, responsibility mapping, conformance, evidence, assisted migration, verification-purpose separation, and distributed architecture-decision consistency semantics.