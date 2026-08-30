# 0.4.0 — Representative Rule Families and Vertical Slices

> **Status:** DRAFT / SCOPE DESIGN SOURCE  
> **Parent:** `planning/0.4.0.md`

## 1. Framework proof over universal coverage

0.4.0 is not intended to operationalize every PTSIP Specification rule. The release should establish a reusable remediation framework and prove it with representative complete rule families.

The minimum representative outcomes discussed are:

```text
1. deterministic remediation
2. owner-intent-required remediation
3. unsatisfiable remediation
4. tool-capability-gap remediation
```

`EXTERNAL_FACT_REQUIRED` should also be represented at the domain/diagnostic boundary even if it is not the primary first mutation slice.

## 2. Deterministic representative family

A deterministic rule family should prove:

```text
repository evidence
    ↓
deterministic facts
    ↓
applicable normative constraint
    ↓
modeled legal candidates
    ↓
hard elimination
    ↓
1 semantic survivor
    ↓
Authority Gate
    ↓
physical plan if authorized
    ↓
postcondition verification
```

The test must show that determinacy is not achieved through confidence scoring and that mutation authorization is evaluated separately.

## 3. Owner-intent representative family

The owner-intent family should prove that PTSIP performs all deterministic work before escalation.

```text
facts + constraints
    ↓
2+ legal semantic survivors
    ↓
Escalation Proof
    ↓
minimal unresolved project-intent question
    ↓
owner answer materialized as authority
    ↓
Fresh Solve / continuation under explicit authority
```

The owner must not be asked which Specification rule applies or to reproduce the Tool's candidate analysis.

## 4. Unsatisfiable representative family

The unsatisfiable family should prove:

```text
all modeled candidates
    ↓
valid elimination reasons
    ↓
0 survivors
    ↓
UNSATISFIABLE
```

PTSIP must not:

- select a least-bad invalid solution;
- ask AI to invent an exception;
- weaken a normative constraint;
- reinterpret evidence as new authority.

A Specification/policy change may be proposed only as a separate governance action.

## 5. Capability-gap representative family

The capability-gap family exists to keep determinacy claims honest.

A valid resolution may exist, but if PTSIP does not model the necessary remediation family or operational rule, the result is:

```text
TOOL_CAPABILITY_GAP
```

This must remain distinct from:

```text
OWNER_INTENT_REQUIRED
UNSATISFIABLE
EXTERNAL_FACT_REQUIRED
```

## 6. External-fact boundary case

A representative diagnostic case should demonstrate:

```text
required fact absent from repository/current evidence
    ↓
EXTERNAL_FACT_REQUIRED
```

When the fact is supplied, it enters with provenance as evidence/fact. It does not become architecture intent unless separately declared as such.

## 7. Candidate first implementation shape

The first vertical slice should be narrow enough to test quickly but complete enough to exercise the architecture.

Candidate sequence:

```text
WU foundation
  typed domain + OperationalRule contract
        ↓
Vertical Slice A
  deterministic rule family through semantic plan
        ↓
Vertical Slice B
  owner-intent + Escalation Proof
        ↓
Vertical Slice C
  Authority Gate + safe apply on an authorized mechanical case
        ↓
Boundary Slice
  unsatisfiable + capability gap + external fact diagnostics
```

This sequence is illustrative and still requires explicit WU approval.

## 8. Rule-selection criteria

The initial representative rules should preferably be:

- already grounded in existing PTSIP normative concepts;
- small enough to model completely;
- able to demonstrate real architecture authority boundaries;
- testable with synthetic repositories/fixtures without large integration cost;
- capable of exercising postcondition verification;
- not dependent on AI;
- not dependent on a new Project Profile schema unless project authority truly requires it.

## 9. Completion criteria for a representative family

A rule family is not "implemented" merely because it detects a violation. Completion should include the supported portion of:

```text
detect
explain
plan
classify authority
apply when authorized
verify
```

If apply is intentionally unsupported, the capability matrix must say so explicitly rather than implying end-to-end remediation.

## 10. Release-level success criterion

0.4.0 succeeds when a representative non-conformant repository can be processed so that PTSIP itself:

1. discovers relevant evidence;
2. derives facts;
3. evaluates the applicable normative rule;
4. enumerates modeled legal semantic remedies;
5. eliminates provably invalid/redundant remedies;
6. classifies deterministic / owner-intent / external-fact / unsatisfiable / capability-gap outcomes correctly;
7. asks only for genuinely missing architecture intent;
8. converts an accepted semantic target into a safe physical plan;
9. rejects stale state;
10. applies only authorized changes;
11. verifies the post-state automatically;
12. avoids requiring a coding agent to manually reconstruct ordinary supported PTSIP reasoning.
