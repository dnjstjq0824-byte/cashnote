# I — Identity & Authorization (인증·소셜로그인·MFA·RBAC/ABAC·감사로그)

**문서 ID**: AZ-I_IDENTITY  
**프로젝트 코드명**: BN Merchant OS (`BN-MERCHANT-OS-GEMINI4-ARGON-MASTER-BLUEPRINT-1000-001`)  
**관련 Work Order 범위**: `WO-MOS-0101 ~ WO-MOS-0160`  
**설계 원칙**: 제0조~제32조 마스터 명령 전수 준수 (독립 플랫폼 원칙, Level 1~4 데이터 수집 전략, 미승인 외부 API 허위 구현 금지)

---

## 1. 목표 (Objective)

제6조 및 제18조에 따라 OIDC 호환 자체 인증 서버와 세분화된 다매장 권한 통제(RBAC + ABAC)를 구축한다. 단일 사용자 계정이 여러 사업자(`Business`)와 매장(`Store`)에 서로 다른 역할로 소속될 수 있는 멀티테넌트 ID 모델을 구현한다.

---

## 2. 사용자 역할 (User Roles & Actors)

- **주요 액터**: 전체 사용자(사장님, 매니저, 회계담당, 세무대리인), 보안 관리자, 시스템 관리자
- **권한 경계**: 모든 요청은 인증된 사용자(`user_id`)와 명시적으로 선택·검증된 사업자(`business_id`) 및 매장(`store_id`) 컨텍스트 하에서만 실행되며, 타 사업장 데이터에 대한 교차 접근은 애플리케이션 미들웨어와 PostgreSQL RLS(Row-Level Security) 양쪽에서 차단된다.

---

## 3. 기능 명세 (Functional Specifications)

- **핵심 기능 1**: 다중 인증 수단: 이메일+비밀번호(Argon2id 해싱), 휴대전화 본인인증, 카카오 OIDC 로그인, Google OIDC 로그인, Apple 로그인(iOS 필수 요건)
- **핵심 기능 2**: 다중요소인증(MFA): TOTP(Google/Microsoft Authenticator), SMS/알림톡 OTP, 모바일 생체인증(WebAuthn / Passkey / FIDO2) 지원. 특히 은행 계좌 연결, 정산 계좌 변경, 사업장 소유권 이전, 대량 데이터 내보내기 시 Step-up MFA 재인증 강제
- **핵심 기능 3**: RBAC + ABAC 하이브리드 인가 엔진: 역할(`OWNER`, `CO_OWNER`, `MANAGER`, `ACCOUNTANT`, `TAX_AGENT`, `STAFF`, `PARTNER_ADMIN`, `SUPER_ADMIN`)과 속성(소속 `store_id`, 데이터 민감도, 허용 기간 `valid_until`, IP 대역)을 결합하여 API 미들웨어에서 검증
- **핵심 기능 4**: 세션 관리: 기기별 활성 세션(`sessions`) 조회, 원격 강제 로그아웃, Refresh Token Rotation(탈취 감지 시 전체 토큰 패밀리 즉시 폐기)
- **핵심 기능 5**: 불변 감사 로그(`audit_logs`): 누가, 언제, 어떤 IP/기기에서, 어느 매장의 어떤 리소스를 조회/생성/수정/내보내기 했는지 해시 체인(`prev_log_hash` → `current_log_hash`)으로 위변조 불가능하게 기록

---

## 4. 사용자 시나리오 (End-to-End User Scenario)

> **실전 시나리오**: 외부 세무대리인 김세무사가 사장님의 초대를 수락하면 `ROLE_TAX_AGENT` 권한이 부여되어 해당 매장의 '세금·증빙·장부 내보내기' API만 호출할 수 있으며, '사장님 커뮤니티 작성'이나 '직원 권한 변경', 'B2B 마켓 결제' API 호출 시 `403 FORBIDDEN (INSUFFICIENT_SCOPE)`으로 즉시 차단된다.

