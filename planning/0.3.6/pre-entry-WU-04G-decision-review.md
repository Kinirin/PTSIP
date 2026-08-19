# WU-04G Pre-Entry Decision Review — Clarification / Adoption Effective-Map Integration

> **Status:** PRE-ENTRY REVIEW ONLY — WU-04G HAS NOT BEEN ENTERED  
> **Purpose:** expose implementation options and decision points before the maintainer authorizes the WU-04G stage  
> **Review baseline:** `fe2a0a77f64641d48ee1bb453e59cba1d8790692`  
> **Predecessor state:** WU-04E and WU-04F are COMPLETE / exact-SHA verified for their stage-scoped contracts  
> **Bound Specification snapshot:** `82abd09360df09a95fbbfb516855fa9ffb49f050`

This file is **not** the WU-04G stage document and the review baseline above is **not** a WU-04G entry baseline. The actual WU-04G document must be created only after the maintainer reviews the decisions below, a fresh development-branch HEAD is read, and WU-04G is explicitly entered.

## 1. Why a pre-entry decision review is required

WU-04E and WU-04F established one resolved/effective architecture boundary:

```text
source ptsip.yaml
    -> validate_profile()
    -> ValidationResult.resolved_profile
    -> ResolvedProfile.effective_payload
    -> downstream validation / conformance
```

Clarification and adoption have not yet fully moved onto this boundary.

Current clarification behavior still loads the selected profile source document directly and reads top-level `components` and `associated_artifacts`. This means a `template` profile can appear to clarification as if it declared no components even though its effective map is complete. A `hybrid` profile can likewise be judged from source overrides rather than the architecture produced after deterministic materialization.

Current adoption behavior is partially modernized but intentionally conservative:

- candidate/reconciliation reads call clarification's declared-component reader;
- canonical direct projection writes only `responsibility_map.mode: explicit`;
- direct projection refuses `template` and `hybrid` source modes;
- canonical Tool 0.3.6 projection stores `classification` as lifecycle authority and does not serialize `lifecycle_owner`;
- `DecisionAnswer` still carries `lifecycle_owner` as an input/migration compatibility fact;
- remote GitHub decision application still uses the same projection path and therefore inherits the explicit-only write restriction.

WU-04G must therefore decide **what is read from the effective map** without accidentally granting clarification/adoption new architecture-writing authority.

## 2. Non-negotiable boundaries inherited from WU-04C through WU-04F

Whatever option is selected, WU-04G must preserve these boundaries:

1. `classification` is canonical lifecycle ownership. Tool 0.3.6 must not restore `TOOLCHAIN` as a canonical PTSIP classification alias.
2. Template selection is explicit project authority. Clarification/adoption must not infer or auto-select a template from repository evidence.
3. Materialization is deterministic and non-authoritative. A consumer must not silently rewrite effective architecture back into source form.
4. `template` / `hybrid` source-mode provenance must remain distinct from lifecycle classification.
5. Associated artifacts are project-owned **non-component** support scopes; they must not be silently promoted to components.
6. Invalid or unresolvable project profiles remain fail-closed. A consumer must not use a weaker raw-profile fallback merely to continue.
7. Existing source declarations must not be overwritten merely because discovery proposes a candidate that overlaps them.
8. Tool 0.3.5 legacy state is migration input, not canonical Tool 0.3.6 state. Legacy migration remains WU-07 and later.
9. VPMS vocabulary remains independent and is not part of WU-04G.

## 3. Current implementation surfaces that WU-04G may touch

Primary read path:

```text
src/ptsip/clarification/generator.py
    _declared_profile_scopes()
    declared_components()
    analyze_clarifications()

src/ptsip/clarification/generator_core.py
    covering_components()
    build_requests()
```

Decision contract / resolution path:

```text
src/ptsip/clarification/model.py
src/ptsip/clarification/resolution/model.py
src/ptsip/clarification/resolution/parser.py
src/ptsip/clarification/resolution/resolver.py
```

Adoption / projection path:

```text
src/ptsip/adoption.py
src/ptsip/clarification/resolution/profile_projection.py
```

Remote decision application path:

```text
src/ptsip/app/service.py
```

Relevant current regression signal from exact-SHA run `32240740753` / job `96030499443` at source SHA `48b75e699a592703e4e03a8462131e4932103677`:

```text
260 passed / 13 failed
```

