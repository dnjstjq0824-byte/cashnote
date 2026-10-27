# BN Merchant OS — 글로벌 소상공인 통합 경영관리 플랫폼 마스터 블루프린트 (1,000 Work Orders)

**프로젝트 코드**: `BN-MERCHANT-OS-GEMINI4-ARGON-MASTER-BLUEPRINT-1000-001`  
**지정 기술 스택**:
- **사용자 웹**: Vue 3 + Nuxt 3 + TypeScript (`apps/web-merchant`)
- **관리자·파트너 웹**: Vue 3 + TypeScript (`apps/web-admin`)
- **모바일 앱**: Flutter 3.24+ Android / iOS (`apps/mobile`)
- **백엔드 및 비동기 워커**: Go 1.22+ Modular Monolith + Isolated Connector Workers
- **데이터베이스**: PostgreSQL 16 (56+ Tables, `NUMERIC(19,4)`, `TIMESTAMPTZ`, Row-Level Security 강제)
- **검색 / 캐시 / 스토리지**: OpenSearch 2.x (`analysis-nori`) / Redis 7.2 / S3-Compatible Object Storage (KMS 암호화)
- **API 계약 / 인증 / 관제 / 인프라**: OpenAPI 3.1.0 / OIDC + RBAC/ABAC + MFA / OpenTelemetry / Docker

---

## 패키지 디렉터리 구조 안내

| 디렉터리 / 파일 | 수량 | 핵심 내용 요약 |
|---|---|---|
| `START_HERE.md`, `README.md`, `MANIFEST.json`, `FILE_CHECKSUMS.sha256` | 4개 | 후속 에이전트 진입점, 전체 파일 매니페스트 및 SHA-256 무결성 해시 |
| `00_MARKET_RESEARCH/` | 6개 | 캐시노트 심층 해부, 국내외 경쟁사 25개 항목 비교표(`CSV`), 한국 금융·세무·배달 API 규제 매트릭스(`CSV`), 수익 모델, 공식 출처 URL |
| `01_A_TO_Z_BLUEPRINT/` | 26개 | `A_ARCHITECTURE.md` ~ `Z_ZERO_TRUST.md` (각 문서당 17개 필수 설계 섹션 완비) |
| `02_SYSTEM_ARCHITECTURE/` | 5개 | 시스템 개요, C4 컴포넌트 다이어그램, Level 1~4 데이터 흐름도, 컨테이너 배포, STRIDE 위협 모델 |
| `03_DATABASE/` | 12개 | ERD, 56개+ 테이블 사전, 복식부기 회계 모델, 데이터 출처 모델, `migrations/0000_full_schema.sql` 및 `0001`~`0007` SQL DDL |
| `04_API_SPECIFICATION/` | 5개 | `openapi.yaml` (OpenAPI 3.1.0 26개 API 그룹), RBAC/ABAC 인가 매트릭스, 커넥터 계약, 대사 계약, API 테스트 케이스 |
| `05_UXUI/` | 8개 | 사용자 대시보드(22), 매장 관리(9), 관리자 콘솔(17) 총 48개 화면 상세 설계 및 `SCREEN_INVENTORY.csv`, 독자 디자인 시스템 |
| `06_FINANCIAL_DATA/` | 6개 | 카드매출 6단계 생명주기 및 18대 예외, 은행 입출금 정규화, 자동 대사 알고리즘, 복식원장 엔진, 부가세 산식, 정합성 불변 조건 |
| `07_EXTERNAL_CONNECTORS/` | 7개 | 여신협회/카드사, 오픈뱅킹, 국세청/홈택스/ASP, POS/PG/VAN, 배달3사(파일 파서+제휴), 리뷰 API 및 `CONNECTOR_CAPABILITIES.csv` |
| `08_AI_ANALYTICS/` | 4개 | 시계열 매출/현금흐름 예측(데이터 충분성 게이트), 매출/메뉴 분석, 비용 이상치 탐지, 환각 및 규제 위반 방지 가드레일 |
| `09_SECURITY_COMPLIANCE/` | 5개 | 개인정보보호, 금융 데이터 암호화(비밀번호 무수집), 3중 멀티테넌트 격리, 동의/철회 수명주기, 금융/마이데이터 규제 게이트 |
| `10_COMMUNITY_COMMERCE/` | 4개 | 사업자 인증 커뮤니티(매출 데이터 분리), 멤버십 구독 머신, 지원사업/금융제휴 경계, B2B 마켓플레이스(주문→장부 자동 기장) |
| `11_WORK_ORDERS_0001_TO_1000/` | 1,003개 | `MASTER_WORK_ORDER.md`, `WORK_ORDER_INDEX.csv`, `DEPENDENCY_GRAPH.md` 및 `individual/WO-MOS-0001.md` ~ `WO-MOS-1000.md` |
| `12_AGENT_EXECUTION/` | 6개 | 16개 멀티 에이전트 총괄 지침, 역할/소유권 매트릭스, 인계 프로토콜, 20단계 실행 시퀀스, 충돌 방지, 완료 보고 템플릿 |
| `13_TESTING/` | 7개 | `MASTER_TEST_MATRIX.csv`, 완료 승인 기준, Scenario A~J E2E 명세(대사·사용자·모바일·보안), 제29조 독립 감사 프로토콜 |
| `14_DEVOPS/` | 5개 | 7단계 CI/CD, 무중단 배포, OpenTelemetry 관제, PITR 백업/복구 드릴, 표준 운영 런북(SOP) |
| `15_FINAL_DELIVERY/` | 5개 | Phase 0~10 로드맵, 요구사항 추적 매트릭스(`CSV`), 납품 체크리스트, 외부 계약 대기(`BLOCKED`) 한계 명시, 최종 마스터 보고서 |
