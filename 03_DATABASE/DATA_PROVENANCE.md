# 데이터 출처 추적 및 신뢰도 관리 모델 (Data Provenance & Lineage)

**문서 ID**: DB-PROV-004  

---

## 1. 출처 등급(Source Level) 및 UI 표시 정책

제4조 및 제31조에 의거하여, 외부 금융 데이터가 연결되지 않았는데 실시간 매출이나 실제 입금이 확인된 것처럼 표시하는 행위는 시스템 레벨에서 불가능하도록 설계한다.

| `source_level` 코드 | 정의 | 필수 FK / 메타데이터 | 프론트엔드 UI 배지 문구 | 대사 확정 가능 여부 |
|---|---|---|---|---|
| `LEVEL_1_MANUAL` | 사장님 또는 매니저가 앱/웹에서 직접 수기 입력한 거래 | `created_by_user_id`, `raw_import_batch_id` | `[수기 입력]` (회색 배지) | 추정 장부 반영 (은행 실제 입금으로 자동 간주하지 않음) |
| `LEVEL_2_FILE` | 여신협회, 카드사, 은행, 홈택스, 배달앱에서 다운로드한 정식 엑셀/CSV 업로드 | `raw_import_batch_id`, `file_sha256`, `s3_object_key` | `[정산파일 업로드 검증]` (파란색 배지) | 카드/배달 정산표 + 은행 거래내역 파일 업로드 시 Level 4 대사 수행 가능 |
| `LEVEL_3_API` | 공식 계약/승인된 외부 Open API 또는 파트너 Webhook 수신 | `source_provider_id`, `external_transaction_id` | `[공식 API 연동 · HH:MM 갱신]` (청록색 배지) | 자동 대사 파이프라인 직결 |
| `LEVEL_4_RECONCILED` | 지급 예정 원장과 실제 은행 입금 원장이 오차 없이 매칭 완료된 상태 | `reconciliation_match_id`, `confidence_score = 100` | `[입금 대사 완료 ✓]` (에메랄드 그린 배지) | 세무/회계 확정 원장 반영 |
