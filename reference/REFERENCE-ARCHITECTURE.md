# PTSIP Reference Architecture

This document is **informative**. Normative requirements are in `spec/PTSIP-SPEC.md` under the bound immutable `0.3.4-draft` revision.

## 1. Product / Toolchain / Contract topology

Illustrative only:

```text
repository/
├─ product/
├─ toolchain/
└─ contracts/
```

PTSIP does not require these names. Mixed/nested ownership is valid when component declarations express the real boundary.

```text
src/
├─ app/                         PRODUCT
├─ plugin/contracts/v1/         NEUTRAL_CONTRACT
├─ install/plugin_build.py      TOOLCHAIN
└─ test/                        TOOLCHAIN or unresolved until decided
```

Directory naming is evidence, not architecture authority.

## 2. External PTSIP Tooling

Preferred operational topology:

```text
External development environment
│
├─ PTSIP Tool installation
├─ cache / pilot / local DecisionStore
│
└──────────── read / explicit authorized write ────────────>
                         Consumer Repository
                         └─ ptsip.yaml (project-owned declaration)
```

Tool-owned local SQLite remains outside the Consumer Repository and is not Git-shared.

## 3. Distributed decision topology

When repository-global coordination is selected, keep three concepts separate:

```text
PTSIP Specification
    -> normative rules and semantics

Decision Authority
    -> which explicit architecture answer won

Consumer Repository Project Profile
    -> durable architecture declaration for this repository revision/worktree
```

Observed repository/artifact evidence is a fourth **evidence source**, not another authority plane:

```text
Observed evidence
    -> what the repository actually does
```

Conformance Evaluation combines declaration + observed evidence against the bound Specification. A Decision Authority does not prove conformance.

## 4. Reference Tool GitHub authority profile

Reference Tool `0.3.4` demonstrates repository-distributed authority using:

```text
refs/heads/ptsip-policy

  authority.json
  decisions/<global-decision-id>.json
```

This GitHub storage shape is an implementation profile, not a universal PTSIP requirement.

The important architecture properties are:

- stable coordination-domain + normalized component-scope identity;
- ordered authority revision;
- non-force compare-and-swap style mutation;
- first-valid-resolution-wins;
- read-side authority freshness;
- deterministic reconciliation;
- fail-closed behavior;
- global/local state separation.

Another backend may conform with a database transaction, ETag/generation, consensus log, or equivalent mechanism.

## 5. Authority freshness

Write serialization is not enough. Before a distributed coordination-sensitive result relies on local declaration state, the implementation must account for relevant current authority state.

```text
analyze local repository/profile
        |
        v
resolve coordination domain + scope
        |
        v
read relevant authority state
        |
        v
compare authority vs local declaration
        |
        v
return/reconcile/fail explicitly
```

A complete local profile can still be stale.

## 6. Read-only absence observation

A check for existing authority should not create history merely to prove history is absent.

Reference Tool GitHub coordination therefore supports a read-only lookup that does not create `refs/heads/ptsip-policy` when the ref does not exist.

```text
peek authority
    |
    +-- no ref / no decision -> absence result, no mutation
    |
    +-- pending/resolved     -> return current authority state
```

## 7. Reconciliation matrix

```text
Local declaration absent + no authority
    -> pending only if the active operation actually needs a decision

Local declaration absent + resolved authority
    -> validate and safely project winner locally

Local declaration present + no authority
    -> use project declaration; do not fabricate history

Local declaration equivalent + resolved authority
    -> consistent/resolved; no formatting rewrite required

Local declaration conflicting + resolved authority
    -> explicit authority/profile conflict; no silent overwrite

Repository/profile changes during reconciliation
    -> stale; refuse prepared application
```

Semantic equivalence is architecture meaning, not YAML key order or whitespace.

## 8. Global versus local state

```text
GLOBAL AUTHORITY
    PENDING
    RESOLVED

LOCAL WORKTREE
    declaration missing
    declaration consistent
    locally projected
    stale
    failed
```

A globally resolved winner does not mean every clone is already synchronized.

A clone-local application receipt cannot alter the global winner.

## 9. Fail-closed coordination

When distributed coordination is selected, these failures must not silently create an isolated Local winner:

- authentication/permission failure;
- network unavailability;
- malformed authority state;
- incompatible authority ownership/manifest state;
- conditional-write failure that cannot be reconciled safely;
- inability to establish required read freshness.

Fail closed at the affected architecture-sensitive operation.

## 10. Action-time synchronization

Continuous polling is not required.

```text
coding-agent task
      |
      v
architecture-sensitive boundary reached?
      |
      +-- no  -> continue unrelated work
      |
      +-- yes -> gate / authority freshness / reconciliation
```

This keeps ordinary work independent from background polling while preventing stale clones from creating contradictory second winners.

## 11. Durable component declaration

For structured adoption/resolution, the component model can preserve:

```yaml
components:
  - id: plugin-builder
    classification: TOOLCHAIN
    include: ["src/install/plugin_build.py"]
    purpose: build_and_release
    shipped: false
    runtime_required: false
    lifecycle_owner: DEVELOPMENT_TOOLING
    executable: true
```

Canonical lifecycle ownership is distinct from project-specific release/compatibility metadata.

Boundary-root shorthand remains useful for simple ownership maps, but it cannot represent the complete structured fact set. A write workflow should require component declarations rather than dropping facts.

## 12. Dependency evidence model

```text
contracts
 /     \
v       v
Product   Toolchain

Product -X-> Toolchain runtime/shipped dependency
```

Canonical relationship vocabulary:

```text
IMPORTS LINKS LOADS INVOKES READS GENERATES PACKAGES TESTS PUBLISHES
```

Canonical lifecycle phases:

```text
RUNTIME BUILD TEST RELEASE INSPECTION
```

Canonical provenance:

```text
DECLARED OBSERVED INFERRED
```

Unknown target/phase stays unresolved rather than guessed.

## 13. Separate build contexts

A strong implementation maintains independently resolvable Product and Toolchain dependency contexts.

```text
Product build context
  -> Product dependencies

Toolchain build context
  -> Toolchain dependencies + explicitly declared Product analysis/build inputs
```

The goal is lifecycle/dependency isolation, not directory aesthetics.

## 14. Shared semantics without shared executable ownership

```text
contract/schema
   /      \
  v        v
Product   Toolchain
adapter   validator
```

Shared semantic source is compatible with independently owned executable implementations.

## 15. Product Artifact model

```text
Toolchain builder
      |
      | GENERATES / PACKAGES
      v
Product Artifact
```

Producer ownership does not decide artifact ownership. Inspect resulting artifact contents for Toolchain implementation/dependencies.

## 16. Validation / authority / conformance pipeline

```text
Project Profile ----------+
                          |
Decision Authority -------+--> declaration consistency / coordination
                          |
Observed evidence --------+--> deterministic PTSIP rule evaluator
                          |
Coverage/snapshot --------+
                          |
                          v
       CONFORMANT | NON_CONFORMANT | INCOMPLETE
```

The Decision Authority branch contributes architecture-decision consistency, not observed compliance truth.

## 17. Recommended pre-change questions

Before a boundary-affecting change ask:

1. Why does the component exist?
2. Which lifecycle owns it?
3. Is it shipped with Product?
4. Is Product runtime dependent on it?
5. Is it executable or a Neutral Contract?
6. Is the component boundary coherent?
7. What dependency relationship and lifecycle phase are introduced?
8. Does distributed coordination require an authority freshness check now?
9. Would local/remote declarations be missing, equivalent, or conflicting?
10. Can shared semantics be expressed without shared executable ownership?

Purpose remains the first question.
