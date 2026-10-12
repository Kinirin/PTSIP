<!-- LOCALIZED PROJECTION: README.md is canonical. This dev-branch translation is synchronized to the English source; the main-branch Argos workflow may regenerate it. -->
<p align="right">
  <a href="../../README.md">English</a> | 한국어
</p>

# PTSIP — Primary Lifecycle Ownership and Responsibility Isolation Policy

> 이 문서는 정본인 [`README.md`](../../README.md)의 한국어 번역본입니다. 번역과 영문 원문의 의미가 다른 경우 영문 원문을 우선합니다. 정책 권위는 기계 판독 정책 계약에 있으며 이 번역 문서가 정책을 새로 정의하지 않습니다.

**PTSIP는 아키텍처 책임을 명시적이고 기계가 읽을 수 있는 프로젝트 의도로 기록한 다음, 저장소에서 관측한 증거를 통해 검증합니다.** 특정 디렉터리 구조나 프로그래밍 언어를 강제하지 않으면서 Product, 개발, 전달, 운영 생명주기의 책임을 분리합니다.

> **재사용보다 목적이 우선합니다.** 코드 공유를 최적화하기 전에 책임이 왜 존재하는지, 어느 생명주기가 그 책임을 소유하는지 먼저 판단합니다.

## 현재 버전 및 배포 상태

| 식별 항목 | 현재 소스 트리의 값 |
| --- | --- |
| Tool | `0.3.8a5` — **아직 배포되지 않은 프리릴리스 후보** |
| 마지막으로 배포된 Tool 프리릴리스 | `0.3.8a4` |
| Project Profile | `pp.1.02` |
| Specification 패밀리 | `0.3.7-draft` |
| 바인딩된 불변 `SPEC_REVISION` | `3c47816770d194ae42f98faedc911d980db0e62a` |

Tool, Project Profile, Specification의 식별자는 서로 독립적입니다. 정확한 배포 게이트는 [`0.3.8a5` 변경 사항](../../docs/releasenote/tool/0.3.8a5.md), [Tool 릴리스 이력](../../docs/releasenote/tool/README.md), [릴리스 워크플로](../../.github/workflows/tooling-release.yml)를 참고하세요.

## 빠른 시작

Python **3.11 이상**이 필요합니다. 아래 명령은 PowerShell 기준이지만 기본 CLI 명령은 다른 셸에서도 사용할 수 있습니다.

일반적인 버전 선택 규칙에 따라 공개된 최신 릴리스를 설치하거나 업그레이드하고 식별자를 확인합니다.

```powershell
python -m pip install --upgrade PTSIP
ptsip --version
ptsip spec
```

공개된 프리릴리스 `0.3.8a4`를 명시적으로 설치하려면 다음 명령을 사용합니다. 일반적인 pip 버전 선택에서는 프리릴리스보다 최신 안정 릴리스가 선택될 수 있습니다.

```powershell
python -m pip install "PTSIP==0.3.8a4"
```

이 저장소의 소스로 직접 개발하려면 다음 명령을 사용합니다.

```powershell
git clone https://github.com/Kinirin/PTSIP.git
cd PTSIP
python -m pip install -e ".[dev]"
```

Project Profile을 변경하기 전에 읽기 전용 또는 진단 작업으로 현재 상태를 확인합니다.

```powershell
ptsip doctor .
ptsip inspect .
ptsip pilot .
ptsip validate .
ptsip conform .
```

`validate`는 선언을 검증하고, `conform`은 관측 증거를 토대로 적용 가능한 아키텍처 요구사항을 평가합니다. 유효한 프로파일이라는 사실만으로 적합성이 증명되지는 않습니다. 정확한 옵션은 설치된 CLI의 도움말을 확인하세요.

## 시스템이 구분하는 책임

| 권위/증거 계층 | 담당 책임 |
| --- | --- |
| Specification | 규범적 요구사항과 해당 불변 리비전 |
| Decision Authority | 명시적으로 승인된 아키텍처 결정과 그 조정 |
| Source Project Profile / Responsibility Map | 프로젝트가 소유하는 아키텍처 선언 |
| 관측 및 적합성 평가 | 실제 저장소·산출물 동작과 규칙 평가 |

