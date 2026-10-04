# WU-08: repository-architecture 22개 실패 owner reconciliation

관찰일: 2026-10-04. 기준 commit: `02a3f6fd72607e512a1708867747f474e02426a8`.
검증 대상은 이 commit 이후의 **LOCAL_WORKING_TREE**이며 exact committed-SHA verification이 아니다.
이 문서는 아래 [WU-08 machine record](WU-08.yaml)의 비규범적 review projection이다. 정책, Project Profile, lifecycle 또는 binding authority를 새로 만들지 않는다.

## 결과와 범위

기록된 22개를 같은 test ID로 모두 다시 재현했다. 분류는 A 2개 / B 16개 / C 4개이며, A/B 18개는 수정 후 통과했다.
추가한 fail-closed guard 9개도 통과했다. 대상 재실행은 31개 중 27 PASS / 4 FAIL이다.
C 4개는 skip, xfail, 기대값 완화 또는 failure waiver 없이 실패로 유지한다.

| 기존 failure group | A | B | C | 원래 22개 재실행 |
| --- | ---: | ---: | ---: | --- |
| G1 Self-profile | 0 | 0 | 2 | 0 PASS / 2 FAIL |
| G2 Developer control plane | 1 | 3 | 0 | 4 PASS / 0 FAIL |
| G3 Work packet | 0 | 5 | 0 | 5 PASS / 0 FAIL |
| G4 Output identity | 0 | 1 | 0 | 1 PASS / 0 FAIL |
| G5 Support paths | 0 | 2 | 0 | 2 PASS / 0 FAIL |
| G6 Policy binding | 1 | 0 | 0 | 1 PASS / 0 FAIL |
| G7 Profile identity | 0 | 4 | 0 | 4 PASS / 0 FAIL |
| G8 Bridge / CLI | 0 | 1 | 2 | 1 PASS / 2 FAIL |

## Canonical owner와 결정 경계

owner는 사람 이름을 추정한 결과가 아니라 현재 machine routing 및 exact canonical artifact의 책임 경계다.
C의 artifact owner는 확인됐지만 architecture / retirement 의미는 미결정이다. owner가 DRAFT 또는 APPROVED / pending인 경우 이를 ACTIVE runtime authority로 사용하지 않는다.

| 참조 | 책임 owner / 후보 | exact canonical artifact / 구현 경계 | 상태·주의 |
| --- | --- | --- | --- |
| O1 | 저장소 Project Owner의 self-profile declaration | `developer/profiles/ptsip-repository.yaml` | routing으로 artifact owner 확인; partition 의미 결정은 미완료 |
| O2 | legacy transition evaluator의 격리 unit contract | `developer/automation/transition_evaluator.py` | 현재 branch에서 historical 0.4.0 계획을 활성 authority로 소비하지 않음 |
| O3 | canonical MPD/SFP index와 해당 policy record | `developer/policy/index.yaml`<br>`docs/Support_policy/policy/index.yaml` | 정확한 corpus / relation 검증; DRAFT 관계의 lifecycle 활성화 아님 |
| O4 | 개발자 authorization registry / evaluator | `developer/policy/registries/authorization-transition-registry.yaml`<br>`developer/automation/authorization_transition.py` | 등록된 preauthorized grant 조건을 변경하지 않음 |
| O5 | implementation workflow의 developer-only execution projection | `developer/automation/implementation_workflows.yaml`<br>`developer/automation/implementation_work_packet.py` | MPD-WORK-0002 ACTIVE; fixture는 live task 등록이나 mutation authority가 아님 |
| O6 | verification output metadata / 별도 verification semantics | `developer/policy/VERI/MPD-VERI-0006.yaml`<br>`developer/policy/VERI/MPD-VERI-0001.yaml` | VERI-0006 DRAFT 0.1; VERI-0001 APPROVED / pending transition. 둘 다 operational activation 아님 |
| O7 | canonical Support corpus / schema 및 runtime catalog projection | `docs/Support_policy/policy/index.yaml`<br>`docs/Support_policy/policy/schemas/ptsip-support-feature-policy.schema.json`<br>`developer/policy/schemas/management-policy.schema.json` | retired src/specdata 정책이나 schema를 복구하지 않음 |
| O8 | exact Policy Resolver binding control plane | `developer/policy/policy-resolver-bindings/registry.yaml`<br>`developer/policy/policy-resolver-bindings/bindings.jsonl` | canonical MPD가 normative authority; binding은 routing/projection만 담당 |
| O9 | Project Profile identity registry / public catalog / immutable baseline | `registry/project-profile-contracts.yaml`<br>`profiles/index.yaml`<br>`profiles/history/pp.1.02` | pp.1.02 CURRENT, pp.1.01 SUPERSEDED, transition SEMANTIC_MIGRATION을 기존 상태대로 검증 |
| O10 | root bridge 폐기와 self-profile / local-catalog 정합 결정 | `developer/profiles/ptsip-repository.yaml`<br>`ptsip.yaml`<br>`.ptsip/profiles/index.json`<br>`AGENTS.md` | artifact 경로 확인; 삭제·참조 제거 및 lifecycle 의미는 Project Owner 결정 필요 |
| O11 | Test Mode registry / resolver CLI | `.github/test_modes.yaml`<br>`.github/scripts/resolve_test_modes.py`<br>`developer/profiles/ptsip-repository.yaml` | 필수 manual --mode와 canonical default profile 모두 유지 |

