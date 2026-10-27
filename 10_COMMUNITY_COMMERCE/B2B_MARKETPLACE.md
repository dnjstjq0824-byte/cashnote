# 소상공인 B2B 마켓플레이스 및 장부·재고 자동 연동 설계서

**문서 ID**: COMM-B2B-004  
**관련 조항**: 제16조(B2B Marketplace)

---

## 1. 3대 커머스 계약 및 책임 모델 구분 (`commerce_model`)

제16조에 따라 B2B 마켓에 등록되는 모든 상품(`marketplace_products`)은 다음 세 가지 계약 및 법적 책임 구조를 명확히 구분하여 DB와 화면에 표시한다:

| 모델 코드 | 판매 및 계약 주체 | 세금계산서/영수증 발행 주체 | 배송·교환·환불·분쟁 1차 책임 | 정산 구조 |
|---|---|---|---|---|
| `FULFILLMENT_DIRECT` (직매입/직판) | BN Merchant OS 운영 법인 | 플랫폼 법인이 매입 세금계산서/카드전표 직접 발행 | 플랫폼 고객센터 직접 처리 | 결제 즉시 플랫폼 매출 인식 |
| `CONSIGNMENT_ESCROW` (통신판매중개) | 입점 검증된 B2B 공급업체 (`suppliers`) | 입점 공급업체(또는 PG 위수탁 발행) | 1차 공급업체 처리 + 미해결 시 플랫폼 에스크로 정산 보류 및 중재(`SCR-AD-13`) | 구매확정(배송완료+7일) 후 중개수수료 차감 후 PG 분리정산 |
| `AFFILIATE_OUTLINK` (제휴 링크) | 외부 제휴 쇼핑몰 | 외부 제휴 쇼핑몰 | 외부 제휴 쇼핑몰 (화면 상단 `'외부 제휴사 이동 상품'` 명시) | 제휴 CPS/CPA 정산 |

---

## 2. B2B 마켓 주문 → 매장 비용 장부(`expenses`) 및 재고(`inventory_movements`) 자동 기장 흐름

사장님이 B2B 마켓에서 식자재(예: 식용유 18L 2캔, 99,000원)를 주문하고 결제할 때 `[매장 장부 및 재고에 자동 반영]` 옵션(기본 켜짐)을 유지하면:
1. 결제 완료 시 `expenses` 테이블에 `supply_amount = 90,000`, `vat_amount = 9,000`, `expense_category = COGS_FOOD_MATERIAL`, `qualifying_evidence_type = CARD_SLIP`으로 자동 등록된다.
2. 복식부기 원장(`journal_entries`)에 `[차] 원재료비 90,000 + [차] 부가세대급금 9,000 / [대] 사업용카드미지급금 99,000` 분개가 자동 생성된다.
3. 배송 완료(`DELIVERED`) 웹훅 수신 시 매장 재고(`inventory_items`)의 식용유 수량이 `+2 CAN` 자동 입고(`PURCHASE_IN`) 처리된다.
