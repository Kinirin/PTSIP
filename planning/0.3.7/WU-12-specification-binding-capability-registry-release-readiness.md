# WU-12 — Typed Specification Binding, Capability Registries, Repository Adoption, and Final Release Readiness

> **Status:** COMPLETE / VERIFIED  
> **Target Tool:** `0.3.7`  
> **Predecessor:** WU-11 — PP-aware Specification / release-surface preparation (`COMPLETE / HANDOFF VERIFIED`)  
> **Architecture authority:** ADR-0023  
> **Exact entry baseline:** `328f4f163c36df746a28d5e7d2d77ebd58cb3b99`  
> **Predecessor normative baseline:** `555af435a4bb68140c2c869efa34d12c624d51a4`  
> **Predecessor implementation verification authority:** `9a9159685a4f5de103d79e9d1c38bdbbada25d4c` via `tooling-test` run `33081718965`, job `98550433277`  
> **Final Specification revision:** `3c47816770d194ae42f98faedc911d980db0e62a`  
> **Exact implementation verification authority:** `81f8657fe3522632b5bd0bcf1626e6888a51a1a6` via `tooling-test` run `33147876414`, job `98772842757`  
> **Release role:** Tool `0.3.7` release handoff ready; publication remains a separate explicit action

## 0. Purpose

Complete the long-term-maintainability Specification-binding architecture selected by the project owner without turning WU-10 historical migration bridges into a generic validation bottleneck.

WU-12 separates Tool, Project Profile, Specification, and Project Profile instance identities all the way through runtime validation, repository adoption, release verification, and final exact-SHA evidence.

WU-12 also adds a command-faithful validation-log capture surface for AI coding-agent workflows. The purpose is to eliminate manual copying of oversized validation stdout while preserving what was actually executed without inventing additional validation steps.

WU-12 entered `ACTIVE` under the standing successor-entry rule after WU-11 was recorded `COMPLETE / HANDOFF VERIFIED`. This entry state authorized execution of the already accepted ADR-0023 implementation scope; final publication remains outside WU-12 completion and requires a separate explicit release action.

The completed shape is:

```text
Tool 0.3.7
    ├─ explicit PP capabilities
    └─ explicit Specification capabilities

Project Profile pp.1.01
    └─ explicit SpecificationBinding
         family
         source
         immutable revision

historical migration bridge
    └─ used only when historical migration interpretation is actually required

validation command actually requested
    └─ command-faithful timestamped repository log
```

## 1. Typed Specification binding

Introduce a typed `SpecificationBinding` model with explicit fields for:

```text
family
source
immutable revision
```

The model must preserve exact identity and provide stable malformed/unsupported diagnostics.

Canonical current-generation Project Profiles SHALL no longer infer Specification family from `ptsip.version`.

Expected canonical shape:

```yaml
ptsip:
  version: "pp.1.01"
  specification:
    family: "0.3.7-draft"
    source: "https://github.com/Kinirin/PTSIP"
    revision: "<immutable-specification-revision>"
```

## 2. Specification capability registry

Add an explicit typed registry for Specification capabilities.

The registry must be operation-aware and must not become a second copy of the PP registry.

At minimum distinguish operations such as:

```text
IDENTIFY
VALIDATE
ANALYZE
CONFORM
CREATE_TARGET
```

where the implementation genuinely needs different support boundaries.

The registry must support exact immutable release binding while allowing deliberately supported historical/current families without requiring a migration bridge for ordinary validation.

Unknown family/source/revision combinations fail closed.

## 3. Independent PP / Specification capability composition

Keep:

```text
Tool -> PP capability
Tool -> Specification capability
```

as independent authorities.

Do not create a permanent Cartesian matrix for every Tool × PP × Specification combination.

Normal composition:

```text
require PP capability
    +
require Specification capability
    +
apply explicit cross-contract constraint only if one exists
```

Where a PP contract genuinely requires or excludes a Specification family/semantic feature, represent that relationship through a narrow typed compatibility rule rather than duplicating complete registry entries.

## 4. Generic validator responsibility separation

Remove the inherited rule:

```text
profile.specification.revision == Tool.SPEC_REVISION
```

from generic validation.

Generic validation must not consult historical Project Profile migration bridges merely to determine whether a current declared Specification binding is supported.

The validator pipeline should instead be:

```text
parse PP identity
    -> require PP VALIDATE capability
parse SpecificationBinding
    -> require Specification VALIDATE capability
schema / Responsibility Map validation
    -> optional narrow PP-Spec compatibility constraint
```

Historical migration readers remain separate.

## 5. Canonical schema and embedded contract changes

Synchronize public and embedded `pp.1.01` schema/specdata so canonical current-generation profiles can represent Specification family explicitly.

