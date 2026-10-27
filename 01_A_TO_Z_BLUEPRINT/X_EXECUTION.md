# X — eXecution & Agents (1,000개 Work Order 분해 및 16개 멀티 에이전트 오케스트레이션)

**문서 ID**: AZ-X_EXECUTION  
**프로젝트 코드명**: BN Merchant OS (`BN-MERCHANT-OS-GEMINI4-ARGON-MASTER-BLUEPRINT-1000-001`)  
**관련 Work Order 범위**: `WO-MOS-0001 ~ WO-MOS-1000`  
**설계 원칙**: 제0조~제32조 마스터 명령 전수 준수 (독립 플랫폼 원칙, Level 1~4 데이터 수집 전략, 미승인 외부 API 허위 구현 금지)

---

## 1. 목표 (Objective)

제22조, 제23조, 제24조, 제28조에 따라 전체 플랫폼 구축 과업을 `WO-MOS-0001`부터 `WO-MOS-1000`까지 정확히 1,000개의 원자적(Atomic)이고 실행 가능한 작업지시서로 분해하고, 16개 전문 개발 에이전트가 파일 충돌이나 계약 불일치 없이 병렬/순차 협업하는 오케스트레이션 프로토콜을 정의한다.

---

## 2. 사용자 역할 (User Roles & Actors)

- **주요 액터**: Multi-Agent Development Orchestrator, Lead Architect, 16개 도메인 에이전트 전체, Independent Audit Agent
- **권한 경계**: 모든 요청은 인증된 사용자(`user_id`)와 명시적으로 선택·검증된 사업자(`business_id`) 및 매장(`store_id`) 컨텍스트 하에서만 실행되며, 타 사업장 데이터에 대한 교차 접근은 애플리케이션 미들웨어와 PostgreSQL RLS(Row-Level Security) 양쪽에서 차단된다.

---

## 3. 기능 명세 (Functional Specifications)

- **핵심 기능 1**: 20개 개발 분야 × 각 50개 고유 작업 = 정확히 1,000개 개별 Work Order (`11_WORK_ORDERS_0001_TO_1000/individual/WO-MOS-0001.md` ~ `WO-MOS-1000.md` 및 `WORK_ORDER_INDEX.csv` 수록)
- **핵심 기능 2**: 16개 전문 에이전트 역할 및 파일 소유권(CODEOWNERS) 격리: 동일 시간대에 두 에이전트가 같은 소스 파일이나 마이그레이션 번호를 수정하지 않도록 디렉터리·모듈별 배타적 잠금(Exclusive Path Lock) 규칙 적용
- **핵심 기능 3**: Contract-First Handoff 프로토콜: DB 스키마(`03_DATABASE/`)와 OpenAPI 명세(`04_API_SPECIFICATION/openapi.yaml`)가 먼저 확정(Freeze)된 후 Backend, Frontend, Mobile, QA 에이전트가 동시에 구현을 진행하며, 계약 변경 필요 시 `Lead Architect`의 RFC 승인 후 연관 Work Order에 자동 전파
- **핵심 기능 4**: 구현 에이전트와 독립 감사 에이전트(`Independent Audit Agent`)의 엄격한 분리: 개발 에이전트 스스로 `DONE`을 최종 선언할 수 없으며, 테스트 통과 로그·Git SHA·Acceptance Criteria 충족 증빙을 `Independent Audit Agent`가 검증해야만 완료 처리

---

## 4. 사용자 시나리오 (End-to-End User Scenario)

> **실전 시나리오**: ZIP 패키지를 인수한 후속 개발 오케스트레이터는 `START_HERE.md`를 읽고 `MANIFEST.json`과 `FILE_CHECKSUMS.sha256`으로 무결성을 확인한 뒤, `12_AGENT_EXECUTION/EXECUTION_SEQUENCE.md`의 DAG 위상 정렬 순서에 따라 Phase 0부터 Phase 10까지 에이전트를 투입한다.

1. **Pre-condition**: 사용자가 정상 로그인되어 있고 해당 매장에 대해 역할 권한(`RBAC/ABAC`)을 보유하고 있다. 외부 연동 기능의 경우 `provider_catalog` 및 `data_connections` 상태가 확인된다.
2. **Normal Flow**: 요청이 API Gateway를 통과하여 도메인 서비스에서 처리되고, 원본 출처(`data_source_level`)와 함께 PostgreSQL 트랜잭션 및 `outbox_events`에 기록된다.
3. **Fallback Flow (연동 차단/미계약 시)**: 만약 외부 API가 `CONTRACT_REQUIRED` 또는 `PARTNER_ONLY(BLOCKED)` 상태이면, 가짜 실시간 데이터를 표시하지 않고 즉시 **Level 2(공식 정산파일 CSV/Excel 업로드)** 또는 **Level 1(수기 입력)** 인터페이스로 안내하여 핵심 장부·대사 업무가 중단 없이 수행되도록 보장한다.

---

## 5. 백엔드 아키텍처 및 구현 명세 (Backend Architecture)

- **언어 및 프레임워크**: Go 1.22+, 모듈형 모놀리스 아키텍처 + 독립 비동기 워커(Go Workers)
- **핵심 구현 모듈**: 에이전트 작업 상태 추적기, Git 브랜치 전략(`wo/WO-MOS-XXXX`), CI 계약 검증 게이트
- **트랜잭션 관리**: 모든 상태 변경과 원장 반영은 `pgx/v5` 트랜잭션 내부에서 도메인 데이터 변경 + `audit_logs` 기록 + `outbox_events` 발행을 원자적(Atomic)으로 커밋한다.

---

## 6. 프론트엔드 및 모바일 명세 (Frontend & Mobile)

