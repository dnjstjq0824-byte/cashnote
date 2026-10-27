# B — Business Registration (사업자 등록·소유권 검증·조직·점포 관리)

**문서 ID**: AZ-B_BUSINESS_REGISTRATION  
**프로젝트 코드명**: BN Merchant OS (`BN-MERCHANT-OS-GEMINI4-ARGON-MASTER-BLUEPRINT-1000-001`)  
**관련 Work Order 범위**: `WO-MOS-0101 ~ WO-MOS-0200`  
**설계 원칙**: 제0조~제32조 마스터 명령 전수 준수 (독립 플랫폼 원칙, Level 1~4 데이터 수집 전략, 미승인 외부 API 허위 구현 금지)

---

## 1. 목표 (Objective)

제6조 원칙에 따라 국세청 사업자등록 상태조회(실체 확인)와 실제 사용자의 대표자/관리자 권한 확인(소유권 검증)을 명확히 분리하여 설계한다. 타인이 남의 사업자등록번호만 알고 허위로 매장을 등록해 매출·금융 데이터를 열람하는 보안 사고를 원천 차단한다.

---

## 2. 사용자 역할 (User Roles & Actors)

- **주요 액터**: 예비 사장님, 개인사업자 대표, 법인사업자 대표, 공동대표, 매장 매니저, 회계 담당자, 외부 세무대리인, 플랫폼 심사 관리자
- **권한 경계**: 모든 요청은 인증된 사용자(`user_id`)와 명시적으로 선택·검증된 사업자(`business_id`) 및 매장(`store_id`) 컨텍스트 하에서만 실행되며, 타 사업장 데이터에 대한 교차 접근은 애플리케이션 미들웨어와 PostgreSQL RLS(Row-Level Security) 양쪽에서 차단된다.

---

## 3. 기능 명세 (Functional Specifications)

- **핵심 기능 1**: 국세청 사업자등록정보 진위확인(`POST /nts-businessman/v1/validate`) 및 상태조회(`POST /nts-businessman/v1/status`) 실시간 연동 (과세유형: 일반/간이/면세/법인, 휴·폐업 상태 및 폐업일자 확인)
- **핵심 기능 2**: 3단계 소유권 검증(Ownership Verification): Step 1 본인명의 휴대전화 PASS/통신사 본인인증(CI/DI) + Step 2 국세청 진위확인(사업자번호+개업일자+대표자성명 일치) + Step 3 사업자 명의 은행계좌 1원 입금 인증코드 확인(또는 법인등기부/사업자등록증명원/위임장 관리자 심사)
- **핵심 기능 3**: 개인사업자 및 법인사업자 등록, 단일 사업자번호 하위 다매장(지점/브랜드 매장) 추가 및 복수 사업자번호 통합 관리
- **핵심 기능 4**: 역할 기반 초대 시스템: 매니저(`ROLE_STORE_MANAGER`), 회계담당(`ROLE_ACCOUNTANT`), 세무대리인(`ROLE_TAX_AGENT`), 조회전용(`ROLE_VIEWER`) 초대 링크/알림톡 발송 및 만료 관리
- **핵심 기능 5**: 사업장 소유권 이전(양도·양수 시 기존 데이터 접근 기준일 분리), 조직 탈퇴, 데이터 연결 철회 및 법정 보존기간 기반 파기 정책

---

## 4. 사용자 시나리오 (End-to-End User Scenario)

> **실전 시나리오**: 사장님이 사업자등록번호 `123-45-67890`을 입력하면 국세청 API로 '계속사업자(일반과세자)'임을 1차 확인하되, 상태는 `VERIFICATION_PENDING`으로 두고 금융 데이터 연동을 잠근다. 이어서 본인인증 성명과 대표자 성명 일치를 확인하고 사업자 계좌 1원 인증을 통과하면 `VERIFIED_OWNER`로 승격되어 전체 기능이 활성화된다.