Requirements:

- add the canonical Specification family field under the PP contract;
- keep public and embedded schema bytes synchronized;
- do not mutate immutable historical `0.3.6-draft` schemas as though they originally contained the new field;
- retain historical-source readers/fixtures for actual legacy shape;
- update maintained examples/templates only under the current PP contract.

## 6. Historical source compatibility remains migration-only

WU-10 historical compatibility remains valid and revision-bound.

Examples include:

```text
0.3.4-draft @ b5b17dd... -> pp.1.01 semantic migration
0.3.6-draft @ d6995ed... -> pp.1.01 identity-only migration
```

These bridges are not deleted, but their responsibility is narrowed explicitly to historical migration interpretation.

Regression must prove that removing them from generic validation authority does not weaken historical migration safety.

## 7. PTSIP repository self-adoption

After the new binding model and validator are implemented, migrate the PTSIP repository root from its historical source identity to the current canonical state.

Intended final state:

```text
ptsip.yaml
    PP contract:          pp.1.01
    Specification family: 0.3.7-draft
    Specification source: https://github.com/Kinirin/PTSIP
    Specification revision: final WU-12 immutable normative snapshot
```

The root migration is explicitly authorized by the project-owner decision that created WU-12.

It must still preserve normal mutation safety:

- exact source state;
- schema validity;
- no unintended component/relationship/artifact/policy semantic delta;
- repository coverage remains valid;
- post-write validation succeeds.

Do not create `ptsip_0.3.7.yaml`.

## 8. Final Specification freeze

`555af435a4bb68140c2c869efa34d12c624d51a4` is the WU-11 PP-aware transition Specification baseline, not a permanently forced release revision.

Because WU-12 changes normative Specification-binding/schema surfaces, WU-12 must create or select a later immutable normative snapshot after all affected canonical assets are coherent.

Required chain:

```text
WU-11 transition baseline
    ↓
WU-12 typed binding + schema/validator contract
    ↓
coherent normative assets
    ↓
immutable final Specification snapshot
    ↓
Tool 0.3.7 SPEC_REVISION binding
    ↓
root pp.1.01 + 0.3.7-draft adoption
```

The final release `SPEC_REVISION` points to:

```text
3c47816770d194ae42f98faedc911d980db0e62a
```

The normative freeze commit is separate from later Tool/root/profile binding commits, avoiding self-referential Specification identity.

## 9. Release-contract verifier

Refactor release verification to check independent identities rather than numeric/equality coupling.

The release gate must verify at least:

```text
Tool package/runtime == 0.3.7
Tool bound Specification family == 0.3.7-draft
Tool bound SPEC_REVISION == immutable final WU-12 snapshot
current supported PP target == pp.1.01
root Project Profile == pp.1.01
root SpecificationBinding == supported 0.3.7-draft exact binding
release note namespace == releasenote/tool/0.3.7.md
public/embedded PP schema synchronization
```

Do not infer PP identity from Tool SemVer.

## 10. Workflow and package synchronization

Synchronize:

```text
.github/scripts/verify_release_contract.py
.github/workflows/tooling-test.yml
.github/workflows/release.yml
.github/workflows/tooling-release.yml
pyproject.toml
src/ptsip/constants.py
src/ptsip/spec_identity.py
releasenote/
STATUS.md
README.md / README.ko.md where user-visible identity requires it
```

Preserve self-hosted verification policy and exact-SHA release gates.

Publication artifact evidence must continue to preserve separate Product ownership for:

```text
ptsip-core
ptsip-evidence
ptsip-source-compat
ptsip-migration
ptsip-embedded-contracts
vpms
```

## 11. Command-faithful validation log capture for AI coding agents

Add a lightweight validation-output capture command whose responsibility is **evidence preservation**, not validation-step invention.

The command surface is:

```text
ptsip validation capture -- <exact command and arguments>
```

For example:

```text
ptsip validation capture -- npm run sdk:planes:doctor
```

The authority rule is fixed: PTSIP executes and captures only the command explicitly supplied for that capture invocation.

PTSIP MUST NOT maintain a fixed implicit list such as:

```yaml
validation:
  doctor: "npm run sdk:planes:doctor"
  validate: "npm run sdk:planes:validate"
  conform: "npm run sdk:planes:conform"
```

and MUST NOT infer that `validate` or `conform` were executed merely because `doctor` was executed.

If the only requested command is:

```text
npm run sdk:planes:doctor
```

then the evidence must describe only that command. No additional validation command is synthesized, scheduled, represented as completed, or represented as missing by PTSIP.

A portable implementation cannot recover the stdout/stderr of an arbitrary shell command after that command has already finished. Therefore the supported capture path owns the exact command execution. It MUST NOT scrape shell history and pretend that history proves captured output.

