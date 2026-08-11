# PTSIP Reference Architecture

This document is **informative** unless a project profile adopts a recommendation as a required local rule. PTSIP does not require a Consumer Repository to use any directory shown here.

> **0.3.4-draft alignment note:** the distributed-authority architecture below reflects the published `spec-v0.3.4-draft` design and the verified Reference Tool `0.3.4` implementation. It does not by itself activate a new normative Specification binding.

## 1. Illustrative project topology

```text
repository/
│
├─ product/
│  ├─ app/
│  └─ sdk/
│
├─ toolchain/
│  ├─ sdk/
│  └─ automation/
│
└─ contracts/
```

A real project MAY use completely different names and locations. In particular, PTSIP does not require `product/`, `toolchain/`, `contracts/`, `docs/`, `tools/`, `.ptsip/`, or a root-level profile file.

A mixed repository may legitimately contain nested ownership:

```text
src/
├─ app/                         PRODUCT
├─ plugin/contracts/v1/         NEUTRAL_CONTRACT
├─ install/plugin_build.py      TOOLCHAIN
└─ test/                        TOOLCHAIN or unresolved until ownership is established
```

This is why component ownership, not directory naming, is the useful classification unit.

If one broad root contains materially different shipped state, executable purpose, release owner, compatibility owner, or Product/Toolchain lifecycle responsibility, the ownership model should split it into coherent components rather than hide the difference behind one root classification.

## 2. External PTSIP Tooling topology

The preferred Pilot/inspection model keeps PTSIP implementation and state outside the Consumer Repository:

```text
External development environment
│
├─ PTSIP tooling installation
│  └─ ptsip CLI / validator / pilot engine
│
└─ PTSIP local state
   ├─ cache/
   └─ pilots/

             read-only by default
                     |
                     v
              Consumer Repository
              └─ existing project topology unchanged
```

For a Python implementation, the tool MAY be installed in an isolated virtual environment, user-level tool environment, or other package-managed environment. The physical `site-packages` location is an implementation detail, not a PTSIP repository requirement.

## 3. Project declaration and Decision Authority topology

The `0.3.4-draft` design distinguishes durable architecture declaration from distributed decision coordination:

```text
                    PTSIP Specification
               architecture/conformance contract
                           |
                           v
                    Project Profile
               project-owned declaration
                           ^
                           |
                 safe local projection
                           |
                     Decision Authority
                pending/resolved winner state
```

The three roles are intentionally different:

- the **Specification** defines architecture and conformance semantics;
- the **Project Profile** records the project-owned architecture declaration for a repository/worktree;
- the **Decision Authority** coordinates which explicit architecture answer won when multiple environments can act on the same unresolved scope.

A Decision Authority is not a second Project Profile and is not a conformance database.

For local-only work, an embedded local DecisionStore may be sufficient. It is Tool-owned operational state and should stay outside the Consumer Repository.

For multi-environment coordination, separate clone-local SQLite files are insufficient because each clone could independently accept a different winner. Reference Tool `0.3.4` demonstrates repository-scoped coordination through a dedicated remote Git ref:

```text
refs/heads/ptsip-policy

  authority.json
  decisions/<global-decision-id>.json
```

That GitHub representation is informative/reference-specific. A different distributed backend may satisfy the same consistency model without using Git refs.

## 4. Distributed authority operation model

The recommended coordinated gate flow is:

```text
active architecture-sensitive task
        |
        v
identify normalized component scope
        |
        v
inspect selected local Project Profile
        |
        v
read current relevant Decision Authority state
        |
        v
compare local declaration with authority
        |
        +-- no authority + local complete
        |       -> use project declaration; do not fabricate history
        |
        +-- remote winner + local missing
        |       -> validate + safe local projection
        |
        +-- remote winner + local equivalent
        |       -> consistent/resolved; no rewrite required
        |
        +-- remote winner + local conflicting
        |       -> explicit authority/profile conflict; no overwrite
        |
        +-- unresolved decision
        |       -> request/reuse pending decision
        |
        +-- required authority unavailable
                -> fail closed; no Local fallback winner
```

This is **action-time synchronization**. Continuous polling is not required.

A complete local profile does not allow a distributed coordination-sensitive gate to assume the clone is current. The authority freshness check is what prevents a stale but syntactically complete clone from creating a contradictory second winner.

Read-only authority observation should not bootstrap remote state. In the Reference Tool, absence of `refs/heads/ptsip-policy` can be observed without creating the ref merely to prove that no prior decision exists.

