# Pre-WU-00A — 0.4.0 Release Boundary Classification

> **Status:** APPROVED / RELEASE BOUNDARY CLASSIFICATION  
> **Target Tool version:** `0.4.0`  
> **Integration branch:** `dev/0.4.0`  
> **Entry baseline:** `7a33e57b18a67a28eb30d7adc5bfc8a9b813a3f3`  
> **Parent planning source:** `planning/0.4.0.md`  
> **Classification approval:** project-owner approval, 2026-08-31  
> **Implementation authorization:** not granted by this document; implementation begins only after the subsequent core-contract freeze and approved WU plan

## 1. Purpose

This Pre-WU fixes the release-dependency classification procedure before 0.4.0 implementation is decomposed into WUs.

The governing rule is:

```text
responsibility / capability idea
        ↓
release-dependency classification
        ↓
CORE / OPTIONAL / DEFERRED
        ↓
only then WU decomposition
```

The rejected direction is:

```text
create WU / start implementation
        ↓
allow scope to become release-blocking by accident
```

A planning file, WU number, branch, experiment, implementation already written, or general usefulness is never sufficient by itself to make work release-blocking.

## 2. Classification states

Every new 0.4.0 planning item begins as:

```text
UNCLASSIFIED
```

`UNCLASSIFIED` is not equivalent to core and does not authorize implementation.

After release-dependency review, the item receives exactly one planning class:

### `CORE / RELEASE-BLOCKING`

The item is required to satisfy the approved 0.4.0 product objective, preserve a required architecture/safety invariant, or prove release acceptance. 0.4.0 cannot be declared complete while an approved core item is incomplete.

### `OPTIONAL / NON-BLOCKING`

The item may improve usability, explanation, performance, convenience, coverage, or experimentation, but the approved 0.4.0 core remains complete and truthful without it. Optional work may be implemented during the cycle, isolated on an optional branch, merged by explicit decision, deferred, or abandoned without blocking 0.4.0.

### `DEFERRED / OUT-OF-RELEASE`

The item is intentionally outside the 0.4.0 release implementation boundary. It may remain as future planning context, but it should not consume core WU scope or become a hidden dependency.

## 3. Classification gates

A planning item is reviewed through four gates.

### Gate A — Product Objective Necessity

Question:

> If this item is absent, can Tool 0.4.0 still truthfully deliver the approved authority-aware generic remediation objective?

If no, the item is a core candidate.

### Gate B — Architectural Necessity

Question:

> If this responsibility is deferred, will the core be forced onto a wrong authority, identity, lifecycle, ownership, or safety boundary that would later require structural correction?

If yes, the boundary responsibility is a core candidate even if a richer implementation can be deferred.

### Gate C — Release Proof Necessity

Question:

> If this item is absent, which required 0.4.0 release proof becomes impossible?

A core classification must identify the broken proof or invariant. "Useful", "cleaner", "future-proof", or "already discussed" is not enough.

### Gate D — Deferral Safety

Question:

> Can this item move to 0.4.1 or later without weakening 0.4.0 correctness, authority preservation, safety, or release claims?

If yes, it should normally be `OPTIONAL` or `DEFERRED`, not `CORE`.

## 4. Approved 0.4.0 core boundary

The following responsibilities are `CORE / RELEASE-BLOCKING`.

