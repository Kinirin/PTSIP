| 번호      | 문제                                                               | 성격                      |
| ------- | ---------------------------------------------------------------- | ----------------------- |
| **P01** | Solution Space가 Project Authority를 어떤 입력으로 받는가                   | 구조/권위                   |
| **P02** | `ProjectIntentAuthority`가 자연어 문자열이라 자동 판단하기 어려움                  | 권위 표현                   |
| **P03** | ADR/정책 문서를 PTSIP가 어디까지 자동 해석할 것인가                                | 정책 자동화                  |
| **P04** | 후보가 1개 남았다고 정말 `DETERMINISTIC`이라고 할 수 있는가                        | Solution Space          |
| **P05** | 하나의 문제에서 `DETERMINISTIC`과 `TOOL_CAPABILITY_GAP`이 동시에 생기는 경우      | 결과 모델                   |
| **P06** | mixed-lifecycle component 전체를 Product Artifact에서 빼도 되는가          | **정책 결정**               |
| **P07** | `RemediationDispositionCandidate`와 `SemanticCandidate` 관계        | 후보 모델                   |
| **P08** | `CoverageGap`과 Specification rule의 관계                            | 문제 식별                   |
| **P09** | `DECISION_REQUIRED`가 후보인가, 결과인가                                  | 후보 의미                   |
| **P10** | Candidate가 구체적으로 어느 path/component를 가리키는가                        | 후보 표현                   |
| **P11** | Issue #31의 6개 Candidate를 어떤 조건에서 생성하는가                           | 자동화 정책                  |
| **P12** | 기존 Candidate Evidence가 migration 구조에 결합되어 있음                     | 책임 분리                   |
| **P13** | Conformance CoverageGap과 Remediation CoverageGap 형식이 다름          | 데이터 계약                  |
| **P14** | SemanticRemediationPlan이 실제 target state를 보존하지 않음                | 계획 계약                   |
| **P15** | Semantic Plan이 어떤 repo/evidence/authority 상태에서 만들어졌는지 binding 부족 | Fresh Solve             |
| **P16** | Project Profile과 별도 Decision Authority가 충돌할 때 WU-02가 무엇을 소비할 것인가 | 권위 일관성                  |
| **P17** | mixed-lifecycle 판정을 어떤 증거로 확정할 것인가                               | rule operationalization |
| **P18** | `PTSIP-PKG-001`을 “지원한다”고 말할 수 있는 정확한 범위                          | capability 선언           |
