# 카드·배달 정산 예정 vs 은행 입금 대사 계약 명세 (Reconciliation Contract)

**문서 ID**: API-RECON-004  

---

## 1. 대사 매칭 결정 트리 및 신뢰도 점수(Confidence Score) 산출 공식

`reconciliation_matches` 엔진은 다음 5단계 우선순위 알고리즘으로 `payout_schedules`(지급 예정)와 `bank_deposits`(실제 입금)를 연결한다:

| 우선순위 | 매칭 유형 (`match_type`) | 매칭 조건 | 신뢰도 점수 (`confidence_score`) | 결과 상태 (`match_status`) |
|---|---|---|---|---|
| **Rule 1** | `EXACT_1_TO_1` | 동일 정산계좌 + 입금일자 일치(`scheduled_payout_date == transaction_date`) + 입금자명 카드사/플랫폼 코드 일치 + 금액 오차 `0원` | **100.00** | `MATCHED` |
| **Rule 2** | `FEE_TOLERANCE_MATCH` | 날짜·카드사 일치 + 건별 수수료 원단위 절사/반올림 차이로 인한 금액 오차 `±10원 이내` | **95.00** | `MATCHED` (차액 수수료 보정 분개 생성) |
| **Rule 3** | `BATCH_N_TO_1` | 동일 카드사의 복수 가맹점번호(MID) 또는 공휴일 이연분이 합산되어 1건으로 입금된 경우 (`SUM(scheduled) == actual`) | **92.00** | `MATCHED` |
| **Rule 4** | `VARIANCE_DETECTED` | 날짜·카드사 입금은 확인되었으나 매입후취소 상계 또는 일부 지급보류로 인해 입금액이 예정액보다 작은 경우 (`0 < actual < scheduled`) | **85.00** | `VARIANCE_DETECTED` (차액 원인 분석 리포트 생성) |
| **Rule 5** | `UNMATCHED_DIAGNOSTIC` | 지급 예정일이 경과했으나 매칭되는 은행 입금이 없는 경우: <br/>- 계좌 마지막 동기화 시각이 예정일 18:00 이전이면 → `SYNC_DELAY_PENDING` (오탐 방지)<br/>- 계좌 동기화가 최신인데 입금이 없으면 → `DELAYED_CANDIDATE` | **70.00 ~ 90.00** | `SYNC_DELAY_PENDING` 또는 `DELAYED_CANDIDATE` |
