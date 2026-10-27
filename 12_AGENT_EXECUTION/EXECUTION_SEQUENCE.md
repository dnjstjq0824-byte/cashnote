# 제28조 후속 개발 에이전트 20단계 인수 및 실행 시퀀스 (20-Step Execution Sequence)

**문서 ID**: AGENT-SEQ-004  
**관련 조항**: 제28조(후속 개발 에이전트 인수 프로토콜)

---

## 1. ZIP 패키지 인수 후 필수 20단계 실행 순서

본 블루프린트 ZIP을 수령한 오케스트레이터 및 개발 에이전트는 반드시 다음 20단계를 순서대로 수행한다:

| 단계 | 실행 항목 | 검증 명령 / 대상 문서 | 통과 기준 (Exit Criteria) |
|---|---|---|---|
| **01** | **ZIP 무결성 확인** | `sha256sum -c FILE_CHECKSUMS.sha256` | 전체 파일 체크섬 100% 일치 (변조/손상 0건) |
| **02** | **전체 문서 인벤토리 점검** | `MANIFEST.json` 대조 | A~Z 26개, Work Order 1,000개, DB/API/UXUI 전 문서 존재 확인 |
| **03** | **시장·규제·API 검토** | `00_MARKET_RESEARCH/DATA_ACCESS_REGULATORY_MATRIX.csv` | `PUBLIC_API` vs `CONTRACT_REQUIRED` vs `BLOCKED` 경계 숙지 |
| **04** | **아키텍처 검토** | `02_SYSTEM_ARCHITECTURE/*` | 독립 플랫폼 원칙(타 플랫폼 의존성 배제) 및 모듈형 모놀리스+워커 구조 확인 |
| **05** | **DB·API 계약 검토** | `03_DATABASE/migrations/0000_full_schema.sql`, `04_API_SPECIFICATION/openapi.yaml` | 56개 테이블 DDL 및 26개 API 그룹 스키마 확정 |
| **06** | **1,000개 Work Order 확인** | `11_WORK_ORDERS_0001_TO_1000/WORK_ORDER_INDEX.csv` | `WO-MOS-0001` ~ `WO-MOS-1000` 의존성 및 담당 에이전트 매핑 로드 |
| **07** | **기존 소스 코드베이스 감사** | 대상 Git 리포지토리 스캔 | 기존 소스가 존재하면 스키마/API 일치 여부 감사 |
| **08** | **신규 프로젝트 초기 생성** | `WO-MOS-0051` ~ `0060` 실행 | 신규 소스인 경우 Go + Vue/Nuxt + Flutter 모노레포 스캐폴딩 생성 |
| **09** | **Gap Analysis 수행** | 블루프린트 계약 vs 현재 코드베이스 비교 | 미구현/불일치 Work Order 식별 및 상태 태깅 |
| **10** | **개발 우선순위 확정** | `PHASE_ROADMAP.md` 및 `DEPENDENCY_GRAPH.md` | Phase 1(기반) → Phase 2(장부 MVP) → Phase 3~4(연동/대사) 순차 배치 |
| **11** | **의존성 기반 실행** | DAG 위상 정렬 스케줄링 | 선행 작업 완료 후 후속 에이전트 병렬 투입 |
| **12** | **실제 코드 구현** | Go / Vue 3 / Flutter 코드 작성 | 더미 버튼 제로, 고정소수점 연산, Level 1~2 Fallback 필수 구현 |
| **13** | **DB 마이그레이션 적용** | `0001` ~ `0007` Up/Down 실행 | 56개 테이블, 인덱스, RLS 정책 정상 생성 확인 |
| **14** | **기능 및 계약 테스트** | 단위·통합·OpenAPI 스키마 테스트 | 커버리지 ≥ 85%, 스키마 위반 0건 |
| **15** | **금융 데이터 정합성 검사** | 회계 불변성 테스트 (`FIN-ACC-006`) | 복식부기 차변==대변 100% 일치, 중복 합산 0건, 5단계 금액 분리 확인 |
| **16** | **모바일·웹 E2E 테스트** | Scenario A ~ Scenario J 실행 | Playwright 및 Flutter Integration Test 10대 시나리오 통과 |
| **17** | **보안 및 테넌트 격리 검사** | IDOR 침투·KMS·감사로그 해시체인 검증 | 타 사업장 데이터 열람 0건, 비밀번호 무단 수집 0건 |
| **18** | **독립 감사 (Independent Audit)** | `Independent Audit Agent` 교차 검증 | 구현 에이전트와 분리된 감사관의 증빙 재검증 및 서명 |
| **19** | **승인된 프로덕션 배포** | 4-Eyes 승인 및 Release Gate 통과 | PO 및 보안/법무 승인 완료된 기능만 프로덕션 피처 플래그 활성화 |
| **20** | **Git SHA·로그·증빙 최종 보고** | `REPORTING_TEMPLATE.md` 제출 | 실제 실행된 Git SHA, 테스트 로그, BLOCKED 항목 정직한 분리 보고 |
