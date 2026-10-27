# 신규 사장님 온보딩·매출장부·세금·다매장·리뷰 웹 E2E 테스트 명세서 (Scenario A, B, E, G, H)

**문서 ID**: TEST-USER-003  

---

## 1. Scenario A: 신규 사장님 온보딩 및 2단계 소유권 검증 E2E
1. `POST /v1/auth/register` 호출 및 휴대전화 본인인증(성명: 홍길동) 완료.
2. `POST /v1/businesses`로 사업자등록번호 `123-45-67890` 입력 → 국세청 상태조회 API가 `ACTIVE (계속사업자, 일반과세자)` 반환 →此时 `ownership_verification_status == NTS_VALIDATED`.
3. 이 상태에서 `GET /v1/stores/{id}/bank-deposits` 또는 외부 금융 커넥터 연결 호출 시 `403 OWNERSHIP_VERIFICATION_REQUIRED` 반환 확인.
4. `POST /v1/businesses/{id}/verify-ownership`으로 사업자 계좌 1원 인증코드 `4829` 검증 성공 → `ownership_verification_status == VERIFIED_OWNER` 전환 및 전체 대시보드 진입 확인.

## 2. Scenario B & E: 매출·비용 장부 기장 및 부가세 신고 준비 내보내기 E2E
1. 여신협회 카드매출 엑셀(합계 3,300만 원) 및 식자재 매입 세금계산서 엑셀(합계 1,100만 원), 접대비 카드영수증(55만 원), 승용차 주유비(22만 원) 등록.
2. 손익계산서(P&L) 조회 시 순매출 3,000만 원, 원가 1,000만 원 등 정확히 산출.
3. 부가세 예상 시뮬레이션(`GET /v1/stores/{id}/tax/vat-estimate`) 조회 시 접대비 부가세(5만 원)와 비영업용 승용차 주유비 부가세(2만 원)가 **불공제 매입세액**으로 자동 제외되어 과다 공제 신고 리스크를 예방하는지 확인.

## 3. Scenario G & H: 다매장 권한 분리 및 리뷰 공식 API/클립보드 분기 E2E
1. 강남본점(`store_1`)과 홍대직영점(`store_2`) 생성 후 김매니저를 `store_1`에만 `MANAGER`로 초대.
2. 김매니저 토큰으로 `store_2` 매출 조회 시 `403 AUTH_FORBIDDEN_STORE` 차단 확인.
3. Google Business 리뷰(`supports_direct_api_reply = true`)는 대시보드에서 즉시 답글 API 호출 성공, 배달앱 리뷰(`supports_direct_api_reply = false`)는 AI 답글 초안 생성 및 클립보드 복사 상태(`COPIED_FOR_MANUAL_REPLY`) 기록 확인.
