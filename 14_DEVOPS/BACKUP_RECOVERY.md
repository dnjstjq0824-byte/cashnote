# 데이터베이스 백업·특정시점복구(PITR)·재해복구(DR) 설계서

**문서 ID**: DEVOPS-DR-004  

---

## 1. 금융 원장 등급 백업 아키텍처 (RPO = 0 ~ 5분, RTO ≤ 30분)

1. **실시간 동기/반동기 복제 (Synchronous Standby)**:
   - PostgreSQL 16 Primary 노드의 모든 원장 트랜잭션(`journal_entries`, `sales_transactions`, `reconciliation_matches`)은 다른 가용영역(AZ)의 Standby 노드에 WAL 로그가 기록된 후 커밋 응답을 반환하여 단일 서버 장애 시 **데이터 유실 제로(RPO = 0)**를 보장한다.
2. **지속적 WAL 아카이빙 및 35일 PITR (Point-In-Time Recovery)**:
   - `pgBackRest` / `WAL-G`를 통해 60초 주기로 암호화된 WAL 세그먼트를 S3(WORM Object Lock 활성화 버킷: 랜섬웨어 및 관리자 실수로 인한 백업 삭제 불가)에 전송한다.
3. **주간 자동 복구 훈련 (Automated Weekly Restore Drill)**:
   - 매주 일요일 03:00에 최신 백업을 임시 격리 DB 인스턴스에 자동 복원한 뒤, `SELECT SUM(total_debit_amount) - SUM(total_credit_amount) FROM journal_entries`가 정확히 `0.0000`인지 검증하고 리포트를 생성한다.
