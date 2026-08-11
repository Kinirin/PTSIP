# PTSIP Tool 0.3.3 — Progress Checkpoint (2026-08-11)

> **Status:** Paused / planning checkpoint  
> **Checkpoint date:** `2026-08-11`  
> **Repository branch:** `main`  
> **Repository HEAD before this checkpoint:** `034083d6ac09b78b7ecaf40b558ca4a059300643`  
> **Target Tool version:** `0.3.3`  
> **Bound Specification family:** `0.2.0-draft`  
> **Bound Specification revision:** `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`  
> **Purpose:** Preserve the exact Tool 0.3.3 design and implementation-preparation state before switching to unrelated issue work. No Tool 0.3.3 implementation code is committed by this checkpoint.

## 1. Current repository state

The latest Tool implementation committed to `main` remains Tool `0.3.2` from commit:

```text
ba620456cd510cf1a056073647969b908697795b
```

The only Tool `0.3.3` repository change completed so far is the planning document committed at:

```text
034083d6ac09b78b7ecaf40b558ca4a059300643
planning: define Tool 0.3.3 explicit project adoption
```

Canonical 0.3.3 plan:

```text
planning/PTSIP-TOOL-0.3.3-EXPLICIT-PROJECT-ADOPTION-PLAN.md
```

No `0.3.3` version bump, release note, `ptsip adopt` implementation, gate JSON contract change, gate precondition aggregation, tag, GitHub Release, or PyPI publication has been committed.

## 2. Accepted Consumer Repository feedback for Tool 0.3.3

Three feedback items are accepted for the 0.3.3 workstream.

### Feedback 1 — machine-readable `ptsip gate --json` failures

Current problem:

- normal gate outcomes are structured;
- gate precondition/configuration exceptions can still escape as free-form `PTSIP error: ...` stderr even with `--json`;
- coding agents would therefore need to parse human error strings.

Target:

- when `gate --json` reaches the gate execution contract, failures are returned as structured machine-readable output;
- stable error/precondition codes replace dependence on stderr text parsing;
- successful `NO_DECISION_REQUIRED`, `DECISION_REQUIRED`, `DECISION_ERROR`, and `RESOLVED` semantics remain compatible unless the implementation plan explicitly refines them.

**Progress:** accepted and designed conceptually; implementation not started.

### Feedback 2 — aggregate gate precondition diagnostics

Current problem:

- required gate conditions are exposed sequentially;
- a coding agent may mutate repository state to satisfy one condition only to discover another condition that makes gate execution impossible.

Target:

- only after clarification analysis establishes that a decision is actually required, perform a read-only gate preflight;
- report all locally knowable missing preconditions together, including where applicable GitHub origin, checked-out branch/commit, Control Plane URL, and Control Plane token;
- do not require Control Plane configuration when the selected component is already declared and the result is `NO_DECISION_REQUIRED`;
- do not turn preflight into a network health check.

**Progress:** accepted and designed conceptually; implementation not started.

### Feedback 3 — explicit first-adoption path

Current problem:

- PTSIP already defines component discovery, classification decision, boundary declaration, and Project Profile validation conceptually;
- the Reference Tool lacks a clear local first-adoption entry point that lets a project owner turn a discovered candidate into an authoritative Project Profile declaration without first requiring the Decision Control Plane.

Target:

```text
ptsip inspect
    -> discovered component candidate
    -> explicit project-owner architecture facts
    -> ptsip adopt
    -> validated Project Profile declaration
```

The project owner supplies architecture intent; PTSIP supplies observed candidate identity/scope. The Tool must not infer PRODUCT, TOOLCHAIN, or NEUTRAL_CONTRACT automatically from names or locations.

**Progress:** full implementation plan completed; implementation not committed.

## 3. Planned 0.3.3 session order

The current intended implementation order is:

```text
Session 1 — Explicit Profile Consistency
    Feedback 3 prerequisite

Session 2 — Complete ptsip adopt
    Feedback 3 completion

Session 3 — Gate Machine Interface Hardening
    Feedback 1 + Feedback 2

Session 4 — Integration, Documentation, Release Verification
    all three feedback items
```

Each session must be completed and verified before the next session begins.

## 4. Session 1 preparation state

Session 1 has been designed and a candidate patch/test set has been prepared outside `main`, but it has **not** been applied or committed to the repository.

Prepared change scope:

```text
src/ptsip/clarification/generator.py
src/ptsip/cli.py
tests/test_clarify_gate_profile_033.py
```

Prepared behavior:

- `_declared_components(root, profile_path=None)` forwards an explicit path to `find_profile()`;
- `analyze_clarifications(..., profile_path=None)` supports an explicit Project Profile;
- `ptsip clarify` gains `--profile`;
- `ptsip gate` gains `--profile`;
- root `ptsip.yaml` default behavior remains unchanged;
- a non-root explicit profile can suppress unnecessary clarification/gate decisions;
- `gate --profile` can still short-circuit to `NO_DECISION_REQUIRED` without GitHub origin or Control Plane configuration when no decision is needed;
- clarification and validation are checked against the same explicit profile location.

Prepared patch metadata recorded during this work:

```text
candidate patch source commit id: 52ab01cc2bd2e958fbfad05b58b672096d8731fa
subject: feat(cli): recognize explicit --profile in clarify/gate (Tool 0.3.3 Session 1)
files changed: 3
planned additions/deletions: +178 / -6
```

