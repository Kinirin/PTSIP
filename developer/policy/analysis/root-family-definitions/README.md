# 14 Root Family의 authority_subject와 exclusive_kernel

사용자의 직접 위임에 따라 Developer 14개와 Support 14개를 각각 정의했다. 각 identity는 `(policy_class, responsibility_family)`이다. 같은 Family token은 좌표이며 클래스 간 의미 상속이나 동등성을 뜻하지 않는다.

이 파일과 두 JSON은 developer-only semantic design 산출물이다. `developer.json`과 `support.json`이 정의 데이터를 담고 이 문서는 그 내용을 검토하기 위한 표현이다. 기존 정책 authority를 대체하거나 runtime에서 실행되는 정책을 등록하지 않는다. Support 정의도 여기서는 개발용 설계 산출물이며 실제 SFP 정책 materialization 경로는 resolver가 반환한 `src/policy/<Family>/...`이다.

`CONSERVATIVE_MINIMUM_KERNEL`에 따라 Family별 최소 비대체 권한 두 개를 정의했다. `consumes`는 인접 Family에서 받아야 하는 의미이며 `excludes`는 그 인접 Family가 소유하는 책임이다. 두 항목은 아직 등록된 정책 dependency나 실행 permission이 아니다. 전체 boundary closure는 열려 있고 미정 책임은 실행을 승인하지 않는다.

기준 branch/HEAD: `dev/0.3.8a4` / `18d7ee83ab5d18b2113fcbf5e2855af386b75dde`. 정의 생성은 로컬 entry repair 수정본에서 수행했다. 원본 전체 pytest 결과는 1302 passed / 100 failed / 19 errors / 2 skipped이고 전체 PASS gate는 충족하지 않았다. 정의 생성과 정책 활성화 및 검증 gate는 독립된 상태로 기록한다.

각 `routing_snapshot`은 생성 시 exact class/Family resolver 결과다. ID는 prospective 값이며 등록된 할당이 아니다. Materialization 시 다시 resolve하고 lifecycle 승인, 기존 subject 충돌 확인, 책임 단위 decomposition 및 해당 class의 등록 절차를 수행해야 한다. 생성 위임에서 DRAFT 또는 ACTIVE 상태 승인을 추론하지 않았다.

## 클래스별 authority subject

| Family | Developer subject | Support subject |
|---|---|---|
| NORM | `DEV_RULE_NORMATIVITY` | `SUP_CONSUMER_REQUIREMENT_MEANING` |
| GOV | `DEV_DECISION_LEGITIMACY` | `SUP_PROJECT_AUTHORITY_LEGITIMACY` |
| INTENT | `DEV_WORK_OUTCOME_INTENT` | `SUP_DECLARED_OPERATION_AND_PROJECT_INTENT` |
| ARCH | `DEV_TOOL_RESPONSIBILITY_STRUCTURE` | `SUP_PROJECT_LIFECYCLE_RESPONSIBILITY_DECLARATION` |
| INFO | `DEV_CONTROL_INFORMATION_MEANING` | `SUP_PROFILE_CONTEXT_INFORMATION_SEMANTICS` |
| CNTR | `DEV_AUTOMATION_INTERFACE_OBLIGATIONS` | `SUP_SHIPPED_OPERATION_CAPABILITY_CONTRACT` |
| RISK | `DEV_CHANGE_RISK_DISPOSITION` | `SUP_AUTOMATION_UNCERTAINTY_AND_CONFLICT_RISK` |
| SUPPLY | `DEV_BUILD_VERIFICATION_INPUT_SUPPLY` | `SUP_PROJECT_DEPENDENCY_AND_ARTIFACT_SUPPLY` |
| REAL | `DEV_APPROVED_ARTIFACT_REALIZATION` | `SUP_DETERMINISTIC_CONSUMER_STATE_REALIZATION` |
| ASSURE | `DEV_VERIFICATION_EVIDENCE_JUDGMENT` | `SUP_DECLARATION_EVIDENCE_CONFORMANCE_JUDGMENT` |
| CTRL | `DEV_ACTION_TIME_EXECUTION_ELIGIBILITY` | `SUP_FRESH_CONSUMER_ACTION_ELIGIBILITY` |
| CHANGE | `DEV_VERSIONED_TRANSITION_SEMANTICS` | `SUP_EXPLICIT_PROFILE_AUTHORITY_MIGRATION` |
| OPS | `DEV_SERVICE_CONTINUITY_AND_RECOVERY` | `SUP_AVAILABLE_CAPABILITY_CONTINUITY_AND_RECOVERY` |
| RECORD | `DEV_DECISION_AND_EXECUTION_PROVENANCE` | `SUP_OBSERVATION_AND_OPERATION_PROVENANCE` |

## Developer: PTSIP_DEVELOPER_POLICY

정의 데이터: [developer.json](developer.json) · bootstrap: `developer/policy/MPD-0018.yaml`

### NORM — DEV_RULE_NORMATIVITY

PTSIP 개발에 적용되는 규칙의 규범적 효력, 해석 및 명시적으로 선택된 immutable Specification과의 구속 관계를 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `DEV_RULE_FORCE_AND_PRECEDENCE` | 개발 규칙의 MUST/MUST_NOT 의미, 적용 범위 및 명시적 규범 우선순위를 정한다. bound Specification의 상위 요구를 완화할 수 없다. | GOV의 적법한 승인만으로 규칙의 의무 강도나 상위 Specification 구속력을 바꿀 수 없다. |
| `DEV_NORMATIVE_INTERPRETATION` | 규칙 간 충돌과 해석 가능한 범위를 선택된 규범 revision 안에서 정하고, 새 규범 의미가 필요한 변경을 식별한다. | ASSURE는 정해진 규칙으로 판정하며 새로운 규범 의미를 검증 결과에서 만들어 낼 수 없다. |

**소비 경계**

- `GOV`: 정당한 규칙 제정 주체와 명시적인 승인
- `RECORD`: 선택된 규범 revision 및 변경 근거의 출처

**제외 경계**

- `GOV`: 누가 규칙을 제정하거나 승인할 수 있는지
- `INFO`: 규칙 레코드의 저장 구조와 projection 형태

**판별 사례:** 개발 규칙의 SHOULD를 MUST로 바꾸는 의미 변경은 NORM이 소유하고 승인 적법성은 GOV에서 소비한다.

