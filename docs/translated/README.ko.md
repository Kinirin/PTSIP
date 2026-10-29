<!-- AUTO-GENERATED: registry-driven Go projection; edit canonical README.md instead. -->
<p align="right">
  <a href="../../README.md">en</a> | ko
</p>

# PTSIP — Primary Lifecycle Ownership and Responsibility Isolation Policy

<!-- ptsip-readme-i18n: 36a873876498dc99dc4942643a58fab357c954a6ef8e5090b7d3e02ece388145 -->

**PTSIP 명시된 것과 같이 건축 책임을 기록하십시오, 기계 읽기 쉬운 프로젝트 intent, 그 후에 저장소 증거에 대하여 그것을 verifies.** 특정 폴더 레이아웃 또는 프로그래밍 언어를 강제하지 않고 제품, 개발, 배달 및 작업 수명주기를 분리합니다.

<!-- ptsip-readme-i18n: fe184ad78fd5bd25b5b56d2ec2d68723f1b74501546513df51f70b0d887526ef -->

> **목적은 재사용합니다.** 책임이 존재하고 수명주기가 코드를 공유하기 전에 그것을 소유하는 것을 결정합니다.

<!-- ptsip-readme-i18n: c65bc30eaa46bc4ea9a3206cf9c80db560cd4a599d53f2b2bfad9c8534c54781 -->

## 공개 릴리스 및 구성 요소 버전

<!-- ptsip-readme-i18n: 50c8b1faccea063561e51c1b0cdb3f54045632f1a07e5a53bb55ce44800adbd0 -->

더 보기 **관련 기사 PTSIP 베타는 `0.3.8b5`** (10월 18, 2026). 이 PyPI 배포는 도구, 프로젝트 프로필 및 모든 7 개의 등록된 네이티브 엔진을 하나의 완전한 패키지로 설치합니다.

<!-- ptsip-readme-i18n: 748fdeb33f36f848637e178395de9e0f02e69e1662349c17231e3e2e997ec120 -->

| 이름 * | 이름 * `PTSIP==0.3.8b5` |
| --- | --- |
| PTSIP Package 구성 | `0.3.8.adee8b` |
| PTSIP 도구 | `T.0.3.8.b5` |
| 사업영역 | `pp.1.02` |
| Context / 프로젝션 엔진 | `0.2.1` |
| Storage / Discovery / 계획 / 실행 엔진 | `0.3.1` |
| 검증 엔진 | `0.3.3` (자본계약) `4.0.0`) |

<!-- ptsip-readme-i18n: 9ee82f1d423019dc64f24fbde0aff698b86de5d79c3067b01402941be3b7a37a -->

**출판 artifact versus 개발 소스:** 현재 위치 `main` 소스는 다음 패키지 prerelease 선언 `0.3.8b6` (사용자 정체성 `0.3.8.b6`) 프로젝트 프로필 `pp.2.00` 그리고 독립적으로 버전 도구 `T.0.3.8.b1`. 그것은 **아니다.** 출판된 b5 artifacts를 정의합니다. `dev/0.3.8b2`. 위의 출판 `0.3.8b5` 사실은 역사적인 남아; b6의 소스 선언은 b6이 출판 된 것을 의미하지 않습니다.

<!-- ptsip-readme-i18n: ec82fbe270a8842dd355c0e6b251efd402e87b0f987cbecf2c046eb7ea02701d -->

