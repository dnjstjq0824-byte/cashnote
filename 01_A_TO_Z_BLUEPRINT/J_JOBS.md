# J — Jobs & Notifications (비동기 배치·동기화 워커·푸시·이메일·알림톡)

**문서 ID**: AZ-J_JOBS  
**프로젝트 코드명**: BN Merchant OS (`BN-MERCHANT-OS-GEMINI4-ARGON-MASTER-BLUEPRINT-1000-001`)  
**관련 Work Order 범위**: `WO-MOS-0090, WO-MOS-0815 ~ WO-MOS-0835`  
**설계 원칙**: 제0조~제32조 마스터 명령 전수 준수 (독립 플랫폼 원칙, Level 1~4 데이터 수집 전략, 미승인 외부 API 허위 구현 금지)

---

## 1. 목표 (Objective)

외부 데이터 정기 동기화, 대용량 정산 파일 파싱, 일별/주간/월간 리포트 생성, 미입금 경보, 세금 납부 기한 안내를 처리하는 고신뢰성 백그라운드 잡(Go Workers) 및 멀티채널 알림(카카오 알림톡, FCM/APNs 앱 푸시, 이메일, 인앱 알림) 엔진을 설계한다.

---

## 2. 사용자 역할 (User Roles & Actors)

- **주요 액터**: 매장 대표, 매니저, 회계담당, 시스템 운영자
- **권한 경계**: 모든 요청은 인증된 사용자(`user_id`)와 명시적으로 선택·검증된 사업자(`business_id`) 및 매장(`store_id`) 컨텍스트 하에서만 실행되며, 타 사업장 데이터에 대한 교차 접근은 애플리케이션 미들웨어와 PostgreSQL RLS(Row-Level Security) 양쪽에서 차단된다.

---

## 3. 기능 명세 (Functional Specifications)

- **핵심 기능 1**: 우선순위 분리 워커 큐: `critical-reconciliation`(실시간 대사 및 결제 웹훅), `high-interactive`(사용자 요청 파일 파싱 및 수동 동기화), `default-scheduled`(일일 새벽 정기 수집), `low-batch`(월간 PDF/엑셀 리포트 생성 및 AI 배치 학습)
- **핵심 기능 2**: 일일 모닝 브리핑 스케줄러: 매일 오전 사장님이 지정한 시각(기본 08:30 KST)에 '어제 총매출 + 오늘 입금 예정 금액 + 주의할 미입금/보류 건수 + 신규 부정 리뷰'를 요약하여 카카오 알림톡 또는 앱 푸시로 발송
- **핵심 기능 3**: 스마트 알림 피로도 제어(Notification Throttling & Quiet Hours): 정보통신망법 준수를 위해 야간 시간대(20:50 ~ 익일 08:00) 영리성 알림 발송 원천 차단, 동일 유형 알림 묶음 발송(Digest)
- **핵심 기능 4**: 멀티채널 Fallback 라우팅: 카카오 알림톡 발송 실패(채널 차단/미가입 등) 시 설정에 따라 SMS/LMS 또는 앱 푸시/이메일로 자동 우회 발송
- **핵심 기능 5**: 데드레터 큐(DLQ) 및 관리자 재실행(Replay) 콘솔: 외부 API 일시 장애로 실패한 잡을 지수 백오프로 최대 5회 재시도 후 DLQ로 이동하고 관리자 콘솔에서 단건/일괄 재실행 지원

---

## 4. 사용자 시나리오 (End-to-End User Scenario)

> **실전 시나리오**: 매일 새벽 04:00~06:00 사이 `default-scheduled` 워커가 승인된 매장들의 전일 매출·매입·정산 데이터를 수집하고 대사 엔진을 돌린다. 오전 08:30이 되면 사장님 스마트폰으로 '어제 매출 142만 원, 오늘 입금 예정 118만 원(신한·국민 외), 확인 필요한 입금 지연 후보 1건' 알림톡이 도착한다.

