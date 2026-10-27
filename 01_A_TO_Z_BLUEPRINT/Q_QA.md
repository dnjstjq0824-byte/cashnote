# Q — Quality Assurance (회계 정합성·기능·계약·E2E·보안·장애 테스트 체계)

**문서 ID**: AZ-Q_QA  
**프로젝트 코드명**: BN Merchant OS (`BN-MERCHANT-OS-GEMINI4-ARGON-MASTER-BLUEPRINT-1000-001`)  
**관련 Work Order 범위**: `WO-MOS-0901 ~ WO-MOS-0950`  
**설계 원칙**: 제0조~제32조 마스터 명령 전수 준수 (독립 플랫폼 원칙, Level 1~4 데이터 수집 전략, 미승인 외부 API 허위 구현 금지)

---

## 1. 목표 (Objective)

제26조 및 제29조에 따라 단순 UI 클릭 테스트가 아니라 회계 원장의 차변·대변 일치성, 정산 대사 오탐율 0%, 멀티테넌트 권한 침투 방어, 외부 API 장애 복구력을 검증하는 포렌식 수준의 QA 체계를 정의한다. 실제 외부 API 접근 권한이 없는 영역은 계약 테스트(Contract/Mock Test)와 라이브 운영 검증(Live Verification)으로 정직하게 분리한다.

---

## 2. 사용자 역할 (User Roles & Actors)

- **주요 액터**: QA Agent, Independent Audit Agent, Security Agent, Backend/Frontend/Mobile Engineers
- **권한 경계**: 모든 요청은 인증된 사용자(`user_id`)와 명시적으로 선택·검증된 사업자(`business_id`) 및 매장(`store_id`) 컨텍스트 하에서만 실행되며, 타 사업장 데이터에 대한 교차 접근은 애플리케이션 미들웨어와 PostgreSQL RLS(Row-Level Security) 양쪽에서 차단된다.

---

## 3. 기능 명세 (Functional Specifications)

- **핵심 기능 1**: 5계층 테스트 피라미드: L1 단위 테스트(수수료·세금·영업일·대사 알고리즘, 커버리지 ≥ 85%) → L2 DB 통합 테스트(Testcontainers PostgreSQL 16 실제 DDL·RLS·트랜잭션 검증) → L3 OpenAPI 및 외부 커넥터 계약 테스트(Pact/WireMock 기반 정상·지연·오류 페이로드 검증) → L4 웹(Playwright)·모바일(Flutter Integration Test) E2E 시나리오 A~J 검증 → L5 카오스·보안 침투·회계 불변성 감사
- **핵심 기능 2**: 회계 무결성 자동 불변성 검사(Financial Invariant Property-Based Testing): 수만 건의 무작위 승인·부분취소·매입후취소·합산입금·수수료차액 시퀀스를 주입한 뒤 (1) 전체 분개 차변 합 == 대변 합, (2) 중복 거래 이중 산입 0건, (3) 승인액 ≠ 입금액 혼합 표시 0건임을 수학적으로 검증
- **핵심 기능 3**: 제26조 10대 필수 E2E 시나리오(Scenario A ~ Scenario J) 자동화 테스트 스위트 완비
- **핵심 기능 4**: Mock vs Live 엄격 구분 원칙: 미계약 외부 API(예: 배달앱 비공개 API)는 계약/파일 업로드 테스트 통과 시 `CONTRACT_TEST_PASSED (LIVE_API_BLOCKED)`로 기록하며 절대 상용 라이브 연동 완료로 허위 보고하지 않음

---

## 4. 사용자 시나리오 (End-to-End User Scenario)

> **실전 시나리오**: CI 파이프라인에서 PR이 생성되면 자동으로 PostgreSQL 16 컨테이너가 기동되어 56개 테이블 DDL을 실행하고, 테넌트 A 토큰으로 테넌트 B 데이터를 요청하는 120개 보안 침투 시나리오(Scenario I)와 외부 API 타임아웃 복구 시나리오(Scenario J)가 모두 통과해야만 머지가 허용된다.

