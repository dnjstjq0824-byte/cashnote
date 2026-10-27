# 컴포넌트 구조 및 도메인 경계 다이어그램 (Component Diagrams)

**문서 ID**: ARCH-COMP-002  

---

## 1. 전체 컨텍스트 및 컨테이너 아키텍처 (Mermaid C4 Diagram)

```mermaid
flowchart TB
    subgraph Clients["L1. Client Applications"]
        WebUser["사용자 웹 대시보드<br/>(Vue 3 + Nuxt 3 + TS)"]
        MobileApp["모바일 앱 Android/iOS<br/>(Flutter 3.24+)"]
        WebAdmin["관리자·파트너 콘솔<br/>(Vue 3 + TS)"]
    end

    subgraph Edge["L2. Edge & Security Boundary"]
        WAF["WAF & DDoS Protection"]
        APIGW["API Gateway (Envoy/Go)<br/>JWT Auth / Rate Limit / Idempotency"]
    end

    subgraph CoreZone["L3. Core Application Zone (Go Modular Monolith)"]
        IdentityMod["Identity & RBAC/ABAC Module"]
        BizMod["Business & Store Verification Module"]
        RevMod["Revenue & Card Sales Module"]
        ReconMod["Settlement & Reconciliation Engine"]
        LedgerMod["Immutable Double-Entry Ledger"]
        TaxMod["Tax & e-Invoice Module"]
        OpsMod["Vertical Store Ops & CRM Module"]
        AIMod["AI Forecasting & KPI Module"]
        CommMod["Community & B2B Marketplace Module"]
        AdminMod["Admin & Audit Governance Module"]
    end

    subgraph WorkerDMZ["L4. Connector & Batch Worker DMZ Zone"]
        FileParser["Level 2 Universal File Parser<br/>(CSV/XLSX Streaming)"]
        SyncWorker["Level 3 External API Sync Worker<br/>(Circuit Breaker + Egress Static IP)"]
        ReconWorker["Level 4 Auto-Reconciliation Worker"]
        NotifWorker["Notification Dispatcher<br/>(Alimtalk / FCM / APNs / Email)"]
    end

    subgraph DataZone["L5. Restricted Data Persistence Zone"]
        PG[("PostgreSQL 16 Primary/Replica<br/>56 Tables + Row-Level Security")]
        Redis[("Redis 7.2 Cluster<br/>Cache / Session / Job Queue")]
        OS[("OpenSearch 2.x<br/>Transactions / Policies / Market")]
        S3[("S3-Compatible Storage<br/>KMS Encrypted Receipts & Imports")]
    end

    subgraph External["L6. External Institutions & Providers"]
        NTS["국세청 사업자진위/상태 API<br/>(PUBLIC_API)"]
        CREFIA["여신금융협회/카드사 매통조<br/>(CONTRACT_REQUIRED)"]
        KFTC["금융결제원 오픈뱅킹/펌뱅킹<br/>(APPROVAL_REQUIRED)"]
        ASP["전자세금계산서/현금영수증 ASP<br/>(COMMERCIAL_PROVIDER)"]
        POSPG["제휴 POS / VAN / PG<br/>(CONTRACT / PARTNER)"]
        Delivery["배달 플랫폼<br/>(PARTNER_ONLY / Level 2 File Fallback)"]
    end

    Clients --> WAF --> APIGW
    APIGW --> IdentityMod & BizMod & RevMod & ReconMod & LedgerMod & TaxMod & OpsMod & AIMod & CommMod & AdminMod
    CoreZone --> PG & Redis & OS & S3
    CoreZone -.->|Transactional Outbox| Redis
    Redis --> FileParser & SyncWorker & ReconWorker & NotifWorker
    FileParser & ReconWorker --> PG & S3
    SyncWorker --> NTS & CREFIA & KFTC & ASP & POSPG & Delivery
```

---

## 2. Go 백엔드 내부 패키지 의존성 규칙 (Strict Package Layering)

순환 참조(Circular Dependency)와 도메인 오염을 막기 위해 Go 모듈(`github.com/booking-network/bn-merchant-os`) 내부는 단방향 의존성만 허용한다:

1. `internal/platform/*` (Config, Logger, Telemetry, KMS Crypto, DB RLS Tx Wrapper, Clock, HolidayCalendar) — 최하위 기반 계층
2. `internal/identity/*` 및 `internal/business/*` — 테넌트(`business_id`, `store_id`) 및 권한 검증 계층
3. `internal/ledger/*` — 불변 복식부기 원장 코어 (Revenue, Expense, Tax, Marketplace 모듈이 이벤트를 통해 전표를 발행)
4. `internal/connector/*`, `internal/revenue/*`, `internal/expense/*`, `internal/tax/*`, `internal/operations/*`, `internal/crm/*`, `internal/ai/*`, `internal/commerce/*` — 비즈니스 도메인 계층
5. `internal/admin/*` 및 `cmd/api-server`, `cmd/connector-worker` — 최상위 조합 및 엔트리포인트 계층
