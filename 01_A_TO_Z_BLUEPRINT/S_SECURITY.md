# S — Security & Compliance (개인정보·신용정보·마이데이터·전자금융 규제 및 보안 통제)

**문서 ID**: AZ-S_SECURITY  
**프로젝트 코드명**: BN Merchant OS (`BN-MERCHANT-OS-GEMINI4-ARGON-MASTER-BLUEPRINT-1000-001`)  
**관련 Work Order 범위**: `WO-MOS-0881 ~ WO-MOS-0900, WO-MOS-0935 ~ WO-MOS-0965`  
**설계 원칙**: 제0조~제32조 마스터 명령 전수 준수 (독립 플랫폼 원칙, Level 1~4 데이터 수집 전략, 미승인 외부 API 허위 구현 금지)

---

## 1. 목표 (Objective)

제21조 및 제31조에 따라 개인정보보호법, 신용정보의 이용 및 보호에 관한 법률(신용정보법), 전자금융거래법, 금융소비자보호법, 부가가치세법(전자세금계산서)의 법적 요구사항을 아키텍처와 코드 레벨의 강제 통제 게이트(Regulatory Gate)로 구현한다.

---

## 2. 사용자 역할 (User Roles & Actors)

- **주요 액터**: Security & Compliance Lead, Legal Counsel, Data Protection Officer(DPO), Independent Audit Agent
- **권한 경계**: 모든 요청은 인증된 사용자(`user_id`)와 명시적으로 선택·검증된 사업자(`business_id`) 및 매장(`store_id`) 컨텍스트 하에서만 실행되며, 타 사업장 데이터에 대한 교차 접근은 애플리케이션 미들웨어와 PostgreSQL RLS(Row-Level Security) 양쪽에서 차단된다.

---

## 3. 기능 명세 (Functional Specifications)

- **핵심 기능 1**: 업무 범위별 인허가·등록·계약 매트릭스(`REGULATORY_GATE.md` 연동): 모든 사업자 장부 서비스가 자동으로 본인신용정보관리업(금융 마이데이터) 허가 대상은 아니나, 개인신용정보 전송요구권 행사 대행이나 개인사업자 CB(신용등급 산출), 온라인 대출 중개(금소법), 전자지급결제대행(전금법 PG) 영역은 각 라이선스 취득 전까지 시스템 피처 플래그(`feature_flags`)에서 `REGULATORY_BLOCKED`로 강제 잠금
- **핵심 기능 2**: 무단 크래덴셜 수집 및 보호장치 우회 금지: 금융기관/홈택스/배달앱의 로그인 마스터 비밀번호나 공동인증서 개인키·암호를 서버에 평문 저장하거나 캡차/보안프로그램을 우회하는 모듈 설계 전면 금지
- **핵심 기능 3**: 암호화 표준: 전송 구간 TLS 1.3(내부 마이크로서비스/워커 간 mTLS), 저장 구간 AES-256-GCM 봉투 암호화(KMS CMK 기반 DEK 분리, 테넌트별 암호화 키 컨텍스트 바인딩), 비밀번호 Argon2id
- **핵심 기능 4**: 개인정보·데이터 주권 수명주기: 수집·이용 동의(`connection_consents`), 제3자 제공 동의 분리(필수/선택 명확 구분), 동의 철회 시 외부 연동 토큰 즉시 폐기(Revoke) 및 수집 중단, 회원 탈퇴 시 법정 보존의무 데이터 분리 보관(`isolated_retention_vault`) 후 영구 파기
- **핵심 기능 5**: 침해사고 탐지 및 대응: 비정상 대량 조회(Scraping/Exfiltration 시도), 타 매장 ID 열람 시도(`IDOR`), 권한 상승 시도 감지 시 즉시 세션 차단 및 보안팀 PagerDuty/알림 발송

---

## 4. 사용자 시나리오 (End-to-End User Scenario)

> **실전 시나리오**: 사장님이 '오픈뱅킹 계좌 연동 철회' 버튼을 누르면, 1초 이내에 금융결제원/중계기관 토큰 폐기 API가 호출되고 `data_connections.status = REVOKED`로 전환되며, 향후 스케줄러 수집 대상에서 즉시 제외되고 철회 확인서가 발급된다.