## 22개 test ID별 판정

| # | Group | Test ID | 분류 | Owner | 실패 근거와 조치 | 재실행 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | G1 | `tests/ptsip/repository/test_self_profile.py::test_repository_self_profile_is_valid_complete_and_revision_pinned` | C OWNER_DECISION_REQUIRED | O1 | 5개 dead selector와 113개 미배정 경로; architecture 의미는 테스트가 정할 수 없음<br>→ selector 정리와 경로별 classification/roles/associated-artifact 선언 승인 대기 | FAIL / 결정 대기 |
| 2 | G1 | `tests/ptsip/repository/test_self_profile.py::test_repository_self_profile_resolves_all_discovered_candidates` | C OWNER_DECISION_REQUIRED | O1 | 같은 self-profile 때문에 clarification이 안정적 완료 상태에 도달하지 못함<br>→ 같은 architecture 결정에 종속; 기대 상태를 낮추지 않음 | FAIL / 결정 대기 |
| 3 | G2 | `developer/tests/test_developer_control_plane.py::test_removal_gate_tracks_e4_machine_completion` | B stale test / fixture | O2 | 폐기된 docs/planning/0.4.0 계획을 live fixture로 로딩<br>→ 격리 fixture로 E4 COMPLETE / VALIDATION_PENDING 분기 모두 검증 | PASS |
| 4 | G2 | `developer/tests/test_developer_control_plane.py::test_current_policy_relations_preserve_materialized_relation_set` | B stale test / fixture | O3 | 관계 집합에 index 순서를 강제하고 현재 3개 관계를 누락<br>→ 정확한 multiset 비교와 현재 3개 edge 반영; DRAFT는 비활성 유지 | PASS |
| 5 | G2 | `developer/tests/test_developer_control_plane.py::test_current_policy_indexes_cover_self_contained_corpus` | B stale test / fixture | O3 | family-ID 파일 정렬 순서와 canonical index 선언 순서를 동일시<br>→ ID 중복 검출 및 exact multiset corpus coverage 유지 | PASS |
| 6 | G2 | `developer/tests/test_developer_control_plane.py::test_developer_owner_authorization_uses_mpd_registry` | A 실제 구현 회귀 | O4 | authorization 구현에서 SFP 수를 21로 고정, 현재 index는 22<br>→ canonical selected IDs와 validated IDs의 비어 있지 않은 exact set/count 검증 | PASS |
| 7 | G3 | `developer/tests/test_implementation_work_packet.py::test_packet_builds_exact_mutation_acceptance_and_regression_plan` | B stale test / fixture | O5 | 미등록 live task_context를 unit input으로 기대<br>→ 기존 execution projection을 복사한 격리 fixture; live 등록 없음 | PASS |
| 8 | G3 | `developer/tests/test_implementation_work_packet.py::test_agent_brief_is_compact_and_progressive` | B stale test / fixture | O5 | 과거 live normative rule 수와 branch context에 의존<br>→ canonical rule의 bounded projection fixture와 compact brief 검증 | PASS |
| 9 | G3 | `developer/tests/test_implementation_work_packet.py::test_mutation_source_context_avoids_whole_file_read` | B stale test / fixture | O5 | 같은 미등록 task_context 의존<br>→ 선택적 function/method read fixture와 whole-file read 방지 검증 | PASS |
| 10 | G3 | `developer/tests/test_implementation_work_packet.py::test_read_context_supports_single_selector_projection` | B stale test / fixture | O5 | 같은 미등록 task_context 의존<br>→ 단일 selector projection 동작 검증 | PASS |
| 11 | G3 | `developer/tests/test_implementation_work_packet.py::test_read_context_ids_are_stable` | B stale test / fixture | O5 | 같은 미등록 task_context 의존<br>→ 동일 fixture에서 안정적 read-context ID 검증 | PASS |
| 12 | G4 | `developer/tests/test_policy_plan_output_schema_identity.py::test_output_policy_owns_output_semantics_not_verification_semantics` | B stale test / fixture | O6 | current DRAFT output 정책 0.1을 과거 0.0으로 기대<br>→ metadata 기대값만 0.1로 정합; DRAFT와 verification owner 경계 유지 | PASS |
| 13 | G5 | `developer/tests/test_policy_provenance_retirement.py::test_current_sfp_mpd_corpus_has_no_legacy_source_provenance` | B stale test / fixture | O7 | retired src/ptsip/specdata/SFP 및 legacy MPD subset을 canonical corpus로 기대<br>→ 두 canonical index의 모든 exact record path를 검증 | PASS |
| 14 | G5 | `developer/tests/test_policy_provenance_retirement.py::test_current_policy_schemas_do_not_define_legacy_source_provenance` | B stale test / fixture | O7 | 폐기된 SFP schema 경로를 기대<br>→ docs/Support_policy canonical schema와 runtime catalog projection 정합 검증 | PASS |
| 15 | G6 | `developer/tests/test_policy_resolver.py::test_policy_resolver_binding_plane_is_machine_valid` | A 실제 구현 회귀 | O8 | migration / MPD-0015 exact binding의 MPD-MIGR-0005 ID 중복<br>→ PLAN/MODIFY/VERIFY의 두 section을 동일 ID 한 record로 합침; duplicate 거부 유지 | PASS |
| 16 | G7 | `developer/tests/test_project_profile_registry.py::test_contract_registry_owns_single_current_identity` | B stale test / fixture | O9 | single-current registry는 pp.1.02인데 pp.1.01 기대<br>→ current ID와 canonical schema 기대값 정합 | PASS |
| 17 | G7 | `developer/tests/test_project_profile_registry.py::test_historical_compatibility_identity_is_registered_without_becoming_current` | B stale test / fixture | O9 | 등록된 pp.1.01 SUPERSEDED / SEMANTIC_MIGRATION을 없음으로 기대<br>→ 정확한 lifecycle과 SEMANTIC_MIGRATION 선언 검증; runtime migration 없음 | PASS |
| 18 | G7 | `developer/tests/test_project_profile_registry.py::test_public_profile_catalog_exactly_describes_existing_distribution_assets` | B stale test / fixture | O9 | 기존 catalog의 pp.1.02 / DISTRIBUTED_EXAMPLE metadata 누락<br>→ exact 3개 distribution entry 정합; immutable asset 검증 유지 | PASS |
| 19 | G7 | `developer/tests/test_project_profile_registry.py::test_current_contract_has_immutable_baseline` | B stale test / fixture | O9 | current pp.1.02 baseline을 pp.1.01로 기대<br>→ current immutable baseline byte 검증 유지 | PASS |
| 20 | G8 | `developer/tests/test_self_profile_bridge_removal.py::test_repository_self_profile_is_explicit_and_root_bridge_is_absent` | C OWNER_DECISION_REQUIRED | O10 | root bridge retired라는 운영 계약과 tracked ptsip.yaml 존재가 충돌<br>→ 실제 폐기 여부 owner 승인 대기; 삭제나 기대값 완화 없음 | FAIL / 결정 대기 |
| 21 | G8 | `developer/tests/test_self_profile_bridge_removal.py::test_repository_test_mode_control_plane_defaults_to_explicit_self_profile` | B stale test / fixture | O11 | manual Test Mode CLI의 필수 --mode를 누락<br>→ 명시적 --mode repository-architecture 전달; canonical self-profile default 검증 유지 | PASS |
| 22 | G8 | `developer/tests/test_self_profile_bridge_removal.py::test_current_repository_profile_has_no_root_bridge_dependency` | C OWNER_DECISION_REQUIRED | O10 | canonical self-profile의 include에 ptsip.yaml 의존이 남음<br>→ root bridge 참조 제거와 local catalog/default 정합 결정 대기 | FAIL / 결정 대기 |

