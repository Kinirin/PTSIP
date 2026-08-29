<!-- AUTO-GENERATED: README.md is canonical. Unchanged reviewed translations are preserved; changed source blocks are translated by self-hosted Argos Translate. Do not edit directly. -->
<p align="right">
  <a href="README.md">English</a> | 한국어
</p>

# PTSIP — Primary Lifecycle Ownership and Responsibility Isolation Policy

> 이 문서는 정식 원본인 [`README.md`](README.md)를 기준으로 유지되는 한국어 번역본입니다. 변경되지 않은 번역은 보존하고, 변경된 원문 블록만 self-hosted Argos Translate로 갱신합니다. 프로젝트 사실이나 의미가 상충할 경우 영문 원본을 기준으로 합니다.

**상태:** Tool `0.3.7` WU-12 구현 통합 — exact-SHA 검증 대기<br>
**Tool/패키지 버전:** `0.3.7`<br>
**Project Profile 계약:** `pp.1.01`<br>
**Specification 패밀리:** `0.3.7-draft`<br>
**바인딩된 불변 Specification 리비전:** `3c47816770d194ae42f98faedc911d980db0e62a`<br>
**인증:** Apache 라이센스 2.0<br>
**현지화 문서:** `README.md`가 정식 원본입니다. 현지화된 README 파일은 `main`에서 self-hosted Argos Translate 워크플로로 다시 생성됩니다. 번역과 이 파일의 내용이 충돌하면 이 파일을 기준으로 합니다.

PTSIP는 프로젝트 책임을 **주요 생명주기 소유권(primary lifecycle ownership)**에 따라 분리하면서 명시적 아키텍처 의도, 생명주기 격리, 재현 가능한 적합성, 검증 목적 분리, 여러 환경 간 의사결정 일관성을 보존하기 위한 프로젝트 정의 아키텍처 정책입니다.

> **재사용보다 목적이 우선합니다.** 코드 공유를 최적화하기 전에 하나의 일관된 책임이 왜 존재하고 어느 생명주기가 그것을 소유하는지를 먼저 판단합니다.

Tool `0.3.7`은 현재 `main`의 릴리스 후보에 WU-12 구현과 동결된 Specification binding을 포함합니다. WU-12 구현은 완료되었지만 exact-SHA 검증 이후 저장소가 변경되면 릴리스 준비 전에 새로운 self-hosted workflow 검증을 다시 통과해야 합니다. 공개 경계가 성공하기 전까지 PyPI의 최신 공개 패키지는 Tool `0.3.5`입니다.

## 주요 생명주기 소유권

Tool `0.3.7`의 정식 분류는 계속해서 정확히 다음 다섯 가지입니다.

| 분류 | 의미 |
| --- | --- |
| `PRODUCT` | Product 생명주기가 주로 소유하는 책임입니다. |
| `DEVELOPMENT_TOOLING` | 개발을 생성·검사·검증·변환·생성·마이그레이션·분석·테스트하기 위한 개발 생명주기 책임입니다. |
| `DELIVERY` | 릴리스 준비, 패키징, 공개, 승격, 배포 또는 목적지 인도를 담당하는 책임입니다. |
| `OPERATIONS` | 인도 이후 건강성, 복구, 조정, 유지보수, 운영을 지속적으로 담당하는 책임입니다. |
| `NEUTRAL_CONTRACT` | 실행되지 않고 특정 생명주기를 소유하지 않으며 독립적으로 거버넌스되는 계약 책임입니다. |

`UNKNOWN`, `CONFLICT`, `INCOMPLETE`, `PENDING`, confidence 값과 migration 상태는 워크플로/평가 상태이지 추가 아키텍처 분류가 아닙니다.

### Tool 0.3.5 호환성 경계

Tool `0.3.5`의 역사적 분류는 다음과 같습니다.

```text
PRODUCT
TOOLCHAIN
NEUTRAL_CONTRACT
```

