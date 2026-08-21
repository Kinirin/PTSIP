# WU-04G G5 — Verification Closure Evidence

> **Stage:** WU-04G / G5 — recovery + cross-track integration + final migration audit  
> **Verification source SHA:** `3ee6bb1d8ecff3bbd6b2e63f50e4c9cde3fcd667`  
> **Branch at verification:** `tool-0.3.6-lifecycle-ownership`  
> **Exact-SHA self-hosted workflow:** `tooling-test` run `32454141405`, job `96688080410`  
> **Runner:** self-hosted Windows X64 (`DESKTOP-5HCCQIR`)  
> **Python:** `3.14.6`  
> **WU-04G stage conclusion:** COMPLETE / EXACT-SHA VERIFIED  
> **Repository-wide tooling-test conclusion:** FAILURE — not globally regression-clean, no `self-hosted/tooling-test` success status recorded

## 1. Repository-native G5 gates

The maintainer verified G5 on the exact implementation/documentation snapshot before this closure document was created.

Focused G5 contract run:

```text
source SHA: 9adb05ef498e7c6a94947814fb3c033c572f5d12
TestG5RecoveryAndIntegration
6 passed in 14.92s
```

The only difference between `b06e30450135e90a06d70cb2c8234999b0f431da` (focused-contract implementation) and `9adb05ef498e7c6a94947814fb3c033c572f5d12` was the maintainer-added `setup_dev.bat`; it did not modify G5 production or test semantics.

G1-G4 cross-track integration baseline before the final six G5 focused contracts were added:

```text
73 passed in 124.29s (0:02:04)
```

Final repository-native all-G gate at the exact candidate SHA:

```text
source SHA: 3ee6bb1d8ecff3bbd6b2e63f50e4c9cde3fcd667
wu04g_guard.py all: PASS
working tree: clean
changed tests: 10, all inside the authoritative WU-04G set
wu04g_track_tests.py G5 --mode all-g: 79 passed in 128.50s (0:02:08)
checkout source: repository-local src/
```

The final all-G set added six focused G5 contracts relative to the 73-test integration baseline. Wall time increased by 4.21 seconds while the selected semantic coverage increased by six tests. This is recorded as a duration diagnostic only; it is not used to justify coverage reduction.

## 2. G5 recovery/integration contracts

`TestG5RecoveryAndIntegration` implements and passed all six fixed contracts:

```text
test_invalid_profile_blocks_clarification_without_raw_fallback
test_invalid_profile_exposes_selected_path_and_recovery_information
test_corrected_profile_retry_uses_fresh_resolved_profile
test_repeated_gate_does_not_reopen_effectively_declared_architecture
test_read_only_clarification_does_not_mutate_source_profile
test_cross_track_template_decision_becomes_effective_and_is_not_reasked
```

The implemented D2-B recovery boundary is:

```text
missing profile
    -> normal pre-adoption state
    -> clarification/adoption may proceed

existing but invalid/unresolvable profile
    -> PROFILE_INVALID
    -> fail closed
    -> no raw-profile fallback
    -> expose exact selected profile path and failing stage
    -> PROJECT_CORRECTION_REQUIRED
    -> no reuse of partial effective state
    -> retry only against a fresh repository/profile snapshot
```

## 3. Migration audit

The complete WU-04G structural migration set remained the eight pre-authorized existing files:

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

Authorized new WU-04G test/support files:

```text
tests/ptsip/test_clarification_adoption_effective_map_036.py
tests/ptsip/_wu04g_support.py
```

Final guard report from the WU-04G entry baseline `52a455115d191123504c2fd690ffe499caf0ff6a` showed exactly these ten changed test files and passed scope enforcement. `tests/vpms/**` was not migrated or modified for WU-04G purposes, and no WU-04H/WU-04I implementation or test family was pre-entered.

The G1-G4 replacement ledger in the canonical WU-04G plan remains authoritative for every removed/split/merged historical test. G5 added recovery/integration coverage; it did not delete an additional historical semantic contract. Mutable store/authority/CAS/profile mutation cases remain function-scoped/fresh. The mixed topology file retained its historical topology behavior unchanged, and repository-self-profile VPMS-facing semantics were not redesigned.