**증거는 아키텍처 권위가 아닙니다.** 파일 경로, 패키지, 휴리스틱, 템플릿, AI 신뢰도 및 관측된 import는 검토에 사용할 수 있지만 어느 것도 생명주기 소유권을 암묵적으로 생성할 수 없습니다. Tool은 `INCOMPLETE`를 통과로 취급하지 않습니다.

## 주요 생명주기 소유권

Tool `0.3.8a5`의 정규 분류는 정확히 다음 다섯 가지입니다.

| 분류 | 의미 |
| --- | --- |
| `PRODUCT` | Product 생명주기가 주로 소유하는 책임입니다. |
| `DEVELOPMENT_TOOLING` | 개발 과정에서 생성, 검사, 검증, 변환, 코드 생성, 마이그레이션, 분석, 테스트 등을 지원하는 개발 생명주기 책임입니다. |
| `DELIVERY` | 릴리스 준비, 패키징, 공개, 승격, 배포 또는 전달 대상에 배치하는 책임입니다. |
| `OPERATIONS` | 전달 이후 상태 확인, 복구, 조정, 유지보수 및 지속적 운영을 담당하는 책임입니다. |
| `NEUTRAL_CONTRACT` | 실행하지 않고 특정 생명주기의 소유권도 갖지 않는, 생명주기 독립적인 계약 책임입니다. |

`UNKNOWN`, `CONFLICT`, `INCOMPLETE`, `PENDING`, 신뢰도 값 및 마이그레이션 상태는 워크플로·평가 상태이며 추가적인 아키텍처 분류가 아닙니다.

### Tool 0.3.5 호환성 경계

Tool `0.3.5`에서는 과거에 다음 분류를 사용했습니다.

```text
PRODUCT
TOOLCHAIN
NEUTRAL_CONTRACT
```

Tool `0.3.8a5`는 Tool `0.3.6`에서 확립한 다섯 가지 분류 모델을 유지합니다. 따라서 `TOOLCHAIN`은 **Tool `0.3.5`의 과거 입력값**이며 현재의 정규 별칭이 아닙니다. 과거 Toolchain 책임은 실제 생명주기 소유권에 따라 `DEVELOPMENT_TOOLING`, `DELIVERY`, `OPERATIONS`으로 바뀌거나 분리가 필요할 수 있습니다. `TOOLCHAIN -> DEVELOPMENT_TOOLING`을 일괄 치환하는 것은 금지됩니다.

Tool `0.3.8a5`는 명시적으로 지원하는 과거 소스에 대해 증거에 결합된 현재 대상 직접 마이그레이션을 제공합니다. 마이그레이션 기능은 저장소 도입 권위와 별개이며, 추론 결과를 프로젝트 의도로 바꾸지 않습니다.

## 분류는 경로나 기술로 결정되지 않습니다

분류의 기준은 파일명, 디렉터리, 언어, 프레임워크, 실행 가능 여부, 워크플로 제공자, 컴파일 방식, 실행 시간, 테스트 상태, 신뢰도 점수가 아니라 **해당 책임을 지배하는 생명주기 의무**입니다.

예시:

```text
Product-specific verification responsibility       -> PRODUCT
Reusable verification framework / test SDK         -> DEVELOPMENT_TOOLING
Product runtime implementation                      -> PRODUCT
Release-unit assembly / publication automation      -> DELIVERY
Post-deployment health or recovery automation       -> OPERATIONS
Independent non-executable shared contract          -> NEUTRAL_CONTRACT
```

`tests/`, `tools/`, `deploy/`, `ops/`, `.github/workflows/` 등의 경로는 증거를 이해하기 위한 맥락이지 아키텍처 권위가 아닙니다.

## Responsibility Map v2

Tool `0.3.8a5`에서는 Responsibility Map v2를 프로젝트 소유 아키텍처 선언 모델로 사용합니다. 다음 차원은 서로 독립적으로 유지됩니다.

```text
classification
    = primary lifecycle ownership

roles
    = coarse responsibility characteristics

relationships
    = project-owned typed directed semantics

source/derived provenance
    = where declaration/materialized architecture came from

VPMS Verification Purpose
    = why verification exists and what it protects
```

정규 역할(roles)은 다음과 같습니다.

```text
IMPLEMENTATION
VERIFICATION
AUTOMATION
CONFIGURATION
DOCUMENTATION
GOVERNANCE
```

정규 유형 관계(typed relationships)는 다음과 같습니다.

