# T — Tax & Financial Information (현금영수증·전자세금계산서·부가세 예상·세무사 연계·금융상품 정보)

**문서 ID**: AZ-T_TAX_FINANCE  
**프로젝트 코드명**: BN Merchant OS (`BN-MERCHANT-OS-GEMINI4-ARGON-MASTER-BLUEPRINT-1000-001`)  
**관련 Work Order 범위**: `WO-MOS-0351 ~ WO-MOS-0400, WO-MOS-0765 ~ WO-MOS-0785`  
**설계 원칙**: 제0조~제32조 마스터 명령 전수 준수 (독립 플랫폼 원칙, Level 1~4 데이터 수집 전략, 미승인 외부 API 허위 구현 금지)

---

## 1. 목표 (Objective)

제10조 및 제15조에 따라 전자세금계산서·현금영수증·신용카드 매입매출 증빙을 통합 관리하고, 과세유형별(일반과세자/간이과세자/면세사업자) 부가가치세 예상 세액을 시뮬레이션하며, 세무대리인에게 신고 준비 자료를 원클릭으로 공유하는 워크플로우를 설계한다. 또한 금융상품·정부지원사업 정보 제공 시 법적 경계를 엄격히 준수한다.

---

## 2. 사용자 역할 (User Roles & Actors)

- **주요 액터**: 매장 대표, 회계 담당자, 외부 세무대리인(`ROLE_TAX_AGENT`), Accounting & Tax Agent
- **권한 경계**: 모든 요청은 인증된 사용자(`user_id`)와 명시적으로 선택·검증된 사업자(`business_id`) 및 매장(`store_id`) 컨텍스트 하에서만 실행되며, 타 사업장 데이터에 대한 교차 접근은 애플리케이션 미들웨어와 PostgreSQL RLS(Row-Level Security) 양쪽에서 차단된다.

---

## 3. 기능 명세 (Functional Specifications)

- **핵심 기능 1**: 매출·매입 적격증빙 통합 원장: 전자세금계산서(과세)/전자계산서(면세), 현금영수증(소득공제/지출증빙), 신용카드 매출전표, PG/배달 플랫폼 결제대행 매출 자료 수집 및 수기 종이세금계산서 등록
- **핵심 기능 2**: 부가가치세(VAT) 예상 세액 시뮬레이터: 일반과세자(`매출세액 10% - 매입세액 공제가능분 - 의제매입세액공제 추정 - 신용카드매출전표등 발행세액공제(연 한도 적용)`), 간이과세자(`공급대가 × 업종별 부가가치율 × 10% - 세액공제`, 납부의무 면제 기준금액 판별) 계산 공식 투명 공개 및 면책 고지('본 수치는 참고용 추정치이며 확정 신고세액이 아닙니다')
- **핵심 기능 3**: 불공제 매입세액 자동 분류 보조: 비영업용 소형승용차 구입·유지비, 접대비 관련 매입세액, 면세사업 관련 매입세액, 토지 관련 매입세액을 후보로 하이라이트하여 세무사가 최종 검토하도록 지원
- **핵심 기능 4**: 세무사 공유 패키지(Tax Accountant Pack): 부가세/종합소득세/원천세 신고 기간에 맞춰 매출처별·매입처별 세금계산서 합계표, 카드/현금영수증/배달 매출 집계표, 인건비/고정비 증빙 압축 파일(더존 SmartA / 세무사랑 호환 엑셀 포맷 포함)을 생성하여 세무대리인 전용 포털로 공유
- **핵심 기능 5**: 세금 납부 캘린더 및 증빙 누락 알림: 1기/2기 부가세 확정·예정신고, 5월 종합소득세, 매월 10일 원천세 일정 알림 및 '계좌 이체는 있으나 세금계산서/현금영수증이 누락된 지출 건' 자동 리마인더

---

## 4. 사용자 시나리오 (End-to-End User Scenario)

> **실전 시나리오**: 1월 부가세 확정신고 기간을 앞두고 음식점 사장님이 '세금 신고 준비' 화면을 열면, 하반기 총매출과 적격 매입증빙, 농수산물 의제매입세액공제 추정치가 반영된 예상 부가세 범위가 표시되고, '식자재 계좌이체 3건(합계 120만 원)의 세금계산서가 아직 확인되지 않았습니다'라는 누락 안내가 뜬다. 사장님이 '담당 세무사에게 자료 보내기'를 누르면 세무사랑 호환 엑셀 패키지가 세무사 포털로 안전하게 전달된다.

