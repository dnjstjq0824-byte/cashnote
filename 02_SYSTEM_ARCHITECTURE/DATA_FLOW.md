# 4단계 데이터 수집 및 카드 매출·정산·대사 데이터 흐름도 (Data Flow)

**문서 ID**: ARCH-DATAFLOW-003  

---

## 1. 제4조 4단계 데이터 수집 파이프라인 (Level 1 ~ Level 4)

모든 외부 및 내부 데이터는 수집 경로와 무관하게 **단일 정규화 파이프라인(Canonical Normalization Pipeline)**을 통과하며, 원본 배치 추적성(`raw_import_batch_id`)과 출처 등급(`source_level`)을 유지한다.

```mermaid
sequenceDiagram
    autonumber
    actor Merchant as 사장님 / 매니저
    participant ExtAPI as 외부 기관 API (Level 3)
    participant Ingest as Ingestion Service / File Parser
    participant RawDB as PostgreSQL (raw_import_batches)
    participant Norm as Normalization & Dedup Engine
    participant RevDB as PostgreSQL (Revenue & Ledger)
    participant Recon as Level 4 Reconciliation Engine

    alt Level 1: Direct Input (수기 입력)
        Merchant->>Ingest: POST /v1/stores/{id}/sales (수기 매출/비용 등록)
        Ingest->>RawDB: 원본 JSON 스냅샷 저장 (LEVEL_1_MANUAL)
    else Level 2: File Import (엑셀/CSV 업로드)
        Merchant->>Ingest: POST /v1/stores/{id}/imports/upload (여신협회/배달앱/은행 정산파일)
        Ingest->>RawDB: S3 암호화 저장 + 파일 SHA-256 중복 검사 (LEVEL_2_FILE)
    else Level 3: Authorized API (공식 승인 API)
        ExtAPI->>Ingest: 스케줄 폴링 또는 Inbound Webhook 수신 (LEVEL_3_API)
        Ingest->>RawDB: 전자서명 검증 + 원본 응답 보존
    end

    RawDB->>Norm: 정규화 이벤트 발행 (outbox_events)
    Norm->>Norm: 거래 핑거프린트(dedup_fingerprint) 계산 및 중복 제거
    Norm->>RevDB: sales_transactions / payment_authorizations / bank_deposits 적재
    Norm->>RevDB: 복식부기 분개(journal_entries, journal_lines) 자동 생성
    RevDB->>Recon: 정산 예정(payout_schedules) vs 실제 입금(bank_deposits) 대사 트리거
    Recon->>RevDB: reconciliation_matches 기록 & 미입금 후보(UNPAID_CANDIDATE) 산출
```

---

## 2. 제7조 & 제8조 카드 거래 6단계 생명주기 상태 전이

| 단계 | 상태 코드 | 기준 금액 컬럼 | 회계 분개 (자동 생성) | 비고 |
|---|---|---|---|---|
| **Stage 1. 승인** | `AUTHORIZED` | `authorized_amount` (공급가+부가세+봉사료) | 메모 기장 또는 매출 인식 시점 정책에 따라 외상매출금/카드미수금 가집계 | 승인 후 즉시취소(`VOIDED`) 시 상계 |
| **Stage 2. 매입** | `CAPTURED` | `captured_amount` | `[차] 카드미수금 + [차] 카드수수료(예상) / [대] 카드매출 + [대] 부가세예수금` | 전표 매입 확정 |
| **Stage 3. 수수료 산정** | `FEE_CALCULATED` | `fee_amount` (기본+포인트-우대환급) | 수수료 차액 발생 시 수수료 계정 보정 | 카드사별·우대등급별 요율 적용 |
| **Stage 4. 지급 예정** | `SETTLEMENT_SCHEDULED` | `net_settlement_scheduled_amount` | `payout_schedules` 생성 (카드사별 D+2/D+3 영업일 반영) | 매입후취소 발생 시 차기 지급예정액에서 차감 |
| **Stage 5. 은행 입금** | `BANK_DEPOSITED` | `actual_bank_deposit_amount` | `bank_deposits` 수집 (오픈뱅킹 API 또는 통장 엑셀) | N건 거래가 카드사별 일별 1건으로 합산 입금됨 |
| **Stage 6. 입금 대사** | `RECONCILED_MATCHED` / `UNPAID_CANDIDATE` | `variance_amount = scheduled - actual` | `[차] 보통예금 / [대] 카드미수금` (차액 발생 시 미수금 잔액 유지 및 경보) | 데이터 수집 지연 시 `SYNC_DELAY_PENDING`으로 분리 |
