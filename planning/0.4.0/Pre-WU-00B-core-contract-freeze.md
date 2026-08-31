# Pre-WU-00B — 0.4.0 Core Contract Freeze

> **Status:** ACTIVE / OWNER DECISION REVIEW  
> **Target Tool version:** `0.4.0`  
> **Classification:** `CORE / RELEASE-BLOCKING`  
> **Integration branch:** `dev/0.4.0`  
> **Entry baseline:** `00df774bd0958316c1cfbc40bdbe0ba0209484c3`  
> **Release-boundary authority:** `planning/0.4.0/Pre-WU-00A-release-boundary-classification.md`  
> **Open-decision source:** `planning/0.4.0/09-core-open-decisions-and-non-goals.md`  
> **Implementation authorization:** **NOT GRANTED** while this document remains in decision review

## 1. Purpose

Pre-WU-00A fixed which responsibilities are release-blocking. Pre-WU-00B now freezes the exact contracts required before those responsibilities are decomposed into implementation WUs.

The governing sequence is:

```text
Pre-WU-00A
    release boundary fixed
        ↓
Pre-WU-00B
    core contracts fixed
        ↓
approved CORE WU decomposition
        ↓
implementation
```

This document is intentionally not an implementation plan that silently fills open architecture choices. It is an owner-decision record. Each decision below contains a recommended option, alternatives, consequences, and the exact contract that will become frozen only after project-owner approval.

## 2. Non-negotiable inherited boundaries

The following are already fixed by Pre-WU-00A and the 0.4.0 design sources and are not reopened here:

- `Evidence ≠ Derived Fact ≠ Normative Constraint ≠ Project Architecture Authority`;
- semantic determinacy does not imply mutation authorization;
- current Specification + current authority + current repository state always govern a Fresh Solve;
- history, confidence, heuristics, AI/advisory output, path names, and physical test directories are not architecture authority;
- `SemanticRemediationPlan` and `RepositoryChangePlan` remain separate;
- selective Test Mode success does not equal full release qualification;
- Tool-internal remediation state does not enter `ptsip.yaml` merely for implementation convenience;
- 0.3.7 identity, Specification binding, normalized evidence, source compatibility, exact snapshot, and fail-closed safe-apply foundations are reused where semantically neutral;
- migration-specific state machines are not copied into generic remediation merely because they already exist.

## 3. Freeze decisions

Pre-WU-00B closes exactly eight release-blocking decisions.

```text
D01 representative operational rule families
D02 package/module/Project Profile ownership
D03 canonical typed states and diagnostics
D04 mutation class and authorization defaults
D05 Fresh Solve suspension/resume lifecycle
D06 release acceptance matrix
D07 verification responsibility and Test Mode coverage
D08 final CORE WU decomposition
```

Until the owner approves a decision, its state is `PENDING`.

---

## D01 — Representative operational rule families

### Goal

Choose a small release set that proves the remediation framework without claiming universal Specification coverage.

The Specification currently provides useful distinct boundaries:

- `PTSIP-PKG-001` prohibits `DEVELOPMENT_TOOLING`, `DELIVERY`, and `OPERATIONS` implementation from being included in Product distributable/deployable artifacts unless ownership has validly changed to `PRODUCT`;
- `PTSIP-CLS-001` requires every in-scope component ultimately to have one canonical lifecycle classification, while ambiguity may remain unresolved only before strict conformance/shared dependency legality depends on it;
- `PTSIP-CLS-004` requires classification from governing lifecycle obligation rather than file/path/majority/confidence heuristics;
- `PTSIP-CLS-010` requires split/redesign when one component hides independently governable mixed lifecycle responsibilities;
- `PTSIP-DEP-001` prohibits Product runtime/shipped implementation dependency on non-Product implementation;
- `PTSIP-ART-001` requires sufficient artifact ownership/derivation evidence for strict packaging evaluation.

### Option A — Packaging-centered representative set — **RECOMMENDED**

