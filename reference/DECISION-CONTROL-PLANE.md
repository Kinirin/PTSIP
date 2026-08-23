# PTSIP Tool 0.3.6 Decision Authority and Control Plane

This document is **informative Tool-level reference** for decision coordination when a coding-agent or maintainer task reaches an architecture-sensitive boundary. Normative rules come from the bound Specification; this document does not create a PTSIP classification or replace project-owned architecture.

The distributed authority capability originated in earlier Tool versions, but this document describes its current Tool `0.3.6` role and boundaries.

## 1. Keep authority planes separate

PTSIP Tool `0.3.6` distinguishes:

```text
Specification
    -> normative architecture / conformance / coordination rules

Decision Authority
    -> which explicit coordinated architecture answer won

Project Profile / Responsibility Map
    -> durable project-owned architecture declaration

Observed evidence
    -> what repository/artifacts actually do

Conformance Evaluation
    -> whether declaration + evidence satisfy applicable rules
```

A Decision Authority is not a conformance oracle. A resolved winner does not automatically prove that the repository is conformant, and it does not replace `ptsip.yaml`.

## 2. Local Tool state is not global project authority

Local SQLite is Tool-owned operational state. It is appropriate for deliberately local coordination but is not portable repository-global architecture authority and must not be Git-shared merely to synchronize decisions.

A typical local path on Windows is equivalent to:

```text
%LOCALAPPDATA%\PTSIP\decisions\<repository-fingerprint>\control-plane.sqlite3
```

`PTSIP_HOME` may override the Tool state root.

## 3. Coordination backend selection

The Reference Tool can use:

```text
explicit --control-plane <URL>
    -> hosted HTTP Control Plane

explicit --coordination local
    -> Local DecisionStore

explicit --coordination github
    -> GitHub-coordinated authority

otherwise, when a GitHub origin is available
    -> GitHub-coordinated authority

otherwise
    -> Local DecisionStore
```

The exact CLI contract remains Tool behavior and must be verified against the current release. Backend selection does not change the PTSIP classification model.

## 4. GitHub-coordinated authority

The Reference Tool uses a dedicated ref:

```text
refs/heads/ptsip-policy
```

The authority ref is separate from ordinary source branches. It stores small authority/decision records rather than a shared SQLite database.

GitHub is a Tool backend, not a universal Specification dependency. Another backend may conform when it provides equivalent ordering, freshness, conflict, and stale-writer guarantees.

## 5. Stable decision identity

A coordinated decision needs stable identity independent of incidental local state.

The Tool derives a repository/domain identity and normalized component scope so that two clones asking the same architecture question do not accidentally create independent winners merely because they have different branch names, local clarification IDs, formatting, or temporarily different missing-field sets.

Decision identity must represent the coordinated architecture question, not one clone's transient UI/session state.

## 6. First-valid-resolution-wins

For one coordinated decision identity:

```text
PENDING
    -> first valid accepted explicit resolution
    -> RESOLVED winner
```

After a winner exists:

- a later contradictory answer must not replace it silently;
- a stale writer must reread current authority;
- equivalent answers may converge on the existing winner;
- independent decision scopes may progress independently when backend ordering allows it.

A coding agent may execute an explicit maintainer/user decision. It must not invent the first architecture answer merely because authority is currently unresolved.

## 7. Conditional mutation / stale-writer safety

Distributed writes must use ordered conditional mutation such as Git ref compare-and-swap, transaction generation checks, ETags, consensus ordering, or equivalent semantics.

Conceptually:

```text
1. read authority revision A
2. read current decision state from A
3. prepare candidate state B with A as predecessor
4. attempt conditional publication of B only if authority still equals A
5. on conflict, reread current authority
6. accept current winner / retry only when semantically safe / fail explicitly
```

Never force-update authority simply to make a local answer win.

## 8. Read-side authority freshness

Write serialization alone is insufficient. Before an architecture-sensitive operation relies on local declaration state under distributed coordination, the Tool must account for relevant current authority state.

```text
analyze repository/profile
        |
        v
resolve coordination domain + component scope
        |
        v
read current authority
        |
        v
compare local declaration and winner
        |
        v
consistent / reconcile / conflict / fail closed
```

A complete local profile can still be stale relative to coordinated authority.

PTSIP uses **action-time synchronization**, not continuous background polling.

## 9. Read-only absence checks must remain read-only

Checking whether a decision exists must not create a pending record solely to prove no decision exists.

Create or reuse pending authority state only when the active operation actually requires a coordinated decision.

This keeps authority history meaningful and avoids turning observation into mutation.

