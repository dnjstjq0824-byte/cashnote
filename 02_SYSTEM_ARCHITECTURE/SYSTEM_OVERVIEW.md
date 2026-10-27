# BN Merchant OS 전체 시스템 아키텍처 개요 (System Overview)

**문서 ID**: ARCH-OVERVIEW-001  
**기술 스택**: Vue 3 + Nuxt 3 + TypeScript (사용자 웹), Vue 3 + TypeScript (관리자/파트너 웹), Flutter (Android/iOS), Go 1.22+ (백엔드/워커), PostgreSQL 16, OpenSearch 2.x, Redis 7.2, S3-Compatible Storage, OpenAPI 3.1, OIDC, OpenTelemetry, Docker.

---

## 1. 제1조 독립 플랫폼 설계 원칙 (Standalone Architecture Principle)

BN Merchant OS는 TreeScan, Delivery Note, OLMOA 등 다른 Booking Network 플랫폼에 대해 **어떠한 코드·DB·세션·인프라 직접 의존성도 갖지 않는 100% 독립형 상용 SaaS 플랫폼**으로 구축된다.

1. **독립 인증 및 회원 DB**: 자체 OIDC 호환 Identity Provider(`users`, `identities`, `sessions`)를 운영하며 타 플랫폼 사용자 DB와 자동 공유하지 않는다.
2. **선택적 외부 어댑터(Optional API Adapter)**: 향후 타 Booking Network 서비스와의 연동이 필요한 경우, 가맹점주의 명시적인 제3자 정보제공 동의(`connection_consents`) 하에 외부 제휴사와 동일한 지위의 `Partner Webhook / REST Adapter`를 통해서만 통신한다.
3. **모듈형 모놀리스 + 격리된 금융 커넥터 워커**:
   - **Core API Server (`cmd/api-server`)**: 인증, 사업장 관리, 장부 조회, 세무 추정, CRM, 커뮤니티, B2B 마켓, 관리자 API를 단일 배포 가능한 Go 바이너리 내 명확히 격리된 도메인 패키지로 제공한다.
   - **Isolated Connector Workers (`cmd/connector-worker`)**: 외부 금융/공공/POS/배달 데이터 수집, 대용량 엑셀/CSV 파싱, 정산 대사(Reconciliation) 배치를 별도 네트워크 보안 구역(Connector DMZ)에서 비동기 실행한다.

---

## 2. 6-Layer 시스템 계층 구조

| 계층 (Layer) | 구성 시스템 | 핵심 역할 및 보안 경계 |
|---|---|---|
| **L1. Client Experience Layer** | `web-merchant` (Nuxt 3), `web-admin` (Vue 3), `mobile-app` (Flutter Android/iOS) | 반응형 웹/모바일 UX, 오프라인 입력 큐(SQLCipher/IndexedDB), 생체인증, 영수증 스캔 |
| **L2. Edge & API Gateway Layer** | Envoy / Nginx API Gateway + WAF | TLS 1.3 종료, OIDC JWT 서명 검증, Rate Limiting, `Idempotency-Key` 검사, `X-Store-Id` 주입 |
| **L3. Core Application Layer** | Go Modular Monolith (`internal/*` 14개 도메인 모듈) | RBAC/ABAC 권한 평가, 비즈니스 로직, 복식부기 분개 생성, OpenAPI 3.1 계약 준수 |
| **L4. Asynchronous Worker & AI Layer** | Go Workers (`Asynq`/Redis Streams) + Reconciliation Engine + Forecast Engine | Level 2 파일 스트리밍 파서, Level 3 외부 API 동기화, Level 4 입금 대사 및 누락 진단, 모닝 브리핑 알림톡 발송 |
| **L5. Data & Persistence Layer** | PostgreSQL 16 (RLS 강제), Redis 7.2, OpenSearch 2.x, S3 Storage | 불변 복식원장(`journal_entries`), 시계열 파티셔닝, 전문 검색, 암호화 증빙 파일 보관 |
| **L6. External Partner & Regulatory Gate** | NTS, CREFIA, OpenBanking, Popbill/Barobill, PG/VAN/POS, Alimtalk | `provider_catalog` 기반 승인 상태(`PUBLIC_API`, `CONTRACT_REQUIRED`, `BLOCKED`) 검증 및 mTLS/고정 IP 통신 |
