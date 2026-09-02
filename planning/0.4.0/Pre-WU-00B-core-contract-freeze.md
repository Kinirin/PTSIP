# Pre-WU-00B — 0.4.0 Core Contract Freeze

> **Status:** APPROVED / CORE CONTRACT FROZEN  
> **Target Tool version:** `0.4.0`  
> **Classification:** `CORE / RELEASE-BLOCKING`  
> **Integration branch:** `dev/0.4.0`  
> **Entry baseline:** `00df774bd0958316c1cfbc40bdbe0ba0209484c3`  
> **Release-boundary authority:** `planning/0.4.0/Pre-WU-00A-release-boundary-classification.md`  
> **Open-decision source:** `planning/0.4.0/09-core-open-decisions-and-non-goals.md`  
> **Owner approval:** project-owner approval, 2026-09-02  
> **Implementation authorization:** this freeze authorizes creation of the approved CORE WU documents; repository implementation begins only under those WU execution boundaries

## 1. Purpose

Pre-WU-00A fixed which responsibilities are release-blocking. Pre-WU-00B fixes the exact contracts that those responsibilities must implement and verify before Tool `0.4.0` may claim authority-aware generic remediation.

The governing sequence is now:

```text
Pre-WU-00A
    release boundary fixed
        ↓
Pre-WU-00B
    core contracts frozen
        ↓
approved CORE WU documents
        ↓
implementation
        ↓
release acceptance / exact-SHA qualification
```

This document is the controlling Core Contract Freeze for the initial Tool `0.4.0` release boundary. Earlier alternatives remain recoverable from Git history but are not active contracts after this approval.

## 2. Non-negotiable frozen invariants

The following are release-blocking invariants:

```text
Evidence
    ≠ Derived Fact
    ≠ Normative Constraint
    ≠ Project Architecture Authority
```

and, for remediation routing:

```text
Coverage Gap
    ≠ Remediation Candidate
    ≠ Accepted Architecture Decision
```

Additional frozen boundaries:

- semantic determinacy does not imply mutation authorization;
- current Specification + current authority + current repository state always govern a Fresh Solve;
- history, confidence, heuristics, AI/advisory output, path names, and physical test directories are not architecture authority;
- `SemanticRemediationPlan` and `RepositoryChangePlan` remain separate;
- selective Test Mode success does not equal full release qualification;
- Tool-internal remediation state does not enter `ptsip.yaml` merely for implementation convenience;
- 0.3.7 identity, Specification binding, normalized evidence, source compatibility, exact snapshot, and fail-closed safe-apply foundations are reused where semantically neutral;
- migration-specific state machines are not copied into generic remediation merely because they already exist;
- a blocking diagnostic is not permission to mutate the Project Profile or repository;
- unsupported remediation remains an explicit capability gap rather than an inferred architecture answer.

## 3. Issue #31 as consumer-derived contract evidence

GitHub Issue `#31` (`UX: connect conform unassigned gaps to clarification and remediation triage`) is accepted as real consumer-derived evidence for the frozen remediation architecture.

The issue exposed this unsafe interpretation:

```text
component-ownership:unassigned-relevant-files
        ↓
"assign every file into ptsip.yaml"
```

Tool `0.4.0` must instead preserve the following semantic route:

```text
Coverage Gap / UNASSIGNED
        ↓
non-authoritative deterministic triage
        ↓
RemediationDispositionCandidate(s)
        ↓
Solution Space reduction
        ↓
ResolutionOutcome
        ├─ DETERMINISTIC
        ├─ OWNER_INTENT_REQUIRED
        ├─ EXTERNAL_FACT_REQUIRED
        ├─ UNSATISFIABLE
        └─ TOOL_CAPABILITY_GAP
        ↓
accepted semantic target / authority path when required
        ↓
Authority Gate
        ↓
RepositoryChangePlan only when authorized
```

A candidate disposition may represent, when supported by evidence and modeled capability:

```text
EXISTING_COMPONENT_CANDIDATE
ASSOCIATED_ARTIFACT_CANDIDATE
RELOCATION_CANDIDATE
REMOVAL_CANDIDATE
NEW_COMPONENT_CANDIDATE
DECISION_REQUIRED
```

These values are a non-authoritative candidate/disposition axis. They are not lifecycle classifications and do not authorize profile mutation, relocation, deletion, or component creation.

The exact public UX surface is not frozen here. A dedicated `ptsip triage` command, a `clarify --from-gap` route, or structured next-action metadata may be selected during implementation without changing the semantic contract above. The release-blocking requirement is that an agent-consumable public path exists and does not flatten `UNASSIGNED` into automatic assignment work.

## 4. D01 — Representative operational rule families — APPROVED

The approved release set is packaging-centered and uses existing Specification rules to prove distinct remediation outcomes.