The remaining decision-control failures still include tests expecting canonical `TOOLCHAIN` behavior or persisted `lifecycle_owner`; these expectations conflict with the already active Tool 0.3.6 canonical model and must not be used as justification to restore the old ontology.

## 4. Decision D1 — What is the read authority for clarification/adoption coverage?

### Option D1-A — Continue reading raw source declarations

```text
ptsip.yaml
    -> source components / source associated_artifacts
    -> clarification coverage
```

Advantages:

- smallest code change;
- source document is easy to explain.

Problems:

- template profiles appear structurally empty to the consumer;
- hybrid profiles expose only local source deltas rather than the selected effective architecture;
- duplicates semantics already solved by WU-04D/E;
- can re-ask the maintainer about responsibility already declared through a template.

**Assessment:** not suitable for WU-04G.

### Option D1-B — Consume `validate_profile().resolved_profile.effective_payload`

```text
selected profile
    -> validate_profile()
    -> resolved_profile
    -> effective components / effective associated_artifacts
    -> clarification/adoption read coverage
```

Advantages:

- one architecture interpretation for explicit/template/hybrid;
- reuses the same validated boundary already used by conformance;
- avoids source-mode-specific branches in clarification/adoption;
- preserves source/effective separation because the effective payload is read-only here.

Cost:

- clarification must surface validation/materialization failure instead of silently falling back.

**Provisional recommendation:** **D1-B**.

### Option D1-C — Dual-read raw source and effective map

Use the effective map for coverage but also independently compare raw source declarations.

Advantages:

- may provide extra diagnostics.

Problems:

- creates two potential architecture interpretations;
- increases the chance that source-mode-specific logic returns through the back door;
- duplicated consistency logic should already belong to validation/materialization.

**Assessment:** use source identity/provenance only for explanation; do not make raw source a second coverage authority.

## 5. Decision D2 — What happens if validation/materialization cannot produce a resolved profile?

### Option D2-A — Fall back to raw source coverage

This keeps clarification available even for an invalid template/hybrid profile.

**Risk:** a weaker consumer would reason over architecture that the canonical validator has already rejected.

### Option D2-B — Fail closed and report profile-resolution failure

No clarification/adoption coverage decision is produced until the selected project profile is valid and resolvable.

**Provisional recommendation:** **D2-B**.

Suggested behavior:

```text
invalid / unresolved profile
    -> CLARIFICATION_BLOCKED or PROFILE_CONFLICT
    -> include validation errors
    -> do not manufacture component questions from partial source state
```

The exact public status name should be selected during WU-04G implementation; this review only decides the semantic behavior.

## 6. Decision D3 — How should effective associated artifacts affect clarification?

### Option D3-A — Ignore associated artifacts during candidate coverage

A candidate overlapping an associated artifact may still trigger a component-classification question.

Problem: this contradicts the current architectural meaning of an associated artifact as an already declared non-component support scope.

### Option D3-B — Effective associated-artifact scope suppresses duplicate component clarification

If a discovered candidate is already covered by an effective associated-artifact selector, clarification does not immediately ask the maintainer to classify the same scope as a component.

Promotion remains an explicit owner action if independent lifecycle responsibility has emerged.

**Provisional recommendation:** **D3-B**.

### Option D3-C — Treat associated artifacts as components for coverage

**Reject.** This collapses the component/artifact distinction frozen by WU-02/WU-03.

## 7. Decision D4 — Which selector/coverage algorithm should clarification use?

This is a significant design point.

Current `covering_components()` performs its own normalized-prefix/glob-style coverage check. Validation already owns canonical selector partition/conflict semantics. Leaving both implementations independent can produce cases where validation assigns a path to one owner while clarification concludes that a different declaration covers the candidate.

### Option D4-A — Keep the current clarification-local selector heuristic

Advantages:

- minimal implementation work.

Risks:

- selector semantics can drift from validation;
- exclusion/specificity/partition behavior can diverge;
- difficult to prove explicit/template/hybrid equivalence.

### Option D4-B — Introduce/reuse one shared canonical selector-coverage primitive

Clarification asks the validation/selector layer for coverage using the same normalized selector semantics used for effective partitioning.

Possible shape:

```text
resolve_candidate_coverage(candidate, effective_components, effective_artifacts)
    -> component covers
    -> artifact covers
    -> ambiguous / none
```

The exact API does not need to be decided in this review, but the semantic owner should be the validation/selector layer, not a second clarification-specific interpretation.

