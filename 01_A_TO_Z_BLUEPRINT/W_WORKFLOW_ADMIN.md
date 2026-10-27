# W — Workflow & Administration (관리자 콘솔·고객지원·파트너 운영·데이터 정정·보안 감사)

**문서 ID**: AZ-W_WORKFLOW_ADMIN  
**프로젝트 코드명**: BN Merchant OS (`BN-MERCHANT-OS-GEMINI4-ARGON-MASTER-BLUEPRINT-1000-001`)  
**관련 Work Order 범위**: `WO-MOS-0701 ~ WO-MOS-0750`  
**설계 원칙**: 제0조~제32조 마스터 명령 전수 준수 (독립 플랫폼 원칙, Level 1~4 데이터 수집 전략, 미승인 외부 API 허위 구현 금지)

---

## 1. 목표 (Objective)

제17조 관리자 화면(17개 화면) 요구사항에 따라 플랫폼 운영팀, 보안 감사팀, 고객지원(CS)팀, 외부 데이터 파트너 관리자가 사용하는 통합 Admin Console 및 운영 워크플로우를 설계한다. 모든 관리자 행위는 4-Eyes 승인(중요 작업) 및 불변 감사 로그에 기록된다.

---

## 2. 사용자 역할 (User Roles & Actors)

- **주요 액터**: Platform Super Admin, CS Operator, Partner Manager, Compliance Officer, Financial Reconciliation Specialist
- **권한 경계**: 모든 요청은 인증된 사용자(`user_id`)와 명시적으로 선택·검증된 사업자(`business_id`) 및 매장(`store_id`) 컨텍스트 하에서만 실행되며, 타 사업장 데이터에 대한 교차 접근은 애플리케이션 미들웨어와 PostgreSQL RLS(Row-Level Security) 양쪽에서 차단된다.

---

## 3. 기능 명세 (Functional Specifications)

- **핵심 기능 1**: 관리자 17대 표준 화면 완비: (1) 관리자 홈(전체 플랫폼 KPI·수집성공률·장애알림), (2) 회원 관리, (3) 사업자 인증 심사 큐(서류/소유권 분쟁 심사), (4) 사업장 관리, (5) 데이터 커넥터 카탈로그 및 피처 플래그, (6) 연동 실패 모니터링 및 재시도 콘솔, (7) 정산 대사 오류 및 이의제기(`reconciliation_disputes`) 처리 워크플로우, (8) 금융 파트너 관리, (9) POS 파트너 관리, (10) 배달 플랫폼 정책/파서 관리, (11) 리뷰 플랫폼 연동 관리, (12) 커뮤니티 콘텐츠·신고·제재 관리, (13) B2B 마켓 공급사·상품·분쟁 관리, (14) 구독·결제·환불 관리, (15) 고객지원(CS 티켓 및 Break-Glass 열람 승인), (16) 보안 감사 로그 뷰어, (17) 운영 모니터링(워커 큐·DB·OpenSearch 상태)
- **핵심 기능 2**: 정산 대사 이의제기 및 수동 정정 워크플로우: 가맹점주가 '카드사 입금 누락 진단이 실제 통장 입금과 다릅니다'라고 이의제기를 접수하면, 정산 전문가가 증빙을 검토하여 매칭 룰을 보정하거나 원장 정정 분개(`corrections`)를 제안하고 가맹점주 확인 후 반영
- **핵심 기능 3**: 4-Eyes(2인 승인) 거버넌스: 외부 커넥터 라이브 활성화, 수수료율 마스터 변경, 대규모 환불 실행, 규제 플래그(`REGULATORY_BLOCKED` 해제) 변경은 요청자 외 별도 보안/법무 관리자의 전자승인이 있어야만 실행(`admin_actions` 기록)

---

## 4. 사용자 시나리오 (End-to-End User Scenario)

> **실전 시나리오**: 가맹점주가 법인 공동대표 위임장으로 사업장 소유권 심사를 요청하면, CS 심사관이 '사업자 인증 심사 큐'에서 위임장 진위와 인감증명서를 확인한 후 승인 버튼을 누르고, 승인 사유가 `business_verifications`와 `admin_actions`에 영구 기록된다.

