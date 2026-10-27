# E — Expense Management (비용·경비·영수증·거래처·예산 관리)

**문서 ID**: AZ-E_EXPENSES  
**프로젝트 코드명**: BN Merchant OS (`BN-MERCHANT-OS-GEMINI4-ARGON-MASTER-BLUEPRINT-1000-001`)  
**관련 Work Order 범위**: `WO-MOS-0301 ~ WO-MOS-0330`  
**설계 원칙**: 제0조~제32조 마스터 명령 전수 준수 (독립 플랫폼 원칙, Level 1~4 데이터 수집 전략, 미승인 외부 API 허위 구현 금지)

---

## 1. 목표 (Objective)

제9조에 따라 고정비(임차료, 정직원 인건비, 보험료, 정기 구독료)와 변동비(식자재/원재료비, 배달 수수료, 카드 결제 수수료, 포장비, 일용직 알바비, 광고선전비, 수도광열/공과금)를 체계적으로 분류·추적하고 증빙(세금계산서, 현금영수증, 법인/사업자카드 전표, 종이영수증 OCR)과 1:1 또는 N:M으로 연결한다.

---

## 2. 사용자 역할 (User Roles & Actors)

- **주요 액터**: 매장 대표, 매장 매니저(현장 영수증 등록), 회계 담당자, 세무대리인
- **권한 경계**: 모든 요청은 인증된 사용자(`user_id`)와 명시적으로 선택·검증된 사업자(`business_id`) 및 매장(`store_id`) 컨텍스트 하에서만 실행되며, 타 사업장 데이터에 대한 교차 접근은 애플리케이션 미들웨어와 PostgreSQL RLS(Row-Level Security) 양쪽에서 차단된다.

---

## 3. 기능 명세 (Functional Specifications)

- **핵심 기능 1**: 표준 소상공인 비용 계정과목(`expense_categories`) 트리: 식자재/원재료, 상품매입, 인건비(정직원/파트타임/4대보험), 임차료/관리비, 배달대행/플랫폼수수료, 카드/PG결제수수료, 광고선전비, 공과금(전기/가스/수도/통신), 소모품/포장재, 차량유지비, 접대비, 수선유지비, 기타경비
- **핵심 기능 2**: 거래처(`vendors`) 관리 및 정기 지출(임차료, 정수기 렌탈, 기장료 등) 자동 반복 알림/예정 생성
- **핵심 기능 3**: 모바일 영수증 촬영 및 OCR 파이프라인: 사업자번호, 거래일시, 합계금액, 공급가액, 부가세, 가맹점명 추출 및 신뢰도(`ocr_confidence`) 표시, 중복 영수증 등록 차단
- **핵심 기능 4**: 월별 카테고리별 지출 예산(`expense_budgets`) 설정 및 예산 대비 80%/100% 도달 시 이상 지출 알림
- **핵심 기능 5**: 세무상 적격증빙(세금계산서, 계산서, 신용카드매출전표, 현금영수증) 여부 자동 판별 및 증빙 미비 지출 필터링

---

## 4. 사용자 시나리오 (End-to-End User Scenario)

> **실전 시나리오**: 사장님이 식자재 마트에서 받은 종이 간이영수증을 모바일 앱으로 촬영하면 OCR이 금액 85,000원을 읽어 '식자재/원재료'로 분류하고, 동시에 '적격증빙 미수취(3만 원 초과 간이영수증 가산세 유의)' 안내 배지를 표시한다.

1. **Pre-condition**: 사용자가 정상 로그인되어 있고 해당 매장에 대해 역할 권한(`RBAC/ABAC`)을 보유하고 있다. 외부 연동 기능의 경우 `provider_catalog` 및 `data_connections` 상태가 확인된다.
2. **Normal Flow**: 요청이 API Gateway를 통과하여 도메인 서비스에서 처리되고, 원본 출처(`data_source_level`)와 함께 PostgreSQL 트랜잭션 및 `outbox_events`에 기록된다.
3. **Fallback Flow (연동 차단/미계약 시)**: 만약 외부 API가 `CONTRACT_REQUIRED` 또는 `PARTNER_ONLY(BLOCKED)` 상태이면, 가짜 실시간 데이터를 표시하지 않고 즉시 **Level 2(공식 정산파일 CSV/Excel 업로드)** 또는 **Level 1(수기 입력)** 인터페이스로 안내하여 핵심 장부·대사 업무가 중단 없이 수행되도록 보장한다.

---

## 5. 백엔드 아키텍처 및 구현 명세 (Backend Architecture)

- **언어 및 프레임워크**: Go 1.22+, 모듈형 모놀리스 아키텍처 + 독립 비동기 워커(Go Workers)
- **핵심 구현 모듈**: `internal/expense` 서비스: 지출 등록/수정/정정이력 관리, 거래처 사업자번호 기반 자동 카테고리 추천 엔진, 예산 소진율 계산기
- **트랜잭션 관리**: 모든 상태 변경과 원장 반영은 `pgx/v5` 트랜잭션 내부에서 도메인 데이터 변경 + `audit_logs` 기록 + `outbox_events` 발행을 원자적(Atomic)으로 커밋한다.

---

## 6. 프론트엔드 및 모바일 명세 (Frontend & Mobile)

- **구현 범위**: 지출 내역 리스트, 고정비 vs 변동비 도넛 차트, 영수증 스캔 카메라 뷰 및 크롭 UI, 거래처별 미지급금(AP) 현황판
- **상태 표현 표준**:
  1. `LOADING`: 스켈레톤 UI 및 진행률 표시
  2. `SUCCESS`: 데이터 표시 + 우측 상단 `DataProvenanceBadge`(데이터 출처 Level 1~4 및 최종 갱신 시각)
  3. `EMPTY`: 초기 온보딩 안내 및 첫 데이터 등록/업로드 CTA 버튼
  4. `ERROR`: 사용자 친화적 오류 메시지 + `request_id` 복사 + 재시도 버튼
  5. `BLOCKED`: 외부 기관 계약/승인 대기 상태의 정직한 고지 + 파일 업로드 대체 경로 버튼