**다음 materialization route:** `developer/policy/NORM/MPD-NORM-0001.yaml` · schema `developer/policy/schemas/root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### GOV — DEV_DECISION_LEGITIMACY

개발 정책, 개발 작업 및 개발 control plane에서 누가 어떤 subject에 대해 결정을 내리고 승인할 수 있는지와 그 결정의 정당성을 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `DEV_DECISION_OWNER_AND_MANDATE` | 개발 결정의 정확한 authority subject, 소유자, 위임 범위와 승인 주체를 명시하고 무권한 결정을 배제한다. | INTENT의 작업 목표나 ARCH의 기술 설계가 결정권자의 위임 범위를 대신 만들 수 없다. |
| `DEV_APPROVAL_AND_LIFECYCLE_LEGITIMACY` | 정책 등록, 정책 lifecycle 상태 및 개발 작업 승인의 출처와 대상 범위가 적법한지를 정한다. | CTRL의 실행 가능 판정이나 테스트 PASS가 명시되지 않은 ACTIVE/DRAFT 상태 승인을 만들어 내지 못한다. |

**소비 경계**

- `NORM`: 개발 승인과 권한에 적용되는 상위 규칙
- `RECORD`: 승인 출처와 결정 이력

**제외 경계**

- `CTRL`: 실행 직전 HEAD와 상태를 확인하여 허용 여부를 판정하는 gate
- `ARCH`: 승인된 설계의 책임 구조 자체

**판별 사례:** 새 MPD의 lifecycle 상태를 승인하는 권한은 GOV이며, 등록 직전 상태 일치 검사는 CTRL이다.

**다음 materialization route:** `developer/policy/GOV/MPD-GOV-0001.yaml` · schema `developer/policy/schemas/root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### INTENT — DEV_WORK_OUTCOME_INTENT

Reference Tool 및 개발 작업이 달성하려는 결과, 작업 범위, 비목표와 수락 목적을 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `DEV_OUTCOME_SCOPE_AND_NON_GOALS` | 개발 요청을 명시적인 달성 결과, 포함 범위와 비목표로 한정하고 범위 확장의 필요를 식별한다. | REAL의 구현 편의나 SUPPLY의 사용 가능한 도구가 승인된 작업 범위를 확대할 수 없다. |
| `DEV_ACCEPTANCE_PURPOSE` | 작업 결과를 수락하려는 목적과 성공 조건의 의미를 정하되 실제 충족 여부는 ASSURE에 맡긴다. | ASSURE가 테스트를 통과시켰다는 사실만으로 요청하지 않은 제품 목표가 수락 목적이 되지 않는다. |

**소비 경계**

- `GOV`: 작업 목표와 범위를 정할 권한 및 위임
- `NORM`: 목표가 따라야 하는 개발 규범

**제외 경계**

- `ARCH`: 결과를 실현하는 책임 분할과 연결 구조
- `CHANGE`: 현재 상태에서 목표 상태로 가는 전환 순서

**판별 사례:** 이번 작업의 완료 목표를 28개 Family 정의 생성으로 한정하는 것은 INTENT이며 정책 활성화 승인을 포함하지 않는다.

**다음 materialization route:** `developer/policy/INTENT/MPD-INTENT-0001.yaml` · schema `developer/policy/schemas/root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### ARCH — DEV_TOOL_RESPONSIBILITY_STRUCTURE

Reference Tool과 개발 체계의 lifecycle 책임 소유, 구성 요소 경계 및 명시적인 typed 관계를 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `DEV_LIFECYCLE_RESPONSIBILITY_BOUNDARIES` | Product, Development Tooling, Delivery, Operations 및 Neutral Contract 책임을 governing lifecycle obligation에 따라 분리한다. | SUPPLY의 생산자, 파일 형식 또는 실행 환경이 Artifact의 lifecycle 소유자를 결정하지 않는다. |
| `DEV_EXPLICIT_COMPONENT_RELATIONSHIPS` | 독립적으로 통제할 수 있는 책임 단위의 분할, 경계와 프로젝트가 선언한 typed directed 관계를 정한다. | INFO의 schema나 관측 import 목록만으로 프로젝트의 책임 분할과 관계 의도를 확정할 수 없다. |

**소비 경계**

- `INTENT`: 제품과 개발 체계의 목표 및 제한
- `NORM`: 책임 분류와 경계에 적용되는 규범

**제외 경계**

- `INFO`: 선언을 표현하는 필드와 정보 의미
- `REAL`: 설계대로 코드와 Artifact를 구현하는 행위

**판별 사례:** MPD 자동화가 consumer runtime에 포함되지 않도록 책임 경계를 정하는 것은 ARCH이고 실제 패키지 위반 판정은 ASSURE이다.

**다음 materialization route:** `developer/policy/ARCH/MPD-ARCH-0001.yaml` · schema `developer/policy/schemas/root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### INFO — DEV_CONTROL_INFORMATION_MEANING

개발 정책, planning, self-profile 및 Context Plane 정보의 의미 구조와 canonical semantic source 및 projection의 구분을 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `DEV_CANONICAL_INFORMATION_AND_IDENTITY` | 개발 레코드의 필드, 식별자, 상태 표현 및 canonical 정보 단위의 의미를 정의한다. 식별자 할당 승인의 적법성은 GOV가 소유한다. | GOV의 승인 기록은 정보 필드의 해석과 schema 의미를 대신 정의하지 못한다. |
| `DEV_SOURCE_PROJECTION_SEMANTICS` | semantic source와 결정론적 projection의 관계 및 projection이 독립 authority가 될 수 없는 조건을 정한다. | REAL의 생성 절차나 RECORD의 과거 로그가 현재 정보의 semantic source를 대체하지 못한다. |

**소비 경계**

- `ARCH`: 정보를 소유하는 개발 책임 경계
- `NORM`: canonical 정보 및 identity에 적용되는 규범

**제외 경계**

- `CNTR`: resolver 호출자에게 약속하는 입력과 출력 계약
- `RECORD`: 과거 사건의 provenance와 보존 책임

**판별 사례:** context.source.json이 의미 원본이고 context.json/jsonl이 projection임을 정의하는 것은 INFO이다.

