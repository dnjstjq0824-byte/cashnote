# L — Ledger & Accounting (불변 복식부기 원장·거래 정규화·입금 대사·정정 이력)

**문서 ID**: AZ-L_LEDGER  
**프로젝트 코드명**: BN Merchant OS (`BN-MERCHANT-OS-GEMINI4-ARGON-MASTER-BLUEPRINT-1000-001`)  
**관련 Work Order 범위**: `WO-MOS-0251 ~ WO-MOS-0350`  
**설계 원칙**: 제0조~제32조 마스터 명령 전수 준수 (독립 플랫폼 원칙, Level 1~4 데이터 수집 전략, 미승인 외부 API 허위 구현 금지)

---

## 1. 목표 (Objective)

제7조, 제8조, 제9조에 따라 플랫폼의 심장부인 회계 원장(Core Ledger)과 자동 대사(Reconciliation) 엔진을 설계한다. 소상공인에게는 쉬운 장부 화면을 보여주지만, 내부 데이터베이스는 **불변(Append-Only) 복식부기 분개 원칙(차변 합계 = 대변 합계)**으로 구동하여 단 1원의 오차나 무단 덮어쓰기도 허용하지 않는다.

---

## 2. 사용자 역할 (User Roles & Actors)

- **주요 액터**: Accounting & Tax Agent, Settlement/Reconciliation Agent, 매장 대표, 회계 담당자, 세무대리인, 독립 감사관
- **권한 경계**: 모든 요청은 인증된 사용자(`user_id`)와 명시적으로 선택·검증된 사업자(`business_id`) 및 매장(`store_id`) 컨텍스트 하에서만 실행되며, 타 사업장 데이터에 대한 교차 접근은 애플리케이션 미들웨어와 PostgreSQL RLS(Row-Level Security) 양쪽에서 차단된다.

---

## 3. 기능 명세 (Functional Specifications)

- **핵심 기능 1**: 소상공인 표준 계정과목표(`ledger_accounts`, Chart of Accounts): 자산(101 현금, 102 보통예금, 108 외상매출금/카드미수금, 109 배달플랫폼미수금, 135 부가세대급금, 146 상품/원재료), 부채(251 외상매입금, 253 미지급금, 255 부가세예수금, 257 예수금/원천세), 자본(301 사업주자본금, 302 사업주인출금), 수익(401 카드매출, 402 현금매출, 403 배달매출, 404 기타매출), 비용(501 매출원가, 801 급여, 811 임차료, 831 지급수수료-카드, 832 지급수수료-배달, 833 광고선전비 등)
- **핵심 기능 2**: 거래 단계별 자동 분개(Auto-Journalizing): (1) 카드 승인/매입 시: `[차] 카드미수금 1,094,500 + [차] 지급수수료-카드 5,500 / [대] 카드매출 1,000,000 + [대] 부가세예수금 100,000` → (2) 은행 실제 입금 및 대사 완료 시: `[차] 보통예금 1,094,500 / [대] 카드미수금 1,094,500`
- **핵심 기능 3**: 원본 보존 및 역분개 정정(Immutable Corrections): 이미 기장된 거래를 수정할 때 `UPDATE/DELETE`로 덮어쓰지 않고, 기존 분개를 상계하는 역분개(Reversing Entry)와 새로운 정정 분개(Correcting Entry)를 `corrections` 테이블에 사유·수정자·시각과 함께 연결 기록
- **핵심 기능 4**: 회계기간 마감(`accounting_periods`): 월별/분기별 세무 신고 완료 후 해당 기간을 `LOCKED` 처리하여 과거 장부가 실수로 변경되는 것을 방지(수정 필요 시 차기 조정 분개 생성)
- **핵심 기능 5**: 다중 소스 중복 제거(Deduplication): 동일 거래가 POS(주문), VAN(승인), 여신협회(매입), 은행(입금)에서 각각 들어올 때 하나의 `sales_transactions` 생명주기로 연결하고 중복 매출 산입을 원천 차단

---

## 4. 사용자 시나리오 (End-to-End User Scenario)

