# 보안·개인정보·연결철회·장애복구 E2E 테스트 명세서 (Scenario I & Scenario J)

**문서 ID**: TEST-SEC-005  
**관련 조항**: 제21조, 제26조(Scenario I, Scenario J), 제31조

---

## 1. Scenario I: 개인정보·권한 검사·데이터 연결 철회·보존/파기 정책 E2E

1. **Step 1 (연결 철회 호출)**: 사장님이 설정 화면에서 오픈뱅킹 및 홈택스 연동에 대해 `POST /v1/users/me/consents/{consentId}/revoke` 호출.
2. **Step 2 (즉시 차단 검증)**:
   - `data_connections.status`가 즉시 `REVOKED`로 변경되고 `credentials_encrypted` 컬럼이 `NULL`(제로화)로 영구 삭제되었는지 DB 직접 쿼리로 확인.
   - 직후 수동 동기화 API 또는 백그라운드 워커가 해당 연결로 수집을 시도할 때 `409 CONNECTION_REVOKED`로 즉시 거부되는지 확인.
3. **Step 3 (계정 삭제 및 법정 보존 분리 검증)**:
   - `POST /v1/users/me/account-deletion` 실행 시 `user_profiles`의 성명·연락처·소셜 연동 정보는 즉시 파기되고, 국세기본법/상법상 5년 보존 의무가 있는 원장 레코드는 개인 식별키가 분리된 채 보존 볼트로 격리되는지 확인.

---

## 2. Scenario J: 외부 API 장애·부분 실패·재시도·중복 방지·원장 정합성 E2E

1. **Step 1 (장애 주입)**: 500건의 카드 매입 데이터를 수집하는 도중 250번째 건에서 외부 API `503 Service Unavailable` 및 DB 일시적 커넥션 끊김 주입.
2. **Step 2 (자동 재시도 및 멱등성 검증)**:
   - 워커가 지수 백오프 후 작업을 재실행할 때, 이미 처리된 1~249번째 거래는 `UNIQUE (store_id, dedup_fingerprint)`에 의해 중복 삽입되지 않고 250~500번째 거래만 마저 반영되어 최종 거래 수가 정확히 **500건**(중복 0건)인지 확인.
3. **Step 3 (복식부기 원장 무결성 확인)**:
   - 장애 복구 직후 `GET /v1/admin/audit/integrity-verify`를 호출하여 전체 `journal_entries`의 차변 총합 == 대변 총합 일치 및 `audit_logs` 해시 체인 깨짐 0건 확인.