Tool `0.3.7`은 Tool `0.3.6`에서 확립된 다섯 분류 모델을 유지합니다. 따라서 `TOOLCHAIN`은 **Tool `0.3.5` 레거시 입력**이며 현재 정식 별칭이 아닙니다. 기존 Toolchain 책임은 실제 생명주기 소유권에 따라 `DEVELOPMENT_TOOLING`, `DELIVERY`, `OPERATIONS`로 이동하거나 분리가 필요할 수 있습니다. 일괄적인 `TOOLCHAIN -> DEVELOPMENT_TOOLING` 치환은 금지됩니다.

Tool `0.3.7`은 명시적으로 지원되는 historical source를 위한 증거 기반 direct current-target migration을 제공합니다. Migration capability는 repository adoption authority와 분리되며 inference를 project intent로 바꾸지 않습니다.

## 분류는 경로나 기술이 아닙니다

분류는 파일명, 디렉터리, 언어, 프레임워크, 실행 가능 여부, workflow 제공자, 컴파일 특성, 실행 시간, 테스트 상태, confidence가 아니라 **그 책임을 지배하는 생명주기 의무**를 기준으로 합니다.

예시:

```text
Product-specific verification responsibility       -> PRODUCT
Reusable verification framework / test SDK         -> DEVELOPMENT_TOOLING
Product runtime implementation                      -> PRODUCT
Release-unit assembly / publication automation      -> DELIVERY
Post-deployment health or recovery automation       -> OPERATIONS
Independent non-executable shared contract          -> NEUTRAL_CONTRACT
```

`tests/`, `tools/`, `deploy/`, `ops/`, `.github/workflows/` 같은 경로는 증거 맥락일 뿐 아키텍처 권한이 아닙니다.

## Responsibility Map v2

Tool `0.3.7`은 Responsibility Map v2를 프로젝트 소유 아키텍처 선언 모델로 사용하며 다음 축을 서로 분리합니다.

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

정식 role은 다음과 같습니다.

```text
IMPLEMENTATION
VERIFICATION
AUTOMATION
CONFIGURATION
DOCUMENTATION
GOVERNANCE
```

정식 관계 타입은 다음과 같습니다.

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

associated artifact는 하나의 분류된 anchor component에 종속되는 프로젝트 소유 비컴포넌트 지원 표면입니다. 독립적으로 관리되는 실행 책임이나 생명주기 책임을 숨기는 용도로 사용해서는 안 됩니다.

## explicit / template / hybrid 선언

정식 source mode는 다음과 같습니다.

```text
explicit
    repository directly declares the complete map

template
    repository explicitly selects one immutable revision-bound template

hybrid
    repository explicitly selects a template and adds project-owned
    overrides, extensions, or removals
```

초기 템플릿 카탈로그는 다음과 같습니다.

```text
python-package-library
python-cli-application
mixed-product-development-delivery
```

템플릿 선택은 명시적입니다. PTSIP는 저장소 레이아웃, 언어, 프레임워크 감지, manifest, confidence를 근거로 템플릿을 자동 선택하지 않습니다.

## Source declaration과 Effective Responsibility Map

모든 source mode는 결정적이며 비권위적인 materialization을 통해 하나의 Canonical Effective Responsibility Map으로 해석됩니다.

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

Source Project Profile이 계속 프로젝트 소유 아키텍처 권한입니다. Materialization은 템플릿을 선택하거나 소유권을 추론하거나 잘못된 아키텍처를 수리하거나 source declaration을 자동으로 다시 쓰면 안 됩니다.

해석 결과는 다음과 같은 선언 provenance를 유지합니다.

```text
PROJECT_EXPLICIT
TEMPLATE
PROJECT_OVERRIDE
PROJECT_EXTENSION
PROJECT_REMOVAL
```

and has deterministic digest identity for reproducibility. Neither 입증 또는 digest는 대체 아키텍처 권한을 부여합니다.

## 설치 및 사용

PTSIP는 Python 3.11 이상이 필요합니다.

최신 **공개 릴리스** 설치:

```powershell
python -m pip install PTSIP
```

최신 **공개 릴리스**로 업데이트:

```powershell
python -m pip install --upgrade PTSIP
```

Tool `0.3.7`이 공개되기 전에는 이 명령이 PyPI의 Tool `0.3.5`를 설치할 수 있습니다. 현재 릴리스 후보 소스 개발은 다음과 같이 설치합니다.