1. **Pre-condition**: 사용자가 정상 로그인되어 있고 해당 매장에 대해 역할 권한(`RBAC/ABAC`)을 보유하고 있다. 외부 연동 기능의 경우 `provider_catalog` 및 `data_connections` 상태가 확인된다.
2. **Normal Flow**: 요청이 API Gateway를 통과하여 도메인 서비스에서 처리되고, 원본 출처(`data_source_level`)와 함께 PostgreSQL 트랜잭션 및 `outbox_events`에 기록된다.
3. **Fallback Flow (연동 차단/미계약 시)**: 만약 외부 API가 `CONTRACT_REQUIRED` 또는 `PARTNER_ONLY(BLOCKED)` 상태이면, 가짜 실시간 데이터를 표시하지 않고 즉시 **Level 2(공식 정산파일 CSV/Excel 업로드)** 또는 **Level 1(수기 입력)** 인터페이스로 안내하여 핵심 장부·대사 업무가 중단 없이 수행되도록 보장한다.

---

## 5. 백엔드 아키텍처 및 구현 명세 (Backend Architecture)

- **언어 및 프레임워크**: Go 1.22+, 모듈형 모놀리스 아키텍처 + 독립 비동기 워커(Go Workers)
- **핵심 구현 모듈**: `internal/jobs` 및 `internal/notification` 패키지: Go Asynq / Redis Streams 기반 분산 워커, 분산 락(Redlock, 동일 매장의 중복 동기화 방지), 템플릿 렌더러
- **트랜잭션 관리**: 모든 상태 변경과 원장 반영은 `pgx/v5` 트랜잭션 내부에서 도메인 데이터 변경 + `audit_logs` 기록 + `outbox_events` 발행을 원자적(Atomic)으로 커밋한다.

---

## 6. 프론트엔드 및 모바일 명세 (Frontend & Mobile)

- **구현 범위**: 알림 센터 드로어(읽음/안읽음 필터), 채널별·이벤트별 알림 수신 설정 토글(일일 브리핑, 미입금 알림, 부정 리뷰 알림, 세무 일정, 마케팅 동의), 관리자 백그라운드 잡 모니터
- **상태 표현 표준**:
  1. `LOADING`: 스켈레톤 UI 및 진행률 표시
  2. `SUCCESS`: 데이터 표시 + 우측 상단 `DataProvenanceBadge`(데이터 출처 Level 1~4 및 최종 갱신 시각)
  3. `EMPTY`: 초기 온보딩 안내 및 첫 데이터 등록/업로드 CTA 버튼
  4. `ERROR`: 사용자 친화적 오류 메시지 + `request_id` 복사 + 재시도 버튼
  5. `BLOCKED`: 외부 기관 계약/승인 대기 상태의 정직한 고지 + 파일 업로드 대체 경로 버튼

---

## 7. 데이터베이스 설계 연동 (Database Schema Mapping)

- **연관 테이블**: `background_jobs`, `sync_jobs`, `notifications`, `outbox_events`
- **데이터 무결성 규칙**:
  - 모든 금액 컬럼은 `NUMERIC(19,4)`를 사용하여 부동소수점 반올림 오차를 원천 배제한다.
  - 모든 시각 컬럼은 `TIMESTAMPTZ`로 저장한다.
  - 원본 거래 레코드는 임의 삭제(`DELETE`)하지 않으며, 정정 시 `corrections` 및 역분개(`journal_entries`)로 감사 추적성을 유지한다.

---

## 8. API 계약 명세 (OpenAPI 3.1 Endpoints)

- **주요 엔드포인트**: `GET /v1/notifications`, `PATCH /v1/notifications/{notificationId}/read`, `PUT /v1/users/me/notification-preferences`, `POST /v1/admin/jobs/{jobId}/retry`
- **공통 헤더**:
  - `Authorization: Bearer <OIDC_JWT>`
  - `X-Business-Id: <UUID>`
  - `X-Store-Id: <UUID>`
  - `Idempotency-Key: <UUID>` (모든 `POST`/`PUT`/`PATCH` 상태 변경 요청 시 필수)