1. **Pre-condition**: 사용자가 정상 로그인되어 있고 해당 매장에 대해 역할 권한(`RBAC/ABAC`)을 보유하고 있다. 외부 연동 기능의 경우 `provider_catalog` 및 `data_connections` 상태가 확인된다.
2. **Normal Flow**: 요청이 API Gateway를 통과하여 도메인 서비스에서 처리되고, 원본 출처(`data_source_level`)와 함께 PostgreSQL 트랜잭션 및 `outbox_events`에 기록된다.
3. **Fallback Flow (연동 차단/미계약 시)**: 만약 외부 API가 `CONTRACT_REQUIRED` 또는 `PARTNER_ONLY(BLOCKED)` 상태이면, 가짜 실시간 데이터를 표시하지 않고 즉시 **Level 2(공식 정산파일 CSV/Excel 업로드)** 또는 **Level 1(수기 입력)** 인터페이스로 안내하여 핵심 장부·대사 업무가 중단 없이 수행되도록 보장한다.

---

## 5. 백엔드 아키텍처 및 구현 명세 (Backend Architecture)

- **언어 및 프레임워크**: Go 1.22+, 모듈형 모놀리스 아키텍처 + 독립 비동기 워커(Go Workers)
- **핵심 구현 모듈**: `internal/security` 패키지: KMS 봉투 암호화 모듈, PII 마스킹 로거(구조화 로그 출력 시 전화번호·계좌번호·사업자번호·이메일 자동 마스킹), IDOR 침투 감지기, 동의서 버전 관리기
- **트랜잭션 관리**: 모든 상태 변경과 원장 반영은 `pgx/v5` 트랜잭션 내부에서 도메인 데이터 변경 + `audit_logs` 기록 + `outbox_events` 발행을 원자적(Atomic)으로 커밋한다.

---

## 6. 프론트엔드 및 모바일 명세 (Frontend & Mobile)

- **구현 범위**: 개인정보 및 데이터 연결 동의 관리 센터(동의 내역 조회, 동의서 전문 다운로드, 원클릭 연결 철회, 계정 탈퇴 및 데이터 파기 요청 화면), 관리자 보안 감사 대시보드
- **상태 표현 표준**:
  1. `LOADING`: 스켈레톤 UI 및 진행률 표시
  2. `SUCCESS`: 데이터 표시 + 우측 상단 `DataProvenanceBadge`(데이터 출처 Level 1~4 및 최종 갱신 시각)
  3. `EMPTY`: 초기 온보딩 안내 및 첫 데이터 등록/업로드 CTA 버튼
  4. `ERROR`: 사용자 친화적 오류 메시지 + `request_id` 복사 + 재시도 버튼
  5. `BLOCKED`: 외부 기관 계약/승인 대기 상태의 정직한 고지 + 파일 업로드 대체 경로 버튼

---

## 7. 데이터베이스 설계 연동 (Database Schema Mapping)

- **연관 테이블**: `connection_consents`, `audit_logs`, `admin_actions`, `feature_flags`
- **데이터 무결성 규칙**:
  - 모든 금액 컬럼은 `NUMERIC(19,4)`를 사용하여 부동소수점 반올림 오차를 원천 배제한다.
  - 모든 시각 컬럼은 `TIMESTAMPTZ`로 저장한다.
  - 원본 거래 레코드는 임의 삭제(`DELETE`)하지 않으며, 정정 시 `corrections` 및 역분개(`journal_entries`)로 감사 추적성을 유지한다.

---

## 8. API 계약 명세 (OpenAPI 3.1 Endpoints)

- **주요 엔드포인트**: `GET /v1/users/me/consents`, `POST /v1/users/me/consents/{consentId}/revoke`, `POST /v1/users/me/account-deletion`, `GET /v1/admin/security/audit-logs`
- **공통 헤더**:
  - `Authorization: Bearer <OIDC_JWT>`
  - `X-Business-Id: <UUID>`
  - `X-Store-Id: <UUID>`
  - `Idempotency-Key: <UUID>` (모든 `POST`/`PUT`/`PATCH` 상태 변경 요청 시 필수)

