# PTSIP Tool 0.3.3 — Explicit Project Adoption Plan

> **Status:** Planning / non-normative  
> **Target Tool version:** `0.3.3`  
> **Planning baseline:** `ba620456cd510cf1a056073647969b908697795b`  
> **Bound Specification family:** `0.2.0-draft`  
> **Bound Specification revision:** `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`  
> **Primary goal:** Provide a standard Tool-level path for an existing Consumer Repository to turn discovered component candidates into explicit project-owner-approved PTSIP architecture declarations without requiring the Decision Control Plane.

## 1. Why this plan exists

Consumer Repository feedback from Tool `0.3.1` identified an adoption gap that remains relevant after the Tool `0.3.2` topology and explicit `resolve --profile` improvements.

PTSIP already defines the required architecture model and adoption sequence conceptually:

```text
Component discovery
    -> Classification decision
    -> Boundary declaration
    -> Project Profile validation
```

The three architecture classifications remain:

- `PRODUCT`
- `TOOLCHAIN`
- `NEUTRAL_CONTRACT`

The missing Tool capability is a clear first-adoption entry point. A project owner should be able to inspect an existing repository, select a discovered component candidate, explicitly declare its architecture facts, validate the projected Project Profile, and intentionally apply that declaration.

This is a Tool adoption gap, not a request for a new architecture classification, new authority system, or new normative PTSIP model.

## 2. Release theme

Tool `0.3.3` is scoped around one release theme:

> **Explicit Project Adoption** — existing Consumer Repositories can establish their initial PTSIP Project Profile from discovered component candidates through explicit project-owner declarations.

The intended lifecycle is:

```text
Consumer Repository
        |
        v
   ptsip inspect
        |
        v
Component Candidate
status = UNKNOWN
        |
        v
Project Owner Declaration
PRODUCT / TOOLCHAIN / NEUTRAL_CONTRACT
        |
        v
   ptsip adopt
        |
        +-- candidate identity check
        +-- declaration semantic validation
        +-- profile projection
        +-- dry-run review
        +-- explicit --apply
        |
        v
PTSIP Project Profile
        |
        +-- ptsip validate
        +-- ptsip conform
```

The Decision Control Plane remains available for active coding-agent ambiguity resolution, but it is not a prerequisite for first adoption.

## 3. Authority boundary

Tool `0.3.3` MUST preserve a strict distinction between two entry points.

### `ptsip adopt`

Purpose:

- establish or extend a project-owned architecture declaration during PTSIP adoption;
- accept explicit architecture facts supplied by the project owner or an active user acting with project authority;
- operate locally without requiring a Control Plane decision record.

### `ptsip resolve`

Purpose:

- resolve a pending clarification/decision created through the coding-agent decision workflow;
- preserve first-valid-resolution-wins and Control Plane authority semantics;
- apply the authoritative resolved answer to the local Project Profile.

These paths may reuse the same validation and profile-projection implementation, but they MUST NOT be collapsed into one semantic authority path.

## 4. Proposed CLI surface

The primary new command is:

```text
ptsip adopt
```

Reference invocation:

```powershell
ptsip adopt . `
  --component tools-turbo-sdk `
  --classification TOOLCHAIN `
  --purpose "Repository-local Turbo SDK tooling" `
  --shipped no `
  --runtime-required no `
  --lifecycle-owner DEVELOPMENT_TOOLING `
  --executable yes `
  --profile ptsip.yaml `
  --json
```

The default operation is a dry-run.

Application requires an explicit flag:

```powershell
ptsip adopt . `
  --component tools-turbo-sdk `
  --classification TOOLCHAIN `
  --purpose "Repository-local Turbo SDK tooling" `
  --shipped no `
  --runtime-required no `
  --lifecycle-owner DEVELOPMENT_TOOLING `
  --executable yes `
  --profile ptsip.yaml `
  --apply `
  --json
