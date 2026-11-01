# 공통 정책 계약 필드 후보 v2

공통 정책 구조 v2는 활성 등록되었지만(`active_registration: true`), 개별 필드·책임 배정과 의미 승인에 대한 검토는 아직 완료되지 않았다(`field_review_finalized: false`).
GOV 한 문서에 맞춘 v1 실제 계약·고정 검사 목록·기계적인 역할 조합은 공통 사용 경로에서
제외했다. 그 파일은 후보 인덱스의 `superseded_artifacts`에 남겨 두며 새 문서의 적용 근거로
사용하지 않는다. 현재 활성 라우팅과 Go `policy-check`는 v2 공통 공개 스키마 및 `Family_Selection.json`을 선택한다. 철회된 v7 Family 구조 검증은 활성 진입에서 사용하지 않는다.

## 비교 대상과 적용 한계

[비교 데이터](../review/Policy_Field_Comparison.json)는 같은 공통 스키마로 다음 다섯 원본 경로를
시범 작성한 결과다. 특정 정책을 통과시키는 별도 필드나 스키마 분기는 없다.

| 정책 | 구조에서 구별해야 하는 내용 |
|---|---|
| GOV-0001 | 규칙 선언과 실제 의미·대상·책임 배정의 미확정 |
| ARCH-0001 | 생명주기 소유·구성 요소 경계·대상 영역의 방향성 관계와 책임 결합 관계의 구별 |
| ARCH-0002 | 정책·계획·구현의 권한 계층과 승인 독립성. 시간적 선후 관계로 자동 변환하지 않음 |
| CHANGE-0007 | 퇴역·소유권 이전 기록, 원문 보존·의미 검토·수락 판단의 구별, 과거 미완료 결과 |
| RECORD-0001 | 한 R-ID에 여러 독립 요구사항, 명시된 세 하위 정책의 소유 경계와 총칙의 조정 책임 |

모든 식별·상태 메타데이터는 보존한다. CHANGE-0007은 `RETIRED.Waiting`이며,
과거 문언을 표현하는 구조 사례일 뿐 현행 규범으로 재활성화하지 않는다.
현재 프로필에 없는 두 R-ID를 검사를 통과시키기 위해 추가하지 않는다.
원본의 평가 결과는 `historical_result_refs`로 보존하며 새 평가 결과로 복제하지 않는다.

## 소유와 공통 필드

계약은 `rules[]`에 책임 선언을, `requirements[]`에 개별 요구사항을 둔다.
요구사항을 규칙 아래에 넣었다는 이유로 책임을 배정하는 구조를 제거했다.
요구사항은 `requirement_id`로 식별하며 `responsibility_refs`는 요구사항 형식에서 제거했다. 기존 R-ID 선언은 `rules[].id`에 보존한다.

| 필드 | 소유하는 책임 |
|---|---|
| 계약의 contract_id, Rule_ver | 계약 식별과 자체 직렬화 버전. 특정 값은 공통 스키마에 고정하지 않음 |
| rules[].id, rule, preconditions | 기존 R-ID·규칙명·C-ID 선언. 이름으로 의미나 결합 연산을 추론하지 않음 |
| rules[].source_ref, authority_ref | source_ref는 같은 계약의 source_items.item_id를 가리키는 비어 있지 않은 ID 배열이다. 원본 좌표 객체는 source_items가 보존하며 authority_ref는 별도 권위 원본 참조다 |
| requirements[].requirement_id | 개별 요구사항의 식별자. 계약 안에서 유일해야 하며 R-ID 선언과 구별 |
| meaning_refs | 원본의 정확한 항목·리비전. 복수 원본이 의미를 함께 뒷받침할 수 있음 |
| target_refs | 실제 대상 식별. 공통 scope 문장을 실제 대상의 대용물로 넣지 않음 |
| scope_refs | 적용 범위·상한. 책임별 대상이나 권한의 생성과 구별 |
| condition_refs | 요구사항 적용 조건의 의미 원본·계약. 규칙의 preconditions를 묵시적으로 상속·AND 결합하지 않음 |
| constraint_refs | 금지·비승계·소유 경계·추론 제한의 의미 참조 |
| criteria_refs | 원본이 요구하는 충족 기준. 실제 증거·판정 결과와 구별 |
| evidence_requirement_refs | 적격 증거를 정의하는 등록 계약의 선택 참조. 증거 자체와 구별 |
| relation_binding | 관계 조합의 선언·미확정·비적용 근거 |
| judgement_binding | 요구사항 판정 계약의 선언·미확정·비적용 근거 |
| unresolved_fields | 내용·배정이 미확정인 공통 필드의 명시적 목록 |
| empty_field_basis | 확인된 공집합을 선언할 때 그 필드와 독립 근거를 명시 |
| source_items | 원본의 미배정 조항·기준·경계·주석·기록을 보존하는 비권위적 목록 |
| historical_result_refs | 과거 평가 기록의 원본 참조. 현재 수행 건의 결과를 주장하지 않음 |