## 5. First-winner serialization

Distributed authority writes should use an ordered authority revision and atomic conditional mutation:

```text
read authority revision A
        |
        v
prepare valid mutation B based on A
        |
        v
conditional publish A -> B

if A is no longer current
        |
        v
reject stale mutation
        |
        v
reread authority
        |
        +-- same scope already resolved
        |       -> accept existing winner
        |
        +-- unrelated mutation only
                -> retry/rebase when safe
```

Reference Tool `0.3.4` uses exact-parent Git commits and a non-force ref update as its compare-and-swap primitive. Wall-clock completion time is not the authority ordering mechanism.

The global decision identity should be derived from stable coordination-domain identity plus normalized architecture scope. Clone-local clarification IDs, local database IDs, or temporary missing-field state should not create separate distributed decisions for the same component boundary.

## 6. Global decision state versus local projection state

Do not use one global application flag to imply that every clone is synchronized.

```text
Decision Authority
  decision_id: <stable distributed identity>
  status: PENDING | RESOLVED
  answer: <winner when resolved>

Clone/worktree A
  projection: equivalent | locally-applied

Clone/worktree B
  projection: not-yet-applied

Clone/worktree C
  projection: stale | conflict | failed
```

A global `RESOLVED` state means one architecture answer won. It does not mean every clone has updated `ptsip.yaml`.

Reference Tool `0.3.4` therefore reports clone-local application/reconciliation separately, including a local projection scope. Such a receipt cannot change the global winner.

## 7. Dependency evidence model

```text
                    contracts
                   /         \
                  v           v
          product/sdk       toolchain/sdk
               |                 |
               v                 v
           product/app     validators/build tools

Forbidden default edge:
product/sdk  -X->  toolchain/sdk
```

External PTSIP Tooling may inspect either plane without becoming a project-owned dependency simply by being installed in a developer environment.

Dependency evidence is stronger when relationship type, phase, scope, and provenance are preserved:

```text
product-a --IMPORTS/RUNTIME/OBSERVED----> product-b
builder   --INVOKES/BUILD/OBSERVED------> generator
validator --READS/INSPECTION/OBSERVED---> product source
manifest  --DECLARES equivalent----------> external dependency
```

PTSIP's canonical edge vocabulary includes:

```text
IMPORTS
LINKS
LOADS
INVOKES
READS
GENERATES
PACKAGES
TESTS
PUBLISHES
```

Canonical lifecycle phases include:

```text
RUNTIME
BUILD
TEST
RELEASE
INSPECTION
```

Canonical provenance is:

```text
DECLARED
OBSERVED
INFERRED
```

A relationship may be multi-phase or supported by more than one relationship type. Unknown phase/target remains unresolved. The validator should not manufacture a runtime/build meaning merely to increase automation coverage.

## 8. Evidence node scope is not classification

A dependency target may be outside project-owned PTSIP scope:

```text
project Product component
        |
        +--IMPORTS--> project Toolchain component   PROJECT_COMPONENT
        +--IMPORTS--> third-party package           EXTERNAL_DEPENDENCY
        +--CALLS-----> operating-system API          PLATFORM
        +--LOADS-----> unresolved dynamic target     UNRESOLVED_TARGET
```

Only project-owned PTSIP Components receive one of the three architectural classifications. `EXTERNAL_DEPENDENCY`, `PLATFORM`, and `UNRESOLVED_TARGET` are evidence-node scope/type, not extra planes.

## 9. Separate build contexts

A strong implementation keeps at least dependency resolution separate. One possible Python arrangement is:

```text
product build context
  dependency manifest / lock / environment

toolchain build context
  dependency manifest / lock / environment
```

Equivalent structures are valid for npm, Gradle, Cargo, Go, Bazel, containers, or other systems.

The objective is not directory aesthetics. The objective is that Toolchain-only dependencies cannot appear in Product builds merely because both happen to be installed in one developer environment.

## 10. Shared semantics without shared executable ownership

When both planes must understand the same object model:

```text
contract/schema
      |             |
      v             v
Product adapter   Toolchain validator
```

This preserves one semantic source while allowing independent implementations and lifecycles.

The number of currently observed consumers does not determine neutrality by itself. A contract can remain neutral with one observed consumer at a particular revision if it remains non-executable/non-owning and is not merely Product or Toolchain implementation disguised as a contract.