1. **Pre-condition**: 사용자가 정상 로그인되어 있고 해당 매장에 대해 역할 권한(`RBAC/ABAC`)을 보유하고 있다. 외부 연동 기능의 경우 `provider_catalog` 및 `data_connections` 상태가 확인된다.
2. **Normal Flow**: 요청이 API Gateway를 통과하여 도메인 서비스에서 처리되고, 원본 출처(`data_source_level`)와 함께 PostgreSQL 트랜잭션 및 `outbox_events`에 기록된다.
3. **Fallback Flow (연동 차단/미계약 시)**: 만약 외부 API가 `CONTRACT_REQUIRED` 또는 `PARTNER_ONLY(BLOCKED)` 상태이면, 가짜 실시간 데이터를 표시하지 않고 즉시 **Level 2(공식 정산파일 CSV/Excel 업로드)** 또는 **Level 1(수기 입력)** 인터페이스로 안내하여 핵심 장부·대사 업무가 중단 없이 수행되도록 보장한다.

---

## 5. 백엔드 아키텍처 및 구현 명세 (Backend Architecture)

- **언어 및 프레임워크**: Go 1.22+, 모듈형 모놀리스 아키텍처 + 독립 비동기 워커(Go Workers)
- **핵심 구현 모듈**: `internal/identity` 패키지: OIDC Provider/Relying Party, JWT 서명기(RS256/ES256 비대칭키, JWKS 엔드포인트 제공), Argon2id 패스워드 해셔, Casbin/자체 정책 평가기(Policy Enforcement Point)
- **트랜잭션 관리**: 모든 상태 변경과 원장 반영은 `pgx/v5` 트랜잭션 내부에서 도메인 데이터 변경 + `audit_logs` 기록 + `outbox_events` 발행을 원자적(Atomic)으로 커밋한다.

---

## 6. 프론트엔드 및 모바일 명세 (Frontend & Mobile)

- **구현 범위**: 로그인/회원가입, 소셜 계정 연결 관리, MFA 등록/QR코드 화면, 기기/세션 관리 화면, 매장별 멤버 권한 설정 매트릭스 UI
- **상태 표현 표준**:
  1. `LOADING`: 스켈레톤 UI 및 진행률 표시
  2. `SUCCESS`: 데이터 표시 + 우측 상단 `DataProvenanceBadge`(데이터 출처 Level 1~4 및 최종 갱신 시각)
  3. `EMPTY`: 초기 온보딩 안내 및 첫 데이터 등록/업로드 CTA 버튼
  4. `ERROR`: 사용자 친화적 오류 메시지 + `request_id` 복사 + 재시도 버튼
  5. `BLOCKED`: 외부 기관 계약/승인 대기 상태의 정직한 고지 + 파일 업로드 대체 경로 버튼

---

## 7. 데이터베이스 설계 연동 (Database Schema Mapping)

- **연관 테이블**: `users`, `user_profiles`, `identities`, `sessions`, `roles`, `permissions`, `store_members`, `store_permissions`, `audit_logs`
- **데이터 무결성 규칙**:
  - 모든 금액 컬럼은 `NUMERIC(19,4)`를 사용하여 부동소수점 반올림 오차를 원천 배제한다.
  - 모든 시각 컬럼은 `TIMESTAMPTZ`로 저장한다.
  - 원본 거래 레코드는 임의 삭제(`DELETE`)하지 않으며, 정정 시 `corrections` 및 역분개(`journal_entries`)로 감사 추적성을 유지한다.

---

## 8. API 계약 명세 (OpenAPI 3.1 Endpoints)

- **주요 엔드포인트**: `POST /v1/auth/register`, `POST /v1/auth/login`, `POST /v1/auth/oauth/{provider}`, `POST /v1/auth/mfa/verify`, `POST /v1/auth/token/refresh`, `POST /v1/auth/logout`, `GET /v1/users/me/sessions`
- **공통 헤더**:
  - `Authorization: Bearer <OIDC_JWT>`
  - `X-Business-Id: <UUID>`
  - `X-Store-Id: <UUID>`
  - `Idempotency-Key: <UUID>` (모든 `POST`/`PUT`/`PATCH` 상태 변경 요청 시 필수)

