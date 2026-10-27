# R — Review & CRM (매장 리뷰·고객 방문·단골 분석·마케팅 및 동의 관리)

**문서 ID**: AZ-R_REVIEWS_CRM  
**프로젝트 코드명**: BN Merchant OS (`BN-MERCHANT-OS-GEMINI4-ARGON-MASTER-BLUEPRINT-1000-001`)  
**관련 Work Order 범위**: `WO-MOS-0601 ~ WO-MOS-0650`  
**설계 원칙**: 제0조~제32조 마스터 명령 전수 준수 (독립 플랫폼 원칙, Level 1~4 데이터 수집 전략, 미승인 외부 API 허위 구현 금지)

---

## 1. 목표 (Objective)

제13조에 따라 매장 리뷰 통합 관리, AI 답글 보조, 익명화된 방문/재방문 코호트 분석, 단골 세그먼트 및 적법한 마케팅 동의(Opt-in) 기반 알림톡/쿠폰 캠페인 엔진을 설계한다. 실제 API가 제공하지 않는 개인 고객별 구매 이력을 임의로 추론하거나 허위 표시하지 않는다.

---

## 2. 사용자 역할 (User Roles & Actors)

- **주요 액터**: 매장 대표, 매장 매니저, 마케팅 담당자
- **권한 경계**: 모든 요청은 인증된 사용자(`user_id`)와 명시적으로 선택·검증된 사업자(`business_id`) 및 매장(`store_id`) 컨텍스트 하에서만 실행되며, 타 사업장 데이터에 대한 교차 접근은 애플리케이션 미들웨어와 PostgreSQL RLS(Row-Level Security) 양쪽에서 차단된다.

---

## 3. 기능 명세 (Functional Specifications)

- **핵심 기능 1**: 멀티채널 리뷰 허브: 리뷰 출처(`review_sources`: Google Business Profile, 정식 제휴 배달/지도 플랫폼, 자체 스마트오더/영수증 리뷰, Level 2 업로드 리뷰) 명확 표시, 별점(1~5점) 필터, 미답변 필터, 부정적 리뷰(1~2점 또는 핵심 불만 키워드 포함) 즉시 알림
- **핵심 기능 2**: 리뷰 답변 워크플로우(공식 API vs 복사 보조 정직한 분리): 정식 답변 API가 제공·승인된 플랫폼(예: Google Business Profile API 및 공식 제휴 채널)은 대시보드 내에서 즉시 `POST` 답글 등록을 수행하고, 공개 API가 없는 플랫폼은 AI 맞춤형 답글 초안 생성 + 원클릭 클립보드 복사 + 해당 사장님 앱 딥링크 이동 버튼을 제공
- **핵심 기능 3**: 리뷰 감성 및 키워드 분석: 맛/양, 가격/가성비, 배달/포장 상태, 친절도, 위생 5대 축 긍정·부정 키워드 추세 분석
- **핵심 기능 4**: 익명 집계 방문 분석 vs 식별 단골 CRM 엄격 분리: (1) 카드/POS 비식별 토큰 기반 집계는 '신규 vs 재방문 결제 비율, 시간대별 객단가' 등 익명 통계로만 표시, (2) 매장 포인트 적립이나 스탬프/스마트오더를 통해 **개인정보 수집·이용 및 마케팅 수신 동의(Opt-in)**를 명시적으로 완료한 고객에 한해서만 개별 단골 프로필 및 쿠폰 알림톡 발송 허용
- **핵심 기능 5**: 마케팅 수신 동의 및 수신거부(080 무료수신거부 / 원클릭 Opt-out) 이력 법정 관리

---

## 4. 사용자 시나리오 (End-to-End User Scenario)

> **실전 시나리오**: 매장에 2점짜리 리뷰('배달이 식어서 왔어요')가 등록되면 사장님께 즉시 알림이 가고, AI가 정중한 사과와 보온 포장 개선 약속을 담은 답글 초안 3가지를 제안한다. 또한 최근 30일간 마케팅 수신에 동의한 3회 이상 방문 단골 세그먼트에게만 평일 오후 시간대 2,000원 할인 쿠폰 알림톡을 발송한다.

