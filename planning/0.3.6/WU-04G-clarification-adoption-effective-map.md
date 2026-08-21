# WU-04G — Clarification / Adoption Effective-Map Integration and Decision-Protocol Upgrade

> **Status:** COMPLETE / EXACT-SHA VERIFIED — G0 COMPLETE; G1 VERIFIED; G2 VERIFIED; G3 VERIFIED; G4 VERIFIED; G5 COMPLETE / EXACT-SHA VERIFIED  
> **Parent work unit:** WU-04 — template catalog + deterministic materialization + effective-map consumers  
> **Entry branch:** `tool-0.3.6-lifecycle-ownership`  
> **Entry predecessor:** WU-04F — conformance consumes the effective Responsibility Map  
> **Entry baseline:** `52a455115d191123504c2fd690ffe499caf0ff6a`  
> **Pre-entry review:** `planning/0.3.6/pre-entry-WU-04G-decision-review.md`  
> **Bound Specification snapshot at entry:** `82abd09360df09a95fbbfb516855fa9ffb49f050`  
> **Current WU-04G normative Specification snapshot:** `d6995ed232e845b88d8235b851e80ab54b7804ea`  
> **Exact-SHA verification snapshot:** `3ee6bb1d8ecff3bbd6b2e63f50e4c9cde3fcd667`  
> **G5 closure evidence:** `planning/0.3.6/WU-04G-G5-verification-closure.md`

## 0. Current execution state

```text
G0  normative accepted-decision authority freeze   COMPLETE
G1  effective read + selector coverage             VERIFIED
G2  clarification-answer/v2                        VERIFIED
G3  hybrid safe apply                              VERIFIED
G4  exact profile-path control plane               VERIFIED
G5  recovery + integration + migration audit       COMPLETE / EXACT-SHA VERIFIED
```

G0 froze the accepted-decision authority rule as `PTSIP-RMAP-017` and selected immutable Specification revision:

```text
d6995ed232e845b88d8235b851e80ab54b7804ea
```

Tool constants and the repository root profile are bound to that revision.

G1 implementation and file migration were verified through the repository-native WU-04G track runner. The maintainer-provided OpenAI Local Bridge E2E execution produced:

```text
repository_id:                  Kinirin/PTSIP
task_id:                        wu04g-g1-files
run_id:                         fa46e93a3d7545b7bf793e3df21263c0
status:                         completed
exit_code:                      0
failure_kind:                   null
protocol_version:               2025-11-25
remote_task_execution_verified: true
log_available:                  true
log_truncated:                  false
capability_url_exposed:         false
```

The same track had previously exposed one test-fixture migration omission for a non-root profile parent directory; that fixture was corrected before the passing run. The passing Bridge run is the G1 files-mode verification evidence. It is not the final exact-SHA complete repository regression, which was completed in G5.

G2 implementation and migration were verified after isolating a frozen non-G topology contract from the G2 track selector. The maintainer-provided OpenAI Local Bridge verification produced:

```text
wu04g-scope
run_id:                         c57d0fd7adab45c39739685a6fca581d
status:                         completed
exit_code:                      0
failure_kind:                   null
protocol_version:               2025-11-25
remote_task_execution_verified: true
log_available:                  true
log_truncated:                  false
capability_url_exposed:         false

wu04g-g2-files
run_id:                         95f081cc5f2140cea719afea67a33b2a
status:                         completed
exit_code:                      0
failure_kind:                   null
protocol_version:               2025-11-25
remote_task_execution_verified: true
log_available:                  true
log_truncated:                  false
capability_url_exposed:         false
```

An earlier G2 run, `97def58217ec44fb8b1725ef7299d693`, reached pytest and reported `41 passed / 1 failed`; the sole failure was `test_topology_legacy_boundary_profile_preserves_historical_plane_classification`, a known legacy `0.3.4 boundaries/TOOLCHAIN` topology contract. Section 10 explicitly freezes topology semantics in this mixed file, so the test was neither repaired nor xfailed. The G2 files-mode selector was narrowed to the G2-owned `DecisionAnswer`/projection node while the legacy topology test remained available to later all-G and complete-regression gates. It reappeared in the final full regression and was classified as frozen non-G legacy debt.

G3 projection and accepted-decision hybrid safe apply were verified on final implementation SHA `cf663949bb7005616574db62a2d850bbc1d80bce`. The maintainer-provided OpenAI Local Bridge verification produced:

```text
wu04g-scope
run_id:                         fc9a315beafd485a88fa6d69772cc18a
status:                         completed
exit_code:                      0
failure_kind:                   null
protocol_version:               2025-11-25
remote_task_execution_verified: true
log_available:                  true
log_truncated:                  false
capability_url_exposed:         false

wu04g-g3-files
run_id:                         7993c2bb283b481e9775ea7a08ad45b2
status:                         completed
exit_code:                      0
failure_kind:                   null
protocol_version:               2025-11-25
remote_task_execution_verified: true
log_available:                  true
log_truncated:                  false
capability_url_exposed:         false
```

An earlier G3 files-mode run, `38988b39ba70466396697c350e214af7`, reached pytest at SHA `84d663dfbbfd5ae15a4d4c4cecea607066d291f2` and reported `21 passed / 1 failed`. The sole failure was the new template-adoption integration fixture: it selected the immutable `python-package-library` template without tracked `src/**` and `tests/**` files, so existing profile validation correctly failed closed before projection. The fixture was corrected by adding tracked template-owned `src/package.py` and `tests/test_package.py`; production safe-apply code was unchanged. Final scope and files-mode verification then passed on `cf663949...`.

G4 exact profile-path control-plane implementation was completed at production/test implementation SHA `471baba7656dbbb25d1a976d53bc20193b731def`. Repository-local Bridge invocation hardening then advanced the verification snapshot without changing G4 production semantics. The maintainer directly executed the repository-native scope guard and G4 files-mode runner at verification HEAD:

```text
c7f8060514a4814e5e6ccd9fd75280b2f931d54d
```

Direct verification produced:

```text
wu04g_guard.py scope
Scope base:    52a455115d191123504c2fd690ffe499caf0ff6a
Changed files: 44
Changed tests: 10
result:        PASS

wu04g_track_tests.py G4 --mode files
pytest:        37 passed
wall time:     79.23s (0:01:19)
result:        PASS
```

The selected profile path is now canonical repository-relative decision identity across local and hosted control-plane state. It is persisted with decisions, preserved across retry/rebind, used for exact local/remote projection targets and GitHub file reads/writes, included in non-root GitHub authority identity, and guarded against wrong-path resolution. Hosted stale/CAS races fail closed without applying the selected profile.