패키지 구성, 도구, 각 엔진 및 프로젝트 프로필은 독립적 인 식별을 유지합니다. 이름 * [b5 패키지 릴리스 노트](../releasenote/PTSIP_Package/0.3.8.adee8b.md), [b5 공구 방출 주](../releasenote/tool/T.0.3.8.b5.md), [GitHub 릴리스](https://github.com/Kinirin/PTSIP/releases/tag/ptsip-release-912a2b4956e856bec709cadc322cbd68588cdd5b) · [Tool 출시](../releasenote/tool/README.md). 릴리스 검증 및 출판은 기존에 의해 지배된다 [공지사항](../../.github/workflows/tooling-release.yml); 문서 업데이트는 새로운 출판 증거를 구성하지 않습니다.

<!-- ptsip-readme-i18n: 9535b10be9abb7a87788c3f5107c21822734a4507aeb464f8710fb8c4866b965 -->

## 빠른 시작

<!-- ptsip-readme-i18n: f242b713166dd6bacc6a19f93ccb3bbae69c453cf1153cd6da20f00bef99f73f -->

Python을 요구합니다 **3.11 또는 더 새로운**. 이 명령은 PowerShell을 사용합니다. 기본 CLI 명령은 다른 쉘에서도 작동합니다.

<!-- ptsip-readme-i18n: f6867d259a9a4c78e4cf7a0c6209334151d36f44228abf841879e2d6f3366100 -->

설치하기 **관련 기사 `0.3.8b5` prerelease 명시적으로** (Unpinned pip 명령은 안정되어 있는 방출을 선호할 수 있습니다):

<!-- ptsip-readme-i18n: 713b36be44d0c9f0c644fd5b543e506ae9efd6099a6f91b4c7a9f8947c9265a4 -->

```powershell
python -m pip install --upgrade "PTSIP==0.3.8b5"
ptsip --version
ptsip spec
python -m pip show PTSIP
```

<!-- ptsip-readme-i18n: 57ee689732c1ffba396e724ea9574cc731874a233e52e9f125639ffab4674cc1 -->

일반적으로 선택한 안정적인 출시를 위해, 사용 `python -m pip install --upgrade PTSIP`. 위에 핀 명령은 당신이 특히 b5를 원할 때 요구됩니다; `pip show` 설치된 PyPI 버전을 보고, 도구 및 엔진 식별자는 독립적으로 유지됩니다.

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

<!-- ptsip-readme-i18n: 0f62ecdbc56721065510bb11b4e0bf460a518edcaffd7d4c8f8c0f98b481b5b5 -->

## 분류 기술

<!-- ptsip-readme-i18n: ac020f2a262ee5e7393a619f2afe4cf38f7aab51fd436f5399d9b4bf170d8268 -->

### 시스템이 분리하여 관리하는 대상

<!-- ptsip-readme-i18n: d4c2a5bf97a6a96934b61e47a4a802f4c3185bcead3a3b597520c8d38864a9f5 -->

| 기관/비공개 비행기 | 책임 |
| --- | --- |
| 제품 사양 | Normative 필요조건 및 그들의 immutable 개정 |
| 결정 기관 | Explicit는 건축 대답과 그들의 coordination를 찬성했습니다 |
| Source Project Profile / 책임 지도 | 프로젝트 소유 건축 선언 |
| 관찰 및 준수 | 실제 저장소/artifact 행동과 규칙 평가 |

<!-- ptsip-readme-i18n: d2503344087ea9e2e7c533369ac204caf739dddfe4a4e0e55623756fc841006b -->

**Evidence는 건축 권한이 없습니다.** 파일 경로, 패키지, heuristics, 템플릿, AI 신뢰 및 관찰 된 수입은 검토를 알 수 있습니다. 아무도 조용히 Lifecycle 소유권을 만들 수 있습니다. 도구는 결코 치료 `INCOMPLETE` 패스로 결과.

<!-- ptsip-readme-i18n: 7998cb93bbd81b1348fa669617c4857fa0f2eab3ad3977214957db23f6a887f9 -->

### 주 생명주기 소유권

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

<!-- ptsip-readme-i18n: 57c07a430d478dbaf7752026cf7f0b03a95076035ecbc927b5067feb6d91e424 -->

#### Tool 0.3.5 호환성 경계

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

<!-- ptsip-readme-i18n: 6b253932d8b3ea8f0ec1c2e1003eb42fa835999dae9204654dff674a91918bc5 -->

### 경로나 기술과 구별되는 분류

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

<!-- ptsip-readme-i18n: 7fd04feb1ef92b32da6e3c6781982fce11f15191eb0cdd304e24fa034cb6d233 -->

### 책임 맵 v2

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

<!-- ptsip-readme-i18n: a5390b5b2ab9df31ec9ec455c1a2552687f7373fde604b3f829a99480e683bbd -->

### 명시형·템플릿형·혼합형 선언

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

<!-- ptsip-readme-i18n: 9a7804f45ef2ee7167bb41d06d5f65f183a7786983c31c182860e16afdd1f0ec -->

### 원본 선언과 유효 책임 맵

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

## 프로파일 선택과 도입

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

<!-- ptsip-readme-i18n: 061436551626be327ca0e5ea9b12b7245195fd990201b79663763ccecdf69d7a -->

## 네이티브 기반 엔진

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

<!-- ptsip-readme-i18n: 6a0f00e4bc597018fed6d2a63011ec92397e44407b7ae0c4ede3d49412ce62f7 -->

`0.3.8a5` Linux, Windows 및 macOS의 기본 기반을 검증합니다. **없음** implicitly 보조 소비자 생산 임명, 권위 입원, 활성화, 또는 versioned cutover. 오래된 a5 파이썬 휠은 중립 엔진 계약 데이터를 운반합니다. **아니다.** Rust 소스, native binaries 또는 build 출력. 별도로 출판 **사이트맵** Windows/리눅스 휠에는 모든 7개의 네이티브 엔진이 포함되어 있습니다. [b5 릴리스 기록](../releasenote/PTSIP_Package/0.3.8.adee8b.md). 기존 소비자 API는 사용할 수 있습니다. a5 범위는 문서화되어 있습니다. [관련 기사](../releasenote/tool/0.3.8a5.md).

<!-- ptsip-readme-i18n: 16237957bf45b9495cd88af15fbc6968d93cb312c35004ad238924423610e07e -->

PTSIP 제품정보 **lifecycle 책임 소유**. 독립 [`src/vpms/`](../../src/vpms/) 검증 프로토콜 관리 시스템 (Vertification Protocol Management System)VPMS) 통제 **명시된 Verification Cases가 등록, 선택, 실행 및 보고된 방법**. VPMS 목적/verification 상태는 분리됩니다 PTSIP 분류 및 준수.

