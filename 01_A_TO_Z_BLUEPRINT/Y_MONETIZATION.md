# Y — Yield & Monetization (유료 멤버십·금융/서비스 제휴·B2B 마켓플레이스·정산)

**문서 ID**: AZ-Y_MONETIZATION  
**프로젝트 코드명**: BN Merchant OS (`BN-MERCHANT-OS-GEMINI4-ARGON-MASTER-BLUEPRINT-1000-001`)  
**관련 Work Order 범위**: `WO-MOS-0751 ~ WO-MOS-0800`  
**설계 원칙**: 제0조~제32조 마스터 명령 전수 준수 (독립 플랫폼 원칙, Level 1~4 데이터 수집 전략, 미승인 외부 API 허위 구현 금지)

---

## 1. 목표 (Objective)

제15조, 제16조, 제19조 Marketplace & Subscription 도메인에 따라 SaaS 정기구독 결제 엔진, B2B 소상공인 전용 물품 마켓플레이스(식자재·포장재·비품), 그리고 적법한 범위 내의 제휴 서비스/금융상품 안내 모델을 설계한다.

---

## 2. 사용자 역할 (User Roles & Actors)

- **주요 액터**: 매장 대표(구매자/구독자), B2B 공급업체(`ROLE_SUPPLIER`), 정산 관리자, 마케팅/제휴 담당자
- **권한 경계**: 모든 요청은 인증된 사용자(`user_id`)와 명시적으로 선택·검증된 사업자(`business_id`) 및 매장(`store_id`) 컨텍스트 하에서만 실행되며, 타 사업장 데이터에 대한 교차 접근은 애플리케이션 미들웨어와 PostgreSQL RLS(Row-Level Security) 양쪽에서 차단된다.

---

## 3. 기능 명세 (Functional Specifications)

- **핵심 기능 1**: 유료 멤버십 구독 엔진(`memberships`, `subscriptions`, `payments`, `refunds`): PG 빌링키(정기결제) 발급, 플랜 업그레이드/다운그레이드 일할 계산(Proration), 결제 실패 시 D+1/D+3/D+5 재시도(Dunning Management) 및 유예기간(Grace Period) 후 무료 플랜 안전 전환(기존 데이터 열람 권한은 유지)
- **핵심 기능 2**: B2B 마켓플레이스(`suppliers`, `marketplace_products`, `marketplace_orders`): 사업자 인증(`VERIFIED_OWNER`) 완료 회원에게만 노출되는 도매 폐쇄몰 가격(`B2B Only Pricing`), 박스/묶음 단위 MOQ(최소주문수량), 정기배송 설정, 공급사 입점 심사 및 사업자 검증
- **핵심 기능 3**: B2B 마켓 주문→장부 자동 연동: 사장님이 BN Merchant OS B2B 마켓에서 식자재나 포장용기를 구매하면, 별도 입력 없이 자동으로 해당 매장의 `expenses`(비용 장부)와 `inventory_movements`(재고 입고), `tax_invoices`(매입 세금계산서)에 연결 반영
- **핵심 기능 4**: 공급사 정산 및 거래 분쟁(`marketplace_orders` 에스크로): 구매 확정(또는 배송 완료 후 7일 경과) 시 공급사 수수료를 차감한 정산 예정액을 생성하고, 파손/오배송 분쟁 접수 시 정산 보류 및 부분/전체 환불 처리
- **핵심 기능 5**: 제휴 서비스 및 금융상품 규제 준수 아키텍처: 정부 지원사업/정책자금(공공데이터 무료 안내), 비금융 제휴(세무/통신/POS/방역 신청 연결), 금융상품(대출/카드/보험 — 인허가 상태에 따라 단순 정보 안내와 정식 중개를 엄격 분리)

---

## 4. 사용자 시나리오 (End-to-End User Scenario)

> **실전 시나리오**: Plus 멤버십(월 16,500원)을 구독 중인 카페 사장님이 매월 지급되는 15,000원 마켓 쿠폰을 적용하여 B2B 마켓에서 테이크아웃 컵 1,000개와 원두 5kg을 주문하면, 배송 완료와 동시에 매입 세금계산서와 소모품/원재료 비용 장부에 자동 기장된다.

