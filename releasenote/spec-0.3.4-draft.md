# PTSIP Specification 0.3.4-draft Release Notes

**Status:** Proposed experimental Specification family  
**Design predecessor:** `0.3.3-draft`  
**Active normative baseline:** `0.2.0-draft` revision `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`  
**Implementation evidence:** PTSIP Reference Tool `0.3.4`  
**Tool implementation merge:** `555c528593f700a348d8da84545a62ce61291cae`  
**Tool verification completion:** `8cd0ddf16dc9b56f27f694138a37caae1c49bb4f`  
**Candidate normative snapshot:** To be assigned only by a coherent normative migration commit  
**Identity model:** draft family label + immutable Git revision

The `0.3.4-draft` label is a proposed PTSIP Specification family that incorporates the Explicit Project Adoption design work recorded in `0.3.3-draft` and adds the completed **Distributed Authority Consistency** contract demonstrated by Reference Tool `0.3.4`.

This document is a release-note-level Specification design record. It does not by itself activate `0.3.4-draft`, rebind a Tool, publish a Tool release, or make every current Reference Tool implementation detail normative.

Reference Tool `0.3.4` was implemented and verified while still bound to the active immutable Specification baseline:

```text
Specification family: 0.2.0-draft
Specification revision: a877b2f66a7f94c1b844c979e1b08fb08a9a8e45
```

Therefore the completed Tool behavior is **implementation evidence** for this draft, not a retroactive Specification binding.

`0.3.3-draft` is the design predecessor, but it has not become an active normative snapshot. A future coherent `0.3.4-draft` migration MAY incorporate the accepted `0.3.3-draft` adoption/schema proposals directly from the active `0.2.0-draft` baseline. Activating `0.3.3-draft` first is not a prerequisite.

## 1. Compatibility baseline

Unless explicitly changed by the final coherent normative migration, `0.3.4-draft` preserves the established `0.2.0-draft` architecture and conformance model.

The following invariants remain unchanged:

- PTSIP has exactly three architecture classifications: `PRODUCT`, `TOOLCHAIN`, and `NEUTRAL_CONTRACT`;
- `UNKNOWN`, `CONFLICT`, and `INCOMPLETE` remain decision/evaluation states rather than additional architecture planes;
- a Project Profile is project-owned architecture declaration state and is not itself observed conformance truth;
- declaration and observation remain distinct;
- Product Artifact owner and artifact producer remain distinct concepts;
- project-specific dependency policy may strengthen but may not weaken universal PTSIP rules;
- mandatory-rule violations are not waived by project governance metadata;
- External PTSIP Tooling remains non-intrusive toward Consumer Repository structure;
- a coding agent must not invent missing architecture intent;
- Specification and Reference Tool release identities remain independently versioned;
- immutable Git revision remains the exact identity of a normative draft snapshot.

This family does not create a fourth plane and does not redefine evidence, artifact, dependency, conformance, or diagnostic semantics merely because distributed decision coordination is added.

## 2. Design lineage from 0.3.3-draft

`0.3.3-draft` narrowed its scope to Explicit Project Adoption and durable architecture declaration semantics after the early GitHub authority implementation was correctly recognized as incomplete.

That design record established or proposed the following concepts that `0.3.4-draft` carries forward unless a later normative migration explicitly changes them:

- discovered repository scope is not architecture authority;
- project-owner architectural intent must be explicit;
- adoption must validate before mutation;
- stale or conflicting Project Profile mutation must be refused;
- Project Profile identity/path must remain consistent across one workflow;
- local Tool-owned operational state must not replace durable project architecture declaration;
- `runtime_required` requires a durable, lossless representation if it remains a required classification fact;
- lifecycle ownership used by decision validation requires a durable, lossless Project Profile representation;
- Tool implementation behavior does not become normative solely because it exists in source.

The principal addition in `0.3.4-draft` is that distributed coordination is now sufficiently complete to define a backend-neutral consistency contract.

## 3. Why Distributed Authority Consistency belongs in the Specification

A Consumer Repository may be acted on concurrently by:

- local IDE agents;
- local command-line agents;
- cloud coding agents;
- CI agents;
- multiple clones or worktrees;
- hosted decision-control services.

Separate local operational databases cannot establish one shared answer for the same unresolved architecture scope across those environments.

