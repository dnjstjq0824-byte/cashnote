# OpenTelemetry 기반 통합 관제·메트릭·회계 정합성 알람 명세서

**문서 ID**: DEVOPS-MON-003  

---

## 1. 4대 핵심 관제 영역 및 경보 임계치 (Alerting Thresholds)

| 관제 영역 | 메트릭 명칭 | 정상 기준 (SLO) | 경보 등급 및 즉각 조치 |
|---|---|---|---|
| **회계 원장 무결성** | `bn_mos_ledger_unbalanced_entries_total` | **항상 0** | **P0 CRITICAL (즉시 호출)**: 단 1건이라도 `> 0` 발생 시 해당 매장 원장 쓰기 잠금 및 온콜 엔지니어 즉시 호출 |
| **보안·감사 무결성** | `bn_mos_audit_hash_chain_broken_total`, `bn_mos_idor_blocked_total` | 해시 깨짐 **0**, 동일 IP IDOR 시도 `< 5/min` | **P0 CRITICAL**: 해시 변조 감지 시 배포 중단 및 포렌식 스냅샷 보존; IDOR 반복 시도 IP 자동 WAF 차단 |
| **외부 데이터 수집 & 대사** | `bn_mos_connector_sync_success_rate`, `bn_mos_morning_briefing_completed_at` | 성공률 `≥ 99.0%`, 모닝 브리핑 `08:30 이전 완료` | **P1 HIGH**: 특정 카드사/은행 API 장애 시 자동으로 `SYNC_DELAY_PENDING` 배지 활성화(오탐 미입금 경보 발송 중단) |
| **API & DB 인프라** | `http_server_duration_ms (P95/P99)`, `pg_stat_replication_lag_bytes` | API P95 `< 150ms`, DB 복제 지연 `< 1MB` | **P2 WARN / P1 HIGH**: 워커 오토스케일링 트리거 및 슬로우 쿼리 분석 |
