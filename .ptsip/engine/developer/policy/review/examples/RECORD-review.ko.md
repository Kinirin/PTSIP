# RECORD 정책 2건의 자연어 검토본

검토일: 2026-11-01, Asia/Seoul. 원문 확인 기준 커밋: `32961aef9e5c2f655fd377ae8208012931995735`. 검토 시작 시 `main` 작업 트리는 깨끗했다. 원문·선택 계약의 파일별 SHA-256과 추출 내용은 같은 폴더의 `RECORD.evidence.json`에 있다.

이 문서는 기존 문서의 해설과 개정 준비를 위한 비규범적 검토본이다. 아래에서 “해야 한다”, “금지한다”라고 옮긴 문장은 원문에 적힌 의무·금지의 번역이며, 이 검토본이 새 의무를 선언하는 것은 아니다. 원문 해설, 역사적 관측, 차이, 개정 제안을 구분한다. 사용자 검토 후 새 자연어 정책이 제시되기 전에는 정책 본문·상태·계약·규칙 등록을 변경하지 않는다.

## 먼저 확인할 현재 상태

| 대상 | 문서에 기록된 상태 | 현재 기계 진입 결과 | 검토상 취급 |
| --- | --- | --- | --- |
| `developer/policy/RECORD/MPD-RECORD-0001.yaml` | `RETIRED / Waiting`, 버전 `2.2` | `POLICY_IDENTITY_SELECTION_UNRESOLVED` | 퇴역한 구 구조의 역사적 내용. 현재 정책으로 복구하거나 Family를 추정하지 않음 |
| `developer/policy/RECORD/MPD-RECORD-0002.yaml` | 구 중첩 구조로 `APPROVED / provisional`, 상태 버전 `1` 선언 | RECORD `v2` 계약 선택 성공. 기존 `policy-check`는 공개 구조 `v7`에서 거부 | 후보의 선언 의도와 실제 현재 계약 적합성을 구별 |

실행 명령은 기존 `contract-entry`의 정확한 대상 경로를 사용했다. `0001` dry-run은 프로그램 종료 코드 2, `0002` dry-run은 0이었다. 저장소 `.venv/Scripts/python.exe`를 `PTSIP_PYTHON`으로 선택한 `0002` 본 검증은 `SCHEMA_REJECTED: urn:ptsip:policy:public-structure:v7`로 프로그램 종료 코드 2였다. `go run`의 셸 종료 코드는 이 실패에서 1이다. 기존 검증기는 첫 실패의 원인 경로를 세분화하지 않으므로, 그 메시지만으로 개별 필드의 실패 원인을 확정하지 않았다.

현재 `RECORD_rules.json`의 식별자는 `urn:ptsip:contract:policy:rules:RECORD:v2`이다. 따라서 이 파일을 새로 만들 필요는 없다. `0002` 안의 후보 `v1` 문자열을 현재 계약 식별자로 간주하지 않는다.

## MPD-RECORD-0001 전체 내용 해설

### 문서의 정체성과 책임

구 문서 형식은 `ptsip-developer-root-family-policy/v1`이고, 개발자 정책이며 책임 Family는 RECORD이다. 정책 ID는 `MPD-RECORD-0001`, 제목은 “RECORD 권위 — ACTIVE 출처에서 온 책임”이라는 취지다. 제목의 ACTIVE 표현과 별개로 문서 자신의 기록 상태는 `RETIRED / Waiting`이다.

책임 주제의 식별자는 `DEV_DECISION_AND_EXECUTION_PROVENANCE`이다. 이 문서는 개발 결정, 승인, 작업 실행, 검증 사건에 대해 다음을 소유한다고 선언했다.

- 각 사건의 출처를 남긴다.
- 그 출처를 정확한 revision과 연결한다.
- 이후 감사와 재확인이 가능하도록 보존한다.
- 역사적 기록을 유지하고 서로 연결한다.

이 책임은 다음 두 핵심으로 구분되어 있다.

