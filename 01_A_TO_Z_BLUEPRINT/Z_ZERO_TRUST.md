# Z — Zero Trust & Release (보안·데이터 무결성·독립 감사·상용 출시 검증 게이트)

**문서 ID**: AZ-Z_ZERO_TRUST  
**프로젝트 코드명**: BN Merchant OS (`BN-MERCHANT-OS-GEMINI4-ARGON-MASTER-BLUEPRINT-1000-001`)  
**관련 Work Order 범위**: `WO-MOS-0951 ~ WO-MOS-1000`  
**설계 원칙**: 제0조~제32조 마스터 명령 전수 준수 (독립 플랫폼 원칙, Level 1~4 데이터 수집 전략, 미승인 외부 API 허위 구현 금지)

---

## 1. 목표 (Objective)

제5조 Z항, 제28조, 제29조, 제31조에 따라 '어떠한 내부 네트워크, 에이전트, 외부 입력도 기본적으로 신뢰하지 않는다(Never Trust, Always Verify)'는 Zero-Trust 보안 원칙과 상용 프로덕션 출시 전 반드시 통과해야 하는 20대 Release Gate 및 독립 감사(Independent Audit) 절차를 정의한다.

---

## 2. 사용자 역할 (User Roles & Actors)

- **주요 액터**: Independent Audit Agent, Security & Compliance Lead, Lead Architect, Release Manager, Booking Network 총괄 PO
- **권한 경계**: 모든 요청은 인증된 사용자(`user_id`)와 명시적으로 선택·검증된 사업자(`business_id`) 및 매장(`store_id`) 컨텍스트 하에서만 실행되며, 타 사업장 데이터에 대한 교차 접근은 애플리케이션 미들웨어와 PostgreSQL RLS(Row-Level Security) 양쪽에서 차단된다.

---

## 3. 기능 명세 (Functional Specifications)

- **핵심 기능 1**: Zero-Trust 5대 아키텍처 원칙: (1) 모든 API/워커 요청마다 JWT 서명 + 세션 활성 여부 + `store_id` RBAC/ABAC 권한 + PostgreSQL RLS 이중 검증, (2) 내부 서비스 간 mTLS 및 최소 권한 IAM Role, (3) 금융 원장 및 감사 로그의 암호학적 무결성(SHA-256 해시 체인), (4) 시크릿 하드코딩 제로(전체 Vault/KMS 동적 주입), (5) 모든 외부 입력(API 응답, 엑셀 업로드, 웹훅)을 잠재적 오염 데이터로 간주하고 스키마·체크섬·범위 검증 수행
- **핵심 기능 2**: 제31조 17대 절대 금지 사항 자동 감사 스캐너: 코드베이스 및 UI 문구에 허위 실시간 연동 표시, 비밀번호 평문 수집 필드, 감사 로그 `DELETE/UPDATE` 구문, 타사 브랜드 자산 복제, 하드코딩된 가짜 Git SHA/테스트 PASS가 존재하는지 정적/동적 분석으로 전수 검사
- **핵심 기능 3**: 독립 감사(Independent Audit) 프로토콜: 구현 에이전트가 제출한 산출물을 별도의 `Independent Audit Agent`가 인수하여 SQL DDL 실제 실행, OpenAPI 3.1 스키마 린트, 1,000개 Work Order 무결성, E2E 테스트 로그 및 `FILE_CHECKSUMS.sha256` 일치 여부를 교차 검증
- **핵심 기능 4**: 프로덕션 릴리스 승인 게이트(Go/No-Go Release Checklist): 법무/규제 게이트, 회계 대사 정합성 게이트, 보안 침투 테스트 게이트, 백업/복구 드릴 게이트 통과 및 PO 최종 서명 전까지 프로덕션 배포 차단

---

## 4. 사용자 시나리오 (End-to-End User Scenario)

