# PTSIP 0.3.8 프로젝트 에이전트 분석 기록

2026년 11월 17일부터 23일까지 PTSIP 계약 중앙화와 배포 전환 과정에서 나온 분석을 정리했다. 에이전트 이름과 표시 이름은 당시 실행 기록으로 대조했다. 각 항목의 확인 결과는 해당 작업 시점의 증거다. 현재 구현 상태와 권한은 사용 시점의 저장소와 계획서에서 확인한다.

## Policy v4와 엔진 책임

### `policy_contract` (Socrates)

Policy v4의 Identity, Lifecycle, Rules, Content를 각각 연결하는 루트 스키마와 Go 사용자 계약 검증을 구현했다. 등록 상태 검증에서는 `ACTIVE`의 신규 채택, `LEGACY`의 기존 사용 유지, `DEPRECATED`의 업그레이드 알림을 실제 선택 API로 확인했다. 당시 전체 제품 Go 검증은 최상위 38개가 통과했으며 Windows 심볼릭 링크 사례 1개가 건너뛰어졌다. [에이전트](../../agents/policy_contract.toml) · [Policy 루트 계약](../../../src/ptsip/schemas/policy/public-structure-policy.schema.json) · [Profile 등록 테스트](../../../src/tests/ptsip/project_profile_schema_test.go)

### `rbac_contract` (Schrodinger)

RA/RAC/AEC의 구조와 목록 간 정확한 ID 참조를 구현했다. RA에서 조건 이름과 ID의 대응 및 목록 소속을 확인했다. Validation의 새 계약 검증 경로는 Go CLI의 전용 worker 바인딩을 사용하며, 바인딩이 없으면 구조화된 실패를 반환하는 것을 확인했다. 당시 Validation Rust 테스트 28개가 통과했다. [에이전트](../../agents/rbac_contract.toml) · [RA 스키마](../../../src/ptsip/schemas/storage/rbac/ra-list.schema.json) · [Validation 계약](../../../src/ptsip/engines/validation/contract.json)

### `review` (Kuhn)

독립 검토에서 RA 참조의 이름과 ID 배치 불일치, `prefixItems`를 통한 사용자 필드 바인딩 누락, 중첩 `$id` 충돌을 재현했다. 후속 테스트에서는 Go 라우터와 Rust 세션의 1 MiB 경계, 잘못된 JSON 이후 정상 요청 처리, 중복 JSON 키 거부를 확인했다. [에이전트](../../agents/review.toml) · [Profile 세션 테스트](../../../src/tests/ptsip/profile_session_limits_test.go)

### `profile_analysis` (Kepler)

이전 Profile 모듈의 식별자, 계약 선택, 전환, 카탈로그, 메타데이터, 규칙 레코드 소비 경로를 조사했다. 기술 테스트의 스키마 조회는 `engine_schema_refs`의 현재 등록 경로에서 해석하도록 전환했다. 등록이 없거나 비어 있으면 실패하도록 했으며, 엔진 데이터 계약의 권한 확대 거부 테스트가 통과했다. [에이전트](../../agents/profile_analysis.toml) · [엔진 인덱스](../../../src/ptsip/engines/index.json)

### `engine_analysis` (Rawls)

기존 엔진 스키마 21개를 실제 구현의 소유 책임으로 배정했다. Discovery 10개, Execution 6개, Validation 4개, Storage 1개다. 기존 ID와 Git blob을 유지한 이동을 검증했으며, Discovery가 등록 스키마의 SHA-256과 조회 경로의 무결성을 확인하도록 보완했다. 별도 Evidence 스키마 네 개에는 당시 확인 가능한 Rust 엔진 소유 근거가 없었다. [에이전트](../../agents/engine_analysis.toml) · [엔진 스키마 소유권 테스트](../../../src/tests/ptsip/engine_schema_ownership_test.go) · [Discovery Profile 구현](../../../src/ptsip/engines/discovery/src/profile.rs)

### `plan_review` (Galileo)