1. **Pre-condition**: 사용자가 정상 로그인되어 있고 해당 매장에 대해 역할 권한(`RBAC/ABAC`)을 보유하고 있다. 외부 연동 기능의 경우 `provider_catalog` 및 `data_connections` 상태가 확인된다.
2. **Normal Flow**: 요청이 API Gateway를 통과하여 도메인 서비스에서 처리되고, 원본 출처(`data_source_level`)와 함께 PostgreSQL 트랜잭션 및 `outbox_events`에 기록된다.
3. **Fallback Flow (연동 차단/미계약 시)**: 만약 외부 API가 `CONTRACT_REQUIRED` 또는 `PARTNER_ONLY(BLOCKED)` 상태이면, 가짜 실시간 데이터를 표시하지 않고 즉시 **Level 2(공식 정산파일 CSV/Excel 업로드)** 또는 **Level 1(수기 입력)** 인터페이스로 안내하여 핵심 장부·대사 업무가 중단 없이 수행되도록 보장한다.

---

## 5. 백엔드 아키텍처 및 구현 명세 (Backend Architecture)

- **언어 및 프레임워크**: Go 1.22+, 모듈형 모놀리스 아키텍처 + 독립 비동기 워커(Go Workers)
- **핵심 구현 모듈**: `internal/tax` 패키지: 과세유형별 부가세 시뮬레이션 엔진, 의제매입세액공제/신용카드발행세액공제 한도 계산기, 더존/세무사랑 호환 엑셀 익스포터, 공인 전자세금계산서 ASP(팝빌/바로빌) 어댑터
- **트랜잭션 관리**: 모든 상태 변경과 원장 반영은 `pgx/v5` 트랜잭션 내부에서 도메인 데이터 변경 + `audit_logs` 기록 + `outbox_events` 발행을 원자적(Atomic)으로 커밋한다.

---

## 6. 프론트엔드 및 모바일 명세 (Frontend & Mobile)

- **구현 범위**: 세금 예상 대시보드, 매출/매입 증빙 매칭 보드, 불공제 후보 검토 리스트, 세금 일정 캘린더, 세무사 초대 및 자료 공유 화면, 정책자금/금융정보 안내 보드
- **상태 표현 표준**:
  1. `LOADING`: 스켈레톤 UI 및 진행률 표시
  2. `SUCCESS`: 데이터 표시 + 우측 상단 `DataProvenanceBadge`(데이터 출처 Level 1~4 및 최종 갱신 시각)
  3. `EMPTY`: 초기 온보딩 안내 및 첫 데이터 등록/업로드 CTA 버튼
  4. `ERROR`: 사용자 친화적 오류 메시지 + `request_id` 복사 + 재시도 버튼
  5. `BLOCKED`: 외부 기관 계약/승인 대기 상태의 정직한 고지 + 파일 업로드 대체 경로 버튼

---

## 7. 데이터베이스 설계 연동 (Database Schema Mapping)

- **연관 테이블**: `tax_invoices`, `cash_receipts`, `tax_estimates`, `tax_documents`
- **데이터 무결성 규칙**:
  - 모든 금액 컬럼은 `NUMERIC(19,4)`를 사용하여 부동소수점 반올림 오차를 원천 배제한다.
  - 모든 시각 컬럼은 `TIMESTAMPTZ`로 저장한다.
  - 원본 거래 레코드는 임의 삭제(`DELETE`)하지 않으며, 정정 시 `corrections` 및 역분개(`journal_entries`)로 감사 추적성을 유지한다.

---

## 8. API 계약 명세 (OpenAPI 3.1 Endpoints)

- **주요 엔드포인트**: `GET /v1/stores/{storeId}/tax/vat-estimate`, `GET /v1/stores/{storeId}/tax/invoices`, `POST /v1/stores/{storeId}/tax/invoices/issue`, `GET /v1/stores/{storeId}/tax/cash-receipts`, `POST /v1/stores/{storeId}/tax/export-package`
- **공통 헤더**:
  - `Authorization: Bearer <OIDC_JWT>`
  - `X-Business-Id: <UUID>`
  - `X-Store-Id: <UUID>`
  - `Idempotency-Key: <UUID>` (모든 `POST`/`PUT`/`PATCH` 상태 변경 요청 시 필수)

---

## 9. 도메인 이벤트 (Domain Events)

- **발행/구독 이벤트**: `tax.invoice.collected`, `tax.invoice.issued`, `tax.estimate.calculated`, `tax.package.exported`
- **전달 보장**: Transactional Outbox 테이블(`outbox_events`)에서 Redis Streams / Worker Queue로 At-Least-Once 전달을 보장하며, 컨슈머는 `event_id` 기준 멱등 처리(Idempotent Consumer)를 수행한다.

---

## 10. 데이터 출처 및 신뢰도 등급 (Data Provenance)

- **데이터 원천**: Level 1 수기 입력, Level 2 홈택스 엑셀 업로드, Level 3 공인 전자세금계산서 ASP API(팝빌/바로빌 등) 및 승인된 홈택스 연동
- **출처 태깅 규칙**:
  - `LEVEL_1_MANUAL`: 사용자 직접 입력 (신뢰도: 사용자 책임, 즉시 반영)
  - `LEVEL_2_FILE`: 공식 정산 엑셀/CSV 업로드 (신뢰도: 파일 해시 및 포맷 검증됨)
  - `LEVEL_3_API`: 공식 계약/승인된 외부 API (신뢰도: 기관 전자서명/TLS 검증됨)
  - `LEVEL_4_RECONCILED`: 복수 데이터 소스(예: 카드매입 + 은행입금) 교차 대사 완료 (신뢰도: 최고 등급 확정)

