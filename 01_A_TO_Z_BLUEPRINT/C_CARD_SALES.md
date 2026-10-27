# C — Card Sales (카드 승인·매입·수수료·정산·입금·누락 진단)

**문서 ID**: AZ-C_CARD_SALES  
**프로젝트 코드명**: BN Merchant OS (`BN-MERCHANT-OS-GEMINI4-ARGON-MASTER-BLUEPRINT-1000-001`)  
**관련 Work Order 범위**: `WO-MOS-0201 ~ WO-MOS-0265`  
**설계 원칙**: 제0조~제32조 마스터 명령 전수 준수 (독립 플랫폼 원칙, Level 1~4 데이터 수집 전략, 미승인 외부 API 허위 구현 금지)

---

## 1. 목표 (Objective)

제7조 및 제8조에 따라 카드 거래의 6단계 생명주기(카드 승인 → 카드 매입 → 수수료 계산 → 지급 예정 → 실제 입금 → 은행 입금 대사)를 정밀 모델링한다. 승인액, 매입액, 순정산예정액, 실제 입금액, 회계상 매출액을 절대 하나의 숫자로 혼합하지 않는다.

---

## 2. 사용자 역할 (User Roles & Actors)

- **주요 액터**: 매장 대표, 회계 담당자, 정산/대사 자동화 워커, 세무대리인
- **권한 경계**: 모든 요청은 인증된 사용자(`user_id`)와 명시적으로 선택·검증된 사업자(`business_id`) 및 매장(`store_id`) 컨텍스트 하에서만 실행되며, 타 사업장 데이터에 대한 교차 접근은 애플리케이션 미들웨어와 PostgreSQL RLS(Row-Level Security) 양쪽에서 차단된다.

---

## 3. 기능 명세 (Functional Specifications)

- **핵심 기능 1**: 6단계 금액 분리 저장: `authorized_amount`(승인액), `supply_amount`(공급가액), `vat_amount`(부가세), `service_tip_amount`(봉사료), `captured_amount`(매입확정액), `fee_amount`(카드사 수수료: 기본수수료+포인트수수료-우대환급분), `net_settlement_scheduled_amount`(순정산예정액), `actual_bank_deposit_amount`(실제 은행 입금액)
- **핵심 기능 2**: 18대 예외 시나리오 완벽 처리: 승인 후 즉시취소, 매입 전 취소, 매입 후 취소(차기 정산액에서 상계 차감), 부분 취소, 중복 결제 탐지, 할부 거래(무이자/유이자 수수료 부담 주체 구분), 우대수수료율 변동 차액, 정산 지연, 지급 보류(카드사 리스크 심사/한도 초과), 부분 입금, N건 거래 합산 1건 은행 입금, 카드사별 지급 주기(D+2, D+3 영업일), 공휴일/대체공휴일 이연, 수집 중복 제거
- **핵심 기능 3**: 한국 9대 카드사(신한, KB국민, 삼성, 현대, 롯데, 우리, 하나, BC, NH농협) 및 해외브랜드(Visa, Mastercard, Amex, JCB, UnionPay) 가맹점 수수료율 테이블 및 반기별 영세·중소가맹점 우대수수료율 이력 관리
- **핵심 기능 4**: 미지급·누락 매출 진단 엔진: 매입 완료 후 정산 예정일(`expected_payout_date`)이 지났으나 은행 입금 내역과 매칭되지 않은 건을 탐지하되, 데이터 수집 지연(`SYNC_DELAY_SUSPECTED`)과 실제 미입금 후보(`UNPAID_CANDIDATE`), 확인된 지급보류(`PAYOUT_ON_HOLD`)를 엄격히 구분

---

## 4. 사용자 시나리오 (End-to-End User Scenario)

> **실전 시나리오**: 사장님이 10월 25일(금)에 신한카드로 1,100,000원(공급가 1,000,000원 + 부가세 100,000원)을 결제받았다. 영세가맹점 우대수수료 0.5%(5,500원)가 차감된 순정산예정액 1,094,500원이 주말을 건너뛴 D+2 영업일인 10월 29일(화) 입금 예정으로 잡힌다. 만약 10월 26일에 110,000원 부분취소가 발생하면 상계 후 985,050원으로 정산 예정액이 갱신되고, 10월 29일 실제 신한은행 계좌에 입금된 985,050원과 자동 대사되어 `RECONCILED_MATCHED`로 완료된다.

