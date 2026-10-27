# 개인정보 및 제3자 데이터 연동 동의·철회 관리 명세서 (Consent Lifecycle)

**문서 ID**: SEC-CONSENT-004  

---

## 1. 동의 유형별 분리 및 법적 요건 (`connection_consents`)

포괄적 일괄 강제 동의를 지양하고 목적별로 동의를 명확히 분리한다:

| 동의 코드 (`consent_type`) | 필수/선택 | 동의 시점 | 철회 시 시스템 자동 조치 |
|---|---|---|---|
| `TERMS_OF_SERVICE` | 필수 | 회원가입 시 | 회원 탈퇴 워크플로우로 연결 |
| `PRIVACY_COLLECTION` | 필수 | 회원가입 시 | 회원 탈퇴 및 개인정보 파기/분리보관 |
| `FINANCIAL_DATA_DELEGATION` | 선택 (연동 시 필수) | 카드·은행·홈택스·POS 커넥터 연결 시 | 해당 `data_connections` 즉시 `REVOKED` 전환, 외부 토큰 폐기 API 호출, 정기 수집 중단 |
| `TAX_AGENT_SHARING` | 선택 | 세무대리인(`ROLE_TAX_AGENT`) 초대 또는 세무 패키지 공유 시 | 즉시 세무대리인 접근 권한(`store_members`) 회수 및 공유 다운로드 링크 만료 처리 |
| `THIRD_PARTY_DATA_PROVISION` | 선택 | 제휴 서비스 상담 신청 시 | 제3자 제공 즉시 중단 및 철회 통지 |
| `MARKETING_OPT_IN` | 선택 | 회원가입 또는 설정 화면 | 프로모션 알림톡/이메일/푸시 발송 대상에서 즉시 제외 (수신거부 처리 확인 안내 발송) |