---

## 9. 도메인 이벤트 (Domain Events)

- **발행/구독 이벤트**: `job.scheduled`, `job.completed`, `job.dead_lettered`, `notification.dispatched`, `notification.delivered`
- **전달 보장**: Transactional Outbox 테이블(`outbox_events`)에서 Redis Streams / Worker Queue로 At-Least-Once 전달을 보장하며, 컨슈머는 `event_id` 기준 멱등 처리(Idempotent Consumer)를 수행한다.

---

## 10. 데이터 출처 및 신뢰도 등급 (Data Provenance)

- **데이터 원천**: 내부 도메인 이벤트, 스케줄러 Cron 트리거, 외부 메시징 게이트웨이(DLR 수신)
- **출처 태깅 규칙**:
  - `LEVEL_1_MANUAL`: 사용자 직접 입력 (신뢰도: 사용자 책임, 즉시 반영)
  - `LEVEL_2_FILE`: 공식 정산 엑셀/CSV 업로드 (신뢰도: 파일 해시 및 포맷 검증됨)
  - `LEVEL_3_API`: 공식 계약/승인된 외부 API (신뢰도: 기관 전자서명/TLS 검증됨)
  - `LEVEL_4_RECONCILED`: 복수 데이터 소스(예: 카드매입 + 은행입금) 교차 대사 완료 (신뢰도: 최고 등급 확정)

---

## 11. 오류 처리 및 장애 복구 (Error Handling & Resilience)

- **예외 처리 정책**: 워커 프로세스가 작업 도중 비정상 종료(OOM/Spot 회수)되더라도 Heartbeat 타임아웃 감지 후 다른 워커 노드가 안전하게 작업을 인계받으며, DB 트랜잭션 멱등성 키로 중복 알림 발송 및 중복 원장 기장을 방지
- **표준 에러 코드**: `AUTH_FORBIDDEN_STORE`, `CONNECTOR_EXTERNAL_BLOCKED`, `LEDGER_UNBALANCED_ENTRY`, `IMPORT_DUPLICATE_BATCH`, `RATE_LIMIT_EXCEEDED`, `UPSTREAM_TIMEOUT_RETRYABLE`

---

## 12. 개인정보 및 보안 통제 (Privacy & Security Controls)

- **보안 설계**: 알림톡/SMS 본문에 계좌번호 전체나 비밀번호 등 민감정보 노출 금지(마스킹 처리 후 앱 딥링크로 유도).
- **감사 추적**: 모든 민감 데이터 조회 및 변경은 `audit_logs`에 요청자 ID, 대상 매장 ID, 액션 코드, IP, User-Agent, 변경 전후 요약 및 해시 체인과 함께 기록된다.

---

## 13. 외부 승인 및 법적 선행 요건 (External Approval & Regulatory Requirements)

- **승인·규제 요건**: 카카오 비즈니스 알림톡 템플릿 사전 검수 승인(`APPROVAL_REQUIRED`) 및 발신번호 사전등록제(전기통신사업법 제84조의2) 등록 필수.
- **차단(Blocked) 정책**: 필수 외부 계약이나 인허가가 완료되기 전까지 해당 라이브 연동 기능은 `feature_flags` 및 `provider_catalog`에서 `BLOCKED`로 분류하며, 가짜 API 응답으로 상용 완료를 가장하는 행위를 엄격히 금지한다.

---

## 14. 성능 및 확장성 목표 (Performance & SLA Targets)

- **목표 지표**: 인앱/푸시 알림 발송 지연 < 2초, 오전 08:00~09:00 10만 매장 모닝 브리핑 발송 처리량 ≥ 1,000 msg/sec.

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

- **직접 매핑 범위**: `WO-MOS-0090, WO-MOS-0815 ~ WO-MOS-0835`
- **상세 개별 명세 위치**: `11_WORK_ORDERS_0001_TO_1000/individual/` 내 해당 번호 파일 및 `11_WORK_ORDERS_0001_TO_1000/WORK_ORDER_INDEX.csv` 참조.