필수 필드는 작성 상태를 빠짐없이 보고하기 위한 형식이다.
모든 요구사항에 조건·예외·외부 증거 계약이 반드시 존재한다는 새 규범 의무를 만들지 않는다.
값이 없는 선택 필드는 빈 배열·빈 객체·null·빈 문자열로 쓰지 않고 생략한다. 미확정은 값이 있는 `unresolved_fields` 또는 바인딩 상태로 보존하고, 확인된 공집합은 실제 필드를 생략한 뒤 `empty_field_basis`에 근거를 남길 수 있다.
일부 참조를 선언했어도 그 배정이 덜 확인됐으면 `unresolved_fields`에 남길 수 있다.
목록의 선언이나 공집합 근거 참조는 그 근거의 실질적 타당성을 검증한 결과가 아니다.

`source_items.role`은 작성·보존을 위한 분류이며 원본의 규범 의미를 새로 소유하지 않는다.
CHANGE의 기존 scope 문장은 퇴역·이전 기록이므로 record로 보존하고 실제 scope 배정은
미확정으로 남겼다. 규범적 금지를 단순한 제외 대상으로 바꾸지 않았다.

RECORD의 세 하위 문서 연결은 총칙이 그 소유 경계를 명시한다는 후보 표현이다.
하위 기록 수행 책임·상세 요구사항·R-ID를 총칙으로 가져오지 않는다.
부모의 R-ID는 규칙 선언에만 보존하고, 여섯 요구사항은 각각 requirement_id로 식별한다. 하위 정책의 수행 책임을 총칙에 이전하지 않는다.

## 공통 바인딩 상태

관계와 판정은 같은 닫힌 바인딩 형식을 쓴다.

- `DECLARED`: 선택 참조와 원본 근거를 선언한다. 입력이 있으면 등록된 포트에 참조로 바인딩한다.
- `UNRESOLVED`: 확인이 필요한 원본 참조를 남긴다. 가짜 계약이나 역할 조합을 만들지 않는다.
- `NOT_APPLICABLE`: 독립 원본 근거를 선언한다. 생략이나 실패를 비적용으로 바꾸지 않는다.

이 상태는 작성 상태이며 PASS/FAIL 등의 실제 평가 결과가 아니다.
DECLARED는 참조 해석·정확한 버전 검증·심사 A·B·승인 완료를 뜻하지 않는다.
NOT_APPLICABLE, 특히 판정 연결의 비적용도 요구사항 자동 충족을 뜻하지 않는다.
선택된 정확한 계약 버전의 필수 포트·중복 바인딩·자료형·대상·평가 문맥 대조는 후속 검사 대상이다.

승인된 다섯 관계의 고유 의미, 심사 A·B와 대응 대조, 기본·조건부 관계의 구별,
전체 적용 관계 결과 보존, 원본 기준에 근거한 별도 요구사항 판정의 설계 경계는 유지한다.
Family 이름이나 문장의 계층·화살표·관계라는 표현만으로 유형을 자동 선택하지 않는다.

## 스키마와 등록 데이터의 분리

공통 Registry는 계약 리비전과 호환 집합을 표현한다. 특정 리비전·작성 단계·승인 상태를 고정하지 않는다.
선택 계약은 등록된 선택 규칙·입력 포트·출력 계약을 참조한다. 특정 선택 알고리즘을 공통 필드로 강제하지 않는다.
판정 계약과 Judgement_Rule은 의미 원본·입력 포트·기준·판정 규칙·결과 계약을 참조한다.
GOV용 source_review_ref, 고정 Boolean 검사 목록, 고정 결과 문자열을 제거했다.
실제 알고리즘·결과 어휘·버전 승인 범위는 별도로 등록된 데이터 계약이 소유해야 한다.