```

### Initial argument set

Planned arguments:

- repository path;
- `--component <candidate-id>`;
- `--classification PRODUCT|TOOLCHAIN|NEUTRAL_CONTRACT`;
- `--purpose <text>`;
- `--shipped yes|no`;
- `--runtime-required yes|no`;
- `--lifecycle-owner PRODUCT|DEVELOPMENT_TOOLING|INDEPENDENT`;
- `--executable yes|no`;
- `--profile <path>`;
- `--apply`;
- `--json`.

Tool `0.3.3` SHOULD NOT initially add a user-supplied `--include` selector override. Candidate selectors are discovered repository evidence and should remain tied to the selected candidate unless a later explicit component-boundary editing feature is designed.

## 5. Existing implementation to reuse

Tool `0.3.3` should avoid parallel validation or Project Profile write paths.

### 5.1 Declaration model

Reuse the existing `DecisionAnswer` model for:

- classification;
- purpose;
- shipped state;
- Product runtime requirement;
- lifecycle owner;
- executable state.

### 5.2 Semantic validation

Reuse the existing `validate_answer()` rules so that adoption and decision resolution cannot drift semantically.

Current invariants include:

```text
PRODUCT
    -> lifecycle owner must be PRODUCT

TOOLCHAIN
    -> lifecycle owner must be DEVELOPMENT_TOOLING
    -> shipped must be false
    -> runtime_required must be false

NEUTRAL_CONTRACT
    -> executable must be false
    -> lifecycle owner must be INDEPENDENT
```

No second adoption-specific classification validator should be introduced unless the two workflows later gain genuinely different normative requirements.

### 5.3 Project Profile projection and write

Reuse the existing local profile projection pipeline:

```text
prepare_local_profile()
    -> project_payload()
    -> schema/profile validation
    -> PreparedLocalProfile

write_prepared_local_profile()
    -> concurrent-profile-change guard
    -> atomic replacement
```

The existing behavior already supports both:

- creating a new component-based Project Profile when one does not exist; and
- extending an existing profile while refusing conflicting declared facts.

The same writer should be shared by `adopt` and `resolve`.

## 6. Candidate evidence boundary

`ptsip adopt` must begin from a component candidate discovered by the existing inspection pipeline.

The selected candidate provides at least:

- candidate ID;
- include selectors;
- anchors;
- evidence IDs;
- current decision state.

The project owner provides architecture intent; the Tool provides observed candidate scope.

Conceptually:

```text
PTSIP observation
    -> candidate identity / include / anchors / evidence

Project owner authority
    -> classification / purpose / lifecycle facts