Git propagation of `ptsip.yaml` is also insufficient as the only coordination mechanism because a valid architecture decision may be accepted before the resulting Project Profile change is committed, pushed, fetched, merged, or observed by another environment.

PTSIP therefore distinguishes:

```text
PTSIP Specification
    -> normative architecture and conformance contract

Consumer Repository Project Profile
    -> project-owned, revision-bound architecture declaration

Decision Authority
    -> coordination state for unresolved/resolved architecture decisions
```

The roles MUST remain distinct.

Distributed Authority Consistency means that participating environments within one coordination domain observe one authoritative decision winner at architecture-sensitive operation boundaries without requiring immediate Project Profile propagation between clones.

## 4. Decision Authority

A **Decision Authority** is the authority used to coordinate unresolved and resolved architecture decisions for a defined coordination domain.

A Decision Authority does not replace the Project Profile.

```text
Decision Authority
    -> which explicit architecture answer won for a coordinated decision

Project Profile
    -> which architecture declaration is represented by a repository revision/worktree
```

An implementation MAY provide local, repository-distributed, or hosted authority backends.

A backend claiming distributed coordination MUST satisfy the consistency requirements in this draft regardless of its storage technology.

Local SQLite, a Git ref, a transactional service, or another persistence mechanism is an implementation choice unless separately standardized by an interoperability schema.

## 5. Coordination domain

A **coordination domain** is the scope within which one architecture decision identity has one authoritative winner.

A conforming distributed implementation MUST identify its coordination domain unambiguously for an architecture-decision operation.

For repository-scoped coordination, repository identity is normally part of the domain identity.

An implementation MUST NOT silently switch from a selected distributed coordination domain to an isolated local domain when the distributed backend becomes unavailable.

Such a switch could create two independently valid winners for what participants believe is the same architecture decision.

## 6. Stable distributed decision identity

The same architectural component scope MUST map to the same distributed decision identity across participating clones and environments.

Distributed decision identity MUST NOT depend solely on clone-local or temporary values such as:

- local clarification IDs;
- branch-local missing-field lists;
- temporary local Project Profile completeness;
- Local DecisionStore record IDs;
- one process's operation identifier;
- incidental candidate display names when the normalized architecture scope is otherwise the same.

The identity MUST be derived from stable coordination-domain identity plus a deterministic normalized architecture/component scope.

The exact hash or string encoding is an interoperability concern rather than universal architecture semantics unless separately standardized.

## 7. Authority revision and ordered state

A distributed Decision Authority MUST expose state strongly enough for participants to distinguish the authority state they read from a later authority state.

The implementation MAY represent this as:

- an immutable Git commit/ref revision;
- a database transaction/version;
- an ETag or generation number;
- a consensus log index;
- another ordered conditional-write token.

Wall-clock time alone SHOULD NOT determine the winning decision when a stronger ordered authority revision is available.

The purpose of the authority revision is to prevent a stale participant from replacing a newer accepted decision merely because it completed later.

## 8. First-valid-resolution-wins

For one distributed decision identity, the first valid accepted resolution wins.

A later contradictory resolution MUST NOT silently replace an already accepted authoritative winner.

A distributed authority mutation MUST use atomic conditional-write, compare-and-swap, transaction, consensus, or equivalent semantics that prevent a stale writer from overwriting a newer state.

Conceptually:

```text
read authority revision A
        |
        v
prepare valid mutation based on A
        |
        v
conditionally publish A -> B

if authority no longer equals A
        |
        v
reject stale write
        |
        v
reread current authority
        |
        +-- same-scope winner already resolved
        |       -> accept winner; do not retry contradictory answer
        |
        +-- unrelated authority mutation only
                -> retry/rebase only when safe
```

Concurrent independent decisions for different component scopes MAY both eventually succeed after safe retry against the latest authority state.

Concurrent contradictory decisions for the same scope MUST converge to one accepted winner.

## 9. Authority freshness at architecture-sensitive boundaries

Distributed write serialization alone is insufficient.

When an architecture-sensitive operation is using distributed coordination, the implementation MUST account for the relevant current authority state before returning a result that assumes the local Project Profile is authoritative for that coordinated scope.

A complete local declaration MUST NOT automatically cause an early success result when a relevant distributed authority record may already exist.

Conceptually:

```text
analyze local repository/profile
        |
        v
resolve relevant coordination scope
        |
        v
read current distributed authority state
        |
        v
compare authority with local declaration
        |
        v
return final gate/reconciliation result
```

This is the read-side complement to first-valid-resolution-wins.

Without authority freshness, a stale clone could contain a complete but obsolete Project Profile and incorrectly continue without observing the repository-global winner.

## 10. Read-only authority observation MUST NOT fabricate history

Checking whether distributed authority state exists SHOULD be possible without creating new decision history.

A read-only authority observation MUST NOT bootstrap a new distributed authority record, pending decision, branch, database row, or equivalent state merely to prove that no prior coordinated decision exists, unless the backend fundamentally cannot perform non-mutating observation and that limitation is explicitly part of the backend contract.

For the Reference Tool GitHub profile, authority `peek` is read-only and absence of `refs/heads/ptsip-policy` does not create the ref.

The Specification-level principle is:

> absence checking is not itself an architecture decision mutation.

## 11. Project Profile and authority reconciliation

A distributed implementation MUST define deterministic reconciliation semantics between the selected local Project Profile and the relevant authority state.

At minimum the following cases must be distinguished:

| Local Project Profile | Distributed Authority | Required semantics |
| --- | --- | --- |
| declaration absent | no decision | create/reuse a pending decision only when the active operation actually requires one |
| declaration absent | resolved winner | validate and safely project/reconcile the winner locally |
| declaration present | no authority decision | use the local project declaration and do not fabricate authority history solely for bookkeeping |
| declaration present and semantically equivalent | resolved equivalent winner | report consistency/resolution without rewriting merely equivalent profile text |
| declaration present and semantically conflicting | resolved different winner | expose an authority/profile conflict and do not silently overwrite either side |
| local repository/profile changed during reconciliation | any remote state | refuse stale application and require re-analysis |

The exact result names are interoperability details unless standardized separately, but these semantic distinctions are required.

## 12. Semantic equivalence

Authority/Profile reconciliation MUST compare architecture meaning rather than incidental serialization formatting.

Equivalent declarations MUST NOT become conflicts solely because of differences such as:

- YAML formatting;
- key ordering;
- insignificant whitespace;
- Tool-generated versus manually formatted equivalent content.

Equivalence MUST be based on the normalized architecture facts relevant to the coordinated component scope.

A Tool MAY preserve additional non-conflicting Project Profile metadata when applying or comparing a coordinated answer.

## 13. Missing local declaration + resolved authority winner

When the local Project Profile lacks the relevant declaration but the distributed authority already contains a valid resolved winner, a conforming implementation MAY reconcile the winner into the selected local Project Profile.

Automatic local projection MUST occur only after equivalent safeguards for:

1. repository/coordination scope identity;
2. authoritative answer validation;
3. repository evidence freshness;
4. current Project Profile freshness;
5. projected profile validity;
6. concurrent profile-content protection;
7. atomic or safely replaceable local write;
8. explicit operation result reporting.

The authoritative classification answer MUST be preserved exactly.

A local projection is not a new architecture decision.

## 14. Equivalent local declaration + resolved authority winner

When the selected local Project Profile already expresses the same architecture meaning as the resolved authority winner, the implementation SHOULD report a consistent/resolved state without rewriting the Project Profile solely to demonstrate synchronization.

The operation MAY record a clone-local reconciliation receipt.

Equivalent local content does not need to be replaced by Tool-generated formatting.

## 15. Conflicting local declaration + resolved authority winner

When a resolved distributed winner conflicts with an existing local Project Profile declaration for the same coordinated scope, the implementation MUST NOT silently overwrite or reclassify the Project Profile.

It MUST expose an explicit authority/profile conflict state and stop any affected architecture-sensitive operation that requires one unambiguous coordinated answer.

The conflict result SHOULD make clear, when safe and machine-readable, that:

- a distributed winner exists;
- the local declaration differs semantically;
- no automatic overwrite occurred;
- explicit reconciliation is required.

Neither "local file wins because it is complete" nor "remote winner overwrites the file automatically" is a generally safe default for an existing conflict.

## 16. Local declaration + no distributed decision

A complete local Project Profile declaration remains meaningful project-owned architecture state when no distributed decision exists for that scope.

A distributed-capable Tool MUST NOT fabricate a pending or resolved authority record solely because the repository already contains a valid declaration and the Tool wants complete historical bookkeeping.