| Required outcome | Representative rule family | Frozen proof responsibility |
| --- | --- | --- |
| `DETERMINISTIC` | `PTSIP-PKG-001` | With ownership already explicit and definite Product Artifact leakage established, the Tool determines the semantic target that the Product Artifact no longer contains explicitly non-Product implementation. It does not silently reclassify ownership. |
| `OWNER_INTENT_REQUIRED` | `PTSIP-CLS-001` + `PTSIP-CLS-004/011` | When evidence cannot resolve governing lifecycle intent, deterministic analysis stops after reducing the solution space and requests only the unresolved project intent. |
| `UNSATISFIABLE` | `PTSIP-DEP-001` locked-authority fixture | Current constraints/authority eliminate every modeled legal remedy; the Tool returns no survivor instead of weakening the rule or choosing a least-bad candidate. |
| `TOOL_CAPABILITY_GAP` | `PTSIP-CLS-010` component-split case | The Tool proves that split/redesign is required while the initial release does not claim a complete generic split transformation family. |
| `EXTERNAL_FACT_REQUIRED` | `PTSIP-ART-001` boundary | Missing artifact/derivation truth is requested as provenance-bound evidence, not as project intent. |

The first mutation-capable vertical slice is `PTSIP-PKG-001` packaging isolation.

```text
semantic target
    ≠ physical edit strategy
```

The initial physical remediation family may be deliberately narrow and build-backend/configuration specific. If applicability cannot be proven, the Tool returns `TOOL_CAPABILITY_GAP` rather than improvising another mutation.

Issue #31 adds a second representative routing proof, not a competing mutation family:

```text
unassigned coverage fact
    ↓
non-authoritative disposition candidates
    ↓
semantic resolution / escalation
```

This proves safe routing and owner-intent boundaries while the packaging family proves the complete mutation-capable path.

## 5. D02 — Package, module, and Project Profile ownership — APPROVED

Generic remediation is an independent durable Product responsibility.

Project Profile ownership to be introduced by the first implementation WU:

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

Canonical package responsibility:

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

Directory responsibilities:

- `domain/` — typed remediation facts, gaps, candidates, outcomes, semantic plans, escalation and authorization records;
- `rules/` — Operational Rule implementations and declared capability; never a second normative policy source;
- `solution/` — candidate enumeration, proof-based elimination, equivalence/dominance reduction, survivor cardinality;
- `planning/` — semantic and physical planning contracts without mutation;
- `authority/` — Project Intent Authority consumption and mutation Authorization Gate; distinct from distributed GitHub Decision Authority;
- `execution/` — RepositoryChangePlan realization using neutral exact-state/safe-apply primitives;
- `verification/` — postcondition and remediation-result verification logic, distinct from physical pytest placement.

These internal directories are not separate PTSIP components merely because they are separate implementation layers.

Generic remediation must remain distinct from `ptsip-migration`.

## 6. D03 — Canonical typed state and diagnostics contracts — APPROVED

The frozen semantic concepts are:

```text
NormalizedEvidence              = inherited evidence/provenance input
CoverageGap                     = observed/derived coverage insufficiency; not architecture intent
DerivedFact                     = deterministic fact derived from evidence
NormativeConstraint             = applicable Specification obligation
ProjectIntentAuthority          = explicit project-owned semantic intent
RemediationDispositionCandidate = non-authoritative triage candidate for a gap/finding
SemanticCandidate               = one modeled semantic target candidate
ResolutionOutcome               = semantic solve outcome
SemanticRemediationPlan         = decided semantic target and proof
EscalationProof                 = exact unresolved semantic/fact boundary
MutationClass                   = physical/semantic impact class
AuthorizationDecision           = mutation permission result
RepositoryChangePlan            = physical operations for an already-decided target
PostconditionResult             = semantic/materialization/conformance verification result
```

`NormalizedEvidence` reuses the established evidence/provenance contract; 0.4.0 does not create a parallel remediation Evidence authority stack.

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

`MutationClass` and `AuthorizationDecision` are separate axes.

The term `ProjectIntentAuthority` is deliberately narrower than generic `Authority`; it must not be confused with distributed GitHub Decision Authority/CAS state or with mutation permission.

A `RemediationDispositionCandidate` is explicitly non-authoritative. Candidate generation may use repository evidence, but candidate existence cannot materialize Project Intent Authority.

## 7. D04 — Authorization defaults and irreversibility classes — APPROVED

The frozen default is explicit-apply and conservative.