**다음 materialization route:** `developer/policy/INFO/MPD-INFO-0001.yaml` · schema `developer/policy/schemas/root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### CNTR — DEV_AUTOMATION_INTERFACE_OBLIGATIONS

개발 resolver, IWP, command plane 및 검증 자동화가 호출자와 맺는 입력, 출력, 오류 및 호환 의무를 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `DEV_AUTOMATION_REQUEST_RESULT_CONTRACT` | 호출 가능한 command vocabulary, typed 입력과 출력, 오류 의미 및 precondition/postcondition 약속을 명시한다. | INFO의 내부 데이터 모델만으로 외부 호출에 허용된 command와 결과 약속을 만들 수 없다. |
| `DEV_INTERFACE_COMPATIBILITY_COMMITMENT` | 개발 자동화 인터페이스의 보장 범위와 호환성 약속을 정하고 미등록 command나 추측 fallback이 계약이 되는 것을 금지한다. | CHANGE가 전환 일정을 정하거나 REAL이 편의 기능을 구현해도 계약상 지원 의무가 자동으로 생기지 않는다. |

**소비 경계**

- `INFO`: 입출력 정보형 및 identity 의미
- `NORM`: 인터페이스에 적용되는 개발 규범

**제외 경계**

- `CTRL`: 특정 요청이 현재 상태에서 실제 실행 가능한지
- `CHANGE`: 호환성 변경의 영향과 전환 계획

**판별 사례:** branch_control의 등록된 command vocabulary를 정하는 것은 CNTR이고 특정 branch 생성의 허용 판정은 CTRL이다.

**다음 materialization route:** `developer/policy/CNTR/MPD-CNTR-0001.yaml` · schema `developer/policy/schemas/root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### RISK — DEV_CHANGE_RISK_DISPOSITION

Tool 구현, 개발 변경 및 개발 공급망의 위험 식별, 처리 의무와 위임 범위 안의 잔여 위험 수락을 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `DEV_RISK_IDENTIFICATION_AND_TREATMENT` | 개발 변경의 손실 가능성, 영향 범위 및 예방·완화·회복 조치의 필요를 명시한다. | ASSURE의 테스트 결과만으로 검사하지 않은 실패 가능성과 처리 의무가 사라지지 않는다. |
| `DEV_RESIDUAL_RISK_ACCEPTANCE_LIMIT` | 잔여 위험의 수락 조건과 수락 권한 요구를 정한다. MUST 위반 면제, 닫힌 License Authority 진입 또는 무권한 실행을 위험 수락으로 승인할 수 없다. | GOV의 위임이 있어도 RISK 밖의 상위 규범과 별도 authority boundary를 대체하지 못한다. |

**소비 경계**

- `GOV`: 잔여 위험을 수락할 주체와 위임 한계
- `ASSURE`: 검증된 위험 감소 효과 및 증거 공백

**제외 경계**

- `ASSURE`: 규칙 충족과 검증 증거의 적합성 판정
- `CTRL`: 현재 작업의 실행 gate

**판별 사례:** migration 검증 공백을 위험으로 기록하고 추가 검증을 요구하는 것은 RISK이며 공백 상태를 PASS로 바꾸지는 않는다.

**다음 materialization route:** `developer/policy/RISK/MPD-RISK-0001.yaml` · schema `developer/policy/schemas/root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### SUPPLY — DEV_BUILD_VERIFICATION_INPUT_SUPPLY

개발, build, verification 및 delivery에 투입하거나 인계하는 dependency, resource와 Artifact의 공급 적합성 및 출처를 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `DEV_INPUT_ADMISSION_AND_PROVENANCE` | 개발 입력과 환경 dependency의 정확한 출처, revision, 필요 조건 및 허용된 공급 경로를 명시한다. | REAL의 성공적인 build나 RISK의 잔여 위험 수락만으로 입력의 출처와 공급 자격을 확정할 수 없다. |
| `DEV_ARTIFACT_SUPPLY_HANDOFF` | 생산된 Artifact를 다음 개발·검증·delivery 단계에 공급하는 단위, provenance 및 인계 조건을 정한다. 생산자와 Artifact lifecycle 소유자는 구별한다. | ARCH가 lifecycle 소유자를 정해도 실제 공급 단위와 출처 확인 의무를 대신 수행하지 않는다. |

**소비 경계**

- `ARCH`: 입력과 Artifact의 책임 소유 경계
- `CNTR`: 공급 단계가 약속한 형태와 호환 요구

**제외 경계**

- `REAL`: 코드 구현 및 Artifact의 조립·생성 절차
- `CHANGE`: dependency/version 변경의 전환 순서

**판별 사례:** 검증에 사용할 Python과 dependency revision의 공급 조건은 SUPPLY이고 그 환경에서의 검증 판정은 ASSURE이다.

**다음 materialization route:** `developer/policy/SUPPLY/MPD-SUPPLY-0001.yaml` · schema `developer/policy/schemas/root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### REAL — DEV_APPROVED_ARTIFACT_REALIZATION

선택된 설계와 명시적으로 허용된 변경 대상을 코드, 정책 표현 및 생성 Artifact로 실제 구현하는 의미 보존 절차를 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `DEV_IMPLEMENTATION_AND_MATERIALIZATION` | 해결된 target과 입력을 사용해 코드와 결정론적 Artifact를 구현·materialize하고 선언된 의미 및 provenance를 보존한다. | INTENT의 목표나 ARCH의 설계 선언만으로 실제 구현 절차와 생성 결과가 존재하게 되지 않는다. |
| `DEV_MUTATION_INTEGRITY` | 허용된 파일 변경의 적용 단위, 원자성 및 실패 시 보존 조건을 정하고 예상하지 않은 overwrite와 부분 적용을 방지한다. | CTRL이 변경을 허용했다는 판정은 실제 쓰기의 원자성과 실패 처리를 대신 보장하지 않는다. |

**소비 경계**

- `ARCH`: 실현할 설계 및 책임 경계
- `CTRL`: 실행 가능한 정확한 대상과 현재 precondition

**제외 경계**

- `GOV`: 정책 lifecycle 상태와 구현 범위의 승인 적법성
- `ASSURE`: 구현 결과의 검증 및 conformance 판정

**판별 사례:** 승인된 semantic source에서 projection을 생성하고 실패 시 원본을 보존하는 행위는 REAL이다.