1. **Pre-condition**: 사용자가 정상 로그인되어 있고 해당 매장에 대해 역할 권한(`RBAC/ABAC`)을 보유하고 있다. 외부 연동 기능의 경우 `provider_catalog` 및 `data_connections` 상태가 확인된다.
2. **Normal Flow**: 요청이 API Gateway를 통과하여 도메인 서비스에서 처리되고, 원본 출처(`data_source_level`)와 함께 PostgreSQL 트랜잭션 및 `outbox_events`에 기록된다.
3. **Fallback Flow (연동 차단/미계약 시)**: 만약 외부 API가 `CONTRACT_REQUIRED` 또는 `PARTNER_ONLY(BLOCKED)` 상태이면, 가짜 실시간 데이터를 표시하지 않고 즉시 **Level 2(공식 정산파일 CSV/Excel 업로드)** 또는 **Level 1(수기 입력)** 인터페이스로 안내하여 핵심 장부·대사 업무가 중단 없이 수행되도록 보장한다.

---

## 5. 백엔드 아키텍처 및 구현 명세 (Backend Architecture)

- **언어 및 프레임워크**: Go 1.22+, 모듈형 모놀리스 아키텍처 + 독립 비동기 워커(Go Workers)
- **핵심 구현 모듈**: `internal/monetization` 및 `internal/marketplace` 패키지: 구독 상태 머신, 일할 계산기, B2B 주문/재고/장부 자동 브릿지, 공급사 정산 스케줄러, 규제 피처 플래그 검사기
- **트랜잭션 관리**: 모든 상태 변경과 원장 반영은 `pgx/v5` 트랜잭션 내부에서 도메인 데이터 변경 + `audit_logs` 기록 + `outbox_events` 발행을 원자적(Atomic)으로 커밋한다.

---

## 6. 프론트엔드 및 모바일 명세 (Frontend & Mobile)

- **구현 범위**: 멤버십 요금제 비교 및 구독 관리 화면, B2B 마켓 홈·카테고리·장바구니·주문내역 화면, 공급사 파트너 센터(상품 등록, 송장 입력, 정산 내역), 정책자금·제휴 혜택 탭
- **상태 표현 표준**:
  1. `LOADING`: 스켈레톤 UI 및 진행률 표시
  2. `SUCCESS`: 데이터 표시 + 우측 상단 `DataProvenanceBadge`(데이터 출처 Level 1~4 및 최종 갱신 시각)
  3. `EMPTY`: 초기 온보딩 안내 및 첫 데이터 등록/업로드 CTA 버튼
  4. `ERROR`: 사용자 친화적 오류 메시지 + `request_id` 복사 + 재시도 버튼
  5. `BLOCKED`: 외부 기관 계약/승인 대기 상태의 정직한 고지 + 파일 업로드 대체 경로 버튼

---

## 7. 데이터베이스 설계 연동 (Database Schema Mapping)

- **연관 테이블**: `marketplace_products`, `suppliers`, `marketplace_orders`, `memberships`, `subscriptions`, `payments`, `refunds`
- **데이터 무결성 규칙**:
  - 모든 금액 컬럼은 `NUMERIC(19,4)`를 사용하여 부동소수점 반올림 오차를 원천 배제한다.
  - 모든 시각 컬럼은 `TIMESTAMPTZ`로 저장한다.
  - 원본 거래 레코드는 임의 삭제(`DELETE`)하지 않으며, 정정 시 `corrections` 및 역분개(`journal_entries`)로 감사 추적성을 유지한다.

---

## 8. API 계약 명세 (OpenAPI 3.1 Endpoints)

- **주요 엔드포인트**: `GET /v1/subscriptions/plans`, `POST /v1/stores/{storeId}/subscriptions`, `POST /v1/stores/{storeId}/subscriptions/cancel`, `GET /v1/marketplace/products`, `POST /v1/stores/{storeId}/marketplace/orders`, `GET /v1/partner/suppliers/settlements`
- **공통 헤더**:
  - `Authorization: Bearer <OIDC_JWT>`
  - `X-Business-Id: <UUID>`
  - `X-Store-Id: <UUID>`
  - `Idempotency-Key: <UUID>` (모든 `POST`/`PUT`/`PATCH` 상태 변경 요청 시 필수)

