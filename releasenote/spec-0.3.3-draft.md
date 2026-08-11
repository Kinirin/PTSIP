# PTSIP Specification 0.3.3-draft Release Notes

**Status:** Proposed experimental Specification family  
**Predecessor:** `0.2.0-draft`  
**Normative baseline:** `0.2.0-draft` revision `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`  
**Candidate normative snapshot:** To be assigned by the coherent normative migration commit  
**Identity model:** draft family label + immutable Git revision

The `0.3.3-draft` label is a new mutable experimental Specification family proposed from the current `0.2.0-draft` normative baseline and the behavior validated by PTSIP Reference Tool `0.3.3`.

This document is a release-note-level design record. It does not by itself make `0.3.3-draft` the active normative Specification or rebind a published Tool. The family becomes usable only after all affected normative documents, schemas, registry data, agent contracts, embedded resources, and Tool binding constants are updated coherently and the resulting immutable Git revision is recorded.

## 1. Compatibility baseline

Unless explicitly changed below, `0.3.3-draft` inherits the normative semantics of `0.2.0-draft` revision `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`.

The following architectural invariants remain unchanged:

- PTSIP has exactly three architecture classifications: `PRODUCT`, `TOOLCHAIN`, and `NEUTRAL_CONTRACT`;
- `UNKNOWN`, `CONFLICT`, and `INCOMPLETE` remain decision/evaluation states rather than additional architecture planes;
- a Project Profile is a project-owned declaration of intended architecture and is not itself conformance truth;
- declarations and observed evidence remain distinct;
- Product Artifact owner and producer remain distinct concepts;
- project-specific dependency policy may strengthen but may not weaken universal PTSIP rules;
- mandatory-rule violations are not waived by project governance metadata;
- external PTSIP tooling remains non-intrusive toward Consumer Repository structure;
- Specification and Reference Tool retain independent release identities even when both happen to use `0.3.3` labels.

## 2. Why a new draft family is needed

Reference Tool `0.3.3` exposed an architectural contract that can no longer be represented cleanly as Tool-only behavior while the Specification remains fixed at `0.2.0-draft`.

The main gap is that PTSIP now distinguishes three different kinds of state:

```text
PTSIP Specification
    -> normative architecture and conformance rules

Consumer Repository Project Profile
    -> durable revision-bound architecture declaration

Decision Coordination State
    -> pending/resolved architecture-decision workflow state
```

A local-only DecisionStore is sufficient for one isolated environment but cannot provide repository-global first-winner semantics when local agents, cloud agents, CI agents, or multiple clones act on the same unresolved component scope.

The new draft therefore makes the separation between **architecture declaration** and **decision authority** explicit at the Specification level.

## 3. Decision Authority model

`0.3.3-draft` introduces **Decision Authority** as the authority that serializes unresolved and resolved architecture decisions for a defined coordination domain.

A Decision Authority is not the Project Profile and does not replace it.

```text
Decision Authority
    -> determines the accepted answer for an unresolved architecture decision

Project Profile
    -> records the architecture declaration applied to a repository revision
```

A conforming implementation MAY use a local store for intentionally local coordination or a distributed backend for multi-environment coordination.

Local operational storage such as SQLite is implementation state. It MUST NOT be treated as a portable architecture source of truth and MUST NOT be required to reconstruct the authoritative project architecture when the durable Project Profile and the selected distributed authority are available.

## 4. Coordination domain and stable decision identity

A distributed Decision Authority MUST identify the same architectural decision consistently across participating environments.

The authority key for one component scope MUST NOT depend solely on clone-local facts such as:

- a local clarification identifier;
- branch-local missing-field state;
- temporary profile incompleteness;
- one worktree's local database identity.

The identity MUST instead be derived from stable coordination-domain identity plus normalized architectural component scope.

The Reference Tool `0.3.3` satisfies this requirement for GitHub repositories through a repository identity plus normalized discovered include selectors and represents the resulting identity as a `gdec-*` decision ID.

The specific `gdec-*` encoding is a Reference Tool interoperability detail unless separately standardized by a normative schema.

## 5. First-valid-resolution-wins

For one authoritative decision identity, the first valid accepted resolution wins.