1. **Pre-condition**: 사용자가 정상 로그인되어 있고 해당 매장에 대해 역할 권한(`RBAC/ABAC`)을 보유하고 있다. 외부 연동 기능의 경우 `provider_catalog` 및 `data_connections` 상태가 확인된다.
2. **Normal Flow**: 요청이 API Gateway를 통과하여 도메인 서비스에서 처리되고, 원본 출처(`data_source_level`)와 함께 PostgreSQL 트랜잭션 및 `outbox_events`에 기록된다.
3. **Fallback Flow (연동 차단/미계약 시)**: 만약 외부 API가 `CONTRACT_REQUIRED` 또는 `PARTNER_ONLY(BLOCKED)` 상태이면, 가짜 실시간 데이터를 표시하지 않고 즉시 **Level 2(공식 정산파일 CSV/Excel 업로드)** 또는 **Level 1(수기 입력)** 인터페이스로 안내하여 핵심 장부·대사 업무가 중단 없이 수행되도록 보장한다.

---

## 5. 백엔드 아키텍처 및 구현 명세 (Backend Architecture)

- **언어 및 프레임워크**: Go 1.22+, 모듈형 모놀리스 아키텍처 + 독립 비동기 워커(Go Workers)
- **핵심 구현 모듈**: `internal/crm` 패키지: 리뷰 수집/정규화 워커, 감성 키워드 추출기, 코호트 재방문율 계산기, 마케팅 동의 검증기(동의 없는 대상 발송 시 하드 블록)
- **트랜잭션 관리**: 모든 상태 변경과 원장 반영은 `pgx/v5` 트랜잭션 내부에서 도메인 데이터 변경 + `audit_logs` 기록 + `outbox_events` 발행을 원자적(Atomic)으로 커밋한다.

---

## 6. 프론트엔드 및 모바일 명세 (Frontend & Mobile)

- **구현 범위**: 통합 리뷰 인박스, 별점 추이 차트, AI 답글 톤앤매너 선택기(정중함/친근함/간결함), 고객 세그먼트 빌더, 쿠폰 캠페인 성과(발송수/열람수/매장 사용 전환액) 대시보드
- **상태 표현 표준**:
  1. `LOADING`: 스켈레톤 UI 및 진행률 표시
  2. `SUCCESS`: 데이터 표시 + 우측 상단 `DataProvenanceBadge`(데이터 출처 Level 1~4 및 최종 갱신 시각)
  3. `EMPTY`: 초기 온보딩 안내 및 첫 데이터 등록/업로드 CTA 버튼
  4. `ERROR`: 사용자 친화적 오류 메시지 + `request_id` 복사 + 재시도 버튼
  5. `BLOCKED`: 외부 기관 계약/승인 대기 상태의 정직한 고지 + 파일 업로드 대체 경로 버튼

---

## 7. 데이터베이스 설계 연동 (Database Schema Mapping)

- **연관 테이블**: `customer_segments`, `review_sources`, `reviews`, `review_responses`, `connection_consents`
- **데이터 무결성 규칙**:
  - 모든 금액 컬럼은 `NUMERIC(19,4)`를 사용하여 부동소수점 반올림 오차를 원천 배제한다.
  - 모든 시각 컬럼은 `TIMESTAMPTZ`로 저장한다.
  - 원본 거래 레코드는 임의 삭제(`DELETE`)하지 않으며, 정정 시 `corrections` 및 역분개(`journal_entries`)로 감사 추적성을 유지한다.

---

## 8. API 계약 명세 (OpenAPI 3.1 Endpoints)

- **주요 엔드포인트**: `GET /v1/stores/{storeId}/reviews`, `POST /v1/stores/{storeId}/reviews/{reviewId}/reply`, `POST /v1/stores/{storeId}/reviews/{reviewId}/ai-draft`, `GET /v1/stores/{storeId}/crm/segments`, `POST /v1/stores/{storeId}/crm/campaigns`
- **공통 헤더**:
  - `Authorization: Bearer <OIDC_JWT>`
  - `X-Business-Id: <UUID>`
  - `X-Store-Id: <UUID>`
  - `Idempotency-Key: <UUID>` (모든 `POST`/`PUT`/`PATCH` 상태 변경 요청 시 필수)

