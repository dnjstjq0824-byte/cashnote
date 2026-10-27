# U — UX/UI (사용자 대시보드·사업장 관리·관리자 콘솔 48개 화면 및 디자인 시스템)

**문서 ID**: AZ-U_UXUI  
**프로젝트 코드명**: BN Merchant OS (`BN-MERCHANT-OS-GEMINI4-ARGON-MASTER-BLUEPRINT-1000-001`)  
**관련 Work Order 범위**: `WO-MOS-0501 ~ WO-MOS-0600, WO-MOS-0701 ~ WO-MOS-0750`  
**설계 원칙**: 제0조~제32조 마스터 명령 전수 준수 (독립 플랫폼 원칙, Level 1~4 데이터 수집 전략, 미승인 외부 API 허위 구현 금지)

---

## 1. 목표 (Objective)

제17조에 명시된 사용자 대시보드(22개 화면), 사업장 관리(9개 화면), 관리자 콘솔(17개 화면) 등 총 48개 이상의 핵심 화면을 빠짐없이 설계한다. 캐시노트 이상의 직관성과 사용 편의성을 제공하되, 고유한 브랜드·디자인 자산은 독자적인 `BN Merchant Design System (Argon Slate & Trust Emerald)`으로 구축한다.

---

## 2. 사용자 역할 (User Roles & Actors)

- **주요 액터**: Product/UX Agent, Frontend Agent, Mobile Flutter Agent, 전체 사용자
- **권한 경계**: 모든 요청은 인증된 사용자(`user_id`)와 명시적으로 선택·검증된 사업자(`business_id`) 및 매장(`store_id`) 컨텍스트 하에서만 실행되며, 타 사업장 데이터에 대한 교차 접근은 애플리케이션 미들웨어와 PostgreSQL RLS(Row-Level Security) 양쪽에서 차단된다.

---

## 3. 기능 명세 (Functional Specifications)

- **핵심 기능 1**: 48개 핵심 화면 인벤토리 완비 (`05_UXUI/SCREEN_INVENTORY.csv` 및 `05_UXUI/*.md`에 와이어프레임·컴포넌트·API·DB·권한·상태별 명세 전수 수록)
- **핵심 기능 2**: 재무 숫자 가독성 원칙: 모든 통화 숫자는 고정폭 숫자 폰트(`tabular-nums`, Pretendard / Inter Variable)로 우측 정렬하며, 승인액(파란색 계열)·정산예정액(청록색 계열)·실제입금액(에메랄드 그린)·미입금/보류(앰버/코랄 레드)·비용(슬레이트/로즈)의 시맨틱 컬러 코딩을 전 화면 일관 적용
- **핵심 기능 3**: Progressive Disclosure(점진적 정보 공개): 1차 화면에서는 사장님이 가장 궁금해하는 핵심 요약 3가지(오늘 들어올 돈, 어제 번 돈, 확인 필요한 이상 징후)를 3초 안에 파악하게 하고, 탭/클릭 시 전표 단위 상세 원장과 대사 증빙까지 드릴다운
- **핵심 기능 4**: 정직한 데이터 상태 시각화(Zero Fake UI): 연동되지 않은 계좌나 배달앱 화면에 가짜 샘플 그래프를 띄워 연동된 것처럼 오인하게 만드는 디자인을 금지하고, 명확한 `연동 전 / 파일 업로드 대기 / 계약 승인 대기(BLOCKED)` 상태 카드와 대체 행동 버튼(엑셀 업로드 또는 수기 입력)을 제공

---

## 4. 사용자 시나리오 (End-to-End User Scenario)

> **실전 시나리오**: 60대 식당 사장님이 스마트폰에서 '큰 글씨 모드'를 켜면 본문 폰트가 18px 이상, 핵심 매출 숫자가 28px 고대비로 확대되고, 복잡한 회계 용어 대신 '오늘 통장에 들어올 카드값', '아직 안 들어온 카드값 확인하기'와 같이 쉬운 일상어 툴팁이 함께 표시된다.

