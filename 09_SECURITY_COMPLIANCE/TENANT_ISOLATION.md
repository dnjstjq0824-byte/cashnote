# 멀티테넌트 다중 사업장 데이터 격리 설계서 (Multi-Tenant Isolation)

**문서 ID**: SEC-TENANT-003  
**관련 조항**: 제19조, 제21조, 제26조(Scenario G & Scenario I)

---

## 1. 3중 테넌트 격리 방어선 (Triple-Layer Tenant Isolation)

사업장 A의 사용자가 사업장 B의 금융·매출·세무 데이터를 절대 조회하거나 수정할 수 없도록 다음 3중 방어선을 동시에 가동한다:

1. **Layer 1 — API Gateway & RBAC/ABAC Middleware**:
   - 모든 `/v1/stores/{storeId}/*` 요청에 대해 JWT의 `sub`(`user_id`)가 해당 `storeId`의 활성 멤버(`store_members.invitation_status = 'ACTIVE'`)인지 Redis 캐시 + PostgreSQL 원본으로 검증한다. 불일치 시 도메인 핸들러에 도달하기 전에 `403 FORBIDDEN (AUTH_FORBIDDEN_STORE)`으로 즉시 거부하고 `audit_logs`에 보안 이벤트를 남긴다.
2. **Layer 2 — Mandatory Repository Parameter**:
   - Go `sqlc`로 생성되는 모든 도메인 쿼리의 첫 번째 파라미터는 반드시 `store_id UUID`를 받도록 정적 린터(CI Custom Linter)로 강제한다.
3. **Layer 3 — PostgreSQL Row-Level Security (RLS)**:
   - 모든 테넌트 테이블에 `ENABLE ROW LEVEL SECURITY` 및 `FORCE ROW LEVEL SECURITY`를 적용하고, DB 트랜잭션 래퍼(`WithTenantTx`)가 실행하는 `SET LOCAL app.current_store_id = '<verified_store_uuid>'`와 일치하는 행만 조회·수정되도록 DB 엔진 커널 레벨에서 격리한다.
