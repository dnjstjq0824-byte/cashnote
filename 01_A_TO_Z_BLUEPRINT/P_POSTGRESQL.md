# P — PostgreSQL (엔터프라이즈 DB 아키텍처·56개 테이블 DDL·정합성·멀티테넌트 격리)

**문서 ID**: AZ-P_POSTGRESQL  
**프로젝트 코드명**: BN Merchant OS (`BN-MERCHANT-OS-GEMINI4-ARGON-MASTER-BLUEPRINT-1000-001`)  
**관련 Work Order 범위**: `WO-MOS-0851 ~ WO-MOS-0880`  
**설계 원칙**: 제0조~제32조 마스터 명령 전수 준수 (독립 플랫폼 원칙, Level 1~4 데이터 수집 전략, 미승인 외부 API 허위 구현 금지)

---

## 1. 목표 (Objective)

제19조에 명시된 10개 도메인 56개 이상의 전체 핵심 테이블을 누락 없이 설계하고, 금융 데이터 무결성(`NUMERIC(19,4)`, `CHECK` 제약, `UNIQUE` 멱등성 키), 멀티테넌트 격리(Row-Level Security), 감사 추적성, 파티셔닝 및 마이그레이션 전략을 정의한다.

---

## 2. 사용자 역할 (User Roles & Actors)

- **주요 액터**: Database Architect, Backend Engineer, Settlement/Reconciliation Agent, Security Auditor
- **권한 경계**: 모든 요청은 인증된 사용자(`user_id`)와 명시적으로 선택·검증된 사업자(`business_id`) 및 매장(`store_id`) 컨텍스트 하에서만 실행되며, 타 사업장 데이터에 대한 교차 접근은 애플리케이션 미들웨어와 PostgreSQL RLS(Row-Level Security) 양쪽에서 차단된다.

---

## 3. 기능 명세 (Functional Specifications)

- **핵심 기능 1**: 10대 도메인 56개 핵심 테이블 완비: Identity(6), Business(5), Connectors(7), Revenue(10), Expenses & Ledger(8), Tax(4), Store Operations(6), CRM & Analytics(6), Marketplace & Subscription(7), Operations(6) — 상세 DDL은 `03_DATABASE/migrations/`에 수록
- **핵심 기능 2**: 금융 금액 데이터 타입 표준: 부동소수점(`FLOAT`/`REAL`) 사용을 전면 금지하고 모든 통화 금액은 `NUMERIC(19,4)` 및 정수 최소통화단위(`BIGINT`) 병행 검증, 모든 시각은 `TIMESTAMPTZ`(UTC 저장, 매장 타임존 변환 표시)로 통일
- **핵심 기능 3**: 멀티테넌트 이중 방어선(Defense-in-Depth): 모든 매장 종속 테이블에 `store_id UUID NOT NULL REFERENCES stores(id)` 또는 `business_id UUID NOT NULL`을 강제하고, 애플리케이션 쿼리 조건 + PostgreSQL Row-Level Security(`CREATE POLICY tenant_isolation_policy ... USING (store_id = current_setting('app.current_store_id')::uuid)`)를 동시 적용
- **핵심 기능 4**: 대용량 시계열 테이블 월별 Range Partitioning: `sales_transactions`, `payment_authorizations`, `journal_lines`, `audit_logs` 테이블은 거래일/생성일 기준 파티셔닝 지원 구조로 설계
- **핵심 기능 5**: 원본 추적(Data Provenance) 공통 컬럼 표준: 주요 거래 테이블에 `source_level`, `source_provider_id`, `raw_import_batch_id`, `external_ref_id`, `dedup_fingerprint` 필수 포함

---

## 4. 사용자 시나리오 (End-to-End User Scenario)

> **실전 시나리오**: 만약 백엔드 개발자가 실수로 `WHERE store_id = $1` 조건을 누락한 채 `SELECT * FROM sales_transactions` 쿼리를 실행하더라도, DB 트랜잭션 시작 시 설정된 `SET LOCAL app.current_store_id = '...'`와 PostgreSQL RLS 정책에 의해 현재 인가된 매장의 데이터만 반환되며 타 매장 데이터 유출이 100% 차단된다.

