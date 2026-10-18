<!-- AUTO-GENERATED: registry-driven Go projection; edit canonical README.md instead. -->
<p align="right">
  <a href="../../README.md">en</a> | ko
</p>

# PTSIP — Primary Lifecycle Ownership and Responsibility Isolation Policy

<!-- ptsip-readme-i18n: 36a873876498dc99dc4942643a58fab357c954a6ef8e5090b7d3e02ece388145 -->

**PTSIP 명시된 것과 같이 건축 책임을 기록하십시오, 기계 읽기 쉬운 프로젝트 intent, 그 후에 저장소 증거에 대하여 그것을 verifies.** 특정 폴더 레이아웃 또는 프로그래밍 언어를 강제하지 않고 제품, 개발, 배달 및 작업 수명주기를 분리합니다.

<!-- ptsip-readme-i18n: fe184ad78fd5bd25b5b56d2ec2d68723f1b74501546513df51f70b0d887526ef -->

> **목적은 재사용합니다.** 책임이 존재하고 수명주기가 코드를 공유하기 전에 그것을 소유하는 것을 결정합니다.

<!-- ptsip-readme-i18n: 2326fa36ed3b50d8502638bc070c7a1166c6a7b36fa1b48e83ca40212d5f30d0 -->

## 구성요소 버전

<!-- ptsip-readme-i18n: 6e7fcdcfcdcbfa82cc47f917cdcf89fa00a44354840677a07a05718937ab8a86 -->

| 식별자 | 소스 트리 값 |
| --- | --- |
| PTSIP_Package | `0.3.8.<SHA6>` — 게시 완료 후 결정되는 구성 식별자 |
| PTSIP_Tool | `T.0.3.8.b5` (PTSIP PyPI Package `0.3.8b5`) |
| Project Profile | `pp.1.02` |
| Engine | Validation `0.3.3`; Context/Projection `0.2.1`; Storage/Discovery/Planning/Execution `0.3.1` |

<!-- ptsip-readme-i18n: 1b5e6c193ce4a49f1fb453b3c05c9006e0f7431f820a9e86f038656d72da9768 -->

Package 구성, Tool, 각 엔진과 Project Profile은 독립적인 버전을 유지합니다. 변경 불가능한 Specification 바인딩은 canonical 구성과 shipping manifest에 기록됩니다. [Tool `0.3.8b1` 변경 내용](../releasenote/tool/0.3.8b1.md), [Tool 릴리스 이력](../releasenote/tool/README.md), [릴리스 workflow](../../.github/workflows/tooling-release.yml)를 참조하세요. 릴리스 검증은 `.github/test_modes.yaml`의 모드로 선택하여 `tooling-test.yml`에서만 수행하고, `tooling-release.yml`은 정확한 source의 검증 증거를 소비하여 게시합니다. 실제 게시 상태와 정확한 Package 구성은 버전별 릴리스 산출물에 기록됩니다.

<!-- ptsip-readme-i18n: 9535b10be9abb7a87788c3f5107c21822734a4507aeb464f8710fb8c4866b965 -->

## 빠른 시작

<!-- ptsip-readme-i18n: f242b713166dd6bacc6a19f93ccb3bbae69c453cf1153cd6da20f00bef99f73f -->

Python을 요구합니다 **3.11 또는 더 새로운**. 이 명령은 PowerShell을 사용합니다. 기본 CLI 명령은 다른 쉘에서도 작동합니다.

<!-- ptsip-readme-i18n: 450fa203dde1e07be1879bb3d82689ac61853f7ac8979364b94e3a59128d711f -->

최근 일반적으로 선택한 출판 된 릴리스를 설치 한 다음 identities를 검사하십시오.

<!-- ptsip-readme-i18n: fcde07b23764806188c438e5686eb83d8f1f878cd522fe314c066d3b7e847990 -->

```powershell
python -m pip install --upgrade PTSIP
ptsip --version
ptsip spec
```

<!-- ptsip-readme-i18n: 7278b556e430da33c3958ee8d688408d287478586bf3973deaf94104540f5438 -->

