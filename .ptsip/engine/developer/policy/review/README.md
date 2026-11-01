# 정책 자연어 검토를 위한 보조 작업

GitHub Actions manual dispatch, reusable workflow inputs, artifact retrieval,
and web-agent access are documented in [ACTIONS.md](ACTIONS.md).

이 문서는 정책 개정 작업을 돕는 비규범적 작업 안내다. 정책·승인·계약·라우팅의 원본이 아니며, 사용자의 현재 지시와 유효한 기계 계약을 대신하지 않는다. 이 도구와 아래 작업 순서는 2026-11-01 사용자가 요청한 번역 효율, 계약 확장 보고, 책임 분리 검토를 지원한다. 다른 작업에 자동 적용되는 새 정책으로 등록하지 않았다.

## 작업 순서

1. 정확한 대상 정책에 기존 `contract-entry --dry-run`을 실행한다. 현재 계약을 선택할 수 없는 구 문서는 `UNRESOLVED` 상태를 유지하며 역사적 해설만 작성한다. 파일명으로 Family를 추정하거나 구 등록부를 복원하지 않는다.
2. 대상 문서는 전체를 읽고, `source_scope` 등 원문 출처는 관련 구간을 사람이 지정한다. 출처 목록만으로 관련 규칙을 판정하지 않는다. 처음 지정한 파일·JSON Pointer·선정 이유를 검토본에 남겨 다음 작업에 재사용한다.
3. 보조 도구로 지정 구간과 파일 SHA-256을 수집한다. 원문, 번역, 번역자의 해석·추정, 차이·미해결 사항을 구별한다. 의무·금지·허용·조건·예외·대상·상태를 보존하고, 과거 구현 관측을 현재 규칙이나 현재 구현 완료로 바꾸지 않는다.
4. 자연어 검토본을 사용자에게 제공하고 새 자연어 정책 선언을 기다린다. 번역 완료나 구조 검증 통과를 정책 내용 승인으로 취급하지 않는다.
5. 새 선언을 받으면 기계 정책을 쓰기 전에 아래 책임 분리 검토와 기존 필드 매핑을 작성한다. 기존 계약 안에서 표현할 수 있는 부분은 기존 필드·등록 참조를 사용한다.
6. 계약 부족이 확인된 부분은 부족한 의미, 기존 필드로 표현하지 못하는 이유, 대안, 확장 범위를 보고한다. 사용자가 확정하지 않은 필드·구조·조건·권한을 만들어 공백을 메우지 않는다. 승인된 내용이 있어도 직렬화 계약이 없으면 해당 부분의 규범적 작성을 보류한다.
7. 작성 후 이 도구의 필드 목록 점검과 기존 `contract-entry`/`policy-check`를 함께 실행한다. 필드 목록 점검, JSON Schema 적합성, 등록 참조, 의미 보존, 사용자 승인, 실행 권한은 서로 다른 결과로 보고한다.

## 기존 필드 매핑과 확장 보고

작성 전 다음 표를 작업별로 채운다. 이 표는 검토 자료이며 새 정책 필드가 아니다.

| 새 자연어 내용 | 책임/조건/경계/평가 구분 | 기존 필드 또는 등록 참조 | 표현 불가 부분 | 처리 |
| --- | --- | --- | --- | --- |
| 사용자가 제시한 문장 | 의미 역할 | 실제로 선택된 계약의 경로 | 없으면 없음 | 기존 구조 사용 / 질문 필요 |

`Identity`, `Definition`, `Boundary`, `Evaluation`, `title`을 현재 Family 계약에서 확인한다. 단어가 비슷하다는 이유로 경계나 평가 문자열에 실행 규칙을 숨겨 넣지 않는다. `Definition[].id`, `rule`, `preconditions`도 기존 책임·규칙·조건 소유자의 등록과 일치해야 한다. 새 규칙 내용의 승인이 새 R-ID·조건·권한 등록을 자동으로 뜻하지 않는다.

확장이 필요할 때는 최소한 다음을 보고한다.

- 기존 구조에서 표현할 수 없는 의미와 실제 실패 예시
- 기존 필드·등록 참조 재사용, 별도 책임 문서, 계약 확장의 대안
- 같은 Family의 다른 정책에서도 계속 사용할 수 있는지
- 공통 계약과 다른 Family에 미치는 영향, 호환성과 검증기 변경 범위
- 이번 정책에만 쓰이는 일회성 상태·작업 번호·파일명을 필드명으로 만들고 있지 않은지
- 승인된 변경 범위와 미확정 부분

다른 Family에서 사용 가능하다는 이유만으로 즉시 공통화하지 않는다. 반대로 독립 Family라는 이유로 공통 구조의 영향을 생략하지 않는다. JSON Schema의 `additionalProperties: true` 또는 생략은 기술적 허용이며, 필드 추가에 대한 사용자 승인 근거가 아니다.

## 책임 분리 검토

문서 길이나 규칙 개수로 기계적으로 분리하지 않는다. 다음은 사용자 요청을 작업에 적용하기 위한 판단 질문이다.

- 작업 영역·책임 영역이 크게 다르고, 한쪽을 바꿔도 다른 쪽의 의미를 바꾸지 않고 검토·검증할 수 있는가?
- 결정 주체, 적용 조건, 생명주기, 산출물 또는 증거의 소유가 서로 다른가?
- 공동으로 유지해야 하는 규칙보다 서로 독립적인 규칙이 주가 되는가?
- 분리하면 의무나 예외가 누락되거나 순환 참조·중복 규칙이 생기는가?