## 10. Local declaration and remote authority reconciliation

Use semantic architecture meaning, not YAML formatting.

| Local declaration | Authority | Required behavior |
| --- | --- | --- |
| absent | no decision | create/reuse pending only when needed |
| absent | resolved winner | validate and safely project/reconcile winner locally |
| present | no decision | use project declaration; do not fabricate authority history solely for bookkeeping |
| present + equivalent | resolved equivalent | report consistency without formatting-only rewrite |
| present + conflicting | resolved different winner | expose conflict; do not silently overwrite either side |
| repository/profile changed | any state | reject stale application and re-analyze |

A complete Project Profile is not sufficient reason to skip relevant authority reads when distributed coordination applies.

## 11. Global authority state is not local projection state

Keep these models separate:

```text
GLOBAL AUTHORITY
    PENDING
    RESOLVED

LOCAL CLONE / WORKTREE
    missing
    consistent
    locally applied
    stale
    failed
```

A global `RESOLVED` state means one architecture answer won. It does not mean every clone has already written that answer locally.

A local application receipt cannot redefine or reopen the global winner.

## 12. Project Profile mutation boundary

A Decision Authority winner does not itself rewrite `ptsip.yaml`.

Local projection/application remains a separate operation that must:

1. validate the accepted answer against the bound Specification;
2. preserve exact selected profile path;
3. verify repository/profile freshness;
4. prepare only the minimum authorized declaration delta;
5. reject stale or conflicting local state;
6. avoid formatting-only rewrites when semantics are already equivalent.

For template/hybrid declarations, an accepted project decision may authorize a transition or override only within the Tool's defined declaration semantics. Template materialization alone cannot authorize project-owned writes.

## 13. Tool 0.3.6 classification boundary

Decision coordination does not create a lifecycle classification.

Canonical Tool `0.3.6` classifications remain exactly:

```text
PRODUCT
DEVELOPMENT_TOOLING
DELIVERY
OPERATIONS
NEUTRAL_CONTRACT
```

`TOOLCHAIN` is Tool `0.3.5` legacy migration input only.

The decision system must not use path layout, framework, repository heuristics, template similarity, or confidence as a substitute for explicit project architecture intent.

## 14. Decision facts in Tool 0.3.6

Canonical Tool `0.3.6` architecture ownership is represented by `classification` itself. A second canonical `lifecycle_owner` field must not compete with it.

Current explicit clarification/adoption decisions center on facts such as:

```text
classification
purpose
shipped
runtime_required
executable
```

Additional project-owned roles, relationships, associated artifacts, or release/compatibility metadata remain separate architecture facts when applicable.

Legacy Tool `0.3.5` `TOOLCHAIN`/`lifecycle_owner` data is migration evidence, not canonical Tool `0.3.6` authority.

## 15. Gate behavior

A gate may determine that:

```text
NO_DECISION_REQUIRED
DECISION_REQUIRED
resolved/consistent authority is available
conflict/freshness failure prevents the affected operation
```

When a decision is required, stop only the affected architecture-sensitive boundary. Do not invent intent or weaken fail-closed behavior simply to continue automation.

## 16. Adoption behavior

`ptsip adopt` is project-owner adoption behavior. Dry-run is non-mutating. Apply requires explicit authorization and stale-state protection.

On coordinated GitHub use, the Tool may establish/reuse the accepted authority winner and then project the authorized declaration locally, but those remain logically separate operations.

## 17. Conformance remains independent

Even when:

```text
authority winner is fresh
local declaration is consistent
profile validation passes
```

conformance may still be:

```text
CONFORMANT
NON_CONFORMANT
INCOMPLETE
```

because observed dependency, artifact, lifecycle, snapshot, or coverage evidence is evaluated separately.

Do not use authority synchronization as proof of conformance.

## 18. Tool 0.3.6 release status

Tool `0.3.6` development is complete and the release candidate is bound to:

```text
Specification 0.3.6-draft
SPEC_REVISION d6995ed232e845b88d8235b851e80ab54b7804ea
```

Final development-branch exact verification authority is `452d0f8b0c78bdebb180ceb2b9994485f59eb43a` from workflow run/job `32640319047 / 97196299107`.

The remaining boundary is exact-main release verification and publication. This document must not describe Tool `0.3.6` as published until that release boundary actually succeeds.

## 19. Tool 0.3.6.1 migration boundary

Tool `0.3.6.1` owns the assisted migration continuation from legacy Tool `0.3.5` architecture evidence and declarations.

Decision Authority remains relevant to accepted explicit project decisions, but migration evidence, inference, and proposals do not become authority automatically.