> **실전 시나리오**: 릴리스 후보(Release Candidate) 빌드가 생성되면 자동 감사 파이프라인이 가동되어 56개 DB 테이블의 RLS 활성화 여부, 감사 로그 테이블의 `DELETE` 권한 차단 여부, 미승인 외부 커넥터의 `BLOCKED` 상태 유지 여부를 검증하고, 이상이 없을 때만 `RELEASE_AUDIT_CERTIFICATE`를 발행한다.

1. **Pre-condition**: 사용자가 정상 로그인되어 있고 해당 매장에 대해 역할 권한(`RBAC/ABAC`)을 보유하고 있다. 외부 연동 기능의 경우 `provider_catalog` 및 `data_connections` 상태가 확인된다.
2. **Normal Flow**: 요청이 API Gateway를 통과하여 도메인 서비스에서 처리되고, 원본 출처(`data_source_level`)와 함께 PostgreSQL 트랜잭션 및 `outbox_events`에 기록된다.
3. **Fallback Flow (연동 차단/미계약 시)**: 만약 외부 API가 `CONTRACT_REQUIRED` 또는 `PARTNER_ONLY(BLOCKED)` 상태이면, 가짜 실시간 데이터를 표시하지 않고 즉시 **Level 2(공식 정산파일 CSV/Excel 업로드)** 또는 **Level 1(수기 입력)** 인터페이스로 안내하여 핵심 장부·대사 업무가 중단 없이 수행되도록 보장한다.

---

## 5. 백엔드 아키텍처 및 구현 명세 (Backend Architecture)

- **언어 및 프레임워크**: Go 1.22+, 모듈형 모놀리스 아키텍처 + 독립 비동기 워커(Go Workers)
- **핵심 구현 모듈**: `internal/audit` 및 `scripts/verify_blueprint.py`: 원장 해시 체인 검증기, RLS 정책 전수 검사 쿼리, 시크릿/금칙어 정적 스캐너
- **트랜잭션 관리**: 모든 상태 변경과 원장 반영은 `pgx/v5` 트랜잭션 내부에서 도메인 데이터 변경 + `audit_logs` 기록 + `outbox_events` 발행을 원자적(Atomic)으로 커밋한다.

---

## 6. 프론트엔드 및 모바일 명세 (Frontend & Mobile)

- **구현 범위**: 출시 준비 상태(Release Readiness) 대시보드 및 미승인/차단(`BLOCKED`) 기능 투명성 공지 배너
- **상태 표현 표준**:
  1. `LOADING`: 스켈레톤 UI 및 진행률 표시
  2. `SUCCESS`: 데이터 표시 + 우측 상단 `DataProvenanceBadge`(데이터 출처 Level 1~4 및 최종 갱신 시각)
  3. `EMPTY`: 초기 온보딩 안내 및 첫 데이터 등록/업로드 CTA 버튼
  4. `ERROR`: 사용자 친화적 오류 메시지 + `request_id` 복사 + 재시도 버튼
  5. `BLOCKED`: 외부 기관 계약/승인 대기 상태의 정직한 고지 + 파일 업로드 대체 경로 버튼

---

## 7. 데이터베이스 설계 연동 (Database Schema Mapping)

- **연관 테이블**: `audit_logs`, `admin_actions`, `feature_flags`
- **데이터 무결성 규칙**:
  - 모든 금액 컬럼은 `NUMERIC(19,4)`를 사용하여 부동소수점 반올림 오차를 원천 배제한다.
  - 모든 시각 컬럼은 `TIMESTAMPTZ`로 저장한다.
  - 원본 거래 레코드는 임의 삭제(`DELETE`)하지 않으며, 정정 시 `corrections` 및 역분개(`journal_entries`)로 감사 추적성을 유지한다.

---

## 8. API 계약 명세 (OpenAPI 3.1 Endpoints)

- **주요 엔드포인트**: `GET /v1/admin/audit/integrity-verify`, `POST /v1/admin/release-gates/evaluate`
- **공통 헤더**:
  - `Authorization: Bearer <OIDC_JWT>`
  - `X-Business-Id: <UUID>`
  - `X-Store-Id: <UUID>`
  - `Idempotency-Key: <UUID>` (모든 `POST`/`PUT`/`PATCH` 상태 변경 요청 시 필수)