1. **Pre-condition**: 사용자가 정상 로그인되어 있고 해당 매장에 대해 역할 권한(`RBAC/ABAC`)을 보유하고 있다. 외부 연동 기능의 경우 `provider_catalog` 및 `data_connections` 상태가 확인된다.
2. **Normal Flow**: 요청이 API Gateway를 통과하여 도메인 서비스에서 처리되고, 원본 출처(`data_source_level`)와 함께 PostgreSQL 트랜잭션 및 `outbox_events`에 기록된다.
3. **Fallback Flow (연동 차단/미계약 시)**: 만약 외부 API가 `CONTRACT_REQUIRED` 또는 `PARTNER_ONLY(BLOCKED)` 상태이면, 가짜 실시간 데이터를 표시하지 않고 즉시 **Level 2(공식 정산파일 CSV/Excel 업로드)** 또는 **Level 1(수기 입력)** 인터페이스로 안내하여 핵심 장부·대사 업무가 중단 없이 수행되도록 보장한다.

---

## 5. 백엔드 아키텍처 및 구현 명세 (Backend Architecture)

- **언어 및 프레임워크**: Go 1.22+, 모듈형 모놀리스 아키텍처 + 독립 비동기 워커(Go Workers)
- **핵심 구현 모듈**: `internal/admin` 패키지: 관리자 전용 RBAC/ABAC 미들웨어(사내 VPN/Zero-Trust IP 대역 및 하드웨어 MFA 필수), 4-Eyes 승인 상태 머신, CS 마스킹 해제(Break-Glass) 임시 토큰 발급기
- **트랜잭션 관리**: 모든 상태 변경과 원장 반영은 `pgx/v5` 트랜잭션 내부에서 도메인 데이터 변경 + `audit_logs` 기록 + `outbox_events` 발행을 원자적(Atomic)으로 커밋한다.

---

## 6. 프론트엔드 및 모바일 명세 (Frontend & Mobile)

- **구현 범위**: `apps/web-admin` (Vue 3 + TypeScript): 고밀도 운영 테이블, JSON/Diff 비교 뷰어, 커넥터 상태 제어판, 감사 로그 타임라인
- **상태 표현 표준**:
  1. `LOADING`: 스켈레톤 UI 및 진행률 표시
  2. `SUCCESS`: 데이터 표시 + 우측 상단 `DataProvenanceBadge`(데이터 출처 Level 1~4 및 최종 갱신 시각)
  3. `EMPTY`: 초기 온보딩 안내 및 첫 데이터 등록/업로드 CTA 버튼
  4. `ERROR`: 사용자 친화적 오류 메시지 + `request_id` 복사 + 재시도 버튼
  5. `BLOCKED`: 외부 기관 계약/승인 대기 상태의 정직한 고지 + 파일 업로드 대체 경로 버튼

---

## 7. 데이터베이스 설계 연동 (Database Schema Mapping)

- **연관 테이블**: `admin_actions`, `audit_logs`, `feature_flags`, `business_verifications`, `reconciliation_disputes`
- **데이터 무결성 규칙**:
  - 모든 금액 컬럼은 `NUMERIC(19,4)`를 사용하여 부동소수점 반올림 오차를 원천 배제한다.
  - 모든 시각 컬럼은 `TIMESTAMPTZ`로 저장한다.
  - 원본 거래 레코드는 임의 삭제(`DELETE`)하지 않으며, 정정 시 `corrections` 및 역분개(`journal_entries`)로 감사 추적성을 유지한다.

---

## 8. API 계약 명세 (OpenAPI 3.1 Endpoints)

- **주요 엔드포인트**: `GET /v1/admin/dashboard`, `POST /v1/admin/business-verifications/{id}/approve`, `POST /v1/admin/connectors/{code}/toggle`, `POST /v1/admin/reconciliation-disputes/{id}/resolve`, `GET /v1/admin/audit-logs`
- **공통 헤더**:
  - `Authorization: Bearer <OIDC_JWT>`
  - `X-Business-Id: <UUID>`
  - `X-Store-Id: <UUID>`
  - `Idempotency-Key: <UUID>` (모든 `POST`/`PUT`/`PATCH` 상태 변경 요청 시 필수)

