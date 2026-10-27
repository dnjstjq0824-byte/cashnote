# V — Vertical Store Operations (음식점·카페·도소매·서비스업별 매장·상품·재고·직원 관리)

**문서 ID**: AZ-V_VERTICAL_OPERATIONS  
**프로젝트 코드명**: BN Merchant OS (`BN-MERCHANT-OS-GEMINI4-ARGON-MASTER-BLUEPRINT-1000-001`)  
**관련 Work Order 범위**: `WO-MOS-0425 ~ WO-MOS-0450, WO-MOS-0975 ~ WO-MOS-0985`  
**설계 원칙**: 제0조~제32조 마스터 명령 전수 준수 (독립 플랫폼 원칙, Level 1~4 데이터 수집 전략, 미승인 외부 API 허위 구현 금지)

---

## 1. 목표 (Objective)

제5조 V항 및 제19조 Store Operations 도메인에 따라 업종별(음식점/주점, 카페/베이커리, 도소매/편집숍, 뷰티/네일/서비스업)로 상이한 매장 현장 운영 요구사항(메뉴·레시피 원가, 바코드 재고 수불, 시술/예약 객단가, 정직원·파트타임 근무 및 인건비 추정)을 모듈형으로 설계한다.

---

## 2. 사용자 역할 (User Roles & Actors)

- **주요 액터**: 음식점/카페/도소매/서비스업 사장님, 매장 매니저, 주방/재고 담당자
- **권한 경계**: 모든 요청은 인증된 사용자(`user_id`)와 명시적으로 선택·검증된 사업자(`business_id`) 및 매장(`store_id`) 컨텍스트 하에서만 실행되며, 타 사업장 데이터에 대한 교차 접근은 애플리케이션 미들웨어와 PostgreSQL RLS(Row-Level Security) 양쪽에서 차단된다.

---

## 3. 기능 명세 (Functional Specifications)

- **핵심 기능 1**: 업종별 운영 프로파일(`stores.vertical_type`): `FNB_RESTAURANT`(음식점: 배달비중·식자재원가율·테이블회전율 중심), `FNB_CAFE_BAKERY`(카페/베이커리: 시간대별 피크·원두/우유 소모량·스탬프 단골 중심), `RETAIL_COMMERCE`(도소매: 바코드 SKU·재고수불부·안전재고·온오프라인 재고동기화 중심), `SERVICE_BEAUTY_ACADEMY`(서비스/뷰티/운동: 예약/노쇼율·담당직원별 실적·정액권/선불권 선수금 차감 중심)
- **핵심 기능 2**: 상품·메뉴 마스터(`products`) 및 주문(`orders`) 정규화: POS/배달앱마다 다르게 찍히는 메뉴명(예: `'아이스아메리카노(L)'` vs `'아아 라지'`)을 단일 표준 SKU로 매핑하는 메뉴 별칭(Alias) 사전
- **핵심 기능 3**: 재고 수불(`inventory_items`, `inventory_movements`): 입고(매입/B2B 마켓 주문 수령), 판매 차감(POS/배달 주문 연동 또는 일마감 차감), 폐기/로스(Loss), 매장 간 재고 이동 기록 및 안전재고(`reorder_point`) 미달 시 알림 + B2B 마켓 재주문 연결
- **핵심 기능 4**: 직원 및 근무 관리(`employees`): 정직원·단기/일용직(알바) 등록, 근로계약 형태, 시급/월급, 주휴수당 조건 계산 보조, 근무 스케줄 및 예상 인건비율 자동 반영

---

## 4. 사용자 시나리오 (End-to-End User Scenario)

> **실전 시나리오**: 치킨집 사장님이 생닭 안전재고를 50마리로 설정해 두었는데 금요일 저녁 POS/배달 주문으로 45마리가 판매되어 잔여 재고가 30마리로 떨어지면, 앱에 '생닭 재고 부족 예상(잔여 30마리)' 알림이 뜨고 거래처 발주서 또는 B2B 마켓 주문 화면으로 바로 연결된다.

