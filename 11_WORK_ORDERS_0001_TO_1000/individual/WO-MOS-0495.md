# WO-MOS-0495 — 배달앱 엑셀 헤더 변경 시 동적 매핑 스키마 업데이트로 파싱이 정상 복구되는지 검증하는 유연성 테스트

## 1. 기본 식별 정보 (Identification)
- **Work Order ID**: `WO-MOS-0495`
- **작업명**: 배달앱 엑셀 헤더 변경 시 동적 매핑 스키마 업데이트로 파싱이 정상 복구되는지 검증하는 유연성 테스트
- **개발 분야**: `DOM-10-DELIVERY` (배달 플랫폼 매출·정산)
- **소속 개발 Phase**: Phase 3~4
- **참조 A~Z 명세**: `01_A_TO_Z_BLUEPRINT/D_DATA_INTEGRATION.md, 07_EXTERNAL_CONNECTORS/DELIVERY_APPS.md`

## 2. 비즈니스 목적 (Business Purpose)
- BN Merchant OS의 [배달 플랫폼 매출·정산] 영역에서 **배달앱 엑셀 헤더 변경 시 동적 매핑 스키마 업데이트로 파싱이 정상 복구되는지 검증하는 유연성 테스트** 기능을 상용 수준으로 구현·검증하여, 소상공인이 외부 연동 상태와 무관하게 정확하고 안전하게 매장 재무·운영 데이터를 관리할 수 있도록 한다.

## 3. 구현 목표 및 20단계 상세 범위 (Implementation Goal & Scope)
1. 요구사항 및 도메인 경계 검증 (`01_A_TO_Z_BLUEPRINT/D_DATA_INTEGRATION.md, 07_EXTERNAL_CONNECTORS/DELIVERY_APPS.md` 참조)
2. 데이터 모델 및 스키마 매핑 검증 (`orders, settlement_records, payout_schedules, bank_deposits, reconciliation_matches`)
3. 테넌트 격리 컨텍스트(`store_id`, `business_id`) 및 RBAC/ABAC 권한 규칙 바인딩
4. Level 1(수기) / Level 2(파일 업로드) / Level 3(공식 API) / Level 4(자동 대사) 데이터 출처(`source_level`) 처리
5. 외부 승인 상태(`PARTNER_ONLY (실시간 API 미제휴 시 BLOCKED) / LEVEL_2_FILE 기본 탑재`) 게이트 확인 및 미승인 시 안전한 Fallback/BLOCKED 처리
6. 핵심 비즈니스 로직 및 고정밀 Decimal(`NUMERIC(19,4)`) 연산 구현
7. 멱등성 키(`Idempotency-Key`) 및 중복 방지 지문(`dedup_fingerprint`) 검증 로직 구현
8. 도메인 이벤트 발행(`outbox_events`) 및 비동기 워커 연동
9. 복식부기 원장(`journal_entries`) 또는 감사 추적(`audit_logs`) 원자적 트랜잭션 연동
10. OpenAPI 3.1 계약(`14_DeliveryConnector, 26_ImportExport`) 요청/응답/에러 스키마 바인딩
11. 프론트엔드/모바일 5대 상태(`Loading`, `Loaded`, `Empty`, `Error`, `Blocked`) UI 연동
12. 데이터 갱신 시각 및 `DataProvenanceBadge` 노출 구현
13. 경계값 및 예외 상황(타임아웃, 부분 실패, 중복 유입, 권한 부족) 방어 로직 구현
14. 개인정보/민감정보 암호화(`AES-256-GCM`) 및 구조화 로그 마스킹 적용
15. 단위 테스트(Unit Test: 정상·경계·오류 케이스) 작성
16. DB 통합 및 RLS 테넌트 격리 침투 테스트 작성
17. OpenAPI 스키마 계약 테스트 및 E2E 시나리오 연결 검증
18. 성능 프로파일링(쿼리 실행계획 인덱스 타는지 확인 및 P95 목표 달성)
19. 코드 정적 분석(Linter, Security Scanner, 금칙어 검사) 통과
20. 실행 로그·테스트 리포트 증빙 아티팩트 생성 및 독립 감사(`Independent Audit Agent`) 제출