**Provisional recommendation:** **D4-B**.

### Option D4-C — Use only tracked-path partition assignments

Advantages:

- directly reuses validated partition output.

Limitation:

- candidate selectors/anchors can represent intended scope beyond the exact currently tracked path set;
- partition-only matching may be insufficient for adoption planning.

**Assessment:** useful evidence, but probably not sufficient as the only coverage primitive.

## 8. Decision D5 — What should happen to `DecisionAnswer.lifecycle_owner` in WU-04G?

Current state:

```text
DecisionAnswer.classification      = canonical Tool 0.3.6 ownership authority
DecisionAnswer.lifecycle_owner     = compatibility/input fact only
projected Tool 0.3.6 profile       = does not serialize lifecycle_owner
```

### Option D5-A — Keep clarification answer format v1 unchanged for WU-04G

Keep `lifecycle_owner` required in the answer payload but validate it only as a compatibility fact consistent with the canonical classification.

Required cleanup:

- canonical answers use `DEVELOPMENT_TOOLING`, not `TOOLCHAIN`;
- stale tests that expect `TOOLCHAIN` to validate must be migrated;
- projection continues to omit `lifecycle_owner` from Tool 0.3.6 profiles.

Advantages:

- avoids combining effective-map read integration with a public answer-format migration;
- preserves already stored/transported v1 decision shape until the planned migration work.

**Provisional recommendation:** **D5-A for WU-04G**, with explicit debt recorded for later answer-contract migration.

### Option D5-B — Create `ptsip-clarification-answer/v2` now and remove `lifecycle_owner`

Advantages:

- cleaner canonical interface immediately.

Risks:

- broadens WU-04G from read integration into transport/store/API migration;
- touches CLI, GitHub issue rendering/parsing, local store, remote service, existing decision records, and compatibility tests;
- may overlap WU-07/WU-10 migration responsibilities.

**Assessment:** architecturally clean, but likely too broad for WU-04G unless the maintainer explicitly chooses to expand scope.

### Option D5-C — Make `lifecycle_owner` optional in v1

This creates a transitional contract without a version bump.

**Risk:** hidden format drift and ambiguous compatibility behavior.

**Assessment:** not preferred.

## 9. Decision D6 — How far may adoption write into template/hybrid source modes?

This is the most important authority decision for WU-04G.

### Option D6-A — Effective-map read for all modes; direct write remains explicit-only

Behavior:

```text
explicit
    effective read -> clarification/adoption reconciliation
    direct projection/write allowed

template
    effective read -> already-declared scope recognized
    uncovered new candidate -> decision may be requested
    direct automatic profile mutation not performed

hybrid
    effective read -> template + project overrides/removals recognized
    uncovered new candidate -> decision may be requested
    direct automatic profile mutation not performed in WU-04G
```

Advantages:

- exactly matches the current WU-04G label: clarification/adoption **read integration**;
- prevents automatic template-to-hybrid conversion;
- does not invent hybrid extension authority from discovery evidence;
- preserves WU-04C non-authoritative materialization boundary.

Cost:

- a valid decision for an uncovered template/hybrid candidate may require a later explicit owner action before source mutation.

**Provisional recommendation:** **D6-A**.

### Option D6-B — Allow hybrid extension writes and automatically convert template -> hybrid when needed

Possible behavior:

```text
template + new accepted component
    -> source_mode becomes hybrid
    -> add project extension component
```

Advantages:

- convenient end-to-end adoption.

Risks:

- source-mode change is an architecture-authority decision;
- an accepted component classification does not necessarily imply consent to change declaration mode;
- creates new hybrid source serialization/write semantics not required for read integration;
- significantly increases CAS and conflict complexity.

**Assessment:** defer unless explicitly authorized as a separate write-design decision.

### Option D6-C — Materialize template/hybrid to explicit and write the full effective map

**Reject.** This destroys declaration-source/provenance semantics and converts derived materialization output into project-owned source authority.

## 10. Decision D7 — What happens to a valid decision that cannot be auto-applied because the profile is template/hybrid?

This matters especially for the remote GitHub decision-control path.

### Option D7-A — Refuse to accept the decision unless automatic projection can succeed

Advantages:

- simple state machine.

Problem:

- can force the maintainer to answer the same architecture question again after changing declaration mode.

### Option D7-B — Accept the authoritative decision, preserve it, and mark source application as manual/pending

Possible state:

```text
RESOLVED
application_status = MODE_CHANGE_REQUIRED | MANUAL_RECONCILIATION_REQUIRED
```

The exact enum/name would be decided during implementation.

Advantages:

- preserves first-valid-resolution semantics;
- avoids re-asking the maintainer;
- separates architecture decision from source-mode write authorization.

Risk:

- requires a carefully defined application state and retry path.

**Provisional recommendation if D6-A is chosen:** **D7-B**, but only if the state-machine change remains small and explicit. Otherwise defer remote template/hybrid application entirely and keep WU-04G focused on read/no-duplicate-question behavior.

### Option D7-C — Auto-convert source mode and apply immediately

This is equivalent to choosing D6-B and inherits its authority risks.

## 11. Decision D8 — Should remote GitHub projection continue to assume root `ptsip.yaml`?

Local clarification/adoption already accepts an explicit `--profile` path. Remote `DecisionService` currently reads and writes repository-root `ptsip.yaml` at the gate revision.

### Option D8-A — Keep root-only remote projection in WU-04G

Advantages:

- avoids adding remote profile-path identity to the decision protocol now.

### Option D8-B — Carry exact profile path through gate/store/remote CAS

Advantages:

- local and remote behavior become symmetric;
- avoids applying a decision to the wrong profile if the repository intentionally selects a non-root profile.

Cost:

- changes gate payload, persisted decision record, GitHub file reads, CAS write target, and tests.

**Provisional recommendation:** **D8-A for WU-04G unless a real current workflow already depends on remote non-root profiles.** Record D8-B for later control-plane hardening.

## 12. Recommended decision package for maintainer review

This is **not yet an accepted design**. It is the smallest package that preserves established authority boundaries:

```text
D1  B  use validated ResolvedProfile.effective_payload as read authority
D2  B  fail closed when resolved profile is unavailable
D3  B  effective associated-artifact coverage suppresses duplicate component questions
D4  B  share canonical selector coverage semantics with validation
D5  A  keep clarification-answer/v1 + lifecycle_owner compatibility fact for this WU
D6  A  all modes read effective map; automatic source writes stay explicit-only
D7  B* preserve valid decision without auto mode conversion when practical
D8  A  keep remote root-profile assumption in this WU unless a concrete need requires expansion
```

`D7-B*` is intentionally marked conditional. The maintainer may instead choose to keep remote template/hybrid decision application entirely out of WU-04G if preserving a manual-pending state would enlarge the stage too much.

## 13. Expected WU-04G runtime boundary if the recommended package is accepted

```text
repository discovery
      |
      +--> candidate evidence
      |
selected Project Profile
      |
      v
validate_profile()
      |
      v
ResolvedProfile.effective_payload
      |
      +--> effective component selectors
      +--> effective associated-artifact selectors
      |
      v
canonical candidate-coverage resolution
      |
      +--> already declared component     -> no clarification
      +--> declared associated artifact   -> no duplicate component clarification
      +--> uncovered candidate            -> clarification request
      +--> ambiguous coverage             -> fail/review, never guess ownership

accepted decision
      |
      +--> explicit source mode            -> existing safe projection/CAS path
      +--> template/hybrid source mode     -> no silent materialize-to-explicit or mode conversion
```

## 14. Focused regression contract to prepare after decisions

If WU-04G is entered, focused tests should be separate from historical 0.3.3 decision tests. Suggested new target:

```text
tests/ptsip/test_clarification_adoption_effective_map_036.py
```

Minimum contract:

1. explicit, template, and hybrid declarations with equivalent effective coverage produce the same `NO_CLARIFICATION_REQUIRED` result;
2. a template-provided component prevents the same candidate from being re-asked;
3. a hybrid override selector is the selector clarification sees;
4. a hybrid-removed component no longer suppresses a genuinely uncovered candidate;
5. an effective associated artifact suppresses duplicate component clarification for its support scope;
6. invalid/unknown template revision blocks clarification/adoption read rather than falling back to raw source;
7. selector coverage uses the canonical shared semantics and reports ambiguity instead of guessing;
8. explicit adoption still writes canonical Tool 0.3.6 state and never serializes `lifecycle_owner`;
9. canonical `DEVELOPMENT_TOOLING` decision answers validate; canonical PTSIP `TOOLCHAIN` answers do not;
10. template/hybrid adoption does not silently convert mode or serialize the full effective map;
11. repeated clarification/gate operations do not re-open a question already covered by effective architecture;
12. source profile remains unchanged during read-only clarification analysis.