1. **Pre-condition**: 사용자가 정상 로그인되어 있고 해당 매장에 대해 역할 권한(`RBAC/ABAC`)을 보유하고 있다. 외부 연동 기능의 경우 `provider_catalog` 및 `data_connections` 상태가 확인된다.
2. **Normal Flow**: 요청이 API Gateway를 통과하여 도메인 서비스에서 처리되고, 원본 출처(`data_source_level`)와 함께 PostgreSQL 트랜잭션 및 `outbox_events`에 기록된다.
3. **Fallback Flow (연동 차단/미계약 시)**: 만약 외부 API가 `CONTRACT_REQUIRED` 또는 `PARTNER_ONLY(BLOCKED)` 상태이면, 가짜 실시간 데이터를 표시하지 않고 즉시 **Level 2(공식 정산파일 CSV/Excel 업로드)** 또는 **Level 1(수기 입력)** 인터페이스로 안내하여 핵심 장부·대사 업무가 중단 없이 수행되도록 보장한다.

---

## 5. 백엔드 아키텍처 및 구현 명세 (Backend Architecture)

- **언어 및 프레임워크**: Go 1.22+, 모듈형 모놀리스 아키텍처 + 독립 비동기 워커(Go Workers)
- **핵심 구현 모듈**: `internal/operations` 패키지: SKU 별칭 매칭기, 재고 수불 원장(FIFO/이동평균 원가 계산기), 주휴수당·인건비 시뮬레이터, 선불권(선수금 부채 → 시술 시 매출 전환) 회계 브릿지
- **트랜잭션 관리**: 모든 상태 변경과 원장 반영은 `pgx/v5` 트랜잭션 내부에서 도메인 데이터 변경 + `audit_logs` 기록 + `outbox_events` 발행을 원자적(Atomic)으로 커밋한다.

---

## 6. 프론트엔드 및 모바일 명세 (Frontend & Mobile)

- **구현 범위**: 업종별 맞춤형 홈 위젯, 상품/메뉴 및 원가율 관리 표, 재고 실사(모바일 바코드 스캔 지원) 및 입출고 내역 화면, 직원 명부 및 월 예상 인건비 캘린더
- **상태 표현 표준**:
  1. `LOADING`: 스켈레톤 UI 및 진행률 표시
  2. `SUCCESS`: 데이터 표시 + 우측 상단 `DataProvenanceBadge`(데이터 출처 Level 1~4 및 최종 갱신 시각)
  3. `EMPTY`: 초기 온보딩 안내 및 첫 데이터 등록/업로드 CTA 버튼
  4. `ERROR`: 사용자 친화적 오류 메시지 + `request_id` 복사 + 재시도 버튼
  5. `BLOCKED`: 외부 기관 계약/승인 대기 상태의 정직한 고지 + 파일 업로드 대체 경로 버튼

---

## 7. 데이터베이스 설계 연동 (Database Schema Mapping)

- **연관 테이블**: `products`, `orders`, `inventory_items`, `inventory_movements`, `employees`, `store_metrics`
- **데이터 무결성 규칙**:
  - 모든 금액 컬럼은 `NUMERIC(19,4)`를 사용하여 부동소수점 반올림 오차를 원천 배제한다.
  - 모든 시각 컬럼은 `TIMESTAMPTZ`로 저장한다.
  - 원본 거래 레코드는 임의 삭제(`DELETE`)하지 않으며, 정정 시 `corrections` 및 역분개(`journal_entries`)로 감사 추적성을 유지한다.

---

## 8. API 계약 명세 (OpenAPI 3.1 Endpoints)

- **주요 엔드포인트**: `GET /v1/stores/{storeId}/products`, `POST /v1/stores/{storeId}/inventory/movements`, `GET /v1/stores/{storeId}/inventory/alerts`, `GET /v1/stores/{storeId}/employees`, `POST /v1/stores/{storeId}/employees`
- **공통 헤더**:
  - `Authorization: Bearer <OIDC_JWT>`
  - `X-Business-Id: <UUID>`
  - `X-Store-Id: <UUID>`
  - `Idempotency-Key: <UUID>` (모든 `POST`/`PUT`/`PATCH` 상태 변경 요청 시 필수)