1. **Pre-condition**: 사용자가 정상 로그인되어 있고 해당 매장에 대해 역할 권한(`RBAC/ABAC`)을 보유하고 있다. 외부 연동 기능의 경우 `provider_catalog` 및 `data_connections` 상태가 확인된다.
2. **Normal Flow**: 요청이 API Gateway를 통과하여 도메인 서비스에서 처리되고, 원본 출처(`data_source_level`)와 함께 PostgreSQL 트랜잭션 및 `outbox_events`에 기록된다.
3. **Fallback Flow (연동 차단/미계약 시)**: 만약 외부 API가 `CONTRACT_REQUIRED` 또는 `PARTNER_ONLY(BLOCKED)` 상태이면, 가짜 실시간 데이터를 표시하지 않고 즉시 **Level 2(공식 정산파일 CSV/Excel 업로드)** 또는 **Level 1(수기 입력)** 인터페이스로 안내하여 핵심 장부·대사 업무가 중단 없이 수행되도록 보장한다.

---

## 5. 백엔드 아키텍처 및 구현 명세 (Backend Architecture)

- **언어 및 프레임워크**: Go 1.22+, 모듈형 모놀리스 아키텍처 + 독립 비동기 워커(Go Workers)
- **핵심 구현 모듈**: Go `testing` + `testify` + `pgx` Testcontainers, k6 부하 테스트 스크립트
- **트랜잭션 관리**: 모든 상태 변경과 원장 반영은 `pgx/v5` 트랜잭션 내부에서 도메인 데이터 변경 + `audit_logs` 기록 + `outbox_events` 발행을 원자적(Atomic)으로 커밋한다.

---

## 6. 프론트엔드 및 모바일 명세 (Frontend & Mobile)

- **구현 범위**: Vitest + Vue Test Utils + Playwright E2E, Flutter `integration_test`
- **상태 표현 표준**:
  1. `LOADING`: 스켈레톤 UI 및 진행률 표시
  2. `SUCCESS`: 데이터 표시 + 우측 상단 `DataProvenanceBadge`(데이터 출처 Level 1~4 및 최종 갱신 시각)
  3. `EMPTY`: 초기 온보딩 안내 및 첫 데이터 등록/업로드 CTA 버튼
  4. `ERROR`: 사용자 친화적 오류 메시지 + `request_id` 복사 + 재시도 버튼
  5. `BLOCKED`: 외부 기관 계약/승인 대기 상태의 정직한 고지 + 파일 업로드 대체 경로 버튼

---

## 7. 데이터베이스 설계 연동 (Database Schema Mapping)

- **연관 테이블**: 격리된 Ephemeral Test DB (`bn_mos_test_{run_id}`)
- **데이터 무결성 규칙**:
  - 모든 금액 컬럼은 `NUMERIC(19,4)`를 사용하여 부동소수점 반올림 오차를 원천 배제한다.
  - 모든 시각 컬럼은 `TIMESTAMPTZ`로 저장한다.
  - 원본 거래 레코드는 임의 삭제(`DELETE`)하지 않으며, 정정 시 `corrections` 및 역분개(`journal_entries`)로 감사 추적성을 유지한다.

---

## 8. API 계약 명세 (OpenAPI 3.1 Endpoints)

- **주요 엔드포인트**: 전체 26개 API 그룹 대상 OpenAPI Schema Validation 미들웨어 테스트
- **공통 헤더**:
  - `Authorization: Bearer <OIDC_JWT>`
  - `X-Business-Id: <UUID>`
  - `X-Store-Id: <UUID>`
  - `Idempotency-Key: <UUID>` (모든 `POST`/`PUT`/`PATCH` 상태 변경 요청 시 필수)

---

## 9. 도메인 이벤트 (Domain Events)

- **발행/구독 이벤트**: `qa.test_run.completed`, `qa.invariant.violation_detected`
- **전달 보장**: Transactional Outbox 테이블(`outbox_events`)에서 Redis Streams / Worker Queue로 At-Least-Once 전달을 보장하며, 컨슈머는 `event_id` 기준 멱등 처리(Idempotent Consumer)를 수행한다.

---

## 10. 데이터 출처 및 신뢰도 등급 (Data Provenance)

- **데이터 원천**: 합성 골든 데이터셋(Golden Accounting Fixtures: 한국 9대 카드사·배달3사·홈택스 실제 케이스 모사 익명화 테스트셋)
- **출처 태깅 규칙**:
  - `LEVEL_1_MANUAL`: 사용자 직접 입력 (신뢰도: 사용자 책임, 즉시 반영)
  - `LEVEL_2_FILE`: 공식 정산 엑셀/CSV 업로드 (신뢰도: 파일 해시 및 포맷 검증됨)
  - `LEVEL_3_API`: 공식 계약/승인된 외부 API (신뢰도: 기관 전자서명/TLS 검증됨)
  - `LEVEL_4_RECONCILED`: 복수 데이터 소스(예: 카드매입 + 은행입금) 교차 대사 완료 (신뢰도: 최고 등급 확정)

