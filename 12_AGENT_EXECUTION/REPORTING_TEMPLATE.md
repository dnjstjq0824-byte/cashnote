# Work Order 완료 및 독립 감사 제출 표준 보고서 양식 (Reporting Template)

**문서 ID**: AGENT-REPORT-006  

---

## 표준 제출 양식 (Markdown / JSON 겸용)

```markdown
# WORK ORDER EXECUTION & AUDIT REPORT
- **Work Order ID**: WO-MOS-XXXX
- **Title**: [작업명]
- **Implementing Agent**: [담당 에이전트명]
- **Git Branch & Commit SHA**: `wo/WO-MOS-XXXX` @ `<40-char-actual-git-sha>`
- **Execution Status**: `AUDIT_PASSED` | `CONTRACT_TEST_PASSED_LIVE_BLOCKED` | `FAILED`

## 1. Files Created / Modified (with SHA-256)
- `path/to/file1.go` — `sha256: ...`
- `path/to/file1_test.go` — `sha256: ...`

## 2. Verification & Test Evidence
- **Unit & Invariant Tests**: Executed N tests, 0 failures (`go test -v -race ./...`)
- **DB & RLS Isolation Tests**: Verified cross-tenant query returns 0 rows / 403 Forbidden
- **OpenAPI Contract Check**: 0 schema violations
- **E2E Scenario Link**: Scenario [A~J] verified

## 3. External API & Regulatory Gate Status
- **External Dependency**: [예: 여신금융협회 / 오픈뱅킹 / 배달앱 / 해당없음]
- **Live API Status**: `LIVE_VERIFIED` | `BLOCKED_PENDING_CONTRACT (Level 2 File Import Active)`

## 4. Independent Audit Sign-Off (Completed ONLY by Independent Audit Agent)
- **Auditor Agent**: `Independent Audit Agent`
- **Audit Timestamp**: YYYY-MM-DDTHH:MM:SSZ
- **Audit Verdict**: `PASS` | `REJECT`
- **Auditor Signature Hash**: `<sha256-of-evidence-bundle>`
```