1. **Pre-condition**: 사용자가 정상 로그인되어 있고 해당 매장에 대해 역할 권한(`RBAC/ABAC`)을 보유하고 있다. 외부 연동 기능의 경우 `provider_catalog` 및 `data_connections` 상태가 확인된다.
2. **Normal Flow**: 요청이 API Gateway를 통과하여 도메인 서비스에서 처리되고, 원본 출처(`data_source_level`)와 함께 PostgreSQL 트랜잭션 및 `outbox_events`에 기록된다.
3. **Fallback Flow (연동 차단/미계약 시)**: 만약 외부 API가 `CONTRACT_REQUIRED` 또는 `PARTNER_ONLY(BLOCKED)` 상태이면, 가짜 실시간 데이터를 표시하지 않고 즉시 **Level 2(공식 정산파일 CSV/Excel 업로드)** 또는 **Level 1(수기 입력)** 인터페이스로 안내하여 핵심 장부·대사 업무가 중단 없이 수행되도록 보장한다.

---

## 5. 백엔드 아키텍처 및 구현 명세 (Backend Architecture)

- **언어 및 프레임워크**: Go 1.22+, 모듈형 모놀리스 아키텍처 + 독립 비동기 워커(Go Workers)
- **핵심 구현 모듈**: `internal/revenue/cardsales` 패키지: 한국 영업일/공휴일 캘린더 계산기, 카드사별 정산주기 룰 엔진, 매입후취소 상계(Offsetting) 계산기, 합산 입금 Subset-Sum/Window 매칭 엔진
- **트랜잭션 관리**: 모든 상태 변경과 원장 반영은 `pgx/v5` 트랜잭션 내부에서 도메인 데이터 변경 + `audit_logs` 기록 + `outbox_events` 발행을 원자적(Atomic)으로 커밋한다.

---

## 6. 프론트엔드 및 모바일 명세 (Frontend & Mobile)

- **구현 범위**: 카드매출 캘린더, 승인 vs 매입 vs 입금예정 vs 실입금 워터폴 비교 차트, 카드사별 수수료율 및 미입금 진단 배지, 미입금 후보 상세 모달 및 이의제기/수동매칭 UI
- **상태 표현 표준**:
  1. `LOADING`: 스켈레톤 UI 및 진행률 표시
  2. `SUCCESS`: 데이터 표시 + 우측 상단 `DataProvenanceBadge`(데이터 출처 Level 1~4 및 최종 갱신 시각)
  3. `EMPTY`: 초기 온보딩 안내 및 첫 데이터 등록/업로드 CTA 버튼
  4. `ERROR`: 사용자 친화적 오류 메시지 + `request_id` 복사 + 재시도 버튼
  5. `BLOCKED`: 외부 기관 계약/승인 대기 상태의 정직한 고지 + 파일 업로드 대체 경로 버튼

---

## 7. 데이터베이스 설계 연동 (Database Schema Mapping)

- **연관 테이블**: `payment_authorizations`, `payment_captures`, `payment_cancellations`, `acquisition_records`, `settlement_records`, `payout_schedules`, `reconciliation_matches`, `reconciliation_disputes`
- **데이터 무결성 규칙**:
  - 모든 금액 컬럼은 `NUMERIC(19,4)`를 사용하여 부동소수점 반올림 오차를 원천 배제한다.
  - 모든 시각 컬럼은 `TIMESTAMPTZ`로 저장한다.
  - 원본 거래 레코드는 임의 삭제(`DELETE`)하지 않으며, 정정 시 `corrections` 및 역분개(`journal_entries`)로 감사 추적성을 유지한다.

---

## 8. API 계약 명세 (OpenAPI 3.1 Endpoints)

- **주요 엔드포인트**: `GET /v1/stores/{storeId}/card-sales/summary`, `GET /v1/stores/{storeId}/card-sales/transactions`, `GET /v1/stores/{storeId}/settlements/calendar`, `GET /v1/stores/{storeId}/reconciliation/unpaid-diagnostics`, `POST /v1/stores/{storeId}/reconciliation/disputes`
- **공통 헤더**:
  - `Authorization: Bearer <OIDC_JWT>`
  - `X-Business-Id: <UUID>`
  - `X-Store-Id: <UUID>`
  - `Idempotency-Key: <UUID>` (모든 `POST`/`PUT`/`PATCH` 상태 변경 요청 시 필수)