- **구현 범위**: N/A (개발 오케스트레이션 체계)
- **상태 표현 표준**:
  1. `LOADING`: 스켈레톤 UI 및 진행률 표시
  2. `SUCCESS`: 데이터 표시 + 우측 상단 `DataProvenanceBadge`(데이터 출처 Level 1~4 및 최종 갱신 시각)
  3. `EMPTY`: 초기 온보딩 안내 및 첫 데이터 등록/업로드 CTA 버튼
  4. `ERROR`: 사용자 친화적 오류 메시지 + `request_id` 복사 + 재시도 버튼
  5. `BLOCKED`: 외부 기관 계약/승인 대기 상태의 정직한 고지 + 파일 업로드 대체 경로 버튼

---

## 7. 데이터베이스 설계 연동 (Database Schema Mapping)

- **연관 테이블**: 마이그레이션 번호 대역 사전 할당(예: Identity `0001~0010`, Business `0011~0020`, Revenue `0021~0035` 등)
- **데이터 무결성 규칙**:
  - 모든 금액 컬럼은 `NUMERIC(19,4)`를 사용하여 부동소수점 반올림 오차를 원천 배제한다.
  - 모든 시각 컬럼은 `TIMESTAMPTZ`로 저장한다.
  - 원본 거래 레코드는 임의 삭제(`DELETE`)하지 않으며, 정정 시 `corrections` 및 역분개(`journal_entries`)로 감사 추적성을 유지한다.

---

## 8. API 계약 명세 (OpenAPI 3.1 Endpoints)

- **주요 엔드포인트**: `04_API_SPECIFICATION/openapi.yaml` 단일 진실 공급원(Single Source of Truth)
- **공통 헤더**:
  - `Authorization: Bearer <OIDC_JWT>`
  - `X-Business-Id: <UUID>`
  - `X-Store-Id: <UUID>`
  - `Idempotency-Key: <UUID>` (모든 `POST`/`PUT`/`PATCH` 상태 변경 요청 시 필수)

---

## 9. 도메인 이벤트 (Domain Events)

- **발행/구독 이벤트**: `agent.wo.started`, `agent.wo.submitted_for_audit`, `agent.wo.audit_passed`, `agent.wo.blocked`
- **전달 보장**: Transactional Outbox 테이블(`outbox_events`)에서 Redis Streams / Worker Queue로 At-Least-Once 전달을 보장하며, 컨슈머는 `event_id` 기준 멱등 처리(Idempotent Consumer)를 수행한다.

---

## 10. 데이터 출처 및 신뢰도 등급 (Data Provenance)

- **데이터 원천**: `11_WORK_ORDERS_0001_TO_1000/` 및 `12_AGENT_EXECUTION/` 문서군
- **출처 태깅 규칙**:
  - `LEVEL_1_MANUAL`: 사용자 직접 입력 (신뢰도: 사용자 책임, 즉시 반영)
  - `LEVEL_2_FILE`: 공식 정산 엑셀/CSV 업로드 (신뢰도: 파일 해시 및 포맷 검증됨)
  - `LEVEL_3_API`: 공식 계약/승인된 외부 API (신뢰도: 기관 전자서명/TLS 검증됨)
  - `LEVEL_4_RECONCILED`: 복수 데이터 소스(예: 카드매입 + 은행입금) 교차 대사 완료 (신뢰도: 최고 등급 확정)

---

## 11. 오류 처리 및 장애 복구 (Error Handling & Resilience)

- **예외 처리 정책**: 선행 작업(Prerequisite WO)이 실패하거나 외부 승인이 `BLOCKED`인 경우, 의존하는 후속 작업은 가짜 API를 만들지 않고 즉시 계약 인터페이스(Stub/Level 2 File Import) 경로로 우회하거나 `BLOCKED_EXTERNAL_APPROVAL`로 정직하게 상태를 기록
- **표준 에러 코드**: `AUTH_FORBIDDEN_STORE`, `CONNECTOR_EXTERNAL_BLOCKED`, `LEDGER_UNBALANCED_ENTRY`, `IMPORT_DUPLICATE_BATCH`, `RATE_LIMIT_EXCEEDED`, `UPSTREAM_TIMEOUT_RETRYABLE`

---

## 12. 개인정보 및 보안 통제 (Privacy & Security Controls)

- **보안 설계**: 에이전트가 테스트 코드나 감사 스크립트를 임의 삭제/우회(`skip`)하여 빌드를 통과시키는 행위를 CI Lint 규칙으로 원천 차단.
- **감사 추적**: 모든 민감 데이터 조회 및 변경은 `audit_logs`에 요청자 ID, 대상 매장 ID, 액션 코드, IP, User-Agent, 변경 전후 요약 및 해시 체인과 함께 기록된다.

---

## 13. 외부 승인 및 법적 선행 요건 (External Approval & Regulatory Requirements)

- **승인·규제 요건**: 제28조 인수 프로토콜 20단계 전체 준수.
- **차단(Blocked) 정책**: 필수 외부 계약이나 인허가가 완료되기 전까지 해당 라이브 연동 기능은 `feature_flags` 및 `provider_catalog`에서 `BLOCKED`로 분류하며, 가짜 API 응답으로 상용 완료를 가장하는 행위를 엄격히 금지한다.

---

## 14. 성능 및 확장성 목표 (Performance & SLA Targets)

- **목표 지표**: 파일 충돌 발생률 0%, 모든 1,000개 Work Order의 Requirement → API/DB → Test Case 추적성 100%.

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

- **직접 매핑 범위**: `WO-MOS-0001 ~ WO-MOS-1000`
- **상세 개별 명세 위치**: `11_WORK_ORDERS_0001_TO_1000/individual/` 내 해당 번호 파일 및 `11_WORK_ORDERS_0001_TO_1000/WORK_ORDER_INDEX.csv` 참조.
