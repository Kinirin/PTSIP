# 0.4.0 — Authority Gate and Safe Apply

> **Status:** DRAFT / DESIGN SOURCE  
> **Parent:** `planning/0.4.0.md`

## 1. Determinacy does not authorize mutation

The Tool may prove that exactly one valid semantic target remains and still lack permission to edit the repository.

```text
Semantic determinacy
        ↓
Authority Gate
        ↓
Mutation authorization decision
```

This boundary must remain explicit in APIs and logs.

## 2. Authority Gate outcomes

The planned decision set is comparable to:

```text
AUTO_AUTHORIZED
MECHANICALLY_AUTHORIZED
OWNER_CONFIRMATION_REQUIRED
NOT_AUTHORIZED
```

Exact names remain subject to implementation review. The semantic distinction is required.

The Authority Gate is evaluated after semantic target determination and before generation/execution of repository mutations.

## 3. Irreversibility-aware execution

Authorization strength should reflect semantic impact and reversibility.

```text
mechanical + reversible
    → auto apply may be allowed

structural but semantic-preserving
    → guarded apply

architecture-semantic
    → owner confirmation

destructive / irreversible
    → stronger confirmation and verification
```

The classification should influence:

- whether automatic application is permitted;
- snapshot/precondition strictness;
- rollback expectations;
- required confirmation;
- postcondition verification depth.

## 4. Safe-apply lifecycle

```text
Semantic Remediation Plan
        ↓
Authority Gate
        ↓
Repository Change Plan
        ↓
Exact Pre-state Snapshot
        ↓
Authorization binding
        ↓
Apply
        ↓
Post-state Validation
        ↓
Full relevant PTSIP Conformance
        ↓
Commit / Promotion
```

The design extends 0.3.7 exact-state and guarded-apply behavior into generic remediation.

## 5. Stale-state rejection

A plan is bound to the repository state on which its evidence/facts and semantic decision were derived.

If the repository changes before apply:

```text
planned pre-state ≠ current pre-state
        ↓
reject plan
        ↓
Fresh Solve required
```

The Tool must not silently rebase a semantic decision onto new evidence.

## 6. Repository Change Plan

The physical execution plan should contain only the mutations needed to realize an already-determined semantic target.

It should not re-decide architecture intent.

Potential responsibilities include:

- create/update/delete/move operations;
- ordering constraints;
- preconditions;
- rollback/recovery metadata for reversible actions;
- exact snapshot binding;
- expected postconditions.

The physical plan may choose among semantically equivalent edit orders without asking the project owner.

## 7. Destructive actions

Destructive or irreversible actions require stronger handling.

The Tool must not treat a destructive operation as automatically authorized merely because the semantic target is deterministic.

Examples of stronger handling may include:

- explicit confirmation token/decision;
- exact target inventory;
- proof that removed data is not required by current authority;
- stronger postcondition verification;
- no implicit fallback after partial failure.

## 8. Failure and recovery

0.4.0 should inherit the fail-closed recovery principles from 0.3.7 migration work:

- apply only accepted semantic deltas;
- maintain exact pre-state identity;
- do not continue from an ambiguous partial state;
- recovery must be tied to an explicit plan/ledger or equivalent durable execution record;
- post-apply validation failure blocks promotion/commit qualification.

## 9. Verification after apply

Successful file mutation is not successful remediation.

The Tool must prove the post-state:

```text
expected semantic target materialized
required architecture invariants preserved
applicable rule now conformant
no unauthorized semantic delta introduced
relevant full conformance passes
```

## 10. Verification expectations

Focused tests should cover:

- deterministic target but owner confirmation still required;
- reversible mechanical auto-authorized path;
- stale snapshot rejection;
- unauthorized plan refusal;
- destructive action requiring stronger authorization;
- partial/apply failure fails closed;
- physical plan does not invent semantic choices;
- postcondition failure blocks success;
- Fresh Solve required after pre-state drift.