Tool `0.3.8b1`을 명시하여 설치하려면 다음 명령을 사용합니다. 일반적인 pip 버전 선택은 안정 버전을 선택할 수 있습니다:

<!-- ptsip-readme-i18n: 0a66fc01621beb4beec85223a0b244e94545e97f2afabfd5688c003c11b92223 -->

```powershell
python -m pip install "PTSIP==0.3.8b1"
```

<!-- ptsip-readme-i18n: eeb5b10629d40c38fe851ec3b201220313d0abdf230abe498265d788fcc7c847 -->

이 저장소를 직접 개발하려면:

<!-- ptsip-readme-i18n: b40f2830b196ea5d9cd199b4d0d41e4b913ac34accb5fdae5c6a25ab576de114 -->

```powershell
git clone https://github.com/Kinirin/PTSIP.git
cd PTSIP
python -m pip install -e ".[dev]"
```

<!-- ptsip-readme-i18n: 948ebd3fe11235737df4956edb43c78a321418b123066d80c4d109fc6eeb5b56 -->

Project Profile을 변경하기 전에 읽기 전용 또는 진단 작업을 시작하십시오.

<!-- ptsip-readme-i18n: 3889dfcdd346e61f21dcfe7a6df5d7ac36b0961c3e4ba95e748ab0e0b160188c -->

```powershell
ptsip doctor .
ptsip inspect .
ptsip pilot .
ptsip validate .
ptsip conform .
```

<!-- ptsip-readme-i18n: e5c71eb869d6011586372874957ed00276b169f59fde7095ed0b9f94330b0473 -->

`validate` 확인 선언. `conform` 관찰 된 증거에 대한 적용 가능한 아키텍처 요구 사항을 평가합니다. 유효한 단면도는 적합의 증거가 아닙니다; 정확한 지원된 선택권을 위한 설치된 CLI 도움을 상담하십시오.

<!-- ptsip-readme-i18n: 172930755fd531ee7adc8673d3f14e63d822f2964c1f9eb80379b4fe640941b2 -->

## 시스템은 별도의 유지

<!-- ptsip-readme-i18n: d4c2a5bf97a6a96934b61e47a4a802f4c3185bcead3a3b597520c8d38864a9f5 -->

| 기관/비공개 비행기 | 책임 |
| --- | --- |
| 제품 사양 | Normative 필요조건 및 그들의 immutable 개정 |
| 결정 기관 | Explicit는 건축 대답과 그들의 coordination를 찬성했습니다 |
| Source Project Profile / 책임 지도 | 프로젝트 소유 건축 선언 |
| 관찰 및 준수 | 실제 저장소/artifact 행동과 규칙 평가 |

<!-- ptsip-readme-i18n: d2503344087ea9e2e7c533369ac204caf739dddfe4a4e0e55623756fc841006b -->

**Evidence는 건축 권한이 없습니다.** 파일 경로, 패키지, heuristics, 템플릿, AI 신뢰 및 관찰 된 수입은 검토를 알 수 있습니다. 아무도 조용히 Lifecycle 소유권을 만들 수 있습니다. 도구는 결코 치료 `INCOMPLETE` 패스로 결과.

<!-- ptsip-readme-i18n: 59d315379e83c74a959d6a47ede5d780cd5a696508215697029ba9b2a982a083 -->

## 생명주기 소유권

<!-- ptsip-readme-i18n: 95fd662c19fb6d736995808187c17691c2b3b4a5a8a2eae72c69e5e9bc8f92ec -->

Canonical 공구 `0.3.8a5` 분류는 정확히 남아:

<!-- ptsip-readme-i18n: 94cf51ba65b485288ddf459b2ce39b0e58cdbb1d0dd13e6a2e6503abc1e549e4 -->

