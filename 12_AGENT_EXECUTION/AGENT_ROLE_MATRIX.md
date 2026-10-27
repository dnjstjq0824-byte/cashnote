# 16개 전문 에이전트 역할·책임·소유 경로 매트릭스 (Agent Role & Ownership Matrix)

**문서 ID**: AGENT-ROLE-002  
**관련 조항**: 제24조(최소 에이전트 역할 16종)

---

## 1. 16개 필수 에이전트 권한 및 배타적 파일 소유권(CODEOWNERS)

| 번호 | 에이전트 명칭 | 담당 Work Order 대역 | 배타적 소유 디렉터리 (Exclusive Path Lock) | 핵심 책임 |
|---|---|---|---|---|
| 01 | **Lead Architect** | `0051–0100` (및 전사 RFC 승인) | `cmd/`, `internal/platform/`, `01_A_TO_Z_BLUEPRINT/`, `02_SYSTEM_ARCHITECTURE/`, `04_API_SPECIFICATION/openapi.yaml` | 시스템 경계, 공통 미들웨어, API/DB 계약 변경 승인 |
| 02 | **Market Research Agent** | `0001–0050` | `00_MARKET_RESEARCH/` | 캐시노트 및 국내외 경쟁사 분석, 외부 API/규제 실사 매트릭스 관리 |
| 03 | **Product/UX Agent** | `0501–0510`, `0551–0560` | `05_UXUI/`, `packages/ui-design-system/tokens/` | 48개 화면 와이어프레임, 디자인 토큰, 접근성(큰 글씨 모드) 설계 |
| 04 | **Frontend Agent** | `0501–0550` | `apps/web-merchant/`, `packages/ui-design-system/components/` | Vue 3 + Nuxt 3 사용자 웹 대시보드 및 사업장 관리 31개 화면 구현 |
| 05 | **Mobile Flutter Agent** | `0551–0600` | `apps/mobile/` | Flutter Android/iOS 앱, 생체인증, 영수증 스캐너, 오프라인 동기화 구현 |
| 06 | **Backend Agent** | `0101–0200`, `0601–0650`, `0751–0850` | `internal/identity/`, `internal/business/`, `internal/crm/`, `internal/commerce/`, `internal/community/`, `internal/search/` | 인증/권한, 사업자 검증, 리뷰/CRM, 구독/B2B 마켓, 커뮤니티/OpenSearch 구현 |
| 07 | **Database Agent** | `0851–0880` | `03_DATABASE/`, `internal/db/` | 56개 테이블 SQL DDL, 파티셔닝, 인덱스 튜닝, RLS 정책, PITR 백업 검증 |
| 08 | **Financial Data Integration Agent** | `0401–0500` | `07_EXTERNAL_CONNECTORS/`, `internal/connector/`, `internal/pos/`, `internal/delivery/` | Level 2 정산 엑셀 파서, Level 3 카드·은행·홈택스·POS·배달 커넥터 구현 |
| 09 | **Settlement/Reconciliation Agent** | `0201–0300` | `06_FINANCIAL_DATA/`, `internal/revenue/`, `internal/reconciliation/` | 카드 6단계 생명주기, 수수료 계산, 정산 예정 캘린더, 은행 입금 자동 대사 및 미지급 진단 |
| 10 | **Accounting & Tax Agent** | `0301–0400` | `internal/ledger/`, `internal/expense/`, `internal/tax/` | 불변 복식부기 분개장, 역분개 정정, 손익계산서(P&L), 부가세 시뮬레이터, 세무사 팩 |
| 11 | **AI/Analytics Agent** | `0651–0700` | `08_AI_ANALYTICS/`, `internal/analytics/`, `internal/ai/` | 매출/현금흐름 시계열 예측, 비용 이상치 탐지, 상권 비교, 환각 방지 Q&A |
| 12 | **Security Agent** | `0881–0890` | `09_SECURITY_COMPLIANCE/`, `internal/security/` | KMS 봉투 암호화, PII 마스킹, 규제 게이트(`REGULATORY_BLOCKED`), 침투 방어 |
| 13 | **Admin Agent** | `0701–0750` | `apps/web-admin/`, `internal/admin/` | 관리자 콘솔 17개 화면, 파트너 콘솔, 4-Eyes 승인, CS Break-Glass 열람 통제 |
| 14 | **DevOps Agent** | `0891–0900` | `14_DEVOPS/`, `deploy/`, `.github/workflows/`, `docker-compose.yml` | Docker 컨테이너화, CI/CD, OpenTelemetry/Prometheus 관제, 무중단 배포 |
| 15 | **QA Agent** | `0901–0950` | `13_TESTING/`, `tests/e2e/`, `tests/contract/` | 단위·통합·계약·Playwright/Flutter E2E Scenario A~J 및 부하 테스트 수행 |
| 16 | **Independent Audit Agent** | `0951–1000` | `15_FINAL_DELIVERY/`, `MANIFEST.json`, `FILE_CHECKSUMS.sha256` | 전 산출물 교차 검증, 회계 불변성 감사, 해시 체크섬 서명 및 최종 출시 승인 |