This preserves the distinction between:

```text
architecture declaration
    !=
decision workflow history
```

Distributed authority is used when coordination is required; it is not mandatory retroactive history for every pre-existing Project Profile declaration.

## 17. Global decision state and local projection state are different

A distributed authority owns global decision state such as:

```text
PENDING
RESOLVED
```

A clone/worktree may separately have local projection/application state such as:

```text
not yet projected
locally applied
stale
failed
consistent without rewrite
```

A global `RESOLVED` state MUST NOT imply that every clone has applied the corresponding Project Profile declaration.

A single global `application_status = APPLIED` MUST NOT be interpreted as proof that every participating environment has received or written the declaration.

If application receipts are recorded, they MUST be scoped so they cannot alter or redefine which architecture decision won.

The Reference Tool `0.3.4` reports clone-local application receipts with `scope = LOCAL_PROJECTION`; that exact field spelling is an interoperability detail, while the global/local state separation is normative candidate semantics.

## 18. Fail-closed distributed coordination

When distributed coordination is selected for a new, changed, unresolved, or consistency-sensitive architecture decision and the authority cannot be safely read or mutated, the implementation MUST NOT silently fall back to an isolated Local DecisionStore winner.

Failure causes may include:

- authentication failure;
- permission failure;
- network unavailability;
- malformed authority data;
- incompatible authority ownership/manifest state;
- unsafe conditional-write failure;
- inability to establish required read freshness.

The affected coordinated architecture operation MUST fail closed with an explicit coordination error.

Existing Project Profile data MAY remain readable for operations that do not require a new coordinated authority conclusion.

Fail-closed coordination prevents split-brain first-winner state.

## 19. Action-time synchronization

`0.3.4-draft` does not require continuous background polling.

A conforming implementation MAY use **action-time synchronization**: consult the relevant distributed authority when an active architecture-sensitive operation reaches a boundary where authoritative coordinated state matters.

Correctness MUST NOT depend on every clone continuously polling or immediately receiving another clone's `ptsip.yaml` commit.

Action-time synchronization SHOULD be scoped to relevant components and operations so ordinary unrelated read-only commands are not unnecessarily converted into continuous network-dependent workflows.

## 20. Explicit Project Adoption under distributed coordination

The Explicit Project Adoption semantics inherited from `0.3.3-draft` continue to apply.

For distributed coordination, an adoption operation that represents a new shared architecture decision MUST coordinate against the selected authority before committing a contradictory local winner.

The following principles apply:

- dry-run/planning remains non-mutating;
- candidate discovery remains separate from project-owner architecture intent;
- an equivalent existing distributed winner MAY be reused;
- a different already resolved winner MUST NOT be overwritten;
- a conflicting local adoption answer MUST NOT become a second winner;
- a pre-existing complete local declaration with no distributed decision does not require fabricated authority history solely for bookkeeping;
- local Project Profile mutation remains subject to stale-evidence and concurrent-content protection.

The Reference Tool command `ptsip adopt` is one implementation of this concept.

## 21. Explicit resolution under distributed coordination

A coding-agent or Tool resolution operation MUST record explicit project-owner/human architecture facts rather than infer missing intent.

When distributed coordination is selected, resolution MUST:

- operate on the stable distributed decision identity for the component scope;
- read the current authoritative decision state;
- reject replacement of a different already resolved winner;
- serialize a pending resolution through safe conditional mutation;
- use the accepted winner for subsequent local reconciliation;
- avoid creating distinct global decisions from clone-local branch/revision metadata when the normalized coordinated architecture scope is the same.

First-valid-resolution-wins is a coordination rule, not permission for a Tool to invent the first answer.

## 22. Authority backend selection and visibility

The Specification does not require one universal backend-selection CLI algorithm.

A conforming Tool MUST nevertheless make the selected authority mode unambiguous to the operation and SHOULD expose it in machine-readable results where coordination behavior differs.

Possible authority scopes include:

```text
LOCAL
DISTRIBUTED_REPOSITORY
HOSTED_CONTROL_PLANE
```

An implementation MAY support additional backends if their consistency semantics are documented.

Selection of a distributed authority MUST NOT degrade silently to an isolated local authority after failure.

## 23. GitHub-coordinated Reference Tool profile

Reference Tool `0.3.4` demonstrates one concrete distributed authority implementation for GitHub repositories.

