# AI 환각(Hallucination) 방지 및 규제 준수 검증 규격 (AI Validation & Guardrails)

**문서 ID**: AI-VAL-004  
**관련 조항**: 제14조, 제15조, 제31조

---

## 1. AI 경영 비서 및 분석 엔진 5대 안전 가드레일 (Hard Guardrails)

1. **Zero Numerics Fabrication (숫자 생성 금지 — Deterministic Tool Calling)**:
   - LLM이 매출액, 입금액, 수수료, 부가세 예상액을 직접 암산하거나 추론하여 생성하는 것을 전면 금지한다. 모든 숫자는 반드시 Go 백엔드의 결정론적 SQL/회계 엔진(`internal/analytics`, `internal/tax`)이 계산한 JSON 결과만 템플릿에 바인딩하여 설명한다.
2. **금융·신용·세무 오인 표시 절대 금지 (제14조 & 제31조)**:
   - AI 분석 결과를 **"대출 승인 가능 확정"**, **"공식 신용등급/신용점수"**, **"국세청 확정 신고 완료"**처럼 표시하는 문구 생성을 정규식 및 출력 분류기(Output Guardrail Classifier)로 100% 차단한다.
3. **예측 정확도 백테스팅(Backtesting) 및 MAPE 모니터링**:
   - 매주 예측치(`forecasts.predicted_value`)와 실제 발생 매출(`kpi_snapshots.gross_sales_amount`) 간의 평균절대백분율오차($	ext{MAPE}$)를 계산하여 `MAPE > 25%`인 매장은 자동으로 신뢰도 등급을 `LOW`로 낮추고 모델을 재보정한다.
4. **프롬프트 인젝션 및 타 매장 데이터 참조 차단**:
   - 매장 운영 질의응답(Q&A) 시 컨텍스트는 오직 현재 인가된 단일 `store_id`의 집계 요약본으로만 제한되며, 타 매장의 원장 데이터나 시스템 프롬프트에는 접근할 수 없다.