### 11.1 Repository log location

Captured output is stored under:

```text
docs/ptsip/validation/
```

Each capture creates a new timestamped `.log` file. Existing logs are not overwritten or removed by default.

Filename identity uses date/time only, not Tool version, PP version, Specification version, or release version. Example shape:

```text
docs/ptsip/validation/20260828T111700123+0900.log
```

The exact timestamp serialization must be filesystem-safe on supported platforms and sufficiently precise to avoid normal capture collisions.

Version-based filename partitioning is intentionally rejected because it would introduce a separate version-audit obligation for validation logs.

### 11.2 Minimal log metadata

Do not create or require `summary.md`.

Do not require a separate presentation document merely to explain the log set.

Each log itself contains only the minimal provenance required to understand the captured execution before the raw combined output, such as:

```text
captured_at: <date-time>
command: <exact executed command>
exit_code: <child process exit code>
--- output ---
<combined stdout/stderr>
```

The primary freshness signal is repository change itself: a newly added validation log means a capture occurred. The timestamp in the file name/log is the secondary confirmation of when that capture occurred.

### 11.3 Exit behavior

The complete stdout and stderr of the supplied command must be persisted even when the supplied command fails.

PTSIP keeps terminal output concise and points to the generated log rather than replaying the entire captured stream by default.

After evidence persistence and local commit handling complete, the capture command preserves the supplied command's exit status so automation does not turn a failed validation into a false success.

### 11.4 Local automatic commit, no automatic push

A successful evidence-write path automatically prepares a local Git commit for the newly created validation log.

The capture feature MUST:

- commit only the newly generated validation evidence path;
- never use broad staging such as `git add .`;
- preserve unrelated worktree and index changes;
- never push automatically;
- leave the local commit reversible by the user before any explicit push;
- report commit failure without deleting the generated log.

The auto-commit is evidence bookkeeping, not publication authority. Push remains an explicit separate user action.

### 11.5 Non-goals of capture

The capture feature must not:

- define a mandatory `doctor -> validate -> conform` workflow;
- infer missing validation stages from command names;
- claim that commands not executed were executed;
- overwrite or prune prior logs as part of ordinary capture;
- partition log filenames by Tool/PP/Specification version;
- create a required `summary.md`;
- push commits automatically;
- give captured logs architecture, migration, adoption, or release authority by themselves.

## 12. Focused tests

Add/organize focused tests for at least:

- typed SpecificationBinding parse/serialization;
- malformed/missing family/source/revision diagnostics;
- supported and unsupported Specification capability lookup;
- PP capability independent from Specification capability;
- narrow cross-contract constraint behavior;
- generic validator does not require historical bridge;
- generic validator does not require exact equality with Tool `SPEC_REVISION` unless the selected capability says so;
- historical source migration still requires explicit historical bridge;
- canonical pp.1.01 schema requires/accepts explicit family as selected by the final contract;
- immutable legacy schemas remain unchanged;
- root PTSIP adoption is identity/binding-safe and semantically stable;
- release contract reports Tool / PP / Specification identities independently;
- validation capture executes only the exact supplied command and does not synthesize `doctor`, `validate`, or `conform` neighbors;
- arbitrary command stdout/stderr is captured into one new timestamped log;
- capture of a failing command still persists the log and preserves the child exit code;
- repeated captures create new timestamped logs without overwriting/removing prior evidence;
- validation log filenames contain date/time identity but no Tool/PP/Specification version identity;
- validation capture does not require `summary.md`;
- auto-commit includes only the newly generated validation log, preserves unrelated worktree/index state, and does not push.

Touched/new tests continue the gradual purpose/role-based test organization rather than trigger a wholesale test migration.

## 13. Full regression and distribution verification

Run the complete repository regression against the exact final WU-12 source SHA.

At minimum verify:

- WU-09 PP identity tests;
- WU-10 direct-convergence / execution / recovery tests;
- WU-12 binding/capability tests;
- WU-12 validation-capture tests;
- repository root profile validation with zero unexpected coverage gaps;
- CLI `ptsip --version` and `ptsip spec` identities;
- package build/twine;
- Product Artifact exact-snapshot binding;
- installed-wheel smoke;
- no tests or local migration ledgers/checkpoints packaged;
- VPMS boundary remains unchanged unless separately authorized.

## 14. Exact-SHA release-readiness gate

The final self-hosted workflow must verify the exact dispatched SHA and record success only after all required tests, release-contract checks, build/artifact validation, and installed-wheel smoke pass.

Final authority:

```text
source SHA: 81f8657fe3522632b5bd0bcf1626e6888a51a1a6
workflow:   tooling-test
run:        33147876414
job:        98772842757
status:     self-hosted/tooling-test = success
```

