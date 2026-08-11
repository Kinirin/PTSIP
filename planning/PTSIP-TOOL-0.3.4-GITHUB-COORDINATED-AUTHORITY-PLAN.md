# PTSIP Tool 0.3.4 — GitHub-Coordinated Authority Plan

> **Status:** APPROVED / PLANNING  
> **Target Tool version:** `0.3.4`  
> **Planning baseline:** `f49fd391ff8237360d9fe5ebc8438eca5bfb4076`  
> **Predecessor Tool source:** `0.3.3` source-only migration  
> **Current active Tool binding:** `0.2.0-draft` revision `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`  
> **Specification note:** the proposed `0.3.3-draft` Specification family is a separate workstream and does not automatically rebind Tool `0.3.4`.  
> **Release lineage:** Tool `0.3.3` is permanently untagged/unpublished; the next Tool release candidate begins at `0.3.4`.

## 1. Why Tool 0.3.4 exists

Tool `0.3.3` completed the explicit project-adoption workstream and profile-path symmetry. During that source migration, the project also explored GitHub-coordinated decision authority and merged an initial implementation.

The original Tool `0.3.3` GitHub authority amendment is withdrawn from the `0.3.3` completion scope because the implementation does not yet establish the complete repository-global consistency contract.

The initial source already demonstrates useful write-side primitives:

- repository-scoped GitHub authority state;
- `refs/heads/ptsip-policy` bootstrap;
- authority-manifest ownership checks;
- stable `gdec-*` component-scope identity;
- non-force Git ref compare-and-swap;
- first-winner protection for concurrent contradictory writes;
- basic stale-clone projection when the local declaration is missing;
- fail-closed behavior instead of silent Local SQLite fallback when GitHub coordination is selected.

Those primitives are retained as a **prototype baseline**, not discarded.

Tool `0.3.4` must complete the missing read-side and reconciliation semantics before GitHub-coordinated authority is treated as a release-quality contract.

## 2. Release objective

Tool `0.3.4` has one primary release objective:

> **Distributed Authority Consistency** — a GitHub-coordinated repository must observe one repository-global architecture-decision winner at architecture-sensitive operation boundaries without depending on immediate `ptsip.yaml` commit/push/fetch propagation.

The intended model is:

```text
architecture-sensitive operation
        |
        v
resolve repository/component coordination scope
        |
        v
read selected authority revision
        |
        v
compare authority state with selected Project Profile
        |
        +-- consistent -> continue
        +-- remote winner / local missing -> safe local projection
        +-- remote winner / local conflicting -> explicit conflict/reconciliation result
        +-- no winner / decision needed -> register or reuse pending decision
        +-- authority unavailable -> fail closed for coordinated decision work
```

Continuous background polling is not required.

## 3. State ownership model

Tool `0.3.4` must preserve three different state roles.

```text
PTSIP Specification
    -> normative architecture and conformance contract

Consumer Repository Project Profile
    -> project-owned, revision-bound architecture declaration

Decision Authority
    -> coordination state for unresolved/resolved architecture decisions
```

These roles must not collapse into one another.

`ptsip.yaml` remains project-owned and Git-tracked by default.

Local `control-plane.sqlite3` remains Tool-owned operational state under `PTSIP_HOME` and must not be shared through Git.

A GitHub authority is not a replacement for the Project Profile. It determines the accepted coordinated answer when a component scope requires a shared architecture decision.

## 4. Authority backend selection

The planned Tool-level selection order remains:

```text
explicit --control-plane <URL>
    -> hosted HTTP Control Plane

explicit --coordination local
    -> embedded Local DecisionStore

explicit --coordination github
    -> GitHub authority

otherwise GitHub origin present
    -> GitHub authority

otherwise
    -> Local DecisionStore
```

`--control-plane` and `--coordination` remain mutually exclusive.

Backend choice must be visible in structured results.

If GitHub coordination is selected for a new, changed, or unresolved architecture decision and the authority cannot be safely read or written, the operation must fail closed. It must not silently create a separate Local DecisionStore winner.