After pytest had already reported `37 passed`, Windows emitted an ignored pytest temporary-directory cleanup callback `PermissionError` for `%TEMP%\pytest-of-rhkrt\pytest-current`. This was post-test cleanup noise rather than a test failure and did not alter the G4 semantic result. OpenAI Local Bridge transport/acceptance instability encountered before the direct run is not G4 product evidence and is not a prerequisite for G5 entry; repository-native guards/runners are the stage gate.

G5 completed D2-B recovery/retry integration and the final WU-04G migration audit. `TestG5RecoveryAndIntegration` passed all six fixed contracts (`6 passed in 14.92s`). The final repository-native G1-G5 gate at exact candidate SHA `3ee6bb1d8ecff3bbd6b2e63f50e4c9cde3fcd667` passed `wu04g_guard.py all` with a clean working tree and reported `79 passed in 128.50s` for `wu04g_track_tests.py G5 --mode all-g`.

The required exact-SHA self-hosted complete repository regression was then run by GitHub Actions `tooling-test` run `32454141405`, job `96688080410`, on self-hosted Windows X64 / Python 3.14.6. The workflow checked out and verified the exact same SHA before running the complete repository pytest suite. Result: `303 passed / 8 failed in 295.61s`. All eight failures were reviewed and classified; none is a G1-G5 focused/selected WU-04G contract failure. Because the full repository pytest step failed, later Tool/Specification/profile verification, build/twine, wheel smoke, and `self-hosted/tooling-test` success-status recording were skipped. WU-04G exact-SHA verification is therefore stage-scoped and does not assert repository-wide regression cleanliness or release readiness.

This completes the WU-04G completion gate. WU-04H is now the next permitted stage but remains not entered and has no stage document yet. WU-04I remains locked.

## 1. Accepted maintainer decision package

The maintainer selected the following options for WU-04G:

```text
D1-B  ResolvedProfile.effective_payload is the clarification/adoption read authority
D2-B  invalid/unresolvable profiles fail closed, with post-failure recovery work extended
D3-B  effective associated-artifact coverage suppresses duplicate component questions
D4-B  selector/candidate coverage uses one shared canonical primitive
D5-B  create ptsip-clarification-answer/v2 and remove lifecycle_owner from the canonical answer
D6-B  accepted adoption decisions may extend hybrid and convert template -> hybrid when required
D7-C  eligible accepted decisions automatically perform that hybrid conversion/application
D8-B  exact selected profile path travels through the complete decision protocol and remote CAS
D9-B  optimize/migrate WU-04G-participating test files with coverage-preservation proof
```

This package intentionally expands WU-04G beyond the provisional read-only recommendation. The expansion is accepted and was implemented without weakening the authority boundaries frozen by WU-04C.

## 2. Non-negotiable authority boundary

WU-04G preserves all of the following:

- `classification` remains canonical lifecycle ownership;
- canonical PTSIP Tool 0.3.6 never restores `TOOLCHAIN` as a classification alias;
- template identity (`id + immutable revision`) remains project-selected authority;
- materialization remains deterministic and non-authoritative;
- an effective map is a runtime/read model and must never be dumped wholesale back as project-owned explicit source;
- associated artifacts remain non-component architecture;
- invalid/unresolvable profiles fail closed;
- VPMS remains outside WU-04G runtime implementation.

### Accepted-decision authority for D6-B / D7-C

Automatic `template -> hybrid` conversion is permitted only when there is an accepted project-owned clarification/adoption decision that requires a new source declaration.

The required authority chain is:

```text
repository evidence
    -> discovers candidate only
    -> does NOT authorize architecture

maintainer / authorized project decision
    -> supplies canonical component decision
    -> authorizes the project-owned declaration change

Tool safe-apply layer
    -> may encode that accepted decision as a hybrid project extension/replacement
    -> may change source mode template -> hybrid only to represent the accepted decision

materializer
    -> only reproduces the resulting declaration deterministically
```

Discovery, path heuristics, confidence, candidate identity, or the materializer itself MUST NOT trigger a source-mode conversion.

## 3. G0 — normative authority freeze before runtime implementation

**Execution status: COMPLETE.** `PTSIP-RMAP-017` is frozen and Tool/root binding uses immutable revision `d6995ed232e845b88d8235b851e80ab54b7804ea`.

D6-B and D7-C add a new normative meaning that is not explicitly frozen by the current `PTSIP-RMAP-013` through `PTSIP-RMAP-016`: an accepted project decision may authorize a source-mode transition and project-owned hybrid extension.

Therefore WU-04G begins with a normative precondition:

1. update the canonical Responsibility Map Specification/registry contract to define accepted clarification/adoption decisions as project-owned declaration authority for the exact accepted change;
2. state that automatic template -> hybrid conversion is allowed only as a representation of that accepted project decision;
3. retain exact template `id + immutable revision` during conversion;
4. prohibit materialize-to-explicit writeback;
5. prohibit unrelated implicit overrides/removals or cascading repair;
6. align canonical and embedded registry copies;
7. select a new immutable `SPEC_REVISION` only after the normative change is committed;
8. update Tool constants/root binding after that immutable revision exists.

Runtime implementation that depends on D6-B/D7-C MUST NOT precede this freeze.

## 4. D1-B — effective-map read authority

Clarification/adoption coverage uses:

```text
selected profile
    -> validate_profile()
    -> ValidationResult.resolved_profile
    -> ResolvedProfile.effective_payload
```

Effective components and effective associated artifacts are the only architecture coverage authority for explicit/template/hybrid source modes.

Raw source remains available for source identity, provenance, projection, and safe-write decisions, but it is not a second coverage interpretation.

## 5. D2-B — fail closed plus recovery workflow

If parse/schema/binding/materialization/effective semantic/selector validation cannot produce a valid `ResolvedProfile`, clarification/adoption does not fall back to raw source or manufacture candidate questions.

The implemented failure path is:

```text
profile resolution failure
    -> block architecture-dependent clarification/adoption
    -> preserve validation/materialization errors
    -> classify the failing stage
    -> expose the exact selected profile path
    -> expose a non-authoritative remediation/retry plan
    -> require project/source correction
    -> re-run validation against a fresh repository/profile snapshot
    -> resume clarification/adoption only after a valid ResolvedProfile exists
```

A missing profile remains a normal pre-adoption state and may proceed to clarification/adoption. An existing invalid/unresolvable profile fails closed with `PROJECT_CORRECTION_REQUIRED`. The remediation surface explains required corrections but does not infer architecture or silently repair declarations. Partial effective state from a failed attempt is not reused as authority on retry.

## 6. D3-B / D4-B — associated-artifact suppression and canonical selector coverage

Effective associated-artifact coverage suppresses duplicate component clarification for the same declared support scope. This does not permanently prevent later explicit promotion/re-evaluation if independent lifecycle responsibility emerges.

Clarification-local selector interpretation is replaced by/reduced to a shared canonical candidate-coverage primitive owned by the validation/selector layer.

The semantic result shape distinguishes at least:

```text
covered by one component
covered by one associated artifact
uncovered
ambiguous/conflicting
```

Ambiguity fails/reviews rather than guesses ownership. The shared primitive preserves include/exclude/specificity semantics used by canonical validation rather than creating a second selector dialect.

## 7. D5-B — clarification answer v2

WU-04G introduces:

```text
ptsip-clarification-answer/v2
```

Canonical v2 decision fields are:

```text
classification
purpose
shipped
runtime_required
executable
```

`lifecycle_owner` is removed from the canonical answer contract and is not serialized into canonical Tool 0.3.6 profiles.

Implemented migration behavior:

- newly generated clarification requests/answers use v2;
- new canonical decision storage uses v2 semantics;
- existing v1 data may be read only through an explicit compatibility/migration path where needed for already-persisted decisions;
- v1 `lifecycle_owner` never overrides canonical `classification`;
- legacy PTSIP `TOOLCHAIN` remains migration input only and is not automatically translated to `DEVELOPMENT_TOOLING`;
- replacement tests prove the v1-to-v2 migration boundary rather than silently deleting historical coverage.

WU-04G owns this decision-protocol migration because the maintainer explicitly selected D5-B.

## 8. D6-B / D7-C — automatic template -> hybrid safe apply

For an accepted decision requiring a declaration not already represented by the effective map:

### explicit source

Apply through the canonical explicit projection path.

### template source

The Tool may automatically transform:

```text
template(id, revision)
    -> hybrid(same id, same revision)
         + exact project-owned accepted extension/replacement
```

Only the minimum project-owned delta needed to represent the accepted decision may be written.

### hybrid source

The Tool may add/update the exact project-owned hybrid extension/replacement needed by the accepted decision while preserving unrelated overrides/removals.

### Prohibited writes

The Tool must not:

- serialize the entire effective map into explicit form;
- copy template-owned entities into project overrides merely because they exist;
- change the selected template revision;
- create unrelated removals/relationships;
- overwrite a conflicting existing project declaration silently;
- repair invalid downstream relationships heuristically.

A conflict that prevents lossless safe apply remains fail-closed and reports the conflict.

## 9. D8-B — exact profile path through decision protocol and CAS

The selected Project Profile path is part of the decision/control-plane identity.

It flows through:

```text
clarification/gate creation
    -> persisted decision record
    -> issue / agent resolution context
    -> exact subject revision
    -> remote file read
    -> projection validation
    -> branch-head stale check
    -> CAS write to the same exact profile path
```

Rules:

- store a normalized repository-relative profile path;
- reject paths outside the repository;
- do not silently substitute root `ptsip.yaml` when a different profile was selected;
- stale/retry logic preserves the selected path;
- a changed selected profile path requires a new/rebound gate rather than applying an old decision to a different file;
- local and remote application semantics converge on the same path identity.

## 10. D9-B — track-owned incremental test migration/optimization

### 10.1 Stage ownership rule

Test migration responsibility follows the active WU-04 sub-stage.

```text
WU-04A  COMPLETE
WU-04B  COMPLETE
WU-04C  COMPLETE
WU-04D  COMPLETE
WU-04E  COMPLETE / VERIFIED
WU-04F  COMPLETE / VERIFIED
WU-04G  COMPLETE / EXACT-SHA VERIFIED
WU-04H  NEXT / NOT ENTERED
WU-04I  LOCKED
```

Rules:

1. WU-04H- and WU-04I-owned test migration was forbidden while those stages were LOCKED. WU-04H may now be entered only through the normal fresh-HEAD/sub-document process; WU-04I remains locked.
2. Tests owned only by already completed WU-04A through WU-04F are not retroactively migrated inside G; their historical migration is deferred to a later dedicated pass.
3. **File-level participation overrides historical filename/version ownership for G:** if a current test file contains one or more contracts that directly exercise WU-04G behavior, the entire file enters the WU-04G structural migration scope.
4. Entering the whole file into structural migration scope does **not** transfer semantic ownership of unrelated contracts. Non-G contracts inside that file remain behaviorally equivalent unless their own stage is active.
5. A mixed file may therefore be reorganized, deduplicated, and moved onto shared stateless builders as one file, while H/I, VPMS, topology, old conformance, or other non-G assertions inside it remain semantically frozen.
6. New H/I behavior, H/I test files, VPMS effective-map behavior, or WU-04I final-regression contracts were not pre-created in G.

The intended long-term effect is deliberately front-loaded: early active stages may own several mixed historical files; later stages should inherit fewer unmigrated files as shared test infrastructure becomes cleaner.

### 10.2 Migration is distributed across G1-G5, not deferred to G5

WU-04G migration happened with the functional track that caused the semantic/API change:

```text
G1  effective read / selector coverage
    -> migrate all G-scope files participating in G1

G2  clarification-answer/v2
    -> migrate all G-scope files participating in G2

G3  projection / template->hybrid safe apply
    -> migrate all G-scope files participating in G3

G4  exact profile-path / local+remote control plane / CAS
    -> migrate all G-scope files participating in G4

G5  recovery + cross-track integration
    -> migrate G5-specific coverage only
    -> audit G1-G4 migration ledger
    -> no catch-all historical cleanup
```

Each track left its own focused tests green before the next functional track began so failures could be attributed to the functional change that caused the migration rather than to a late bulk refactor.

Before the first optimization, the historical exact-SHA run remained a comparison point:

```text
workflow run: 32240740753
job:          96030499443
source SHA:   48b75e699a592703e4e03a8462131e4932103677
pytest:       260 passed / 13 failed
wall time:    431.81s (0:07:11)
```

G5 duration diagnostics:

```text
pre-final G1-G4/G5 integration baseline: 73 passed in 124.29s
final G1-G5 all-G gate:                  79 passed in 128.50s
exact-SHA full repository regression:   303 passed / 8 failed in 295.61s
```

Runtime reduction is evidence of useful optimization, never permission to delete semantic coverage.

### 10.3 Authoritative file-level WU-04G migration set

The following existing files were in **whole-file structural migration scope** because at least one current contract directly exercised clarification/adoption/decision-protocol/profile-projection/profile-path behavior owned by WU-04G:

```text
tests/ptsip/test_clarification.py
tests/ptsip/test_adoption_033.py
tests/ptsip/test_decision_control_plane.py
tests/ptsip/test_local_control_plane_033.py
tests/ptsip/test_github_authority_033.py
tests/ptsip/test_github_authority_034.py
tests/ptsip/test_repository_self_profile_035.py
tests/ptsip/test_topology_032.py
```

New G-owned test/support files authorized by this plan:

```text
tests/ptsip/test_clarification_adoption_effective_map_036.py
tests/ptsip/_wu04g_support.py
```

The final G5 guard reported exactly these ten changed test files relative to the WU-04G entry baseline. No additional WU-04G `test_*_036.py` file or new test directory was created.

