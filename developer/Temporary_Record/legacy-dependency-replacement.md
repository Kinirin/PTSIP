# 작업 기록 — 이전 정책 의존성 차단과 대체

기록 시작: 2026-10-17  
작업 상태: 개혁 진행 중, 대체 범위 전체 확인 전

## 목적과 책임

이전 정책 및 계약을 신규 설계의 암묵적인 권위로 사용하지 않도록 구분한다. 이 작업은 이전 파일을 즉시 삭제하거나 예전 정책의 모든 의미를 부정하는 작업이 아니다. **사용자가 승인한 의미만 새 계약 형태로 재정의하고, 예전 스키마·호출법·구조의 의존성을 이어받지 않는 것**이 목표이다.

## 과거 자료로 다룰 대상

- `developer/policy/approvals/`, `analysis/`, `contracts/`, `policy-resolver-bindings/`, `legacy/`, `registries/`, `schemas/`
- `developer/policy/MPD-BOUND-0001.yaml`
- `developer/policy/policy-resolver-contract.yaml`
- `developer/policy/source-application-review.yaml`
- `developer/state/`, `developer/verification/`, `developer/bindings/`
- `developer/contracts/lifecycle/policy_status.json`
- `developer/contracts/lifecycle/policy_version.json`

위 경로 아래의 이전 자료에는 새 정책을 작성하지 않는다. 사용자가 요청하지 않은 삭제·덮어쓰기는 하지 않는다. 읽고 참고하는 것은 가능하나, 기존 규칙을 새 계약의 기본값으로 그대로 적용하지 않는다.

## 확인된 사용자 결정과 간단한 이유

- 이전 정책이 유용할 수 있으므로 완전히 무시하지 않는다. 다만 사용자가 검토한 의미만 새로운 형태로 채택한다.
- 이전 Resolver·Registry·스키마가 존재한다는 이유로 신규 정책의 권위나 검증 성공을 상속하지 않는다.
- 참조 대상이 많아 정확한 의존성 목록과 실제 존재 여부는 아직 모두 확정되지 않았다.
- 정식 정책의 구현·대체 여부를 사용자가 결정한다. 임시 규칙별 부분 만료는 허용하지 않으며, 미완료 작업을 먼저 해결한 뒤 임시 폴더 전체를 최종적으로 정리한다.

## 마지막 확인된 작업 지점

폐기 예정 범위와 참조 원칙은 사용자 설명으로 확인했다. 과거 참조가 신규 구조에서 모두 차단되었다거나, 대체 대상이 빠짐없이 연결되었다는 기계 검증 결과는 없다.

## 남은 작업

- 신규 계약 검토에 필요한 이전 의미만 제한적으로 비교한다.
- 과거 규칙의 새 구조 반영은 사용자 승인 후 수행한다.
- 이전 참조·호출이 필요한 곳에서는 자동 승계를 피하고 미결정 사항을 사용자에게 알린다.
- 전체 대체를 방해하는 남은 참조를 개혁 작업의 우선순위로 다룬다.

이 문서는 새로운 의존성을 생성하는 Registry나 대체 자동화가 아니다. 사용자가 정식 정책으로 충분히 대체됐다고 결정하고 삭제를 요청하면 `developer/Temporary_Record/` 전체를 함께 삭제한다.