| ID | Core responsibility | Why release-blocking | Proof that breaks if removed |
| --- | --- | --- | --- |
| C01 | Typed remediation authority domain | Evidence, facts, normative constraints, project intent, semantic decision, and mutation authorization must not collapse into one authority concept. | Authority Firewall cannot be structurally verified. |
| C02 | Fresh Solve and current-state binding | Every solve must derive from current Specification + current authority + current repository state. | Stale history or old state could influence current semantics. |
| C03 | Operational Rule contract / executable Specification bridge | Supported normative rules need an explicit executable bridge without turning `ptsip.yaml` into Tool-internal policy. | Generic remediation cannot prove which supported rule produced evaluation/remediation behavior. |
| C04 | Declared modeled remediation-family capability | Determinacy must be scoped to what the Tool actually models. | One survivor could be falsely reported as globally unique instead of `TOOL_CAPABILITY_GAP`. |
| C05 | Solution Space Engine | Candidate generation, proof-based elimination, semantic reduction, and survivor cardinality are the mechanism that reduces owner questions. | Deterministic / owner-intent / unsatisfiable outcomes cannot be proven. |
| C06 | Elimination/ranking separation | Confidence, AI preference, cost, or blast radius must not silently become architecture authority. | Legal semantic candidates could be deleted without normative/authority proof. |
| C07 | Semantic target / physical mutation separation | `SemanticRemediationPlan` must be distinct from `RepositoryChangePlan`. | Edit-order alternatives could be presented as architecture choices or physical planning could re-decide intent. |
| C08 | Canonical resolution outcome semantics | The core must distinguish deterministic, owner-intent-required, external-fact-required, unsatisfiable, and capability-gap states. | Unsupported capability, missing facts, missing intent, and impossibility could collapse into generic uncertainty. |
| C09 | Escalation Proof and owner-question minimization | Owner input is allowed only after deterministic reduction identifies the exact missing semantic authority. | PTSIP would still require the user/agent to reconstruct ordinary Specification reasoning. |
| C10 | Owner intent / external fact separation and resumable Fresh Solve | Missing technical evidence must not be materialized as project intent; accepted owner intent must enter through an explicit authority path. | External evidence and architecture authority could become interchangeable. |
| C11 | Advisory non-authority invariant | The core must remain correct with no AI/advisor and must prohibit advisory output from becoming authority or candidate-elimination proof. | AI/provider state could become a hidden correctness or authority dependency. |
| C12 | Authority Gate before repository mutation | Semantic determinacy is not mutation authorization. | A unique semantic target could be applied without valid mutation permission. |
| C13 | Mutation impact/reversibility classification sufficient for safe authorization | Mechanical/reversible, structural semantic-preserving, architecture-semantic, and destructive effects must be distinguishable enough to enforce authorization strength. | Authorization cannot safely vary with semantic impact and irreversibility. |
| C14 | Repository Change Plan | Physical mutations, ordering, preconditions, expected postconditions, and recovery information must realize an already-decided semantic target without inventing intent. | Safe apply has no inspectable execution contract. |
| C15 | Exact pre-state binding and stale-state rejection | A plan must be invalidated when current repository state differs from the state used to solve it. | A semantic decision could be silently rebased onto different evidence. |
| C16 | Fail-closed safe apply / recovery inheritance | 0.4.0 must reuse or preserve the 0.3.7 exact-state/recovery safety semantics rather than create a weaker generic path. | Partial or ambiguous mutation states could be promoted. |
| C17 | Postcondition and conformance verification | File mutation alone is not successful remediation; expected semantics and relevant conformance must be re-proven. | The Tool could report success after producing an invalid architecture. |
| C18 | Representative end-to-end rule coverage | The framework must be proven by representative complete slices, not only by abstract types. | 0.4.0 would claim a remediation framework without an executable vertical proof. |
| C19 | Required outcome coverage | Release verification must prove deterministic, owner-intent-required, unsatisfiable, and capability-gap behavior; external-fact-required must be proven at least at the domain/diagnostic boundary. | Outcome taxonomy would exist without behavioral evidence. |
| C20 | Verification responsibility and selective Test Mode integration for durable remediation responsibility | New durable remediation verification ownership must follow `ptsip.yaml → component_ref → Test Mode → pytest target` when selective execution is useful. | 0.4.0 would regress to physical-test-path authority or unowned verification. |
| C21 | Full exact-SHA release qualification | Selective Test Mode success must remain distinct from full repository/release qualification. | Selective success could weaken the meaning of release readiness. |
| C22 | 0.3.7 contract inheritance / no parallel identity-evidence-authority stack | Tool Version, Project Profile identities, Specification binding, normalized evidence, source compatibility, snapshot, and safe-apply foundations must be reused where semantically applicable. | 0.4.0 could create conflicting parallel authority or identity systems. |
| C23 | Project Profile surface stability | Tool-internal remediation state must not enter `ptsip.yaml` unless the project genuinely needs to declare new truth, intent, authority, or analysis contract. | Tool convenience metadata could become accidental project architecture authority. |

### Core minimum-set rule

Before a core item survives into the WU graph, the planning record must be able to answer:

```text
If this item is removed, which product objective,
architecture/safety invariant, or release proof breaks?
```

If there is no concrete answer, the item must be demoted to `OPTIONAL` or `DEFERRED`.

## 5. Approved optional boundary

The following responsibilities are `OPTIONAL / NON-BLOCKING` unless separately promoted through the procedure in this document.

