# 미지급·누락 매출 진단 및 자동 입금 대사(Reconciliation) 알고리즘 명세서

**문서 ID**: FIN-RECON-003  
**관련 Work Order**: `WO-MOS-0251` ~ `WO-MOS-0300`

---

## 1. Level 4 자동 대사 엔진 의사코드 (Go Reference Algorithm)

```go
// ReconcileStorePayouts matches scheduled payouts against actual bank deposits
// without ever conflating sync lag with confirmed unpaid defaults.
func (e *ReconciliationEngine) ReconcileStorePayouts(ctx context.Context, storeID uuid.UUID, asOfDate time.Time) (*ReconciliationReport, error) {
    schedules, err := e.repo.GetPendingPayoutSchedules(ctx, storeID, asOfDate)
    if err != nil {
        return nil, err
    }
    deposits, err := e.repo.GetUnmatchedInboundDeposits(ctx, storeID, asOfDate.AddDate(0, 0, -14), asOfDate.AddDate(0, 0, 3))
    if err != nil {
        return nil, err
    }
    lastBankSyncAt, err := e.repo.GetLatestBankDataTimestamp(ctx, storeID)
    if err != nil {
        return nil, err
    }

    for _, sched := range schedules {
        // 1. Exact 1:1 Match (Same Partner, Same Date, Variance == 0)
        if dep := findExactMatch(sched, deposits); dep != nil {
            e.recordMatch(ctx, sched, dep, "EXACT_1_TO_1", "MATCHED", 0, "NONE", 100.0)
            continue
        }
        // 2. Fee Rounding Tolerance Match (±10 KRW)
        if dep, diff := findToleranceMatch(sched, deposits, 10); dep != nil {
            e.recordMatch(ctx, sched, dep, "FEE_TOLERANCE_MATCH", "MATCHED", diff, "FEE_RATE_DIFFERENCE", 95.0)
            continue
        }
        // 3. Holiday Shift or Next-Business-Day Match (+1~3 Business Days)
        if dep, diff := findShiftedDateMatch(sched, deposits, 10); dep != nil {
            e.recordMatch(ctx, sched, dep, "EXACT_1_TO_1", "MATCHED", diff, "HOLIDAY_SHIFT", 92.0)
            continue
        }
        // 4. Partial Deposit / Post-Capture Cancel Offset Match
        if dep, diff := findPartialVarianceMatch(sched, deposits); dep != nil {
            e.recordMatch(ctx, sched, dep, "SPLIT_1_TO_N", "VARIANCE_DETECTED", diff, "POST_CAPTURE_CANCEL_OFFSET", 85.0)
            continue
        }
        // 5. Unmatched: Strictly distinguish Sync Delay vs Delayed Candidate
        if sched.ScheduledPayoutDate.Before(asOfDate) {
            if lastBankSyncAt.Before(sched.ScheduledPayoutDate.Add(18 * time.Hour)) {
                e.recordUnmatched(ctx, sched, "SYNC_DELAY_PENDING", "DATA_SYNC_LAGGING", 75.0)
            } else {
                e.recordUnmatched(ctx, sched, "DELAYED_CANDIDATE", "UNKNOWN_SHORTFALL", 88.0)
            }
        }
    }
    return e.buildSummaryReport(ctx, storeID)
}
```