1. **사건 출처와 원본의 연결** (`DEV_EVENT_PROVENANCE_AND_SOURCE_BINDING`): 개발 기록에 대상, 수행 주체, 승인 근거, 원본 SHA, 시점, 결과의 출처를 함께 연결한다. 서로 다른 revision의 증거를 섞지 않는다. ASSURE가 판정을 내리거나 GOV가 결정을 했다는 사실만으로, 그 사건의 출처를 다시 확인할 수 있는 기록 보존이 대체되지는 않는다는 뜻이다.
2. **역사적 보존과 추적** (`DEV_HISTORY_RETENTION_AND_TRACEABILITY`): 원본 검증 로그, receipt, 완료 산출물의 보존·조회·이력 연결을 정한다. 과거 기록을 현재 권한이나 현재 PASS로 올려 해석하지 않는다. INFO의 현재 canonical 레코드나 OPS의 회복 상태가 역사적 사건의 원본 출처와 보존을 대신하지 못한다는 뜻이다.

`source_scope`에는 `MPD-0010`, `MPD-0011`, `MPD-0012`, `MPD-RELS-0001`, `MPD-WORK-0002`가 적혀 있다. 이 목록만으로 각 원문 전체가 RECORD 책임에 편입된다는 뜻으로 해석하지 않았다. 실제 수록된 단위와 맞닿는 부분만 아래에서 대조했다.

### 수록 단위 1 — 정확한 스냅샷의 식별

`unit_mpd_0010_b72085fb0868`의 값은 `GIT_REVISION`이다. 자연어로는 “정확한 스냅샷의 정체성은 Git revision으로 식별한다”는 내용이다.

`MPD-0010` 원문의 `/rules/management_policy_identity_lifecycle_preflight/policy_version_contract/exact_snapshot_identity`와 값이 일치한다. 이 한 항목을 해설하기 위해 원문의 확장 설계, 정책 ID 할당, 임시 결정 허용 규칙까지 끌어오지는 않았다.

### 수록 단위 2 — Project Profile 자동화의 과거 구현 기록

`unit_mpd_0011_b2d8b41a01e2`는 새로운 동작 명령보다 “당시 무엇이 구현되어 있다고 기록했는가”에 해당한다. `MPD-0011`의 `/rules/implementation_state`와 일치한다.

| 원문 항목 | 원문 값 | 자연어 해설 |
| --- | --- | --- |
| `policy_record` | `MATERIALIZED` | 정책 기록이 작성되어 있다고 기록했다. |
| `policy_index_registration` | `MATERIALIZED` | 정책 인덱스에 등록되어 있다고 기록했다. |
| `policy_resolver_routing` | `MATERIALIZED` | 정책 resolver의 라우팅이 구현되어 있다고 기록했다. |
| `authority_plane_registration` | `MATERIALIZED` | 권위 plane 등록이 구현되어 있다고 기록했다. |
| `operationalization_level` | `L4_LOCAL_AUTO_REMEDIATION_L3_REMOTE_AND_RELEASE_VERIFY` | 로컬 자동 복구는 L4, 원격·릴리스 검증은 L3 수준이라고 명명했다. 이 두 수준의 상세 정의는 이 수록 단위에 없다. |
| `public_profile_catalog` | `MATERIALIZED` | 공개 Profile 카탈로그가 구현되어 있다고 기록했다. |
| `canonical_pp_contract_registry` | `MATERIALIZED` | canonical PP 계약 등록부가 구현되어 있다고 기록했다. |
| `t2_delta_resolver` | `MATERIALIZED` | T2 변경분 resolver가 구현되어 있다고 기록했다. |
| `transition_reconciler` | `MATERIALIZED_AND_HOOK_INVOKED` | 전이 정합화 기능이 구현되어 hook에서 호출된다고 기록했다. |
| `h3_hook_activation` | `MATERIALIZED_SHARED_INSTALLER` | H3 hook 활성화가 공통 설치기를 통해 구현되어 있다고 기록했다. |
| `remote_commit_verifier` | `MATERIALIZED_PUSH_VERIFY_ONLY` | 원격 commit 검증기는 push 검증만 수행하는 형태로 구현되어 있다고 기록했다. |
| `release_transition_verifier` | `MATERIALIZED_EXACT_SHA_VERIFY_ONLY` | 릴리스 전이 검증기는 정확한 SHA에 대한 검증만 수행하는 형태로 구현되어 있다고 기록했다. |
| `runtime_current_pp_registry_resolution` | `MATERIALIZED` | 실행 시점의 현재 PP 등록부 해석이 구현되어 있다고 기록했다. |
| `packaging_registry_projection` | `MATERIALIZED` | 패키징용 등록부 projection이 구현되어 있다고 기록했다. |
| `distribution_contract_verifier` | `MATERIALIZED` | 배포 계약 검증기가 구현되어 있다고 기록했다. |
| `claim` | `LOCAL_REMOTE_RELEASE_PP_AUTOMATION_ACTIVE` | 로컬·원격·릴리스 PP 자동화가 활성화되어 있다는 당시의 주장을 담았다. |