| 회사연혁 | 이름 * |
| --- | --- |
| `PRODUCT` | 제품 수명주기에 의해 주로 소유 된 책임. |
| `DEVELOPMENT_TOOLING` | 개발 수명주기 책임 생성, 검사, 검증, 변환, 생성, 마이그레이션, 분석, 테스트, 또는 그렇지 않으면 지원 개발. |
| `DELIVERY` | 출시 준비, 포장, 출판, 프로모션, 배포, 또는 배송 대상에 대한 책임. |
| `OPERATIONS` | 건강, 회복, 재조합, 유지 보수 및 운영에 대한 후 배달 책임. |
| `NEUTRAL_CONTRACT` | Lifecycle-independentGovernance와 비결 계약 책임. |

<!-- ptsip-readme-i18n: cc7afa4a273d81e296ba3978187bd60e489ba0eebbfdf84413e3e43f6ba716d9 -->

`UNKNOWN`, `CONFLICT`, `INCOMPLETE`, `PENDING`, 신뢰 값 및 마이그레이션 상태는 작업 흐름/평가 상태, 추가 분류 되지.

<!-- ptsip-readme-i18n: 15e1604255153cdc9d26507c6f060adc9919778a00205e9f435349aaf362d027 -->

### 도구 0.3.5 호환성 경계

<!-- ptsip-readme-i18n: e4db6f0d9ca63e5a1eecaa5ff84663e1dbd07197a732ad0116d897ad93a05ea8 -->

제품 정보 `0.3.5` 관련 제품:

<!-- ptsip-readme-i18n: 17278f1429d5f3bccfddf7dd59e0566accaaf1724f288eddd7ccfbe242e082d3 -->

```text
PRODUCT
TOOLCHAIN
NEUTRAL_CONTRACT
```

<!-- ptsip-readme-i18n: cec6f91da6aa3da228d14e6a2bfaccf7f7afa4b7be00711eebbe1b2b624be99a -->

제품 정보 `0.3.8a5` Tool에 의해 설치된 5등급 모델 유지 `0.3.6`. `TOOLCHAIN` 견적 요청 **관련 상품 `0.3.5` 이름 ***, 현재 canonical 별명으로. 법적 책임은 될 수있다 `DEVELOPMENT_TOOLING`, `DELIVERY`, `OPERATIONS`, 또는 실제 수명주기 소유권에 따라 분할이 필요합니다. 팟캐스트 `TOOLCHAIN -> DEVELOPMENT_TOOLING` 관련 상품

<!-- ptsip-readme-i18n: b1b4be02ba068cbca057f453affa716d53574825c2d1c9e5913daa36ee2d50cd -->

제품 정보 `0.3.8a5` 명시적으로 지원되는 역사적 소스에 대한 증거 반 방향 직접 현재 표적 이동을 제공합니다. Migration 기능은 저장소 채택 권위에서 분리되고 프로젝트 intent로 방해하지 않습니다.

<!-- ptsip-readme-i18n: 302df01bc949d7d5f580f763ce382813a0620dae3e80c17bf253306eb36d0596 -->

## 분류는 경로 또는 기술이 아닙니다.

<!-- ptsip-readme-i18n: ec0f168b034d4b884cee7a43ae19b48896f18cf35af5f6298c0023eadfef79c1 -->

Classification은 파일명, 디렉토리, 언어, 프레임 워크, 실행 가능한 상태, 워크플로우 공급자, 컴파일 동작, 런타임 기간, 테스트 상태, 또는 신뢰 점수를 지칭합니다.

<!-- ptsip-readme-i18n: 1889f03338b79bbad8bdcdc925ef5e83069569c17e433174ef9a49f62e7d5ee7 -->

예제:

<!-- ptsip-readme-i18n: 8ee2d47784ef1adcce8f3485025a59909853b741545b7573328e2b1989347088 -->

```text
Product-specific verification responsibility       -> PRODUCT
Reusable verification framework / test SDK         -> DEVELOPMENT_TOOLING
Product runtime implementation                      -> PRODUCT
Release-unit assembly / publication automation      -> DELIVERY
Post-deployment health or recovery automation       -> OPERATIONS
Independent non-executable shared contract          -> NEUTRAL_CONTRACT
```

<!-- ptsip-readme-i18n: ff895fa8816bc8d326470bcca3e86ee5d076051d0352b03507cae37ee7b22c61 -->