```

The Tool MUST NOT automatically infer `PRODUCT`, `TOOLCHAIN`, or `NEUTRAL_CONTRACT` merely from directory names, package names, or candidate anchors.

## 7. Dry-run and application transaction

### 7.1 Dry-run

The default invocation should be read-only and perform:

1. repository discovery;
2. initial repository snapshot capture;
3. inventory/dependency/component candidate discovery;
4. selected candidate existence validation;
5. existing Project Profile discovery/loading;
6. repository snapshot comparison after evidence collection;
7. declaration semantic validation;
8. Project Profile projection;
9. projected profile validation;
10. structured adoption-plan output.

A dry-run MUST NOT create or modify the Project Profile.

### 7.2 Apply

`--apply` should perform the same validated planning sequence and then:

1. verify that the repository evidence baseline has not changed since candidate analysis;
2. preserve the existing concurrent Project Profile change guard;
3. atomically write the prepared Project Profile;
4. rerun Project Profile validation;
5. re-evaluate the selected candidate against the applied declaration;
6. return the final adoption result.

The adoption layer should add a repository-level stale-evidence guard because the existing writer protects the profile file itself but does not by itself guarantee that the component candidate evidence remained unchanged during the operation.

## 8. Result semantics

The exact JSON schema may be finalized during implementation, but Tool `0.3.3` should establish a stable adoption result family such as:

```text
format: ptsip-adoption/v1
```

Planned status semantics:

- `ADOPTION_PLAN` — dry-run projection is valid and ready for explicit application;
- `ADOPTED` — declaration was applied and the resulting Project Profile validates;
- `ALREADY_DECLARED` — the selected candidate is already covered by an equivalent valid declaration and no rewrite is needed;
- `CONFLICT` — supplied architecture facts conflict with an existing declaration or semantic rules;
- `UNKNOWN_COMPONENT` — the supplied candidate ID is not present in the current repository evidence;
- `STALE_EVIDENCE` — repository state changed after candidate analysis and application is refused.

The output should also report remaining discovered candidates that still require architecture declaration where practical.

Example dry-run shape:

```json
{
  "format": "ptsip-adoption/v1",
  "status": "ADOPTION_PLAN",
  "apply": false,
  "candidate": {
    "id": "tools-turbo-sdk",
    "include": ["tools/turbo_sdk/**"]
  },
  "declaration": {
    "classification": "TOOLCHAIN",
    "purpose": "Repository-local Turbo SDK tooling",
    "shipped": false,
    "runtime_required": false,
    "lifecycle_owner": "DEVELOPMENT_TOOLING",
    "executable": true
  },
  "profile": {
    "path": "ptsip.yaml",
    "projected_valid": true
  }
}
```

## 9. Explicit profile-path consistency prerequisite

Tool `0.3.2` added explicit profile-path support to `resolve`, while `validate` and `conform` already accept explicit profiles.

The clarification/gate analysis currently discovers only the repository-root default profile. If Tool `0.3.3` allows adoption into a non-root Project Profile without correcting this, a declaration may be successfully adopted and later appear undeclared to `clarify` or `gate`.

Therefore Tool `0.3.3` must include the following consistency prerequisite:

```text
ptsip clarify --profile <path>
ptsip gate --profile <path>
```

Internal clarification analysis should accept the same explicit profile path and use it when loading declared components.

After this change, these commands should be able to operate against the same Project Profile location:

```text
adopt
resolve
validate
conform
clarify
gate
```

This is considered required adoption consistency, not an unrelated expansion of Tool `0.3.3` scope.

## 10. Multiple candidates

Tool `0.3.3` should keep one architecture decision per `ptsip adopt` invocation.

It should not introduce a second batch declaration file or adoption manifest format in this release.

For a repository with several candidates, adoption proceeds incrementally:

```text
candidate A -> adopt -> full Project Profile validation
candidate B -> adopt -> full Project Profile validation
candidate C -> adopt -> full Project Profile validation
```

Each application must preserve all previously valid declarations.

Structured output may expose remaining candidates requiring declaration, but partial adoption must never be misreported as full repository conformance.

## 11. Proposed implementation structure

The intended code impact is deliberately small:

```text
src/ptsip/
|-- adoption.py                         # new adoption orchestration
|-- cli.py                              # adopt command + profile plumbing
|-- clarification/
|   |-- generator.py                    # explicit profile support
|   `-- resolution/
|       |-- model.py                    # reuse DecisionAnswer
|       |-- resolver.py                 # reuse validate_answer
|       `-- profile_projection.py       # reuse profile projection/write
|-- validation/
|   `-- profile.py                      # reuse validate_profile
`-- constants.py                        # Tool 0.3.3 identity
```

Planned tests:

```text
tests/
|-- test_adoption_033.py                # new adoption contract tests
|-- test_decision_control_plane.py      # explicit-profile regressions
|-- test_tooling.py
`-- test_release_readiness_030.py
```

A cosmetic relocation of `profile_projection.py` is intentionally out of scope. Reuse is preferred over path/name refactoring that does not improve the adoption boundary.

## 12. Required verification scenarios

Tool `0.3.3` is not complete until the following scenarios are covered.

1. New repository with no Project Profile:
   - dry-run performs no write;
   - `--apply` creates a valid Project Profile.
2. `PRODUCT` adoption succeeds with valid Product ownership facts.
3. `TOOLCHAIN` adoption succeeds with valid Development Tooling ownership facts.
4. `NEUTRAL_CONTRACT` adoption succeeds only with valid non-executable independent ownership facts.
5. Semantically invalid facts are rejected before profile modification.
6. Unknown candidate ID is rejected without modification.
7. Equivalent existing declaration returns `ALREADY_DECLARED` without unnecessary rewrite.
8. Conflicting existing classification/facts return `CONFLICT` and preserve the existing declaration.
9. Adoption adds a new component to an existing profile without modifying unrelated components.
10. Explicit non-root `--profile` is supported.
11. `clarify --profile` recognizes the applied declaration and does not regenerate an unnecessary clarification.
12. `gate --profile` recognizes the applied declaration and does not create an unnecessary decision gate.
13. Repository evidence changes between analysis and application cause a stale-evidence refusal.
14. Concurrent Project Profile modification remains protected by the existing writer guard.
15. Adoption dry-run preserves repository non-intrusion.
16. Existing `resolve` decision-control-plane semantics remain unchanged.
17. Tool `0.3.2` topology classification-preservation behavior remains unchanged.

