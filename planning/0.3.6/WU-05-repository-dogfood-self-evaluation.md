# WU-05 — Repository Dogfood / Self-Evaluation

> **Status:** ACTIVE  
> **Parent release:** Tool `0.3.6` — Primary Lifecycle Ownership Foundation  
> **Entry branch:** `tool-0.3.6-lifecycle-ownership`  
> **Entry predecessor:** WU-04 — COMPLETE / EXACT-SHA VERIFIED  
> **WU-04 implementation verification snapshot:** `eadb07f78f9690ca0180bfbba194c5c9602e1838`  
> **WU-05 exact entry baseline:** `c87e597409592f0794a5d404d621b6f73dd00cc6`  
> **Bound Specification:** `0.3.6-draft @ d6995ed232e845b88d8235b851e80ab54b7804ea`  
> **Successor:** WU-06 — full regression / package / distribution verification; locked until WU-05 completion

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

### D2 — downstream semantic consistency

Validation, conformance, clarification, and gate behavior must consume the same declared/effective architecture semantics.

Equivalent architecture must not produce contradictory lifecycle ownership merely because a downstream command observes different repository evidence.

### D3 — read-only dogfood surfaces remain non-mutating

Read-only evaluation commands must not change:

```text
ptsip.yaml
Decision Authority state
tracked repository content
Effective Map source declaration
```

without an explicit project-owned write operation.

### D4 — no hidden migration dependency

Tool `0.3.6` repository self-adoption must be usable without the deferred Tool `0.3.6.1` legacy-reader/migration pipeline.

### D5 — Specification binding remains stable

Dogfood alone does not move:

```text
Specification: 0.3.6-draft
SPEC_REVISION: d6995ed232e845b88d8235b851e80ab54b7804ea
```

If dogfood exposes an actual missing or incorrect normative rule, that is recorded explicitly for review rather than silently changing the binding.

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

## 7. Verification evidence

The WU-05 completion record must capture at least:

```text
entry baseline
candidate/completion SHA
working-tree/source stability where material
Tool identity
Specification identity
validate result
Effective Map / coverage result
conform result
clarify result
local gate result
all discovered findings and their ownership classification
any project-owned declaration correction performed
```

A WU-04 exact-SHA PASS may be used as predecessor evidence, but it does not replace WU-05's repository-consumer evaluation.

Likewise, WU-05 completion does not replace WU-06's final full regression and package/distribution verification.

## 8. Completion gate

WU-05 is complete only when:

- the repository has been exercised as a Tool `0.3.6` consumer from this entry baseline or a documented descendant;
- the canonical root declaration remains valid or any WU-05-owned declaration defect is corrected;
- Effective Responsibility Map coverage is coherent and complete under the self-profile contract;
- conformance output is reviewed, not merely executed;
- clarification and local gate behavior are reviewed for unwanted architecture reinterpretation;
- read-only evaluation remains non-mutating;
- every dogfood finding is classified by ownership;
- no Tool `0.3.6.1` migration implementation is entered;
- no WU-06/WU-07 release work is entered early;
- no new Specification revision is created unless a genuine normative defect is separately reviewed;
- completion evidence is recorded here, `planning/0.3.6.md`, and `STATUS.md`.

After completion:

```text
WU-05 COMPLETE
    -> fresh branch HEAD
    -> create WU-06 stage document with exact entry baseline
    -> WU-06 ACTIVE
```

## 9. Initial execution sequence

```text
1. record exact entry baseline
2. establish repository/Tool/Specification identity
3. run and review repository validation + Effective Map coverage
4. run and review repository conformance
5. run and review clarification and local gate
6. verify read-only stability / non-mutation
7. classify and fix only WU-05-owned defects
8. repeat affected dogfood checks
9. record WU-05 completion evidence
```

WU-05 is ACTIVE from exact entry baseline `c87e597409592f0794a5d404d621b6f73dd00cc6`.