```text
plan / solve operation
    → no mutation authorization

MECHANICAL_REVERSIBLE
    → may become AUTHORIZED only by explicit write/apply invocation
      bound to the exact plan/snapshot

STRUCTURAL_SEMANTIC_PRESERVING
    → OWNER_CONFIRMATION_REQUIRED
      confirmation bound to the exact plan digest

ARCHITECTURE_SEMANTIC
    → explicit ProjectIntentAuthority must already resolve semantics
      + OWNER_CONFIRMATION_REQUIRED for mutation

DESTRUCTIVE
    → explicit semantic authority
      + explicit destructive confirmation bound to exact plan/inventory
      + no automatic fallback
```

Unknown or unsupported mutation class:

```text
NOT_AUTHORIZED
```

Invocation/session authorization is execution permission, not durable Project Profile authority. No new `ptsip.yaml` authorization field is introduced merely for Tool execution state.

## 8. D05 — Fresh Solve suspension/resume lifecycle — APPROVED

The frozen lifecycle is:

```text
Current Specification
+ Current Project Authority
+ Current Repository State
        ↓
Fresh Solve
        ↓
OWNER_INTENT_REQUIRED or EXTERNAL_FACT_REQUIRED
        ↓
materialize exactly one appropriate input:
- accepted project intent through explicit authority path
- provenance-bound external evidence/fact
        ↓
recapture repository state + authority freshness
        ↓
Fresh Solve from the beginning
        ↓
ResolutionOutcome
        ↓
Authority Gate when applicable
```

No paused candidate set resumes as though repository evidence and authority remained unchanged.

The same stale-state rule applies after any repository drift:

```text
planned pre-state ≠ current pre-state
        ↓
reject plan
        ↓
Fresh Solve
```

## 9. D06 — Exact release acceptance matrix — APPROVED

Tool `0.4.0` cannot claim the generic remediation framework complete unless every release-blocking row passes.

| Proof area | Minimum release evidence |
| --- | --- |
| Authority firewall | evidence, derived facts, candidate dispositions, confidence and advisory output cannot directly create Project Intent Authority or mutation permission |
| Coverage-gap routing | `UNASSIGNED`/coverage gaps remain facts; an agent-consumable public path distinguishes candidate dispositions and does not automatically assign, relocate, remove, or create components |
| Operational Rule binding | representative rules expose stable `rule_id`, applicability, evaluation, modeled remediation capability and verification contract without Tool-version-to-Spec inference |
| Deterministic solve | one modeled semantic survivor with proof-based eliminations; confidence/ranking does not eliminate candidates |
| Owner intent | 2+ legal semantic survivors produce `OWNER_INTENT_REQUIRED` + `EscalationProof`; accepted intent triggers Fresh Solve |
| External fact | missing technical fact produces `EXTERNAL_FACT_REQUIRED`; supplied value becomes provenance-bound evidence and triggers Fresh Solve |
| Unsatisfiable | zero legal survivors produces `UNSATISFIABLE`; no least-bad candidate or waiver is selected |
| Capability gap | unmodeled required remediation returns `TOOL_CAPABILITY_GAP` without asking the owner to decide Tool capability |
| Authorization | deterministic semantic target still respects mutation class and explicit authorization rules |
| Safe apply | `RepositoryChangePlan` is exact-snapshot bound; stale state and unauthorized execution are rejected |
| Recovery | ambiguous/partial failure remains fail-closed and cannot be promoted as success |
| Postcondition | expected semantic target, applicable rule conformance, architecture invariants and absence of unauthorized semantic delta are verified |
| 0.3.7 inheritance | Specification binding, normalized evidence, source compatibility, snapshot and reusable safety contracts are reused or explicitly adapted; no parallel identity/authority stack |
| Project Profile | new remediation implementation/verification ownership is explicit; coverage remains complete and valid |
| Test Mode | remediation selective mode resolves through declared verification `component_ref`; selective success remains non-release authority |
| Repository regression | complete repository pytest succeeds at release checkpoint |
| Distribution | sdist/wheel build, distribution checks, installed-wheel smoke and Product Artifact verification pass |
| Exact-SHA qualification | self-hosted `tooling-test` full qualification succeeds for the exact release-candidate SHA |

The Issue #31 root problem is therefore release-relevant as a **routing/authority-safety acceptance proof**. A particular command name or rich UX presentation is not itself release authority.

OPTIONAL / NON-BLOCKING work under `planning/0.4.0-op/` remains outside release acceptance unless separately promoted.

## 10. D07 — CORE verification responsibility and Test Mode coverage — APPROVED

One durable remediation verification responsibility and one Test Mode are approved.

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

Watch paths are execution-selection evidence only. They may include remediation source and relevant shared contracts but do not duplicate Project Profile classification, roles, or purpose.

The invariant remains:

```text
Verification Responsibility
    ≠ Physical Test Location
    ≠ CI Execution Mode
```

and:

```text
Selective Test Mode success
    ≠ full repository/release qualification
```

## 11. D08 — Final CORE WU decomposition — APPROVED

The CORE implementation is decomposed into vertical responsibility slices, not one WU per internal module layer.