1. **Pre-condition**: 사용자가 정상 로그인되어 있고 해당 매장에 대해 역할 권한(`RBAC/ABAC`)을 보유하고 있다. 외부 연동 기능의 경우 `provider_catalog` 및 `data_connections` 상태가 확인된다.
2. **Normal Flow**: 요청이 API Gateway를 통과하여 도메인 서비스에서 처리되고, 원본 출처(`data_source_level`)와 함께 PostgreSQL 트랜잭션 및 `outbox_events`에 기록된다.
3. **Fallback Flow (연동 차단/미계약 시)**: 만약 외부 API가 `CONTRACT_REQUIRED` 또는 `PARTNER_ONLY(BLOCKED)` 상태이면, 가짜 실시간 데이터를 표시하지 않고 즉시 **Level 2(공식 정산파일 CSV/Excel 업로드)** 또는 **Level 1(수기 입력)** 인터페이스로 안내하여 핵심 장부·대사 업무가 중단 없이 수행되도록 보장한다.

---

## 5. 백엔드 아키텍처 및 구현 명세 (Backend Architecture)

- **언어 및 프레임워크**: Go 1.22+, 모듈형 모놀리스 아키텍처 + 독립 비동기 워커(Go Workers)
- **핵심 구현 모듈**: API 응답에 UI 렌더링 힌트(`data_freshness_status`, `source_badge_type`, `actionable_fallback`) 포함
- **트랜잭션 관리**: 모든 상태 변경과 원장 반영은 `pgx/v5` 트랜잭션 내부에서 도메인 데이터 변경 + `audit_logs` 기록 + `outbox_events` 발행을 원자적(Atomic)으로 커밋한다.

---

## 6. 프론트엔드 및 모바일 명세 (Frontend & Mobile)

- **구현 범위**: Figma/Storybook 호환 디자인 토큰(`tokens.json`: Color, Typography, Spacing, Elevation, BorderRadius), 반응형 Grid(Mobile 4-col, Tablet 8-col, Desktop 12-col)
- **상태 표현 표준**:
  1. `LOADING`: 스켈레톤 UI 및 진행률 표시
  2. `SUCCESS`: 데이터 표시 + 우측 상단 `DataProvenanceBadge`(데이터 출처 Level 1~4 및 최종 갱신 시각)
  3. `EMPTY`: 초기 온보딩 안내 및 첫 데이터 등록/업로드 CTA 버튼
  4. `ERROR`: 사용자 친화적 오류 메시지 + `request_id` 복사 + 재시도 버튼
  5. `BLOCKED`: 외부 기관 계약/승인 대기 상태의 정직한 고지 + 파일 업로드 대체 경로 버튼

---

## 7. 데이터베이스 설계 연동 (Database Schema Mapping)

- **연관 테이블**: 화면별 매핑된 도메인 테이블 (`05_UXUI/SCREEN_INVENTORY.csv` 참조)
- **데이터 무결성 규칙**:
  - 모든 금액 컬럼은 `NUMERIC(19,4)`를 사용하여 부동소수점 반올림 오차를 원천 배제한다.
  - 모든 시각 컬럼은 `TIMESTAMPTZ`로 저장한다.
  - 원본 거래 레코드는 임의 삭제(`DELETE`)하지 않으며, 정정 시 `corrections` 및 역분개(`journal_entries`)로 감사 추적성을 유지한다.

---

## 8. API 계약 명세 (OpenAPI 3.1 Endpoints)

- **주요 엔드포인트**: 화면별 바인딩된 OpenAPI 엔드포인트 (`05_UXUI/SCREEN_INVENTORY.csv` 참조)
- **공통 헤더**:
  - `Authorization: Bearer <OIDC_JWT>`
  - `X-Business-Id: <UUID>`
  - `X-Store-Id: <UUID>`
  - `Idempotency-Key: <UUID>` (모든 `POST`/`PUT`/`PATCH` 상태 변경 요청 시 필수)