**다음 materialization route:** `developer/policy/REAL/MPD-REAL-0001.yaml` · schema `developer/policy/schemas/root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### ASSURE — DEV_VERIFICATION_EVIDENCE_JUDGMENT

개발 결과와 release 후보에 대한 증거의 적합성, 필요한 검증 coverage 및 정확한 source snapshot에 구속된 검증 판정을 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `DEV_VERIFICATION_COVERAGE_AND_ADMISSIBILITY` | 기계적으로 선택된 Test Mode와 규범상 필요한 검증 항목의 coverage 및 증거의 admissibility를 평가한다. | CTRL의 gate 목록이나 RECORD의 로그 존재만으로 필요한 검증 coverage와 증거 적합성이 충족되지 않는다. |
| `DEV_EXACT_SOURCE_VERIFICATION_OUTCOME` | 검증 판정을 실행한 source SHA, 환경 및 Artifact snapshot에 결합하고 functional PASS와 PTSIP conformance 및 VPMS purpose를 구별한다. | RECORD가 후속 documentation commit을 보존하거나 GOV가 release를 승인해도 미검증 SHA를 PASS로 만들 수 없다. |

**소비 경계**

- `NORM`: 판정에 적용되는 규범 요구
- `INTENT`: 작업 수락 목적과 성공 조건

**제외 경계**

- `CTRL`: 검증 결과를 소비하여 release나 실행을 허용하는 gate
- `RECORD`: 로그와 판정 출처의 보존

**판별 사례:** ci/pp-transition success를 전체 pytest PASS로 확대하지 않고 실행 corpus와 SHA를 구분하는 판정은 ASSURE이다.

**다음 materialization route:** `developer/policy/ASSURE/MPD-ASSURE-0001.yaml` · schema `developer/policy/schemas/root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### CTRL — DEV_ACTION_TIME_EXECUTION_ELIGIBILITY

개발 변경, branch command 및 release 동작이 실행 직전의 정확한 scope, 승인과 fresh state에서 허용되는지를 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `DEV_FRESH_PRECONDITION_AND_SCOPE_GATE` | branch, HEAD, task context, selector와 승인 범위를 실행 직전에 확인하고 stale하거나 unlisted인 요청을 fail-closed한다. | GOV의 적법한 과거 승인이나 INFO의 유효한 레코드만으로 현재 HEAD와 대상이 일치함을 보장할 수 없다. |
| `DEV_TRANSITION_GATE_COMPOSITION` | 정책 및 계약이 정한 gate들을 fresh evidence와 결합해 특정 실행의 허용·차단을 판정한다. 무권한 상태 변화나 새로운 정책 의미를 생성하지 않는다. | ASSURE의 PASS가 실행 권한, 정확한 대상 및 모든 transition precondition을 대신 충족하지 않는다. |

**소비 경계**

- `GOV`: 정당한 실행 승인과 exact scope
- `ASSURE`: 정확한 snapshot에 대한 필요한 검증 결과

**제외 경계**

- `REAL`: 허용된 mutation의 실제 적용과 실패 처리
- `NORM`: gate가 따르는 규범의 의미

**판별 사례:** commit 직전에 IWP의 HEAD와 target 일치를 확인하는 것은 CTRL이고 파일을 실제 쓰는 것은 REAL이다.

**다음 materialization route:** `developer/policy/CTRL/MPD-CTRL-0001.yaml` · schema `developer/policy/schemas/root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### CHANGE — DEV_VERSIONED_TRANSITION_SEMANTICS

개발 규범, 정책, profile 및 Tool의 명시적인 old→new 변경 관계, 영향과 안전한 전환 순서를 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `DEV_CHANGE_IMPACT_AND_COMPATIBILITY` | 변경 전후 의미, identity와 호환 약속의 차이 및 영향 받는 책임을 명시한다. 의미적 근거 없이 version 문자열이나 Family 이름을 치환하지 않는다. | INFO의 유효한 새 schema나 CNTR의 새 계약만으로 기존 상태와의 의미적 대응이 확정되지 않는다. |
| `DEV_MIGRATION_ORDER_AND_CONTINUITY` | 승인된 변경의 migration 경로, 선행 조건, staged legacy 책임 보존 및 retirement 조건을 정한다. | REAL의 변환 성공이나 OPS의 회복 조치만으로 전환 순서와 legacy authority 폐기 조건을 정당화할 수 없다. |

**소비 경계**

- `CNTR`: 변경 전후의 호환 의무
- `ARCH`: 영향 받는 책임 단위와 경계

**제외 경계**

- `GOV`: 변경과 lifecycle 상태 승인의 정당성
- `REAL`: 변환과 쓰기의 실행 mechanics

**판별 사례:** legacy 6 Family를 14 Root Family로 의미 단위 분해하는 전환 관계는 CHANGE이며 일대일 token 치환을 허용하지 않는다.

**다음 materialization route:** `developer/policy/CHANGE/MPD-CHANGE-0001.yaml` · schema `developer/policy/schemas/root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### OPS — DEV_SERVICE_CONTINUITY_AND_RECOVERY

인계 이후 개발 자동화와 개발 환경 및 control plane의 지속 운영, 상태 유지, 장애 회복과 반복 실행의 안전성을 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `DEV_OPERATIONAL_HEALTH_AND_RECOVERY` | 이미 등록되고 사용 가능한 개발 capability의 상태 감시, 장애 분류 및 authority를 보존하는 회복 절차를 정한다. | SUPPLY의 환경 공급이나 REAL의 일회성 구현만으로 지속 운영과 장애 회복 책임이 충족되지 않는다. |
| `DEV_RETRY_RECONCILIATION_CONTINUITY` | 승인된 operation identity와 revision을 보존하는 retry 및 reconciliation을 정하고 충돌을 은폐하지 않는다. RESERVED capability의 persistence나 실행 의미를 새로 만들지 않는다. | CHANGE의 migration 계획이나 CTRL의 단일 실행 허용은 반복 실행의 동일성 및 운영 연속성을 대신 정의하지 않는다. |

**소비 경계**

- `CNTR`: 운영 capability와 오류 및 재시도 계약
- `CTRL`: 각 회복 실행의 현재 eligibility

**제외 경계**

- `SUPPLY`: 초기 환경·dependency 공급 및 Artifact 인계
- `CHANGE`: 새 capability 활성화와 버전 전환 의미

**판별 사례:** 개발 자동화가 중단된 뒤 기존 operation identity로 안전하게 재시도하는 운영 절차는 OPS이다.

