# PTSIP Reference Architecture

This document is **informative** unless a project profile adopts a recommendation as a required local rule. PTSIP does not require a Consumer Repository to use any directory shown here.

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

## 3. Dependency evidence model

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

## 4. Evidence node scope is not classification

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

## 5. Separate build contexts

A strong implementation keeps at least dependency resolution separate. One possible Python arrangement is:

```text
product build context
  dependency manifest / lock / environment

toolchain build context
  dependency manifest / lock / environment
```

Equivalent structures are valid for npm, Gradle, Cargo, Go, Bazel, containers, or other systems.

The objective is not directory aesthetics. The objective is that Toolchain-only dependencies cannot appear in Product builds merely because both happen to be installed in one developer environment.

## 6. Shared semantics without shared executable ownership

When both planes must understand the same object model:

```text
contract/schema
      |             |
      v             v
Product adapter   Toolchain validator
```

This preserves one semantic source while allowing independent implementations and lifecycles.

The number of currently observed consumers does not determine neutrality by itself. A contract can remain neutral with one observed consumer at a particular revision if it remains non-executable/non-owning and is not merely Product or Toolchain implementation disguised as a contract.

## 7. Anti-pattern: shared executable common package

```text
             shared/common
             /           \
            v             v
      Product SDK     Toolchain SDK
```

This is not automatically forbidden in every possible case, but it is architecture-sensitive. It frequently creates release coupling and therefore requires explicit classification and ownership rather than being accepted merely as DRY refactoring.

## 8. Product Artifact model

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

## 9. Recommended ownership test

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

## 10. Component profile model

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

## 11. Optional project-specific component dependency policy

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

## 12. Evidence, diagnostics, and conformance architecture

The recommended architecture is:

```text
Consumer Repository
        |
        v
Deterministic Collector
  snapshot / inventory / manifests / dependency edges / artifact evidence
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

The collector establishes evidence. The optional coding agent makes bounded ownership judgments supported by evidence IDs. External evidence is input, not automatic truth. The deterministic evaluator applies PTSIP rules. The agent or external validator is not the source of PTSIP normative rules and does not approve strict conformance.

The stable diagnostic reference format is `ptsip-diagnostic/v1`.

## 13. Profile Validation versus Conformance Evaluation

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

A repository may have a perfectly valid profile and still be non-conformant.

A repository may also have no profile and still be inspected for Core evidence, while Enforced Conformance requires a machine-readable declaration and immutable specification binding.

## 14. Conformance outcomes and evidence coverage

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

## 15. Stable diagnostics

A diagnostic instance and a normative rule are different identities:

```text
rule_id: PTSIP-DEP-001

diagnostic_id: finding-0001
diagnostic_id: finding-0002
```

One rule can produce many findings. A stable diagnostic should also preserve outcome effect, severity, evidence IDs, source/target component when applicable, message, and evaluator/provenance information.

The reference contract is `schemas/ptsip-diagnostic.schema.json`.

## 16. Lifecycle triggers versus release coupling

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

## 17. Snapshot integrity

A Pilot should record repository state before and after evidence collection. If HEAD or observed tracked state changes during the scan, the evidence set should be invalidated and rerun.

This distinction matters when another process performs a pull, checkout, generation step, or other repository mutation concurrently with the PTSIP inspection. A changed snapshot proves that the evidence set is unstable; it does not by itself prove which process caused the change.

## 18. Exceptions and transitions

An approved PTSIP exception documents a real normative deviation; it does not transform the deviation into strict conformance. While the active PTSIP `MUST`/`MUST NOT` violation remains, strict Core/Enforced conformance is blocked.

Migration/transition metadata is a planning concept. It may describe current and target architecture, but:

```text
Transition != Exception != Conformance
```

A target architecture does not rewrite the current observed result.
