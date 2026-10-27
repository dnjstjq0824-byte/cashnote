# 매장 리뷰 및 매장 정보 플랫폼 연동 명세서

**문서 ID**: CONN-REV-006  
**승인 상태 분류**: Google Business Profile API (`APPROVAL_REQUIRED`), 국내 지도/배달 리뷰 (`PARTNER_ONLY` / `LEVEL_2_FILE`)

---

## 1. 플랫폼별 리뷰 수집 및 답변(Reply) 가능 범위 정의 (제13조 & 제26조 Scenario H 반영)

| 플랫폼 | 공식 Open API 제공 여부 | 수집 방식 | 답글 등록 방식 |
|---|---|---|---|
| **Google Business Profile** | 공식 제공 (`mybusiness.googleapis.com`) | OAuth 2.0 매장 소유자 인가 후 실시간/정기 수집 (`LEVEL_3_API`) | 대시보드 내에서 `PUT .../reviews/{id}/reply` 공식 API로 즉시 답글 게시 |
| **자체 스마트오더 / 영수증 리뷰** | 자체 내장 API (`PUBLIC_API`) | 결제 고객 영수증 QR/알림톡 피드백 즉시 수집 | 대시보드에서 즉시 답글 및 보상 쿠폰 발송 |
| **네이버 플레이스 / 카카오맵 / 배달3사** | 일반 서드파티용 공개 리뷰 답글 Open API 미개방 (`PARTNER_ONLY`) | 공식 제휴 체결 시 API 연동 / 미제휴 시 사장님 내보내기 파일 업로드(`LEVEL_2_FILE`) 또는 알림 공유 | **AI 맞춤 답글 초안 생성 + 원클릭 클립보드 복사 + 해당 사장님 관리앱 실행 딥링크** 제공 (무단 봇 답글 금지) |