| Required outcome | Representative rule family | Intended proof |
| --- | --- | --- |
| `DETERMINISTIC` | `PTSIP-PKG-001` | With component ownership already explicit and definite Product Artifact leakage established, the semantic target is that the Product Artifact no longer contains the explicitly non-Product implementation. Ownership is not reclassified by the Tool. |
| `OWNER_INTENT_REQUIRED` | `PTSIP-CLS-001` + `PTSIP-CLS-004/011` determination boundary | When a coherent in-scope component lacks enough project authority to select one lifecycle, deterministic evidence/rule analysis stops before classification and asks only for the unresolved governing lifecycle intent. |
| `UNSATISFIABLE` | `PTSIP-DEP-001` synthetic locked-authority case | A fixture fixes Product/non-Product ownership and a required runtime coupling such that all declared modeled remedies would violate current constraints/authority; the Tool reports no legal survivor instead of weakening the rule. |
| `TOOL_CAPABILITY_GAP` | `PTSIP-CLS-010` component-split case | The Tool can prove that a mixed-lifecycle component requires split/redesign while the initial release intentionally does not model a complete generic split transformation family. |
| `EXTERNAL_FACT_REQUIRED` boundary | `PTSIP-ART-001` | Strict artifact evaluation lacks required external artifact/derivation truth; the missing item is requested as provenanced evidence, not owner intent. |

The first mutation-capable vertical slice is therefore packaging isolation, because it has a strong existing 0.3.7 artifact-verification foundation and naturally demonstrates:

```text
semantic target
    ≠ physical edit strategy
```

A supported initial physical remediation family may be deliberately narrow, for example an explicit packaging exclusion in a known fixture/build configuration. If the Tool cannot prove that its physical family is applicable, it must return a capability gap rather than improvise another edit.

**Advantages**

- exercises a real mandatory rule and existing Product Artifact verification;
- gives a deterministic semantic target without allowing the Tool to reclassify project ownership;
- naturally proves plan/apply/postcondition separation;
- gives the first safe-apply slice a concrete repository mutation rather than only abstract domain types.

**Costs / limits**

- physical package mutation support must start narrow and capability-declared;
- package-manager/build-backend diversity cannot be implied as supported in 0.4.0;
- the unsatisfiable and capability-gap cases still require separate synthetic fixtures.

### Option B — Classification-centered first release

Start with `PTSIP-CLS-001/004/010/011` as the main family and defer packaging mutation until later.

**Advantage:** smaller initial repository mutation surface.  
**Disadvantage:** weaker proof of the complete `detect → solve → authorize → physical plan → apply → verify` release promise.

### Freeze proposal

```text
D01 = Option A
status = PENDING OWNER APPROVAL
```

---

## D02 — Package, module, and Project Profile ownership

### Current repository fact

The Project Profile uses explicit path ownership. `ptsip-core` includes selected paths such as `src/ptsip/*.py`, `src/ptsip/app/**`, `src/ptsip/repository/**`, and others; it does **not** own an arbitrary future `src/ptsip/remediation/**` tree.

### Option A — Independent durable Product responsibility — **RECOMMENDED**

Declare:

```text
component: ptsip-remediation
classification: PRODUCT
role: IMPLEMENTATION
include:
  - src/ptsip/remediation/**

component: ptsip-remediation-verification
classification: PRODUCT
role: VERIFICATION
include:
  - tests/ptsip/remediation/**
```

Conceptual internal package boundaries:

```text
src/ptsip/remediation/
├─ domain/
├─ rules/
├─ solution/
├─ planning/
├─ authority/
├─ execution/
└─ verification/
```

These subdirectories are implementation responsibilities inside one durable Product component. They do not become separate PTSIP components merely because they have separate modules.

**Directory responsibilities**

- `domain/` — typed remediation facts, candidates, outcomes, semantic plans, escalation/authorization records;
- `rules/` — Operational Rule implementations and capability declaration, never a second normative policy source;
- `solution/` — candidate enumeration, proof-based elimination, equivalence/dominance reduction, survivor cardinality;
- `planning/` — conversion from resolved semantic target to inspectable semantic/physical planning contracts without mutation;
- `authority/` — Project Intent/Architecture Authority consumption and mutation-authorization gate; does not replace distributed GitHub Decision Authority;
- `execution/` — repository change-plan realization using neutral exact-state/safe-apply primitives;
- `verification/` — postcondition and remediation-result verification logic, distinct from physical pytest placement.

**Advantages**

- explicit durable responsibility like migration/evidence/source-compat;
- prevents generic remediation semantics from becoming migration-specific implementation detail;
- makes future Project Profile/Test Mode ownership clear;
- avoids accidental ownership through broad `ptsip-core` selectors.

**Cost**

- requires a Project Profile component and verification component update before source/tests are introduced;
- adds one durable responsibility to release qualification and dependency review.

### Option B — Extend `ptsip-core`

Add `src/ptsip/remediation/**` to `ptsip-core` and route verification into `ptsip-core-verification`.

**Advantage:** fewer Project Profile components.  
**Disadvantage:** generic remediation becomes harder to isolate as a durable capability and weakens the responsibility boundary established for migration/evidence/source compatibility.

### Freeze proposal

```text
D02 = Option A
status = PENDING OWNER APPROVAL
```