---

## 9. 도메인 이벤트 (Domain Events)

- **발행/구독 이벤트**: `auth.user.registered`, `auth.login.succeeded`, `auth.login.failed_lockout`, `auth.mfa.challenged`, `auth.permission.changed`
- **전달 보장**: Transactional Outbox 테이블(`outbox_events`)에서 Redis Streams / Worker Queue로 At-Least-Once 전달을 보장하며, 컨슈머는 `event_id` 기준 멱등 처리(Idempotent Consumer)를 수행한다.

---

## 10. 데이터 출처 및 신뢰도 등급 (Data Provenance)

- **데이터 원천**: 사용자 직접 입력, 카카오/구글/애플 OIDC 토큰, 본인인증 CI/DI
- **출처 태깅 규칙**:
  - `LEVEL_1_MANUAL`: 사용자 직접 입력 (신뢰도: 사용자 책임, 즉시 반영)
  - `LEVEL_2_FILE`: 공식 정산 엑셀/CSV 업로드 (신뢰도: 파일 해시 및 포맷 검증됨)
  - `LEVEL_3_API`: 공식 계약/승인된 외부 API (신뢰도: 기관 전자서명/TLS 검증됨)
  - `LEVEL_4_RECONCILED`: 복수 데이터 소스(예: 카드매입 + 은행입금) 교차 대사 완료 (신뢰도: 최고 등급 확정)

---

## 11. 오류 처리 및 장애 복구 (Error Handling & Resilience)

- **예외 처리 정책**: 연속 5회 비밀번호/OTP 오입력 시 15분간 계정 잠금 및 본인 알림톡 발송; 새로운 국가/기기 로그인 시 보안 경고 발송
- **표준 에러 코드**: `AUTH_FORBIDDEN_STORE`, `CONNECTOR_EXTERNAL_BLOCKED`, `LEDGER_UNBALANCED_ENTRY`, `IMPORT_DUPLICATE_BATCH`, `RATE_LIMIT_EXCEEDED`, `UPSTREAM_TIMEOUT_RETRYABLE`

---

## 12. 개인정보 및 보안 통제 (Privacy & Security Controls)

- **보안 설계**: 비밀번호 원문 로그 기록 절대 금지, JWT Access Token 수명 15분 이내 단기 유지, 모든 권한 검증은 프론트엔드 숨김 처리에 의존하지 않고 백엔드 DB/서비스 계층에서 강제(Server-side Enforcement).
- **감사 추적**: 모든 민감 데이터 조회 및 변경은 `audit_logs`에 요청자 ID, 대상 매장 ID, 액션 코드, IP, User-Agent, 변경 전후 요약 및 해시 체인과 함께 기록된다.

---

## 13. 외부 승인 및 법적 선행 요건 (External Approval & Regulatory Requirements)

- **승인·규제 요건**: 카카오/구글/애플 개발자 콘솔 앱 등록 및 OIDC 심사 필요. 애플 로그인은 iOS 앱 스토어 심사 가이드라인(소셜 로그인 제공 시 필수 포함) 준수.
- **차단(Blocked) 정책**: 필수 외부 계약이나 인허가가 완료되기 전까지 해당 라이브 연동 기능은 `feature_flags` 및 `provider_catalog`에서 `BLOCKED`로 분류하며, 가짜 API 응답으로 상용 완료를 가장하는 행위를 엄격히 금지한다.

---

## 14. 성능 및 확장성 목표 (Performance & SLA Targets)

- **목표 지표**: JWT/권한 미들웨어 검증 시간 < 3ms, 로그인 API P95 < 200ms.

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

- **직접 매핑 범위**: `WO-MOS-0101 ~ WO-MOS-0160`
- **상세 개별 명세 위치**: `11_WORK_ORDERS_0001_TO_1000/individual/` 내 해당 번호 파일 및 `11_WORK_ORDERS_0001_TO_1000/WORK_ORDER_INDEX.csv` 참조.