같은 길 `tests/`, `tools/`, `deploy/`, `ops/`, 또는 `.github/workflows/` 증거 컨텍스트만 있습니다. 그들은 권위가되지 않습니다.

<!-- ptsip-readme-i18n: a5232d5b2f89fc212656886f2cfd0b144bdb4ede49424f5364c0f3dacd8690f8 -->

## 책임 지도 v2

<!-- ptsip-readme-i18n: 497953fa1961cd0bea655c4d409d4a9ddda31aae80e971607471137eb0fd9ea1 -->

제품 정보 `0.3.8a5` Responsibility Map v2를 프로젝트 소유 건축 선언 모델로 사용합니다. 그것은 몇몇 axes를 자주 지킵니다:

<!-- ptsip-readme-i18n: e40dcc8e1b877770a26bfdbb25399ccf24ce934a29574b1eae45746f1bb53209 -->

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

<!-- ptsip-readme-i18n: 6e360f7614ef2f8dfd0af219f2b5bbb9dbf68588e938602bc5d35facf256b499 -->

Canonical 역할은:

<!-- ptsip-readme-i18n: 54a1beccb3d169d1c140133a6160e633a0a184e70cc28ae38685dd4d4c8e9e3e -->

```text
IMPLEMENTATION
VERIFICATION
AUTOMATION
CONFIGURATION
DOCUMENTATION
GOVERNANCE
```

<!-- ptsip-readme-i18n: 67be0ad2ea1f402876f36e863ca58ca74b51e443ffbe6dca4f2633bb7356edd1 -->

Canonical 유형 관계는:

<!-- ptsip-readme-i18n: 7509c363a6f5bbcfaf05e27575a8a5ef61140abd8520e65ae7196ae60c841a94 -->

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

<!-- ptsip-readme-i18n: e5b969455ea611406a9c950a16a7ac9e0d429d0e22fe4c71ca17eb2bbee0e244 -->

관련 artifact는 1개의 분류한 닻 성분에 프로젝트 소유한 비 성분 지원 표면 이하입니다. 그것은 독립적으로 지배 가능한 실행 또는 수명주기 책임을 숨길 수 없습니다.

<!-- ptsip-readme-i18n: 886eb33764e429a661eb5cc7844d1f62762431228823e429e804cf607ba7e626 -->

## Explicit, 템플릿 및 하이브리드 선언

<!-- ptsip-readme-i18n: 44484dc156ad8584283e1a3f1862bfe5ab305434d4cfafc22071971623144d84 -->

Canonical 근원 형태는:

<!-- ptsip-readme-i18n: 7c542e4fe8bc92fa464896b33aef9c6463f7e5c4067fb42ef365df41ac04f79c -->

```text
explicit
    repository directly declares the complete map

template
    repository explicitly selects one immutable revision-bound template

hybrid
    repository explicitly selects a template and adds project-owned
    overrides, extensions, or removals
```

<!-- ptsip-readme-i18n: dd8b7fb992486ea1429984b577b0063a5bd31c55f698b002c646f1f3eec5b949 -->

초기 템플릿 카탈로그는 다음과 같습니다 :

<!-- ptsip-readme-i18n: ed00056918a2d5a86ebe3b23c6bba2197109cb00664ea36b03703fb5b3256bbd -->

```text
python-package-library
python-cli-application
mixed-product-development-delivery
```

<!-- ptsip-readme-i18n: 1e5cc5f775f4f0dc770ea90841a72fe145aa3d66b5d47a77d1fdf95dc70f9e49 -->

템플릿 선택은 명시되어 있습니다. PTSIP 저장소 레이아웃, 언어, 프레임 워크 감지, 표시, 또는 신뢰에서 템플릿을 자동으로 선택하지 않습니다.

<!-- ptsip-readme-i18n: 0803a2a6aa0786b91b5499217ba9be58bad206f1068629fc3e25acebd3271ae9 -->

## Source 선언 및 효과적인 책임 지도

<!-- ptsip-readme-i18n: c89060fb5b0ce92d5a9046549eba7969eb06f89b7b5f7010928aefd92ef07487 -->