```text
IMPORTS
LINKS
LOADS
INVOKES
READS
GENERATES
BUILDS
PACKAGES
PUBLISHES
DEPLOYS
VERIFIES
MANAGES
DOCUMENTS
SPECIFIES
GOVERNS
```

연관 산출물(associated artifact)은 하나의 분류된 기준 컴포넌트에 종속되는 프로젝트 소유의 비컴포넌트 지원 영역입니다. 이를 독립적으로 관리해야 할 실행 책임이나 생명주기 책임을 숨기는 데 사용해서는 안 됩니다.

## 명시적·템플릿·하이브리드 선언

정규 소스 모드는 다음과 같습니다.

```text
explicit
    repository directly declares the complete map

template
    repository explicitly selects one immutable revision-bound template

hybrid
    repository explicitly selects a template and adds project-owned
    overrides, extensions, or removals
```

초기 템플릿 카탈로그에는 다음 항목이 포함됩니다.

```text
python-package-library
python-cli-application
mixed-product-development-delivery
```

템플릿은 명시적으로 선택해야 합니다. PTSIP는 저장소 구조, 언어, 프레임워크 탐지, 매니페스트 또는 신뢰도를 근거로 템플릿을 자동 선택하지 않습니다.

## 소스 선언과 Effective Responsibility Map

모든 소스 모드는 권위가 없는 결정적 구체화(deterministic materialization)를 통해 해석됩니다.

```text
Source Project Profile
        |
        v
explicit / template / hybrid
        |
        v
deterministic materialization
        |
        v
Canonical Effective Responsibility Map
        |
        +--> validation / conformance
        +--> clarification / adoption
        +--> narrow VPMS read-only projection
```

Source Project Profile은 계속 프로젝트 소유의 아키텍처 권위로 남습니다. 구체화 과정은 템플릿을 선택하거나, 소유권을 추론하거나, 잘못된 아키텍처를 복구하거나, 원본 선언을 암묵적으로 다시 작성할 수 없습니다.

해석된 결과에는 다음과 같은 선언 출처 정보가 보존됩니다.

```text
PROJECT_EXPLICIT
TEMPLATE
PROJECT_OVERRIDE
PROJECT_EXTENSION
PROJECT_REMOVAL
```

재현성을 위해 결정적 다이제스트 식별자도 사용합니다. 그러나 출처 정보와 다이제스트 어느 것도 아키텍처 권위를 대신하지는 않습니다.

## 프로파일 선택과 도입

저장소 로컬 PTSIP 설정을 영속화하는 경우 [`.ptsip/index.json`](../../.ptsip/index.json)은 정규 `.ptsip/` 네임스페이스를 라우팅하고, [`.ptsip/profiles/index.json`](../../.ptsip/profiles/index.json)은 `default_profile`을 통해 활성 `*.ptsip.yaml` Project Profile을 선택합니다. 저장소 루트의 `ptsip.yaml`은 과거 마이그레이션·호환성 입력이며 두 번째 활성 선언 권위가 아닙니다.

`explicit`, `template`, `hybrid` 선언은 모두 Canonical Effective Responsibility Map으로 구체화됩니다. 구체화기는 등록된 구조와 출처를 해석할 수 있지만 **템플릿 자동 선택, 컴포넌트 소유자 창작, 상충하는 프로젝트 의도의 임의 수정은 금지**됩니다.

기존 컴포넌트의 선언 변경은 적용하기 전에 다음 명령으로 검토합니다.

```powershell
ptsip adopt . `
  --component tools `
  --classification DEVELOPMENT_TOOLING `
  --purpose "Repository-local generation tooling" `
  --shipped no `
  --runtime-required no `
  --executable yes `
  --json