## C: 필요한 두 가지 owner 결정

1. **Self-profile partition (test #1, #2)**: 5개 unmatched selector의 정리와 113개 경로의 정확한 책임 선언이 필요하다.
   unmatched는 component의 `product-documentation:README.ko.md`, `ptsip-canonical-contracts:spec/**`, `repository-maintenance:docs/planning/**` 및 associated artifact의 `ptsip-governance-support:adoption/**`, `ptsip-governance-support:agents/**`이다.
   미배정은 `docs/translated` 2개, `src/agent_contracts` 106개, 테스트 5개이다.
   정확한 경로 목록은 machine record의 `failure_owner_reconciliation.owner_decisions.self_profile_partition.unassigned_paths`에 고정했다.
   기존 4개 경로에 대한 제한 승인만으로 나머지 경로의 classification / roles / associated-artifact owner 또는 `analysis_inputs`를 추론하지 않았다.
2. **Root bridge retirement (test #20, #22)**: `AGENTS.md`는 root bridge 폐기를 명시하지만 `ptsip.yaml`은 tracked 상태이고 canonical self-profile에도 include가 남아 있다.
   실제 폐기와 참조 제거를 적용할지, 현재 선언을 유지할지 Project Owner 결정이 필요하다. local catalog의 `main.ptsip.yaml`과의 정합도 함께 다뤄야 한다.
   이번 수정은 bridge 삭제, self-profile 선언 변경 또는 absent/no-dependency 기대값 완화를 하지 않았다.

## 회귀 방어와 검증

- Authorization: missing / duplicate / unknown / empty validated corpus는 모두 HOLD_NOT_AUTHORIZED.
- Binding: PLAN / MODIFY / VERIFY 모두 두 MIGR-0005 section을 보존하고 동일 ID는 한 번만 반환한다. 주입한 duplicate는 여전히 blocking error.
- Work packet: 격리 unit fixture와 별개로 미등록 live task_context는 여전히 fail-closed.

원본 22개 재현 report: `architecture-22-before.xml` (22 FAIL), SHA256 `d2a4806bcdce91421c2c5ce0531bb9462f972330df580e94647f2d9689d7de58`.
수정 후 대상 report: `architecture-22-after.xml` (27 PASS / 4 FAIL), SHA256 `a7fc9944e0c542328fb5178fb75a9572523e9219f7f88ffbcd6b80e83cacf496`.
Report root: `C:/Users/rhkrt/AppData/Local/Temp/ptsip-wu08-owner-d5e7b11d4e0b4dd79ae56682dde4fe0b`.
이 hash는 report byte integrity이며 canonical contract digest가 아니다.

자동 선택된 필수 Test Mode도 모두 재실행했다. 대상 debug 재실행은 필수 검증을 대체하지 않는다.

| 자동 선택 mode | 전체 | PASS | FAIL | 최종 판정 |
| --- | ---: | ---: | ---: | --- |
| repository-architecture | 343 | 339 | 4 | 위 C 4개만 실패; 원래 A/B 18개 및 추가 guard 9개 통과 |
| ptsip-contract | 33 | 32 | 1 | 기존 별도 실패 유지: `tests/ptsip/contracts/test_spec_assets.py::test_example_profile_validates_against_canonical_schema` |

필수 검증 합계는 376개 / 371 PASS / 5 FAIL / 0 ERROR / 0 SKIP이다.
contract 실패는 기존 `pp.1.02` 예제를 `pp.1.01` schema로 검증하는 경우다. 해당 테스트·schema·예제는 이번 범위에서 수정하지 않았다.
각 report와 hash, original 22개 / 8개 group / 9개 guard의 exact test-ID 대조 결과는 machine record의 `failure_owner_reconciliation.required_automatic_verification` 및 `result_review`에 기록했다.
정책·계획·Test Mode registry·Policy Resolver binding·Policy-Plan consistency·Context Plane·diff 검증은 PASS다.
필수 검증 후 변경은 결과 machine record와 이 review projection뿐이다. source/test/binding은 검증된 working-tree byte hash로 고정했다.
기존 7개 mode의 893 PASS / 61 FAIL 관찰은 변경하지 않는다. 현재 요청은 그중 architecture 22개에 한정하며 core / vpms 등 다른 실패를 해결했다고 주장하지 않는다.
WU-08은 BLOCKED이며 canonical digest / candidate / runtime activation 또는 release readiness를 승인하지 않았다.
