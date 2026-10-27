# 에이전트 간 작업 인계 및 계약 동기화 프로토콜 (Agent Handoff Protocol)

**문서 ID**: AGENT-HANDOFF-003  

---

## 1. 5단계 표준 작업 인계(Handoff) 절차

1. **Step 1 (Work Order Claim & Lock)**: 담당 에이전트는 선행 Work Order(`prerequisite_wo`)가 `AUDIT_PASSED` 상태인지 확인한 후 대상 파일 경로에 배타적 락을 획득하고 상태를 `IN_PROGRESS`로 전환한다.
2. **Step 2 (Implementation & Self-Verification)**: 로컬 격리 컨테이너 환경에서 코드를 구현하고 L1(단위), L2(DB/RLS), L3(OpenAPI 계약) 테스트를 실행한다.
3. **Step 3 (Handoff Artifact Generation)**: 구현 에이전트는 수정/생성된 파일 목록, SHA-256 해시, 테스트 실행 표준 출력(`stdout/stderr`), 그리고 외부 승인 대기(`BLOCKED`) 여부를 `REPORTING_TEMPLATE.md` 포맷으로 작성하여 제출한다.
4. **Step 4 (QA & Security Verification)**: `QA Agent`와 `Security Agent`가 테넌트 격리 침투 테스트 및 회계 차변/대변 일치성 검사를 수행한다.
5. **Step 5 (Independent Audit Sign-Off)**: `Independent Audit Agent`가 구현 에이전트의 자체 보고서를 신뢰하지 않고 직접 테스트와 체크섬을 재실행하여 이상이 없을 때만 `AUDIT_PASSED`로 최종 마감하고 후속 의존 에이전트에게 큐를 개방한다.