**다음 materialization route:** `developer/policy/OPS/MPD-OPS-0001.yaml` · schema `developer/policy/schemas/root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### RECORD — DEV_DECISION_AND_EXECUTION_PROVENANCE

개발 결정, 승인, 작업 실행 및 검증 사건의 출처와 exact revision 결합, 감사 가능성과 역사적 보존을 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `DEV_EVENT_PROVENANCE_AND_SOURCE_BINDING` | 개발 기록에 subject, 수행 주체, 승인 근거, source SHA, 시점과 결과 출처를 결합하고 다른 revision의 증거를 혼합하지 않는다. | ASSURE의 판정이나 GOV의 결정만으로 그 사건의 재확인 가능한 provenance가 보존되지 않는다. |
| `DEV_HISTORY_RETENTION_AND_TRACEABILITY` | 원본 검증 로그, receipt와 완료 산출물의 보존·조회 및 이력 연결을 정한다. 과거 기록을 현재 권한이나 현재 PASS로 승격하지 않는다. | INFO의 현재 canonical 레코드나 OPS의 회복 상태가 역사적 사건의 원본 출처와 보존을 대체하지 못한다. |

**소비 경계**

- `INFO`: 기록 정보형과 identity 의미
- `GOV`: 보존해야 할 정당한 승인과 결정 식별

**제외 경계**

- `GOV`: 현재 유효한 결정권과 승인 정당성 판정
- `ASSURE`: 기록된 증거로 규범 충족 여부를 판정하는 책임

**판별 사례:** 원본 SHA의 pytest 결과와 로컬 patch 검증 결과를 별도 receipt로 보존하는 것은 RECORD이다.

**다음 materialization route:** `developer/policy/RECORD/MPD-RECORD-0001.yaml` · schema `developer/policy/schemas/root-family-policy.schema.json` · 현재 슬롯 `RESERVED`


## Support: PTSIP_SUPPORT_FEATURE

정의 데이터: [support.json](support.json) · bootstrap: `src/policy/SFP-0024.yaml`

### NORM — SUP_CONSUMER_REQUIREMENT_MEANING

shipped Support 계약에서 consumer에게 적용되는 PTSIP 요구의 효력과 해석 및 immutable bound Specification에 대한 구속 관계를 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `SUP_REQUIREMENT_FORCE_AND_BOUND_SPEC` | consumer 요구의 MUST/MUST_NOT 의미와 bound Specification 우선순위를 명시한다. project-local policy가 universal PTSIP 요구를 약화할 수 없다. | GOV의 project decision이나 consumer가 원하는 INTENT만으로 universal 요구를 면제할 수 없다. |
| `SUP_CONFORMANCE_RULE_INTERPRETATION` | consumer 규범의 적용 조건과 해석 범위를 명시된 Specification revision과 shipped SFP 안에서 정한다. | ASSURE의 zero findings나 관측된 관행이 누락된 규범 의미를 생성하거나 미지원 해석을 승인하지 않는다. |

**소비 경계**

- `INFO`: 명시적으로 선택된 Specification binding의 정보형
- `RECORD`: binding과 규범 출처 provenance

**제외 경계**

- `GOV`: Project Authority의 결정권과 적법한 winner
- `CNTR`: consumer operation 호출 인터페이스와 입출력 약속

**판별 사례:** local 정책으로 PTSIP의 mandatory lifecycle 경계를 완화할 수 없다는 소비자 규칙은 Support NORM이다.

**다음 materialization route:** `src/policy/NORM/SFP-NORM-0001.yaml` · schema `src/policy/schemas/ptsip-support-root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### GOV — SUP_PROJECT_AUTHORITY_LEGITIMACY

consumer Project Authority에서 정확한 subject에 대해 누가 결정할 수 있는지, Authority Role의 효력 및 정당한 coordinated winner를 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `SUP_SUBJECT_ROLE_AND_DECISION_MANDATE` | subject applicability, 결정 주체 및 Authority Role의 효력을 shipped registry의 정확한 identity와 명시적 relationship으로만 해석한다. | ARCH의 완전한 local profile이나 AI의 의미 유사성이 Project Authority와 역할 권한을 만들어 내지 못한다. |
| `SUP_COORDINATED_WINNER_AND_AUTHORITY_SCOPE` | first-valid-resolution-wins와 global authority/local projection 구분에 따라 어느 적법한 결정이 해당 subject의 winner인지 정한다. | CTRL의 fresh read 성공이나 RECORD의 가장 최근 기록이라는 사실만으로 authority winner가 바뀌지 않는다. |

**소비 경계**

- `NORM`: decision legitimacy에 적용되는 shipped 규범
- `CNTR`: authority coordination의 등록된 계약

**제외 경계**

- `ARCH`: winner를 반영한 프로젝트 책임 선언의 내용
- `CTRL`: 특정 mutation 직전 authority freshness와 stale-writer 방지

**판별 사례:** 동일 subject의 첫 유효 결정이 winner인지를 정하는 것은 Support GOV이고 freshness 검사는 CTRL이다.

**다음 materialization route:** `src/policy/GOV/SFP-GOV-0001.yaml` · schema `src/policy/schemas/ptsip-support-root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### INTENT — SUP_DECLARED_OPERATION_AND_PROJECT_INTENT

consumer가 명시적으로 요청한 지원 operation의 목적과 project가 선언한 목표, scope 및 비목표를 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `SUP_EXPLICIT_OPERATION_PURPOSE_AND_SCOPE` | consumer 요청을 지원 operation의 명시적 목적과 변경 범위로 한정하고 요청하지 않은 adoption이나 mutation을 추론하지 않는다. | REAL의 자동 변환 가능성이나 OPS의 회복 편의가 사용자 요청 범위를 확대할 수 없다. |
| `SUP_PROJECT_GOAL_DECLARATION` | 프로젝트가 명시한 결과 목표와 설계 제약을 유지하며 관측된 파일, framework 또는 confidence를 프로젝트 의도로 승격하지 않는다. | ARCH의 책임 분류나 ASSURE의 관측 결과는 프로젝트가 무엇을 달성하려는지 대신 선언할 수 없다. |

**소비 경계**

- `GOV`: 프로젝트 목표와 operation 요청의 정당한 주체
- `CNTR`: 실제로 지원되는 operation 범위

**제외 경계**

- `ARCH`: 목표를 구현하는 durable architecture declaration
- `ASSURE`: 선언과 증거에 대한 규범 충족 판정

**판별 사례:** inspect만 요청한 consumer의 operation 목적을 read-only로 유지하는 것은 Support INTENT이다.

**다음 materialization route:** `src/policy/INTENT/SFP-INTENT-0001.yaml` · schema `src/policy/schemas/ptsip-support-root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### ARCH — SUP_PROJECT_LIFECYCLE_RESPONSIBILITY_DECLARATION

