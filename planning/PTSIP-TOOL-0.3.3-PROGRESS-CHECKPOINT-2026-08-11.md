# PTSIP Tool 0.3.3 — Progress Checkpoint (2026-08-11)

> **Status:** Implementation / final integration checkpoint  
> **Checkpoint date:** `2026-08-11`  
> **Implementation branch:** `tool-0.3.3-github-authority`  
> **Base `main` SHA:** `d22b694e8f3ac1d06780b77e4f35e55c1aed3d77`  
> **Target Tool version:** `0.3.3`  
> **Bound Specification:** `0.2.0-draft` revision `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`

## 1. Scope transition

The earlier checkpoint recorded a paused planning-only state. Work subsequently resumed from freshly verified `main` SHA `d22b694e8f3ac1d06780b77e4f35e55c1aed3d77`.

The project then approved a new dependency order:

```text
Local DecisionStore
    -> GitHub-coordinated authority
    -> explicit ptsip adopt
    -> profile reconciliation
    -> Tool 0.3.3 integration/release verification
```

This explicitly supersedes the original plan's assumption that a new distributed authority would remain outside Tool 0.3.3. The amendment is recorded in:

```text
planning/PTSIP-TOOL-0.3.3-GITHUB-COORDINATED-AUTHORITY-AMENDMENT.md
```

Global `gate --json` exception-envelope redesign and aggregate gate-precondition diagnostics remain separate follow-up work and are not silently included in this migration.

## 2. Implemented Tool 0.3.3 source scope

### Explicit profile consistency

Implemented:

- `ptsip clarify --profile`;
- `ptsip gate --profile`;
- shared explicit profile resolution through clarification analysis;
- root `ptsip.yaml` remains the default;
- declared explicit profiles suppress unnecessary clarification/gate decisions.

### Explicit project adoption

Implemented:

- new `ptsip adopt` subcommand;
- one discovered candidate per invocation;
- explicit fixed architecture facts;
- no automatic Product/Toolchain/Neutral classification;
- dry-run by default;
- explicit `--apply` mutation boundary;
- deterministic answer validation;
- existing profile preservation/conflict checks;
- stale repository evidence check;
- prepared profile schema validation;
- concurrent profile-content guard;
- atomic profile write and post-write validation;
- `ptsip-adoption/v1` result shape;
- statuses including `ADOPTION_PLAN`, `ADOPTED`, `ALREADY_DECLARED`, `CONFLICT`, `UNKNOWN_COMPONENT`, `STALE_EVIDENCE`, and coordinated `ALREADY_RESOLVED`.

Non-GitHub adoption is a direct project-owner declaration and does not create a Local DecisionStore record merely to simulate a pending workflow.

### GitHub-coordinated authority

Implemented:

- default GitHub coordination for repositories whose origin resolves to GitHub owner/repository identity;
- dedicated `refs/heads/ptsip-policy` authority ref;
- automatic orphan/root authority bootstrap;
- `authority.json` ownership manifest;
- refusal to modify an existing same-named branch whose authority manifest is missing/incompatible;
- JSON decision records under `decisions/`;
- repository + normalized include-selector `gdec-*` identity independent of local clarification IDs;
- exact-parent Git commits and non-force ref updates for global compare-and-swap;
- first-valid-resolution-wins across clones/environments;
- action-time gate reconciliation into a stale clone's selected Project Profile;
- no continuous polling;
- `GH_TOKEN`/`GITHUB_TOKEN` cloud authentication plus authenticated `gh` fallback;
- fail-closed behavior when GitHub coordination is selected but unavailable;
- explicit `--coordination local|github` and hosted `--control-plane <URL>` selection.

### Local DecisionStore boundary

Preserved:

- Local DecisionStore under external `PTSIP_HOME`;
- no repository SQLite file;
- Local first-valid-resolution-wins behavior;
- non-GitHub repositories default to Local coordination;
- explicit GitHub `--coordination local` remains available for deliberately isolated operation.

### Specification binding

Preserved exactly:

```text
Specification: 0.2.0-draft
revision: a877b2f66a7f94c1b844c979e1b08fb08a9a8e45
```

A temporary implementation idea to persist `runtime_required` directly in the Project Profile schema was rejected because it would widen the immutable bound Specification contract. The schema/package copies remain unchanged. The fact is retained in decision authority records instead.

## 3. Tool identity and documentation

Completed in migration source:

- `src/ptsip/constants.py` -> Tool `0.3.3`;
- `pyproject.toml` -> package `0.3.3`;
- `releasenote/0.3.3.md`;
- README adoption/authority/backend/fail-closed documentation;
- `reference/DECISION-CONTROL-PLANE.md` rewritten for the three authority modes;
- `adoption/ADOPTION-GUIDE.md` updated for explicit adoption and multi-environment coordination;
- `STATUS.md` advanced to Tool 0.3.3 migration source;
- release-readiness Tool identity tests advanced to 0.3.3.

## 4. Added regression coverage

New tests cover:

- adoption dry-run performs no repository mutation;
- local adoption apply creates a valid project declaration without creating Local DecisionStore state;
- repeated adoption is idempotent;
- explicit non-root profile is recognized by adopt/clarify/gate;
- invalid architecture facts and unknown candidates do not write a profile;
- distinct clone-local clarification IDs with the same component scope converge on one `gdec-*` decision;
- first GitHub-coordinated winner cannot be replaced by a contradictory second answer;
- one clone can accept an architecture decision while another clone has not received a `ptsip.yaml` commit, and the second clone reconciles the winner at gate time;
- GitHub coordination does not create a Local DecisionStore in the stale clone.

Additional pre-merge tests are still being added for authority manifest collision, strict authority answer parsing, and the remaining adoption classification/conflict/stale scenarios.

## 5. Final integration work remaining

Before merge to `main`:

1. complete remaining focused regression cases;
2. finalize the one-job temporary Python 3.14 PR verification workflow;
3. run complete `pytest -q`;
4. verify Tool identity and exact Specification binding;
5. verify CLI help for `adopt`, `clarify`, `gate`, `resolve`, `validate`, and `conform`;
6. build wheel/sdist;
7. run `twine check`;
8. inspect PR diff for accidental Specification/schema changes;
9. verify current remote `main` has not moved unexpectedly;
10. merge only after the final verified head is clean.

The temporary verification workflow must not remain on `main` after migration completion.

## 6. Publication boundary

Completing and merging Tool 0.3.3 source does **not** authorize or imply:

- `tool-v0.3.3` tag creation;
- GitHub Release publication;
- PyPI publication.

Those remain separate release actions.