---

## 9. 도메인 이벤트 (Domain Events)

- **발행/구독 이벤트**: `audit.integrity.verified`, `audit.violation.critical`, `release.gate.approved`
- **전달 보장**: Transactional Outbox 테이블(`outbox_events`)에서 Redis Streams / Worker Queue로 At-Least-Once 전달을 보장하며, 컨슈머는 `event_id` 기준 멱등 처리(Idempotent Consumer)를 수행한다.

---

## 10. 데이터 출처 및 신뢰도 등급 (Data Provenance)

- **데이터 원천**: 전체 시스템 코드베이스, DB 카탈로그(`pg_policies`, `information_schema`), CI/CD 파이프라인 아티팩트
- **출처 태깅 규칙**:
  - `LEVEL_1_MANUAL`: 사용자 직접 입력 (신뢰도: 사용자 책임, 즉시 반영)
  - `LEVEL_2_FILE`: 공식 정산 엑셀/CSV 업로드 (신뢰도: 파일 해시 및 포맷 검증됨)
  - `LEVEL_3_API`: 공식 계약/승인된 외부 API (신뢰도: 기관 전자서명/TLS 검증됨)
  - `LEVEL_4_RECONCILED`: 복수 데이터 소스(예: 카드매입 + 은행입금) 교차 대사 완료 (신뢰도: 최고 등급 확정)

---

## 11. 오류 처리 및 장애 복구 (Error Handling & Resilience)

- **예외 처리 정책**: 감사 로그 해시 체인 불일치나 RLS 비활성화 테이블이 단 1개라도 발견되면 즉시 프로덕션 배포 파이프라인을 중단(Hard Stop)하고 최고 보안 경보 발령
- **표준 에러 코드**: `AUTH_FORBIDDEN_STORE`, `CONNECTOR_EXTERNAL_BLOCKED`, `LEDGER_UNBALANCED_ENTRY`, `IMPORT_DUPLICATE_BATCH`, `RATE_LIMIT_EXCEEDED`, `UPSTREAM_TIMEOUT_RETRYABLE`

---

## 12. 개인정보 및 보안 통제 (Privacy & Security Controls)

- **보안 설계**: Zero-Trust 원칙에 따라 개발자나 DBA 개인 계정에 프로덕션 DB 직접 접속 권한을 상시 부여하지 않으며, 한시적 JIT(Just-In-Time) 승인 및 전 쿼리 레코딩 프록시를 경유.
- **감사 추적**: 모든 민감 데이터 조회 및 변경은 `audit_logs`에 요청자 ID, 대상 매장 ID, 액션 코드, IP, User-Agent, 변경 전후 요약 및 해시 체인과 함께 기록된다.

---

## 13. 외부 승인 및 법적 선행 요건 (External Approval & Regulatory Requirements)

- **승인·규제 요건**: 금융·개인정보 법률 검토 완료 확인서 및 외부 API 이용계약서 사본이 등록된 기능만 프로덕션 피처 플래그 활성화 허용.
- **차단(Blocked) 정책**: 필수 외부 계약이나 인허가가 완료되기 전까지 해당 라이브 연동 기능은 `feature_flags` 및 `provider_catalog`에서 `BLOCKED`로 분류하며, 가짜 API 응답으로 상용 완료를 가장하는 행위를 엄격히 금지한다.

---

## 14. 성능 및 확장성 목표 (Performance & SLA Targets)

- **목표 지표**: 일일 전체 원장 및 감사 로그 무결성 검증 배치 < 60초.

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

- **직접 매핑 범위**: `WO-MOS-0951 ~ WO-MOS-1000`
- **상세 개별 명세 위치**: `11_WORK_ORDERS_0001_TO_1000/individual/` 내 해당 번호 파일 및 `11_WORK_ORDERS_0001_TO_1000/WORK_ORDER_INDEX.csv` 참조.
