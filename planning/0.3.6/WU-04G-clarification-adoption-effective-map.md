# WU-04G — Clarification / Adoption Effective-Map Integration and Decision-Protocol Upgrade

> **Status:** ACTIVE — maintainer decisions D1-B through D9-B accepted; implementation not yet started  
> **Parent work unit:** WU-04 — template catalog + deterministic materialization + effective-map consumers  
> **Entry branch:** `tool-0.3.6-lifecycle-ownership`  
> **Entry predecessor:** WU-04F — conformance consumes the effective Responsibility Map  
> **Entry baseline:** `52a455115d191123504c2fd690ffe499caf0ff6a`  
> **Pre-entry review:** `planning/0.3.6/pre-entry-WU-04G-decision-review.md`  
> **Bound Specification snapshot at entry:** `82abd09360df09a95fbbfb516855fa9ffb49f050`

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

This package intentionally expands WU-04G beyond the provisional read-only recommendation. The expansion is accepted and must be implemented without weakening the authority boundaries frozen by WU-04C.

## 2. Non-negotiable authority boundary

WU-04G must preserve all of the following:

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
    -> may change source mode template -> hybrid when required to represent the accepted decision

materializer
    -> only reproduces the resulting declaration deterministically
```

Discovery, path heuristics, confidence, candidate identity, or the materializer itself MUST NOT trigger a source-mode conversion.

## 3. G0 — normative authority freeze before runtime implementation

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

Clarification/adoption coverage must use:

```text
selected profile
    -> validate_profile()
    -> ValidationResult.resolved_profile
    -> ResolvedProfile.effective_payload
```

Effective components and effective associated artifacts are the only architecture coverage authority for explicit/template/hybrid source modes.

Raw source remains available for source identity, provenance, projection, and safe-write decisions, but it is not a second coverage interpretation.

## 5. D2-B — fail closed plus recovery workflow

If parse/schema/binding/materialization/effective semantic/selector validation cannot produce a valid `ResolvedProfile`, clarification/adoption must not fall back to raw source or manufacture candidate questions.

The failure path is extended beyond a terminal error:

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

The remediation surface may explain required corrections but MUST NOT infer architecture or silently repair declarations. Partial effective state from a failed attempt must not be reused as authority on retry.

Public status names may be finalized during implementation, but failure and recovery state must be machine-readable enough for local CLI and remote control-plane callers to continue deterministically after the project fixes the profile.

## 6. D3-B / D4-B — associated-artifact suppression and canonical selector coverage

Effective associated-artifact coverage suppresses duplicate component clarification for the same declared support scope. This does not permanently prevent later explicit promotion/re-evaluation if independent lifecycle responsibility emerges.

Clarification-local selector interpretation must be replaced by/reduced to a shared canonical candidate-coverage primitive owned by the validation/selector layer.

Required semantic result shape should distinguish at least:

```text
covered by one component
covered by one associated artifact
uncovered
ambiguous/conflicting
```

Ambiguity must fail/review rather than guess ownership. The shared primitive must preserve include/exclude/specificity semantics used by canonical validation rather than creating a second selector dialect.

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

Required migration behavior:

- newly generated clarification requests/answers use v2;
- new canonical decision storage uses v2 semantics;
- existing v1 data may be read only through an explicit compatibility/migration path where needed for already-persisted decisions;
- v1 `lifecycle_owner` must never override canonical `classification`;
- legacy PTSIP `TOOLCHAIN` remains migration input only and is not automatically translated to `DEVELOPMENT_TOOLING`;
- replacement tests must prove the v1-to-v2 migration boundary rather than silently deleting historical coverage.

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

A conflict that prevents lossless safe apply remains fail-closed and must report the conflict.

## 9. D8-B — exact profile path through decision protocol and CAS

The selected Project Profile path becomes part of the decision/control-plane identity.

It must flow through:

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
- stale/retry logic must preserve the selected path;
- a changed selected profile path requires a new/rebound gate rather than applying an old decision to a different file;
- local and remote application semantics should converge on the same path identity.

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
WU-04G  ACTIVE
WU-04H  LOCKED
WU-04I  LOCKED
```

Rules:

1. WU-04H- and WU-04I-owned test migration is forbidden while those stages are LOCKED. Their migration happens only when the owning stage is entered.
2. Tests owned only by already completed WU-04A through WU-04F are not retroactively migrated inside G; their historical migration is deferred to a later dedicated pass.
3. **File-level participation overrides historical filename/version ownership for G:** if a current test file contains one or more contracts that directly exercise WU-04G behavior, the entire file enters the WU-04G structural migration scope.
4. Entering the whole file into structural migration scope does **not** transfer semantic ownership of unrelated contracts. Non-G contracts inside that file must remain behaviorally equivalent unless their own stage is active.
5. A mixed file may therefore be reorganized, deduplicated, and moved onto shared stateless builders as one file, while H/I, VPMS, topology, old conformance, or other non-G assertions inside it remain semantically frozen.
6. New H/I behavior, H/I test files, VPMS effective-map behavior, or WU-04I final-regression contracts must not be pre-created in G.

The intended long-term effect is deliberately front-loaded: early active stages may own several mixed historical files; later stages should inherit fewer unmigrated files as shared test infrastructure becomes cleaner.

### 10.2 Migration is distributed across G1-G5, not deferred to G5

WU-04G migration must happen with the functional track that causes the semantic/API change:

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

Each track must leave its own focused tests green before the next functional track begins. This is required so failures can be attributed to the functional change that caused the migration rather than to a late bulk refactor.

Before the first optimization, capture a diagnostic baseline without weakening coverage:

```text
python -m pytest -q --durations=30 --durations-min=0.05
```

The previous exact-SHA run remains a historical comparison point only:

```text
workflow run: 32240740753
job:          96030499443
source SHA:   48b75e699a592703e4e03a8462131e4932103677
pytest:       260 passed / 13 failed
wall time:    431.81s (0:07:11)
```

Runtime reduction is evidence of useful optimization, never permission to delete semantic coverage.

### 10.3 Authoritative file-level WU-04G migration set

The implementation agent may trust this list and does not need to perform another repository-wide ownership analysis before starting G.

The following existing files are in **whole-file structural migration scope** because at least one current contract directly exercises clarification/adoption/decision-protocol/profile-projection/profile-path behavior owned by WU-04G:

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

No additional WU-04G `test_*_036.py` file or new test directory should be created unless this stage document is amended.

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

`test_repository_self_profile_035.py` is intentionally included even though it also contains VPMS-facing assertions: it directly calls `analyze_clarifications()` for repository self-coverage. Structural migration is allowed for the file, but WU-04H semantics must not be added or altered.

`test_topology_032.py` is intentionally included because it directly exercises `DecisionAnswer`, local profile projection, and explicit non-root profile path behavior. The file may be structurally migrated to the G test infrastructure, but topology-specific migration outcomes are not to be redesigned or repaired under WU-04G.

### 10.4 Files excluded from WU-04G migration

The following are explicitly outside G structural migration even if a G production change makes them fail indirectly:

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