A later contradictory answer MUST NOT silently replace an already accepted authoritative resolution.

A distributed Decision Authority MUST serialize competing mutations using an atomic conditional-write, compare-and-swap, transaction, consensus, or equivalent mechanism that prevents a stale writer from overwriting a newer accepted state.

Conceptually:

```text
read authority revision A
    -> prepare valid mutation B based on A
    -> conditionally publish A -> B

if authority already moved from A
    -> reject stale mutation
    -> reread authoritative state
    -> reuse accepted winner or retry only when no conflicting winner exists
```

Wall-clock time alone MUST NOT determine which decision is authoritative when a stronger ordered authority revision is available.

## 6. Distributed conflict semantics

Concurrent answers for different unresolved component scopes MAY both succeed after deterministic retry/rebase against the latest authority revision.

Concurrent contradictory answers for the same component scope MUST converge to exactly one accepted winner within one coordination domain.

After observing an already resolved winner, an implementation MUST NOT retry a contradictory answer as though the decision were still pending.

The losing operation SHOULD return a stable machine-readable result equivalent to `ALREADY_RESOLVED` or `CONFLICT` together with the accepted classification and decision identity when safe to expose.

## 7. Action-time synchronization

`0.3.3-draft` does not require continuous background polling.

A PTSIP implementation MAY synchronize with its selected Decision Authority at architecture-sensitive operation boundaries.

This model is called **action-time synchronization**:

```text
architecture-sensitive operation
    -> inspect selected Project Profile
    -> if required intent is missing, consult Decision Authority
    -> reconcile an existing winner or request an explicit decision
```

Correctness MUST NOT depend on every clone continuously holding an identical `ptsip.yaml` worktree snapshot.

## 8. Project Profile reconciliation

When the selected Project Profile does not yet contain a declaration but the selected Decision Authority already contains a valid resolved winner for that component scope, a conforming implementation MAY project the authoritative answer into the local Project Profile.

Such projection MUST:

- preserve the accepted authoritative answer;
- reject a conflicting existing project declaration rather than silently overwrite it;
- protect against concurrent local profile modification;
- validate the resulting Project Profile before considering application successful;
- preserve the distinction between global decision resolution and clone-local profile application.

The existence of a resolved Decision Authority record does not imply that every clone has already applied that decision to its worktree.

## 9. Fail-closed distributed coordination

When a project or operation has selected a distributed Decision Authority and that authority cannot be reached, authenticated, validated, or safely mutated, the implementation MUST NOT silently create an independent local winner for the same decision.

For new or changed architecture decisions, the operation MUST fail closed with a coordination error.

Existing committed Project Profile declarations MAY remain readable and usable according to normal Specification rules.

This requirement prevents split-brain architecture authority.

## 10. Explicit project adoption

`0.3.3-draft` formalizes **Explicit Project Adoption** as the process that establishes or extends a project-owned architecture declaration from discovered repository scope plus explicit project-owner facts.

Candidate discovery contributes observed scope/evidence. The project owner contributes architectural intent.

An adoption implementation MUST NOT infer `PRODUCT`, `TOOLCHAIN`, or `NEUTRAL_CONTRACT` solely from directory names, package names, file names, or similar repository labels.

A safe adoption transaction SHOULD provide a read-only planning mode and MUST perform equivalent validation before mutation, including:

- selected component/candidate identity validation;
- repository evidence freshness validation;
- explicit architecture-fact validation;
- projected Project Profile validation;
- conflict detection against existing declarations;
- concurrent profile-content protection;
- deterministic/idempotent behavior for equivalent declarations.

The Reference Tool command `ptsip adopt` is one implementation of this Specification-level adoption concept; the CLI spelling itself is not a universal protocol requirement.

## 11. Durable decision facts and Project Profile schema

Reference Tool `0.3.3` validates architecture decisions using the explicit facts:

- `classification`;
- `purpose`;
- `shipped`;
- `runtime_required`;
- `lifecycle_owner`;
- `executable`.

The predecessor `0.2.0-draft` Project Profile schema does not contain a durable `runtime_required` component field. Tool `0.3.3` intentionally retained that fact only in decision/adoption workflow state rather than silently widening the immutable predecessor schema.