---

## 9. 도메인 이벤트 (Domain Events)

- **발행/구독 이벤트**: `security.consent.granted`, `security.consent.revoked`, `security.idor_attempt.blocked`, `security.token.rotated`
- **전달 보장**: Transactional Outbox 테이블(`outbox_events`)에서 Redis Streams / Worker Queue로 At-Least-Once 전달을 보장하며, 컨슈머는 `event_id` 기준 멱등 처리(Idempotent Consumer)를 수행한다.

---

## 10. 데이터 출처 및 신뢰도 등급 (Data Provenance)

- **데이터 원천**: 사용자 동의 이벤트, API 게이트웨이 접근 로그, DB 감사 로그
- **출처 태깅 규칙**:
  - `LEVEL_1_MANUAL`: 사용자 직접 입력 (신뢰도: 사용자 책임, 즉시 반영)
  - `LEVEL_2_FILE`: 공식 정산 엑셀/CSV 업로드 (신뢰도: 파일 해시 및 포맷 검증됨)
  - `LEVEL_3_API`: 공식 계약/승인된 외부 API (신뢰도: 기관 전자서명/TLS 검증됨)
  - `LEVEL_4_RECONCILED`: 복수 데이터 소스(예: 카드매입 + 은행입금) 교차 대사 완료 (신뢰도: 최고 등급 확정)

---

## 11. 오류 처리 및 장애 복구 (Error Handling & Resilience)

- **예외 처리 정책**: KMS 또는 감사 로그 기록 실패 시 금융 데이터 조회/변경 요청 자체를 Fail-Closed(안전 차단)로 처리하여 감사 로그 없는 무단 열람 원천 방지
- **표준 에러 코드**: `AUTH_FORBIDDEN_STORE`, `CONNECTOR_EXTERNAL_BLOCKED`, `LEDGER_UNBALANCED_ENTRY`, `IMPORT_DUPLICATE_BATCH`, `RATE_LIMIT_EXCEEDED`, `UPSTREAM_TIMEOUT_RETRYABLE`

---

## 12. 개인정보 및 보안 통제 (Privacy & Security Controls)

- **보안 설계**: 운영자(`SUPER_ADMIN`)라 할지라도 가맹점주의 명시적 고객지원 열람 동의 티켓 번호(`support_ticket_id`)와 사유를 입력하지 않으면 개별 매장의 상세 매출·계좌 화면을 열람할 수 없도록 Break-Glass 통제 적용.
- **감사 추적**: 모든 민감 데이터 조회 및 변경은 `audit_logs`에 요청자 ID, 대상 매장 ID, 액션 코드, IP, User-Agent, 변경 전후 요약 및 해시 체인과 함께 기록된다.

---

## 13. 외부 승인 및 법적 선행 요건 (External Approval & Regulatory Requirements)

- **승인·규제 요건**: 개인정보보호법 제28조의2(가명정보), 제29조(안전조치의무), 신용정보법, 전자금융감독규정 보안성 기준 준수.
- **차단(Blocked) 정책**: 필수 외부 계약이나 인허가가 완료되기 전까지 해당 라이브 연동 기능은 `feature_flags` 및 `provider_catalog`에서 `BLOCKED`로 분류하며, 가짜 API 응답으로 상용 완료를 가장하는 행위를 엄격히 금지한다.

---

## 14. 성능 및 확장성 목표 (Performance & SLA Targets)

- **목표 지표**: 암호화/복호화 오버헤드 < 2ms, 감사 로그 해시 체인 검증 배치 일 1회 무결성 100%.

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

- **직접 매핑 범위**: `WO-MOS-0881 ~ WO-MOS-0900, WO-MOS-0935 ~ WO-MOS-0965`
- **상세 개별 명세 위치**: `11_WORK_ORDERS_0001_TO_1000/individual/` 내 해당 번호 파일 및 `11_WORK_ORDERS_0001_TO_1000/WORK_ORDER_INDEX.csv` 참조.