# H/VPMS remains locked; no G migration
tests/vpms/**
```

Do not rewrite an excluded test merely because it fails after a G change. First classify whether the failure is:

```text
G production regression
mechanical interface incompatibility
pre-existing known debt
later-stage semantic expectation
```

If a genuinely required edit falls outside the authoritative G set, amend this document before touching the test.

### 10.5 Whole-file migration meaning

For the eight existing files in Section 10.3, the agent is authorized to clean the entire file rather than only the G-specific function.

Allowed whole-file structural migration includes:

- replace repeated Git init/config/commit helper implementations with one stateless helper/factory;
- replace repeated v2 answer/gate/profile payload construction with shared immutable builders;
- parameterize multiple cases that prove the same semantic contract;
- group tests by current responsibility inside the existing file;
- remove dead helper functions after all callers move to the shared support layer;
- move repeated immutable expected values/constants to the support module;
- reduce repeated expensive read-only setup by using a proven immutable module/class fixture where Section 10.6 permits it;
- split one historical test into clearer replacement contracts where v2 or profile-path semantics require it;
- merge historical tests only under the replacement rule in Section 10.7.

Whole-file migration does **not** authorize:

- changing unrelated topology/VPMS/conformance semantics;
- making a locked H/I behavior pass by implementing it early;
- deleting a test just because its filename refers to an older Tool version;
- sharing mutable state between tests whose isolation is evidence;
- changing workflow parallelism or using reduced-suite success as the authoritative gate.

### 10.6 Required sharing policy

The implementation agent should follow this policy directly rather than conducting another general fixture-architecture investigation.

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

The support module must not become a generic repository-wide test framework in WU-04G.

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

Recommended use: G1 explicit/template/hybrid **read-only effective-coverage** fixtures and repeated assertions over one immutable resolved/clarification result.

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

The following mappings are already decided and should not require rediscovery by the implementation agent:

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
    -> may be replaced only after all old assertions map into
    -> TestG4ProfilePathControlPlane::test_non_root_profile_path_survives_local_gate_and_reconciliation

test_topology_032.py::test_resolution_projection_respects_explicit_profile_path
    -> may be replaced only after all projection/path assertions map into
    -> TestG4ProfilePathControlPlane::test_non_root_profile_path_is_projection_target

test_repository_self_profile_035.py::test_repository_self_profile_resolves_all_discovered_candidates
    -> preserve or move structurally within the same file;
    -> it remains the repository dogfood clarification read contract and is not replaced by a synthetic fixture
```

All other tests in the eight-file G set default to **preserve**, though their helper/fixture structure may be migrated file-wide.

During implementation, append any additional merge/split mapping to the WU-04G migration ledger before deleting the old test.

## 11. Implementation tracks with mandatory file migration

### G0 — normative contract

Specification/registry rule for accepted-decision hybrid authority and new immutable `SPEC_REVISION`.

Test migration rule:

- no structural migration;
- update exact specification binding constants mechanically where required;
- do not use the revision change as a reason to repair completed A-F tests.

### G1 — effective read + selector coverage + file migration

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

Required work:

1. introduce/reuse the canonical selector coverage primitive;
2. make explicit/template/hybrid equivalent coverage one parametrized contract;
3. migrate associated-artifact suppression to effective-map coverage;
4. preserve partial-declaration semantics as a distinct contract;
5. move repeated immutable profile/component builders in the three whole-file targets to `_wu04g_support.py` where useful;
6. for repository self-profile tests, preserve VPMS-facing assertions unchanged while allowing file-wide helper cleanup;
7. use conditional shared read-only fixtures only under Section 10.6.

G1 focused tests and the three migrated files must pass their G-owned expectations before G2 starts.

### G2 — clarification-answer/v2 + file migration

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

Required work:

1. canonical new answers become v2 and omit `lifecycle_owner`;
2. keep one explicit v1 compatibility reader boundary rather than preserving v1 as canonical;
3. replace old canonical `TOOLCHAIN` success expectations with canonical rejection + legacy migration-only tests;
4. update CLI/render/GitHub/local-control payload expectations consistently;
5. migrate repeated v1/v2 answer/request builders out of the whole-file set;
6. preserve store/authority/first-valid-resolution semantics and state isolation;
7. topology-specific semantic tests remain frozen even though their shared DecisionAnswer/projection helpers may migrate.

G2 focused tests and all six participating migrated files must pass their G-owned interfaces before G3 starts.

### G3 — projection + automatic hybrid conversion + file migration

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

Required work:

1. preserve explicit adoption as the baseline path;
2. add isolated template->hybrid and existing-hybrid safe-apply cases;
3. assert exact template id/revision preservation;
4. assert only the accepted project-owned delta is written;
5. keep unrelated overrides/removals unchanged;
6. keep source conflict/no-write/atomicity cases independent;
7. every mutation test creates fresh repository/profile state; no module-scoped apply fixture;
8. `test_topology_032.py` may adopt the shared projection/profile builders, but its topology migration outcomes are not changed under G3.

G3 focused tests and all three participating files must pass their G-owned semantics before G4 starts.

### G4 — exact profile-path control plane + file migration

Production targets:

```text
src/ptsip/app/service.py
src/ptsip/app/store.py
src/ptsip/app/local_client.py
src/ptsip/app/github_authority.py / reconciliation surfaces as required
gate/CLI payload construction and GitHub CAS surfaces
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

Required work:

1. assert the normalized repository-relative selected profile path at every boundary that actually carries it: gate, store, retry/rebind, file read, projection, stale check, CAS;
2. preserve non-root profile behavior locally and remotely;
3. keep wrong-path, stale-revision, branch-head/CAS, first-valid-resolution, and declaration-conflict cases separate;
4. every stateful control-plane test gets isolated repository/store/authority state;
5. migrate repeated authority/control-plane payload builders, not mutable authority history;
6. topology file may migrate the explicit profile-path/projection test and helpers, but topology migration semantics remain unchanged.

G4 focused tests and all six participating files must pass their G-owned contracts before G5 starts.

### G5 — recovery + cross-track integration + final migration audit

Production responsibility:

- finish D2 post-failure recovery/retry behavior;
- integrate G1-G4 without introducing a second architecture interpretation.

Whole-file G migration targets participating in G5:

```text
tests/ptsip/test_adoption_033.py   # invalid/no-write/retry edge where directly required
```

Primary new focused structure:

```text
tests/ptsip/test_clarification_adoption_effective_map_036.py::TestG5RecoveryAndIntegration
```

G5 is not a general cleanup stage. Required work:

1. add invalid-profile recovery and fresh-retry integration coverage;
2. rerun all G1-G4 focused classes;
3. audit the complete eight-file migration ledger;
4. verify every removed/split/merged historical test has a named replacement;
5. verify all non-G semantic contracts in mixed files remained equivalent;
6. verify H/I and `tests/vpms/**` remain unentered/unmodified for migration purposes;
7. run duration diagnostics again and record before/after observations;
8. run one exact-SHA self-hosted complete repository regression and classify every remaining failure.

G5 must not reopen G1-G4 files for opportunistic performance cleanup unless a missing migration-ledger obligation is discovered.

## 12. Fixed structure of the new WU-04G focused tests

Create exactly:

```text
tests/ptsip/test_clarification_adoption_effective_map_036.py
tests/ptsip/_wu04g_support.py
```

Do not create a new WU-04G test directory. Pytest classes are namespaces only and must not hold mutable class-level state.

### 12.1 `test_clarification_adoption_effective_map_036.py`

Required class structure:

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

Prefer one parametrized implementation for the three equivalent-mode cases. Immutable read-only fixtures may be shared according to Section 10.6.

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

These tests should be pure/parser/store tests where possible; do not create Git repositories for answer validation that does not require repository semantics.

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

Do not merge the last four failure contracts into one generic conflict test.

#### `TestG5RecoveryAndIntegration`

```text
test_invalid_profile_blocks_clarification_without_raw_fallback
test_invalid_profile_exposes_selected_path_and_recovery_information
test_corrected_profile_retry_uses_fresh_resolved_profile
test_repeated_gate_does_not_reopen_effectively_declared_architecture
test_read_only_clarification_does_not_mutate_source_profile
test_cross_track_template_decision_becomes_effective_and_is_not_reasked
```

### 12.2 `_wu04g_support.py`

This file is stateless support code only. Preferred contents:

```text
init_fresh_git_repo(path, *, remote=None)
commit_all(repo, message="fixture")
canonical_v2_answer(...)
legacy_v1_answer(...)
clarification_request_payload(...)
gate_payload(..., profile_path=...)
component_payload(...)
associated_artifact_payload(...)
explicit_profile_payload(...)
template_profile_payload(...)
hybrid_profile_payload(...)
write_profile(path, payload)
```

Rules:

- helpers may create fresh state supplied by the calling test;
- helpers must not cache a mutable repository/store/authority globally;
- no module singleton `DecisionStore`, `MemoryAuthority`, `FakeGitHub`, or accepted decision;
- no automatic architecture inference inside fixtures;
- helper defaults must use canonical Tool 0.3.6 vocabulary.

## 13. Focused regression and migration evidence contract

WU-04G maintains a migration ledger throughout G1-G5:

```text
old test
    -> structural action
    -> replacement/preserved destination
    -> semantic reason
    -> focused verification result
```

After each track:

1. run the new focused class for that track;
2. run the existing whole-file migration targets participating in that track;
3. record newly replaced/merged tests in the ledger;
4. record any timing observation relevant to the migrated files;
5. do not wait until G5 to discover whether the migration preserved coverage.

The final G5 evidence must contain:

- before/after duration diagnostics;
- number/list of existing G-scope files migrated;
- migration ledger for every removed/split/merged test;
- confirmation that mutable-state isolation rules were preserved;
- confirmation that non-G semantics in mixed files were not altered;
- focused G1-G5 pass result;
- exact-SHA full repository regression result and classification of remaining failures.

## 14. Completion gate

WU-04G is complete only when:

- G0 normative accepted-decision authority is frozen and the Tool binds to the resulting immutable Specification snapshot;
- clarification/adoption reads use only validated effective architecture;
- invalid profiles fail closed and expose deterministic recovery/retry information;
- associated-artifact and selector coverage use canonical shared semantics;
- clarification-answer/v2 is canonical and `lifecycle_owner` is removed from new decisions;
- required v1 compatibility/migration handling is explicit and tested;
- accepted template/hybrid decisions apply as minimal project-owned hybrid changes without materialize-to-explicit writeback;
- exact selected profile path is preserved through local/remote decision state and CAS;
- all eight existing G-participating test files have undergone the file-level migration allowed by Section 10, with non-G semantics preserved;
- the new focused file follows the fixed G1-G5 class structure in Section 12;
- the G1-G5 migration ledger is complete;
- focused WU-04G tests pass;
- one exact-SHA self-hosted complete repository regression is reviewed and remaining failures are classified;
- WU-04H remains locked until this gate is reviewed.

## 15. Out of scope and locked/deferred migration

WU-04G does not authorize:

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

WU-04H MUST NOT be entered or pre-created before WU-04G completion is reviewed.