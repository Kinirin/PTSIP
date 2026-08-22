# WU-05 — Repository Dogfood / Self-Evaluation

> **Status:** COMPLETE — exact repository-consumer snapshot reviewed  
> **Parent release:** Tool `0.3.6` — Primary Lifecycle Ownership Foundation  
> **Entry branch:** `tool-0.3.6-lifecycle-ownership`  
> **Entry predecessor:** WU-04 — COMPLETE / EXACT-SHA VERIFIED  
> **WU-04 implementation verification snapshot:** `eadb07f78f9690ca0180bfbba194c5c9602e1838`  
> **WU-05 exact entry baseline:** `c87e597409592f0794a5d404d621b6f73dd00cc6`  
> **WU-05 exact dogfood verification snapshot:** `5010fa81ebd104eb329f29217d4c5fecec51cacb`  
> **Bound Specification:** `0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea`  
> **Successor:** WU-06 — full regression / package / distribution verification; remains locked until a fresh post-WU-05 branch-HEAD entry review

## 0. Purpose

WU-05 dogfoods the Tool `0.3.6` architecture foundation against the PTSIP repository itself.

The repository is treated as a real consumer of the current policy rather than as a special fixture. The stage verifies that the project-owned `ptsip.yaml` declaration and its resolved Effective Responsibility Map remain useful, coherent, and non-contradictory when exercised through the public Tool `0.3.6` surfaces.

WU-05 does not redesign the lifecycle model. It evaluates whether the already-implemented model works on the repository that defines and ships it.

## 1. Entry authority

WU-04 completed exact-SHA implementation verification at:

```text
eadb07f78f9690ca0180bfbba194c5c9602e1838
```

with:

```text
GitHub Actions run/job:       32588090216 / 97067421324
self-hosted Windows X64
Python 3.14.6
complete repository pytest:   327 passed / 0 failed
repository profile:           valid=true, warnings=[], unassigned_count=0
self-hosted/tooling-test:      success
```

The post-verification WU-04 closure records advanced the branch to:

```text
c87e597409592f0794a5d404d621b6f73dd00cc6
```

A fresh branch-HEAD read confirmed that SHA immediately before WU-05 entry. It is therefore the exact WU-05 entry baseline.

The repository-consumer dogfood commands were then executed against the documented descendant:

```text
5010fa81ebd104eb329f29217d4c5fecec51cacb
```

That SHA remained both the local reported commit and remote development-branch HEAD throughout the reviewed dogfood sequence. No Tool source, Specification, schema, workflow, profile, or other tracked repository content was changed during the dogfood execution.

## 2. Scope

WU-05 owns repository self-evaluation of the Tool `0.3.6` foundation already present at entry.

The dogfood evaluation must cover at least:

1. **project declaration validity** — root `ptsip.yaml` remains a valid canonical Tool `0.3.6` project-owned declaration;
2. **Effective Responsibility Map coherence** — materialized component, associated-artifact, relationship, and policy state represents the repository without unresolved selector conflict or uncovered tracked-file responsibility;
3. **conformance behavior** — `ptsip conform . --json` evaluates the repository from the validated Effective Responsibility Map rather than an alternate raw-profile interpretation;
4. **clarification behavior** — `ptsip clarify . --json` does not invent architecture decisions already represented by project-owned declarations and remains read-only;
5. **decision gate behavior** — local `ptsip gate . --coordination local --json` remains consistent with the declared architecture and does not mutate source when no decision is required;
6. **public diagnostic consistency** — relevant read-only public commands report Tool/Specification/profile identity consistently enough to use the repository as a realistic Tool `0.3.6` consumer;
7. **repository authority boundary** — discovery, paths, workflow names, or confidence signals may provide evidence but must not override the project-owned Responsibility Map.

## 3. Non-goals

WU-05 does **not** implement or enter the Tool `0.3.6.1` migration continuation:

```text
candidate-discovery evidence expansion
stronger evidence/provenance normalization
Tool 0.3.5 legacy reader
migration analyzer
component split / relationship proposals
migration preview / confirmation / safe apply
```

WU-05 also does not enter Tool `0.3.6` release-closure stages early:

```text
WU-06 full regression / package / distribution verification
WU-07 final Specification freeze / release preparation
```

The following are not authorized merely to make dogfood output look cleaner:

- changing canonical lifecycle classifications;
- introducing a new architecture authority source;
- blind `TOOLCHAIN -> DEVELOPMENT_TOOLING` migration;
- auto-selecting templates from repository evidence;
- broad refactoring unrelated to an observed repository self-adoption defect;
- weakening fail-closed behavior or selector/coverage checks;
- moving `SPEC_REVISION` without a genuine normative defect.

## 4. Dogfood execution model

The intended flow is:

```text
fresh WU-05 entry tree
    -> Tool / Specification identity
    -> validate repository declaration
    -> inspect resolved/effective repository architecture
    -> conform repository
    -> clarify repository
    -> local decision gate
    -> review outputs against declared architecture
    -> classify any defect
```