모든 근원 형태는 deterministic, non-authoritative materialization를 통해 해결합니다:

<!-- ptsip-readme-i18n: 68ebf09697daa8afbcbd35d82fab6cdb3f85a65eacb0985df91445bae33a8188 -->

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

<!-- ptsip-readme-i18n: 3b4101827be9d09adf839fb91daca3c8a35535a125c56570d9f5273dd1f1c285 -->

Source Project Profile은 프로젝트 소유 건축 권한을 유지한다. 재료화는 템플릿, 인페 소유권, 수리 잘못된 아키텍처를 선택하거나, 조용히 소스 선언을 다시 작성해야합니다.

<!-- ptsip-readme-i18n: 65089781b08d2d6eb6102ff3c4c86d31f5cd1e38e1eab3a0b4be8e8649df30b5 -->

해결된 전망은 다음과 같은 선언 입증을 유지합니다:

<!-- ptsip-readme-i18n: b4c370ea8bf5d7645b939ba3e0a795db92470cde552bebaad816e5f040beefa0 -->

```text
PROJECT_EXPLICIT
TEMPLATE
PROJECT_OVERRIDE
PROJECT_EXTENSION
PROJECT_REMOVAL
```

<!-- ptsip-readme-i18n: 24ff169af9d0f0935a17ddce17bc5feac4d5725bf0711b2deb11b03474e3e90b -->

and has deterministic digest identity for reproducibility. Neither 입증 또는 digest는 대체 아키텍처 권한을 부여합니다.

<!-- ptsip-readme-i18n: dcaa281031e97283191553132c54f1e871bb5575d4bbf6b2c24615b4984d1532 -->

## 단면도 선택과 채택

<!-- ptsip-readme-i18n: 24cf16af287dfddb4f096617de286be6e836f1d0713a7b3a407a213edd9b5f4f -->

repository-local의 경우 PTSIP 구성은 persisted, [`.ptsip/index.json`](../../.ptsip/index.json) 운하의 길 `.ptsip/` 네임스페이스 및 [`.ptsip/profiles/index.json`](../../.ptsip/profiles/index.json) 활동 선택 `*.ptsip.yaml` 사업소개 `default_profile`. 저장소 뿌리 `ptsip.yaml` 역사적인 마이그레이션 / 호환성 입력, 두 번째 활성 선언 기관이 아닙니다.

<!-- ptsip-readme-i18n: 3d6269f0d481d12db999f171a5882d7e5b3e9c9120e73ce7ef6127b17925441f -->

`explicit`, `template`· `hybrid` Canonical 효과적인 책임 지도에 모든 물자를 선언합니다. Materializer는 등록 된 구조와 검증을 해결할 수 있습니다. 그것은해야합니다 **아니다.** 자동 선택 템플릿, 구성 요소 소유자를 발명, 또는 분쟁 해결 프로젝트 의도.

<!-- ptsip-readme-i18n: dff283f8c6a9f673b792d492b35188fc94de5ca91350f87e72d8b8ec91861e07 -->

기존 구성 요소의 경우, 적용하지 않고 선언 변경을 검사:

<!-- ptsip-readme-i18n: be7f1072bf1036e6ff7d9bee95297f296a3a664d618cfe06641dbb43759a7cde -->

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

<!-- ptsip-readme-i18n: d8fef10237172227d3a4851ad9deb50c2c36afae3ba56bc8e555c5306ced69f5 -->

추가하여 소유자 검토 변경 만 적용 `--apply`. 이것은 illustrative 명령이며, 지정된 모든 디렉토리를 분류하지 않습니다. `tools`. Canonical 소유권은 `classification`; 역사 `lifecycle_owner` 필드는 마이그레이션 증거입니다.

<!-- ptsip-readme-i18n: e9e0a8607ce9d67dfa7e972445f525f22fd4d9605d14a100d4f86f3dce2ed7f0 -->

