# WO-MOS-0251 — 카드사 정산 예정액과 실제 은행 입금액의 대사 엔진 구현

## 1. 기본 식별 정보 (Identification)
- **Work Order ID**: `WO-MOS-0251`
- **작업명**: 카드사 정산 예정액과 실제 은행 입금액의 대사 엔진 구현
- **개발 분야**: `DOM-06-RECON-BANK` (은행 입금·미지급·대사 엔진)
- **소속 개발 Phase**: Phase 4
- **참조 A~Z 명세**: `01_A_TO_Z_BLUEPRINT/C_CARD_SALES.md, L_LEDGER.md`

## 2. 비즈니스 목적 (Business Purpose)
- BN Merchant OS의 [은행 입금·미지급·대사 엔진] 영역에서 **카드사 정산 예정액과 실제 은행 입금액의 대사 엔진 구현** 기능을 상용 수준으로 구현·검증하여, 소상공인이 외부 연동 상태와 무관하게 정확하고 안전하게 매장 재무·운영 데이터를 관리할 수 있도록 한다.

## 3. 구현 목표 및 20단계 상세 범위 (Implementation Goal & Scope)
1. 승인된 카드매출 데이터 수집 (`payment_authorizations`)
2. 카드 매입 내역 수집 (`payment_captures`, `acquisition_records`)
3. 수수료 계산 규칙 (9대 카드사별·신용/체크·영세/중소 우대수수료율 및 원단위 절사/반올림)
4. 지급 예정일 기록 (`payout_schedules.scheduled_payout_date`, 공휴일/주말 익영업일 이연 반영)
5. 승인된 은행 입금 데이터 연동 (`bank_deposits`: 오픈뱅킹 API Level 3 및 통장 엑셀 업로드 Level 2)
6. 거래 매칭 키 설계 (`store_id` + `source_partner_code` + `scheduled_payout_date` + `target_account`)
7. 복수 거래 합산 입금 처리 (`BATCH_N_TO_1`: N건 카드 승인/매입이 1건의 카드사 일괄 입금으로 들어오는 구조 매칭)
8. 부분 입금 (`PARTIALLY_DEPOSITED`: 일부 전표 선입금 및 잔여 미수금 추적)
9. 수수료 차이 (`FEE_TOLERANCE_MATCH`: 단수 절사 오차 ±10원 허용 매칭 및 수수료 차액 보정 분개)
10. 지급 지연 (`DELAYED_CANDIDATE`: 예정일 경과 미입금 탐지)
11. 취소·환불 (`POST_CAPTURE_CANCEL_OFFSET`: 매입 후 취소 금액이 차기 입금액에서 상계 차감된 케이스 역산 매칭)
12. 미입금 후보 상태 (`DELAYED_CANDIDATE` vs `CONFIRMED_UNPAID` vs `SYNC_DELAY_PENDING` 엄격 분리)
13. 매칭 신뢰도 (`confidence_score` 0.00 ~ 100.00 산출 및 매칭 근거 코드 기록)
14. 사용자 확인 (가맹점주 대사 결과 열람 및 수동 매칭 확정 UI/API)
15. 데이터 정정 (`corrections` 기반 원본 보존 역분개 및 매칭 오버라이드)
16. 재대사 (누락 통장 내역 추가 업로드 또는 정정 시 멱등 재대사 실행)
17. 알림 (미입금 후보 및 지급보류 발생 시 모닝 브리핑 알림톡/푸시 발송)
18. 감사 로그 (`audit_logs` 해시 체인 기록)
19. 파일 업로드 대체 경로 (은행/카드사 API 미연결 시에도 엑셀 업로드만으로 100% 동일 대사 수행)
20. 실제 API 연결 여부 표시 (`DataProvenanceBadge`로 `LEVEL_2_FILE` vs `LEVEL_3_API` vs `LEVEL_4_RECONCILED` 명시)

## 4. 선행 작업 (Prerequisites)
- **선행 Work Order**: `WO-MOS-0201, WO-MOS-0250`
- **선행 계약/스키마**: `03_DATABASE/migrations/0000_full_schema.sql`, `04_API_SPECIFICATION/openapi.yaml`