분리가 명확하면 별도 지시를 기다리지 않고 책임별 문서 구성을 권장하고 그 근거를 먼저 제시한다. 경계가 모호하거나 추가 추론·전환 비용이 큰 경우에는 기계 정책을 작성하기 전에 사용자에게 합칠지 나눌지 질문한다. 사용자의 새 선언이 오기 전에 후보 분리를 최종 결정으로 만들지 않는다. 다른 Family의 최종 소유권, 신규 정책 ID, 승인·상태를 추정하지 않는다.

## 보조 명령의 범위

Go helper는 기존 `internal/contractentry.Resolve`와 `contract_authoring.ReadDocument`를 재사용한다. 별도 계약·스키마·등록된 validator나 실행 hook을 추가하지 않는다. 표준 출력만 쓰며 정책 파일을 수정하지 않는다.

- `--target`은 문서 전체와 기존 계약 선택 결과를 수집한다. 구 문서 선택 실패는 `selection_problem`으로 보존하며, 읽기 전용 검토 자료 수집은 계속한다.
- `--source FILE#/pointer`는 명시한 구간만 출력한다. `source_scope` 재귀 탐색, 의미 기반 자동 선정, 유사 이름 폴백은 하지 않는다. 잘못된 경로·구간은 실패하며 파일 전체로 확대하지 않는다.
- `field_findings`는 선택된 Family의 `Policy_Rules`에 한정하여 명시된 `properties/items/required`와 비교한다. 중첩 추가 필드도 표시한다. 공통 공개 구조 전체나 `Policy_Status`·`Authority`를 검사하지 않는다.
- 열린 객체, 복잡한 스키마 분기·객체 참조는 미검사로 표시한다. 타입·값·조건·허가·재사용성·책임 분리의 의미 판정은 기존 검증기와 사람의 검토 대상이다. 추가 필드 표시가 곧 스키마 위반이라는 뜻도 아니다.
- `--verify`는 패킷에 포착된 파일들의 바이트 해시가 같은지만 확인한다. 전체 계약 그래프의 최신성·완전성·진위·번역의 정확성은 보증하지 않는다. 새로운 작업에서는 기존 계약 진입도 다시 실행한다.
- 출력 상태는 `NON_NORMATIVE_REVIEW`, `schema_validation: NOT_PERFORMED`다. 종료 코드 0은 수집 성공이며, 선택 실패나 검토 항목이 없다는 뜻이 아니다. 잘못된 입력·누락 구간·변경된 파일은 종료 코드 2다. `go run`은 비정상 종료를 셸 코드 1과 `exit status 2`로 전달할 수 있다.

자동화는 반복해서 확인된 조회·추출·비교부터 추가한다. 새로운 의미 추정이나 복잡한 분류 체계를 미리 구현하지 않는다. 이 명령을 호출하는 것은 명시적인 작업 선택이며 자동 정책 개정 경로의 활성화가 아니다.

## 현재 RECORD 문서 검토

저장소 루트의 PowerShell에서 실행한다. `PTSIP_PYTHON`은 해당 프로세스에서 기존 YAML 직렬화/검증 런타임을 선택한다. 설치나 환경 파일 변경은 하지 않는다.

```powershell
$env:PTSIP_PYTHON = (Resolve-Path .venv/Scripts/python.exe).Path
go -C .ptsip/engine/developer run ./policy/review/cmd/policy-review --root ../../.. `
  --target developer/policy/RECORD/MPD-RECORD-0001.yaml `
  --target developer/policy/RECORD/MPD-RECORD-0002.yaml `
  --target developer/policy/RECORD/MPD-RECORD-0003.yaml `
  --target developer/policy/RECORD/MPD-RECORD-0004.yaml
```

출력을 파일로 저장하려면 PowerShell 7에서 `> <새 파일 경로>`를 명시한다. 기존 검토본을 덮어쓰지 않도록 새 경로를 사용한다. 이 도구는 출력 파일 생성을 직접 수행하지 않는다.

```powershell
go -C .ptsip/engine/developer run ./policy/review/cmd/policy-review --root ../../.. --verify .ptsip/engine/developer/policy/review/examples/RECORD.current.evidence.json
go -C .ptsip/engine/developer test ./policy/review/... ./internal/contractentry -count=1
```

개정 전 자연어 검토본은 `examples/RECORD-review.ko.md`, 당시 수집 근거는 `examples/RECORD.evidence.json`이다. 둘 다 시점이 고정된 비규범적 역사 자료다. `0001`은 새 선언으로 개정되었고 구 `0002` 후보의 내용은 삭제한 뒤 사용자의 명시적 지시에 따라 같은 번호를 정책 결정 기록에 재사용했다. 따라서 과거 근거의 `--verify`는 원문 변경을 보고하는 것이 정상이다.

새 선언과 후속 기능 범위는 `examples/RECORD-declaration-2026-11-01.ko.md`, 작성 후보 이력은 `examples/RECORD-proposal.json`에 보존했다. 현재 네 정책의 검토 패킷은 `examples/RECORD.current.evidence.json`이다. 원본 정책의 권위와 상태는 `developer/policy/RECORD/`의 각 문서에서 다시 확인한다. 개정 전의 다섯 원문 선정 결과는 역사적 번역에서만 재사용하고 새 정책에 자동 편입하지 않는다.