---

## 9. 도메인 이벤트 (Domain Events)

- **발행/구독 이벤트**: `subscription.activated`, `subscription.payment_failed`, `subscription.canceled`, `marketplace.order.placed`, `marketplace.order.settled`
- **전달 보장**: Transactional Outbox 테이블(`outbox_events`)에서 Redis Streams / Worker Queue로 At-Least-Once 전달을 보장하며, 컨슈머는 `event_id` 기준 멱등 처리(Idempotent Consumer)를 수행한다.

---

## 10. 데이터 출처 및 신뢰도 등급 (Data Provenance)

- **데이터 원천**: PG사 정기/단건 결제 API, 택배 배송조회 API, 내부 장부/재고 연동 이벤트
- **출처 태깅 규칙**:
  - `LEVEL_1_MANUAL`: 사용자 직접 입력 (신뢰도: 사용자 책임, 즉시 반영)
  - `LEVEL_2_FILE`: 공식 정산 엑셀/CSV 업로드 (신뢰도: 파일 해시 및 포맷 검증됨)
  - `LEVEL_3_API`: 공식 계약/승인된 외부 API (신뢰도: 기관 전자서명/TLS 검증됨)
  - `LEVEL_4_RECONCILED`: 복수 데이터 소스(예: 카드매입 + 은행입금) 교차 대사 완료 (신뢰도: 최고 등급 확정)

---

## 11. 오류 처리 및 장애 복구 (Error Handling & Resilience)

- **예외 처리 정책**: 구독 결제 중 중복 클릭이나 네트워크 재시도로 이중 과금이 발생하지 않도록 `Idempotency-Key`(`store_id + plan_id + billing_cycle_start`)를 PG사 및 DB에 강제 적용
- **표준 에러 코드**: `AUTH_FORBIDDEN_STORE`, `CONNECTOR_EXTERNAL_BLOCKED`, `LEDGER_UNBALANCED_ENTRY`, `IMPORT_DUPLICATE_BATCH`, `RATE_LIMIT_EXCEEDED`, `UPSTREAM_TIMEOUT_RETRYABLE`

---

## 12. 개인정보 및 보안 통제 (Privacy & Security Controls)

- **보안 설계**: 사장님의 매출·정산 데이터를 동의 없이 B2B 공급사나 제휴 금융사에 절대 공유하지 않으며(제16조 및 제31조 준수), 맞춤형 추천에 데이터를 활용할 경우 별도의 선택적 목적 동의(`connection_consents`)가 있는 경우에만 실행.
- **감사 추적**: 모든 민감 데이터 조회 및 변경은 `audit_logs`에 요청자 ID, 대상 매장 ID, 액션 코드, IP, User-Agent, 변경 전후 요약 및 해시 체인과 함께 기록된다.

---

## 13. 외부 승인 및 법적 선행 요건 (External Approval & Regulatory Requirements)

- **승인·규제 요건**: 통신판매업/통신판매중개업 신고, 전자상거래법 청약철회/환불 규정, 금융소비자보호법(대출성 상품 중개업 미등록 시 대출 비교·모집 기능 `BLOCKED`) 엄격 준수.
- **차단(Blocked) 정책**: 필수 외부 계약이나 인허가가 완료되기 전까지 해당 라이브 연동 기능은 `feature_flags` 및 `provider_catalog`에서 `BLOCKED`로 분류하며, 가짜 API 응답으로 상용 완료를 가장하는 행위를 엄격히 금지한다.

---

## 14. 성능 및 확장성 목표 (Performance & SLA Targets)

- **목표 지표**: 구독 결제 및 권한 즉시 반영 < 500ms, B2B 마켓 상품 검색 < 100ms.

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

- **직접 매핑 범위**: `WO-MOS-0751 ~ WO-MOS-0800`
- **상세 개별 명세 위치**: `11_WORK_ORDERS_0001_TO_1000/individual/` 내 해당 번호 파일 및 `11_WORK_ORDERS_0001_TO_1000/WORK_ORDER_INDEX.csv` 참조.