> **실전 시나리오**: 사장님이 수기로 잘못 입력한 50,000원 현금매출을 500,000원으로 수정하면, 시스템은 기존 50,000원 분개를 삭제하지 않고 `-50,000원` 역분개와 `+500,000원` 신규 분개를 생성하여 `corrections` 테이블에 수정 전후 스냅샷과 수정 사유를 남긴다.

1. **Pre-condition**: 사용자가 정상 로그인되어 있고 해당 매장에 대해 역할 권한(`RBAC/ABAC`)을 보유하고 있다. 외부 연동 기능의 경우 `provider_catalog` 및 `data_connections` 상태가 확인된다.
2. **Normal Flow**: 요청이 API Gateway를 통과하여 도메인 서비스에서 처리되고, 원본 출처(`data_source_level`)와 함께 PostgreSQL 트랜잭션 및 `outbox_events`에 기록된다.
3. **Fallback Flow (연동 차단/미계약 시)**: 만약 외부 API가 `CONTRACT_REQUIRED` 또는 `PARTNER_ONLY(BLOCKED)` 상태이면, 가짜 실시간 데이터를 표시하지 않고 즉시 **Level 2(공식 정산파일 CSV/Excel 업로드)** 또는 **Level 1(수기 입력)** 인터페이스로 안내하여 핵심 장부·대사 업무가 중단 없이 수행되도록 보장한다.

---

## 5. 백엔드 아키텍처 및 구현 명세 (Backend Architecture)

- **언어 및 프레임워크**: Go 1.22+, 모듈형 모놀리스 아키텍처 + 독립 비동기 워커(Go Workers)
- **핵심 구현 모듈**: `internal/ledger` 및 `internal/reconciliation` 패키지: 복식부기 밸런스 검증기(`SUM(debit) == SUM(credit)` 강제), 원장 락(Advisory Lock per `store_id`), 3-Way Matching 엔진(POS/주문 ↔ 카드/배달 정산 ↔ 은행 입금)
- **트랜잭션 관리**: 모든 상태 변경과 원장 반영은 `pgx/v5` 트랜잭션 내부에서 도메인 데이터 변경 + `audit_logs` 기록 + `outbox_events` 발행을 원자적(Atomic)으로 커밋한다.

---

## 6. 프론트엔드 및 모바일 명세 (Frontend & Mobile)

- **구현 범위**: 간편 장부 뷰(일일 입출금/손익 요약) 및 전문가용 복식원장/계정별원장/시산표(Trial Balance) 토글 뷰, 거래 수정 이력(Diff) 타임라인 모달
- **상태 표현 표준**:
  1. `LOADING`: 스켈레톤 UI 및 진행률 표시
  2. `SUCCESS`: 데이터 표시 + 우측 상단 `DataProvenanceBadge`(데이터 출처 Level 1~4 및 최종 갱신 시각)
  3. `EMPTY`: 초기 온보딩 안내 및 첫 데이터 등록/업로드 CTA 버튼
  4. `ERROR`: 사용자 친화적 오류 메시지 + `request_id` 복사 + 재시도 버튼
  5. `BLOCKED`: 외부 기관 계약/승인 대기 상태의 정직한 고지 + 파일 업로드 대체 경로 버튼

---

## 7. 데이터베이스 설계 연동 (Database Schema Mapping)

- **연관 테이블**: `ledger_accounts`, `journal_entries`, `journal_lines`, `corrections`, `accounting_periods`, `reconciliation_matches`, `reconciliation_disputes`
- **데이터 무결성 규칙**:
  - 모든 금액 컬럼은 `NUMERIC(19,4)`를 사용하여 부동소수점 반올림 오차를 원천 배제한다.
  - 모든 시각 컬럼은 `TIMESTAMPTZ`로 저장한다.
  - 원본 거래 레코드는 임의 삭제(`DELETE`)하지 않으며, 정정 시 `corrections` 및 역분개(`journal_entries`)로 감사 추적성을 유지한다.

---

## 8. API 계약 명세 (OpenAPI 3.1 Endpoints)