1. **Pre-condition**: 사용자가 정상 로그인되어 있고 해당 매장에 대해 역할 권한(`RBAC/ABAC`)을 보유하고 있다. 외부 연동 기능의 경우 `provider_catalog` 및 `data_connections` 상태가 확인된다.
2. **Normal Flow**: 요청이 API Gateway를 통과하여 도메인 서비스에서 처리되고, 원본 출처(`data_source_level`)와 함께 PostgreSQL 트랜잭션 및 `outbox_events`에 기록된다.
3. **Fallback Flow (연동 차단/미계약 시)**: 만약 외부 API가 `CONTRACT_REQUIRED` 또는 `PARTNER_ONLY(BLOCKED)` 상태이면, 가짜 실시간 데이터를 표시하지 않고 즉시 **Level 2(공식 정산파일 CSV/Excel 업로드)** 또는 **Level 1(수기 입력)** 인터페이스로 안내하여 핵심 장부·대사 업무가 중단 없이 수행되도록 보장한다.

---

## 5. 백엔드 아키텍처 및 구현 명세 (Backend Architecture)

- **언어 및 프레임워크**: Go 1.22+, 모듈형 모놀리스 아키텍처 + 독립 비동기 워커(Go Workers)
- **핵심 구현 모듈**: `internal/business` 서비스: NTS API 클라이언트(Rate Limit 및 재시도 내장), OCR 문서 검증 워커, 1원 계좌인증 검증기, 소유권 상태 머신(`UNVERIFIED` → `NTS_VALIDATED` → `DOCUMENT_SUBMITTED` → `BANK_VERIFIED` → `VERIFIED_OWNER` / `REJECTED` / `REVOKED`)
- **트랜잭션 관리**: 모든 상태 변경과 원장 반영은 `pgx/v5` 트랜잭션 내부에서 도메인 데이터 변경 + `audit_logs` 기록 + `outbox_events` 발행을 원자적(Atomic)으로 커밋한다.

---

## 6. 프론트엔드 및 모바일 명세 (Frontend & Mobile)

- **구현 범위**: 사업자 등록 마법사(4-Step Stepper UI), 사업자등록증 촬영/업로드 드롭존, 1원 인증 4자리 숫자 입력 모달, 다매장 스위처 드롭다운, 팀원 초대 및 권한 매트릭스 설정 화면
- **상태 표현 표준**:
  1. `LOADING`: 스켈레톤 UI 및 진행률 표시
  2. `SUCCESS`: 데이터 표시 + 우측 상단 `DataProvenanceBadge`(데이터 출처 Level 1~4 및 최종 갱신 시각)
  3. `EMPTY`: 초기 온보딩 안내 및 첫 데이터 등록/업로드 CTA 버튼
  4. `ERROR`: 사용자 친화적 오류 메시지 + `request_id` 복사 + 재시도 버튼
  5. `BLOCKED`: 외부 기관 계약/승인 대기 상태의 정직한 고지 + 파일 업로드 대체 경로 버튼

---

## 7. 데이터베이스 설계 연동 (Database Schema Mapping)

- **연관 테이블**: `businesses`, `business_verifications`, `stores`, `store_members`, `store_permissions` 테이블
- **데이터 무결성 규칙**:
  - 모든 금액 컬럼은 `NUMERIC(19,4)`를 사용하여 부동소수점 반올림 오차를 원천 배제한다.
  - 모든 시각 컬럼은 `TIMESTAMPTZ`로 저장한다.
  - 원본 거래 레코드는 임의 삭제(`DELETE`)하지 않으며, 정정 시 `corrections` 및 역분개(`journal_entries`)로 감사 추적성을 유지한다.

---

## 8. API 계약 명세 (OpenAPI 3.1 Endpoints)

- **주요 엔드포인트**: `POST /v1/businesses`, `POST /v1/businesses/{businessId}/verify-nts`, `POST /v1/businesses/{businessId}/verify-bank-penny`, `POST /v1/businesses/{businessId}/documents`, `POST /v1/stores`, `POST /v1/stores/{storeId}/members/invite`, `POST /v1/stores/{storeId}/ownership-transfer`
- **공통 헤더**:
  - `Authorization: Bearer <OIDC_JWT>`
  - `X-Business-Id: <UUID>`
  - `X-Store-Id: <UUID>`
  - `Idempotency-Key: <UUID>` (모든 `POST`/`PUT`/`PATCH` 상태 변경 요청 시 필수)

