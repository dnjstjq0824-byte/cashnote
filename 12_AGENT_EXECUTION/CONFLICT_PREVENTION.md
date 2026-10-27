# 멀티 에이전트 동시 수정 충돌 방지 및 계약 변경 규격 (Conflict Prevention)

**문서 ID**: AGENT-CONFLICT-005  

---

## 1. 4대 충돌 방지 메커니즘

1. **디렉터리 레벨 배타적 락 (Directory-Level Mutex)**:
   - 각 도메인 패키지는 단일 에이전트에게만 쓰기 권한이 부여된다. 예를 들어 `internal/reconciliation/`은 오직 `Settlement/Reconciliation Agent`만 수정할 수 있으며, `Frontend Agent`는 백엔드 코드를 직접 수정할 수 없다.
2. **DB 마이그레이션 번호 대역 사전 분할 (Migration Number Partitioning)**:
   - 두 에이전트가 동일한 마이그레이션 시퀀스 번호를 생성해 충돌하는 것을 막기 위해 도메인별로 번호 대역(`0001~0099` Core/DB Agent, `0100~0199` Identity/Biz, `0200~0299` Revenue/Recon 등)을 고정 할당한다.
3. **공유 계약(`openapi.yaml`) 변경 시 RFC-Lock 프로토콜**:
   - API 요청/응답 필드를 변경해야 할 경우, 개별 에이전트가 임의로 `openapi.yaml`을 수정하지 않고 `Lead Architect`에게 Contract Change Request를 제출한다. 승인 시 `openapi.yaml`이 갱신되고 영향받는 모든 Frontend/Mobile/QA Work Order에 재검증 플래그가 자동 전파된다.
4. **도메인 간 통신은 오직 Go Interface 및 Outbox Event로만 연결**:
   - 예를 들어 `B2B Marketplace` 모듈이 매장 비용 장부를 생성할 때 `expenses` 테이블에 직접 `INSERT` 쿼리를 날리지 않고, `ExpenseService.RecordMarketplacePurchase(ctx, cmd)` 인터페이스 또는 `marketplace.order.placed` 도메인 이벤트를 통해서만 연동한다.