---

## D03 — Canonical typed state and diagnostics contracts

### Option A — Explicit semantic types + two-axis authorization — **RECOMMENDED**

Reuse the existing normalized evidence contract instead of creating a second generic `Evidence` stack.

Freeze these semantic names/concepts:

```text
NormalizedEvidence        = inherited evidence/provenance input
DerivedFact               = deterministic fact derived from evidence
NormativeConstraint       = applicable Specification obligation
ProjectIntentAuthority    = explicit project-owned semantic intent
SemanticCandidate         = one modeled legal semantic target candidate
ResolutionOutcome         = semantic solve outcome
SemanticRemediationPlan   = decided semantic target and proof
EscalationProof           = exact unresolved semantic/fact boundary
MutationClass             = physical/semantic impact class
AuthorizationDecision     = mutation permission result
RepositoryChangePlan      = physical operations for an already-decided target
PostconditionResult       = semantic/materialization/conformance verification result
```

Canonical `ResolutionOutcome` values:

```text
DETERMINISTIC
OWNER_INTENT_REQUIRED
EXTERNAL_FACT_REQUIRED
UNSATISFIABLE
TOOL_CAPABILITY_GAP
```

Canonical mutation impact axis:

```text
MECHANICAL_REVERSIBLE
STRUCTURAL_SEMANTIC_PRESERVING
ARCHITECTURE_SEMANTIC
DESTRUCTIVE
```

Canonical authorization axis:

```text
AUTHORIZED
OWNER_CONFIRMATION_REQUIRED
NOT_AUTHORIZED
```

`MutationClass` and `AuthorizationDecision` remain separate. This replaces a single mixed enum such as `AUTO_AUTHORIZED` / `MECHANICALLY_AUTHORIZED` that would combine *what kind of change this is* with *whether this invocation may execute it*.

The term `ProjectIntentAuthority` is deliberately narrower than a generic `Authority` type so it cannot be confused with existing distributed GitHub Decision Authority/CAS state or mutation permission.

### Option B — One generalized Authority/AuthorityDecision model

**Advantage:** fewer types.  
**Disadvantage:** high risk of collapsing project intent, distributed decision state, and mutation permission into one concept. Not recommended.

### Freeze proposal

```text
D03 = Option A
status = PENDING OWNER APPROVAL
```

---

## D04 — Authorization defaults and irreversibility classes

### Option A — Explicit-apply conservative default — **RECOMMENDED**

No repository mutation is implicit merely because a semantic result is deterministic.

```text
plan / solve operation
    → no mutation authorization

MECHANICAL_REVERSIBLE
    → may become AUTHORIZED by explicit write/apply invocation
      bound to the exact plan/snapshot

STRUCTURAL_SEMANTIC_PRESERVING
    → OWNER_CONFIRMATION_REQUIRED
      with confirmation bound to the exact plan digest

ARCHITECTURE_SEMANTIC
    → explicit ProjectIntentAuthority must already resolve semantics
      + OWNER_CONFIRMATION_REQUIRED for mutation

DESTRUCTIVE
    → explicit semantic authority
      + explicit destructive confirmation bound to exact plan/inventory
      + no automatic fallback
```

Unknown/unsupported mutation class:

```text
NOT_AUTHORIZED
```

This does not require a new `ptsip.yaml` authorization field. Invocation/session authorization is execution permission, not durable project architecture intent.

### Option B — Automatic mechanical mutation by default

The Tool would auto-apply reversible mechanical changes after determinacy.

**Advantage:** lower interaction cost.  
**Disadvantage:** weakens the already-approved `Determinacy ≠ Mutation Authorization` boundary and makes ordinary analysis commands riskier.

### Freeze proposal

```text
D04 = Option A
status = PENDING OWNER APPROVAL
```

---

## D05 — Fresh Solve suspension/resume lifecycle

### Option A — Materialize input, then always Fresh Solve — **RECOMMENDED**

Freeze this state machine:

```text
Current Specification
+ Current Project Authority
+ Current Repository State
        ↓
Fresh Solve
        ↓
┌───────────────────────────────┐
│ OWNER_INTENT_REQUIRED         │
│ EXTERNAL_FACT_REQUIRED        │
└───────────────────────────────┘
        ↓
materialize one of:
- accepted project intent through explicit authority path
- provenance-bound external evidence/fact
        ↓
recapture current state / authority freshness
        ↓
Fresh Solve from the beginning
        ↓
ResolutionOutcome
        ↓
Authority Gate when applicable
```

No paused candidate set is resumed as if the repository and authority were unchanged.

### Option B — Continue the old solve from the suspended candidate set