---

## 9. 도메인 이벤트 (Domain Events)

- **발행/구독 이벤트**: `business.registered`, `business.nts_verified`, `business.ownership_verified`, `store.member_invited`, `store.ownership_transferred`
- **전달 보장**: Transactional Outbox 테이블(`outbox_events`)에서 Redis Streams / Worker Queue로 At-Least-Once 전달을 보장하며, 컨슈머는 `event_id` 기준 멱등 처리(Idempotent Consumer)를 수행한다.

---

## 10. 데이터 출처 및 신뢰도 등급 (Data Provenance)

- **데이터 원천**: 공공데이터포털 국세청 API (`PUBLIC_API`), 본인인증 기관, 오픈뱅킹/중계기관 1원 입금 API, 사용자 업로드 증빙서류
- **출처 태깅 규칙**:
  - `LEVEL_1_MANUAL`: 사용자 직접 입력 (신뢰도: 사용자 책임, 즉시 반영)
  - `LEVEL_2_FILE`: 공식 정산 엑셀/CSV 업로드 (신뢰도: 파일 해시 및 포맷 검증됨)
  - `LEVEL_3_API`: 공식 계약/승인된 외부 API (신뢰도: 기관 전자서명/TLS 검증됨)
  - `LEVEL_4_RECONCILED`: 복수 데이터 소스(예: 카드매입 + 은행입금) 교차 대사 완료 (신뢰도: 최고 등급 확정)

---

## 11. 오류 처리 및 장애 복구 (Error Handling & Resilience)

- **예외 처리 정책**: 국세청 API 타임아웃 시 큐 재시도 및 수동 심사 폴백; 이미 타 계정에 `VERIFIED_OWNER`로 등록된 사업자번호 중복 등록 시 소유권 분쟁(Dispute) 티켓 생성 및 양측 증빙 심사 절차 가동
- **표준 에러 코드**: `AUTH_FORBIDDEN_STORE`, `CONNECTOR_EXTERNAL_BLOCKED`, `LEDGER_UNBALANCED_ENTRY`, `IMPORT_DUPLICATE_BATCH`, `RATE_LIMIT_EXCEEDED`, `UPSTREAM_TIMEOUT_RETRYABLE`

---

## 12. 개인정보 및 보안 통제 (Privacy & Security Controls)

- **보안 설계**: 사업자등록증 이미지 내 주민등록번호 뒷자리 자동 마스킹(`******`) 후 암호화 스토리지 저장. 소유권 미검증(`VERIFIED_OWNER` 미만) 상태에서 외부 금융 커넥터 토큰 발급 절대 차단.
- **감사 추적**: 모든 민감 데이터 조회 및 변경은 `audit_logs`에 요청자 ID, 대상 매장 ID, 액션 코드, IP, User-Agent, 변경 전후 요약 및 해시 체인과 함께 기록된다.

---

## 13. 외부 승인 및 법적 선행 요건 (External Approval & Regulatory Requirements)

- **승인·규제 요건**: 국세청 API는 공공데이터포털 활용신청으로 즉시 사용(`PUBLIC_API`). 본인인증 및 1원 계좌인증은 인증기관/오픈뱅킹 이용계약 필요.
- **차단(Blocked) 정책**: 필수 외부 계약이나 인허가가 완료되기 전까지 해당 라이브 연동 기능은 `feature_flags` 및 `provider_catalog`에서 `BLOCKED`로 분류하며, 가짜 API 응답으로 상용 완료를 가장하는 행위를 엄격히 금지한다.

---

## 14. 성능 및 확장성 목표 (Performance & SLA Targets)

- **목표 지표**: 국세청 진위/상태 조회 응답 < 800ms, 권한 변경 즉시 Redis 캐시 무효화(< 50ms).

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

- **직접 매핑 범위**: `WO-MOS-0101 ~ WO-MOS-0200`
- **상세 개별 명세 위치**: `11_WORK_ORDERS_0001_TO_1000/individual/` 내 해당 번호 파일 및 `11_WORK_ORDERS_0001_TO_1000/WORK_ORDER_INDEX.csv` 참조.