Ordinary read-only Project Profile operations such as validation are not automatically converted into network-dependent coordination operations unless the Tool `0.3.4` command contract explicitly says so.

## 5. GitHub authority storage contract

The GitHub reference implementation continues to use:

```text
refs/heads/ptsip-policy
```

with:

```text
authority.json

decisions/
    <global-decision-id>.json
```

The authority ref must be rejected when an existing branch with the same name does not contain a compatible PTSIP authority manifest for the same repository/ref identity.

Authority data must remain outside the normal Consumer Repository worktree.

## 6. Stable decision identity

The same architectural component scope must map to the same distributed decision identity across clones and environments.

The identity must not depend on clone-local values such as:

- clarification ID;
- branch-local missing-field list;
- temporary local Project Profile completeness;
- Local SQLite record identity.

The current prototype derives `gdec-*` from:

```text
GitHub owner/repository identity
    + normalized discovered component include selectors
```

Tool `0.3.4` must audit and regression-test this identity rule against equivalent selector spellings, multiple clones, branch divergence, and candidate-ID differences.

## 7. Write-side serialization

The GitHub authority must preserve first-valid-resolution-wins through conditional Git ref mutation:

```text
read authority HEAD A
    -> create mutation commit B with parent A
    -> update ptsip-policy A -> B with force=false
```

If the authority moved after A was read:

- the stale mutation must not overwrite the newer HEAD;
- the Tool must reread authority state;
- a contradictory resolved winner must be accepted as the winner rather than retried as pending;
- non-conflicting mutations for other decision documents may retry against the latest authority state.

Wall-clock time alone must not determine the winner when ordered authority revisions are available.

## 8. Read-side authority freshness

This is a required Tool `0.3.4` correction.

In GitHub-coordinated mode, `ptsip gate` must not conclude `NO_DECISION_REQUIRED` solely because the selected local Project Profile already contains architecture facts.

For the relevant component scope, the Tool must determine whether a distributed authority record exists and, when one exists, compare the authoritative resolved answer with the local declaration before allowing a coordination-sensitive gate to complete.

The intended ordering is:

```text
local evidence/profile analysis
        |
        v
component coordination scope
        |
        v
GitHub authority read at current authority HEAD
        |
        v
local/remote comparison
        |
        v
final gate result
```

This closes the gap where a stale clone can otherwise return `NO_DECISION_REQUIRED` without observing a newer repository-global winner.

## 9. Reconciliation matrix

Tool `0.3.4` must explicitly handle at least the following states.

| Local Project Profile | GitHub authority | Required behavior |
| --- | --- | --- |
| declaration absent | no decision | create/reuse pending decision when the active task requires one |
| declaration absent | resolved winner | safely project the winner into the selected local profile |
| declaration present | no authority decision | use the project declaration; do not fabricate a decision solely for history |
| declaration present and equivalent | resolved equivalent winner | no-op / resolved consistency result |
| declaration present and conflicting | resolved different winner | do not silently overwrite; return an explicit authority/profile conflict requiring reconciliation |
| local repository/profile changes during reconciliation | any remote state | refuse stale application and require re-analysis |

Equivalent comparison must use architecture semantics, not incidental formatting.

## 10. Conflict semantics

A local Project Profile and a resolved distributed authority may disagree because of stale Git propagation, manual profile edits, branch divergence, or earlier prototype behavior.

Tool `0.3.4` must not silently decide that either file text or authority record should overwrite the other without exposing the conflict.

A stable machine-readable result family must distinguish at least:

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

Exact naming may be finalized during implementation, but the Tool must not collapse authority/profile disagreement into ordinary `NO_DECISION_REQUIRED` or silently rewrite the Project Profile.

An explicit future reconciliation command or explicit resolution mode may be added if required; automatic reclassification is not permitted.

## 11. Action-time synchronization

Tool `0.3.4` uses action-time synchronization rather than continuous polling.

A coordinated agent checks the distributed authority at the boundary where the current task actually needs authoritative architecture state.

Correctness must not require every clone to immediately receive another clone's `ptsip.yaml` commit.

Action-time synchronization must be bounded to relevant component scopes and must not turn all PTSIP read-only commands into continuous remote synchronization clients.

