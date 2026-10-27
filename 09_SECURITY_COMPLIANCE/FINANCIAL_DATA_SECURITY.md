# 금융 데이터 보안·암호화·비밀번호 무수집 아키텍처 (Financial Data Security)

**문서 ID**: SEC-FIN-002  
**관련 조항**: 제21조, 제31조

---

## 1. 제21조 & 제31조 절대 원칙: 금융기관 비밀번호·인증서 무단 수집 및 우회 금지

1. **Zero Master Password Storage**:
   - BN Merchant OS는 은행 인터넷뱅킹 ID/비밀번호, 카드사 웹사이트 비밀번호, 배달앱 사장님 비밀번호, 공동인증서 개인키(`.key`) 및 암호를 자체 DB나 로그에 평문/가역 형태로 무단 수집·저장하는 테이블 컬럼조차 두지 않는다.
2. **공식 위임 토큰(OAuth 2.0 / API Key) 및 KMS 봉투 암호화**:
   - 금융결제원 오픈뱅킹 `access_token`/`refresh_token`, PG사 상점 API Key, 전자세금계산서 ASP 연동 토큰은 AWS KMS(또는 HashiCorp Vault)의 Customer Master Key(CMK)로 파생된 테넌트별 Data Encryption Key(DEK)를 사용해 `AES-256-GCM`으로 암호화(`data_connections.credentials_encrypted`)하여 저장한다.
   - 추가 인증 데이터(AAD, Additional Authenticated Data)에 `store_id`와 `business_id`를 바인딩하여, 설령 DB 바이트가 다른 매장 레코드로 복사되더라도 복호화가 불가능하도록 설계한다.