`0.3.3-draft` proposes `runtime_required` as a normative component declaration fact.

The Project Profile schema migration MUST define:

```yaml
runtime_required: true | false
```

with semantics:

- `true` means the Product requires the component's executable/runtime implementation during Product runtime;
- `false` means no such Product runtime requirement is declared;
- a `TOOLCHAIN` component MUST NOT declare `runtime_required: true`;
- absence semantics for migrated legacy profiles MUST be explicit and MUST NOT be guessed as `false` during strict evaluation.

This is a `SCHEMA_CHANGE` and may also affect conformance or profile-validation behavior.

`lifecycle_owner` is already a required decision fact in the Reference Tool but is not represented under that exact field name in the predecessor Project Profile component schema. The normative migration MUST either add a canonical `lifecycle_owner` field or explicitly define a lossless mapping to existing ownership fields. It MUST NOT leave decision validation and durable Project Profile semantics ambiguous.

## 12. Decision validation semantics

The following decision constraints validated by Reference Tool `0.3.3` are candidates for normative adoption in this family:

- `PRODUCT` requires Product lifecycle ownership;
- `TOOLCHAIN` requires development-tooling lifecycle ownership;
- `TOOLCHAIN` cannot be shipped as part of the Product;
- `TOOLCHAIN` cannot be required by the Product at runtime;
- `NEUTRAL_CONTRACT` must be non-executable;
- `NEUTRAL_CONTRACT` requires independent lifecycle ownership.

The final normative migration MUST reconcile these facts with existing `PTSIP-DEP-*`, `PTSIP-LCY-*`, neutrality, artifact, and component-boundary rules rather than creating a second contradictory classification rule system.

## 13. GitHub-coordinated authority reference profile

The Specification-level requirements above are backend-neutral.

The Reference Tool `0.3.3` provides one concrete distributed Decision Authority profile for GitHub repositories:

```text
refs/heads/ptsip-policy
```

with:

```text
authority.json

decisions/
    <global-decision-id>.json
```

and non-force Git ref updates that act as compare-and-swap serialization.

This GitHub ref layout is a Reference Tool interoperability contract, not a requirement that every PTSIP implementation use GitHub. An independent implementation MAY use another backend if it satisfies the same authority, stable identity, atomic conflict, reconciliation, and fail-closed semantics.

GitHub Projects, Issues, dashboards, or other human interfaces MAY present decision state but MUST NOT be treated as authoritative merely because they display it. If used as an authority backend, they must independently satisfy the same atomic decision requirements.

## 14. Local and hosted authority compatibility

`0.3.3-draft` permits multiple authority backends with explicit scope:

```text
LOCAL
    -> isolated coordination domain

DISTRIBUTED_REPOSITORY
    -> repository-scoped shared authority

HOSTED_CONTROL_PLANE
    -> externally hosted shared authority
```

Backend selection is an implementation concern, but a conforming Tool MUST make the selected authority unambiguous for an architecture-decision operation.

A selected distributed authority MUST NOT silently fall back to an isolated local authority on network or authentication failure.

## 15. Coding-agent contract implications

A coding agent operating under `0.3.3-draft` MUST:

- inspect the Project Profile before inventing architecture intent;
- stop the affected boundary-sensitive work when required intent is unresolved;
- consult the selected Decision Authority when the active workflow requires coordinated resolution;
- record only explicit project-owner/human facts as an architecture answer;
- not create a second local winner when distributed coordination is selected but unavailable;
- accept and reconcile an already authoritative winner rather than asking the user to decide the same component scope again;
- distinguish authoritative decision resolution from local Project Profile application.

These requirements extend the existing rule that a coding agent must not invent missing architecture intent.

## 16. Machine-readable result expectations

Decision/adoption/gate interfaces SHOULD expose stable machine-readable states sufficient for coding agents and CI to distinguish at least:

- no decision required;
- decision required;
- resolved/authoritative winner available;
- already resolved by another winner;
- stale evidence;
- declaration conflict;
- coordination unavailable/error;
- local application failure.

The exact JSON envelopes remain interoperability-schema work. The existing Tool `0.3.3` `gate --json` exception-envelope limitations are not silently standardized by this draft.

## 17. Preserved 0.2.0-draft semantics

