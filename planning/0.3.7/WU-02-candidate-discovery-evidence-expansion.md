# WU-02 — Candidate-Discovery Evidence Expansion

> **Status:** COMPLETE / FOCUSED TEST VERIFIED  
> **Target Tool:** `0.3.7`  
> **Roadmap predecessor:** WU-01 — draft profile transition state model (`COMPLETE / FOCUSED TEST VERIFIED`)  
> **WU-02 exact entry baseline:** `6d364055532c592e8c6f778f5a145148e7f7e29a`  
> **Bound Specification at entry:** `0.3.7-draft @ b648d9e026f502b14481ba2d0606d9acc88a31fc`  
> **WU-02 implementation content SHA:** `749d2a5969ad9f04805db87901cb141b3e419395`  
> **Focused verification:** `10 passed / 0 failed` on the WU-02 isolated focused harness  
> **Successor:** WU-03 — evidence/provenance normalization

## 0. Purpose

Expand deterministic repository candidate discovery so PTSIP can observe a broader and more precise set of architecture-relevant evidence without converting evidence into project-owned architecture authority.

This work retains the cancelled Tool `0.3.6.1` WU-01 intent, but all evidence must now be attributable to the exact source-profile generation and repository snapshot being evaluated.

## 1. Core boundary

```text
repository observation
    -> candidate evidence
    -> candidate identity / selector evidence
    -> source-specific migration analysis

repository observation
    -X-> automatic lifecycle classification
    -X-> automatic architecture mutation
    -X-> automatic Required/Removal/Async obligation status without source evaluation
```

Discovery is descriptive. It does not own migration-completion semantics.

## 2. Initial scope

Candidate discovery evaluates existing evidence classes and extends them only where the signal is deterministic and reviewable. Sources include:

- manifests and package metadata;
- source/package roots;
- test roots and verification entrypoints;
- CI/workflow-invoked scripts;
- release/package assembly inputs;
- schema and neutral-contract groups;
- generated or embedded contract copies;
- operational/maintenance scripts where objective evidence exists;
- dependency edges identifying architecture-relevant boundaries without deciding ownership.

## 3. Generation-aware evidence identity

Equivalent repository observations may participate in more than one source migration during Sequential Work. Discovery preserves enough context to answer:

```text
what was observed?
where was it observed?
under which repository snapshot?
for which source-profile evaluation was it collected?
```

Source-generation context does not make evidence authoritative; it prevents evidence from being accidentally reused as if a previous source evaluation had already decided a later source's obligations.

Candidate identity and source-evaluation identity are deliberately separate:

```text
normalized selector scope
    -> stable candidate ID

repository snapshot
+ source profile path/version/revision/content hash
+ transition mode/final point
    -> evaluation context ID
```

Therefore equivalent observations from two source generations may share one stable candidate ID while retaining distinct evaluation contexts.

## 4. Work tracks

### WU-02A — discovery inventory — COMPLETE

Existing discovery surfaces were inventoried before implementation.

The reusable pipeline is:

```text
repository.snapshot.repository_files
    -> inspection.inventory
    -> inspection.dependencies / dependencies_030
       -> Python / GitHub Actions base evidence
       -> JavaScript / TypeScript adapter
       -> Go adapter
       -> .NET adapter
    -> inspection.components legacy ComponentCandidate discovery
    -> validation.components selector / coverage semantics
```

The important duplication risk was selector matching. `validation/components.py` already owns canonical selector normalization, containment, specificity, ambiguity, and candidate coverage. WU-02 therefore does not create a second selector engine.

The legacy `ComponentCandidate` path used by pilot/adoption/clarification was not rewritten. WU-02 adds a sibling evidence layer so existing Tool 0.3.6 behavior stays stable.

### WU-02B — deterministic evidence expansion — COMPLETE

Added:

```text
src/ptsip/inspection/candidate_evidence.py
```

The new evidence layer observes deterministic candidate signals for:

```text
MANIFEST
PACKAGE_ASSEMBLY_INPUT
SOURCE_PACKAGE_ROOT
TEST_ROOT
TOOL_ROOT
CONTRACT_GROUP
EMBEDDED_CONTRACT_COPY
MAINTENANCE_SCRIPT
CI_INVOKED_SCRIPT
RELEASE_ACTIVITY_SOURCE
RELEASE_ACTIVITY_TARGET
DEPENDENCY_BOUNDARY
```

Every observation records:

- a deterministic evidence ID;
- observation kind;
- provenance (`OBSERVED`, `DECLARED`, or `INFERRED` where applicable);
- contributing adapter;
- concrete path when available;
- reviewable detail text.

Stable candidate IDs are derived from normalized selector scope, not from discovery order or source-generation identity. Duplicate observations on the same selector converge into one candidate with multiple observations.

### WU-02C — selector and coverage integration — COMPLETE

Candidate coverage uses the existing canonical function:

```text
validation.components.resolve_candidate_coverage()
```

No parallel glob matcher, specificity algorithm, or declaration-coverage implementation was added.

Source-profile `components` and `associated_artifacts` are used only as declared selector coverage evidence. Coverage does not create a lifecycle classification or migration obligation.

