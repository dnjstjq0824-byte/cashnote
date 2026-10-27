# START HERE — BN Merchant OS 후속 개발 에이전트 최우선 진입 지침서

**문서 ID**: ROOT-START-001  
**마스터 명령 코드**: `BN-MERCHANT-OS-GEMINI4-ARGON-MASTER-BLUEPRINT-1000-001`  
**패키지 버전**: `V1.0` (`BN_MERCHANT_OS_MASTER_BLUEPRINT_1000_V1_0.zip`)

---

## 1. 환영 및 최상위 임무 선언

본 패키지를 인수한 **Multi-Agent Development Orchestrator** 및 16개 전문 개발 에이전트는 코드를 단 한 줄이라도 작성하기 전에 반드시 본 문서(`START_HERE.md`)를 처음부터 끝까지 숙지해야 한다.

- **프로젝트 목표**: 한국신용데이터의 캐시노트를 주요 벤치마킹 대상으로 하되, 단순 매출 조회 앱을 넘어 **사업장 등록·소유권 검증 → 카드/배달/현금/POS 매출 통합 → 6단계 정산 및 미입금 감사 → 불변 복식부기 원장 및 손익(P&L) → 세금/증빙 → 리뷰/단골 CRM → AI 예측 → 사장님 커뮤니티 및 B2B 마켓 → 다매장/글로벌 확장**을 아우르는 독립형 상용 소상공인 통합 경영관리 SaaS 플랫폼(`BN Merchant OS`)을 구축한다.
- **독립 플랫폼 원칙 (제1조)**: TreeScan, Delivery Note, OLMOA 등 다른 Booking Network 플랫폼에 대한 코드·DB·세션 직접 의존성을 절대 만들지 않는다.

---

## 2. 제28조 필수 20단계 실행 프로토콜 (Immediate Action Checklist)

1. **Step 1 (ZIP 및 파일 무결성 확인)**: 루트 디렉터리에서 `sha256sum -c FILE_CHECKSUMS.sha256`을 실행하여 전체 파일이 단 1바이트의 변조 없이 일치하는지 확인하라.
2. **Step 2 (전체 문서 인벤토리 로드)**: `MANIFEST.json`을 파싱하여 16개 디렉터리와 1,000개 개별 Work Order 파일 위치를 메모리에 로드하라.
3. **Step 3 (시장·규제·API 경계 숙지)**: `00_MARKET_RESEARCH/DATA_ACCESS_REGULATORY_MATRIX.csv` 및 `09_SECURITY_COMPLIANCE/REGULATORY_GATE.md`를 읽고, 즉시 호출 가능한 `PUBLIC_API`(국세청 상태조회 등)와 계약/인허가 전까지 `BLOCKED`로 두어야 하는 외부 API를 구분하라.
4. **Step 4 (아키텍처 검토)**: `01_A_TO_Z_BLUEPRINT/A_ARCHITECTURE.md` 및 `02_SYSTEM_ARCHITECTURE/SYSTEM_OVERVIEW.md`를 검토하라.
5. **Step 5 (DB·API 마스터 계약 고정)**:
   - PostgreSQL 16 스키마: `03_DATABASE/migrations/0000_full_schema.sql` (10개 도메인 56개 이상 전체 테이블 및 RLS 정책)
   - OpenAPI 3.1.0 명세: `04_API_SPECIFICATION/openapi.yaml` (26개 전체 API 그룹)
6. **Step 6 ~ Step 20 (1,000개 Work Order 순차/병렬 실행 및 독립 감사)**:
   - 상세 절차는 `12_AGENT_EXECUTION/EXECUTION_SEQUENCE.md`와 `11_WORK_ORDERS_0001_TO_1000/WORK_ORDER_INDEX.csv`를 따라 `WO-MOS-0001`부터 `WO-MOS-1000`까지 수행하고, 매 작업마다 `Independent Audit Agent`의 교차 검증을 거치라.