Its current profile uses:

```text
refs/heads/ptsip-policy

  authority.json
  decisions/<global-decision-id>.json
```

with a global `gdec-*` identity derived from repository identity plus normalized component include scope and conditional non-force Git ref updates for write serialization.

Read-side authority freshness uses a non-mutating lookup path when checking for existing authority state.

This GitHub representation is implementation/interoperability evidence, not a universal PTSIP requirement.

`0.3.4-draft` does NOT require every conforming implementation to use:

- GitHub;
- `refs/heads/ptsip-policy`;
- `authority.json`;
- `gdec-*` IDs;
- GitHub REST APIs;
- Git refs as a database.

Another backend may conform if it satisfies the same stable identity, ordered-state, conditional mutation, freshness, reconciliation, global/local state separation, and fail-closed requirements.

## 24. GitHub Projects, Issues, and human interfaces

A dashboard, Issue, Project board, chat interface, or similar human interaction surface MAY display or collect decision information.

Such a surface is not authoritative merely because users can see or edit it.

If a human interface is used as part of the authority backend, the underlying implementation must still satisfy the same atomic winner, freshness, identity, and conflict requirements.

Presentation state MUST NOT silently override the authoritative decision record.

## 25. Durable architecture facts carried from 0.3.3-draft

The design work from `0.3.3-draft` identified a representation gap between structured decision/adoption facts and the predecessor Project Profile schema.

The candidate fact set remains:

- `classification`;
- `purpose`;
- `shipped`;
- `runtime_required`;
- `lifecycle_owner`;
- `executable`.

If these facts remain required for normative classification/adoption validation, the final `0.3.4-draft` schema migration MUST preserve them losslessly in durable Project Profile semantics.

In particular:

- `runtime_required` requires explicit durable semantics rather than hidden Tool-only state;
- `TOOLCHAIN` MUST NOT be declared as Product-runtime-required;
- lifecycle ownership must have one canonical field or a lossless mapping to existing profile fields;
- legacy absence semantics must be explicit and MUST NOT be guessed when strict evaluation depends on the fact.

This remains a `SCHEMA_CHANGE` candidate and may affect profile-validation or conformance evaluability.

## 26. Candidate architecture-fact validation relationships

The following relationships carried from `0.3.3-draft` remain candidates for coherent normative adoption:

- `PRODUCT` requires Product lifecycle ownership;
- `TOOLCHAIN` requires development-tooling lifecycle ownership;
- `TOOLCHAIN` cannot be shipped as part of the Product;
- `TOOLCHAIN` cannot be required by the Product at runtime;
- `NEUTRAL_CONTRACT` must be non-executable;
- `NEUTRAL_CONTRACT` requires independent lifecycle ownership.

The final normative migration MUST reconcile these relationships with existing Product Artifact, dependency, lifecycle, neutrality, and coherent-component rules.

A parallel classification rule system MUST NOT be created merely because decision coordination uses structured answers.

## 27. Coding-agent contract implications

A coding agent operating under the proposed `0.3.4-draft` semantics MUST:

- inspect and preserve the selected Project Profile rather than invent missing architecture intent;
- distinguish observed repository scope from project-owner architecture intent;
- consult the selected distributed authority when an architecture-sensitive coordinated operation requires authority freshness;
- accept an already authoritative winner rather than ask the user to create a contradictory second winner;
- not treat a complete but stale local declaration as sufficient when relevant distributed authority state must be checked;
- not silently overwrite a conflicting local declaration with a remote winner;
- not create a separate local winner when distributed coordination is unavailable;
- distinguish global decision resolution from clone-local profile projection;
- refuse stale local application when repository/profile state changes during reconciliation.

The agent remains an executor of explicit architecture decisions, not the architecture authority.

## 28. Machine-readable result expectations

Distributed authority interfaces SHOULD expose stable machine-readable distinctions sufficient for coding agents, CI, and orchestration to distinguish at least:

- no decision required;
- decision required;
- resolved authoritative winner available;
- already resolved by another winner;
- authority/profile conflict;
- stale evidence or stale local application;
- coordination unavailable;
- local projection/application failure;
- consistent equivalent local/remote state.

Reference Tool `0.3.4` currently uses states including:

```text
NO_DECISION_REQUIRED
DECISION_REQUIRED
RESOLVED
ALREADY_RESOLVED
AUTHORITY_PROFILE_CONFLICT
STALE_EVIDENCE
COORDINATION_UNAVAILABLE
DECISION_ERROR
```

and reconciliation/application states such as `CONSISTENT` and `LOCAL_PROJECTION`.

The semantic distinctions are Specification candidates. Exact strings and JSON envelope shapes remain interoperability-schema details unless separately standardized.

## 29. Preserved predecessor semantics

Except for the explicit additions above, `0.3.4-draft` preserves predecessor architecture principles including:

- exactly three architecture classifications;
- coherent component boundaries;
- deterministic selector precedence;
- declaration versus observation separation;
- evidence provenance and rule-relative coverage;
- Product Artifact owner/producer distinction;
- dependency edge and lifecycle-phase semantics;
- Product-to-Toolchain runtime prohibition;
- lifecycle independence requirements;
- Neutral Contract non-executable/independent semantics;
- Project Profile validation versus conformance evaluation;
- stable diagnostic identity;
- optional stricter project dependency policy;
- mandatory-rule remediation without waiver;
- Consumer Repository non-intrusion;
- immutable Specification binding.

Distributed authority coordinates architecture decisions; it does not change what `PRODUCT`, `TOOLCHAIN`, or `NEUTRAL_CONTRACT` mean.

## 30. Explicit non-goals

`0.3.4-draft` does not introduce or require:

- a fourth architecture classification;
- automatic or unconstrained LLM architecture classification;
- directory-name-based ownership inference;
- shared SQLite through Git;
- a mandatory `.PTSIP/` or `.ptsip/` Consumer Repository directory;
- continuous background authority polling;
- GitHub as a universal PTSIP dependency;
- automatic overwrite/reclassification of an existing conflicting Project Profile;
- one global application flag pretending every clone is synchronized;
- fabricated distributed decision history for every pre-existing local declaration;
- mandatory conformance waivers;
- organization-wide profile composition;
- unrelated evidence-adapter expansion;
- a requirement that Specification and Tool version numbers match.

## 31. Required normative migration assets

Before `0.3.4-draft` can become an active Specification binding, one coherent normative migration must update all affected normative and machine-readable assets together.

Expected assets include at minimum:

- `spec/PTSIP-SPEC.md`;
- `spec/PTSIP-CONFORMANCE.md` where durable facts affect validation/evaluability;
- `spec/PTSIP-TERMINOLOGY.md` for Decision Authority, coordination domain, authority revision, authority freshness, reconciliation, and local projection terminology;
- `schemas/ptsip-profile.schema.json` for accepted durable architecture facts;
- any accepted decision-authority/interoperability schema;
- `registry/ptsip-registry.yaml` where rule/schema metadata changes;
- `agents/AGENT-CONTRACT.md` for agent-visible normative authority behavior;
- reference/adoption documentation aligned with the final contract;
- embedded `src/ptsip/specdata/*` resources;
- an ADR recording the normative transition and its compatibility consequences;
- Tool Specification-family/version/revision binding constants only when a Tool is explicitly rebound to the completed normative snapshot.

All changed normative assets MUST agree at one immutable Git revision.

That migration revision becomes the first canonical normative snapshot of the `0.3.4-draft` family.

## 32. Change classification

Expected change categories for the first `0.3.4-draft` normative migration include:

- `CLARIFICATION` — explicit separation of Project Profile, global decision state, and local projection state;
- `NORMATIVE_ADDITION` — Decision Authority, stable distributed decision identity, authority freshness, conditional mutation, reconciliation matrix, fail-closed distributed coordination, action-time synchronization;
- `SCHEMA_CHANGE` — accepted durable `runtime_required`/lifecycle-ownership representation inherited from the adoption design work;
- `CONFORMANCE_CHANGE` — only where new durable architecture facts alter profile validity or applicability/evaluability of mandatory rules;
- `NORMATIVE_BREAKING` — only where predecessor profile meaning cannot be preserved and the incompatibility is explicitly justified.

Existing rule IDs MUST NOT be repurposed for incompatible meaning.

New rule IDs should be assigned only in the coherent normative migration/ADR rather than invented by this release-note draft.

## 33. Historical compatibility and Tool binding

Existing Tools remain bound to the immutable Specification revision recorded by those Tools.