### WU-02D — ambiguity handling — COMPLETE

When canonical coverage resolution returns multiple equally valid declared owners or overlapping component/artifact scope, the candidate remains:

```text
AMBIGUOUS
```

The evidence layer does not choose an owner by path naming, activity naming, confidence, or discovery order.

Uncovered candidates remain:

```text
UNCOVERED
```

and are still valid evidence. They are not automatically classified.

Serialized candidate evidence explicitly reports:

```text
authority = EVIDENCE_ONLY
```

and does not contain lifecycle `classification`, Required/Removal/Async obligation, or mutation authority fields.

### WU-02E — generation/snapshot binding — COMPLETE

Candidate discovery now binds each evaluation to the WU-01 transition state.

The context records:

- resolved repository root;
- exact Git HEAD when present;
- repository status fingerprint;
- tracked/repository content fingerprint;
- transition mode;
- Final Point path when present;
- source profile path;
- source draft version;
- immutable source Specification revision;
- source profile SHA-256 content identity;
- deterministic evaluation context ID.

The Final Point is rejected as a source-generation context. Eligible sources follow WU-01 `ordered_sources`; when no migration target exists, canonical `ptsip.yaml` is the source context.

`validate_candidate_discovery_context()` reuses WU-01 snapshot validation and reports stale evidence through `STALE_TRANSITION_SNAPSHOT`.

### WU-02F — repository fixtures and regression — COMPLETE

Added the role-scoped focused test file:

```text
tests/ptsip/inspection/test_candidate_evidence_037.py
```

The tests cover:

- deterministic candidate IDs and stable evaluation identity;
- shared selector/coverage integration;
- duplicate observation convergence;
- explicit coverage ambiguity;
- uncovered evidence without automatic classification;
- CI-invoked + maintenance-script evidence convergence;
- release-phase and cross-root dependency evidence;
- embedded contract-copy evidence;
- multi-generation candidate identity with distinct source contexts;
- Final Point source rejection;
- stale repository/profile context detection.

Focused verification result:

```text
10 passed / 0 failed
```

The focused suite was executed in an isolated harness reproducing the WU-02 candidate module behavior together with the existing WU-01 transition contract, repository snapshot behavior, inventory behavior, and canonical selector/coverage semantics. Dependency observations were injected as fixed evidence fixtures so language-adapter behavior would not mask candidate-layer verification.

This result is recorded as focused WU verification, not as GitHub Actions exact-SHA full regression. The existing legacy `inspection/components.py` candidate path was not modified; the shared selector/coverage implementation is directly exercised by the new focused suite. Full repository regression and self-hosted `tooling-test.yml` exact-SHA verification remain WU-08 release-readiness gates.

## 5. Non-goals

WU-02 does not authorize:

- creation or promotion of Temporary PTSIP Profile Files;
- lifecycle migration writes;
- Required/Removal/Async final categorization;
- template selection by evidence;
- target proposals;
- safe-apply writes;
- release workflow changes unrelated to actual verification needs.

No such mutation was performed by WU-02.

## 6. Delivered surfaces

```text
src/ptsip/inspection/candidate_evidence.py
tests/ptsip/inspection/test_candidate_evidence_037.py
planning/0.3.7/WU-02-candidate-discovery-evidence-expansion.md
```

No existing inspection adapter, legacy candidate API, canonical profile, schema, workflow, or Responsibility Map declaration was changed for WU-02.

## 7. Completion gate

WU-02 completion is satisfied as follows:

- expanded discovery is deterministic on a stable repository snapshot — **satisfied**;
- every candidate can identify why and where it exists — **satisfied through structured observations**;
- evidence is associated with the correct source migration without inheriting prior source decisions — **satisfied through generation-aware evaluation context**;
- duplicate observations converge on stable candidate identity where appropriate — **satisfied**;
- ambiguity remains explicit — **satisfied**;
- discovery does not silently assign lifecycle ownership or migration obligation category — **satisfied**;
- focused and participating shared discovery/coverage behavior is verified — **10 focused tests passed; legacy candidate path unchanged**;
- WU-03 was not entered before this completion record — **satisfied**.

## 8. Entry discipline

WU-02 entered on exact `dev/0.3.7` baseline `6d364055532c592e8c6f778f5a145148e7f7e29a` after WU-01 completion was recorded and the branch HEAD was freshly revalidated.

WU-02 completion itself does not authorize lifecycle migration writes, Temporary PTSIP Profile promotion, or bypass of any project-owner decision gate.

## 9. Successor auto-entry authorization

The project owner has approved automatic successor-WU entry for the remainder of Tool `0.3.7` work.

After an active WU reaches its documented completion gate, the next WU MAY be changed from `PRE-CREATED / LOCKED` to `ACTIVE` without a separate approval message, provided that:

1. the predecessor completion is recorded;
2. a fresh exact `dev/0.3.7` HEAD is captured for the successor entry baseline;
3. no unresolved project-owner decision, architecture conflict, or explicit confirmation gate blocks entry;
4. only the successor entry state is automated — implementation beyond that WU's approved scope is not implicitly authorized by this rule.

If any blocking decision or conflict remains, the successor MUST stay locked until it is resolved.