---

## 7. 데이터베이스 설계 연동 (Database Schema Mapping)

- **연관 테이블**: `expenses`, `expense_categories`, `vendors`, `tax_documents`
- **데이터 무결성 규칙**:
  - 모든 금액 컬럼은 `NUMERIC(19,4)`를 사용하여 부동소수점 반올림 오차를 원천 배제한다.
  - 모든 시각 컬럼은 `TIMESTAMPTZ`로 저장한다.
  - 원본 거래 레코드는 임의 삭제(`DELETE`)하지 않으며, 정정 시 `corrections` 및 역분개(`journal_entries`)로 감사 추적성을 유지한다.

---

## 8. API 계약 명세 (OpenAPI 3.1 Endpoints)

- **주요 엔드포인트**: `POST /v1/stores/{storeId}/expenses`, `GET /v1/stores/{storeId}/expenses`, `PATCH /v1/stores/{storeId}/expenses/{expenseId}`, `POST /v1/stores/{storeId}/vendors`, `POST /v1/stores/{storeId}/expenses/ocr`
- **공통 헤더**:
  - `Authorization: Bearer <OIDC_JWT>`
  - `X-Business-Id: <UUID>`
  - `X-Store-Id: <UUID>`
  - `Idempotency-Key: <UUID>` (모든 `POST`/`PUT`/`PATCH` 상태 변경 요청 시 필수)

---

## 9. 도메인 이벤트 (Domain Events)

- **발행/구독 이벤트**: `expense.recorded`, `expense.corrected`, `expense.budget_threshold_exceeded`
- **전달 보장**: Transactional Outbox 테이블(`outbox_events`)에서 Redis Streams / Worker Queue로 At-Least-Once 전달을 보장하며, 컨슈머는 `event_id` 기준 멱등 처리(Idempotent Consumer)를 수행한다.

---

## 10. 데이터 출처 및 신뢰도 등급 (Data Provenance)

- **데이터 원천**: Level 1 수기/영수증 촬영, Level 2 카드사 이용내역/은행 출금내역 엑셀 업로드, Level 3 사업자카드/오픈뱅킹 출금/홈택스 매입 세금계산서·현금영수증 연동
- **출처 태깅 규칙**:
  - `LEVEL_1_MANUAL`: 사용자 직접 입력 (신뢰도: 사용자 책임, 즉시 반영)
  - `LEVEL_2_FILE`: 공식 정산 엑셀/CSV 업로드 (신뢰도: 파일 해시 및 포맷 검증됨)
  - `LEVEL_3_API`: 공식 계약/승인된 외부 API (신뢰도: 기관 전자서명/TLS 검증됨)
  - `LEVEL_4_RECONCILED`: 복수 데이터 소스(예: 카드매입 + 은행입금) 교차 대사 완료 (신뢰도: 최고 등급 확정)

---

## 11. 오류 처리 및 장애 복구 (Error Handling & Resilience)

- **예외 처리 정책**: 은행 계좌 이체 출금 내역과 전자세금계산서 매입 내역이 동일한 거래처 대금 지급인 경우 중복 비용 산입을 방지하기 위해 '매입 발생(발생주의)'과 '대금 지급(현금주의)'을 분리 매칭
- **표준 에러 코드**: `AUTH_FORBIDDEN_STORE`, `CONNECTOR_EXTERNAL_BLOCKED`, `LEDGER_UNBALANCED_ENTRY`, `IMPORT_DUPLICATE_BATCH`, `RATE_LIMIT_EXCEEDED`, `UPSTREAM_TIMEOUT_RETRYABLE`

---

## 12. 개인정보 및 보안 통제 (Privacy & Security Controls)

- **보안 설계**: 업로드된 영수증 원본 이미지는 S3 Pre-signed URL(유효시간 5분)로만 열람 가능하며 타 매장 접근 불가.
- **감사 추적**: 모든 민감 데이터 조회 및 변경은 `audit_logs`에 요청자 ID, 대상 매장 ID, 액션 코드, IP, User-Agent, 변경 전후 요약 및 해시 체인과 함께 기록된다.

---

## 13. 외부 승인 및 법적 선행 요건 (External Approval & Regulatory Requirements)

- **승인·규제 요건**: 외부 OCR API 사용 시 개인정보 처리위탁 고지 또는 자체 On-Premise/컨테이너 OCR 엔진(PaddleOCR/Tesseract) 기본 사용.
- **차단(Blocked) 정책**: 필수 외부 계약이나 인허가가 완료되기 전까지 해당 라이브 연동 기능은 `feature_flags` 및 `provider_catalog`에서 `BLOCKED`로 분류하며, 가짜 API 응답으로 상용 완료를 가장하는 행위를 엄격히 금지한다.

---

## 14. 성능 및 확장성 목표 (Performance & SLA Targets)

- **목표 지표**: 영수증 OCR 추출 응답 < 2.5초, 월간 지출 집계 조회 < 100ms.

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

- **직접 매핑 범위**: `WO-MOS-0301 ~ WO-MOS-0330`
- **상세 개별 명세 위치**: `11_WORK_ORDERS_0001_TO_1000/individual/` 내 해당 번호 파일 및 `11_WORK_ORDERS_0001_TO_1000/WORK_ORDER_INDEX.csv` 참조.