consumer Project Profile와 Responsibility Map의 프로젝트 소유 lifecycle 책임, 역할 및 명시적 typed 관계를 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `SUP_PRIMARY_LIFECYCLE_OWNER_AND_BOUNDARY` | classification을 governing lifecycle obligation에 따른 primary ownership으로 선언하고 material mixed 책임 및 unresolved 소유를 구분한다. | SUPPLY의 dependency 분류나 관측 파일의 executable 여부가 프로젝트 lifecycle ownership을 자동 결정하지 않는다. |
| `SUP_EXPLICIT_ROLES_AND_TYPED_RELATIONSHIPS` | roles, associated artifacts 및 typed directed relationships를 프로젝트의 명시적 architecture 사실로 선언하고 evidence에서 자동 생성하지 않는다. | INFO의 필드 schema나 REAL의 template materialization이 프로젝트 소유 architecture 의도를 만들어 내지 못한다. |

**소비 경계**

- `INTENT`: 명시적인 프로젝트 목표와 제약
- `GOV`: distributed coordination이 적용될 때 정당한 architecture winner

**제외 경계**

- `INFO`: Source/Effective Map 및 필드의 정보 의미
- `ASSURE`: declaration과 실제 evidence의 conformance 평가

**판별 사례:** Product-specific verification의 lifecycle owner를 명시적 obligation에 따라 선언하는 것은 Support ARCH이다.

**다음 materialization route:** `src/policy/ARCH/SFP-ARCH-0001.yaml` · schema `src/policy/schemas/ptsip-support-root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### INFO — SUP_PROFILE_CONTEXT_INFORMATION_SEMANTICS

consumer profile, canonical effective metadata, Context Plane 및 관측 evidence의 정보 의미와 semantic source/projection 구분을 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `SUP_CANONICAL_INFORMATION_MODEL` | profile의 classification, roles, relationships, source_mode 및 evidence 정보형을 서로 다른 의미 축으로 정의한다. | ARCH의 특정 project 선언이나 ASSURE의 판정이 정보 축의 의미를 합치거나 새 canonical alias를 만들 수 없다. |
| `SUP_SEMANTIC_SOURCE_AND_PROJECTION_BINDING` | Source Profile 및 context semantic source와 결정론적 effective/projection 정보의 결합과 non-authoritative derivation 관계를 명시한다. | REAL의 출력 파일 존재나 RECORD의 과거 snapshot은 현재 source 의미를 대신 소유하지 않는다. |

**소비 경계**

- `NORM`: canonical 정보에 적용되는 shipped 규범
- `ARCH`: project가 명시적으로 소유한 architecture 사실

**제외 경계**

- `CNTR`: operation request/result의 외부 약속
- `RECORD`: 과거 observation과 실행 사건의 기록 보존

**판별 사례:** source_mode와 derived origin을 lifecycle classification과 별도 정보 축으로 유지하는 것은 Support INFO이다.

**다음 materialization route:** `src/policy/INFO/SFP-INFO-0001.yaml` · schema `src/policy/schemas/ptsip-support-root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### CNTR — SUP_SHIPPED_OPERATION_CAPABILITY_CONTRACT

설치된 PTSIP가 consumer에게 노출하는 operation과 capability의 입력, 결과, 오류, precondition/postcondition 및 호환 보장을 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `SUP_OPERATION_REQUEST_RESULT_OBLIGATIONS` | 지원 operation ID, typed 입력 및 출력, 오류와 read/write side effect 약속을 shipped contract로 명시한다. | INFO의 profile schema나 내부 구현 함수의 존재만으로 consumer operation이 지원되는 계약을 만들 수 없다. |
| `SUP_CAPABILITY_AVAILABILITY_AND_COMPATIBILITY` | 지원 가능한 capability와 호환 입력의 범위 및 unavailable/reserved 상태의 계약상 의미를 정한다. 미등록 alias나 capability는 실행 계약이 아니다. | OPS의 회복 필요나 CHANGE의 migration 요구가 RESERVED Task/runtime persistence를 활성 지원 계약으로 바꾸지 못한다. |

**소비 경계**

- `INFO`: 입출력 schema와 vocabulary의 정보 의미
- `NORM`: operation이 준수해야 하는 consumer 규범

**제외 경계**

- `CTRL`: 현재 repository/authority 상태에서 호출을 실행해도 되는지
- `REAL`: materialization 및 prepared write의 구현 절차

**판별 사례:** PTSIP-OP-CONFORM-001의 결과 schema와 side effect 약속은 Support CNTR이다.

**다음 materialization route:** `src/policy/CNTR/SFP-CNTR-0001.yaml` · schema `src/policy/schemas/ptsip-support-root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### RISK — SUP_AUTOMATION_UNCERTAINTY_AND_CONFLICT_RISK

consumer 지원 동작의 증거 공백, 불확실성 및 authority/mutation 충돌 위험과 그 처리 한계를 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `SUP_UNCERTAINTY_AND_CONFLICT_TREATMENT` | 누락 evidence, unresolved subject와 conflict가 유발하는 위험 및 추가 사실 확보 또는 사용자 판단이 필요한 조건을 명시한다. | ASSURE의 finding 개수나 REAL의 변환 성공률만으로 관측하지 못한 위험이 제거되지 않는다. |
| `SUP_RISK_ACCEPTANCE_WITHOUT_AUTHORITY_SUBSTITUTION` | 명시적 위임 안에서만 잔여 위험 처리 조건을 정한다. AI confidence, 위험 수락 또는 편의상 alias가 normative conformance, authority winner나 무권한 쓰기를 대신 승인하지 않는다. | GOV의 범위 제한된 승인이나 INTENT의 목표가 universal MUST 위반과 unresolved authority를 허용하지 않는다. |

**소비 경계**

- `ASSURE`: 검증 evidence의 확정 결과와 공백
- `GOV`: 위험 처리 결정 주체의 명시적인 권한

**제외 경계**

- `ASSURE`: CONFORMANT/NON_CONFORMANT/INCOMPLETE의 규범 판정
- `CTRL`: 불확실성 존재 시 특정 mutation의 실행 허용 여부

**판별 사례:** 관측 evidence가 없어 automation 오판 위험을 식별하는 것은 Support RISK이고 INCOMPLETE 판정은 ASSURE이다.

**다음 materialization route:** `src/policy/RISK/SFP-RISK-0001.yaml` · schema `src/policy/schemas/ptsip-support-root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### SUPPLY — SUP_PROJECT_DEPENDENCY_AND_ARTIFACT_SUPPLY