```powershell
python -m pip install -e ".[dev]"
```

주요 명령:

```powershell
ptsip --version
ptsip spec
ptsip doctor .
ptsip inspect .
ptsip pilot .
ptsip adopt --help
ptsip validate .
ptsip clarify .
ptsip gate .
ptsip resolve --help
ptsip conform .
```

기본 프로젝트 소유 프로필은 저장소 루트의 `ptsip.yaml`이며 프로젝트는 `--profile`을 통해 다른 명시적 경로를 일관되게 사용할 수 있습니다.

## Adoption과 Decision Authority

저장소 증거는 아키텍처 권한이 아닙니다. Candidate discovery, 경로명, 템플릿, heuristic, agent confidence는 검토를 지원할 수 있지만 프로젝트 의도를 만들어낼 수 없습니다.

Tool `0.3.7`에서는 `classification` 자체가 주요 생명주기 소유권 권한입니다. 새로운 정식 결정은 다음과 같은 사실을 사용합니다.

```text
classification
purpose
shipped
runtime_required
executable
```

역사적 `lifecycle_owner`는 레거시 마이그레이션 증거이며 Tool `0.3.7`의 두 번째 소유권 권한이 아닙니다.

Dry-run 예시:

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

검토한 후 명시적으로 적용합니다.

```powershell
ptsip adopt . `
  --component tools `
  --classification DEVELOPMENT_TOOLING `
  --purpose "Repository-local generation tooling" `
  --shipped no `
  --runtime-required no `
  --executable yes `
  --apply `
  --json
```

준비된 write는 저장소/프로필 상태가 바뀌면 stale로 거부되어야 합니다.

PTSIP는 다음 네 가지를 분리합니다.

```text
Specification
    -> normative rules

Decision Authority
    -> which explicit coordinated architecture answer won

Project Profile / Responsibility Map
    -> durable project-owned declaration

Observed evidence
    -> what the repository and artifacts actually do
```

Decision Authority는 `ptsip.yaml`을 대체하지 않으며 conformance를 증명하지도 않습니다.

## 분산 의사결정 조정

Reference Tool은 다음 전용 Git ref를 통해 저장소 분산 의사결정 조정을 지원합니다.

```text
refs/heads/ptsip-policy
```

GitHub은 Tool backend이며 보편적인 Specification 의존성이 아닙니다. 조정 모델은 안정적인 decision identity, first-valid-resolution-wins, stale-writer-safe conditional mutation, authority freshness, 결정적 reconciliation, fail-closed 동작, global decision state와 clone-local application state 분리를 보존합니다.

PTSIP는 지속적인 background polling이 아니라 action-time synchronization을 사용합니다.

## Product Artifact 경계

Artifact 소유권은 producer 소유권과 독립적입니다. `DEVELOPMENT_TOOLING` 또는 `DELIVERY` component가 `PRODUCT` artifact를 만들 수 있지만 결과 artifact는 Product package 경계를 만족해야 합니다.

Tool `0.3.7`은 snapshot-bound Product Artifact evidence를 지원합니다. 릴리스 검증은 패키징 설정을 증거로 간주하지 않고 실제 빌드된 distribution 내용을 확인하며, `PTSIP-PKG-001`에 따른 확정적인 non-Product 구현 유입을 거부합니다.

## VPMS — Verification Purpose Management System

PTSIP와 VPMS는 서로 다른 질문에 답합니다.

```text
PTSIP
    Who owns this responsibility across its lifecycle?

VPMS
    Why does this Verification Case exist, and what does it protect?
```

PTSIP classification과 VPMS Verification Purpose는 별개의 축입니다. PTSIP core는 VPMS에 의존하지 않으며 VPMS는 이미 해석된 PTSIP metadata의 좁은 read-only projection만 소비합니다.

현재 VPMS 호환성 vocabulary에는 `PRODUCT | TOOLCHAIN`이 남아 있을 수 있습니다. VPMS의 `TOOLCHAIN`은 Tool `0.3.7`의 PTSIP 정식 분류가 아닙니다.