## 11. Anti-pattern: shared executable common package

```text
             shared/common
             /           \
            v             v
      Product SDK     Toolchain SDK
```

This is not automatically forbidden in every possible case, but it is architecture-sensitive. It frequently creates release coupling and therefore requires explicit classification and ownership rather than being accepted merely as DRY refactoring.

## 12. Product Artifact model

Artifact owner and artifact producer are separate:

```text
Toolchain builder
     TOOLCHAIN
         |
         | GENERATES / PACKAGES
         v
Product installer / bundle / plugin archive
     PRODUCT ARTIFACT
```

This is valid when the produced artifact itself satisfies Product packaging rules.

Useful artifact evidence includes:

```text
artifact_id
classification / owner
producer_component
artifact_type
shipping_scope
contained paths/components
derivation relationships
provenance
```

A Toolchain producer is not automatically Product merely because it creates a Product Artifact. Conversely, a Product Artifact is not clean merely because its producer is Toolchain; its contents still have to exclude Toolchain-owned implementation and Toolchain-only dependencies under `PTSIP-PKG-001`.

The reference interoperability shape is `schemas/ptsip-artifact-evidence.schema.json` (`ptsip-artifact-evidence/v1`).

## 13. Recommended ownership test

Before classifying a component, ask in order:

1. Why does this component exist?
2. Who consumes it?
3. Is it shipped with the Product?
4. Which plane owns compatibility?
5. Which plane should trigger version/release decisions when it changes?
6. Is it executable or a declarative contract?
7. Can one classification/purpose/lifecycle statement coherently describe the whole component?
8. Can the semantic contract be shared without sharing executable code?
9. Would sharing create a cross-plane dependency edge?
10. Is the observed dependency target project-owned, external/platform, or unresolved?

The answer to question 1 has priority over superficial code similarity.

For an already coordinated scope, classification review and authority review are separate questions. After explicit architecture intent is established, a distributed operation should additionally ask whether a Decision Authority already contains a winner for the same normalized scope before creating a new decision.

## 14. Component profile model

The reference profile has two alternative ownership modes.

### Uniform-root shorthand

Use `boundaries` when roots have uniform ownership.

### Component declarations

Mixed repositories should prefer explicit components:

```yaml
components:
  - id: product-runtime
    classification: PRODUCT
    include: ["src/**"]
    purpose: product_runtime

  - id: plugin-builder
    classification: TOOLCHAIN
    include: ["src/install/plugin_build.py"]
    purpose: build_and_release
```

The exact file is more specific than the broad `src/**` selector, so it owns that file. If two different components have equal-specificity selectors for one path, the profile is ambiguous and validation should fail rather than choose arbitrarily.

A reference profile uses either `boundaries` or `components`, not both.

The proposed `0.3.4-draft` adoption work also identifies structured architecture facts such as shipped state, runtime requirement, lifecycle owner, and executable state. Reference Tool workflow state may carry facts not yet represented by the active `0.2.0-draft` Project Profile schema. A future coherent schema migration must preserve any normatively required facts losslessly rather than silently treating missing values as false.

## 15. Optional project-specific component dependency policy

A project may be stricter than the universal PTSIP minimum.

For example, a Toolchain SDK suite can declare default-deny same-plane dependencies with explicit allows:

```yaml
component_dependency_policy:
  default: deny
  allow:
    - from: migration-sdk
      to: validation-sdk
    - from: build-sdk
      to: schema-sdk
    - from: test-sdk
      to: validation-sdk
```

This does not make all Toolchain-to-Toolchain dependencies universally forbidden by PTSIP. It is repository-specific architecture policy represented through the PTSIP profile.

A project-specific allow never overrides a universal PTSIP prohibition.

## 16. Evidence, decision, diagnostics, and conformance architecture

The recommended architecture keeps decision coordination out of the conformance-rule engine:

```text
Consumer Repository
        |
        v
Deterministic Collector
  snapshot / inventory / manifests / dependency edges / artifact evidence
        |
        +---------------------------+
        |                           |
        v                           v
Candidate/clarification scope   Existing Project Profile
        |                           |
        v                           |
Explicit project-owner facts        |
        |                           |
        v                           |
Decision Authority (when needed)    |
  first-winner coordination         |
        |                           |
        +------ safe projection ----+
                    |
                    v
              Project Profile
                    |
                    v
Coverage Evaluator
  blocking vs non-blocking gaps per applicable rule
        |
        +------------------------------+
        |                              |
        v                              v
Constrained Classification        External evidence (optional)
Agent (optional)                  provenance-bound input
        |                              |
        +---------------+--------------+
                        v
Deterministic Rule Evaluator
  declaration <-> observed evidence
  universal PTSIP rules
  optional project component policy
                        |
                        v
Stable Diagnostics
  diagnostic instance ID + rule ID + evidence IDs
                        |
                        v
Conformance Outcome
  CONFORMANT | NON_CONFORMANT | INCOMPLETE
```