이 표는 현재 환경에서 기능을 재실행하여 입증한 결과가 아니다. 과거의 `MATERIALIZED`·`ACTIVE` 문자열을 현재 구현 완료·실행 권한으로 바꾸지 않는다.

### 수록 단위 3 — 에이전트 계약 plane의 과거 스냅샷

`unit_mpd_0012_7bc423f87b11`는 `MPD-0012`의 `/rules/repository_materialization_snapshot`과 일치한다.

- 당시 관측한 브랜치는 `dev/0.3.8`이었다.
- 이 기록 직전 HEAD는 `df018272ef31ef84c3cc158d00235af946862c09`로 기록되어 있다.
- 당시 에이전트 계약 루트는 `src/agent_contracts`, 계약 집합 상태는 `CURRENT`, 권위 범위는 `CODING_AGENT_BEHAVIOR`로 기록되어 있다.
- Markdown에 규범적으로 의존하는 것은 `FORBIDDEN`, 외부 기계 계약은 `EXPLICIT_BINDING_ONLY`, 즉 명시적 바인딩을 통해서만 사용한다고 기록했다.
- 계약 종류로 `SPEC`, `OPERATION`, `ACTION`, `CONDITION`, `MUTATION_GATE`, `IO_SCHEMA`, `VOCABULARY`, `BINDING`을 열거했다. 각각 사양, 작업, 행동, 조건, 변경 gate, 입출력 schema, 어휘, 바인딩을 가리키는 원문 명칭이다.
- Markdown 정리 자동화는 `MATERIALIZED_SIMULATION_AND_GUARDED_APPLY`, 즉 시뮬레이션과 보호 조건을 둔 적용 기능이 구현된 것으로 기록했다.
- 완료된 구현 항목은 `NORMATIVE_OPERATIONAL_MARKDOWN_PHYSICAL_REMOVAL`, 즉 규범·운영 Markdown의 물리적 제거다.
- 구 Markdown의 당시 물리적 존재 상태는 `ABSENT`로 기록했다.
- 이 ledger에서 수행한 구현 작업은 `AGENT_CONTRACT_MACHINE_PLANE_M0_M5_VERIFIED`, 즉 에이전트 계약 기계 plane의 M0~M5가 검증되었다고 기록했다.

여기서 `CURRENT`, `ABSENT`, `VERIFIED`는 원문 스냅샷의 관측·주장이다. 현재 브랜치·계약 루트·구현 상태로 다시 선언하지 않는다. 원문에 별도로 있는 6단계 제어 plane 설계 전체와 잠정·보류 결정은 이 수록 단위의 번역 범위에 포함하지 않았다.

### 수록 단위 4 — 릴리스 노트의 기록 구성

`unit_mpd_rels_0001_8b9a13c6427c`는 다음을 담고 있다.