- **주요 엔드포인트**: `GET /v1/stores/{storeId}/ledger/entries`, `POST /v1/stores/{storeId}/ledger/manual-entry`, `POST /v1/stores/{storeId}/ledger/entries/{entryId}/correct`, `GET /v1/stores/{storeId}/ledger/trial-balance`, `POST /v1/stores/{storeId}/accounting-periods/{periodId}/lock`
- **공통 헤더**:
  - `Authorization: Bearer <OIDC_JWT>`
  - `X-Business-Id: <UUID>`
  - `X-Store-Id: <UUID>`
  - `Idempotency-Key: <UUID>` (모든 `POST`/`PUT`/`PATCH` 상태 변경 요청 시 필수)

---

## 9. 도메인 이벤트 (Domain Events)

- **발행/구독 이벤트**: `ledger.entry.posted`, `ledger.entry.reversed`, `ledger.period.locked`, `reconciliation.matched`
- **전달 보장**: Transactional Outbox 테이블(`outbox_events`)에서 Redis Streams / Worker Queue로 At-Least-Once 전달을 보장하며, 컨슈머는 `event_id` 기준 멱등 처리(Idempotent Consumer)를 수행한다.

---

## 10. 데이터 출처 및 신뢰도 등급 (Data Provenance)

- **데이터 원천**: 정규화된 매출/매입/정산/입출금 이벤트 및 수기 조정 입력
- **출처 태깅 규칙**:
  - `LEVEL_1_MANUAL`: 사용자 직접 입력 (신뢰도: 사용자 책임, 즉시 반영)
  - `LEVEL_2_FILE`: 공식 정산 엑셀/CSV 업로드 (신뢰도: 파일 해시 및 포맷 검증됨)
  - `LEVEL_3_API`: 공식 계약/승인된 외부 API (신뢰도: 기관 전자서명/TLS 검증됨)
  - `LEVEL_4_RECONCILED`: 복수 데이터 소스(예: 카드매입 + 은행입금) 교차 대사 완료 (신뢰도: 최고 등급 확정)

---

## 11. 오류 처리 및 장애 복구 (Error Handling & Resilience)

- **예외 처리 정책**: 차변과 대변의 합계가 0.0001 단위라도 불일치하면 DB 트랜잭션 전체를 즉시 Rollback하고 `LEDGER_UNBALANCED_ENTRY` 치명적 경보 발생
- **표준 에러 코드**: `AUTH_FORBIDDEN_STORE`, `CONNECTOR_EXTERNAL_BLOCKED`, `LEDGER_UNBALANCED_ENTRY`, `IMPORT_DUPLICATE_BATCH`, `RATE_LIMIT_EXCEEDED`, `UPSTREAM_TIMEOUT_RETRYABLE`

---

## 12. 개인정보 및 보안 통제 (Privacy & Security Controls)

- **보안 설계**: 원장 테이블(`journal_entries`, `journal_lines`)에 대해서는 애플리케이션 DB 계정에 `DELETE` 권한을 아예 부여하지 않고 `INSERT` 전용(상태 플래그만 갱신)으로 운영.
- **감사 추적**: 모든 민감 데이터 조회 및 변경은 `audit_logs`에 요청자 ID, 대상 매장 ID, 액션 코드, IP, User-Agent, 변경 전후 요약 및 해시 체인과 함께 기록된다.

---

## 13. 외부 승인 및 법적 선행 요건 (External Approval & Regulatory Requirements)

- **승인·규제 요건**: 한국 채택 기업회계기준(K-GAAP / 중소기업회계기준) 및 소득세법 간편장부/복식부기 의무자 기재 요건 준수.
- **차단(Blocked) 정책**: 필수 외부 계약이나 인허가가 완료되기 전까지 해당 라이브 연동 기능은 `feature_flags` 및 `provider_catalog`에서 `BLOCKED`로 분류하며, 가짜 API 응답으로 상용 완료를 가장하는 행위를 엄격히 금지한다.

---

## 14. 성능 및 확장성 목표 (Performance & SLA Targets)

- **목표 지표**: 건별 분개 생성 및 밸런스 검증 < 15ms, 월간 5만 건 시산표 집계 < 250ms.

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

- **직접 매핑 범위**: `WO-MOS-0251 ~ WO-MOS-0350`
- **상세 개별 명세 위치**: `11_WORK_ORDERS_0001_TO_1000/individual/` 내 해당 번호 파일 및 `11_WORK_ORDERS_0001_TO_1000/WORK_ORDER_INDEX.csv` 참조.
