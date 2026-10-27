# 보안 위협 모델링 및 방어 아키텍처 (STRIDE Threat Model)

**문서 ID**: ARCH-THREAT-005  

---

## 1. STRIDE 기반 10대 핵심 위협 및 아키텍처 통제

| 위협 ID | STRIDE 분류 | 위협 시나리오 | 치명도 | 아키텍처 방어 대책 (Mitigation) | 검증 테스트 ID |
|---|---|---|---|---|---|
| **TM-01** | **Elevation of Privilege / IDOR** | 사업장 A의 사장님/직원이 API 파라미터의 `storeId`를 사업장 B의 UUID로 변조하여 타 매장의 매출·계좌 내역을 조회 | CRITICAL | 1) API 미들웨어에서 `store_members` 테이블 조회 검증 2) DB 트랜잭션 시작 시 `SET LOCAL app.current_store_id` 설정 3) PostgreSQL 56개 테이블 RLS 정책으로 교차 접근 원천 차단 | `SEC-E2E-001` (Scenario I) |
| **TM-02** | **Spoofing (소유권 도용)** | 공격자가 공개된 타인의 사업자등록번호로 매장을 허위 등록하여 카드매출·세금 정보를 연동 시도 | CRITICAL | 제6조 준수: 국세청 상태조회만으로 소유권을 인정하지 않으며, 대표자 본인명의 휴대전화 인증(CI) + 사업자 명의 계좌 1원 입금 인증(또는 서류 심사) 통과 전까지 `VERIFIED_OWNER` 부여 및 커넥터 연결 차단 | `SEC-E2E-002` (Scenario A) |
| **TM-03** | **Tampering (원장·감사로그 변조)** | 내부자 또는 해커가 DB에 접근하여 입금 금액이나 세금 추정 근거, 감사 로그를 사후 수정/삭제 | CRITICAL | `journal_entries`, `journal_lines`, `audit_logs` 테이블에 대해 애플리케이션 DB 계정의 `DELETE` 권한 박탈, 정정 시 `corrections` 역분개만 허용, `audit_logs` SHA-256 해시 체인 일일 검증 | `SEC-E2E-003` |
| **TM-04** | **Information Disclosure (자격증명 유출)** | 외부 API 토큰이나 은행 계좌번호가 로그 파일이나 DB 덤프에서 평문 노출 | HIGH | KMS 기반 AES-256-GCM 봉투 암호화(`*_encrypted`), 구조화 로거 레벨에서 정규식 기반 PII/Token 자동 마스킹 필터 적용, 타사 로그인 비밀번호 수집 자체를 금지 | `SEC-E2E-004` |
| **TM-05** | **Repudiation (부인 공격)** | 매니저나 세무대리인이 데이터를 내보내기하거나 지출을 수정한 뒤 행위를 부인 | HIGH | 모든 상태 변경 및 엑셀 내보내기 요청에 OIDC 세션 ID, 기기 핑거프린트, IP, `Idempotency-Key`를 `audit_logs`에 불변 기록 | `SEC-E2E-005` |
| **TM-06** | **Denial of Service / 악성 파일 업로드** | 공격자가 1GB짜리 압축 폭탄(Zip Bomb) 또는 수식 인젝션(CSV Formula Injection `=CMD|...`)이 포함된 정산 엑셀 업로드 | HIGH | 업로드 파일 최대 크기 제한(50MB), 스트리밍 파서 압축비 제한, 셀 문자열이 `=`, `+`, `-`, `@`로 시작할 경우 CSV 내보내기 시 이스케이프 강제 | `SEC-E2E-006` |
| **TM-07** | **Replay / Duplicate Webhook** | VAN/PG 웹훅이나 네트워크 재시도로 인해 동일 카드 매출이나 구독 결제가 2번 중복 반영 | HIGH | 모든 웹훅에 `X-Signature-Timestamp`(5분 윈도우) + HMAC 검증 적용, DB 레벨 `UNIQUE (store_id, dedup_fingerprint)` 제약으로 중복 삽입 100% 차단 | `SEC-E2E-007` (Scenario J) |