Evaluation_Record는 공통 문맥·고정 버전·증거 바인딩을 유지하면서 관계가 없는 평가도 표현한다.
관계 결과를 기록한다면 심사·적용·관계 결과·요약의 연결 구획을 함께 요구한다.
v2 공통 구조의 Go 검증기와 계약 선택은 후속 전환에서 활성화되었다. 관계·판정 의미의 실질 평가와 정책 승인·실행 권한 부여는 여전히 이 필드 비교의 범위 밖이다.

## 남은 검토

원본 문장과 특정 R-ID의 대응이 명시되지 않은 ARCH·CHANGE·GOV 항목은 이름으로 배정하지 않았다.
원문 복합 문장의 최종 원자 경계, 책임 배정, 대상·조건·제약·기준의 등록 항목,
선택·판정 계약과 포트 바인딩의 실제 의미는 추가 검토 대상이다.
구조 검사를 통과해도 의미 동등성·적용 승인·정책 충족을 주장하지 않는다.

## 2026-11-02 Family별 필드와 소유 경로

기존 `policy/Rules/*_rules.json` 14개는 구조 비교 자료로만 사용한다. 활성 선택·작성은
`Policy_Rules/Family_Selection.json`과 `Public_Policy_Rules.schema.json`을 따른다.
`schemas/Policy`와 `schemas/compile-sealed-rbac`의 스키마는 현재 등록·소비 경로에서 제외한다.
식별·생명주기와 규칙 레코드는 `contracts/policy`, RBAC 계약은 `contracts/compile-sealed-rbac`에 둔다.
역사 자료와 검증 영수증의 좌표·해시는 현재 경로로 다시 쓰지 않는다.

모든 Family는 계약 식별·버전, 규칙 ID·이름·원본 항목, 독립 요구사항 ID와 원본 의미·범위를
표현한다. Family별 차이는 아래 검토 관점을 추가하는 방식이며, 별도 자연어 필드를 만들지 않는다.

| Family | 대상 관점 | 제약 관점 | 판정 기준 관점 | 증거 요구 관점 |
| --- | --- | --- | --- | --- |
| NORM | 필수 검토 | 선택 | 선택 | 선택 |
| GOV | 필수 검토 | 선택 | 필수 검토 | 필수 검토 |
| INTENT | 선택 | 필수 검토 | 필수 검토 | 선택 |
| ARCH | 필수 검토 | 선택 | 필수 검토 | 선택 |
| INFO | 필수 검토 | 선택 | 필수 검토 | 선택 |
| CNTR | 필수 검토 | 선택 | 필수 검토 | 선택 |
| RISK | 필수 검토 | 선택 | 필수 검토 | 선택 |
| SUPPLY | 필수 검토 | 선택 | 필수 검토 | 필수 검토 |
| REAL | 필수 검토 | 선택 | 필수 검토 | 선택 |
| ASSURE | 필수 검토 | 선택 | 필수 검토 | 필수 검토 |
| CTRL | 필수 검토 | 선택 | 필수 검토 | 선택 |
| CHANGE | 필수 검토 | 선택 | 필수 검토 | 선택 |
| OPS | 필수 검토 | 선택 | 필수 검토 | 선택 |
| RECORD | 필수 검토 | 선택 | 필수 검토 | 필수 검토 |

필수 검토는 원본에 없는 의무를 추가하지 않는다. 원본 참조, 명시적 미결정,
원본 근거가 있는 값 없음/비적용을 구별한다. 미결정 필드는 값을 생성하지 않고
`unresolved_fields`에 남긴다. 빈 필드는 생략한다. 기존 필수 배열은 빈 배열을 허용했으므로
필수 키였다는 이유만으로 새 대상·예외·증거 의무가 존재한다고 추정하지 않는다.

공통 Envelope와 관점별 제약은 `Public_Policy_Rules.schema.json`의 `$defs`에서 재사용하며,
Family별 조합도 같은 파일이 소유한다. 상위 공개 구조는 한 번만 위임한다.
정책 제목은 `Policy_Identity.title`이 소유하고 규칙에 중복 제목을 요구하지 않는다.

