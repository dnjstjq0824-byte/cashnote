# 배달 플랫폼(배달의민족·쿠팡이츠·요기요) 매출·정산 통합 명세서

**문서 ID**: CONN-DELIV-005  
**승인 상태 분류**: `PARTNER_ONLY` (공식 제휴 전 실시간 API는 `BLOCKED`) + `LEVEL_2_FILE` (공식 정산 명세서 엑셀/CSV 파서 기본 탑재)

---

## 1. 제11조 준수: 무단 스크래핑 배제 및 정직한 투트랙(Two-Track) 아키텍처

1. **절대 금지 원칙**: 배달의민족(배민셀프서비스), 쿠팡이츠(쿠팡이츠스토어), 요기요(사장님사이트)의 사장님 ID/비밀번호를 수집하여 약관 및 캡차를 우회해 무단 로그인·스크래핑하는 방식을 **절대 채택하지 않는다**.
2. **Track 1 — Level 2 공식 정산파일 자동 파서 (출시 첫날부터 즉시 가동)**:
   - 사장님이 각 배달앱 사장님 사이트에서 정식으로 내려받은 **월간/주간 정산 내역 엑셀(`.xlsx`)/CSV 파일**을 업로드하면 다음 11개 항목을 1원 단위로 자동 분해하여 원장에 기장한다:
     - `gross_order_amount` (주문 금액 합계)
     - `canceled_order_amount` (주문 취소 금액)
     - `platform_mediation_fee` (중개 이용료 / 주문중개수수료)
     - `delivery_agency_fee` (사장님 부담 배달비 / 배달대행료)
     - `payment_gateway_fee` (외부 결제수수료 / 카드·간편결제 정산수수료)
     - `merchant_coupon_cost` (사장님 부담 할인쿠폰·프로모션 비용)
     - `ad_product_deduction` (울트라콜·오픈리스트·클릭당 광고비 차감액)
     - `vat_on_fees` (수수료/배달비에 대한 부가가치세 — 매입세액 공제 대상 분리)
     - `scheduled_settlement_date` (지급 예정일)
     - `net_settled_amount` (실제 정산 입금 예정액)
     - `bank_reconciliation_status` (사업용 계좌 실제 입금액과의 일치 여부)
3. **Track 2 — Level 3 공식 B2B 파트너 API Adapter (`PARTNER_ONLY`)**:
   - 각 배달 플랫폼 또는 공식 POS 주문중계 파트너십 계약이 체결된 경우에만 OAuth2/Webhook 기반 자동 동기화를 활성화한다.