1. **Pre-condition**: 사용자가 정상 로그인되어 있고 해당 매장에 대해 역할 권한(`RBAC/ABAC`)을 보유하고 있다. 외부 연동 기능의 경우 `provider_catalog` 및 `data_connections` 상태가 확인된다.
2. **Normal Flow**: 요청이 API Gateway를 통과하여 도메인 서비스에서 처리되고, 원본 출처(`data_source_level`)와 함께 PostgreSQL 트랜잭션 및 `outbox_events`에 기록된다.
3. **Fallback Flow (연동 차단/미계약 시)**: 만약 외부 API가 `CONTRACT_REQUIRED` 또는 `PARTNER_ONLY(BLOCKED)` 상태이면, 가짜 실시간 데이터를 표시하지 않고 즉시 **Level 2(공식 정산파일 CSV/Excel 업로드)** 또는 **Level 1(수기 입력)** 인터페이스로 안내하여 핵심 장부·대사 업무가 중단 없이 수행되도록 보장한다.

---

## 5. 백엔드 아키텍처 및 구현 명세 (Backend Architecture)

- **언어 및 프레임워크**: Go 1.22+, 모듈형 모놀리스 아키텍처 + 독립 비동기 워커(Go Workers)
- **핵심 구현 모듈**: `internal/db` 패키지: `pgxpool` 커넥션 풀, 트랜잭션 컨텍스트 래퍼(`WithTenantTx(ctx, storeID, fn)`), `golang-migrate` 기반 순방향/역방향 스키마 마이그레이션
- **트랜잭션 관리**: 모든 상태 변경과 원장 반영은 `pgx/v5` 트랜잭션 내부에서 도메인 데이터 변경 + `audit_logs` 기록 + `outbox_events` 발행을 원자적(Atomic)으로 커밋한다.

---

## 6. 프론트엔드 및 모바일 명세 (Frontend & Mobile)

- **구현 범위**: N/A (백엔드 및 관리자 DB 상태 모니터링 뷰와 연동)
- **상태 표현 표준**:
  1. `LOADING`: 스켈레톤 UI 및 진행률 표시
  2. `SUCCESS`: 데이터 표시 + 우측 상단 `DataProvenanceBadge`(데이터 출처 Level 1~4 및 최종 갱신 시각)
  3. `EMPTY`: 초기 온보딩 안내 및 첫 데이터 등록/업로드 CTA 버튼
  4. `ERROR`: 사용자 친화적 오류 메시지 + `request_id` 복사 + 재시도 버튼
  5. `BLOCKED`: 외부 기관 계약/승인 대기 상태의 정직한 고지 + 파일 업로드 대체 경로 버튼

---

## 7. 데이터베이스 설계 연동 (Database Schema Mapping)

- **연관 테이블**: PostgreSQL 16 (`uuid-ossp`, `pgcrypto`, `pg_trgm`, `btree_gist` 확장 사용)
- **데이터 무결성 규칙**:
  - 모든 금액 컬럼은 `NUMERIC(19,4)`를 사용하여 부동소수점 반올림 오차를 원천 배제한다.
  - 모든 시각 컬럼은 `TIMESTAMPTZ`로 저장한다.
  - 원본 거래 레코드는 임의 삭제(`DELETE`)하지 않으며, 정정 시 `corrections` 및 역분개(`journal_entries`)로 감사 추적성을 유지한다.

---

## 8. API 계약 명세 (OpenAPI 3.1 Endpoints)

- **주요 엔드포인트**: 내부 Repository 계층 (`03_DATABASE/` 문서 및 DDL과 1:1 매핑)
- **공통 헤더**:
  - `Authorization: Bearer <OIDC_JWT>`
  - `X-Business-Id: <UUID>`
  - `X-Store-Id: <UUID>`
  - `Idempotency-Key: <UUID>` (모든 `POST`/`PUT`/`PATCH` 상태 변경 요청 시 필수)

---

## 9. 도메인 이벤트 (Domain Events)