---

## 9. 도메인 이벤트 (Domain Events)

- **발행/구독 이벤트**: `inventory.low_stock.alert`, `inventory.movement.recorded`, `employee.payroll_estimate.updated`
- **전달 보장**: Transactional Outbox 테이블(`outbox_events`)에서 Redis Streams / Worker Queue로 At-Least-Once 전달을 보장하며, 컨슈머는 `event_id` 기준 멱등 처리(Idempotent Consumer)를 수행한다.

---

## 10. 데이터 출처 및 신뢰도 등급 (Data Provenance)

- **데이터 원천**: POS/배달 주문 데이터, B2B 마켓 구매 내역, 사장님/매니저 재고 실사 입력
- **출처 태깅 규칙**:
  - `LEVEL_1_MANUAL`: 사용자 직접 입력 (신뢰도: 사용자 책임, 즉시 반영)
  - `LEVEL_2_FILE`: 공식 정산 엑셀/CSV 업로드 (신뢰도: 파일 해시 및 포맷 검증됨)
  - `LEVEL_3_API`: 공식 계약/승인된 외부 API (신뢰도: 기관 전자서명/TLS 검증됨)
  - `LEVEL_4_RECONCILED`: 복수 데이터 소스(예: 카드매입 + 은행입금) 교차 대사 완료 (신뢰도: 최고 등급 확정)

---

## 11. 오류 처리 및 장애 복구 (Error Handling & Resilience)

- **예외 처리 정책**: 재고 차감 시 실사 누락으로 계산상 음수 재고(`negative_stock`)가 발생하더라도 주문 수집을 중단하지 않고 `STOCK_RECONCILIATION_NEEDED` 플래그를 띄워 마감 시 보정하도록 처리
- **표준 에러 코드**: `AUTH_FORBIDDEN_STORE`, `CONNECTOR_EXTERNAL_BLOCKED`, `LEDGER_UNBALANCED_ENTRY`, `IMPORT_DUPLICATE_BATCH`, `RATE_LIMIT_EXCEEDED`, `UPSTREAM_TIMEOUT_RETRYABLE`

---

## 12. 개인정보 및 보안 통제 (Privacy & Security Controls)

- **보안 설계**: 직원 정보(`employees`) 저장 시 급여 계좌번호는 암호화하며, 노무/세무 신고를 위한 주민등록번호 수집은 원천징수 신고 기능 사용 시에만 별도 암호화 볼트에 격리 저장.
- **감사 추적**: 모든 민감 데이터 조회 및 변경은 `audit_logs`에 요청자 ID, 대상 매장 ID, 액션 코드, IP, User-Agent, 변경 전후 요약 및 해시 체인과 함께 기록된다.

---

## 13. 외부 승인 및 법적 선행 요건 (External Approval & Regulatory Requirements)

- **승인·규제 요건**: 근로기준법 주휴수당 및 최저임금 기준 연도별 파라미터 테이블 유지(노무 자문 면책 고지 포함).
- **차단(Blocked) 정책**: 필수 외부 계약이나 인허가가 완료되기 전까지 해당 라이브 연동 기능은 `feature_flags` 및 `provider_catalog`에서 `BLOCKED`로 분류하며, 가짜 API 응답으로 상용 완료를 가장하는 행위를 엄격히 금지한다.

---

## 14. 성능 및 확장성 목표 (Performance & SLA Targets)

- **목표 지표**: 주문 1건당 BOM 재고 차감 처리 < 10ms.

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

- **직접 매핑 범위**: `WO-MOS-0425 ~ WO-MOS-0450, WO-MOS-0975 ~ WO-MOS-0985`
- **상세 개별 명세 위치**: `11_WORK_ORDERS_0001_TO_1000/individual/` 내 해당 번호 파일 및 `11_WORK_ORDERS_0001_TO_1000/WORK_ORDER_INDEX.csv` 참조.