---

## 9. 도메인 이벤트 (Domain Events)

- **발행/구독 이벤트**: 사용자 인터랙션 이벤트 및 실시간 SSE 데이터 갱신 이벤트
- **전달 보장**: Transactional Outbox 테이블(`outbox_events`)에서 Redis Streams / Worker Queue로 At-Least-Once 전달을 보장하며, 컨슈머는 `event_id` 기준 멱등 처리(Idempotent Consumer)를 수행한다.

---

## 10. 데이터 출처 및 신뢰도 등급 (Data Provenance)

- **데이터 원천**: API 응답 및 로컬 사용자 설정
- **출처 태깅 규칙**:
  - `LEVEL_1_MANUAL`: 사용자 직접 입력 (신뢰도: 사용자 책임, 즉시 반영)
  - `LEVEL_2_FILE`: 공식 정산 엑셀/CSV 업로드 (신뢰도: 파일 해시 및 포맷 검증됨)
  - `LEVEL_3_API`: 공식 계약/승인된 외부 API (신뢰도: 기관 전자서명/TLS 검증됨)
  - `LEVEL_4_RECONCILED`: 복수 데이터 소스(예: 카드매입 + 은행입금) 교차 대사 완료 (신뢰도: 최고 등급 확정)

---

## 11. 오류 처리 및 장애 복구 (Error Handling & Resilience)

- **예외 처리 정책**: 모든 폼 입력에서 서버 측 유효성 검사 오류(`422 Unprocessable Entity`)가 반환되면 해당 입력 필드 하단에 구체적인 한국어 해결 방법을 포커스와 함께 즉시 표시
- **표준 에러 코드**: `AUTH_FORBIDDEN_STORE`, `CONNECTOR_EXTERNAL_BLOCKED`, `LEDGER_UNBALANCED_ENTRY`, `IMPORT_DUPLICATE_BATCH`, `RATE_LIMIT_EXCEEDED`, `UPSTREAM_TIMEOUT_RETRYABLE`

---

## 12. 개인정보 및 보안 통제 (Privacy & Security Controls)

- **보안 설계**: 공용 PC나 매장 포스기에서 사용 시 10분간 입력이 없으면 화면 숫자를 블러 처리하는 '자리비움 금액 가리기(Privacy Mode)' 단축키 지원.
- **감사 추적**: 모든 민감 데이터 조회 및 변경은 `audit_logs`에 요청자 ID, 대상 매장 ID, 액션 코드, IP, User-Agent, 변경 전후 요약 및 해시 체인과 함께 기록된다.

---

## 13. 외부 승인 및 법적 선행 요건 (External Approval & Regulatory Requirements)

- **승인·규제 요건**: 캐시노트 상표·로고·일러스트·전용 서체 사용 일체 금지.
- **차단(Blocked) 정책**: 필수 외부 계약이나 인허가가 완료되기 전까지 해당 라이브 연동 기능은 `feature_flags` 및 `provider_catalog`에서 `BLOCKED`로 분류하며, 가짜 API 응답으로 상용 완료를 가장하는 행위를 엄격히 금지한다.

---

## 14. 성능 및 확장성 목표 (Performance & SLA Targets)

- **목표 지표**: 모든 화면 초기 인터랙티브 도달 시간(TTI) < 1.5초, 모바일 터치 타깃 최소 크기 48×48dp 준수.

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

- **직접 매핑 범위**: `WO-MOS-0501 ~ WO-MOS-0600, WO-MOS-0701 ~ WO-MOS-0750`
- **상세 개별 명세 위치**: `11_WORK_ORDERS_0001_TO_1000/individual/` 내 해당 번호 파일 및 `11_WORK_ORDERS_0001_TO_1000/WORK_ORDER_INDEX.csv` 참조.