- **발행/구독 이벤트**: `db.migration.applied`, `db.slow_query.detected`
- **전달 보장**: Transactional Outbox 테이블(`outbox_events`)에서 Redis Streams / Worker Queue로 At-Least-Once 전달을 보장하며, 컨슈머는 `event_id` 기준 멱등 처리(Idempotent Consumer)를 수행한다.

---

## 10. 데이터 출처 및 신뢰도 등급 (Data Provenance)

- **데이터 원천**: 전체 애플리케이션 도메인 트랜잭션
- **출처 태깅 규칙**:
  - `LEVEL_1_MANUAL`: 사용자 직접 입력 (신뢰도: 사용자 책임, 즉시 반영)
  - `LEVEL_2_FILE`: 공식 정산 엑셀/CSV 업로드 (신뢰도: 파일 해시 및 포맷 검증됨)
  - `LEVEL_3_API`: 공식 계약/승인된 외부 API (신뢰도: 기관 전자서명/TLS 검증됨)
  - `LEVEL_4_RECONCILED`: 복수 데이터 소스(예: 카드매입 + 은행입금) 교차 대사 완료 (신뢰도: 최고 등급 확정)

---

## 11. 오류 처리 및 장애 복구 (Error Handling & Resilience)

- **예외 처리 정책**: 데드락(Deadlock) 방지를 위해 복수 원장 계좌 업데이트 시 항상 `account_id` 사전순(Lexicographical Order)으로 Row Lock(`SELECT ... FOR UPDATE`) 획득; Serialization/Deadlock 오류 발생 시 최대 3회 지수 백오프 트랜잭션 재시도
- **표준 에러 코드**: `AUTH_FORBIDDEN_STORE`, `CONNECTOR_EXTERNAL_BLOCKED`, `LEDGER_UNBALANCED_ENTRY`, `IMPORT_DUPLICATE_BATCH`, `RATE_LIMIT_EXCEEDED`, `UPSTREAM_TIMEOUT_RETRYABLE`

---

## 12. 개인정보 및 보안 통제 (Privacy & Security Controls)

- **보안 설계**: 개인식별정보(주민등록번호, 카드 PAN 원문, 계좌 비밀번호) DB 저장 금지; 계좌번호·휴대전화번호 등 주요 필드는 `pgcrypto`/애플리케이션 레벨 AES-256-GCM 암호화 컬럼(`*_encrypted`)과 검색용 Blind Index(`*_hmac_hash`)를 분리 저장.
- **감사 추적**: 모든 민감 데이터 조회 및 변경은 `audit_logs`에 요청자 ID, 대상 매장 ID, 액션 코드, IP, User-Agent, 변경 전후 요약 및 해시 체인과 함께 기록된다.

---

## 13. 외부 승인 및 법적 선행 요건 (External Approval & Regulatory Requirements)

- **승인·규제 요건**: 개인정보보호법상 법정 보존기간(상법/국세기본법 장부 및 증빙 5년, 전자상거래법 계약/결제 5년, 소비자분쟁 3년, 접속기록 1~2년) 준수를 위한 소프트 삭제 및 분리 보관(`archived_at`) 스키마 적용.
- **차단(Blocked) 정책**: 필수 외부 계약이나 인허가가 완료되기 전까지 해당 라이브 연동 기능은 `feature_flags` 및 `provider_catalog`에서 `BLOCKED`로 분류하며, 가짜 API 응답으로 상용 완료를 가장하는 행위를 엄격히 금지한다.

---

## 14. 성능 및 확장성 목표 (Performance & SLA Targets)

- **목표 지표**: 단건 PK/인덱스 조회 < 2ms, 복합 집계 쿼리 < 50ms, Lock Wait Timeout 3초 제한.

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

- **직접 매핑 범위**: `WO-MOS-0851 ~ WO-MOS-0880`
- **상세 개별 명세 위치**: `11_WORK_ORDERS_0001_TO_1000/individual/` 내 해당 번호 파일 및 `11_WORK_ORDERS_0001_TO_1000/WORK_ORDER_INDEX.csv` 참조.
