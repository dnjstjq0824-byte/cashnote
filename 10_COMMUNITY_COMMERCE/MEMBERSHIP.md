# 유료 멤버십(Free / Plus / Pro / Enterprise) 구독 및 결제 정산 설계서

**문서 ID**: COMM-MEM-002  
**관련 조항**: 제16조, 제20조(Subscription API)

---

## 1. 멤버십 상태 머신 (Subscription Lifecycle State Machine)

```mermaid
stateDiagram-v2
    [*] --> FREE : 매장 생성 기본값
    FREE --> TRIALING : 14일 무료 체험 시작 (최초 1회)
    FREE --> ACTIVE : 유료 플랜 즉시 결제 (PG 빌링키)
    TRIALING --> ACTIVE : 체험 종료일 정기결제 성공
    TRIALING --> FREE : 체험 기간 중 해지 예약 또는 결제 실패
    ACTIVE --> CANCELED_AT_PERIOD_END : 사용자가 [구독 해지] 클릭 (잔여 기간 혜택 유지)
    CANCELED_AT_PERIOD_END --> FREE : 이용 기간 만료일 도래
    ACTIVE --> PAST_DUE : 정기결제일 카드 한도초과/잔액부족 실패
    PAST_DUE --> ACTIVE : D+1 / D+3 / D+5 재결제 성공 또는 결제수단 변경
    PAST_DUE --> EXPIRED : 7일 유예기간 경과 (무료 플랜 전환, 기존 장부 데이터 열람 100% 보존)
```

---

## 2. 소비자 보호 및 환불 정산 규칙

- **데이터 인질극(Data Lock-in) 절대 금지**: 유료 구독이 해지되거나 `EXPIRED`되어 무료 플랜으로 전환되더라도, 과거에 수집·입력된 매출·비용·세무 원장 데이터의 조회 및 엑셀 내보내기 권한은 영구히 보장한다.
- **청약철회 및 일할 환불**: 결제 후 7일 이내에 유료 전용 기능(미입금 상세 진단 내보내기, 마켓 쿠폰팩 사용 등) 사용 이력이 없으면 100% 전액 즉시 환불하며, 사용 이력이 있거나 7일 경과 시 잔여 일수 일할 계산 환불 공식을 투명하게 표시하고 처리한다.