```

프로젝트 소유자가 변경을 검토하고 승인한 뒤에만 `--apply`를 추가합니다. 위 예시는 모든 `tools` 디렉터리를 같은 분류로 처리하라는 규칙이 아닙니다. 현재 정규 소유권 필드는 `classification`이며 과거 `lifecycle_owner`는 마이그레이션 증거로만 사용합니다.

선택적 GitHub 조정 Decision Authority는 **해당 모드가 설정된 경우** `refs/heads/ptsip-policy`를 사용합니다. 권위의 최신 상태를 조회하고 조건부 변경을 적용하여 오래된 작성자의 변경을 거부합니다. 조정에 실패하면 다른 로컬 Decision Store로 암묵적으로 전환하지 않고 새 아키텍처 권위 쓰기를 차단합니다. GitHub는 모든 Specification에 필수인 의존성이 아닙니다.

## 네이티브 기반과 Consumer 경계

독립적으로 빌드되는 **일곱 개의 Rust 엔진**은 버전이 부여된 언어 중립 계약과 등록된 진입점 바인딩을 [`src/ptsip/engines/`](../../src/ptsip/engines/) 아래에 보유합니다. [공식 엔진 인덱스](../../src/ptsip/engines/index.json)를 참고하세요.

| 엔진 | 담당 기능 |
| --- | --- |
| Context | 식별자와 원본에 결합된 검증된 Context 스냅샷 로드 |
| Storage | 제공자 소유권을 가져오지 않고 제공자 연산 중개 |
| Validation | 스키마, 계약, 타입 지정 입출력, 사전조건, Case 검증 |
| Discovery | 등록된 소스 범위와 호출 바인딩 해석 |
| Planning | 의존성 순서 결정, 변경 영향 평가, 네이티브 호출 준비 |
| Execution | 호출자가 허용한 요청을 제한된 조건에서 실행하고 효과·중지 결과 기록 |
| Projection | 검증된 Context를 선언된 출력 형식으로 렌더링 |

`0.3.8a5`는 Linux, Windows, macOS에서 네이티브 기반을 검증하지만 이것이 Consumer의 프로덕션 설치, 권위 승인, 활성화, 버전별 전환을 암묵적으로 허용하지는 않습니다. Python wheel에는 중립적인 엔진 계약 데이터가 포함되며 **Rust 소스, 네이티브 바이너리, 빌드 산출물은 포함되지 않습니다.** 기존 Consumer API는 계속 사용할 수 있습니다. 자세한 범위는 [릴리스 경계](../../docs/releasenote/tool/0.3.8a5.md)를 확인하세요.

PTSIP는 **어느 생명주기가 책임을 소유하는가**를 다룹니다. 별도 [`src/vpms/`](../../src/vpms/)에 있는 Verification Protocol Management System(VPMS)은 **명시적 Verification Case를 어떻게 등록·선택·실행·보고하는가**를 다룹니다. VPMS의 목적 및 검증 상태는 PTSIP 분류·적합성과 분리됩니다.

VPMS 공개 조합 API는 호출자가 명시적으로 제공하는 정의, 참조 및 어댑터와 함께 `load_registry_snapshot`, `resolve_selection`, `run_cases`를 사용합니다. 선택적 PTSIP 브리지는 검증된 불변 Effective Responsibility Map 데이터만 소비하며 새로운 프로젝트 권위를 만들지 않습니다. VPMS의 PASS는 PTSIP의 `CONFORMANT`를 의미하지 않으며, 그 반대도 마찬가지입니다.

PTSIP의 적합성 평가 결과는 `CONFORMANT`(종료 코드 `0`), `NON_CONFORMANT`(`5`), `INCOMPLETE`(`6`)입니다. 필수 증거가 없으면 실패 폐쇄 원칙을 적용합니다. Product 배포판 검증은 실제 스냅샷에 결합된 빌드 산출물을 검사합니다. 빌드 단계를 담당하는 컴포넌트가 그 결과물의 Product 소유자라는 뜻은 아닙니다.

## 개발 워크플로와 저장소 책임 지도

| 경로 | 책임 |
| --- | --- |
| [`.ptsip/`](../../.ptsip/) | Consumer 저장소 인덱스, 프로파일, Context 관리. Task/runtime은 예약 상태 |
| [`src/policy/index.yaml`](../../src/policy/index.yaml) | 현재 Support Feature 정책 레지스트리(`ACTIVE`/`DRAFT`/`RETIRED`) |
| [`developer/policy/`](../../developer/policy/) | 개발자용 기계 판독 정책과 바인딩 |
| [`developer/planning/`](../../developer/planning/) 및 [`planning/`](../../planning/) | 정책에서 도출된 실행 계획, 작업 단위, 과거 기록 |
| [`developer/automation/`](../../developer/automation/) | Go 기반 개발 도구와 정책·계획·릴리스 제어 |
| [`src/ptsip/`](../../src/ptsip/) | Python Consumer Tool, 적합성 평가, 프로파일 구체화, 엔진 계약 |
| [`src/ptsip/engines/`](../../src/ptsip/engines/) | 독립적으로 소유되는 일곱 네이티브 엔진의 구현과 계약 |
| [`src/vpms/`](../../src/vpms/) | 독립 검증 서브시스템 |
| [`src/tests/`](../../src/tests/) 및 [`developer/tests/`](../../developer/tests/) | Consumer, 개발자, 네이티브 검증 |
| [`registry/`](../../registry/) 및 [`schemas/`](../../schemas/) | Specification 레지스트리와 스키마 |
| [`.github/workflows/`](../../.github/workflows/) | 테스트, 릴리스, README 번역 워크플로 |
| [`docs/releasenote/`](../../docs/releasenote/) | Tool, Project Profile, Specification 릴리스 기록 |

코딩 에이전트용 [`AGENTS.md`](../../AGENTS.md)는 **정책 권위의 대체물이 아니라** 실행 지침을 제공하는 진입 프로젝션입니다. 정책이 계획을 지배하고 계획이 구현을 지배하므로 하위 계층이 상위의 권위 있는 결정을 암묵적으로 변경할 수 없습니다. [CONTRIBUTING.md](../../CONTRIBUTING.md)와 [개발 자동화 안내](../../developer/automation/README.md)를 참고하세요.

범위를 제한한 의존성 분석과 검증 계획은 참고용이며 현재의 아키텍처 선언이나 정확한 소스 기준 릴리스 검증을 대신하지 않습니다.

```powershell
ptsip dependency analyze .
ptsip dependency review-pack . --max-items 8 --max-context-bytes 12000
ptsip dependency validation-plan . --changed src/ptsip/dependency_analysis.py --json
```

[`.github/workflows/tooling-release.yml`](../../.github/workflows/tooling-release.yml)의 PREPARE와 PUBLISH는 각각 정확한 소스에 대한 검증을 요구하는 별개의 단계입니다. 문서, 테스트, 워크플로 또는 Tool 구현만 변경했다고 해서 불변 Specification 리비전이 자동으로 바뀌지 않습니다.

## 릴리스 노트 및 참고 자료

- [현재 `0.3.8a5` 릴리스 후보](../../docs/releasenote/tool/0.3.8a5.md): 현재 정책 검증, 등록된 Check 범위, 일곱 엔진의 기반. **배포 대기 중**.
- [공개된 `0.3.8a4` 기준선](../../docs/releasenote/tool/0.3.8a4.md): Go 기반 개발 자동화, Project Profile, 릴리스 검증.
- [과거 Tool 릴리스 노트](../../docs/releasenote/tool/README.md): 이전 릴리스와 네임스페이스 및 Context Plane 마이그레이션.
- [Specification/Project Profile/Tool 릴리스 기록](../../docs/releasenote/README.md): 각각 독립적으로 버전이 관리되는 생명주기 기록.

## 라이선스 권위

PTSIP의 라이선스 권위는 [`License-Authority/`](../../License-Authority/)에만 존재합니다. 기계 판독 진입점은 [`License-Authority/license-authority.yaml`](../../License-Authority/license-authority.yaml), 법적 효력이 있는 본문은 [`License-Authority/LICENSE.md`](../../License-Authority/LICENSE.md)입니다.

License Authority에 명시된 현재 효력 발생 시점은 UTC 기준 `2026-09-15T00:00:00Z`입니다.

이 저장소의 다른 위치에 있는 라이선스 관련 문구는 정보 제공 또는 맥락 설명을 위한 것일 뿐입니다. 그러한 문구는 라이선스의 일부가 되거나 라이선스를 변경하거나 참조를 통해 자동 편입되지 않습니다.

일반 Tool 릴리스, Specification 검증, Tool 버전 변경, 문서의 언급, 키워드 일치, AI의 의미 추정은 License Authority 진입을 승인하지 않습니다. 진입은 `License-Authority/license-authority.yaml`에서 선언한 트리거로만 허용됩니다.

## 언어 및 프로젝트 상태

영문 README가 프로젝트 개요의 정본입니다. [번역 문서](../../docs/translated/)는 등록된 Argos Translate 워크플로를 사용해 `main`에서 다시 생성됩니다. 번역은 영문 개요나 기계 판독 정책보다 우선하지 않습니다. PTSIP는 현재 실험적 프로젝트이며 릴리스 관련 주장은 정확한 소스 및 검증 증거에 결합됩니다.