<!-- ptsip-readme-i18n: 6adb8874ce50a04b8eecdde8da719038cc931a19c246101c6351c41ad70ac705 -->

VPMS 공공 조성 용도 `load_registry_snapshot`, `resolve_selection`· `run_cases`명시된 콜러 소유 정의, 참고 및 어댑터와 함께. 선택 사항 PTSIP 브리지는 유효성 검사 가능한 효과적인 책임 지도 자료를 소비합니다; 그것은 프로젝트 권위를 수립하지 않습니다. VPMS PASS는 불완전하지 않습니다 PTSIP `CONFORMANT`, 또는 부 versa.

<!-- ptsip-readme-i18n: 2bac3c0822e391b501ffcee913f39f9c9bdd592209e4675b3a78a743ae9c8038 -->

PTSIP 적합성 `CONFORMANT` (예) `0`), `NON_CONFORMANT` (`5`), 또는 `INCOMPLETE` (`6`). 필요한 증거를 미끄러운 것은 실패합니다. 제품 유통 검증은 실제 스냅 샷-바운드 내장 된 artifacts를 검사합니다. 빌드 단계를 소유하는 생산품 구성 요소는 결과 제품 artifact의 소유자를 자동으로하지 않습니다.

<!-- ptsip-readme-i18n: 44b4c80bf0e32e26be3a57fbe31563cafda7fe2bda0406787ace36140fa6e190 -->

## 개발자 워크플로와 저장소 안내

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

<!-- ptsip-readme-i18n: ec29ec63dd8a391155d2a3a6fc7db2117e00baca7b1723219532b3f48a64e77b -->

### 릴리스 노트와 배경

<!-- ptsip-readme-i18n: a49fcb30463c21a3fcc301bcedbd8a1eb29397611c6e314ac45e12792c96fd31 -->