The repository declaration remains architecture authority throughout this sequence.

Observed repository facts may reveal that the current declaration is incomplete or inaccurate. Such evidence can justify a proposed project-owned correction, but it does not itself authorize a lifecycle classification or map rewrite.

## 5. Required invariants

### D1 — declaration and effective coverage

The repository must resolve without selector conflict and with complete responsibility coverage for tracked project files under the current self-profile contract.

Expected invariant:

```text
source ptsip.yaml
    -> valid
    -> deterministic resolved profile
    -> responsibility_map_coverage.unassigned_count = 0
```

**WU-05 result: PASS.**

Observed at the exact dogfood verification snapshot:

```text
valid: true
errors: []
warnings: []
source_mode: explicit
materialized: true
component selector conflicts: 0
associated-artifact conflicts: 0
scan_errors: []
responsibility_map_coverage.unassigned_count: 0
effective_map_digest: sha256:c009d4fb98daa8a3dc0fd2dfbdd974b467cf107c8c1e7814e460059f4c0c4a01
```

The component partition separately reported 13 paths outside component ownership, but all 13 were project-declared associated-artifact paths under `ptsip-governance-support`. Final Responsibility Map coverage therefore remained complete with zero unassigned tracked files.

### D2 — downstream semantic consistency

Validation, conformance, clarification, and gate behavior must consume the same declared/effective architecture semantics.

Equivalent architecture must not produce contradictory lifecycle ownership merely because a downstream command observes different repository evidence.

**WU-05 result: PASS for architecture semantics.**

`ptsip conform . --json` consumed the same validated profile/effective-map identity and digest. Its result was `INCOMPLETE`, not because a contradictory lifecycle ownership or definite architecture violation was found, but because strict Enforced Conformance retained blocking evidence gaps.

Observed conformance summary:

```text
level: ENFORCED
outcome: INCOMPLETE
definite architecture violations: 0
conformance report audit: PASS
snapshot comparison: STABLE
effective_map_digest: sha256:c009d4fb98daa8a3dc0fd2dfbdd974b467cf107c8c1e7814e460059f4c0c4a01
```

The reviewed blocking evidence consisted of:

```text
1   Product Artifact / artifact-binding evidence concern
13  unresolved Python dependency targets involving distribution/import-name mismatch
6   executable-component independent-build manifest evidence gaps
20  total blocking gaps
```

These observations did not cause conformance to invent or override project-owned architecture.

### D3 — read-only dogfood surfaces remain non-mutating

Read-only evaluation commands must not change:

```text
ptsip.yaml
Decision Authority state
tracked repository content
Effective Map source declaration
```

without an explicit project-owned write operation.

**WU-05 result: PASS.**

Conformance and clarification reported stable before/after repository snapshots. Clarification reported:

```text
status: NO_CLARIFICATION_REQUIRED
inference.mode: DETERMINISTIC_RULES_ONLY
llm_calls: 0
speculative_classification: false
component_candidate_count: 6
clarification_count: 0
requests: []
snapshot.comparison: STABLE
```

Local gate reported:

```text
status: NO_DECISION_REQUIRED
backend: LOCAL
decisions: []
profile_path: ptsip.yaml
```

Post-gate repository checks produced no output for either:

```text
git diff -- ptsip.yaml
git status --short
```

Therefore the final observed working tree was clean and no dogfood command mutated the project-owned declaration or tracked repository state.

### D4 — no hidden migration dependency

Tool `0.3.6` repository self-adoption must be usable without the deferred Tool `0.3.6.1` legacy-reader/migration pipeline.

**WU-05 result: PASS.**

No Tool `0.3.6.1` migration implementation was entered or required to validate, materialize, conform, clarify, or gate the current canonical Tool `0.3.6` repository declaration.

### D5 — Specification binding remains stable

Dogfood alone does not move:

```text
Specification: 0.3.6-draft
SPEC_REVISION: d6995ed232e845b88d8235b851e80ab54b7804ea
```

If dogfood exposes an actual missing or incorrect normative rule, that is recorded explicitly for review rather than silently changing the binding.

**WU-05 result: PASS.** No normative defect was established and the immutable binding did not move.

## 6. Defect classification

Any finding discovered during dogfood must be classified before modification:

```text
A. repository declaration defect
B. Tool 0.3.6 implementation defect
C. test/fixture/documentation drift
D. deferred Tool 0.3.6.1 migration/discovery concern
E. release-closure concern owned by WU-06/WU-07
F. genuine normative Specification defect
```

Only A-C defects that are necessary for the Tool `0.3.6` repository dogfood contract belong to normal WU-05 correction scope.

D and E must remain deferred to their owning work units. F requires explicit Specification review before changing the immutable binding.

### Final WU-05 finding classification