consumer project의 dependency, 외부 evidence 및 distribution Artifact 공급 관계의 명시적 입력 자격과 출처를 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `SUP_DEPENDENCY_SOURCE_AND_ADMISSION` | consumer가 선언하거나 정확하게 등록한 dependency/source identity, 출처 및 공급 요구를 해석한다. unregistered alias나 관측 import에서 프로젝트 요구를 추론하지 않는다. | ARCH의 lifecycle ownership이나 observed dependency 목록만으로 dependency 공급 요구와 허용된 source 관계가 결정되지 않는다. |
| `SUP_SUPPLIED_ARTIFACT_PROVENANCE_AND_HANDOFF` | imported evidence와 소비하는 Artifact 단위의 provenance 및 공급 인계 관계를 명시하고 생산자와 consumer-side artifact owner를 구별한다. | ASSURE가 Artifact를 검사했다는 사실만으로 출처와 공급 관계가 선언되거나 architecture owner가 결정되지 않는다. |

**소비 경계**

- `ARCH`: project-declared dependency와 Artifact 책임 경계
- `CNTR`: 공급 interface와 호환 입력 조건

**제외 경계**

- `ARCH`: Product/Delivery 등 primary lifecycle ownership
- `ASSURE`: built Artifact의 PTSIP-PKG-001 conformance 검사

**판별 사례:** 외부 dependency evidence의 exact source provenance와 허용된 명시적 alias 관계는 Support SUPPLY이다.

**다음 materialization route:** `src/policy/SUPPLY/SFP-SUPPLY-0001.yaml` · schema `src/policy/schemas/ptsip-support-root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### REAL — SUP_DETERMINISTIC_CONSUMER_STATE_REALIZATION

consumer가 선택한 source 선언을 deterministic effective metadata와 허용된 repository 변경으로 실현하는 의미 보존 실행 절차를 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `SUP_DECLARATION_PRESERVING_MATERIALIZATION` | 명시적으로 선택된 immutable template/source에서 effective map과 projection을 결정론적으로 materialize하고 source facts 및 derived provenance를 보존한다. | ARCH의 설계 의도나 INFO의 schema가 실제 materialization 결과를 생성하지 않으며 생성 과정도 새 architecture 의도를 추론할 수 없다. |
| `SUP_PREPARED_WRITE_AND_FAILURE_INTEGRITY` | 허용된 prepared write의 적용 단위, 원자성과 실패 보존을 정한다. read-only operation은 repository-local state를 materialize하지 않는다. | CTRL의 fresh eligibility나 GOV의 승인은 쓰기 중 partial mutation과 실패 보존의 실행 mechanics를 대신 보장하지 않는다. |

**소비 경계**

- `INFO`: 선택된 semantic source 및 projection 정보 의미
- `CTRL`: 쓰기 직전 검증된 exact target과 precondition

**제외 경계**

- `ARCH`: project가 선택한 architecture 선언
- `CHANGE`: legacy 의미에서 새 선언 의미로 대응시키는 migration 결정

**판별 사례:** 명시적 template revision에서 Effective Responsibility Map을 생성하는 행위는 Support REAL이며 ownership 추론은 허용되지 않는다.

**다음 materialization route:** `src/policy/REAL/SFP-REAL-0001.yaml` · schema `src/policy/schemas/ptsip-support-root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### ASSURE — SUP_DECLARATION_EVIDENCE_CONFORMANCE_JUDGMENT

consumer declaration과 실제 evidence가 적용 규범을 충족하는지, evidence가 충분한지 및 snapshot-bound Artifact conformance를 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `SUP_THREE_OUTCOME_CONFORMANCE_EVALUATION` | mandatory 위반과 blocking evidence gap을 구별하여 완료 결과를 CONFORMANT, NON_CONFORMANT 또는 INCOMPLETE로 판정한다. zero findings는 자동 conformance가 아니다. | GOV의 coordinated winner나 완전한 ARCH 선언만으로 관측 evidence까지 규범에 적합하다고 판정할 수 없다. |
| `SUP_EVIDENCE_COVERAGE_AND_ARTIFACT_BINDING` | coverage와 admissibility 및 실제 built Artifact/source snapshot 결합을 평가하고 functional/VPMS PASS와 PTSIP conformance를 독립적으로 유지한다. | RECORD의 receipt 존재나 SUPPLY의 Artifact provenance만으로 필요한 검사와 conformance 판단이 완료되지 않는다. |

**소비 경계**

- `NORM`: 선택된 bound Specification 및 applicable SFP 규칙
- `ARCH`: 평가할 명시적 project architecture declaration

**제외 경계**

- `CTRL`: conformance 결과를 소비하는 실행 gate
- `RECORD`: 관측 결과와 판정 receipt의 보존

**판별 사례:** profile이 완전해도 실제 Artifact evidence가 누락되면 INCOMPLETE로 판정하는 것은 Support ASSURE이다.