| ID | Optional responsibility | Why non-blocking |
| --- | --- | --- |
| O01 | Provider-neutral `AdvisoryResolver` runtime/plugin boundary | The deterministic core and Escalation Proof can function without an advisor. The core invariant is only that advisory output cannot become authority. |
| O02 | Concrete OpenAI/Claude/Gemini/local advisory providers | Provider implementations improve explanation/ranking but are not required for correctness. |
| O03 | Advisory explanation, comparison, and survivor-ranking enhancements | Ranking among already-legal survivors is convenience, not semantic elimination authority. |
| O04 | Additional operational rule families beyond the approved representative release set | More coverage is valuable, but universal coverage is not the 0.4.0 claim. |
| O05 | Advanced automatic external-fact acquisition | Core only needs to classify `EXTERNAL_FACT_REQUIRED`, accept provenanced evidence through an explicit path, and Fresh Solve. Automatic retrieval can be added later. |
| O06 | Advanced rollback/recovery optimization beyond required fail-closed safety | 0.4.0 must preserve safe failure/recovery semantics, but richer optimization is not required to prove the core. |
| O07 | Human-facing comprehensive capability/support matrix | The Tool must not overclaim supported families; a richer reporting surface can remain optional if machine behavior already exposes capability gaps correctly. |
| O08 | Workflow UI simplification from separate `scope` + `mode` to a single selector including `full` | This is development-infrastructure cleanup and does not define remediation architecture correctness. |
| O09 | Additional Test Mode subdivision below one durable remediation verification responsibility | Internal implementation layers do not require one Test Mode each. Add only when a durable independent verification responsibility appears. |
| O10 | Non-authoritative lightweight remediation-history hooks that do not persist or constrain Fresh Solve | Such hooks may support later optimization but cannot influence current candidate elimination or authority. |
| O11 | Additional UX/report rendering beyond the minimum inspectable remediation result and Escalation Proof | Better presentation may improve adoption but does not define semantic correctness. |

Optional work follows this lifecycle:

```text
OPTIONAL
   ↓
implemented directly only when explicitly selected
or isolated on optional branch
   ↓
focused verification
   ↓
explicit merge decision
   ├─ merge
   ├─ defer
   └─ abandon
```

No optional item blocks the 0.4.0 release merely because implementation was started.

## 6. Approved deferred boundary

The following responsibilities are `DEFERRED / OUT-OF-RELEASE` for the initial 0.4.0 boundary.

| ID | Deferred responsibility | Reason |
| --- | --- | --- |
| D01 | Autonomous LLM architect/advisor | Not required for deterministic remediation and introduces unnecessary authority-risk surface in the core release. |
| D02 | Natural-language Specification synthesis | Normative-policy authoring is a separate governance problem, not required generic remediation. |
| D03 | Persistent remediation-history database / learning optimization | Fresh Solve must remain independent of history; persistent optimization can be designed after the core proves itself. |
| D04 | Universal operationalization of all PTSIP Specification rules | 0.4.0 proves the framework with representative complete families rather than claiming universal coverage. |
| D05 | Large secondary remediation-policy DSL | Operational rules should implement Specification behavior without creating a second policy authority. |
| D06 | Project Profile fields for confidence, AI provider state, remediation history, optimization hints, or Tool capability metadata | These are Tool concerns unless a later approved design proves that the project must declare them as authority/intent. |
| D07 | Broad self-optimization/autonomous remediation-learning system | It is unnecessary for the 0.4.0 product promise and risks history/heuristics becoming authority. |

Deferred items should not receive a 0.4.0 implementation WU unless they are first explicitly reclassified.

## 7. Explicit invariants/non-goals that are not promotable features

The following are not optional features waiting for promotion; they are rejected behaviors under the current architecture:

- evidence, derived facts, AI output, confidence, history, or registry convenience metadata implicitly becoming project architecture authority;
- advisory ranking silently deleting a legal semantic candidate;
- semantic determinacy automatically granting mutation authorization;
- stale-state semantic decisions being silently rebased onto changed repository evidence;
- physical edit-order alternatives being presented as distinct owner architecture choices when semantics are equivalent;
- selecting a least-bad invalid candidate from an unsatisfiable solution space;
- treating `TOOL_CAPABILITY_GAP` as owner ambiguity;
- treating `EXTERNAL_FACT_REQUIRED` as project intent;
- self-authorizing Specification/policy changes;
- weakening full exact-SHA release qualification because selective tests passed;
- expanding Project Profile authority solely to store Tool implementation/optimization state.

Changing one of these is not an `OPTIONAL → CORE` promotion. It requires a separate architecture/governance decision and, where applicable, Specification/Project Profile authority changes.

## 8. Promotion rule: OPTIONAL → CORE

Optional work never becomes release-blocking by implementation accident.

Required process:

```text
OPTIONAL
   ↓
Promotion Review
   ↓
identify the exact approved 0.4.0 objective/invariant/release proof
that cannot be satisfied without this item
   ↓
compatibility + verification consequences
   ↓
project-owner approval
   ↓
CORE
```

