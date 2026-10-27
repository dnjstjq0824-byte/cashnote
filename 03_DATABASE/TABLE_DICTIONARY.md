# 10대 도메인 56개 전체 테이블 사전 (Table Dictionary)

**문서 ID**: DB-DICT-002  

---

## 도메인별 전체 56개 테이블 매핑 및 격리 키 명세

| 번호 | 도메인 | 테이블명 | PK | 테넌트 격리 키 | 보존 정책 (법정 근거) | 핵심 역할 요약 |
|---|---|---|---|---|---|---|
| 01 | Identity | `users` | `id (UUID)` | Self (`user_id`) | 탈퇴 후 지체없이 파기 (법정 의무분 분리보관) | 통합 사용자 마스터 계정 |
| 02 | Identity | `user_profiles` | `user_id` | Self (`user_id`) | 회원 자격 유지 기간 | 성명, 타임존, 큰글씨 모드, 본인인증 CI 해시 |
| 03 | Identity | `identities` | `id` | `user_id` | 회원 자격 유지 기간 | 카카오·구글·애플 OIDC 및 로컬 인증 수단 매핑 |
| 04 | Identity | `sessions` | `id` | `user_id` | 만료 후 90일 | 기기별 Refresh Token 패밀리 및 원격 로그아웃 제어 |
| 05 | Identity | `roles` | `id` | Global System | 영구 | `OWNER`, `MANAGER`, `ACCOUNTANT`, `TAX_AGENT` 등 역할 정의 |
| 06 | Identity | `permissions` | `id` | Global System | 영구 | 세분화된 도메인·액션 권한 코드 사전 |
| 07 | Business | `businesses` | `id` | `id (business_id)` | 국세기본법 5년 | 사업자등록번호, 과세유형, 휴폐업 상태, 소유권 검증 상태 |
| 08 | Business | `business_verifications` | `id` | `business_id` | 5년 | 국세청 API 진위확인, 1원 계좌인증, 사업자등록증 심사 이력 |
| 09 | Business | `stores` | `id` | `id (store_id)` | 5년 | 개별 매장(점포), 업종(`vertical_type`), 통화, 영업일 마감시각 |
| 10 | Business | `store_members` | `id` | `store_id` | 5년 | 매장별 소속 구성원(직원/매니저/세무사) 및 유효기간 |
| 11 | Business | `store_permissions` | `id` | `store_id` | 5년 | 매장 멤버별 개별 권한 ALLOW/DENY 오버라이드 |
| 12 | Connectors | `provider_catalog` | `id` | Global System | 영구 | 외부 기관/API 카탈로그 및 승인 상태(`PUBLIC_API`, `BLOCKED` 등) |
| 13 | Connectors | `data_connections` | `id` | `store_id` | 철회 후 즉시 토큰 파기 | 매장별 외부 데이터 연동 상태 및 KMS 암호화 토큰 |
| 14 | Connectors | `connection_consents` | `id` | `store_id` | 5년 | 개인정보·제3자 제공·데이터 연동 동의 및 철회 증빙 |
| 15 | Connectors | `raw_import_batches` | `id` | `store_id` | 5년 | Level 1~3 유입 배치 원본 메타데이터 및 파일 SHA-256 |
| 16 | Connectors | `sync_jobs` | `id` | `store_id` | 1년 | 외부 API/파일 동기화 작업 실행 이력 및 처리 건수 |
| 17 | Connectors | `sync_cursors` | `id` | `store_id` | 연동 유지 기간 | 증분 수집을 위한 스트림별 마지막 커서 위치 |
| 18 | Connectors | `ingestion_errors` | `id` | `store_id` | 1년 | 파일 업로드/API 파싱 시 행 단위 오류 내역 및 해결 상태 |
| 19 | Revenue | `sales_transactions` | `id` | `store_id` (RLS) | 국세기본법 5년 | 전 채널(카드·현금·배달·PG) 정규화 매출 마스터 |
| 20 | Revenue | `payment_authorizations` | `id` | `store_id` (RLS) | 여전법/국세법 5년 | 카드 결제 승인 내역(마스킹 PAN, 승인번호, 할부) |
| 21 | Revenue | `payment_captures` | `id` | `store_id` (RLS) | 5년 | 카드 전표 매입 확정, 수수료율(우대율 반영), 지급예정액 |
| 22 | Revenue | `payment_cancellations` | `id` | `store_id` (RLS) | 5년 | 매입 전 취소, 매입 후 취소, 부분취소, 차지백 상계 내역 |
| 23 | Revenue | `acquisition_records` | `id` | `store_id` (RLS) | 5년 | 카드사별 일별 매입 합계 및 정산 예정 집계 |
| 24 | Revenue | `payout_schedules` | `id` | `store_id` (RLS) | 5년 | 카드사·PG·배달앱 정산 지급 예정 스케줄 및 공휴일 이연 추적 |
| 25 | Revenue | `settlement_records` | `id` | `store_id` (RLS) | 5년 | 채널별 총매출 대비 중개수수료·배달비·광고비 차감 정산 명세 |
| 26 | Revenue | `bank_deposits` | `id` | `store_id` (RLS) | 5년 | 사업용 계좌 실제 입출금 내역 및 대사 상태 |
| 27 | Revenue | `reconciliation_matches` | `id` | `store_id` (RLS) | 5년 | 지급 예정(`payout_schedules`) ↔ 실제 입금(`bank_deposits`) 매칭 결과 및 미입금 진단 |
| 28 | Revenue | `reconciliation_disputes` | `id` | `store_id` (RLS) | 5년 | 오탐 진단 수정 및 카드사/플랫폼 정산 차액 이의제기 이력 |
| 29 | Expenses & Ledger | `expense_categories` | `id` | `store_id` | 영구 | 고정비/변동비 및 손익계산서(P&L) 그룹핑 표준 비용 카테고리 |
| 30 | Expenses & Ledger | `vendors` | `id` | `store_id` | 5년 | 식자재·임대인·용역 등 매입 거래처 및 암호화 계좌 정보 |
| 31 | Expenses & Ledger | `expenses` | `id` | `store_id` (RLS) | 5년 | 비용·경비 지출 내역, 적격증빙 종류, 부가세 공제 적격 여부 |
| 32 | Expenses & Ledger | `accounting_periods` | `id` | `store_id` (RLS) | 5년 | 월별 회계 마감(`OPEN` / `LOCKED`) 통제 |
| 33 | Expenses & Ledger | `ledger_accounts` | `id` | `store_id` | 영구 | 복식부기 계정과목표(자산·부채·자본·수익·비용) |
| 34 | Expenses & Ledger | `journal_entries` | `id` | `store_id` (RLS) | 상법 10년 / 세법 5년 | 불변 복식부기 분개 전표 헤더 (`total_debit == total_credit`) |
| 35 | Expenses & Ledger | `journal_lines` | `id` | `store_id` (RLS) | 상법 10년 / 세법 5년 | 분개 전표 차변/대변 상세 라인 |
| 36 | Expenses & Ledger | `corrections` | `id` | `store_id` (RLS) | 10년 | 원본 삭제 없는 역분개 및 정정 분개 감사 연결 테이블 |
| 37 | Tax | `tax_invoices` | `id` | `store_id` (RLS) | 부가세법 5년 | 매출/매입 전자세금계산서 및 계산서 원장 |
| 38 | Tax | `cash_receipts` | `id` | `store_id` (RLS) | 5년 | 매출/매입 현금영수증(소득공제용/지출증빙용) 원장 |
| 39 | Tax | `tax_estimates` | `id` | `store_id` (RLS) | 5년 | 부가세·종소세 예상 세액 시뮬레이션 결과 및 적용 산식 스냅샷 |
| 40 | Tax | `tax_documents` | `id` | `store_id` (RLS) | 5년 | 영수증 스캔본, 세무사 공유 패키지 파일 및 만료 관리 |
| 41 | Store Ops | `products` | `id` | `store_id` (RLS) | 5년 | 매장 판매 메뉴/상품 마스터 및 POS/배달앱 메뉴 별칭 매핑 |
| 42 | Store Ops | `orders` | `id` | `store_id` (RLS) | 5년 | POS·키오스크·배달앱 개별 주문 헤더 및 품목 JSON |
| 43 | Store Ops | `inventory_items` | `id` | `store_id` (RLS) | 5년 | 재고 품목 현재고, 안전재고 기준점, 평균 단가 |
| 44 | Store Ops | `inventory_movements` | `id` | `store_id` (RLS) | 5년 | 입고·판매차감·폐기·실사조정 재고 수불 이력 |
| 45 | Store Ops | `employees` | `id` | `store_id` (RLS) | 근로기준법 3년 | 정직원·파트타임·일용직 근무자 조건 및 시급/월급 정보 |
| 46 | Store Ops | `store_metrics` | `id` | `store_id` (RLS) | 5년 | 월별 매출·영업이익·원가율·인건비율 목표치 |
| 47 | CRM & Analytics | `customer_segments` | `id` | `store_id` (RLS) | 3년 | 익명 방문 코호트 및 마케팅 동의 단골 세그먼트 |
| 48 | CRM & Analytics | `review_sources` | `id` | `store_id` (RLS) | 연동 기간 | 플랫폼별 리뷰 연동 상태 및 공식 답글 API 지원 여부 |
| 49 | CRM & Analytics | `reviews` | `id` | `store_id` (RLS) | 3년 | 수집된 매장 리뷰, 별점, 감성 라벨, 키워드, 답글 상태 |
| 50 | CRM & Analytics | `review_responses` | `id` | `store_id` (RLS) | 3년 | 공식 API 답글 또는 AI 초안 클립보드 복사 이력 |
| 51 | CRM & Analytics | `kpi_snapshots` | `id` | `store_id` (RLS) | 5년 | 일·주·월 단위 사전 집계된 매출·비용·손익·객단가 스냅샷 |
| 52 | CRM & Analytics | `forecasts` | `id` | `store_id` (RLS) | 3년 | AI 매출·현금흐름·재고소진 예측치 및 신뢰구간(P10~P90) |
| 53 | CRM & Analytics | `insight_reports` | `id` | `store_id` (RLS) | 3년 | 일일 모닝 브리핑 및 월간 손익 진단 리포트 아카이브 |
| 54 | Marketplace & Sub | `suppliers` | `id` | Partner Scope | 5년 | B2B 마켓 공급업체 정보 및 판매 모델(`FULFILLMENT_DIRECT` 등) |
| 55 | Marketplace & Sub | `marketplace_products` | `id` | Global Catalog | 5년 | 소상공인 B2B 전용 식자재·비품 상품 카탈로그 |
| 56 | Marketplace & Sub | `marketplace_orders` | `id` | `store_id` | 전자상거래법 5년 | B2B 마켓 주문, 배송, 정산 및 매장 비용 장부 자동 연결 |
| 57 | Marketplace & Sub | `memberships` | `id` | Global System | 영구 | Free / Plus / Pro / Enterprise 구독 요금제 마스터 |
| 58 | Marketplace & Sub | `subscriptions` | `id` | `store_id` | 5년 | 매장별 활성 멤버십 구독 상태 및 빌링키 |
| 59 | Marketplace & Sub | `payments` | `id` | `store_id` | 전자상거래법 5년 | 구독료 및 B2B 마켓 결제 트랜잭션(`idempotency_key` 강제) |
| 60 | Marketplace & Sub | `refunds` | `id` | `store_id` | 5년 | 구독 및 마켓 주문 취소/환불 처리 내역 |
| 61 | Operations | `audit_logs` | `id` | `store_id` | 개인정보보호법 2년+ | SHA-256 해시 체인 기반 불변 보안 감사 로그 |
| 62 | Operations | `notifications` | `id` | `store_id` | 1년 | 알림톡·푸시·이메일·SMS 발송 및 열람 이력 |
| 63 | Operations | `background_jobs` | `id` | `store_id` | 90일 | 비동기 워커 작업 상태 및 Dead-Letter 큐 관리 |
| 64 | Operations | `feature_flags` | `id` | Global System | 영구 | 기능 플래그 및 규제 게이트(`REGULATORY_BLOCKED` 등) |
| 65 | Operations | `admin_actions` | `id` | Admin Scope | 5년 | 관리자 4-Eyes 승인 및 CS Break-Glass 열람 감사 기록 |
| 66 | Operations | `outbox_events` | `id` | `store_id` | 발행 후 14일 | Transactional Outbox 도메인 이벤트 버스 테이블 |