#### File responsibility map

| File | G track participation | Whole-file migration permission | Semantic restrictions |
| --- | --- | --- | --- |
| `test_clarification.py` | G1, G2 | yes | preserve remote parsing/discovery and read-only snapshot semantics |
| `test_adoption_033.py` | G1, G3, G4, G5 | yes | preserve stale-evidence, no-path-heuristic and legacy refusal semantics |
| `test_decision_control_plane.py` | G2, G3, G4 | yes | preserve first-valid-resolution, CAS conflict and retry isolation |
| `test_local_control_plane_033.py` | G2, G4 | yes | preserve LOCAL backend and per-test DecisionStore/state isolation |
| `test_github_authority_033.py` | G2, G4 | yes | preserve authority-head/history isolation and stale-clone semantics |
| `test_github_authority_034.py` | G2, G4 | yes | preserve read-only authority, reconciliation, unavailable-authority semantics |
| `test_repository_self_profile_035.py` | G1 | yes | clarification contract may change for G; VPMS/self-profile contracts remain semantically frozen |
| `test_topology_032.py` | G2, G3, G4 | yes | projection/path helpers may migrate; topology migration behavior remains semantically frozen |

`test_repository_self_profile_035.py` is intentionally included even though it also contains VPMS-facing assertions: it directly calls `analyze_clarifications()` for repository self-coverage. Structural migration was allowed for the file, but WU-04H semantics were not added or altered.

`test_topology_032.py` is intentionally included because it directly exercises `DecisionAnswer`, local profile projection, and explicit non-root profile path behavior. The file was structurally migrated to the G test infrastructure where appropriate, but topology-specific migration outcomes were not redesigned or repaired under WU-04G.

### 10.4 Files excluded from WU-04G migration

The following were explicitly outside G structural migration even if a G production change made them fail indirectly:

```text
# completed E/F and conformance/evidence families — defer historical migration
tests/ptsip/test_conformance_030.py
tests/ptsip/test_conformance_effective_map_036.py
tests/ptsip/test_conformance_engine_030.py
tests/ptsip/test_evidence_correctness_023.py
tests/ptsip/test_merge_gate_followup_030.py
tests/ptsip/test_merge_gate_remediation_030.py
tests/ptsip/test_profile_validation_036.py
tests/ptsip/test_remaining_030.py

# H/VPMS remained locked during G; no G migration
tests/vpms/**
```

Excluded-test failures were classified rather than rewritten under D9-B. The final full regression exposed `test_evidence_correctness_023.py`, `test_merge_gate_remediation_030.py`, and `test_remaining_030.py` failures plus non-G stale binding expectations; none was used as authorization for a catch-all G5 cleanup.

### 10.5 Whole-file migration meaning

For the eight existing files in Section 10.3, the agent was authorized to clean the entire file rather than only the G-specific function.

Allowed whole-file structural migration included:

- replace repeated Git init/config/commit helper implementations with one stateless helper/factory;
- replace repeated v2 answer/gate/profile payload construction with shared immutable builders;
- parameterize multiple cases that prove the same semantic contract;
- group tests by current responsibility inside the existing file;
- remove dead helper functions after all callers move to the shared support layer;
- move repeated immutable expected values/constants to the support module;
- reduce repeated expensive read-only setup by using a proven immutable module/class fixture where Section 10.6 permits it;
- split one historical test into clearer replacement contracts where v2 or profile-path semantics require it;
- merge historical tests only under the replacement rule in Section 10.7.

Whole-file migration did **not** authorize:

- changing unrelated topology/VPMS/conformance semantics;
- making a locked H/I behavior pass by implementing it early;
- deleting a test just because its filename refers to an older Tool version;
- sharing mutable state between tests whose isolation is evidence;
- changing workflow parallelism or using reduced-suite success as the authoritative gate.

### 10.6 Required sharing policy

The implementation followed this policy directly rather than conducting another general fixture-architecture investigation.

#### SAFE TO SHARE

Share code/data freely when immutable or stateless:

```text
canonical v2 answer dict builders
legacy v1 compatibility input builders
component/selector/associated-artifact payload builders
explicit/template/hybrid profile text or dict builders
gate/request payload builders
expected status/diagnostic constants
path normalization input cases
stateless git-init/commit helper functions that create a fresh repository per call
```

Shared definitions used by two or more G-scope files belong in:

```text
tests/ptsip/_wu04g_support.py
```

The support module did not become a generic repository-wide test framework in WU-04G.

#### CONDITIONALLY SHAREABLE

Share an actual object/result only when **all consumers are read-only and the input snapshot is identical**:

```text
initialized clean Git repository baseline
immutable selected profile
ValidationResult / ResolvedProfile
canonical selector-coverage result
read-only clarification analysis result
```

Required conditions:

1. no consuming test writes files, commits, mutates profile/source mode, changes branch/remotes, resolves/applies a decision, or changes authority/store state;
2. all consumers use the same environment/language/coordination mode and repository revision;
3. freshness/staleness/retry behavior is not the subject under test;
4. the fixture is test-owned (`tmp_path_factory` or equivalent), never the developer working repository;
5. when any condition is uncertain, use isolated function-scoped state.

G5 verification confirmed mutable store/authority/CAS/profile mutation cases remain function-scoped/fresh where required.

#### MUST NOT BE SHARED ACROSS SEMANTICALLY INDEPENDENT TESTS

```text
DecisionStore / SQLite database
MemoryAuthority or authority head/document history
FakeGitHub mutation/call history
accepted-decision winner state
CAS expected head / branch head
repository working tree used by apply/mutation
selected profile object/path when path mutation/rebind is under test
repository revision used for stale-evidence/retry
snapshot-before/snapshot-after mutation state
source profile mutated by template->hybrid conversion
monkeypatch/capsys-dependent process state
```

This state stays function-scoped/fresh unless a single test itself intentionally advances it through multiple steps. Isolation here is a semantic assertion, not redundant setup.

### 10.7 Consolidation and replacement rule

A test can be merged, split, moved, or removed only if its old semantic contract has an explicit destination.

Required condition:

```text
same/superseded semantic contract
+ replacement named before deletion
+ every material assertion mapped
+ mutable-state isolation preserved
+ replacement focused test passes
```

Do not collapse different failure causes merely because the public status is the same. In particular these remain independent:

```text
stale revision conflict
wrong selected profile path
existing declaration conflict
first-valid-resolution conflict
CAS branch-head conflict
invalid/unresolvable profile
```

### 10.8 Pre-authorized migration ledger entries

The following mappings were decided before implementation and were preserved:

```text
test_clarification.py::test_declared_component_purpose_suppresses_question
    -> replace/merge into
    -> TestG1EffectiveReadCoverage::test_equivalent_component_coverage_suppresses_clarification[explicit]

test_clarification.py::test_associated_artifact_scope_does_not_reopen_component_clarification
    -> replace/merge into
    -> TestG1EffectiveReadCoverage::test_effective_associated_artifact_suppresses_component_question

test_decision_control_plane.py::test_structured_answer_parser_and_validation
    -> split/replace with
    -> TestG2DecisionProtocolV2::test_v2_parser_accepts_canonical_answer_without_lifecycle_owner
       + TestG2DecisionProtocolV2::test_v1_reader_is_explicit_compatibility_only

test_decision_control_plane.py::test_toolchain_runtime_answer_is_conflict
    -> split/replace with
    -> TestG2DecisionProtocolV2::test_v2_rejects_toolchain_classification
       + TestG2DecisionProtocolV2::test_v1_toolchain_is_not_auto_translated

test_adoption_033.py::test_adopt_explicit_profile_is_seen_by_clarify_and_gate
    -> preserved while its profile-path semantics are additionally covered by
    -> TestG4ProfilePathControlPlane::test_non_root_profile_path_survives_local_gate_and_reconciliation

test_topology_032.py::test_resolution_projection_respects_explicit_profile_path
    -> preserved as the G-owned mixed-file projection node and additionally covered by
    -> TestG4ProfilePathControlPlane::test_non_root_profile_path_is_projection_target

test_repository_self_profile_035.py::test_repository_self_profile_resolves_all_discovered_candidates
    -> preserved;
    -> remains the repository dogfood clarification read contract and is not replaced by a synthetic fixture
```

No additional historical semantic contract was deleted in G5. Temporary G5 recovery coverage added to `test_adoption_033.py` remained additive while the fixed six-contract `TestG5RecoveryAndIntegration` class became the canonical focused G5 surface.

## 11. Implementation tracks with mandatory file migration

### G0 — normative contract

Specification/registry rule for accepted-decision hybrid authority and new immutable `SPEC_REVISION`.

Test migration rule:

- no structural migration;
- update exact specification binding constants mechanically where required;
- do not use the revision change as a reason to repair completed A-F tests.

### G1 — effective read + selector coverage + file migration

**Execution status: VERIFIED.** The production changes, participating file migrations, and `TestG1EffectiveReadCoverage` passed the `wu04g-g1-files` Bridge task with exit code 0 in run `fa46e93a3d7545b7bf793e3df21263c0`.

Production targets:

```text
src/ptsip/clarification/generator.py
src/ptsip/clarification/generator_core.py
src/ptsip/validation/components.py (or a narrow shared selector module)
```

Whole-file G migration targets participating in G1:

```text
tests/ptsip/test_clarification.py
tests/ptsip/test_adoption_033.py
tests/ptsip/test_repository_self_profile_035.py
```

New focused structure used in G1:

```text
tests/ptsip/test_clarification_adoption_effective_map_036.py::TestG1EffectiveReadCoverage
tests/ptsip/_wu04g_support.py
```

Required work completed:

1. introduce/reuse the canonical selector coverage primitive;
2. make explicit/template/hybrid equivalent coverage one parametrized contract;
3. migrate associated-artifact suppression to effective-map coverage;
4. preserve partial-declaration semantics as a distinct contract;
5. move repeated immutable profile/component builders in the three whole-file targets to `_wu04g_support.py` where useful;
6. for repository self-profile tests, preserve VPMS-facing assertions unchanged while allowing file-wide helper cleanup;
7. use conditional shared read-only fixtures only under Section 10.6.

### G2 — clarification-answer/v2 + file migration

**Execution status: VERIFIED.** Canonical v2 answer, explicit v1 compatibility boundary, canonical `TOOLCHAIN` rejection, v2 render/store/CLI/local/GitHub surfaces, and the G2 migration targets passed the repository-native G2 gate after preserving the mixed topology file's frozen non-G semantics. Scope run `c57d0fd7adab45c39739685a6fca581d` and files run `95f081cc5f2140cea719afea67a33b2a` both completed with exit code 0.

Production targets:

```text
src/ptsip/clarification/model.py
src/ptsip/clarification/resolution/model.py
src/ptsip/clarification/resolution/parser.py
src/ptsip/clarification/resolution/resolver.py
src/ptsip/clarification/render.py
transport/store/CLI answer surfaces as required
```

Whole-file G migration targets participating in G2:

```text
tests/ptsip/test_clarification.py
tests/ptsip/test_decision_control_plane.py
tests/ptsip/test_local_control_plane_033.py
tests/ptsip/test_github_authority_033.py
tests/ptsip/test_github_authority_034.py
tests/ptsip/test_topology_032.py
```

New focused structure used in G2:

```text
tests/ptsip/test_clarification_adoption_effective_map_036.py::TestG2DecisionProtocolV2
tests/ptsip/_wu04g_support.py
```

Required work completed:

1. canonical new answers became v2 and omit `lifecycle_owner`;
2. one explicit v1 compatibility reader boundary remains rather than preserving v1 as canonical;
3. old canonical `TOOLCHAIN` success expectations were replaced with canonical rejection + legacy migration-only tests;
4. CLI/render/GitHub/local-control payload expectations use consistent v2 semantics;
5. repeated v1/v2 answer/request builders were migrated where useful;
6. store/authority/first-valid-resolution semantics and state isolation were preserved;
7. topology-specific semantic tests remained frozen even though their shared DecisionAnswer/projection helpers participated structurally.

### G3 — projection + automatic hybrid conversion + file migration

**Execution status: VERIFIED.** Accepted project decisions now project through the explicit baseline or, when a new source declaration is required, through a minimal template/hybrid safe-apply delta. Template identity is preserved exactly, unrelated hybrid overrides/removals remain unchanged, materialize-to-explicit writeback is prohibited, existing project declaration conflicts fail closed, and no-op decisions do not mutate template/hybrid source. Scope run `fc9a315beafd485a88fa6d69772cc18a` and G3 files run `7993c2bb283b481e9775ea7a08ad45b2` both completed with exit code 0 on final implementation SHA `cf663949bb7005616574db62a2d850bbc1d80bce`.

Production targets:

```text
src/ptsip/clarification/resolution/profile_projection.py
src/ptsip/adoption.py
```

Whole-file G migration targets participating in G3:

```text
tests/ptsip/test_adoption_033.py
tests/ptsip/test_decision_control_plane.py
tests/ptsip/test_topology_032.py
```

New focused structure used in G3:

```text
tests/ptsip/test_clarification_adoption_effective_map_036.py::TestG3HybridSafeApply
tests/ptsip/_wu04g_support.py
```

Required work completed:

1. preserve explicit adoption as the baseline path;
2. add isolated template->hybrid and existing-hybrid safe-apply cases;
3. assert exact template id/revision preservation;
4. assert only the accepted project-owned delta is written;
5. keep unrelated overrides/removals unchanged;
6. keep source conflict/no-write/atomicity cases independent;
7. every mutation test creates fresh repository/profile state; no module-scoped apply fixture;
8. `test_topology_032.py` adopted shared projection/profile builders where useful without changing topology migration outcomes.

