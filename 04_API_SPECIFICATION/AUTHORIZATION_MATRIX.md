# 역할 기반(RBAC) 및 속성 기반(ABAC) API 인가 매트릭스 (Authorization Matrix)

**문서 ID**: API-AUTHZ-002  

---

## 1. 매장 역할별 API 접근 권한 매트릭스

| API 그룹 / 도메인 | `OWNER` (대표자) | `CO_OWNER` (공동대표) | `MANAGER` (매장매니저) | `ACCOUNTANT` (회계담당) | `TAX_AGENT` (세무대리인) | `STAFF` (일반직원) |
|---|---|---|---|---|---|---|
| **03. 사업자 소유권 이전 / 폐업** | ALLOW (MFA 필수) | DENY | DENY | DENY | DENY | DENY |
| **04~05. 직원 초대 / 권한 변경** | ALLOW | ALLOW | ALLOW (STAFF만) | DENY | DENY | DENY |
| **06~07. 카드매출·정산 조회** | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW (읽기전용) | DENY |
| **08. 은행계좌 잔액·입출금 조회** | ALLOW | ALLOW | DENY (기본 차단, 옵션 허용) | ALLOW | ALLOW (세무통장만) | DENY |
| **09. 미지급 진단·이의제기** | ALLOW | ALLOW | ALLOW (조회만) | ALLOW | ALLOW (조회만) | DENY |
| **10. 비용·영수증 OCR 등록** | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW (검토/분류) | ALLOW (본인 영수증만) |
| **11. 원장 역분개 정정 / 마감** | ALLOW | ALLOW | DENY | ALLOW | ALLOW (검토 의견) | DENY |
| **12. 세금 추정·신고 패키지 다운로드** | ALLOW | ALLOW | DENY | ALLOW | ALLOW (MFA 필수) | DENY |
| **15~16. 리뷰 답글·단골 캠페인** | ALLOW | ALLOW | ALLOW | DENY | DENY | ALLOW (초안 작성만) |
| **20~21. 유료 구독·B2B 마켓 결제** | ALLOW | ALLOW | ALLOW (한도 내 발주) | ALLOW (정산조회) | DENY | DENY |
| **26. 정산 파일(CSV/Excel) 업로드** | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | DENY |

---

## 2. 서버 측 강제 검증 알고리즘 (3-Step Enforcement)

1. **Step 1 (Authentication)**: `Authorization: Bearer <JWT>`의 OIDC 서명, 만료시각(`exp`), 세션 폐기 여부(`sessions.revoked_at IS NULL`) 검증.
2. **Step 2 (Store Ownership & Membership ABAC)**: 요청 경로/헤더의 `store_id`에 대해 `store_members` 테이블에서 `(store_id = $1 AND user_id = $2 AND invitation_status = 'ACTIVE' AND (valid_until IS NULL OR valid_until > NOW()))`를 조회하고, 사업자 소유권 상태(`businesses.ownership_verification_status`)가 금융 연동 요구 수준을 충족하는지 확인.
3. **Step 3 (PostgreSQL RLS Injection)**: DB 커넥션 트랜잭션 시작 즉시 `SELECT set_config('app.current_store_id', $1, true)`를 실행하여 SQL 레벨에서 타 매장 행 접근을 원천 봉쇄.