## 4. Exact-SHA complete repository regression

GitHub Actions `tooling-test` run `32454141405`, job `96688080410`, was manually dispatched from `tool-0.3.6-lifecycle-ownership` and checked out exactly:

```text
3ee6bb1d8ecff3bbd6b2e63f50e4c9cde3fcd667
```

The workflow verified the checkout SHA before creating an isolated Python 3.14 virtual environment and installing `PTSIP==0.3.6` editable with dev dependencies.

Complete repository pytest result:

```text
303 passed / 8 failed
wall time: 295.61s (0:04:55)
workflow step: Run complete repository regression = FAILURE
```

Because that step failed, the later workflow steps were correctly skipped:

```text
Verify Tool, Specification, and repository profile   SKIPPED
Build and verify distributions                       SKIPPED
Reinstall built wheel and smoke public surfaces      SKIPPED
Record successful exact-SHA tooling verification     SKIPPED
```

Therefore this evidence does **not** claim repository-wide regression cleanliness, release readiness, successful distribution verification, or a `self-hosted/tooling-test` success status.

## 5. Classification of all eight remaining failures

None of the eight failures is a WU-04G focused/participating contract failure.

| Failure | Classification | WU-04G disposition |
| --- | --- | --- |
| `test_evidence_correctness_023.py::test_dependency_evaluator_reports_ran_instead_of_inferring_from_empty_findings` | excluded completed/deferred evidence family; legacy `0.3.4` / `TOOLCHAIN` fixture semantics | classify only; do not rewrite under D9-B |
| `test_merge_gate_remediation_030.py::test_ambiguous_release_workflow_blocks_lifecycle_coverage` | excluded completed/deferred merge-gate/lifecycle-evidence semantics | classify only; do not repair in G5 |
| `test_release_readiness_030.py::test_tool_036_package_runtime_and_spec_binding_match` | stale Specification-revision expectation (`82abd093...` vs current `d6995ed...`) outside authoritative G migration set | later/out-of-scope mechanical binding debt |
| `test_remaining_030.py::test_agent_decision_conflict_never_overrides_profile` | excluded historical agent/conformance expectation | classify only; no G5 semantic rewrite |
| `test_tool_identity_035.py::test_tool_036_preserves_specification_binding` | stale Specification-revision expectation outside authoritative G migration set | later/out-of-scope mechanical binding debt |
| `test_tooling.py::test_spec_identity` | stale Specification-revision expectation outside authoritative G migration set | later/out-of-scope mechanical binding debt |
| `test_tooling.py::test_component_profile_allows_specific_nested_override` | stale fixture/specification binding (`82abd093...`) rejected by current `d6995ed...` Tool binding | later/out-of-scope mechanical binding debt |
| `test_topology_032.py::test_topology_legacy_boundary_profile_preserves_historical_plane_classification` | explicitly frozen non-G legacy topology contract using `0.3.4 boundaries/TOOLCHAIN` semantics | preserved unchanged; known since G2 |

The exact failure summary contains no `TestG1EffectiveReadCoverage`, `TestG2DecisionProtocolV2`, `TestG3HybridSafeApply`, `TestG4ProfilePathControlPlane`, `TestG5RecoveryAndIntegration`, or other G-owned selected contract.

## 6. Closure interpretation

The canonical WU-04G completion gate requires one exact-SHA self-hosted complete repository regression to be reviewed and all remaining failures to be classified; it does not require the whole repository workflow to be green before WU-04G itself may close.

Accordingly:

```text
G0 COMPLETE
G1 VERIFIED
G2 VERIFIED
G3 VERIFIED
G4 VERIFIED
G5 COMPLETE / EXACT-SHA VERIFIED
WU-04G COMPLETE / EXACT-SHA VERIFIED
```

This is stage-scoped verification only. The repository remains globally regression-dirty at the verification SHA because eight non-G/deferred/historical failures remain, and the `tooling-test` workflow conclusion is `failure`.

WU-04H may become the next stage only after the canonical master/status documents record this reviewed WU-04G closure. WU-04I remains locked.
