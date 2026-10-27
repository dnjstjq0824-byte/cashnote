# 카드매출 6단계 생명주기 및 수수료·예외 처리 정밀 설계서

**문서 ID**: FIN-CARD-001  
**관련 조항**: 제7조(핵심 매출·정산·입금 엔진), 제8조(미지급·누락 매출 진단 엔진)

---

## 1. 카드 거래 6단계 금액 정의 및 수학적 불변식

단일 카드 승인 거래 $T_i$에 대하여 다음 금액 변수를 엄격히 분리 정의한다:

- $A_i$ (`authorized_amount`): 카드 승인 총액 $= S_i (	ext{공급가액}) + V_i (	ext{부가가치세}) + P_i (	ext{봉사료})$
- $C^{	ext{pre}}_i$: 매입 전 취소 금액 (당일 승인 취소 등)
- $M_i$ (`captured_amount`): 전표 매입 확정 금액 $= A_i - C^{	ext{pre}}_i$
- $r_{	ext{fee}}$ (`applied_fee_rate`): 적용 수수료율 (신용/체크 구분 및 영세·중소가맹점 우대수수료율 반영)
- $F_i$ (`fee_amount`): 카드사 공제 수수료 $= 	ext{round\_rule}(M_i 	imes r_{	ext{fee}})$ (카드사별 원단위 절사 `FLOOR` 또는 사사오입 `HALF_UP` 규칙 테이블 적용)
- $C^{	ext{post}}_k$: 이전 매출분 중 당기 정산일에서 상계 차감되는 매입 후 취소(환불/차지백) 금액
- $E_i$ (`net_settlement_scheduled_amount`): 순정산 입금 예정액 $= M_i - F_i - C^{	ext{post}}_k$
- $D_j$ (`actual_bank_deposit_amount`): 실제 은행 계좌 입금액

---

## 2. 18대 카드 정산 예외 케이스 처리 매트릭스

| 예외 번호 | 예외 시나리오 | 원장 및 정산 스케줄 처리 알고리즘 |
|---|---|---|
| **EX-01** | 승인 후 당일 즉시 취소 (매입 전 취소) | `payment_authorizations.auth_status = VOIDED_SAME_DAY`, 매입 대상(`payment_captures`) 생성 제외, 일매출 집계에서 총매출·취소액 각각 병기 후 순매출 0원 처리 |
| **EX-02** | 매입 완료 후 전액 취소 (일시불/할부) | 원 거래일 매출은 유지하되 취소일자에 `payment_cancellations`(`POST_CAPTURE_FULL_CANCEL`) 생성 후, 해당 카드사의 차기 지급 예정액(`payout_schedules.hold_or_offset_amount`)에서 상계 차감 |
| **EX-03** | 부분 취소 (Partial Cancellation) | 원 승인번호 참조 하에 부분 취소 금액만큼만 `canceled_amount` 누적 및 잔여 유효 매입액 기준으로 수수료 재계산 |
| **EX-04** | 차기 정산 예정액보다 매입후취소 상계액이 큰 경우 (음수 정산) | 당일 입금 예정액을 `0원`으로 표시하고 미상계 잔액(Negative Carryover)을 다음 영업일 정산 스케줄로 자동 이월 추적 |
| **EX-05** | 동일 카드·동일 금액 3분 이내 중복 승인 | `DUPLICATE_AUTH_SUSPECTED` 경보 생성 (단, 실제 2건 결제일 수 있으므로 자동 삭제하지 않고 사장님 확인 토글 제공) |
| **EX-06** | 할부 거래 (무이자 vs 유이자 vs 가맹점 부담 무이자) | 일반 카드사 무이자는 일시불과 동일 정산; 가맹점 부담 무이자 특약 시 추가 할부수수료율을 `applied_fee_rate`에 합산 반영 |
| **EX-07** | 반기별 영세·중소가맹점 우대수수료 소급 환급 | 매년 1월/7월 우대수수료 재산정 및 신규가맹점 환급액을 별도 `FEE_REFUND_ADJUSTMENT` 전표로 분리 기장(일반 매출과 혼합 금지) |
| **EX-08** | 카드사별 지급 주기 차이 (D+2 vs D+3 영업일) | 카드사별·전표매입방식(EDC/VAN 직매입)별 지급주기 마스터 테이블 참조하여 `expected_payout_date` 산출 |
| **EX-09** | 주말·법정공휴일·대체공휴일·근로자의날(금융권 휴무) | 한국 금융결제원/은행 영업일 캘린더를 적용하여 휴일 도래 시 익영업일로 `scheduled_payout_date` 이동 및 `original_holiday_shifted_from` 기록 |
| **EX-10** | 복수 거래 합산 입금 (N건 승인 → 1건 은행 입금) | 동일 카드사·동일 지급예정일의 모든 매입 건 순정산액을 합산한 `payout_schedules` 헤더와 `bank_deposits` 단건을 `BATCH_N_TO_1` 매칭 |
| **EX-11** | 복수 가맹점번호(MID) 보유 매장의 통합 입금 vs 분리 입금 | 매장 내 홀 단말기 MID와 키오스크 MID가 같은 계좌로 들어올 때 개별/합산 Subset-Sum 양방향 대사 지원 |
| **EX-12** | 카드사 지급 보류 (Risk Hold / 한도 초과 / 압류) | 매입은 완료되었으나 카드사 정산 명세상 보류 코드가 확인되면 `CONFIRMED_ON_HOLD`로 즉시 분류하고 긴급 알림 발송 |
| **EX-13** | 부분 입금 (일부 전표만 선입금, 고액 할부/해외카드 지연 입금) | 입금 확인된 금액만큼 `PARTIALLY_DEPOSITED` 처리하고 차액(`variance_amount`)만 미수금 잔액으로 추적 |
| **EX-14** | 건별 수수료 원단위 절사 vs 합계 절사 오차 | 건별 수수료 합계와 일괄 정산 수수료 간 `±10원` 이내 단수 차이 발생 시 `FEE_TOLERANCE_MATCH`로 자동 흡수 및 수수료 보정 분개 생성 |
| **EX-15** | 체크카드 vs 신용카드 수수료율 혼선 | BIN 테이블 및 승인 전표의 `card_type`(`CREDIT` vs `CHECK_DEBIT`)을 구분하여 체크카드 우대 요율(예: 0.25%~1.0%) 정확히 적용 |
| **EX-16** | 해외 발행 카드 (Visa/Master/Amex/JCB/UnionPay 해외 BIN) | 국내 우대수수료 미적용 대상임을 식별하고 해외카드 일반 수수료율 및 D+3~D+5 정산 주기 적용 |
| **EX-17** | POS 승인 내역과 여신협회 매입 내역의 중복 수집 | `(store_id, card_acquirer_code, approval_number, approved_date, authorized_amount)` 해시 키로 단일 거래로 병합하고 상태를 `AUTHORIZED → CAPTURED`로 승격 |
| **EX-18** | 외부 데이터 수집 지연 (은행 API/파일 미갱신) | 지급 예정일이 지났더라도 계좌 데이터의 마지막 동기화 시각이 예정일 마감 이전이면 `SYNC_DELAY_SUSPECTED`로 분류하여 허위 미입금 경보 차단 |
