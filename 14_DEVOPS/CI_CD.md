# CI/CD 자동 검증·보안 스캔·계약 게이트 파이프라인 명세서

**문서 ID**: DEVOPS-CICD-001  

---

## 1. 7-Stage CI/CD 파이프라인 아키텍처

모든 Pull Request(`wo/WO-MOS-XXXX`)와 릴리스 브랜치는 다음 7단계 자동 게이트를 통과해야만 병합 및 배포될 수 있다:

1. **Stage 1 — Static Analysis & Forbidden Pattern Gate**:
   - Go (`golangci-lint`), TypeScript (`eslint`, `vue-tsc`), Dart (`flutter analyze`).
   - 커스텀 규칙 검사: (a) 금액 연산에 `float32`/`float64` 사용 시 빌드 실패, (b) `journal_entries`나 `audit_logs`에 `DELETE` 쿼리 작성 시 빌드 실패, (c) GitLeaks 시크릿 노출 감지 시 즉시 차단.
2. **Stage 2 — Schema & Contract Validation Gate**:
   - `03_DATABASE/migrations/*.sql`을 Ephemeral PostgreSQL 16 컨테이너에 Up → Down → Up 순차 적용하여 마이그레이션 무결성과 35개 테이블 RLS 활성화 여부 검증.
   - `04_API_SPECIFICATION/openapi.yaml` OpenAPI 3.1 린트 및 클라이언트 SDK 재생성 시 Diff가 없는지(계약 동기화) 검증.
3. **Stage 3 — Unit & Accounting Invariant Property Gate**:
   - 수수료·세금·정산일·대사 알고리즘 단위 테스트 및 10,000건 무작위 전표 차변==대변 불변성 테스트 실행.
4. **Stage 4 — Multi-Tenant Isolation & Security Penetration Gate**:
   - 사업장 A 토큰으로 사업장 B 리소스 접근 시 100% 403/0행 차단 검증 및 컨테이너/의존성 취약점(Trivy Critical/High == 0) 스캔.
5. **Stage 5 — E2E Scenario Suite (Scenario A ~ Scenario J)**:
   - Playwright 웹 E2E 및 Flutter Integration Test 실행 (외부 미계약 API는 격리된 WireMock Contract Server 사용).
6. **Stage 6 — Independent Audit Artifact Signing**:
   - 테스트 로그, 커버리지, 바이너리/컨테이너 이미지 다이제스트(`sha256:...`)를 서명하여 아카이빙.
7. **Stage 7 — Blue-Green Deployment & Auto-Rollback**:
   - 카나리/블루-그린 배포 후 5분간 에러율(`> 0.1%`) 또는 P99 레이턴시(`> 500ms`), 원장 불균형 경보 발생 시 30초 이내 이전 버전으로 자동 롤백.
