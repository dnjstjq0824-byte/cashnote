# 금융·회계 데이터 정합성 보장 및 오차 제로(Zero-Discrepancy) 검증 규격

**문서 ID**: FIN-ACC-006  

---

## 1. 7대 데이터 정합성 불변 조건 (Accounting & Reconciliation Invariants)

1. **Invariant 1 (무손실 십진 연산)**: Go 백엔드에서는 `shopspring/decimal`을 사용하고 PostgreSQL에서는 `NUMERIC(19,4)`를 사용하여 `0.1 + 0.2 != 0.3`과 같은 IEEE 754 부동소수점 오차가 절대 발생하지 않는다.
2. **Invariant 2 (복식부기 대차평균)**: 모든 `journal_entries` 레코드에 대해 $\sum 	ext{debit\_amount} = \sum 	ext{credit\_amount}$가 트랜잭션 커밋 시점에 강제된다.
3. **Invariant 3 (다중 채널 중복 산입 금지)**: 동일 카드 결제가 POS, VAN, 여신협회, 은행 입금 4곳에서 수집되더라도 `sales_transactions`의 매출액은 단 1번만 계상되며, 각 수집 이벤트는 거래의 상태(`AUTHORIZED → CAPTURED → SETTLED`)만 전이시킨다.
4. **Invariant 4 (원본 불변성)**: 한 번 생성된 거래나 원장 전표는 `DELETE`되지 않으며 모든 수정은 `corrections` 테이블의 역분개 체인을 통해서만 이루어진다.
5. **Invariant 5 (세금·수수료 분리)**: 부가가치세 예수금과 카드/배달 수수료는 매출에서 임의로 차감하여 순액으로만 덮어쓰지 않고 반드시 총액(Gross)과 차감 항목을 각각 보존한다.
6. **Invariant 6 (출처 투명성)**: 모든 집계 숫자는 드릴다운 시 어떤 원본 배치(`raw_import_batch_id`)의 몇 번째 행 또는 어떤 API 응답에서 유래했는지 100% 역추적(Lineage) 가능해야 한다.
7. **Invariant 7 (시간대 일관성)**: 모든 거래는 UTC `TIMESTAMPTZ`와 매장 기준 영업일(`business_date`, 심야 영업 마감 보정 포함) 두 가지 축을 모두 보유한다.