엔진 배치에 대한 승인과 새 native operation ID의 등록 결정을 구분했다. 승인된 `projection.profile.metadata`에서는 세 역사 Profile 버전의 출력 차이를 보존하고 한글, 따옴표, 줄바꿈과 잘못된 입력을 테스트했다. 당시 자산 스테이징은 통과했으며 해당 구현 커밋의 전체 wheel 빌드는 엔진 인벤토리 확인에서 실패했다. [에이전트](../../agents/plan_review.toml) · [메타데이터 투영](../../../src/ptsip/engines/projection/src/profile.rs) · [투영 테스트](../../../src/tests/ptsip/projection_profile_test.go) · [구현 커밋](https://github.com/Kinirin/PTSIP/commit/44146421c3ec30c187afb8df550a1f9a91087469)

## Profile과 Repository

### `pp300_design` (Pasteur)

공식 pp.3.00 스키마에 `DEFAULT` 모드와 필수 component 필드 다섯 개를 정의했다. `dependency`에는 pp.2.00의 `default`, `allow`, `deny`, `from`, `to` 정의를 재사용했다. 역사 스키마의 원본 식별자 보존, 정확한 버전 선택과 새 구조 검증을 포함한 당시 집중 테스트 75건이 통과했다. [에이전트](../../agents/pp300_design.toml) · [pp.3.00 스키마](../../../src/ptsip/schemas/storage/profile/project-profile.schema.json) · [pp.3.00 테스트](../../../src/tests/ptsip/project_profile_pp300_test.go)

### `repository_design` (Nietzsche)

당시 Python Repository 사용 문장 74개와 호출 파일 48개를 조사했으며, 정상 migration 경로에서 실제 호출자 24개를 확인했다. 등록된 엔진 6개를 비교해 Context에 관측 스냅샷, Discovery에 저장소와 후보 탐색, Validation에 경로·전환·수렴 판단을 배정했다. 등록 후보 일곱 개의 사유와 배정 근거는 계획서의 `repository_native_import_design`에 남겼다. [에이전트](../../agents/repository_design.toml) · [중앙화 계획서](../../../src/planning/0.3.8/ptsip-schema-centralization.plan.json) · [등록 엔진 목록](../../../src/ptsip/engines/index.json)

### `plan_scope` (Faraday)

공식 pp.3.00과 불변 이력, 사용자 확장 계약의 개별 결정, Evidence의 기존 소비 책임을 계획에 기록했다. Repository 분석을 WU-07·13·14의 실행 조건에 연결했다. 계획서 JSON과 변경분 검사가 통과했다. 당시 미등록으로 기록했던 일곱 operation의 후속 승인은 계획서의 현재 결정을 확인한다. [에이전트](../../agents/plan_scope.toml) · [중앙화 계획서](../../../src/planning/0.3.8/ptsip-schema-centralization.plan.json)

### `projection_transfer` (Boole)

공식 스키마의 버전, 경로, 수명주기, SHA-256을 중앙 설정에 두고, 지원 기능 조회는 Discovery, 전환 조회와 검증은 Validation으로 옮겼다. 당시 Go 최상위 테스트 60개, Rust 테스트 54개와 소스 아카이브 자산 대조가 통과했다. 해당 시점의 설치 검증은 wheel 빌드 실패로 보류됐다. [에이전트](../../agents/projection_transfer.toml) · [중앙 설정](../../../src/ptsip/schemas/config.json) · [Discovery 버전별 지원](../../../src/ptsip/engines/discovery/profile-support.json) · [Validation 전환](../../../src/ptsip/engines/validation/profile-transitions.json) · [구현 커밋](https://github.com/Kinirin/PTSIP/commit/b9ffe4caf40e138d2e9d43c7833b484e6d957d13)

### `execution_transfer` (Pauli)

읽기 전용 `validation.profile.identity-rewrite`가 원본 바이트, 등록된 identity 전환, 허용된 필드차이, 대상 스키마의 ID와 다이제스트를 검사하도록 했다. 변조된 바이트, 등록되지 않은 전환, 관계없는 필드 변경이 거부됨을 테스트했다. 당시 실제 변경을 실행할 호출자에는 권한 소비에 필요한 행위자·요청·아티팩트 증거를 공급하는 신뢰된 생산자가 없었다. [에이전트](../../agents/execution_transfer.toml) · [검증 구현](../../../src/ptsip/engines/validation/src/profile_identity_rewrite.rs) · [회귀 테스트](../../../src/tests/ptsip/profile_identity_rewrite_verification_test.go)

### `pp_semantic_migration` (Mill)

`projection.migration.pp200-to-pp300`은 원본 선언과 참조 ID를 보존하고 역사 기본값을 각 component에 복사한다. `allow`와 `deny` 관계는 `from`으로 선언된 component에 분배한다. 명시형·템플릿·혼합형을 포함한 집중 하위 테스트 11개가 통과했다. 충돌하는 관계, 매핑이 없는 필드, 손상된 스키마 다이제스트는 후보를 생성하기 전에 거부됐다. [에이전트](../../agents/pp_semantic_migration.toml) · [전환 투영](../../../src/ptsip/engines/projection/src/profile_conversion.rs) · [전환 테스트](../../../src/tests/ptsip/projection_migration_test.go)

## 배포, 버전 및 소비자 전환

### `go_distribution` (Tesla)

불변 Git 커밋의 파일과 엔진·Profile 등록 다이제스트로 재현 가능한 제품 소스 아카이브를 구성했다. 경로와 제외 규칙, 기존 출력 덮어쓰기 방지, stale 엔진 다이제스트 거부를 확인했다. 당시 테스트 증거는 소스 아카이브 단계까지 확보됐다. [에이전트](../../agents/go_distribution.toml) · [소스 아카이브 구현](../../../src/build/source.go) · [배포 테스트](../../../src/tests/build/go_product_distribution_test.go)

### `rbac_package_cutover` (Volta)

배포 검증이 등록된 RA/RAC/AEC와 설치본의 Rust Discovery Profile 계약을 사용하도록 전환했다. 설치 smoke 검증은 매니페스트, 계약, 인덱스의 식별자와 버전이 일치하는지 확인하며, 제거 대상 Python 경로도 검사한다. 관련 Go·Discovery 테스트와 smoke 스크립트 문법 검사가 통과했다. 작업 당시 실제 wheel 설치는 실행되지 않았다. [에이전트](../../agents/rbac_package_cutover.toml) · [RBAC 패키징 테스트](../../../src/tests/build/rbac_package_test.go) · [설치 검증기](../../../src/tests/build/installation.go)

### `release_build_verifier` (Ramanujan)

릴리스 계약이 정의하는 검증을 `src/tests/build`에 구현했다. 정확한 Git 소스 바인딩, 등록 Profile·RBAC, 아카이브와 wheel RECORD, 네이티브 원본, 격리 설치가 대상이다. 제품 소스 아카이브의 재현성, 개발 후보 제외, 변조된 wheel과 누락된 Go 라우터 거부 테스트가 통과했다. [에이전트](../../agents/release_build_verifier.toml) · [릴리스 계약 등록](../../../.ptsip/contracts/release/index.json) · [빌드 검증 테스트](../../../src/tests/build/verification_test.go)

### `policy_shape` (James)

두 정책 소스에서 중복 YAML 키 네 쌍을 확인했다. 세 쌍에는 제안과 후속 결정이 함께 있었으며, 나머지 한 쌍은 기존 엔진 범위와 소비자 전환 범위가 중첩됐다. 하나의 구조 안에서 시점과 범위를 명시할 수 있음을 확인했다. 이 분석에서는 정책 소스를 변경하지 않았다. [에이전트](../../agents/policy_shape.toml) · [정책 계약 가드](../../../.github/workflows/policy-contract-guard.yml)

### `approval_trace` (Carson)

퇴역한 승인 기록을 직접 읽는 테스트가 현행 Validation·Projection 버전 확인에 사용할 수 없는 원인을 추적했다. 삭제된 Python 래퍼에 연결된 테스트를 교체해 등록된 Projection에서 원본 JSON 포인터와 component ID를 유지하는지, Validation에서 오래된 스냅샷을 거부하는지 검증했다. 당시 전체 제품 Go 테스트와 `go vet`가 통과했다. [에이전트](../../agents/approval_trace.toml) · [중앙화 계획서](../../../src/planning/0.3.8/ptsip-schema-centralization.plan.json) · [Projection 테스트](../../../src/tests/ptsip/projection_migration_test.go)

### `release_go_sum` (Noether)

Go는 종속 모듈의 `replace`를 상속하지 않으므로 각 진입 모듈이 저장소의 `common_contract`를 독립적으로 연결해야 함을 확인했다. 네 진입 모듈의 읽기 전용 컴파일로 원인을 재현하고 검증했다. 이후 릴리스 컨트롤러와 릴리스 테스트 전체가 읽기 전용 모드에서 통과했다. [에이전트](../../agents/release_go_sum.toml) · [컨트롤러 모듈](../../../.ptsip/engine/developer/Distribution/release/controller/go.mod) · [릴리스 인수 테스트](../../../.ptsip/engine/developer/Distribution/release/test/admission_test.go)

### `version_gate` (Herschel)

버전 원본에서 파생 투영, 독립 스냅샷, 정확한 clean SHA의 wheel로 이어지는 순서를 확인했다. 소스 인벤토리 결과와 배포 검증 결과를 구분했다. 후속 Pilot 분석에서는 등록 엔진별 입력·출력을 비교해 관측된 의존성을 pp.3.00 경계 위반으로 판정할 계약이 당시 없음을 확인했다. [에이전트](../../agents/version_gate.toml) · [버전 변경 계획서](../../../src/planning/0.3.8/ptsip-version-change.plan.json) · [Pilot 구현](../../../src/ptsip/pilot/runner.go)

### `specdata_native` (Pascal)

pp.1.01·pp.1.02와 선택되지 않은 pp.2.00의 중복 사본을 제거하는 과정에서 선택된 이력 원본과 Git blob을 확인했다. 당시 이력 fixture와 Go 자산 스테이징 검증이 통과했다. 후속 작업에서는 JavaScript·TypeScript 가져오기와 npm 선언을 Go로 관측하고 계산된 가져오기와 모호한 상대 경로를 미확정으로 남겼다. [에이전트](../../agents/specdata_native.toml) · [이력 스키마 테스트](../../../src/tests/ptsip/project_profile_schema_test.go) · [JavaScript 의존성 관측](../../../src/ptsip/inspection/dependency_javascript.go)

### `go_retirement` (Maxwell)

등록된 `discovery.repository.inspect` 결과에서 파일 인벤토리를 구성하고 관측 오류를 보존했다. 이후 순수 Go로 Python의 정적 import, 상대 import, 제한된 리터럴 동적 import를 관측했다. 증거로 경로나 값을 단정할 수 없는 경우에는 미해결로 반환한다. 실제 Discovery 라우팅과 의존성 집중 테스트가 통과했다. [에이전트](../../agents/go_retirement.toml) · [Go 인벤토리](../../../src/ptsip/inspection/inventory.go) · [Python 의존성 관측 테스트](../../../src/tests/ptsip/dependency_python_test.go)

## 2026년 11월 23일 후속 실행

- `compose` (Dirac)는 `release_build_verifier` 업무에 이어 릴리스 소비자가 새 버전 원본과 구성 계약을 읽는 경로를 구현했다. 등록과 소비자 전환은 작업 중 검토했다. wheel 구성 증빙과 설치 결과는 별도의 검증으로 남았다. [릴리스 검증 에이전트](../../agents/release_build_verifier.toml) · [릴리스 구성 소비자](../../../.ptsip/engine/developer/Distribution/release/controller/composition.go)
- `snapshot` (Meitner)는 `version_gate` 업무에 이어 스냅샷과 의존성 그래프의 독립 생성, 정확한 최종 커밋 검증과 wheel 선행 검사를 구현했다. 빌드 모듈 Go 컴파일, 워크플로 YAML과 Bash 문법 검사를 확인했다. 원격 실행 상태는 해당 커밋의 GitHub Actions에서 확인한다. [버전 검증 에이전트](../../agents/version_gate.toml) · [빌드 진입점](../../../src/build/main.go)
- `central` (Mencius)는 `pp300_design` 및 `plan_scope` 업무에 이어 pp.2.00 후보를 지정된 이력 위치로 이동했다. 원본 Git blob `e9fba283d16d521a037658899cf7066c5769a2eb`, 기존 `$id`, 미등록 초안 상태를 대조했다. Profile 집중 테스트 7개와 공통 RBAC 구조 적합성 테스트가 통과했다. [Profile 계약 에이전트](../../agents/pp300_design.toml) · [계획 범위 에이전트](../../agents/plan_scope.toml) · [Profile 이력 테스트](../../../src/tests/ptsip/project_profile_schema_test.go)