The collector establishes evidence. Candidate discovery does not assign architecture ownership. Explicit project-owner facts resolve missing intent. A Decision Authority coordinates one winner when distributed coordination is needed. The resulting Project Profile remains the durable declaration. External evidence is input, not automatic truth. The deterministic evaluator applies PTSIP rules.

Neither the coding agent nor the Decision Authority is the source of PTSIP normative rules, and a resolved decision does not approve strict conformance.

The stable diagnostic reference format is `ptsip-diagnostic/v1`.

## 17. Profile Validation versus Conformance Evaluation

These are distinct operations:

```text
Profile Validation
  schema / IDs / selectors / references / policy consistency
                |
                X  does not imply
                |
Conformance Evaluation
  declaration + observed evidence + artifacts + coverage + rules
```

Under the proposed distributed model there is another independent distinction:

```text
Authority reconciliation
  local declaration <-> distributed winner
                |
                X  does not imply
                |
Conformance Evaluation
```

A repository may have a perfectly valid profile and still be non-conformant.

A repository may also have no profile and still be inspected for Core evidence, while Enforced Conformance requires a machine-readable declaration and immutable specification binding.

## 18. Conformance outcomes and evidence coverage

A completed evaluation uses:

```text
CONFORMANT
NON_CONFORMANT
INCOMPLETE
```

A useful decision rule is:

```text
Definite mandatory violation?
    yes -> NON_CONFORMANT
    no
     |
     v
Blocking evidence gap for an applicable mandatory rule?
    yes -> INCOMPLETE
    no  -> eligible for CONFORMANT if all other requirements are satisfied
```

This avoids the incorrect rule:

```text
no findings -> conformant
```

A parser/adapter gap that cannot affect any applicable mandatory rule may be non-blocking. A missing runtime dependency target or missing Product Artifact content check that can hide a prohibited boundary is blocking.

An authority/profile conflict is not automatically a Product/Toolchain rule violation; it is an architecture-declaration consistency problem that must be reconciled before a future distributed-authority-bound Enforced Conformance claim can rely on that scope unambiguously.

## 19. Stable diagnostics

A diagnostic instance and a normative rule are different identities:

```text
rule_id: PTSIP-DEP-001

diagnostic_id: finding-0001
diagnostic_id: finding-0002
```

One rule can produce many findings. A stable diagnostic should also preserve outcome effect, severity, evidence IDs, source/target component when applicable, message, and evaluator/provenance information.

The reference contract is `schemas/ptsip-diagnostic.schema.json`.

## 20. Lifecycle triggers versus release coupling

A Toolchain-only change causing a Product workflow to start is not automatically a lifecycle violation.

Distinguish:

```text
workflow trigger
Product artifact changed?
Product version/release decision changed?
Product publication/deployment happened?
Product compatibility obligation changed?
```

`PTSIP-LCY-001` is concerned with whether Toolchain-only change forces Product lifecycle obligation, not merely whether a shared monorepo workflow executed.

## 21. Snapshot integrity

A Pilot should record repository state before and after evidence collection. If HEAD or observed tracked state changes during the scan, the evidence set should be invalidated and rerun.

This distinction matters when another process performs a pull, checkout, generation step, or other repository mutation concurrently with the PTSIP inspection. A changed snapshot proves that the evidence set is unstable; it does not by itself prove which process caused the change.

The same principle applies to local profile projection after a distributed decision: if the repository/profile changed after analysis but before application, the projection should fail stale rather than overwrite newer local state.

## 22. Exceptions and transitions

A project governance approval may document a real normative deviation; it does not transform the deviation into strict conformance. While an active PTSIP `MUST`/`MUST NOT` violation remains, strict Core/Enforced conformance is blocked.

Migration/transition metadata is a planning concept. It may describe current and target architecture, but:

```text
Transition != Governance approval != Conformance
```

A target architecture does not rewrite the current observed result.