VPMS PASS는 PTSIP `CONFORMANT`를 의미하지 않고, PTSIP `CONFORMANT` 역시 기능 검증 PASS를 의미하지 않습니다.

## Conformance

`ptsip conform`은 source declaration을 Effective Responsibility Map으로 해석한 뒤 선언된 아키텍처와 관찰된 evidence를 적용 가능한 PTSIP 규칙에 따라 평가합니다.

완료된 결과는 다음과 같습니다.

| Exit code | 결과 |
| --- | --- |
| `0` | `CONFORMANT` |
| `5` | `NON_CONFORMANT` |
| `6` | `INCOMPLETE` |

유효한 프로필만으로 conformance가 증명되지는 않습니다. 필수 규칙 결과를 숨길 수 있는 증거 부족은 fail-closed `INCOMPLETE`로 남으며 Tool은 불확실한 저장소를 억지로 green으로 만들지 않습니다.

## Tool과 Specification 생명주기

PTSIP Tool과 PTSIP Specification은 독립적으로 버전 관리됩니다.

- `pyproject.toml`은 Tool/package source version을 소유합니다.
- `ptsip --version`은 설치된 Tool version을 보고합니다.
- `ptsip spec`은 Tool에 바인딩된 정확한 Specification family와 immutable revision을 보고합니다.
- `spec/`, `schemas/`, `registry/`는 canonical Specification asset입니다.
- `src/ptsip/specdata/`는 Tool에 포함된 동일한 machine-readable asset입니다.

Tool `0.3.7`은 독립적인 PP 및 Specification 정체성에 바인딩됩니다.

```text
Project Profile pp.1.01
Specification 0.3.7-draft
SPEC_REVISION 3c47816770d194ae42f98faedc911d980db0e62a
```

새 immutable revision은 실제 normative change가 있을 때만 필요합니다. release workflow, test, planning, status, documentation-only 변경만으로 `SPEC_REVISION`을 이동하지 않습니다.

## Tool 0.3.7 검증 및 릴리스 상태

WU-12 구현과 불변 Specification freeze는 완료되었습니다. 릴리스 handoff 중 repository CI infrastructure가 변경되었으므로 현재 `main` source state는 새로운 exact-SHA `tooling-test` 검증이 필요합니다.

```text
Tool:              0.3.7
Project Profile:   pp.1.01
Specification:     0.3.7-draft @ 3c47816770d194ae42f98faedc911d980db0e62a
exact source SHA:  pending final post-CI main commit
tooling-test:      PENDING for the current exact main SHA
publication:       NOT RUN
```

이전 workflow evidence는 해당 SHA에 대한 역사적 증거로만 유지되며 이후 source state를 검증하지 않습니다.

남은 릴리스 경계는 다음과 같습니다.

```text
final main exact SHA
    -> tooling-test.yml on that exact SHA
    -> require self-hosted/tooling-test success
    -> full regression, independent identity, distribution, artifact, and wheel smoke PASS
    -> release.yml and reviewed draft publication from the same source identity
    -> tooling-release.yml publication verification and PyPI Trusted Publishing
```

현재 구현과 handoff 경계는 [`STATUS.md`](STATUS.md), [`planning/0.3.7/WU-12-specification-binding-capability-registry-release-readiness.md`](planning/0.3.7/WU-12-specification-binding-capability-registry-release-readiness.md), [`releasenote/tool/0.3.7.md`](releasenote/tool/0.3.7.md)를 참고하십시오.

## Consumer Repository 비침투 원칙

PTSIP는 Consumer Repository가 Tool 사용만을 위해 PTSIP 전용 `.ptsip/`, cache, report, hidden state directory를 만들도록 요구하지 않습니다. External inspection과 Pilot은 기본적으로 read-only이며 Tool 소유 local state는 사용자가 저장소 경로를 명시적으로 선택하지 않는 한 Consumer Repository 밖에 둡니다.

## 프로젝트 상태

PTSIP는 여전히 experimental 단계입니다. 현재 exact-main completion gate가 성공하기 전까지 Tool `0.3.7`은 공개되지 않은 release candidate입니다. 과거 Tool release와 Specification 기록은 [`releasenote/`](releasenote/) 아래에 보존됩니다.