- 역사적 승인 참조는 `developer/policy/approvals/MPA-20261013-UNIFIED-RELEASE-SPECIFICATION.yaml`이다.
- 당시 릴리스 노트 구성 계약 참조는 `.github/checks/release/composition-contract.json`, 표현 계약 참조는 `.github/checks/release/release-presentation.json`이다.
- Package 노트 파일 패턴은 `docs/releasenote/PTSIP_Package/{package_version}.md`다.
- 엔진별 노트 파일 패턴은 `docs/releasenote/engine/{engine}/{engine}-engine-{version}.md`다.
- 여러 엔진의 노트를 하나의 병합 노트 파일로 만드는 것은 `FORBIDDEN`이다.
- 하나의 source SHA에 대한 하나의 Release 본문 안에 구성요소별 노트 section을 모으는 방식은 허용된 구성으로 기록되어 있다. 노트 원본 파일을 하나로 합치는 것과 Release 본문에서 묶어 보여 주는 것을 구분한다.
- 기록된 노트 영역은 `tool`, `project-profile`, `PTSIP_Package`, `engine`이다.
- 과거의 평면 릴리스 노트들을 일괄 이동하지 않았다고 기록했다 (`historical_flat_release_notes_bulk_moved: false`). 이 값만으로 앞으로의 모든 이동을 금지한다고 확대 해석하지 않는다.

**원문 간 차이:** 현재 보관된 `developer/policy/legacy/MPD-RELS-0001.yaml`의 `/rules/authority_semantics`는 영역을 `tool`, `project-profile`, `specification`으로 열거한다. RECORD 수록 단위와 동일하지 않다. 두 파일은 과거 평면 노트의 일괄 이동 여부 `false`는 공유한다. 별도로 읽은 2026-10-13 승인 기록의 `approval.source_reference`에는 `specification`을 `PTSIP_Package`로 바꾼 표현과 엔진별 파일 패턴, source SHA별 단일 Release 지시가 기록되어 있다. 이는 역사적 차이를 이해하기 위한 근거이며 현재 승인 권위로 호출한 것이 아니다.

이 검토본은 옛 계약 경로를 현재 계약 경로로 고쳐 쓰지 않았고, 승인 기록에 함께 있는 테스트·플랫폼·설치기·버전 결정까지 RECORD 정책 내용에 추가하지 않았다. 새 정책에서 유지할 릴리스 기록 책임과 세부 형식 계약의 관계는 사용자의 새 선언에 따라 검토한다.

### 수록 단위 5 — 실행 결과의 인계

`unit_mpd_work_0002_a794e002b8bd`는 다음 두 요구를 담는다.

1. 실행 결과를 구조화된 형태로 제공해야 한다 (`structured_result_required: true`).
2. 실행 후의 결과 상태를 제공해야 한다 (`resulting_state_required: true`).

이는 `MPD-WORK-0002`의 `/rules/work_execution_lifecycle/execution_output`과 일치한다. 원문에 별도로 있는 후속 실행 허용, 기존 승인 유지, 이탈 시 재해석·상향 보고 규칙은 이 두 항목에 편입하지 않았다.

### 관계와 과거 전환 기록

`relations.supersedes`, `amends`, `extends`, `depends_on`은 모두 빈 목록이다. 이 문서가 명시한 해당 관계는 없다는 뜻이며, 실제로 모든 의존성이 없다는 분석 결과로 확대하지 않는다.

과거 전환 기록은 목표를 `APPROVED / Applying`, 전환 상태를 `COMPLETE`로 적었다. `TR-001`은 원본 책임 상태 보존 요구 (`ROOT_FAMILY_SOURCE_STATE_PRESERVATION`)이고 `SATISFIED`로 기록되어 있다. 참조는 `developer/policy/approvals/MPA-20261006-ROOT-RECORD-0001-ACTIVE.yaml`이다. 후속 행동은 `VERIFY_SOURCE_PROJECTION`, 실행 방식은 `SERIAL_REQUIRED`, 완료 확인은 `EXACT_SOURCE_RECONSTRUCTION_AND_STATE_MATCH`, 즉 순차 실행으로 출처 projection을 검증하고 원본의 정확한 재구성과 상태 일치를 확인한다는 내용이다.

**상태 해석 주의:** 위 전환의 `COMPLETE`·`SATISFIED`와 문서 자신의 `RETIRED / Waiting`을 모두 그대로 보존했다. 또한 참조된 2026-10-06 승인 기록은 목표 상태를 `ACTIVE`로 적고 있어, 현재 수록된 전환 목표 문자열과 다르다. 과거 전환 기록이나 승인 파일을 이용해 현재 상태를 자동 교정하지 않는다.