---

## 11. 오류 처리 및 장애 복구 (Error Handling & Resilience)

- **예외 처리 정책**: 전자세금계산서 직접 발행 기능은 국세청 시스템사업자 인증 또는 공인 ASP 연동 계약이 활성화된 경우에만 실행되며, 미계약 시 `BLOCKED_PROVIDER_CONTRACT` 상태와 홈택스 직접 발행 안내 제공
- **표준 에러 코드**: `AUTH_FORBIDDEN_STORE`, `CONNECTOR_EXTERNAL_BLOCKED`, `LEDGER_UNBALANCED_ENTRY`, `IMPORT_DUPLICATE_BATCH`, `RATE_LIMIT_EXCEEDED`, `UPSTREAM_TIMEOUT_RETRYABLE`

---

## 12. 개인정보 및 보안 통제 (Privacy & Security Controls)

- **보안 설계**: 세무사 공유 링크는 만료 기한(기본 14일), 비밀번호/OTP 2차 인증, 다운로드 횟수 제한 및 접근 IP 감사 로그를 강제하며, 세무대리인이 아닌 제3자에게 전달될 수 없도록 보호.
- **감사 추적**: 모든 민감 데이터 조회 및 변경은 `audit_logs`에 요청자 ID, 대상 매장 ID, 액션 코드, IP, User-Agent, 변경 전후 요약 및 해시 체인과 함께 기록된다.

---

## 13. 외부 승인 및 법적 선행 요건 (External Approval & Regulatory Requirements)

- **승인·규제 요건**: 세무사법 제2조 및 제20조 준수: 플랫폼은 장부 정리 및 신고 준비 자료 생성·전달 도구만 제공하며, 유상 세무대리(세무신고 대행 및 세무자문)는 반드시 적법한 자격을 갖춘 세무사/회계사가 수행하도록 분리. 금융상품 추천 역시 대출 모집업 등록 전까지는 객관적 정책자금 공고 안내로 한정.
- **차단(Blocked) 정책**: 필수 외부 계약이나 인허가가 완료되기 전까지 해당 라이브 연동 기능은 `feature_flags` 및 `provider_catalog`에서 `BLOCKED`로 분류하며, 가짜 API 응답으로 상용 완료를 가장하는 행위를 엄격히 금지한다.

---

## 14. 성능 및 확장성 목표 (Performance & SLA Targets)

- **목표 지표**: 6개월 치 부가세 추정 시뮬레이션 재계산 < 200ms, 세무 신고용 엑셀 패키지 생성 < 3초.

---

## 15. 테스트 전략 (Verification & Testing Strategy)

- **단위 테스트(Unit Test)**: 도메인 계산 로직, 상태 머신 전이, 경계값 및 예외 케이스 검증
- **통합 테스트(Integration Test)**: 실제 PostgreSQL 16 컨테이너 기반 DDL, FK/UNIQUE 제약, RLS 테넌트 격리 검증
- **계약/E2E 테스트(Contract & E2E Test)**: OpenAPI 3.1 스키마 일치성 검증 및 제26조 필수 E2E 시나리오(Scenario A~J) 연동 검증

---

## 16. 완료 승인 기준 (Acceptance Criteria)

1. 본 문서에 정의된 모든 API 엔드포인트와 DB 테이블이 `03_DATABASE/` 및 `04_API_SPECIFICATION/openapi.yaml`과 명칭·타입 불일치 없이 100% 일치한다.
2. 디자인만 존재하는 더미 버튼이 없으며, 로딩·성공·빈 상태·오류·연동차단(`BLOCKED`) 5가지 상태가 모두 구현 및 검증된다.
3. 타 사업장(`store_id`) 데이터에 대한 무단 접근 시도가 서버 측에서 `403 Forbidden` 또는 빈 결과(RLS)로 100% 차단된다.
4. 미계약 외부 API에 의존하는 기능이 Level 1(수기 입력) 및 Level 2(공식 파일 업로드) 대체 경로를 통해 중단 없이 동작한다.
5. 단위·통합·E2E 테스트 로그 및 아티팩트 증빙이 독립 감사 에이전트(`Independent Audit Agent`)의 검증을 통과한다.

---

## 17. 관련 Work Order 매핑 (Related Work Orders)

- **직접 매핑 범위**: `WO-MOS-0351 ~ WO-MOS-0400, WO-MOS-0765 ~ WO-MOS-0785`
- **상세 개별 명세 위치**: `11_WORK_ORDERS_0001_TO_1000/individual/` 내 해당 번호 파일 및 `11_WORK_ORDERS_0001_TO_1000/WORK_ORDER_INDEX.csv` 참조.