---

## 9. 도메인 이벤트 (Domain Events)

- **발행/구독 이벤트**: `admin.verification.decided`, `admin.feature_flag.changed`, `admin.break_glass.accessed`
- **전달 보장**: Transactional Outbox 테이블(`outbox_events`)에서 Redis Streams / Worker Queue로 At-Least-Once 전달을 보장하며, 컨슈머는 `event_id` 기준 멱등 처리(Idempotent Consumer)를 수행한다.

---

## 10. 데이터 출처 및 신뢰도 등급 (Data Provenance)

- **데이터 원천**: 전체 플랫폼 운영 메타데이터 및 감사 로그
- **출처 태깅 규칙**:
  - `LEVEL_1_MANUAL`: 사용자 직접 입력 (신뢰도: 사용자 책임, 즉시 반영)
  - `LEVEL_2_FILE`: 공식 정산 엑셀/CSV 업로드 (신뢰도: 파일 해시 및 포맷 검증됨)
  - `LEVEL_3_API`: 공식 계약/승인된 외부 API (신뢰도: 기관 전자서명/TLS 검증됨)
  - `LEVEL_4_RECONCILED`: 복수 데이터 소스(예: 카드매입 + 은행입금) 교차 대사 완료 (신뢰도: 최고 등급 확정)

---

## 11. 오류 처리 및 장애 복구 (Error Handling & Resilience)

- **예외 처리 정책**: 관리자가 결제/정산 원장 데이터를 임의로 직접 수정(Raw SQL Update)하는 기능은 UI와 API 모두에서 존재하지 않으며, 오직 정정 분개 워크플로우를 통해서만 이력이 남는 방식으로 처리(제31조 준수)
- **표준 에러 코드**: `AUTH_FORBIDDEN_STORE`, `CONNECTOR_EXTERNAL_BLOCKED`, `LEDGER_UNBALANCED_ENTRY`, `IMPORT_DUPLICATE_BATCH`, `RATE_LIMIT_EXCEEDED`, `UPSTREAM_TIMEOUT_RETRYABLE`

---

## 12. 개인정보 및 보안 통제 (Privacy & Security Controls)

- **보안 설계**: 관리자 콘솔은 일반 사용자 웹과 도메인·포트·인증 쿠키를 완전히 분리하며, 모든 관리자 조회/변경은 `admin_actions`에 변경 전(`before_state`)과 변경 후(`after_state`) JSON을 남김.
- **감사 추적**: 모든 민감 데이터 조회 및 변경은 `audit_logs`에 요청자 ID, 대상 매장 ID, 액션 코드, IP, User-Agent, 변경 전후 요약 및 해시 체인과 함께 기록된다.

---

## 13. 외부 승인 및 법적 선행 요건 (External Approval & Regulatory Requirements)

- **승인·규제 요건**: 개인정보보호법상 개인정보취급자 접속기록 월 1회 이상 점검 리포트 자동 생성 지원.
- **차단(Blocked) 정책**: 필수 외부 계약이나 인허가가 완료되기 전까지 해당 라이브 연동 기능은 `feature_flags` 및 `provider_catalog`에서 `BLOCKED`로 분류하며, 가짜 API 응답으로 상용 완료를 가장하는 행위를 엄격히 금지한다.

---

## 14. 성능 및 확장성 목표 (Performance & SLA Targets)

- **목표 지표**: 관리자 검색 및 감사 로그 필터링 P95 < 200ms.

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

- **직접 매핑 범위**: `WO-MOS-0701 ~ WO-MOS-0750`
- **상세 개별 명세 위치**: `11_WORK_ORDERS_0001_TO_1000/individual/` 내 해당 번호 파일 및 `11_WORK_ORDERS_0001_TO_1000/WORK_ORDER_INDEX.csv` 참조.