## MPD-RECORD-0002 전체 내용 해설

이 문서는 개발자 정책이며 Family는 RECORD, ID는 `MPD-RECORD-0002`다. 제목은 “RECORD 신규 독립 규칙 계약 후보”다.

상태 부분은 구 중첩 구조에서 상태 버전 `1`, `APPROVED`, 세부 상태 `provisional`을 선언한다. 상태 계약 문자열은 `urn:ptsip:contract:policy:record:rule-structure:provisional`이다. 이것은 문서에 기록된 값의 해설이며, 현재 lifecycle 계약 검증을 통과했다거나 승인 범위가 확정되었다는 판정이 아니다.

`Authority`에는 `authority_id: MPD-RECORD-0002`와 `urn:ptsip:authority:action:policy:policy-modification` 행동 참조가 기록되어 있다. 정책 수정 행동을 가리키는 선언을 담았다는 뜻이다. 이 선언 자체를 실제 수정·실행 권한 부여로 해석하지 않는다.

`Policy_Rules`의 여섯 항목은 다음 의미다.

| 항목 | 값 | 자연어 해설 |
| --- | --- | --- |
| `independent_rule_refs` | 빈 목록 | 선택해 적은 독립 규칙 참조가 아직 없다. |
| `candidate_contract_id` | `urn:ptsip:contract:policy:rules:RECORD:v1` | 이 후보가 가리킨 계약 식별자는 RECORD v1이다. 현재 등록 파일의 v2와 차이가 있다. |
| `candidate_contract_path` | `.ptsip/contracts/policy/Rules/RECORD_rules.json` | 후보가 가리킨 계약 파일 경로다. |
| `candidate_scope` | `기록 추적` | 후보의 범위를 기록 추적으로 적었다. |
| `semantic_rework_required` | `true` | 정책 의미의 재작성이 필요하다고 기록했다. |
| `normative_content_from_retired_policies` | `FORBIDDEN` | 퇴역 정책의 내용을 새 규범으로 가져오는 것을 금지한다고 적었다. 과거 내용을 검토용으로 읽어 해설하는 것과 새 정책의 규범으로 승계하는 것을 구별한다. |

이 파일에는 기록 보존 기간, 구체적 사건 필드, 보존 위치, 책임 분배, 검증 절차 등의 상세 규칙은 선언되어 있지 않다. 이를 번역자가 새로 보충하지 않았다.

## 관련 원문 구간 선정표

이 표는 이번 번역에 대해 사람이 판단한 근거다. Family의 공식 소유권 매핑이나 범용 자동 분류 규칙이 아니다. helper는 이 선정 결과만 반복 추출한다.

| 출처 | 선택한 JSON Pointer | 선정 이유 | 가져오지 않은 범위 |
| --- | --- | --- | --- |
| `MPD-0010` | `/rules/management_policy_identity_lifecycle_preflight/policy_version_contract/exact_snapshot_identity` | 수록 단위의 `GIT_REVISION`과 일치 | 설계 우선순위, ID·승인·버전 전이·임시 결정 규칙 전체 |
| `MPD-0011` | `/rules/implementation_state` | 수록된 과거 구현 상태와 일치 | PP 전이 권위, hook 활성화 절차, 다른 정책 관계 |
| `MPD-0012` | `/rules/repository_materialization_snapshot` | 브랜치·HEAD·계약 plane 관측과 일치 | 설계 단계 전체, 잠정·보류 결정, 별도 검증·활성화 규칙 |
| `MPD-RELS-0001` | `/rules/authority_semantics` | 노트 영역과 일괄 이동 여부 대조. 영역 차이는 별도 표시 | 다른 문서에서의 릴리스 승인 전체를 자동 병합하지 않음 |
| `MPD-WORK-0002` | `/rules/work_execution_lifecycle/execution_output` | 구조화된 결과·결과 상태 요구와 일치 | 후속 실행, 이탈·상향 보고 규칙 |