## 5. 우선순위 (Priority)
- `P0_CRITICAL`

## 6. 담당 에이전트 (Assigned Agent)
- **주 담당(Primary)**: `Settlement/Reconciliation Agent`
- **검수/감사(Audit)**: `Independent Audit Agent` (구현 담당자의 자체 완료 선언 금지)

## 7. 수정 대상 파일 (Files to Modify)
- `internal/dom_06_recon_bank/service.go`
- `internal/dom_06_recon_bank/repository.go`
- `04_API_SPECIFICATION/openapi.yaml` (계약 동기화 확인)

## 8. 생성 대상 파일 (Files to Create)
- `internal/dom_06_recon_bank/wo_mos_0251.go`
- `internal/dom_06_recon_bank/wo_mos_0251_test.go`
- `apps/web-merchant/src/modules/dom_06_recon_bank/wo_mos_0251.vue` (또는 도메인 대응 아티팩트)

## 9. Frontend 명세 (Frontend Specification)
- Vue 3 + Nuxt 3 + TypeScript (`apps/web-merchant`) 또는 관리자 웹(`apps/web-admin`) 연동.
- 디자인만 있는 더미 버튼 금지. 반드시 `LoadingSkeleton`, `DataLoaded`, `EmptyState`, `ErrorState`, `BlockedFallbackState` 5가지 상태와 우측 상단 `DataProvenanceBadge`(데이터 출처 Level 1~4 및 마지막 갱신 시각)를 렌더링한다.

## 10. Backend 명세 (Backend Specification)
- Go 1.22+ 모듈형 모놀리스 도메인 서비스 및 비동기 Go Worker(`Asynq`)로 구현.
- 모든 금액 연산은 부동소수점(`float64`)을 배제하고 `shopspring/decimal` 고정소수점으로 처리하며, DB 트랜잭션 내부에서 도메인 상태 변경 + `audit_logs` 기록 + `outbox_events` 발행을 원자적(Atomic)으로 수행한다.

## 11. DB 명세 (Database Mapping)
- **연관 테이블**: `bank_deposits, payout_schedules, reconciliation_matches, reconciliation_disputes, journal_entries`
- **격리 및 제약**: PK/FK, `UNIQUE (store_id, dedup_fingerprint)`, `NUMERIC(19,4)`, `TIMESTAMPTZ`, 그리고 PostgreSQL Row-Level Security(`store_id = current_tenant_store_id()`) 정책을 강제한다.

## 12. API 명세 (API Contract)
- **대상 엔드포인트 그룹**: `08_BankDeposits, 09_Reconciliation`
- **필수 헤더**: `Authorization: Bearer <JWT>`, `X-Store-Id: <UUID>`, `Idempotency-Key: <UUID>`

## 13. 데이터 출처 (Data Provenance)
- `LEVEL_1_MANUAL` (수기 입력), `LEVEL_2_FILE` (공식 정산 엑셀/CSV 업로드), `LEVEL_3_API` (공식 계약 API), `LEVEL_4_RECONCILED` (교차 대사 완료) 중 실제 유입 경로를 레코드마다 태깅하며, 미연동 데이터를 실시간 확인된 것처럼 허위 표시하지 않는다.

## 14. 외부 승인 상태 (External Approval Status)
- **분류 상태**: `APPROVAL_REQUIRED (오픈뱅킹) / LEVEL_2_FILE 기본 지원`
- **차단(Blocked) 정책**: 외부 기관 계약/인허가가 필요한 API가 미승인 상태일 경우 가짜 API를 만들지 않고 `BLOCKED`로 분류하며, Level 1~2(수기/파일 업로드) 대체 경로가 즉시 동작하도록 보장한다.

## 15. 주요 알고리즘 (Core Algorithm)
- 입력 스키마 및 체크섬 검증 → 테넌트/소유권(`VERIFIED_OWNER`) 확인 → 중복 지문(`SHA-256`) 대조 → 고정밀 Decimal 도메인 계산(승인·매입·수수료·정산예정·실입금 분리 및 영업일 캘린더 반영) → 불변 복식부기 원장(`journal_entries`) 분개 생성 → 감사 로그 해시 체인 갱신.