## 13. Implementation sessions

Work should proceed in complete sessions. Do not partially build a later layer before the current layer is verified.

### Session 1 — Explicit profile consistency

Scope:

- add explicit profile-path handling to clarification analysis;
- add `--profile` to `clarify`;
- add `--profile` to `gate`;
- add regression tests for root and non-root profiles.

Completion boundary:

```text
clarify/gate consistently recognize the same declaration used by
adopt/resolve/validate/conform.
```

### Session 2 — Complete `ptsip adopt`

Scope:

- candidate selection;
- `DecisionAnswer` construction;
- `validate_answer()` reuse;
- dry-run planning;
- repository stale-evidence guard;
- profile projection reuse;
- explicit `--apply`;
- post-write validation;
- idempotency and conflict handling;
- structured JSON result;
- complete `test_adoption_033.py` coverage.

Completion boundary:

```text
inspect -> adopt dry-run -> adopt --apply -> validate
works against a disposable Consumer Repository without the Control Plane.
```

### Session 3 — Tool identity and adoption documentation

Scope:

- `TOOL_VERSION = 0.3.3`;
- package version `0.3.3`;
- `releasenote/0.3.3.md`;
- README / README.ko adoption entry points;
- `adoption/ADOPTION-GUIDE.md` Tool workflow alignment.

Completion boundary:

Documentation, CLI help, package metadata, and Tool identity describe the same adoption contract.

### Session 4 — Release-boundary verification

Required verification:

```text
full pytest suite
ptsip --version
ptsip spec --json
ptsip adopt --help
ptsip clarify --help
ptsip gate --help
ptsip validate --help
ptsip conform --help
python -m build
python -m twine check dist/*
```

Also run an end-to-end disposable Consumer Repository pilot:

```text
inspect
-> adopt dry-run
-> adopt --apply
-> validate
-> clarify against the same profile
-> gate against the same profile
```

Hosted GitHub Actions verification should remain limited to the minimum final job set required to protect the release boundary.

## 14. Explicit non-goals for Tool 0.3.3

The following are intentionally excluded from this release:

- global `gate --json` failure-contract redesign;
- aggregate gate precondition diagnostics;
- a new Specification revision;
- a new Project Profile schema unless implementation proves an unavoidable compatibility defect;
- a fourth architecture classification;
- automatic/LLM ownership classification;
- directory-name-based ownership decisions;
- mandatory Decision Control Plane participation in first adoption;
- a new authority database or server-side adoption store;
- batch adoption manifests;
- topology migration behavior changes;
- conformance-rule changes;
- automatic reclassification of existing declarations.

The previously reported gate JSON and precondition-diagnostic issues remain valid backlog items, but they are not part of the Tool `0.3.3` release objective.

## 15. Release acceptance criteria

Tool `0.3.3` is acceptable only when all of the following are true:

1. A repository with no existing PTSIP Project Profile can establish one from a discovered candidate without a Decision Control Plane.
2. The Tool never silently chooses the component classification for the project owner.
3. Adoption reuses the existing decision semantic validator rather than introducing a divergent rule set.
4. Adoption reuses the existing validated profile projection/write path.
5. Dry-run is non-intrusive and application requires explicit `--apply`.
6. Existing declarations cannot be silently reclassified or overwritten by conflicting adoption input.
7. Explicit profile paths remain consistent across adoption, resolution, validation, conformance, clarification, and gate analysis.
8. Applied adoption results in a Project Profile that passes normal `ptsip validate` rules.
9. Existing Tool `0.3.2` topology semantics and Tool `0.3.1` decision-control-plane authority semantics regressions remain covered.
10. Tool `0.3.3` remains bound to the existing immutable `0.2.0-draft` Specification revision unless a separate normative decision explicitly changes that binding.

## 16. Decision summary

Tool `0.3.3` should solve the adoption boundary with the smallest new authority surface possible:

```text
observed candidate
        |
        v
explicit project-owner facts
        |
        v
shared semantic validation
        |
        v
shared Project Profile projection
        |
        v
reviewable dry-run
        |
        v
explicit apply
        |
        v
validated architecture declaration
```

This completes the missing first-adoption path while preserving the existing PTSIP architecture model, Project Profile schema, Decision Control Plane semantics, and Tool/Specification version separation.