다섯 출처의 실제 경로는 모두 `developer/policy/legacy/<ID>.yaml`이다. 이 경로는 역사적 열람에만 사용했다. 추가로 읽은 두 승인 파일은 위의 릴리스 변경과 전환 상태 차이 해설에만 사용했다. helper의 다섯 `sources`에는 이 승인 파일 전체를 넣지 않았다.

## 새 선언을 기계 정책으로 작성하기 전 검토할 사항

### 현재 계약에서의 표현과 확장

현재 RECORD `v2`는 `Policy_Rules`에 `Identity`, `Definition`, `Boundary`, `Evaluation`, `title`을 요구한다. `0002`에는 이 다섯 필드가 모두 없고, 대신 위 여섯 후보용 필드가 있다. helper는 이를 **필수 선언 필드 누락 5건, 명시적으로 선언되지 않은 필드 6건**으로 표시했다. 이 6건이 모두 JSON Schema의 추가 필드 금지 위반이라는 주장은 하지 않는다. RECORD의 `structure.additionalProperties`는 `true`이고 일부 중첩 객체도 열려 있기 때문이다.

새 자연어 내용이 주어지면 먼저 기존 책임 정의·등록 참조, 경계, 평가 항목으로 각각 매핑한다. 표현되지 않는 내용은 보고하고, 일회성 필드를 몰래 추가하거나 모든 의미를 자유 문자열에 몰아넣지 않는다. 이 검토에서는 새 필드, 공통 계약 변경, 다른 Family 확장, 새 R-ID·조건·권한을 생성하지 않았다.

### 책임 분리 후보 — 아직 결정되지 않음

구 문서에는 서로 다른 종류의 내용이 함께 있다.

| 내용 | 작성 전 검토 방향 |
| --- | --- |
| 사건의 대상·주체·승인 근거·SHA·시점·결과 출처 연결 | 사건 출처 책임의 정책 후보 |
| 로그·receipt·완료 산출물의 보존·조회·이력 연결, 과거 기록의 현재 권위 승격 방지 | 역사 보존·추적 책임의 정책 후보. 앞 후보와 적용 조건·검증·변경 단위가 독립적인지 검토하여 분리 권장 여부 결정 |
| PP·에이전트 plane의 과거 구현 스냅샷 | 역사적 사실·증거 기록으로 보존할 내용인지 먼저 검토. 과거 상태값을 영구 정책 의무로 복제하지 않음 |
| 릴리스 노트 영역·파일 패턴·Release 본문 구성 | 기록 보존 책임과 릴리스 표현 형식 책임의 경계를 별도로 확인. RECORD에 전부 넣거나 다른 Family의 소유권을 추정하지 않음 |
| 실행 결과·결과 상태의 인계 | 사건 기록 책임에 필요한 부분과 실행 생명주기 책임에 필요한 부분을 구별하여 중복 여부 검토 |

따라서 기존 파일 두 개를 반드시 일대일로 유지하는 것을 전제로 삼지 않는다. 새 선언의 작업·책임 영역이 명확히 독립적이면 문서 분리를 먼저 권장한다. 경계가 모호하거나 추가 판단·전환 비용이 크면 기계 정책을 쓰기 전에 사용자에게 분리 여부를 질문한다. 현재는 새 자연어 정책을 아직 받지 않았으므로 파일별 최종 책임과 정책 ID를 배정하지 않았다.

## 검토 완료와 미완료의 경계

- 완료: 두 대상 문서 전체 해설, 다섯 출처의 관련 구간 선정·추출, 원문 간 차이 표시, 기존 계약 선택·필드 목록 점검, 책임 분리 후보 제시.
- 미완료: 사용자의 새 자연어 정책 선언과 내용 승인, 최종 책임 분리 결정, 그 내용의 등록 계약 매핑, 필요할 경우의 계약 확장 결정, 새 기계 정책 작성·검증.
- 현재 미해결: `0001`의 현재 계약 진입 불가, `0002`의 공개 구조 검증 실패와 후보 v1/현 계약 v2 차이. 이번 읽기 전용 검토는 이 상태들을 해결하거나 승인한 것이 아니다.