This patch ID is preparation evidence only. It is not a commit currently present on PTSIP `main`.

### Important Session 1 test transition

The prepared Session 1 regression suite still contains a test for the current pre-Feedback-1 behavior:

```text
gate --json with an undeclared component and no GitHub origin
    -> exit 2
    -> human stderr contains "GitHub origin"
```

That expectation is intentionally valid only until Session 3.

When Feedback 1 is implemented, the test must be replaced with a structured JSON failure assertion and must no longer freeze free-form stderr as the final 0.3.3 contract.

## 5. Session 1 completion boundary when work resumes

Session 1 is not considered complete merely because the prepared patch exists.

Before moving to Session 2, resume from the current `main` HEAD and:

1. re-check the current remote `main` SHA;
2. verify whether intervening issue work changed `src/ptsip/clarification/generator.py`, `src/ptsip/cli.py`, profile resolution, or related tests;
3. rebase/reconstruct the prepared Session 1 change if necessary rather than blindly applying a stale patch;
4. run the focused Session 1 tests;
5. run the complete existing pytest regression suite;
6. verify CLI help for `clarify --profile` and `gate --profile`;
7. verify root-profile behavior remains unchanged;
8. commit Session 1 only after all of the above pass.

No cached assumption about repository SHA should be used when work resumes.

## 6. Session 2 planned completion boundary

Session 2 will implement the Feedback 3 core capability: `ptsip adopt`.

Planned reuse:

```text
ComponentCandidate / discover_component_candidates()
DecisionAnswer
validate_answer()
prepare_local_profile()
write_prepared_local_profile()
validate_profile()
repository snapshot comparison
```

Planned behavior:

- one discovered candidate per adoption invocation;
- no automatic classification;
- no user `--include` override in the initial 0.3.3 design;
- dry-run by default;
- explicit `--apply` for Project Profile mutation;
- support explicit `--profile`;
- refuse semantic conflicts;
- preserve existing declarations;
- refuse stale repository evidence;
- keep profile concurrent-change protection;
- report `ADOPTION_PLAN`, `ADOPTED`, `ALREADY_DECLARED`, `CONFLICT`, `UNKNOWN_COMPONENT`, or `STALE_EVIDENCE` as appropriate;
- after application, `validate`, `clarify --profile`, and `gate --profile` must recognize the same declaration.

Session 2 is complete only when the end-to-end adoption boundary is working, not when only the CLI parser or profile writer exists.

## 7. Session 3 planned completion boundary

Feedback 1 and Feedback 2 should be implemented together as one gate contract change rather than as two partial rewrites.

Planned gate flow:

```text
clarification analysis
        |
        +-- no decision needed
        |      -> NO_DECISION_REQUIRED
        |
        `-- decision needed
               -> collect locally knowable gate preconditions
               -> return all missing preconditions together
               -> otherwise execute Control Plane gate
```

Planned structured distinction:

```text
PRECONDITION_FAILED
GATE_ERROR
NO_DECISION_REQUIRED
DECISION_REQUIRED
DECISION_ERROR
RESOLVED
```

Exact final field names/schema remain implementation-time decisions, but the machine interface must not require free-form stderr parsing.

Preflight should cover locally knowable configuration/state only. HTTP failures, timeouts, malformed Control Plane responses, and server-side errors belong to runtime gate error handling, not precondition collection.

## 8. Session 4 planned completion boundary

Final integration must verify all three feedback items together.

Minimum end-to-end sequence:

```text
inspect
-> adopt dry-run
-> adopt --apply
-> validate
-> clarify --profile
-> gate --profile
```

Additional gate contract verification:

- `NO_DECISION_REQUIRED` does not require unnecessary Control Plane configuration;
- JSON gate failures remain machine-readable;
- multiple missing gate preconditions are reported together;
- runtime Control Plane failures are distinct from local preconditions;
- existing Decision Control Plane first-valid-resolution-wins semantics are unchanged;
- Tool 0.3.2 topology classification-preservation behavior remains unchanged.

Release work then includes version identity, `releasenote/0.3.3.md`, README/adoption documentation alignment, full pytest, CLI identity/help checks, package build, and `twine check`.

## 9. Explicitly not completed yet

At this checkpoint, none of the following should be represented as implemented:

- `ptsip adopt`;
- Tool version `0.3.3` in package/constants;
- `clarify --profile` on `main`;
- `gate --profile` on `main`;
- structured `gate --json` failure output;
- aggregate gate precondition diagnostics;
- `ptsip-adoption/v1` final contract;
- Tool 0.3.3 release note;
- Tool 0.3.3 release tag;
- Tool 0.3.3 GitHub Release;
- Tool 0.3.3 PyPI publication.

## 10. Resume point

When Tool 0.3.3 work resumes, use this checkpoint and the canonical plan together:

```text
planning/PTSIP-TOOL-0.3.3-EXPLICIT-PROJECT-ADOPTION-PLAN.md
planning/PTSIP-TOOL-0.3.3-PROGRESS-CHECKPOINT-2026-08-11.md
```

Resume at **Session 1 verification/reapplication**, not Session 2.

The prepared Session 1 patch may be used as a reference only after checking the then-current `main` SHA and affected-file changes. Do not assume the patch remains directly applicable after unrelated issue work.
