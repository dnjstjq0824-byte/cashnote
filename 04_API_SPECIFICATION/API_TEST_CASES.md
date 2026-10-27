# 26개 API 그룹 표준 계약 및 보안 테스트 케이스 명세 (API Test Cases)

**문서 ID**: API-TEST-005  

---

## 1. 필수 검증 매트릭스 (정상·경계값·보안·멱등성·차단 상태)

| 테스트 ID | 대상 API 그룹 | 엔드포인트 | 검증 시나리오 | 기대 HTTP 상태 및 에러 코드 |
|---|---|---|---|---|
| `API-TC-01` | 01. Auth | `POST /v1/auth/login` | 5회 연속 잘못된 비밀번호 입력 후 6회차 시도 | `429 Too Many Requests` (`AUTH_ACCOUNT_LOCKED_15M`) |
| `API-TC-02` | 03. Business | `POST /v1/businesses` | 국세청 API 정상 조회 후 아직 1원 계좌인증을 안 한 상태에서 카드매출 연동 요청 | `403 Forbidden` (`OWNERSHIP_VERIFICATION_REQUIRED`) |
| `API-TC-03` | 04~06. Isolation | `GET /v1/stores/{storeB}/card-sales/summary` | 사업장 A의 `OWNER` JWT 토큰으로 사업장 B의 `storeId` 조회 시도 | `403 Forbidden` (`AUTH_FORBIDDEN_STORE`) + `audit_logs` 침투 시도 기록 |
| `API-TC-04` | 05. RBAC | `POST /v1/stores/{storeA}/ledger/entries/{id}/correct` | `TAX_AGENT`(조회전용 세무사) 권한으로 원장 직접 역분개 정정 시도 | `403 Forbidden` (`INSUFFICIENT_ROLE_PERMISSION`) |
| `API-TC-05` | 09. Recon | `GET /v1/stores/{storeA}/reconciliation/diagnostics` | 은행 계좌 동기화가 2일 전 멈춘 상태에서 금일 정산 예정건 진단 조회 | 미입금 확정 사고로 오탐하지 않고 `SYNC_DELAY_PENDING` 상태 반환 |
| `API-TC-06` | 11. Ledger | `POST /v1/stores/{storeA}/ledger/manual-entry` | 차변 100,000원, 대변 99,990원으로 불균형 전표 입력 시도 | `422 Unprocessable Entity` (`LEDGER_UNBALANCED_ENTRY`) |
| `API-TC-07` | 14. Delivery | `POST /v1/stores/{storeA}/connections` | 미승인 배달앱 실시간 스크래핑 연동 요청 | `422 Unprocessable Entity` (`CONNECTOR_EXTERNAL_BLOCKED`) + Level 2 파일 업로드 안내 반환 |
| `API-TC-08` | 20. Subscription | `POST /v1/stores/{storeA}/subscriptions` | 동일한 `Idempotency-Key`로 구독 결제 요청 2회 동시 전송 | 첫 요청 `201 Created`, 두 번째 요청 추가 PG 과금 없이 동일 응답 반환 |
| `API-TC-09` | 26. Import | `POST /v1/stores/{storeA}/imports/upload` | 동일한 여신협회 엑셀 파일을 두 번 중복 업로드 | `409 Conflict` (`IMPORT_DUPLICATE_FILE_SHA256`) |