GitHub 좌표 결정 권한 사용 `refs/heads/ptsip-policy` **설정하기**. 그것은 신선한 권위를 읽고 stale 작가를 거부하는 조건부 mutations에 의존합니다. Coordination failures block new Architecture-authority writes 오히려 침묵적으로 관련 지역 결정점으로 떨어지는; GitHub는 보편적인 명세 의존성 아닙니다.

<!-- ptsip-readme-i18n: a0f828429acdffc13f8c6b00ff58c657f0b47181eac4d2e6ed0f5aaf4ccb0e5f -->

## 기초 및 소비자 경계

<!-- ptsip-readme-i18n: 2a1ac3b007368a8e02ee202c6c506a40777123786567d638a283f5ca13039c0f -->

7 독립적으로 내장 **Rust 엔진** 버전, 언어 중립 계약 및 등록 된 엔트리 포인트 바인딩 [`src/ptsip/engines/`](../../src/ptsip/engines/). 참조 [공식적인 엔진 색인](../../src/ptsip/engines/index.json).

<!-- ptsip-readme-i18n: 8842cd10ccec0902c8ab9d79dbecec3ca123514236ca59efe9a19453c9f9a8eb -->

| 제품 정보 | 공급 능력 |
| --- | --- |
| 설정하기 | 유효성 검사, ID/source-bound Context 스냅샷 |
| 제품 정보 | 공급자 소유권을 가지고 없는 Mediate 공급자 가동 |
| 인증 및 인증 | schemas, 계약, 유형 I / O, 사전 조건 및 사례 확인 |
| 팟캐스트 | 등록된 근원 마감 및 invocation 바인딩을 해결하십시오 |
| 회사연혁 | 주문 의존도, 변화 평가 및 네이티브 invocation 준비 |
| 계정 관리 | bounded caller-admitted 요청 및 기록 효과/스톱 |
| 사업소개 | Render validated Context에 선언된 출력 형식 |

<!-- ptsip-readme-i18n: dd2c5e0094bb02273829f4ad2b86c5896d16ba74d1296268b2010fb8f5068d3c -->

PTSIP `0.3.8b5`는 네이티브 엔진 7개를 모두 포함하는 Windows x64·Linux x64 wheel과 Rust 소스·lockfile을 포함하는 source 배포본을 제공합니다. 설치는 authority admission, activation 또는 API cutover를 허용하지 않습니다. 기존 소비자 API는 유지됩니다.

<!-- ptsip-readme-i18n: 17ab9f498abaac0cd0332fe5fcd7aee9652dedcbdd3b4eaf1a829ed7704329da -->

**Validation 참조 지원:** 내부 `$ref`(JSON Pointer·anchor·재귀 스키마), 외부 `$id` 참조와 `$dynamicRef`·`$dynamicAnchor`를 지원합니다. 선택적인 `schema_resolution`으로 명시한 디렉터리에서 JSON 스키마를 탐색하고 허용한 origin에서 누락된 HTTP(S) 참조를 가져올 수 있습니다. 결과에는 리소스 URI와 SHA256 증거가 포함됩니다. [지원 안내와 완전한 요청 예제](https://github.com/Kinirin/PTSIP/blob/dev/0.3.8b2/docs/validation-schema-references.md)를 참조하거나 `ptsip-engine-validation --help`를 실행하세요.

<!-- ptsip-readme-i18n: 16237957bf45b9495cd88af15fbc6968d93cb312c35004ad238924423610e07e -->

PTSIP 제품정보 **lifecycle 책임 소유**. 독립 [`src/vpms/`](../../src/vpms/) 검증 프로토콜 관리 시스템 (Vertification Protocol Management System)VPMS) 통제 **명시된 Verification Cases가 등록, 선택, 실행 및 보고된 방법**. VPMS 목적/verification 상태는 분리됩니다 PTSIP 분류 및 준수.

<!-- ptsip-readme-i18n: 6adb8874ce50a04b8eecdde8da719038cc931a19c246101c6351c41ad70ac705 -->

VPMS 공공 조성 용도 `load_registry_snapshot`, `resolve_selection`· `run_cases`명시된 콜러 소유 정의, 참고 및 어댑터와 함께. 선택 사항 PTSIP 브리지는 유효성 검사 가능한 효과적인 책임 지도 자료를 소비합니다; 그것은 프로젝트 권위를 수립하지 않습니다. VPMS PASS는 불완전하지 않습니다 PTSIP `CONFORMANT`, 또는 부 versa.