### G4 — exact profile-path control plane + file migration

**Execution status: VERIFIED.** G4 production/test implementation completed at `471baba7656dbbb25d1a976d53bc20193b731def`. After repository-local verification-entrypoint hardening, the maintainer directly verified the complete G4 files-mode scope at HEAD `c7f8060514a4814e5e6ccd9fd75280b2f931d54d`: `wu04g_guard.py scope` passed with 44 changed files / 10 changed tests and `wu04g_track_tests.py G4 --mode files` reported `37 passed in 79.23s`.

Production targets:

```text
src/ptsip/app/service.py
src/ptsip/app/store.py
src/ptsip/app/local_client.py
src/ptsip/app/github_authority.py / reconciliation surfaces as required
gate/CLI payload construction and GitHub CAS surfaces
src/ptsip/repository/profile_path.py
```

Whole-file G migration targets participating in G4:

```text
tests/ptsip/test_adoption_033.py
tests/ptsip/test_decision_control_plane.py
tests/ptsip/test_local_control_plane_033.py
tests/ptsip/test_github_authority_033.py
tests/ptsip/test_github_authority_034.py
tests/ptsip/test_topology_032.py
```

New focused structure used in G4:

```text
tests/ptsip/test_clarification_adoption_effective_map_036.py::TestG4ProfilePathControlPlane
tests/ptsip/_wu04g_support.py
```

Verified behavior:

1. normalized repository-relative selected profile path is carried through gate, store, retry/rebind, file read, projection, stale check, and CAS;
2. non-root profiles remain exact local and remote projection/write targets;
3. persisted decisions retain their selected path and historical missing-path records map only to root `ptsip.yaml` compatibility;
4. retry/rebind preserves the selected path while updating the expected revision;
5. root GitHub authority identity remains backward-compatible while non-root authority identity includes the canonical path;
6. a changed profile path cannot apply an old decision and returns `PROFILE_PATH_MISMATCH` locally;
7. stale subject revision and branch-head/CAS races fail closed with no selected-profile mutation;
8. stateful repository/store/authority/CAS cases remain isolated and the mixed topology file's non-G semantics remain frozen.

The G4 files-mode pytest set included the six participating existing files, the one G4-owned projection node from mixed topology, and `TestG4ProfilePathControlPlane`. All 37 selected tests passed. The post-pytest Windows temporary-directory cleanup warning for `pytest-current` is not a semantic test failure and is tracked separately from G4 product verification.

### G5 — recovery + cross-track integration + final migration audit

**Execution status: COMPLETE / EXACT-SHA VERIFIED.**

Production responsibility completed:

- D2 post-failure recovery/retry behavior;
- G1-G4 integration without introducing a second architecture interpretation.

Whole-file G migration target participating in G5:

```text
tests/ptsip/test_adoption_033.py   # invalid/no-write/retry edge where directly required
```

Primary focused structure:

```text
tests/ptsip/test_clarification_adoption_effective_map_036.py::TestG5RecoveryAndIntegration
```

G5 completion evidence:

```text
focused G5:
    6 passed in 14.92s

pre-final G1-G4/G5 integration baseline:
    73 passed in 124.29s

final exact candidate SHA:
    3ee6bb1d8ecff3bbd6b2e63f50e4c9cde3fcd667

wu04g_guard.py all:
    PASS
    working tree clean
    changed tests: 10, all in authoritative WU-04G set

wu04g_track_tests.py G5 --mode all-g:
    79 passed in 128.50s
```

Exact-SHA self-hosted complete repository regression:

```text
workflow run: 32454141405
job:          96688080410
source SHA:   3ee6bb1d8ecff3bbd6b2e63f50e4c9cde3fcd667
runner:       self-hosted Windows X64
Python:       3.14.6
pytest:       303 passed / 8 failed
wall time:    295.61s (0:04:55)
```

All eight remaining failures were classified. None was a focused/selected WU-04G contract failure. Detailed evidence and classification are recorded in `planning/0.3.6/WU-04G-G5-verification-closure.md`.

The complete repository workflow conclusion was `failure`; post-pytest Tool/Specification/profile verification, build/twine, wheel smoke, and `self-hosted/tooling-test` status recording were skipped. G5 exact-SHA verification therefore closes WU-04G only and does not establish repository-wide regression cleanliness or release readiness.

## 12. Fixed structure of the new WU-04G focused tests

Created exactly:

```text
tests/ptsip/test_clarification_adoption_effective_map_036.py
tests/ptsip/_wu04g_support.py
```

No new WU-04G test directory was created. Pytest classes are namespaces only and hold no mutable class-level state.

### 12.1 `test_clarification_adoption_effective_map_036.py`

Implemented class structure:

```python
class TestG1EffectiveReadCoverage:
    # G1 only
    ...

class TestG2DecisionProtocolV2:
    # G2 only
    ...

class TestG3HybridSafeApply:
    # G3 only
    ...

class TestG4ProfilePathControlPlane:
    # G4 only
    ...

class TestG5RecoveryAndIntegration:
    # G5 only
    ...
```

Required contract allocation:

#### `TestG1EffectiveReadCoverage`

```text
test_equivalent_component_coverage_suppresses_clarification[explicit]
test_equivalent_component_coverage_suppresses_clarification[template]
test_equivalent_component_coverage_suppresses_clarification[hybrid]
test_template_component_is_not_reasked
test_hybrid_override_selector_is_effective_selector
test_hybrid_removal_exposes_uncovered_candidate
test_effective_associated_artifact_suppresses_component_question
test_selector_ambiguity_fails_review_instead_of_guessing
```

#### `TestG2DecisionProtocolV2`

```text
test_v2_parser_accepts_canonical_answer_without_lifecycle_owner
test_v2_rejects_lifecycle_owner_field_when_contract_requires_exact_v2_shape
test_v2_rejects_toolchain_classification
test_v1_reader_is_explicit_compatibility_only
test_v1_toolchain_is_not_auto_translated
test_new_rendered_clarification_uses_v2
test_new_stored_decision_uses_v2_semantics
```

#### `TestG3HybridSafeApply`

```text
test_explicit_apply_remains_canonical_baseline
test_template_decision_converts_to_hybrid_preserving_template_identity
test_template_to_hybrid_writes_only_accepted_project_delta
test_existing_hybrid_decision_preserves_unrelated_overrides_and_removals
test_existing_project_declaration_conflict_fails_without_partial_write
test_invalid_decision_never_mutates_profile
```

Every test that mutates source profile state uses a fresh function-scoped repository/profile.

#### `TestG4ProfilePathControlPlane`

