# 16개 멀티 에이전트 총괄 개발 수행 명령서 (Master Agent Directive)

**문서 ID**: AGENT-DIR-001  
**관련 조항**: 제24조(멀티 에이전트 개발 운영), 제28조(후속 개발 에이전트 인수 프로토콜), 제31조(절대 금지 사항)

---

## 1. 멀티 에이전트 오케스트레이션 최상위 원칙

1. **Contract-First Execution (계약 선행 원칙)**:
   - 모든 에이전트는 코드 작성 전에 `03_DATABASE/migrations/0000_full_schema.sql`(PostgreSQL 16 스키마)과 `04_API_SPECIFICATION/openapi.yaml`(OpenAPI 3.1 명세)을 단일 진실 공급원(Single Source of Truth)으로 삼는다.
2. **Exclusive File Ownership (파일 소유권 격리)**:
   - 동일한 소스 파일이나 마이그레이션 번호를 두 개 이상의 에이전트가 동시에 수정하는 것을 금지한다 (`AGENT_ROLE_MATRIX.md` 및 `CONFLICT_PREVENTION.md` 참조).
3. **Separation of Implementation and Audit (구현과 감사의 절대 분리)**:
   - 구현 담당 에이전트(`Backend Agent`, `Frontend Agent` 등)는 스스로 Work Order를 최종 `DONE`으로 마감할 수 없다. 반드시 테스트 실행 로그와 Git SHA를 첨부하여 `Independent Audit Agent`에게 제출(`SUBMITTED_FOR_AUDIT`)하고, 감사 통과 후에만 완료 선언된다.
4. **Honest Handling of External Blockers (미승인 외부 API 정직한 처리)**:
   - 필수 외부 계약이나 법적 인허가가 없는 기능(예: 배달앱 비공개 API, 대출 모집 중개)을 가짜 성공 응답으로 허위 구현하는 것을 엄격히 금지한다. 해당 영역은 계약 인터페이스 + Level 1~2(수기/파일 업로드) 대체 경로를 구현하고 라이브 연동은 `BLOCKED`로 기록한다.