<!-- ptsip-readme-i18n: 2bac3c0822e391b501ffcee913f39f9c9bdd592209e4675b3a78a743ae9c8038 -->

PTSIP 적합성 `CONFORMANT` (예) `0`), `NON_CONFORMANT` (`5`), 또는 `INCOMPLETE` (`6`). 필요한 증거를 미끄러운 것은 실패합니다. 제품 유통 검증은 실제 스냅 샷-바운드 내장 된 artifacts를 검사합니다. 빌드 단계를 소유하는 생산품 구성 요소는 결과 제품 artifact의 소유자를 자동으로하지 않습니다.

<!-- ptsip-readme-i18n: 44b4c80bf0e32e26be3a57fbe31563cafda7fe2bda0406787ace36140fa6e190 -->

## 개발자 워크플로우 및 저장소 맵

<!-- ptsip-readme-i18n: 4e65bcaaafc5058626617215e0d8eb5886684f38a986ff276a832289c9b61023 -->

| 오시는길 | 책임 |
| --- | --- |
| [`.ptsip/`](../../.ptsip/) | Consumer-repository indexes, 프로파일, 컨텍스트; 작업/시간은 예약 |
| [`src/policy/index.yaml`](../../src/policy/index.yaml) | 현재 지원 특징 정책 등록 (`ACTIVE`/`DRAFT`/`RETIRED`) |
| [`developer/policy/`](../../developer/policy/) | 개발자 기계 정책 및 바인딩 |
| [`developer/planning/`](../../developer/planning/) · [`planning/`](../../planning/) | 정책 실행 계획, 작업 단위 및 역사 기록 |
| [`developer/automation/`](../../developer/automation/) | 이름 * 개발자 툴링, 정책/계획/출판 제어 |
| [`src/ptsip/`](../../src/ptsip/) | Python Consumer Tool, 적합성, 프로파일 재료화 및 엔진 계약 |
| [`src/ptsip/engines/`](../../src/ptsip/engines/) | 7개의 독립적으로 소유된 네이티브 엔진 구현/계약 |
| [`src/vpms/`](../../src/vpms/) | 독립적 인 검증 subsystem |
| [`src/tests/`](../../src/tests/) · [`developer/tests/`](../../developer/tests/) | 소비자, 개발자 및 네이티브 검증 |
| [`registry/`](../../registry/) · [`schemas/`](../../schemas/) | 사양 등록 및 스키마 |
| [`.github/workflows/`](../../.github/workflows/) | 테스트, 릴리즈 및 README 번역 워크플로우 |
| [`docs/releasenote/`](../releasenote/) | 도구, 프로젝트 프로필 및 사양 릴리스 기록 |

<!-- ptsip-readme-i18n: e2dea99f3cc943476fb0100fdf4d5a615503cc2eda5871849ef79b74253d474c -->

코딩 대리인을 위해, [`AGENTS.md`](../../AGENTS.md) 입학 계획 **아니다.** 관련 기관 정책 거버넌스 계획, 계획 거버넌스 구현; 낮은 수준은 침묵적으로 권한 결정을 변경해야합니다. 이름 * [협력.md](../../CONTRIBUTING.md) 그리고 [개발자 자동화 가이드](../../developer/automation/README.md).

<!-- ptsip-readme-i18n: 72000d513a1151051d5f5393615fa1885113fae106ac0896d27bb082b1b65c32 -->

Bounded Dependency analysis and validation Planning are advisory and do not replace current Architecture 선언 or exact-source release validation:

<!-- ptsip-readme-i18n: 0423f61b0b43ac93ee8929965e85aee75574e7a2a48269c0502baa0e87aa67f1 -->

```powershell
ptsip dependency analyze .
ptsip dependency review-pack . --max-items 8 --max-context-bytes 12000
ptsip dependency validation-plan . --changed src/ptsip/dependency_analysis.py --json
```