**Advantage:** less recomputation.  
**Disadvantage:** risks stale evidence/authority and directly conflicts with C02/C10/C15 Fresh Solve requirements.

### Freeze proposal

```text
D05 = Option A
status = PENDING OWNER APPROVAL
```

---

## D06 — Exact release acceptance matrix

### Option A — Explicit proof matrix — **RECOMMENDED**

Tool 0.4.0 cannot claim generic remediation completion unless all release-blocking rows pass.

| Proof area | Minimum release evidence |
| --- | --- |
| Authority firewall | tests prove evidence/derived facts/advisory output cannot directly create Project Intent Authority or mutation permission |
| Operational Rule binding | representative rules expose stable `rule_id`, applicability, evaluation, modeled remediation capability, and verification contract without Tool-version-to-Spec inference |
| Deterministic solve | one modeled semantic survivor with proof-based eliminations; confidence/ranking does not eliminate candidates |
| Owner intent | 2+ legal semantic survivors produce `OWNER_INTENT_REQUIRED` + EscalationProof; accepted intent triggers Fresh Solve |
| External fact | missing technical fact produces `EXTERNAL_FACT_REQUIRED`; supplied value is provenance-bound evidence and triggers Fresh Solve |
| Unsatisfiable | zero legal survivors produces `UNSATISFIABLE`; no least-bad candidate or waiver is selected |
| Capability gap | unmodeled required remediation returns `TOOL_CAPABILITY_GAP` without asking owner to decide Tool capability |
| Authorization | deterministic semantic target still respects mutation class and explicit authorization rules |
| Safe apply | RepositoryChangePlan is exact-snapshot bound; stale state and unauthorized execution are rejected |
| Recovery | ambiguous/partial failure remains fail-closed and cannot be promoted as success |
| Postcondition | expected semantic target, applicable rule conformance, architecture invariants, and unauthorized semantic-delta absence are verified |
| 0.3.7 inheritance | Specification binding, normalized evidence, source compatibility, snapshot and reusable safety contracts are reused or explicitly adapted; no parallel identity/authority stack |
| Project Profile | new remediation implementation/verification ownership is explicit; coverage remains complete and valid |
| Test Mode | remediation selective mode resolves through declared verification `component_ref`; selective success remains non-release authority |
| Repository regression | complete repository pytest succeeds at release checkpoint |
| Distribution | sdist/wheel build, distribution checks, installed-wheel smoke and Product Artifact verification pass |
| Exact-SHA qualification | self-hosted `tooling-test` full qualification succeeds for the exact release candidate SHA |

OPTIONAL / NON-BLOCKING work under `planning/0.4.0-op/` is not a release acceptance dependency.

### Freeze proposal

```text
D06 = Option A
status = PENDING OWNER APPROVAL
```

---

## D07 — CORE verification responsibility and Test Mode coverage

### Option A — One durable remediation verification responsibility + one Test Mode — **RECOMMENDED**

After D02 ownership is approved, declare:

```text
ptsip-remediation
    Product implementation responsibility
        ↓ verified by
ptsip-remediation-verification
    Product verification responsibility
        ↓ component_ref
Test Mode: ptsip-remediation
        ↓
pytest target: tests/ptsip/remediation
```

Internal directories such as `domain`, `rules`, `solution`, `authority`, and `execution` do not each receive their own Test Mode merely because they are implementation layers.

Watch paths may include the remediation source plus relevant shared contracts as execution-selection evidence, but they do not duplicate Project Profile classification/roles/purpose.

### Option B — Multiple layer-specific Test Modes

**Advantage:** more granular selective CI.  
**Disadvantage:** prematurely turns implementation structure into durable verification architecture and increases registry/workflow maintenance.

### Option C — No remediation Test Mode

**Advantage:** no control-plane expansion.  
**Disadvantage:** fails the approved C20 release responsibility because remediation is a durable new responsibility and selective verification is materially useful during 0.4.0 development.

### Freeze proposal

```text
D07 = Option A
status = PENDING OWNER APPROVAL
```

---

## D08 — Final CORE WU decomposition

### Option A — Vertical responsibility slices — **RECOMMENDED**

Do not create one WU for every internal layer. Use session-complete vertical responsibilities:

```text
WU-01 — Remediation Responsibility + Typed Foundation
  - declare ptsip-remediation / verification ownership
  - create package boundary
  - typed domain contracts
  - OperationalRule contract / capability declaration
  - inherited evidence + Specification-binding adapters
  - authority-firewall tests

WU-02 — Solution Space + Deterministic Rule Slice
  - first approved deterministic operational rule
  - fact derivation / applicability
  - candidate generation and proof-based elimination
  - equivalence/dominance reduction
  - ResolutionOutcome
  - SemanticRemediationPlan
  - no physical apply yet unless needed by the approved slice boundary

WU-03 — Escalation + Fresh Solve Boundaries
  - OWNER_INTENT_REQUIRED
  - EXTERNAL_FACT_REQUIRED
  - EscalationProof
  - explicit intent/evidence materialization paths
  - mandatory Fresh Solve resume behavior

WU-04 — Authority Gate + Repository Change Plan + Safe Apply
  - MutationClass / AuthorizationDecision
  - deterministic packaging-isolation physical plan
  - exact snapshot binding
  - explicit authorization
  - apply / fail-closed recovery
  - postcondition verification

WU-05 — Boundary Outcomes + Remediation Verification Closure
  - UNSATISFIABLE
  - TOOL_CAPABILITY_GAP
  - remaining representative fixtures
  - one durable remediation Test Mode
  - cross-WU integration verification

WU-06 — Repository Dogfood + Full Release Qualification
  - PTSIP repository self-adoption of remediation responsibility
  - full regression
  - Project Profile coverage / conformance review
  - package/distribution/Product Artifact checks
  - exact-SHA self-hosted tooling-test qualification
  - release readiness handoff
```

A WU is a responsibility/completion boundary, not a mandatory Git sub-branch.

### Option B — One layer per WU

Domain → rules → solver → planner → escalation → authority → execution → verification each become separate WUs.

**Advantage:** small code diffs.  
**Disadvantage:** leaves many sessions with only partial horizontal architecture and delays the first executable end-to-end proof. It also increases cross-session planning overhead.

### Freeze proposal

```text
D08 = Option A
status = PENDING OWNER APPROVAL
```

---

## 4. Proposed 0.3.7 reuse map

This map prevents both blind reuse and parallel reimplementation.

| Existing foundation | 0.4.0 treatment |
| --- | --- |
| `src/ptsip/evidence/**` normalized evidence/provenance | **REUSE** as evidence input; do not create a second remediation Evidence authority stack |
| `src/ptsip/specification_binding.py` and capability contracts | **REUSE** for exact normative binding; Operational Rule support must not infer from Tool/PP version |
| `src/ptsip/repository/snapshot.py` `RepositorySnapshot` / comparison | **REUSE / GENERALIZE ONLY IF REQUIRED** as neutral exact-state primitive |
| `src/ptsip/source_compat/**` | **CONSUME WHEN SOURCE COMPATIBILITY IS RELEVANT**; do not duplicate readers |
| `src/ptsip/migration/execution_model.py` | **DO NOT REUSE AS GENERIC DOMAIN MODEL**; it contains migration-specific phases/source/final-point semantics |
| migration exact-state/guard ideas | **EXTRACT OR ADAPT ONLY NEUTRAL PRIMITIVES** after explicit review; do not import migration semantic state machine into remediation |
| GitHub Decision Authority / reconciliation | **KEEP SEMANTICALLY DISTINCT** from Project Intent Authority and mutation AuthorizationDecision |

The existing repository snapshot model already provides Git HEAD, status fingerprint, tracked-content fingerprint, and fail-closed observation errors; this is an appropriate candidate neutral primitive for D05/D04 stale-state binding.

## 5. Proposed approval set

The current recommended freeze set is:

```text
D01 = A  Packaging-centered representative rules
D02 = A  Independent ptsip-remediation responsibility
D03 = A  Explicit semantic types + two-axis authorization
D04 = A  Explicit-apply conservative authorization
D05 = A  Always Fresh Solve after supplied intent/fact
D06 = A  Explicit release acceptance matrix
D07 = A  One remediation verification responsibility + one Test Mode
D08 = A  Vertical responsibility WU sequence
```

These are recommendations, not approvals.

## 6. Owner review protocol

The owner may approve the complete recommended set with one decision or override individual items.

Examples:

```text
권고안 전체 승인
```

or:

```text
D01=A
D02=A
D03=A
D04 수정: ...
D05=A
D06=A
D07=A
D08=A
```

An overridden item is updated in this document before the freeze is declared complete.

## 7. Completion gate

Pre-WU-00B becomes complete only when:

```text
D01–D08 owner-approved
        ↓
this document updated to APPROVED / CORE CONTRACT FROZEN
        ↓
09-core-open-decisions-and-non-goals reconciled
        ↓
planning/0.4.0/README current gate updated
        ↓
implementation WU documents may be created
```

Until then:

```text
0.4.0 remediation implementation = NOT AUTHORIZED
```
