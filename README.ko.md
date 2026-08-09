<p align="right">
  <a href="README.md">English</a> | 한국어
</p>

# PTSIP — Product–Toolchain SDK Isolation Policy

> 이 문서는 [`README.md`](README.md)의 한국어 번역본입니다. 번역본과 영문 원본의 의미가 상충하는 경우 영문 원본을 기준으로 합니다.

**상태:** 초안 명세 0.2.0  
**정식 저장소:** `https://github.com/kwaksinwoo01/ptsip`  
**명세 분야:** 소프트웨어 아키텍처 / SDK 거버넌스 / 개발 툴체인 격리  
**Reference Tool:** `0.1.0a1`  
**라이선스:** Apache License 2.0

PTSIP(Product–Toolchain SDK Isolation Policy)는 소프트웨어 개발 키트(SDK)를 그 **목적, 패키징 책임, 의존성 경계, 빌드 환경, 생명주기**에 따라 관리하기 위한 프로젝트 정의 아키텍처 정책입니다.

PTSIP는 두 가지 주요 SDK plane을 구분합니다.

- **Product SDK Plane** — 제품의 일부이거나 제품을 지원하거나 제품과 함께 배포되는 SDK와 라이브러리.
- **Toolchain SDK Plane** — 제품을 빌드, 검증, 마이그레이션, 테스트, 생성, 검사, 릴리스하거나 그 밖의 방식으로 개발하는 데 사용되는 SDK와 개발 도구.

핵심 규칙은 다음과 같습니다.

> **재사용보다 목적이 우선합니다.** 코드 공유 가능성을 고려하기 전에, 컴포넌트가 왜 존재하는지와 어느 생명주기가 해당 컴포넌트를 소유하는지를 기준으로 분류합니다.

PTSIP는 host/target 분리, build-time/runtime 분리, 툴체인 격리 또는 독립적인 생명주기 관리가 새로운 개념이라고 주장하지 않습니다. PTSIP는 이러한 기존 개념들을 결합하여, 명시적인 적합성 기준과 기계 판독 가능한 프로젝트 규칙을 갖춘 더 강한 SDK 거버넌스 경계로 정의한 정책입니다.

## Consumer Repository 비침투성

PTSIP는 도입하는 저장소에 PTSIP 전용 `docs/`, `tools/`, `.ptsip/`, 캐시 또는 보고서 디렉터리를 생성하도록 요구하지 않습니다. 외부 PTSIP 검사 및 Pilot 도구는 기본적으로 Consumer Repository에 대해 읽기 전용으로 동작하며, 사용자가 명시적으로 다른 방식을 선택하지 않는 한 도구가 소유하는 상태는 해당 저장소 외부에 유지되어야 합니다.

프로젝트는 강제 적합성 검사를 위해 기계 판독 가능한 프로필을 자발적으로 제공할 수 있지만, 프로필의 위치는 필수 저장소 토폴로지가 아니라 프로젝트/구성의 관심사로 유지됩니다.

## Reference Tool

이 정식 저장소에는 `src/ptsip/` 아래에 독립적으로 버전이 관리되는 **PTSIP Reference Tool**도 포함되어 있습니다. 저장소는 공유하지만 Specification과 Tool의 릴리스 생명주기는 공유하지 않습니다.

Python 배포 패키지 이름과 CLI 명령은 모두 `ptsip`입니다. 최초 알파 릴리스 `0.1.0a1`은 PyPI Trusted Publishing을 통해 게시됩니다.

```powershell
pip install ptsip==0.1.0a1
ptsip --version
ptsip spec
ptsip doctor .
ptsip inspect .
ptsip pilot .
ptsip validate .
```

`0.1.0a1`은 프리릴리스이므로 최신 프리릴리스를 명시적으로 선택하려는 경우 `pip install --pre ptsip`도 사용할 수 있습니다.

소스 개발에서는 다음을 사용할 수 있습니다.

```powershell
pip install -e ".[dev]"
```

`inspect`와 `pilot`은 Consumer Repository를 수정하지 않습니다. Pilot 상태는 기본적으로 저장소 외부에 저장됩니다(Windows에서는 `%LOCALAPPDATA%\PTSIP`, 그 밖의 플랫폼에서는 이에 대응하는 사용자 상태 디렉터리). `PTSIP_HOME`을 사용해 해당 위치를 재정의할 수 있습니다.

Tool 릴리스는 `tool-v*` 태그/릴리스 네임스페이스를 사용합니다. Specification 릴리스는 별도의 `spec-v*` 네임스페이스를 사용할 수 있으므로, 하나의 Git 저장소 안에서도 두 생명주기를 독립적으로 관리할 수 있습니다.

## PTSIP가 존재하는 이유

규모가 크거나 장기간 유지되는 코드베이스에서는 validator, schema helper, generator, migration module 또는 공통 utility가 점차 제품 런타임 코드와 개발 도구 양쪽에서 공유되기 쉽습니다. 이는 다음과 같은 숨은 결합을 만드는 경우가 많습니다.