**다음 materialization route:** `src/policy/ASSURE/SFP-ASSURE-0001.yaml` · schema `src/policy/schemas/ptsip-support-root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### CTRL — SUP_FRESH_CONSUMER_ACTION_ELIGIBILITY

consumer repository/profile/authority의 fresh state를 기준으로 특정 prepared mutation이나 transition의 현재 실행 허용 여부를 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `SUP_FRESH_REPOSITORY_PROFILE_AUTHORITY_GATE` | 실행 직전 repository/profile revision과 관련 distributed authority를 확인한다. 완전한 local declaration만으로 필요한 remote authority 읽기를 생략하지 않는다. | GOV의 이전 winner나 ASSURE의 과거 PASS는 현재 profile/authority freshness를 대신 보장하지 않는다. |
| `SUP_STALE_WRITER_AND_PREAUTHORIZED_TRANSITION_GATE` | expected revision, operation identity와 등록된 transition 조건으로 stale writer 및 무권한 mutation을 차단한다. unresolved 관계는 fail-closed한다. | REAL의 원자적 write 구현이나 OPS의 재시도는 stale 상태에 대한 실행 허용을 만들 수 없다. |

**소비 경계**

- `GOV`: 정당한 authority winner 및 mutation 위임
- `CNTR`: 등록된 action과 transition의 precondition

**제외 경계**

- `GOV`: 누가 결정할 수 있는지와 first-valid winner 의미
- `REAL`: 허용된 mutation의 적용 mechanics

**판별 사례:** prepared profile 이후 source revision이 바뀌었으면 쓰기를 거부하는 것은 Support CTRL이다.

**다음 materialization route:** `src/policy/CTRL/SFP-CTRL-0001.yaml` · schema `src/policy/schemas/ptsip-support-root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### CHANGE — SUP_EXPLICIT_PROFILE_AUTHORITY_MIGRATION

consumer의 기존 profile, context 및 authority 상태에서 명시된 target 의미로 이동하는 변경 대응과 compatibility 및 staged migration을 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `SUP_SOURCE_TARGET_SEMANTIC_MAPPING` | 지원된 source와 target의 의미 대응, 보존할 사실과 unresolved 선택을 명시한다. legacy TOOLCHAIN, lifecycle_owner 또는 version 문자열을 canonical 의미로 기계 치환하지 않는다. | INFO의 새 schema 유효성이나 REAL의 parser 성공만으로 역사적 상태의 의미가 target과 동등해지지 않는다. |
| `SUP_COMPATIBILITY_ORDER_AND_RETIREMENT` | 명시적으로 등록된 migration 경로, 호환 입력의 범위, 선행 validation 및 legacy retirement 조건을 정한다. 문서나 인접 version에서 누락된 hop을 추론하지 않는다. | CNTR의 호환 약속이나 OPS의 회복 절차만으로 semantic replacement와 legacy deletion이 허용되지 않는다. |

**소비 경계**

- `CNTR`: 지원된 migration operation과 호환 입력 계약
- `INFO`: source 및 target의 정보 의미

**제외 경계**

- `GOV`: semantic replacement에 필요한 사용자 승인과 정당성
- `REAL`: 변환 결과의 실제 생성과 적용

**판별 사례:** 0.3.5 관측 사실을 보존하면서 0.3.6 declaration으로 해석할 명시적 대응을 정하는 것은 Support CHANGE이다.

**다음 materialization route:** `src/policy/CHANGE/SFP-CHANGE-0001.yaml` · schema `src/policy/schemas/ptsip-support-root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### OPS — SUP_AVAILABLE_CAPABILITY_CONTINUITY_AND_RECOVERY

설치된 Tool과 consumer repository에 이미 제공된 capability의 지속 운영, 장애 회복 및 authority를 보존하는 반복 reconciliation을 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `SUP_OPERATIONAL_RECOVERY_WITH_AUTHORITY_PRESERVATION` | 지원된 operation의 실패 분류, 회복과 canonical namespace 유지 조건을 정한다. 충돌 source, stale lock 또는 모호한 journal을 임의 폐기하지 않는다. | REAL의 단일 성공 write나 SUPPLY의 환경 공급이 장애 이후 authority 보존과 운영 회복 책임을 대체하지 못한다. |
| `SUP_IDEMPOTENT_RETRY_AND_RECONCILIATION` | 계약에 정의된 operation ID와 expected revision을 보존해 재시도하고 conflict를 검토 가능하게 유지한다. RESERVED .ptsip/tasks 또는 .ptsip/runtime의 미정 persistence 의미를 새로 정의하지 않는다. | CTRL의 한 번의 실행 허용이나 CHANGE의 migration 경로는 반복 시도의 동일성 및 운영 연속성을 대신 정의하지 않는다. |

**소비 경계**

- `CNTR`: 실제로 사용 가능한 capability의 오류와 recovery 계약
- `CTRL`: 각 retry/recovery의 현재 action eligibility

**제외 경계**

- `CHANGE`: capability 활성화 및 semantic source 교체의 전환 의미
- `RECORD`: operation receipt와 recovery 이력의 보존

**판별 사례:** 기존 operation ID와 expected revision을 유지하며 충돌을 숨기지 않고 재시도하는 것은 Support OPS이다.

**다음 materialization route:** `src/policy/OPS/SFP-OPS-0001.yaml` · schema `src/policy/schemas/ptsip-support-root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

### RECORD — SUP_OBSERVATION_AND_OPERATION_PROVENANCE

consumer repository observation, authority 참조 및 operation 결과의 source-bound receipt와 감사 가능한 이력 보존을 소유한다.

| exclusive_kernel | 최소 권한 | 대체할 수 없는 이유 |
|---|---|---|
| `SUP_OBSERVATION_RECEIPT_SOURCE_BINDING` | report와 receipt를 실제 관측 repository/profile/authority revision 및 Artifact snapshot에 결합하고 다른 시점의 결과를 섞지 않는다. | ASSURE의 conformance 판정이나 CTRL의 실행 허용은 그 결과의 재확인 가능한 source provenance를 자동 보존하지 않는다. |
| `SUP_TRACEABILITY_RETENTION_AND_RETRIEVAL` | operation identity, observed evidence와 판단 출처의 보존 및 조회를 정한다. 기록은 새로운 global authority나 live canonical source가 되지 않는다. | GOV의 현재 winner나 INFO의 semantic source는 이전 사건의 history와 원본 receipt를 대체하지 못한다. |

**소비 경계**

- `INFO`: report 및 receipt의 정보형과 identity 의미
- `CNTR`: operation 결과로 약속된 receipt와 출처 필드

**제외 경계**

- `GOV`: project decision winner와 현재 권한의 결정
- `ASSURE`: 보존된 evidence의 conformance 적합성 판정

**판별 사례:** read-only conformance report를 검사한 snapshot과 함께 보존하는 것은 Support RECORD이다.

**다음 materialization route:** `src/policy/RECORD/SFP-RECORD-0001.yaml` · schema `src/policy/schemas/ptsip-support-root-family-policy.schema.json` · 현재 슬롯 `RESERVED`

