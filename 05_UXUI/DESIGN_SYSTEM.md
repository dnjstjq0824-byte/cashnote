# BN Merchant OS 디자인 시스템 명세 (Argon Slate & Trust Emerald Design System)

**문서 ID**: UX-DS-001  
**독립 브랜드 원칙**: 제17조 및 제31조에 따라 캐시노트 등 타사의 로고, 그래픽 자산, 전용 서체, 레이아웃 소스코드를 일체 복제하지 않고 소상공인 재무 가독성에 최적화된 독자 디자인 토큰을 사용한다.

---

## 1. 시맨틱 컬러 토큰 (Semantic Color Palette — WCAG 2.1 AA 4.5:1 이상 명도대비 보장)

| 토큰 명칭 | Hex 코드 | 용도 및 재무적 의미 |
|---|---|---|
| `--color-brand-primary` | `#0F172A` (Argon Deep Slate) | 네비게이션, 핵심 헤더, 권위와 신뢰의 기본 프레임 |
| `--color-brand-accent` | `#059669` (Trust Emerald 600) | **실제 은행 입금 완료(`LEVEL_4_RECONCILED`)**, 대사 일치, 긍정적 손익 |
| `--color-finance-authorized` | `#2563EB` (Royal Blue 600) | **카드/배달 승인 금액(`authorized_amount`)** — 아직 입금 전인 매출 |
| `--color-finance-scheduled` | `#0D9488` (Teal 600) | **수수료 차감 후 입금 예정 금액(`net_settlement_scheduled_amount`)** |
| `--color-finance-warning` | `#D97706` (Amber 600) | **입금 지연 후보(`DELAYED_CANDIDATE`)**, 데이터 갱신 지연 대기, 예산 80% 도달 |
| `--color-finance-danger` | `#DC2626` (Crimson 600) | **확정 미입금/지급보류(`CONFIRMED_UNPAID`)**, 수수료 과다 차감, 부정 리뷰(1~2점) |
| `--color-finance-expense` | `#475569` (Slate 600) | 비용·경비 지출 및 수수료 공제액 |

---

## 2. 타이포그래피 및 고령 사장님 배려 큰 글씨 모드 (Typography & Accessibility)

- **기본 폰트 패밀리**: `Pretendard Variable`, `Inter`, `-apple-system`, `sans-serif`
- **재무 숫자 필수 CSS 규칙**: 모든 금액, 퍼센트, 날짜, 승인번호에는 `font-variant-numeric: tabular-nums;`를 강제하여 테이블 행 간 자릿수가 수직으로 완벽히 정렬되도록 한다.
- **표준 모드 vs 큰 글씨 모드(125% Scale)**:
  - 핵심 히어로 금액: 표준 `28px (1.75rem)` / 큰 글씨 `34px`
  - 섹션 타이틀: 표준 `20px (1.25rem)` / 큰 글씨 `24px`
  - 본문 및 테이블 셀: 표준 `15px` / 큰 글씨 `18px`
  - 최소 터치 영역: 모바일 모든 버튼·탭 `48px × 48px` 이상 강제