- 개발 전용 의존성이 제품 패키징에 유입됩니다.
- 툴체인 변경 때문에 제품 릴리스가 강제됩니다.
- 제품 호환성에 대한 고려가 툴체인의 발전을 가로막습니다.
- 범용 `common` 패키지가 아키텍처상의 소유권을 흐립니다.
- 생명주기의 독립성보다 코드 재사용이 더 중요하게 취급됩니다.

PTSIP는 반대의 선택을 합니다. **생명주기와 책임 경계가 우선이며, 재사용은 조건부입니다.**

## 저장소 구성

### Specification 소유 영역

- [`spec/PTSIP-SPEC.md`](spec/PTSIP-SPEC.md) — 규범적 아키텍처 명세.
- [`spec/PTSIP-TERMINOLOGY.md`](spec/PTSIP-TERMINOLOGY.md) — 정식 용어와 의미.
- [`spec/PTSIP-GOVERNANCE.md`](spec/PTSIP-GOVERNANCE.md) — 명세 변경 및 예외 거버넌스.
- [`spec/PTSIP-CONFORMANCE.md`](spec/PTSIP-CONFORMANCE.md) — PTSIP 적합성을 주장하기 위한 요구사항.
- [`registry/ptsip-registry.yaml`](registry/ptsip-registry.yaml) — 기계 판독 가능한 용어 및 규칙 레지스트리.
- [`schemas/ptsip-profile.schema.json`](schemas/ptsip-profile.schema.json) — 프로젝트 프로필 스키마.
- [`reference/REFERENCE-ARCHITECTURE.md`](reference/REFERENCE-ARCHITECTURE.md) — 참고용 레퍼런스 아키텍처.
- [`adoption/ADOPTION-GUIDE.md`](adoption/ADOPTION-GUIDE.md) — 마이그레이션/도입 절차.
- [`agents/AGENT-CONTRACT.md`](agents/AGENT-CONTRACT.md) — 코딩 에이전트를 위한 간결한 규칙.
- [`profiles/example.ptsip.yaml`](profiles/example.ptsip.yaml) — 프로젝트 프로필 예제.
- [`decisions/`](decisions/) — 명세 아키텍처 결정 기록.
- [`CHANGELOG.md`](CHANGELOG.md) — Specification 변경 이력.

### Reference Tool 소유 영역

- [`src/ptsip/`](src/ptsip/) — 설치 가능한 Python Reference Tool 구현.
- [`tests/`](tests/) — Reference Tool 테스트.
- [`pyproject.toml`](pyproject.toml) — `ptsip`의 PyPI 배포/빌드 메타데이터.
- [`.github/workflows/tooling-test.yml`](.github/workflows/tooling-test.yml) — Python 3.11–3.13 Tool CI.
- [`.github/workflows/tooling-release.yml`](.github/workflows/tooling-release.yml) — `tool-v*` PyPI Trusted Publishing 워크플로.
- [`TOOLING-CHANGELOG.md`](TOOLING-CHANGELOG.md) — 독립적으로 버전이 관리되는 Tool 변경 이력.

### 공유 저장소 자산

- [`LICENSE`](LICENSE) — 이 저장소에 적용되는 Apache License 2.0 조건.
- [`README.md`](README.md) — 프로젝트 개요 및 소유 영역 구성도.
- [`README.ko.md`](README.ko.md) — 프로젝트 개요의 한국어 번역본.

## 규범적 언어

**MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT**, **MAY**라는 단어는 대문자로 표기된 경우에만 BCP 14(RFC 2119 및 이를 갱신한 RFC 8174)의 의미에 따른 규범적 요구사항 키워드로 사용됩니다.

참고 문서:

- RFC 2119: https://www.rfc-editor.org/info/rfc2119/
- RFC 8174: https://www.rfc-editor.org/info/rfc8174/

## 기존 개념과의 관계

PTSIP는 다음 개념들과 관련이 있지만 동일하지는 않습니다.

- host / execution / target 분리
- build-time / runtime 의존성 분리
- 툴체인 격리
- 의존성 그래프 격리
- 독립적인 릴리스 생명주기 관리
- hermetic 또는 reproducible build 관행

## 성숙도

PTSIP 0.2.0-draft는 **프로젝트에서 정의한 초안 명세**이며, ISO, IEEE, IETF, CNCF 또는 그 밖의 외부 산업 표준이 아닙니다. 공개 명세는 PTSIP라는 용어를 재현 가능하게 만드는 것을 목적으로 합니다. 즉 사람, 코딩 에이전트 또는 외부 validator가 적용되는 명세를 식별하고 저장소를 해당 명세에 따라 독립적으로 평가할 수 있어야 합니다.

Reference Tool `0.1.0a1`은 읽기 전용 검사, Pilot 증거 수집 및 프로젝트 프로필 검증을 목적으로 하는 알파 구현입니다. 아직 완전한 자동 PTSIP 적합성 강제를 제공한다고 주장하지 않습니다.

## 라이선스

별도로 명시되지 않는 한 PTSIP 명세와 Reference Tool을 포함한 이 저장소는 **Apache License, Version 2.0**에 따라 라이선스됩니다. 자세한 내용은 [`LICENSE`](LICENSE)를 참조하십시오.