---

## 11. 오류 처리 및 장애 복구 (Error Handling & Resilience)

- **예외 처리 정책**: 테스트 실패 시 실패 시점의 DB 상태 스냅샷, API 요청/응답 전문, Playwright 트레이스 비디오 및 로그를 아티팩트로 자동 보존
- **표준 에러 코드**: `AUTH_FORBIDDEN_STORE`, `CONNECTOR_EXTERNAL_BLOCKED`, `LEDGER_UNBALANCED_ENTRY`, `IMPORT_DUPLICATE_BATCH`, `RATE_LIMIT_EXCEEDED`, `UPSTREAM_TIMEOUT_RETRYABLE`

---

## 12. 개인정보 및 보안 통제 (Privacy & Security Controls)

- **보안 설계**: 테스트 데이터셋에 실제 개인정보나 실존 타인 사업자 계좌번호를 절대 사용하지 않으며, 체크섬 규칙만 만족하는 합성 사업자번호(`000-00-00000` 대역)만 사용.
- **감사 추적**: 모든 민감 데이터 조회 및 변경은 `audit_logs`에 요청자 ID, 대상 매장 ID, 액션 코드, IP, User-Agent, 변경 전후 요약 및 해시 체인과 함께 기록된다.

---

## 13. 외부 승인 및 법적 선행 요건 (External Approval & Regulatory Requirements)

- **승인·규제 요건**: 독립 감사 에이전트(`Independent Audit Agent`)가 개발 에이전트와 분리되어 테스트 증빙 및 SHA-256 해시를 교차 검증.
- **차단(Blocked) 정책**: 필수 외부 계약이나 인허가가 완료되기 전까지 해당 라이브 연동 기능은 `feature_flags` 및 `provider_catalog`에서 `BLOCKED`로 분류하며, 가짜 API 응답으로 상용 완료를 가장하는 행위를 엄격히 금지한다.

---

## 14. 성능 및 확장성 목표 (Performance & SLA Targets)

- **목표 지표**: 전체 단위+통합 테스트 실행 시간 < 4분, E2E Scenario A~J 전체 실행 < 12분.

---

## 15. 테스트 전략 (Verification & Testing Strategy)

- **단위 테스트(Unit Test)**: 도메인 계산 로직, 상태 머신 전이, 경계값 및 예외 케이스 검증
- **통합 테스트(Integration Test)**: 실제 PostgreSQL 16 컨테이너 기반 DDL, FK/UNIQUE 제약, RLS 테넌트 격리 검증
- **계약/E2E 테스트(Contract & E2E Test)**: OpenAPI 3.1 스키마 일치성 검증 및 제26조 필수 E2E 시나리오(Scenario A~J) 연동 검증

---

## 16. 완료 승인 기준 (Acceptance Criteria)

1. 본 문서에 정의된 모든 API 엔드포인트와 DB 테이블이 `03_DATABASE/` 및 `04_API_SPECIFICATION/openapi.yaml`과 명칭·타입 불일치 없이 100% 일치한다.
2. 디자인만 존재하는 더미 버튼이 없으며, 로딩·성공·빈 상태·오류·연동차단(`BLOCKED`) 5가지 상태가 모두 구현 및 검증된다.
3. 타 사업장(`store_id`) 데이터에 대한 무단 접근 시도가 서버 측에서 `403 Forbidden` 또는 빈 결과(RLS)로 100% 차단된다.
4. 미계약 외부 API에 의존하는 기능이 Level 1(수기 입력) 및 Level 2(공식 파일 업로드) 대체 경로를 통해 중단 없이 동작한다.
5. 단위·통합·E2E 테스트 로그 및 아티팩트 증빙이 독립 감사 에이전트(`Independent Audit Agent`)의 검증을 통과한다.

---

## 17. 관련 Work Order 매핑 (Related Work Orders)

- **직접 매핑 범위**: `WO-MOS-0901 ~ WO-MOS-0950`
- **상세 개별 명세 위치**: `11_WORK_ORDERS_0001_TO_1000/individual/` 내 해당 번호 파일 및 `11_WORK_ORDERS_0001_TO_1000/WORK_ORDER_INDEX.csv` 참조.