```text
WU-01 — Remediation Responsibility + Typed Foundation
  - declare ptsip-remediation / ptsip-remediation-verification ownership
  - create package and verification boundaries
  - typed domain contracts
  - CoverageGap / RemediationDispositionCandidate authority separation
  - OperationalRule contract / capability declaration
  - inherited evidence + Specification-binding adapters
  - authority-firewall tests

WU-02 — Solution Space + Deterministic Rule / Coverage Routing Slice
  - approved deterministic operational rule
  - fact derivation / applicability
  - candidate generation and proof-based elimination
  - equivalence/dominance reduction
  - coverage-gap → non-authoritative disposition candidate routing
  - ResolutionOutcome
  - SemanticRemediationPlan
  - no repository mutation yet except if separately required by the WU boundary

WU-03 — Escalation + Fresh Solve Boundaries
  - OWNER_INTENT_REQUIRED
  - EXTERNAL_FACT_REQUIRED
  - EscalationProof
  - DECISION_REQUIRED routing for unresolved coverage/remediation semantics
  - explicit intent/evidence materialization paths
  - mandatory Fresh Solve behavior

WU-04 — Authority Gate + RepositoryChangePlan + Safe Apply
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
  - Issue #31 regression fixture for multiple unassigned dispositions
  - agent-consumable public follow-up route without authority promotion
  - one durable remediation Test Mode
  - cross-WU integration verification

WU-06 — Repository Dogfood + Full Release Qualification
  - PTSIP repository self-adoption of remediation responsibility
  - full regression
  - Project Profile coverage / conformance review
  - package/distribution/Product Artifact checks
  - exact-SHA self-hosted tooling-test qualification
  - release-readiness handoff
```

A WU is a responsibility/completion boundary, not a mandatory Git sub-branch.

## 12. Frozen 0.3.7 reuse map

| Existing foundation | 0.4.0 treatment |
| --- | --- |
| `src/ptsip/evidence/**` normalized evidence/provenance | **REUSE** as evidence input; do not create a second remediation Evidence authority stack |
| `src/ptsip/specification_binding.py` and capability contracts | **REUSE** for exact normative binding; Operational Rule support must not infer from Tool/PP version |
| `src/ptsip/repository/snapshot.py` `RepositorySnapshot` / comparison | **REUSE / GENERALIZE ONLY IF REQUIRED** as neutral exact-state primitive |
| `src/ptsip/source_compat/**` | **CONSUME WHEN SOURCE COMPATIBILITY IS RELEVANT**; do not duplicate readers |
| `src/ptsip/migration/execution_model.py` | **DO NOT REUSE AS GENERIC DOMAIN MODEL**; it contains migration-specific phases/source/final-point semantics |
| migration exact-state/guard ideas | **EXTRACT OR ADAPT ONLY NEUTRAL PRIMITIVES** after explicit review; do not import the migration semantic state machine into remediation |
| GitHub Decision Authority / reconciliation | **KEEP SEMANTICALLY DISTINCT** from Project Intent Authority and `AuthorizationDecision` |
| clarification/candidate discovery and migration proposal concepts relevant to Issue #31 | **REUSE OR ADAPT SEMANTICALLY NEUTRAL CAPABILITIES**; do not expose migration-specific state as generic remediation authority |

The repository snapshot model's Git HEAD, status fingerprint, tracked-content fingerprint and fail-closed observation errors are the approved neutral starting point for exact-state binding.

## 13. Approval record

The project owner approved the complete recommended set on 2026-09-02, with Issue #31 incorporated as a consumer-derived routing/authority-safety acceptance case.

```text
D01 = A  APPROVED — Packaging-centered representative rules
D02 = A  APPROVED — Independent ptsip-remediation responsibility
D03 = A  APPROVED — Explicit semantic types + two-axis authorization
D04 = A  APPROVED — Explicit-apply conservative authorization
D05 = A  APPROVED — Always Fresh Solve after supplied intent/fact or stale-state drift
D06 = A  APPROVED — Explicit release acceptance matrix + coverage-gap routing proof
D07 = A  APPROVED — One remediation verification responsibility + one Test Mode
D08 = A  APPROVED — Vertical responsibility WU sequence
```

## 14. Completion gate

Pre-WU-00B is complete.

```text
D01–D08 owner-approved
        ↓
CORE CONTRACT FROZEN
        ↓
09-core-open-decisions-and-non-goals reconciled
        ↓
planning/0.4.0/README current gate updated
        ↓
approved CORE WU documents may be created
```

This approval does **not** authorize arbitrary repository implementation outside the approved WU boundaries.

Next planning action:

```text
create WU-01 planning document
    ↓
review exact entry baseline / scope / verification / non-goals
    ↓
then begin WU-01 implementation under that approved document
```