<!-- ptsip-readme-i18n: 11518e667820b411755a9cbfcc3f633252809a646cc0c928eaaf26f1875bfdb7 -->

PREPARE 및 PUBLISH는 독립적으로, 정확한 자원 검증 단계 [`.github/workflows/tooling-release.yml`](../../.github/workflows/tooling-release.yml). 문서, 테스트, 워크플로우 또는 도구 전용 변경은 immutable Specification을 변경하지 않습니다.

<!-- ptsip-readme-i18n: bc857e872269efb32e78bc6bffa445c5226891a9128b0dc08c14b839f4892a84 -->

## 노트 및 배경

<!-- ptsip-readme-i18n: 9b64e2d1fe992d064953141ce8b6ea96db3c9384a48531559512ac6cbc876218 -->

- [현재 위치 `0.3.8a5` 공지사항](../releasenote/tool/0.3.8a5.md): 현재 승인, 등록 확인 적용 및 7 엔진 기반; **간행물**.
- [이름 * `0.3.8a4` 기본 정보](../releasenote/tool/0.3.8a4.md): Go-native 개발자 자동화, 프로젝트 프로필 및 릴리스 검증.
- [역사 도구 노트](../releasenote/tool/README.md): 이전 릴리스, namespace 및 Context Plane 마이그레이션.
- [Specification/Project Profile/Tool 릴리즈 레코드](../releasenote/README.md): 독립적으로 versioned Lifecycle 역사.

<!-- ptsip-readme-i18n: e0775e6d76d74833cd954ea3f8f1e7e939f92246af5b0f1c2c3e92d4d041620d -->

## 라이센스 기관

<!-- ptsip-readme-i18n: 0ed51af0b127bcc59078a0f13e688d71bca4941a69fe58ca5653c6765bce3e14 -->

PTSIP 라이센스 당국은 아래에서 독점적으로 포함됩니다. [`License-Authority/`](../../License-Authority/). 기계 읽기 쉬운 입장 점은 입니다 [`License-Authority/license-authority.yaml`](../../License-Authority/license-authority.yaml), 그리고 통제 법적인 원본은 입니다 [`License-Authority/LICENSE.md`](../../License-Authority/LICENSE.md).

<!-- ptsip-readme-i18n: 14628a6a0ca002f678cd9fc046145f14ab421c0c95ae817ae40c007e19a7267e -->

현재 라이센스 당국에 의해 선언 된 수의 효과적인 시간은 `2026-09-15T00:00:00Z` (UTC).

<!-- ptsip-readme-i18n: 273f757fbd6cb2dda56295c1a077a2b63225e0b98e5e1227d557f0d3d7498ab9 -->

이 저장소에 다른 언어의 라이센스 관련 단어는 정보 또는 문맥에만 있습니다. 그것은 라이센스의 일부가되지 않습니다, 수정, 또는 참조에 의해 통합됩니다.

<!-- ptsip-readme-i18n: 1cef3355258f760015285ad4d0f18a62e29b628f2b34417cc2aad74150f106ca -->

일반 도구 릴리스, 사양 검증, 도구 버전 변경, 문서 언급, 키워드 일치, 및 AI semantic 추측은 라이센스 기관 항목을 승인하지 않습니다. Entry는 선언된 트리거만 사용합니다. `License-Authority/license-authority.yaml`.

<!-- ptsip-readme-i18n: 22ddfe0f8e6abeb1ef8baf20648a42115c4c957eeebfb6ac72520fb1d163c561 -->

## 언어 및 프로젝트 상태

<!-- ptsip-readme-i18n: 363cb44c9dc5e13a7b09566f187af0b6c87e8af231e5e8e8ab8f0e01f642f717 -->

이 영어 README는 운하 프로젝트 개요입니다. [관련 문서](./) 관련 상품 `main` 등록 된 Argos 번역 워크플로우를 사용; 번역은 영어 개요 또는 기계 학습 정책을 무시하지 않습니다. PTSIP 실험 및 출시 주장은 정확한 소스 및 검증 증거에 바인딩됩니다.
