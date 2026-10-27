# 국세청 사업자 상태조회·홈택스·전자세금계산서·현금영수증 연동 명세서

**문서 ID**: CONN-TAX-003  
**승인 상태 분류**:
- 국세청 사업자 진위확인/상태조회: `PUBLIC_API` (공공데이터포털 즉시 이용 가능)
- 전자세금계산서·현금영수증 발행 및 홈택스 매입매출 수집: `COMMERCIAL_PROVIDER` (팝빌/바로빌 등 국세청 표준인증 ASP 계약) / 자체 시스템사업자 등록 시 `LICENSE_REQUIRED`

---

## 1. 국세청 사업자등록정보 진위확인 및 상태조회 API (`PUBLIC_API`)

- **공식 엔드포인트**:
  - 상태조회: `POST https://api.odcloud.kr/api/nts-businessman/v1/status?serviceKey={SERVICE_KEY}`
  - 진위확인: `POST https://api.odcloud.kr/api/nts-businessman/v1/validate?serviceKey={SERVICE_KEY}`
- **호출 한도 및 배치 최적화**: 1회 요청당 최대 100개 사업자번호 배열 전송 가능, 일 최대 100만 건. 신규 사업장 등록 시 실시간 단건 호출하며, 기존 등록 매장 및 거래처(`vendors`)의 휴·폐업 상태 변화는 주 1회 100건 단위 배치로 점검하여 '폐업 거래처 수취 세금계산서 불공제 위험'을 사전 경고한다.

## 2. 전자세금계산서 및 현금영수증 ASP 연동 (`COMMERCIAL_PROVIDER`)

- **설계 원칙**: 전자세금계산서를 국세청에 직접 전송하려면 부가가치세법상 전자세금계산서 시스템사업자 표준인증이 필요하므로, 초기 상용화 단계에서는 국세청 공인 시스템사업자인 **팝빌(Popbill)** 또는 **바로빌(Barobill)**의 공식 REST API/Webhook Adapter를 연결하고, 동시에 홈택스 엑셀 업로드(Level 2)를 기본 제공한다.
