# 실서비스 표준 운영 런북 (Standard Operations Runbook)

**문서 ID**: DEVOPS-RUNBOOK-005  

---

## 1. 주요 장애 및 운영 시나리오별 즉각 대응 절차 (SOP)

### SOP-01: 외부 기관(여신협회/특정 카드사/오픈뱅킹) 서버 정기점검 또는 응답 지연 발생 시
1. 관리자 콘솔 `SCR-AD-05`에서 해당 `provider_code`의 상태를 점검 모드로 전환하거나 Circuit Breaker가 자동으로 `OPEN`되었는지 확인한다.
2. 대사 워커(`bn-mos-worker-recon`)가 해당 기관의 미수집 데이터를 **실제 미입금 사고(`CONFIRMED_UNPAID`)로 잘못 분류하지 않고 `SYNC_DELAY_PENDING`으로 보류**하고 있는지 `SCR-AD-06`에서 확인한다.
3. 긴급 확인이 필요한 사장님은 기다리지 않고 즉시 엑셀 파일을 올릴 수 있도록 대시보드 상단에 `[기관 점검 안내 및 정산파일 직접 업로드 바로가기]` 배너를 노출한다.

### SOP-02: 배달 플랫폼(배민/쿠팡이츠/요기요) 정산 엑셀 다운로드 양식(컬럼명) 변경 감지 시
1. `ingestion_errors`에 `HEADER_SCHEMA_MISMATCH` 경보가 유입되면 `SCR-AD-10`(배달 파서 스튜디오)에 진입한다.
2. 새 엑셀 헤더명을 표준 필드(`gross_order_amount`, `platform_mediation_fee`, `delivery_agency_fee`, `net_settled_amount`)에 매핑하는 신규 스키마 버전을 추가하고 Dry-Run 테스트를 통과시킨다.
3. 코드 재배포 없이 즉시 활성화한 뒤 실패했던 업로드 배치를 `SCR-AD-06`에서 일괄 재처리(`Replay`)한다.
