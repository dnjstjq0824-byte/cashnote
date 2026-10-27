# 은행 계좌 및 금융결제원 오픈뱅킹 데이터 연동 명세서

**문서 ID**: CONN-BANK-002  
**승인 상태 분류**: `APPROVAL_REQUIRED` (금융결제원 오픈뱅킹 이용기관 승인) / `COMMERCIAL_PROVIDER` (펌뱅킹/계좌 중계기관)

---

## 1. 금융결제원 오픈뱅킹 공동업무 API 연동 규격

- **공식 근거**: 금융결제원 오픈API 개발자사이트 (`https://developers.kftc.or.kr/dev/openapi/open-banking`) 및 오픈뱅킹공동업무 이용약관 제14조(이용적합성 승인).
- **핵심 활용 API**:
  1. `POST /oauth/2.0/token`: 3-Legged OAuth 2.0 기반 사용자 계좌 연결 인가코드 토큰 발급.
  2. `GET /v2.0/account/balance/fin_num`: 핀테크이용번호 기반 실시간 계좌 잔액 조회.
  3. `GET /v2.0/account/transaction_list/fin_num`: 기간별 입출금 거래내역 조회 (`bank_deposits` 적재 및 카드/배달 정산금 자동 대사).
  4. `POST /v2.0/transfer/deposit/fin_num` (또는 1원 입금 인증 전용 API): 사업장 소유권 확인(`SCR-SM-01`)을 위한 사업자 명의 계좌 1원 입금 및 입금자명 4자리 난수 검증.
- **미승인 기간 Fallback (Level 2)**:
  - 오픈뱅킹 이용적합성 심사 완료 전에도 모든 시중은행(국민/신한/우리/하나/농협/기업/카카오/토스 등)의 기업·개인 인터넷뱅킹 거래내역 엑셀/CSV 파일을 자동 인식하는 `UniversalBankStatementParser`를 제공한다.