```text
A. repository declaration defect
   NONE

B. Tool 0.3.6 implementation defect
   NONE CONFIRMED

C. test/fixture/documentation drift
   NONE IDENTIFIED

D. deferred Tool 0.3.6.1 migration/discovery concern
   NONE ENTERED

E. WU-06/WU-07 release-closure concern
   Product Artifact evidence / exact artifact snapshot binding
   -> DEFERRED TO WU-06

F. genuine normative Specification defect
   NONE
```

Two initially suspected implementation defects were reviewed more closely and were **not** confirmed as Tool `0.3.6` correctness defects:

```text
OBS-01 — Python distribution/import-name mismatch
PyYAML -> yaml and PyJWT -> jwt remain unresolved by the native dependency scanner.
The evaluator does not misclassify these as project-owned or non-conformant; it retains blocking uncertainty and returns INCOMPLETE.
This is a native evidence-coverage limitation with specification-consistent fail-closed behavior, not a WU-05-required correctness fix.

OBS-02 — independent-build manifest evidence
The current evaluator requires executable components to expose component-declared, component-owned manifest evidence and deliberately blocks shared cross-lifecycle manifest evidence.
The canonical profile schema makes `manifests` optional, and the Specification permits a structurally valid profile to remain insufficient for strict conformance when an optional fact/evidence is required for a mandatory evaluation.
The repository therefore remains valid while strict conformance remains INCOMPLETE; no false NON_CONFORMANT result was produced.
```

Neither observation authorizes a `ptsip.yaml` rewrite merely to obtain a green conformance output.

## 7. Verification evidence

The WU-05 completion record captures:

```text
entry baseline:                     c87e597409592f0794a5d404d621b6f73dd00cc6
exact dogfood verification SHA:     5010fa81ebd104eb329f29217d4c5fecec51cacb
working tree after gate:            CLEAN
Tool identity:                      0.3.6
Specification family:               0.3.6-draft
SPEC_REVISION:                       d6995ed232e845b88d8235b851e80ab54b7804ea
validate:                            PASS
Effective Map coverage:              unassigned_count=0
Effective Map digest:                sha256:c009d4fb98daa8a3dc0fd2dfbdd974b467cf107c8c1e7814e460059f4c0c4a01
conform:                             ENFORCED / INCOMPLETE / reviewed
conformance definite violations:     0
conformance audit:                   PASS
conformance snapshot:                STABLE
clarify:                             NO_CLARIFICATION_REQUIRED
clarify snapshot:                    STABLE
local gate:                          NO_DECISION_REQUIRED / LOCAL / decisions=[]
project-owned declaration changes:   NONE
confirmed WU-05 A-C defects:          NONE
normative Specification defect:       NONE
```

A WU-04 exact-SHA PASS remains predecessor evidence, but it does not replace this WU-05 repository-consumer evaluation.

Likewise, WU-05 completion does not replace WU-06's final full regression and package/distribution verification.

## 8. Completion gate

WU-05 completion requirements are satisfied:

- the repository was exercised as a Tool `0.3.6` consumer on documented descendant SHA `5010fa81...` of the exact entry baseline;
- the canonical root declaration remains valid;
- Effective Responsibility Map coverage is coherent and complete under the self-profile contract;
- conformance output was reviewed rather than treated as a binary green/red signal;
- clarification and local gate did not reinterpret project-owned architecture;
- read-only evaluation remained non-mutating and the final working tree was clean;
- every material dogfood finding was classified by ownership;
- no confirmed A-C defect required a WU-05 source or profile correction;
- no Tool `0.3.6.1` migration implementation was entered;
- no WU-06/WU-07 release work was entered early;
- no new Specification revision was created;
- completion evidence is recorded in this stage record and is propagated to the release-level planning/status records.

Therefore:

```text
WU-05 COMPLETE
exact repository-consumer verification authority:
5010fa81ebd104eb329f29217d4c5fecec51cacb
```

The documentation commits that record this completion occur after the verification SHA and do not replace the exact dogfood verification authority.

After this closure is fully recorded:

```text
WU-05 COMPLETE
    -> fresh branch HEAD
    -> create WU-06 stage document with that exact entry baseline
    -> WU-06 ACTIVE
```

WU-06 is **not** entered by this WU-05 closure record.

## 9. Initial execution sequence — completed

```text
1. record exact entry baseline                         DONE
2. establish repository/Tool/Specification identity   DONE
3. run/review validation + Effective Map coverage     DONE / PASS
4. run/review conformance                             DONE / INCOMPLETE REVIEWED
5. run/review clarification + local gate              DONE / PASS
6. verify read-only stability / non-mutation          DONE / PASS
7. classify and fix only WU-05-owned defects          DONE / no confirmed fix required
8. repeat affected dogfood checks                     NOT REQUIRED — no WU-05 source correction
9. record WU-05 completion evidence                   DONE
```

WU-05 is COMPLETE with exact dogfood verification authority `5010fa81ebd104eb329f29217d4c5fecec51cacb`.