Reference Tool `0.3.4` completed and verified Distributed Authority Consistency while remaining bound to:

```text
0.2.0-draft
a877b2f66a7f94c1b844c979e1b08fb08a9a8e45
```

Creating this `0.3.4-draft` document does not retroactively change Tool `0.3.4` behavior or binding.

Likewise, `0.3.3-draft` remains a design predecessor rather than a required historical active binding.

A later Tool release that claims `0.3.4-draft` conformance MUST bind the immutable revision produced by the completed normative migration and MUST package matching normative resources.

## 34. Reference Tool 0.3.4 implementation evidence

Reference Tool `0.3.4` provides concrete evidence that the proposed distributed consistency contract is implementable without continuous polling or shared SQLite.

The completed implementation demonstrates:

- repository-global first-winner protection through conditional Git ref mutation;
- stable repository/component-scope decision identity;
- gate-time authority freshness even when a local Project Profile is already complete;
- read-only absence lookup that does not bootstrap authority history;
- semantic equivalent/missing/conflicting reconciliation;
- explicit `AUTHORITY_PROFILE_CONFLICT` without silent overwrite;
- fail-closed `COORDINATION_UNAVAILABLE` behavior without Local DecisionStore fallback;
- safe stale-profile/evidence refusal;
- clone-local projection receipts distinct from global decision state;
- coordinated `adopt` and `resolve` behavior;
- preservation of explicit Local and hosted control-plane backends;
- no continuous background polling.

Final Tool verification completed in GitHub Actions run `31471025526` on Python `3.14.6` with:

- `134 passed` in the complete pytest suite;
- Tool identity `PTSIP Tool 0.3.4` verified;
- exact bound Specification identity verified;
- package build passed;
- wheel and sdist `twine check` passed;
- installed-wheel smoke checks passed.

This verification supports the feasibility of the Specification design. It does not itself make the implementation normative.

## 35. Acceptance criteria for the first 0.3.4-draft normative snapshot

The first `0.3.4-draft` normative snapshot is ready only when all of the following are true:

1. every changed normative asset identifies the same `0.3.4-draft` family and immutable revision;
2. exactly three architecture classifications remain;
3. Explicit Project Adoption semantics are incorporated coherently or explicitly superseded;
4. durable architecture facts required by normative decision validation have lossless Project Profile representation;
5. Decision Authority and Project Profile responsibilities are explicitly distinct;
6. coordination-domain and distributed decision identity rules are deterministic enough for independent implementations;
7. stale writers cannot replace an accepted same-scope winner;
8. a distributed coordinated read cannot return success without accounting for relevant existing authority state;
9. read-only absence checks do not fabricate decision history;
10. missing/equivalent/conflicting local/remote states have deterministic reconciliation semantics;
11. semantic equivalence is independent of incidental profile formatting;
12. conflicting local/remote architecture is never silently overwritten;
13. distributed authority failure cannot silently create a separate Local winner;
14. global decision state is distinct from clone-local profile application state;
15. a global resolution does not imply every clone has applied the declaration;
16. action-time synchronization is sufficient; continuous polling is not required;
17. existing complete local declarations with no distributed decision remain valid without fabricated authority history;
18. coding agents cannot invent missing architecture intent or create contradictory second winners;
19. GitHub-specific storage details remain reference-profile details unless separately standardized;
20. compatibility and migration consequences from `0.2.0-draft` and the `0.3.3-draft` design work are documented;
21. the final normative merge commit is recorded as the first immutable `0.3.4-draft` snapshot.

## 36. Activation boundary

The current state after creating this document is intentionally:

```text
Active Specification:
    0.2.0-draft @ a877b2f66a7f94c1b844c979e1b08fb08a9a8e45

Design predecessor:
    0.3.3-draft (proposed, not active)

New Specification design record:
    0.3.4-draft (proposed, not active)

Reference Tool source:
    0.3.4

Distributed Authority Consistency implementation:
    completed and verified

Tool 0.3.4 Specification binding:
    still 0.2.0-draft @ a877b2f66a7f94c1b844c979e1b08fb08a9a8e45
```

The next Specification step is not another Tool implementation merely to justify this draft. The next step is a deliberate coherent normative migration that decides the final rule text, schema changes, terminology, registry entries, agent contract, interoperability surfaces, compatibility behavior, and immutable binding revision.