## 16. 보안 요구사항 (Security Controls)
- 개인정보·금융자격증명 `AES-256-GCM` 봉투 암호화(KMS 연동), 타사 로그인 비밀번호 평문 수집 금지, 구조화 로그 PII 자동 마스킹, SQLi/XSS/CSRF/IDOR 방어.

## 17. 권한 통제 (Authorization — RBAC/ABAC)
- `04_API_SPECIFICATION/AUTHORIZATION_MATRIX.md`에 정의된 역할(`OWNER`, `CO_OWNER`, `MANAGER`, `ACCOUNTANT`, `TAX_AGENT`, `STAFF`, `SUPER_ADMIN`) 및 매장 소속(`store_members`) 여부를 서버 미들웨어와 DB RLS 양쪽에서 강제 검증한다.

## 18. 예외 상황 처리 (Exception Handling)
- 외부 API 타임아웃/5xx 발생 시 Circuit Breaker 작동 및 지수 백오프 재시도; 파일 파싱 중 손상된 행 발견 시 행 번호와 원인을 `ingestion_errors`에 기록; 데이터 동기화 지연 시 미입금 확정 사고로 단정하지 않고 `SYNC_DELAY_PENDING`으로 분리 표시.

## 19. 데이터 정합성 불변 조건 (Data Integrity Invariants)
- 1) 복식부기 차변 합계 == 대변 합계(`chk_journal_balanced`), 2) 동일 거래 중복 합산 0건, 3) 원본 거래 임의 `DELETE/UPDATE` 금지(정정 시 `corrections` 역분개 사용), 4) 승인액·매입액·입금예정액·실입금액·회계매출액 혼합 금지.

## 20. 성능 기준 (Performance SLA)
- 단건/목록 API 응답 P95 < 150ms (P99 < 300ms), 파일 파싱 10,000건당 < 3초, 메모리 누수 및 DB 데드락 0건.

## 21. 테스트 방법 (Testing Methodology)
- **L1 단위 테스트**: `go test -v -race ./internal/dom_06_recon_bank/...` (경계값·수수료·세금·상태전이 검증)
- **L2 DB/보안 통합 테스트**: Testcontainers PostgreSQL 16 기반 DDL·트랜잭션·타 매장 IDOR 차단(403/RLS) 검증
- **L3/L4 계약 및 E2E 테스트**: OpenAPI 스키마 검증 및 Playwright/Flutter 시나리오(Scenario A~J) 검증 (미계약 외부 API는 격리된 Contract Mock 테스트와 라이브 검증을 명확히 분리)

## 22. 완료 승인 기준 (Acceptance Criteria)
- [ ] 구현 범위 20개 항목이 누락 없이 코드·스키마·문서에 반영되었다.
- [ ] 중복 거래 이중 집계 및 차변/대변 불균형이 0건임이 테스트로 입증되었다.
- [ ] 정산 예정·입금 완료·보류·데이터 지연 상태가 정확히 구분되며 오탐을 확정 사고로 표시하지 않는다.
- [ ] 타 사업장(`store_id`) 데이터 무단 조회 시도가 서버 측에서 100% 차단된다.
- [ ] 재실행(Re-run) 시에도 동일한 결과를 보장하는 멱등성(Idempotency)이 검증되었다.
- [ ] DB·API·프론트엔드·E2E 테스트가 모두 통과하고 실제 실행 로그 증빙이 제출되었다.

## 23. 제출 증빙 (Required Evidence Artifacts)
- Git Commit SHA, 단위/통합/E2E 테스트 통과 실행 로그(`test_output.log`), OpenAPI/SQL 검증 출력, 변경 파일 SHA-256 체크섬.

## 24. 완료 선언 조건 (Definition of Done)
- 담당 에이전트(`Settlement/Reconciliation Agent`)가 구현 및 자체 테스트 증빙을 제출한 후, **`Independent Audit Agent`가 교차 검증을 수행하여 `AUDIT_PASSED` 서명을 기록했을 때만 최종 완료(`DONE`)로 선언**한다 (미확보 외부 API 라이브 연동은 허위 완료 선언 없이 `CONTRACT_TEST_PASSED / LIVE_BLOCKED`로 정직하게 기록).