```text
test_non_root_profile_path_is_projection_target
test_non_root_profile_path_survives_local_gate_and_reconciliation
test_selected_profile_path_is_persisted_with_decision
test_selected_profile_path_survives_retry_and_rebind
test_remote_cas_reads_and_writes_exact_selected_profile_path
test_changed_profile_path_does_not_apply_old_decision
test_stale_revision_does_not_apply_to_selected_profile
test_branch_head_conflict_does_not_apply_selected_profile
```

The last four failure contracts remain independent rather than merged into one generic conflict test.

#### `TestG5RecoveryAndIntegration`

```text
test_invalid_profile_blocks_clarification_without_raw_fallback
test_invalid_profile_exposes_selected_path_and_recovery_information
test_corrected_profile_retry_uses_fresh_resolved_profile
test_repeated_gate_does_not_reopen_effectively_declared_architecture
test_read_only_clarification_does_not_mutate_source_profile
test_cross_track_template_decision_becomes_effective_and_is_not_reasked
```

All six passed in the focused G5 verification run.

### 12.2 `_wu04g_support.py`

This file is stateless support code only. It contains WU-04G-scoped fresh-repository/profile and canonical payload helpers. It does not cache mutable repository/store/authority state globally and does not infer architecture.

## 13. Focused regression and migration evidence contract

WU-04G maintained a migration ledger throughout G1-G5:

```text
old test
    -> structural action
    -> replacement/preserved destination
    -> semantic reason
    -> focused verification result
```

### G1 verification ledger

```text
test_clarification.py::test_declared_component_purpose_suppresses_question
    -> structurally migrated onto shared stateless WU-04G builders
    -> semantic replacement destination: TestG1EffectiveReadCoverage::test_equivalent_component_coverage_suppresses_clarification[explicit]
    -> focused/files verification: PASS via wu04g-g1-files run fa46e93a3d7545b7bf793e3df21263c0

test_clarification.py::test_associated_artifact_scope_does_not_reopen_component_clarification
    -> structurally migrated onto shared stateless WU-04G builders
    -> semantic replacement destination: TestG1EffectiveReadCoverage::test_effective_associated_artifact_suppresses_component_question
    -> focused/files verification: PASS via same run

historical partial-declaration raw-profile expectation
    -> replaced with valid-profile narrower-selector coverage contract
    -> semantic reason: D1-B/D2-B forbid schema-invalid raw source from acting as architecture authority
    -> focused/files verification: PASS via same run
```

### G2 verification ledger

```text
test_decision_control_plane.py::test_structured_answer_parser_and_validation
    -> split/replaced by canonical v2 parser + explicit v1 compatibility contracts
    -> destinations:
       TestG2DecisionProtocolV2::test_v2_parser_accepts_canonical_answer_without_lifecycle_owner
       TestG2DecisionProtocolV2::test_v1_reader_is_explicit_compatibility_only
    -> focused/files verification: PASS via wu04g-g2-files run 95f081cc5f2140cea719afea67a33b2a

test_decision_control_plane.py::test_toolchain_runtime_answer_is_conflict
    -> split/replaced by canonical TOOLCHAIN rejection + no-auto-translation legacy contract
    -> destinations:
       TestG2DecisionProtocolV2::test_v2_rejects_toolchain_classification
       TestG2DecisionProtocolV2::test_v1_toolchain_is_not_auto_translated
    -> focused/files verification: PASS via same run

test_topology_032.py::test_resolution_projection_respects_explicit_profile_path
    -> preserved as the G2-owned DecisionAnswer/projection interface in the mixed topology file
    -> topology migration outcomes remain semantically frozen
    -> focused/files verification: PASS via same run

legacy topology contract test_topology_legacy_boundary_profile_preserves_historical_plane_classification
    -> preserved unchanged; not a G2 semantic contract
    -> earlier mixed-file run 97def58217ec44fb8b1725ef7299d693 exposed it as the sole failure after 41 passes
    -> final exact-SHA full regression exposed the same contract again; classified as frozen non-G legacy debt
```

The G2 scope guard passed in run `c57d0fd7adab45c39739685a6fca581d` before the successful G2 files-mode run.

### G3 verification ledger

```text
explicit projection/adoption baseline
    -> preserved as canonical explicit safe-apply behavior
    -> destination: TestG3HybridSafeApply::test_explicit_apply_remains_canonical_baseline
    -> focused/files verification: PASS via wu04g-g3-files run 7993c2bb283b481e9775ea7a08ad45b2

template accepted-decision projection
    -> adds template->hybrid safe-apply path only when accepted project delta is required
    -> preserves exact template id/revision and writes only responsibility_map.overrides.components delta
    -> destinations:
       TestG3HybridSafeApply::test_template_decision_converts_to_hybrid_preserving_template_identity
       TestG3HybridSafeApply::test_template_to_hybrid_writes_only_accepted_project_delta
    -> focused/files verification: PASS via same run

existing hybrid project state
    -> preserves unrelated overrides/removals; conflicting project declarations fail closed without partial mutation
    -> destinations:
       TestG3HybridSafeApply::test_existing_hybrid_decision_preserves_unrelated_overrides_and_removals
       TestG3HybridSafeApply::test_existing_project_declaration_conflict_fails_without_partial_write
       TestG3HybridSafeApply::test_invalid_decision_never_mutates_profile
    -> focused/files verification: PASS via same run

test_adoption_033.py template CLI integration
    -> canonical v2 decision input; deprecated lifecycle_owner test input removed
    -> verifies persisted template->hybrid minimal project delta through adoption apply
    -> initial run 38988b39ba70466396697c350e214af7 failed only because the new fixture omitted tracked src/** and tests/** files required by the selected template
    -> fixture corrected at cf663949bb7005616574db62a2d850bbc1d80bce without production change
    -> focused/files verification: PASS via same final run

test_topology_032.py::test_resolution_projection_respects_explicit_profile_path
    -> preserved as the G3-owned explicit projection helper node in the mixed topology file
    -> topology migration semantics remain frozen
    -> focused/files verification: PASS via same run
```

The final G3 scope guard passed in run `fc9a315beafd485a88fa6d69772cc18a` before the successful G3 files-mode run.

### G4 verification ledger