## 12. Local profile projection

When a resolved authority winner exists and the selected Project Profile is missing the declaration, local projection may proceed only after:

1. confirming the repository/component scope matches the decision;
2. validating the authoritative answer with the shared semantic validator;
3. checking repository/profile evidence freshness;
4. preparing the projected Project Profile;
5. validating the projected profile;
6. preserving concurrent profile-content protection;
7. atomically writing the profile;
8. reporting a local reconciliation result.

If a conflicting declaration already exists, this automatic projection path must stop with an explicit conflict result.

## 13. Application-state model

Tool `0.3.4` must separate:

```text
GLOBAL DECISION STATE
    PENDING / RESOLVED

LOCAL PROJECTION STATE
    whether this worktree/revision has applied or reconciled the winner
```

A single global `application_status = APPLIED` must not imply that every clone has received the declaration.

The current prototype's application fields must therefore be audited.

Acceptable implementation directions include:

- keeping local projection status only in the operation result/local state; or
- storing separate application receipts keyed by an explicit repository revision or other non-authoritative application identity.

Application tracking must never alter which architecture answer won.

## 14. `ptsip adopt` integration

`ptsip adopt` remains the explicit project-owner architecture-declaration path introduced in Tool `0.3.3`.

For a non-GitHub/local-only repository, adoption may apply directly after normal validation without fabricating a Local DecisionStore record.

For GitHub-coordinated adoption, `--apply` must coordinate against the distributed authority before local mutation when the operation represents a new shared architecture decision.

If another environment already resolved the same component scope:

- an equivalent winner may be reused;
- a different winner must not be overwritten;
- local conflicting input must not be applied as a second winner.

Dry-run remains read-only.

## 15. `ptsip resolve` integration

`ptsip resolve` must continue to record only explicit project-owner/user facts.

In GitHub-coordinated mode it must:

- read the current authoritative decision record;
- reject a different already-resolved winner;
- serialize a pending resolution through CAS;
- use the accepted winner for subsequent local projection;
- never treat clone-local branch/revision metadata as a separate global decision identity.

Local and explicit hosted backends retain their own documented scope/semantics.

## 16. Authentication and failure boundary

GitHub coordination may authenticate through:

- `GH_TOKEN`;
- `GITHUB_TOKEN`;
- authenticated `gh` CLI for interactive environments.

The credential must have the repository permissions required by the authority implementation.

Authentication, network, malformed authority state, incompatible authority manifest, or unsafe ref-update failures must produce a coordination error rather than silent Local fallback.

The Tool must not partially mutate the local Project Profile after a failed authoritative decision operation.

## 17. Existing prototype audit

The current `main` source already contains an early GitHub authority implementation. Tool `0.3.4` begins by auditing rather than blindly rewriting it.

At minimum audit:

- `src/ptsip/app/github_authority.py`;
- CLI backend-selection and gate ordering;
- `adopt` coordinated application;
- `resolve` coordinated resolution;
- authority answer parsing/validation;
- manifest/bootstrap safety;
- CAS retry behavior;
- application-state fields;
- stale-clone tests;
- failure/exception JSON behavior where it affects the new consistency contract.

Existing correct write-side code should be retained when it satisfies the finalized Tool `0.3.4` contract.

## 18. Required verification scenarios

Tool `0.3.4` is not complete until tests cover at least:

1. two clones derive the same global decision ID for the same normalized include scope;
2. two concurrent contradictory resolutions converge to one winner;
3. two concurrent independent component decisions can both eventually succeed;
4. an existing non-PTSIP `ptsip-policy` branch is refused;
5. GitHub authority unavailability produces no Local fallback winner;
6. local declaration missing + remote winner safely projects locally;
7. local declaration equivalent + remote winner returns a consistent no-op/resolved result;
8. local declaration conflicting + remote winner returns explicit conflict without silent overwrite;
9. a stale clone with a complete but obsolete local declaration still checks authority at gate time;
10. repository/profile mutation during reconciliation causes stale refusal;
11. GitHub adoption reuses an equivalent existing winner;
12. GitHub adoption refuses a different existing winner;
13. GitHub resolve cannot replace an accepted winner;
14. application-state reporting does not imply global application across clones;
15. Local DecisionStore mode remains operational when explicitly/local-only selected;
16. hosted HTTP Control Plane mode remains operational when explicitly selected;
17. no continuous polling/background reminder is introduced;
18. complete pytest, CLI smoke, package build, installed-wheel smoke, and `twine check` pass before publication consideration.