Earlier WU-11 or WU-10 runs remain evidence for their own SHAs but do not substitute for this final WU-12 evidence.

Validation-capture commits are ordinary repository history and do not replace the final exact-SHA workflow authority.

## 15. Non-goals

WU-12 did not:

- redesign PP version grammar;
- invent new lifecycle classifications;
- make historical bridge availability equivalent to generic Specification support;
- create automatic migration authority from registry presence;
- require a Cartesian Tool × PP × Specification table where independent capability composition is sufficient;
- retroactively rewrite historical release/schema identities;
- turn validation capture into a fixed validation workflow registry;
- infer execution of validation commands that were not actually supplied;
- automatically push validation-log commits;
- publish Tool `0.3.7` as part of WU-12 completion.

## 16. Completion gate

WU-12 is complete because:

- typed SpecificationBinding exists and is used by canonical validation;
- Specification capability registry is explicit, typed, and operation-aware;
- PP and Specification capabilities are independently composed;
- historical migration bridge is no longer generic validation authority;
- current PP schema/specdata and maintained examples are synchronized;
- command-faithful validation capture persists exact supplied command output under `docs/ptsip/validation/` without inventing additional validation stages;
- validation logs use timestamp identity rather than version identity and ordinary capture does not overwrite/remove earlier logs;
- capture of failed validation commands still produces evidence and preserves the child exit code;
- validation capture creates no required `summary.md`;
- validation-log auto-commit is path-scoped and local-only, with no automatic push;
- PTSIP root is validly adopted to `pp.1.01` with explicit `0.3.7-draft` binding;
- final immutable WU-12 Specification revision is `3c47816770d194ae42f98faedc911d980db0e62a` and Tool `0.3.7` binds it;
- generic validation, historical migration, validation capture, and release verification pass their focused suites;
- complete regression passes at the exact final SHA;
- package/distribution/artifact/smoke verification passes;
- final self-hosted exact-SHA workflow succeeds;
- release documentation records Tool, PP, Specification, source SHA, and artifact identity independently;
- final release handoff is ready without Tool/PP/Specification coupling.

## 17. Completion evidence

The final exact-SHA verification authority is:

```text
Tool:                    0.3.7
Project Profile:         pp.1.01
Specification family:   0.3.7-draft
Specification revision: 3c47816770d194ae42f98faedc911d980db0e62a
Source SHA:              81f8657fe3522632b5bd0bcf1626e6888a51a1a6
Workflow run:            33147876414
Workflow job:            98772842757
Runner:                  DESKTOP-5HCCQIR / self-hosted Windows X64
Python:                  3.14.7
```

Verified results:

- release Specification contract: PASS; 20 release-bound Specification assets checked;
- complete repository regression: `505 passed / 0 failed`;
- root profile: `valid=true`, `errors=[]`, `warnings=[]`;
- canonical PP identity: `pp.1.01`;
- exact SpecificationBinding: `0.3.7-draft @ 3c47816770d194ae42f98faedc911d980db0e62a`;
- component selector conflicts: 0;
- unmatched selectors: 0;
- combined Responsibility Map coverage: `unassigned_count=0` (`275` component paths + `30` associated-artifact paths);
- build: `ptsip-0.3.7-py3-none-any.whl` and `ptsip-0.3.7.tar.gz` produced successfully;
- `twine check`: wheel PASS, sdist PASS;
- embedded-contract verification: PASS;
- Product Artifact exact-snapshot binding: PASS;
- verified workflow wheel SHA-256: `af894a04fadb32806c3ed8a4e94d51277ef948f87408027c7992b86153695971`;
- no blocking artifact-evidence gaps and no definite `PTSIP-PKG-001` violations were accepted by the artifact gate;
- artifact-aware conformance reported `INCOMPLETE`, which is an allowed non-global outcome for this artifact-specific gate; the required artifact-boundary and exact-snapshot evaluators ran successfully and their binding checks passed;
- built wheel reinstall: PASS;
- installed `ptsip --version`: `PTSIP Tool 0.3.7`;
- installed `ptsip spec`: Tool `0.3.7`, PP `pp.1.01`, Specification `0.3.7-draft @ 3c47816770d194ae42f98faedc911d980db0e62a`;
- VPMS smoke boundary: PASS;
- commit status `self-hosted/tooling-test=success` recorded on exact SHA `81f8657fe3522632b5bd0bcf1626e6888a51a1a6`.

This completion record is a documentation descendant of the verified implementation. It does not replace `81f8657fe3522632b5bd0bcf1626e6888a51a1a6` as the exact implementation verification authority, and it does not itself publish Tool `0.3.7`.