```text
canonical selected profile-path identity
    -> added repository-relative normalization and repository containment checks
    -> root ptsip.yaml retains historical decision identity; non-root paths are deterministically path-bound
    -> destination: src/ptsip/repository/profile_path.py
    -> verification: PASS at c7f8060514a4814e5e6ccd9fd75280b2f931d54d

DecisionStore selected profile persistence
    -> profile_path persisted in DecisionRecord/SQLite with historical root default
    -> retry/rebind updates subject revision while preserving canonical selected path and accepted answer semantics
    -> destinations:
       TestG4ProfilePathControlPlane::test_selected_profile_path_is_persisted_with_decision
       TestG4ProfilePathControlPlane::test_selected_profile_path_survives_retry_and_rebind
    -> verification: PASS in 37-test G4 files-mode run

local selected-profile reconciliation
    -> gate and resolve carry the exact selected path; resolve without --profile reuses persisted decision path
    -> a different supplied profile fails with PROFILE_PATH_MISMATCH and no source mutation
    -> destinations:
       TestG4ProfilePathControlPlane::test_non_root_profile_path_survives_local_gate_and_reconciliation
       TestG4ProfilePathControlPlane::test_changed_profile_path_does_not_apply_old_decision
    -> verification: PASS in same run

hosted GitHub authority + remote CAS
    -> non-root authority identity includes canonical profile path while root identity remains compatible
    -> remote projection reads/writes the exact selected profile path
    -> stale subject revision and branch-head CAS race both fail closed without selected-profile write
    -> destinations:
       TestG4ProfilePathControlPlane::test_remote_cas_reads_and_writes_exact_selected_profile_path
       TestG4ProfilePathControlPlane::test_stale_revision_does_not_apply_to_selected_profile
       TestG4ProfilePathControlPlane::test_branch_head_conflict_does_not_apply_selected_profile
       test_github_authority_033.py::test_github_authority_profile_path_is_part_of_scope_identity
    -> verification: PASS in same run

test_topology_032.py::test_resolution_projection_respects_explicit_profile_path
    -> preserved as the G4-owned projection/path node in the mixed topology file
    -> historical topology migration semantics remain frozen and outside G4 files-mode selection
    -> verification: PASS in same run
```

### G5 verification ledger

```text
invalid existing profile fail-closed recovery
    -> added structured recovery state without treating a missing profile as invalid
    -> destinations:
       TestG5RecoveryAndIntegration::test_invalid_profile_blocks_clarification_without_raw_fallback
       TestG5RecoveryAndIntegration::test_invalid_profile_exposes_selected_path_and_recovery_information
    -> focused verification: 6/6 G5 contracts PASS

fresh retry after project correction
    -> failed attempt cannot reuse partial effective state
    -> destination: TestG5RecoveryAndIntegration::test_corrected_profile_retry_uses_fresh_resolved_profile
    -> focused verification: PASS

already-effective architecture
    -> repeated gate does not reopen an applied decision
    -> destination: TestG5RecoveryAndIntegration::test_repeated_gate_does_not_reopen_effectively_declared_architecture
    -> focused verification: PASS

read-only clarification
    -> source profile remains byte-for-byte unmodified by analysis
    -> destination: TestG5RecoveryAndIntegration::test_read_only_clarification_does_not_mutate_source_profile
    -> focused verification: PASS

cross-track template decision integration
    -> accepted decision becomes minimal hybrid project delta and effective coverage suppresses re-questioning
    -> destination: TestG5RecoveryAndIntegration::test_cross_track_template_decision_becomes_effective_and_is_not_reasked
    -> focused verification: PASS
```

Final migration/audit evidence:

```text
G5 focused:                    6 passed in 14.92s
G1-G5 all-G:                   79 passed in 128.50s
scope guard:                   PASS
working tree:                  clean
changed tests from entry base: 10, exactly authoritative G set + focused/support
H/I or tests/vpms migration:   none
```

Exact-SHA full repository regression and failure classification:

```text
run:        32454141405
job:        96688080410
source SHA: 3ee6bb1d8ecff3bbd6b2e63f50e4c9cde3fcd667
pytest:     303 passed / 8 failed in 295.61s
```

Failure classes:

```text
1  frozen legacy topology / 0.3.4 boundaries+TOOLCHAIN contract
3  excluded/deferred evidence / merge-gate / historical agent-conformance expectations
4  stale or out-of-scope Specification-binding fixture expectations
0  WU-04G focused/selected contract failures
```

The repository-wide workflow conclusion remained failure and no `self-hosted/tooling-test` success status was recorded. Full per-test classification is in `planning/0.3.6/WU-04G-G5-verification-closure.md`.

## 14. Completion gate

WU-04G completion gate was reviewed against exact-SHA evidence `3ee6bb1d8ecff3bbd6b2e63f50e4c9cde3fcd667` and is satisfied because:

- G0 normative accepted-decision authority is frozen and the Tool binds to the resulting immutable Specification snapshot;
- clarification/adoption reads use only validated effective architecture;
- invalid profiles fail closed and expose deterministic recovery/retry information;
- associated-artifact and selector coverage use canonical shared semantics;
- clarification-answer/v2 is canonical and `lifecycle_owner` is removed from new decisions;
- required v1 compatibility/migration handling is explicit and tested;
- accepted template/hybrid decisions apply as minimal project-owned hybrid changes without materialize-to-explicit writeback;
- exact selected profile path is preserved through local/remote decision state and CAS;
- all eight existing G-participating test files underwent the file-level migration allowed by Section 10, with non-G semantics preserved;
- the new focused file follows the fixed G1-G5 class structure in Section 12;
- the G1-G5 migration ledger is complete;
- focused WU-04G tests pass;
- one exact-SHA self-hosted complete repository regression was reviewed and all remaining failures were classified;
- WU-04H remained unentered throughout G and is only now eligible to become the next stage through the normal entry process.

Therefore:

```text
G5     COMPLETE / EXACT-SHA VERIFIED
WU-04G COMPLETE / EXACT-SHA VERIFIED
WU-04H NEXT / NOT ENTERED
WU-04I LOCKED
```

This stage-scoped exact-SHA verification must not be confused with repository-wide `tooling-test` success. Run `32454141405` concluded failure because eight classified non-G/deferred/historical tests remain; build/wheel smoke and success-status recording were skipped.

## 15. Out of scope and locked/deferred migration

WU-04G did not authorize or perform:

- WU-04H VPMS effective-map implementation or its stage-owned test migration;
- WU-04I final WU-04 regression-stage implementation or its stage-owned test migration;
- pre-creating H/I test files or migration documents;
- retrospective migration of files owned only by completed WU-04A through WU-04F;
- Tool 0.3.5 legacy architecture migration beyond the narrow clarification-answer v1 compatibility needed for D5-B;
- blind `TOOLCHAIN -> DEVELOPMENT_TOOLING` translation;
- topology behavior redesign even though the mixed `test_topology_032.py` file is structurally in G scope;
- VPMS semantic redesign even though the mixed `test_repository_self_profile_035.py` file is structurally in G scope;
- materialize-to-explicit writeback;
- repository-wide test-directory restructuring;
- pytest-xdist/worker-count/workflow optimization;
- replacing the final full regression with a reduced changed-file suite.

WU-04H may now be entered only by freshly reading the development branch HEAD, creating its stage document at that exact entry baseline, marking H ACTIVE, and then beginning H-owned implementation. WU-04I remains locked until H completes.