---

## 9. 도메인 이벤트 (Domain Events)

- **발행/구독 이벤트**: `card.authorization.ingested`, `card.capture.confirmed`, `card.settlement.scheduled`, `card.reconciliation.mismatch_detected`
- **전달 보장**: Transactional Outbox 테이블(`outbox_events`)에서 Redis Streams / Worker Queue로 At-Least-Once 전달을 보장하며, 컨슈머는 `event_id` 기준 멱등 처리(Idempotent Consumer)를 수행한다.

---

## 10. 데이터 출처 및 신뢰도 등급 (Data Provenance)

- **데이터 원천**: Level 1 수기입력, Level 2 여신협회/카드사/VAN 엑셀 업로드, Level 3 승인된 VAN/PG/여신협회 API
- **출처 태깅 규칙**:
  - `LEVEL_1_MANUAL`: 사용자 직접 입력 (신뢰도: 사용자 책임, 즉시 반영)
  - `LEVEL_2_FILE`: 공식 정산 엑셀/CSV 업로드 (신뢰도: 파일 해시 및 포맷 검증됨)
  - `LEVEL_3_API`: 공식 계약/승인된 외부 API (신뢰도: 기관 전자서명/TLS 검증됨)
  - `LEVEL_4_RECONCILED`: 복수 데이터 소스(예: 카드매입 + 은행입금) 교차 대사 완료 (신뢰도: 최고 등급 확정)

---

## 11. 오류 처리 및 장애 복구 (Error Handling & Resilience)

- **예외 처리 정책**: 동일 승인번호(`approval_no` + `trade_date` + `card_Acquire_code` + `amount`)가 POS와 여신협회에서 중복 유입될 경우 정규화 해시(`dedup_fingerprint`)로 병합하고 원본 레코드는 각각 `raw_import_batches`에 보존
- **표준 에러 코드**: `AUTH_FORBIDDEN_STORE`, `CONNECTOR_EXTERNAL_BLOCKED`, `LEDGER_UNBALANCED_ENTRY`, `IMPORT_DUPLICATE_BATCH`, `RATE_LIMIT_EXCEEDED`, `UPSTREAM_TIMEOUT_RETRYABLE`

---

## 12. 개인정보 및 보안 통제 (Privacy & Security Controls)

- **보안 설계**: 카드번호(PAN)는 반드시 마스킹된 형태(`1234-56**-****-7890`)만 저장하며 CVV/유효기간은 절대 수집·저장하지 않음(PCI-DSS 범위 최소화).
- **감사 추적**: 모든 민감 데이터 조회 및 변경은 `audit_logs`에 요청자 ID, 대상 매장 ID, 액션 코드, IP, User-Agent, 변경 전후 요약 및 해시 체인과 함께 기록된다.

---

## 13. 외부 승인 및 법적 선행 요건 (External Approval & Regulatory Requirements)

- **승인·규제 요건**: 여신협회 Open API 직접 연동은 `CONTRACT_REQUIRED`. 미계약 시 Level 2 파일 업로드 및 승인된 PG/VAN 연동으로 동작.
- **차단(Blocked) 정책**: 필수 외부 계약이나 인허가가 완료되기 전까지 해당 라이브 연동 기능은 `feature_flags` 및 `provider_catalog`에서 `BLOCKED`로 분류하며, 가짜 API 응답으로 상용 완료를 가장하는 행위를 엄격히 금지한다.

---

## 14. 성능 및 확장성 목표 (Performance & SLA Targets)

- **목표 지표**: 일 10만 건 매장 카드 전표 대사 배치 < 10초, 월별 카드매출 캘린더 조회 P95 < 120ms.

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

- **직접 매핑 범위**: `WO-MOS-0201 ~ WO-MOS-0265`
- **상세 개별 명세 위치**: `11_WORK_ORDERS_0001_TO_1000/individual/` 내 해당 번호 파일 및 `11_WORK_ORDERS_0001_TO_1000/WORK_ORDER_INDEX.csv` 참조.