정책 문서의 생명주기 구획은 `Policy_Lifecycle`이다. 내부 상태 값과 버전·계약 값은 유지한다.
규칙별 권한 연결은 `authority_ref: [AUTH-0001]`로 쓰고, 같은 계약의
`authority_ref[].id`에 대응시킨다. 권한 원본 좌표 또는 미결정 선택 바인딩은 계약 목록에
한 번 보존한다. ID의 연결 검사는 권한의 유효 승인·행사 가능성을 대신하지 않는다.
2026-11-01에 제출된 로컬 검증 로그에서 `MPD-ARCH-0001`의 v2 `policy-check`는 `VALID`(범위: `STRUCTURE_AND_DECLARED_REFERENCES`)이고 세 Go 테스트 패키지는 `ok`이다. 증거는 `.ptsip/engine/developer/policy/verification/v2/evidence/MPD-ARCH-0001-ed873e7b.json`과 동일 경로의 원본 `.log`에 보존한다. 이는 정책 준수·의미 동등성·생명주기 전환 또는 실행 권한의 승인을 의미하지 않는다.


## CHANGE-0001 공통 필드 후보 B 실험 (미승인)

기준 SHA: `9dfd231b1c385dd8df9d6ad5cbcfe8c3a1f3a549`. 총칙 원본의 2개 R-ID, 5개 subjects, 5개 exclusions, 3개 evidence 기준, 원본 scope와 과거 `incomplete_fulfillment`를 19개 항목 식별자로 추적한다. 원본 텍스트는 기준 리비전이 보존한다. 4개 요구사항은 **후보 그룹화**이며 원자성·책임 연결은 확정되지 않았다.

선택한 B 후보는 `source_documents`(경로·정확한 revision), `source_items`(원본 항목·비권위적 항목 분류), `bindings`(용도·상태·참조 ID)를 분리한다. `SOURCE_IDENTIFIED`는 출처를 식별했다는 뜻이지 원문 의미·규칙 충족·관계 판정의 승인 의미가 아니다. 빠진 용도는 `UNRESOLVED`이며 확인된 공집합과 비적용은 별도 근거를 요구한다. 독립적인 `unresolved_fields` 중복 보고는 없다. 원본 0009에 속하는 상세 규칙을 총칙으로 복제하지 않는다.

시험 후보 파일: `.ptsip/contracts/policy/Policy_Rules/Field_Binding_Trial.schema.json`. 비교·반례: `.ptsip/contracts/policy/review/CHANGE-0001-Field-Trial.json`. 이 파일들은 **미등록 작성 시험**이며 현재 v2 라우팅, 검증기, 승인을 변경하지 않는다. 대상 YAML은 현재 등록된 v2 스키마에 불일치하는 것이 예상되므로 이를 은폐하거나 legacy 본문과 병기하지 않는다.


### 동일 요구사항 자료의 A/B 계측 (승인 전 비교)

공통 출처 색인 19항목을 양 후보에서 똑같이 쓰고, 임시 4개 요구사항만 compact JSON으로 직렬화했을 때 **A 1,830자 / B 1,470자**다. 요구사항의 직접 필드 슬롯은 **A 48 / B 8**이며, 별도 미확정 필드 이름 보고는 **A 16개 / B 0개**(대신 B는 직접 미확정 바인딩 4개)다. 이는 시뮬레이션의 제한된 값이며 유효한 의미 배정이나 실행속도 개선 수치가 아니다. B는 목적과 상태를 분리하여 기계 해석을 단순화하지만 ID 조회 단계가 추가되고 관계별 추론이 완전히 제거되는 것은 아니다.

공통 후보는 현재 Identity/Lifecycle/R-ID 타입을 자체적으로 다시 정의하지 않고 기존 외부 계약을 참조한다. 타 정책·특정 Family·리비전 값을 공통 필드의 상수로 넣지 않았다.


## CHANGE-0001 작성 연결 정정 (2026-11-01)