- [이름 * `0.3.8b5` 제품 정보](../releasenote/PTSIP_Package/0.3.8.adee8b.md): 구성 `0.3.8.adee8b`, 공구 `T.0.3.8.b5`, 단면도 `pp.1.02` 그리고 7개의 엔진.
- [제품 정보 `T.0.3.8.b5` 기타](../releasenote/tool/T.0.3.8.b5.md): 검증 `0.3.3` / 기능 `4.0.0`, 반복 `$ref` · `$dynamicRef` 지원.
- [게시 b5 GitHub 릴리스](https://github.com/Kinirin/PTSIP/releases/tag/ptsip-release-912a2b4956e856bec709cadc322cbd68588cdd5b): 원본 출판물 및 artifact 기록.
- [역사 도구 노트](../releasenote/tool/README.md): 이전 릴리스 및 소스 전용 후보.
- [포장, 명세 및 프로젝트 단면도 방출 기록](../releasenote/README.md): 독립적 인 버전의 역사.

<!-- ptsip-readme-i18n: b50888879356a6555b71914c35cc9cd5ee86842b93efff3691f3911e1fe40ee0 -->

### 상세 기술 문서

<!-- ptsip-readme-i18n: b9486dfaad35921905b6a79683b5972a76c75bf8701bac91fa6feb4fbd5ae5ba -->

릴리스 요약과 상세 기술 문서는 별도로 기록합니다.

<!-- ptsip-readme-i18n: 328c53f3408172287cbe874d9f4407879b5e8a8202e17f4e159980a49be02fac -->

- [엔진 인터페이스 및 기능](../../src/ptsip/engines/index.json)
- [Project Profile 계약 및 채택](../../src/ptsip/profiles/index.yaml)
- [Schema 참고 검증](../validation-schema-references.md)
- [문서 포맷 및 아카이브 계약](../../.ptsip/contracts/release/Platform/Common/releasenote.json)

<!-- ptsip-readme-i18n: e0775e6d76d74833cd954ea3f8f1e7e939f92246af5b0f1c2c3e92d4d041620d -->

## 라이선스 권한

<!-- ptsip-readme-i18n: 0ed51af0b127bcc59078a0f13e688d71bca4941a69fe58ca5653c6765bce3e14 -->

PTSIP 라이센스 당국은 아래에서 독점적으로 포함됩니다. [`License-Authority/`](../../License-Authority/). 기계 읽기 쉬운 입장 점은 입니다 [`License-Authority/license-authority.yaml`](../../License-Authority/license-authority.yaml), 그리고 통제 법적인 원본은 입니다 [`License-Authority/LICENSE.md`](../../License-Authority/LICENSE.md).

<!-- ptsip-readme-i18n: 14628a6a0ca002f678cd9fc046145f14ab421c0c95ae817ae40c007e19a7267e -->

현재 라이센스 당국에 의해 선언 된 수의 효과적인 시간은 `2026-09-15T00:00:00Z` (UTC).

<!-- ptsip-readme-i18n: 273f757fbd6cb2dda56295c1a077a2b63225e0b98e5e1227d557f0d3d7498ab9 -->

이 저장소에 다른 언어의 라이센스 관련 단어는 정보 또는 문맥에만 있습니다. 그것은 라이센스의 일부가되지 않습니다, 수정, 또는 참조에 의해 통합됩니다.

<!-- ptsip-readme-i18n: 1cef3355258f760015285ad4d0f18a62e29b628f2b34417cc2aad74150f106ca -->

일반 도구 릴리스, 사양 검증, 도구 버전 변경, 문서 언급, 키워드 일치, 및 AI semantic 추측은 라이센스 기관 항목을 승인하지 않습니다. Entry는 선언된 트리거만 사용합니다. `License-Authority/license-authority.yaml`.

<!-- ptsip-readme-i18n: 4d3cfd47f9492904acbf3ea7a8fb44921dd4bd9544325ac0d8ce1f8bce62471e -->

### 언어와 프로젝트 상태

<!-- ptsip-readme-i18n: 363cb44c9dc5e13a7b09566f187af0b6c87e8af231e5e8e8ab8f0e01f642f717 -->

이 영어 README는 운하 프로젝트 개요입니다. [관련 문서](./) 관련 상품 `main` 등록 된 Argos 번역 워크플로우를 사용; 번역은 영어 개요 또는 기계 학습 정책을 무시하지 않습니다. PTSIP 실험 및 출시 주장은 정확한 소스 및 검증 증거에 바인딩됩니다.