## 4. 선행 작업 (Prerequisites)
- **선행 Work Order**: `WO-MOS-0494`
- **선행 계약/스키마**: `03_DATABASE/migrations/0000_full_schema.sql`, `04_API_SPECIFICATION/openapi.yaml`

## 5. 우선순위 (Priority)
- `P1_HIGH`

## 6. 담당 에이전트 (Assigned Agent)
- **주 담당(Primary)**: `Financial Data Integration Agent`
- **검수/감사(Audit)**: `Independent Audit Agent` (구현 담당자의 자체 완료 선언 금지)

## 7. 수정 대상 파일 (Files to Modify)
- `internal/dom_10_delivery/service.go`
- `internal/dom_10_delivery/repository.go`
- `04_API_SPECIFICATION/openapi.yaml` (계약 동기화 확인)

## 8. 생성 대상 파일 (Files to Create)
- `internal/dom_10_delivery/wo_mos_0495.go`
- `internal/dom_10_delivery/wo_mos_0495_test.go`
- `apps/web-merchant/src/modules/dom_10_delivery/wo_mos_0495.vue` (또는 도메인 대응 아티팩트)

## 9. Frontend 명세 (Frontend Specification)
- Vue 3 + Nuxt 3 + TypeScript (`apps/web-merchant`) 또는 관리자 웹(`apps/web-admin`) 연동.
- 디자인만 있는 더미 버튼 금지. 반드시 `LoadingSkeleton`, `DataLoaded`, `EmptyState`, `ErrorState`, `BlockedFallbackState` 5가지 상태와 우측 상단 `DataProvenanceBadge`(데이터 출처 Level 1~4 및 마지막 갱신 시각)를 렌더링한다.

## 10. Backend 명세 (Backend Specification)
- Go 1.22+ 모듈형 모놀리스 도메인 서비스 및 비동기 Go Worker(`Asynq`)로 구현.
- 모든 금액 연산은 부동소수점(`float64`)을 배제하고 `shopspring/decimal` 고정소수점으로 처리하며, DB 트랜잭션 내부에서 도메인 상태 변경 + `audit_logs` 기록 + `outbox_events` 발행을 원자적(Atomic)으로 수행한다.

## 11. DB 명세 (Database Mapping)
- **연관 테이블**: `orders, settlement_records, payout_schedules, bank_deposits, reconciliation_matches`
- **격리 및 제약**: PK/FK, `UNIQUE (store_id, dedup_fingerprint)`, `NUMERIC(19,4)`, `TIMESTAMPTZ`, 그리고 PostgreSQL Row-Level Security(`store_id = current_tenant_store_id()`) 정책을 강제한다.

## 12. API 명세 (API Contract)
- **대상 엔드포인트 그룹**: `14_DeliveryConnector, 26_ImportExport`
- **필수 헤더**: `Authorization: Bearer <JWT>`, `X-Store-Id: <UUID>`, `Idempotency-Key: <UUID>`

## 13. 데이터 출처 (Data Provenance)
- `LEVEL_1_MANUAL` (수기 입력), `LEVEL_2_FILE` (공식 정산 엑셀/CSV 업로드), `LEVEL_3_API` (공식 계약 API), `LEVEL_4_RECONCILED` (교차 대사 완료) 중 실제 유입 경로를 레코드마다 태깅하며, 미연동 데이터를 실시간 확인된 것처럼 허위 표시하지 않는다.

## 14. 외부 승인 상태 (External Approval Status)
- **분류 상태**: `PARTNER_ONLY (실시간 API 미제휴 시 BLOCKED) / LEVEL_2_FILE 기본 탑재`
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
- **L1 단위 테스트**: `go test -v -race ./internal/dom_10_delivery/...` (경계값·수수료·세금·상태전이 검증)
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
- 담당 에이전트(`Financial Data Integration Agent`)가 구현 및 자체 테스트 증빙을 제출한 후, **`Independent Audit Agent`가 교차 검증을 수행하여 `AUDIT_PASSED` 서명을 기록했을 때만 최종 완료(`DONE`)로 선언**한다 (미확보 외부 API 라이브 연동은 허위 완료 선언 없이 `CONTRACT_TEST_PASSED / LIVE_BLOCKED`로 정직하게 기록).
