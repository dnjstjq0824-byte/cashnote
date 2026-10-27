# POS·VAN·PG 멀티 프로바이더 어댑터 명세서 (Provider Adapter Pattern)

**문서 ID**: CONN-POS-004  
**승인 상태 분류**: PG사 Open API (`CONTRACT_REQUIRED`), POS/VAN사 (`PARTNER_ONLY` + `LEVEL_2_FILE`)

---

## 1. 이기종 POS·VAN·PG 표준 정규화 및 중복 제거 설계 (제12조 반영)

매장마다 사용하는 POS(페이히어, 오케이포스, 아임유, 토스플레이스, 포스뱅크 등), VAN, PG(토스페이먼츠, 나이스페이, KG이니시스, NHN KCP)가 다르므로 다음 3가지 수집 채널을 단일 표준 이벤트(`CanonicalPosOrderEvent`)로 정규화한다:

1. **PG사 공식 REST API (예: 토스페이먼츠)**:
   - 가맹점이 발급받은 상점 Secret Key를 등록하면 `GET https://api.tosspayments.com/v1/settlements` 및 `GET /v1/transactions`를 호출하여 온라인/간편결제 승인·취소·정산 데이터를 실시간 수집한다.
2. **제휴 POS / 테이블오더 / 키오스크 Inbound Webhook (`POST /v1/stores/{storeId}/connectors/pos/events`)**:
   - 판매 시각, 주문 ID, 결제 ID, 결제수단, 상품(SKU/수량/단가/할인), 취소 여부, 수수료, 거래 상태를 실시간 수신한다.
3. **3-Way 중복 제거 알고리즘 (POS 매출 ↔ 카드사 매입 ↔ 은행 입금)**:
   - POS에서 올라온 카드 결제 건(`approval_number = 38491022`, `amount = 33,000`)과 익일 여신협회에서 수집된 매입 건(`approval_number = 38491022`, `amount = 33,000`)이 만날 때, 새로운 매출 행을 추가하지 않고 기존 `sales_transactions` 레코드에 `payment_captures` 정보를 보강(Enrich)하여 **중복 매출 합산(Double Counting)을 100% 방지**한다.