---

## 9. 도메인 이벤트 (Domain Events)

- **발행/구독 이벤트**: `review.ingested`, `review.negative_alert`, `review.replied`, `crm.campaign.dispatched`
- **전달 보장**: Transactional Outbox 테이블(`outbox_events`)에서 Redis Streams / Worker Queue로 At-Least-Once 전달을 보장하며, 컨슈머는 `event_id` 기준 멱등 처리(Idempotent Consumer)를 수행한다.

---

## 10. 데이터 출처 및 신뢰도 등급 (Data Provenance)

- **데이터 원천**: 공식 리뷰 API(Google Business Profile 등), 자체 영수증/스마트오더 리뷰, Level 2 리뷰 내보내기 파일, 적법 동의된 POS/포인트 적립 데이터
- **출처 태깅 규칙**:
  - `LEVEL_1_MANUAL`: 사용자 직접 입력 (신뢰도: 사용자 책임, 즉시 반영)
  - `LEVEL_2_FILE`: 공식 정산 엑셀/CSV 업로드 (신뢰도: 파일 해시 및 포맷 검증됨)
  - `LEVEL_3_API`: 공식 계약/승인된 외부 API (신뢰도: 기관 전자서명/TLS 검증됨)
  - `LEVEL_4_RECONCILED`: 복수 데이터 소스(예: 카드매입 + 은행입금) 교차 대사 완료 (신뢰도: 최고 등급 확정)

---

## 11. 오류 처리 및 장애 복구 (Error Handling & Resilience)

- **예외 처리 정책**: 마케팅 캠페인 발송 요청 시 수신자 중 단 1명이라도 마케팅 동의가 없거나 철회된 경우 자동 제외하고 제외 사유 리포트 생성; 야간 시간대(20:50~08:00) 발송 요청은 익일 08:05 예약 발송으로 자동 전환
- **표준 에러 코드**: `AUTH_FORBIDDEN_STORE`, `CONNECTOR_EXTERNAL_BLOCKED`, `LEDGER_UNBALANCED_ENTRY`, `IMPORT_DUPLICATE_BATCH`, `RATE_LIMIT_EXCEEDED`, `UPSTREAM_TIMEOUT_RETRYABLE`

---

## 12. 개인정보 및 보안 통제 (Privacy & Security Controls)

- **보안 설계**: 익명 결제 토큰으로부터 개인의 실명이나 연락처를 역추적하려는 시도를 원천 금지하며, 마케팅 동의 일시·채널·약관 버전을 감사 로그에 영구 보존.
- **감사 추적**: 모든 민감 데이터 조회 및 변경은 `audit_logs`에 요청자 ID, 대상 매장 ID, 액션 코드, IP, User-Agent, 변경 전후 요약 및 해시 체인과 함께 기록된다.

---

## 13. 외부 승인 및 법적 선행 요건 (External Approval & Regulatory Requirements)

- **승인·규제 요건**: 정보통신망법 제50조(영리목적 광고성 정보 전송 제한 및 매 2년 수신동의 재확인 의무) 준수.
- **차단(Blocked) 정책**: 필수 외부 계약이나 인허가가 완료되기 전까지 해당 라이브 연동 기능은 `feature_flags` 및 `provider_catalog`에서 `BLOCKED`로 분류하며, 가짜 API 응답으로 상용 완료를 가장하는 행위를 엄격히 금지한다.

---

## 14. 성능 및 확장성 목표 (Performance & SLA Targets)

- **목표 지표**: 리뷰 목록 필터링 조회 < 100ms, AI 답글 초안 생성 < 2.0초.

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

- **직접 매핑 범위**: `WO-MOS-0601 ~ WO-MOS-0650`
- **상세 개별 명세 위치**: `11_WORK_ORDERS_0001_TO_1000/individual/` 내 해당 번호 파일 및 `11_WORK_ORDERS_0001_TO_1000/WORK_ORDER_INDEX.csv` 참조.
