# ADR-0005 — Activate PTSIP Specification 0.3.4-draft

**Status:** Accepted  
**Decision:** Activate `0.3.4-draft` as the next normative draft family snapshot after `0.2.0-draft`  
**Design lineage:** `0.3.3-draft` Explicit Project Adoption + `0.3.4-draft` Distributed Authority Consistency

## Context

The active `0.2.0-draft` snapshot established Product/Toolchain/Neutral Contract classification, evidence-relative conformance, artifact-owner/producer separation, lifecycle/build isolation, and Consumer Repository non-intrusion.

Reference Tool `0.3.3` then demonstrated explicit project-owner adoption, but its structured decision answer contained facts that the predecessor Project Profile could not preserve losslessly, especially `runtime_required` and canonical lifecycle ownership.

Reference Tool `0.3.4` completed repository-distributed first-winner coordination and read-side authority freshness. That implementation demonstrated that local SQLite alone cannot coordinate multiple clones and that Git propagation of `ptsip.yaml` alone is insufficient at the instant an architecture decision is accepted.

The published `spec-v0.3.4-draft` design release proposed incorporating both workstreams into one backend-neutral Specification family.

## Decision

PTSIP `0.3.4-draft` becomes the active draft family at the immutable normative migration revision produced by this migration.

The migration adopts these decisions:

1. PTSIP still has exactly three architecture classifications: `PRODUCT`, `TOOLCHAIN`, `NEUTRAL_CONTRACT`.
2. Candidate discovery remains evidence, not architecture authority.
3. Explicit adoption/resolution must preserve supplied architecture facts losslessly: `classification`, `purpose`, `shipped`, `runtime_required`, `lifecycle_owner`, `executable`.
4. Canonical lifecycle owners are `PRODUCT`, `DEVELOPMENT_TOOLING`, and `INDEPENDENT`.
5. `release_owner` and `compatibility_owner` remain separate project metadata and are not aliases for canonical `lifecycle_owner`.
6. Boundary-root shorthand remains structurally available, but a structured write workflow must refuse mutation when shorthand cannot preserve required facts losslessly.
7. Decision Authority and Project Profile are distinct: authority records which explicit answer won; the profile records durable repository architecture declaration.
8. Distributed coordination uses stable coordination-domain/decision identity and first-valid-resolution-wins semantics protected by ordered conditional mutation.
9. Distributed architecture-sensitive reads must account for relevant current authority state even when the local profile is already complete.
10. Read-only absence observation must not fabricate decision history merely to prove absence.
11. Missing/equivalent/conflicting local/remote states require deterministic reconciliation semantics; semantic equivalence is architecture meaning, not serialization formatting.
12. An existing local declaration conflicting with a resolved authority winner must not be overwritten silently.
13. Distributed coordination fails closed when required freshness or safe mutation cannot be established; it must not silently fall back to an isolated Local winner.
14. Global decision state is distinct from clone-local profile projection/application state.
15. Action-time synchronization is sufficient; continuous polling is not required.
16. Decision Authority is not a conformance oracle. Consumer Repository Conformance Evaluation remains evidence-relative and deterministic.

## Normative rule additions

This migration assigns stable rule IDs:

- `PTSIP-ADP-001` — Explicit project adoption preserves architecture intent losslessly.
- `PTSIP-AUT-001` — Decision Authority and Project Profile responsibilities are distinct.
- `PTSIP-AUT-002` — Stable distributed decision identity.
- `PTSIP-AUT-003` — First valid resolution wins through ordered conditional mutation.
- `PTSIP-AUT-004` — Authority freshness at architecture-sensitive boundaries.
- `PTSIP-AUT-005` — Safe authority/profile reconciliation.
- `PTSIP-AUT-006` — Fail-closed distributed coordination.
- `PTSIP-AUT-007` — Global decision state is distinct from local projection state.

`PTSIP-AUT-*` rules apply to implementations claiming distributed coordination capability. They are not extra Consumer Repository architecture rules and do not create another PTSIP plane.

## Schema consequences

`schemas/ptsip-profile.schema.json` moves to `0.3.4-draft` and adds durable `runtime_required` and canonical `lifecycle_owner` component fields.

Component declarations may continue to include project metadata such as `release_owner` and `compatibility_owner`, but those fields do not substitute for `lifecycle_owner`.

Existing `0.2.0-draft` profiles remain historical declarations interpreted only under their immutable predecessor revision. Migration to the new family is explicit; tooling must not silently reinterpret an old profile as `0.3.4-draft`.

## Tool binding consequence

The coherent normative migration state is first committed as an immutable snapshot. Only after that snapshot SHA exists may a Reference Tool binding commit set `SPEC_VERSION = "0.3.4-draft"` and `SPEC_REVISION` to that snapshot.

This two-commit shape is intentional: a Git commit cannot safely contain a literal self-reference to its own SHA. The binding commit therefore follows the immutable normative snapshot and points backward to it.

## Compatibility

The Product/Toolchain/Neutral architecture meaning and established rule IDs remain stable. The principal schema migration is the explicit durable representation of architecture facts required by adoption/decision validation.

A project using boundary-root shorthand may continue to validate structurally, but strict evaluation or structured mutation may require migration to component declarations when the shorthand lacks facts needed for the operation.

## Rejected alternatives

### Share `control-plane.sqlite3` through Git

Rejected. Tool-owned local operational state is not portable project architecture authority and shared SQLite through Git has concurrency/merge semantics unsuitable for distributed first-winner coordination.

### Treat `ptsip.yaml` propagation as instantaneous authority

Rejected. A valid decision may be accepted before the profile change is committed, pushed, fetched, or observed by another clone.

### Let complete local profile always win

Rejected. A complete local profile may be stale relative to an already accepted distributed winner.

### Let remote winner overwrite any local declaration automatically

Rejected. Existing semantic conflict requires explicit reconciliation; silent overwrite destroys project-owned declaration context.

### Fall back to Local DecisionStore on distributed failure

Rejected. That creates split-brain winners for participants that believe they share one coordination domain.