## 19. Implementation sessions

### Session 1 — Authority-read audit and result contract

- audit current prototype behavior;
- define coordinated gate ordering;
- define authority/profile comparison semantics;
- define machine-readable conflict and unavailable states;
- add regression tests that reproduce the stale-complete-profile gap.

Completion boundary:

```text
a GitHub-coordinated gate cannot return success without accounting for
an existing relevant authority winner.
```

### Session 2 — Reconciliation and application-state correction

- implement missing/equivalent/conflicting reconciliation matrix;
- preserve stale-evidence/concurrent-profile guards;
- correct the global-vs-local application-state model;
- verify multi-clone projection behavior.

Completion boundary:

```text
remote winner + stale local state converges safely or fails explicitly;
no silent overwrite occurs.
```

### Session 3 — Adopt/resolve/backend integration

- align `adopt` and `resolve` with the finalized authority contract;
- preserve explicit local/hosted backend selection;
- verify fail-closed behavior;
- audit authentication and manifest ownership errors.

Completion boundary:

```text
all write-enabled coordinated architecture operations share one
first-winner authority model.
```

### Session 4 — Tool 0.3.4 identity and documentation

- bump Tool/package identity to `0.3.4` only when implementation scope is coherent;
- add `releasenote/0.3.4.md`;
- align README/reference/adoption/agent-facing documentation;
- state the final Tool/Specification binding explicitly;
- preserve the permanent Tool `0.3.3` no-tag/no-publication record.

### Session 5 — Release-boundary verification

Run the complete release-readiness suite and disposable multi-clone GitHub-authority scenarios before any `tool-v0.3.4` tag, GitHub Release, or PyPI publication is created.

## 20. Explicit non-goals

Tool `0.3.4` does not automatically include:

- continuous authority polling;
- GitHub Projects as an authority database;
- shared SQLite through Git;
- a fourth PTSIP architecture classification;
- automatic/LLM architecture classification;
- automatic overwrite/reclassification of a conflicting Project Profile;
- a mandatory `.PTSIP/` or `.ptsip/` Consumer Repository directory;
- silent binding to the proposed `0.3.3-draft` Specification family;
- global gate exception-envelope redesign beyond what is required to make the new authority consistency states machine-readable;
- unrelated conformance-rule expansion.

## 21. Release acceptance criteria

Tool `0.3.4` may be considered a publication candidate only when all of the following are true:

1. repository-global first-winner writes remain CAS-safe;
2. coordinated reads observe relevant authority state even when the local profile already contains a declaration;
3. missing/equivalent/conflicting local declarations have deterministic reconciliation behavior;
4. conflicting local/remote architecture state is never silently overwritten;
5. distributed-authority failure cannot create a second local winner;
6. global decision resolution is distinct from clone-local profile application;
7. Local and hosted backends retain explicit supported behavior;
8. `adopt`, `gate`, and `resolve` share the finalized authority semantics;
9. the complete regression/release-readiness suite passes;
10. Tool/Specification binding is explicitly recorded without silently adopting a proposed Specification family;
11. Tool `0.3.3` remains permanently without tag, GitHub Release, or PyPI publication.

## 22. Publication lineage

The Tool publication sequence intentionally skips `0.3.2` and `0.3.3` as source-only migration versions.

```text
published: Tool 0.3.1
source-only: Tool 0.3.2
source-only, permanently untagged: Tool 0.3.3
next publication candidate: Tool 0.3.4
```

No `tool-v0.3.4` tag, GitHub Release, or PyPI publication is implied by this planning document. Publication occurs only after the Tool `0.3.4` acceptance criteria and explicit release decision are satisfied.