The new decision-authority model does not change the predecessor family's established evidence, artifact, conformance, diagnostic, dependency-policy, non-intrusion, or immutable-binding principles except where a later coherent normative migration explicitly says otherwise.

In particular, `0.3.3-draft` does not introduce:

- a fourth architecture classification;
- a mandatory `.PTSIP/` or `.ptsip/` directory in Consumer Repositories;
- shared SQLite through Git;
- continuous authority polling;
- automatic LLM architecture classification;
- a waiver path for mandatory conformance failures;
- GitHub as a universal requirement for PTSIP;
- a requirement that Specification and Reference Tool versions remain numerically equal.

## 18. Required normative migration assets

Before `0.3.3-draft` can become an active Specification binding, one coherent migration must update all affected normative and machine-readable assets together, including at minimum:

- `spec/PTSIP-SPEC.md`;
- `spec/PTSIP-CONFORMANCE.md` where decision facts affect evaluability or mandatory rules;
- `spec/PTSIP-TERMINOLOGY.md` for Decision Authority, coordination domain, action-time synchronization, and reconciliation terminology;
- `schemas/ptsip-profile.schema.json`;
- any new decision-authority/interoperability schema accepted as normative;
- `registry/ptsip-registry.yaml`;
- `agents/AGENT-CONTRACT.md`;
- reference/adoption documentation where Specification wording changes implementation guidance;
- embedded `src/ptsip/specdata/*` resources;
- Tool Specification-family/version/revision binding constants;
- an ADR recording the normative transition from `0.2.0-draft` to `0.3.3-draft`.

The migration commit itself becomes the first immutable normative snapshot of the `0.3.3-draft` family.

## 19. Change classification

Expected change categories for the first `0.3.3-draft` normative snapshot are:

- `CLARIFICATION` — distinguish Project Profile state from decision workflow state;
- `NORMATIVE_ADDITION` — Decision Authority, distributed first-winner, reconciliation, explicit adoption, and fail-closed semantics;
- `SCHEMA_CHANGE` — durable `runtime_required` and resolved lifecycle-ownership representation;
- `CONFORMANCE_CHANGE` — only where durable decision facts alter profile validity or applicable mandatory-rule evaluation;
- `NORMATIVE_BREAKING` — only for changes that cannot preserve predecessor profile meaning and are explicitly documented.

Existing rule IDs MUST NOT be repurposed for incompatible semantics. New rule IDs, if required, should be assigned only in the coherent normative migration/ADR rather than invented by this release-note draft.

## 20. Historical compatibility and Tool binding

Existing Tools remain bound to the immutable Specification revision recorded by those Tools.

Reference Tool `0.3.3` was implemented and verified while still bound to:

```text
Specification family: 0.2.0-draft
Specification revision: a877b2f66a7f94c1b844c979e1b08fb08a9a8e45
```

Creating this `0.3.3-draft` release-note document does not retroactively change that historical binding.

A later Tool build or release that claims `0.3.3-draft` conformance MUST bind the immutable revision produced by the completed normative migration and MUST package matching normative resources.

## 21. Acceptance criteria for the first 0.3.3-draft snapshot

The first `0.3.3-draft` normative snapshot is ready only when all of the following are true:

- every changed normative asset agrees on Specification family `0.3.3-draft`;
- the immutable migration revision is recorded consistently;
- exactly three architecture classifications remain;
- Decision Authority and Project Profile responsibilities are explicitly distinct;
- local operational state is not treated as distributed architectural authority;
- distributed same-scope decisions have deterministic identity across environments;
- stale writers cannot replace an accepted winner;
- distributed coordination failure cannot silently create a local split-brain winner;
- resolved remote decisions can be reconciled without pretending every clone already applied them;
- Project Profile schema has explicit semantics for `runtime_required`;
- lifecycle ownership has a canonical durable representation;
- adoption cannot infer architectural ownership from names alone;
- coding-agent behavior is synchronized with the new authority semantics;
- predecessor `0.2.0-draft` immutable bindings remain historically reproducible;
- complete schema, registry, embedded-resource, CLI, and conformance regression tests pass against the new immutable snapshot.

Until these conditions are met, `0.2.0-draft` revision `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45` remains the active normative baseline.