If D7-B is accepted, add remote/local state tests proving an accepted decision can remain authoritative while source application is explicitly pending/manual.

## 15. Proposed parallel-review plan before implementation

The reason for this pre-entry document is to allow a second agent to review the risky boundary independently before implementation begins.

### Review Track A — Effective-read and selector semantics

Read-only review targets:

```text
src/ptsip/clarification/generator.py
src/ptsip/clarification/generator_core.py
src/ptsip/validation/profile.py
src/ptsip/validation/components.py
src/ptsip/validation/materialization.py (or current materializer implementation)
```

Questions for reviewer:

- Is `resolved_profile.effective_payload` sufficient as the only declaration read authority?
- Can current partition/selector primitives be reused without exposing mutable internal state?
- How should candidate-level selector coverage represent ambiguity, include/exclude, and specificity?
- Are associated-artifact selectors sufficient to suppress duplicate component questions without implying promotion?

Expected output: comments on D1-D4 and concrete API-shape risks. **No code changes before maintainer decision.**

### Review Track B — Decision/control-plane and write-authority semantics

Read-only review targets:

```text
src/ptsip/clarification/resolution/model.py
src/ptsip/clarification/resolution/parser.py
src/ptsip/clarification/resolution/resolver.py
src/ptsip/clarification/resolution/profile_projection.py
src/ptsip/adoption.py
src/ptsip/app/service.py
src/ptsip/app/store.py
```

Questions for reviewer:

- Should `clarification-answer/v1` remain unchanged through WU-04G?
- Is `lifecycle_owner` compatibility input still useful enough to keep until WU-07/WU-10?
- If template/hybrid write remains prohibited, should a valid decision be stored as resolved/manual-application-required or rejected before authoritative CAS?
- Could any current remote path accidentally convert derived effective state into source authority?
- Does root-only remote profile selection need to be addressed now or can it remain a separate hardening item?

Expected output: comments on D5-D8 and state-machine/CAS risks. **No code changes before maintainer decision.**

### Maintainer / primary implementation track

After both reviews are returned:

1. maintainer selects D1-D8 decisions;
2. unresolved disagreement is documented explicitly rather than averaged;
3. fresh branch HEAD is read;
4. only then create the official WU-04G stage document with that exact entry SHA;
5. implementation tasks are split by file ownership to avoid concurrent edits;
6. run focused tests first, then exact-SHA self-hosted repository regression when the stage gate is ready.

## 16. Suggested implementation split after decisions are accepted

A low-conflict split would be:

```text
Agent / Track A
    clarification effective-read adapter
    shared candidate coverage primitive
    clarification-focused tests

Agent / Track B
    decision answer compatibility cleanup
    remote/local decision application state behavior if D7 requires it
    decision-control focused tests

Primary integrator
    adoption reconciliation wiring
    cross-track integration tests
    planning/status documents
    final exact-SHA verification classification
```

Do not implement Track A and Track B against different interpretations of D5-D7. The maintainer decision must be recorded first.

## 17. Explicitly deferred topics

Unless the maintainer expands WU-04G, this review recommends deferring:

- Tool 0.3.5 legacy profile migration;
- automatic `TOOLCHAIN -> DEVELOPMENT_TOOLING` translation;
- clarification-answer v2 migration;
- automatic template -> hybrid conversion;
- materialize-to-explicit writeback;
- topology migration behavior;
- VPMS effective-map bridge;
- repository-wide historical fixture cleanup unrelated to G;
- remote non-root profile protocol changes without a demonstrated current need.

## 18. Maintainer decision checklist for the next session

The next session can decide these directly:

```text
D1  Read authority: raw source / resolved effective map / dual read
D2  Invalid profile: raw fallback / fail closed
D3  Associated artifacts: ignore / suppress duplicate component question
D4  Selector semantics: local heuristic / shared canonical coverage / partition-only
D5  DecisionAnswer: keep v1 compatibility / create v2 / make legacy field optional
D6  Template/hybrid write: explicit-only / hybrid auto-extension / materialize-to-explicit
D7  Non-auto-applicable valid decision: reject / preserve as manual-pending / auto-convert
D8  Remote profile path: root-only for now / carry exact selected profile path
```

No decision in this checklist is considered accepted merely because a provisional recommendation appears in this review. The official WU-04G document must record the maintainer-selected package and its exact entry baseline.