The following are insufficient promotion reasons:

```text
already implemented
has a branch
has a WU number
would be useful
reduces future effort
makes the design cleaner
AI can do it
```

## 9. Demotion rule: CORE → OPTIONAL / DEFERRED

Core classification may be corrected when implementation evidence proves that the release objective and required safety/release proofs are independently complete without the item.

Required record:

- which former dependency no longer exists;
- which replacement responsibility satisfies the proof;
- compatibility consequences;
- verification evidence;
- project-owner approval of the changed release boundary.

Core status is therefore a release-dependency decision, not an immutable architectural truth.

## 10. Responsibility splitting rule

A broad feature name must not be classified as one indivisible item when its responsibilities have different release relationships.

Example:

```text
AI Advisory
│
├─ advisory cannot become Authority
│      → CORE invariant
│
├─ deterministic core works with no provider
│      → CORE invariant
│
├─ provider-neutral runtime boundary
│      → OPTIONAL
│
├─ concrete provider integrations
│      → OPTIONAL
│
└─ autonomous architect
       → DEFERRED
```

Example:

```text
Safe Apply
│
├─ exact pre-state binding
│      → CORE
├─ stale-state rejection
│      → CORE
├─ mutation authorization separation
│      → CORE
├─ fail-closed recovery semantics
│      → CORE
└─ advanced rollback optimization
       → OPTIONAL
```

## 11. Reconciliation of previously open 0.4.0 planning questions

This Pre-WU resolves the release-classification aspect of several questions previously left open in `planning/0.4.0.md` and `planning/0.4.0/09-experimental-non-goals-and-open-decisions.md`.

### Resolved here

- concrete advisory providers are `OPTIONAL / NON-BLOCKING`;
- the provider-neutral advisory runtime boundary is also `OPTIONAL / NON-BLOCKING` for the initial 0.4.0 release; only its non-authority/no-hidden-dependency invariants are core;
- persistent remediation-history storage/optimization is `DEFERRED / OUT-OF-RELEASE`;
- universal Specification rule operationalization is `DEFERRED / OUT-OF-RELEASE`;
- workflow `scope`/`mode` simplification is `OPTIONAL / NON-BLOCKING`;
- optional/deferred experiments do not receive release dependency merely because they are explored during the 0.4.0 cycle.

### Still intentionally unresolved for Pre-WU-00B or later WU planning

- exact initial representative operational rule families;
- exact package/module/component ownership boundaries;
- exact canonical enum/type names and machine-readable result shapes;
- exact mutation-authorization defaults for reversible mechanical changes;
- exact reuse/extraction boundary between existing migration execution primitives and generic remediation execution;
- exact release acceptance matrix and representative fixtures;
- final core WU decomposition and per-WU verification responsibility/Test Mode coverage.

These unresolved items are no longer allowed to change which capability families are release-blocking by accident. They may refine the approved core implementation contract only within this release boundary unless a formal reclassification is approved.

## 12. Pre-WU-00A completion criteria

Pre-WU-00A is complete when all of the following are true:

- [x] classification states are defined;
- [x] `UNCLASSIFIED` is explicitly non-core;
- [x] Product Objective / Architecture / Release Proof / Deferral Safety gates are defined;
- [x] the initial `CORE / RELEASE-BLOCKING` responsibility set is recorded;
- [x] the initial `OPTIONAL / NON-BLOCKING` responsibility set is recorded;
- [x] the initial `DEFERRED / OUT-OF-RELEASE` responsibility set is recorded;
- [x] explicit non-goal/invariant violations are separated from promotable optional features;
- [x] optional promotion requires explicit owner approval;
- [x] core demotion requires explicit evidence and owner approval;
- [x] broad feature names are required to be split by responsibility when release relationships differ;
- [x] previously open release-classification questions are reconciled;
- [x] Pre-WU-00B inputs are identified.

## 13. Handoff to Pre-WU-00B — Core Contract Freeze

Pre-WU-00B must consume only the approved core boundary and freeze the implementation contracts required before WU-01.

Its minimum decisions are:

```text
1. exact remediation component/package ownership
2. canonical resolution and authorization type/state names
3. exact first representative Operational Rule families
4. existing 0.3.7 contract reuse/extraction map
5. exact semantic lifecycle and resumable Fresh Solve transitions
6. release acceptance matrix
7. initial durable remediation verification responsibility and Test Mode decision
8. core WU dependency/decomposition plan
```

Optional and deferred items must not be pulled into Pre-WU-00B merely to make the design more complete.

The next authorized planning step after this document is therefore:

```text
Pre-WU-00B — Core Contract Freeze
```