현재 Policy_Rules / Contract_Types v2에 등록된 식별 작성 유형 `INDEXED_SOURCE_BINDINGS`을 사용하도록 교정한다. 원본은 일반 블록 YAML이고, 별도 Field_Binding_Trial.schema.json은 미등록·퇴역 시험으로만 보존한다. 한 문서에는 한 개의 작성 형식만 존재한다. 원본 주소와 리비전 `source_documents`, 원본 포인터 `source_items`, 범주별 참조·작성 상태 `source_bindings`, 별도 책임 R-ID `responsibility_binding`, 증거 요건·관계·판정 선택 `IndexedSelectionBinding`은 모두 공통 타입에 정의된다. 판정 계약은 DECLARED가 아닌 UNRESOLVED로 유지한다. v2 legacy inline 정책은 동일 공통 타입 내에서 독립적으로 계속 인정되지만 새 indexed 문서가 실패할 때 auto fallback 하지 않는다.

비교·반례: `review/CHANGE-0001-Field-Trial.json`의 correction. 등록 검사: Go `contract-entry` -> `policy-check`, 여기에 indexed 참조 해석 검사를 추가했다. 해당 Go 실행과 외부 `$ref` 포함 정식 스키마 검증은 아직 환경상 미실행이며 PASS로 간주하지 않는다. 


### 등록된 v2에서 수행한 검증 (GitHub Actions)

실제 대상 문서와 다른 정책 5건은 `go -C .ptsip/engine/developer test ./policy/verification/v2 ./internal/contractentry ./policy/test -count=1` 의 새 테스트 사례를 통해 구조·참조 검증됨. Ubuntu 24.04와 Windows 2025 모두 2026-11-01 Actions run `41319900563`에서 0 종료 / 성공. 근거: https://github.com/Kinirin/PTSIP/actions/runs/41319900563. 검증이 선택한 public schema: `urn:ptsip:candidate:contract:policy:public-structure:v2` / rules schema: `urn:ptsip:candidate:contract:policy:rules:v2` / indexed type: `urn:ptsip:candidate:contract:policy:types:v2#/$defs/IndexedContract`. 

정식 등록 구조의 PASS는 중복·미해결 ID·원본 역할 타입·작성 상태의 기본 정합성에 한한다. source SHA의 완전한 역사적 pointer 내용 검증, 원문 의미 보존의 최종 승인, 실제 책임 배정, 관계 적합성 및 적용 완료를 대신하지 않는다. 새 본문은 기존 시험 문서 9,049자에 비해 10,002자로 늘었지만 원본 경로와 리비전 선언은 각 1개이며 19개 항목 주소를 공유한다. 크기 축소가 아니라 중복 데이터 저장과 반복 해석의 구조적 제거를 시험한 것이다.

## Consolidated identity, status and version owners

`Policy_Identity/Public_Policy_Identity.json` owns the policy-kind and Root Family vocabularies
in its named `$defs`. Routing, Family selection and identity validation reference that single file.
`Policy_lifecycle/status/Policy_status.json` owns the combined DRAFT, APPROVED and RETIRED
string vocabulary. The shared status record and lifecycle wrapper reference it without copying values.
`Policy_lifecycle/version/Policy_version.json` owns version-token generation shape only; the existing
string constraint is preserved without introducing a new lexical pattern or increment algorithm.
The reserved `policy/Plans/` directory and its catalog entry have been removed.

## GOV required structure and omitted empty fields

GOV field presence requirements are owned by `Public_Policy_Rules.schema.json` under
`$defs.PolicyDocumentRules`. The public policy envelope delegates through one reference and
does not accumulate Family-specific conditional bodies. Contract headers keep `contract_id`
as a string sibling of `rules`, `requirements` and `source_items`; rules do not nest inside it.
GOV contracts require rules, requirements and source items, with rule id/rule/source_ref and
requirement_id identifiers. `unresolved_fields` is required when pending bindings exist and
is omitted when empty. Empty optional fields are never emitted. Omission does not imply
approval, satisfaction or non-applicability.

`rules[].source_ref` is an array of contract-local item IDs, never an inline source address.
Both source representations use `source_items[].item_id`. Missing/duplicate item IDs,
dangling rule references and duplicate requirement IDs fail the reference-integrity stage.
`responsibility_refs` and its redundant requirement binding are removed from requirement
instances; explicit responsibility participants in an independent relation remain a separate
contract responsibility.
